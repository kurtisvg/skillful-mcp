package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"skillful-mcp/internal/clientmanager"

	monty "github.com/ewhauser/gomonty"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type executeCodeInput struct {
	Code string `json:"code" jsonschema:"python code that orchestrates tool calls via call_tool(skill_name, tool_name, **kwargs) and returns a computed result"`
}

const executeCodeDescription = `Execute Python code in a secure sandbox to orchestrate multiple tool calls and return a computed result.

The code has access to a built-in function:
  call_tool(skill_name: str, tool_name: str, **kwargs) -> str

call_tool sends kwargs as the tool's input arguments and returns the text result.

IMPORTANT: Only call tools that were returned by use_skill or described in resources. Do not guess tool names or schemas — first call use_skill to discover the available tools and their input schemas for a given skill, then write code that calls those tools.`

func RegisterExecuteCode(s *mcp.Server, mgr *clientmanager.Manager) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "execute_code",
		Description: executeCodeDescription,
	}, func(ctx context.Context, req *mcp.CallToolRequest, input executeCodeInput) (*mcp.CallToolResult, any, error) {
		if input.Code == "" {
			result := &mcp.CallToolResult{}
			result.SetError(fmt.Errorf("code must not be empty"))
			return result, nil, nil
		}

		runner, err := monty.New(input.Code, monty.CompileOptions{ScriptName: "script.py"})
		if err != nil {
			result := &mcp.CallToolResult{}
			result.SetError(fmt.Errorf("compile error: %w", err))
			return result, nil, nil
		}

		value, err := runner.Run(ctx, monty.RunOptions{
			Functions: map[string]monty.ExternalFunction{
				"call_tool": makeCallToolFn(ctx, mgr),
			},
		})
		if err != nil {
			result := &mcp.CallToolResult{}
			result.SetError(fmt.Errorf("runtime error: %w", err))
			return result, nil, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: value.String()}},
		}, nil, nil
	})
}

// makeCallToolFn creates a Monty external function that proxies tool calls
// to downstream MCP servers.
//
// Python signature: call_tool(skill_name: str, tool_name: str, **kwargs) -> str
func makeCallToolFn(ctx context.Context, mgr *clientmanager.Manager) monty.ExternalFunction {
	return func(_ context.Context, call monty.Call) (monty.Result, error) {
		if len(call.Args) < 2 {
			return monty.Return(monty.String("error: call_tool requires skill_name and tool_name as positional arguments")), nil
		}

		skillName, ok := call.Args[0].Raw().(string)
		if !ok {
			return monty.Return(monty.String("error: skill_name must be a string")), nil
		}
		toolName, ok := call.Args[1].Raw().(string)
		if !ok {
			return monty.Return(monty.String("error: tool_name must be a string")), nil
		}

		session, err := mgr.GetSession(skillName)
		if err != nil {
			return monty.Return(monty.String(fmt.Sprintf("error: %v", err))), nil
		}

		// Convert kwargs to tool arguments.
		args := make(map[string]any)
		for _, pair := range call.Kwargs {
			key, ok := pair.Key.Raw().(string)
			if !ok {
				continue
			}
			args[key] = montyValueToAny(pair.Value)
		}

		result, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		})
		if err != nil {
			return monty.Return(monty.String(fmt.Sprintf("error: %v", err))), nil
		}

		if result.IsError {
			text := extractText(result)
			return monty.Return(monty.String(fmt.Sprintf("error: %s", text))), nil
		}

		return monty.Return(monty.String(extractText(result))), nil
	}
}

// montyValueToAny converts a Monty Value to a Go value suitable for JSON tool arguments.
func montyValueToAny(v monty.Value) any {
	raw := v.Raw()
	switch val := raw.(type) {
	case int64:
		return val
	case float64:
		return val
	case string:
		return val
	case bool:
		return val
	case monty.Dict:
		m := make(map[string]any, len(val))
		for _, pair := range val {
			if key, ok := pair.Key.Raw().(string); ok {
				m[key] = montyValueToAny(pair.Value)
			}
		}
		return m
	case []monty.Value:
		list := make([]any, len(val))
		for i, item := range val {
			list[i] = montyValueToAny(item)
		}
		return list
	default:
		// Fall back to JSON marshaling the Value.
		data, err := json.Marshal(v)
		if err != nil {
			return v.String()
		}
		return json.RawMessage(data)
	}
}

// extractText pulls the first text content from a CallToolResult.
func extractText(result *mcp.CallToolResult) string {
	for _, c := range result.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			return tc.Text
		}
	}
	return ""
}
