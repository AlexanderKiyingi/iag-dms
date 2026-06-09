package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	platformotel "github.com/alvor-technologies/iag-platform-go/otel"
	platformserviceauth "github.com/alvor-technologies/iag-platform-go/serviceauth"
	"github.com/jackc/pgx/v5/pgxpool"

	dmsdb "github.com/iag/dms/backend/db"
	"github.com/iag/dms/backend/internal/config"
	"github.com/iag/dms/backend/internal/consumer"
	"github.com/iag/dms/backend/internal/financeclient"
	"github.com/iag/dms/backend/internal/db"
	"github.com/iag/dms/backend/internal/events"
	"github.com/iag/dms/backend/internal/migrate"
	"github.com/iag/dms/backend/internal/middleware"
	"github.com/iag/dms/backend/internal/models"
	"github.com/iag/dms/backend/internal/outbox"
	"github.com/iag/dms/backend/internal/platformauth"
	"github.com/iag/dms/backend/internal/router"
	"github.com/iag/dms/backend/internal/seed"
	"github.com/iag/dms/backend/internal/store"
)

func main() {
	configureLogger()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}

	ctx, cancelApp := context.WithCancel(context.Background())
	defer cancelApp()

	tp, err := platformotel.Init(ctx, platformotel.Config{
		ServiceName: cfg.ServiceName,
		Environment: cfg.Environment,
	})
	if err != nil {
		slog.Warn("otel disabled", "err", err)
	} else {
		defer func() {
			shutdownCtx, c := context.WithTimeout(context.Background(), 5*time.Second)
			defer c()
			_ = tp.Shutdown(shutdownCtx)
		}()
	}

	var pool *pgxpool.Pool
	if !cfg.UseMemoryStore {
		connectCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		pool, err = db.Connect(connectCtx, cfg.DatabaseURL)
		cancel()
		if err != nil {
			slog.Error("connect postgres", "err", err)
			os.Exit(1)
		}
		defer pool.Close()

		if cfg.AutoMigrate {
			if err := autoMigrate(context.Background(), pool); err != nil {
				slog.Error("auto-migrate", "err", err)
				os.Exit(1)
			}
		}
		if cfg.SeedOnEmpty {
			if err := seed.Run(context.Background(), pool); err != nil {
				slog.Error("seed", "err", err)
				os.Exit(1)
			}
		}
	} else {
		slog.Warn("STORE_MODE=memory — in-memory data only")
	}

	verifier := platformauth.NewVerifier(cfg.JWKSURL, cfg.JWTIssuer, cfg.Audience)
	jwksCtx, jwksCancel := context.WithTimeout(ctx, 10*time.Second)
	if err := verifier.Refresh(jwksCtx); err != nil {
		jwksCancel()
		slog.Error("jwks refresh", "err", err)
		os.Exit(1)
	}
	jwksCancel()
	verifier.StartRefreshLoop(ctx, 15*time.Minute)

	if cfg.ServiceClientSecret != "" {
		go registerPermissionsLoop(ctx, cfg)
	}

	platformAuth := middleware.NewPlatformAuth(verifier)
	eventBus := events.New(events.Config{Brokers: cfg.KafkaBrokers, Enabled: cfg.EventBusEnabled})
	defer eventBus.Close()

	repo := store.New(pool)
	if pool != nil && eventBus.Enabled() {
		outboxStore := outbox.NewStore(pool)
		eventBus.SetOutbox(outboxStore)
		outboxPublisher := outbox.NewPublisher(outboxStore, outboxDispatcher{bus: eventBus})
		go outboxPublisher.Run(ctx)
		slog.Info("outbox publisher started")
	}

	if cfg.ConsumerEnabled && len(cfg.KafkaBrokers) > 0 && pool != nil {
		commercial := consumer.NewCommercial(consumer.Config{
			Brokers: cfg.KafkaBrokers,
			GroupID: cfg.ConsumerGroupID,
			Topic:   cfg.ConsumerTopic,
		}, repo)
		go func() {
			if err := commercial.Run(ctx); err != nil {
				slog.Warn("commercial consumer stopped", "err", err)
			}
		}()
		defer commercial.Close()
	}
	if cfg.OperationsConsumerEnabled && len(cfg.KafkaBrokers) > 0 && pool != nil {
		ops := consumer.NewOperations(consumer.OperationsConfig{
			Brokers: cfg.KafkaBrokers,
			GroupID: cfg.OperationsConsumerGroupID,
			Topic:   cfg.OperationsConsumerTopic,
		}, repo)
		go func() {
			if err := ops.Run(ctx); err != nil {
				slog.Warn("operations consumer stopped", "err", err)
			}
		}()
		defer ops.Close()
	}

	financeClient := financeclient.New(financeclient.Config{
		BaseURL:         cfg.FinanceURL,
		TokenURL:        cfg.AuthTokenURL,
		ServiceClientID: cfg.ServiceClientID,
		ServiceSecret:   cfg.ServiceClientSecret,
	})

	engine := router.New(router.Options{
		Cfg:          cfg,
		Repo:         repo,
		PlatformAuth: platformAuth,
		Events:       eventBus,
		Finance:      financeClient,
	})

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       120 * time.Second,
	}

	listenErr := make(chan error, 1)
	go func() {
		slog.Info("DMS API listening",
			"addr", cfg.Addr,
			"audience", cfg.Audience,
			"gatewayPrefix", cfg.GatewayAPIPrefix,
			"store", map[bool]string{true: "memory", false: "postgres"}[cfg.UseMemoryStore],
		)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			listenErr <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-stop:
		slog.Info("shutdown", "signal", sig.String())
	case err := <-listenErr:
		slog.Error("listener died", "err", err)
		os.Exit(1)
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelShutdown()
	_ = srv.Shutdown(shutdownCtx)
	cancelApp()
}

