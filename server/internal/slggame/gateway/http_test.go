package gateway

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/cowpeatechnology/my-immortal-sect/server/internal/slggame/authority"
)

func TestBootstrapCanRestoreLatestSavedSessionAndReset(t *testing.T) {
	server, err := NewAuthorityHTTPServer()
	if err != nil {
		t.Fatalf("create authority http server: %v", err)
	}

	preview := httptest.NewServer(server.Handler())
	defer preview.Close()

	resetResponse := postBootstrap(t, preview.URL, authority.SessionBootstrapModeReset)
	if resetResponse.Snapshot.SessionID != "preview-player/preview-local" {
		t.Fatalf("expected preview-player/preview-local session, got %q", resetResponse.Snapshot.SessionID)
	}
	if resetResponse.Identity.PlayerID != "preview-player" {
		t.Fatalf("expected preview-player identity, got %+v", resetResponse.Identity)
	}
	if resetResponse.Identity.PlayerToken == "" {
		t.Fatalf("expected bootstrap to return a player token")
	}
	if resetResponse.Snapshot.GameTick != 0 {
		t.Fatalf("expected clean reset to start at game tick 0, got %d", resetResponse.Snapshot.GameTick)
	}
	if resetResponse.Snapshot.Stockpile.SpiritWood != 0 {
		t.Fatalf("expected clean reset stockpile to start empty, got %+v", resetResponse.Snapshot.Stockpile)
	}

	commandResponse := postCommand(t, preview.URL, map[string]any{
		"name": "collect_stockpile",
		"payload": map[string]any{
			"resourceKind": "spirit_wood",
			"amount":       1,
			"resourceTile": map[string]any{
				"col": 2,
				"row": 4,
			},
		},
	})
	if commandResponse.Snapshot.Stockpile.SpiritWood != 1 {
		t.Fatalf("expected mutated authority stockpile after collect, got %+v", commandResponse.Snapshot.Stockpile)
	}

	restoreResponse := postBootstrap(t, preview.URL, authority.SessionBootstrapModeRestoreLatest)
	if restoreResponse.Snapshot.GameTick != commandResponse.Snapshot.GameTick {
		t.Fatalf("expected restored snapshot game tick %d, got %d", commandResponse.Snapshot.GameTick, restoreResponse.Snapshot.GameTick)
	}
	if restoreResponse.Snapshot.Stockpile != commandResponse.Snapshot.Stockpile {
		t.Fatalf("expected restored stockpile %+v, got %+v", commandResponse.Snapshot.Stockpile, restoreResponse.Snapshot.Stockpile)
	}
	if len(restoreResponse.Snapshot.ResourceNodes) != len(commandResponse.Snapshot.ResourceNodes) {
		t.Fatalf("expected restored resource-node shape length %d, got %d", len(commandResponse.Snapshot.ResourceNodes), len(restoreResponse.Snapshot.ResourceNodes))
	}
	if restoreResponse.Snapshot.ResourceNodes[0] != commandResponse.Snapshot.ResourceNodes[0] {
		t.Fatalf("expected restored first resource node %+v, got %+v", commandResponse.Snapshot.ResourceNodes[0], restoreResponse.Snapshot.ResourceNodes[0])
	}

	secondResetResponse := postBootstrap(t, preview.URL, authority.SessionBootstrapModeReset)
	if secondResetResponse.Snapshot.Stockpile.SpiritWood != 0 {
		t.Fatalf("expected reset stockpile to clear authority gather progress, got %+v", secondResetResponse.Snapshot.Stockpile)
	}
	if secondResetResponse.Snapshot.GameTick != 0 {
		t.Fatalf("expected reset game tick to return to 0, got %d", secondResetResponse.Snapshot.GameTick)
	}
}

func TestSnapshotAcceptsReturnedPlayerSessionIDWithoutDoublePrefix(t *testing.T) {
	server, err := NewAuthorityHTTPServer()
	if err != nil {
		t.Fatalf("create authority http server: %v", err)
	}

	preview := httptest.NewServer(server.Handler())
	defer preview.Close()

	bootstrap := postBootstrap(t, preview.URL, authority.SessionBootstrapModeReset)
	snapshot := getSnapshot(t, preview.URL, bootstrap.Identity.PlayerSessionID, bootstrap.Identity.PlayerID, bootstrap.Identity.PlayerToken)

	if snapshot.Identity.PlayerSessionID != bootstrap.Identity.PlayerSessionID {
		t.Fatalf("expected stable player session id %q, got %q", bootstrap.Identity.PlayerSessionID, snapshot.Identity.PlayerSessionID)
	}
	if snapshot.Snapshot.SessionID != bootstrap.Snapshot.SessionID {
		t.Fatalf("expected stable snapshot session id %q, got %q", bootstrap.Snapshot.SessionID, snapshot.Snapshot.SessionID)
	}
}

func postBootstrap(t *testing.T, baseURL string, mode authority.SessionBootstrapMode) authority.SessionResponse {
	t.Helper()
	return postJSON[authority.SessionResponse](t, baseURL+"/v1/authority/m1/session/bootstrap", map[string]any{
		"sessionId": "preview-local",
		"playerId":  "preview-player",
		"mode":      mode,
	})
}

func postCommand(t *testing.T, baseURL string, command map[string]any) authority.SessionResponse {
	t.Helper()
	return postJSON[authority.SessionResponse](t, baseURL+"/v1/authority/m1/session/command", map[string]any{
		"sessionId": "preview-local",
		"playerId":  "preview-player",
		"command":   command,
	})
}

func getSnapshot(t *testing.T, baseURL string, sessionID string, playerID string, playerToken string) authority.SessionResponse {
	t.Helper()

	request, err := http.NewRequest(
		http.MethodGet,
		baseURL+"/v1/authority/m1/session/snapshot?sessionId="+url.QueryEscape(sessionID)+"&playerId="+url.QueryEscape(playerID)+"&playerToken="+url.QueryEscape(playerToken),
		nil,
	)
	if err != nil {
		t.Fatalf("build snapshot request: %v", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("do snapshot request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		var errorBody authority.ErrorBody
		_ = json.NewDecoder(response.Body).Decode(&errorBody)
		t.Fatalf("expected status 200, got %d (%s)", response.StatusCode, errorBody.Error)
	}

	var decoded authority.SessionResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode snapshot response: %v", err)
	}
	return decoded
}

func postJSON[T any](t *testing.T, url string, payload map[string]any) T {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}

	request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		var errorBody authority.ErrorBody
		_ = json.NewDecoder(response.Body).Decode(&errorBody)
		t.Fatalf("expected status 200, got %d (%s)", response.StatusCode, errorBody.Error)
	}

	var decoded T
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return decoded
}
