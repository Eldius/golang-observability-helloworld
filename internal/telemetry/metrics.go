package telemetry

import (
	"context"
	"github.com/eldius/golang-observability-helloworld/internal/config"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.18.0"
	"google.golang.org/grpc/encoding/gzip"
)

func InitMetrics(serviceName string) {
	l := slog.Default()
	l.Debug("init tracer begin")

	ctx := context.Background()

	// initialize trace provider
	mp := initMetricsProvider(ctx, config.GetMetricsEndpoint(), serviceName)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	l.Debug("finished metrics provider configuration")

	l.Debug("starting runtime instrumentation")
	if err := runtime.Start(runtime.WithMinimumReadMemStatsInterval(time.Second)); err != nil {
		l.With("error", err).Warn("failed to start runtime monitoring")
		return
	}

	l.Debug("ending metrics provider")

	go waitMetrics(mp)
}

func initMetricsProvider(ctx context.Context, endpoint, serviceName string) otelmetric.MeterProvider {
	if endpoint == "" {
		return nil
	}
	exporter := otelMetricsExporter(ctx, endpoint)

	provider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(exporter)),
		metric.WithResource(defaultResources(endpoint, serviceName)))

	// set global tracer provider & text propagators
	otel.SetMeterProvider(provider)

	return provider
}

func defaultResources(endpoint, serviceName string) *resource.Resource {
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(serviceName),
		semconv.ServiceVersionKey.String("0"),
		attribute.String("environment", "test"),
	)
	return res
}

func otelMetricsExporter(ctx context.Context, endpoint string) metric.Exporter {
	l := slog.Default()
	l.Debug("configuring metric export for '%s'", endpoint)

	var opts []otlpmetricgrpc.Option

	opts = append(opts,
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithCompressor(gzip.Name),
		otlpmetricgrpc.WithTimeout(10*time.Second))

	exporter, err := otlpmetricgrpc.New(
		ctx,
		opts...,
	)
	if err != nil {
		l.With("error", err).Warn("failed to configure otel metrics exporter")
		return nil
	}

	return exporter
}

func waitMetrics(mp otelmetric.MeterProvider) {
	defer func() {
		if p, ok := mp.(*metric.MeterProvider); ok {
			l := slog.Default()
			if err := p.Shutdown(context.Background()); err != nil {
				l.With("error", err).Debug("error shutting down metric provider")
			}
		}
	}()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	<-ctx.Done()
}
