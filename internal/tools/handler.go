package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/Azure/aks-mcp/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
)

// logToolCall logs the start of a tool call
func logToolCall(toolName string, arguments interface{}) {
	// Try to format as JSON for better readability
	if jsonBytes, err := json.Marshal(arguments); err == nil {
		log.Printf("\n>>> [%s] %s", toolName, string(jsonBytes))
	} else {
		log.Printf("\n>>> [%s] %v", toolName, arguments)
	}
}

// logToolResult logs the result or error of a tool call
func logToolResult(result string, err error) {
	if err != nil {
		log.Printf("    ERROR: %v", err)
	} else if len(result) > 500 {
		log.Printf("    Result: %d bytes (truncated): %.500s...", len(result), result)
	} else {
		log.Printf("    Result: %s", result)
	}
}

// CreateToolHandler creates an adapter that converts CommandExecutor to the format expected by MCP server
func CreateToolHandler(executor CommandExecutor, cfg *config.ConfigData) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if cfg.Verbose {
			logToolCall(req.Params.Name, req.Params.Arguments)
		}

		args, ok := req.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("arguments must be a map[string]interface{}, got " + fmt.Sprintf("%T", req.Params.Arguments)), nil
		}
		result, err := executor.Execute(args, cfg)

		if cfg.Verbose {
			logToolResult(result, err)
		}

		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}

// CreateResourceHandler creates an adapter that converts ResourceHandler to the format expected by MCP server
func CreateResourceHandler(handler ResourceHandler, cfg *config.ConfigData) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if cfg.Verbose {
			logToolCall(req.Params.Name, req.Params.Arguments)
		}

		args, ok := req.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("arguments must be a map[string]interface{}, got " + fmt.Sprintf("%T", req.Params.Arguments)), nil
		}
		result, err := handler.Handle(args, cfg)

		if cfg.Verbose {
			logToolResult(result, err)
		}

		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}
