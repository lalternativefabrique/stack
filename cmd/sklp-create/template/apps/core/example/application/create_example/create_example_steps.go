package create_example

import (
	"context"
	"errors"
	"fmt"

	"github.com/cucumber/godog"

	"github.com/lalternativefabrique/eventkit/pkg/cqrs"
	"github.com/lalternativefabrique/eventkit/pkg/db"
	"github.com/lalternativefabrique/eventkit/pkg/ddd"

	"app/core/example/domain"
	"app/core/example/infrastructure"
)

// replay rebuilds an Example aggregate from its event history so a scenario
// can assert on the resulting write-side state.
func replay(e *domain.Example, history []ddd.EventEnvelope[domain.ID]) error {
	return ddd.LoadFromHistory[domain.ID, *domain.Example](e, &e.BaseAggregateRoot, history)
}

// createExampleTestContext holds the state shared across the steps of one
// scenario. It is reset in the Before hook so scenarios stay isolated.
type createExampleTestContext struct {
	store   *db.InMemoryStore[domain.ID]
	handler *Handler
	result  Result
	err     error
}

func (c *createExampleTestContext) iCreateANewExampleNamed(name string) error {
	c.result, c.err = c.handler.Handle(context.Background(), Command{Name: name})
	return nil
}

func (c *createExampleTestContext) iTryToCreateANewExampleWithNoName() error {
	c.result, c.err = c.handler.Handle(context.Background(), Command{Name: ""})
	return nil
}

func (c *createExampleTestContext) theCreationShouldSucceed() error {
	if c.err != nil {
		return fmt.Errorf("expected success, got error: %w", c.err)
	}
	if c.result.ID == "" {
		return fmt.Errorf("expected a non-empty aggregate ID")
	}
	return nil
}

func (c *createExampleTestContext) theExampleShouldHaveName(expected string) error {
	history, err := c.store.Load(context.Background(), c.result.ID)
	if err != nil {
		return err
	}
	e := domain.New(c.result.ID)
	if err := replay(e, history); err != nil {
		return err
	}
	if e.Name() != expected {
		return fmt.Errorf("expected name %q, got %q", expected, e.Name())
	}
	return nil
}

func (c *createExampleTestContext) theCreationShouldFailWithAValidationError() error {
	if c.err == nil {
		return fmt.Errorf("expected a validation error, got none")
	}
	if !errors.Is(c.err, cqrs.ErrValidation) {
		return fmt.Errorf("expected cqrs.ErrValidation, got: %w", c.err)
	}
	return nil
}

func (c *createExampleTestContext) anEventShouldBeRecorded(eventType string) error {
	history, err := c.store.Load(context.Background(), c.result.ID)
	if err != nil {
		return err
	}
	for _, env := range history {
		if env.EventType == eventType {
			return nil
		}
	}
	return fmt.Errorf("event %q not found in stream", eventType)
}

func (c *createExampleTestContext) theEventShouldCarryTheExampleID() error {
	history, err := c.store.Load(context.Background(), c.result.ID)
	if err != nil {
		return err
	}
	if len(history) == 0 {
		return fmt.Errorf("no events recorded")
	}
	if history[0].AggregateID != c.result.ID {
		return fmt.Errorf("expected aggregate ID %q, got %q", c.result.ID, history[0].AggregateID)
	}
	return nil
}

func (c *createExampleTestContext) theEventShouldCarryTheName(expected string) error {
	history, err := c.store.Load(context.Background(), c.result.ID)
	if err != nil {
		return err
	}
	if len(history) == 0 {
		return fmt.Errorf("no events recorded")
	}
	created, ok := history[0].Payload.(domain.ExampleCreated)
	if !ok {
		return fmt.Errorf("expected ExampleCreated payload, got %T", history[0].Payload)
	}
	if created.Name != expected {
		return fmt.Errorf("expected name %q, got %q", expected, created.Name)
	}
	return nil
}

// RegisterSteps wires the step definitions and resets state before each scenario.
func RegisterSteps(ctx *godog.ScenarioContext) {
	tc := &createExampleTestContext{}

	ctx.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		tc.store = db.NewInMemoryStore[domain.ID]()
		tc.handler = NewHandler(infrastructure.NewRepository(tc.store))
		tc.result = Result{}
		tc.err = nil
		return ctx, nil
	})

	ctx.Step(`^I create a new Example named "([^"]*)"$`, tc.iCreateANewExampleNamed)
	ctx.Step(`^I try to create a new Example with no name$`, tc.iTryToCreateANewExampleWithNoName)
	ctx.Step(`^the creation should succeed$`, tc.theCreationShouldSucceed)
	ctx.Step(`^the Example should have name "([^"]*)"$`, tc.theExampleShouldHaveName)
	ctx.Step(`^the creation should fail with a validation error$`, tc.theCreationShouldFailWithAValidationError)
	ctx.Step(`^an "([^"]*)" event should be recorded$`, tc.anEventShouldBeRecorded)
	ctx.Step(`^the event should carry the Example ID$`, tc.theEventShouldCarryTheExampleID)
	ctx.Step(`^the event should carry the name "([^"]*)"$`, tc.theEventShouldCarryTheName)
}
