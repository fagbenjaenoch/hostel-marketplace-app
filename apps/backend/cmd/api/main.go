package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/fagbenjaenoch/dorms-ng/internal/config"
	"github.com/fagbenjaenoch/dorms-ng/internal/database"
	"github.com/fagbenjaenoch/dorms-ng/internal/logger"
	"github.com/fagbenjaenoch/dorms-ng/internal/observability"
	"github.com/fagbenjaenoch/dorms-ng/internal/routes"
	"github.com/fagbenjaenoch/dorms-ng/internal/secrets"
	"github.com/fagbenjaenoch/dorms-ng/internal/server"
	workerpool "github.com/fagbenjaenoch/dorms-ng/internal/workers"
)

const DefaultContextTimeout = 10

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	logger := logger.New(cfg)

	err = secrets.SetupSecretsManager(cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to setup secrets manager")
	}

	db, reg, err := database.Initialize(cfg, &logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer func() {
		_ = reg.Unregister() // unregister observability at the db level
	}()

	srv, err := server.New(cfg, db, &logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create server")
	}

	obs := observability.NewObservability(srv)

	// setup log, metrics and trace telemetry
	shutdownFns, err := obs.SetupObservability()
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize observability")
	}
	defer func() {
		shutdownCtx := context.Background()

		var shutdownErr error
		for _, fn := range shutdownFns {
			if err := fn(shutdownCtx); err != nil {
				shutdownErr = errors.Join(shutdownErr, err)
			}
		}

		if shutdownErr != nil {
			logger.Err(shutdownErr).Msg("failed to shutdown observability")
		}
	}()

	srv.SetupHttpServer(routes.New(srv))

	go func() {
		if err := srv.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal().Err(err).Msg("failed to start server")
		}
	}()

	// workers setup
	workerCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	njs, err := workerpool.SetupNATSJetStream(workerCtx, cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to setup NATS JetStream")
	}
	logger.Info().Msg("NATS JetStream setup successfully")

	err = workerpool.Start(workerCtx, njs)
	if err != nil {
		logger.Error().Err(err).Msg("workers failed")
	}

	// shutdown sequence
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	<-ctx.Done()

	logger.Info().Msg("server shutting down...")

	// Doesn't block if no connections, but will otherwise wait until the timeout deadline
	ctx, cancel = context.WithTimeout(context.Background(), DefaultContextTimeout*time.Second)
	if err := srv.Shutdown(ctx); err != nil {
		logger.Err(err).Msg("failed to shutdown server")
	}
	stop()
	cancel()

	logger.Info().Msg("server exited properly")
}
