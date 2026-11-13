package config

import "os"

type Config struct {
	Port          string
	DatabaseURL   string
	AllowedOrigin string
}

func Load() Config {
	return Config{
		Port:          env("PORT", "8080"),
		DatabaseURL:   env("DATABASE_URL", ""),
		AllowedOrigin: env("ALLOWED_ORIGIN", "http://localhost:5173"),
	}
}

func env(key, fallback string) string {
	value := os.Getenv(key)

	if value != "" {
		return value
	}

	return fallback
}