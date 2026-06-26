// Package sink persists captured events and their normalized views. Module 1
// ships a JSONL file sink; module 2 will add a database sink behind the same
// interface without touching the collection or normalize layers.
package sink

import (
	"github.com/jzlxiaohei/agenttape/internal/event"
	"github.com/jzlxiaohei/agenttape/internal/normalize"
)

// Record is one persisted unit: the raw source event plus its normalized view
// (or the error explaining why normalization did not apply, e.g. a hook event or
// an unrecognized provider).
type Record struct {
	Event          *event.SourceEvent            `json:"event"`
	Normalized     *normalize.NormalizedEnvelope `json:"normalized,omitempty"`
	NormalizeError string                        `json:"normalize_error,omitempty"`
}

// Sink stores records. Implementations must be safe for concurrent use.
type Sink interface {
	Write(Record) error
	Close() error
}
