package authority

import "github.com/anthdm/hollywood/actor"

type SessionActor struct {
	sessions map[string]*sessionState
}

func NewSessionActor() actor.Receiver {
	return &SessionActor{
		sessions: map[string]*sessionState{},
	}
}

func (a *SessionActor) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Initialized:
		return
	case BootstrapSession:
		sessionID := normalizedSessionID(msg.SessionID)
		session := newSessionState(sessionID)
		a.sessions[sessionID] = session
		ctx.Respond(SessionResponse{Snapshot: session.snapshot()})
	case GetSessionSnapshot:
		session := a.getOrCreateSession(msg.SessionID)
		ctx.Respond(SessionResponse{Snapshot: session.snapshot()})
	case ExecuteCommand:
		session := a.getOrCreateSession(msg.SessionID)
		result, err := session.executeCommand(msg.Command)
		if err != nil {
			ctx.Respond(err)
			return
		}
		ctx.Respond(SessionResponse{
			Snapshot: session.snapshot(),
			Result:   &result,
		})
	}
}

func (a *SessionActor) getOrCreateSession(sessionID string) *sessionState {
	key := normalizedSessionID(sessionID)
	if session := a.sessions[key]; session != nil {
		return session
	}
	session := newSessionState(key)
	a.sessions[key] = session
	return session
}

func normalizedSessionID(sessionID string) string {
	if sessionID == "" {
		return "preview-local"
	}
	return sessionID
}
