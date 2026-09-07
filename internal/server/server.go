// Package server sets up the MCP server.
package server

import (
	"context"
	"log/slog"

	"github.com/SergeyKo17/jot/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server wraps the MCP server.
type Server struct {
	mcp *mcp.Server
}

// New builds and returns mcp server and tools.
func New(_ *slog.Logger, tools *tools.Tools) *Server {
	srv := mcp.NewServer(&mcp.Implementation{
		Name: "jot", Version: "0.1.0",
	}, nil)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "remember",
		Description: "Store a factual statement in long-term memory. Call this when the user shares a decision, preference, architectural choice, or any fact worth remembering across sessions. One fact per call.",
	}, tools.HandleRemember)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "recall",
		Description: "Search memories by text. Optionally filter by project or tag. Returns up to 10 most relevant facts.",
	}, tools.HandleRecall)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "forget",
		Description: "Delete a memory by ID. Use when a fact is outdated, wrong, or no longer relevant.",
	}, tools.HandleForget)

	return &Server{mcp: srv}
}

// Run starts mcp server.
func (s *Server) Run(ctx context.Context) error {
	return s.mcp.Run(ctx, &mcp.StdioTransport{})
}
