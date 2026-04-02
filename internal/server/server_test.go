package server_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"skillful-mcp/internal/clientmanager"
	"skillful-mcp/internal/server"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// startDownstream creates a fake MCP server with the given tools and resources,
// and returns a connected client session.
func startDownstream(t *testing.T, ctx context.Context, tools []mcp.Tool, resources []mcp.Resource) *mcp.ClientSession {
	t.Helper()

	s := mcp.NewServer(&mcp.Implementation{Name: "downstream"}, nil)
	for _, tool := range tools {
		tool := tool
		mcp.AddTool(s, &tool, func(ctx context.Context, req *mcp.CallToolRequest, input map[string]any) (*mcp.CallToolResult, any, error) {
			// Echo back the tool name and arguments for verification.
			resp := map[string]any{"tool": tool.Name, "args": input}
			data, _ := json.Marshal(resp)
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
			}, nil, nil
		})
	}
	for _, r := range resources {
		r := r
		s.AddResource(&r, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{
					{URI: r.URI, MIMEType: "text/plain", Text: "content of " + r.Name},
				},
			}, nil
		})
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

func TestE2EMultipleSkills(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up two downstream servers with different tools and resources.
	dbSession := startDownstream(t, ctx,
		[]mcp.Tool{
			{Name: "query", Description: "Run a SQL query"},
			{Name: "list_tables", Description: "List database tables"},
		},
		nil,
	)
	fsSession := startDownstream(t, ctx,
		[]mcp.Tool{
			{Name: "read_file", Description: "Read a file"},
		},
		[]mcp.Resource{
			{URI: "file:///tmp/test.txt", Name: "test.txt", Description: "A test file"},
		},
	)

	mgr := clientmanager.NewFromSessions(map[string]*mcp.ClientSession{
		"database":   dbSession,
		"filesystem": fsSession,
	})
	defer mgr.Close()

	// Create the upstream server and connect a test client.
	upstream := server.NewServer(mgr)
	serverT, clientT := mcp.NewInMemoryTransports()
	go func() { _ = upstream.Run(ctx, serverT) }()

	client := mcp.NewClient(&mcp.Implementation{Name: "e2e-client"}, nil)
	session, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatal(err)
	}

	// --- list_skills: should return both skills sorted ---
	t.Run("list_skills", func(t *testing.T) {
		result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "list_skills"})
		if err != nil {
			t.Fatal(err)
		}
		tc := result.Content[0].(*mcp.TextContent)
		var names []string
		if err := json.Unmarshal([]byte(tc.Text), &names); err != nil {
			t.Fatal(err)
		}
		if len(names) != 2 || names[0] != "database" || names[1] != "filesystem" {
			t.Errorf("expected [database, filesystem], got %v", names)
		}
	})

	// --- use_skill database: should list 2 tools, no resources ---
	t.Run("use_skill_database", func(t *testing.T) {
		result, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      "use_skill",
			Arguments: map[string]any{"skill_name": "database"},
		})
		if err != nil {
			t.Fatal(err)
		}
		tc := result.Content[0].(*mcp.TextContent)
		var info struct {
			Skill     string                  `json:"skill"`
			Tools     []struct{ Name string } `json:"tools"`
			Resources []any                   `json:"resources"`
		}
		if err := json.Unmarshal([]byte(tc.Text), &info); err != nil {
			t.Fatal(err)
		}

		if info.Skill != "database" {
			t.Errorf("skill = %q, want database", info.Skill)
		}
		if len(info.Tools) != 2 {
			t.Fatalf("expected 2 tools, got %d", len(info.Tools))
		}
		toolNames := map[string]bool{}
		for _, tool := range info.Tools {
			toolNames[tool.Name] = true
		}
		if !toolNames["query"] || !toolNames["list_tables"] {
			t.Errorf("expected query and list_tables, got %v", info.Tools)
		}
		if len(info.Resources) != 0 {
			t.Errorf("expected 0 resources, got %d", len(info.Resources))
		}
	})

	// --- use_skill filesystem: should list 1 tool and 1 resource ---
	t.Run("use_skill_filesystem", func(t *testing.T) {
		result, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      "use_skill",
			Arguments: map[string]any{"skill_name": "filesystem"},
		})
		if err != nil {
			t.Fatal(err)
		}
		tc := result.Content[0].(*mcp.TextContent)
		var info struct {
			Tools     []struct{ Name string } `json:"tools"`
			Resources []struct{ URI string }  `json:"resources"`
		}
		if err := json.Unmarshal([]byte(tc.Text), &info); err != nil {
			t.Fatal(err)
		}

		if len(info.Tools) != 1 || info.Tools[0].Name != "read_file" {
			t.Errorf("expected [read_file], got %v", info.Tools)
		}
		if len(info.Resources) != 1 || info.Resources[0].URI != "file:///tmp/test.txt" {
			t.Errorf("expected [file:///tmp/test.txt], got %v", info.Resources)
		}
	})

	// --- read_resource from filesystem ---
	t.Run("read_resource", func(t *testing.T) {
		result, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name: "read_resource",
			Arguments: map[string]any{
				"skill_name":   "filesystem",
				"resource_uri": "file:///tmp/test.txt",
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		if result.IsError {
			t.Fatal("tool returned error")
		}
		er := result.Content[0].(*mcp.EmbeddedResource)
		if er.Resource.Text != "content of test.txt" {
			t.Errorf("resource text = %q, want 'content of test.txt'", er.Resource.Text)
		}
	})

	// --- execute_code: basic math ---
	t.Run("execute_code_math", func(t *testing.T) {
		result, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      "execute_code",
			Arguments: map[string]any{"code": "1 + 2 + 3"},
		})
		if err != nil {
			t.Fatal(err)
		}
		tc := result.Content[0].(*mcp.TextContent)
		if tc.Text != "6" {
			t.Errorf("result = %q, want '6'", tc.Text)
		}
	})

	// --- execute_code: call a single downstream tool ---
	t.Run("execute_code_call_tool", func(t *testing.T) {
		code := `call_tool("database", "query", sql="SELECT 1")`
		result, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      "execute_code",
			Arguments: map[string]any{"code": code},
		})
		if err != nil {
			t.Fatal(err)
		}
		if result.IsError {
			tc := result.Content[0].(*mcp.TextContent)
			t.Fatalf("execute_code returned error: %s", tc.Text)
		}
		tc := result.Content[0].(*mcp.TextContent)
		// The downstream echoes back {"tool":"query","args":{"sql":"SELECT 1"}}
		var resp map[string]any
		if err := json.Unmarshal([]byte(tc.Text), &resp); err != nil {
			t.Fatalf("failed to parse response %q: %v", tc.Text, err)
		}
		if resp["tool"] != "query" {
			t.Errorf("tool = %v, want 'query'", resp["tool"])
		}
		args := resp["args"].(map[string]any)
		if args["sql"] != "SELECT 1" {
			t.Errorf("args.sql = %v, want 'SELECT 1'", args["sql"])
		}
	})

	// --- execute_code: call multiple tools and combine results ---
	t.Run("execute_code_multi_tool", func(t *testing.T) {
		code := `
a = call_tool("database", "query", sql="SELECT 1")
b = call_tool("filesystem", "read_file", path="/tmp/test.txt")
a + " | " + b
`
		result, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      "execute_code",
			Arguments: map[string]any{"code": code},
		})
		if err != nil {
			t.Fatal(err)
		}
		if result.IsError {
			tc := result.Content[0].(*mcp.TextContent)
			t.Fatalf("execute_code returned error: %s", tc.Text)
		}
		tc := result.Content[0].(*mcp.TextContent)
		// Both calls return JSON, joined by " | "
		if tc.Text == "" {
			t.Error("expected non-empty result from multi-tool code")
		}
		// Verify both tool responses are present
		var resp1, resp2 map[string]any
		parts := splitOnce(tc.Text, " | ")
		if len(parts) != 2 {
			t.Fatalf("expected 2 parts separated by ' | ', got %q", tc.Text)
		}
		if err := json.Unmarshal([]byte(parts[0]), &resp1); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal([]byte(parts[1]), &resp2); err != nil {
			t.Fatal(err)
		}
		if resp1["tool"] != "query" {
			t.Errorf("first tool = %v, want 'query'", resp1["tool"])
		}
		if resp2["tool"] != "read_file" {
			t.Errorf("second tool = %v, want 'read_file'", resp2["tool"])
		}
	})

	// --- execute_code: call_tool with unknown skill returns error string ---
	t.Run("execute_code_unknown_skill", func(t *testing.T) {
		code := `call_tool("nonexistent", "foo")`
		result, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      "execute_code",
			Arguments: map[string]any{"code": code},
		})
		if err != nil {
			t.Fatal(err)
		}
		tc := result.Content[0].(*mcp.TextContent)
		if tc.Text == "" {
			t.Error("expected error message in result")
		}
	})

	// --- use_skill with unknown skill returns error ---
	t.Run("use_skill_unknown", func(t *testing.T) {
		result, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      "use_skill",
			Arguments: map[string]any{"skill_name": "nonexistent"},
		})
		if err != nil {
			t.Fatal(err)
		}
		if !result.IsError {
			t.Error("expected error for unknown skill")
		}
	})
}

func splitOnce(s, sep string) []string {
	i := strings.Index(s, sep)
	if i < 0 {
		return []string{s}
	}
	return []string{s[:i], s[i+len(sep):]}
}
