// Package gate holds consistency checks over the shipped skill and agent markdown.
//
// Unlike internal/installer's tests, which build a synthetic fstest.MapFS, these read the
// REAL tree. That makes `go test ./...` actual evidence about the files this repo ships.
package gate

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const root = "../.."

// planReviewSkill dispatches the PLAN REVIEW agents. Every agent it names must exist.
const planReviewSkill = "skills/plan-review/SKILL.md"

var wantAgents = []string{
	"adversarial-plan-reviewer",
	"plan-blindspot-hunter",
	"plan-consistency-checker",
}

func read(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(b)
}

var subagentRe = regexp.MustCompile(`(?m)^\s*subagent_type:\s*(\S+)\s*$`)
var nameRe = regexp.MustCompile(`(?m)^name:\s*(\S+)\s*$`)

// TestDispatchedAgentsExist parses agent names out of the skill rather than hardcoding
// them, so a skill naming an agent that was never created is caught.
func TestDispatchedAgentsExist(t *testing.T) {
	m := subagentRe.FindAllStringSubmatch(read(t, planReviewSkill), -1)
	if len(m) == 0 {
		t.Fatalf("%s declares no `subagent_type: <name>` lines; the dispatch format changed "+
			"and this gate can no longer verify it", planReviewSkill)
	}
	seen := map[string]bool{}
	for _, match := range m {
		name := match[1]
		seen[name] = true
		rel := filepath.Join("agents", name+".md")
		body, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Errorf("%s dispatches %q but %s does not exist", planReviewSkill, name, rel)
			continue
		}
		// InstallAgents copies by FILENAME; the Agent tool dispatches by `name:`.
		// A mismatch installs an agent that cannot be invoked.
		nm := nameRe.FindStringSubmatch(string(body))
		if nm == nil {
			t.Errorf("%s has no `name:` frontmatter", rel)
			continue
		}
		if nm[1] != name {
			t.Errorf("%s declares name %q, want %q (must equal filename)", rel, nm[1], name)
		}
	}
	for _, want := range wantAgents {
		if !seen[want] {
			t.Errorf("%s does not dispatch %q", planReviewSkill, want)
		}
	}
}

// TestReviewerStoppingRules guards the round-1 edit: the find-something-or-you-failed
// instruction must stay gone, and the no-praise paragraph must survive.
func TestReviewerStoppingRules(t *testing.T) {
	body := read(t, "agents/adversarial-plan-reviewer.md")
	if strings.Contains(body, "you did not look hard enough") {
		t.Error("adversarial-plan-reviewer is again told a nil result means it failed; " +
			"that instruction is why it stopped at first blood")
	}
	for _, want := range []string{
		"Praise is noise. Findings are the product.", // must not be lost to a wide edit range
		"legitimate outcome",
		"Do not stop at first blood",
		"Claim Ledger",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("adversarial-plan-reviewer missing %q", want)
		}
	}
}

// TestAgentPromptsAreVerbatim diffs the shipped agent bodies against the prompts that were
// actually measured in the head-to-head experiment. The recall figures cited in the README
// and commit history attach to these exact bytes; paraphrasing invalidates them.
func TestAgentPromptsAreVerbatim(t *testing.T) {
	for _, name := range []string{"plan-blindspot-hunter", "plan-consistency-checker"} {
		canonical, err := os.ReadFile(filepath.Join("testdata", name+".body.txt"))
		if err != nil {
			t.Fatalf("read canonical prompt for %s: %v", name, err)
		}
		shipped := read(t, filepath.Join("agents", name+".md"))
		_, body, ok := strings.Cut(strings.TrimPrefix(shipped, "---\n"), "\n---\n")
		if !ok {
			t.Fatalf("agents/%s.md has no frontmatter terminator", name)
		}
		want, got := strings.TrimSpace(string(canonical)), strings.TrimSpace(body)
		if !strings.HasSuffix(got, want) {
			t.Errorf("agents/%s.md body is not the measured prompt verbatim.\n"+
				"The recall figures do not transfer to paraphrased text.\n"+
				"canonical=%d chars, shipped tail does not match", name, len(want))
		}
	}
}

