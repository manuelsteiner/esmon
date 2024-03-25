package main

import (
    "fmt"
    "io"
    "encoding/json"
	"log"
    "net/http"
    "strings"
    "sync"
    "time"

    "github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
    "github.com/charmbracelet/lipgloss/table"
    "golang.org/x/sync/errgroup"
)

const (
    logo = ` _____   ____    __  __                 
| ____| / ___|  |  \/  |   ___    _ __  
|  _|   \___ \  | |\/| |  / _ \  | '_ \ 
| |___   ___) | | |  | | | (_) | | | | |
|_____| |____/  |_|  |_|  \___/  |_| |_|`

    baseUrl = "http://localhost:9200"
    clusterHealthPath = "/_cluster/health"
    clusterStatsPath = "/_cluster/stats"
)

var (
	overviewStyle = lipgloss.NewStyle().Height(5).MarginBottom(2)
    infoStyle = lipgloss.NewStyle().Height(5)
    clusterInfoStyle = lipgloss.NewStyle().Height(5).MarginRight(10)
    clusterHealthGreenStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("29"))
    clusterHealthYellowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
    clusterHealthRedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
    commandInfoStyle = lipgloss.NewStyle().Height(5)
    logoStyle = lipgloss.NewStyle().Align(lipgloss.Right).Bold(true)

    contentStyle = lipgloss.NewStyle().Height(1)

	statusStyle = lipgloss.NewStyle().Height(1)
    statusGreenStyle = statusStyle.Copy().Background(lipgloss.Color("29"))
    statusYellowStyle = statusStyle.Copy().Foreground(lipgloss.Color("16")).Background(lipgloss.Color("220"))
    statusRedStyle = statusStyle.Copy().Background(lipgloss.Color("196"))
    statusErrorStyle = statusStyle.Copy().Background(lipgloss.Color("240"))
    statusRefreshIndicatorGreenStyle = lipgloss.NewStyle().Inherit(statusGreenStyle)
    statusRefreshIndicatorYellowStyle = lipgloss.NewStyle().Inherit(statusYellowStyle)
    statusRefreshIndicatorRedStyle = lipgloss.NewStyle().Inherit(statusRedStyle)
    statusRefreshIndicatorErrorStyle = lipgloss.NewStyle().Inherit(statusErrorStyle)
    statusRefreshInfoGreenStyle = lipgloss.NewStyle().Inherit(statusGreenStyle)
    statusRefreshInfoYellowStyle = lipgloss.NewStyle().Inherit(statusYellowStyle)
    statusRefreshInfoRedStyle = lipgloss.NewStyle().Inherit(statusRedStyle)
    statusRefreshInfoErrorStyle = lipgloss.NewStyle().Inherit(statusErrorStyle)

    kvTableKeyStyle = lipgloss.NewStyle().PaddingRight(1)
    kvTableValueStyle = lipgloss.NewStyle().PaddingLeft(1)
    
    commands = [][]string { 
        {"<s>", "Shard allocation"},
        {"<r>", "Relocating shards"},
        {"<n>", "Node overview"},
        {"<i>", "Index overview"},
        {"<c>", "Clusters"},
    }
)

type errMsg error

type ContentScreen int

const (
    ShardAllocation ContentScreen = iota
    RelocatingShards
    NodeOverview
    IndexOverview
    Clusters
) 

type refreshingMsg bool
type refreshErrorMsg error

type autorefreshIntervalChangeMsg uint
type autorefreshTickMsg time.Time

type ClusterData struct {
    clusterInfo ClusterInfo
    clusterStats ClusterStats
}

type ClusterInfo struct {
	ClusterName                 string  `json:"cluster_name"`
	Status                      string  `json:"status"`
	TimedOut                    bool    `json:"timed_out"`
	NumberOfNodes               int     `json:"number_of_nodes"`
	NumberOfDataNodes           int     `json:"number_of_data_nodes"`
	ActivePrimaryShards         int     `json:"active_primary_shards"`
	ActiveShards                int     `json:"active_shards"`
	RelocatingShards            int     `json:"relocating_shards"`
	InitializingShards          int     `json:"initializing_shards"`
	UnassignedShards            int     `json:"unassigned_shards"`
	DelayedUnassignedShards     int     `json:"delayed_unassigned_shards"`
	NumberOfPendingTasks        int     `json:"number_of_pending_tasks"`
	NumberOfInFlightFetch       int     `json:"number_of_in_flight_fetch"`
	TaskMaxWaitingInQueueMillis int     `json:"task_max_waiting_in_queue_millis"`
	ActiveShardsPercentAsNumber float64 `json:"active_shards_percent_as_number"`
}

