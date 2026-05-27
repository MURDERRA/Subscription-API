package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	Port        string
	LogLevel    string
	LogDir      string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("файл .env не найден, используются переменные окружения")
	}
	return &Config{
		DatabaseURL: mustEnv("DATABASE_URL"),
		Port:        getEnv("APP_PORT", "8000"),
		LogLevel:    getEnv("LOG_LEVEL", "INFO"),
		LogDir:      getEnv("LOG_DIR", "./logs"),
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("обязательная переменная окружения %s не задана", key)
	}
	return v
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