type outboxDispatcher struct {
	bus *events.Bus
}

func (d outboxDispatcher) DispatchOutbox(ctx context.Context, row outbox.Row) error {
	if d.bus == nil {
		return nil
	}
	return d.bus.DispatchOutbox(ctx, row.EventType, row.EventKey, row.Payload)
}

func configureLogger() {
	level := slog.LevelInfo
	if strings.EqualFold(os.Getenv("LOG_LEVEL"), "debug") {
		level = slog.LevelDebug
	}
	var handler slog.Handler
	if strings.ToLower(os.Getenv("LOG_FORMAT")) == "json" {
		handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	} else {
		handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	}
	slog.SetDefault(slog.New(handler))
}

func autoMigrate(parent context.Context, pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(parent, 2*time.Minute)
	defer cancel()
	applied, err := migrate.Up(ctx, pool, dmsdb.Migrations())
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	if len(applied) == 0 {
		slog.Info("schema already up to date")
	} else {
		slog.Info("migrations applied", "versions", applied)
	}
	return nil
}

func registerPermissionsLoop(ctx context.Context, cfg config.Config) {
	saClient := platformserviceauth.NewClient(platformserviceauth.Options{
		TokenURL:     cfg.AuthTokenURL,
		ClientID:     cfg.ServiceClientID,
		ClientSecret: cfg.ServiceClientSecret,
		Audience:     "iag.authentication",
	})
	descriptors := models.PermissionDescriptors()
	perms := make([]platformserviceauth.Permission, 0, len(descriptors))
	for _, d := range descriptors {
		perms = append(perms, platformserviceauth.Permission{Name: d.Name, Description: d.Description})
	}
	backoff := time.Second
	for {
		regCtx, c := context.WithTimeout(ctx, 10*time.Second)
		err := platformserviceauth.RegisterPermissions(regCtx, saClient, cfg.JWTIssuer, "dms", perms)
		c()
		if err == nil {
			slog.Info("permissions registered", "count", len(perms))
			return
		}
		slog.Warn("permissions register failed", "err", err, "retry_in", backoff)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 5*time.Minute {
			backoff *= 2
		}
	}
}
