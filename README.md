# jot

Lightweight MCP memory server. 4 tools, single binary, zero config.

## Why

Most memory servers use vector search — they run a separate embedding model to understand meaning.
jot takes a different approach: let the LLM do the thinking.

An LLM already understands context, synonyms, and intent. Instead of embedding text into vectors,
jot stores plain text in SQLite with full-text search (FTS5) and relies on the LLM to form
effective queries. Detailed tool descriptions guide the LLM on when and how to search,
what to store, and how to structure data.

The result: a memory server with no external dependencies, no API keys, no Docker,
no Python — just one binary.

## Features

| Tool | Description |
|------|-------------|
| `remember` | Store a fact with project scope and tags |
| `recall` | Full-text search with BM25 ranking, project/tag filters |
| `forget` | Delete a fact by ID |
| `update` | Partial update — change text, project, or tags |

- **FTS5 with OR matching** — each word matched independently, ranked by relevance
- **Project scoping** — separate memories by project
- **Tag filtering** — categorize facts (architecture, decision, team, bug, ...)
- **Guided tool descriptions** — LLM knows when to call each tool, how to form queries, what to store

## Quick start

### Build

```bash
go build -o jot ./cmd/jot/
```

### Setup

Add to your MCP client config (Claude Desktop, Claude Code, Cursor, etc.):

```json
{
  "mcpServers": {
    "jot": {
      "command": "/path/to/jot"
    }
  }
}
```

Restart the client. jot creates its database automatically on first run.

## Usage

jot is registered as an MCP server but does not activate automatically —
this is intentional during the testing phase so you can choose
which chats use jot and which don't.

To activate, just say "use jot" in the chat. The LLM will read
the tool descriptions and handle the rest — storing facts, searching,
updating. No further prompting needed.

To enable jot in all chats, add "use jot for memory" to your system prompt.

## How it works

jot stores memories in SQLite with an FTS5 virtual table for full-text search.

When the LLM calls `recall`, each word in the query is matched independently (OR logic).
Results are ranked by BM25 — records matching more words appear first.
Optional `project` and `tag` filters narrow the results.

The key insight: LLMs naturally generate synonyms and translations when forming search queries.
A query like "database migration schema" will find a record about "goose embedded SQL migrations"
because the LLM includes relevant terms. This makes FTS5 + a well-prompted LLM competitive
with vector search — without the infrastructure cost.

## Design decisions

**Why not vector search?** Vector search needs an embedding model — either a paid API (OpenAI)
or a local model (~100MB, Python dependency). This adds cost, latency, and complexity.
jot's thesis: the LLM already in the conversation can do the semantic work by forming good queries.

**Why Go?** Single static binary. No runtime, no package manager, no virtual environment.
Download and run.

**Why SQLite?** Zero setup, single file, portable. FTS5 is built in.
WAL mode for concurrent reads. Goose migrations embedded via `go:embed`.

**Why 4 tools?** Fewer tools = less LLM confusion, fewer tokens in system prompt.
Competitors expose 9-20 tools. jot covers the full CRUD cycle with the minimum surface.

## Roadmap

- [ ] Prompt tuning: refine tool descriptions for higher recall accuracy, better tag usage, and fewer wasted calls
- [ ] Benchmark: compare recall quality with alternative memory servers and built-in memory
- [ ] Keywords field: LLM-generated synonyms stored alongside text for better search recall
- [ ] BM25 score threshold: filter low-relevance results as memory grows
- [ ] Batch operations: delete/update by project or tag
- [ ] Logging: structured request/response log with rotation

## License

[MIT](https://github.com/SergeyKo17/jot/blob/dev/LICENSE)
