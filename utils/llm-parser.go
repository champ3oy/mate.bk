package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mate/models"
	"net/http"
	"os"
	"time"
)

type PaymentDetails struct {
	Amount        string `json:"amount"`
	Sender        string `json:"sender"`
	TransactionID string `json:"transaction_id"`
	Fee           string `json:"fee"`
	Tax           string `json:"tax"`
	Balance       string `json:"balance"`
}

func ExtractEntitiesFromSMS(message string) (*models.LLMResponse, error) {
	apiKey := os.Getenv("HUGGING_FACE_API")
	modelEndpoint := "https://api-inference.huggingface.co/models/meta-llama/Llama-3.2-1B-Instruct/v1/chat/completions"

	// Enhanced prompt with clear transaction type classification rules
	prompt := fmt.Sprintf(`
Extract the following details from the given SMS message and output strictly in JSON format with key-value pairs. 
Pay special attention to the transaction type classification rules below:

Transaction Type Classification Rules:
1. DEBIT transactions (money going out) are indicated by phrases like:
   - "Payment made to"
   - "You paid"
   - "Cash Out"
   - "Transfer to"
   - "Withdrawal"
   - "Purchase"
   - "You bought"
   - "You sent"
   - "Debited"

2. CREDIT transactions (money coming in) are indicated by phrases like:
   - "Payment received"
   - "You received"
   - "Cash In"
   - "Transfer from"
   - "Deposit"
   - "Credited"
   - "Sent you"
   - "Payment from"

3. Default Rules:
   - If the message indicates money leaving the account, it's a "debit"
   - If the message indicates money entering the account, it's a "credit"
   - If unclear, look for keywords indicating direction of money flow

Extract these fields:
- Amount (e.g., GHS 10.00)
- Sender (who the money is from)
- Receiver (who the money is sent to)
- Transaction ID (e.g., 12345678911)
- Fee (if any)
- Tax (if any)
- Balance (Current balanced if mentioned else Available balanced)
- Type (must be either "credit" or "debit" based on the rules above)
- Reference (any ref or reference if available)

Do not include any explanation, only return the JSON.

SMS Message: "%s"

Output Format:
{
  "amount": "value",
  "sender": "value",
  "receiver": "value",
  "transaction_id": "value",
  "fee": "value",
  "tax": "value",
  "balance": "value",
  "type": "value",
  "reference": "value"
}
`, message)

	// Prepare the request data
	data := map[string]interface{}{
		"model": "meta-llama/Llama-3.2-1B-Instruct",
		"messages": []map[string]interface{}{
			{
				"role":    "system",
				"content": prompt,
			},
		},
		"max_tokens":  700,
		"stream":      false,
		"temperature": 0.1, // Added: Lower temperature for more consistent outputs
		"top_p":       0.9, // Added: Reduce randomness while maintaining some flexibility
	}

	// Convert data map to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request data: %w", err)
	}

	// Create and configure HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create a POST request
	req, err := http.NewRequest("POST", modelEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var result *models.LLMResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return result, nil
}
