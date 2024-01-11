package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/eldius/golang-observability-helloworld/internal/config"
	"github.com/eldius/golang-observability-helloworld/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"math/rand"
	"os"
	"time"
)

func main() {
	telemetry.InitMetrics("hello-world")
	telemetry.InitTracer("hello-world")

	go mainLoop()

	fmt.Println("Press 'Enter' to continue...")
	_, err := bufio.NewReader(os.Stdin).ReadBytes('\n')
	if err != nil {
		config.NewLogger(context.Background()).With("error", err).Error("ErrorReadingFromConsole")
		panic(err)
	}
	os.Exit(0)
}

func mainLoop() {
	for {
		process()
	}
}

func process() {
	ctx := context.Background()
	l := config.NewLogger(ctx)
	ctx, span := telemetry.NewSpan(ctx, "mainSpan")
	iterationID := rand.Int63()
	l = l.With("interation_id", iterationID, "context", ctx, "span", span, "context", span.SpanContext())

	l.Info("StartIteration")
	span.SetAttributes(attribute.Int64("random_int_attribute", iterationID))
	span.SetAttributes(attribute.String("random_string_attribute", fmt.Sprintf("%d", iterationID)))
	defer func() {
		l.Info("StopIteration")
		span.End()
	}()

	time.Sleep(time.Duration(rand.Intn(10)) * time.Second)
}
