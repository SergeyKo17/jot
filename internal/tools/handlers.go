package tools

import (
	"context"
	"log/slog"

	"github.com/SergeyKo17/jot/internal/store"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type RememberInput struct {
	Text    string `json:"text" jsonschema:"A fact to remember"`
	Project string `json:"project,omitempty"`
}

type RememberOutput struct {
	Status string
}

type Tools struct {
	Store *store.Store
	Log   *slog.Logger
}

func New(store *store.Store, log *slog.Logger) *Tools {
	return &Tools{Store: store, Log: log}
}

func (t *Tools) HandleRemember(ctx context.Context, req *mcp.CallToolRequest, input RememberInput) (*mcp.CallToolResult, RememberOutput, error) {
	// h.Store.Save(...)
	return nil, RememberOutput{Status: "ok"}, nil
}
