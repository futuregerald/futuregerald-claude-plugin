package state

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/futuregerald/futuregerald-claude-plugin/internal/backup"
	"github.com/futuregerald/futuregerald-claude-plugin/internal/catalog"
	"github.com/futuregerald/futuregerald-claude-plugin/internal/platform"
)

type Receipt struct {
	ID               string   `json:"id"`
	Harness          string   `json:"harness"`
	Scope            string   `json:"scope"`
	RootDir          string   `json:"root_dir"`
	Track            string   `json:"track"`
	Skills           []string `json:"skills"`
	InstalledFiles   []string `json:"installed_files"`
	HooksMerged      bool     `json:"hooks_merged"`
	HooksFileCreated bool     `json:"hooks_file_created"`
	Version          string   `json:"version"`
	CommitSHA        string   `json:"commit_sha,omitempty"`
	LastBackupID     string   `json:"last_backup_id,omitempty"`
	UpdatedAt        string   `json:"updated_at"`
}

type statePayload struct {
	Receipts []Receipt `json:"receipts"`
}

type UninstallResult struct {
	Harness        string
	Scope          string
	RemovedFiles   []string
	HooksCleaned   bool
	RestoredConfig string
}

type Store struct {
	FilePath string
}

func NewStore(filePath string) *Store {
	return &Store{FilePath: filePath}
}

func ReceiptKey(harness, scope, rootDir string) string {
	cleanRoot := filepath.Clean(rootDir)
	return fmt.Sprintf("%s:%s:%s", strings.ToLower(harness), strings.ToLower(scope), cleanRoot)
}

func (s *Store) Load() ([]Receipt, error) {
	data, err := os.ReadFile(s.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var payload statePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("parsing state.json: %w", err)
	}
	return payload.Receipts, nil
}

func (s *Store) Save(receipts []Receipt) error {
	if s.FilePath == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.FilePath), 0700); err != nil {
		return err
	}
	payload := statePayload{Receipts: receipts}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.FilePath, append(data, '\n'), 0600)
}

func (s *Store) Upsert(r Receipt) error {
	r.RootDir = filepath.Clean(r.RootDir)
	if r.ID == "" {
		r.ID = ReceiptKey(r.Harness, r.Scope, r.RootDir)
	}
	if r.UpdatedAt == "" {
		r.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}

	receipts, err := s.Load()
	if err != nil {
		return err
	}
	key := ReceiptKey(r.Harness, r.Scope, r.RootDir)
	replaced := false
	for i := range receipts {
		if ReceiptKey(receipts[i].Harness, receipts[i].Scope, receipts[i].RootDir) == key {
			receipts[i] = r
			replaced = true
			break
		}
	}
	if !replaced {
		receipts = append(receipts, r)
	}
	return s.Save(receipts)
}

func (s *Store) Find(harness, scope, rootDir string) (*Receipt, bool, error) {
	receipts, err := s.Load()
	if err != nil {
		return nil, false, err
	}
	key := ReceiptKey(harness, scope, rootDir)
	for _, r := range receipts {
		if ReceiptKey(r.Harness, r.Scope, r.RootDir) == key {
			cp := r
			return &cp, true, nil
		}
	}
	return nil, false, nil
}

func (s *Store) List(harnessFilter string) ([]Receipt, error) {
	receipts, err := s.Load()
	if err != nil {
		return nil, err
	}
	if harnessFilter == "" {
		return receipts, nil
	}
	var out []Receipt
	for _, r := range receipts {
		if strings.EqualFold(r.Harness, harnessFilter) {
			out = append(out, r)
		}
	}
	return out, nil
}

func (s *Store) Remove(harness, scope, rootDir string) error {
	receipts, err := s.Load()
	if err != nil {
		return err
	}
	key := ReceiptKey(harness, scope, rootDir)
	var kept []Receipt
	for _, r := range receipts {
		if ReceiptKey(r.Harness, r.Scope, r.RootDir) != key {
			kept = append(kept, r)
		}
	}
	return s.Save(kept)
}

