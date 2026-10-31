// Package domain holds the Example aggregate — a minimal, event-sourced
// bounded context that demonstrates the go/eda CQRS building blocks
// (ddd.BaseAggregateRoot + ddd.Raise + LoadFromHistory). It is an EXAMPLE:
// copy it to a real context name, or delete it. Modelled on
// github.com/lalternativefabrique/eventkit/examples/banking.
package domain

import (
	"fmt"

	"github.com/lalternativefabrique/eventkit/pkg/cqrs"
	"github.com/lalternativefabrique/eventkit/pkg/ddd"
)

// ID is the aggregate identifier type (a UUID string).
type ID = string

// --- Domain events (plain structs implementing ddd event kinds) ---

// ExampleCreated is raised when a new Example is created.
type ExampleCreated struct {
	Name string `json:"name"`
}

func (ExampleCreated) EventKind() string { return "example.created" }

// ExampleRenamed is raised when an Example is renamed.
type ExampleRenamed struct {
	Name string `json:"name"`
}

func (ExampleRenamed) EventKind() string { return "example.renamed" }

// --- Aggregate ---

// Example is the event-sourced aggregate root.
type Example struct {
	ddd.BaseAggregateRoot[ID]
	name    string
	created bool
}

// New returns an empty Example ready to Load or Create.
func New(id ID) *Example {
	e := &Example{}
	e.Init(id, "Example", ddd.SystemClock{})
	return e
}

// Apply mutates in-memory state from an event. It must be exhaustive and
// side-effect free (it also runs during history replay).
func (e *Example) Apply(env ddd.EventEnvelope[ID]) error {
	switch p := env.Payload.(type) {
	case ExampleCreated:
		e.name = p.Name
		e.created = true
	case ExampleRenamed:
		e.name = p.Name
	default:
		return fmt.Errorf("%w: %T", ddd.ErrUnknownEvent, env.Payload)
	}
	return nil
}

// Name is a read accessor (write side stays encapsulated).
func (e *Example) Name() string { return e.name }

// Create raises ExampleCreated after validating the invariant.
func (e *Example) Create(name string) error {
	if e.created {
		return fmt.Errorf("%w: example already created", cqrs.ErrValidation)
	}
	if name == "" {
		return fmt.Errorf("%w: name is required", cqrs.ErrValidation)
	}
	return ddd.Raise[ID, *Example](e, &e.BaseAggregateRoot, ExampleCreated{Name: name}, e.Apply)
}

// Rename raises ExampleRenamed.
func (e *Example) Rename(name string) error {
	if !e.created {
		return fmt.Errorf("%w: example does not exist", cqrs.ErrValidation)
	}
	if name == "" {
		return fmt.Errorf("%w: name is required", cqrs.ErrValidation)
	}
	return ddd.Raise[ID, *Example](e, &e.BaseAggregateRoot, ExampleRenamed{Name: name}, e.Apply)
}
