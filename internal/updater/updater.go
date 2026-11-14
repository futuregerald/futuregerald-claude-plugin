package updater

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/futuregerald/futuregerald-claude-plugin/internal/catalog"
	"github.com/futuregerald/futuregerald-claude-plugin/internal/installer"
	"github.com/futuregerald/futuregerald-claude-plugin/internal/platform"
	"github.com/futuregerald/futuregerald-claude-plugin/internal/state"
)

const (
	DefaultRepo        = "futuregerald/futuregerald-claude-plugin"
	maxSingleFileBytes = 10 * 1024 * 1024
	maxArchiveBytes    = 100 * 1024 * 1024
)

type Runner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return nil, fmt.Errorf("%s %s: %s: %w", name, strings.Join(args, " "), msg, err)
		}
		return nil, fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return stdout.Bytes(), nil
}

type CommitSummary struct {
	SHA     string `json:"sha"`
	Message string `json:"message"`
	Date    string `json:"date,omitempty"`
}

type CheckReport struct {
	Harness         string          `json:"harness"`
	Scope           string          `json:"scope"`
	Track           string          `json:"track"`
	InstalledSHA    string          `json:"installed_sha"`
	LatestSHA       string          `json:"latest_sha"`
	UpdateAvailable bool            `json:"update_available"`
	ChangedSkills   []string        `json:"changed_skills"`
	ChangedFiles    []string        `json:"changed_files"`
	Commits         []CommitSummary `json:"commits"`
	CurrentVersion  string          `json:"current_version"`
	LatestVersion   string          `json:"latest_version,omitempty"`
	BinaryUpdate    bool            `json:"binary_update_available"`
}

type ghCommitResponse struct {
	SHA    string `json:"sha"`
	Commit struct {
		Message   string `json:"message"`
		Committer struct {
			Date string `json:"date"`
		} `json:"committer"`
	} `json:"commit"`
}

type ghCompareResponse struct {
	Commits []ghCommitResponse `json:"commits"`
	Files   []struct {
		Filename string `json:"filename"`
	} `json:"files"`
}

type ghReleaseResponse struct {
	TagName string `json:"tag_name"`
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return strings.TrimSpace(s[:idx])
	}
	return s
}

func normalizeVersion(v string) string {
	return strings.TrimPrefix(strings.TrimSpace(v), "v")
}

func CompareSemver(a, b string) int {
	pa := strings.Split(normalizeVersion(a), ".")
	pb := strings.Split(normalizeVersion(b), ".")
	for i := 0; i < 3; i++ {
		va := 0
		vb := 0
		if i < len(pa) {
			va, _ = strconv.Atoi(pa[i])
		}
		if i < len(pb) {
			vb, _ = strconv.Atoi(pb[i])
		}
		if va > vb {
			return 1
		}
		if va < vb {
			return -1
		}
	}
	return 0
}

func extractChangedSkills(files []string) []string {
	seen := make(map[string]bool)
	var skills []string
	for _, f := range files {
		slashed := filepath.ToSlash(f)
		parts := strings.Split(slashed, "/")
		if len(parts) >= 5 && parts[0] == "internal" && parts[1] == "catalog" && parts[3] == "skills" {
			skillName := parts[4]
			if skillName != "" && !seen[skillName] {
				seen[skillName] = true
				skills = append(skills, skillName)
			}
		}
	}
	sort.Strings(skills)
	return skills
}

