package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"
)

var snapshotSeq uint64

type BlobEntry struct {
	BlobID       string `json:"blob_id,omitempty"`
	OriginalPath string `json:"original_path"`
	Mode         uint32 `json:"mode,omitempty"`
	Existed      bool   `json:"existed"`
}

type Manifest struct {
	ID             string      `json:"id"`
	Harness        string      `json:"harness"`
	Scope          string      `json:"scope"`
	Track          string      `json:"track"`
	CreatedAt      string      `json:"created_at"`
	HasStateBackup bool        `json:"has_state_backup"`
	Entries        []BlobEntry `json:"entries"`
}

type RestoreResult struct {
	SnapshotID    string
	RestoredFiles []string
	RemovedFiles  []string
	StateRestored bool
}

type Manager struct {
	BackupsRoot  string
	MaxSnapshots int
}

func NewManager(backupsRoot string) *Manager {
	return &Manager{
		BackupsRoot:  backupsRoot,
		MaxSnapshots: 10,
	}
}

func sanitizeSlug(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "default"
	}
	return b.String()
}

func (m *Manager) CreateSnapshot(harness, scope, track string, targetPaths []string, stateJSONPath string) (*Manifest, error) {
	if err := rejectSymlinkInChain(m.BackupsRoot); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(m.BackupsRoot, 0700); err != nil {
		return nil, fmt.Errorf("creating backups root: %w", err)
	}

	now := time.Now().UTC()
	seq := atomic.AddUint64(&snapshotSeq, 1)
	id := fmt.Sprintf("%s-%03d-%s-%s", now.Format("20060102-150405"), seq%1000, sanitizeSlug(harness), sanitizeSlug(scope))
	snapDir := filepath.Join(m.BackupsRoot, id)
	filesDir := filepath.Join(snapDir, "files")
	if err := os.MkdirAll(filesDir, 0700); err != nil {
		return nil, fmt.Errorf("creating snapshot files dir: %w", err)
	}

	seen := make(map[string]bool, len(targetPaths))
	var uniquePaths []string
	for _, p := range targetPaths {
		if p == "" {
			continue
		}
		clean := filepath.Clean(p)
		if !seen[clean] {
			seen[clean] = true
			uniquePaths = append(uniquePaths, clean)
		}
	}
	sort.Strings(uniquePaths)

	manifest := &Manifest{
		ID:        id,
		Harness:   harness,
		Scope:     scope,
		Track:     track,
		CreatedAt: now.Format(time.RFC3339Nano),
	}

	for idx, origPath := range uniquePaths {
		info, err := os.Lstat(origPath)
		if err != nil || info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			manifest.Entries = append(manifest.Entries, BlobEntry{
				OriginalPath: origPath,
				Existed:      false,
			})
			continue
		}

		data, err := os.ReadFile(origPath)
		if err != nil {
			return nil, fmt.Errorf("reading %s for backup: %w", origPath, err)
		}
		blobID := fmt.Sprintf("%04d", idx)
		blobPath := filepath.Join(filesDir, blobID)
		if err := os.WriteFile(blobPath, data, 0600); err != nil {
			return nil, fmt.Errorf("writing backup blob %s: %w", blobID, err)
		}
		manifest.Entries = append(manifest.Entries, BlobEntry{
			BlobID:       blobID,
			OriginalPath: origPath,
			Mode:         uint32(info.Mode().Perm()),
			Existed:      true,
		})
	}

	if stateJSONPath != "" {
		if info, err := os.Lstat(stateJSONPath); err == nil && !info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
			if stateBytes, err := os.ReadFile(stateJSONPath); err == nil {
				if err := os.WriteFile(filepath.Join(snapDir, "state.json.bak"), stateBytes, 0600); err == nil {
					manifest.HasStateBackup = true
				}
			}
		}
	}

	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(snapDir, "manifest.json"), append(manifestBytes, '\n'), 0600); err != nil {
		return nil, fmt.Errorf("writing manifest.json: %w", err)
	}

	if m.MaxSnapshots > 0 {
		_ = m.PruneOld(m.MaxSnapshots)
	}

	return manifest, nil
}

func (m *Manager) ListSnapshots(harnessFilter string) ([]Manifest, error) {
	entries, err := os.ReadDir(m.BackupsRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var out []Manifest
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		mPath := filepath.Join(m.BackupsRoot, entry.Name(), "manifest.json")
		data, err := os.ReadFile(mPath)
		if err != nil {
			continue
		}
		var mf Manifest
		if err := json.Unmarshal(data, &mf); err != nil {
			continue
		}
		if harnessFilter != "" && !strings.EqualFold(mf.Harness, harnessFilter) {
			continue
		}
		out = append(out, mf)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt != out[j].CreatedAt {
			return out[i].CreatedAt > out[j].CreatedAt
		}
		return out[i].ID > out[j].ID
	})
	return out, nil
}

func (m *Manager) PruneOld(keep int) error {
	if keep <= 0 {
		return nil
	}
	all, err := m.ListSnapshots("")
	if err != nil || len(all) <= keep {
		return err
	}
	for _, old := range all[keep:] {
		if isValidSnapshotID(old.ID) {
			_ = os.RemoveAll(filepath.Join(m.BackupsRoot, old.ID))
		}
	}
	return nil
}

