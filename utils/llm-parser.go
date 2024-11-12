package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mate/models"
	"net/http"
	"os"
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

	// Define the prompt to send to the model
	prompt := fmt.Sprintf(`
Extract the following details from the given SMS message and output strictly in JSON format with key-value pairs:
- Amount (e.g., GHS 10.00)
- Sender (who the money is from)
- Receiver (who the money is sent to)
- Transaction ID (e.g., 12345678911)
- Fee (if any)
- Tax (if any)
- Balance (if mentioned)
- Type (indicate whether the transaction is a credit or a debit. A transaction is a debit if it involves sending money (e.g., payment for goods/services or cash out). A transaction is a credit if it involves receiving money (e.g., payment received).)
- Reference (any ref or reference if available; clarify that it may be referred to as either "ref" or "reference")

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
  "reference": "value
}
`, message)

	// Prepare the request data in JSON format with the correct structure
	data := map[string]interface{}{
		"model": "meta-llama/Llama-3.2-1B-Instruct", // Specify the model
		"messages": []map[string]interface{}{
			{
				"role":    "system",
				"content": prompt,
			},
		},
		"max_tokens": 700,   // Optional: Limit the number of tokens for the response
		"stream":     false, // Optional: Use streaming for the response (can be false if you want the full response at once)
	}

	// Convert data map to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	// Create a POST request
	req, err := http.NewRequest("POST", modelEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	// Set necessary headers
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check for successful response
	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("Error from Hugging Face API: %s", body)
	}

	// Parse the response
	var result *models.LLMResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	return result, nil
}
