package service

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/k0kubun/pp"
	"github.com/sirupsen/logrus"
	"github.com/vmindtech/vke-cluster-agent/internal/dto/resource"
)

const (
	getClusterEndpoint = "/cluster"
)

type IVKEService interface {
	GetCluster(clusterID string, token string, vkeURL string) (*resource.VKEClusterResponse, error)
}

type vkeService struct {
	logger *logrus.Logger
}

func NewVKEService(logger *logrus.Logger) IVKEService {
	return &vkeService{
		logger: logger,
	}
}

func (v *vkeService) GetCluster(clusterID string, token string, vkeURL string) (*resource.VKEClusterResponse, error) {
	url := fmt.Sprintf("%s/%s/%s?details=true", getClusterEndpoint, vkeURL, clusterID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		v.logger.WithFields(logrus.Fields{
			"cluster_id": clusterID,
			"url":        url,
			"error":      err,
		}).Error("Failed to create request")
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("X-Auth-Token", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		v.logger.WithFields(logrus.Fields{
			"cluster_id": clusterID,
			"url":        url,
			"error":      err,
		}).Error("Failed to send request")
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		v.logger.WithFields(logrus.Fields{
			"cluster_id":  clusterID,
			"status_code": resp.StatusCode,
			"url":         url,
		}).Error("Unexpected status code received")
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		v.logger.WithFields(logrus.Fields{
			"cluster_id": clusterID,
			"error":      err,
		}).Error("Failed to read response body")
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	var cluster resource.VKEClusterResponse
	if err := json.Unmarshal(body, &cluster); err != nil {
		v.logger.WithFields(logrus.Fields{
			"cluster_id": clusterID,
			"body":       string(body),
			"error":      err,
		}).Error("Failed to parse JSON response")
		return nil, fmt.Errorf("error parsing JSON: %v", err)
	}

	v.logger.WithFields(logrus.Fields{
		"cluster_id":     clusterID,
		"cluster_name":   cluster.ClusterName,
		"cluster_status": cluster.ClusterStatus,
	}).Info("Successfully retrieved cluster information")

	pp.Println(cluster)

	return &cluster, nil
}
