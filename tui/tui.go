package tui

import (
	"context"
	"errors"
	"esmon/arguments"
	"esmon/config"
	"esmon/constants"
	"esmon/elasticsearch"
	"esmon/tui/clusterscreen"
	"esmon/tui/indexscreen"
	"esmon/tui/loadingscreen"
	"esmon/tui/nodescreen"
	"esmon/tui/relocatingshardsscreen"
	"esmon/tui/shardallocationscreen"
	"esmon/tui/styles"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

var (
	defaultTheme = styles.GetTheme(nil)

	logoStyle = lipgloss.NewStyle().Align(lipgloss.Right).Bold(true).Foreground(defaultTheme.ForegroundColorLight)

	overviewStyle            = lipgloss.NewStyle().Height(styles.OverviewHeight).MarginBottom(2)
	infoStyle                = lipgloss.NewStyle().Height(styles.OverviewHeight)
	clusterInfoStyle         = lipgloss.NewStyle().Height(styles.OverviewHeight).MarginRight(10)
	clusterHealthGreenStyle  = lipgloss.NewStyle().Foreground(defaultTheme.BackgroundColorStatusGreen)
	clusterHealthYellowStyle = lipgloss.NewStyle().Foreground(defaultTheme.BackgroundColorStatusYellow)
	clusterHealthRedStyle    = lipgloss.NewStyle().Foreground(defaultTheme.BackgroundColorStatusRed)
	commandInfoStyle         = lipgloss.NewStyle().Height(styles.OverviewHeight)

	contentStyle = lipgloss.NewStyle().Height(1).Border(lipgloss.RoundedBorder()).Foreground(defaultTheme.ForegroundColorLight)
	compactModePaddingStyle = lipgloss.NewStyle().Height(1)

	statusStyle                       = lipgloss.NewStyle().Height(1)
	statusGreenStyle                  = statusStyle.Copy().Foreground(defaultTheme.ForegroundColorLight).Background(defaultTheme.BackgroundColorStatusGreen)
	statusYellowStyle                 = statusStyle.Copy().Foreground(defaultTheme.ForegroundColorDark).Background(defaultTheme.BackgroundColorStatusYellow)
	statusRedStyle                    = statusStyle.Copy().Foreground(defaultTheme.ForegroundColorLight).Background(defaultTheme.BackgroundColorStatusRed)
	statusErrorStyle                  = statusStyle.Copy().Foreground(defaultTheme.ForegroundColorLight).Background(defaultTheme.BackgroundColorStatusError)
	statusRefreshIndicatorGreenStyle  = lipgloss.NewStyle().Inherit(statusGreenStyle)
	statusRefreshIndicatorYellowStyle = lipgloss.NewStyle().Inherit(statusYellowStyle)
	statusRefreshIndicatorRedStyle    = lipgloss.NewStyle().Inherit(statusRedStyle)
	statusRefreshIndicatorErrorStyle  = lipgloss.NewStyle().Inherit(statusErrorStyle)
	statusRefreshInfoGreenStyle       = lipgloss.NewStyle().Inherit(statusGreenStyle)
	statusRefreshInfoYellowStyle      = lipgloss.NewStyle().Inherit(statusYellowStyle)
	statusRefreshInfoRedStyle         = lipgloss.NewStyle().Inherit(statusRedStyle)
	statusRefreshInfoErrorStyle       = lipgloss.NewStyle().Inherit(statusErrorStyle)

	kvTableKeyStyle   = lipgloss.NewStyle().PaddingRight(1).Foreground(defaultTheme.ForegroundColorLight)
	kvTableValueStyle = lipgloss.NewStyle().PaddingLeft(1).Foreground(defaultTheme.ForegroundColorLight)

	defaultKeyMap = keyMap{
		shardAllocation: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("<s>", "Shard allocation"),
		),
		relocatingShards: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("<r>", "Relocating shards"),
		),
		nodeOverview: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("<n>", "Node overview"),
		),
		indexOverview: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("<i>", "Index overview"),
		),
		clusters: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("<c>", "Clusters"),
		),
		compactMode: key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("<v>", "Compact view"),
		),
		refresh: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("<R>", "refresh"),
		),
		changeAutorefreshInterval: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("<a>", "change"),
		),
		quit: key.NewBinding(
			key.WithKeys("q", "ctr-c"),
			key.WithHelp("<q, C-c>", "Quit"),
		),
	}

	mainMenuKeyMap = []*key.Binding{
		&defaultKeyMap.shardAllocation,
		&defaultKeyMap.relocatingShards,
		&defaultKeyMap.nodeOverview,
		&defaultKeyMap.indexOverview,
		&defaultKeyMap.clusters,
		&defaultKeyMap.compactMode,
	}

    refreshContextCancelFunc context.CancelFunc
    refreshTickContextCancelFunc context.CancelFunc
)

