package openai

import (
	"encoding/json"
	"strings"

	"agenttape/internal/event"
	"agenttape/internal/normalize"
	"agenttape/internal/normalize/shared"
)

// ResponsesNormalizer implements normalize.Normalizer for the OpenAI Responses
// API (the wire format Codex uses).
type ResponsesNormalizer struct{}

// NewResponses returns a Responses-API normalizer.
func NewResponses() *ResponsesNormalizer { return &ResponsesNormalizer{} }

func (n *ResponsesNormalizer) Name() string { return "openai-responses" }

func (n *ResponsesNormalizer) Detect(ev *event.SourceEvent) (float64, bool) {
	if ev.Capture == nil {
		return 0, false
	}
	if strings.Contains(ev.Capture.Target+" "+ev.Capture.URL, "/responses") {
		return 0.95, true
	}
	body, ok := normalize.RequestBody(ev)
	if !ok {
		return 0, false
	}
	var probe struct {
		Input json.RawMessage `json:"input"`
		Model string          `json:"model"`
	}
	if json.Unmarshal([]byte(body), &probe) == nil && len(probe.Input) > 0 && probe.Model != "" {
		return 0.6, true
	}
	return 0, false
}

func (n *ResponsesNormalizer) Normalize(ev *event.SourceEvent) (*normalize.NormalizedEnvelope, error) {
	env := &normalize.NormalizedEnvelope{Provider: normalize.ProviderRef{Name: n.Name(), WireAPI: "openai.responses"}}
	if ev.Capture != nil {
		env.Provider.Endpoint = ev.Capture.Target
	}
	if body, ok := normalize.RequestBody(ev); ok {
		if req, model := parseResponsesRequest(body); req != nil {
			env.Request = req
			env.Provider.Model = model
		}
	}
	if body, ok := normalize.ResponseBody(ev); ok {
		if resp, model := parseResponsesResponse(body); resp != nil {
			env.Response = resp
			if env.Provider.Model == "" {
				env.Provider.Model = model
			}
		}
	}
	env.Signals = deriveSignals(env)
	return env, nil
}

type responsesReq struct {
	Model        string          `json:"model"`
	Instructions string          `json:"instructions"`
	Input        json.RawMessage `json:"input"`
	Tools        []responsesTool `json:"tools"`
	Reasoning    json.RawMessage `json:"reasoning"`
	ToolChoice   json.RawMessage `json:"tool_choice"`
	Stream       bool            `json:"stream"`
}

