// Package providers wires the available provider normalizers into a registry.
// It is the single place that imports every provider sub-package, keeping the
// providers themselves independent of each other (they never import this).
package providers

import (
	"agenttape/internal/normalize"
	"agenttape/internal/normalize/anthropic"
	"agenttape/internal/normalize/openai"
)

// Registry returns a registry with all built-in normalizers registered.
func Registry() *normalize.Registry {
	r := normalize.NewRegistry()
	r.Register(anthropic.New())
	r.Register(openai.NewResponses())
	r.Register(openai.NewChat())
	return r
}
