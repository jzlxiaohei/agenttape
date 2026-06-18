package shared

import (
	"bufio"
	"strings"
)

// SSEEvent is one parsed Server-Sent Event: its event name (may be empty) and
// the concatenated data payload. This is pure transport parsing — interpreting
// what an event means is each provider's job.
type SSEEvent struct {
	Event string
	Data  string
}

// ParseSSE splits a raw SSE stream into events. Multi-line data fields are
// joined with "\n" per the SSE spec. Comment lines (starting with ":") and the
// "[DONE]" sentinel are skipped.
func ParseSSE(raw string) []SSEEvent {
	var events []SSEEvent
	var name string
	var data []string

	flush := func() {
		if len(data) == 0 && name == "" {
			return
		}
		joined := strings.Join(data, "\n")
		if joined != "[DONE]" {
			events = append(events, SSEEvent{Event: name, Data: joined})
		}
		name = ""
		data = data[:0]
	}

	sc := bufio.NewScanner(strings.NewReader(raw))
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for sc.Scan() {
		line := sc.Text()
		switch {
		case line == "":
			flush()
		case strings.HasPrefix(line, ":"):
			// comment, ignore
		case strings.HasPrefix(line, "event:"):
			name = strings.TrimSpace(line[len("event:"):])
		case strings.HasPrefix(line, "data:"):
			data = append(data, strings.TrimPrefix(line[len("data:"):], " "))
		}
	}
	flush()
	return events
}

// IsSSE reports whether body looks like an SSE stream rather than a single JSON
// document.
func IsSSE(body string) bool {
	t := strings.TrimLeft(body, " \t\r\n")
	return strings.HasPrefix(t, "event:") || strings.HasPrefix(t, "data:")
}
