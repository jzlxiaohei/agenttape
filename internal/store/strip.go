package store

import "agenttape/internal/normalize"

// stripBlockRaw clears ContentBlock.Raw throughout an envelope before it is
// persisted. The full original payload is on disk (design choice C), so keeping a
// per-block copy in normalized_json would be redundant and bloat the DB. The
// record is discarded after the write, so mutating in place is safe.
func stripBlockRaw(env *normalize.NormalizedEnvelope) {
	if env.Request != nil {
		stripBlocks(env.Request.System)
		stripMessages(env.Request.Messages)
	}
	if env.Response != nil {
		stripMessages(env.Response.Output)
	}
}

func stripMessages(msgs []normalize.Message) {
	for i := range msgs {
		stripBlocks(msgs[i].Content)
	}
}

func stripBlocks(blocks []normalize.ContentBlock) {
	for i := range blocks {
		blocks[i].Raw = nil
		if blocks[i].ToolResult != nil {
			stripBlocks(blocks[i].ToolResult.Content)
		}
	}
}
