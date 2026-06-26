// Package normalize turns provider-agnostic SourceEvents into a provider-neutral
// semantic model (NormalizedEnvelope). Each provider gets its own sub-package
// implementing Normalizer; they share only the atomic helpers in
// normalize/shared. Adding a provider must not touch existing ones.
package normalize

import "encoding/json"

// EnvelopeSchemaVersion identifies the normalized model revision.
const EnvelopeSchemaVersion = "agenttape.normalized.v1"

// BlockType classifies a content block. The type is always derived from the
// provider's own structural type field — never guessed from keywords.
type BlockType string

const (
	BlockText       BlockType = "text"
	BlockReasoning  BlockType = "reasoning"
	BlockToolCall   BlockType = "tool_call"
	BlockToolResult BlockType = "tool_result"
	BlockImage      BlockType = "image"
	BlockUnknown    BlockType = "unknown" // structurally present but unmapped
)

// ContentBlock is one typed unit of message content. Raw always holds the
// original provider block so nothing is lost and power users can drill in.
type ContentBlock struct {
	Type       BlockType       `json:"type"`
	Text       string          `json:"text,omitempty"`
	ToolCall   *ToolCall       `json:"tool_call,omitempty"`
	ToolResult *ToolResult     `json:"tool_result,omitempty"`
	Raw        json.RawMessage `json:"raw,omitempty"`
}

// Message is a role-tagged sequence of typed content blocks.
type Message struct {
	Role    string         `json:"role"`
	Content []ContentBlock `json:"content,omitempty"`
}

// Tool is a provider-neutral tool/function definition.
type Tool struct {
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema,omitempty"`
}

// ToolCall is a model request to invoke a tool.
type ToolCall struct {
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// ToolResult is the result fed back for a tool call.
type ToolResult struct {
	ToolCallID string         `json:"tool_call_id,omitempty"`
	Content    []ContentBlock `json:"content,omitempty"`
	IsError    bool           `json:"is_error,omitempty"`
}

// Usage is token accounting. Provider-specific extras (cache tokens, tiers) go
// in Extra so the neutral fields stay comparable across providers.
type Usage struct {
	InputTokens  int64          `json:"input_tokens,omitempty"`
	OutputTokens int64          `json:"output_tokens,omitempty"`
	TotalTokens  int64          `json:"total_tokens,omitempty"`
	Extra        map[string]any `json:"extra,omitempty"`
}

// SectionStat is the size of one request section (system / tools / messages).
// ApproxTokens is explicitly approximate (see shared.ApproxTokens).
type SectionStat struct {
	Name         string `json:"name"`
	Bytes        int64  `json:"bytes"`
	ApproxTokens int64  `json:"approx_tokens"`
}

// TagSignal is a structural hint about what an exchange contains. Signals that
// cannot be determined structurally MUST set Suspected=true so the UI can show
// them as "疑似" rather than fact (see next.md 3.3).
type TagSignal struct {
	Tag        string  `json:"tag"`
	Confidence float64 `json:"confidence"`
	Suspected  bool    `json:"suspected,omitempty"`
	Evidence   string  `json:"evidence,omitempty"`
}

// ProviderRef names the detected provider and model. Name is the normalizer's
// own id (e.g. "anthropic", "openai-responses"), not a guess.
type ProviderRef struct {
	Name     string `json:"name"`
	Model    string `json:"model,omitempty"`
	Endpoint string `json:"endpoint,omitempty"`
	WireAPI  string `json:"wire_api,omitempty"`
}

// RequestEnvelope is the provider-neutral request model.
type RequestEnvelope struct {
	System     []ContentBlock `json:"system,omitempty"`
	Messages   []Message      `json:"messages,omitempty"`
	Tools      []Tool         `json:"tools,omitempty"`
	Parameters map[string]any `json:"parameters,omitempty"`
	Sections   []SectionStat  `json:"sections,omitempty"`
}

// ResponseEnvelope is the provider-neutral response model.
type ResponseEnvelope struct {
	Output     []Message  `json:"output,omitempty"`
	FinalText  string     `json:"final_text,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	Usage      *Usage     `json:"usage,omitempty"`
	StopReason string     `json:"stop_reason,omitempty"`
}

// NormalizedEnvelope is the full semantic view downstream modules consume.
type NormalizedEnvelope struct {
	SchemaVersion string            `json:"schema_version"`
	EventID       string            `json:"event_id"`
	Provider      ProviderRef       `json:"provider"`
	Request       *RequestEnvelope  `json:"request,omitempty"`
	Response      *ResponseEnvelope `json:"response,omitempty"`
	Signals       []TagSignal       `json:"signals,omitempty"`
	Metadata      map[string]any    `json:"metadata,omitempty"`
}