func isPreservedLocalFile(filePath string) bool {
	slashed := filepath.ToSlash(filePath)
	return strings.HasSuffix(slashed, ".local.md") || strings.HasSuffix(slashed, "/references/config.md")
}

var knownBreakerFiles = []string{
	"autonomous-breaker.sh",
	"autonomous_breaker.py",
	"breaker_rules.py",
	"replay_breaker.py",
	"session_metrics.py",
}

func RemoveCircuitBreakerHook(hooksJSONPath, hooksDir string, hooksFileCreated bool) (bool, error) {
	cleaned := false
	if hooksDir != "" {
		for _, bf := range knownBreakerFiles {
			p := filepath.Join(hooksDir, bf)
			if _, err := os.Lstat(p); err == nil {
				if err := os.Remove(p); err == nil {
					cleaned = true
				}
			}
		}
		backup.PruneEmptyParents(hooksDir, 1)
	}

	if hooksJSONPath == "" {
		return cleaned, nil
	}
	data, err := os.ReadFile(hooksJSONPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cleaned, nil
		}
		return cleaned, err
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		return cleaned, fmt.Errorf("parsing %s: %w", hooksJSONPath, err)
	}
	if _, exists := parsed["autonomous-circuit-breaker"]; !exists {
		return cleaned, nil
	}

	delete(parsed, "autonomous-circuit-breaker")
	cleaned = true

	if len(parsed) == 0 && hooksFileCreated {
		if err := os.Remove(hooksJSONPath); err != nil && !os.IsNotExist(err) {
			return cleaned, err
		}
		return cleaned, nil
	}

	out, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return cleaned, err
	}
	if err := os.WriteFile(hooksJSONPath, append(out, '\n'), 0644); err != nil {
		return cleaned, err
	}
	return cleaned, nil
}

func DiscoverLegacyReceipt(env platform.Env, cat *catalog.Catalog, harness, scope, projectDir string) (*Receipt, bool) {
	paths, err := env.ResolvePaths(harness, scope, projectDir)
	if err != nil {
		return nil, false
	}

	knownSkills := make(map[string]bool)
	for _, sm := range cat.SkillsForHarness(harness) {
		knownSkills[sm.Name] = true
	}

	var foundSkills []string
	var installedFiles []string

	if paths.SkillsDir != "" {
		if entries, err := os.ReadDir(paths.SkillsDir); err == nil {
			for _, entry := range entries {
				if !entry.IsDir() || !knownSkills[entry.Name()] {
					continue
				}
				skillRoot := filepath.Join(paths.SkillsDir, entry.Name())
				foundSkills = append(foundSkills, entry.Name())
				_ = filepath.WalkDir(skillRoot, func(p string, d fs.DirEntry, wErr error) error {
					if wErr != nil || d.IsDir() || isPreservedLocalFile(p) {
						return nil
					}
					installedFiles = append(installedFiles, filepath.Clean(p))
					return nil
				})
			}
		}
	}

	if paths.AgentsDir != "" {
		agentsSrc := path.Join(harness, "agents")
		if harness == catalog.HarnessAntigravity {
			agentsSrc = "antigravity/plugins/antigravity-subagents/agents"
		}
		if entries, err := fs.ReadDir(cat.FS(), agentsSrc); err == nil {
			for _, entry := range entries {
				if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
					continue
				}
				candidate := filepath.Join(paths.AgentsDir, entry.Name())
				if info, err := os.Lstat(candidate); err == nil && !info.IsDir() {
					installedFiles = append(installedFiles, filepath.Clean(candidate))
				}
			}
		}
		if paths.PluginJSON != "" {
			if info, err := os.Lstat(paths.PluginJSON); err == nil && !info.IsDir() {
				installedFiles = append(installedFiles, filepath.Clean(paths.PluginJSON))
			}
		}
	}

	if paths.CommandsDir != "" {
		cmdSrc := path.Join(harness, "commands")
		_ = fs.WalkDir(cat.FS(), cmdSrc, func(p string, d fs.DirEntry, wErr error) error {
			if wErr != nil || d.IsDir() || !strings.HasSuffix(p, ".md") {
				return nil
			}
			rel := strings.TrimPrefix(p, cmdSrc+"/")
			candidate := filepath.Join(paths.CommandsDir, filepath.FromSlash(rel))
			if info, err := os.Lstat(candidate); err == nil && !info.IsDir() {
				installedFiles = append(installedFiles, filepath.Clean(candidate))
			}
			return nil
		})
	}

	hooksMerged := false
	if paths.HooksJSON != "" {
		if data, err := os.ReadFile(paths.HooksJSON); err == nil {
			var parsed map[string]any
			if json.Unmarshal(data, &parsed) == nil {
				if _, ok := parsed["autonomous-circuit-breaker"]; ok {
					hooksMerged = true
				}
			}
		}
	}

	if len(installedFiles) == 0 && !hooksMerged {
		return nil, false
	}

	sort.Strings(foundSkills)
	sort.Strings(installedFiles)

	track := "custom"
	skillSet := make(map[string]bool, len(foundSkills))
	for _, s := range foundSkills {
		skillSet[s] = true
	}
	if skillSet["start"] || skillSet["lifecycle"] {
		track = "opinionated"
	}

	return &Receipt{
		ID:             ReceiptKey(harness, scope, paths.RootDir),
		Harness:        harness,
		Scope:          scope,
		RootDir:        filepath.Clean(paths.RootDir),
		Track:          track,
		Skills:         foundSkills,
		InstalledFiles: installedFiles,
		HooksMerged:    hooksMerged,
		Version:        "3.x",
		UpdatedAt:      time.Now().UTC().Format(time.RFC3339),
	}, true
}

