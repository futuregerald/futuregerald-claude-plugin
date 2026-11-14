package catalog

import (
	"os"
	"path"
	"regexp"
	"strings"
	"testing"
)

func TestLoadEmbeddedCatalog(t *testing.T) {
	t.Parallel()
	cat, err := Load(FS)
	if err != nil {
		t.Fatalf("Load(FS) failed: %v", err)
	}
	if cat.Version != "4.0.0" {
		t.Errorf("Version = %q, want 4.0.0", cat.Version)
	}
	wantTracks := []string{"opinionated", "basics", "product", "copy-polish", "custom"}
	if len(cat.Tracks) != len(wantTracks) {
		t.Fatalf("len(Tracks) = %d, want %d", len(cat.Tracks), len(wantTracks))
	}
	for i, id := range wantTracks {
		if cat.Tracks[i].ID != id {
			t.Errorf("Tracks[%d].ID = %q, want %q", i, cat.Tracks[i].ID, id)
		}
	}
	if cat.Explainers.RepoOverview == "" || len(cat.Explainers.Concepts) < 4 {
		t.Errorf("Explainers incomplete: %+v", cat.Explainers)
	}
}

func TestLoadFromDirFS(t *testing.T) {
	t.Parallel()
	cat, err := Load(os.DirFS("."))
	if err != nil {
		t.Fatalf("Load(os.DirFS(.)) failed: %v", err)
	}
	if len(cat.Skills) < 60 {
		t.Errorf("expected at least 60 skills in catalog, got %d", len(cat.Skills))
	}
}

func TestSupportedHarnessesAndBatonPassExclusivity(t *testing.T) {
	t.Parallel()
	cat, err := Load(FS)
	if err != nil {
		t.Fatal(err)
	}

	for _, h := range SupportedHarnesses() {
		if !IsSupportedHarness(h) {
			t.Errorf("expected %q to be supported", h)
		}
		skills, err := cat.TrackSkills("opinionated", h)
		if err != nil {
			t.Fatalf("TrackSkills(opinionated, %s) failed: %v", h, err)
		}
		hasBaton := false
		hasOrch := false
		hasSwarm := false
		hasUpdater := false
		for _, s := range skills {
			switch s {
			case "baton-pass":
				hasBaton = true
			case "antigravity-orchestration":
				hasOrch = true
			case "peer-pipeline-swarm":
				hasSwarm = true
			case "plugin-updater":
				hasUpdater = true
			}
		}
		if !hasUpdater {
			t.Errorf("harness %s opinionated track missing plugin-updater", h)
		}
		if h == HarnessAntigravity {
			if !hasBaton || !hasOrch || !hasSwarm {
				t.Errorf("antigravity opinionated track missing exclusive skills: baton=%v orch=%v swarm=%v", hasBaton, hasOrch, hasSwarm)
			}
		} else {
			if hasBaton || hasOrch || hasSwarm {
				t.Errorf("harness %s leaked antigravity-only skill: baton=%v orch=%v swarm=%v", h, hasBaton, hasOrch, hasSwarm)
			}
		}
	}

	for _, unsupported := range []string{"copilot", "cursor", "vscode", "claude-desktop", "gemini"} {
		if IsSupportedHarness(unsupported) {
			t.Errorf("expected %q to be unsupported in v4", unsupported)
		}
	}
}

func TestEveryCatalogSkillDirectoryHasCompleteMetadata(t *testing.T) {
	t.Parallel()
	cat, err := Load(FS)
	if err != nil {
		t.Fatal(err)
	}

	sharedEntries, err := FS.ReadDir("shared/skills")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range sharedEntries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), "codebase-memory-") {
			continue
		}
		if entry.Name() == "baton-pass" || entry.Name() == "antigravity-orchestration" || entry.Name() == "peer-pipeline-swarm" {
			t.Errorf("antigravity-only skill %q must not exist in shared/skills", entry.Name())
		}
		meta, ok := cat.SkillMeta(entry.Name())
		if !ok {
			t.Errorf("shared skill %q missing from tracks.yaml skills metadata", entry.Name())
			continue
		}
		if meta.Category == "" || meta.SlashCommand == "" || meta.Short == "" || meta.Long == "" || meta.StarterPrompt == "" {
			t.Errorf("shared skill %q has incomplete metadata: %+v", entry.Name(), meta)
		}
	}

	agyEntries, err := FS.ReadDir("antigravity/skills")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range agyEntries {
		if !entry.IsDir() {
			continue
		}
		meta, ok := cat.SkillMeta(entry.Name())
		if !ok {
			t.Errorf("antigravity skill %q missing from tracks.yaml skills metadata", entry.Name())
			continue
		}
		if meta.Short == "" || meta.Long == "" {
			t.Errorf("antigravity skill %q missing short/long descriptions", entry.Name())
		}
	}
}

