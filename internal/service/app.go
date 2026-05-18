package service

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/vmindtech/vke-cluster-agent/config"
	"github.com/vmindtech/vke-cluster-agent/internal/dto/request"
	"github.com/vmindtech/vke-cluster-agent/internal/dto/resource"
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
	CheckVKEClusterCertificateExpiration(ctx context.Context, expireDates chan<- time.Time)
	RenewMasterNodesCertificates() error
	RestartWorkerNodes() error
}

type appService struct {
	iOpenstackService  IOpenstackService
	iVKEClusterService IVKEService
	k8sClient          *kubernetes.Clientset
	k8sConfig          *rest.Config
}

const rke2KubeconfigPath = "/etc/rancher/rke2/rke2.yaml"

type serviceState struct {
	ActiveState string
	ExecMainPID string
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

func (a *appService) CheckVKEClusterCertificateExpiration(ctx context.Context, expireDates chan<- time.Time) {
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
		select {
		case <-ctx.Done():
			return
		default:
		}

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

		expireDate := getClusterResponse.Data.ClusterCertificateExpireDate
		if IsExpired(getCurrentTime(), expireDate, constants.OneWeekMaintenanceWindow) {
			klog.V(0).InfoS("Certificate expiration detected",
				"cluster_id", clID,
				"expire_date", expireDate,
				"component", "certificate_checker")
			select {
			case expireDates <- expireDate:
			case <-ctx.Done():
				return
			default:
				klog.V(2).InfoS("Certificate expiration already queued",
					"cluster_id", clID,
					"expire_date", expireDate,
					"component", "certificate_checker")
			}
		}

		timer := time.NewTimer(constants.VKECheckCertificateExpirationInterval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return
		case <-timer.C:
		}
	}
}

func (a *appService) RenewMasterNodesCertificates() error {
	clID := config.GlobalConfig.GetVKEConfig().ClusterID
	token := a.getLatestToken()
	if token == "" {
		return fmt.Errorf("failed to get token")
	}

	cluster, err := a.iVKEClusterService.GetCluster(clID, token, config.GlobalConfig.GetVKEConfig().VKEURL)
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

	previousExpireDate := cluster.Data.ClusterCertificateExpireDate

	if currentNode.Name == firstMaster.Name {
		klog.V(0).InfoS("Processing first master node",
			"cluster_id", clID,
			"node", currentNode.Name,
			"expire_date", previousExpireDate)

		if err := restartService("rke2-server"); err != nil {
			return err
		}

		time.Sleep(constants.RKE2RestartWaitDuration)

		kubeconfigData, updatedExpireDate, err := waitForKubeconfigCertificateRotation(
			rke2KubeconfigPath,
			previousExpireDate,
			constants.KubeconfigVerificationTimeout,
		)
		if err != nil {
			return err
		}

		kubeconfigModel, err := parseKubeconfig(kubeconfigData)
		if err != nil {
			return err
		}

		if err := updateKubeconfigForCluster(&kubeconfigModel, cluster); err != nil {
			return err
		}

		updatedKubeconfigData, err := yaml.Marshal(kubeconfigModel)
		if err != nil {
			return fmt.Errorf("failed to marshal kubeconfig: %v", err)
		}

		kubeconfigBase64 := base64.StdEncoding.EncodeToString(updatedKubeconfigData)
		updateToken := a.getLatestToken()
		if updateToken == "" {
			return fmt.Errorf("failed to get token for kubeconfig update")
		}

		if err := a.iVKEClusterService.UpdateKubeconfig(
			clID,
			updateToken,
			config.GlobalConfig.GetVKEConfig().VKEURL,
			kubeconfigBase64,
		); err != nil {
			return fmt.Errorf("failed to update kubeconfig: %v", err)
		}

		clReq := request.UpdateClusterRequest{
			ClusterCertificateExpireDate: updatedExpireDate,
			ClusterName:                  cluster.Data.ClusterName,
			ClusterVersion:               cluster.Data.ClusterVersion,
			ClusterStatus:                cluster.Data.ClusterStatus,
			ClusterAPIAccess:             cluster.Data.ClusterAPIAccess,
		}

		updateToken = a.getLatestToken()
		if updateToken == "" {
			return fmt.Errorf("failed to get token for cluster update")
		}

		if err := a.iVKEClusterService.UpdateCluster(
			clID,
			updateToken,
			config.GlobalConfig.GetVKEConfig().VKEURL,
			clReq); err != nil {
			return fmt.Errorf("failed to update cluster: %v", err)
		}

		klog.V(0).InfoS("Validated renewed kubeconfig certificate on first master",
			"cluster_id", clID,
			"node", currentNode.Name,
			"expire_date", updatedExpireDate)
		return nil
	}

	klog.V(0).InfoS("Waiting for first master certificate renewal to complete",
		"cluster_id", clID,
		"node", currentNode.Name,
		"first_master", firstMaster.Name,
		"expire_date", previousExpireDate)

	updatedCluster, err := a.waitForClusterCertificateUpdate(previousExpireDate, constants.ClusterUpdateVerificationTimeout)
	if err != nil {
		return err
	}

	klog.V(0).InfoS("First master update detected, restarting current master",
		"cluster_id", clID,
		"node", currentNode.Name,
		"first_master", firstMaster.Name,
		"expire_date", updatedCluster.Data.ClusterCertificateExpireDate)

	if err := restartService("rke2-server"); err != nil {
		return err
	}

	time.Sleep(constants.RKE2RestartWaitDuration)

	_, updatedExpireDate, err := waitForKubeconfigCertificateRotation(
		rke2KubeconfigPath,
		previousExpireDate,
		constants.KubeconfigVerificationTimeout,
	)
	if err != nil {
		return err
	}

	klog.V(0).InfoS("Validated renewed kubeconfig certificate on master node",
		"cluster_id", clID,
		"node", currentNode.Name,
		"expire_date", updatedExpireDate)

	return nil
}

