package main

import "github.com/charmbracelet/lipgloss"

var (
	green  = lipgloss.Color("#00FF88")
	dim    = lipgloss.Color("#444444")
	text   = lipgloss.Color("#CCCCCC")
	yellow = lipgloss.Color("#FFCC00")
	red    = lipgloss.Color("#FF4444")
	blue   = lipgloss.Color("#4499FF")
	white  = lipgloss.Color("#FFFFFF")

	styleBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(dim).
			Padding(0, 1)

	styleHeader = lipgloss.NewStyle().
			Foreground(green).
			Bold(true)

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
				Foreground(dim).
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
