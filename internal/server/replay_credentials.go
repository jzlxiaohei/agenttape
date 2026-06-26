package server

// replayCredentialConflictMessage explains why replay/curl cannot run yet. Live
// session routing is persisted across a tracelab restart, but credentials are
// deliberately memory-only. Subscription sessions can be rehydrated when the
// still-running client sends one more proxied request; key-mode sessions need the
// user to re-enter the key because the agent only has a placeholder.
func replayCredentialConflictMessage(s *Server, sessionID string) string {
	if s.Sessions.NeedsKey(sessionID) {
		return "this API-key session lost its key on restart; re-enter the key for it, then run again"
	}
	return "no in-memory credentials for that session; if the original subscription client is still running, send one request through it to refresh credentials, otherwise launch a session first"
}
