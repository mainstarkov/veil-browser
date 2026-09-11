package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Address         string
	AllowedOrigin   string
	SessionTTL      time.Duration
	CleanupInterval time.Duration
	SecureCookies   bool
}

func Load() (Config, error) {
	ttl, err := durationFromEnv("SESSION_TTL", 30*time.Minute)
	if err != nil {
		return Config{}, err
	}
	cleanup, err := durationFromEnv("SESSION_CLEANUP_INTERVAL", time.Minute)
	if err != nil {
		return Config{}, err
	}
	secureCookies, err := boolFromEnv("SECURE_COOKIES", false)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Address:         stringFromEnv("HTTP_ADDR", ":8080"),
		AllowedOrigin:   stringFromEnv("APP_ORIGIN", "http://localhost:3000"),
		SessionTTL:      ttl,
		CleanupInterval: cleanup,
		SecureCookies:   secureCookies,
	}, nil
}

func stringFromEnv(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func durationFromEnv(name string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", name)
	}
	return duration, nil
}

func boolFromEnv(name string, fallback bool) (bool, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}
	result, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s must be true or false", name)
	}
	return result, nil
}
