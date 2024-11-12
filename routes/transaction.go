package routes

import (
	"encoding/json"
	"fmt"
	"log"
	"mate/config"
	"mate/models"
	"mate/utils"
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type TransactionHandler struct{}

func NewTransactionHandler() *TransactionHandler {
	return &TransactionHandler{}
}

func (h *TransactionHandler) Consume(c *fiber.Ctx) error {
	var input struct {
		Message string `json:"message"`
		Time    string `json:"time"`
		Sender  string `json:"sender"`
	}

	// Parse request body
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(models.Response{
			Success: false,
			Error:   "Invalid input",
		})
	}

	var transaction models.Transaction

	userId := c.Params("userId")
	filter := bson.M{"userid": userId}
	user, err := config.FindOne("users", filter)
	if err != nil {
		log.Println(err)
		return c.Status(400).JSON(models.Response{
			Success: false,
			Error:   "User not found",
		})
	}

	var userx models.User // Create an instance of models.User

	err = user.Decode(&userx) // Decode into the instance
	if err != nil {

		log.Println(err)
		return c.Status(400).JSON(models.Response{

			Success: false,
			Error:   "Error decoding user",
		})

	}

	transaction.UserID = userx.ID.String()

	// parse sms
	transactionLLM, err := utils.ExtractEntitiesFromSMS(input.Message)
	if err != nil {
		log.Println(err)
		return c.Status(400).JSON(models.Response{
			Success: false,
			Error:   "Error parsing message",
		})
	}

	escapedJSON := transactionLLM.Choices[0].Message.Content

	cleanJSON := strings.Replace(escapedJSON, "\\n", "", -1)

	// Remove unnecessary text ".deep array"
	cleanJSON = strings.Replace(cleanJSON, ".deep array", "", -1)

	if !strings.HasSuffix(cleanJSON, "}") {
		cleanJSON += "}"
	}

	// Remove unnecessary text using regex
	re := regexp.MustCompile(`"receiver":\s*"(.*?)\s*Current Balance:.*?"`)
	cleanJSON = re.ReplaceAllString(cleanJSON, `"receiver": "$1"`)

	var transactionx models.LLMTransaction

	err = json.Unmarshal([]byte(cleanJSON), &transactionx)
	if err != nil {
		fmt.Println(err)
		fmt.Println(cleanJSON)
		return c.Status(400).JSON(models.Response{
			Success: false,
			Error:   "Error turning LLM response to Struct",
		})
	}

	amount, err := utils.ConvertCurrencyToFloat(transactionx.Amount)
	if err != nil {
		log.Println(err)
		return c.Status(400).JSON(models.Response{
			Success: false,
			Error:   "Error converting amount to number",
		})
	}
	transaction.Amount = amount
	transaction.Sender = transactionx.Sender
	transaction.Receiver = transactionx.Receiver
	transaction.TransactionID = transactionx.TransactionID
	transaction.Type = transactionx.Type
	transaction.RawSMS = input.Message
	transaction.Reference = transactionx.Reference
	transaction.Timestamp = time.Now()
	transaction.Origin = input.Sender
	transaction.Date = input.Time

	transaction.Fee, err = utils.ConvertCurrencyToFloat(transactionx.Fee)
	if err != nil {
		return c.Status(400).JSON(models.Response{
			Success: false,
			Error:   "Error converting fee to number",
		})
	}

	transaction.Tax, err = utils.ConvertCurrencyToFloat(transactionx.Tax)
	if err != nil {
		fmt.Println(transactionx)
		fmt.Println(err)
		return c.Status(400).JSON(models.Response{
			Success: false,
			Error:   "Error converting tax to number",
		})
	}

	transaction.BalanceAfter, err = utils.ConvertCurrencyToFloat(transactionx.Balance)
	if err != nil {
		return c.Status(400).JSON(models.Response{
			Success: false,
			Error:   "Error converting balance to number",
		})
	}

	err = config.InsertOne("transactions", transaction)
	if err != nil {
		return c.Status(400).JSON(models.Response{
			Success: false,
			Error:   "Error saving transaction",
		})
	}

	// Return success response
	return c.JSON(models.Response{
		Success: true,
		Data: fiber.Map{
			"message": "Message parsed successfully",
			"success": true,
		},
	})
}

func (h *TransactionHandler) GetTransactions(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)
	userId := user.ID.String()

	// Parse query parameters for type and date
	transactionType := c.Query("type") // e.g., "credit" or "debit"
	date := c.Query("date")            // e.g., "2023-10-01"

	// Create a filter for the transactions
	filter := bson.M{"userid": userId}
	if transactionType != "" {
		filter["type"] = transactionType
	}
	if date != "" {
		filter["date"] = date // Assuming you have a date field in your transaction model
	}

	// Add options for sorting or limiting if needed
	options := options.Find() // Create options if needed
	// Example: options.SetSort(bson.D{{"date", -1}}) // Sort by date descending

	cursor, err := config.Find("transactions", filter, options) // Pass options here
	if err != nil {
		log.Println(err)
		return c.Status(400).JSON(models.Response{
			Success: false,
			Error:   "Error fetching transactions",
		})
	}
	defer cursor.Close(c.Context())

	var transactions []models.Transaction
	if err = cursor.All(c.Context(), &transactions); err != nil {
		log.Println(err)
		return c.Status(400).JSON(models.Response{
			Success: false,
			Error:   "Error decoding transactions",
		})
	}

	// Update the transactions slice to only include the required fields
	var filteredTransactions []struct {
		Type      string    `json:"type"`
		Amount    float64   `json:"amount"`
		Timestamp time.Time `json:"timestamp"`
	}

	for _, transaction := range transactions {
		filteredTransactions = append(filteredTransactions, struct {
			Type      string    `json:"type"`
			Amount    float64   `json:"amount"`
			Timestamp time.Time `json:"timestamp"`
		}{
			Type:      transaction.Type,
			Amount:    transaction.Amount,
			Timestamp: transaction.Timestamp,
		})
	}

	// Return success response with filtered transactions
	return c.JSON(models.Response{
		Success: true,
		Data:    filteredTransactions,
	})
}

//
