package styles

import (
    "esmon/config"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/lipgloss"
)

const (
    OverviewHeight = 5

	logoColor = "15"

	spinnerColor = "202"

	foregroundColorLight       = "15"
	foregroundColorDark        = "16"
	foregroundColorLightMuted  = "245"
	foregroundColorDarkMuted   = "240"
	foregroundColorHighlighted = "202"

	backgroundColorStatusGreen   = "29"
	backgroundColorStatusYellow = "220"
	backgroundColorStatusRed    = "196"
	backgroundColorStatusError  = "240"

	borderColor=       "15"
	borderColorMuted = "240"
)

type Theme struct {
	LogoColor lipgloss.Color

	SpinnerColor lipgloss.Color

	ForegroundColorLight       lipgloss.Color
	ForegroundColorDark        lipgloss.Color
	ForegroundColorLightMuted  lipgloss.Color
	ForegroundColorDarkMuted   lipgloss.Color
	ForegroundColorHighlighted lipgloss.Color

	BackgroundColorStatusGreen   lipgloss.Color
	BackgroundColorStatusYellow lipgloss.Color
	BackgroundColorStatusRed    lipgloss.Color
	BackgroundColorStatusError  lipgloss.Color

	BorderColor      lipgloss.Color
	BorderColorMuted lipgloss.Color
}

type ThemeChangeMsg Theme

var (
    defaultTheme = Theme {
        LogoColor: lipgloss.Color(logoColor),

        SpinnerColor: lipgloss.Color(spinnerColor),

        ForegroundColorLight:       lipgloss.Color(foregroundColorLight) ,
        ForegroundColorDark:        lipgloss.Color(foregroundColorDark),
        ForegroundColorLightMuted:  lipgloss.Color(foregroundColorLightMuted),
        ForegroundColorDarkMuted:   lipgloss.Color(foregroundColorDarkMuted),
        ForegroundColorHighlighted: lipgloss.Color(foregroundColorHighlighted),

        BackgroundColorStatusGreen:   lipgloss.Color(backgroundColorStatusGreen),
        BackgroundColorStatusYellow: lipgloss.Color(backgroundColorStatusYellow),
        BackgroundColorStatusRed:    lipgloss.Color(backgroundColorStatusRed),
        BackgroundColorStatusError:  lipgloss.Color(backgroundColorStatusError),

        BorderColor:      lipgloss.Color(borderColor),
        BorderColorMuted: lipgloss.Color(borderColorMuted),
    }

    HelpStyle = help.Styles {
        Ellipsis:       lipgloss.NewStyle().Foreground(lipgloss.Color(foregroundColorLightMuted)),
        ShortKey:       lipgloss.NewStyle().Foreground(lipgloss.Color(foregroundColorLightMuted)),
        ShortDesc:      lipgloss.NewStyle().Foreground(lipgloss.Color(foregroundColorLightMuted)),
        ShortSeparator: lipgloss.NewStyle().Foreground(lipgloss.Color(foregroundColorLightMuted)),
        FullKey:        lipgloss.NewStyle().Foreground(lipgloss.Color(foregroundColorLightMuted)),
        FullDesc:       lipgloss.NewStyle().Foreground(lipgloss.Color(foregroundColorLightMuted)),
        FullSeparator:  lipgloss.NewStyle().Foreground(lipgloss.Color(foregroundColorLightMuted)),
    }
)

func GetTheme(overrides *config.ThemeConfig) Theme {
    theme := defaultTheme

	if overrides != nil && overrides.LogoColor != "" {
        theme.LogoColor = lipgloss.Color(overrides.LogoColor)
    }

	if overrides != nil && overrides.SpinnerColor != "" {
        theme.SpinnerColor = lipgloss.Color(overrides.SpinnerColor)
    }

	if overrides != nil && overrides.ForegroundColorLight != "" {
        theme.ForegroundColorLight = lipgloss.Color(overrides.ForegroundColorLight)
    }
	if overrides != nil && overrides.ForegroundColorDark != "" {
        theme.ForegroundColorDark = lipgloss.Color(overrides.ForegroundColorDark)
    }
	if overrides != nil && overrides.ForegroundColorLightMuted != "" {
        theme.ForegroundColorLightMuted = lipgloss.Color(overrides.ForegroundColorLightMuted)
    }
	if overrides != nil && overrides.ForegroundColorDarkMuted != "" {
        theme.ForegroundColorDarkMuted = lipgloss.Color(overrides.ForegroundColorDarkMuted)
    }
	if overrides != nil && overrides.ForegroundColorHighlighted != "" {
        theme.ForegroundColorHighlighted = lipgloss.Color(overrides.ForegroundColorHighlighted)
    }

	if overrides != nil && overrides.BackgroundColorStatusGreen != "" {
        theme.BackgroundColorStatusGreen = lipgloss.Color(overrides.BackgroundColorStatusGreen)
    }
	if overrides != nil && overrides.BackgroundColorStatusYellow != "" {
        theme.BackgroundColorStatusYellow = lipgloss.Color(overrides.BackgroundColorStatusYellow)
    }
	if overrides != nil && overrides.BackgroundColorStatusRed != "" {
        theme.BackgroundColorStatusRed = lipgloss.Color(overrides.BackgroundColorStatusRed)
    }
	if overrides != nil && overrides.BackgroundColorStatusError != "" {
        theme.BackgroundColorStatusError = lipgloss.Color(overrides.BackgroundColorStatusError)
    }
	
	if overrides != nil && overrides.BorderColor!= "" {
        theme.BorderColor= lipgloss.Color(overrides.BorderColor)
    }
	if overrides != nil && overrides.BorderColorMuted != "" {
        theme.BorderColorMuted = lipgloss.Color(overrides.BorderColorMuted)
    }

	if overrides != nil && overrides.ForegroundColorLightMuted != "" {
        HelpStyle.Ellipsis = HelpStyle.Ellipsis.Foreground(lipgloss.Color(overrides.ForegroundColorLightMuted))
        HelpStyle.ShortKey = HelpStyle.ShortKey.Foreground(lipgloss.Color(overrides.ForegroundColorLightMuted))
        HelpStyle.ShortDesc = HelpStyle.ShortDesc.Foreground(lipgloss.Color(overrides.ForegroundColorLightMuted))
        HelpStyle.ShortSeparator = HelpStyle.ShortSeparator.Foreground(lipgloss.Color(overrides.ForegroundColorLightMuted))
        HelpStyle.FullKey = HelpStyle.FullKey.Foreground(lipgloss.Color(overrides.ForegroundColorLightMuted))
        HelpStyle.FullDesc = HelpStyle.FullDesc.Foreground(lipgloss.Color(overrides.ForegroundColorLightMuted))
        HelpStyle.FullSeparator = HelpStyle.FullSeparator.Foreground(lipgloss.Color(overrides.ForegroundColorLightMuted))
    }

    return theme
}