func isMasterNode(node *v1.Node) bool {
	labels := node.Labels
	_, isMaster := labels["node-role.kubernetes.io/master"]
	_, isControlPlane := labels["node-role.kubernetes.io/control-plane"]
	return isMaster || isControlPlane
}

func getFirstMasterNode(client *kubernetes.Clientset) (*v1.Node, error) {
	nodes, err := client.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{
		LabelSelector: "node-role.kubernetes.io/control-plane",
	})
	if err != nil || len(nodes.Items) == 0 {
		nodes, err = client.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{
			LabelSelector: "node-role.kubernetes.io/master",
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list master nodes: %v", err)
		}
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

func (a *appService) waitForClusterCertificateUpdate(previousExpireDate time.Time, timeout time.Duration) (*resource.VKEClusterResponse, error) {
	clID := config.GlobalConfig.GetVKEConfig().ClusterID
	vkeURL := config.GlobalConfig.GetVKEConfig().VKEURL
	deadline := time.Now().Add(timeout)

	var lastErr error
	for time.Now().Before(deadline) {
		token := a.getLatestToken()
		if token == "" {
			lastErr = fmt.Errorf("failed to get token while waiting for first master update")
			time.Sleep(constants.ServiceVerificationInterval)
			continue
		}

		cluster, err := a.iVKEClusterService.GetCluster(clID, token, vkeURL)
		if err == nil && cluster.Data.ClusterCertificateExpireDate.After(previousExpireDate) {
			return cluster, nil
		}
		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf(
				"cluster certificate expiry did not advance beyond %s",
				previousExpireDate.UTC().Format(time.RFC3339),
			)
		}

		time.Sleep(constants.ServiceVerificationInterval)
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("cluster certificate expiry did not advance beyond %s", previousExpireDate.UTC().Format(time.RFC3339))
	}

	return nil, fmt.Errorf("timed out waiting for first master certificate update: %w", lastErr)
}

func parseKubeconfig(kubeconfigData []byte) (model.KubeConfig, error) {
	var kubeconfigModel model.KubeConfig
	if err := yaml.Unmarshal(kubeconfigData, &kubeconfigModel); err != nil {
		return model.KubeConfig{}, fmt.Errorf("failed to unmarshal kubeconfig: %v", err)
	}

	return kubeconfigModel, nil
}

func updateKubeconfigForCluster(kubeconfigModel *model.KubeConfig, cluster *resource.VKEClusterResponse) error {
	if len(kubeconfigModel.Clusters) == 0 || len(kubeconfigModel.Contexts) == 0 || len(kubeconfigModel.Users) == 0 {
		return fmt.Errorf("kubeconfig is missing required clusters, contexts, or users")
	}

	kubeconfigModel.Clusters[0].Cluster.Server = fmt.Sprintf("https://%s:6443", cluster.Data.ClusterEndpoint)
	kubeconfigModel.Clusters[0].Name = cluster.Data.ClusterName
	kubeconfigModel.Contexts[0].Context.Cluster = cluster.Data.ClusterName
	kubeconfigModel.Contexts[0].Context.User = cluster.Data.ClusterName
	kubeconfigModel.Contexts[0].Name = cluster.Data.ClusterName
	kubeconfigModel.CurrentContext = cluster.Data.ClusterName
	kubeconfigModel.Users[0].Name = cluster.Data.ClusterName

	return nil
}

func getKubeconfigCertificateExpiration(kubeconfigModel model.KubeConfig) (time.Time, error) {
	if len(kubeconfigModel.Users) == 0 {
		return time.Time{}, fmt.Errorf("kubeconfig does not contain any users")
	}

	clientCertificateData := kubeconfigModel.Users[0].User.ClientCertificateData
	if clientCertificateData == "" {
		return time.Time{}, fmt.Errorf("kubeconfig does not contain client certificate data")
	}

	certificateBytes, err := base64.StdEncoding.DecodeString(clientCertificateData)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to decode kubeconfig client certificate: %v", err)
	}

	certificateBlock, _ := pem.Decode(certificateBytes)
	if certificateBlock == nil {
		return time.Time{}, fmt.Errorf("failed to decode PEM block from kubeconfig client certificate")
	}

	certificate, err := x509.ParseCertificate(certificateBlock.Bytes)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse kubeconfig client certificate: %v", err)
	}

	return certificate.NotAfter, nil
}

