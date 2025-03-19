package config

import (
	"os"

	"github.com/joho/godotenv"
)

var (
	MongoURI    string
	MongoDBName string
)

// Load environment variables
func InitConfig() {
	_ = godotenv.Overload() // Ignore error if .env is missing

	MongoURI = os.Getenv("MONGO_URI")
	MongoDBName = os.Getenv("MONGO_DB_NAME")
}
