package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

var (
	MongoURI    string
	MongoDBName string
)

// Load environment variables
func InitConfig() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	MongoURI = os.Getenv("MONGO_URI")
	MongoDBName = os.Getenv("MONGO_DB_NAME")
}
