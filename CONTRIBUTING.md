# Contributing to jot

Thank you for your interest in contributing to jot!

## Development Environment

Requirements:
- Go 1.26+
- Make

```bash
git clone https://github.com/SergeyKo17/jot.git
cd jot
go mod tidy
go build ./cmd/jot/
```

Run tests and linter:

```bash
make test
make lint
```

Requires [golangci-lint](https://golangci-lint.run/welcome/install/).

## Branch Naming

| Pattern | When |
|---------|------|
| `main` | Stable code, merge only via PR |
| `dev` | Active development branch |
| `feature/<name>` | New feature (`feature/add-recall-tool`) |
| `fix/<name>` | Bug fix (`fix/fts5-trigger-sync`) |
| `chore/<name>` | Project setup, deps, CI (`chore/add-goreleaser`) |
| `docs/<name>` | Documentation changes (`docs/add-install-guide`) |

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
type(scope): short description
```

**Types:**

| Type | When |
|------|------|
| `feat` | New feature |
| `fix` | Bug fix |
| `refactor` | Code change without behavior change |
| `test` | Adding or updating tests |
| `docs` | Documentation |
| `chore` | Build, deps, CI, tooling |

**Scope** is optional, refers to the affected package:

```
feat(store): add FTS5 full-text search
feat(tools): add remember handler
fix(store): fix trigger sync on update
chore(deps): update MCP SDK to v1.7.1
docs: add installation guide
```

**Rules:**
- Use English
- Lowercase after type prefix
- No period at the end
- Imperative mood: "add feature", not "added feature"
- Keep the first line under 72 characters

## Pull Requests

- One feature or fix per PR
- Create a branch from `dev`, open PR back into `dev`
- Provide a short description of what changed and why
- Make sure `make test` and `make lint` pass
- Run `gofmt -w .` before committing

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- No global variables
- Pass dependencies explicitly (no init-time side effects except driver registration)
- Keep packages small and focused
