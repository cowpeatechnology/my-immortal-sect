package gateway

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/cowpeatechnology/my-immortal-sect/server/internal/slggame/authority"
)

const requestTimeout = 3 * time.Second

type AuthorityHTTPServer struct {
	engine     *actor.Engine
	sessionPID *actor.PID
	mux        *http.ServeMux
}

func NewAuthorityHTTPServer() (*AuthorityHTTPServer, error) {
	engine, err := actor.NewEngine(actor.NewEngineConfig())
	if err != nil {
		return nil, err
	}

	sessionPID := engine.Spawn(authority.NewSessionActor, "m1-authority-session")
	server := &AuthorityHTTPServer{
		engine:     engine,
		sessionPID: sessionPID,
		mux:        http.NewServeMux(),
	}
	server.routes()
	return server, nil
}

func (s *AuthorityHTTPServer) Handler() http.Handler {
	return s.withCORS(s.mux)
}

func (s *AuthorityHTTPServer) routes() {
	s.mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	s.mux.HandleFunc("/v1/authority/m1/session/bootstrap", s.handleBootstrap)
	s.mux.HandleFunc("/v1/authority/m1/session/snapshot", s.handleSnapshot)
	s.mux.HandleFunc("/v1/authority/m1/session/command", s.handleCommand)
}

func (s *AuthorityHTTPServer) handleBootstrap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	var body struct {
		SessionID string `json:"sessionId"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	response, err := s.request(authority.BootstrapSession{SessionID: body.SessionID})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *AuthorityHTTPServer) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	response, err := s.request(authority.GetSessionSnapshot{SessionID: r.URL.Query().Get("sessionId")})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *AuthorityHTTPServer) handleCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	var body struct {
		SessionID string                    `json:"sessionId"`
		Command   authority.CommandEnvelope `json:"command"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	response, err := s.request(authority.ExecuteCommand{
		SessionID: body.SessionID,
		Command:   body.Command,
	})
	if err != nil {
		status := http.StatusBadRequest
		var syntaxErr *json.SyntaxError
		if errors.As(err, &syntaxErr) {
			status = http.StatusBadRequest
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *AuthorityHTTPServer) request(message any) (authority.SessionResponse, error) {
	result, err := s.engine.Request(s.sessionPID, message, requestTimeout).Result()
	if err != nil {
		return authority.SessionResponse{}, err
	}
	if messageErr, ok := result.(error); ok {
		return authority.SessionResponse{}, messageErr
	}

	response, ok := result.(authority.SessionResponse)
	if !ok {
		return authority.SessionResponse{}, errors.New("unexpected actor response type")
	}
	return response, nil
}

func (s *AuthorityHTTPServer) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func decodeJSON(r *http.Request, target any) error {
	if r.Body == nil {
		return nil
	}
	defer r.Body.Close()
	if r.ContentLength == 0 {
		return nil
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func writeMethodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, authority.ErrorBody{Error: "method_not_allowed"})
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, authority.ErrorBody{Error: err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
