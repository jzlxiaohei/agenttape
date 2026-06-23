package anthropic

import (
	"tracelab/internal/normalize"
)

// deriveSignals emits structural tag hints we can read from the typed structure
// (tool calls, reasoning) — high-confidence facts only.
//
// Compaction is deliberately NOT signaled here: it can't be judged from a single
// request (keyword-matching the injected /compact prompt false-positives on any
// conversation that merely discusses compaction). It's a cross-event judgment —
// see internal/server/compaction.go (graded episodes), decided once the request
// after the boundary has arrived.
func deriveSignals(env *normalize.NormalizedEnvelope) []normalize.TagSignal {
	var sigs []normalize.TagSignal

	if hasToolActivity(env) {
		sigs = append(sigs, normalize.TagSignal{Tag: "tool_call", Confidence: 1, Evidence: "typed tool_use/tool_result blocks present"})
	}
	if hasReasoning(env) {
		sigs = append(sigs, normalize.TagSignal{Tag: "reasoning", Confidence: 1, Evidence: "typed thinking blocks present"})
	}
	return sigs
}

func hasToolActivity(env *normalize.NormalizedEnvelope) bool {
	if env.Response != nil && len(env.Response.ToolCalls) > 0 {
		return true
	}
	if env.Request != nil {
		for _, m := range env.Request.Messages {
			for _, b := range m.Content {
				if b.Type == normalize.BlockToolCall || b.Type == normalize.BlockToolResult {
					return true
				}
			}
		}
	}
	return false
}

func hasReasoning(env *normalize.NormalizedEnvelope) bool {
	check := func(msgs []normalize.Message) bool {
		for _, m := range msgs {
			for _, b := range m.Content {
				if b.Type == normalize.BlockReasoning {
					return true
				}
			}
		}
		return false
	}
	if env.Response != nil && check(env.Response.Output) {
		return true
	}
	return env.Request != nil && check(env.Request.Messages)
}
