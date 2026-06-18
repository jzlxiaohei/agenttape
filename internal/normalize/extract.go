package normalize

import "tracelab/internal/event"

// RequestBody returns the request body as text, preferring the transport-decoded
// payload and falling back to the raw artifact. The second result is false when
// no request body is present.
func RequestBody(ev *event.SourceEvent) (string, bool) {
	if ev.Capture != nil {
		return bodyFor(ev, ev.Capture.Request.DecodedPayloadID, ev.Capture.Request.BodyArtifactID, event.RoleRequestBody)
	}
	return "", false
}

// ResponseBody returns the response body as text (decoded if available). For
// streaming responses this is the raw SSE text; provider normalizers reassemble
// it, since reassembly is provider-specific.
func ResponseBody(ev *event.SourceEvent) (string, bool) {
	if ev.Capture != nil {
		return bodyFor(ev, ev.Capture.Response.DecodedPayloadID, ev.Capture.Response.BodyArtifactID, event.RoleResponseBody)
	}
	return "", false
}

// HookPayload returns the raw hook payload text.
func HookPayload(ev *event.SourceEvent) (string, bool) {
	if ev.Hook == nil {
		return "", false
	}
	if a := ev.Artifact(ev.Hook.PayloadArtifactID); a != nil {
		if b, err := a.Bytes(); err == nil {
			return string(b), true
		}
	}
	if len(ev.Hook.Payload) > 0 {
		return string(ev.Hook.Payload), true
	}
	return "", false
}

func bodyFor(ev *event.SourceEvent, decodedID, rawID string, role event.ArtifactRole) (string, bool) {
	if decodedID != "" {
		if p := ev.Decoded(decodedID); p != nil {
			if b, err := p.Bytes(); err == nil {
				return string(b), true
			}
		}
	}
	if rawID != "" {
		if a := ev.Artifact(rawID); a != nil {
			if b, err := a.Bytes(); err == nil {
				return string(b), true
			}
		}
	}
	// Fallback: first decoded/raw artifact with the right role.
	for i := range ev.DecodedPayloads {
		if ev.DecodedPayloads[i].Role == role {
			if b, err := ev.DecodedPayloads[i].Bytes(); err == nil {
				return string(b), true
			}
		}
	}
	for i := range ev.RawArtifacts {
		if ev.RawArtifacts[i].Role == role {
			if b, err := ev.RawArtifacts[i].Bytes(); err == nil {
				return string(b), true
			}
		}
	}
	return "", false
}
