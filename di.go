package goboilerplate

import (
	"github.com/sirupsen/logrus"
	"github.com/vmindtech/vke-cluster-agent/internal/handler"
	"github.com/vmindtech/vke-cluster-agent/internal/route"
	"github.com/vmindtech/vke-cluster-agent/internal/service"
)

func InitHealthCheckHandler() handler.IHealthCheckHandler {
	iHealthCheckHandler := handler.NewHealthCheckHandler()
	return iHealthCheckHandler
}

func InitRoute(l *logrus.Logger) route.IRoute {
	iOpenstackService := service.NewOpenstackService(l)
	iVKEClusterService := service.NewVKEService(l)
	iAppService := service.NewAppService(l, iOpenstackService, iVKEClusterService)

	iAppHandler := handler.NewAppHandler(iAppService)
	iRoute := route.NewRoute(iAppHandler)
	return iRoute
}

func InitAppService(l *logrus.Logger) service.IAppService {
	iOpenstackService := service.NewOpenstackService(l)
	iVKEClusterService := service.NewVKEService(l)
	iAppService := service.NewAppService(l, iOpenstackService, iVKEClusterService)
	return iAppService
}
