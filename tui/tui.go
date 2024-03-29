package tui

import (
	"esmon/arguments"
	"esmon/config"
	"esmon/constants"
	"esmon/elasticsearch"
	"esmon/tui/clusterscreen"
	"esmon/tui/loadingscreen"
	"fmt"
	"os"
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
	overviewStyle            = lipgloss.NewStyle().Height(5).MarginBottom(2)
	infoStyle                = lipgloss.NewStyle().Height(5)
	clusterInfoStyle         = lipgloss.NewStyle().Height(5).MarginRight(10)
	clusterHealthGreenStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("29"))
	clusterHealthYellowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	clusterHealthRedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	commandInfoStyle         = lipgloss.NewStyle().Height(5)
	logoStyle                = lipgloss.NewStyle().Align(lipgloss.Right).Bold(true)

	contentStyle = lipgloss.NewStyle().Height(1).Border(lipgloss.RoundedBorder())

	statusStyle                       = lipgloss.NewStyle().Height(1)
	statusGreenStyle                  = statusStyle.Copy().Foreground(lipgloss.Color("15")).Background(lipgloss.Color("29"))
	statusYellowStyle                 = statusStyle.Copy().Foreground(lipgloss.Color("16")).Background(lipgloss.Color("220"))
	statusRedStyle                    = statusStyle.Copy().Foreground(lipgloss.Color("15")).Background(lipgloss.Color("196"))
	statusErrorStyle                  = statusStyle.Copy().Foreground(lipgloss.Color("15")).Background(lipgloss.Color("240"))
	statusRefreshIndicatorGreenStyle  = lipgloss.NewStyle().Inherit(statusGreenStyle)
	statusRefreshIndicatorYellowStyle = lipgloss.NewStyle().Inherit(statusYellowStyle)
	statusRefreshIndicatorRedStyle    = lipgloss.NewStyle().Inherit(statusRedStyle)
	statusRefreshIndicatorErrorStyle  = lipgloss.NewStyle().Inherit(statusErrorStyle)
	statusRefreshInfoGreenStyle       = lipgloss.NewStyle().Inherit(statusGreenStyle)
	statusRefreshInfoYellowStyle      = lipgloss.NewStyle().Inherit(statusYellowStyle)
	statusRefreshInfoRedStyle         = lipgloss.NewStyle().Inherit(statusRedStyle)
	statusRefreshInfoErrorStyle       = lipgloss.NewStyle().Inherit(statusErrorStyle)

	kvTableKeyStyle   = lipgloss.NewStyle().PaddingRight(1)
	kvTableValueStyle = lipgloss.NewStyle().PaddingLeft(1)

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
	}

)

type errMsg error

type keyMap struct {
    shardAllocation key.Binding
    relocatingShards key.Binding
    nodeOverview key.Binding
    indexOverview key.Binding
    clusters key.Binding
    refresh key.Binding
    changeAutorefreshInterval key.Binding
    quit key.Binding
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

	loadingScreen loadingscreen.Model
	clusterScreen clusterscreen.Model

	screen screen

	clusterConfig  []config.ClusterConfig
	currentCluster *config.ClusterConfig
	clusterData    *elasticsearch.ClusterData

	defaultCredentials elasticsearch.Credentials

	refreshing   bool
	refreshError bool
	lastRefresh  time.Time

	refreshIntervalSeconds uint

	refreshSpinner spinner.Model

	httpTimeoutSeconds uint

	err error
}

