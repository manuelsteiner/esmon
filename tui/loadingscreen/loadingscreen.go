package loadingscreen

import (
	"esmon/constants"
	"esmon/tui/styles"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	loadingSpinnerStyle = lipgloss.NewStyle().Height(1).MarginTop(1).Align(lipgloss.Center).Bold(true).Foreground(lipgloss.Color(styles.SpinnerColor))
)

type Model struct {
	width  int
	height int

	loadingSpinner spinner.Model
}

func New() Model {
	m := Model{}

	loadingSpinner := spinner.New()
	loadingSpinner.Spinner = spinner.MiniDot
	loadingSpinner.Style = loadingSpinnerStyle
	m.loadingSpinner = loadingSpinner

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
	default:
		var cmd tea.Cmd
		m.loadingSpinner, cmd = m.loadingSpinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Top, lipgloss.NewStyle().Bold(true).Render(constants.Logo), m.loadingSpinner.View()))
}
