package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/ttl256/gophermart-loyalty/internal/accrual"
	"github.com/ttl256/gophermart-loyalty/internal/auth"
	"github.com/ttl256/gophermart-loyalty/internal/config"
	"github.com/ttl256/gophermart-loyalty/internal/handler"
	"github.com/ttl256/gophermart-loyalty/internal/logger"
	"github.com/ttl256/gophermart-loyalty/internal/repository"
	"github.com/ttl256/gophermart-loyalty/internal/service"
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
	logger.Initialize(cfg.LogLevel)
	logger := slog.Default()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	repo, err := repository.NewDBStorage(ctx, cfg.DSN)
	if err != nil {
		return fmt.Errorf("initializing repo: %w", err)
	}
	defer repo.Close()
	err = repo.RepoPing(ctx)
	if err != nil {
		return fmt.Errorf("pinging repo: %w", err)
	}
	err = repo.Migrate(repository.MigrationActionUp)
	if err != nil {
		return fmt.Errorf("migration: %w", err)
	}

	authSvc := service.NewAuthService(repo)
	if cfg.Secret == "" {
		cfg.Secret = rand.Text()
	}
	orderSvc := service.NewOrderService(repo)

	// h := handler.NewHTTPHandler(authSvc, cfg.Secret, 1*time.Hour)
	h := handler.HTTPHandler{
		AuthService:  authSvc,
		OrderService: orderSvc,
		JWT:          auth.NewManager(cfg.Secret, 1*time.Hour),
		Logger:       slog.Default(),
	}
	srv := &http.Server{
		Addr:         cfg.Address,
		Handler:      h.Routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second, //nolint: mnd //fine
		WriteTimeout: 30 * time.Second, //nolint: mnd //fine
	}

	const fetchAccrualFreq = 10 * time.Second
	client := accrual.NewClient(cfg.AccrualAddress)
	worker := accrual.NewWorker(repo, client, fetchAccrualFreq)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return runServer(srv, logger)
	})
	g.Go(func() error {
		return shutdownServer(ctx, srv, logger)
	})
	g.Go(func() error {
		return worker.Run(ctx)
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
