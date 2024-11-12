package main

import (
	"log"
	"mate/config"
	"mate/middleware"
	"mate/routes"

	"github.com/gofiber/fiber/v2"
)

func main() {
	config.InitConfig()
	config.ConnectToDB()

	app := fiber.New()

	// Initialize handlers
	userHandler := routes.NewUserHandler()
	transactionHandler := routes.NewTransactionHandler()

	// Public routes
	app.Post("/register", userHandler.Register)
	app.Post("/consume/:userId", transactionHandler.Consume)

	// Protected routes
	api := app.Group("/api", middleware.APIKeyAuth())
	api.Get("/transaction", transactionHandler.GetTransactions)
	// api.Get("/transactions", transactionHandler.List)

	log.Fatal(app.Listen(":3001"))
}