func Check(ctx context.Context, runner Runner, receipt state.Receipt, currentVersion string) (*CheckReport, error) {
	if runner == nil {
		runner = ExecRunner{}
	}

	commitBytes, err := runner.Run(ctx, "gh", "api", fmt.Sprintf("repos/%s/commits/main", DefaultRepo))
	if err != nil {
		return nil, fmt.Errorf("checking latest commit on main via gh: %w", err)
	}
	var head ghCommitResponse
	if err := json.Unmarshal(commitBytes, &head); err != nil {
		return nil, fmt.Errorf("parsing latest commit response: %w", err)
	}

	report := &CheckReport{
		Harness:        receipt.Harness,
		Scope:          receipt.Scope,
		Track:          receipt.Track,
		InstalledSHA:   receipt.CommitSHA,
		LatestSHA:      head.SHA,
		CurrentVersion: normalizeVersion(currentVersion),
	}

	if receipt.CommitSHA == "" {
		report.UpdateAvailable = head.SHA != ""
		if head.SHA != "" {
			report.Commits = []CommitSummary{
				{
					SHA:     head.SHA,
					Message: firstLine(head.Commit.Message),
					Date:    head.Commit.Committer.Date,
				},
			}
		}
	} else if !strings.HasPrefix(head.SHA, receipt.CommitSHA) && head.SHA != receipt.CommitSHA {
		report.UpdateAvailable = true
		cmpBytes, cmpErr := runner.Run(ctx, "gh", "api", fmt.Sprintf("repos/%s/compare/%s...main", DefaultRepo, receipt.CommitSHA))
		if cmpErr == nil {
			var cmp ghCompareResponse
			if json.Unmarshal(cmpBytes, &cmp) == nil {
				for _, c := range cmp.Commits {
					report.Commits = append(report.Commits, CommitSummary{
						SHA:     c.SHA,
						Message: firstLine(c.Commit.Message),
						Date:    c.Commit.Committer.Date,
					})
				}
				for _, f := range cmp.Files {
					report.ChangedFiles = append(report.ChangedFiles, f.Filename)
				}
				report.ChangedSkills = extractChangedSkills(report.ChangedFiles)
			}
		}
	}

	if relBytes, relErr := runner.Run(ctx, "gh", "api", fmt.Sprintf("repos/%s/releases/latest", DefaultRepo)); relErr == nil {
		var rel ghReleaseResponse
		if json.Unmarshal(relBytes, &rel) == nil && rel.TagName != "" {
			report.LatestVersion = normalizeVersion(rel.TagName)
			if CompareSemver(report.LatestVersion, report.CurrentVersion) > 0 {
				report.BinaryUpdate = true
			}
		}
	}

	return report, nil
}

