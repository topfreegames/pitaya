package tracing

import (
	"context"

	"github.com/topfreegames/pitaya/v2/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// Options holds configuration options for OpenTelemetry
type Options struct {
	Disabled    bool
	Probability float64
	ServiceName string
	Endpoint    string
}

// InitializeOtel configures a global OpenTelemetry tracer
func InitializeOtel(options Options) error {
	logger.Log.Infof("Configuring OpenTelemetry with options: %+v", options)

	if options.Disabled {
		return nil
	}

	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(options.ServiceName),
		),
	)
	if err != nil {
		return err
	}

	client := otlptracegrpc.NewClient(
		otlptracegrpc.WithEndpoint(options.Endpoint),
		otlptracegrpc.WithInsecure(),
	)

	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(options.Probability)),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return nil
}
