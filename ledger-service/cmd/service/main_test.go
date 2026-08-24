package main

import (
	"log/slog"
	"strings"
	"testing"
)

func TestLogLevel(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  slog.Level
	}{
		{name: "debug", input: "debug", want: slog.LevelDebug},
		{name: "info", input: "info", want: slog.LevelInfo},
		{name: "warn", input: "warn", want: slog.LevelWarn},
		{name: "error", input: "error", want: slog.LevelError},
		{
			name:  "defensive fallback",
			input: "unexpected",
			want:  slog.LevelInfo,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := logLevel(test.input); got != test.want {
				t.Errorf(
					"logLevel(%q) = %v, want %v",
					test.input,
					got,
					test.want,
				)
			}
		})
	}
}

func TestRunFailsFastWhenRequiredConfigurationIsMissing(t *testing.T) {
	t.Setenv("SERVICE_NAME", "")
	t.Setenv("SERVICE_VERSION", "")

	err := run()
	if err == nil {
		t.Fatal("run() error = nil, want configuration error")
	}
	if !strings.Contains(err.Error(), "SERVICE_NAME is required") ||
		!strings.Contains(err.Error(), "SERVICE_VERSION is required") {
		t.Fatalf("run() error = %q, want missing required variable names", err)
	}
}
