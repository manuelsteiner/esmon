package elasticsearch

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	clusterHealthPath = "/_cluster/health"
	clusterStatsPath  = "/_cluster/stats"
)

type ClusterData struct {
	ClusterInfo  ClusterInfo
	ClusterStats ClusterStats
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

func FetchData(endpoint string) (*ClusterData, error) {
	clusterData := ClusterData{}

	errorGroup := errgroup.Group{}

	errorGroup.Go(func() error {
		clusterInfo, err := fetchClusterInfo(endpoint)
		if err != nil {
			return err
		}
		clusterData.ClusterInfo = *clusterInfo
		return nil
	})

	errorGroup.Go(func() error {
		clusterStats, err := fetchClusterStats(endpoint)
		if err != nil {
			return err
		}
		clusterData.ClusterStats = *clusterStats
		return nil
	})

	if err := errorGroup.Wait(); err != nil {
		return nil, err
	} else {
		return &clusterData, nil
	}
}

func fetchClusterInfo(endpoint string) (*ClusterInfo, error) {
	httpClient := http.Client{Timeout: 60 * time.Second}

	resp, err := httpClient.Get(endpoint + clusterHealthPath)
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

func fetchClusterStats(endpoint string) (*ClusterStats, error) {
	httpClient := http.Client{Timeout: 60 * time.Second}

	resp, err := httpClient.Get(endpoint + clusterStatsPath)
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
