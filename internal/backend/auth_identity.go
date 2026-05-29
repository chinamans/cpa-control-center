package backend

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const authFilesDirEnv = "CPA_AUTH_FILES_DIR"

type localAuthIdentity struct {
	Name             string
	ID               string
	AuthIndex        string
	Path             string
	Email            string
	ChatGPTAccountID string
	PlanType         string
}

type localAuthIdentityIndex struct {
	byName  map[string]localAuthIdentity
	byEmail map[string]localAuthIdentity
}

func loadLocalAuthIdentityIndex() localAuthIdentityIndex {
	index := localAuthIdentityIndex{
		byName:  make(map[string]localAuthIdentity),
		byEmail: make(map[string]localAuthIdentity),
	}
	for _, dir := range localAuthIdentityDirs() {
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			continue
		}
		_ = filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
				return nil
			}
			identity, ok := readLocalAuthIdentity(path)
			if ok {
				index.add(identity)
			}
			return nil
		})
	}
	return index
}

func localAuthIdentityDirs() []string {
	var dirs []string
	if raw := strings.TrimSpace(os.Getenv(authFilesDirEnv)); raw != "" {
		dirs = append(dirs, filepath.SplitList(raw)...)
	}
	dirs = append(dirs,
		"/auth-files",
		"/mnt/appdata/account-upload-manager-data/processed",
	)

	seen := make(map[string]struct{}, len(dirs))
	unique := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		if _, ok := seen[dir]; ok {
			continue
		}
		seen[dir] = struct{}{}
		unique = append(unique, dir)
	}
	return unique
}

func readLocalAuthIdentity(path string) (localAuthIdentity, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return localAuthIdentity{}, false
	}
	var item map[string]any
	if err := json.Unmarshal(data, &item); err != nil {
		return localAuthIdentity{}, false
	}

	identity := localAuthIdentity{
		Name:             stringOr(strings.TrimSpace(stringValue(item["name"])), filepath.Base(path)),
		ID:               strings.TrimSpace(stringValue(item["id"])),
		AuthIndex:        strings.TrimSpace(stringOr(stringValue(item["auth_index"]), stringValue(item["authIndex"]))),
		Path:             path,
		Email:            strings.TrimSpace(stringValue(item["email"])),
		ChatGPTAccountID: extractChatGPTAccountID(item),
		PlanType:         extractIDTokenPlanType(item),
	}
	if identity.ChatGPTAccountID == "" {
		return localAuthIdentity{}, false
	}
	return identity, true
}

func (index localAuthIdentityIndex) add(identity localAuthIdentity) {
	for _, name := range localAuthIdentityNameCandidates(identity) {
		key := localAuthIdentityKey(name)
		if key == "" {
			continue
		}
		if _, exists := index.byName[key]; !exists {
			index.byName[key] = identity
		}
	}
	if emailKey := localAuthIdentityKey(identity.Email); emailKey != "" {
		if _, exists := index.byEmail[emailKey]; !exists {
			index.byEmail[emailKey] = identity
		}
	}
}

func (index localAuthIdentityIndex) enrich(record AccountRecord) AccountRecord {
	if strings.TrimSpace(record.ChatGPTAccountID) != "" {
		return record
	}
	identity, ok := index.lookup(record)
	if !ok {
		return record
	}
	record.ChatGPTAccountID = identity.ChatGPTAccountID
	if record.Email == "" {
		record.Email = identity.Email
	}
	if record.PlanType == "" {
		record.PlanType = identity.PlanType
	}
	if record.IDTokenPlanType == "" {
		record.IDTokenPlanType = identity.PlanType
	}
	return record
}

func (index localAuthIdentityIndex) lookup(record AccountRecord) (localAuthIdentity, bool) {
	for _, name := range accountRecordIdentityNameCandidates(record) {
		key := localAuthIdentityKey(name)
		if key == "" {
			continue
		}
		if identity, ok := index.byName[key]; ok {
			return identity, true
		}
	}
	if emailKey := localAuthIdentityKey(record.Email); emailKey != "" {
		if identity, ok := index.byEmail[emailKey]; ok {
			return identity, true
		}
	}
	return localAuthIdentity{}, false
}

func localAuthIdentityNameCandidates(identity localAuthIdentity) []string {
	return identityLookupCandidates(
		identity.Name,
		identity.ID,
		identity.AuthIndex,
		identity.Path,
	)
}

func accountRecordIdentityNameCandidates(record AccountRecord) []string {
	return identityLookupCandidates(
		record.Name,
		record.AuthIndex,
	)
}

func identityLookupCandidates(values ...string) []string {
	seen := make(map[string]struct{})
	var candidates []string
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		for _, candidate := range []string{
			value,
			normalizeManagedAccountName(value),
			strings.TrimSuffix(value, ".json"),
			strings.TrimSuffix(normalizeManagedAccountName(value), ".json"),
		} {
			key := localAuthIdentityKey(candidate)
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			candidates = append(candidates, candidate)
		}
	}
	for _, value := range values {
		add(value)
	}
	return candidates
}

func localAuthIdentityKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
