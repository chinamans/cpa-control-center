package backend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestBackendMaintainOnlyActsOnWeeklyQuotaLimitedAccounts(t *testing.T) {
	files := []map[string]any{
		{
			"name":       "weekly-limit.json",
			"type":       "codex",
			"provider":   "codex",
			"auth_index": "weekly-limit",
			"id_token":   `{"chatgpt_account_id":"acct-weekly-limit","plan_type":"pro"}`,
		},
		{
			"name":       "five-hour-limit.json",
			"type":       "codex",
			"provider":   "codex",
			"auth_index": "five-hour-limit",
			"id_token":   `{"chatgpt_account_id":"acct-five-hour-limit","plan_type":"pro"}`,
		},
		{
			"name":       "weekly-recovered.json",
			"type":       "codex",
			"provider":   "codex",
			"auth_index": "weekly-recovered",
			"disabled":   true,
			"id_token":   `{"chatgpt_account_id":"acct-weekly-recovered","plan_type":"pro"}`,
		},
	}

	var mu sync.Mutex
	var disabled []string
	var reenabled []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v0/management/auth-files":
			_ = json.NewEncoder(w).Encode(map[string]any{"files": files})
		case r.Method == http.MethodPost && r.URL.Path == "/v0/management/api-call":
			var body struct {
				AuthIndex string `json:"authIndex"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			switch body.AuthIndex {
			case "weekly-limit":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"status_code": 401,
					"body":        `{"error":{"type":"usage_limit_reached","message":"weekly quota exhausted","plan_type":"pro","resets_in_seconds":602639}}`,
				})
			case "five-hour-limit":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"status_code": 401,
					"body":        `{"error":{"type":"usage_limit_reached","message":"5-hour quota exhausted","plan_type":"pro","resets_in_seconds":16687}}`,
				})
			case "weekly-recovered":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"status_code": 200,
					"body":        `{"plan_type":"pro","rate_limit":{"allowed":true,"limit_reached":false}}`,
				})
			default:
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPatch && r.URL.Path == "/v0/management/auth-files/status":
			var body struct {
				Name     string `json:"name"`
				Disabled bool   `json:"disabled"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			mu.Lock()
			if body.Disabled {
				disabled = append(disabled, body.Name)
			} else {
				reenabled = append(reenabled, body.Name)
			}
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
		default:
			http.NotFound(w, r)
		}
	}))
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
		ProbeWorkers:    3,
		ActionWorkers:   3,
		TimeoutSeconds:  5,
		Retries:         0,
		UserAgent:       defaultUserAgent,
		QuotaAction:     "disable",
		AutoReenable:    true,
	})
	if err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	if err := service.store.UpsertCurrentAccount(AccountRecord{
		Name:             "weekly-recovered.json",
		Type:             "codex",
		Provider:         "codex",
		State:            stateQuotaWeeklyLimited,
		StateKey:         stateQuotaWeeklyLimited,
		QuotaLimited:     true,
		QuotaLimitKind:   "weekly",
		Disabled:         true,
		ManagedReason:    "quota_disabled",
		AuthIndex:        "weekly-recovered",
		ChatGPTAccountID: "acct-weekly-recovered",
		UpdatedAt:        nowISO(),
		LastSeenAt:       nowISO(),
		LastProbedAt:     nowISO(),
	}); err != nil {
		t.Fatalf("UpsertCurrentAccount: %v", err)
	}

	result, err := service.RunMaintain(MaintainOptions{
		QuotaAction:  "disable",
		AutoReenable: true,
	})
	if err != nil {
		t.Fatalf("RunMaintain: %v", err)
	}
	if len(result.QuotaActionResults) != 1 || !result.QuotaActionResults[0].OK || result.QuotaActionResults[0].Name != "weekly-limit.json" {
		t.Fatalf("expected only weekly-limit.json to be disabled, got %+v", result.QuotaActionResults)
	}
	if len(result.ReenableResults) != 1 || !result.ReenableResults[0].OK || result.ReenableResults[0].Name != "weekly-recovered.json" {
		t.Fatalf("expected recovered weekly quota account to be re-enabled, got %+v", result.ReenableResults)
	}

	records, err := service.ListAccounts(AccountFilter{Type: "codex"})
	if err != nil {
		t.Fatalf("ListAccounts: %v", err)
	}
	states := make(map[string]AccountRecord, len(records))
	for _, record := range records {
		states[record.Name] = record
	}
	if states["weekly-limit.json"].StateKey != stateQuotaWeeklyLimited || states["weekly-limit.json"].ManagedReason != "quota_disabled" || !states["weekly-limit.json"].Disabled {
		t.Fatalf("expected weekly-limit.json to be disabled as weekly quota, got %+v", states["weekly-limit.json"])
	}
	if states["five-hour-limit.json"].StateKey != stateQuota5hLimited || states["five-hour-limit.json"].Disabled {
		t.Fatalf("expected five-hour-limit.json to remain enabled 5-hour quota, got %+v", states["five-hour-limit.json"])
	}
	if states["weekly-recovered.json"].StateKey != stateNormal || states["weekly-recovered.json"].Disabled || states["weekly-recovered.json"].ManagedReason != "" {
		t.Fatalf("expected weekly-recovered.json to be enabled and normal, got %+v", states["weekly-recovered.json"])
	}

	mu.Lock()
	defer mu.Unlock()
	if len(disabled) != 1 || disabled[0] != "weekly-limit.json" {
		t.Fatalf("expected only weekly-limit.json disabled, got %+v", disabled)
	}
	if len(reenabled) != 1 || reenabled[0] != "weekly-recovered.json" {
		t.Fatalf("expected weekly-recovered.json re-enabled, got %+v", reenabled)
	}
}
