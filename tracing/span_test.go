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
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/v2/constants"
	pcontext "github.com/topfreegames/pitaya/v2/context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var tracerProvider *sdktrace.TracerProvider

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}

func setup() {
	tracerProvider = sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
}

func shutdown() {
	_ = tracerProvider.Shutdown(context.Background())
}

func TestExtractSpan(t *testing.T) {
	ctx, span := otel.Tracer("test").Start(context.Background(), "op")
	defer span.End()

	spanCtx, err := ExtractSpan(ctx)
	assert.NoError(t, err)
	assert.Equal(t, span.SpanContext(), spanCtx)
	assert.True(t, spanCtx.IsValid())
}

func TestExtractSpanInjectedSpan(t *testing.T) {
	ctx, span := otel.Tracer("test").Start(context.Background(), "someOp")
	defer span.End()

	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	ctx = pcontext.AddToPropagateCtx(context.Background(), constants.SpanPropagateCtxKey, carrier)

	spanCtx, err := ExtractSpan(ctx)
	assert.NoError(t, err)
	assert.Equal(t, span.SpanContext().TraceID(), spanCtx.TraceID())
	assert.Equal(t, span.SpanContext().SpanID(), spanCtx.SpanID())
	assert.True(t, spanCtx.IsValid())
}

func TestExtractSpanNoSpan(t *testing.T) {
	spanCtx, err := ExtractSpan(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, trace.SpanContext{}, spanCtx)
}

func TestExtractSpanBadInjected(t *testing.T) {
	ctx := pcontext.AddToPropagateCtx(context.Background(), constants.SpanPropagateCtxKey, []byte("nope"))
	spanCtx, err := ExtractSpan(ctx)
	assert.Equal(t, constants.ErrInvalidSpanCarrier, err)
	assert.Equal(t, trace.SpanContext{}, spanCtx)
}

func TestInjectSpanContextWithoutSpan(t *testing.T) {
	origCtx := context.Background()
	ctx, err := InjectSpan(origCtx)
	assert.NoError(t, err)
	assert.Equal(t, origCtx, ctx)
}

func TestInjectSpan(t *testing.T) {
	ctx, span := otel.Tracer("test").Start(context.Background(), "op")
	defer span.End()

	injectedCtx, err := InjectSpan(ctx)
	assert.NoError(t, err)
	assert.NotEqual(t, ctx, injectedCtx)
	encodedCtx := pcontext.GetFromPropagateCtx(injectedCtx, constants.SpanPropagateCtxKey)
	assert.NotNil(t, encodedCtx)
}

func TestStartSpan(t *testing.T) {
	parentCtx := context.Background()
	opName := "test-operation"
	attributes := []attribute.KeyValue{
		attribute.String("key1", "value1"),
		attribute.Int("key2", 42),
	}

	ctx, span := StartSpan(parentCtx, opName, attributes...)
	defer span.End()

	assert.NotNil(t, span)
	assert.NotEqual(t, parentCtx, ctx)

	spanCtx := span.SpanContext()
	assert.True(t, spanCtx.IsValid())
	assert.NotEmpty(t, spanCtx.TraceID())
	assert.NotEmpty(t, spanCtx.SpanID())

	// Check if attributes are set
	spanAttrs := span.(interface{ Attributes() []attribute.KeyValue }).Attributes()
	assert.Len(t, spanAttrs, len(attributes))
	for i, attr := range attributes {
		assert.Equal(t, attr, spanAttrs[i])
	}
}

func TestFinishSpan(t *testing.T) {
	t.Run("with error", func(t *testing.T) {
		ctx, span := StartSpan(context.Background(), "test-operation")
		testErr := errors.New("test error")

		FinishSpan(ctx, testErr)

		spanStatus := span.(interface{ Status() codes.Code }).Status()
		assert.Equal(t, codes.Error, spanStatus)

		spanEvents := span.(interface{ Events() []sdktrace.Event }).Events()
		assert.Len(t, spanEvents, 1)
		assert.Equal(t, "exception", spanEvents[0].Name)
		assert.Equal(t, testErr.Error(), spanEvents[0].Attributes[0].Value.AsString())
	})

	t.Run("without error", func(t *testing.T) {
		ctx, span := StartSpan(context.Background(), "test-operation")

		FinishSpan(ctx, nil)

		spanStatus := span.(interface{ Status() codes.Code }).Status()
		assert.Equal(t, codes.Unset, spanStatus)

		spanEvents := span.(interface{ Events() []sdktrace.Event }).Events()
		assert.Len(t, spanEvents, 0)
	})

	t.Run("nil context", func(t *testing.T) {
		assert.NotPanics(t, func() {
			FinishSpan(nil, nil)
		})
	})
}
func TestFinishSpanNilCtx(t *testing.T) {
	assert.NotPanics(t, func() { FinishSpan(nil, nil) })
}

func TestFinishSpanCtxWithoutSpan(t *testing.T) {
	assert.NotPanics(t, func() { FinishSpan(context.Background(), nil) })
}

func TestFinishSpanWithErr(t *testing.T) {
	ctx, span := StartSpan(context.Background(), "my-op", attribute.String("hi", "hello"))
	assert.NotPanics(t, func() { FinishSpan(ctx, errors.New("hello")) })

	spanStatus := span.(interface{ Status() codes.Code }).Status()
	assert.Equal(t, codes.Error, spanStatus)

	spanEvents := span.(interface{ Events() []sdktrace.Event }).Events()
	assert.Len(t, spanEvents, 1)
	assert.Equal(t, "exception", spanEvents[0].Name)
	assert.Equal(t, "hello", spanEvents[0].Attributes[0].Value.AsString())
}

func TestFinishSpanWithoutErr(t *testing.T) {
	ctx, span := StartSpan(context.Background(), "my-op", attribute.String("hi", "hello"))
	assert.NotPanics(t, func() { FinishSpan(ctx, nil) })

	spanStatus := span.(interface{ Status() codes.Code }).Status()
	assert.Equal(t, codes.Unset, spanStatus)

	spanEvents := span.(interface{ Events() []sdktrace.Event }).Events()
	assert.Len(t, spanEvents, 0)
}
