package httpcap

import (
	"net/http"
	"strings"
)

// hopByHop headers must not be forwarded across the proxy boundary.
var hopByHop = map[string]bool{
	"connection":          true,
	"keep-alive":          true,
	"proxy-authenticate":  true,
	"proxy-authorization": true,
	"te":                  true,
	"trailer":             true,
	"transfer-encoding":   true,
	"upgrade":             true,
}

// sensitive headers are redacted before an event is persisted. We still forward
// them upstream — redaction only affects what is stored on disk.
var sensitive = map[string]bool{
	"authorization":       true,
	"cookie":              true,
	"set-cookie":          true,
	"x-api-key":           true,
	"anthropic-api-key":   true,
	"openai-organization": true,
}

func copyHeaders(dst, src http.Header) {
	for k, vs := range src {
		if hopByHop[strings.ToLower(k)] {
			continue
		}
		for _, v := range vs {
			dst.Add(k, v)
		}
	}
}

// redactHeaders returns a copy with sensitive values replaced by "[redacted]".
func redactHeaders(h http.Header) map[string][]string {
	out := make(map[string][]string, len(h))
	for k, vs := range h {
		if sensitive[strings.ToLower(k)] {
			out[k] = []string{"[redacted]"}
			continue
		}
		out[k] = append([]string(nil), vs...)
	}
	return out
}
