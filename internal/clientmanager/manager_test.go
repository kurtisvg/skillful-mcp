package clientmanager

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// startFakeServer creates an in-memory MCP server with one registered tool,
// connects a client to it, and returns the client session.
func startFakeServer(t *testing.T, ctx context.Context, toolName string) *mcp.ClientSession {
	t.Helper()

	server := mcp.NewServer(&mcp.Implementation{Name: "fake-server"}, nil)
	mcp.AddTool(server, &mcp.Tool{
		Name:        toolName,
		Description: "A test tool",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input struct{}) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "ok"}},
		}, nil, nil
	})

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	go func() { _ = server.Run(ctx, serverTransport) }()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect test client: %v", err)
	}

	return session
}

func TestListServerNames(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sessions := map[string]*mcp.ClientSession{
		"bravo":   startFakeServer(t, ctx, "tool_b"),
		"alpha":   startFakeServer(t, ctx, "tool_a"),
		"charlie": startFakeServer(t, ctx, "tool_c"),
	}
	m := NewFromSessions(sessions)
	defer m.Close()

	names := m.ListServerNames()
	if len(names) != 3 {
		t.Fatalf("expected 3 skills, got %d", len(names))
	}
	// Should be sorted alphabetically.
	expected := []string{"alpha", "bravo", "charlie"}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("names[%d] = %q, want %q", i, name, expected[i])
		}
	}
}

func TestGetSessionValid(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	session := startFakeServer(t, ctx, "test_tool")
	m := NewFromSessions(map[string]*mcp.ClientSession{"myskill": session})
	defer m.Close()

	s, err := m.GetSession("myskill")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s == nil {
		t.Fatal("expected non-nil session")
	}
}

func TestGetSessionUnknown(t *testing.T) {
	m := NewFromSessions(map[string]*mcp.ClientSession{})

	_, err := m.GetSession("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown skill")
	}
}

func TestListToolsViaSession(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	session := startFakeServer(t, ctx, "my_test_tool")
	m := NewFromSessions(map[string]*mcp.ClientSession{"testskill": session})
	defer m.Close()

	s, err := m.GetSession("testskill")
	if err != nil {
		t.Fatal(err)
	}

	result, err := s.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools error: %v", err)
	}
	if len(result.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(result.Tools))
	}
	if result.Tools[0].Name != "my_test_tool" {
		t.Errorf("tool name = %q, want 'my_test_tool'", result.Tools[0].Name)
	}
}

func TestCloseIdempotent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	session := startFakeServer(t, ctx, "tool")
	m := NewFromSessions(map[string]*mcp.ClientSession{"s": session})

	// Should not panic on multiple closes.
	m.Close()
	m.Close()
}
