# Contributing to Rancher MCP Server

Thank you for your interest in contributing to the Rancher AI MCP Server! This guide will help you get started with development and understand our contribution process.

## Getting Started

### Prerequisites

- Go 1.24 or later
- Access to a Kubernetes cluster (for testing)
- Basic understanding of Kubernetes and the Model Context Protocol (MCP)

### Setting Up Your Development Environment

1. **Fork and Clone the Repository**

```bash
git clone https://github.com/<your-username>/rancher-ai-mcp.git
cd rancher-ai-mcp
```

2. **Install Dependencies**

```bash
go mod download
```

3. **Run Tests**

```bash
go test -v -cover ./...
```

4. **Build the Project**

```bash
go build -o mcp-server .
```

### Project Structure

```
pkg/
├── client/         # Kubernetes client wrapper - use it to fetch/update/create resources in Kubernetes clusters
├── toolsets/       # Tool collections
│   ├── core/      # Core Kubernetes tools
│   ├── security/  # Security-related tools (example)
│   └── ...        # Other domain-specific toolsets
├── response/       # Response formatting utilities
└── converter/      # Data transformation utilities
```

### Adding a New Toolset

1. Create a new directory under `pkg/toolsets/` (e.g., `pkg/toolsets/security/`)
2. Implement the `toolsAdder` interface:

```go
type toolsAdder interface {
    AddTools(mcpServer *mcp.Server)
}
```

3. Create a `tools.go` file with your tool implementations:

```go
package security

import (
    "mcp/pkg/client"
    "github.com/modelcontextprotocol/go-sdk/mcp"
)

type Tools struct {
    client *client.Client
}

func NewTools(client *client.Client) *Tools {
    return &Tools{client: client}
}

func (t *Tools) AddTools(mcpServer *mcp.Server) {
    mcp.AddTool(mcpServer, &mcp.Tool{
        Name: "scanForVulnerabilities",
        Description: "Scan cluster for security vulnerabilities",
        Meta: map[string]any{"toolset": "security"}, // make sure this is unique for this toolset
        Handler: t.handleVulnerabilityScan,
    })
    // Add more tools here...
}
```
4. Make sure the toolset annotation is unique
5. Add comprehensive tests in `tools_test.go`
6. Update `pkg/toolsets/toolsets.go` to include your new toolset
7. Update documentation in README.md

## Reporting Issues

When reporting bugs or requesting features:

1. **Search Existing Issues** - Check if it's already reported
2. **Use Issue Templates** - Follow the provided templates
3. **Provide Details**:
   - Clear description of the issue or feature
   - Steps to reproduce (for bugs)
   - Expected vs actual behavior
   - Environment details (Go version, Rancher version, Kubernetes version, etc.)
   - Relevant logs or error messages
