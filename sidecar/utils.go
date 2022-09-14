package sidecar

import (
	"context"
	"github.com/opentracing/opentracing-go"
	"github.com/topfreegames/pitaya/v2/pkg/logger"
	"github.com/topfreegames/pitaya/v2/pkg/tracing"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	"google.golang.org/grpc/metadata"
)

func checkError(err error) {
	if err != nil {
		logger.Log.Fatalf("failed to start sidecar: %s", err)
	}
}

// TODO need to replace pitayas own jaeger config with something that actually makes sense
func configureJaeger(debug bool) {
	cfg, err := jaegercfg.FromEnv()
	if debug {
		cfg.ServiceName = "pitaya-sidecar"
		cfg.Sampler.Type = "const"
		cfg.Sampler.Param = 1
	}
	if cfg.ServiceName == "" {
		logger.Log.Error("Could not init jaeger tracer without ServiceName, either set environment JAEGER_SERVICE_NAME or cfg.ServiceName = \"my-api\"")
		return
	}
	if err != nil {
		logger.Log.Error("Could not parse Jaeger env vars: %s", err.Error())
		return
	}
	tracer, _, err := cfg.NewTracer()
	if err != nil {
		logger.Log.Error("Could not initialize jaeger tracer: %s", err.Error())
		return
	}
	opentracing.SetGlobalTracer(tracer)
	logger.Log.Infof("Tracer configured for %s", cfg.Reporter.LocalAgentHostPort)
}


func getCtxWithParentSpan(ctx context.Context, op string) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		carrier := opentracing.HTTPHeadersCarrier(md)
		spanContext, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, carrier)
		if err != nil {
			logger.Log.Debugf("tracing: could not extract span from context!")
		} else {
			return tracing.StartSpan(ctx, op, opentracing.Tags{}, spanContext)
		}
	}
	return ctx
}

