// Package tools implements MCP tool handlers.
package tools

import (
	"context"
	"errors"
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

// RecallInput is the input for the recall tool.
type RecallInput struct {
	Text    string `json:"text" jsonschema:"Search query for finding memories"`
	Project string `json:"project,omitempty" jsonschema:"Filter by project scope"`
	Tag     string `json:"tag,omitempty" jsonschema:"Filter by tag"`
}

// RecallOutput is the result of searching memories.
type RecallOutput struct {
	Memories []store.MemoryRow `json:"memories"`
}

// HandleRecall searches memories by text with optional filters.
func (t *Tools) HandleRecall(ctx context.Context, _ *mcp.CallToolRequest, input RecallInput) (*mcp.CallToolResult, RecallOutput, error) {
	if input.Text == "" {
		return toolError("text is required"), RecallOutput{}, nil
	}

	rows, err := t.store.Search(ctx, store.SearchParams{
		Text:    input.Text,
		Project: input.Project,
		Tag:     input.Tag,
	})
	if err != nil {
		t.log.Error("search memories", "err", err)
		return toolError("internal error: could not search memories"), RecallOutput{}, nil
	}

	return nil, RecallOutput{Memories: rows}, nil
}

// ForgetInput is the input for the forget tool.
type ForgetInput struct {
	ID int64 `json:"id" jsonschema:"ID of the memory to delete"`
}

// ForgetOutput is the result of deleting a memory.
type ForgetOutput struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
}

// HandleForget deletes a memory by ID.
func (t *Tools) HandleForget(ctx context.Context, _ *mcp.CallToolRequest, input ForgetInput) (*mcp.CallToolResult, ForgetOutput, error) {
	if input.ID <= 0 {
		return toolError("id is required"), ForgetOutput{}, nil
	}

	err := t.store.Delete(ctx, input.ID)
	if err != nil {
		if errors.Is(err, store.ErrMemoryNotFound) {
			return toolError("memory not found"), ForgetOutput{}, nil
		}
		t.log.Error("delete memory", "err", err)
		return toolError("internal error: could not delete memory"), ForgetOutput{}, nil
	}

	return nil, ForgetOutput{ID: input.ID, Status: "deleted"}, nil
}
