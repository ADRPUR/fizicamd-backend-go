package config

import (
	"os"
	"strconv"
	"strings"
)

// Config holds runtime configuration loaded from environment variables.
type Config struct {
	DatabaseURL          string
	JWTSecret            string
	JWTIssuer            string
	AccessTTLSeconds     int64
	RefreshTTLSeconds    int64
	MediaStoragePath     string
	MetricsDiskPath      string
	MetricsSampleSeconds int
	CorsOrigins          []string
}

func Load() Config {
	return Config{
		DatabaseURL:          mustEnv("DATABASE_URL"),
		JWTSecret:            mustEnv("JWT_SECRET"),
		JWTIssuer:            envOr("JWT_ISSUER", "fizicamd"),
		AccessTTLSeconds:     int64(envOrInt("ACCESS_TTL_SECONDS", 14400)),
		RefreshTTLSeconds:    int64(envOrInt("REFRESH_TTL_SECONDS", 1209600)),
		MediaStoragePath:     envOr("MEDIA_STORAGE_PATH", "storage/media"),
		MetricsDiskPath:      envOr("METRICS_DISK_PATH", "storage/media"),
		MetricsSampleSeconds: envOrInt("METRICS_SAMPLE_INTERVAL", 5),
		CorsOrigins:          parseCSV(envOr("CORS_ORIGINS", "")),
	}
}

func mustEnv(key string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		panic("missing env var: " + key)
	}
	return value
}

func envOr(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envOrInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			items = append(items, value)
		}
	}
	return items
}
