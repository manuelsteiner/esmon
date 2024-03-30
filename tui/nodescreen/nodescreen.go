package nodescreen

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
	nodeTableColumns []table.Column = []table.Column{
		{Title: "Name", Width: 20},
		{Title: "Transport", Width: 20},
		{Title: "CPU Usage", Width: 10},
		{Title: "Load Average", Width: 10},
		{Title: "MEM Usage", Width: 10},
		{Title: "Free Disk Space", Width: 10},
	}

	nodeTableRows []table.Row

	nodeTableStyles = table.DefaultStyles()
)

type NodeMsg []elasticsearch.NodeStats

type Model struct {
	width  int
	height int

	nodeTable table.Model
}

func New(theme *styles.Theme) Model {
	m := Model{}

	m.nodeTable = table.New(
		table.WithColumns(nodeTableColumns),
		table.WithRows(nodeTableRows),
		table.WithFocused(true),
	)

	nodeTableStyles.Header = nodeTableStyles.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(theme.BorderColorMuted)).
		BorderBottom(true).
		Bold(false).
		Foreground(lipgloss.Color(theme.ForegroundColorLight))
	nodeTableStyles.Selected = nodeTableStyles.Selected.
		Foreground(lipgloss.Color(theme.ForegroundColorHighlighted)).
		Bold(false)
	m.nodeTable.SetStyles(nodeTableStyles)

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

		for index := range nodeTableColumns {
			nodeTableColumns[index].Width = m.width/len(nodeTableColumns) - 2
		}

		m.nodeTable.SetHeight(m.height - 3)
		m.nodeTable.SetColumns(nodeTableColumns)

	case styles.ThemeChangeMsg:
		var theme = styles.Theme(msg)
		setStyles(&theme)

		m.nodeTable.SetStyles(nodeTableStyles)

	case NodeMsg:
		var nodeTableRows []table.Row

		for _, row := range msg {
			nodeTableRows = append(nodeTableRows, table.Row{
				row.Name,
				row.TransportAddress,
				fmt.Sprintf("%d%%", row.Os.CPU.Percent),
				fmt.Sprintf("%f", row.Os.CPU.LoadAverage.One5M),
				strings.ToUpper(row.Os.Mem.Used),
				strings.ToUpper(row.Fs.Total.Free),
			})
		}

		m.nodeTable.SetRows(nodeTableRows)

	}

	m.nodeTable, cmd = m.nodeTable.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	return m.nodeTable.View()
}

func setStyles(theme *styles.Theme) {
	nodeTableStyles.Header = nodeTableStyles.Header.
		BorderForeground(lipgloss.Color(theme.BorderColorMuted)).
		Foreground(lipgloss.Color(theme.ForegroundColorLight))
	nodeTableStyles.Selected = nodeTableStyles.Selected.
		Foreground(lipgloss.Color(theme.ForegroundColorHighlighted))
}