type responsesTool struct {
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

func parseResponsesRequest(body string) (*normalize.RequestEnvelope, string) {
	var rb responsesReq
	if json.Unmarshal([]byte(body), &rb) != nil {
		return nil, ""
	}
	req := &normalize.RequestEnvelope{Parameters: map[string]any{}}
	if rb.Instructions != "" {
		req.System = []normalize.ContentBlock{{Type: normalize.BlockText, Text: rb.Instructions}}
	}
	req.Messages = mapItems(rb.Input)
	for _, t := range rb.Tools {
		req.Tools = append(req.Tools, normalize.Tool{Name: t.Name, Description: t.Description, InputSchema: t.Parameters})
	}
	req.Parameters["model"] = rb.Model
	req.Parameters["stream"] = rb.Stream
	if len(rb.Reasoning) > 0 {
		req.Parameters["reasoning"] = json.RawMessage(rb.Reasoning)
	}
	if len(rb.ToolChoice) > 0 {
		req.Parameters["tool_choice"] = json.RawMessage(rb.ToolChoice)
	}
	req.Sections = responsesSections(rb)
	return req, rb.Model
}

func responsesSections(rb responsesReq) []normalize.SectionStat {
	mk := func(name, text string) normalize.SectionStat {
		return normalize.SectionStat{Name: name, Bytes: int64(len(text)), ApproxTokens: shared.ApproxTokens(text)}
	}
	toolsJSON, _ := json.Marshal(rb.Tools)
	return []normalize.SectionStat{
		mk("system", rb.Instructions),
		mk("tools", string(toolsJSON)),
		mk("messages", string(rb.Input)),
	}
}

// --- response ---

type responsesUsage struct {
	InputTokens        int64 `json:"input_tokens"`
	OutputTokens       int64 `json:"output_tokens"`
	TotalTokens        int64 `json:"total_tokens"`
	InputTokensDetails struct {
		CachedTokens int64 `json:"cached_tokens"`
	} `json:"input_tokens_details"`
	OutputTokensDetails struct {
		ReasoningTokens int64 `json:"reasoning_tokens"`
	} `json:"output_tokens_details"`
}

type responseObject struct {
	Model  string          `json:"model"`
	Status string          `json:"status"`
	Output json.RawMessage `json:"output"`
	Usage  responsesUsage  `json:"usage"`
}

func parseResponsesResponse(body string) (*normalize.ResponseEnvelope, string) {
	obj, items, ok := finalResponseObject(body)
	if !ok {
		return nil, ""
	}
	resp := &normalize.ResponseEnvelope{StopReason: obj.Status}
	// Prefer the completed object's output; some streams leave it an empty array
	// and deliver items via output_item.done events instead.
	resp.Output = mapItems(obj.Output)
	if len(resp.Output) == 0 {
		resp.Output = mapItemList(items)
	}
	resp.Usage = obj.Usage.toNeutral()

	var text []string
	for _, m := range resp.Output {
		for _, b := range m.Content {
			switch b.Type {
			case normalize.BlockText:
				if m.Role == "assistant" {
					text = append(text, b.Text)
				}
			case normalize.BlockToolCall:
				if b.ToolCall != nil {
					resp.ToolCalls = append(resp.ToolCalls, *b.ToolCall)
				}
			}
		}
	}
	resp.FinalText = strings.Join(text, "")
	return resp, obj.Model
}

// finalResponseObject extracts the completed response object plus any items
// delivered via output_item.done events (some streams leave response.output
// empty and stream items individually). For non-streaming bodies it parses the
// single JSON object directly.
func finalResponseObject(body string) (responseObject, []respItem, bool) {
	if !shared.IsSSE(body) {
		var obj responseObject
		if json.Unmarshal([]byte(body), &obj) == nil && len(obj.Output) > 0 {
			return obj, nil, true
		}
		return responseObject{}, nil, false
	}
	var obj responseObject
	var items []respItem
	found := false
	for _, ev := range shared.ParseSSE(body) {
		var d struct {
			Type     string         `json:"type"`
			Response responseObject `json:"response"`
			Item     respItem       `json:"item"`
		}
		if json.Unmarshal([]byte(ev.Data), &d) != nil {
			continue
		}
		switch d.Type {
		case "response.completed":
			obj = d.Response
			found = true
		case "response.output_item.done":
			items = append(items, d.Item)
		}
	}
	return obj, items, found
}

// mapItemList converts already-decoded respItems into neutral messages.
func mapItemList(items []respItem) []normalize.Message {
	out := make([]normalize.Message, 0, len(items))
	for _, it := range items {
		if m, ok := mapItem(it); ok {
			out = append(out, m)
		}
	}
	return out
}

func (u responsesUsage) toNeutral() *normalize.Usage {
	nu := &normalize.Usage{InputTokens: u.InputTokens, OutputTokens: u.OutputTokens, TotalTokens: u.TotalTokens}
	if nu.TotalTokens == 0 {
		nu.TotalTokens = u.InputTokens + u.OutputTokens
	}
	if u.InputTokensDetails.CachedTokens != 0 || u.OutputTokensDetails.ReasoningTokens != 0 {
		nu.Extra = map[string]any{
			"cached_tokens":    u.InputTokensDetails.CachedTokens,
			"reasoning_tokens": u.OutputTokensDetails.ReasoningTokens,
		}
	}
	return nu
}
