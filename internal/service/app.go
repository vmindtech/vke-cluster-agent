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
	"github.com/vmindtech/vke-cluster-agent/pkg/constants"
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

	klog.V(2).InfoS("Starting master certificate renewal process",
		"cluster_id", clID,
		"component", "certificate_renewer")

	nodes, err := a.k8sClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{
		LabelSelector: constants.MasterNodeLabelSelector,
	})
	if err != nil {
		klog.ErrorS(err, "Failed to list master nodes",
			"cluster_id", clID,
			"component", "certificate_renewer")
		return fmt.Errorf("failed to list master nodes: %v", err)
	}

	if len(nodes.Items) > 0 {
		firstMaster := nodes.Items[0]
		klog.V(0).InfoS("Starting certificate renewal process on first master node",
			"cluster_id", clID,
			"node", firstMaster.Name,
			"component", "certificate_renewer")

		cmd := exec.Command("systemctl", "restart", "rke2-server")
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			klog.ErrorS(err, "Failed to restart RKE2 server",
				"cluster_id", clID,
				"node", firstMaster.Name,
				"stderr", stderr.String(),
				"component", "certificate_renewer")
			return fmt.Errorf("failed to restart RKE2 server: %v, stderr: %s", err, stderr.String())
		}

		time.Sleep(constants.RKE2RestartWaitDuration)

		kubeconfigData, err := os.ReadFile("/etc/rancher/rke2/rke2.yaml")
		if err != nil {
			return fmt.Errorf("failed to read kubeconfig: %v", err)
		}

		kubeconfigBase64 := base64.StdEncoding.EncodeToString(kubeconfigData)

		vkeURL := config.GlobalConfig.GetVKEConfig().VKEURL
		token := a.getLatestToken()

		err = a.iVKEClusterService.UpdateKubeconfig(clID, token, vkeURL, kubeconfigBase64)
		if err != nil {
			return fmt.Errorf("failed to update kubeconfig in VKE API: %v", err)
		}

		klog.Info("Successfully updated kubeconfig in VKE API")

		for i := 1; i < len(nodes.Items); i++ {
			node := nodes.Items[i]
			klog.InfoS("Starting certificate renewal process on subsequent master node", "node", node.Name)

			cmd := exec.Command("systemctl", "restart", "rke2-server")
			var stderr bytes.Buffer
			cmd.Stderr = &stderr

			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to restart RKE2 server on node %s: %v, stderr: %s",
					node.Name, err, stderr.String())
			}

			time.Sleep(constants.RKE2RestartWaitDuration)
		}
	}

	return nil
}

func (a *appService) RestartWorkerNodes() error {
	clID := config.GlobalConfig.GetVKEConfig().ClusterID

	klog.V(2).InfoS("Starting worker nodes restart process",
		"cluster_id", clID,
		"component", "worker_restarter")

	nodes, err := a.k8sClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{
		LabelSelector: constants.WorkerNodeLabelSelector,
	})
	if err != nil {
		klog.ErrorS(err, "Failed to list worker nodes",
			"cluster_id", clID,
			"component", "worker_restarter")
		return fmt.Errorf("failed to list worker nodes: %v", err)
	}

	for _, node := range nodes.Items {
		klog.V(0).InfoS("Restarting RKE2 agent on worker node",
			"cluster_id", clID,
			"node", node.Name,
			"node_uid", node.UID,
			"component", "worker_restarter")

		cmd := exec.Command("systemctl", "restart", "rke2-agent")
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			klog.ErrorS(err, "Failed to restart RKE2 agent",
				"cluster_id", clID,
				"node", node.Name,
				"node_uid", node.UID,
				"stderr", stderr.String(),
				"component", "worker_restarter")
			return fmt.Errorf("failed to restart RKE2 agent on node %s: %v, stderr: %s",
				node.Name, err, stderr.String())
		}

		time.Sleep(constants.RKE2RestartWaitDuration)
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
