package backend

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestProbeAccountRefreshesMissingChatGPTAccountIDFromInventory(t *testing.T) {
	serverState := &fakeCPAServer{
		files: []map[string]any{
			{
				"name":       "0520-kedaya-christinafisher8309@outlook.com.json",
				"type":       "codex",
				"provider":   "codex",
				"auth_index": "healthy",
				"email":      "christinafisher8309@outlook.com",
				"account_id": "acct-refreshed",
				"id_token":   "",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(serverState.handler))
	defer server.Close()

	dataDir := t.TempDir()
	service, err := New(dataDir, nil)
	if err != nil {
		t.Fatalf("New backend: %v", err)
	}
	defer service.Close()

	_, err = service.SaveSettings(AppSettings{
		BaseURL:         server.URL,
		ManagementToken: "token",
		Locale:          localeEnglish,
		TargetType:      "codex",
		ProbeWorkers:    4,
		ActionWorkers:   2,
		TimeoutSeconds:  5,
		Retries:         0,
		UserAgent:       defaultUserAgent,
		QuotaAction:     "disable",
		ExportDirectory: filepath.Join(dataDir, "exports"),
	})
	if err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	staleRecord := AccountRecord{
		Name:         "0520-kedaya-christinafisher8309@outlook.com.json",
		Type:         "codex",
		Provider:     "codex",
		AuthIndex:    "healthy",
		Email:        "christinafisher8309@outlook.com",
		State:        stateError,
		StateKey:     stateError,
		Status:       stateError,
		LastSeenAt:   nowISO(),
		LastProbedAt: nowISO(),
		UpdatedAt:    nowISO(),
	}
	if err := service.store.UpsertCurrentAccount(staleRecord); err != nil {
		t.Fatalf("UpsertCurrentAccount: %v", err)
	}

	probed, err := service.ProbeAccount(staleRecord.Name)
	if err != nil {
		t.Fatalf("ProbeAccount: %v", err)
	}
	if probed.ChatGPTAccountID != "acct-refreshed" {
		t.Fatalf("expected refreshed account id, got %q", probed.ChatGPTAccountID)
	}
	if probed.ProbeErrorKind == "missing_chatgpt_account_id" || probed.StateKey == stateError {
		t.Fatalf("expected successful probe after refresh, got %+v", probed)
	}

	serverState.mu.Lock()
	defer serverState.mu.Unlock()
	if serverState.fetches != 1 {
		t.Fatalf("expected one auth-files refresh, got %d", serverState.fetches)
	}
	if serverState.apiCalls != 1 || len(serverState.apiAuths) != 1 || serverState.apiAuths[0] != "healthy" {
		t.Fatalf("expected one probe against refreshed auth index, got calls=%d auths=%v", serverState.apiCalls, serverState.apiAuths)
	}
}

func TestProbeAccountRefreshesMissingChatGPTAccountIDFromLocalAuthFile(t *testing.T) {
	serverState := &fakeCPAServer{
		files: []map[string]any{
			{
				"name":       "0520-kedaya-christinafisher8309@outlook.com.json",
				"type":       "codex",
				"provider":   "codex",
				"auth_index": "healthy",
				"email":      "christinafisher8309@outlook.com",
				"id_token":   "",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(serverState.handler))
	defer server.Close()

	dataDir := t.TempDir()
	authDir := t.TempDir()
	t.Setenv(authFilesDirEnv, authDir)
	authFileDir := filepath.Join(authDir, "1c66e924-68a8-4ae4-ac57-99bbd10bb707")
	if err := os.MkdirAll(authFileDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(authFileDir, "0520-kedaya-christinafisher8309@outlook.com.json"), []byte(`{
		"email": "christinafisher8309@outlook.com",
		"account_id": "acct-from-local-json",
		"id_token": ""
	}`), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	service, err := New(dataDir, nil)
	if err != nil {
		t.Fatalf("New backend: %v", err)
	}
	defer service.Close()

	_, err = service.SaveSettings(AppSettings{
		BaseURL:         server.URL,
		ManagementToken: "token",
		Locale:          localeEnglish,
		TargetType:      "codex",
		ProbeWorkers:    4,
		ActionWorkers:   2,
		TimeoutSeconds:  5,
		Retries:         0,
		UserAgent:       defaultUserAgent,
		QuotaAction:     "disable",
		ExportDirectory: filepath.Join(dataDir, "exports"),
	})
	if err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	staleRecord := AccountRecord{
		Name:         "0520-kedaya-christinafisher8309@outlook.com.json",
		Type:         "codex",
		Provider:     "codex",
		AuthIndex:    "healthy",
		Email:        "christinafisher8309@outlook.com",
		State:        stateError,
		StateKey:     stateError,
		Status:       stateError,
		LastSeenAt:   nowISO(),
		LastProbedAt: nowISO(),
		UpdatedAt:    nowISO(),
	}
	if err := service.store.UpsertCurrentAccount(staleRecord); err != nil {
		t.Fatalf("UpsertCurrentAccount: %v", err)
	}

	probed, err := service.ProbeAccount(staleRecord.Name)
	if err != nil {
		t.Fatalf("ProbeAccount: %v", err)
	}
	if probed.ChatGPTAccountID != "acct-from-local-json" {
		t.Fatalf("expected local auth account id, got %q", probed.ChatGPTAccountID)
	}
	if probed.ProbeErrorKind == "missing_chatgpt_account_id" || probed.StateKey == stateError {
		t.Fatalf("expected successful probe after local auth refresh, got %+v", probed)
	}
}
