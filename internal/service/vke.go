package service

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/vmindtech/vke-cluster-agent/internal/dto/request"
	"github.com/vmindtech/vke-cluster-agent/internal/dto/resource"
	"k8s.io/klog"
)

const (
	getClusterEndpoint = "cluster"
)

type IVKEService interface {
	GetCluster(clusterID string, token string, vkeURL string) (*resource.VKEClusterResponse, error)
	UpdateKubeconfig(clusterID string, token string, vkeURL string, kubeconfig string) error
	UpdateCluster(clusterID string, token string, vkeURL string, cluster request.UpdateClusterRequest) error
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

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client := &http.Client{Transport: tr}
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
		clusterID, respDecoder.Data.ClusterName, respDecoder.Data.ClusterStatus)

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

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client := &http.Client{Transport: tr}
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

func (v *vkeService) UpdateCluster(clusterID string, token string, vkeURL string, cluster request.UpdateClusterRequest) error {
	url := fmt.Sprintf("%s/cluster/%s", vkeURL, clusterID)

	jsonData, err := json.Marshal(cluster)
	if err != nil {
		return fmt.Errorf("error marshaling cluster: %v", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("X-Auth-Token", token)
	req.Header.Set("Content-Type", "application/json")

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client := &http.Client{Transport: tr}
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
