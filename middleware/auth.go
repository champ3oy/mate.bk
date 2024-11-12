package middleware

import (
	"mate/config"
	"mate/models"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
)

func APIKeyAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		apiKey := c.Get("Authorization")
		if apiKey == "" {
			return c.Status(401).JSON(models.Response{
				Success: false,
				Error:   "API key required",
			})
		}

		// Verify API key
		filter := bson.M{"api_key": apiKey}
		var user models.User
		result, err := config.FindOne("users", filter)

		if err != nil {
			return c.Status(401).JSON(models.Response{
				Success: false,
				Error:   "Invalid API key",
			})
		}

		// Decode the result into the User struct
		if err := result.Decode(&user); err != nil {
			return c.Status(500).JSON(models.Response{
				Success: false,
				Error:   "Error decoding user data",
			})
		}

		// Add user to context
		c.Locals("user", user)
		return c.Next()
	}
}
