package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"mate/models"
)

func ParseTransaction(message string) (*models.Transaction, error) {
	message = strings.TrimSpace(message)

	// Dynamic regex patterns
	amountPattern := regexp.MustCompile(`(?i)(GHS\s*[\d,.]+)`)
	transactionIDPattern := regexp.MustCompile(`(?i)(Transaction ID|Financial Transaction Id|Transaction Id):?\s*(\d+)`)
	feePattern := regexp.MustCompile(`(?i)(Fee|Fee charged):?\s*GHS\s*([\d.]+)`)
	taxPattern := regexp.MustCompile(`(?i)(Tax Charged):?\s*([\d.]+)`)
	balanceBeforePattern := regexp.MustCompile(`(?i)(Available Balance):?\s*GHS\s*([\d,.]+)`)
	balanceAfterPattern := regexp.MustCompile(`(?i)(Current Balance):?\s*GHS\s*([\d,.]+)`)
	senderPattern := regexp.MustCompile(`(?i)(from|to)\s+(.+?)\s*(Current Balance|Reference|Transaction ID)`)
	recipientPattern := regexp.MustCompile(`(?i)(to|for)\s+(.+?)\s*(Fee|Balance|Transaction)`)
	referencePattern := regexp.MustCompile(`(?i)(Reference|Ref):?\s*([^\n]+)`)

	var transaction models.Transaction

	// Extract amount
	amountMatch := amountPattern.FindStringSubmatch(message)
	if len(amountMatch) > 0 {
		transaction.Amount = parseAmount(amountMatch[1])
	}

	// Extract balance
	balanceAfterMatch := balanceAfterPattern.FindStringSubmatch(message)
	if len(balanceAfterMatch) > 0 {
		transaction.BalanceAfter = parseAmount(balanceAfterMatch[2])
	}
	balanceBeforeMatch := balanceBeforePattern.FindStringSubmatch(message)
	if len(balanceBeforeMatch) > 0 {
		transaction.BalanceAfter = parseAmount(balanceBeforeMatch[2])
	}

	// Extract fee
	feeMatch := feePattern.FindStringSubmatch(message)
	if len(feeMatch) > 0 {
		transaction.Fee = parseAmount(feeMatch[2])
	}

	// Extract tax
	taxMatch := taxPattern.FindStringSubmatch(message)
	if len(taxMatch) > 0 {
		transaction.Tax = parseAmount(taxMatch[2])
	}

	// Extract transaction ID
	transactionIDMatch := transactionIDPattern.FindStringSubmatch(message)
	if len(transactionIDMatch) > 0 {
		transaction.TransactionID = transactionIDMatch[2]
	}

	// Extract sender or recipient
	senderMatch := senderPattern.FindStringSubmatch(message)
	if len(senderMatch) > 0 {
		transaction.Sender = senderMatch[2]
	}

	// Extract recipient
	recipientMatch := recipientPattern.FindStringSubmatch(message)
	if len(recipientMatch) > 0 {
		transaction.Receiver = recipientMatch[2]
	}

	// Extract reference (if any)
	referenceMatch := referencePattern.FindStringSubmatch(message)
	if len(referenceMatch) > 0 {
		transaction.Reference = referenceMatch[2]
	}

	// Determine transaction type based on presence of keywords
	if strings.Contains(message, "Payment received") {
		transaction.Type = "received"
	} else if strings.Contains(message, "Payment for") {
		transaction.Type = "sent"
	} else if strings.Contains(message, "Cash out made") {
		transaction.Type = "cash_out"
	} else {
		return nil, fmt.Errorf("unrecognized transaction type")
	}

	// Add the current date to the transaction
	transaction.Timestamp = time.Now()

	return &transaction, nil
}

// Helper function to parse currency (GHS) amounts
func parseAmount(value string) float64 {
	val, err := strconv.ParseFloat(strings.ReplaceAll(value, ",", ""), 64)
	if err != nil {
		return 0.0
	}
	return val
}
