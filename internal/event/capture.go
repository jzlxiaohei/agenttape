package event

// CaptureEvent describes one HTTP exchange (or CONNECT tunnel) observed by a
// capture adapter. Body content stays out of this struct — it lives in
// RawArtifacts / DecodedPayloads referenced by ID — so binary and compressed
// payloads are preserved without lossy handling.
type CaptureEvent struct {
	Protocol string       `json:"protocol"` // "http"
	Method   string       `json:"method,omitempty"`
	URL      string       `json:"url,omitempty"`    // proxy-facing URL
	Target   string       `json:"target,omitempty"` // upstream URL
	Request  HTTPMessage  `json:"request"`
	Response HTTPMessage  `json:"response,omitzero"`
	Tunnel   *TunnelEvent `json:"tunnel,omitempty"`
}

// HTTPMessage carries headers, status, and pointers to body artifacts. It holds
// only raw HTTP facts — no provider interpretation.
type HTTPMessage struct {
	Headers          map[string][]string `json:"headers,omitempty"`
	StatusCode       int                 `json:"status_code,omitempty"`
	ContentType      string              `json:"content_type,omitempty"`
	ContentEncoding  string              `json:"content_encoding,omitempty"`
	TransferEncoding []string            `json:"transfer_encoding,omitempty"`
	BodyArtifactID   string              `json:"body_artifact_id,omitempty"`
	DecodedPayloadID string              `json:"decoded_payload_id,omitempty"`
	BodySizeBytes    int64               `json:"body_size_bytes,omitempty"`
	StreamPayloadIDs []string            `json:"stream_payload_ids,omitempty"`
}

// TunnelEvent records byte counts for an opaque CONNECT tunnel (no MITM).
type TunnelEvent struct {
	BytesToTarget int64 `json:"bytes_to_target,omitempty"`
	BytesToClient int64 `json:"bytes_to_client,omitempty"`
}
