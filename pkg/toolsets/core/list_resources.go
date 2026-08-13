package core

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rancher/rancher-ai-mcp/internal/middleware"
	"github.com/rancher/rancher-ai-mcp/pkg/client"
	"github.com/rancher/rancher-ai-mcp/pkg/response"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/util/jsonpath"
)

// listKubernetesResourcesParams specifies the parameters needed to list kubernetes resources.
type listKubernetesResourcesParams struct {
	Namespace     string `json:"namespace" jsonschema:"the namespace where the resources are located. It must be empty for all namespaces or cluster-wide resources"`
	Kind          string `json:"kind" jsonschema:"the type of Kubernetes resource (e.g., Pod, Deployment, Service)"`
	Cluster       string `json:"cluster" jsonschema:"the name of the Kubernetes cluster"`
	Limit         int64  `json:"limit,omitempty" jsonschema:"maximum number of resources to return, defaults to 100"`
	Offset        int64  `json:"offset,omitempty" jsonschema:"how many resources to skip from the start of the full list before returning results. Defaults to 0 (start at the first resource). Use it together with limit to page through results: set offset=0 for the first page, then increase offset by limit for each next page. For example, with limit=10: offset=0 returns resources 1-10, offset=10 returns resources 11-20, offset=20 returns resources 21-30. When more resources are available, the response tells you the exact offset to use for the next page"`
	LabelSelector string `json:"labelSelector,omitempty" jsonschema:"optional label selector to filter resources (e.g. app=nginx)"`
	JSONPath      string `json:"jsonPath,omitempty" jsonschema:"optional JSONPath filter predicate to select matching resources. Use @ to reference a resource, e.g. @.status.phase==\"Running\" or @.metadata.labels.app==\"nginx\". Only resources matching the predicate are returned"`
}

// listKubernetesResources lists Kubernetes resources of a specific kind and namespace.
func (t *Tools) listKubernetesResources(ctx context.Context, toolReq *mcp.CallToolRequest, params listKubernetesResourcesParams) (*mcp.CallToolResult, any, error) {
	zap.L().Debug("listKubernetesResource called", zap.String("resourceKind", params.Kind))

	resources, err := t.client.GetResources(ctx, client.ListParams{
		Cluster:       params.Cluster,
		Kind:          params.Kind,
		Namespace:     params.Namespace,
		Token:         middleware.Token(ctx),
		LabelSelector: params.LabelSelector,
	})
	if err != nil {
		zap.L().Error("failed to list resources", zap.String("tool", "listKubernetesResource"), zap.Error(err))
		return nil, nil, err
	}

	if params.JSONPath != "" {
		resources, err = filterByJSONPath(resources, params.JSONPath)
		if err != nil {
			zap.L().Error("failed to filter resources by jsonpath", zap.String("tool", "listKubernetesResources"), zap.Error(err))
			return nil, nil, err
		}
	}

	page := t.paginator.SortAndPaginate(resources, params.Offset, params.Limit)

	filterSuffix := ""
	if params.JSONPath != "" {
		filterSuffix = " matching the JSONPath filter"
	}

	var mcpResponse string
	if note := t.paginator.BuildNote(page, filterSuffix); note != "" {
		mcpResponse, err = response.CreateMcpResponse(page.Items, params.Cluster, note)
	} else {
		mcpResponse, err = response.CreateMcpResponse(page.Items, params.Cluster)
	}
	if err != nil {
		zap.L().Error("failed to create mcp response", zap.String("tool", "listKubernetesResource"), zap.Error(err))
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: mcpResponse}},
	}, nil, nil
}

