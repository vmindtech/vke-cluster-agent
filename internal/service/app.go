package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/vmindtech/vke-cluster-agent/config"
	"github.com/vmindtech/vke-cluster-agent/internal/model"
	"github.com/vmindtech/vke-cluster-agent/pkg/constants"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

type IAppService interface {
	GetOpenstackSession(pjID, applicationCredentialID, applicationCredentialSecret, identityURL string) (*gophercloud.ProviderClient, error)
	CheckVKEClusterCertificateExpiration(isExpired chan bool)
	RenewMasterNodesCertificates() error
	RestartWorkerNodes() error
}

type appService struct {
	iOpenstackService  IOpenstackService
	iVKEClusterService IVKEService
	k8sClient          *kubernetes.Clientset
	k8sConfig          *rest.Config
}

func NewAppService(iOpenstackService IOpenstackService, iVKEClusterService IVKEService, k8sClient *kubernetes.Clientset, k8sConfig *rest.Config) IAppService {
	return &appService{
		iOpenstackService:  iOpenstackService,
		iVKEClusterService: iVKEClusterService,
		k8sClient:          k8sClient,
		k8sConfig:          k8sConfig,
	}
}

func (a *appService) GetOpenstackSession(pjID, applicationCredentialID, applicationCredentialSecret, identityURL string) (*gophercloud.ProviderClient, error) {
	return a.iOpenstackService.ValidateAndCreateSession(pjID, applicationCredentialID, applicationCredentialSecret, identityURL)
}

func (a *appService) CheckVKEClusterCertificateExpiration(isExpired chan bool) {
	clID := config.GlobalConfig.GetVKEConfig().ClusterID
	vkeURL := config.GlobalConfig.GetVKEConfig().VKEURL

	var getCurrentTime func() time.Time
	if config.GlobalConfig.GetIsTestMode() {
		getCurrentTime = func() time.Time {
			return time.Now().AddDate(0, 0, 359)
		}
	} else {
		getCurrentTime = time.Now
	}

	for {
		token := a.getLatestToken()
		if token == "" {
			klog.ErrorS(nil, "Failed to get token",
				"cluster_id", clID,
				"component", "certificate_checker")
			return
		}

		klog.V(2).InfoS("Token refreshed successfully",
			"cluster_id", clID,
			"component", "certificate_checker")

		getClusterResponse, err := a.iVKEClusterService.GetCluster(clID, token, vkeURL)
		if err != nil {
			klog.ErrorS(err, "Failed to get cluster info",
				"cluster_id", clID,
				"vke_url", vkeURL,
				"component", "certificate_checker")
			return
		}

		klog.V(2).InfoS("Retrieved cluster information",
			"cluster_id", clID,
			"component", "certificate_checker")

		if IsExpired(getCurrentTime(), getClusterResponse.Data.ClusterCertificateExpireDate, constants.OneWeekMaintenanceWindow) {
			klog.V(0).InfoS("Certificate expiration detected",
				"cluster_id", clID,
				"expire_date", getClusterResponse.Data.ClusterCertificateExpireDate,
				"component", "certificate_checker")
			isExpired <- true
		}

		time.Sleep(constants.VKECheckCertificateExpirationInterval)
	}
}

