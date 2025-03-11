package main

import (
	"flag"
	"os"
	"time"

	di "github.com/vmindtech/vke-cluster-agent"
	"github.com/vmindtech/vke-cluster-agent/config"
	"github.com/vmindtech/vke-cluster-agent/pkg/constants"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	configureManager := config.NewConfigureManager()
	clID := configureManager.GetVKEConfig().ClusterID

	klog.V(0).InfoS("Starting VKE cluster agent",
		"cluster_id", clID,
		"env", configureManager.GetWebConfig().Env,
		"app_name", configureManager.GetWebConfig().AppName,
		"version", configureManager.GetWebConfig().Version,
		"component", "startup")

	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		klog.ErrorS(err, "Failed to get in-cluster config",
			"cluster_id", clID,
			"component", "startup")
		os.Exit(1)
	}

	k8sClient, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		klog.ErrorS(err, "Failed to create k8s client",
			"cluster_id", clID,
			"component", "startup")
		os.Exit(1)
	}

	appService := di.InitAppService(k8sClient, k8sConfig)

	for {
		isExpired := make(chan bool)
		go appService.CheckVKEClusterCertificateExpiration(isExpired)

		select {
		case expired := <-isExpired:
			if expired {
				klog.V(0).Info("Certificate expiration detected, starting renewal process")

				if err := appService.RenewMasterNodesCertificates(); err != nil {
					klog.Errorf("Failed to renew master certificates: %v", err)
					continue
				}

				if err := appService.RestartWorkerNodes(); err != nil {
					klog.Errorf("Failed to restart worker nodes: %v", err)
					continue
				}

				klog.V(0).Info("Certificate renewal process completed successfully")
			}
		case <-time.After(constants.RenewalProcessTimeout):
			klog.V(2).Info("Renewal process timed out, restarting check cycle")
		}

		time.Sleep(constants.CertificateCheckInterval)
	}
}
