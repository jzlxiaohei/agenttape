// Package source defines the collection layer. Every adapter (HTTP capture,
// hooks, importers) converts whatever it observes into a single
// event.SourceEvent and hands it to an Emitter. Downstream code never learns how
// the data arrived — that decoupling is the whole point (see next.md 1.2).
package source

import (
	"net/http"

	"github.com/jzlxiaohei/agenttape/internal/event"
)

// Emitter receives a fully-built SourceEvent from an adapter.
type Emitter func(*event.SourceEvent)

// Adapter is one data source. It exposes an HTTP handler mounted by the server;
// concurrent sources share one process and are told apart only by their route.
type Adapter interface {
	Name() string
	Handler() http.Handler
}

// IDGen produces unique, opaque identifiers for events and session tokens.
type IDGen interface {
	NewID() string
}
