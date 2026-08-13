package workerpool

import (
	"context"

	"github.com/fagbenjaenoch/dorms-ng/internal/logger"
	"github.com/nats-io/nats.go/jetstream"
)

func Start(ctx context.Context, njs *NATSJetStream) error {
	searchWorkers, err := SetupSearchWorkers(ctx, njs)
	if err != nil {
		return err
	}

	logger := logger.GetGlobalLogger()

	go func() {
		err = searchWorkers.Run(ctx, func(ctx context.Context, msg jetstream.Msg) error {
			logger.Info().Str("subject", msg.Subject()).Str("msg", string(msg.Data())).Msg("received a stream message")
			return nil
		})

	}()

	return nil
}
