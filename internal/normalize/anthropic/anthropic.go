// Package anthropic normalizes Claude Messages API exchanges. It is fully
// independent of other provider packages and only uses normalize + shared.
package anthropic

import (
	"encoding/json"
	"strings"

	"tracelab/internal/event"
	"tracelab/internal/normalize"
	"tracelab/internal/normalize/shared"
)

// Normalizer implements normalize.Normalizer for the Anthropic Messages API.
type Normalizer struct{}

// New returns an Anthropic normalizer.
func New() *Normalizer { return &Normalizer{} }

// Name is the provider id used in ProviderRef.Name.
func (n *Normalizer) Name() string { return "anthropic" }

// Detect recognizes Anthropic traffic by endpoint, headers, and body shape.
func (n *Normalizer) Detect(ev *event.SourceEvent) (float64, bool) {
	if ev.Capture == nil {
		return 0, false
	}
	url := ev.Capture.Target + " " + ev.Capture.URL
	if strings.Contains(url, "/v1/messages") {
		return 0.95, true
	}
	if hasHeader(ev.Capture.Request.Headers, "Anthropic-Version") {
		return 0.85, true
	}
	body, ok := normalize.RequestBody(ev)
	if !ok {
		return 0, false
	}
	var probe struct {
		Messages  json.RawMessage `json:"messages"`
		MaxTokens *int            `json:"max_tokens"`
		System    json.RawMessage `json:"system"`
	}
	if json.Unmarshal([]byte(body), &probe) == nil && probe.Messages != nil && probe.MaxTokens != nil {
		return 0.6, true
	}
	return 0, false
}

// anthropic request wire types.
type reqBody struct {
	Model     string          `json:"model"`
	System    json.RawMessage `json:"system"`
	Messages  []wireMessage   `json:"messages"`
	Tools     []wireTool      `json:"tools"`
	MaxTokens int             `json:"max_tokens"`
	Temp      *float64        `json:"temperature"`
	TopP      *float64        `json:"top_p"`
	Stream    bool            `json:"stream"`
	Thinking  json.RawMessage `json:"thinking"`
}

type wireMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type wireTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// Normalize parses the request and (reassembled) response into the neutral model.
func (n *Normalizer) Normalize(ev *event.SourceEvent) (*normalize.NormalizedEnvelope, error) {
	env := &normalize.NormalizedEnvelope{Provider: normalize.ProviderRef{Name: n.Name(), WireAPI: "anthropic.messages"}}
	if ev.Capture != nil {
		env.Provider.Endpoint = ev.Capture.Target
	}

	if body, ok := normalize.RequestBody(ev); ok {
		req, err := parseRequest(body)
		if err == nil {
			env.Request = req
			env.Provider.Model = modelOf(body)
		}
	}
	if body, ok := normalize.ResponseBody(ev); ok {
		resp, model := parseResponse(body)
		env.Response = resp
		if env.Provider.Model == "" {
			env.Provider.Model = model
		}
	}
	env.Signals = deriveSignals(env)
	return env, nil
}

func parseRequest(body string) (*normalize.RequestEnvelope, error) {
	var rb reqBody
	if err := json.Unmarshal([]byte(body), &rb); err != nil {
		return nil, err
	}
	req := &normalize.RequestEnvelope{Parameters: map[string]any{}}

	req.System = parseSystem(rb.System)
	for _, m := range rb.Messages {
		req.Messages = append(req.Messages, normalize.Message{
			Role:    m.Role,
			Content: parseBlocks(m.Content),
		})
	}
	for _, t := range rb.Tools {
		req.Tools = append(req.Tools, normalize.Tool{Name: t.Name, Description: t.Description, InputSchema: t.InputSchema})
	}

	req.Parameters["model"] = rb.Model
	req.Parameters["max_tokens"] = rb.MaxTokens
	req.Parameters["stream"] = rb.Stream
	if rb.Temp != nil {
		req.Parameters["temperature"] = *rb.Temp
	}
	if rb.TopP != nil {
		req.Parameters["top_p"] = *rb.TopP
	}
	if len(rb.Thinking) > 0 {
		req.Parameters["thinking"] = json.RawMessage(rb.Thinking)
	}

	req.Sections = sectionStats(rb)
	return req, nil
}

func parseSystem(raw json.RawMessage) []normalize.ContentBlock {
	if len(raw) == 0 {
		return nil
	}
	// system can be a plain string or an array of text blocks.
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return []normalize.ContentBlock{{Type: normalize.BlockText, Text: s}}
	}
	return parseBlocks(raw)
}

func sectionStats(rb reqBody) []normalize.SectionStat {
	stat := func(name string, v any) normalize.SectionStat {
		b, _ := json.Marshal(v)
		return normalize.SectionStat{Name: name, Bytes: int64(len(b)), ApproxTokens: shared.ApproxTokens(string(b))}
	}
	return []normalize.SectionStat{
		stat("system", json.RawMessage(rb.System)),
		stat("tools", rb.Tools),
		stat("messages", rb.Messages),
	}
}

func modelOf(body string) string {
	var m struct {
		Model string `json:"model"`
	}
	_ = json.Unmarshal([]byte(body), &m)
	return m.Model
}

func hasHeader(h map[string][]string, key string) bool {
	for k := range h {
		if strings.EqualFold(k, key) {
			return true
		}
	}
	return false
}
