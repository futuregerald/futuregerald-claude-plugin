package catalog

import (
	"io/fs"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func extractRawFrontmatter(t *testing.T, relPath string) (string, string) {
	t.Helper()
	b, err := fs.ReadFile(FS, relPath)
	if err != nil {
		t.Fatalf("reading %s: %v", relPath, err)
	}
	text := string(b)
	lines := strings.Split(text, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		t.Fatalf("%s missing leading YAML frontmatter", relPath)
	}
	endIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			endIdx = i
			break
		}
	}
	if endIdx < 0 {
		t.Fatalf("%s has unclosed YAML frontmatter", relPath)
	}
	return strings.Join(lines[1:endIdx], "\n"), text
}

func parseAgentFrontmatter(t *testing.T, relPath string) (map[string]string, string) {
	t.Helper()
	rawFM, text := extractRawFrontmatter(t, relPath)
	fields := make(map[string]string)
	for _, line := range strings.Split(rawFM, "\n") {
		colon := strings.IndexByte(line, ':')
		if colon > 0 && line[0] != ' ' && line[0] != '\t' {
			key := strings.TrimSpace(line[:colon])
			val := strings.TrimSpace(line[colon+1:])
			fields[key] = val
		}
	}
	return fields, text
}

type openCodeAgentFrontmatter struct {
	Name        string          `yaml:"name"`
	Description string          `yaml:"description"`
	Mode        string          `yaml:"mode"`
	Tools       map[string]bool `yaml:"tools"`
}

