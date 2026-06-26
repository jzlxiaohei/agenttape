package openai

import (
	"encoding/json"
	"strings"

	"agenttape/internal/event"
	"agenttape/internal/normalize"
	"agenttape/internal/normalize/shared"
)

// ChatNormalizer implements normalize.Normalizer for the OpenAI Chat
// Completions API (also the de-facto shape for many OpenAI-compatible servers).
type ChatNormalizer struct{}

// NewChat returns a Chat Completions normalizer.
func NewChat() *ChatNormalizer { return &ChatNormalizer{} }

func (n *ChatNormalizer) Name() string { return "openai-chat" }

func (n *ChatNormalizer) Detect(ev *event.SourceEvent) (float64, bool) {
	if ev.Capture == nil {
		return 0, false
	}
	if strings.Contains(ev.Capture.Target+" "+ev.Capture.URL, "/chat/completions") {
		return 0.95, true
	}
	body, ok := normalize.RequestBody(ev)
	if !ok {
		return 0, false
	}
	var probe struct {
		Messages []json.RawMessage `json:"messages"`
		Model    string            `json:"model"`
		Input    json.RawMessage   `json:"input"`
	}
	// messages+model but NOT a Responses-API input array.
	if json.Unmarshal([]byte(body), &probe) == nil && len(probe.Messages) > 0 && probe.Model != "" && probe.Input == nil {
		return 0.5, true
	}
	return 0, false
}

type chatReq struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Tools    []chatTool    `json:"tools"`
	Stream   bool          `json:"stream"`
	Temp     *float64      `json:"temperature"`
}

type chatTool struct {
	Function struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Parameters  json.RawMessage `json:"parameters"`
	} `json:"function"`
}

type chatMessage struct {
	Role       string          `json:"role"`
	Content    json.RawMessage `json:"content"`
	ToolCalls  []chatToolCall  `json:"tool_calls"`
	ToolCallID string          `json:"tool_call_id"`
}

type chatToolCall struct {
	ID       string `json:"id"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

func (n *ChatNormalizer) Normalize(ev *event.SourceEvent) (*normalize.NormalizedEnvelope, error) {
	env := &normalize.NormalizedEnvelope{Provider: normalize.ProviderRef{Name: n.Name(), WireAPI: "openai.chat"}}
	if ev.Capture != nil {
		env.Provider.Endpoint = ev.Capture.Target
	}
	if body, ok := normalize.RequestBody(ev); ok {
		if req, model := parseChatRequest(body); req != nil {
			env.Request = req
			env.Provider.Model = model
		}
	}
	if body, ok := normalize.ResponseBody(ev); ok {
		if resp, model := parseChatResponse(body); resp != nil {
			env.Response = resp
			if env.Provider.Model == "" {
				env.Provider.Model = model
			}
		}
	}
	env.Signals = deriveSignals(env)
	return env, nil
}

func parseChatRequest(body string) (*normalize.RequestEnvelope, string) {
	var rb chatReq
	if json.Unmarshal([]byte(body), &rb) != nil {
		return nil, ""
	}
	req := &normalize.RequestEnvelope{Parameters: map[string]any{}}
	for _, m := range rb.Messages {
		msg := mapChatMessage(m)
		if m.Role == "system" || m.Role == "developer" {
			req.System = append(req.System, msg.Content...)
			continue
		}
		req.Messages = append(req.Messages, msg)
	}
	for _, t := range rb.Tools {
		req.Tools = append(req.Tools, normalize.Tool{Name: t.Function.Name, Description: t.Function.Description, InputSchema: t.Function.Parameters})
	}
	req.Parameters["model"] = rb.Model
	req.Parameters["stream"] = rb.Stream
	if rb.Temp != nil {
		req.Parameters["temperature"] = *rb.Temp
	}
	req.Sections = chatSections(rb)
	return req, rb.Model
}

func mapChatMessage(m chatMessage) normalize.Message {
	msg := normalize.Message{Role: m.Role}
	if m.Role == "tool" {
		msg.Content = []normalize.ContentBlock{{
			Type:       normalize.BlockToolResult,
			ToolResult: &normalize.ToolResult{ToolCallID: m.ToolCallID, Content: mapContent(m.Content)},
		}}
		return msg
	}
	msg.Content = mapContent(m.Content)
	for _, tc := range m.ToolCalls {
		msg.Content = append(msg.Content, normalize.ContentBlock{
			Type:     normalize.BlockToolCall,
			ToolCall: &normalize.ToolCall{ID: tc.ID, Name: tc.Function.Name, Arguments: shared.SafeRawJSON([]byte(tc.Function.Arguments))},
		})
	}
	return msg
}

func chatSections(rb chatReq) []normalize.SectionStat {
	var sys, msgs strings.Builder
	for _, m := range rb.Messages {
		if m.Role == "system" || m.Role == "developer" {
			sys.Write(m.Content)
			continue
		}
		msgs.Write(m.Content)
	}
	toolsJSON, _ := json.Marshal(rb.Tools)
	mk := func(name, text string) normalize.SectionStat {
		return normalize.SectionStat{Name: name, Bytes: int64(len(text)), ApproxTokens: shared.ApproxTokens(text)}
	}
	return []normalize.SectionStat{
		mk("system", sys.String()),
		mk("tools", string(toolsJSON)),
		mk("messages", msgs.String()),
	}
}

// --- response ---

func parseChatResponse(body string) (*normalize.ResponseEnvelope, string) {
	if shared.IsSSE(body) {
		return reassembleChatSSE(body)
	}
	var obj struct {
		Model   string `json:"model"`
		Choices []struct {
			Message      chatMessage `json:"message"`
			FinishReason string      `json:"finish_reason"`
		} `json:"choices"`
		Usage chatUsage `json:"usage"`
	}
	if json.Unmarshal([]byte(body), &obj) != nil || len(obj.Choices) == 0 {
		return nil, ""
	}
	ch := obj.Choices[0]
	msg := mapChatMessage(ch.Message)
	msg.Role = "assistant"
	return buildChatResponse([]normalize.Message{msg}, ch.FinishReason, obj.Usage.toNeutral()), obj.Model
}

type chatUsage struct {
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
	TotalTokens      int64 `json:"total_tokens"`
}

func (u chatUsage) toNeutral() *normalize.Usage {
	if u.PromptTokens == 0 && u.CompletionTokens == 0 && u.TotalTokens == 0 {
		return nil
	}
	total := u.TotalTokens
	if total == 0 {
		total = u.PromptTokens + u.CompletionTokens
	}
	return &normalize.Usage{InputTokens: u.PromptTokens, OutputTokens: u.CompletionTokens, TotalTokens: total}
}

func buildChatResponse(out []normalize.Message, finish string, usage *normalize.Usage) *normalize.ResponseEnvelope {
	resp := &normalize.ResponseEnvelope{Output: out, StopReason: finish, Usage: usage}
	var text []string
	for _, m := range out {
		for _, b := range m.Content {
			switch b.Type {
			case normalize.BlockText:
				text = append(text, b.Text)
			case normalize.BlockToolCall:
				if b.ToolCall != nil {
					resp.ToolCalls = append(resp.ToolCalls, *b.ToolCall)
				}
			}
		}
	}
	resp.FinalText = strings.Join(text, "")
	return resp
}
