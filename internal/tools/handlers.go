// Package tools implements MCP tool handlers.
package tools

import (
	"context"
	"log/slog"

	"github.com/SergeyKo17/jot/internal/store"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RememberInput is the input for the remember tool.
type RememberInput struct {
	Text    string   `json:"text" jsonschema:"A fact to remember"`
	Project string   `json:"project,omitempty" jsonschema:"Project scope, e.g. repo name"`
	Tags    []string `json:"tags,omitempty" jsonschema:"Categories like architecture, decision, preference"`
}

// RememberOutput is the result of storing a memory.
type RememberOutput struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
}

// Tools handles MCP tool requests.
type Tools struct {
	store *store.Store
	log   *slog.Logger
}

// New creates a new Tools instance.
func New(store *store.Store, log *slog.Logger) *Tools {
	return &Tools{store: store, log: log}
}

// HandleRemember stores a fact in memory.
func (t *Tools) HandleRemember(ctx context.Context, call *mcp.CallToolRequest, input RememberInput) (*mcp.CallToolResult, RememberOutput, error) {
	if input.Text == "" {
		return toolError("text is required"), RememberOutput{}, nil
	}

	var source string
	if info := call.ClientInfo(); info != nil {
		source = info.Name
	}

	id, err := t.store.Save(ctx, store.Memory{
		Text:    input.Text,
		Project: input.Project,
		Tags:    input.Tags,
		Source:  source,
	})
	if err != nil {
		t.log.Error("save memory", "err", err)
		return toolError("internal error: could not save memory"), RememberOutput{}, nil
	}

	return nil, RememberOutput{ID: id, Status: "created"}, nil
}

func toolError(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}
}