func TestTrackCrossReferenceClosure_C4(t *testing.T) {
	t.Parallel()
	cat, err := Load(FS)
	if err != nil {
		t.Fatal(err)
	}

	refRe := regexp.MustCompile(`\.\./([a-z0-9-]+)/SKILL\.md`)

	for _, trackID := range []string{"opinionated", "basics", "product", "copy-polish"} {
		for _, harness := range SupportedHarnesses() {
			skills, err := cat.TrackSkills(trackID, harness)
			if err != nil {
				t.Fatalf("TrackSkills(%s, %s) error: %v", trackID, harness, err)
			}
			inTrack := make(map[string]bool, len(skills))
			for _, s := range skills {
				inTrack[s] = true
			}

			for _, skillName := range skills {
				skillPath := path.Join("shared/skills", skillName, "SKILL.md")
				if harness == HarnessAntigravity {
					overridePath := path.Join("antigravity/skills", skillName, "SKILL.md")
					if _, err := FS.Open(overridePath); err == nil {
						skillPath = overridePath
					}
				}
				data, err := FS.ReadFile(skillPath)
				if err != nil {
					t.Fatalf("reading %s for (%s, %s): %v", skillPath, trackID, harness, err)
				}
				for _, line := range strings.Split(string(data), "\n") {
					matches := refRe.FindAllStringSubmatch(line, -1)
					for _, m := range matches {
						targetSkill := m[1]
						lowerLine := strings.ToLower(line)
						isGuardedOptional := strings.Contains(lowerLine, "when installed") || strings.Contains(lowerLine, "if installed") || strings.Contains(lowerLine, "optional")
						if !inTrack[targetSkill] && (trackID == "opinionated" || trackID == "product" || !isGuardedOptional) {
							t.Errorf("track %q on harness %q includes skill %q (%s) which references ../%s/SKILL.md without optional guard or inclusion",
								trackID, harness, skillName, skillPath, targetSkill)
						}
					}
				}
			}
		}
	}
}

func TestHarnessTemplatesDistinctAndBatonFreeOutsideAntigravity(t *testing.T) {
	t.Parallel()
	cat, err := Load(FS)
	if err != nil {
		t.Fatal(err)
	}

	seenGlobal := make(map[string]string)
	seenProject := make(map[string]string)

	for _, h := range SupportedHarnesses() {
		for _, light := range []bool{false, true} {
			gBytes, err := cat.GlobalConfigTemplate(h, light)
			if err != nil {
				t.Fatalf("GlobalConfigTemplate(%s, light=%v) error: %v", h, light, err)
			}
			gStr := string(gBytes)
			if len(strings.TrimSpace(gStr)) == 0 {
				t.Fatalf("GlobalConfigTemplate(%s, light=%v) is empty", h, light)
			}
			if h != HarnessAntigravity {
				if strings.Contains(gStr, "baton-pass") || strings.Contains(gStr, "autonomous-circuit-breaker") {
					t.Errorf("GlobalConfigTemplate(%s, light=%v) leaked baton-pass or circuit-breaker", h, light)
				}
			}
			if !light {
				for prevH, prevContent := range seenGlobal {
					if prevContent == gStr {
						t.Errorf("GlobalConfigTemplate(%s) is identical to %s", h, prevH)
					}
				}
				seenGlobal[h] = gStr
			}
		}

		pBytes, err := cat.ProjectConfigTemplate(h)
		if err != nil {
			t.Fatalf("ProjectConfigTemplate(%s) error: %v", h, err)
		}
		pStr := string(pBytes)
		if !strings.Contains(pStr, "{{PROJECT_NAME}}") || !strings.Contains(pStr, "<!-- LANGUAGE_SPECIFIC -->") {
			t.Errorf("ProjectConfigTemplate(%s) missing placeholders", h)
		}
		if h != HarnessAntigravity {
			if strings.Contains(pStr, "baton-pass") || strings.Contains(pStr, "autonomous-circuit-breaker") {
				t.Errorf("ProjectConfigTemplate(%s) leaked baton-pass or circuit-breaker", h)
			}
		}
		for prevH, prevContent := range seenProject {
			if prevContent == pStr {
				t.Errorf("ProjectConfigTemplate(%s) is identical to %s", h, prevH)
			}
		}
		seenProject[h] = pStr
	}
}

func TestCommandsForSkillsFiltering(t *testing.T) {
	t.Parallel()
	cat, err := Load(FS)
	if err != nil {
		t.Fatal(err)
	}

	copySkills, err := cat.TrackSkills("copy-polish", HarnessClaude)
	if err != nil {
		t.Fatal(err)
	}
	cmds := cat.CommandsForSkills(HarnessClaude, copySkills)
	hasCopyEdit := false
	hasPluginUpdate := false
	hasStart := false
	for _, c := range cmds {
		switch c {
		case "copy-edit.md":
			hasCopyEdit = true
		case "plugin-update.md":
			hasPluginUpdate = true
		case "start.md":
			hasStart = true
		}
	}
	if !hasCopyEdit || !hasPluginUpdate {
		t.Errorf("expected copy-edit.md and plugin-update.md in copy-polish commands, got %v", cmds)
	}
	if hasStart {
		t.Errorf("did not expect start.md in copy-polish commands, got %v", cmds)
	}
}
