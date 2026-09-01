package database

import (
	"database/sql"
	"embed"
	"time"

	"github.com/XSAM/otelsql"
	"github.com/fagbenjaenoch/dorms-ng/internal/config"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

//go:embed migrations/*.sql
var embedMigration embed.FS

func newDB(config *config.Config, logger *zerolog.Logger) (*sql.DB, metric.Registration, error) {
	db, err := otelsql.Open("pgx", config.DB.URI, otelsql.WithAttributes(
		semconv.DBSystemNamePostgreSQL,
		semconv.DBNamespace(config.DB.Name),
	),
	)
	if err != nil {
		return nil, nil, err
	}

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(20)
	db.SetConnMaxLifetime(10 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)

	reg, err := otelsql.RegisterDBStatsMetrics(db, otelsql.WithAttributes(
		semconv.DBSystemNamePostgreSQL,
		semconv.DBNamespace(config.DB.Name),
	))
	if err != nil {
		return nil, nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, reg, err
	}
	logger.Info().Msg("database setup successfully")

	return db, reg, nil
}

func Initialize(config *config.Config, logger *zerolog.Logger) (*sql.DB, metric.Registration, error) {
	goose.SetBaseFS(embedMigration)

	if err := goose.SetDialect("pgx"); err != nil {
		panic(err)
	}

	db, reg, err := newDB(config, logger)
	if err != nil {
		return nil, nil, err
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return nil, nil, err
	}

	logger.Info().Msg("database migrations applied successfully")

	return db, reg, nil
}

func Close(db *sql.DB) error {
	return db.Close()
}
