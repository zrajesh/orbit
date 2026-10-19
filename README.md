# orbit

a generalized sandboxed ai-agent

## Layout

- `internal/core` – provider-neutral types (`Message`, `ContentBlock`, `CompletionRequest`, `Provider`)
- `internal/provider` – LLM adapters; wire types stay private to each adapter (Anthropic today)
- `internal/tools` – `Tool` contract, `Registry`, sandboxed `read_file` and `list_dir`
- `internal/agent` – the agentic loop: budget, typed events, `Session`
- `cmd/orbit` – CLI entry point

## Run

```sh
export ANTHROPIC_API_KEY=...
go run ./cmd/orbit "read go.mod and summarize it"
```

Flags: `-model`, `-root` (sandbox root), `-max-turns`, `-max-tokens`. Ctrl+C cancels the run cleanly and still reports token usage.

## Test

```sh
go test -race ./...
```
