package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultHTTPAddress       = ":8080"
	defaultLogLevel          = "info"
	defaultEnvironment       = "local"
	defaultShutdownTimeout   = 10 * time.Second
	defaultReadHeaderTimeout = 5 * time.Second
	defaultReadTimeout       = 15 * time.Second
	defaultWriteTimeout      = 15 * time.Second
	defaultIdleTimeout       = 60 * time.Second
	defaultDBMaxConnections  = 10
	defaultDBMinConnections  = 1
	defaultDBMaxLifetime     = 30 * time.Minute
	defaultDBMaxIdleTime     = 5 * time.Minute
	defaultDBHealthPeriod    = 30 * time.Second
	defaultDBConnectTimeout  = 5 * time.Second
	defaultDBConnectAttempts = 5
	defaultDBRetryDelay      = 2 * time.Second
	maxTimeout               = 5 * time.Minute
	maxPoolLifetime          = 24 * time.Hour
	maxPoolConnections       = 100
	maxConnectAttempts       = 20
)

// Config contains the process configuration read at startup.
type Config struct {
	HTTPAddress             string
	LogLevel                string
	Environment             string
	ShutdownTimeout         time.Duration
	ServiceName             string
	ServiceVersion          string
	ReadHeaderTimeout       time.Duration
	ReadTimeout             time.Duration
	WriteTimeout            time.Duration
	IdleTimeout             time.Duration
	DatabaseURL             string
	DBMaxConnections        int
	DBMinConnections        int
	DBMaxConnectionLifetime time.Duration
	DBMaxConnectionIdleTime time.Duration
	DBHealthCheckPeriod     time.Duration
	DBConnectTimeout        time.Duration
	DBConnectMaxAttempts    int
	DBConnectRetryDelay     time.Duration
}

// Load reads configuration from the process environment and validates it.
func Load() (Config, error) {
	return load(os.LookupEnv)
}

type lookupEnv func(string) (string, bool)

func load(lookup lookupEnv) (Config, error) {
	var errs []error

	cfg := Config{
		HTTPAddress:             optional(lookup, "HTTP_ADDRESS", defaultHTTPAddress),
		LogLevel:                strings.ToLower(optional(lookup, "LOG_LEVEL", defaultLogLevel)),
		Environment:             strings.ToLower(optional(lookup, "ENVIRONMENT", defaultEnvironment)),
		ServiceName:             required(lookup, "SERVICE_NAME", &errs),
		ServiceVersion:          required(lookup, "SERVICE_VERSION", &errs),
		ShutdownTimeout:         duration(lookup, "SHUTDOWN_TIMEOUT", defaultShutdownTimeout, &errs),
		ReadHeaderTimeout:       duration(lookup, "HTTP_READ_HEADER_TIMEOUT", defaultReadHeaderTimeout, &errs),
		ReadTimeout:             duration(lookup, "HTTP_READ_TIMEOUT", defaultReadTimeout, &errs),
		WriteTimeout:            duration(lookup, "HTTP_WRITE_TIMEOUT", defaultWriteTimeout, &errs),
		IdleTimeout:             duration(lookup, "HTTP_IDLE_TIMEOUT", defaultIdleTimeout, &errs),
		DatabaseURL:             required(lookup, "DATABASE_URL", &errs),
		DBMaxConnections:        integer(lookup, "DB_MAX_CONNECTIONS", defaultDBMaxConnections, &errs),
		DBMinConnections:        integer(lookup, "DB_MIN_CONNECTIONS", defaultDBMinConnections, &errs),
		DBMaxConnectionLifetime: duration(lookup, "DB_MAX_CONNECTION_LIFETIME", defaultDBMaxLifetime, &errs),
		DBMaxConnectionIdleTime: duration(lookup, "DB_MAX_CONNECTION_IDLE_TIME", defaultDBMaxIdleTime, &errs),
		DBHealthCheckPeriod:     duration(lookup, "DB_HEALTH_CHECK_PERIOD", defaultDBHealthPeriod, &errs),
		DBConnectTimeout:        duration(lookup, "DB_CONNECT_TIMEOUT", defaultDBConnectTimeout, &errs),
		DBConnectMaxAttempts:    integer(lookup, "DB_CONNECT_MAX_ATTEMPTS", defaultDBConnectAttempts, &errs),
		DBConnectRetryDelay:     duration(lookup, "DB_CONNECT_RETRY_DELAY", defaultDBRetryDelay, &errs),
	}

	validateAddress(cfg.HTTPAddress, &errs)
	validateChoice("LOG_LEVEL", cfg.LogLevel, []string{"debug", "info", "warn", "error"}, &errs)
	validateChoice("ENVIRONMENT", cfg.Environment, []string{"local", "development", "test", "staging", "production"}, &errs)
	validateIdentifier("SERVICE_NAME", cfg.ServiceName, &errs)
	validateIdentifier("SERVICE_VERSION", cfg.ServiceVersion, &errs)
	validateTimeout("SHUTDOWN_TIMEOUT", cfg.ShutdownTimeout, &errs)
	validateTimeout("HTTP_READ_HEADER_TIMEOUT", cfg.ReadHeaderTimeout, &errs)
	validateTimeout("HTTP_READ_TIMEOUT", cfg.ReadTimeout, &errs)
	validateTimeout("HTTP_WRITE_TIMEOUT", cfg.WriteTimeout, &errs)
	validateTimeout("HTTP_IDLE_TIMEOUT", cfg.IdleTimeout, &errs)
	validateDatabaseURL(cfg.DatabaseURL, &errs)
	validateRange("DB_MAX_CONNECTIONS", cfg.DBMaxConnections, 1, maxPoolConnections, &errs)
	validateRange("DB_MIN_CONNECTIONS", cfg.DBMinConnections, 0, maxPoolConnections, &errs)
	if cfg.DBMinConnections > cfg.DBMaxConnections {
		errs = append(errs, fmt.Errorf("DB_MIN_CONNECTIONS must not exceed DB_MAX_CONNECTIONS"))
	}
	validateDuration("DB_MAX_CONNECTION_LIFETIME", cfg.DBMaxConnectionLifetime, maxPoolLifetime, &errs)
	validateDuration("DB_MAX_CONNECTION_IDLE_TIME", cfg.DBMaxConnectionIdleTime, maxPoolLifetime, &errs)
	validateDuration("DB_HEALTH_CHECK_PERIOD", cfg.DBHealthCheckPeriod, maxTimeout, &errs)
	validateDuration("DB_CONNECT_TIMEOUT", cfg.DBConnectTimeout, maxTimeout, &errs)
	validateRange("DB_CONNECT_MAX_ATTEMPTS", cfg.DBConnectMaxAttempts, 1, maxConnectAttempts, &errs)
	validateDuration("DB_CONNECT_RETRY_DELAY", cfg.DBConnectRetryDelay, maxTimeout, &errs)

	if len(errs) > 0 {
		return Config{}, fmt.Errorf("invalid configuration: %w", errors.Join(errs...))
	}
	return cfg, nil
}

