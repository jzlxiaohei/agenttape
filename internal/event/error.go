package event

// TraceError records a non-fatal problem encountered while capturing or
// decoding. We never drop data on error: the raw artifact is preserved and the
// failure is recorded here for downstream visibility.
type TraceError struct {
	Code    string         `json:"code,omitempty"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

func (e *TraceError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}