type ClusterStats struct {
    Indices struct {
	    Store struct {
		    Size                    string `json:"size"`
		    SizeInBytes             int    `json:"size_in_bytes"`
		    TotalDataSetSize        string `json:"total_data_set_size"`
		    TotalDataSetSizeInBytes int    `json:"total_data_set_size_in_bytes"`
		    Reserved                string `json:"reserved"`
		    ReservedInBytes         int    `json:"reserved_in_bytes"`
	    } `json:"store"`
    } `json:"indices"`
}

type clusterDataMsg ClusterData

type mainModel struct {
    width int
    height int

    contentScreen ContentScreen

    clusterData ClusterData

    refreshing bool
    refreshError bool
    lastRefresh time.Time

    refreshIntervalSeconds uint

    refreshSpinner spinner.Model

    err error
}

func newModel() mainModel {
	m := mainModel{}

    m.contentScreen = ShardAllocation

    m.clusterData = ClusterData{}

    var waitGroup sync.WaitGroup

    waitGroup.Add(1)
    go func() {
        clusterInfo, _ :=  fetchClusterInfo()
        m.clusterData.clusterInfo = *clusterInfo
        waitGroup.Done()
    }()

    waitGroup.Add(1)
    go func() {
        clusterStats, _ := fetchClusterStats() 
        m.clusterData.clusterStats = *clusterStats
        waitGroup.Done()
    }()

    waitGroup.Wait()

    m.refreshing = false
    m.refreshError = false
    m.lastRefresh = time.Now()

    m.refreshIntervalSeconds = 0  

    refreshSpinner := spinner.New()
    refreshSpinner.Spinner = spinner.MiniDot
    m.refreshSpinner = refreshSpinner

	return m
}

func (m mainModel) Init() tea.Cmd {
    if m.refreshIntervalSeconds == 0 {
        return m.refreshSpinner.Tick
    } else {
        return tea.Batch(autorefreshTick(m.refreshIntervalSeconds), m.refreshSpinner.Tick)
    }
}