func readKubeconfigCertificateExpiration(path string) ([]byte, time.Time, error) {
	kubeconfigData, err := os.ReadFile(path)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("failed to read kubeconfig: %v", err)
	}

	kubeconfigModel, err := parseKubeconfig(kubeconfigData)
	if err != nil {
		return nil, time.Time{}, err
	}

	expireDate, err := getKubeconfigCertificateExpiration(kubeconfigModel)
	if err != nil {
		return nil, time.Time{}, err
	}

	return kubeconfigData, expireDate, nil
}

func waitForKubeconfigCertificateRotation(path string, previousExpireDate time.Time, timeout time.Duration) ([]byte, time.Time, error) {
	deadline := time.Now().Add(timeout)

	var lastErr error
	for time.Now().Before(deadline) {
		kubeconfigData, expireDate, err := readKubeconfigCertificateExpiration(path)
		if err == nil && expireDate.After(previousExpireDate) {
			return kubeconfigData, expireDate, nil
		}
		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf(
				"kubeconfig certificate expiry did not advance beyond %s",
				previousExpireDate.UTC().Format(time.RFC3339),
			)
		}

		time.Sleep(constants.ServiceVerificationInterval)
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("kubeconfig certificate expiry did not advance beyond %s", previousExpireDate.UTC().Format(time.RFC3339))
	}

	return nil, time.Time{}, fmt.Errorf("timed out waiting for kubeconfig certificate refresh: %w", lastErr)
}

// runHostSystemctl runs systemctl on the host. With hostPID, nsenter into init namespaces
// is preferred; chroot is used only when nsenter is unavailable.
func runHostSystemctl(args ...string) *exec.Cmd {
	if _, err := exec.LookPath("nsenter"); err == nil {
		cmdArgs := append([]string{
			"-t", constants.HostInitPID,
			"-m", "-p", "-i", "-n", "-u",
			"--",
			"systemctl",
		}, args...)
		return exec.Command("nsenter", cmdArgs...)
	}

	systemctlPath := constants.HostSystemctlPath
	cmdArgs := append([]string{constants.HostRootPath, systemctlPath}, args...)
	return exec.Command("chroot", cmdArgs...)
}

func runHostSystemctlOutput(args ...string) ([]byte, error) {
	cmd := runHostSystemctl(args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = strings.TrimSpace(string(output))
		}
		if errMsg != "" {
			return output, fmt.Errorf("%w: %s", err, errMsg)
		}
		return output, err
	}

	return output, nil
}

func getServiceState(serviceName string) (serviceState, error) {
	activeStateOutput, err := runHostSystemctlOutput("is-active", serviceName)
	activeState := strings.TrimSpace(string(activeStateOutput))
	if err != nil && activeState == "" {
		return serviceState{}, fmt.Errorf("failed to get active state for %s: %v", serviceName, err)
	}

	pidOutput, err := runHostSystemctlOutput("show", serviceName, "--property=ExecMainPID", "--value")
	if err != nil {
		return serviceState{}, fmt.Errorf("failed to get main pid for %s: %v", serviceName, err)
	}

	return serviceState{
		ActiveState: activeState,
		ExecMainPID: strings.TrimSpace(string(pidOutput)),
	}, nil
}

func waitForServiceRestart(serviceName string, previousState serviceState, timeout time.Duration) (serviceState, error) {
	deadline := time.Now().Add(timeout)

	var lastErr error
	for time.Now().Before(deadline) {
		currentState, err := getServiceState(serviceName)
		if err == nil && currentState.ActiveState == "active" && currentState.ExecMainPID != "" && currentState.ExecMainPID != "0" && currentState.ExecMainPID != previousState.ExecMainPID {
			return currentState, nil
		}
		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf(
				"service %s state=%s pid=%s previous_pid=%s",
				serviceName,
				currentState.ActiveState,
				currentState.ExecMainPID,
				previousState.ExecMainPID,
			)
		}

		time.Sleep(constants.ServiceVerificationInterval)
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("service %s did not report a new active main pid", serviceName)
	}

	return serviceState{}, fmt.Errorf("timed out verifying restart for %s: %w", serviceName, lastErr)
}

func restartService(serviceName string) error {
	previousState, err := getServiceState(serviceName)
	if err != nil {
		return err
	}

	cmd := runHostSystemctl("restart", serviceName)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to restart %s: %v, stderr: %s", serviceName, err, stderr.String())
	}

	currentState, err := waitForServiceRestart(serviceName, previousState, constants.ServiceVerificationTimeout)
	if err != nil {
		return err
	}

	klog.V(0).InfoS("Verified systemd service restart",
		"service", serviceName,
		"previous_pid", previousState.ExecMainPID,
		"current_pid", currentState.ExecMainPID)

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
