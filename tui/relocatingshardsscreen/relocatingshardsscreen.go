package relocatingshardsscreen

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

	shardTableColumns []table.Column = []table.Column{
		{Title: "Index", Width: 20},
		{Title: "Shard", Width: 10},
		{Title: "Source", Width: 20},
		{Title: "Target", Width: 20},
		{Title: "Progress [%]", Width: 20},
		{Title: "↓Time", Width: 10},
	}

	shardTableRows []table.Row

	shardTableStyles = table.DefaultStyles()

	helpStyle = lipgloss.NewStyle().Height(1).Foreground(defaultTheme.ForegroundColorLightMuted)
)

type ShardMsg []elasticsearch.Recovery

type Model struct {
	width  int
	height int

	shardTable table.Model
}

func New(theme *styles.Theme) Model {
	m := Model{}

	m.shardTable = table.New(
		table.WithColumns(shardTableColumns),
		table.WithRows(shardTableRows),
		table.WithFocused(true),
	)

	shardTableStyles.Header = shardTableStyles.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(theme.BorderColorMuted)).
		BorderBottom(true).
		Bold(false).
		Foreground(lipgloss.Color(theme.ForegroundColorLight))
	shardTableStyles.Selected = shardTableStyles.Selected.
		Foreground(lipgloss.Color(theme.ForegroundColorHighlighted)).
		Bold(false)
	m.shardTable.SetStyles(shardTableStyles)

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

		for index := range shardTableColumns {
			shardTableColumns[index].Width = m.width/len(shardTableColumns) - 2
		}

		m.shardTable.SetHeight(m.height - 3)
		m.shardTable.SetColumns(shardTableColumns)

		helpStyle.Width(m.width - 2)

	case styles.ThemeChangeMsg:
		var theme = styles.Theme(msg)
		setStyles(&theme)

		m.shardTable.SetStyles(shardTableStyles)

	case ShardMsg:
		var shardTableRows []table.Row

		for _, row := range msg {
			shard := fmt.Sprint(row.ID)
			if row.Primary {
				shard += "[P]"
			} else {
                shard += "[R]"
            }

			shardTableRows = append(shardTableRows, table.Row{
				row.Index.Name,
                shard,
				row.Source.Peer.PeerName(),
				row.Target.Peer.PeerName(),
                fmt.Sprintf(
                    "%s (%s/%s)",
                    strings.TrimSuffix(row.Index.Size.Percent, "%"),
                    strings.ToUpper(row.Index.Size.Recovered),
                    strings.ToUpper(row.Index.Size.Total),
                ),
                row.TotalTime,
			})
		}

		m.shardTable.SetRows(shardTableRows)

	}

	m.shardTable, cmd = m.shardTable.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	return lipgloss.JoinVertical(
		lipgloss.Top,
		m.shardTable.View(),
		helpStyle.Render("[P] Primary Shard • [R] Replica Shard"),
	)
}

func setStyles(theme *styles.Theme) {
	shardTableStyles.Header = shardTableStyles.Header.
		BorderForeground(lipgloss.Color(theme.BorderColorMuted)).
		Foreground(lipgloss.Color(theme.ForegroundColorLight))
	shardTableStyles.Selected = shardTableStyles.Selected.
		Foreground(lipgloss.Color(theme.ForegroundColorHighlighted))

	helpStyle = helpStyle.Foreground(lipgloss.Color(theme.ForegroundColorLightMuted))
}
