package elasticsearch

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"esmon/config"
	"io"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	clusterHealthPath = "/_cluster/health"
	clusterStatsPath  = "/_cluster/stats"
)

type Credentials struct {
	Username string
	Password string
}

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

    req, err := http.NewRequest("GET", endpoint + clusterHealthPath, nil)
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

    req, err := http.NewRequest("GET", endpoint + clusterStatsPath, nil)
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
	err = json.Unmarshal(body, &clusterStats)
	if err != nil {
		return nil, err
	}

	return &clusterStats, nil
}
