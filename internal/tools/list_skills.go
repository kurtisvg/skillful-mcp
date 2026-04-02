package tools

import (
	"context"
	"encoding/json"

	"skillful-mcp/internal/clientmanager"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type listSkillsInput struct{}

func RegisterListSkills(s *mcp.Server, mgr *clientmanager.Manager) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_skills",
		Description: "List all available skill names",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input listSkillsInput) (*mcp.CallToolResult, any, error) {
		names := mgr.ListServerNames()
		data, err := json.Marshal(names)
		if err != nil {
			result := &mcp.CallToolResult{}
			result.SetError(err)
			return result, nil, nil
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil
	})
}