type errMsg error

type keyMap struct {
	shardAllocation           key.Binding
	relocatingShards          key.Binding
	nodeOverview              key.Binding
	indexOverview             key.Binding
	clusters                  key.Binding
	compactMode               key.Binding
	refresh                   key.Binding
	changeAutorefreshInterval key.Binding
	quit                      key.Binding
}

type screen int

const (
	loading screen = iota
	shardAllocation
	relocatingShards
	nodeOverview
	indexOverview
	clusters
)

type refreshingMsg bool
type refreshErrorMsg error

type autorefreshIntervalChangeMsg uint
type autorefreshTickMsg time.Time

type initMsg struct {
	args           arguments.Args
	config         config.Config
	currentCluster *config.ClusterConfig
	clusterData    *elasticsearch.ClusterData
}

type clusterDataMsg *elasticsearch.ClusterData

type mainModel struct {
	width  int
	height int

	theme styles.Theme

	loadingScreen loadingscreen.Model
	shardAllocationScreen shardallocationscreen.Model
	relocatingShardsScreen relocatingshardsscreen.Model
	nodeScreen    nodescreen.Model
	indexScreen    indexscreen.Model
	clusterScreen clusterscreen.Model

	screen screen
    compactMode bool

	clusterConfig  []config.ClusterConfig
	currentCluster *config.ClusterConfig
	clusterData    *elasticsearch.ClusterData

	defaultCredentials elasticsearch.Credentials

	refreshing   bool
	refreshError bool
	lastRefresh  time.Time

	refreshIntervalSeconds uint

	refreshSpinner spinner.Model

	httpConfig config.HttpConfig

	err error
}

func NewMainModel() mainModel {
	m := mainModel{}

	m.loadingScreen = loadingscreen.New(&defaultTheme)
	m.shardAllocationScreen = shardallocationscreen.New(&defaultTheme)
	m.relocatingShardsScreen = relocatingshardsscreen.New(&defaultTheme)
	m.nodeScreen = nodescreen.New(&defaultTheme)
	m.indexScreen = indexscreen.New(&defaultTheme)
	m.clusterScreen = clusterscreen.New(&defaultTheme)

	m.screen = loading
    m.compactMode = false

	m.refreshing = false
	m.refreshError = false

	refreshSpinner := spinner.New()
	refreshSpinner.Spinner = spinner.MiniDot
	m.refreshSpinner = refreshSpinner

	return m
}

func (m mainModel) Init() tea.Cmd {
	var cmds []tea.Cmd

	cmds = append(cmds, tea.SetWindowTitle(constants.WindowTitle))
	cmds = append(cmds, initProgram())
	cmds = append(cmds, m.loadingScreen.Init())
	cmds = append(cmds, m.shardAllocationScreen.Init())
	cmds = append(cmds, m.relocatingShardsScreen.Init())
	cmds = append(cmds, m.nodeScreen.Init())
	cmds = append(cmds, m.indexScreen.Init())
	cmds = append(cmds, m.clusterScreen.Init())
	cmds = append(cmds, m.refreshSpinner.Tick)

	return tea.Batch(cmds...)
}

