package workerpool

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/fagbenjaenoch/dorms-ng/internal/utils"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog"
)

// Handler processes a single message. Returning nil acks the message.
// Returning an error naks it for redelivery (subject to MaxDeliver/backoff).
// A handler must respect ctx cancellation — the pool cancels it if
// HandlerTimeout elapses or the pool is shutting down.
type Handler func(ctx context.Context, msg jetstream.Msg) error

// Pool consumes from a JetStream pull consumer and dispatches messages to
// Handler with bounded concurrency.
type Pool struct {
	cfg      *Config
	stream   *jetstream.Stream
	consumer *jetstream.Consumer
	logger   *zerolog.Logger

	sem chan struct{}
	wg  sync.WaitGroup
}

// NewWorkerPool builds a Pool. nc must already be connected; the Pool does not own
// its lifecycle (call nc.Close() yourself after Run returns).
func NewWorkerPool(ctx context.Context, logger *zerolog.Logger, stream *jetstream.Stream, cons *jetstream.Consumer, cfg *Config) (*Pool, error) {
	utils.SetStructDefaults(cfg)
	logger.Debug().Str("concurrency", strconv.Itoa(cfg.Concurrency)).Str("batch", strconv.Itoa(cfg.FetchBatchSize)).Msg("new worker pool")
	return &Pool{
		cfg:      cfg,
		logger:   logger,
		stream:   stream,
		consumer: cons,
		sem:      make(chan struct{}, cfg.Concurrency),
	}, nil
}

// Run creates (or attaches to) the durable pull consumer and processes
// messages with handler until ctx is cancelled. It blocks until all
// in-flight handlers finish, then returns ctx.Err(). Run is not safe to
// call concurrently on the same Pool.
func (p *Pool) Run(ctx context.Context, handler Handler) error {

	p.logger.Info().
		Str("stream", (*p.stream).CachedInfo().Config.Name).
		Str("durable", (*p.consumer).CachedInfo().Name).
		Int("concurrency", p.cfg.Concurrency).
		Int("fetch_batch", p.cfg.FetchBatchSize).
		Int("max_ack_pending", p.cfg.MaxAckPending).
		Msg("workerpool starting")

	fetchErrBackoff := time.Second

fetchLoop:
	for {
		select {
		case <-ctx.Done():
			break fetchLoop
		default:
		}

		msgs, err := (*p.consumer).Fetch(p.cfg.FetchBatchSize, jetstream.FetchMaxWait(p.cfg.FetchMaxWait))
		if err != nil {
			if isBenignFetchErr(err) {
				continue
			}
			p.logger.Warn().
				Str("err", err.Error()).
				Str("backoff", fetchErrBackoff.String()).
				Msg("fetch error, backing off")

			select {
			case <-time.After(fetchErrBackoff):
				if fetchErrBackoff < p.cfg.RetryMaxDuration {
					fetchErrBackoff *= 2
				}
			case <-ctx.Done():
				break fetchLoop
			}
			continue
		}

		for msg := range msgs.Messages() {
			select {
			case p.sem <- struct{}{}:
			case <-ctx.Done():
				break fetchLoop
			}
			p.wg.Add(1)
			go func(m jetstream.Msg) {
				defer func() { <-p.sem; p.wg.Done() }()
				p.handle(ctx, handler, m)
			}(msg)
		}

		if err := msgs.Error(); err != nil && !isBenignFetchErr(err) {
			p.logger.Warn().
				Str("err", err.Error()).
				Msg("batch completed with error")
		}
	}

	// consumeCtx, err := p.consumer.Consume(func(msg jetstream.Msg) {
	// 	handler(ctx, msg)
	// })
	// if err != nil {
	// 	return nil
	// }

	p.logger.Info().
		Msg("workerpool draining in-flight handlers")
	p.wg.Wait()

	return ctx.Err()

	// return consumeCtx
}

func isBenignFetchErr(err error) bool {
	return errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, nats.ErrTimeout) ||
		errors.Is(err, jetstream.ErrNoMessages)
}

// handle runs one message through the handler with panic recovery, a
// per-message timeout, and ack/nak/term decisions based on delivery count.
func (p *Pool) handle(parentCtx context.Context, handler Handler, msg jetstream.Msg) {

	hctx, cancel := context.WithTimeout(parentCtx, p.cfg.HandlerTimeout)
	defer cancel()

	err := p.runHandler(hctx, handler, msg)
	if err == nil {
		if ackErr := msg.Ack(); ackErr != nil {
			p.logger.Warn().
				Str("err", ackErr.Error()).
				Msg("ack failed")
		}
		return
	}

	attempt := deliveryAttempt(msg)
	p.logger.Warn().
		Str("err", err.Error()).
		Int("attempt", attempt).
		Int("max_retries", p.cfg.MaxRetries).
		Msg("handler failed")

	if attempt >= p.cfg.MaxRetries {
		p.terminate(msg, err)
		return
	}

	delay := backoffFor(attempt, p.cfg.RetryDelay, p.cfg.RetryMaxDuration)
	if nakErr := msg.NakWithDelay(delay); nakErr != nil {
		p.logger.Warn().
			Str("err", nakErr.Error()).
			Msg("nak failed")
	}
}

func (p *Pool) runHandler(ctx context.Context, handler Handler, msg jetstream.Msg) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("handler panic: %v", r)
		}
	}()
	return handler(ctx, msg)
}

func (p *Pool) terminate(msg jetstream.Msg, cause error) {
	if p.cfg.DLQSubject != "" {
		go p.publishDLQ(msg, cause)
	}
	if termErr := msg.Term(); termErr != nil {
		p.logger.Warn().
			Str("err", termErr.Error()).
			Msg("term failed")
	}
}

func (p *Pool) publishDLQ(msg jetstream.Msg, cause error) {
	headers := nats.Header{}
	for k, v := range msg.Headers() {
		headers[k] = v
	}
	headers.Set("Dlq-Reason", cause.Error())
	headers.Set("Dlq-Original-Subject", msg.Subject())

	out := &nats.Msg{
		Subject: p.cfg.DLQSubject,
		Header:  headers,
		Data:    msg.Data(),
	}

	njs, err := GetGlobalNatsJetstreamConnection()
	if err != nil {
		panic(err)
	}
	// Best-effort: DLQ publish failures are logged, not retried, so they
	// can't themselves create unbounded backlog.
	if _, err := njs.js.PublishMsg(context.Background(), out); err != nil {
		p.logger.Error().
			Str("err", err.Error()).
			Str("subject", p.cfg.DLQSubject).
			Msg("dlq publish failed")
	}
}

func deliveryAttempt(msg jetstream.Msg) int {
	meta, err := msg.Metadata()
	if err != nil || meta == nil {
		return 1
	}
	return int(meta.NumDelivered)
}

func backoffFor(attempt int, base, max time.Duration) time.Duration {
	d := base * time.Duration(attempt)
	if d > max {
		return max
	}
	return d
}
