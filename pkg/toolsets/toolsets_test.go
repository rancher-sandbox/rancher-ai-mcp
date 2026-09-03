package toolsets

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rancher/rancher-ai-mcp/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllToolSets(t *testing.T) {
	c, err := client.NewClient(true, "https://fake-url")
	require.NoError(t, err)
	toolsets := allToolSets(c, false)

	assert.NotNil(t, toolsets)
	assert.Len(t, toolsets, 3, "should have exactly 3 toolsets (core, fleet, and provisioning)")
}

func TestToolSchemasValidity(t *testing.T) {
	c, err := client.NewClient(true, "https://fake-url")
	require.NoError(t, err)

	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "v1.0.0",
	}, nil)
	AddAllTools(c, mcpServer, false)

	handler := mcp.NewStreamableHTTPHandler(func(request *http.Request) *mcp.Server {
		return mcpServer
	}, &mcp.StreamableHTTPOptions{})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	serverAddr := "http://" + listener.Addr().String()
	server := &http.Server{Handler: handler}
	go func() {
		_ = server.Serve(listener)
	}()
	defer server.Shutdown(context.Background())

	ctx := context.Background()
	transport := &mcp.StreamableClientTransport{Endpoint: serverAddr}
	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v1.0.0"}, nil)

	var cs *mcp.ClientSession
	require.Eventually(t, func() bool {
		var err error
		cs, err = mcpClient.Connect(ctx, transport, nil)
		return err == nil
	}, 2*time.Second, 100*time.Millisecond)
	defer cs.Close()

	toolsResult, err := cs.ListTools(ctx, &mcp.ListToolsParams{})
	require.NoError(t, err)
	require.NotEmpty(t, toolsResult.Tools)

	for _, tool := range toolsResult.Tools {
		t.Run(tool.Name, func(t *testing.T) {
			schemaBytes, err := json.Marshal(tool.InputSchema)
			require.NoError(t, err)

			var schemaMap map[string]any
			require.NoError(t, json.Unmarshal(schemaBytes, &schemaMap), "schema for %s must be valid JSON", tool.Name)

			validateSchemaNode(t, tool.Name, "", schemaMap)
		})
	}
}

// validateSchemaNode checks that a schema node complies with Gemini/LLM strict requirements:
// 1. If "properties" is present, "type" must be "object".
// 2. If "items" is present, "type" must be "array".
// 3. No "anyOf" containing null unions or array/items inside anyOf.
func validateSchemaNode(t *testing.T, toolName, path string, node map[string]any) {
	nodeType, hasType := node["type"].(string)
	types, hasTypes := node["types"].([]any)
	if hasTypes {
		assert.Fail(t, fmt.Sprintf("[%s%s] multi-type 'types' is not supported by Gemini: %v", toolName, path, types))
	}

	if _, hasProps := node["properties"]; hasProps {
		assert.True(t, hasType && nodeType == "object", "[%s%s] 'properties' is only allowed when type == 'object', got type=%v", toolName, path, node["type"])
	}

	if _, hasItems := node["items"]; hasItems {
		assert.True(t, hasType && nodeType == "array", "[%s%s] 'items' is only allowed when type == 'array', got type=%v", toolName, path, node["type"])
	}

	if anyOf, hasAnyOf := node["anyOf"].([]any); hasAnyOf {
		for i, sub := range anyOf {
			if subMap, ok := sub.(map[string]any); ok {
				validateSchemaNode(t, toolName, fmt.Sprintf("%s.anyOf[%d]", path, i), subMap)
			}
		}
	}

	if props, ok := node["properties"].(map[string]any); ok {
		for propName, propVal := range props {
			if propMap, ok := propVal.(map[string]any); ok {
				validateSchemaNode(t, toolName, fmt.Sprintf("%s.%s", path, propName), propMap)
			}
		}
	}

	if items, ok := node["items"].(map[string]any); ok {
		validateSchemaNode(t, toolName, fmt.Sprintf("%s.items", path), items)
	}
}