func isValidSnapshotID(id string) bool {
	if id == "" || id == "." || id == ".." {
		return false
	}
	return !strings.ContainsAny(id, `/\`) && !strings.Contains(id, "..")
}

func isValidBlobID(blobID string) bool {
	if blobID == "" || blobID == "." || blobID == ".." {
		return false
	}
	return !strings.ContainsAny(blobID, `/\`) && !strings.Contains(blobID, "..")
}

func isPreservedPath(p string) bool {
	slashed := filepath.ToSlash(p)
	return strings.HasSuffix(slashed, ".local.md") || strings.HasSuffix(slashed, "/references/config.md")
}

func rejectSymlinkInChain(targetPath string) error {
	clean := filepath.Clean(targetPath)
	cur := filepath.Dir(clean)
	for i := 0; i < 3; i++ {
		if cur == "" || cur == "." || cur == string(os.PathSeparator) {
			break
		}
		base := filepath.Base(cur)
		if (cur == "/var" || cur == "/tmp" || cur == "/etc") || base == "var" && filepath.Dir(cur) == "/" {
			break
		}
		info, err := os.Lstat(cur)
		if err != nil {
			if os.IsNotExist(err) {
				cur = filepath.Dir(cur)
				continue
			}
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing path traversal through symlinked parent %s", cur)
		}
		cur = filepath.Dir(cur)
	}
	return nil
}

func (m *Manager) RestoreSnapshot(id string, stateJSONPath string) (*RestoreResult, error) {
	if !isValidSnapshotID(id) {
		return nil, fmt.Errorf("invalid snapshot id %q", id)
	}
	if err := rejectSymlinkInChain(m.BackupsRoot); err != nil {
		return nil, err
	}

	snapDir := filepath.Join(m.BackupsRoot, id)
	manifestBytes, err := os.ReadFile(filepath.Join(snapDir, "manifest.json"))
	if err != nil {
		return nil, fmt.Errorf("reading snapshot %s: %w", id, err)
	}

	var mf Manifest
	if err := json.Unmarshal(manifestBytes, &mf); err != nil {
		return nil, fmt.Errorf("parsing snapshot %s: %w", id, err)
	}

	res := &RestoreResult{SnapshotID: mf.ID}
	for _, entry := range mf.Entries {
		if entry.OriginalPath == "" {
			continue
		}
		if err := rejectSymlinkInChain(entry.OriginalPath); err != nil {
			return nil, err
		}
		if entry.Existed {
			if !isValidBlobID(entry.BlobID) {
				return nil, fmt.Errorf("invalid blob id %q in snapshot %s", entry.BlobID, id)
			}
			blobBytes, err := os.ReadFile(filepath.Join(snapDir, "files", entry.BlobID))
			if err != nil {
				return nil, fmt.Errorf("reading blob %s: %w", entry.BlobID, err)
			}
			if info, lErr := os.Lstat(entry.OriginalPath); lErr == nil && info.Mode()&os.ModeSymlink != 0 {
				if err := os.Remove(entry.OriginalPath); err != nil {
					return nil, err
				}
			}
			if err := os.MkdirAll(filepath.Dir(entry.OriginalPath), 0755); err != nil {
				return nil, err
			}
			mode := os.FileMode(entry.Mode)
			if mode == 0 {
				mode = 0644
			}
			if err := os.WriteFile(entry.OriginalPath, blobBytes, mode); err != nil {
				return nil, err
			}
			res.RestoredFiles = append(res.RestoredFiles, entry.OriginalPath)
		} else {
			if isPreservedPath(entry.OriginalPath) {
				continue
			}
			if _, err := os.Lstat(entry.OriginalPath); err == nil {
				if err := os.Remove(entry.OriginalPath); err == nil {
					res.RemovedFiles = append(res.RemovedFiles, entry.OriginalPath)
					PruneEmptyParents(filepath.Dir(entry.OriginalPath), 3)
				}
			}
		}
	}

	if mf.HasStateBackup && stateJSONPath != "" {
		if err := rejectSymlinkInChain(stateJSONPath); err != nil {
			return nil, err
		}
		if stateBytes, err := os.ReadFile(filepath.Join(snapDir, "state.json.bak")); err == nil {
			if err := os.MkdirAll(filepath.Dir(stateJSONPath), 0700); err == nil {
				if err := os.WriteFile(stateJSONPath, stateBytes, 0600); err == nil {
					res.StateRestored = true
				}
			}
		}
	}

	return res, nil
}

func (m *Manager) RestoreSingleFile(id string, targetPath string) (bool, error) {
	if !isValidSnapshotID(id) {
		return false, fmt.Errorf("invalid snapshot id %q", id)
	}
	snapDir := filepath.Join(m.BackupsRoot, id)
	manifestBytes, err := os.ReadFile(filepath.Join(snapDir, "manifest.json"))
	if err != nil {
		return false, err
	}
	var mf Manifest
	if err := json.Unmarshal(manifestBytes, &mf); err != nil {
		return false, err
	}
	cleanTarget := filepath.Clean(targetPath)
	for _, entry := range mf.Entries {
		if filepath.Clean(entry.OriginalPath) != cleanTarget {
			continue
		}
		if !entry.Existed {
			return false, nil
		}
		if !isValidBlobID(entry.BlobID) {
			return false, fmt.Errorf("invalid blob id %q", entry.BlobID)
		}
		if err := rejectSymlinkInChain(cleanTarget); err != nil {
			return false, err
		}
		blobBytes, err := os.ReadFile(filepath.Join(snapDir, "files", entry.BlobID))
		if err != nil {
			return false, err
		}
		if err := os.MkdirAll(filepath.Dir(cleanTarget), 0755); err != nil {
			return false, err
		}
		mode := os.FileMode(entry.Mode)
		if mode == 0 {
			mode = 0644
		}
		if err := os.WriteFile(cleanTarget, blobBytes, mode); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func PruneEmptyParents(dir string, maxLevels int) {
	cur := filepath.Clean(dir)
	for i := 0; i < maxLevels; i++ {
		if cur == "" || cur == "." || cur == string(os.PathSeparator) {
			return
		}
		if err := os.Remove(cur); err != nil {
			return
		}
		cur = filepath.Dir(cur)
	}
}
