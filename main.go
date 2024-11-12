package main

import (
	"log"
	"mate/config"
	"mate/middleware"
	"mate/routes"
	"time"

	"log/slog"

	slogbetterstack "github.com/samber/slog-betterstack"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	config.InitConfig()
	config.ConnectToDB()

	logtail := slog.New(slogbetterstack.Option{Token: "LaqNpxUmKuSH74TGTRKS8iML"}.NewBetterstackHandler())

	app := fiber.New()

	// Initialize handlers
	userHandler := routes.NewUserHandler()
	transactionHandler := routes.NewTransactionHandler()

	app.Use(logger.New(logger.Config{
		TimeFormat: time.RFC3339Nano,
		TimeZone:   "Africa/Accra",
		Done: func(c *fiber.Ctx, logString []byte) {
			bodyBytes := c.Body()
			logtail.Info(string(logString) + " | Request Body: " + string(bodyBytes))
		},
	}))

	// Public routes
	app.Post("/register", userHandler.Register)
	app.Post("/consume/:userId", transactionHandler.Consume)

	// Protected routes
	api := app.Group("/api", middleware.APIKeyAuth())
	api.Get("/transaction", transactionHandler.GetTransactions)
	// api.Get("/transactions", transactionHandler.List)

	log.Fatal(app.Listen(":3001"))
}
