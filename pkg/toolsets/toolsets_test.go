package toolsets

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewToolSetsWithAllTools(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "v1.0.0",
	}, nil)
	assert.NotNil(t, server)

	toolsets := NewToolSetsWithAllTools(server)

	assert.NotNil(t, toolsets)
	assert.Equal(t, 1, len(toolsets.toolsAdders))
}

func TestAddTools(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "v1.0.0",
	}, nil)
	require.NotNil(t, server)

	// Create mock toolsAdders
	mock1 := &mockToolsAdder{}
	mock2 := &mockToolsAdder{}

	toolsets := &ToolSets{
		toolsAdders: []toolsAdder{mock1, mock2},
	}

	// Call AddTools
	toolsets.AddTools(server)

	// Verify that AddTools was called on all adders
	assert.True(t, mock1.addToolsCalled, "AddTools should be called on first mock")
	assert.True(t, mock2.addToolsCalled, "AddTools should be called on second mock")
	assert.Equal(t, server, mock1.server, "Server should be passed to first mock")
	assert.Equal(t, server, mock2.server, "Server should be passed to second mock")
}

// mockToolsAdder is a mock implementation of the toolsAdder interface for testing
type mockToolsAdder struct {
	addToolsCalled bool
	server         *mcp.Server
}

func (m *mockToolsAdder) AddTools(mcpServer *mcp.Server) {
	m.addToolsCalled = true
	m.server = mcpServer
}
