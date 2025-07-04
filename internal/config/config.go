package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	GeminiAPIKey string
	DatabaseURL  string
	HTTPPort     string
	LogLevel     string
	JWTSecret    string
}

var AppConfig Config

func LoadConfig() {
	err := godotenv.Load() // Load .env file if it exists
	if err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	AppConfig = Config{
		GeminiAPIKey: getEnv("GEMINI_API_KEY", ""),
		DatabaseURL:  getEnv("DATABASE_URL", "gwi_chatbot.db"),
		HTTPPort:     getEnv("HTTP_PORT", "8080"),
		LogLevel:     getEnv("LOG_LEVEL", "INFO"),
		JWTSecret:    getEnv("JWT_SECRET", ""),
	}

	if AppConfig.GeminiAPIKey == "" {
		log.Fatal("GEMINI_API_KEY environment variable is required")
	}
	
	if AppConfig.JWTSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}
}

func getEnv(key string, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}
