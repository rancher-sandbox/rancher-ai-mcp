package toolsets

import (
	"mcp/pkg/client"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllToolSets(t *testing.T) {
	client := client.NewClient(true)
	toolsets := allToolSets(client)

	assert.NotNil(t, toolsets)
	assert.Len(t, toolsets, 1, "should have exactly 1 toolset (core)")
}
