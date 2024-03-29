package clusterscreen

import (
	"esmon/config"
	"esmon/constants"
	"esmon/tui/styles"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	clusterTableColumns []table.Column = []table.Column{
		{Title: "Alias", Width: 20},
		{Title: "Endpoint", Width: 20},
		{Title: "Username", Width: 20},
		{Title: "Password", Width: 20},
	}

	clusterTableRows []table.Row

	clusterTableStyles = table.DefaultStyles()

    defaultKeyMap = keyMap{
        enter: key.NewBinding(
            key.WithKeys("enter"),
            key.WithHelp("<⏎>", "select"),
        ),
    }
)

type ClusterMsg []config.ClusterConfig
type ClusterChangeMsg string

type Model struct {
	width  int
	height int

	clusterTable table.Model

    help help.Model
}

type keyMap struct {
    enter key.Binding
}

func New() Model {
	m := Model{}

	m.clusterTable = table.New(
		table.WithColumns(clusterTableColumns),
		table.WithRows(clusterTableRows),
		table.WithFocused(true),
	)

	clusterTableStyles.Header = clusterTableStyles.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(styles.BorderColorMuted)).
		BorderBottom(true).
		Bold(false)
	clusterTableStyles.Selected = clusterTableStyles.Selected.
		Foreground(lipgloss.Color(styles.ForegroundColorHighlighted)).
		Bold(false)
	m.clusterTable.SetStyles(clusterTableStyles)

    m.help = help.New()
    m.help.Styles = styles.HelpStyle

	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

		for index := range clusterTableColumns {
			clusterTableColumns[index].Width = m.width/len(clusterTableColumns) - 2
		}

		m.clusterTable.SetHeight(m.height - 3)
		m.clusterTable.SetColumns(clusterTableColumns)

        m.help.Width = m.width - 2

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, defaultKeyMap.enter):
			clusterAlias := m.clusterTable.SelectedRow()[1]
			cmds = append(cmds, selectCluster(clusterAlias))
		}

	case ClusterMsg:
		for _, row := range msg {
			password := ""
			if row.Password != "" {
				password = constants.RedactedPassword
			}

			clusterTableRows = append(clusterTableRows, table.Row{
				row.Alias, row.Endpoint, row.Username, password,
			})
		}

		m.clusterTable.SetRows(clusterTableRows)

	}

	m.clusterTable, cmd = m.clusterTable.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
    return m.clusterTable.View() + m.help.View(defaultKeyMap)
}

func selectCluster(endpoint string) tea.Cmd {
	return func() tea.Msg {
		return ClusterChangeMsg(endpoint)
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.enter}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.enter},{}}
}
