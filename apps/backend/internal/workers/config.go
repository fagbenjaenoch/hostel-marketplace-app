package workerpool

import "time"

// Config controls both throughput and, more importantly for production use,
// the worst-case memory footprint of the pool. The three knobs that matter
// most for memory are Concurrency, FetchBatchSize, and MaxAckPending — see
// the README for the sizing formula.
type Config struct {
	// Concurrency is the max number of messages handled at the same time
	// by this process. This is the hard ceiling on concurrent handler
	// goroutines and is enforced client-side with a semaphore.
	Concurrency int `default:"4"`

	// FetchBatchSize is how many messages a single pull request asks for.
	// Messages returned by a Fetch call are held in memory before (and
	// while) being dispatched to workers, so this should not be set much
	// higher than Concurrency — otherwise you're buffering messages that
	// have no worker free to process them yet.
	FetchBatchSize int `default:"10"`

	// FetchMaxWait bounds how long a Fetch call blocks waiting for
	// messages before returning (possibly empty-handed). Keep this short
	// (1-5s) so shutdown and re-fetch loops stay responsive.
	FetchMaxWait time.Duration `default:"5"`

	// MaxAckPending is the server-side cap on how many unacked messages
	// JetStream will let this consumer hold at once, across the whole
	// process (and across all replicas sharing the durable name). This is
	// the primary memory-safety knob: it bounds total in-flight messages
	// even if you (or a future contributor) misconfigure Concurrency or
	// spin up extra pool instances. Rule of thumb: set it equal to or
	// slightly above Concurrency, never far above it.
	MaxAckPending int `default:"4"`

	// AckWait is how long JetStream waits for an Ack before considering
	// the message unacked and eligible for redelivery. Should comfortably
	// exceed your p99 handler duration; too short causes duplicate
	// processing, too long delays failure recovery.
	AckWait time.Duration `default:"5"`

	// MaxRetries caps redelivery attempts before a message is terminated
	// (and optionally sent to DLQSubject). Prevents poison messages from
	// looping forever and slowly consuming worker capacity.
	MaxRetries int `default:"10"`

	// RetryDelay is the unit used to compute Nak redelivery delay:
	// delay = RetryDelay * attempt, capped at RetryMaxDuration.
	RetryDelay       time.Duration `default:"5"`
	RetryMaxDuration time.Duration `default:"60"`

	// DLQSubject, if non-empty, receives a copy of the original message
	// payload (plus headers describing the failure) once MaxRetries is
	// exhausted. If empty, exhausted messages are simply terminated.
	DLQSubject string

	// HandlerTimeout bounds a single handler invocation. If zero, it
	// defaults to AckWait minus a small safety margin, so a stuck handler
	// can't silently hold a semaphore slot (and its share of the
	// MaxAckPending budget) forever.
	HandlerTimeout time.Duration `default:"30"`
}