// filterByJSONPath returns the subset of objs matching the given JSONPath predicate
// expression (the body of a kubectl-style [?(...)] filter, e.g. `@.status.phase=="Running"`).
// The objects are wrapped as a list so the filter can iterate over them, mirroring
// kubectl's `{.items[?(<predicate>)]}` selector semantics.
//
// `||` is supported at any nesting depth, including inside [?(...)] predicates.
// Each OR branch is evaluated as an independent query and results are merged
// in original input order without duplicates.
func filterByJSONPath(objs []*unstructured.Unstructured, expr string) ([]*unstructured.Unstructured, error) {
	branches := expandAllOR(expr)
	if len(branches) == 1 {
		return filterByJSONPathSingle(objs, expr)
	}

	seen := make(map[*unstructured.Unstructured]bool, len(objs))
	for _, branch := range branches {
		matches, err := filterByJSONPathSingle(objs, branch)
		if err != nil {
			return nil, err
		}
		for _, obj := range matches {
			seen[obj] = true
		}
	}
	ordered := make([]*unstructured.Unstructured, 0, len(seen))
	for _, obj := range objs {
		if seen[obj] {
			ordered = append(ordered, obj)
		}
	}
	return ordered, nil
}

// expandAllOR recursively expands every || in a JSONPath predicate expression
// into a flat list of OR-free sub-expressions. Both top-level || and || nested
// inside [?(...)] predicates are expanded.
func expandAllOR(expr string) []string {
	s := splitFirstOR(expr)
	if s == nil {
		return []string{expr}
	}
	var out []string
	out = append(out, expandAllOR(s.prefix+s.left+s.suffix)...)
	out = append(out, expandAllOR(s.prefix+s.right+s.suffix)...)
	return out
}

type orSplit struct {
	prefix, left, right, suffix string
}

// splitFirstOR locates the first || in expr that is not inside a string literal
// and returns how to split the expression around it.
//
// Top-level: "A || B" → {left:"A", right:"B"}
// Nested:    "P[?(A || B)]S" → {prefix:"P[?(", left:"A", right:"B", suffix:")]S"}
//
// Returns nil when no || is present.
func splitFirstOR(expr string) *orSplit {
	inDouble, inSingle := false, false
	for i := 0; i < len(expr); i++ {
		switch {
		case inDouble:
			if expr[i] == '"' {
				inDouble = false
			}
		case inSingle:
			if expr[i] == '\'' {
				inSingle = false
			}
		default:
			switch expr[i] {
			case '"':
				inDouble = true
			case '\'':
				inSingle = true
			case '|':
				if i+1 >= len(expr) || expr[i+1] != '|' {
					continue
				}
				if bracketDepthAt(expr, i) == 0 {
					return &orSplit{
						left:  strings.TrimSpace(expr[:i]),
						right: strings.TrimSpace(expr[i+2:]),
					}
				}
				// Nested: reconstruct two expressions by splitting the enclosing [?( ... )]
				predicateOpen := findLastPredicateOpen(expr[:i])
				if predicateOpen < 0 {
					return nil
				}
				contentStart := predicateOpen + 3 // after [?(
				closePos := findPredicateClose(expr, contentStart)
				if closePos < 0 {
					return nil
				}
				return &orSplit{
					prefix: expr[:contentStart],
					left:   strings.TrimSpace(expr[contentStart:i]),
					right:  strings.TrimSpace(expr[i+2 : closePos]),
					suffix: expr[closePos:],
				}
			}
		}
	}
	return nil
}

// bracketDepthAt counts the number of unmatched '[' before position pos.
func bracketDepthAt(expr string, pos int) int {
	depth, inDouble, inSingle := 0, false, false
	for i := 0; i < pos; i++ {
		switch {
		case inDouble:
			if expr[i] == '"' {
				inDouble = false
			}
		case inSingle:
			if expr[i] == '\'' {
				inSingle = false
			}
		default:
			switch expr[i] {
			case '"':
				inDouble = true
			case '\'':
				inSingle = true
			case '[':
				depth++
			case ']':
				depth--
			}
		}
	}
	return depth
}

