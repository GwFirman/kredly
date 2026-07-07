package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	Environment        string
	GroqAPIKeys        []string
	GroqBaseURL        string
	DatabaseURL        string
	APIURL             string
	FrontendURL        string
	GoogleClientID     string
	GoogleClientSecret string
	ResendAPIKey       string
}

func LoadConfig() *Config {
	// Try loading from .env, but don't fail if it doesn't exist (useful for production environments)
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: No .env file found, relying on system environment variables")
	}

	port := getEnv("PORT", "8080")
	env := getEnv("ENVIRONMENT", "development")

	// Read keys with fallback support
	rawKeys := getEnv("GROQ_API_KEYS", "")
	if rawKeys == "" {
		rawKeys = getEnv("GROQ_API_KEY", "")
	}

	var groqAPIKeys []string
	if rawKeys != "" {
		parts := strings.Split(rawKeys, ",")
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" && trimmed != "gsk_your_api_key_here" {
				groqAPIKeys = append(groqAPIKeys, trimmed)
			}
		}
	}

	groqBaseURL := getEnv("GROQ_BASE_URL", "https://api.groq.com/openai/v1")
	databaseURL := getEnv("DATABASE_URL", "mongodb://localhost:27017/kredly")
	apiURL := getEnv("API_URL", "http://localhost:8080")
	frontendURL := getEnv("FRONTEND_URL", "http://localhost:3000")
	googleClientID := getEnv("GOOGLE_CLIENT_ID", "")
	googleClientSecret := getEnv("GOOGLE_CLIENT_SECRET", "")
	resendAPIKey := getEnv("RESEND_API_KEY", "")

	if len(groqAPIKeys) == 0 {
		log.Println("Warning: GROQ_API_KEYS is not set or still set to default placeholder value. Please update it in your .env file.")
	}

	return &Config{
		Port:               port,
		Environment:        env,
		GroqAPIKeys:        groqAPIKeys,
		GroqBaseURL:        groqBaseURL,
		DatabaseURL:        databaseURL,
		APIURL:             apiURL,
		FrontendURL:        frontendURL,
		GoogleClientID:     googleClientID,
		GoogleClientSecret: googleClientSecret,
		ResendAPIKey:       resendAPIKey,
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
