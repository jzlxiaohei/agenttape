package server

import (
	"net/http"
	"strings"
)

// agentProvider captures everything provider-specific about launching a coding
// agent through agenttape's proxy and re-attaching its session after a restart.
// Adding a new provider is meant to be ONE entry in agentProviders below — no
// other code on the launch / inject / re-attach path branches on which provider
// it is; they all go through this table.
//
// Implemented today: cc (Anthropic / Claude Code) and codex (OpenAI / Codex CLI).
// Other providers are intentionally NOT implemented — but the seam is here. To add,
// say, Gemini: append an entry with its subscription/api-key upstreams, the env var
// its CLI reads the key from, and how a raw API key becomes auth headers. See
// docs/SECURITY.md §"Adding a provider" for the full checklist and the security
// invariants any new provider must keep (placeholder swap, no secrets on disk).
type agentProvider struct {
	Kind     string // launch kind + UI selector: "cc", "codex"
	Client   string // capture/hook client id: "claude_code", "codex_cli"
	Provider string // normalize/wire id, persisted on the session: "anthropic", "openai-responses"
	KeyEnv   string // env var the agent CLI reads its API key from
	SubURL   string // subscription-mode upstream
	KeyURL   string // api-key-mode upstream (subscription tokens and API keys hit different hosts)
	// injectAuth turns a real API key into the header(s) the proxy injects on
	// forward, so the agent itself only ever holds a placeholder. This is the ONE
	// place a provider's key→header shape is defined; the re-inject endpoint reuses it.
	injectAuth func(apiKey string) http.Header
}

var agentProviders = map[string]agentProvider{
	"cc": {
		Kind:     "cc",
		Client:   "claude_code",
		Provider: "anthropic",
		KeyEnv:   "ANTHROPIC_API_KEY",
		SubURL:   "https://api.anthropic.com",
		KeyURL:   "https://api.anthropic.com",
		injectAuth: func(key string) http.Header {
			h := http.Header{}
			h.Set("Content-Type", "application/json")
			h.Set("x-api-key", key)
			h.Set("anthropic-version", "2023-06-01")
			return h
		},
	},
	"codex": {
		Kind:     "codex",
		Client:   "codex_cli",
		Provider: "openai-responses",
		KeyEnv:   "OPENAI_API_KEY",
		// codex splits by mode: a ChatGPT-subscription token only works against the
		// Codex backend, while a real OpenAI API key only works against the platform
		// API. Getting this wrong surfaces as 401 "missing scopes: api.responses.write".
		SubURL: "https://chatgpt.com/backend-api/codex",
		KeyURL: "https://api.openai.com/v1",
		injectAuth: func(key string) http.Header {
			h := http.Header{}
			h.Set("Content-Type", "application/json")
			h.Set("Authorization", "Bearer "+key)
			return h
		},
	},
}

// agentProviderByKind looks up a spec by launch kind ("cc" | "codex").
func agentProviderByKind(kind string) (agentProvider, bool) {
	p, ok := agentProviders[kind]
	return p, ok
}

// agentProviderByProvider finds a spec by its persisted provider id. Used when
// re-injecting a key into a session restored after a restart, where all we kept is
// the (non-secret) provider id — not which UI "kind" the user originally picked.
func agentProviderByProvider(provider string) (agentProvider, bool) {
	for _, p := range agentProviders {
		if p.Provider == provider {
			return p, true
		}
	}
	return agentProvider{}, false
}

// upstreamFor returns the upstream URL for this provider in the given credential
// mode ("key" | "subscription").
func (p agentProvider) upstreamFor(mode string) string {
	if mode == "key" {
		return p.KeyURL
	}
	return p.SubURL
}

// providerForUpstream infers the wire/provider id from an upstream URL. Used by the
// low-level /_register path (CLI launch), where only client+upstream are known. The
// Anthropic host is unmistakable; everything else is treated as the OpenAI responses
// wire (the only other implemented provider).
func providerForUpstream(upstream string) string {
	if strings.Contains(upstream, "anthropic") {
		return agentProviders["cc"].Provider
	}
	return agentProviders["codex"].Provider
}
