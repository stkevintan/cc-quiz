package config

import (
	"os"
	"strings"
)

type Config struct {
	AppEnv             string
	HTTPAddr           string
	DBPath             string
	JWTSecret          string
	CORSAllowedOrigins []string
}

func Load() Config {
	return Config{
		AppEnv:             getEnv("APP_ENV", "development"),
		HTTPAddr:           getEnv("HTTP_ADDR", ":8080"),
		DBPath:             getEnv("DB_PATH", "data/app.db"),
		JWTSecret:          getEnv("JWT_SECRET", "local-development-secret-change-me"),
		CORSAllowedOrigins: splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173")),
	}
}

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			items = append(items, trimmed)
		}
	}
	return items
}
