package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                string
	ExternalEmbedSecret string
}

func LoadConfig() *Config {
	_ = godotenv.Load()

	return &Config{
		Port:                getEnv("PORT", "3000"),
		ExternalEmbedSecret: getEnv("EXTERNAL_EMBED_SECRET", ""),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