func Uninstall(env platform.Env, _ *catalog.Catalog, receipt Receipt, backupMgr *backup.Manager, restoreConfig bool) (*UninstallResult, error) {
	paths, err := env.ResolvePaths(receipt.Harness, receipt.Scope, receipt.RootDir)
	if err != nil {
		return nil, err
	}

	res := &UninstallResult{
		Harness: receipt.Harness,
		Scope:   receipt.Scope,
	}

	cleanConfig := ""
	if paths.ConfigFile != "" {
		cleanConfig = filepath.Clean(paths.ConfigFile)
	}
	cleanHooksJSON := ""
	if paths.HooksJSON != "" {
		cleanHooksJSON = filepath.Clean(paths.HooksJSON)
	}

	if restoreConfig && backupMgr != nil && receipt.LastBackupID != "" && cleanConfig != "" {
		restored, err := backupMgr.RestoreSingleFile(receipt.LastBackupID, cleanConfig)
		if err == nil && restored {
			res.RestoredConfig = cleanConfig
		}
	}

	for _, f := range receipt.InstalledFiles {
		cleanFile := filepath.Clean(f)
		if isPreservedLocalFile(cleanFile) {
			continue
		}
		if cleanHooksJSON != "" && cleanFile == cleanHooksJSON {
			continue
		}
		if cleanConfig != "" && cleanFile == cleanConfig {
			if res.RestoredConfig != "" || !restoreConfig {
				continue
			}
		}
		if _, err := os.Lstat(cleanFile); err == nil {
			if err := os.Remove(cleanFile); err == nil {
				res.RemovedFiles = append(res.RemovedFiles, cleanFile)
				backup.PruneEmptyParents(filepath.Dir(cleanFile), 3)
			}
		}
	}

	if receipt.HooksMerged || (receipt.Harness == catalog.HarnessAntigravity && receipt.Scope == "global") {
		cleaned, err := RemoveCircuitBreakerHook(paths.HooksJSON, paths.HooksDir, receipt.HooksFileCreated)
		if err != nil {
			return nil, err
		}
		res.HooksCleaned = cleaned
	}

	return res, nil
}
