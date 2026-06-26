// Package openai normalizes OpenAI-family exchanges. It hosts two independent
// normalizers — the Responses API (used by Codex) and Chat Completions — which
// share helpers within this package but not with other providers.
package openai

import (
	"encoding/json"

	"github.com/jzlxiaohei/agenttape/internal/normalize"
)

// respItem covers every Responses-API item variant (request input and response
// output use the same shapes). Type drives the mapping; we never guess.
type respItem struct {
	Type    string          `json:"type"`
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
	// function_call / custom_tool_call
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
	Input     json.RawMessage `json:"input"`
	CallID    string          `json:"call_id"`
	// function_call_output / custom_tool_call_output
	Output json.RawMessage `json:"output"`
	// reasoning
	Summary json.RawMessage `json:"summary"`
}

// mapItems converts a list of Responses-API items into neutral messages.
func mapItems(raw json.RawMessage) []normalize.Message {
	var items []respItem
	if json.Unmarshal(raw, &items) != nil {
		return nil
	}
	out := make([]normalize.Message, 0, len(items))
	for _, it := range items {
		if m, ok := mapItem(it); ok {
			out = append(out, m)
		}
	}
	return out
}

func mapItem(it respItem) (normalize.Message, bool) {
	raw, _ := json.Marshal(it)
	switch it.Type {
	case "message", "":
		return normalize.Message{Role: orDefault(it.Role, "user"), Content: mapContent(it.Content)}, true
	case "reasoning":
		return normalize.Message{Role: "assistant", Content: []normalize.ContentBlock{{
			Type: normalize.BlockReasoning, Text: reasoningText(it.Summary), Raw: raw,
		}}}, true
	case "function_call", "custom_tool_call", "tool_search_call":
		args := it.Arguments
		if len(args) == 0 {
			args = it.Input
		}
		return normalize.Message{Role: "assistant", Content: []normalize.ContentBlock{{
			Type:     normalize.BlockToolCall,
			ToolCall: &normalize.ToolCall{ID: it.CallID, Name: orDefault(it.Name, it.Type), Arguments: unquoteJSON(args)},
			Raw:      raw,
		}}}, true
	case "function_call_output", "custom_tool_call_output", "tool_search_output":
		return normalize.Message{Role: "tool", Content: []normalize.ContentBlock{{
			Type:       normalize.BlockToolResult,
			ToolResult: &normalize.ToolResult{ToolCallID: it.CallID, Content: outputBlocks(it.Output)},
			Raw:        raw,
		}}}, true
	default:
		return normalize.Message{Role: orDefault(it.Role, "unknown"), Content: []normalize.ContentBlock{{
			Type: normalize.BlockUnknown, Raw: raw,
		}}}, true
	}
}

// contentPart is one part of a message item's content array.
type contentPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func mapContent(raw json.RawMessage) []normalize.ContentBlock {
	if len(raw) == 0 {
		return nil
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return []normalize.ContentBlock{{Type: normalize.BlockText, Text: s}}
	}
	var parts []contentPart
	if json.Unmarshal(raw, &parts) != nil {
		return []normalize.ContentBlock{{Type: normalize.BlockUnknown, Raw: raw}}
	}
	out := make([]normalize.ContentBlock, 0, len(parts))
	for _, p := range parts {
		switch p.Type {
		case "input_text", "output_text", "text", "summary_text":
			out = append(out, normalize.ContentBlock{Type: normalize.BlockText, Text: p.Text})
		case "input_image", "output_image", "image_url":
			out = append(out, normalize.ContentBlock{Type: normalize.BlockImage})
		default:
			r, _ := json.Marshal(p)
			out = append(out, normalize.ContentBlock{Type: normalize.BlockUnknown, Raw: r})
		}
	}
	return out
}

func reasoningText(summary json.RawMessage) string {
	blocks := mapContent(summary)
	var s string
	for _, b := range blocks {
		s += b.Text
	}
	return s
}

func outputBlocks(out json.RawMessage) []normalize.ContentBlock {
	if len(out) == 0 {
		return nil
	}
	var s string
	if json.Unmarshal(out, &s) == nil {
		return []normalize.ContentBlock{{Type: normalize.BlockText, Text: s}}
	}
	return mapContent(out)
}

// unquoteJSON turns a JSON-encoded string (the Responses API encodes tool
// arguments as a string) back into raw JSON. The result is always valid JSON: if
// the unquoted payload is not itself valid JSON, the original quoted string is
// kept so downstream JSON encoding never breaks. Non-string inputs pass through.
func unquoteJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		if json.Valid([]byte(s)) {
			return json.RawMessage(s)
		}
		return raw // keep the valid quoted-string form
	}
	if json.Valid(raw) {
		return raw
	}
	return nil
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
