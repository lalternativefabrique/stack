// Package eventhandlers holds this context's integration-event handlers. Each
// handler implements consumer.EventHandler and writes only business logic in
// Handle; all JetStream redelivery semantics (Term on permanent errors, bounded
// MaxDeliver, staged BackOff, the DLQ stream, ack heartbeats, the reconnect
// loop) are provided by github.com/lalternativefabrique/eventkit/pkg/consumer.
//
// Do NOT hand-roll a JetStream subscription here. Copy this file, change the
// Subject/DurableName and the body of Handle — nothing else.
package eventhandlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lalternativefabrique/eventkit/pkg/consumer"
	"github.com/nats-io/nats.go"
)

// ExampleCreated is the integration-event payload this handler consumes.
type ExampleCreated struct {
	EventID string `json:"event_id"`
	ID      string `json:"id"`
	Name    string `json:"name"`
}

// ExampleCreatedHandler is a sample durable consumer. Delete it once you wire
// your own, or keep it as the template for new ones.
type ExampleCreatedHandler struct{}

func NewExampleCreatedHandler() *ExampleCreatedHandler { return &ExampleCreatedHandler{} }

func (*ExampleCreatedHandler) Name() string        { return "example-created" }
func (*ExampleCreatedHandler) Subject() string     { return "integration.example.created" }
func (*ExampleCreatedHandler) DurableName() string { return "example-created-consumer" }
func (*ExampleCreatedHandler) MaxDeliver() int     { return 3 }

func (*ExampleCreatedHandler) Handle(ctx context.Context, msg *nats.Msg) error {
	var ev ExampleCreated
	if err := json.Unmarshal(msg.Data, &ev); err != nil {
		// Malformed payload: no retry can fix it -> dead-letter immediately.
		return consumer.Permanent(fmt.Errorf("decode example.created: %w", err))
	}

	// ... your business logic. Returning a plain (non-permanent) error here
	// triggers a bounded, backed-off retry; after MaxDeliver the message is
	// routed to the DLQ stream automatically.
	_ = ev
	return nil
}
