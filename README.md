# Watermill OpenTelemetry integration

Bringing distributed tracing support to [Watermill](https://watermill.io/) with [OpenTelemetry](https://opentelemetry.io/). 

## Usage

### Installation

```shell
go get github.com/nkonev/watermill-opentelemetry
```

### For publishers

```go
package example

import (
    "github.com/ThreeDotsLabs/watermill-googlecloud/pkg/googlecloud"
    "github.com/ThreeDotsLabs/watermill/message"
    "github.com/garsue/watermillzap"
	wotel "github.com/nkonev/watermill-opentelemetry"
    "go.uber.org/zap"
)

type PublisherConfig struct {
	Name         string
	GCPProjectID string
}

// NewPublisher instantiates a GCP Pub/Sub Publisher with tracing capabilities.
func NewPublisher(logger *zap.Logger, config PublisherConfig) (message.Publisher, error) {
	publisher, err := googlecloud.NewPublisher(
        googlecloud.PublisherConfig{ProjectID: config.GCPProjectID},
        watermillzap.NewLogger(logger),
    )
	if err != nil {
		return nil, err
	}

	if config.Name == "" {
		return wotel.NewPublisherDecorator(publisher), nil
	}

	return wotel.NewNamedPublisherDecorator(config.Name, publisher), nil
}
```

### For subscribers

A tracing middleware can be defined at the router level:

```go
package example

import (
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	wotel "github.com/nkonev/watermill-opentelemetry"
)

func InitTracedRouter() (*message.Router, error) {
	router, err := message.NewRouter(message.RouterConfig{}, watermill.NopLogger{})
	if err != nil {
		return nil, err
	}

	router.AddMiddleware(wotel.Trace())

	return router, nil
}
```

Alternatively, individual handlers can be traced: 

```go
package example

import (
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	wotel "github.com/nkonev/watermill-opentelemetry"
)

func InitRouter() (*message.Router, error) {
	router, err := message.NewRouter(message.RouterConfig{}, watermill.NopLogger{})
	if err != nil {
		return nil, err
	}
    
    // subscriber definition omitted for clarity
    subscriber := (message.Subscriber)(nil)

	router.AddNoPublisherHandler(
        "handler_name",
        "subscribeTopic",
        subscriber,
        wotel.TraceNoPublishHandler(func(msg *message.Message) error {
            return nil
        }),
    )

	return router, nil
}
```

### Contributors

- [@K-Phoen](https://github.com/K-Phoen)
- [@jeespers](https://github.com/jeespers)
- [@nkonev](https://github.com/nkonev)

## License

Apache 2.0, see [LICENSE.md](LICENSE.md).