func (a *appService) RenewMasterNodesCertificates() error {
	clID := config.GlobalConfig.GetVKEConfig().ClusterID
	cluster, err := a.iVKEClusterService.GetCluster(clID, a.getLatestToken(), config.GlobalConfig.GetVKEConfig().VKEURL)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %v", err)
	}

	if cluster.Data.ClusterStatus != constants.ClusterStatusActive {
		return fmt.Errorf("cluster is not active")
	}

	currentNode, err := getCurrentNode(a.k8sClient)
	if err != nil {
		return fmt.Errorf("failed to get current node: %v", err)
	}

	if !isMasterNode(currentNode) {
		return nil
	}

	firstMaster, err := getFirstMasterNode(a.k8sClient)
	if err != nil {
		return fmt.Errorf("failed to determine first master node: %v", err)
	}

	isFirstMaster := currentNode.Name == firstMaster.Name
	isOtherMaster := !isFirstMaster && isMasterNode(currentNode)

	if isFirstMaster {
		klog.V(0).InfoS("Processing first master node",
			"node", currentNode.Name)

		if err := restartService("rke2-server"); err != nil {
			return err
		}

		kubeconfigData, err := os.ReadFile("/etc/rancher/rke2/rke2.yaml")
		if err != nil {
			return fmt.Errorf("failed to read kubeconfig: %v", err)
		}

		var kubeconfigModel model.KubeConfig
		if err = yaml.Unmarshal(kubeconfigData, &kubeconfigModel); err != nil {
			return fmt.Errorf("failed to unmarshal kubeconfig: %v", err)
		}

		kubeconfigModel.Clusters[0].Cluster.Server = fmt.Sprintf("https://%s:6443", cluster.Data.ClusterEndpoint)
		kubeconfigModel.Clusters[0].Name = cluster.Data.ClusterName
		kubeconfigModel.Contexts[0].Context.Cluster = cluster.Data.ClusterName
		kubeconfigModel.Contexts[0].Context.User = cluster.Data.ClusterName
		kubeconfigModel.Contexts[0].Name = cluster.Data.ClusterName
		kubeconfigModel.CurrentContext = cluster.Data.ClusterName
		kubeconfigModel.Users[0].Name = cluster.Data.ClusterName

		updatedKubeconfigData, err := yaml.Marshal(kubeconfigModel)
		if err != nil {
			return fmt.Errorf("failed to marshal kubeconfig: %v", err)
		}

		kubeconfigBase64 := base64.StdEncoding.EncodeToString(updatedKubeconfigData)
		if err := a.iVKEClusterService.UpdateKubeconfig(
			clID,
			a.getLatestToken(),
			config.GlobalConfig.GetVKEConfig().VKEURL,
			kubeconfigBase64,
		); err != nil {
			return fmt.Errorf("failed to update kubeconfig: %v", err)
		}

		return nil
	}

	if isOtherMaster {
		klog.V(2).InfoS("Processing other master node, waiting before restart",
			"node", currentNode.Name)
		time.Sleep(2 * time.Minute)
		return restartService("rke2-server")
	}

	return nil
}

func isMasterNode(node *v1.Node) bool {
	_, isMaster := node.Labels["node-role.kubernetes.io/master"]
	_, isControlPlane := node.Labels["node-role.kubernetes.io/control-plane"]
	return isMaster || isControlPlane
}

func getFirstMasterNode(client *kubernetes.Clientset) (*v1.Node, error) {
	nodes, err := client.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{
		LabelSelector: "node-role.kubernetes.io/master=,node-role.kubernetes.io/control-plane=",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list master nodes: %v", err)
	}

	if len(nodes.Items) == 0 {
		return nil, fmt.Errorf("no master nodes found")
	}

	firstMaster := nodes.Items[0]
	for _, node := range nodes.Items {
		if node.CreationTimestamp.Before(&firstMaster.CreationTimestamp) {
			firstMaster = node
		}
	}
	return &firstMaster, nil
}

func getCurrentNode(client *kubernetes.Clientset) (*v1.Node, error) {
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		return nil, fmt.Errorf("NODE_NAME environment variable is not set")
	}
	return client.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
}

func restartService(serviceName string) error {
	cmd := exec.Command("systemctl", "restart", serviceName)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to restart %s: %v, stderr: %s", serviceName, err, stderr.String())
	}
	return nil
}

func (a *appService) RestartWorkerNodes() error {
	clID := config.GlobalConfig.GetVKEConfig().ClusterID

	klog.V(2).InfoS("Starting worker nodes restart process",
		"cluster_id", clID,
		"component", "worker_restarter")

	currentNode, err := getCurrentNode(a.k8sClient)
	if err != nil {
		return fmt.Errorf("failed to get current node: %v", err)
	}

	if isMasterNode(currentNode) {
		klog.V(2).InfoS("Skipping restart on master node",
			"cluster_id", clID,
			"node", currentNode.Name,
			"component", "worker_restarter")
		return nil
	}

	klog.V(0).InfoS("Restarting RKE2 agent on worker node",
		"cluster_id", clID,
		"node", currentNode.Name,
		"node_uid", currentNode.UID,
		"component", "worker_restarter")

	if err := restartService("rke2-agent"); err != nil {
		klog.ErrorS(err, "Failed to restart RKE2 agent",
			"cluster_id", clID,
			"node", currentNode.Name,
			"node_uid", currentNode.UID,
			"component", "worker_restarter")
		return fmt.Errorf("failed to restart RKE2 agent on node %s: %v", currentNode.Name, err)
	}

	return nil
}

func (a *appService) getLatestToken() string {
	pjID := config.GlobalConfig.GetVKEConfig().ProjectID
	applicationCredentialID := config.GlobalConfig.GetVKEConfig().ApplicationCredentialID
	applicationCredentialSecret := config.GlobalConfig.GetVKEConfig().ApplicationCredentialSecret
	identityURL := config.GlobalConfig.GetVKEConfig().IdentityURL

	providerClient, err := a.GetOpenstackSession(pjID, applicationCredentialID, applicationCredentialSecret, identityURL)
	if err != nil {
		klog.Error(err, "Failed to get openstack session for token refresh")
		return ""
	}

	return providerClient.Token()
}
