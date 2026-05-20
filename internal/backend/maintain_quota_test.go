package backend

import (
	"context"
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

func TestBackendMaintainRecoversDisabledWeeklyAccountWithoutManagedReason(t *testing.T) {
	files := []map[string]any{
		{
			"name":       "external-disabled-recovered.json",
			"type":       "codex",
			"provider":   "codex",
			"auth_index": "external-disabled-recovered",
			"disabled":   true,
			"id_token":   `{"chatgpt_account_id":"acct-external-recovered","plan_type":"pro"}`,
		},
	}

	var mu sync.Mutex
	var reenabled []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v0/management/auth-files":
			_ = json.NewEncoder(w).Encode(map[string]any{"files": files})
		case r.Method == http.MethodPost && r.URL.Path == "/v0/management/api-call":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status_code": 200,
				"body":        `{"plan_type":"pro","rate_limit":{"allowed":true,"limit_reached":false}}`,
			})
		case r.Method == http.MethodPatch && r.URL.Path == "/v0/management/auth-files/status":
			var body struct {
				Name     string `json:"name"`
				Disabled bool   `json:"disabled"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if !body.Disabled {
				mu.Lock()
				reenabled = append(reenabled, body.Name)
				mu.Unlock()
			}
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
		ProbeWorkers:    1,
		ActionWorkers:   1,
		TimeoutSeconds:  5,
		Retries:         0,
		UserAgent:       defaultUserAgent,
		QuotaAction:     "disable",
		AutoReenable:    true,
	})
	if err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	result, err := service.RunMaintain(MaintainOptions{
		QuotaAction:  "disable",
		AutoReenable: true,
	})
	if err != nil {
		t.Fatalf("RunMaintain: %v", err)
	}
	if result.Scan.RecoveredCount != 1 {
		t.Fatalf("expected scan to count one recovered account, got %+v", result.Scan)
	}
	if len(result.ReenableResults) != 1 || !result.ReenableResults[0].OK || result.ReenableResults[0].Name != "external-disabled-recovered.json" {
		t.Fatalf("expected disabled recovered account to be re-enabled, got %+v", result.ReenableResults)
	}

	records, err := service.ListAccounts(AccountFilter{Type: "codex"})
	if err != nil {
		t.Fatalf("ListAccounts: %v", err)
	}
	if len(records) != 1 || records[0].StateKey != stateNormal || records[0].Disabled || records[0].ManagedReason != "" {
		t.Fatalf("expected account to end enabled and normal, got %+v", records)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(reenabled) != 1 || reenabled[0] != "external-disabled-recovered.json" {
		t.Fatalf("expected external-disabled-recovered.json re-enabled, got %+v", reenabled)
	}
}

func TestDisabledInventoryAccountWithoutManagedReasonCountsAsWeeklyQuotaLimited(t *testing.T) {
	files := []map[string]any{
		{
			"name":       "external-disabled-weekly.json",
			"type":       "codex",
			"provider":   "codex",
			"auth_index": "external-disabled-weekly",
			"disabled":   true,
			"id_token":   `{"chatgpt_account_id":"acct-external-weekly","plan_type":"pro"}`,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v0/management/auth-files" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"files": files})
	}))
	defer server.Close()

	service, err := New(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("New backend: %v", err)
	}
	defer service.Close()

	_, err = service.SaveSettings(AppSettings{
		BaseURL:         server.URL,
		ManagementToken: "token",
		Locale:          localeEnglish,
		TargetType:      "codex",
		ProbeWorkers:    1,
		ActionWorkers:   1,
		TimeoutSeconds:  5,
		Retries:         0,
		UserAgent:       defaultUserAgent,
		QuotaAction:     "disable",
	})
	if err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	if _, err := service.SyncInventory(); err != nil {
		t.Fatalf("SyncInventory: %v", err)
	}

	snapshot, err := service.GetDashboardSnapshot()
	if err != nil {
		t.Fatalf("GetDashboardSnapshot: %v", err)
	}
	if snapshot.Summary.QuotaWeeklyLimitedCount != 1 || snapshot.Summary.QuotaLimitedCount != 1 || snapshot.Summary.PendingCount != 0 {
		t.Fatalf("expected disabled inventory account to count as weekly quota limited, got %+v", snapshot.Summary)
	}

	records, err := service.ListAccounts(AccountFilter{Type: "codex", State: stateQuotaWeeklyLimited})
	if err != nil {
		t.Fatalf("ListAccounts: %v", err)
	}
	if len(records) != 1 || records[0].Name != "external-disabled-weekly.json" || records[0].ManagedReason != "quota_disabled" {
		t.Fatalf("expected weekly quota disabled record, got %+v", records)
	}
}

func TestDisabledWeeklyAccountProbeErrorKeepsWeeklyQuotaLimitedState(t *testing.T) {
	record := AccountRecord{
		Name:          "external-disabled-missing-id.json",
		Type:          "codex",
		Provider:      "codex",
		Disabled:      true,
		ManagedReason: "quota_disabled",
		State:         stateQuotaWeeklyLimited,
		StateKey:      stateQuotaWeeklyLimited,
	}

	probed := NewClient().ProbeUsage(context.Background(), AppSettings{Locale: localeEnglish}, record)
	if probed.StateKey != stateQuotaWeeklyLimited || probed.QuotaLimitKind != "weekly" || !probed.QuotaLimited || probed.Error {
		t.Fatalf("expected disabled weekly account to keep weekly quota state on probe error, got %+v", probed)
	}
}
