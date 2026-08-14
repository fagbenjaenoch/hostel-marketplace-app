package workerpool

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fagbenjaenoch/dorms-ng/internal/config"
	"github.com/fagbenjaenoch/dorms-ng/internal/logger"
	"github.com/fagbenjaenoch/dorms-ng/internal/secrets"
	infisical "github.com/infisical/go-sdk"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog"
)

func connectNATS(ctx context.Context, config *config.Config) (*nats.Conn, error) {
	secretsClient := secrets.GetSecretClient()
	if secretsClient == nil {
		return nil, errors.New("secrets client is nil")
	}

	natsUserJWT, err := secretsClient.Secrets().Retrieve(infisical.RetrieveSecretOptions{
		SecretKey:   "NATS_USER_JWT",
		Environment: config.Primary.Env,
		ProjectID:   config.Infisical.ProjectID,
		SecretPath:  config.Infisical.NATSSecretPath,
	})
	if err != nil {
		return nil, errors.New("failed to retrieve nats credentials: " + err.Error())
	}

	natsUserSeed, err := secretsClient.Secrets().Retrieve(infisical.RetrieveSecretOptions{
		SecretKey:   "NATS_USER_SEED",
		Environment: config.Primary.Env,
		ProjectID:   config.Infisical.ProjectID,
		SecretPath:  config.Infisical.NATSSecretPath,
	})
	if err != nil {
		return nil, errors.New("failed to retrieve nats credentials: " + err.Error())
	}

	logger := logger.GetGlobalLogger()

	nc, _ := nats.Connect(config.NATS.URL,
		nats.UserJWTAndSeed(natsUserJWT.SecretValue, natsUserSeed.SecretValue),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			logger.Error().Msg("nats disconnected")
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			logger.Info().Msg("nats reconnected")
		}),
		nats.Name(fmt.Sprintf("%s-%s", config.Observability.AppName, config.Primary.Env)),
	)
	return nc, nil
}

func setupJetStream(nc *nats.Conn) (jetstream.JetStream, error) {
	js, err := jetstream.New(nc)
	if err != nil {
		return nil, errors.New("failed to create jetstream: " + err.Error())
	}
	return js, nil
}

type NATSJetStream struct {
	nc *nats.Conn
	js jetstream.JetStream
}

var globalNATSJetstream *NATSJetStream

func SetupNATSJetStream(ctx context.Context, config *config.Config) (*NATSJetStream, error) {
	nc, err := connectNATS(ctx, config)
	if err != nil {
		return nil, err
	}

	js, err := setupJetStream(nc)
	if err != nil {
		return nil, err
	}

	globalNATSJetstream = &NATSJetStream{nc: nc, js: js}
	return globalNATSJetstream, nil
}

func (ns *NATSJetStream) CreateStream(ctx context.Context, name string, subjects []string) (jetstream.Stream, error) {
	stream, err := ns.js.CreateStream(ctx, jetstream.StreamConfig{
		Name:     "SEARCH",
		Subjects: subjects,
		Storage:  jetstream.FileStorage,
		MaxBytes: 100 * 1024 * 1024,
	})
	if err != nil {
		return nil, err
	}

	return stream, nil
}

func (ns *NATSJetStream) CreateConsumer(ctx context.Context, stream, consumerName string) (jetstream.Consumer, error) {
	consumer, err := ns.js.CreateOrUpdateConsumer(ctx, stream, jetstream.ConsumerConfig{
		Durable:   consumerName,
		AckPolicy: jetstream.AckExplicitPolicy,
	})
	if err != nil {
		return nil, err
	}

	return consumer, nil
}

type Event interface {
	Type() string
	Payload() []byte
}

func (njs *NATSJetStream) PublishMessage(ctx context.Context, logger *zerolog.Logger, event Event) error {
	ack, err := njs.js.Publish(ctx, event.Type(), event.Payload())
	if err != nil {
		return err
	}

	logger.Info().Msgf("message stored in %s at sequence %d", ack.Stream, ack.Sequence)
	return nil
}

func GetGlobalNatsJetstreamConnection() (*NATSJetStream, error) {
	if globalNATSJetstream != nil {
		return globalNATSJetstream, nil
	}

	return nil, errors.New("nats jetstream connection has not been initiated")
}