func TestNoAntigravityMechanicsOrLegacy4SectionHandoffInNonAntigravityCatalog(t *testing.T) {
	t.Parallel()

	forbidden := []string{
		"Hydrogen",
		"gemini-4-hydrogen",
		"swarm_wait",
		"baton-pass",
		"start-small",
		"peer-pipeline-swarm",
		"antigravity-orchestration",
		"4-section handoff",
	}

	for _, root := range []string{"shared", "claude", "zed", "opencode"} {
		err := fs.WalkDir(FS, root, func(p string, d fs.DirEntry, wErr error) error {
			if wErr != nil || d.IsDir() {
				return wErr
			}
			b, rErr := fs.ReadFile(FS, p)
			if rErr != nil {
				t.Fatalf("reading %s: %v", p, rErr)
			}
			content := string(b)
			lowerContent := strings.ToLower(content)
			for _, term := range forbidden {
				if term == "4-section handoff" {
					if strings.Contains(lowerContent, term) {
						t.Errorf("%s still contains legacy %q", p, term)
					}
					continue
				}
				if strings.Contains(content, term) {
					t.Errorf("%s leaked Antigravity-only mechanic %q outside antigravity/", p, term)
				}
			}
			if strings.HasPrefix(p, "claude/agents/") || strings.HasPrefix(p, "opencode/agents/") {
				if strings.Contains(lowerContent, "gerald.onyango@gmail.com") {
					t.Errorf("%s contains personal email address", p)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walking %s: %v", root, err)
		}
	}
}

func TestClaudeReadOnlyReviewersEnforceExplicitAllowlistAndContextBundle(t *testing.T) {
	t.Parallel()

	reviewers := []string{
		"claude/agents/reviewer.md",
		"claude/agents/code-quality-reviewer.md",
		"claude/agents/security-reviewer.md",
		"claude/agents/spec-reviewer.md",
		"claude/agents/adversarial-plan-reviewer.md",
		"claude/agents/sql-reviewer.md",
	}

	forbiddenMCPSubstrings := []string{
		"*",
		"delete_project",
		"index_repository",
		"ingest_traces",
	}

	for _, agentPath := range reviewers {
		fields, body := parseAgentFrontmatter(t, agentPath)
		toolsLine, ok := fields["tools"]
		if !ok || toolsLine == "" {
			t.Errorf("%s missing explicit tools: frontmatter grant", agentPath)
			continue
		}
		for _, bad := range forbiddenMCPSubstrings {
			if strings.Contains(toolsLine, bad) {
				t.Errorf("%s tools: %q contains forbidden wildcard or mutating MCP grant %q", agentPath, toolsLine, bad)
			}
		}
		granted := make(map[string]bool)
		for _, tok := range strings.Split(toolsLine, ",") {
			name := strings.TrimSpace(tok)
			if name != "" {
				granted[name] = true
			}
		}
		for _, forbiddenTool := range []string{"Bash", "Edit", "Write"} {
			if granted[forbiddenTool] {
				t.Errorf("%s must be read-only, but grants %q in tools: %q", agentPath, forbiddenTool, toolsLine)
			}
		}
		for _, requiredTool := range []string{
			"Read",
			"Grep",
			"Glob",
			"ToolSearch",
			"mcp__codebase-memory-mcp__search_code",
			"mcp__codebase-memory-mcp__search_graph",
			"mcp__codebase-memory-mcp__query_graph",
			"mcp__codebase-memory-mcp__get_code_snippet",
		} {
			if !granted[requiredTool] {
				t.Errorf("%s missing expected read-only tool %q in tools: %q", agentPath, requiredTool, toolsLine)
			}
		}
		if strings.Contains(body, "Run `git diff") {
			t.Errorf("%s instructs reviewer to Run `git diff despite having no Bash tool", agentPath)
		}
	}

	for _, contextReviewer := range []string{
		"claude/agents/code-quality-reviewer.md",
		"claude/agents/security-reviewer.md",
		"claude/agents/sql-reviewer.md",
		"claude/agents/spec-reviewer.md",
	} {
		_, body := parseAgentFrontmatter(t, contextReviewer)
		if !strings.Contains(body, "/tmp/subagent-context-") {
			t.Errorf("%s prompt template must consume /tmp/subagent-context-<project>-<stream>.md", contextReviewer)
		}
	}
}

func TestOpenCodeAgentsEnforceValidToolsMapReadOnlyReviewersAnd5SectionHandoff(t *testing.T) {
	t.Parallel()

	reviewers := []string{
		"opencode/agents/reviewer.md",
		"opencode/agents/code-quality-reviewer.md",
		"opencode/agents/security-reviewer.md",
		"opencode/agents/spec-reviewer.md",
		"opencode/agents/adversarial-plan-reviewer.md",
		"opencode/agents/sql-reviewer.md",
	}

	for _, agentPath := range reviewers {
		rawFM, body := extractRawFrontmatter(t, agentPath)
		var parsed openCodeAgentFrontmatter
		if err := yaml.Unmarshal([]byte(rawFM), &parsed); err != nil {
			t.Fatalf("%s YAML frontmatter failed OpenCode schema unmarshal: %v", agentPath, err)
		}
		if parsed.Tools == nil {
			t.Fatalf("%s missing OpenCode tools boolean map", agentPath)
		}
		for _, mutTool := range []string{"bash", "edit", "write"} {
			val, present := parsed.Tools[mutTool]
			if !present || val {
				t.Errorf("%s must explicitly set tools.%s: false, got present=%v val=%v", agentPath, mutTool, present, val)
			}
		}
		for toolKey, enabled := range parsed.Tools {
			if enabled && (strings.Contains(toolKey, "delete_project") || strings.Contains(toolKey, "index_repository") || strings.Contains(toolKey, "ingest_traces") || strings.Contains(toolKey, "*")) {
				t.Errorf("%s enables mutating or wildcard tool %q", agentPath, toolKey)
			}
		}
		if strings.Contains(body, "Run `git diff") {
			t.Errorf("%s instructs reviewer to Run `git diff despite bash: false", agentPath)
		}
		if strings.Contains(body, "/superpowers:") {
			t.Errorf("%s contains stale /superpowers: reference", agentPath)
		}
	}

	for _, contextReviewer := range []string{
		"opencode/agents/code-quality-reviewer.md",
		"opencode/agents/security-reviewer.md",
		"opencode/agents/sql-reviewer.md",
		"opencode/agents/spec-reviewer.md",
	} {
		_, body := extractRawFrontmatter(t, contextReviewer)
		if !strings.Contains(body, "/tmp/subagent-context-") {
			t.Errorf("%s must consume /tmp/subagent-context-<project>-<stream>.md", contextReviewer)
		}
	}

	rawImplFM, implBody := extractRawFrontmatter(t, "opencode/agents/implementer.md")
	var parsedImpl openCodeAgentFrontmatter
	if err := yaml.Unmarshal([]byte(rawImplFM), &parsedImpl); err != nil {
		t.Fatalf("opencode/agents/implementer.md YAML frontmatter failed OpenCode schema unmarshal: %v", err)
	}
	for _, needle := range []string{
		"5-section",
		"Acceptance Criteria & Success Metrics Met",
		"/tmp/subagent-context-",
	} {
		if !strings.Contains(implBody, needle) {
			t.Errorf("opencode/agents/implementer.md missing %q", needle)
		}
	}
}

func TestClaudeLowEffortHelpersAndBundlerScriptEmbedded(t *testing.T) {
	t.Parallel()

	for _, helperPath := range []string{
		"claude/agents/context-finder.md",
		"claude/agents/runner.md",
	} {
		fields, _ := parseAgentFrontmatter(t, helperPath)
		if fields["model"] != "sonnet" {
			t.Errorf("%s model = %q, want sonnet", helperPath, fields["model"])
		}
		if fields["effort"] != "low" {
			t.Errorf("%s effort = %q, want low", helperPath, fields["effort"])
		}
	}

	_, implBody := parseAgentFrontmatter(t, "claude/agents/implementer.md")
	for _, needle := range []string{
		"5-section",
		"Acceptance Criteria & Success Metrics Met",
		"/tmp/subagent-context-",
	} {
		if !strings.Contains(implBody, needle) {
			t.Errorf("claude/agents/implementer.md missing %q", needle)
		}
	}

	scriptPath := "shared/skills/subagent-driven-development/scripts/build_subagent_context.py"
	scriptBytes, err := fs.ReadFile(FS, scriptPath)
	if err != nil {
		t.Fatalf("expected %s in embedded catalog FS: %v", scriptPath, err)
	}
	scriptLines := strings.Split(string(scriptBytes), "\n")
	if len(scriptLines) == 0 || scriptLines[0] != "#!/usr/bin/env python3" {
		t.Errorf("%s must start with #!/usr/bin/env python3", scriptPath)
	}
	for i, line := range scriptLines {
		if i == 0 {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			t.Errorf("%s line %d starts with # comment: %q", scriptPath, i+1, line)
		}
	}
}
