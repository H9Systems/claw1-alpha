package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	green  = lipgloss.Color("#3DFFB5")
	dim    = lipgloss.Color("#6F7782")
	text   = lipgloss.Color("#D8DEE9")
	yellow = lipgloss.Color("#FFD166")
	red    = lipgloss.Color("#FF5C7A")
	blue   = lipgloss.Color("#7DB7FF")
	white  = lipgloss.Color("#FFFFFF")
	cyan   = lipgloss.Color("#7EE7FF")
	ink    = lipgloss.Color("#101820")

	styleBox = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(dim).
			Padding(0, 1)

	styleHeader = lipgloss.NewStyle().
			Foreground(white).
			Bold(true)

	styleBrand = lipgloss.NewStyle().
			Foreground(ink).
			Background(green).
			Bold(true).
			Padding(0, 1)

	styleKicker = lipgloss.NewStyle().
			Foreground(cyan)

	styleLabel = lipgloss.NewStyle().
			Foreground(dim).
			Width(18)

	styleValue = lipgloss.NewStyle().
			Foreground(text)

	styleDim = lipgloss.NewStyle().
			Foreground(dim)

	styleGreen = lipgloss.NewStyle().
			Foreground(green)

	styleYellow = lipgloss.NewStyle().
			Foreground(yellow)

	styleRed = lipgloss.NewStyle().
			Foreground(red)

	styleBlue = lipgloss.NewStyle().
			Foreground(blue)

	styleInputActive = lipgloss.NewStyle().
				Foreground(green)

	styleInputInactive = lipgloss.NewStyle().
				Foreground(dim)

	styleSectionTitle = lipgloss.NewStyle().
				Foreground(white).
				Bold(true)

	styleTab = lipgloss.NewStyle().
			Foreground(dim).
			Padding(0, 2)

	styleTabActive = lipgloss.NewStyle().
			Foreground(green).
			Bold(true).
			Underline(true).
			Padding(0, 2)

	styleButton = lipgloss.NewStyle().
			Foreground(text)

	styleButtonActive = lipgloss.NewStyle().
				Foreground(green).
				Bold(true)

	styleKeys = lipgloss.NewStyle().
			Foreground(dim).
			MarginTop(1)
)

func dot(color lipgloss.Color) string {
	return lipgloss.NewStyle().Foreground(color).Render("●")
}

func circle() string {
	return styleDim.Render("○")
}

func statusPill(label string, color lipgloss.Color) string {
	return lipgloss.NewStyle().
		Foreground(ink).
		Background(color).
		Bold(true).
		Padding(0, 1).
		Render(label)
}

func rule(width int) string {
	if width < 12 {
		width = 12
	}
	return styleDim.Render(strings.Repeat("─", width))
}
