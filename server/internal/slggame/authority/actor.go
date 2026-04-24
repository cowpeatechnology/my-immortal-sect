package authority

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/anthdm/hollywood/actor"
)

const authorityTickInterval = time.Second

type advanceAuthorityClock struct {
	Seconds int
}

type SessionActor struct {
	sessions      map[string]*sessionState
	savedSessions map[string]AuthoritySessionSaveEnvelope
	playerTokens  map[string]string
	tickRepeater  actor.SendRepeater
	tickRepeating bool
}

func NewSessionActor() actor.Receiver {
	return &SessionActor{
		sessions:      map[string]*sessionState{},
		savedSessions: map[string]AuthoritySessionSaveEnvelope{},
		playerTokens:  map[string]string{},
	}
}

func (a *SessionActor) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Initialized:
		a.tickRepeater = ctx.SendRepeat(ctx.PID(), advanceAuthorityClock{Seconds: 1}, authorityTickInterval)
		a.tickRepeating = true
		return
	case actor.Stopped:
		if a.tickRepeating {
			a.tickRepeater.Stop()
			a.tickRepeating = false
		}
		return
	case BootstrapSession:
		identity, sessionID, err := a.resolveIdentity(msg.PlayerID, msg.PlayerToken, msg.SessionID)
		if err != nil {
			ctx.Respond(err)
			return
		}
		session, err := a.bootstrapSession(sessionID, msg.Mode)
		if err != nil {
			ctx.Respond(err)
			return
		}
		a.sessions[sessionID] = session
		ctx.Respond(SessionResponse{Identity: identity, Snapshot: session.snapshot()})
	case advanceAuthorityClock:
		for _, session := range a.sessions {
			session.advanceResourceNodes(msg.Seconds)
			a.captureSessionSave(session)
		}
	case GetSessionSnapshot:
		identity, sessionID, err := a.resolveIdentity(msg.PlayerID, msg.PlayerToken, msg.SessionID)
		if err != nil {
			ctx.Respond(err)
			return
		}
		session := a.getOrCreateSession(sessionID)
		ctx.Respond(SessionResponse{Identity: identity, Snapshot: session.snapshot()})
	case ExportSessionSave:
		session := a.getOrCreateSession(msg.SessionID)
		save, err := session.exportSaveEnvelope()
		if err != nil {
			ctx.Respond(err)
			return
		}
		ctx.Respond(save)
	case ImportSessionSave:
		session, err := restoreSessionStateFromSaveEnvelope(msg.Save)
		if err != nil {
			ctx.Respond(err)
			return
		}
		a.sessions[normalizedSessionID(session.SessionID)] = session
		a.captureSessionSave(session)
		ctx.Respond(SessionResponse{Snapshot: session.snapshot()})
	case ExecuteCommand:
		identity, sessionID, err := a.resolveIdentity(msg.PlayerID, msg.PlayerToken, msg.SessionID)
		if err != nil {
			ctx.Respond(err)
			return
		}
		session := a.getOrCreateSession(sessionID)
		result, err := session.executeCommand(msg.Command)
		if err != nil {
			ctx.Respond(err)
			return
		}
		a.captureSessionSave(session)
		ctx.Respond(SessionResponse{
			Identity: identity,
			Snapshot: session.snapshot(),
			Result:   &result,
		})
	case SyncExternalDefenseContext:
		session := a.getOrCreateSession(msg.SessionID)
		session.syncExternalDefenseContext(msg.Context)
		a.captureSessionSave(session)
		ctx.Respond(SessionResponse{Snapshot: session.snapshot()})
	}
}

func (a *SessionActor) getOrCreateSession(sessionID string) *sessionState {
	key := normalizedSessionID(sessionID)
	if session := a.sessions[key]; session != nil {
		a.captureSessionSave(session)
		return session
	}
	session := newSessionState(key)
	a.sessions[key] = session
	a.captureSessionSave(session)
	return session
}

func (a *SessionActor) bootstrapSession(sessionID string, mode SessionBootstrapMode) (*sessionState, error) {
	if normalizeBootstrapMode(mode) == SessionBootstrapModeReset {
		session := newSessionState(sessionID)
		a.captureSessionSave(session)
		return session, nil
	}

	if save, ok := a.savedSessions[sessionID]; ok {
		session, err := restoreSessionStateFromSaveEnvelope(save)
		if err != nil {
			return nil, err
		}
		a.captureSessionSave(session)
		return session, nil
	}
	return a.getOrCreateSession(sessionID), nil
}

func (a *SessionActor) captureSessionSave(session *sessionState) {
	save, err := session.exportSaveEnvelope()
	if err != nil {
		return
	}
	a.savedSessions[normalizedSessionID(session.SessionID)] = save
}

func normalizeBootstrapMode(mode SessionBootstrapMode) SessionBootstrapMode {
	switch mode {
	case SessionBootstrapModeReset:
		return SessionBootstrapModeReset
	case SessionBootstrapModeRestoreLatest, "":
		return SessionBootstrapModeRestoreLatest
	default:
		return SessionBootstrapModeRestoreLatest
	}
}

func normalizedSessionID(sessionID string) string {
	if sessionID == "" {
		return "preview-local"
	}
	return sessionID
}

func (a *SessionActor) resolveIdentity(playerID, playerToken, sessionID string) (PlayerIdentity, string, error) {
	normalizedPlayerID := normalizePlayerID(playerID)
	boundToken, ok := a.playerTokens[normalizedPlayerID]
	if !ok {
		if playerToken == "" {
			generated, err := generatePreviewToken()
			if err != nil {
				return PlayerIdentity{}, "", err
			}
			playerToken = generated
		}
		a.playerTokens[normalizedPlayerID] = playerToken
		boundToken = playerToken
	} else if playerToken != "" && playerToken != boundToken {
		return PlayerIdentity{}, "", fmt.Errorf("player token mismatch for %s", normalizedPlayerID)
	}

	canonicalSessionID := normalizePlayerSessionID(normalizedPlayerID, normalizedSessionID(sessionID))
	return PlayerIdentity{
		PlayerID:        normalizedPlayerID,
		PlayerToken:     boundToken,
		PlayerSessionID: canonicalSessionID,
	}, canonicalSessionID, nil
}

func normalizePlayerID(playerID string) string {
	if playerID == "" {
		return "preview-player"
	}
	return playerID
}

func normalizePlayerSessionID(playerID, sessionID string) string {
	prefix := fmt.Sprintf("%s/", playerID)
	if strings.HasPrefix(sessionID, prefix) {
		return sessionID
	}
	return fmt.Sprintf("%s/%s", playerID, sessionID)
}

func generatePreviewToken() (string, error) {
	var buf [12]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("generate preview token: %w", err)
	}
	return hex.EncodeToString(buf[:]), nil
}
