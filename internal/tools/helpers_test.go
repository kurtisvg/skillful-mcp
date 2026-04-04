package tools

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// newFakeSession creates an in-memory MCP server with one tool, connects a
// client, and returns the session. Used by tests across the tools package.
func newFakeSession(t *testing.T, ctx context.Context, configure ...func(*mcp.Server)) *mcp.ClientSession {
	t.Helper()
	s := mcp.NewServer(&mcp.Implementation{Name: "fake"}, nil)
	mcp.AddTool(s, &mcp.Tool{Name: "fake_tool", Description: "A test tool"}, func(ctx context.Context, req *mcp.CallToolRequest, input struct{}) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil, nil
	})
	for _, fn := range configure {
		fn(s)
	}
	serverT, clientT := mcp.NewInMemoryTransports()
	go func() { _ = s.Run(ctx, serverT) }()
	client := mcp.NewClient(&mcp.Implementation{Name: "test"}, nil)
	session, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatal(err)
	}
	return session
}
