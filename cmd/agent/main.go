package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	di "github.com/vmindtech/vke-cluster-agent"
	"github.com/vmindtech/vke-cluster-agent/config"
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

	// Start certificate expiration check
	isExpired := make(chan bool)
	go appService.CheckVKEClusterCertificateExpiration(isExpired)

	// Start certificate renewal process
	go func() {
		if err := appService.RenewMasterNodesCertificates(); err != nil {
			klog.ErrorS(err, "Failed to renew master certificates",
				"cluster_id", clID,
				"component", "certificate_renewal")
			return
		}

		if err := appService.RestartWorkerNodes(); err != nil {
			klog.ErrorS(err, "Failed to restart worker nodes",
				"cluster_id", clID,
				"component", "worker_restart")
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	klog.V(0).InfoS("Shutting down VKE cluster agent",
		"cluster_id", clID,
		"component", "shutdown")
}
