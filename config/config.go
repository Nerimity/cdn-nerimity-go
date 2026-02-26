package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                string
	ExternalEmbedSecret string
	JwtSecret           string
	InternalSecret      string
}

func LoadConfig() *Config {
	_ = godotenv.Load()

	config := &Config{
		Port:                getEnv("PORT", "3000"),
		ExternalEmbedSecret: getEnv("EXTERNAL_EMBED_SECRET", ""),
		JwtSecret:           getEnv("JWT_SECRET", ""),
		InternalSecret:      getEnv("INTERNAL_SECRET", ""),
	}

	if config.ExternalEmbedSecret == "" {
		println("EXTERNAL_EMBED_SECRET is empty. Please set it in the .env file.")
	}

	if config.JwtSecret == "" {
		println("JWT_SECRET is empty. Please set it in the .env file.")
	}
	if config.InternalSecret == "" {
		println("INTERNAL_SECRET is empty. Please set it in the .env file.")
	}

	return config
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
