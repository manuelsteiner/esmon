package elasticsearch

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"esmon/config"
	"fmt"
	"io"
	"net/http"
	"slices"
	"sort"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	clusterHealthPath = "/_cluster/health?human"
	clusterStatsPath  = "/_cluster/stats?human"
	recoveryPath      = "/_recovery?active_only&human"
	nodeStatsPath     = "/_nodes/stats/indices,os,fs?human"
	masterNodePath    = "/_nodes/_master/stats/indices,os,fs?human"
)

type Credentials struct {
	Username string
	Password string
}

type ClusterData struct {
	ClusterInfo  ClusterInfo
	ClusterStats ClusterStats
    Recoveries     []Recovery
	NodeStats    []NodeStats
	MasterNode   *NodeStats
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
	ActiveShardsPercent         string  `json:"active_shards_percent"`
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

type Recovery struct {
	ID                int    `json:"id"`
	Type              string `json:"type"`
	Stage             string `json:"stage"`
	Primary           bool   `json:"primary"`
	StartTime         string `json:"start_time"`
	StartTimeInMillis int64  `json:"start_time_in_millis"`
	StopTime          string `json:"stop_time"`
	StopTimeInMillis  int    `json:"stop_time_in_millis"`
	TotalTime         string `json:"total_time"`
	TotalTimeInMillis int    `json:"total_time_in_millis"`
    Source            RecoveryPeer `json:"source"`
    Target            RecoveryPeer `json:"target"`
    Shard             string // manually added while fetching
	Index struct {
        Name string // manually added while fetching
		Size struct {
			Total                        string `json:"total"`
			TotalInBytes                 int    `json:"total_in_bytes"`
			Reused                       string `json:"reused"`
			ReusedInBytes                int    `json:"reused_in_bytes"`
			Recovered                    string `json:"recovered"`
			RecoveredInBytes             int    `json:"recovered_in_bytes"`
			RecoveredFromSnapshot        string `json:"recovered_from_snapshot"`
			RecoveredFromSnapshotInBytes int    `json:"recovered_from_snapshot_in_bytes"`
			Percent                      string `json:"percent"`
		} `json:"size"`
		Files struct {
			Total     int    `json:"total"`
			Reused    int    `json:"reused"`
			Recovered int    `json:"recovered"`
			Percent   string `json:"percent"`
		} `json:"files"`
		TotalTime                  string `json:"total_time"`
		TotalTimeInMillis          int    `json:"total_time_in_millis"`
		SourceThrottleTime         string `json:"source_throttle_time"`
		SourceThrottleTimeInMillis int    `json:"source_throttle_time_in_millis"`
		TargetThrottleTime         string `json:"target_throttle_time"`
		TargetThrottleTimeInMillis int    `json:"target_throttle_time_in_millis"`
	} `json:"index"`
}

type RecoveryPeerInterface interface {
    Type() string
    PeerName() string
}

type RecoveryPeer struct {
    Peer RecoveryPeerInterface
}

func (p *RecoveryPeer) UnmarshalJSON(data []byte) error {

    var nodePeer NodePeer 
    if err := json.Unmarshal(data, &nodePeer); err == nil {
        p.Peer = nodePeer
        return nil
    } 

    var repositoryPeer NodePeer 
    if err := json.Unmarshal(data, &nodePeer); err == nil {
        p.Peer = repositoryPeer
        return nil
    } 
 
    return errors.New("Can not unmarshl RecoveryPeer")
}

type NodePeer struct {
	ID               string `json:"id"`
	Host             string `json:"host"`
	TransportAddress string `json:"transport_address"`
	IP               string `json:"ip"`
	Name             string `json:"name"`
}

func (n NodePeer) Type() string {
    return "Node"
}

func (n NodePeer) PeerName() string {
    return n.Name
}

type RepositoryPeer struct {
	Repository  string `json:"repository"`
	Snapshot    string `json:"snapshot"`
}

func (n RepositoryPeer) Type() string {
    return "Repository"
}

func (n RepositoryPeer) PeerName() string {
    return n.Repository
}

type NodeStats struct {
	Timestamp        int64    `json:"timestamp"`
    Id               string    // manually added when fetching
	Name             string   `json:"name"`
	TransportAddress string   `json:"transport_address"`
	Host             string   `json:"host"`
	IP               string   `json:"ip"`
	Roles            []string `json:"roles"`
	Indices          struct {
		Docs struct {
			Count   int `json:"count"`
			Deleted int `json:"deleted"`
		} `json:"docs"`
		ShardStats struct {
			TotalCount int `json:"total_count"`
		} `json:"shard_stats"`
		Store struct {
			Size                    string `json:"size"`
			SizeInBytes             int    `json:"size_in_bytes"`
			TotalDataSetSize        string `json:"total_data_set_size"`
			TotalDataSetSizeInBytes int    `json:"total_data_set_size_in_bytes"`
			Reserved                string `json:"reserved"`
			ReservedInBytes         int    `json:"reserved_in_bytes"`
		} `json:"store"`
	} `json:"indices"`
	Os struct {
		Timestamp int64 `json:"timestamp"`
		CPU       struct {
			Percent     int `json:"percent"`
			LoadAverage struct {
				OneM  float64 `json:"1m"`
				FiveM float64 `json:"5m"`
				One5M float64 `json:"15m"`
			} `json:"load_average"`
		} `json:"cpu"`
		Mem struct {
			Total                string `json:"total"`
			TotalInBytes         int64  `json:"total_in_bytes"`
			AdjustedTotal        string `json:"adjusted_total"`
			AdjustedTotalInBytes int64  `json:"adjusted_total_in_bytes"`
			Free                 string `json:"free"`
			FreeInBytes          int64  `json:"free_in_bytes"`
			Used                 string `json:"used"`
			UsedInBytes          int64  `json:"used_in_bytes"`
			FreePercent          int    `json:"free_percent"`
			UsedPercent          int    `json:"used_percent"`
		} `json:"mem"`
		Swap struct {
			Total        string `json:"total"`
			TotalInBytes int    `json:"total_in_bytes"`
			Free         string `json:"free"`
			FreeInBytes  int    `json:"free_in_bytes"`
			Used         string `json:"used"`
			UsedInBytes  int    `json:"used_in_bytes"`
		} `json:"swap"`
	} `json:"os"`
	Fs struct {
		Timestamp int64 `json:"timestamp"`
		Total     struct {
			Total            string `json:"total"`
			TotalInBytes     int64  `json:"total_in_bytes"`
			Free             string `json:"free"`
			FreeInBytes      int64  `json:"free_in_bytes"`
			Available        string `json:"available"`
			AvailableInBytes int64  `json:"available_in_bytes"`
		} `json:"total"`
	} `json:"fs"`
}

func GetCredentials(clusterConfig *config.ClusterConfig, defaultCredentials *Credentials) (*Credentials, error) {
	credentials := Credentials{
		Username: defaultCredentials.Username,
		Password: defaultCredentials.Password,
	}

	if clusterConfig.Username != "" {
		credentials.Username = clusterConfig.Username
	}

	if clusterConfig.Password != "" {
		credentials.Password = clusterConfig.Password
	}

	if credentials.Username == "" || credentials.Password == "" {
		return nil, errors.New("Neither cluster nor default credentials were provided.")
	}

	return &credentials, nil
}

func FetchData(ctx context.Context, endpoint string, credentials *Credentials, timeoutSeconds uint, insecure bool) (*ClusterData, error) {
	clusterData := ClusterData{}

	errorGroup := errgroup.Group{}

	errorGroup.Go(func() error {
		clusterInfo, err := fetchClusterInfo(ctx, endpoint, credentials, timeoutSeconds, insecure)
		if err != nil {
			return err
		}
		clusterData.ClusterInfo = *clusterInfo
		return nil
	})

	errorGroup.Go(func() error {
		clusterStats, err := fetchClusterStats(ctx, endpoint, credentials, timeoutSeconds, insecure)
		if err != nil {
			return err
		}
		clusterData.ClusterStats = *clusterStats
		return nil
	})

	errorGroup.Go(func() error {
		recoveries, err := fetchRecoveries(ctx, endpoint, credentials, timeoutSeconds, insecure)
		if err != nil {
			return err
		}
		clusterData.Recoveries = *recoveries
		return nil
	})

	errorGroup.Go(func() error {
		nodeStats, err := fetchNodeStats(ctx, endpoint, credentials, timeoutSeconds, insecure)
		if err != nil {
			return err
		}
		clusterData.NodeStats = *nodeStats
		return nil
	})

	var masterNodeId string
	errorGroup.Go(func() error {
		masterNodeIdValue, err := fetchMasterNodeId(ctx, endpoint, credentials, timeoutSeconds, insecure)
		if err != nil {
			return err
		}
		masterNodeId = *masterNodeIdValue
		return nil
	})

	if err := errorGroup.Wait(); err != nil {
		return nil, err
	}

	sort.Slice(clusterData.Recoveries, func(i, j int) bool {
		return clusterData.Recoveries[i].TotalTimeInMillis > clusterData.Recoveries[j].TotalTimeInMillis
	})

	sort.Slice(clusterData.NodeStats, func(i, j int) bool {
		return clusterData.NodeStats[i].Name < clusterData.NodeStats[j].Name
	})

	index := slices.IndexFunc(
		clusterData.NodeStats,
		func(s NodeStats) bool {
			return s.Id == masterNodeId
		})

	if index == -1 {
		return nil, errors.New(fmt.Sprintf("Unable to find master node with ID %s in node list", masterNodeId))
	}

	clusterData.MasterNode = &clusterData.NodeStats[index]

	return &clusterData, nil
}

func httpClient(timeoutSeconds uint, insecure bool) http.Client {
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: insecure}
	httpClient := http.Client{Timeout: time.Duration(timeoutSeconds) * time.Second, Transport: customTransport}

	return httpClient
}