// findLastPredicateOpen returns the start position of the last '[?(' in s.
func findLastPredicateOpen(s string) int {
	pos, inDouble, inSingle := -1, false, false
	for i := 0; i < len(s); i++ {
		switch {
		case inDouble:
			if s[i] == '"' {
				inDouble = false
			}
		case inSingle:
			if s[i] == '\'' {
				inSingle = false
			}
		default:
			switch s[i] {
			case '"':
				inDouble = true
			case '\'':
				inSingle = true
			case '[':
				if i+2 < len(s) && s[i+1] == '?' && s[i+2] == '(' {
					pos = i
				}
			}
		}
	}
	return pos
}

// findPredicateClose returns the position of the ')' that closes the predicate
// whose content begins at contentStart (the first character after the opening '(').
func findPredicateClose(expr string, contentStart int) int {
	depth, inDouble, inSingle := 0, false, false
	for i := contentStart; i < len(expr); i++ {
		switch {
		case inDouble:
			if expr[i] == '"' {
				inDouble = false
			}
		case inSingle:
			if expr[i] == '\'' {
				inSingle = false
			}
		default:
			switch expr[i] {
			case '"':
				inDouble = true
			case '\'':
				inSingle = true
			case '(':
				depth++
			case ')':
				if depth == 0 {
					return i
				}
				depth--
			}
		}
	}
	return -1
}

func filterByJSONPathSingle(objs []*unstructured.Unstructured, expr string) ([]*unstructured.Unstructured, error) {
	// Expressions containing nested [?(...)] predicates cannot be composed with
	// the outer {.items[?(...)]} wrapper; evaluate each object individually instead.
	if strings.Contains(expr, "[?(") {
		return filterByJSONPathPerObject(objs, expr)
	}

	items := make([]interface{}, len(objs))
	for i, obj := range objs {
		items[i] = obj.Object
	}
	input := map[string]interface{}{"items": items}

	jp := jsonpath.New("filter").AllowMissingKeys(true)
	if err := jp.Parse("{.items[?(" + expr + ")]}"); err != nil {
		return nil, fmt.Errorf("invalid jsonPath filter %q: %w", expr, err)
	}

	results, err := jp.FindResults(input)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate jsonPath filter %q: %w", expr, err)
	}

	// Collect the map pointers of matched objects.
	matched := make(map[uintptr]bool)
	for _, group := range results {
		for _, v := range group {
			if v.Kind() == reflect.Interface {
				v = v.Elem()
			}
			if v.Kind() == reflect.Map {
				matched[v.Pointer()] = true
			}
		}
	}

	// Return originals in input order so callers preserve stable ordering.
	filtered := make([]*unstructured.Unstructured, 0, len(matched))
	for _, obj := range objs {
		if matched[reflect.ValueOf(obj.Object).Pointer()] {
			filtered = append(filtered, obj)
		}
	}
	return filtered, nil
}

// filterByJSONPathPerObject evaluates expr on each object individually and
// returns those for which the expression produces at least one result.
// The leading '@' is replaced with '.' so the expression addresses the object root.
func filterByJSONPathPerObject(objs []*unstructured.Unstructured, expr string) ([]*unstructured.Unstructured, error) {
	standaloneExpr := expr
	if strings.HasPrefix(standaloneExpr, "@") {
		standaloneExpr = "." + standaloneExpr[1:]
	}
	jp := jsonpath.New("filter").AllowMissingKeys(true)
	if err := jp.Parse("{" + standaloneExpr + "}"); err != nil {
		return nil, fmt.Errorf("invalid jsonPath filter %q: %w", expr, err)
	}
	var filtered []*unstructured.Unstructured
	for _, obj := range objs {
		results, err := jp.FindResults(obj.Object)
		if err != nil {
			continue
		}
		for _, group := range results {
			if len(group) > 0 {
				filtered = append(filtered, obj)
				break
			}
		}
	}
	return filtered, nil
}
