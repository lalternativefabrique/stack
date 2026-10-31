package rename_example

import (
	"context"
	"errors"
	"fmt"

	"github.com/cucumber/godog"
	"github.com/google/uuid"

	"github.com/lalternativefabrique/eventkit/pkg/cqrs"
	"github.com/lalternativefabrique/eventkit/pkg/db"
	"github.com/lalternativefabrique/eventkit/pkg/ddd"

	"app/core/example/application/create_example"
	"app/core/example/domain"
	"app/core/example/infrastructure"
)

// renameExampleTestContext holds the state shared across the steps of one
// scenario. It is reset in the Before hook so scenarios stay isolated.
type renameExampleTestContext struct {
	store   *db.InMemoryStore[domain.ID]
	repo    *infrastructure.Repository
	handler *Handler
	id      domain.ID
	err     error
}

func (c *renameExampleTestContext) anExampleNamedExists(name string) error {
	res, err := create_example.NewHandler(c.repo).Handle(context.Background(), create_example.Command{Name: name})
	if err != nil {
		return err
	}
	c.id = res.ID
	return nil
}

func (c *renameExampleTestContext) iRenameItTo(name string) error {
	_, c.err = c.handler.Handle(context.Background(), Command{ID: c.id, Name: name})
	return nil
}

func (c *renameExampleTestContext) iTryToRenameAnUnknownExample() error {
	_, c.err = c.handler.Handle(context.Background(), Command{ID: uuid.NewString(), Name: "ghost"})
	return nil
}

func (c *renameExampleTestContext) theRenameShouldSucceed() error {
	if c.err != nil {
		return fmt.Errorf("expected success, got error: %w", c.err)
	}
	return nil
}

func (c *renameExampleTestContext) theExampleShouldHaveName(expected string) error {
	history, err := c.store.Load(context.Background(), c.id)
	if err != nil {
		return err
	}
	e := domain.New(c.id)
	if err := ddd.LoadFromHistory[domain.ID, *domain.Example](e, &e.BaseAggregateRoot, history); err != nil {
		return err
	}
	if e.Name() != expected {
		return fmt.Errorf("expected name %q, got %q", expected, e.Name())
	}
	return nil
}

func (c *renameExampleTestContext) theRenameShouldFailWithAValidationError() error {
	if c.err == nil {
		return fmt.Errorf("expected a validation error, got none")
	}
	if !errors.Is(c.err, cqrs.ErrValidation) {
		return fmt.Errorf("expected cqrs.ErrValidation, got: %w", c.err)
	}
	return nil
}

func (c *renameExampleTestContext) theRenameShouldFailWithANotFoundError() error {
	if c.err == nil {
		return fmt.Errorf("expected a not-found error, got none")
	}
	if !errors.Is(c.err, cqrs.ErrNotFound) {
		return fmt.Errorf("expected cqrs.ErrNotFound, got: %w", c.err)
	}
	return nil
}

func (c *renameExampleTestContext) anEventShouldBeRecorded(eventType string) error {
	history, err := c.store.Load(context.Background(), c.id)
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

// RegisterSteps wires the step definitions and resets state before each scenario.
func RegisterSteps(ctx *godog.ScenarioContext) {
	tc := &renameExampleTestContext{}

	ctx.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		tc.store = db.NewInMemoryStore[domain.ID]()
		tc.repo = infrastructure.NewRepository(tc.store)
		tc.handler = NewHandler(tc.repo)
		tc.id = ""
		tc.err = nil
		return ctx, nil
	})

	ctx.Step(`^an Example named "([^"]*)" exists$`, tc.anExampleNamedExists)
	ctx.Step(`^I rename it to "([^"]*)"$`, tc.iRenameItTo)
	ctx.Step(`^I try to rename it to "([^"]*)"$`, tc.iRenameItTo)
	ctx.Step(`^I try to rename an unknown Example$`, tc.iTryToRenameAnUnknownExample)
	ctx.Step(`^the rename should succeed$`, tc.theRenameShouldSucceed)
	ctx.Step(`^the Example should have name "([^"]*)"$`, tc.theExampleShouldHaveName)
	ctx.Step(`^the rename should fail with a validation error$`, tc.theRenameShouldFailWithAValidationError)
	ctx.Step(`^the rename should fail with a not-found error$`, tc.theRenameShouldFailWithANotFoundError)
	ctx.Step(`^an "([^"]*)" event should be recorded$`, tc.anEventShouldBeRecorded)
}