func ExtractRepoTarball(r io.Reader, destDir string) error {
	cleanDest := filepath.Clean(destDir)
	if err := os.MkdirAll(cleanDest, 0755); err != nil {
		return err
	}

	gzr, err := gzip.NewReader(io.LimitReader(r, maxArchiveBytes))
	if err != nil {
		return fmt.Errorf("opening gzip tarball: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	var totalBytes int64

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar entry: %w", err)
		}

		if hdr.Typeflag == tar.TypeXGlobalHeader || hdr.Typeflag == tar.TypeXHeader {
			continue
		}
		if hdr.Typeflag == tar.TypeSymlink || hdr.Typeflag == tar.TypeLink {
			return fmt.Errorf("refusing symlink or hardlink in archive: %s", hdr.Name)
		}
		if hdr.Typeflag != tar.TypeDir && hdr.Typeflag != tar.TypeReg && hdr.Typeflag != tar.TypeRegA {
			return fmt.Errorf("unsupported tar entry type %d for %s", hdr.Typeflag, hdr.Name)
		}

		rawSlash := strings.ReplaceAll(hdr.Name, `\`, "/")
		if strings.HasPrefix(rawSlash, "/") || strings.Contains(rawSlash, ":") {
			return fmt.Errorf("illegal absolute or drive path in archive: %s", hdr.Name)
		}
		for _, seg := range strings.Split(rawSlash, "/") {
			if seg == ".." {
				return fmt.Errorf("illegal path traversal in archive: %s", hdr.Name)
			}
		}

		trimmed := strings.TrimPrefix(rawSlash, "./")
		slashIdx := strings.IndexByte(trimmed, '/')
		if slashIdx < 0 {
			continue
		}
		relSlash := strings.TrimPrefix(trimmed[slashIdx+1:], "/")
		if relSlash == "" || relSlash == "." {
			continue
		}

		cleanedSlash := path.Clean(relSlash)
		if cleanedSlash == "." || cleanedSlash == ".." || strings.HasPrefix(cleanedSlash, "../") {
			return fmt.Errorf("illegal relative path in archive: %s", hdr.Name)
		}

		targetPath := filepath.Join(cleanDest, filepath.FromSlash(cleanedSlash))
		relCheck, relErr := filepath.Rel(cleanDest, targetPath)
		if relErr != nil || relCheck == ".." || strings.HasPrefix(relCheck, ".."+string(os.PathSeparator)) {
			return fmt.Errorf("archive path escapes destination: %s", hdr.Name)
		}

		if hdr.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return err
			}
			continue
		}

		if hdr.Size > maxSingleFileBytes {
			return fmt.Errorf("archive file %s exceeds 10 MiB limit (%d bytes)", hdr.Name, hdr.Size)
		}
		if totalBytes+hdr.Size > maxArchiveBytes {
			return fmt.Errorf("archive exceeds 100 MiB total extraction limit")
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}
		mode := os.FileMode(0644)
		if hdr.Mode&0111 != 0 {
			mode = 0755
		}
		f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
		if err != nil {
			return err
		}
		written, copyErr := io.Copy(f, io.LimitReader(tr, maxSingleFileBytes+1))
		closeErr := f.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if written > maxSingleFileBytes {
			return fmt.Errorf("archive file %s exceeded 10 MiB limit during read", hdr.Name)
		}
		totalBytes += written
		if totalBytes > maxArchiveBytes {
			return fmt.Errorf("archive exceeded 100 MiB total extraction limit")
		}
	}

	return nil
}

type ApplyOptions struct {
	Backup                bool
	Force                 bool
	CurrentVersion        string
	ProjectConfigRenderer func(base []byte, harness string, projectDir string) []byte
}

func Apply(ctx context.Context, runner Runner, env platform.Env, receipt state.Receipt, opts ApplyOptions) (*installer.InstallResult, string, error) {
	if runner == nil {
		runner = ExecRunner{}
	}

	commitBytes, err := runner.Run(ctx, "gh", "api", fmt.Sprintf("repos/%s/commits/main", DefaultRepo))
	if err != nil {
		return nil, "", fmt.Errorf("fetching latest commit SHA via gh: %w", err)
	}
	var head ghCommitResponse
	if err := json.Unmarshal(commitBytes, &head); err != nil {
		return nil, "", fmt.Errorf("parsing commit response: %w", err)
	}

	tarballBytes, err := runner.Run(ctx, "gh", "api", fmt.Sprintf("repos/%s/tarball/main", DefaultRepo))
	if err != nil {
		return nil, "", fmt.Errorf("downloading tarball/main via gh: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "skill-installer-update-*")
	if err != nil {
		return nil, "", err
	}
	defer os.RemoveAll(tmpDir)

	if err := ExtractRepoTarball(bytes.NewReader(tarballBytes), tmpDir); err != nil {
		return nil, "", fmt.Errorf("extracting update tarball: %w", err)
	}

	catalogDir := filepath.Join(tmpDir, "internal", "catalog")
	catFS := os.DirFS(catalogDir)
	cat, err := catalog.Load(catFS)
	if err != nil {
		return nil, "", fmt.Errorf("loading updated catalog: %w", err)
	}

	inst := installer.New(catFS, installer.Options{Force: true})
	projDir := receipt.RootDir
	if receipt.Scope == "global" {
		projDir = "."
	}

	trackID := receipt.Track
	if trackID == "" {
		trackID = "opinionated"
	}

	var customSkills []string
	if trackID == "custom" {
		customSkills = receipt.Skills
	}

	res, err := inst.ExecutePlan(installer.InstallPlan{
		Env:                   env,
		Catalog:               cat,
		Harness:               receipt.Harness,
		Scope:                 receipt.Scope,
		ProjectDir:            projDir,
		TrackID:               trackID,
		CustomSkills:          customSkills,
		Mode:                  "full",
		Backup:                opts.Backup,
		Force:                 true,
		Version:               opts.CurrentVersion,
		CommitSHA:             head.SHA,
		ProjectConfigRenderer: opts.ProjectConfigRenderer,
	})
	if err != nil {
		return nil, "", err
	}

	return res, head.SHA, nil
}

func validateReleaseURL(rawURL string, allowTestHost bool) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	if allowTestHost && (u.Hostname() == "127.0.0.1" || u.Hostname() == "localhost") {
		return nil
	}
	if u.Scheme != "https" || (u.Hostname() != "github.com" && u.Hostname() != "api.github.com") {
		return fmt.Errorf("disallowed release download URL %q", rawURL)
	}
	return nil
}

func SelfUpgradeBinary(ctx context.Context, env platform.Env, exePath, targetVersion string, client *http.Client, baseReleaseURL string) (string, error) {
	if client == nil {
		client = http.DefaultClient
	}
	normVer := normalizeVersion(targetVersion)
	if normVer == "" {
		return "", fmt.Errorf("target version is required for binary self-upgrade")
	}

	allowTestHost := baseReleaseURL != ""
	if baseReleaseURL == "" {
		baseReleaseURL = fmt.Sprintf("https://github.com/%s/releases/download/v%s", DefaultRepo, normVer)
	}
	baseReleaseURL = strings.TrimRight(baseReleaseURL, "/")

	ext := "tar.gz"
	binName := "skill-installer"
	if env.GOOS == "windows" {
		ext = "zip"
		binName = "skill-installer.exe"
	}
	archiveName := fmt.Sprintf("skill-installer_%s_%s_%s.%s", normVer, env.GOOS, env.GOARCH, ext)
	checksumsURL := baseReleaseURL + "/checksums.txt"
	archiveURL := baseReleaseURL + "/" + archiveName

	if err := validateReleaseURL(checksumsURL, allowTestHost); err != nil {
		return "", err
	}
	if err := validateReleaseURL(archiveURL, allowTestHost); err != nil {
		return "", err
	}

	req1, err := http.NewRequestWithContext(ctx, http.MethodGet, checksumsURL, nil)
	if err != nil {
		return "", err
	}
	resp1, err := client.Do(req1)
	if err != nil {
		return "", fmt.Errorf("downloading checksums.txt: %w", err)
	}
	defer resp1.Body.Close()
	if resp1.StatusCode != http.StatusOK {
		return "", fmt.Errorf("checksums.txt HTTP %d", resp1.StatusCode)
	}
	checksumBytes, err := io.ReadAll(io.LimitReader(resp1.Body, 1024*1024))
	if err != nil {
		return "", err
	}

	expectedHash := ""
	for _, line := range strings.Split(string(checksumBytes), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == archiveName {
			expectedHash = strings.ToLower(fields[0])
			break
		}
	}
	if expectedHash == "" {
		return "", fmt.Errorf("checksum for %s not found in checksums.txt", archiveName)
	}

	req2, err := http.NewRequestWithContext(ctx, http.MethodGet, archiveURL, nil)
	if err != nil {
		return "", err
	}
	resp2, err := client.Do(req2)
	if err != nil {
		return "", fmt.Errorf("downloading %s: %w", archiveName, err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		return "", fmt.Errorf("archive download HTTP %d", resp2.StatusCode)
	}

	tmpDir, err := os.MkdirTemp("", "skill-installer-self-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, archiveName)
	af, err := os.Create(archivePath)
	if err != nil {
		return "", err
	}
	hasher := sha256.New()
	_, copyErr := io.Copy(io.MultiWriter(af, hasher), io.LimitReader(resp2.Body, maxArchiveBytes))
	_ = af.Close()
	if copyErr != nil {
		return "", copyErr
	}
	actualHash := hex.EncodeToString(hasher.Sum(nil))
	if actualHash != expectedHash {
		return "", fmt.Errorf("SHA-256 mismatch for %s: expected %s, got %s", archiveName, expectedHash, actualHash)
	}

	extractedBin := filepath.Join(tmpDir, binName)
	if ext == "zip" {
		if err := extractBinaryFromZip(archivePath, binName, extractedBin); err != nil {
			return "", err
		}
	} else {
		if err := extractBinaryFromTarGz(archivePath, binName, extractedBin); err != nil {
			return "", err
		}
	}

	newBytes, err := os.ReadFile(extractedBin)
	if err != nil {
		return "", err
	}

	if env.GOOS == "windows" {
		oldPath := exePath + ".old"
		_ = os.Remove(oldPath)
		if _, err := os.Stat(exePath); err == nil {
			if err := os.Rename(exePath, oldPath); err != nil {
				return "", fmt.Errorf("renaming active Windows executable: %w", err)
			}
		}
		if err := os.WriteFile(exePath, newBytes, 0755); err != nil {
			_ = os.Rename(oldPath, exePath)
			return "", err
		}
		return normVer, nil
	}

	stagingPath := exePath + ".new"
	if err := os.WriteFile(stagingPath, newBytes, 0755); err != nil {
		return "", err
	}
	if err := os.Rename(stagingPath, exePath); err != nil {
		_ = os.Remove(stagingPath)
		return "", err
	}
	return normVer, nil
}

func extractBinaryFromTarGz(archivePath, binName, destPath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if path.Base(strings.ReplaceAll(hdr.Name, `\`, "/")) == binName && (hdr.Typeflag == tar.TypeReg || hdr.Typeflag == tar.TypeRegA) {
			out, err := os.OpenFile(destPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(out, io.LimitReader(tr, maxArchiveBytes))
			closeErr := out.Close()
			if copyErr != nil {
				return copyErr
			}
			return closeErr
		}
	}
	return fmt.Errorf("binary %s not found in tar.gz", binName)
}

func extractBinaryFromZip(archivePath, binName, destPath string) error {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer zr.Close()

	for _, zf := range zr.File {
		if path.Base(strings.ReplaceAll(zf.Name, `\`, "/")) == binName && !zf.FileInfo().IsDir() {
			rc, err := zf.Open()
			if err != nil {
				return err
			}
			out, err := os.OpenFile(destPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
			if err != nil {
				rc.Close()
				return err
			}
			_, copyErr := io.Copy(out, io.LimitReader(rc, maxArchiveBytes))
			rc.Close()
			closeErr := out.Close()
			if copyErr != nil {
				return copyErr
			}
			return closeErr
		}
	}
	return fmt.Errorf("binary %s not found in zip", binName)
}
