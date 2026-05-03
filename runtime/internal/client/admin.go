package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type AdminAPI struct {
	baseURL string
	client  *http.Client
	nodeID  string
}

type NodeStatus struct {
	NodeID    string    `json:"node_id"`
	Hostname  string    `json:"hostname"`
	Status    string    `json:"status"`
	Version   string    `json:"version"`
	Plugins   []string  `json:"plugins"`
	LastSeen  time.Time `json:"last_seen"`
	NodeGroup string    `json:"node_group"`
}

func NewAdminAPI(baseURL string) *AdminAPI {
	return &AdminAPI{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		nodeID: "",
	}
}

func (api *AdminAPI) RegisterNode(node *NodeStatus) error {
	if api.baseURL == "" {
		api.baseURL = "http://localhost:8080"
	}

	url := fmt.Sprintf("%s/api/nodes", api.baseURL)

	data, err := json.Marshal(node)
	if err != nil {
		return err
	}

	resp, err := api.client.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registration failed: %d", resp.StatusCode)
	}

	var result struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	api.nodeID = fmt.Sprintf("%d", result.ID)
	return nil
}

func (api *AdminAPI) Heartbeat(status string) error {
	if api.nodeID == "" {
		return fmt.Errorf("node not registered")
	}

	url := fmt.Sprintf("%s/api/nodes/%s/connect", api.baseURL, api.nodeID)

	resp, err := api.client.Post(url, "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (api *AdminAPI) GetPlugins() ([]string, error) {
	url := fmt.Sprintf("%s/api/plugins", api.baseURL)

	resp, err := api.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var plugins []struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&plugins); err != nil {
		return nil, err
	}

	names := make([]string, len(plugins))
	for i, p := range plugins {
		names[i] = p.Name
	}

	return names, nil
}

func (api *AdminAPI) DownloadPlugin(pluginID int64) ([]byte, error) {
	url := fmt.Sprintf("%s/api/plugins/%d/download", api.baseURL, pluginID)

	resp, err := api.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
