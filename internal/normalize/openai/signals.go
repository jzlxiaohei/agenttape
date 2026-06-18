package openai

import "tracelab/internal/normalize"

// deriveSignals emits structural tag hints shared by both OpenAI normalizers.
// Tool calls and reasoning are read from typed structure (facts); we emit no
// keyword-based suspected signals here yet.
func deriveSignals(env *normalize.NormalizedEnvelope) []normalize.TagSignal {
	var sigs []normalize.TagSignal
	if hasTool(env) {
		sigs = append(sigs, normalize.TagSignal{Tag: "tool_call", Confidence: 1, Evidence: "typed function_call/tool_result items present"})
	}
	if hasReasoning(env) {
		sigs = append(sigs, normalize.TagSignal{Tag: "reasoning", Confidence: 1, Evidence: "typed reasoning items present"})
	}
	return sigs
}

func hasTool(env *normalize.NormalizedEnvelope) bool {
	if env.Response != nil && len(env.Response.ToolCalls) > 0 {
		return true
	}
	return anyBlock(env, func(b normalize.ContentBlock) bool {
		return b.Type == normalize.BlockToolCall || b.Type == normalize.BlockToolResult
	})
}

func hasReasoning(env *normalize.NormalizedEnvelope) bool {
	return anyBlock(env, func(b normalize.ContentBlock) bool { return b.Type == normalize.BlockReasoning })
}

func anyBlock(env *normalize.NormalizedEnvelope, pred func(normalize.ContentBlock) bool) bool {
	scan := func(msgs []normalize.Message) bool {
		for _, m := range msgs {
			for _, b := range m.Content {
				if pred(b) {
					return true
				}
			}
		}
		return false
	}
	if env.Request != nil && scan(env.Request.Messages) {
		return true
	}
	return env.Response != nil && scan(env.Response.Output)
}