func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
        case tea.WindowSizeMsg:
            m.width, m.height = msg.Width, msg.Height

            logoRender := lipgloss.NewStyle().Render(logo)
            logoWidth, _ := lipgloss.Size(logoRender)

            statusRefreshInfoWidth := statusRefreshInfoWidth(m.refreshIntervalSeconds)
            overviewStyle.Width(m.width)
            infoStyle.Width(m.width - logoWidth) 
            contentStyle.Width(m.width)
            contentStyle.Height(m.height - 8)
            statusStyle.Width(m.width)
            statusGreenStyle.Width(m.width)
            statusYellowStyle.Width(m.width)
            statusRedStyle.Width(m.width)
            statusErrorStyle.Width(m.width)
            statusRefreshIndicatorGreenStyle.Width(m.width - statusRefreshInfoWidth)
            statusRefreshIndicatorYellowStyle.Width(m.width - statusRefreshInfoWidth)
            statusRefreshIndicatorRedStyle.Width(m.width - statusRefreshInfoWidth)
            statusRefreshIndicatorErrorStyle.Width(m.width - statusRefreshInfoWidth)

            return m, nil

        case tea.KeyMsg:
            switch msg.String() {
                case "s":
                    m.contentScreen = ShardAllocation
                    return m, nil
                case "r":
                    m.contentScreen = RelocatingShards
                    return m, nil
                case "n":
                    m.contentScreen = NodeOverview
                    return m, nil
                case "i":
                    m.contentScreen = IndexOverview
                    return m, nil
                case "c":
                    m.contentScreen = Clusters
                    return m, nil
                case "R":
                    if m.refreshIntervalSeconds == 0  && !m.refreshing {
                        m.refreshing = true
                        //m.refreshError = false
                        return m, refreshData()
                    } else {
                        return m, nil
                    }
                case "a":
                    return m, changeAutorefreshInterval(m.refreshIntervalSeconds)
                case "ctrl+c", "q":
                    return m, tea.Quit
                default:
                    return m, nil
            }

        case refreshErrorMsg:
            m.refreshing = false
            m.refreshError = true
            return m, nil

        case autorefreshIntervalChangeMsg:
            lastRefreshIntervalSeconds := m.refreshIntervalSeconds
            m.refreshIntervalSeconds = uint(msg)

            statusRefreshInfoWidth := statusRefreshInfoWidth(m.refreshIntervalSeconds)

            statusRefreshIndicatorGreenStyle.Width(m.width - statusRefreshInfoWidth)
            statusRefreshIndicatorYellowStyle.Width(m.width - statusRefreshInfoWidth)
            statusRefreshIndicatorRedStyle.Width(m.width - statusRefreshInfoWidth)
            statusRefreshIndicatorErrorStyle.Width(m.width - statusRefreshInfoWidth)
            
            if  lastRefreshIntervalSeconds == 0 && m.refreshIntervalSeconds > 0  {
                return m, autorefreshTick(m.refreshIntervalSeconds)
            } else {
                return m, nil
            }

        case autorefreshTickMsg:
            if m.refreshIntervalSeconds == 0  {
                return m, nil
            } else {
                m.refreshing = true
                //m.refreshError = false
                return m, tea.Sequence(refreshData(), autorefreshTick(m.refreshIntervalSeconds))
            }

        case clusterDataMsg:
            m.refreshing = false
            m.refreshError = false
            m.clusterData = ClusterData(msg)
            m.lastRefresh = time.Now()
            return m, nil

        case errMsg:
            m.refreshing = false
            m.err = msg
            return m, nil

        default:
            var command tea.Cmd
            m.refreshSpinner, command = m.refreshSpinner.Update(msg)
            return m, command
    }
}

func (m mainModel) View() string {
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
                    if row == 2 {
                        switch {
                            case m.clusterData.clusterInfo.Status == "green":
                                return kvTableValueStyle.Copy().Inherit(clusterHealthGreenStyle)
                            case m.clusterData.clusterInfo.Status == "yellow":
                                return kvTableValueStyle.Copy().Inherit(clusterHealthYellowStyle)
                            case m.clusterData.clusterInfo.Status == "red":
                                return kvTableValueStyle.Copy().Inherit(clusterHealthRedStyle)
                        }
                    }
                    return kvTableValueStyle
                default:
                    return lipgloss.NewStyle()
            }
        })

    clusterInfoTable.Row("Cluster:", m.clusterData.clusterInfo.ClusterName)
    clusterInfoTable.Row("Status:", m.clusterData.clusterInfo.Status)
    clusterInfoTable.Row("Nodes:", fmt.Sprintf("%d", m.clusterData.clusterInfo.NumberOfNodes))
    clusterInfoTable.Row("Size:", strings.ToUpper(m.clusterData.clusterStats.Indices.Store.Size))
    clusterInfoTable.Row("Relocating shards:", fmt.Sprintf("%d", m.clusterData.clusterInfo.RelocatingShards))

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

    contentString := ""
    switch {
        case m.contentScreen == ShardAllocation:
            contentString = "Shard Allocation"
        case m.contentScreen == RelocatingShards:
            contentString = "Relocating Shards"
        case m.contentScreen == NodeOverview:
            contentString = "Node Overview"
        case m.contentScreen == IndexOverview:
            contentString = "Index Overview"
        case m.contentScreen == Clusters:
            contentString = "Clusters"
    }

    refreshingString := ""
    if m.refreshing {
        refreshingString = fmt.Sprintf("%s Refreshing", m.refreshSpinner.View())
    } else {
        refreshErrorString := ""
        if m.refreshError {
            refreshErrorString = "⚠ "
        }

        refreshingString = fmt.Sprintf("%sLast refresh at %s", refreshErrorString, m.lastRefresh.Format("15:04:05"))
    }

    statusRefreshIndicatorRender := ""
    switch {
        case m.refreshError == true:
            statusRefreshIndicatorRender = statusRefreshIndicatorErrorStyle.Render(refreshingString)
        case m.clusterData.clusterInfo.Status == "green":
            statusRefreshIndicatorRender = statusRefreshIndicatorGreenStyle.Render(refreshingString)
        case m.clusterData.clusterInfo.Status == "yellow":
            statusRefreshIndicatorRender = statusRefreshIndicatorYellowStyle.Render(refreshingString)
        case m.clusterData.clusterInfo.Status == "red":
            statusRefreshIndicatorRender = statusRefreshIndicatorRedStyle.Render(refreshingString)
    }

    refreshInfoString := refreshInfoStatus(m.refreshIntervalSeconds)                                 
    statusRefreshInfoRender := ""
    switch {
        case m.refreshError == true:
            statusRefreshInfoRender  = statusRefreshInfoErrorStyle.Render(refreshInfoString)
        case m.clusterData.clusterInfo.Status == "green":
            statusRefreshInfoRender  = statusRefreshInfoGreenStyle.Render(refreshInfoString)
        case m.clusterData.clusterInfo.Status == "yellow":
            statusRefreshInfoRender = statusRefreshInfoYellowStyle.Render(refreshInfoString)
        case m.clusterData.clusterInfo.Status == "red":
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
                logoStyle.Render(logo))),
        contentStyle.Render(contentString),
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
        refreshInfoString += fmt.Sprintf("%s ", intervalString)
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
    return tea.Tick(time.Duration(intervalSeconds) * time.Second, func(t time.Time) tea.Msg {
		return autorefreshTickMsg(t)
	})
}

