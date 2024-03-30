package loadingscreen

import (
	"esmon/constants"
	"esmon/tui/styles"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
    defaultTheme = styles.GetTheme(nil)
    logoStyle = lipgloss.NewStyle().Bold(true).Foreground(defaultTheme.LogoColor)
	loadingSpinnerStyle = lipgloss.NewStyle().Height(1).MarginTop(1).Align(lipgloss.Center).Bold(true).Foreground(defaultTheme.SpinnerColor)
)

type Model struct {
	width  int
	height int

	loadingSpinner spinner.Model
}

func New(theme *styles.Theme) Model {
	m := Model{}

	loadingSpinner := spinner.New()
	loadingSpinner.Spinner = spinner.MiniDot
	loadingSpinner.Style = loadingSpinnerStyle
	m.loadingSpinner = loadingSpinner

    setStyles(theme)

	return m
}

func (m Model) Init() tea.Cmd {
	return m.loadingSpinner.Tick
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var (
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

		logoRender := lipgloss.NewStyle().Render(constants.Logo)
		logoWidth, _ := lipgloss.Size(logoRender)

		loadingSpinnerStyle.Width(logoWidth)
    
    case styles.ThemeChangeMsg:
        var theme = styles.Theme(msg)
        setStyles(&theme)
	default:
		var cmd tea.Cmd
		m.loadingSpinner, cmd = m.loadingSpinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Top, logoStyle.Render(constants.Logo), m.loadingSpinner.View()))
}

func setStyles(theme *styles.Theme) {
    logoStyle = logoStyle.Foreground(theme.LogoColor)
    loadingSpinnerStyle = loadingSpinnerStyle.Foreground(theme.SpinnerColor)
}
