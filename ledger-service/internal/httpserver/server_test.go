package httpserver

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/juliotinti/valorio-app/valorio/ledger-service/internal/config"
)

func TestNewServerAppliesConfiguredTimeouts(t *testing.T) {
	cfg := config.Config{
		HTTPAddress:       "127.0.0.1:8080",
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       3 * time.Second,
		WriteTimeout:      4 * time.Second,
		IdleTimeout:       5 * time.Second,
	}
	handler := http.NewServeMux()

	server := newServer(cfg, handler)

	if server.Addr != cfg.HTTPAddress || server.Handler != handler {
		t.Fatalf("newServer() did not apply address or handler: %+v", server)
	}
	if server.ReadHeaderTimeout != cfg.ReadHeaderTimeout || server.ReadTimeout != cfg.ReadTimeout ||
		server.WriteTimeout != cfg.WriteTimeout || server.IdleTimeout != cfg.IdleTimeout {
		t.Fatalf("newServer() did not apply timeouts: %+v", server)
	}
}

func TestServeWaitsForInFlightRequestDuringShutdown(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}

	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})
	handler := http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		close(requestStarted)
		<-releaseRequest
		response.WriteHeader(http.StatusNoContent)
	})
	server := &http.Server{Handler: handler, ReadHeaderTimeout: time.Second}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx, cancel := context.WithCancel(context.Background())
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- serve(ctx, server, listener, 2*time.Second, logger)
	}()

	requestDone := make(chan error, 1)
	go func() {
		response, requestErr := http.Get("http://" + listener.Addr().String())
		if requestErr == nil {
			requestErr = response.Body.Close()
		}
		requestDone <- requestErr
	}()

	select {
	case <-requestStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("request did not reach handler")
	}

	cancel()
	select {
	case err := <-serveDone:
		t.Fatalf("serve() returned before in-flight request completed: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	close(releaseRequest)
	if err := receiveWithin(t, requestDone); err != nil {
		t.Fatalf("HTTP request error = %v", err)
	}
	if err := receiveWithin(t, serveDone); err != nil {
		t.Fatalf("serve() error = %v", err)
	}
}

func receiveWithin(t *testing.T, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for operation")
		return nil
	}
}
