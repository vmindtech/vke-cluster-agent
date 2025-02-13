package di

import (
	"github.com/vmindtech/vke-cluster-agent/internal/service"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func InitAppService(k8sClient *kubernetes.Clientset, k8sConfig *rest.Config) service.IAppService {
	openstackService := service.NewOpenstackService()
	vkeService := service.NewVKEService()
	return service.NewAppService(openstackService, vkeService, k8sClient, k8sConfig)
}
