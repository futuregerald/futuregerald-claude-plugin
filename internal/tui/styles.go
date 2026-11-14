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
	switch cat {
	case "workflow":
		style = style.Foreground(colorBrand)
	case "product":
		style = style.Foreground(colorViolet)
	case "quality":
		style = style.Foreground(colorSuccess)
	case "copy-ui":
		style = style.Foreground(colorPink)
	case "architecture":
		style = style.Foreground(colorAmber)
	default:
		style = style.Foreground(colorBlue)
	}
	return style.Render("[" + cat + "]")
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
