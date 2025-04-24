package opentelemetry

import (
	"fmt"
	"go.opentelemetry.io/otel/propagation"
	"strings"

	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
)

const publisherTracerName = "watermill/publisher"

// PublisherDecorator decorates a standard watermill publisher to add tracing capabilities.
type PublisherDecorator struct {
	pub           message.Publisher
	publisherName string
	config        config
	tracer        trace.Tracer
}

// NewPublisherDecorator instantiates a PublisherDecorator with a default name.
func NewPublisherDecorator(pub message.Publisher, options ...Option) message.Publisher {
	return NewNamedPublisherDecorator(structName(pub), pub, options...)
}

// NewNamedPublisherDecorator instantiates a PublisherDecorator with a provided name.
func NewNamedPublisherDecorator(name string, pub message.Publisher, options ...Option) message.Publisher {
	config := config{}

	for _, opt := range options {
		opt(&config)
	}

	return &PublisherDecorator{
		pub:           pub,
		publisherName: name,
		config:        config,
		tracer:        otel.Tracer(publisherTracerName),
	}
}

// Publish implements the watermill Publisher interface and creates traces.
// Publishing of messages are delegated to the decorated Publisher.
func (p *PublisherDecorator) Publish(topic string, messages ...*message.Message) error {
	if len(messages) == 0 {
		return nil
	}

	spans := make([]trace.Span, len(messages))
	for i, msg := range messages {
		span := p.startSpan(topic, msg)
		spans[i] = span
	}

	err := p.pub.Publish(topic, messages...)

	for _, span := range spans {
		p.endSpan(err, span)
	}

	return err
}

func (p *PublisherDecorator) startSpan(topic string, msg *message.Message) trace.Span {
	msgctx := msg.Context()
	spanName := message.PublisherNameFromCtx(msgctx)
	if spanName == "" {
		spanName = p.publisherName
	}

	ctx, span := p.tracer.Start(msgctx, spanName, trace.WithSpanKind(trace.SpanKindProducer))
	msg.SetContext(ctx)

	p.getPropagator().Inject(ctx, metadataWrapper{msg.Metadata})

	spanAttributes := []attribute.KeyValue{
		semconv.MessagingDestinationKindTopic,
		semconv.MessagingDestinationKey.String(topic),
		semconv.MessagingOperationProcess,
	}
	msgName := msg.Metadata.Get("name")
	if len(msgName) > 0 {
		spanAttributes = append(spanAttributes, semconv.MessageTypeKey.String(msgName))
	}
	spanAttributes = append(spanAttributes, p.config.spanAttributes...)
	span.SetAttributes(spanAttributes...)

	return span
}

func (p *PublisherDecorator) endSpan(err error, span trace.Span) {
	if err != nil {
		span.RecordError(err)
	}
	span.End()
}

// Close implements the watermill Publisher interface.
func (p *PublisherDecorator) Close() error {
	return p.pub.Close()
}

func (p *PublisherDecorator) getPropagator() propagation.TextMapPropagator {
	if p.config.textMapPropagator != nil {
		return p.config.textMapPropagator
	} else {
		return otel.GetTextMapPropagator()
	}
}

func structName(v interface{}) string {
	if s, ok := v.(fmt.Stringer); ok {
		return s.String()
	}

	s := fmt.Sprintf("%T", v)
	// trim the pointer marker, if any
	return strings.TrimLeft(s, "*")
}

type metadataWrapper struct {
	message.Metadata
}

func (mw metadataWrapper) Keys() []string {
	i := 0
	r := make([]string, len(mw.Metadata))

	for k := range mw.Metadata {
		r[i] = k
		i++
	}

	return r
}