func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

		logoRender := lipgloss.NewStyle().Render(constants.Logo)
		logoWidth, _ := lipgloss.Size(logoRender)

		statusRefreshInfoWidth := statusRefreshInfoWidth(m.refreshIntervalSeconds)
		overviewStyle.Width(m.width)
		infoStyle.Width(m.width - logoWidth)
		contentStyle.Width(m.width - 2)
		contentStyle.Height(m.height -  styles.OverviewHeight - 5)
		compactModePaddingStyle.Height(m.height -  2*styles.OverviewHeight - 3)
		statusStyle.Width(m.width)
		statusGreenStyle.Width(m.width)
		statusYellowStyle.Width(m.width)
		statusRedStyle.Width(m.width)
		statusErrorStyle.Width(m.width)
		statusRefreshIndicatorGreenStyle.Width(m.width - statusRefreshInfoWidth)
		statusRefreshIndicatorYellowStyle.Width(m.width - statusRefreshInfoWidth)
		statusRefreshIndicatorRedStyle.Width(m.width - statusRefreshInfoWidth)
		statusRefreshIndicatorErrorStyle.Width(m.width - statusRefreshInfoWidth)

		m.loadingScreen, cmd = m.loadingScreen.Update(msg)
		cmds = append(cmds, cmd)

		m.shardAllocationScreen, cmd = m.shardAllocationScreen.Update(tea.WindowSizeMsg{
			Width: m.width - 2, Height: m.height - styles.OverviewHeight - 5,
		})
		cmds = append(cmds, cmd)

		m.relocatingShardsScreen, cmd = m.relocatingShardsScreen.Update(tea.WindowSizeMsg{
			Width: m.width - 2, Height: m.height - styles.OverviewHeight - 5,
		})
		cmds = append(cmds, cmd)

		m.nodeScreen, cmd = m.nodeScreen.Update(tea.WindowSizeMsg{
			Width: m.width - 2, Height: m.height - styles.OverviewHeight - 5,
		})
		cmds = append(cmds, cmd)

		m.indexScreen, cmd = m.indexScreen.Update(tea.WindowSizeMsg{
			Width: m.width - 2, Height: m.height - styles.OverviewHeight - 5,
		})
		cmds = append(cmds, cmd)

		m.clusterScreen, cmd = m.clusterScreen.Update(tea.WindowSizeMsg{
			Width: m.width - 2, Height: m.height - styles.OverviewHeight - 5,
		})
		cmds = append(cmds, cmd)

	case tea.KeyMsg:
		if m.screen == loading {
			break
		}

		switch {
		case key.Matches(msg, defaultKeyMap.shardAllocation) && !m.compactMode:
			m.screen = shardAllocation
		case key.Matches(msg, defaultKeyMap.relocatingShards) && !m.compactMode:
			m.screen = relocatingShards
		case key.Matches(msg, defaultKeyMap.nodeOverview) && !m.compactMode:
			m.screen = nodeOverview
		case key.Matches(msg, defaultKeyMap.indexOverview) && !m.compactMode:
			m.screen = indexOverview
		case key.Matches(msg, defaultKeyMap.clusters) && !m.compactMode:
			m.screen = clusters
		case key.Matches(msg, defaultKeyMap.compactMode):
			m.compactMode = !m.compactMode
		case key.Matches(msg, defaultKeyMap.refresh):
			if m.currentCluster != nil && m.refreshIntervalSeconds == 0 && !m.refreshing {
				m.refreshing = true
				cmds = append(
					cmds,
					refreshData(
						m.currentCluster,
						&m.defaultCredentials,
						m.httpConfig,
					),
				)
			}
		case key.Matches(msg, defaultKeyMap.changeAutorefreshInterval):
			cmds = append(cmds, changeAutorefreshInterval(m.refreshIntervalSeconds))
		case key.Matches(msg, defaultKeyMap.quit):
            if refreshContextCancelFunc != nil {
                refreshContextCancelFunc()
            }
            if refreshTickContextCancelFunc != nil {
                refreshTickContextCancelFunc()
            }
			cmds = append(cmds, tea.Quit)
		default:
            switch m.screen {
            case shardAllocation:
			    m.shardAllocationScreen, cmd = m.shardAllocationScreen.Update(msg)
			    cmds = append(cmds, cmd)
            case relocatingShards:
			    m.relocatingShardsScreen, cmd = m.relocatingShardsScreen.Update(msg)
			    cmds = append(cmds, cmd)
            case nodeOverview:
			    m.nodeScreen, cmd = m.nodeScreen.Update(msg)
			    cmds = append(cmds, cmd)
            case indexOverview:
			    m.indexScreen, cmd = m.indexScreen.Update(msg)
			    cmds = append(cmds, cmd)
            case clusters:
			    m.clusterScreen, cmd = m.clusterScreen.Update(msg)
			    cmds = append(cmds, cmd)
            }
		}

	case refreshErrorMsg:
		m.refreshing = false
		m.refreshError = true

	case autorefreshIntervalChangeMsg:
        if refreshTickContextCancelFunc != nil {
            refreshTickContextCancelFunc()
        }

		m.refreshIntervalSeconds = uint(msg)

		statusRefreshInfoWidth := statusRefreshInfoWidth(m.refreshIntervalSeconds)

		statusRefreshIndicatorGreenStyle.Width(m.width - statusRefreshInfoWidth)
		statusRefreshIndicatorYellowStyle.Width(m.width - statusRefreshInfoWidth)
		statusRefreshIndicatorRedStyle.Width(m.width - statusRefreshInfoWidth)
		statusRefreshIndicatorErrorStyle.Width(m.width - statusRefreshInfoWidth)

		if m.refreshIntervalSeconds > 0 {
			cmds = append(cmds, autorefreshTick(m.refreshIntervalSeconds))
		}

	case autorefreshTickMsg:
		if m.refreshIntervalSeconds > 0 {
			m.refreshing = true
			cmds = append(
				cmds,
				tea.Sequence(
					refreshData(
						m.currentCluster,
						&m.defaultCredentials,
						m.httpConfig,
					),
					autorefreshTick(m.refreshIntervalSeconds),
				),
			)
		}

	case initMsg:
		m.theme = styles.GetTheme(&msg.config.Theme)
		setStyles(m.theme)

		m.loadingScreen, cmd = m.loadingScreen.Update(styles.ThemeChangeMsg(m.theme))
		cmds = append(cmds, cmd)

		m.shardAllocationScreen, cmd = m.shardAllocationScreen.Update(styles.ThemeChangeMsg(m.theme))
		cmds = append(cmds, cmd)

		m.relocatingShardsScreen, cmd = m.relocatingShardsScreen.Update(styles.ThemeChangeMsg(m.theme))
		cmds = append(cmds, cmd)

		m.nodeScreen, cmd = m.nodeScreen.Update(styles.ThemeChangeMsg(m.theme))
		cmds = append(cmds, cmd)

		m.indexScreen, cmd = m.indexScreen.Update(styles.ThemeChangeMsg(m.theme))
		cmds = append(cmds, cmd)

		m.clusterScreen, cmd = m.clusterScreen.Update(styles.ThemeChangeMsg(m.theme))
		cmds = append(cmds, cmd)

		m.clusterConfig = msg.config.Clusters
		m.currentCluster = msg.currentCluster
		m.clusterData = msg.clusterData

		m.defaultCredentials = elasticsearch.Credentials{
			Username: msg.args.Username,
			Password: msg.args.Password,
		}

		m.refreshIntervalSeconds = msg.config.General.RefreshInterval

		httpInsecure := msg.config.Http.Insecure
		if msg.args.Insecure != nil {
			httpInsecure = *msg.args.Insecure
		}

		m.httpConfig = config.HttpConfig{
			Timeout:  msg.config.Http.Timeout,
			Insecure: httpInsecure,
		}

		if m.clusterData != nil {
			m.lastRefresh = time.Now()

            m.shardAllocationScreen, cmd = m.shardAllocationScreen.Update(
                shardallocationscreen.ShardAllocationMsg(m.clusterData.ShardStores),
            )
			cmds = append(cmds, cmd)

            m.relocatingShardsScreen, cmd = m.relocatingShardsScreen.Update(
                relocatingshardsscreen.ShardMsg(m.clusterData.Recoveries),
            )
			cmds = append(cmds, cmd)

			m.nodeScreen, cmd = m.nodeScreen.Update(
                nodescreen.NodeMsg {
                    Nodes: m.clusterData.NodeStats,
                    MasterNode: m.clusterData.MasterNode,
                },
            )
			cmds = append(cmds, cmd)

            m.indexScreen, cmd = m.indexScreen.Update(
                indexscreen.IndexMsg(m.clusterData.IndexStats),
            )
			cmds = append(cmds, cmd)
		} else {
			m.refreshError = true
		}

        m.compactMode = msg.args.CompactMode

		if m.currentCluster != nil {
			m.screen = shardAllocation
		} else {
			m.refreshError = true
			m.screen = clusters
		}

		statusRefreshInfoWidth := statusRefreshInfoWidth(m.refreshIntervalSeconds)

		statusRefreshIndicatorGreenStyle.Width(m.width - statusRefreshInfoWidth)
		statusRefreshIndicatorYellowStyle.Width(m.width - statusRefreshInfoWidth)
		statusRefreshIndicatorRedStyle.Width(m.width - statusRefreshInfoWidth)
		statusRefreshIndicatorErrorStyle.Width(m.width - statusRefreshInfoWidth)

		m.clusterScreen, cmd = m.clusterScreen.Update(clusterscreen.ClusterMsg(m.clusterConfig))
		cmds = append(cmds, cmd)

		if m.currentCluster != nil && m.refreshIntervalSeconds > 0 {
			cmds = append(cmds, autorefreshTick(m.refreshIntervalSeconds))
		}

	case clusterscreen.ClusterChangeMsg:
        if refreshContextCancelFunc != nil {
            refreshContextCancelFunc()
        }
        if refreshTickContextCancelFunc != nil {
            refreshTickContextCancelFunc()
        }

		index := slices.IndexFunc(
			m.clusterConfig,
			func(c config.ClusterConfig) bool {
				return c.Endpoint == string(msg)
			})

		m.currentCluster = &m.clusterConfig[index]
		m.clusterData = nil

		m.refreshing = true
		m.lastRefresh = time.Time{}

		cmds = append(
			cmds,
			refreshData(
				m.currentCluster,
				&m.defaultCredentials,
				m.httpConfig,
			),
		)

	case clusterDataMsg:
		m.refreshing = false
		m.refreshError = false
		m.clusterData = msg
		m.lastRefresh = time.Now()

	case errMsg:
		m.refreshing = false
		m.err = msg

	default:
		m.refreshSpinner, cmd = m.refreshSpinner.Update(msg)
		cmds = append(cmds, cmd)

		m.loadingScreen, cmd = m.loadingScreen.Update(msg)
		cmds = append(cmds, cmd)

	}

	return m, tea.Batch(cmds...)
}