func NewMainModel() mainModel {
	m := mainModel{}

	m.loadingScreen = loadingscreen.New()
	m.clusterScreen = clusterscreen.New()

	m.screen = loading

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
		contentStyle.Height(m.height - 10)
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

		m.clusterScreen, cmd = m.clusterScreen.Update(tea.WindowSizeMsg{
			Width: m.width - 2, Height: m.height - 10,
		})
		cmds = append(cmds, cmd)

	case tea.KeyMsg:
		if m.screen == loading {
			break
		}

		switch {
		case key.Matches(msg, defaultKeyMap.shardAllocation):
			m.screen = shardAllocation
		case key.Matches(msg, defaultKeyMap.relocatingShards):
			m.screen = relocatingShards
		case key.Matches(msg, defaultKeyMap.nodeOverview):
			m.screen = nodeOverview
		case key.Matches(msg, defaultKeyMap.indexOverview):
			m.screen = indexOverview
		case key.Matches(msg, defaultKeyMap.clusters):
			m.screen = clusters
		case key.Matches(msg, defaultKeyMap.refresh):
			if m.currentCluster != nil && m.refreshIntervalSeconds == 0 && !m.refreshing {
				m.refreshing = true
                cmds = append(
                    cmds, 
                    refreshData(
                        m.currentCluster,
                        &m.defaultCredentials,
                        m.refreshIntervalSeconds,
                    ),
                )
			}
		case key.Matches(msg, defaultKeyMap.changeAutorefreshInterval):
            cmds = append(cmds, changeAutorefreshInterval(m.refreshIntervalSeconds))
		case key.Matches(msg, defaultKeyMap.quit):
            cmds = append(cmds, tea.Quit)
		default:
			m.clusterScreen, cmd = m.clusterScreen.Update(msg)
            cmds = append(cmds, cmd)
		}

	case refreshErrorMsg:
		m.refreshing = false
		m.refreshError = true

	case autorefreshIntervalChangeMsg:
		lastRefreshIntervalSeconds := m.refreshIntervalSeconds
		m.refreshIntervalSeconds = uint(msg)

		statusRefreshInfoWidth := statusRefreshInfoWidth(m.refreshIntervalSeconds)

		statusRefreshIndicatorGreenStyle.Width(m.width - statusRefreshInfoWidth)
		statusRefreshIndicatorYellowStyle.Width(m.width - statusRefreshInfoWidth)
		statusRefreshIndicatorRedStyle.Width(m.width - statusRefreshInfoWidth)
		statusRefreshIndicatorErrorStyle.Width(m.width - statusRefreshInfoWidth)

		if lastRefreshIntervalSeconds == 0 && m.refreshIntervalSeconds > 0 {
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
                        m.refreshIntervalSeconds,
                    ),
                    autorefreshTick(m.refreshIntervalSeconds),
                ),
            )
		}

	case initMsg:
		m.clusterConfig = msg.config.Clusters
		m.currentCluster = msg.currentCluster
		m.clusterData = msg.clusterData

		m.defaultCredentials = elasticsearch.Credentials{
			Username: msg.args.Username,
			Password: msg.args.Password,
		}

		m.refreshIntervalSeconds = msg.config.General.RefreshInterval
		m.httpTimeoutSeconds = msg.config.Http.Timeout

        if m.clusterData != nil {
		    m.lastRefresh = time.Now()
        } else {
            m.refreshError = true
        }

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
                m.refreshIntervalSeconds,
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
			switch {
			case col == 0:
				return kvTableKeyStyle
			case col == 1:
				if row == 2 && m.clusterData != nil {
					switch {
					case m.clusterData.ClusterInfo.Status == "green":
						return kvTableValueStyle.Copy().Inherit(clusterHealthGreenStyle)
					case m.clusterData.ClusterInfo.Status == "yellow":
						return kvTableValueStyle.Copy().Inherit(clusterHealthYellowStyle)
					case m.clusterData.ClusterInfo.Status == "red":
						return kvTableValueStyle.Copy().Inherit(clusterHealthRedStyle)
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

	clusterInfoTable.Row("Cluster:", clusterName)
	clusterInfoTable.Row("Status:", clusterStatus)
	clusterInfoTable.Row("Nodes:", clusterNodes)
	clusterInfoTable.Row("Data:", clusterSize)
	clusterInfoTable.Row("Relocating shards:", clusterRelocatingShards)

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
			switch {
			case col == 0:
				return kvTableKeyStyle
			case col == 1:
				return kvTableValueStyle
			default:
				return lipgloss.NewStyle()
			}
		})

	contentRender := ""
	switch {
	case m.screen == shardAllocation:
		contentRender = "Shard Allocation"
	case m.screen == relocatingShards:
		contentRender = "Relocating Shards"
	case m.screen == nodeOverview:
		contentRender = "Node Overview"
	case m.screen == indexOverview:
		contentRender = "Index Overview"
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
	return tea.Tick(time.Duration(intervalSeconds)*time.Second, func(t time.Time) tea.Msg {
		return autorefreshTickMsg(t)
	})
}

func initProgram() tea.Cmd {
	return func() tea.Msg {
		args, err := arguments.Parse()
		if err != nil {
			fmt.Println("Failed to parse arguments: ", err)
			os.Exit(1)
		}

		conf, err := config.Load(args.Config)
		if err != nil {
			fmt.Println("Failed to load configuration file: ", err)
			os.Exit(1)
		}

		if err := config.Validate(conf); err != nil {
			fmt.Println("Failed to validate configuration file: ", err)
			os.Exit(1)
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
				fmt.Printf("Failed to find cluster with alias %s in configuration.\n", args.Cluster)
				os.Exit(1)
			}

			currentCluster = &conf.Clusters[index]
		}

		if currentCluster != nil {
            credentials, err := elasticsearch.GetCredentials(
                currentCluster, 
                &elasticsearch.Credentials{Username: args.Username, Password: args.Password},
            )

            if err  == nil {
                clusterData, err = elasticsearch.FetchData(
                    currentCluster.Endpoint,
                    credentials,
                    conf.General.RefreshInterval,
                )
            }
		}

		return initMsg{*args, *conf, currentCluster, clusterData}
	}
}

func refreshData(currentCluster *config.ClusterConfig, defaultCredentials *elasticsearch.Credentials, refreshIntervalSeconds uint) tea.Cmd {
	return func() tea.Msg {
        credentials, err := elasticsearch.GetCredentials(currentCluster, defaultCredentials)
        if err != nil {
            return errMsg(err)
        }

		clusterData, err := elasticsearch.FetchData(currentCluster.Endpoint, credentials, refreshIntervalSeconds)
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
