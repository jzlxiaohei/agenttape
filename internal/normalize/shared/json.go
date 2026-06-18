package shared

import "encoding/json"

// SafeRawJSON returns b as a json.RawMessage only if it is valid JSON. Invalid
// or empty input yields nil, so callers never embed malformed JSON into a struct
// that will later be marshaled (which would fail the whole encode).
func SafeRawJSON(b []byte) json.RawMessage {
	if len(b) == 0 || !json.Valid(b) {
		return nil
	}
	return json.RawMessage(b)
}
