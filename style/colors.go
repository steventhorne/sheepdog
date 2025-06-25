package style

import "github.com/charmbracelet/lipgloss"

var (
	colorFg        = lipgloss.Color("#a0a8b7")
	colorRed       = lipgloss.Color("#e55561")
	colorOrange    = lipgloss.Color("#cc9057")
	colorYellow    = lipgloss.Color("#e2b86b")
	colorGreen     = lipgloss.Color("#8ebd6b")
	colorCyan      = lipgloss.Color("#48b0bd")
	colorBlue      = lipgloss.Color("#4fa6ed")
	colorPurple    = lipgloss.Color("#bf68d9")
	colorGray      = lipgloss.Color("#535965")
	colorLightGray = lipgloss.Color("#7a818e")

	colorIdle    = colorFg
	colorRunning = colorYellow
	colorReady   = colorGreen
	colorErrored = colorRed
)
