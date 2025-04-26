package opentelemetry

import (
	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
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
	config := &config{}

	for _, opt := range options {
		opt(config)
	}

	var tracer trace.Tracer
	if config.tracer != nil {
		tracer = config.tracer
	} else {
		tracer = otel.Tracer(subscriberTracerName)
	}

	spanOptions := []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(config.spanAttributes...),
	}

	return func(msg *message.Message) ([]*message.Message, error) {
		msgctx := msg.Context()
		spanName := message.HandlerNameFromCtx(msgctx)
		ctxWithSpan := getPropagator(config).Extract(msgctx, metadataWrapper{msg.Metadata})
		ctx, span := tracer.Start(ctxWithSpan, spanName, spanOptions...)

		spanAttributes := []attribute.KeyValue{
			semconv.MessagingDestinationKindTopic,
			semconv.MessagingDestinationKey.String(message.SubscribeTopicFromCtx(ctx)),
			semconv.MessagingOperationReceive,
		}
		msgName := msg.Metadata.Get("name")
		if len(msgName) > 0 {
			spanAttributes = append(spanAttributes, semconv.MessageTypeKey.String(msgName))
		}
		span.SetAttributes(spanAttributes...)
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

func getPropagator(config *config) propagation.TextMapPropagator {
	if config.textMapPropagator != nil {
		return config.textMapPropagator
	} else {
		return otel.GetTextMapPropagator()
	}
}
