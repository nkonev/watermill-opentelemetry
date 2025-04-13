package opentelemetry

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
)

// config represents the configuration options available for subscriber
// middlewares and publisher decorators.
type config struct {
	spanAttributes    []attribute.KeyValue
	textMapPropagator propagation.TextMapPropagator
}

// Option provides a convenience wrapper for simple options that can be
// represented as functions.
type Option func(*config)

// WithSpanAttributes includes the given attributes to the generated Spans.
func WithSpanAttributes(attributes ...attribute.KeyValue) Option {
	return func(c *config) {
		c.spanAttributes = attributes
	}
}

// WithTextMapPropagator includes the given attributes to the generated Spans.
func WithTextMapPropagator(p propagation.TextMapPropagator) Option {
	return func(c *config) {
		c.textMapPropagator = p
	}
}
