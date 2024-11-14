package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LLMTransaction struct {
	Amount        string `json:"amount"`
	CounterParty  string `json:"counterParty"`
	TransactionID string `json:"transaction_id"`
	Fee           string `json:"fee"`
	Tax           string `json:"tax"`
	Balance       string `json:"balance"`
	Type          string `json:"type"`
	Reference     string `json:"reference"`
}

type Transaction struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID        string             `json:"userid"`
	Type          string             `json:"type"`
	Amount        float64            `json:"amount"`
	Fee           float64            `json:"fee"`
	Tax           float64            `json:"tax"`
	BalanceBefore float64            `json:"balance_before"`
	BalanceAfter  float64            `json:"balance_after"`
	Date          string             `json:"date"`
	Sender        string             `json:"sender"`
	Receiver      string             `json:"receiver"`
	TransactionID string             `json:"transaction_id"`
	Reference     string             `json:"reference"`
	RawSMS        string             `bson:"raw_sms" json:"raw_sms"`
	Source        string             `bson:"source" json:"source"`
	Timestamp     time.Time          `bson:"timestamp" json:"timestamp"`
	Origin        string             `bson:"origin" json:"origin"`
}
