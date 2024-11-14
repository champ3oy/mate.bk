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

	parsedMsg := utils.ParseTransaction(input.Message)

	fmt.Println(parsedMsg)

	transaction.Amount = amount
	transaction.Sender = transactionx.CounterParty
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

	// Get time periods
	now := time.Now()
	today := now.Format("2006-01-02")
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")

	// Create a filter for the transactions
	filter := bson.M{"userid": userId}
	if transactionType != "" {
		filter["type"] = transactionType
	}
	if date != "" {
		filter["date"] = date
	}

	// Get all transactions
	cursor, err := config.Find("transactions", filter, options.Find().SetSort(bson.D{{Key: "timestamp", Value: 1}}))
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

	// Initialize analysis structures
	type HourlyStats struct {
		Hour         string  `json:"hour"`
		Transactions int     `json:"transactions"`
		Volume       float64 `json:"volume"`
	}

	type UserStats struct {
		Name   string  `json:"name"`
		Amount float64 `json:"value"`
		Type   string  `json:"type"`
	}

	type DailyStats struct {
		Date    string  `json:"date"`
		Credits float64 `json:"credits"`
		Debits  float64 `json:"debits"`
		Balance float64 `json:"balance"`
		NetFlow float64 `json:"netFlow"`
	}

	// Initialize analysis maps
	hourlyStats := make(map[string]*HourlyStats)
	userStats := make(map[string]*UserStats)
	dailyStats := make(map[string]*DailyStats)
	origins := make(map[string]int)

	// Initialize calculation variables
	var (
		totalDebit      float64
		totalCredit     float64
		todayDebit      float64
		todayCredit     float64
		yesterdayDebit  float64
		yesterdayCredit float64
		totalFees       float64
		totalTax        float64
		maxAmount       float64
		minAmount       float64
		maxBalance      float64
		minBalance      float64
	)

	// Process each transaction
	for _, tx := range transactions {
		// Basic amounts
		if tx.Type == "debit" {
			totalDebit += tx.Amount
			if tx.Date == today {
				todayDebit += tx.Amount
			} else if tx.Date == yesterday {
				yesterdayDebit += tx.Amount
			}
		} else if tx.Type == "credit" {
			totalCredit += tx.Amount
			if tx.Date == today {
				todayCredit += tx.Amount
			} else if tx.Date == yesterday {
				yesterdayCredit += tx.Amount
			}
		}

		// Track min/max amounts and balances
		if tx.Amount > maxAmount {
			maxAmount = tx.Amount
		}
		if tx.Amount < minAmount || minAmount == 0 {
			minAmount = tx.Amount
		}
		if tx.BalanceAfter > maxBalance {
			maxBalance = tx.BalanceAfter
		}
		if tx.BalanceAfter < minBalance || minBalance == 0 {
			minBalance = tx.BalanceAfter
		}

		// Accumulate fees and tax
		totalFees += tx.Fee
		totalTax += tx.Tax

		// Hourly statistics
		hour := tx.Timestamp.Format("15:00")
		if _, exists := hourlyStats[hour]; !exists {
			hourlyStats[hour] = &HourlyStats{Hour: hour}
		}
		hourlyStats[hour].Transactions++
		hourlyStats[hour].Volume += tx.Amount

		// User statistics
		userKey := tx.Sender
		if userKey == "" {
			userKey = tx.Receiver
		}
		if _, exists := userStats[userKey]; !exists {
			userStats[userKey] = &UserStats{Name: userKey, Type: tx.Type}
		}
		userStats[userKey].Amount += tx.Amount

		// Daily statistics
		if _, exists := dailyStats[tx.Date]; !exists {
			dailyStats[tx.Date] = &DailyStats{Date: tx.Date}
		}
		if tx.Type == "credit" {
			dailyStats[tx.Date].Credits += tx.Amount
		} else {
			dailyStats[tx.Date].Debits += tx.Amount
		}
		dailyStats[tx.Date].Balance = tx.BalanceAfter
		dailyStats[tx.Date].NetFlow = dailyStats[tx.Date].Credits - dailyStats[tx.Date].Debits

		// Track transaction origins
		origins[tx.Origin]++
	}

	// Calculate percentage changes
	debitChange := calculatePercentageChange(todayDebit, yesterdayDebit)
	creditChange := calculatePercentageChange(todayCredit, yesterdayCredit)

	// Calculate final balance and its percentage change
	finalBalance := totalCredit - totalDebit
	previousBalance := yesterdayCredit - yesterdayDebit
	balanceChange := calculatePercentageChange(finalBalance, previousBalance)

	// Convert maps to slices for JSON
	hourlyStatsSlice := make([]HourlyStats, 0, len(hourlyStats))
	for _, stats := range hourlyStats {
		hourlyStatsSlice = append(hourlyStatsSlice, *stats)
	}

	userStatsSlice := make([]UserStats, 0, len(userStats))
	for _, stats := range userStats {
		userStatsSlice = append(userStatsSlice, *stats)
	}

	dailyStatsSlice := make([]DailyStats, 0, len(dailyStats))
	for _, stats := range dailyStats {
		dailyStatsSlice = append(dailyStatsSlice, *stats)
	}

	// Prepare origin data for charts
	originData := make([]map[string]interface{}, 0)
	for origin, count := range origins {
		originData = append(originData, map[string]interface{}{
			"name":       origin,
			"value":      count,
			"percentage": float64(count) / float64(len(transactions)) * 100,
		})
	}

	// Return comprehensive analysis
	return c.JSON(models.Response{
		Success: true,
		Data: fiber.Map{
			"transactions": transactions,
			"basicStats": fiber.Map{
				"totalTransactions": len(transactions),
				"netFlow": fiber.Map{
					"amount":           finalBalance,
					"percentageChange": balanceChange,
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
			"transactionAnalysis": fiber.Map{
				"averageTransaction": (totalCredit + totalDebit) / float64(len(transactions)),
				"maxTransaction":     maxAmount,
				"minTransaction":     minAmount,
				"maxBalance":         maxBalance,
				"minBalance":         minBalance,
				"totalFees":          totalFees,
				"totalTax":           totalTax,
			},
			"timeAnalysis": fiber.Map{
				"hourlyStats": hourlyStatsSlice,
				"dailyStats":  dailyStatsSlice,
			},
			"userAnalysis": fiber.Map{
				"userStats":   userStatsSlice,
				"uniqueUsers": len(userStats),
			},
			"originAnalysis": fiber.Map{
				"origins":       originData,
				"primaryOrigin": getMostFrequentOrigin(origins),
			},
		},
	})
}

// Helper function to calculate percentage change
func calculatePercentageChange(current, previous float64) float64 {
	if previous == 0 {
		return 0
	}
	return ((current - previous) / math.Abs(previous)) * 100
}

// Helper function to get most frequent origin
func getMostFrequentOrigin(origins map[string]int) string {
	var maxCount int
	var mostFrequent string
	for origin, count := range origins {
		if count > maxCount {
			maxCount = count
			mostFrequent = origin
		}
	}
	return mostFrequent
}
