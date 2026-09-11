package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("APP_ORIGIN", "")
	t.Setenv("SESSION_TTL", "")
	t.Setenv("SESSION_CLEANUP_INTERVAL", "")
	t.Setenv("SECURE_COOKIES", "")

	config, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if config.Address != ":8080" || config.AllowedOrigin != "http://localhost:3000" {
		t.Fatalf("unexpected defaults: %+v", config)
	}
	if config.SessionTTL != 30*time.Minute || config.CleanupInterval != time.Minute {
		t.Fatalf("unexpected durations: %+v", config)
	}
}

func TestLoadRejectsInvalidDuration(t *testing.T) {
	t.Setenv("SESSION_TTL", "forever")
	if _, err := Load(); err == nil {
		t.Fatal("Load() accepted an invalid duration")
	}
}
