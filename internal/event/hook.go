package event

import "encoding/json"

// HookEvent captures a lifecycle or tool event emitted by a coding-agent
// harness hook. The raw payload is also kept as a RawArtifact so the hook
// contract can evolve without losing fidelity. No semantic classification
// happens here — that is the normalize layer's job.
type HookEvent struct {
	Runtime           string          `json:"runtime"`    // e.g. "claude_code", "codex"
	EventName         string          `json:"event_name"` // harness-defined, raw
	SessionID         string          `json:"session_id,omitempty"`
	TurnID            string          `json:"turn_id,omitempty"`
	ToolCallID        string          `json:"tool_call_id,omitempty"`
	PayloadArtifactID string          `json:"payload_artifact_id,omitempty"`
	Payload           json.RawMessage `json:"payload,omitempty"`
	Metadata          map[string]any  `json:"metadata,omitempty"`
}

// NewHookEvent wraps a raw hook payload in the same SourceEvent contract used by
// capture and import adapters, proving the collection layer is decoupled from
// how data arrives.
func NewHookEvent(id, runtime, eventName string, payload []byte) SourceEvent {
	artifactID := id + ":hook_payload"
	artifact := NewRawArtifact(artifactID, RoleHookPayload, payload)
	artifact.MediaType = "application/json"

	hook := HookEvent{Runtime: runtime, EventName: eventName, PayloadArtifactID: artifactID}
	if json.Valid(payload) {
		hook.Payload = append(json.RawMessage(nil), payload...)
	}

	ev := New(id, KindHook, SourceRef{Kind: SourceHook, Adapter: "hook", Client: runtime})
	ev.Hook = &hook
	ev.RawArtifacts = []RawArtifact{artifact}
	return ev
}
