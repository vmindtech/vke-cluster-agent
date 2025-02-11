package main

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/sirupsen/logrus"

	"github.com/vmindtech/vke-cluster-agent/pkg/response"
	"github.com/vmindtech/vke-cluster-agent/pkg/utils"
	"github.com/vmindtech/vke-cluster-agent/pkg/validation"

	di "github.com/vmindtech/vke-cluster-agent"
	"github.com/vmindtech/vke-cluster-agent/internal/middleware"
	"github.com/vmindtech/vke-cluster-agent/internal/route"
)

type application struct {
	Logger         *logrus.Logger
	LanguageBundle *i18n.Bundle
}

func initApplication(a *application) *fiber.App {
	app := fiber.New(fiber.Config{
		// Override default error handler - Internal server err
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			errBag := utils.ErrorBag{Code: utils.UnexpectedErrCode, Message: utils.UnexpectedMsg}
			code, _ := strconv.Atoi(utils.UnexpectedErrCode)
			return c.Status(code).JSON(response.NewErrorResponse(c.Context(), errBag))
		},
	})

	// Health check routes
	a.addHealthCheckRoutes(app)

	// Common middleware
	a.addCommonMiddleware(app)

	r := di.InitRoute(a.Logger)
	r.SetupRoutes(&route.AppContext{
		App: app,
	})

	// 404 handler
	app.Use(func(c *fiber.Ctx) error {
		errBag := utils.ErrorBag{Code: utils.NotFoundErrCode, Message: utils.NotFoundMsg}

		return c.Status(fiber.StatusNotFound).JSON(response.NewErrorResponse(c.Context(), errBag))
	})

	// Check VKE Cluster Certificate Expiration
	var isExpired chan bool

	go di.InitAppService(a.Logger).CheckVKEClusterCertificateExpiration(isExpired)
	return app
}

func (a *application) addCommonMiddleware(app *fiber.App) {
	app.Use(middleware.RecoverMiddleware(a.Logger))
	app.Use(requestid.New())
	app.Use(middleware.LoggerMiddleware(a.Logger))
	app.Use(middleware.LocalizerMiddleware(a.LanguageBundle))
	app.Use(cors.New())

	// Validator
	validator := validation.InitValidator()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(utils.ValidatorKey, validator)

		return c.Next()
	})

	// Tokenizer
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(utils.TokenizerKey)

		return c.Next()
	})
}

func (a *application) addHealthCheckRoutes(app *fiber.App) {
	healthCheckHandler := di.InitHealthCheckHandler()
	app.Get("/liveness", healthCheckHandler.Liveness)
	app.Get("/readiness", healthCheckHandler.Readiness)
}
