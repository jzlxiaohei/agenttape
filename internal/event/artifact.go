package event

import (
	"encoding/base64"
	"unicode/utf8"
)

// ArtifactRole labels what part of an exchange an artifact represents.
type ArtifactRole string

const (
	RoleRequestBody  ArtifactRole = "request_body"
	RoleResponseBody ArtifactRole = "response_body"
	RoleStreamChunk  ArtifactRole = "stream_chunk"
	RoleHookPayload  ArtifactRole = "hook_payload"
)

// BodyEncoding tells consumers how to read Body vs BodyBase64.
type BodyEncoding string

const (
	EncUTF8   BodyEncoding = "utf8"
	EncBase64 BodyEncoding = "base64"
)

// RawArtifact preserves bytes exactly as captured at the source boundary.
// UTF-8 payloads go in Body; anything else is base64 in BodyBase64. We never do
// lossy string conversion, so binary and compressed bodies survive intact.
type RawArtifact struct {
	ID                string            `json:"id"`
	Role              ArtifactRole      `json:"role"`
	MediaType         string            `json:"media_type,omitempty"`
	ContentEncoding   string            `json:"content_encoding,omitempty"`  // gzip/zstd/...
	TransferEncoding  []string          `json:"transfer_encoding,omitempty"` // chunked/...
	BodyEncoding      BodyEncoding      `json:"body_encoding"`
	Body              string            `json:"body,omitempty"`
	BodyBase64        string            `json:"body_base64,omitempty"`
	SizeBytes         int64             `json:"size_bytes"`
	RedactionApplied  bool              `json:"redaction_applied,omitempty"`
	RedactionStrategy string            `json:"redaction_strategy,omitempty"`
	Metadata          map[string]string `json:"metadata,omitempty"`
}

// DecodedPayload is a consumer-ready view of a RawArtifact after transport
// decoding (gzip/zstd) and/or SSE stream reassembly. It still records which
// decoders ran so the path from raw bytes to decoded text is auditable.
type DecodedPayload struct {
	ID               string            `json:"id"`
	SourceArtifactID string            `json:"source_artifact_id"`
	Role             ArtifactRole      `json:"role"`
	MediaType        string            `json:"media_type,omitempty"`
	Decoders         []string          `json:"decoders,omitempty"` // e.g. ["gzip","sse"]
	BodyEncoding     BodyEncoding      `json:"body_encoding"`
	Body             string            `json:"body,omitempty"`
	BodyBase64       string            `json:"body_base64,omitempty"`
	SizeBytes        int64             `json:"size_bytes"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

// Bytes returns the artifact's raw bytes regardless of encoding.
func (a RawArtifact) Bytes() ([]byte, error) { return decodeBody(a.BodyEncoding, a.Body, a.BodyBase64) }

// Bytes returns the payload's bytes regardless of encoding.
func (p DecodedPayload) Bytes() ([]byte, error) {
	return decodeBody(p.BodyEncoding, p.Body, p.BodyBase64)
}

func decodeBody(enc BodyEncoding, body, b64 string) ([]byte, error) {
	if enc == EncBase64 {
		return base64.StdEncoding.DecodeString(b64)
	}
	return []byte(body), nil
}

// NewRawArtifact builds a source-boundary artifact without assuming the payload
// is valid text.
func NewRawArtifact(id string, role ArtifactRole, body []byte) RawArtifact {
	a := RawArtifact{ID: id, Role: role, SizeBytes: int64(len(body)), BodyEncoding: EncUTF8}
	if utf8.Valid(body) {
		a.Body = string(body)
		return a
	}
	a.BodyEncoding = EncBase64
	a.BodyBase64 = base64.StdEncoding.EncodeToString(body)
	return a
}

// NewDecodedPayload builds a decoded view that downstream code can index and
// normalize.
func NewDecodedPayload(id, sourceArtifactID string, role ArtifactRole, body []byte, decoders ...string) DecodedPayload {
	p := DecodedPayload{
		ID:               id,
		SourceArtifactID: sourceArtifactID,
		Role:             role,
		Decoders:         decoders,
		SizeBytes:        int64(len(body)),
		BodyEncoding:     EncUTF8,
	}
	if utf8.Valid(body) {
		p.Body = string(body)
		return p
	}
	p.BodyEncoding = EncBase64
	p.BodyBase64 = base64.StdEncoding.EncodeToString(body)
	return p
}