func (m mainModel) View() string {
	if m.screen == loading {
		return m.loadingScreen.View()
	}
	if m.err != nil {
		return m.err.Error()
	}

	clusterInfoTable := table.New().
		BorderTop(false).
		BorderRight(false).
		BorderBottom(false).
		BorderLeft(false).
		BorderColumn(false).
		StyleFunc(func(row, col int) lipgloss.Style {
			switch col {
			case 0:
				return kvTableKeyStyle
			case 1:
				if row == 2 && m.clusterData != nil {
					switch {
					case m.clusterData.ClusterInfo.Status == "green":
						return kvTableValueStyle.Copy().Foreground(m.theme.BackgroundColorStatusGreen)
					case m.clusterData.ClusterInfo.Status == "yellow":
						return kvTableValueStyle.Copy().Foreground(m.theme.BackgroundColorStatusYellow)
					case m.clusterData.ClusterInfo.Status == "red":
						return kvTableValueStyle.Copy().Foreground(m.theme.BackgroundColorStatusRed)
					}
				}
				return kvTableValueStyle
			default:
				return lipgloss.NewStyle()
			}
		})

	clusterName := ""
	if m.clusterData != nil {
		clusterName = m.clusterData.ClusterInfo.ClusterName
	}
	clusterStatus := ""
	if m.clusterData != nil {
		clusterStatus = m.clusterData.ClusterInfo.Status
	}
	clusterNodes := ""
	if m.clusterData != nil {
		clusterNodes = fmt.Sprintf("%d", m.clusterData.ClusterInfo.NumberOfNodes)
	}
	clusterSize := ""
	if m.clusterData != nil {
		clusterSize = strings.ToUpper(m.clusterData.ClusterStats.Indices.Store.Size)
	}
	clusterRelocatingShards := ""
	if m.clusterData != nil {
		clusterRelocatingShards = fmt.Sprintf("%d", m.clusterData.ClusterInfo.RelocatingShards)
	}
	clusterActiveShardsPercent := ""
	if m.clusterData != nil {
		clusterActiveShardsPercent =  m.clusterData.ClusterInfo.ActiveShardsPercent
	}

	clusterInfoTable.Row("Cluster:", clusterName)
	clusterInfoTable.Row("Status:", clusterStatus)
	clusterInfoTable.Row("Nodes:", clusterNodes)
	clusterInfoTable.Row("Data:", clusterSize)
	clusterInfoTable.Row("Relocating shards:", clusterRelocatingShards)
	clusterInfoTable.Row("Active shards:", clusterActiveShardsPercent)

	var commands [][]string
	for _, keyBinding := range mainMenuKeyMap {
		commands = append(commands, []string{keyBinding.Help().Key, keyBinding.Help().Desc})
	}
	commandTable := table.New().
		Rows(commands...).
		BorderTop(false).
		BorderRight(false).
		BorderBottom(false).
		BorderLeft(false).
		BorderColumn(false).
		StyleFunc(func(row, col int) lipgloss.Style {
			switch col {
			case 0:
                if row == int(m.screen) {
                    return kvTableKeyStyle.Copy().Foreground(m.theme.ForegroundColorHighlighted)
                } else {
				    return kvTableKeyStyle
                }
			case 1:
                if row == int(m.screen) {
                    return kvTableValueStyle.Copy().Foreground(m.theme.ForegroundColorHighlighted)
                } else {
				    return kvTableValueStyle
                }
			default:
				return lipgloss.NewStyle()
			}
		})

	contentRender := ""
	switch {
	case m.screen == shardAllocation:
		contentRender = m.shardAllocationScreen.View()
	case m.screen == relocatingShards:
		contentRender = m.relocatingShardsScreen.View()
	case m.screen == nodeOverview:
		contentRender = m.nodeScreen.View()
	case m.screen == indexOverview:
		contentRender = m.indexScreen.View()
	case m.screen == clusters:
		contentRender = m.clusterScreen.View()
	}

	refreshingString := ""
	if m.refreshing {
		refreshingString = fmt.Sprintf("%s Refreshing", m.refreshSpinner.View())
	} else {
		refreshErrorString := ""
		if m.refreshError {
			refreshErrorString = "⚠ "
		}

		if m.lastRefresh.IsZero() {
			refreshingString = refreshErrorString
		} else {
			refreshingString = fmt.Sprintf("%sLast refresh at %s", refreshErrorString, m.lastRefresh.Format("15:04:05"))
		}
	}

	statusRefreshIndicatorRender := ""
	switch {
	case m.clusterData == nil:
		statusRefreshIndicatorRender = statusRefreshIndicatorErrorStyle.Render(refreshingString)
	case m.refreshError == true:
		statusRefreshIndicatorRender = statusRefreshIndicatorErrorStyle.Render(refreshingString)
	case m.clusterData.ClusterInfo.Status == "green":
		statusRefreshIndicatorRender = statusRefreshIndicatorGreenStyle.Render(refreshingString)
	case m.clusterData.ClusterInfo.Status == "yellow":
		statusRefreshIndicatorRender = statusRefreshIndicatorYellowStyle.Render(refreshingString)
	case m.clusterData.ClusterInfo.Status == "red":
		statusRefreshIndicatorRender = statusRefreshIndicatorRedStyle.Render(refreshingString)
	}

	refreshInfoString := refreshInfoStatus(m.refreshIntervalSeconds)
	statusRefreshInfoRender := ""
	switch {
	case m.clusterData == nil:
		statusRefreshInfoRender = statusRefreshInfoErrorStyle.Render(refreshInfoString)
	case m.refreshError == true:
		statusRefreshInfoRender = statusRefreshInfoErrorStyle.Render(refreshInfoString)
	case m.clusterData.ClusterInfo.Status == "green":
		statusRefreshInfoRender = statusRefreshInfoGreenStyle.Render(refreshInfoString)
	case m.clusterData.ClusterInfo.Status == "yellow":
		statusRefreshInfoRender = statusRefreshInfoYellowStyle.Render(refreshInfoString)
	case m.clusterData.ClusterInfo.Status == "red":
		statusRefreshInfoRender = statusRefreshInfoRedStyle.Render(refreshInfoString)
	}

    if m.compactMode {
        return lipgloss.JoinVertical(
            lipgloss.Top,
            logoStyle.Copy().PaddingBottom(1).Render(constants.Logo),
            lipgloss.
                NewStyle().
                PaddingBottom(1).
                Foreground(m.theme.ForegroundColorLight).
                Render("<v> Normal mode"),
            clusterInfoTable.Render(),
            compactModePaddingStyle.Render(),
            statusStyle.Render(
                lipgloss.JoinHorizontal(
                    lipgloss.Top,
                    statusRefreshIndicatorRender,
                    statusRefreshInfoRender,
                ),
            ),
        )
    }

	return lipgloss.JoinVertical(
		lipgloss.Top,
		overviewStyle.Render(
			lipgloss.JoinHorizontal(
				lipgloss.Top,
				infoStyle.Render(
					lipgloss.JoinHorizontal(
						lipgloss.Top,
						clusterInfoStyle.Render(clusterInfoTable.Render()),
						commandInfoStyle.Render(commandTable.Render()))),
				logoStyle.Render(constants.Logo))),
		contentStyle.Render(contentRender),
		statusStyle.Render(
			lipgloss.JoinHorizontal(
				lipgloss.Top,
				statusRefreshIndicatorRender,
				statusRefreshInfoRender)))
}

