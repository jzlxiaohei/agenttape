# tracelab deeper viewer direction

This note captures the next product direction for `tracelab`: move beyond a
trace viewer into an agent-session debugger.

The core question should be:

> Why did the agent do this turn, what did it cost, and did it move the task forward?

`claude-tap` is strong as a broad local trace viewer. `tracelab` should
differentiate by going deeper on agent behavior: correlating HTTP traffic,
harness hooks, context evolution, tool lifecycles, compaction, and subagents.

## Current strengths

The current viewer is already deeper than a request list in several ways:

- Session-level turns split by `UserPromptSubmit` hooks.
- Request-level rounds split by structural `role=user` message boundaries.
- Semantic request diff based on normalized message sequences.
- Token composition for `system`, `tools`, and `messages`.
- Cross-links from normalized `tool_call` blocks to harness hook events through
  `tool_call_id`.

These are the right foundations. The next step is to turn them into a coherent
behavior model.

## 1. Causality graph

Represent a session as a chain of causal spans instead of only a list of events.

Example chain:

```text
UserPromptSubmit
  -> LLM request
  -> assistant tool_call
  -> PreToolUse
  -> PostToolUse
  -> tool_result
  -> next LLM request
  -> final response
```

Useful span types:

- `turn`
- `llm_request`
- `tool_call`
- `hook_event`
- `tool_result`
- `compaction`
- `subagent`

The UI does not need a complex graph at first. A practical first version is a
"Why chain" panel for the selected event:

- upstream cause
- current event
- downstream consequences
- linked evidence

## 2. Context evolution ledger

Extend semantic diff into a per-turn context ledger.

For each turn, show:

- messages added
- messages removed
- unchanged messages retained
- compact summaries introduced
- system prompt changes
- tool schema changes
- request parameter changes
- section token deltas for `system`, `tools`, and `messages`

Example summary:

```text
Turn 4
+ 3 messages
- 18 messages
+ 1 compact summary
tools unchanged
messages tokens +12.4%
system tokens unchanged
```

This should answer how context grows, shrinks, gets summarized, or gets
polluted over time.

## 3. Compaction and subagent episodes

Do not treat compaction and subagents as simple tags. Treat them as episodes.

For compaction, show:

- request before compaction
- request after compaction
- removed messages
- introduced summary text
- token delta
- whether tool results survived the compaction

For subagents, show:

- start and stop hook events
- parent turn
- initial context passed to the subagent
- tool calls and token usage inside the subagent
- result returned to the parent agent
- whether the parent retained or ignored the result

This is one of the strongest research-oriented differences from ordinary HTTP
trace viewers.

## 4. Tool lifecycle quality

Build a tool lifecycle view from normalized tool calls, hooks, and following
requests.

For each tool call, track:

- model-emitted `tool_call`
- matching `PreToolUse` hook
- approval or permission outcome
- matching `PostToolUse` hook
- tool result content
- error status
- retry count
- whether the tool result appears in the next LLM request
- approximate context tokens consumed by the tool result

Useful questions:

- Did this tool call produce useful information?
- Was it repeated unnecessarily?
- Did the model ignore the result?
- Did a large tool output dominate the next request?
- How much token cost came from tool output?

## 5. Explain this turn

Add a deterministic "Explain this turn" panel. Avoid LLM-generated summaries at
first; derive the report from stored events and normalized data.

Example output:

```text
User asked:
  Add a round-based viewer.

Request delta:
  + 1 user message
  + 2 retained tool results
  messages +612 approx tokens

Model did:
  reasoning -> Read -> Edit -> Bash

Runtime:
  PreToolUse Read approved
  PostToolUse Read exit 0
  PreToolUse Edit approved
  PostToolUse Edit exit 0

Next request:
  retained Edit result
  retained Bash output

Outcome:
  completion returned final response
```

Every line should link back to evidence in the trace.

## Implementation priority

1. Materialize turns and tool spans in the backend store.
   - Do not rely only on frontend-derived grouping.
   - Keep the frontend grouping as a fallback or development aid.

2. Add per-turn context delta.
   - Reuse normalized request messages.
   - Store added, removed, retained counts.
   - Store section token deltas.

3. Add tool lifecycle records.
   - Link `tool_call_id` across HTTP normalized blocks and hook events.
   - Detect retries and permission failures.
   - Detect whether tool results are retained in the next request.

4. Add compaction episode view.
   - Start with structural signals and hook events.
   - Mark uncertain detections as suspected.

5. Add deterministic turn explanations.
   - Evidence-first.
   - LLM summaries can come later, but should cite concrete event IDs.

## Product positioning

`tracelab` should not compete by supporting the most clients first. It should
compete on depth:

```text
claude-tap  = broad local trace viewer
tracelab    = agent behavior debugger / flight recorder
```

The winning experience is not "I can see the raw request." It is:

> I understand why this agent turn happened, what context changed, which tools
> mattered, where token cost came from, and what evidence supports that answer.
