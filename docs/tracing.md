Tracing
=======

Pitaya supports tracing using [OpenTelemetry](https://opentelemetry.io/).

### Using OpenTelemetry tracing

Pitaya supports tracing using [OpenTelemetry](https://opentelemetry.io/). To enable and configure OpenTelemetry tracing, you can use standard OpenTelemetry environment variables.

First, make sure to call the `InitializeOtel` function in your main application:

```go
func main() {
    // ...
    err := tracing.InitializeOtel()
    if err != nil {
        logger.Log.Errorf("Failed to initialize OpenTelemetry: %v", err)
    }
    // ...
}
```

### Configuration Options

OpenTelemetry can be configured using standard environment variables. Here are some key variables you might want to set:

- `OTEL_SERVICE_NAME`: The name of your service.
- `OTEL_EXPORTER_OTLP_ENDPOINT`: The endpoint of your OpenTelemetry collector.
- `OTEL_EXPORTER_OTLP_PROTOCOL`: The protocol to use (e.g., `grpc` or `http/protobuf`).
- `OTEL_TRACES_SAMPLER`: The sampling strategy to use.
- `OTEL_TRACES_SAMPLER_ARG`: The argument for the sampling strategy.
- `OTEL_SDK_DISABLED`: Set to `true` to disable tracing.

For a complete list of OpenTelemetry environment variables, refer to the [OpenTelemetry specification](https://opentelemetry.io/docs/concepts/sdk-configuration/general-sdk-configuration/).

### Testing Locally

To test OpenTelemetry tracing locally, you can use Jaeger as your tracing backend. First, start Jaeger using Docker:

```bash
make run-jaeger-aio
make run-cluster-example-frontend-tracing
make run-cluster-example-backend-tracing
```

The last two commands will run your Pitaya servers with OpenTelemetry configured with the following envs:

```bash
OTEL_SERVICE_NAME=example-frontend OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317 OTEL_EXPORTER_OTLP_PROTOCOL=grpc OTEL_TRACES_SAMPLER=parentbased_traceidratio OTEL_TRACES_SAMPLER_ARG="1"

OTEL_SERVICE_NAME=example-backend OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317 OTEL_EXPORTER_OTLP_PROTOCOL=grpc OTEL_TRACES_SAMPLER=parentbased_traceidratio OTEL_TRACES_SAMPLER_ARG="1"
```

Access the Jaeger UI at http://localhost:16686 to view and analyze your traces.