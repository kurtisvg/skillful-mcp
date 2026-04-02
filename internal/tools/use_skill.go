package tools

import (
	"context"
	"encoding/json"

	"skillful-mcp/internal/clientmanager"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type useSkillInput struct {
	SkillName string `json:"skill_name" jsonschema:"name of the skill to inspect"`
}

type skillInfo struct {
	Skill     string          `json:"skill"`
	Tools     []*mcp.Tool     `json:"tools"`
	Resources []*mcp.Resource `json:"resources"`
}

func RegisterUseSkill(s *mcp.Server, mgr *clientmanager.Manager) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "use_skill",
		Description: "List tools and resources available in a specific skill. Use list_skills to list valid skills.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input useSkillInput) (*mcp.CallToolResult, any, error) {
		session, err := mgr.GetSession(input.SkillName)
		if err != nil {
			result := &mcp.CallToolResult{}
			result.SetError(err)
			return result, nil, nil
		}

		toolsResult, err := session.ListTools(ctx, nil)
		if err != nil {
			result := &mcp.CallToolResult{}
			result.SetError(err)
			return result, nil, nil
		}

		info := skillInfo{
			Skill:     input.SkillName,
			Tools:     toolsResult.Tools,
			Resources: []*mcp.Resource{},
		}

		// Resources are optional — some servers don't support them.
		resourcesResult, err := session.ListResources(ctx, nil)
		if err == nil && resourcesResult != nil {
			info.Resources = resourcesResult.Resources
		}

		data, err := json.Marshal(info)
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
