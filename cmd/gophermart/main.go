package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/ttl256/gophermart-loyalty/internal/config"
	"github.com/ttl256/gophermart-loyalty/internal/handler"
	"github.com/ttl256/gophermart-loyalty/internal/logger"
	"golang.org/x/sync/errgroup"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%+v", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.BuildConfig(os.Args[1:], os.Stdout)
	if err != nil {
		if errors.Is(err, arg.ErrHelp) {
			return nil
		}
		return fmt.Errorf("building config: %w", err)
	}
	err = logger.Initialize(cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("initializing logger: %w", err)
	}
	logger := slog.Default()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	h := handler.NewHTTPHandler()
	srv := &http.Server{
		Addr:         cfg.Address,
		Handler:      h.Routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second, //nolint: mnd //fine
		WriteTimeout: 30 * time.Second, //nolint: mnd //fine
	}

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return runServer(srv, logger)
	})
	g.Go(func() error {
		return shutdownServer(ctx, srv, logger)
	})
	err = g.Wait()
	if err != nil {
		return fmt.Errorf("waiting for server to shutdown: %w", err)
	}
	logger.Info("exiting")

	return nil
}

func runServer(srv *http.Server, logger *slog.Logger) error {
	logger.Info("started http server", slog.String("address", srv.Addr))
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("serving http", slog.Any("error", fmt.Sprintf("%+v", err)))
		return fmt.Errorf("serving http: %w", err)
	}
	return nil
}

func shutdownServer(ctx context.Context, srv *http.Server, logger *slog.Logger) error {
	const maxShutdownDuration = 1 * time.Minute
	<-ctx.Done()
	logger.InfoContext(ctx, "received shutdown signal")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), maxShutdownDuration)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.ErrorContext(ctx, "shutting down server", slog.Any("error", fmt.Sprintf("%+v", err)))
		return fmt.Errorf("shutting down server: %w", err)
	}
	logger.InfoContext(ctx, "server is shutdown")
	return nil
}
