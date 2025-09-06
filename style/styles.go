// Package style defines the visual styling for Sheepdog.
package style

import "github.com/charmbracelet/lipgloss"

var (
	StyleDetails = lipgloss.NewStyle().
			Padding(0, 2).
			Border(lipgloss.RoundedBorder())
	StyleDetailsHeader = lipgloss.NewStyle().
				AlignHorizontal(lipgloss.Center).
				Border(lipgloss.NormalBorder(), false, false, true, false)

	StyleList = lipgloss.NewStyle().
			Width(WidthSidenav)
	StyleListHeader = lipgloss.NewStyle().
			Width(WidthSidenav).
			AlignHorizontal(lipgloss.Center).
			Border(lipgloss.NormalBorder(), false, false, true, false)

	StyleItem = lipgloss.NewStyle().
			Width(WidthSidenav)
	StyleItemIdle = StyleItem.
			Foreground(colorIdle)
	StyleItemRunning = StyleItem.
				Foreground(colorRunning)
	StyleItemReady = StyleItem.
			Foreground(colorReady)
	StyleItemErrored = StyleItem.
				Foreground(colorErrored)

	StyleEnum = lipgloss.NewStyle().
			MarginRight(1)
	StyleEnumIdle = StyleEnum.
			Foreground(colorIdle)
	StyleEnumRunning = StyleEnum.
				Foreground(colorRunning)
	StyleEnumReady = StyleEnum.
			Foreground(colorReady)
	StyleEnumErrored = StyleEnum.
				Foreground(colorErrored)
)
