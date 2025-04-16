package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func LoadEnv() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Println(err)
	}
}

func GetEnv(key string, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return fallback
	}

	return value
}
