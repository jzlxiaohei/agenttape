# tracelab

Local-first lab for studying coding-agent ↔ LLM traffic (Claude Code, Codex, …).
A clean rebuild of the `aethertrace` experiment, addressing the issues in
`../aethertrace/next.md`.

> **Status: Module 1 — capture + normalize.** This delivers the foundation only:
> collect exchanges from multiple sources and normalize them into one
> provider-neutral model. Storage/tagging (M2), the React viewer (M3), and the
> material/eval library (M4) build on top later, module by module (per next.md 9.1).

## The one idea: decouple "how data arrives" from "what it means"

```
 source adapters            event.SourceEvent          normalize layer
 ──────────────             ─────────────────          ───────────────
 httpcap (reverse proxy) ┐                         ┌─ anthropic
 hook (harness events)   ├──►  raw, provider-  ──► ├─ openai-responses (Codex)
 (importers, later)      ┘     agnostic facts      └─ openai-chat
                                     │                      │
                                     ▼                      ▼
                                   sink  ◄────  NormalizedEnvelope
                              (JSONL now, DB later)   (typed blocks, usage,
                                                       section %, tag signals)
```

- **`internal/event`** is the stable boundary. It carries raw HTTP/hook facts and
  **no provider semantics**. Adding a source or a provider never touches the other.
- Adapters and providers are independent implementations behind small interfaces;
  they share only atomic helpers (`normalize/shared`). See `CONVENTIONS.md`.

## What Module 1 already does (mapped to next.md)

| next.md | delivered |
| --- | --- |
| 1.1 don't mutate global config | launcher uses env (cc) / `-c` single-run overrides (codex); never writes `~/.codex/config.toml` |
| 1.2 / 7.1 decouple sources; hooks | `httpcap` + `hook` adapters emit the same `SourceEvent` |
| 3.1 concurrent cc/codex sessions | per-session token → `correlation.session_id`; one proxy, isolated sessions |
| 3.3 / 4.1 no keyword/hard-coded parsing | content blocks typed from provider structure; uncertain tags marked `suspected` (疑似) |
| 4.x token usage / section ratio | `usage` + per-section approximate token share (system/tools/messages) |
| 8.2 Go layering | layered packages, independent providers, file-size limits |

## Run it

Build:

```bash
go build -o tracelab ./cmd/tracelab
```

Start the capture service:

```bash
./tracelab serve -listen 127.0.0.1:8787 -out traces.jsonl
```

Launch a client through it (nothing global is modified):

```bash
./tracelab launch -kind cc    -- <claude args>
./tracelab launch -kind codex -- <codex args>
```

Inspect captured + normalized records (provider, model, section token %, signals):

```bash
./tracelab dump traces.jsonl
```

Harness hooks can post to the same pipeline:

```bash
curl -X POST 'http://127.0.0.1:8787/_hook?runtime=claude_code&event=PreToolUse&session=<id>' \
  --data-binary @hook.json
```

## Test

```bash
go test ./...
```

Golden tests run the real captured fixtures in `testdata/` through the
normalizers; `internal/server` exercises the full capture→normalize→sink path
with concurrent cc + codex sessions; `internal/source/httpcap` asserts the
launcher is non-invasive.

## Not in Module 1

DB + tag persistence (M2); React viewer with terminal-style session tabs,
search, jq-style explore, token-ratio charts, replay (M3); material + eval
library (M4). Design notes for those live in `../aethertrace/next.md` and the
approved plan.
