// Package event defines the stable handoff contract between tracelab's
// collection layer (source adapters) and everything downstream (normalize,
// sink). A SourceEvent carries raw, faithfully captured facts about one
// observed exchange — HTTP request/response bytes or a harness hook payload.
//
// This package MUST NOT contain any provider semantics. It does not know what
// "anthropic", "openai" or "codex" mean. Provider detection and semantic
// interpretation live entirely in the normalize layer. Keeping this boundary
// clean is what lets new data sources and new providers be added independently.
package event

// SchemaVersion identifies the SourceEvent contract revision.
const SchemaVersion = "tracelab.event.v1"

// EventKind is the shape of the observed exchange.
type EventKind string

const (
	KindHTTPExchange EventKind = "http_exchange"
	KindHTTPTunnel   EventKind = "http_tunnel"
	KindHook         EventKind = "hook_event"
	KindImport       EventKind = "import_event"
)

// SourceKind identifies the family of adapter that produced the event.
type SourceKind string

const (
	SourceCapture SourceKind = "capture"
	SourceHook    SourceKind = "hook"
	SourceImport  SourceKind = "import"
)

// SourceRef describes which adapter produced an event and under what mode.
type SourceRef struct {
	Kind    SourceKind        `json:"kind"`
	Adapter string            `json:"adapter"`          // e.g. "httpcap", "hook"
	Mode    string            `json:"mode,omitempty"`   // e.g. "reverse_https"
	Client  string            `json:"client,omitempty"` // e.g. "claude_code", "codex_cli"
	Hints   map[string]string `json:"hints,omitempty"`
}

// Correlation ties events into sessions and turns. session_id is stamped by the
// launcher (via an injected per-session token) so concurrent cc/codex sessions
// never get mixed together. No semantic interpretation happens here — these are
// identifiers, not classifications.
type Correlation struct {
	SessionID string            `json:"session_id,omitempty"`
	TurnID    string            `json:"turn_id,omitempty"`
	RequestID string            `json:"request_id,omitempty"`
	ParentID  string            `json:"parent_id,omitempty"`
	Hints     map[string]string `json:"hints,omitempty"`
}

// Timing records when the exchange started and how long it took.
type Timing struct {
	StartedAt   string `json:"started_at"`
	CompletedAt string `json:"completed_at,omitempty"`
	DurationMS  int64  `json:"duration_ms,omitempty"`
}

// SourceEvent is the single, stable object every adapter emits. Downstream code
// consumes only this shape and must never need to know how the data arrived.
type SourceEvent struct {
	SchemaVersion   string           `json:"schema_version"`
	ID              string           `json:"id"`
	Kind            EventKind        `json:"kind"`
	Source          SourceRef        `json:"source"`
	Correlation     Correlation      `json:"correlation,omitzero"`
	Timing          Timing           `json:"timing"`
	Capture         *CaptureEvent    `json:"capture,omitempty"`
	Hook            *HookEvent       `json:"hook,omitempty"`
	RawArtifacts    []RawArtifact    `json:"raw_artifacts,omitempty"`
	DecodedPayloads []DecodedPayload `json:"decoded_payloads,omitempty"`
	Metadata        map[string]any   `json:"metadata,omitempty"`
	Error           *TraceError      `json:"error,omitempty"`
}

// New returns a SourceEvent with the current schema version set.
func New(id string, kind EventKind, src SourceRef) SourceEvent {
	return SourceEvent{
		SchemaVersion: SchemaVersion,
		ID:            id,
		Kind:          kind,
		Source:        src,
	}
}

// Artifact returns the raw artifact with the given id, or nil.
func (e *SourceEvent) Artifact(id string) *RawArtifact {
	for i := range e.RawArtifacts {
		if e.RawArtifacts[i].ID == id {
			return &e.RawArtifacts[i]
		}
	}
	return nil
}

// Decoded returns the decoded payload with the given id, or nil.
func (e *SourceEvent) Decoded(id string) *DecodedPayload {
	for i := range e.DecodedPayloads {
		if e.DecodedPayloads[i].ID == id {
			return &e.DecodedPayloads[i]
		}
	}
	return nil
}
