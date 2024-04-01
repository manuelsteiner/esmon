package shardallocationscreen

import (
	"esmon/elasticsearch"
	"esmon/tui/styles"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	defaultTheme = styles.GetTheme(nil)

	shardAllocationTableColumns []table.Column = []table.Column{
		{Title: "↑Index [*]", Width: 20},
		{Title: "Shard", Width: 20},
		{Title: "Primary nodes", Width: 20},
		{Title: "Replica nodes", Width: 20},
	}

	shardAllocationTableRows []table.Row

	shardAllocationTableStyles = table.DefaultStyles()

	helpStyle = lipgloss.NewStyle().Height(1).Foreground(defaultTheme.ForegroundColorLightMuted)
)

type ShardAllocationMsg []elasticsearch.ShardStores

type Model struct {
	width  int
	height int

	shardAllocationTable table.Model
}

func New(theme *styles.Theme) Model {
	m := Model{}

	m.shardAllocationTable = table.New(
		table.WithColumns(shardAllocationTableColumns),
		table.WithRows(shardAllocationTableRows),
		table.WithFocused(true),
	)

	shardAllocationTableStyles.Header = shardAllocationTableStyles.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(theme.BorderColorMuted)).
		BorderBottom(true).
		Bold(false).
		Foreground(lipgloss.Color(theme.ForegroundColorLight))
	shardAllocationTableStyles.Selected = shardAllocationTableStyles.Selected.
		Foreground(lipgloss.Color(theme.ForegroundColorHighlighted)).
		Bold(false)
	m.shardAllocationTable.SetStyles(shardAllocationTableStyles)

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

		for index := range shardAllocationTableColumns {
			shardAllocationTableColumns[index].Width = m.width/len(shardAllocationTableColumns) - 2
		}

		m.shardAllocationTable.SetHeight(m.height - 3)
		m.shardAllocationTable.SetColumns(shardAllocationTableColumns)

		helpStyle.Width(m.width - 2)

	case styles.ThemeChangeMsg:
		var theme = styles.Theme(msg)
		setStyles(&theme)

		m.shardAllocationTable.SetStyles(shardAllocationTableStyles)

	case ShardAllocationMsg:
		var shardAllocationTableRows []table.Row

		for _, row := range msg {
            var primaryNodes []string
            var replicaNodes []string

            for _, store := range row.Stores {
                if store.Allocation == "primary" {
                    primaryNodes = append(primaryNodes, store.Name)
                } else {
                    replicaNodes = append(replicaNodes, store.Name)
                }
            }

			shardAllocationTableRows = append(shardAllocationTableRows, table.Row{
				row.Index,
				row.Shard,
                strings.Join(primaryNodes, ", "),
                strings.Join(replicaNodes, ", "),
			})
		}

		m.shardAllocationTable.SetRows(shardAllocationTableRows)

	}

	m.shardAllocationTable, cmd = m.shardAllocationTable.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	return lipgloss.JoinVertical(
		lipgloss.Top,
		m.shardAllocationTable.View(),
		helpStyle.Render("[★] Sorting by index first, shard second"),
	)
}

func setStyles(theme *styles.Theme) {
	shardAllocationTableStyles.Header = shardAllocationTableStyles.Header.
		BorderForeground(lipgloss.Color(theme.BorderColorMuted)).
		Foreground(lipgloss.Color(theme.ForegroundColorLight))
	shardAllocationTableStyles.Selected = shardAllocationTableStyles.Selected.
		Foreground(lipgloss.Color(theme.ForegroundColorHighlighted))

	helpStyle = helpStyle.Foreground(lipgloss.Color(theme.ForegroundColorLightMuted))
}
