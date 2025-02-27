package service

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	r, err := http.NewRequest("GET", fmt.Sprintf("%s/%s/%s?details=true", vkeURL, getClusterEndpoint, clusterID), nil)
	if err != nil {
		klog.Errorf("Failed to create request - cluster_id: %s", clusterID)
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Auth-Token", token)

	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		klog.Errorf("Failed to send request - cluster_id: %s", clusterID)
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		klog.Errorf("Unexpected status code received - cluster_id: %s, status_code: %d",
			clusterID, resp.StatusCode)
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var respDecoder resource.VKEClusterResponse
	if err = json.NewDecoder(resp.Body).Decode(&respDecoder); err != nil {
		klog.Errorf("Failed to decode response - cluster_id: %s", clusterID)
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	klog.V(2).Infof("Successfully retrieved cluster information - cluster_id: %s, cluster_name: %s, status: %s",
		clusterID, respDecoder.ClusterName, respDecoder.ClusterStatus)

	return &respDecoder, nil
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
