package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/vmindtech/vke-cluster-agent/pkg/response"

	"github.com/vmindtech/vke-cluster-agent/config"
	"github.com/vmindtech/vke-cluster-agent/internal/dto/resource"
	"github.com/vmindtech/vke-cluster-agent/internal/service"
	"github.com/vmindtech/vke-cluster-agent/pkg/utils"
)

type IAppHandler interface {
	App(c *fiber.Ctx) error
}

type appHandler struct {
	appService service.IAppService
}

func NewAppHandler(as service.IAppService) IAppHandler {
	return &appHandler{
		appService: as,
	}
}

func (a *appHandler) App(c *fiber.Ctx) error {
	err := c.JSON(response.NewSuccessResponse(&resource.AppResource{
		App:     config.GlobalConfig.GetWebConfig().AppName,
		Env:     config.GlobalConfig.GetWebConfig().Env,
		Time:    time.Now(),
		Version: config.GlobalConfig.GetWebConfig().Version,
	}))

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			response.NewErrorResponseWithDetails(err, utils.FailedToGetAppMsg, "", "", ""))
	}

	return nil
}
