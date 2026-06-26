package event_test

import (
	"testing"

	"agenttape/internal/event"
)

// consume is a source-agnostic consumer: it works purely off the SourceEvent
// contract and never inspects how the data was collected. If both an HTTP
// capture and a hook event can flow through this unchanged, the collection
// layer is genuinely decoupled (next.md 1.2).
func consume(ev *event.SourceEvent) (schemaOK bool, payloads int) {
	schemaOK = ev.SchemaVersion == event.SchemaVersion
	for _, a := range ev.RawArtifacts {
		if b, err := a.Bytes(); err == nil && len(b) > 0 {
			payloads++
		}
	}
	return
}

func TestDecoupling_HTTPAndHookAreIsomorphic(t *testing.T) {
	// An HTTP capture event.
	reqArt := event.NewRawArtifact("h:req", event.RoleRequestBody, []byte(`{"model":"x"}`))
	httpEv := event.New("h", event.KindHTTPExchange, event.SourceRef{Kind: event.SourceCapture, Adapter: "httpcap"})
	httpEv.Capture = &event.CaptureEvent{Protocol: "http", Method: "POST", Request: event.HTTPMessage{BodyArtifactID: reqArt.ID}}
	httpEv.RawArtifacts = []event.RawArtifact{reqArt}

	// A hook event, built via the shared constructor.
	hookEv := event.NewHookEvent("k", "claude_code", "PreToolUse", []byte(`{"tool":"Bash"}`))

	for _, tc := range []struct {
		name string
		ev   *event.SourceEvent
	}{
		{"http", &httpEv},
		{"hook", &hookEv},
	} {
		schemaOK, payloads := consume(tc.ev)
		if !schemaOK {
			t.Errorf("%s: schema version not set on contract", tc.name)
		}
		if payloads != 1 {
			t.Errorf("%s: expected 1 readable payload, got %d", tc.name, payloads)
		}
	}
}

func TestRawArtifactBinaryRoundTrip(t *testing.T) {
	bin := []byte{0xff, 0x00, 0xfe, 0x01}
	a := event.NewRawArtifact("b", event.RoleResponseBody, bin)
	if a.BodyEncoding != event.EncBase64 {
		t.Fatalf("binary should be base64-encoded, got %s", a.BodyEncoding)
	}
	got, err := a.Bytes()
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if string(got) != string(bin) {
		t.Errorf("round trip mismatch: %v != %v", got, bin)
	}
}
