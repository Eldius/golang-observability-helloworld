package telemetry

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/eldius/golang-observability-helloworld/internal/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log/slog"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"time"
)

const (
	TraceIDContextKey = "X-Trace-Id"
)

var tracerInstance trace.Tracer

var (
	counterID uint64
	prefix    string
)

func init() {
	hostname, err := os.Hostname()
	if hostname == "" || err != nil {
		hostname = "localhost"
	}
	var buf [12]byte
	var b64 string
	for len(b64) < 10 {
		rand.Read(buf[:])
		b64 = base64.StdEncoding.EncodeToString(buf[:])
		b64 = strings.NewReplacer("+", "", "/", "").Replace(b64)
	}

	prefix = fmt.Sprintf("%s/%s", hostname, b64[0:10])
}

func InitTracer(serviceName string) {
	l := slog.Default()
	l.Debug("init tracer begin")

	// initialize trace provider
	tp := initTracerProvider(context.Background(), config.GetTracesEndpoint(), serviceName)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	l.Debug("finished tracer configuration")

	l.Debug("ending tracer provider")

	go waitTraces(tp)
}

func initTracerProvider(ctx context.Context, endpoint, serviceName string) trace.TracerProvider {
	if endpoint == "" {
		return nil
	}
	exporter := otelTraceExporter(ctx, endpoint)

	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(serviceName),
		semconv.ServiceVersionKey.String("0"),
		attribute.String("environment", "test"),
	)

	// Register the trace exporter with a TracerProvider, using a batch
	// span processor to aggregate spans before export.
	bsp := sdktrace.NewBatchSpanProcessor(exporter)
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	// set global tracer provider & text propagators
	otel.SetTracerProvider(provider)
	tracerInstance = provider.Tracer(serviceName)

	return provider
}

func otelTraceExporter(ctx context.Context, endpoint string) sdktrace.SpanExporter {
	l := slog.Default()
	l.Debug(fmt.Sprintf("configuring trace export for '%s'", endpoint))

	var err error
	conn, err := grpc.DialContext(
		ctx,
		endpoint,
		// Note the use of insecure transport here. TLS is recommended in production.
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		l.With("error", err).Error("failed to create gRPC connection to collector")
		panic(err)
	}

	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		l.With("error", err).Error("failed to setup exporter")
		panic(err)
	}

	return exporter
}

func waitTraces(tp trace.TracerProvider) {
	defer func() {
		if p, ok := tp.(*sdktrace.TracerProvider); ok {
			l := slog.Default()
			if err := p.Shutdown(context.Background()); err != nil {
				l.With("error", err).Debug("error shutting down tracer provider")
			}
		}
	}()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	<-ctx.Done()
}

func NewSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	myid := atomic.AddUint64(&counterID, 1)
	traceID := fmt.Sprintf("%s-%06d", prefix, myid)
	ctx = context.WithValue(ctx, TraceIDContextKey, traceID)

	//return otel.GetTracerProvider().Tracer("testing").Start(ctx, "testing_again")
	//otelhttp.WithRouteTag(pattern, http.HandlerFunc(handlerFunc))
	//otelhttp.NewHandler(mux, "/")
	return tracerInstance.
		Start(ctx,
			name,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithTimestamp(time.Now()))

	//return tracerInstance.Start(ctx, name, trace.WithNewRoot(), trace.WithSpanKind(trace.SpanKindServer), trace.WithTimestamp(time.Now()))
}

func NotifyError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err, trace.WithStackTrace(true))
}
