Tracing
=======

Pitaya supports tracing using [OpenTracing](http://opentracing.io/).

### Using Jaeger tracing

First set the required environment variables:

```bash
export JAEGER_DISABLED=false
export JAEGER_SERVICE_NAME=my-pitaya-server
export JAEGER_SAMPLER_PARAM=1 #Ajust accordingly
```

With these environment variables set, you can use the following code to configure Jaeger:

```go
func configureJaeger(config *viper.Viper, logger logrus.FieldLogger) {
	cfg, err := jaegercfg.FromEnv()
	if cfg.ServiceName == "" {
		logger.Error("Could not init jaeger tracer without ServiceName, either set environment JAEGER_SERVICE_NAME or cfg.ServiceName = \"my-api\"")
		return
	}
	if err != nil {
		logger.Error("Could not parse Jaeger env vars: %s", err.Error())
		return
	}
	options := jaeger.Options{ // import "github.com/topfreegames/pitaya/v2/tracing/jaeger"
		Disabled:    cfg.Disabled,
		Probability: cfg.Sampler.Param,
		ServiceName: cfg.ServiceName,
	}
	jaeger.Configure(options)
}
```

Then in your main function:

```go
func main() {
    // ...
    configureJaeger(config, logger)
    // ...
}
```

Ensure to run this Jaeger initialization code in all your server types. Only changing the "JAEGER_SERVICE_NAME" env var between different types.

### Testing Locally
```bash
make run-jaeger-aio
make run-cluster-example-frontend-tracing
make run-cluster-example-backend-tracing
```

Then access Jaeger UI at http://localhost:16686