func refreshInfoStatus(refreshIntervalSeconds uint) string {
	refreshInfoString := "Autorefresh: "

	if refreshIntervalSeconds == 0 {
		refreshInfoString += "OFF"
		refreshInfoString += " | <R> refresh <a> change"
	} else {
		intervalString := ""
		if refreshIntervalSeconds < 60 {
			intervalString = fmt.Sprintf("%ds", refreshIntervalSeconds)
		} else {
			intervalString = fmt.Sprintf("%dm", refreshIntervalSeconds/60)
		}
		refreshInfoString += intervalString
		refreshInfoString += " | <a> change"
	}

	return refreshInfoString
}

func statusRefreshInfoWidth(refreshIntervalSeconds uint) int {
	statusRefreshInfoRender := lipgloss.NewStyle().Render(refreshInfoStatus(refreshIntervalSeconds))
	statusRefreshInfoWidth, _ := lipgloss.Size(statusRefreshInfoRender)
	return statusRefreshInfoWidth
}

func autorefreshTick(intervalSeconds uint) tea.Cmd {
    return func() tea.Msg {
        var ctx context.Context
        ctx, refreshTickContextCancelFunc = context.WithCancel(context.Background())

		timer := time.NewTimer(time.Duration(intervalSeconds) * time.Second)

        select {
        case t := <- timer.C:
		    return autorefreshTickMsg(t)
        case <- ctx.Done():
            return nil
        }
	}
}

