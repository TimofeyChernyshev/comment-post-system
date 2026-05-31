package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/TimofeyChernyshev/comment-post-system/internal/application"
	"github.com/TimofeyChernyshev/comment-post-system/internal/infrastructure/config"
	"github.com/TimofeyChernyshev/comment-post-system/internal/infrastructure/graphql"
	"github.com/TimofeyChernyshev/comment-post-system/internal/infrastructure/graphql/graph"
	httpserver "github.com/TimofeyChernyshev/comment-post-system/internal/infrastructure/receiver/http"
	"github.com/TimofeyChernyshev/comment-post-system/internal/infrastructure/storage/memory"
	"github.com/TimofeyChernyshev/comment-post-system/internal/infrastructure/storage/postgres"
	"github.com/TimofeyChernyshev/comment-post-system/internal/infrastructure/subscription"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type App struct {
	cfg *config.Config

	httpServer *httpserver.Server

	postService     *application.PostService
	commentService  *application.CommentService
	subscriptionMgr *subscription.SubscriptionManager
}

func main() {
	if err := godotenv.Load(); err != nil {
		slog.Warn("no .env file found")
	}

	cfg, err := config.Load()
	if err != nil {
		os.Exit(1)
	}

	setLogger(cfg)

	app, err := buildApp(cfg)
	if err != nil {
		slog.Error("app build failed", "error", err)
		os.Exit(1)
	}

	if err := app.start(); err != nil {
		slog.Error("app stopped with error", "error", err)
		os.Exit(1)
	}
}

func buildApp(cfg *config.Config) (*App, error) {
	if err := runMigrations(cfg.DatabaseURL); err != nil {
		return nil, fmt.Errorf("migrations failed: %w", err)
	}

	var (
		commentRepo application.CommentRepository
		postRepo    application.PostRepository
		txManager   application.TransactionManager
	)
	switch cfg.StorageType {
	case config.MemoryStorage:
		commentRepo = memory.NewCommentRepository()
		postRepo = memory.NewPostRepository()
		txManager = memory.NewTransactionManager()
	case config.PostgresStorage:
		ctx, cancel := context.WithTimeout(context.Background(), cfg.DBConntectionTimeout)
		defer cancel()

		pool, err := initPool(ctx, cfg.DatabaseURL)
		if err != nil {
			return nil, err
		}

		commentRepo = postgres.NewCommentRepository(pool)
		postRepo = postgres.NewPostRepository(pool)
		txManager = postgres.NewTransactionManager(pool)
	}

	subMgr := subscription.NewSubscriptionManager(cfg.SubscriptionConfig.BufferSize)

	postService := application.NewPostService(txManager, postRepo, cfg.DomainLimits.MaxPostTitleLength, cfg.DomainLimits.MaxPostContentLength, cfg.DomainLimits.MaxPostPageSize)
	commentService := application.NewCommentService(txManager, postRepo, commentRepo, cfg.DomainLimits.MaxCommentLength, cfg.DomainLimits.MaxCommentPageSize)

	graphqlResolver := graph.NewResolver(postService, commentService, subMgr)

	gqlHandler := graphql.NewGraphQLHandler(
		graphqlResolver,
		cfg.GraphQLConfig.MaxComplexity,
		cfg.GraphQLConfig.QueryCacheSize,
		cfg.GraphQLConfig.APQCacheSize,
		cfg.GraphQLConfig.HandshakeTimeout,
		cfg.GraphQLConfig.KeepAlivePingInterval,
	)

	gqlHandlerWithMiddleware := graphql.LoaderMiddleware(commentService, gqlHandler, cfg.Dataloader.BatchCapacity, cfg.Dataloader.WaitTime)

	httpSrv := httpserver.New(cfg.Port, gqlHandlerWithMiddleware, cfg.HTTPServerConfig.ReadTimeout, cfg.HTTPServerConfig.WriteTimeout, cfg.HTTPServerConfig.IdleTimeout)

	return &App{
		cfg:             cfg,
		httpServer:      httpSrv,
		postService:     postService,
		commentService:  commentService,
		subscriptionMgr: subMgr,
	}, nil
}

func (a *App) start() error {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	errCh := make(chan error, 1)

	go func() {
		slog.Info("graphql server starting")
		if err := a.httpServer.Start(); err != nil {
			errCh <- err
		}
	}()

	slog.Info("app started")

	select {
	case sig := <-ctx.Done():
		slog.Info("shutdown signal received", "signal", sig)
		return a.Shutdown()

	case err := <-errCh:
		slog.Error("server error", "error", err)
		return a.Shutdown()
	}
}

func (a *App) Shutdown() error {
	slog.Info("app shutting down")

	ctx, cancel := context.WithTimeout(
		context.Background(),
		a.cfg.ShutdownTimeout,
	)
	defer cancel()

	if err := a.httpServer.Stop(ctx); err != nil {
		slog.Error("http shutdown error", "error", err)
	}

	if a.subscriptionMgr != nil {
		a.subscriptionMgr.Close()
	}

	slog.Info("app shutdown complete")
	return nil
}

func setLogger(cfg *config.Config) {
	level := slog.LevelInfo
	switch strings.ToLower(cfg.Logging.Level) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if strings.ToLower(cfg.Logging.Format) == "text" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
}

func initPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database URL: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	return pool, nil
}

func runMigrations(connString string) error {
	m, err := migrate.New(
		"file://migrations",
		connString,
	)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer func() {
		_, _ = m.Close()
	}()

	if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}
