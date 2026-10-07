package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port             string
	DatabaseURL      string
	JWTSecret        string
	JWTTTL           time.Duration
	PBKDF2Iterations int
}

// Load reads a .env file (if present) and then the process environment.
// Real environment variables win over values in .env.
func Load() (Config, error) {
	_ = godotenv.Load() // missing .env is fine (e.g. in production)

	cfg := Config{
		Port:        getenv("PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
	}

	if cfg.DatabaseURL == "" {
		return cfg, errors.New("DATABASE_URL is required")
	}
	if len(cfg.JWTSecret) < 32 {
		return cfg, errors.New("JWT_SECRET is required and must be at least 32 characters")
	}

	ttlMin, err := strconv.Atoi(getenv("JWT_TTL_MINUTES", "60"))
	if err != nil || ttlMin <= 0 {
		return cfg, fmt.Errorf("JWT_TTL_MINUTES must be a positive integer")
	}
	cfg.JWTTTL = time.Duration(ttlMin) * time.Minute

	cfg.PBKDF2Iterations, err = strconv.Atoi(getenv("PBKDF2_ITERATIONS", "600000"))
	if err != nil || cfg.PBKDF2Iterations < 100000 {
		return cfg, fmt.Errorf("PBKDF2_ITERATIONS must be an integer >= 100000")
	}

	return cfg, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
