
package config

import (
	"os"
	"strings"
)

type Config struct {
	Port           string
	DatabaseURL    string
	AllowedOrigins []string
}

func Load() Config {
	return Config{
		Port:        env("PORT", "8080"),
		DatabaseURL: env("DATABASE_URL", ""),
		AllowedOrigins: parseOrigins(
			env("ALLOWED_ORIGIN", "http://localhost:5173"),
		),
	}
}

func env(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))

	if value != "" {
		return value
	}

	return fallback
}

func parseOrigins(value string) []string {
	var origins []string

	for _, origin := range strings.Split(value, ",") {
		origin = strings.TrimRight(strings.TrimSpace(origin), "/")

		if origin != "" {
			origins = append(origins, origin)
		}
	}

	return origins
}
