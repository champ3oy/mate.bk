package routes

import (
	"encoding/json"
	"fmt"
	"log"
	"mate/config"
	"mate/models"
	"mate/utils"
	"math"
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

	if input.Sender != "MobileMoney" && input.Sender != "ATMoney" && input.Sender != "Fidelity" {
		return c.Status(400).JSON(models.Response{
			Success: false,
			Error:   "Invalid sender",
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

	var userx models.User
	err = user.Decode(&userx)
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

	parsedMsg, _ := utils.ParseTransaction(input.Message)

	transaction.Amount = amount
	transaction.Sender = transactionx.Sender
	transaction.Receiver = transactionx.Receiver
	transaction.TransactionID = transactionx.TransactionID
	transaction.Type = string(parsedMsg.Type)
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
	transactionType := c.Query("type")
	date := c.Query("date")

	// Get today's date in the format stored in your database
	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	// Create a filter for the transactions
	filter := bson.M{"userid": userId}
	if transactionType != "" {
		filter["type"] = transactionType
	}
	if date != "" {
		filter["date"] = date
	}

	// Get all transactions
	cursor, err := config.Find("transactions", filter, options.Find())
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

	// Calculate totals and daily changes
	var (
		totalDebit       float64
		totalCredit      float64
		todayDebit       float64
		todayCredit      float64
		yesterdayDebit   float64
		yesterdayCredit  float64
		debitChange      float64
		creditChange     float64
		finalBalance     float64
		percentageChange float64
	)

	// Calculate totals and separate today's/yesterday's transactions
	for _, transaction := range transactions {
		if transaction.Type == "debit" {
			totalDebit += transaction.Amount
			if transaction.Date == today {
				todayDebit += transaction.Amount
			} else if transaction.Date == yesterday {
				yesterdayDebit += transaction.Amount
			}
		} else if transaction.Type == "credit" {
			totalCredit += transaction.Amount
			if transaction.Date == today {
				todayCredit += transaction.Amount
			} else if transaction.Date == yesterday {
				yesterdayCredit += transaction.Amount
			}
		}
	}

	// Calculate percentage changes
	if yesterdayDebit > 0 {
		debitChange = ((todayDebit - yesterdayDebit) / yesterdayDebit) * 100
	}
	if yesterdayCredit > 0 {
		creditChange = ((todayCredit - yesterdayCredit) / yesterdayCredit) * 100
	}

	// Calculate final balance and its percentage change
	finalBalance = totalCredit - totalDebit
	previousBalance := (yesterdayCredit - yesterdayDebit)
	if previousBalance != 0 {
		percentageChange = ((finalBalance - previousBalance) / math.Abs(previousBalance)) * 100
	}

	// Return success response with filtered transactions and calculations
	return c.JSON(models.Response{
		Success: true,
		Data: fiber.Map{
			"transactions": transactions,
			"stats": fiber.Map{
				"balance": fiber.Map{
					"amount":           finalBalance,
					"percentageChange": percentageChange,
				},
				"income": fiber.Map{
					"amount":           totalCredit,
					"percentageChange": creditChange,
				},
				"expense": fiber.Map{
					"amount":           totalDebit,
					"percentageChange": debitChange,
				},
			},
		},
	})
}
