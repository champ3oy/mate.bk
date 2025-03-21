package main

import (
	"fmt"
	"log"
	"mate/config"
	"mate/middleware"
	"mate/routes"
	"time"

	"log/slog"

	slogbetterstack "github.com/samber/slog-betterstack"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/idempotency"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/monitor"
)

func main() {
	config.InitConfig()
	config.ConnectToDB()

	logtail := slog.New(slogbetterstack.Option{Token: "LaqNpxUmKuSH74TGTRKS8iML"}.NewBetterstackHandler())

	app := fiber.New()

	// Initialize handlers
	userHandler := routes.NewUserHandler()
	transactionHandler := routes.NewTransactionHandler()

	app.Use(idempotency.New(idempotency.Config{
		KeyHeaderValidate: func(k string) error {
			if l, wl := len(k), 3; l != wl { // UUID length is 36 chars
				return fmt.Errorf("%w: invalid length: %d != %d", idempotency.ErrInvalidIdempotencyKey, l, wl)
			}

			return nil
		},
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
	}))

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
	app.Post("/login", userHandler.Login)
	app.Post("/consume/:userId", transactionHandler.Consume)
	app.Get("/metrics", monitor.New())

	// Protected routes
	api := app.Group("/api", middleware.APIKeyAuth())
	api.Get("/transaction", transactionHandler.GetTransactions)
	// api.Get("/transactions", transactionHandler.List)

	log.Fatal(app.Listen(":3001"))
}
