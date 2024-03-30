package elasticsearch

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"esmon/config"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	clusterHealthPath = "/_cluster/health"
	clusterStatsPath  = "/_cluster/stats"
	nodeStatsPath     = "/_nodes/stats"
)

type Credentials struct {
	Username string
	Password string
}

type ClusterData struct {
	ClusterInfo  ClusterInfo
	ClusterStats ClusterStats
	NodeStats    []NodeStats
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

type NodeStats struct {
	Timestamp        int64    `json:"timestamp"`
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
	} `json:"os"`
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

func FetchData(endpoint string, credentials *Credentials, timeoutSeconds uint, insecure bool) (*ClusterData, error) {
	clusterData := ClusterData{}

	errorGroup := errgroup.Group{}

	errorGroup.Go(func() error {
		clusterInfo, err := fetchClusterInfo(endpoint, credentials, timeoutSeconds, insecure)
		if err != nil {
			return err
		}
		clusterData.ClusterInfo = *clusterInfo
		return nil
	})

	errorGroup.Go(func() error {
		clusterStats, err := fetchClusterStats(endpoint, credentials, timeoutSeconds, insecure)
		if err != nil {
			return err
		}
		clusterData.ClusterStats = *clusterStats
		return nil
	})

	errorGroup.Go(func() error {
		nodeStats, err := fetchNodeStats(endpoint, credentials, timeoutSeconds, insecure)
		if err != nil {
			return err
		}
		clusterData.NodeStats = *nodeStats
		return nil
	})

	if err := errorGroup.Wait(); err != nil {
		return nil, err
	} else {
		return &clusterData, nil
	}
}

func httpClient(timeoutSeconds uint, insecure bool) http.Client {
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: insecure}
	httpClient := http.Client{Timeout: time.Duration(timeoutSeconds) * time.Second, Transport: customTransport}

	return httpClient
}

func fetchClusterInfo(endpoint string, credentials *Credentials, timeoutSeconds uint, insecure bool) (*ClusterInfo, error) {
	httpClient := httpClient(timeoutSeconds, insecure)

	req, err := http.NewRequest("GET", endpoint+clusterHealthPath, nil)
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

func fetchClusterStats(endpoint string, credentials *Credentials, timeoutSeconds uint, insecure bool) (*ClusterStats, error) {
	httpClient := httpClient(timeoutSeconds, insecure)

	req, err := http.NewRequest("GET", endpoint+clusterStatsPath, nil)
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

func fetchNodeStats(endpoint string, credentials *Credentials, timeoutSeconds uint, insecure bool) (*[]NodeStats, error) {
	httpClient := httpClient(timeoutSeconds, insecure)

	req, err := http.NewRequest("GET", endpoint+nodeStatsPath, nil)
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
	for _, nodeInfo := range nodeInfos {
		var nodeStats NodeStats
		if err = json.Unmarshal(nodeInfo, &nodeStats); err != nil {
			return nil, err
		}
		nodeStatsArray = append(nodeStatsArray, nodeStats)
	}

	return &nodeStatsArray, nil
}
