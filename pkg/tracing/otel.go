package tracing

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/topfreegames/pitaya/v3/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func InitializeOtel() error {
	// Print OpenTelemetry configuration
	printOtelConfig()

	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithOS(),
	)
	if err != nil {
		return err
	}

	client := otlptracegrpc.NewClient()

	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	logger.Log.Info("OpenTelemetry initialized using environment variables")
	return nil
}

func printOtelConfig() {
	vars := []string{
		"OTEL_SERVICE_NAME",
		"OTEL_EXPORTER_OTLP_ENDPOINT",
		"OTEL_EXPORTER_OTLP_PROTOCOL",
		"OTEL_TRACES_SAMPLER",
		"OTEL_TRACES_SAMPLER_ARG",
		"OTEL_RESOURCE_ATTRIBUTES",
		"OTEL_SDK_DISABLED",
	}

	config := make([]string, 0, len(vars))
	for _, v := range vars {
		value := os.Getenv(v)
		if value == "" {
			value = "<not set>"
		}
		config = append(config, fmt.Sprintf("%s=%s", v, value))
	}

	logger.Log.Info(fmt.Sprintf("OpenTelemetry Configuration: %s", strings.Join(config, ", ")))
}
