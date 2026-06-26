package openai

import (
	"encoding/json"

	"agenttape/internal/normalize"
	"agenttape/internal/normalize/shared"
)

// reassembleChatSSE reconstructs a Chat Completions response from its streamed
// delta chunks.
func reassembleChatSSE(body string) (*normalize.ResponseEnvelope, string) {
	var model, finish, text string
	var usage *normalize.Usage
	toolCalls := map[int]*normalize.ToolCall{}
	toolArgs := map[int]string{}
	var toolOrder []int

	for _, ev := range shared.ParseSSE(body) {
		var d struct {
			Model   string `json:"model"`
			Choices []struct {
				Delta struct {
					Content   string          `json:"content"`
					ToolCalls []deltaToolCall `json:"tool_calls"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
			Usage *chatUsage `json:"usage"`
		}
		if json.Unmarshal([]byte(ev.Data), &d) != nil {
			continue
		}
		if d.Model != "" {
			model = d.Model
		}
		if d.Usage != nil {
			usage = d.Usage.toNeutral()
		}
		for _, c := range d.Choices {
			text += c.Delta.Content
			if c.FinishReason != "" {
				finish = c.FinishReason
			}
			for _, tc := range c.Delta.ToolCalls {
				if _, seen := toolCalls[tc.Index]; !seen {
					toolCalls[tc.Index] = &normalize.ToolCall{ID: tc.ID, Name: tc.Function.Name}
					toolOrder = append(toolOrder, tc.Index)
				}
				if tc.ID != "" {
					toolCalls[tc.Index].ID = tc.ID
				}
				if tc.Function.Name != "" {
					toolCalls[tc.Index].Name = tc.Function.Name
				}
				toolArgs[tc.Index] += tc.Function.Arguments
			}
		}
	}

	content := []normalize.ContentBlock{}
	if text != "" {
		content = append(content, normalize.ContentBlock{Type: normalize.BlockText, Text: text})
	}
	for _, idx := range toolOrder {
		tc := toolCalls[idx]
		tc.Arguments = shared.SafeRawJSON([]byte(toolArgs[idx]))
		content = append(content, normalize.ContentBlock{Type: normalize.BlockToolCall, ToolCall: tc})
	}
	out := []normalize.Message{{Role: "assistant", Content: content}}
	return buildChatResponse(out, finish, usage), model
}

type deltaToolCall struct {
	Index    int    `json:"index"`
	ID       string `json:"id"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}
