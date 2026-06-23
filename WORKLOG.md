# tracelab — work log & notes

A running summary of what's built, the key decisions, and the problems found +
fixed. Newest phase on top. (Architecture lives in `CONVENTIONS.md` and the
`.claude/skills/`; this file is the narrative + gotchas.)

---

## Phase: codex hooks → flow graph → routing → diff → replay → launch → replay lib

### What's delivered

**Capture**
- HTTP reverse-proxy capture (cc + codex) and **harness hooks** for both runtimes.
  - cc hooks via `--settings`; **codex hooks** via `-c hooks.<E>=[...]` (TOML) +
    `--dangerously-bypass-hook-trust` (per-invocation, no `~/.codex/config.toml`
    write). codex events carry `tool_use_id` == responses `call_id`.
  - `tool_name` extracted from hook payloads → shown on flow cards.
  - Hooks are **receipt-timestamped** (RFC3339Nano, same as HTTP) so they
    interleave on the timeline.

**Viewer (React, MVVM)**
- **Flow graph (hook-first)**: a turn's spine is the hook timeline; each hook
  card shows its payload inline (folded by default, key fields surfaced). HTTP is
  not a first-layer node — a `request #N` chip opens the exchange in a **side
  sheet**. Correlation is structural (no jumping): hook→request by strict causal
  ordering (tool hook → producing completion; UserPromptSubmit → triggered
  request), `is_completion` only.
- **Context diff in flow**: a ⇄ on the request chip opens the Diff tab; semantic
  diff shows a one-line change classification (+N tool results / system changed /
  suspected compaction).
- **URL routing**: navigation state lives in the URL (`/sessions/:id?tab=…&
  req_id=…&turn_id=…`), shareable + reload-safe; Go viewer has a real SPA
  fallback for deep links.
- **Replay**: re-send a captured completion (optionally edited body, CodeMirror
  JSON editor) to upstream, normalize, compare original vs replay. Two-step
  confirm; not persisted.

**Launch page (`/launch`)** — opt-in
- Start cc/codex through the proxy in a chosen terminal app (auto-detected:
  Terminal/iTerm/Ghostty/…). Subscription or API-key mode.
- Always shows a copy-paste "run it yourself" command.

**Replay library (`/cases`)** — eval groundwork
- `replay_cases` table + predefined "你是谁" seeds (one per wire format) + "save
  as case" from captures. Run a case against a chosen **live session** (supplies
  credentials), editable body, reuses the replay pipeline. Not persisted.

### Key design decisions
- **Credentials in process memory only, never on disk.** Captured auth headers
  and API-key-mode inject auth live in `Sessions` (memory); they die with the
  process. Consequence: only sessions captured/registered in the *current* serve
  process are replayable (others → 409).
- **API-key launch = proxy-inject.** The agent gets a placeholder key; the proxy
  swaps in the real key (held in memory) on forward — key never reaches the
  agent/terminal/disk. The copy-paste manual command keeps the key in the user's
  own shell instead.
- **Replay reuses capture's normalize pipeline** — resend verbatim bytes, build a
  SourceEvent, run the same registry. No provider-specific replay code.
- **Correlation is exact, not heuristic** — based on the fixed tool-call
  lifecycle ordering, filtered to real completions.
- **Replay library is "grown", not separate** — a case is provider-neutral
  request material; a session supplies credentials at run time.

### Security model (Launch)
- Server-launch is gated by `-allow-launch`, **ON by default**; pass
  `-allow-launch=false` to disable, after which the server executes nothing and
  the page only shows the command to run yourself. (Still origin-checked +
  localhost-bound below.)
- **Cross-origin POSTs rejected** (Origin check) — blocks CSRF / DNS-rebinding
  from triggering local exec.
- Working dir is validated before launch (clear 400, no silent `cd` failure).

### Problems found & fixed
- Hook events had **empty `started_at`** → all hooks sorted before all HTTP and
  correlation computed null. Fix: stamp receipt time + idempotent migration
  backfilling old rows from `created_at`.
- Correlation initially included **control/probe requests** (`is_completion=
  false`) → hooks linked to a probe. Fix: only real completions are targets.
- A stray `./tracelab serve` (no `-data`) squatted :8787 with an empty dir →
  looked like data loss; data was intact. (Process kills are denied to the agent;
  the good instance runs on **:8788**.)
- Editing JSON in a `<textarea>` was poor → CodeMirror `CodeEditor`.

### Pending / next
- **codex desktop launch** — DONE. `/api/codex-desktop/{status,install,restore}`:
  backs up `~/.codex/config.toml` verbatim, writes proxy routing (+ optional hook
  capture) via `launcher.MergeCodexDesktopConfig`, restores byte-exact. Gated on
  `-allow-launch`; subscription only (chatgpt backend upstream); conflict-confirm
  via a `.tracelab-desktop-state.json`. Caveat: hooks need a one-time in-app
  `/hooks` trust (no persistent bypass exists; desktop can't pass the flag).
  **Not verifiable headless** — needs a real Mac + Codex desktop to confirm.
- **Replay library → eval**: foundation is in place (`executeCaseThroughSession`
  + the normalized envelope's final_text / tool_calls / usage / stop_reason /
  signals are ready-made assertion material), but **actively shelved**. Building
  it out (assertions, batch runs, compare matrix, scoring, 降智/regression
  history) would turn tracelab into a maintained eval/automation platform; the
  intent is to stay a **personal research workbench**. Deferred by choice, not
  blocked — revisit only if that positioning changes.
- Minor: `Session` type lives in `httpcap` though it's cross-adapter (low-pri
  refactor — move to `internal/source`).

### How to run / test
```bash
# build
env -u GOROOT go build -o ./tracelab ./cmd/tracelab
(cd frontend && npm run build)

# serve (click-to-launch is ON by default; pass -allow-launch=false to disable)
./tracelab serve -data /tmp/tldata2 -listen 127.0.0.1:8788 -viewer ./frontend/dist
# viewer: http://127.0.0.1:8788/viewer/

# launch a session (CLI, or the /launch page):
./tracelab launch -kind codex -server http://127.0.0.1:8788

# replay lib test: /viewer/cases → 你是谁 → pick the live session → run (billed)
```
Notes: macOS only for web-launch; replay/cases make **real billed** upstream
calls; only sessions live in the current serve process are replayable.
