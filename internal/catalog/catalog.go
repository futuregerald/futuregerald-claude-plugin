package catalog

import (
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	HarnessClaude      = "claude"
	HarnessAntigravity = "antigravity"
	HarnessZed         = "zed"
	HarnessOpenCode    = "opencode"
)

type Track struct {
	ID                string   `yaml:"id"`
	Name              string   `yaml:"name"`
	Badge             string   `yaml:"badge"`
	Tagline           string   `yaml:"tagline"`
	Description       string   `yaml:"description"`
	InstallAgents     bool     `yaml:"install_agents"`
	InstallHooks      bool     `yaml:"install_hooks"`
	LightweightConfig bool     `yaml:"lightweight_config"`
	Skills            []string `yaml:"skills"`
	AntigravitySkills []string `yaml:"antigravity_skills"`
}

type ConceptExplainer struct {
	Title   string `yaml:"title"`
	Summary string `yaml:"summary"`
}

type Explainers struct {
	RepoOverview string             `yaml:"repo_overview"`
	Concepts     []ConceptExplainer `yaml:"concepts"`
}

type SkillMetadata struct {
	Name            string `yaml:"name"`
	Category        string `yaml:"category"`
	SlashCommand    string `yaml:"slash_command"`
	Short           string `yaml:"short"`
	Long            string `yaml:"long"`
	StarterPrompt   string `yaml:"starter_prompt"`
	AntigravityOnly bool   `yaml:"antigravity_only"`
}

type Catalog struct {
	Version    string          `yaml:"version"`
	Tracks     []Track         `yaml:"tracks"`
	Explainers Explainers      `yaml:"explainers"`
	Skills     []SkillMetadata `yaml:"skills"`
	fsys       fs.FS
	trackByID  map[string]Track
	skillByID  map[string]SkillMetadata
}

func SupportedHarnesses() []string {
	return []string{HarnessClaude, HarnessAntigravity, HarnessZed, HarnessOpenCode}
}

func IsSupportedHarness(harness string) bool {
	switch harness {
	case HarnessClaude, HarnessAntigravity, HarnessZed, HarnessOpenCode:
		return true
	default:
		return false
	}
}

func IsAntigravityOnlySkill(skillName string) bool {
	switch skillName {
	case "antigravity-orchestration", "peer-pipeline-swarm", "baton-pass":
		return true
	default:
		return false
	}
}

func Load(fsys fs.FS) (*Catalog, error) {
	data, err := fs.ReadFile(fsys, "tracks.yaml")
	if err != nil {
		return nil, fmt.Errorf("reading tracks.yaml: %w", err)
	}
	var cat Catalog
	if err := yaml.Unmarshal(data, &cat); err != nil {
		return nil, fmt.Errorf("parsing tracks.yaml: %w", err)
	}
	cat.fsys = fsys
	cat.trackByID = make(map[string]Track, len(cat.Tracks))
	for _, tr := range cat.Tracks {
		cat.trackByID[tr.ID] = tr
	}
	cat.skillByID = make(map[string]SkillMetadata, len(cat.Skills))
	for _, sm := range cat.Skills {
		cat.skillByID[sm.Name] = sm
	}
	return &cat, nil
}

func (c *Catalog) FS() fs.FS {
	return c.fsys
}

func (c *Catalog) TrackByID(id string) (Track, bool) {
	tr, ok := c.trackByID[id]
	return tr, ok
}

func (c *Catalog) SkillMeta(name string) (SkillMetadata, bool) {
	sm, ok := c.skillByID[name]
	return sm, ok
}

func (c *Catalog) SkillsForHarness(harness string) []SkillMetadata {
	var out []SkillMetadata
	for _, sm := range c.Skills {
		if sm.AntigravityOnly && harness != HarnessAntigravity {
			continue
		}
		out = append(out, sm)
	}
	return out
}

func (c *Catalog) TrackSkills(trackID, harness string) ([]string, error) {
	if !IsSupportedHarness(harness) {
		return nil, fmt.Errorf("unsupported harness: %s", harness)
	}
	tr, ok := c.trackByID[trackID]
	if !ok {
		return nil, fmt.Errorf("unknown track: %s", trackID)
	}
	seen := make(map[string]bool)
	var out []string
	for _, s := range tr.Skills {
		if IsAntigravityOnlySkill(s) && harness != HarnessAntigravity {
			continue
		}
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	if harness == HarnessAntigravity {
		for _, s := range tr.AntigravitySkills {
			if !seen[s] {
				seen[s] = true
				out = append(out, s)
			}
		}
	}
	return out, nil
}

func (c *Catalog) GlobalConfigTemplate(harness string, lightweight bool) ([]byte, error) {
	var rel string
	switch harness {
	case HarnessClaude:
		if lightweight {
			rel = "claude/CLAUDE-LIGHT.md"
		} else {
			rel = "claude/CLAUDE-GLOBAL.md"
		}
	case HarnessAntigravity:
		if lightweight {
			rel = "antigravity/GEMINI-LIGHT.md"
		} else {
			rel = "antigravity/GEMINI.md"
		}
	case HarnessZed:
		rel = "zed/AGENTS-GLOBAL.md"
	case HarnessOpenCode:
		rel = "opencode/AGENTS-GLOBAL.md"
	default:
		return nil, fmt.Errorf("unsupported harness: %s", harness)
	}
	return fs.ReadFile(c.fsys, rel)
}

func (c *Catalog) ProjectConfigTemplate(harness string) ([]byte, error) {
	var rel string
	switch harness {
	case HarnessClaude:
		rel = "claude/templates/CLAUDE-BASE.md"
	case HarnessAntigravity:
		rel = "antigravity/templates/GEMINI-BASE.md"
	case HarnessZed:
		rel = "zed/templates/AGENTS-ZED.md"
	case HarnessOpenCode:
		rel = "opencode/templates/AGENTS-OPENCODE.md"
	default:
		return nil, fmt.Errorf("unsupported harness: %s", harness)
	}
	return fs.ReadFile(c.fsys, rel)
}

var commandToSkill = map[string]string{
	"start.md":         "start",
	"begin.md":         "begin",
	"lifecycle.md":     "lifecycle",
	"shape.md":         "shape",
	"plan-review.md":   "plan-review",
	"fix-check.md":     "fix-check",
	"copy-edit.md":     "copy-edit",
	"write-a-prd.md":   "write-a-prd",
	"ui-design.md":     "ui-design",
	"team-pulse.md":    "team-pulse",
	"plugin-update.md": "plugin-updater",
}

func (c *Catalog) CommandsForSkills(harness string, installedSkills []string) []string {
	if harness != HarnessClaude && harness != HarnessOpenCode {
		return nil
	}
	skillSet := make(map[string]bool, len(installedSkills))
	for _, s := range installedSkills {
		skillSet[s] = true
	}

	cmdRoot := path.Join(harness, "commands")
	var matched []string
	_ = fs.WalkDir(c.fsys, cmdRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".md") {
			return nil
		}
		rel := strings.TrimPrefix(p, cmdRoot+"/")
		if rel == "plugin-update.md" {
			matched = append(matched, rel)
			return nil
		}
		if reqSkill, mapped := commandToSkill[rel]; mapped {
			if skillSet[reqSkill] {
				matched = append(matched, rel)
			}
			return nil
		}
		if installedSkills == nil || skillSet["start"] || skillSet["writing-plans"] {
			matched = append(matched, rel)
		}
		return nil
	})
	sort.Strings(matched)
	return matched
}
