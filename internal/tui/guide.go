package tui

import (
	"fmt"
	"strings"

	"github.com/futuregerald/futuregerald-claude-plugin/internal/catalog"
)

func HarnessLoadLines(harness string) []string {
	switch harness {
	case catalog.HarnessAntigravity:
		return []string{
			"Global rules live in ~/.gemini/GEMINI.md (or GEMINI.md in your repo).",
			"Skills live in ~/.gemini/config/skills/ (or .agents/skills/ per project) and expose native /slash commands.",
			"Subagents live in ~/.gemini/config/plugins/antigravity-subagents/agents/.",
		}
	case catalog.HarnessClaude:
		return []string{
			"Rules live in ~/.claude/CLAUDE.md (global) or CLAUDE.md (project).",
			"Slash commands live in .claude/commands/ (or ~/.claude/commands/) -- type /start, /shape, /write-a-prd, or /plugin-update.",
			"Skills in .claude/skills/ load automatically whenever your task matches their description.",
		}
	case catalog.HarnessZed:
		return []string{
			"Rules live in AGENTS.md (project) or ~/.config/zed/AGENTS.md (%APPDATA%\\Zed\\AGENTS.md on Windows).",
			"Every skill in .agents/skills/ or ~/.agents/skills/ is automatically available as a /slash command in Zed's assistant panel.",
		}
	case catalog.HarnessOpenCode:
		return []string{
			"Rules live in AGENTS.md (project) or ~/.config/opencode/AGENTS.md (global).",
			"Skills, subagents, and commands live under .opencode/ (or ~/.config/opencode/).",
		}
	default:
		return nil
	}
}

func RenderPlainGuide(cat *catalog.Catalog, harness, trackID string) string {
	if harness == "" {
		harness = catalog.HarnessClaude
	}
	if trackID == "" {
		trackID = "opinionated"
	}

	var b strings.Builder
	appName := HarnessDisplayName(harness)
	track, hasTrack := cat.TrackByID(trackID)
	trackName := trackID
	if hasTrack {
		trackName = track.Name
	}

	b.WriteString(fmt.Sprintf("%s  %s -- Quick Tour & Command Guide (%s / %s)\n",
		BotAvatar(1), TitleStyle.Render("skill-installer v4.0"), appName, trackName))
	b.WriteString(strings.Repeat("-", 76) + "\n\n")

	b.WriteString(TitleStyle.Render("1. THE BASICS: SLASH COMMANDS vs. SKILLS") + "\n\n")
	for _, c := range cat.Explainers.Concepts {
		b.WriteString(fmt.Sprintf("  * %s\n    %s\n\n", BadgeStyle.Render(c.Title), sanitizeASCII(c.Summary)))
	}

	b.WriteString(TitleStyle.Render(fmt.Sprintf("2. HOW %s LOADS YOUR SETUP", strings.ToUpper(appName))) + "\n\n")
	for _, line := range HarnessLoadLines(harness) {
		b.WriteString("  * " + line + "\n")
	}
	b.WriteString("\n")

	b.WriteString(TitleStyle.Render("3. KEEPING SKILLS UPDATED & SAFE") + "\n\n")
	b.WriteString("  * In your AI chat: Ask \"Are there updates to the plugin?\" or type /plugin-update (uses gh CLI or GitHub MCP).\n")
	b.WriteString("  * From terminal:   Run `skill-installer update` to check and apply changes from main.\n")
	b.WriteString("  * Rollback / Undo: Run `skill-installer restore` to roll back a snapshot, or `skill-installer uninstall` to remove.\n\n")

	skills, err := cat.TrackSkills(trackID, harness)
	if err != nil || len(skills) == 0 {
		for _, sm := range cat.SkillsForHarness(harness) {
			skills = append(skills, sm.Name)
		}
	}

	b.WriteString(TitleStyle.Render(fmt.Sprintf("4. WHAT YOU CAN TYPE NOW (%d SKILLS IN %s)", len(skills), strings.ToUpper(trackName))) + "\n\n")
	b.WriteString(RenderSkillTable(cat, harness, trackID, true, skills))
	return b.String()
}

func sanitizeASCII(s string) string {
	r := strings.NewReplacer(
		"—", "--",
		"–", "-",
		"•", "*",
		"·", "-",
		"’", "'",
		"‘", "'",
		"“", "\"",
		"”", "\"",
		"→", "->",
		"…", "...",
	)
	return r.Replace(s)
}

func RenderSkillTable(cat *catalog.Catalog, harness, trackID string, showLong bool, filterSkills []string) string {
	if harness == "" {
		harness = catalog.HarnessClaude
	}

	var names []string
	if len(filterSkills) > 0 {
		names = filterSkills
	} else if trackID != "" && trackID != "custom" {
		if ts, err := cat.TrackSkills(trackID, harness); err == nil {
			names = ts
		}
	}
	if len(names) == 0 {
		for _, sm := range cat.SkillsForHarness(harness) {
			names = append(names, sm.Name)
		}
	}

	var b strings.Builder
	header := fmt.Sprintf("  %s %s %s %s\n",
		PadRight("SKILL", 31),
		PadRight("CATEGORY", 16),
		PadRight("SLASH / TRIGGER", 31),
		"SHORT DESCRIPTION",
	)
	b.WriteString(StepPillStyle.Render(header))
	b.WriteString("  " + strings.Repeat("-", 104) + "\n")

	for _, name := range names {
		sm, ok := cat.SkillMeta(name)
		if !ok {
			continue
		}
		if sm.AntigravityOnly && harness != catalog.HarnessAntigravity {
			continue
		}
		slash := sm.SlashCommand
		if slash == "" {
			slash = "(auto)"
		}
		b.WriteString(fmt.Sprintf("  %s %s %s %s\n",
			PadRight(TitleStyle.Render(sm.Name), 31),
			PadRight(CategoryBadge(sm.Category), 16),
			PadRight(SlashStyle.Render(slash), 31),
			sanitizeASCII(sm.Short),
		))
		if showLong {
			if sm.Long != "" {
				b.WriteString(fmt.Sprintf("      %s\n", SubtitleStyle.Render(sanitizeASCII(sm.Long))))
			}
			if sm.StarterPrompt != "" {
				b.WriteString(fmt.Sprintf("      Try typing: %s\n", SlashStyle.Render(fmt.Sprintf("%q", sanitizeASCII(sm.StarterPrompt)))))
			}
			b.WriteString("\n")
		}
	}
	return b.String()
}
