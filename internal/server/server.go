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
		Name: "remember",
		Description: `Store a factual statement in long-term memory. Call this when the user shares a decision, preference, architectural choice, convention, or any fact worth remembering across sessions.

When to store:
- User states a decision ("we chose X over Y because...")
- User describes team structure, roles, responsibilities
- User mentions deadlines, plans, or milestones
- User shares a convention or process ("we do X this way")
- User reports a known bug or limitation

When NOT to store:
- User is asking a question or thinking out loud
- Temporary tasks or debugging details
- Information that will change within hours

How to store:
- One fact per call. Do not combine multiple facts into one entry.
- Always set project to the relevant project name.
- Use tags to categorize: architecture, decision, preference, convention, tooling, team, deadline, bug.
- tags is a JSON array: ["architecture", "decision"], not a comma-separated string.`,
	}, tools.HandleRemember)
	mcp.AddTool(srv, &mcp.Tool{
		Name: "recall",
		Description: `Search long-term memory by text. Returns up to 10 most relevant facts ranked by relevance.

When to search:
- Before answering questions about past decisions, preferences, team, deadlines, or architecture.
- When the user asks about something that could have been discussed in a previous session.
- When the user references a project by name.

How search works: each word in the query is matched independently (OR). A record matching more words ranks higher.

How to search effectively:
- For broad questions, send one query with key terms from the topic.
- For precise lookup, send 2-3 separate recall calls with different angles. Example: instead of "notifications orders delivery NATS architecture", try "notifications delivery", then "NATS async".
- Try searching in both the user's language and English — facts may be stored in either.
- Always pass project when you know which project the question is about.
- Use tag to filter by category (e.g. "architecture", "decision", "team", "bug").`,
	}, tools.HandleRecall)
	mcp.AddTool(srv, &mcp.Tool{
		Name: "forget",
		Description: `Delete a memory by ID. Use when a fact is outdated, wrong, or no longer relevant.

When to use:
- User says something is no longer true or was a mistake.
- A fact has been superseded by a newer decision.
- User explicitly asks to remove a memory.

How to use:
- Always recall first to find the memory and its ID.
- Do not guess IDs — only use IDs returned by recall.`,
	}, tools.HandleForget)
	mcp.AddTool(srv, &mcp.Tool{
		Name: "update",
		Description: `Update a memory by ID. Pass only the fields to change — others stay unchanged.

When to use:
- A fact needs correction but is not entirely wrong.
- Tags or project need to be added or changed.
- User refines a previous decision.

How to use:
- Always recall first to find the memory and its ID.
- Do not guess IDs — only use IDs returned by recall.
- Prefer update over forget+remember — it preserves the original ID and creation date.
- When the user corrects a previously stated fact, update the existing memory instead of creating a new one.`,
	}, tools.HandleUpdate)

	return &Server{mcp: srv}
}

// Run starts mcp server.
func (s *Server) Run(ctx context.Context) error {
	return s.mcp.Run(ctx, &mcp.StdioTransport{})
}
