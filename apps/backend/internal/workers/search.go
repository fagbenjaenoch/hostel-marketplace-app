package workerpool

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fagbenjaenoch/dorms-ng/internal/config"
	"github.com/fagbenjaenoch/dorms-ng/internal/logger"
)

func SetupSearchWorkers(ctx context.Context, njs *NATSJetStream) (*Pool, error) {
	searchStream, err := njs.CreateStream(ctx, "SEARCH", []string{"search.>", "hostel.create", "institution.create"})
	if err != nil {
		return nil, fmt.Errorf("failed to create stream: %s", err.Error())
	}

	searchConsumer, err := njs.CreateConsumer(ctx, searchStream.CachedInfo().Config.Name, "search-processor")
	if err != nil {
		return nil, errors.New("failed to create jetstream consumer")
	}

	cfg := config.GetGlobalConfig()
	logger := logger.GetGlobalLogger()

	searchWorkerConfig := &Config{
		Concurrency:      int(cfg.Workers.Concurrency),
		FetchBatchSize:   int(cfg.Workers.FetchBatchSize),
		FetchMaxWait:     5 * time.Second,
		MaxAckPending:    int(cfg.Workers.MaxAckPending),
		AckWait:          time.Duration(cfg.Workers.AckWait) * time.Second,
		MaxRetries:       int(cfg.Workers.MaxRetries),
		RetryDelay:       time.Duration(cfg.Workers.RetryDelay) * time.Second,
		RetryMaxDuration: 30 * time.Second,
		DLQSubject:       "search_unprocessed",
	}
	searchWorkers, err := NewWorkerPool(ctx, logger, &searchStream, &searchConsumer, searchWorkerConfig)
	if err != nil {
		return nil, errors.New("failed to create worker pool")
	}

	return searchWorkers, nil

}