func initProgram() tea.Cmd {
	return func() tea.Msg {
		args, err := arguments.Parse()
		if err != nil {
            return errMsg(errors.New("Failed to parse arguments: " + err.Error()))
		}

		conf, err := config.Load(args.Config)
		if err != nil {
            return errMsg(errors.New("Failed to load configuratin file: " + err.Error()))
		}

		if err := config.Validate(conf); err != nil {
            return errMsg(errors.New("Failed to validate configuratin file: " + err.Error()))
		}

		var currentCluster *config.ClusterConfig = nil
		var clusterData *elasticsearch.ClusterData = nil

		if args.Endpoint != "" {
			conf.Clusters = []config.ClusterConfig{
				{
					Endpoint: args.Endpoint,
					Username: args.Username,
					Password: args.Password,
				},
			}
			currentCluster = &conf.Clusters[0]
		} else if args.Cluster != "" {
			index := slices.IndexFunc(
				conf.Clusters,
				func(c config.ClusterConfig) bool {
					return c.Alias == args.Cluster
				})

			if index == -1 {
				return errMsg(
                    errors.New(
                        fmt.Sprintf(
                            "Failed to find cluster with alias %s in configuration.\n",
                            args.Cluster,
                        ),
                    ),
                )
			}

			currentCluster = &conf.Clusters[index]
		}

		if currentCluster != nil {
			credentials, err := elasticsearch.GetCredentials(
				currentCluster,
				&elasticsearch.Credentials{Username: args.Username, Password: args.Password},
			)

			var insecure = conf.Http.Insecure
			if args.Insecure != nil {
				insecure = *args.Insecure
			}

			if err == nil {
                var ctx context.Context
                ctx, refreshContextCancelFunc = context.WithCancel(context.Background())
				clusterData, err = elasticsearch.FetchData(
                    ctx,
					currentCluster.Endpoint,
					credentials,
					conf.General.RefreshInterval,
					insecure,
				)
			}
		}

		return initMsg{*args, *conf, currentCluster, clusterData}
	}
}

