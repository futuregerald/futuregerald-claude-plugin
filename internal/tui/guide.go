package tui

import (
	"fmt"
	"strings"

	"github.com/futuregerald/futuregerald-claude-plugin/internal/catalog"
)

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

	b.WriteString(fmt.Sprintf("%s  %s — Quick Tour & Command Guide (%s · %s)\n",
		BotAvatar(1), TitleStyle.Render("skill-installer v4.0"), appName, trackName))
	b.WriteString(strings.Repeat("─", 76) + "\n\n")

	b.WriteString(TitleStyle.Render("1. THE BASICS: SLASH COMMANDS vs. SKILLS") + "\n\n")
	for _, c := range cat.Explainers.Concepts {
		b.WriteString(fmt.Sprintf("  • %s\n    %s\n\n", BadgeStyle.Render(c.Title), c.Summary))
	}

	b.WriteString(TitleStyle.Render(fmt.Sprintf("2. HOW %s LOADS YOUR SETUP", strings.ToUpper(appName))) + "\n\n")
	switch harness {
	case catalog.HarnessAntigravity:
		b.WriteString("  • Global rules live in ~/.gemini/GEMINI.md (or GEMINI.md in your repo).\n")
		b.WriteString("  • Skills live in ~/.gemini/config/skills/ (or .agents/skills/ per project) and expose native /slash commands.\n")
		b.WriteString("  • Subagents live in ~/.gemini/config/plugins/antigravity-subagents/agents/.\n\n")
	case catalog.HarnessClaude:
		b.WriteString("  • Rules live in ~/.claude/CLAUDE.md (global) or CLAUDE.md (project).\n")
		b.WriteString("  • Slash commands live in .claude/commands/ (or ~/.claude/commands/) — type /start, /shape, /write-a-prd, or /plugin-update.\n")
		b.WriteString("  • Skills in .claude/skills/ load automatically whenever your task matches their description.\n\n")
	case catalog.HarnessZed:
		b.WriteString("  • Rules live in AGENTS.md (project) or ~/.config/zed/AGENTS.md (%APPDATA%\\Zed\\AGENTS.md on Windows).\n")
		b.WriteString("  • Every skill in .agents/skills/ or ~/.agents/skills/ is automatically available as a /slash command in Zed's assistant panel.\n\n")
	case catalog.HarnessOpenCode:
		b.WriteString("  • Rules live in AGENTS.md (project) or ~/.config/opencode/AGENTS.md (global).\n")
		b.WriteString("  • Skills, subagents, and commands live under .opencode/ (or ~/.config/opencode/).\n\n")
	}

	b.WriteString(TitleStyle.Render("3. KEEPING SKILLS UPDATED & SAFE") + "\n\n")
	b.WriteString("  • In your AI chat: Ask \"Are there updates to the plugin?\" or type /plugin-update (uses gh CLI or GitHub MCP).\n")
	b.WriteString("  • From terminal:   Run `skill-installer update` to check and apply changes from main.\n")
	b.WriteString("  • Rollback / Undo: Run `skill-installer restore` to roll back a snapshot, or `skill-installer uninstall` to remove.\n\n")

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
	header := fmt.Sprintf("  %-28s %-15s %-18s %s\n", "SKILL", "CATEGORY", "SLASH / TRIGGER", "SHORT DESCRIPTION")
	b.WriteString(StepPillStyle.Render(header))
	b.WriteString("  " + strings.Repeat("─", 96) + "\n")

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
		b.WriteString(fmt.Sprintf("  %-28s %-24s %-18s %s\n",
			TitleStyle.Render(sm.Name),
			CategoryBadge(sm.Category),
			SlashStyle.Render(slash),
			sm.Short,
		))
		if showLong {
			if sm.Long != "" {
				b.WriteString(fmt.Sprintf("      %s\n", SubtitleStyle.Render(sm.Long)))
			}
			if sm.StarterPrompt != "" {
				b.WriteString(fmt.Sprintf("      Try typing: %s\n", SlashStyle.Render(fmt.Sprintf("%q", sm.StarterPrompt))))
			}
			b.WriteString("\n")
		}
	}
	return b.String()
}
