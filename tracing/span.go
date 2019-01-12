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

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/topfreegames/pitaya/constants"
	pcontext "github.com/topfreegames/pitaya/context"
	"github.com/topfreegames/pitaya/logger"
)

func castValueToCarrier(val interface{}) (opentracing.TextMapCarrier, error) {
	if v, ok := val.(opentracing.TextMapCarrier); ok {
		return v, nil
	}
	if m, ok := val.(map[string]interface{}); ok {
		carrier := map[string]string{}
		for k, v := range m {
			if s, ok := v.(string); ok {
				carrier[k] = s
			} else {
				logger.Log.Warnf("value from span carrier cannot be cast to string: %+v", v)
			}
		}
		return opentracing.TextMapCarrier(carrier), nil
	}
	return nil, constants.ErrInvalidSpanCarrier
}

// ExtractSpan retrieves an opentracing span context from the given context.Context
// The span context can be received directly (inside the context) or via an RPC call
// (encoded in binary format)
func ExtractSpan(ctx context.Context) (opentracing.SpanContext, error) {
	var spanCtx opentracing.SpanContext
	span := opentracing.SpanFromContext(ctx)
	if span == nil {
		if s := pcontext.GetFromPropagateCtx(ctx, constants.SpanPropagateCtxKey); s != nil {
			var err error
			carrier, err := castValueToCarrier(s)
			if err != nil {
				return nil, err
			}
			tracer := opentracing.GlobalTracer()
			spanCtx, err = tracer.Extract(opentracing.TextMap, carrier)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, nil
		}
	} else {
		spanCtx = span.Context()
	}
	return spanCtx, nil
}

// InjectSpan retrieves an opentrancing span from the current context and creates a new context
// with it encoded in binary format inside the propagatable context content
func InjectSpan(ctx context.Context) (context.Context, error) {
	span := opentracing.SpanFromContext(ctx)
	if span == nil {
		return ctx, nil
	}
	spanData := opentracing.TextMapCarrier{}
	tracer := opentracing.GlobalTracer()
	err := tracer.Inject(span.Context(), opentracing.TextMap, spanData)
	if err != nil {
		return nil, err
	}
	return pcontext.AddToPropagateCtx(ctx, constants.SpanPropagateCtxKey, spanData), nil
}

// StartSpan starts a new span with a given parent context, operation name, tags and
// optional parent span. It returns a context with the created span.
func StartSpan(
	parentCtx context.Context,
	opName string,
	tags opentracing.Tags,
	reference ...opentracing.SpanContext,
) context.Context {
	var ref opentracing.SpanContext
	if len(reference) > 0 {
		ref = reference[0]
	}
	span := opentracing.StartSpan(opName, opentracing.ChildOf(ref), tags)
	return opentracing.ContextWithSpan(parentCtx, span)
}

// FinishSpan finishes a span retrieved from the given context and logs the error if it exists
func FinishSpan(ctx context.Context, err error) {
	if ctx == nil {
		return
	}
	span := opentracing.SpanFromContext(ctx)
	if span == nil {
		return
	}
	defer span.Finish()
	if err != nil {
		LogError(span, err.Error())
	}
}