func fetchClusterInfo(ctx context.Context, endpoint string, credentials *Credentials, timeoutSeconds uint, insecure bool) (*ClusterInfo, error) {
	httpClient := httpClient(timeoutSeconds, insecure)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint+clusterHealthPath, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(credentials.Username, credentials.Password)

	resp, err := httpClient.Do(req)
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

func fetchClusterStats(ctx context.Context, endpoint string, credentials *Credentials, timeoutSeconds uint, insecure bool) (*ClusterStats, error) {
	httpClient := httpClient(timeoutSeconds, insecure)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint+clusterStatsPath, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(credentials.Username, credentials.Password)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var clusterStats ClusterStats
	if err = json.Unmarshal(body, &clusterStats); err != nil {
		return nil, err
	}

	return &clusterStats, nil
}

func fetchRecoveries(ctx context.Context, endpoint string, credentials *Credentials, timeoutSeconds uint, insecure bool) (*[]Recovery, error) {
	httpClient := httpClient(timeoutSeconds, insecure)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint+recoveryPath, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(credentials.Username, credentials.Password)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rawMap map[string]json.RawMessage
	if err = json.Unmarshal(body, &rawMap); err != nil {
		return nil, err
	}

    var recoveries []Recovery
    for index, indexMap := range rawMap {
	    var shardMap map[string]json.RawMessage
        if err = json.Unmarshal(indexMap, &shardMap); err != nil {
            return nil, err
        }

        var indexRecoveries []Recovery
        if err = json.Unmarshal(shardMap["shards"], &indexRecoveries); err != nil {
            return nil, err
        }

        for _, recovery := range indexRecoveries {
            recovery.Index.Name = index

            if recovery.Type == "PEER" {
                recoveries = append(recoveries, recovery)
            }
        }
    }

	return &recoveries, nil
}

func fetchNodeStats(ctx context.Context, endpoint string, credentials *Credentials, timeoutSeconds uint, insecure bool) (*[]NodeStats, error) {
	httpClient := httpClient(timeoutSeconds, insecure)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint+nodeStatsPath, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(credentials.Username, credentials.Password)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rawMap map[string]json.RawMessage
	if err = json.Unmarshal(body, &rawMap); err != nil {
		return nil, err
	}

	var nodeInfos map[string]json.RawMessage
	if err = json.Unmarshal(rawMap["nodes"], &nodeInfos); err != nil {
		return nil, err
	}

	var nodeStatsArray []NodeStats
	for id, nodeInfo := range nodeInfos {
		var nodeStats NodeStats
		nodeStats.Id = id
		if err = json.Unmarshal(nodeInfo, &nodeStats); err != nil {
			return nil, err
		}
		nodeStatsArray = append(nodeStatsArray, nodeStats)
	}

	return &nodeStatsArray, nil
}

func fetchMasterNodeId(ctx context.Context, endpoint string, credentials *Credentials, timeoutSeconds uint, insecure bool) (*string, error) {
	httpClient := httpClient(timeoutSeconds, insecure)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint+masterNodePath, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(credentials.Username, credentials.Password)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rawMap map[string]json.RawMessage
	if err = json.Unmarshal(body, &rawMap); err != nil {
		return nil, err
	}

	var nodeInfos map[string]json.RawMessage
	if err = json.Unmarshal(rawMap["nodes"], &nodeInfos); err != nil {
		return nil, err
	}

	var masterNodeId string
	for id := range nodeInfos {
		masterNodeId = id
	}

	if masterNodeId == "" {
		return nil, errors.New("Could not determine master node ID")
	}

	return &masterNodeId, nil
}
