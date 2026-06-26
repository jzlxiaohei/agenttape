// Package providers wires the available provider normalizers into a registry.
// It is the single place that imports every provider sub-package, keeping the
// providers themselves independent of each other (they never import this).
package providers

import (
	"github.com/jzlxiaohei/agenttape/internal/normalize"
	"github.com/jzlxiaohei/agenttape/internal/normalize/anthropic"
	"github.com/jzlxiaohei/agenttape/internal/normalize/openai"
)

// Registry returns a registry with all built-in normalizers registered.
func Registry() *normalize.Registry {
	r := normalize.NewRegistry()
	r.Register(anthropic.New())
	r.Register(openai.NewResponses())
	r.Register(openai.NewChat())
	return r
}
