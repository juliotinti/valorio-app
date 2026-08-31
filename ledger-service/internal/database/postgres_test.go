package database

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestPingWithRetrySucceedsAfterTransientFailures(t *testing.T) {
	attempts := 0
	err := pingWithRetry(
		context.Background(),
		3,
		time.Second,
		time.Millisecond,
		func(context.Context) error {
			attempts++
			if attempts < 3 {
				return errors.New("not ready")
			}
			return nil
		},
		discardLogger(),
	)
	if err != nil {
		t.Fatalf("pingWithRetry() error = %v", err)
	}
	if attempts != 3 {
		t.Fatalf("pingWithRetry() attempts = %d, want 3", attempts)
	}
}

func TestPingWithRetryStopsAfterBoundedAttempts(t *testing.T) {
	attempts := 0
	err := pingWithRetry(
		context.Background(),
		2,
		time.Second,
		time.Millisecond,
		func(context.Context) error {
			attempts++
			return errors.New("still unavailable")
		},
		discardLogger(),
	)
	if err == nil || !strings.Contains(err.Error(), "after 2 attempts") {
		t.Fatalf("pingWithRetry() error = %v, want bounded-attempt error", err)
	}
	if attempts != 2 {
		t.Fatalf("pingWithRetry() attempts = %d, want 2", attempts)
	}
}

func TestPingWithRetryHonorsCancellationDuringDelay(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0
	err := pingWithRetry(
		ctx,
		5,
		time.Second,
		time.Hour,
		func(context.Context) error {
			attempts++
			cancel()
			return errors.New("unavailable")
		},
		discardLogger(),
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("pingWithRetry() error = %v, want context cancellation", err)
	}
	if attempts != 1 {
		t.Fatalf("pingWithRetry() attempts = %d, want 1", attempts)
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
