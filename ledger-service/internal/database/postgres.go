// Package database owns PostgreSQL connection-pool construction and lifecycle.
package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/juliotinti/valorio-app/valorio/ledger-service/internal/config"
)

// Open creates a configured pool and waits for PostgreSQL to accept a connection.
func Open(ctx context.Context, cfg config.Config, logger *slog.Logger) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}

	poolConfig.MaxConns = int32(cfg.DBMaxConnections)
	poolConfig.MinConns = int32(cfg.DBMinConnections)
	poolConfig.MaxConnLifetime = cfg.DBMaxConnectionLifetime
	poolConfig.MaxConnIdleTime = cfg.DBMaxConnectionIdleTime
	poolConfig.HealthCheckPeriod = cfg.DBHealthCheckPeriod
	poolConfig.ConnConfig.ConnectTimeout = cfg.DBConnectTimeout

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create PostgreSQL pool: %w", err)
	}

	err = pingWithRetry(
		ctx,
		cfg.DBConnectMaxAttempts,
		cfg.DBConnectTimeout,
		cfg.DBConnectRetryDelay,
		pool.Ping,
		logger,
	)
	if err != nil {
		pool.Close()
		return nil, err
	}

	logger.Info("PostgreSQL connection pool ready", "max_connections", cfg.DBMaxConnections)
	return pool, nil
}

type pingFunc func(context.Context) error

func pingWithRetry(
	ctx context.Context,
	maxAttempts int,
	attemptTimeout time.Duration,
	retryDelay time.Duration,
	ping pingFunc,
	logger *slog.Logger,
) error {
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		attemptCtx, cancel := context.WithTimeout(ctx, attemptTimeout)
		lastErr = ping(attemptCtx)
		cancel()
		if lastErr == nil {
			return nil
		}

		if attempt == maxAttempts {
			break
		}
		logger.Warn("PostgreSQL connection unavailable; retrying", "attempt", attempt, "max_attempts", maxAttempts)

		timer := time.NewTimer(retryDelay)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return fmt.Errorf("wait for PostgreSQL: %w", ctx.Err())
		case <-timer.C:
		}
	}

	return fmt.Errorf("PostgreSQL unavailable after %d attempts: %w", maxAttempts, lastErr)
}
