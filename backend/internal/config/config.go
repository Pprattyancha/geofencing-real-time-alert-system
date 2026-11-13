package config

import "os"

type Config struct{ Port, DatabaseURL, AllowedOrigin string }

func Load() Config {
	return Config{Port: env("PORT", "8080"), DatabaseURL: env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/geofencing?sslmode=disable"), AllowedOrigin: env("ALLOWED_ORIGIN", "http://localhost:5173")}
}
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