func refreshData() tea.Cmd {
    return func() tea.Msg {
        clusterData := ClusterData{}

        errorGroup := errgroup.Group{}

        errorGroup.Go(func() error {
            clusterInfo, err := fetchClusterInfo()
            if err != nil {
                return err
            }
            clusterData.clusterInfo = *clusterInfo
            return nil
        })

        errorGroup.Go(func() error {
            clusterStats, err := fetchClusterStats()
            if err != nil {
                return err
            }
            clusterData.clusterStats = *clusterStats
            return nil
        })

        if err := errorGroup.Wait(); err != nil {
            return refreshErrorMsg(err)
        } else {
            return clusterDataMsg(clusterData)
        }

    }
}

func fetchClusterInfo() (*ClusterInfo, error) {
    httpClient := http.Client{Timeout: 60  * time.Second}

    resp, err := httpClient.Get(baseUrl + clusterHealthPath)
    if err != nil {
	    return nil, err
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body) 
    if err != nil {
	    return nil, err
    }

    var clusterInfo ClusterInfo
    err = json.Unmarshal(body, &clusterInfo)
    if err != nil {
	    return nil, err
    }

    return &clusterInfo, nil
}

func fetchClusterStats() (*ClusterStats, error) {
    httpClient := http.Client{Timeout: 60  * time.Second}

    resp, err := httpClient.Get(baseUrl + clusterStatsPath)
    if err != nil {
	    return nil, err
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body) 
    if err != nil {
	    return nil, err
    }

    var clusterStats ClusterStats
    err = json.Unmarshal(body, &clusterStats)
    if err != nil {
	    return nil, err
    }

    return &clusterStats, nil
}

func changeAutorefreshInterval(currentInterval uint) tea.Cmd {
    return func() tea.Msg {
        switch {
            case currentInterval == 0: 
                return autorefreshIntervalChangeMsg(1)
            case currentInterval == 1: 
                return autorefreshIntervalChangeMsg(5)
            case currentInterval == 5: 
                return autorefreshIntervalChangeMsg(10)
            case currentInterval == 10: 
                return autorefreshIntervalChangeMsg(30)
            case currentInterval == 30: 
                return autorefreshIntervalChangeMsg(60)
            case currentInterval == 60: 
                return autorefreshIntervalChangeMsg(300)
            case currentInterval == 300: 
                return autorefreshIntervalChangeMsg(600)
            case currentInterval == 600: 
                return autorefreshIntervalChangeMsg(0)
            default:
                return autorefreshIntervalChangeMsg(5)
        }
    }
}

func main() {
    p := tea.NewProgram(newModel(), tea.WithAltScreen())

    if _, err := p.Run(); err != nil {
	    log.Fatal(err)
    }
}
