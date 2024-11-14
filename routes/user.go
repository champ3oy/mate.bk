package routes

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"mate/config"
	"mate/models"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct{}

func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

func (h *UserHandler) Register(c *fiber.Ctx) error {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	// Parse request body
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(models.Response{
			Success: false,
			Error:   "Invalid input",
		})
	}

	// Check if email already exists
	filter := bson.M{"email": input.Email}
	result, err := config.FindOne("users", filter)
	if err == nil {
		return c.Status(400).JSON(models.Response{
			Success: false,
			Error:   "Email already exists",
		})
	}
	if result.Err() != nil && result.Err() != mongo.ErrNoDocuments {
		return c.Status(500).JSON(models.Response{
			Success: false,
			Error:   "Error checking existing user",
		})
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(500).JSON(models.Response{
			Success: false,
			Error:   "Error processing request",
		})
	}

	// Generate API key
	apiKey, err := generateAPIKey()
	if err != nil {
		return c.Status(500).JSON(models.Response{
			Success: false,
			Error:   "Error generating API key",
		})
	}

	// Generate User Unique ID
	userID, err := GenerateUniqueID()
	if err != nil {
		return c.Status(500).JSON(models.Response{
			Success: false,
			Error:   "Error generating User Id",
		})
	}

	// Create user
	user := models.User{
		Email:     input.Email,
		Password:  string(hashedPassword),
		ApiKey:    "mate_" + apiKey,
		CreatedAt: time.Now(),
		UserID:    userID,
	}

	// Insert new user into database
	err = config.InsertOne("users", user)
	if err != nil {
		return c.Status(500).JSON(models.Response{
			Success: false,
			Error:   "Error creating user",
		})
	}

	// Return success response
	return c.JSON(models.Response{
		Success: true,
		Data: fiber.Map{
			"email":   user.Email,
			"api_key": user.ApiKey,
			"user_id": user.UserID,
		},
	})
}

func (h *UserHandler) Login(c *fiber.Ctx) error {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	// Parse request body
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(models.Response{
			Success: false,
			Error:   "Invalid input",
		})
	}

	// Find user by email
	filter := bson.M{"email": input.Email}
	result, err := config.FindOne("users", filter)
	if err != nil {
		return c.Status(400).JSON(models.Response{
			Success: false,
			Error:   "User not found",
		})
	}

	// Check password
	var user models.User
	if err := result.Decode(&user); err != nil {
		return c.Status(500).JSON(models.Response{
			Success: false,
			Error:   "Error retrieving user",
		})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		return c.Status(400).JSON(models.Response{
			Success: false,
			Error:   "Invalid credentials",
		})
	}

	// Return success response
	return c.JSON(models.Response{
		Success: true,
		Data: fiber.Map{
			"email":   user.Email,
			"api_key": user.ApiKey,
			"user_id": user.UserID,
		},
	})
}

// Generate a new API key for the user
func generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func GenerateUniqueID() (string, error) {
	// Create a byte slice to hold the random bytes
	bytes := make([]byte, 6) // 6 bytes will give us 12 characters in hex

	// Generate random bytes
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Convert bytes to a hex string and return the first 12 characters
	return hex.EncodeToString(bytes)[:12], nil
}
