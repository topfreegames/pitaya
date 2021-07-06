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
	"io"
	"os"
	"testing"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/constants"
	pcontext "github.com/topfreegames/pitaya/context"
	"github.com/topfreegames/pitaya/tracing/jaeger"
)

var closer io.Closer

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}

func setup() {
	closer, _ = jaeger.Configure(jaeger.Options{ServiceName: "spanTest"})
}

func shutdown() {
	closer.Close()
}

func assertBaggage(t *testing.T, ctx opentracing.SpanContext, expected map[string]string) {
	b := extractBaggage(ctx, true)
	assert.Equal(t, expected, b)
}

func extractBaggage(ctx opentracing.SpanContext, allItems bool) map[string]string {
	b := make(map[string]string)
	ctx.ForeachBaggageItem(func(k, v string) bool {
		b[k] = v
		return allItems
	})
	return b
}

func TestExtractSpan(t *testing.T) {
	span := opentracing.StartSpan("op", opentracing.ChildOf(nil))
	ctx := opentracing.ContextWithSpan(context.Background(), span)
	spanCtx, err := ExtractSpan(ctx)
	assert.NoError(t, err)
	assert.Equal(t, span.Context(), spanCtx)
}

func TestExtractSpanInjectedSpan(t *testing.T) {
        
	span := opentracing.StartSpan("someOp")
        span.SetBaggageItem("some_key", "12345")
	span.SetBaggageItem("some-other-key", "42")
        expectedBaggage := map[string]string{"some_key": "12345", "some-other-key": "42"}

	spanData := opentracing.TextMapCarrier{}
	tracer := opentracing.GlobalTracer()

	err := tracer.Inject(span.Context(), opentracing.TextMap, spanData)
	assert.NoError(t, err)
	ctx := pcontext.AddToPropagateCtx(context.Background(), constants.SpanPropagateCtxKey, spanData)

	spanCtx, err := ExtractSpan(ctx)
	assert.NoError(t, err)
        assertBaggage(t, spanCtx, expectedBaggage)
}

func TestExtractSpanNoSpan(t *testing.T) {
	spanCtx, err := ExtractSpan(context.Background())
	assert.NoError(t, err)
	assert.Nil(t, spanCtx)
}

func TestExtractSpanBadInjected(t *testing.T) {
	ctx := pcontext.AddToPropagateCtx(context.Background(), constants.SpanPropagateCtxKey, []byte("nope"))
	spanCtx, err := ExtractSpan(ctx)
	assert.Equal(t, constants.ErrInvalidSpanCarrier, err)
	assert.Nil(t, spanCtx)
}

func TestInjectSpanContextWithoutSpan(t *testing.T) {
	origCtx := context.Background()
	ctx, err := InjectSpan(origCtx)
	assert.NoError(t, err)
	assert.Equal(t, origCtx, ctx)
}

func TestInjectSpan(t *testing.T) {
	span := opentracing.StartSpan("op", opentracing.ChildOf(nil))
	origCtx := opentracing.ContextWithSpan(context.Background(), span)
	ctx, err := InjectSpan(origCtx)
	assert.NoError(t, err)
	assert.NotEqual(t, origCtx, ctx)
	encodedCtx := pcontext.GetFromPropagateCtx(ctx, constants.SpanPropagateCtxKey)
	assert.NotNil(t, encodedCtx)
}

func TestStartSpan(t *testing.T) {
	origCtx := context.Background()
	ctxWithSpan := StartSpan(origCtx, "my-op", opentracing.Tags{"hi": "hello"})
	assert.NotEqual(t, origCtx, ctxWithSpan)

	span := opentracing.SpanFromContext(ctxWithSpan)
	assert.NotNil(t, span)
}

func TestFinishSpanNilCtx(t *testing.T) {
	assert.NotPanics(t, func() { FinishSpan(nil, nil) })
}

func TestFinishSpanCtxWithoutSpan(t *testing.T) {
	assert.NotPanics(t, func() { FinishSpan(context.Background(), nil) })
}

func TestFinishSpanWithErr(t *testing.T) {
	ctxWithSpan := StartSpan(context.Background(), "my-op", opentracing.Tags{"hi": "hello"})
	assert.NotPanics(t, func() { FinishSpan(ctxWithSpan, errors.New("hello")) })
}

func TestFinishSpan(t *testing.T) {
	ctxWithSpan := StartSpan(context.Background(), "my-op", opentracing.Tags{"hi": "hello"})
	assert.NotPanics(t, func() { FinishSpan(ctxWithSpan, nil) })
}
