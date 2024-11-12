package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    string             `json:"user_id"`
	Balance   float64            `json:"balance"`
	Currency  string             `json:"currency"`
	CreatedAt time.Time          `json:"created_at"`
	Email     string             `bson:"email" json:"email"`
	Password  string             `bson:"password" json:"-"`
	ApiKey    string             `bson:"api_key" json:"api_key"`
}