func setStyles(theme styles.Theme) {
	logoStyle = logoStyle.Foreground(theme.LogoColor)

	clusterHealthGreenStyle = clusterHealthGreenStyle.Foreground(theme.BackgroundColorStatusGreen)
	clusterHealthYellowStyle = clusterHealthYellowStyle.Foreground(theme.BackgroundColorStatusYellow)
	clusterHealthRedStyle = clusterHealthRedStyle.Foreground(theme.BackgroundColorStatusRed)

	contentStyle = contentStyle.Foreground(theme.BorderColor)

	statusGreenStyle = statusGreenStyle.Foreground(theme.ForegroundColorLight).Background(theme.BackgroundColorStatusGreen)
	statusYellowStyle = statusYellowStyle.Foreground(theme.ForegroundColorDark).Background(theme.BackgroundColorStatusYellow)
	statusRedStyle = statusRedStyle.Foreground(theme.ForegroundColorLight).Background(theme.BackgroundColorStatusRed)
	statusErrorStyle = statusErrorStyle.Foreground(theme.ForegroundColorLight).Background(theme.BackgroundColorStatusError)
	statusRefreshIndicatorGreenStyle = statusRefreshIndicatorGreenStyle.Foreground(theme.ForegroundColorLight).Background(theme.BackgroundColorStatusGreen)
	statusRefreshIndicatorYellowStyle = statusRefreshIndicatorYellowStyle.Foreground(theme.ForegroundColorDark).Background(theme.BackgroundColorStatusYellow)
	statusRefreshIndicatorRedStyle = statusRefreshIndicatorRedStyle.Foreground(theme.ForegroundColorLight).Background(theme.BackgroundColorStatusRed)
	statusRefreshIndicatorErrorStyle = statusRefreshIndicatorErrorStyle.Foreground(theme.ForegroundColorLight).Background(theme.BackgroundColorStatusError)
	statusRefreshInfoGreenStyle = statusRefreshInfoGreenStyle.Foreground(theme.ForegroundColorLight).Background(theme.BackgroundColorStatusGreen)
	statusRefreshInfoYellowStyle = statusRefreshInfoYellowStyle.Foreground(theme.ForegroundColorDark).Background(theme.BackgroundColorStatusYellow)
	statusRefreshInfoRedStyle = statusRefreshInfoRedStyle.Foreground(theme.ForegroundColorLight).Background(theme.BackgroundColorStatusRed)
	statusRefreshInfoErrorStyle = statusRefreshInfoErrorStyle.Foreground(theme.ForegroundColorLight).Background(theme.BackgroundColorStatusError)

	kvTableKeyStyle = kvTableKeyStyle.Foreground(theme.ForegroundColorLight)
	kvTableValueStyle = kvTableValueStyle.Foreground(theme.ForegroundColorLight)
}

