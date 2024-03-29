package styles

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/lipgloss"
)

const (
	LogoColor = "15"

	SpinnerColor = "202"

	ForegroundColorLight       = "15"
	ForegroundColorDark        = "16"
	ForegroundColorLightMuted  = "245"
	ForegroundColorDarkMuted   = "240"
	ForegroundColorHighlighted = "202"

	BackgroundColorStatuGreen   = "29"
	BackgroundColorStatusYellow = "220"
	BackgroundColorStatusRed    = "196"
	BackgroundColorStatusError  = "240"

	BorderColorMuted = "240"
)

var (
    HelpStyle = help.Styles {
        Ellipsis:       lipgloss.NewStyle().Foreground(lipgloss.Color(ForegroundColorLightMuted)),
        ShortKey:       lipgloss.NewStyle().Foreground(lipgloss.Color(ForegroundColorLightMuted)),
        ShortDesc:      lipgloss.NewStyle().Foreground(lipgloss.Color(ForegroundColorLightMuted)),
        ShortSeparator: lipgloss.NewStyle().Foreground(lipgloss.Color(ForegroundColorLightMuted)),
        FullKey:        lipgloss.NewStyle().Foreground(lipgloss.Color(ForegroundColorLightMuted)),
        FullDesc:       lipgloss.NewStyle().Foreground(lipgloss.Color(ForegroundColorLightMuted)),
        FullSeparator:  lipgloss.NewStyle().Foreground(lipgloss.Color(ForegroundColorLightMuted)),
    }
)
