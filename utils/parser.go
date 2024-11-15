package utils

import (
	"regexp"
	"strconv"
	"strings"
)

// TransactionType represents the type of transaction
type TransactionType string

const (
	Credit  TransactionType = "credit"
	Debit   TransactionType = "debit"
	Unknown TransactionType = "UNKNOWN"
)

// Transaction represents the parsed transaction details
type Transaction struct {
	Type           TransactionType `json:"type"`
	Amount         float64         `json:"amount"`
	CurrentBalance float64         `json:"current_balance"`
	TransactionID  string          `json:"transaction_id"`
	Fee            float64         `json:"fee"`
	RawMessage     string          `json:"raw_message"`
}

// RawSMS represents the input message structure
type RawSMS struct {
	RawSMS string `json:"raw_sms"`
}

// ParseTransaction parses a raw SMS message and returns transaction details
func ParseTransaction(rawJSON string) Transaction {
	transaction := parseMessage(rawJSON)
	return transaction
}

func determineTransactionType(message string) TransactionType {
	msg := strings.ToLower(message)

	// Credit patterns
	creditPatterns := []string{
		"payment received",
		"money received",
		"cash received",
		"received",
		"credited",
		"credit",
		"deposited",
		"deposit",
		"cash in",
	}

	// Debit patterns
	debitPatterns := []string{
		"payment made",
		"payment for",
		"paid to",
		"paid",
		"payment to",
		"sent to",
		"transferred to",
		"withdrawal",
		"withdrew",
		"debited",
		"cash out",
		"debit alert",
		"debit",
	}

	// First check debit patterns as they are typically more explicit
	for _, pattern := range debitPatterns {
		if strings.Contains(msg, pattern) {
			return Debit
		}
	}

	// Then check credit patterns
	for _, pattern := range creditPatterns {
		if strings.Contains(msg, pattern) {
			return Credit
		}
	}

	// If no patterns match, check for additional context
	// Most messages starting with "Payment" without credit indicators are typically debits
	if strings.HasPrefix(msg, "payment") {
		return Debit
	}

	return Unknown
}

func parseMessage(message string) Transaction {
	transaction := Transaction{
		RawMessage: message,
	}

	// Determine transaction type using the new robust method
	transaction.Type = determineTransactionType(message)

	// Extract amount
	amountRegex := regexp.MustCompile(`GHS\s*(\d+(?:\.\d{2})?)`)
	if matches := amountRegex.FindStringSubmatch(message); len(matches) > 1 {
		if amount, err := strconv.ParseFloat(matches[1], 64); err == nil {
			transaction.Amount = amount
		}
	}

	// Extract current balance
	balanceRegex := regexp.MustCompile(`Current Balance:\s*GHS\s*(\d+(?:\.\d{2})?)`)
	if matches := balanceRegex.FindStringSubmatch(message); len(matches) > 1 {
		if balance, err := strconv.ParseFloat(matches[1], 64); err == nil {
			transaction.CurrentBalance = balance
		}
	}

	// Extract transaction ID
	txIDRegex := regexp.MustCompile(`Transaction (?:ID|Id):\s*(\d+)`)
	if matches := txIDRegex.FindStringSubmatch(message); len(matches) > 1 {
		transaction.TransactionID = matches[1]
	}

	// Extract fee
	feeRegex := regexp.MustCompile(`(?:Fee charged|TRANSACTION FEE):\s*GHS\s*(\d+(?:\.\d{2})?)`)
	if matches := feeRegex.FindStringSubmatch(message); len(matches) > 1 {
		if fee, err := strconv.ParseFloat(matches[1], 64); err == nil {
			transaction.Fee = fee
		}
	}

	return transaction
}
