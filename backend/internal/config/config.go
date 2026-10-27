package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv       string
	HTTPAddr     string
	DatabaseURL  string
	JWTSecret    string
	CookieSecure bool
	StorageDir   string
}

// Load membaca .env bila ada (dev). Di produksi file itu tidak ada dan nilainya datang dari Docker Compose.
func Load() (Config, error) {
	_ = godotenv.Load()

	c := Config{
		AppEnv:      getenv("APP_ENV", "development"),
		HTTPAddr:    getenv("HTTP_ADDR", ":8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		StorageDir:  getenv("STORAGE_DIR", "./storage"),
	}

	secure, err := strconv.ParseBool(getenv("COOKIE_SECURE", "false"))
	if err != nil {
		return Config{}, fmt.Errorf("COOKIE_SECURE harus true atau false: %w", err)
	}
	c.CookieSecure = secure

	if c.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL wajib diisi")
	}
	return c, nil
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}