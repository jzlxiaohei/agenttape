package anthropic

import (
	"encoding/json"
	"strings"

	"github.com/jzlxiaohei/agenttape/internal/normalize"
	"github.com/jzlxiaohei/agenttape/internal/normalize/shared"
)

// parseResponse reconstructs the response from either an SSE stream or a single
// JSON message object. Returns the neutral response and the model name.
func parseResponse(body string) (*normalize.ResponseEnvelope, string) {
	if shared.IsSSE(body) {
		return reassembleSSE(body)
	}
	return parseJSONMessage(body)
}

// --- non-streaming JSON message ---

type wireUsage struct {
	InputTokens         int64 `json:"input_tokens"`
	OutputTokens        int64 `json:"output_tokens"`
	CacheCreationTokens int64 `json:"cache_creation_input_tokens"`
	CacheReadTokens     int64 `json:"cache_read_input_tokens"`
}

func (u wireUsage) toNeutral() *normalize.Usage {
	nu := &normalize.Usage{
		InputTokens:  u.InputTokens,
		OutputTokens: u.OutputTokens,
		TotalTokens:  u.InputTokens + u.OutputTokens,
	}
	if u.CacheCreationTokens != 0 || u.CacheReadTokens != 0 {
		nu.Extra = map[string]any{
			"cache_creation_input_tokens": u.CacheCreationTokens,
			"cache_read_input_tokens":     u.CacheReadTokens,
		}
	}
	return nu
}

func parseJSONMessage(body string) (*normalize.ResponseEnvelope, string) {
	var msg struct {
		Model      string          `json:"model"`
		Content    json.RawMessage `json:"content"`
		StopReason string          `json:"stop_reason"`
		Usage      wireUsage       `json:"usage"`
	}
	if json.Unmarshal([]byte(body), &msg) != nil {
		return nil, ""
	}
	blocks := parseBlocks(msg.Content)
	return buildResponse(blocks, msg.StopReason, msg.Usage.toNeutral()), msg.Model
}

// --- streaming reassembly ---

type blockAcc struct {
	block   normalize.ContentBlock
	jsonBuf strings.Builder // for tool_use input_json_delta
}

func reassembleSSE(body string) (*normalize.ResponseEnvelope, string) {
	var model, stopReason string
	usage := &normalize.Usage{}
	accs := map[int]*blockAcc{}
	var order []int

	for _, ev := range shared.ParseSSE(body) {
		var d struct {
			Type    string `json:"type"`
			Index   int    `json:"index"`
			Message struct {
				Model string    `json:"model"`
				Usage wireUsage `json:"usage"`
			} `json:"message"`
			ContentBlock wireBlock `json:"content_block"`
			Delta        struct {
				Type        string `json:"type"`
				Text        string `json:"text"`
				Thinking    string `json:"thinking"`
				PartialJSON string `json:"partial_json"`
				StopReason  string `json:"stop_reason"`
			} `json:"delta"`
			Usage wireUsage `json:"usage"`
		}
		if json.Unmarshal([]byte(ev.Data), &d) != nil {
			continue
		}
		switch d.Type {
		case "message_start":
			model = d.Message.Model
			usage = d.Message.Usage.toNeutral()
		case "content_block_start":
			a := &blockAcc{block: mapBlock(d.ContentBlock)}
			accs[d.Index] = a
			order = append(order, d.Index)
		case "content_block_delta":
			a := accs[d.Index]
			if a == nil {
				continue
			}
			switch d.Delta.Type {
			case "text_delta":
				a.block.Text += d.Delta.Text
			case "thinking_delta":
				a.block.Text += d.Delta.Thinking
			case "input_json_delta":
				a.jsonBuf.WriteString(d.Delta.PartialJSON)
			}
		case "message_delta":
			if d.Delta.StopReason != "" {
				stopReason = d.Delta.StopReason
			}
			if d.Usage.OutputTokens != 0 {
				usage.OutputTokens = d.Usage.OutputTokens
				usage.TotalTokens = usage.InputTokens + usage.OutputTokens
			}
		}
	}

	blocks := make([]normalize.ContentBlock, 0, len(order))
	for _, idx := range order {
		a := accs[idx]
		if a.block.Type == normalize.BlockToolCall && a.block.ToolCall != nil && a.jsonBuf.Len() > 0 {
			a.block.ToolCall.Arguments = shared.SafeRawJSON([]byte(a.jsonBuf.String()))
		}
		blocks = append(blocks, a.block)
	}
	return buildResponse(blocks, stopReason, usage), model
}

func buildResponse(blocks []normalize.ContentBlock, stopReason string, usage *normalize.Usage) *normalize.ResponseEnvelope {
	resp := &normalize.ResponseEnvelope{StopReason: stopReason, Usage: usage}
	if len(blocks) > 0 {
		resp.Output = []normalize.Message{{Role: "assistant", Content: blocks}}
	}
	var text []string
	for _, b := range blocks {
		switch b.Type {
		case normalize.BlockText:
			text = append(text, b.Text)
		case normalize.BlockToolCall:
			if b.ToolCall != nil {
				resp.ToolCalls = append(resp.ToolCalls, *b.ToolCall)
			}
		}
	}
	resp.FinalText = strings.Join(text, "")
	return resp
}
