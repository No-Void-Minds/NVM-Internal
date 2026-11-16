package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv       string
	HTTPAddr     string
	DatabaseURL  string
	JWTSecret    string
	CookieSecure bool
	// CookieDomain kosong berarti cookie hanya untuk host ini (dev). Di produksi diisi domain induk supaya
	// sesi juga terkirim ke subdomain dashboard dan studio yang dijaga forward_auth.
	CookieDomain string
	StorageDir   string
	// TrustedProxies: CIDR yang boleh mengisi X-Forwarded-For dan X-Real-IP. Kosong berarti tidak ada.
	TrustedProxies []string
}

// Load membaca .env bila ada (dev). Di produksi file itu tidak ada dan nilainya datang dari Docker Compose.
func Load() (Config, error) {
	_ = godotenv.Load()

	c := Config{
		AppEnv:         getenv("APP_ENV", "development"),
		HTTPAddr:       getenv("HTTP_ADDR", ":8080"),
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		JWTSecret:      os.Getenv("JWT_SECRET"),
		StorageDir:     getenv("STORAGE_DIR", "./storage"),
		CookieDomain:   os.Getenv("COOKIE_DOMAIN"),
		TrustedProxies: splitList(os.Getenv("TRUSTED_PROXIES")),
	}

	secure, err := strconv.ParseBool(getenv("COOKIE_SECURE", "false"))
	if err != nil {
		return Config{}, fmt.Errorf("COOKIE_SECURE harus true atau false: %w", err)
	}
	c.CookieSecure = secure
	// Sistem publik lewat HTTPS (revisi 5): cookie sesi tanpa Secure di produksi bisa terkirim lewat HTTP biasa.
	if c.AppEnv == "production" && !c.CookieSecure {
		return Config{}, errors.New("COOKIE_SECURE wajib true di produksi")
	}

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

// splitList membaca daftar dipisah koma; nilai kosong menghasilkan nil, yang bagi Gin berarti tidak memercayai siapa pun.
func splitList(v string) []string {
	var out []string
	for _, s := range strings.Split(v, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}
