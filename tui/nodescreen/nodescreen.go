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
	defaultTheme = styles.GetTheme(nil)

	nodeTableColumns []table.Column = []table.Column{
		{Title: "Name", Width: 20},
		{Title: "Transport", Width: 20},
		{Title: "Shards", Width: 20},
		{Title: "CPU Usage [%]", Width: 10},
		{Title: "Load Average", Width: 10},
		{Title: "MEM Usage", Width: 10},
		{Title: "Free Disk Space", Width: 10},
	}

	nodeTableRows []table.Row

	nodeTableStyles = table.DefaultStyles()

	helpStyle = lipgloss.NewStyle().Height(1).Foreground(defaultTheme.ForegroundColorLightMuted)
)

type NodeMsg struct {
	Nodes      []elasticsearch.NodeStats
	MasterNode *elasticsearch.NodeStats
}

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

		helpStyle.Width(m.width - 2)

	case styles.ThemeChangeMsg:
		var theme = styles.Theme(msg)
		setStyles(&theme)

		m.nodeTable.SetStyles(nodeTableStyles)

	case NodeMsg:
		var nodeTableRows []table.Row

		for _, row := range msg.Nodes {
			nodeName := row.Name
			if row.Id == msg.MasterNode.Id {
				nodeName += "[★]"
			}

			nodeTableRows = append(nodeTableRows, table.Row{
				nodeName,
				row.TransportAddress,
				fmt.Sprintf("%d", row.Indices.ShardStats.TotalCount),
				fmt.Sprintf("%d", row.Os.CPU.Percent),
				fmt.Sprintf("%.2f", row.Os.CPU.LoadAverage.One5M),
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
	return lipgloss.JoinVertical(
		lipgloss.Top,
		m.nodeTable.View(),
		helpStyle.Render("[★] Master Node"),
	)
}

func setStyles(theme *styles.Theme) {
	nodeTableStyles.Header = nodeTableStyles.Header.
		BorderForeground(lipgloss.Color(theme.BorderColorMuted)).
		Foreground(lipgloss.Color(theme.ForegroundColorLight))
	nodeTableStyles.Selected = nodeTableStyles.Selected.
		Foreground(lipgloss.Color(theme.ForegroundColorHighlighted))

	helpStyle = helpStyle.Foreground(lipgloss.Color(theme.ForegroundColorLightMuted))
}
