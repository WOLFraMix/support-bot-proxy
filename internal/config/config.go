package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port              string
	YandexAPIKey      string
	YandexFolderID    string
	PrometheusEnabled bool
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
}

func Load() (*Config, error) {
	port := getEnv("PORT", "8080")
	yandexAPIKey := os.Getenv("YANDEX_API_KEY")
	yandexFolderID := os.Getenv("YANDEX_FOLDER_ID")

	if yandexAPIKey == "" || yandexFolderID == "" {
		return nil, fmt.Errorf("YANDEX_API_KEY и YANDEX_FOLDER_ID обязательны")
	}

	prometheusEnabled, _ := strconv.ParseBool(getEnv("PROMETHEUS_ENABLED", "true"))

	readTimeout, _ := time.ParseDuration(getEnv("READ_TIMEOUT", "10s"))
	writeTimeout, _ := time.ParseDuration(getEnv("WRITE_TIMEOUT", "10s"))

	return &Config{
		Port:              port,
		YandexAPIKey:      yandexAPIKey,
		YandexFolderID:    yandexFolderID,
		PrometheusEnabled: prometheusEnabled,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
	}, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}
