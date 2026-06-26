package server

import (
	"agenttape/internal/source/httpcap"
	"agenttape/internal/store"
)

// liveSessionPersister adapts the store to httpcap.SessionPersister, keeping the
// httpcap adapter free of any store dependency. It translates the non-secret
// SessionRecord to/from the store's LiveSessionRow — credentials never pass through.
type liveSessionPersister struct{ st *store.Store }

func (p *liveSessionPersister) SaveSession(r httpcap.SessionRecord) error {
	return p.st.SaveLiveSession(store.LiveSessionRow{
		ID: r.ID, Token: r.Token, Client: r.Client,
		Upstream: r.Upstream, Provider: r.Provider, Mode: r.Mode,
	})
}

func (p *liveSessionPersister) DeleteSession(id string) error {
	return p.st.DeleteLiveSession(id)
}

func (p *liveSessionPersister) AllSessions() ([]httpcap.SessionRecord, error) {
	rows, err := p.st.AllLiveSessions()
	if err != nil {
		return nil, err
	}
	out := make([]httpcap.SessionRecord, 0, len(rows))
	for _, r := range rows {
		out = append(out, httpcap.SessionRecord{
			ID: r.ID, Token: r.Token, Client: r.Client,
			Upstream: r.Upstream, Provider: r.Provider, Mode: r.Mode,
		})
	}
	return out, nil
}
