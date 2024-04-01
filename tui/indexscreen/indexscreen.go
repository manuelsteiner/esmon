package indexscreen

import (
	"esmon/elasticsearch"
	"esmon/tui/styles"
    "fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	defaultTheme = styles.GetTheme(nil)

	indexTableColumns []table.Column = []table.Column{
		{Title: "Name", Width: 20},
		{Title: "Health", Width: 10},
		{Title: "Status", Width: 10},
		{Title: "Docs count [*]", Width: 20},
		{Title: "↓Storage size [*]", Width: 20},
	}

	indexTableRows []table.Row

	indexTableStyles = table.DefaultStyles()

	helpStyle = lipgloss.NewStyle().Height(1).Foreground(defaultTheme.ForegroundColorLightMuted)
)

type IndexMsg []elasticsearch.IndexStats

type Model struct {
	width  int
	height int

	indexTable table.Model
}

func New(theme *styles.Theme) Model {
	m := Model{}

	m.indexTable = table.New(
		table.WithColumns(indexTableColumns),
		table.WithRows(indexTableRows),
		table.WithFocused(true),
	)

	indexTableStyles.Header = indexTableStyles.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(theme.BorderColorMuted)).
		BorderBottom(true).
		Bold(false).
		Foreground(lipgloss.Color(theme.ForegroundColorLight))
	indexTableStyles.Selected = indexTableStyles.Selected.
		Foreground(lipgloss.Color(theme.ForegroundColorHighlighted)).
		Bold(false)
	m.indexTable.SetStyles(indexTableStyles)

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

		for index := range indexTableColumns {
			indexTableColumns[index].Width = m.width/len(indexTableColumns) - 2
		}

		m.indexTable.SetHeight(m.height - 3)
		m.indexTable.SetColumns(indexTableColumns)

		helpStyle.Width(m.width - 2)

	case styles.ThemeChangeMsg:
		var theme = styles.Theme(msg)
		setStyles(&theme)

		m.indexTable.SetStyles(indexTableStyles)

	case IndexMsg:
		var indexTableRows []table.Row

		for _, row := range msg {
			indexTableRows = append(indexTableRows, table.Row{
				row.Name,
				row.Health,
                row.Status,
				fmt.Sprintf("%d", row.Total.Docs.Count),
				strings.ToUpper(row.Total.Store.Size),
			})
		}

		m.indexTable.SetRows(indexTableRows)

	}

	m.indexTable, cmd = m.indexTable.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	return lipgloss.JoinVertical(
		lipgloss.Top,
		m.indexTable.View(),
		helpStyle.Render("[★] Total size (including replicas)"),
	)
}

func setStyles(theme *styles.Theme) {
	indexTableStyles.Header = indexTableStyles.Header.
		BorderForeground(lipgloss.Color(theme.BorderColorMuted)).
		Foreground(lipgloss.Color(theme.ForegroundColorLight))
	indexTableStyles.Selected = indexTableStyles.Selected.
		Foreground(lipgloss.Color(theme.ForegroundColorHighlighted))

	helpStyle = helpStyle.Foreground(lipgloss.Color(theme.ForegroundColorLightMuted))
}
