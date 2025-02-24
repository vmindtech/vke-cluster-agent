package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/vmindtech/vke-cluster-agent/internal/dto/resource"
	"k8s.io/klog"
)

const (
	getClusterEndpoint = "cluster"
)

type IVKEService interface {
	GetCluster(clusterID string, token string, vkeURL string) (*resource.VKEClusterResponse, error)
	UpdateKubeconfig(clusterID string, token string, vkeURL string, kubeconfig string) error
}

type vkeService struct{}

func NewVKEService() IVKEService {
	return &vkeService{}
}

func (v *vkeService) GetCluster(clusterID string, token string, vkeURL string) (*resource.VKEClusterResponse, error) {
	url := fmt.Sprintf("%s/%s/%s?details=true", vkeURL, getClusterEndpoint, clusterID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		klog.Errorf("Failed to create request - cluster_id: %s, url: %s, error: %v",
			clusterID, url, err)
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("X-Auth-Token", token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		klog.Errorf("Failed to send request - cluster_id: %s, url: %s, error: %v",
			clusterID, url, err)
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		klog.Errorf("Unexpected status code received - cluster_id: %s, url: %s, status_code: %d",
			clusterID, url, resp.StatusCode)
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		klog.Errorf("Failed to read response body - cluster_id: %s, url: %s, error: %v",
			clusterID, url, err)
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	var cluster resource.VKEClusterResponse
	if err := json.Unmarshal(body, &cluster); err != nil {
		klog.Errorf("Failed to parse JSON response - cluster_id: %s, body: %s, error: %v",
			clusterID, string(body), err)
		return nil, fmt.Errorf("error parsing JSON: %v", err)
	}

	klog.V(2).Infof("Successfully retrieved cluster information - cluster_id: %s, cluster_name: %s, status: %s",
		clusterID, cluster.ClusterName, cluster.ClusterStatus)

	return &cluster, nil
}

func (v *vkeService) UpdateKubeconfig(clusterID string, token string, vkeURL string, kubeconfig string) error {
	url := fmt.Sprintf("%s/kubeconfig/%s", vkeURL, clusterID)

	payload := struct {
		Kubeconfig string `json:"kubeconfig"`
	}{
		Kubeconfig: kubeconfig,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling kubeconfig: %v", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("X-Auth-Token", token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
