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
	iAppService := service.NewAppService(l)

	iAppHandler := handler.NewAppHandler(iAppService)
	iRoute := route.NewRoute(iAppHandler)
	return iRoute
}
