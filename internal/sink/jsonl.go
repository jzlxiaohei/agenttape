package sink

import (
	"encoding/json"
	"os"
	"sync"
)

// JSONL appends one JSON record per line to a file. It is the simplest durable
// sink and is trivially re-readable by module 2's ingester.
type JSONL struct {
	mu  sync.Mutex
	f   *os.File
	enc *json.Encoder
}

// NewJSONL opens (creating/appending) the given path for writing.
func NewJSONL(path string) (*JSONL, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	return &JSONL{f: f, enc: json.NewEncoder(f)}, nil
}

// Write appends rec as one JSON line.
func (j *JSONL) Write(rec Record) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.enc.Encode(rec)
}

// Close closes the underlying file.
func (j *JSONL) Close() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.f.Close()
}

// ReadAll loads all records from a JSONL file (used by `tracelab dump` and
// tests).
func ReadAll(path string) ([]Record, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []Record
	dec := json.NewDecoder(f)
	for dec.More() {
		var rec Record
		if err := dec.Decode(&rec); err != nil {
			return out, err
		}
		out = append(out, rec)
	}
	return out, nil
}
