package server

import (
	"log/slog"

	"github.com/SergeyKo17/jot/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Server struct {
	mcp *mcp.Server
}

func New(log *slog.Logger, tools *tools.Tools) *Server {
	srv := mcp.NewServer(&mcp.Implementation{
		Name: "jot", Version: "0.1.0",
	}, nil)

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "remember",
		Description: "Store a fact in long-term memory",
	}, tools.HandleRemember)

	return &Server{mcp: srv}
}
