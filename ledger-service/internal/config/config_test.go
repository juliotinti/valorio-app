package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadValidConfiguration(t *testing.T) {
	env := map[string]string{
		"HTTP_ADDRESS":                "127.0.0.1:9090",
		"LOG_LEVEL":                   "DEBUG",
		"ENVIRONMENT":                 "test",
		"SHUTDOWN_TIMEOUT":            "20s",
		"SERVICE_NAME":                "ledger-service",
		"SERVICE_VERSION":             "1.2.3",
		"HTTP_READ_HEADER_TIMEOUT":    "2s",
		"HTTP_READ_TIMEOUT":           "8s",
		"HTTP_WRITE_TIMEOUT":          "9s",
		"HTTP_IDLE_TIMEOUT":           "45s",
		"DATABASE_URL":                "postgres://ledger_app:secret@ledger-postgres:5432/ledger_db?sslmode=disable",
		"DB_MAX_CONNECTIONS":          "12",
		"DB_MIN_CONNECTIONS":          "2",
		"DB_MAX_CONNECTION_LIFETIME":  "45m",
		"DB_MAX_CONNECTION_IDLE_TIME": "10m",
		"DB_HEALTH_CHECK_PERIOD":      "20s",
		"DB_CONNECT_TIMEOUT":          "3s",
		"DB_CONNECT_MAX_ATTEMPTS":     "4",
		"DB_CONNECT_RETRY_DELAY":      "500ms",
	}

	cfg, err := load(mapLookup(env))
	if err != nil {
		t.Fatalf("load() error = %v", err)
	}
	if cfg.HTTPAddress != "127.0.0.1:9090" || cfg.LogLevel != "debug" || cfg.Environment != "test" {
		t.Fatalf("load() returned unexpected string values: %+v", cfg)
	}
	if cfg.ShutdownTimeout != 20*time.Second || cfg.ReadHeaderTimeout != 2*time.Second ||
		cfg.ReadTimeout != 8*time.Second || cfg.WriteTimeout != 9*time.Second || cfg.IdleTimeout != 45*time.Second {
		t.Fatalf("load() returned unexpected durations: %+v", cfg)
	}
	if cfg.DBMaxConnections != 12 || cfg.DBMinConnections != 2 || cfg.DBConnectMaxAttempts != 4 ||
		cfg.DBMaxConnectionLifetime != 45*time.Minute || cfg.DBMaxConnectionIdleTime != 10*time.Minute ||
		cfg.DBHealthCheckPeriod != 20*time.Second || cfg.DBConnectTimeout != 3*time.Second ||
		cfg.DBConnectRetryDelay != 500*time.Millisecond {
		t.Fatalf("load() returned unexpected database values: %+v", cfg)
	}
}

func TestLoadUsesSafeOptionalDefaults(t *testing.T) {
	cfg, err := load(mapLookup(map[string]string{
		"SERVICE_NAME":    "ledger-service",
		"SERVICE_VERSION": "dev",
		"DATABASE_URL":    "postgres://ledger_app:secret@localhost:5432/ledger_db?sslmode=disable",
	}))
	if err != nil {
		t.Fatalf("load() error = %v", err)
	}

	if cfg.HTTPAddress != ":8080" || cfg.LogLevel != "info" || cfg.Environment != "local" ||
		cfg.ShutdownTimeout != 10*time.Second || cfg.ReadHeaderTimeout != 5*time.Second ||
		cfg.ReadTimeout != 15*time.Second || cfg.WriteTimeout != 15*time.Second || cfg.IdleTimeout != 60*time.Second {
		t.Fatalf("load() defaults = %+v", cfg)
	}
	if cfg.DBMaxConnections != 10 || cfg.DBMinConnections != 1 || cfg.DBConnectMaxAttempts != 5 ||
		cfg.DBMaxConnectionLifetime != 30*time.Minute || cfg.DBMaxConnectionIdleTime != 5*time.Minute ||
		cfg.DBHealthCheckPeriod != 30*time.Second || cfg.DBConnectTimeout != 5*time.Second ||
		cfg.DBConnectRetryDelay != 2*time.Second {
		t.Fatalf("load() database defaults = %+v", cfg)
	}
}

func TestLoadReportsMissingRequiredValuesWithoutValues(t *testing.T) {
	_, err := load(mapLookup(map[string]string{"SERVICE_VERSION": ""}))
	if err == nil {
		t.Fatal("load() error = nil, want missing required values")
	}
	for _, key := range []string{"SERVICE_NAME is required", "SERVICE_VERSION is required", "DATABASE_URL is required"} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("load() error = %q, want %q", err, key)
		}
	}
}

func TestLoadRejectsMalformedValuesWithoutEchoingThem(t *testing.T) {
	secretLikeValue := "not-a-duration-secret-123"
	_, err := load(mapLookup(map[string]string{
		"SERVICE_NAME":     "ledger-service",
		"SERVICE_VERSION":  "dev",
		"HTTP_ADDRESS":     "http://localhost:8080",
		"LOG_LEVEL":        "verbose",
		"ENVIRONMENT":      "somewhere",
		"SHUTDOWN_TIMEOUT": secretLikeValue,
		"DATABASE_URL":     "postgres://ledger_app:database-password@/missing-host",
	}))
	if err == nil {
		t.Fatal("load() error = nil, want malformed configuration error")
	}
	if strings.Contains(err.Error(), secretLikeValue) {
		t.Fatalf("load() error exposed configuration value: %q", err)
	}
}

func TestLoadRejectsUnsafeValues(t *testing.T) {
	_, err := load(mapLookup(map[string]string{
		"SERVICE_NAME":             "ledger-service\nforged-log-line",
		"SERVICE_VERSION":          "dev",
		"SHUTDOWN_TIMEOUT":         "0s",
		"HTTP_READ_HEADER_TIMEOUT": "10m",
		"DATABASE_URL":             "postgres://ledger_app:secret@localhost:5432/ledger_db",
		"DB_MIN_CONNECTIONS":       "11",
		"DB_MAX_CONNECTIONS":       "10",
	}))
	if err == nil {
		t.Fatal("load() error = nil, want unsafe configuration error")
	}
	for _, key := range []string{"SERVICE_NAME", "SHUTDOWN_TIMEOUT", "HTTP_READ_HEADER_TIMEOUT", "DB_MIN_CONNECTIONS"} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("load() error = %q, want reference to %s", err, key)
		}
	}
}

func mapLookup(values map[string]string) lookupEnv {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}