func optional(lookup lookupEnv, key, fallback string) string {
	if value, ok := lookup(key); ok {
		return strings.TrimSpace(value)
	}
	return fallback
}

func required(lookup lookupEnv, key string, errs *[]error) string {
	value, ok := lookup(key)
	value = strings.TrimSpace(value)
	if !ok || value == "" {
		*errs = append(*errs, fmt.Errorf("%s is required", key))
	}
	return value
}

func duration(lookup lookupEnv, key string, fallback time.Duration, errs *[]error) time.Duration {
	value, ok := lookup(key)
	if !ok {
		return fallback
	}
	parsed, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil {
		*errs = append(*errs, fmt.Errorf("%s must be a duration such as 10s or 1m", key))
		return 0
	}
	return parsed
}

func integer(lookup lookupEnv, key string, fallback int, errs *[]error) int {
	value, ok := lookup(key)
	if !ok {
		return fallback
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		*errs = append(*errs, fmt.Errorf("%s must be an integer", key))
		return 0
	}
	return parsed
}

func validateAddress(address string, errs *[]error) {
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		*errs = append(*errs, fmt.Errorf("HTTP_ADDRESS must be in host:port form"))
		return
	}
	portNumber, err := strconv.ParseUint(port, 10, 16)
	if err != nil || portNumber == 0 {
		*errs = append(*errs, fmt.Errorf("HTTP_ADDRESS must contain a port from 1 to 65535"))
	}
}

func validateChoice(key, value string, choices []string, errs *[]error) {
	for _, choice := range choices {
		if value == choice {
			return
		}
	}
	*errs = append(*errs, fmt.Errorf("%s must be one of %s", key, strings.Join(choices, ", ")))
}

func validateIdentifier(key, value string, errs *[]error) {
	if value == "" {
		return
	}
	if len(value) > 128 || strings.ContainsAny(value, "\r\n\t") {
		*errs = append(*errs, fmt.Errorf("%s must be a single-line identifier no longer than 128 characters", key))
	}
}

func validateTimeout(key string, value time.Duration, errs *[]error) {
	if value <= 0 || value > maxTimeout {
		*errs = append(*errs, fmt.Errorf("%s must be greater than zero and no more than %s", key, maxTimeout))
	}
}

func validateDatabaseURL(value string, errs *[]error) {
	if value == "" {
		return
	}
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") ||
		parsed.Hostname() == "" || parsed.User == nil || parsed.User.Username() == "" || strings.Trim(parsed.Path, "/") == "" {
		*errs = append(*errs, fmt.Errorf("DATABASE_URL must be a PostgreSQL URL with user, host, and database name"))
	}
}

func validateRange(key string, value, minimum, maximum int, errs *[]error) {
	if value < minimum || value > maximum {
		*errs = append(*errs, fmt.Errorf("%s must be between %d and %d", key, minimum, maximum))
	}
}

func validateDuration(key string, value, maximum time.Duration, errs *[]error) {
	if value <= 0 || value > maximum {
		*errs = append(*errs, fmt.Errorf("%s must be greater than zero and no more than %s", key, maximum))
	}
}
