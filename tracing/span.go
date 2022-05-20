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
	"github.com/topfreegames/pitaya/v2/constants"
	pcontext "github.com/topfreegames/pitaya/v2/context"
	"github.com/topfreegames/pitaya/v2/logger"
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
// Deprecated: This StartSpan method is deprecated please use StartSpan method provided by ModuleTracer interface.
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
// Deprecated: This FinishSpan method is deprecated please use FinishSpan method provided by ModuleTracer interface.
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

// ModuleTracer interface declares methods related to Tracing,
// namely methods that provide management of tracing.
type ModuleTracer interface {
	// StartSpan starts a new span with a given parent context, operation name and module name.
	// It returns a context with the created span.
	StartSpan(ctx context.Context, operationName string, modulename string, paramTags map[string]interface{}) context.Context

	// FinishSpan finishes a span retrieved from the given context and logs the error if it exists
	FinishSpan(ctx context.Context)
}

type moduleTracerImpl struct {
}

// NewModuleTracer creates a new instance of ModuleTracer and returns a pointer to it.
func NewModuleTracer() ModuleTracer {
	return &moduleTracerImpl{}
}

func (tp *moduleTracerImpl) StartSpan(ctx context.Context, operationName string, modulename string,
	paramTags map[string]interface{}) context.Context {
	var parent opentracing.SpanContext
	if ctx == nil {
		ctx = context.Background()
	}

	if span := opentracing.SpanFromContext(ctx); span != nil {
		parent = span.Context()
	}

	//Parsing tags received as method parameter to opentracing tags map
	tags := opentracing.Tags{}
	for key, element := range paramTags {
		tags[key] = element
	}
	//Adding module name as a span tag
	tags["module.name"] = modulename

	span := opentracing.StartSpan(operationName, opentracing.ChildOf(parent), tags)
	return opentracing.ContextWithSpan(ctx, span)
}

func (tp *moduleTracerImpl) FinishSpan(ctx context.Context) {
	if span := opentracing.SpanFromContext(ctx); span != nil {
		span.Finish()
	}
}
