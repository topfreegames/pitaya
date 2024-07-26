// Copyright (c) TFG Co. All Rights Reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package tracing

import (
	"context"

	"github.com/topfreegames/pitaya/v2/constants"
	pcontext "github.com/topfreegames/pitaya/v2/context"
	"github.com/topfreegames/pitaya/v2/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	otelTrace "go.opentelemetry.io/otel/trace"
)

func castValueToCarrier(val interface{}) (propagation.MapCarrier, error) {
	if v, ok := val.(propagation.MapCarrier); ok {
		return v, nil
	}
	if m, ok := val.(map[string]interface{}); ok {
		carrier := make(propagation.MapCarrier)
		for k, v := range m {
			if s, ok := v.(string); ok {
				carrier[k] = s
			} else {
				logger.Log.Warnf("value from span carrier cannot be cast to string: %+v", v)
			}
		}
		return carrier, nil
	}
	return nil, constants.ErrInvalidSpanCarrier
}

// ExtractSpan retrieves an OpenTelemetry span context from the given context.Context
// The span context can be received directly (inside the context) or via an RPC call
// (encoded in a carrier)
func ExtractSpan(ctx context.Context) (otelTrace.SpanContext, error) {
	span := otelTrace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext(), nil
	}

	if s := pcontext.GetFromPropagateCtx(ctx, constants.SpanPropagateCtxKey); s != nil {
		carrier, err := castValueToCarrier(s)
		if err != nil {
			return otelTrace.SpanContext{}, err
		}

		propagator := otel.GetTextMapPropagator()
		extractedCtx := propagator.Extract(ctx, propagation.MapCarrier(carrier))
		extractedSpan := otelTrace.SpanFromContext(extractedCtx)
		return extractedSpan.SpanContext(), nil
	}

	return otelTrace.SpanContext{}, nil
}

// InjectSpan retrieves an OpenTelemetry span from the current context and creates a new context
// with it encoded in text map format inside the propagatable context content
func InjectSpan(ctx context.Context) (context.Context, error) {
	span := otelTrace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return ctx, nil
	}

	carrier := make(map[string]string)
	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(carrier))

	return pcontext.AddToPropagateCtx(ctx, constants.SpanPropagateCtxKey, carrier), nil
}

// StartSpan starts a new span with a given parent context, operation name, and attributes.
// It returns a context with the created span.
func StartSpan(
	parentCtx context.Context,
	opName string,
	attributes ...attribute.KeyValue,
) (context.Context, trace.Span) {
	tracer := otel.Tracer("pitaya")
	ctx, span := tracer.Start(parentCtx, opName,
		trace.WithAttributes(attributes...),
	)

	return ctx, span
}

// FinishSpan finishes a span retrieved from the given context and logs the error if it exists
func FinishSpan(ctx context.Context, err error) {
	if ctx == nil {
		return
	}
	span := otelTrace.SpanFromContext(ctx)
	if span == nil {
		return
	}
	defer span.End()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}
