package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/vadimbarashkov/medods-test-task/internal/config"
	"github.com/vadimbarashkov/medods-test-task/internal/service"
	"github.com/vadimbarashkov/medods-test-task/pkg/notifier"
	"github.com/vadimbarashkov/medods-test-task/pkg/postgres"

	api "github.com/vadimbarashkov/medods-test-task/internal/http"
	repository "github.com/vadimbarashkov/medods-test-task/internal/repository/postgres"
)

var configPath string

func setupLogger(env string) {
	level := slog.LevelDebug
	if env == config.EnvProd {
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if env == config.EnvProd || env == config.EnvTest {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler).With(slog.String("env", env))
	slog.SetDefault(logger)
}

func main() {
	flag.StringVar(&configPath, "configPath", "config.yml", "Path to config file")
	flag.Parse()

	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Loading config: %v\n", err)
		os.Exit(1)
	}

	setupLogger(cfg.Env)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	slog.Info("connecting to the database")
	pool, err := postgres.New(ctx, cfg.Postgres.DSN())
	if err != nil {
		slog.Error("failed to connect to the database", slog.Any("err", err))
		os.Exit(1)
	}
	defer pool.Close()

	slog.Info("applying database migrations")
	if err := postgres.ApplyMigrations(
		ctx,
		cfg.Postgres.DSN(),
		cfg.Postgres.MigrationsPath,
	); err != nil {
		slog.Error("failed to apply database migrations", slog.Any("err", err))
		os.Exit(1)
	}

	authService := service.NewAuthService(service.AuthServiceParams{
		AccessTokenSecret:  []byte(cfg.Tokens.AccessSecret),
		AccessTokenTTL:     cfg.Tokens.AccessTTL,
		RefreshTokenSecret: []byte(cfg.Tokens.RefreshSecret),
		RefreshTokenTTL:    cfg.Tokens.RefreshTTL,
		RefreshTokenRepo:   repository.NewRefreshTokenRepository(pool),
		EmailNotifier:      notifier.NewStubEmailNotifier(),
	})

	router := api.NewRouter(slog.Default(), authService)

	server := &http.Server{
		Addr:           cfg.Server.Addr(),
		Handler:        router,
		ReadTimeout:    cfg.Server.ReadTimeout,
		WriteTimeout:   cfg.Server.WriteTimeout,
		IdleTimeout:    cfg.Server.IdleTimeout,
		MaxHeaderBytes: cfg.Server.MaxHeaderBytes,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		slog.Info("starting the server", slog.String("addr", cfg.Server.Addr()))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", slog.Any("err", err))
			os.Exit(1)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()

		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		slog.Info("shutting down the server")
		if err := server.Shutdown(ctx); err != nil {
			slog.Error("failed to shutdown the server", slog.Any("err", err))
			os.Exit(1)
		}
	}()

	wg.Wait()
	slog.Info("server stopped gracefully")
}
