package route

import (
	"github.com/gofiber/fiber/v2"

	"github.com/vmindtech/vke-cluster-agent/internal/handler"
)

type AppContext struct {
	App *fiber.App
}

type IRoute interface {
	SetupRoutes(ac *AppContext)
}

type route struct {
	appHandler handler.IAppHandler
}

func NewRoute(
	apHandler handler.IAppHandler,
) IRoute {
	return &route{
		appHandler: apHandler,
	}
}

func (r *route) SetupRoutes(ac *AppContext) {
	api := ac.App.Group("/api")

	// v1 routes
	v1Group := api.Group("/v1")

	r.appRoutes(v1Group)
}

func (r *route) appRoutes(fr fiber.Router) {
	appGroup := fr.Group("/")
	appGroup.Get("/", r.appHandler.App)
}
