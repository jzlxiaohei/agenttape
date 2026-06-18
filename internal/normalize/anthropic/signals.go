package anthropic

import (
	"slices"
	"strings"

	"tracelab/internal/normalize"
)

// deriveSignals emits structural tag hints. Things we can read from the typed
// structure (tool calls, reasoning) are high-confidence facts. Things we can
// only guess (compaction) are marked Suspected so the UI shows them as 疑似.
func deriveSignals(env *normalize.NormalizedEnvelope) []normalize.TagSignal {
	var sigs []normalize.TagSignal

	if hasToolActivity(env) {
		sigs = append(sigs, normalize.TagSignal{Tag: "tool_call", Confidence: 1, Evidence: "typed tool_use/tool_result blocks present"})
	}
	if hasReasoning(env) {
		sigs = append(sigs, normalize.TagSignal{Tag: "reasoning", Confidence: 1, Evidence: "typed thinking blocks present"})
	}
	if ev, ok := compactionMarker(env); ok {
		sigs = append(sigs, normalize.TagSignal{Tag: "compaction", Confidence: 0.5, Suspected: true, Evidence: ev})
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

// compactionMarker is a deliberately conservative, documented heuristic. It is
// keyword-based and therefore only ever yields a SUSPECTED signal — never a
// fact. See next.md 3.3.
var compactionMarkers = []string{
	"This session is being continued from a previous conversation",
	"conversation is summarized",
}

func compactionMarker(env *normalize.NormalizedEnvelope) (string, bool) {
	if env.Request == nil {
		return "", false
	}
	for _, m := range env.Request.Messages {
		for _, b := range m.Content {
			if b.Type != normalize.BlockText {
				continue
			}
			if i := slices.IndexFunc(compactionMarkers, func(m string) bool { return strings.Contains(b.Text, m) }); i >= 0 {
				return "matched marker: " + compactionMarkers[i], true
			}
		}
	}
	return "", false
}
