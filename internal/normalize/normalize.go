package normalize

import (
	"fmt"
	"sort"

	"github.com/jzlxiaohei/agenttape/internal/event"
)

// Normalizer interprets SourceEvents for one provider/wire-API. Detect reports
// how confidently this normalizer recognizes the event (0..1); the registry
// picks the highest scorer. Implementations live in sub-packages and never
// import each other.
type Normalizer interface {
	Name() string
	Detect(ev *event.SourceEvent) (confidence float64, ok bool)
	Normalize(ev *event.SourceEvent) (*NormalizedEnvelope, error)
}

// Registry holds the available normalizers and routes events to the best match.
type Registry struct {
	normalizers []Normalizer
}

// NewRegistry builds an empty registry.
func NewRegistry() *Registry { return &Registry{} }

// Register adds a normalizer. Order does not matter; selection is by confidence.
func (r *Registry) Register(n Normalizer) { r.normalizers = append(r.normalizers, n) }

// Pick returns the highest-confidence normalizer for ev, or nil if none match.
func (r *Registry) Pick(ev *event.SourceEvent) Normalizer {
	type scored struct {
		n Normalizer
		c float64
	}
	var matches []scored
	for _, n := range r.normalizers {
		if c, ok := n.Detect(ev); ok {
			matches = append(matches, scored{n, c})
		}
	}
	if len(matches) == 0 {
		return nil
	}
	sort.SliceStable(matches, func(i, j int) bool { return matches[i].c > matches[j].c })
	return matches[0].n
}

// Normalize selects a normalizer and runs it. Returns an error (not a panic) if
// no provider recognizes the event, so callers can fall back to raw display.
func (r *Registry) Normalize(ev *event.SourceEvent) (*NormalizedEnvelope, error) {
	n := r.Pick(ev)
	if n == nil {
		return nil, fmt.Errorf("normalize: no provider matched event %s", ev.ID)
	}
	env, err := n.Normalize(ev)
	if err != nil {
		return nil, fmt.Errorf("normalize: %s: %w", n.Name(), err)
	}
	if env != nil {
		env.SchemaVersion = EnvelopeSchemaVersion
		env.EventID = ev.ID
	}
	return env, nil
}
