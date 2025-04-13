package opentelemetry

import (
	"context"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
	"log"
)

const subscriberTracerName = "watermill/subscriber"

// Trace defines a middleware that will add tracing.
func Trace(options ...Option) message.HandlerMiddleware {
	return func(h message.HandlerFunc) message.HandlerFunc {
		return TraceHandler(h, options...)
	}
}

// TraceHandler decorates a watermill HandlerFunc to add tracing when a message is received.
func TraceHandler(h message.HandlerFunc, options ...Option) message.HandlerFunc {
	tracer := otel.Tracer(subscriberTracerName)
	config := &config{}

	for _, opt := range options {
		opt(config)
	}

	spanOptions := []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(config.spanAttributes...),
	}

	return func(msg *message.Message) ([]*message.Message, error) {
		msgctx := msg.Context()
		spanName := message.HandlerNameFromCtx(msgctx)
		ctxWithSpan := createContextWithSpan(msgctx, msg.Metadata.Get(MessageMetadataTraceId))
		ctx, span := tracer.Start(ctxWithSpan, spanName, spanOptions...)
		span.SetAttributes(
			semconv.MessagingDestinationKindTopic,
			semconv.MessagingDestinationKey.String(message.SubscribeTopicFromCtx(ctx)),
			semconv.MessagingOperationReceive,
		)
		msg.SetContext(ctx)

		events, err := h(msg)

		if err != nil {
			span.RecordError(err)
		}
		span.End()

		return events, err
	}
}

// TraceNoPublishHandler decorates a watermill NoPublishHandlerFunc to add tracing when a message is received.
func TraceNoPublishHandler(h message.NoPublishHandlerFunc, options ...Option) message.NoPublishHandlerFunc {
	decoratedHandler := TraceHandler(func(msg *message.Message) ([]*message.Message, error) {
		return nil, h(msg)
	}, options...)

	return func(msg *message.Message) error {
		_, err := decoratedHandler(msg)

		return err
	}
}

//var randSource = rand.New(rand.NewSource(1))

func createContextWithSpan(ctx context.Context, traceId string) context.Context {
	traceID, err := trace.TraceIDFromHex(traceId)
	if err != nil {
		log.Printf("Unable to extract traceId from %v", traceId)
		return ctx
	}

	//spanID := trace.SpanID{}
	//_, _ = randSource.Read(spanID[:])

	// https://stackoverflow.com/questions/77161111/golang-set-custom-traceid-and-spanid-in-opentelemetry/77176591#77176591
	// ContextWithRemoteSpanContext
	ctxRet := trace.ContextWithSpanContext(
		ctx,
		trace.NewSpanContext(trace.SpanContextConfig{
			TraceID: traceID,
			// SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		}),
	)

	return ctxRet
}