// TestWritingPlansReferences asserts substance, not existence: a one-line stub must fail.
func TestWritingPlansReferences(t *testing.T) {
	skill := read(t, "skills/writing-plans/SKILL.md")
	refs := map[string][]string{
		"claim-ledger":    {"## The ledger", "## Absence claims", "## Predictions", "## Could Not Verify"},
		"impact-analysis": {"## Upward", "## Downward", "## Invisible edges", "## Contract", "## Coverage"},
		"multiphase":      {"## Contract delta", "## Moving baseline", "## Hotspot", "## Citation drift", "## Gate falsifiability"},
		"indexing":        {"## Which index", "## Build once", "## Staleness", "## Repo scope", "## Graceful degradation"},
	}
	for slug, headings := range refs {
		rel := filepath.Join("skills/writing-plans/references", slug+".md")
		body := read(t, rel)
		if n := strings.Count(body, "\n"); n < 25 {
			t.Errorf("%s is %d lines - a stub, not a reference", rel, n)
		}
		for _, h := range headings {
			if !strings.Contains(body, h) {
				t.Errorf("%s missing section %q", rel, h)
			}
		}
		if !strings.Contains(skill, "references/"+slug+".md") {
			t.Errorf("writing-plans/SKILL.md does not point at references/%s.md", slug)
		}
	}
	// The gate must keep routing to PLAN REVIEW and keep owning the plan path.
	for _, want := range []string{"plan-review", "docs/plans/", "MANDATORY GATE", "Impact Analysis", "Claim Ledger"} {
		if !strings.Contains(skill, want) {
			t.Errorf("writing-plans/SKILL.md lost %q", want)
		}
	}
}

// TestNoHeadingSwallowedByCodeFence catches the real defect fence-parity cannot see: an
// unbalanced-in-practice fence that hides whole sections. The broken file had an EVEN
// fence count and still rendered ## Remember and ## Execution Handoff as a code block.
func TestNoHeadingSwallowedByCodeFence(t *testing.T) {
	for _, rel := range []string{
		"skills/writing-plans/SKILL.md",
		"skills/plan-review/SKILL.md",
	} {
		inside := false
		for i, line := range strings.Split(read(t, rel), "\n") {
			switch {
			case strings.HasPrefix(line, "```"):
				inside = !inside
			case inside && strings.HasPrefix(line, "## "):
				t.Errorf("%s:%d heading %q is inside a code fence", rel, i+1, line)
			}
		}
		if inside {
			t.Errorf("%s ends with an unclosed code fence", rel)
		}
	}
}

// singularGate matches prose still describing PLAN REVIEW as one reviewer.
var singularGate = []string{
	"a fresh adversarial sub-agent",
	"Staff Engineer sub-agent reviews the plan",
	"Staff Engineer reviews plan",
	"dispatches the `adversarial-plan-reviewer` agent",
}

// TestNoTrackedMarkdownDescribesASingleReviewer scans TRACKED markdown only, so the result
// is identical locally and in CI even when untracked skill directories are present.
func TestNoTrackedMarkdownDescribesASingleReviewer(t *testing.T) {
	out, err := exec.Command("git", "-C", root, "ls-files", "*.md").Output()
	if err != nil {
		t.Skipf("git ls-files unavailable: %v", err)
	}
	for _, rel := range strings.Fields(string(out)) {
		if strings.HasPrefix(rel, "docs/plans/") {
			continue // plans are never committed; not part of the shipped contract
		}
		b, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			continue
		}
		body := string(b)
		for _, phrase := range singularGate {
			if strings.Contains(body, phrase) {
				t.Errorf("%s still describes the plan gate as a single reviewer: %q", rel, phrase)
			}
		}
		if strings.Contains(body, "superpowers:code-reviewer") && strings.Contains(body, "Review this plan") {
			t.Errorf("%s routes the PLAN gate to a CODE reviewer", rel)
		}
	}
}