func refreshData(currentCluster *config.ClusterConfig, defaultCredentials *elasticsearch.Credentials, httpConfig config.HttpConfig) tea.Cmd {
	return func() tea.Msg {
		credentials, err := elasticsearch.GetCredentials(currentCluster, defaultCredentials)
		if err != nil {
			return errMsg(err)
		}

        var ctx context.Context
        ctx, refreshContextCancelFunc = context.WithCancel(context.Background())

		clusterData, err := elasticsearch.FetchData(
            ctx,
            currentCluster.Endpoint,
            credentials,
            httpConfig.Timeout,
            httpConfig.Insecure,
        )
		if err != nil {
			return refreshErrorMsg(err)
		}

		return clusterDataMsg(clusterData)
	}
}

func changeAutorefreshInterval(currentInterval uint) tea.Cmd {
	return func() tea.Msg {
		switch currentInterval {
		case 0:
			return autorefreshIntervalChangeMsg(1)
		case 1:
			return autorefreshIntervalChangeMsg(5)
		case 5:
			return autorefreshIntervalChangeMsg(10)
		case 10:
			return autorefreshIntervalChangeMsg(30)
		case 30:
			return autorefreshIntervalChangeMsg(60)
		case 60:
			return autorefreshIntervalChangeMsg(300)
		case 300:
			return autorefreshIntervalChangeMsg(600)
		case 600:
			return autorefreshIntervalChangeMsg(0)
		default:
			return autorefreshIntervalChangeMsg(5)
		}
	}
}
