package server

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/fagbenjaenoch/dorms-ng/internal/config"
	workerpool "github.com/fagbenjaenoch/dorms-ng/internal/workers"
	"github.com/rs/zerolog"
)

type Server struct {
	Config     *config.Config
	httpServer *http.Server
	Logger     *zerolog.Logger
	DB         *sql.DB
	NJS        *workerpool.NATSJetStream
}

func New(config *config.Config, db *sql.DB, logger *zerolog.Logger, njs *workerpool.NATSJetStream) (*Server, error) {
	return &Server{
		Config: config,
		Logger: logger,
		DB:     db,
		NJS:    njs,
	}, nil
}

func (s *Server) SetupHttpServer(handler http.Handler) {
	s.httpServer = &http.Server{
		Addr:         ":" + s.Config.Server.Port,
		Handler:      handler,
		ReadTimeout:  time.Duration(s.Config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(s.Config.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(s.Config.Server.IdleTimeout) * time.Second,
	}
}

func (s *Server) Run() error {
	if s.httpServer == nil {
		return errors.New("HTTP server has not been initialized, run server.SetupHttpServer(handler) to initialize")
	}

	s.Logger.Info().
		Str("port", s.Config.Server.Port).
		Msg("starting server")

	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.Logger.Err(err).Msg("failed to shutdown HTTP server")
		return err
	}

	if err := s.DB.Close(); err != nil {
		s.Logger.Err(err).Msg("failed to close database")
		return err
	}

	return nil
}
