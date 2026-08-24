package observability

import (
	"context"
	"os"
	"time"

	"github.com/fagbenjaenoch/dorms-ng/internal/server"
	"github.com/fagbenjaenoch/dorms-ng/internal/utils"
	hostMetrics "go.opentelemetry.io/contrib/instrumentation/host"
	runtimeMetrics "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

type Observability struct {
	s *server.Server
}

func NewObservability(s *server.Server) *Observability {
	return &Observability{
		s: s,
	}
}

func (o *Observability) SetupObservability() ([]func(context.Context) error, error) {
	var shutdownFuncs []func(context.Context) error

	res, err := o.InitResources(context.Background())
	if err != nil {
		return nil, err
	}

	tracerCloser, err := o.InitTracer(context.Background(), res)
	if err != nil {
		return nil, err
	}
	shutdownFuncs = append(shutdownFuncs, tracerCloser)

	metricCloser, err := o.InitMetrics(context.Background(), res)
	if err != nil {
		return nil, err
	}
	shutdownFuncs = append(shutdownFuncs, metricCloser)

	loggerCloser, err := o.InitLogger(context.Background(), res)
	if err != nil {
		return nil, err
	}
	shutdownFuncs = append(shutdownFuncs, loggerCloser)

	return shutdownFuncs, nil
}

func (o *Observability) InitResources(ctx context.Context) (*resource.Resource, error) {
	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(o.s.Config.Observability.AppName),
			semconv.DeploymentEnvironmentName(o.s.Config.Observability.Environment),
		),
		resource.WithHost(),
		resource.WithOS(),
		resource.WithProcess(),
	)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (o *Observability) InitTracer(ctx context.Context, res *resource.Resource) (func(context.Context) error, error) {
	var exporter sdkTrace.SpanExporter
	var err error

	if utils.IsProduction() {
		exporter, err = otlptracehttp.New(
			ctx,
			otlptracehttp.WithEndpointURL(o.s.Config.Observability.Endpoint),
			otlptracehttp.WithCompression(otlptracehttp.GzipCompression),
		)
		if err != nil {
			return nil, err
		}
	} else {
		file, err := os.Create("trace_debug.json")
		if err != nil {
			panic(err)
		}

		exporter, err = stdouttrace.New(stdouttrace.WithWriter(file), stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, err
		}
		o.s.Logger.Info().Msg("using stdout exporter for local development")
	}

	tp := sdkTrace.NewTracerProvider(
		sdkTrace.WithBatcher(exporter,
			sdkTrace.WithMaxQueueSize(2048),          // 2    MB
			sdkTrace.WithMaxExportBatchSize(512),     // 512 spans
			sdkTrace.WithBatchTimeout(5*time.Second), // 5 seconds
		),
		sdkTrace.WithResource(res),
	)

	// set global tracer provider
	otel.SetTracerProvider(tp)

	// set global propagator for context propagation
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	o.s.Logger.Info().Msg("opentelemetry initialization successful")

	return tp.Shutdown, nil
}

var GlobalMetric *metric.Meter

func (o *Observability) InitMetrics(ctx context.Context, res *resource.Resource) (func(context.Context) error, error) {
	var metricExp sdkmetric.Exporter
	var err error
	var interval = 15 * time.Second

	if utils.IsProduction() {
		metricExp, err = otlpmetrichttp.New(
			ctx,
			otlpmetrichttp.WithCompression(otlpmetrichttp.GzipCompression),
			otlpmetrichttp.WithEndpointURL(o.s.Config.Observability.Endpoint),
		)
		if err != nil {
			return nil, err
		}
	} else {
		file, err := os.Create("metric_debug.json")
		if err != nil {
			panic(err)
		}

		metricExp, err = stdoutmetric.New(stdoutmetric.WithWriter(file), stdoutmetric.WithPrettyPrint())
		if err != nil {
			return nil, err
		}

		interval = 1 * time.Minute

		o.s.Logger.Info().Msg("using console for metrics output")
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp, sdkmetric.WithInterval(interval))),
		sdkmetric.WithResource(res),
	)

	err = runtimeMetrics.Start(
		runtimeMetrics.WithMeterProvider(mp),
	)
	if err != nil {
		return nil, err
	}

	if utils.IsProduction() {
		err = hostMetrics.Start(
			hostMetrics.WithMeterProvider(mp),
		)
		if err != nil {
			return nil, err
		}
	}

	// Set as the global MeterProvider
	otel.SetMeterProvider(mp)
	newMeter := mp.Meter(o.s.Config.Observability.AppName)
	GlobalMetric = &newMeter

	return mp.Shutdown, nil
}

func (o *Observability) InitLogger(ctx context.Context, res *resource.Resource) (func(context.Context) error, error) {
	logExp, err := otlploghttp.New(ctx,
		otlploghttp.WithEndpointURL(o.s.Config.Observability.LoggingEndpoint),
		otlploghttp.WithCompression(otlploghttp.GzipCompression),
	)
	if err != nil {
		return nil, err
	}

	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExp)),
		sdklog.WithResource(res),
	)

	// Set as the global LoggerProvider
	global.SetLoggerProvider(lp)

	return lp.Shutdown, nil
}
