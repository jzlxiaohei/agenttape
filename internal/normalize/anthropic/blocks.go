package anthropic

import (
	"encoding/json"

	"github.com/jzlxiaohei/agenttape/internal/normalize"
)

// wireBlock covers every Anthropic content block variant. Type is read straight
// from the JSON — never inferred from keywords (see CONVENTIONS.md §4).
type wireBlock struct {
	Type string `json:"type"`
	// text
	Text string `json:"text"`
	// thinking / redacted_thinking
	Thinking string `json:"thinking"`
	// tool_use
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
	// tool_result
	ToolUseID string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content"`
	IsError   bool            `json:"is_error"`
}

// parseBlocks maps an Anthropic content value (string or []block) to neutral
// content blocks, preserving the original block in Raw.
func parseBlocks(raw json.RawMessage) []normalize.ContentBlock {
	if len(raw) == 0 {
		return nil
	}
	// content may be a bare string (shorthand for one text block).
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return []normalize.ContentBlock{{Type: normalize.BlockText, Text: s}}
	}
	var blocks []wireBlock
	if json.Unmarshal(raw, &blocks) != nil {
		return []normalize.ContentBlock{{Type: normalize.BlockUnknown, Raw: raw}}
	}
	out := make([]normalize.ContentBlock, 0, len(blocks))
	for _, b := range blocks {
		out = append(out, mapBlock(b))
	}
	return out
}

func mapBlock(b wireBlock) normalize.ContentBlock {
	raw, _ := json.Marshal(b)
	cb := normalize.ContentBlock{Raw: raw}
	switch b.Type {
	case "text":
		cb.Type = normalize.BlockText
		cb.Text = b.Text
	case "thinking", "redacted_thinking":
		cb.Type = normalize.BlockReasoning
		cb.Text = b.Thinking
	case "tool_use":
		cb.Type = normalize.BlockToolCall
		cb.ToolCall = &normalize.ToolCall{ID: b.ID, Name: b.Name, Arguments: b.Input}
	case "tool_result":
		cb.Type = normalize.BlockToolResult
		cb.ToolResult = &normalize.ToolResult{
			ToolCallID: b.ToolUseID,
			IsError:    b.IsError,
			Content:    parseBlocks(b.Content),
		}
	case "image":
		cb.Type = normalize.BlockImage
	default:
		cb.Type = normalize.BlockUnknown
	}
	return cb
}
