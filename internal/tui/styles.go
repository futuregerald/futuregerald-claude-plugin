package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	colorBrand        = lipgloss.AdaptiveColor{Light: "#0284C7", Dark: "#38BDF8"}
	colorSuccess      = lipgloss.AdaptiveColor{Light: "#059669", Dark: "#34D399"}
	colorAmber        = lipgloss.AdaptiveColor{Light: "#D97706", Dark: "#FBBF24"}
	colorPink         = lipgloss.AdaptiveColor{Light: "#DB2777", Dark: "#F472B6"}
	colorViolet       = lipgloss.AdaptiveColor{Light: "#7C3AED", Dark: "#A78BFA"}
	colorBlue         = lipgloss.AdaptiveColor{Light: "#2563EB", Dark: "#60A5FA"}
	colorTeal         = lipgloss.AdaptiveColor{Light: "#0D9488", Dark: "#2DD4BF"}
	colorIndigo       = lipgloss.AdaptiveColor{Light: "#4F46E5", Dark: "#818CF8"}
	colorMuted        = lipgloss.AdaptiveColor{Light: "#64748B", Dark: "#94A3B8"}
	colorSubtle       = lipgloss.AdaptiveColor{Light: "#94A3B8", Dark: "#64748B"}
	colorText         = lipgloss.AdaptiveColor{Light: "#0F172A", Dark: "#F8FAFC"}
	colorCardBorder   = lipgloss.AdaptiveColor{Light: "#CBD5E1", Dark: "#334155"}
	colorActiveBorder = lipgloss.AdaptiveColor{Light: "#0284C7", Dark: "#38BDF8"}

	PageStyle = lipgloss.NewStyle().
			Padding(1, 3)

	BotBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorActiveBorder).
			Padding(0, 2).
			MarginBottom(1)

	BotFaceStyle = lipgloss.NewStyle().
			Foreground(colorBrand).
			Bold(true)

	TitleStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Bold(true)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	StepPillStyle = lipgloss.NewStyle().
			Foreground(colorBrand).
			Bold(true)

	SelectedCardStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorActiveBorder).
				Padding(0, 2).
				MarginBottom(1)

	UnselectedCardStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorCardBorder).
				Padding(0, 2).
				MarginBottom(1)

	BadgeStyle = lipgloss.NewStyle().
			Foreground(colorAmber).
			Bold(true)

	SlashStyle = lipgloss.NewStyle().
			Foreground(colorSuccess).
			Bold(true)

	PromptBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(colorSuccess).
			PaddingLeft(2).
			Foreground(colorText)

	FooterStyle = lipgloss.NewStyle().
			Foreground(colorSubtle).
			MarginTop(1)
)

var botFrames = []string{
	"[ o_o ]",
	"[ ^_^ ]",
	"[ -_- ]",
}

func CardWidth(termWidth int) int {
	if termWidth <= 0 {
		return 84
	}
	w := termWidth - 8
	if w > 86 {
		w = 86
	}
	if w < 52 {
		w = 52
	}
	return w
}

func BotAvatar(frame int) string {
	idx := frame % len(botFrames)
	if idx < 0 {
		idx = 0
	}
	return BotFaceStyle.Render(botFrames[idx])
}

func CategoryBadge(category string) string {
	cat := strings.ToLower(strings.TrimSpace(category))
	style := lipgloss.NewStyle().Bold(true)
	label := cat
	switch cat {
	case "core workflow", "workflow":
		style = style.Foreground(colorBrand)
		label = "workflow"
	case "product & shaping", "product", "shaping":
		style = style.Foreground(colorViolet)
		label = "shaping"
	case "design, copy & ui", "copy-ui", "design-ui":
		style = style.Foreground(colorPink)
		label = "design-ui"
	case "planning & architecture", "architecture":
		style = style.Foreground(colorAmber)
		label = "architecture"
	case "engineering & tdd", "tdd-code":
		style = style.Foreground(colorSuccess)
		label = "tdd-code"
	case "review, security & qa", "quality", "review-qa":
		style = style.Foreground(colorTeal)
		label = "review-qa"
	case "shipping, ops & team", "shipping":
		style = style.Foreground(colorBlue)
		label = "shipping"
	case "research, browser & meta", "research":
		style = style.Foreground(colorIndigo)
		label = "research"
	case "antigravity exclusives", "antigravity":
		style = style.Foreground(colorBrand)
		label = "antigravity"
	default:
		style = style.Foreground(colorBlue)
	}
	return style.Render("[" + label + "]")
}

func HarnessDisplayName(harness string) string {
	switch strings.ToLower(harness) {
	case "antigravity":
		return "Google Antigravity"
	case "claude":
		return "Claude Code"
	case "zed":
		return "Zed Editor"
	case "opencode":
		return "OpenCode"
	default:
		return harness
	}
}

func PadRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}
