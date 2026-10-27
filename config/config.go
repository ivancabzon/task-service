package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort   string
	AppEnv    string
	DBDSN     string
	RedisAddr string
	RedisDB   int
	JWTSecret string
	JWTTTL    time.Duration
}

func Load() (*Config, error) {

	_ = godotenv.Load()

	cfg := &Config{
		AppPort:   getEnv("APP_PORT", "8080"),
		AppEnv:    getEnv("APP_ENV", "development"),
		DBDSN:     os.Getenv("DB_DSN"),
		RedisAddr: getEnv("REDIS_ADDR", "localhost:6379"),
		RedisDB:   getEnvInt("REDIS_DB", 0),
		JWTSecret: os.Getenv("JWT_SECRET"),
	}

	ttlHours := getEnvInt("JWT_TTL_HOURS", 24)
	cfg.JWTTTL = time.Duration(ttlHours) * time.Hour

	if cfg.DBDSN == "" {
		return nil, fmt.Errorf("DB_DSN is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
