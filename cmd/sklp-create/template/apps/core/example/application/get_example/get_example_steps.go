package get_example

import (
	"context"
	"errors"
	"fmt"

	"github.com/cucumber/godog"
	"github.com/google/uuid"

	"github.com/lalternativefabrique/eventkit/pkg/cqrs"
	"github.com/lalternativefabrique/eventkit/pkg/db"

	"app/core/example/application/create_example"
	"app/core/example/application/rename_example"
	"app/core/example/domain"
	"app/core/example/infrastructure"
	"app/core/example/projection"
)

// getExampleTestContext wires the full CQRS loop (write side → event stream →
// projection → read side) so scenarios can assert on query results.
type getExampleTestContext struct {
	repo    *infrastructure.Repository
	proj    *projection.ExampleProjection
	handler *Handler
	id      domain.ID
	result  Result
	err     error
}

func (c *getExampleTestContext) anExampleNamedWasCreated(name string) error {
	res, err := create_example.NewHandler(c.repo).Handle(context.Background(), create_example.Command{Name: name})
	if err != nil {
		return err
	}
	c.id = res.ID
	return nil
}

func (c *getExampleTestContext) itWasRenamedTo(name string) error {
	_, err := rename_example.NewHandler(c.repo).Handle(context.Background(), rename_example.Command{ID: c.id, Name: name})
	return err
}

func (c *getExampleTestContext) iFetchItByItsID() error {
	c.result, c.err = c.handler.Handle(context.Background(), Query{ID: c.id})
	return nil
}

func (c *getExampleTestContext) iFetchAnUnknownExample() error {
	c.result, c.err = c.handler.Handle(context.Background(), Query{ID: uuid.NewString()})
	return nil
}

func (c *getExampleTestContext) theQueryShouldSucceed() error {
	if c.err != nil {
		return fmt.Errorf("expected success, got error: %w", c.err)
	}
	return nil
}

func (c *getExampleTestContext) theReturnedExampleShouldHaveName(expected string) error {
	if c.err != nil {
		return fmt.Errorf("expected success, got error: %w", c.err)
	}
	if c.result.Name != expected {
		return fmt.Errorf("expected name %q, got %q", expected, c.result.Name)
	}
	return nil
}

func (c *getExampleTestContext) theQueryShouldFailWithANotFoundError() error {
	if c.err == nil {
		return fmt.Errorf("expected a not-found error, got none")
	}
	if !errors.Is(c.err, cqrs.ErrNotFound) {
		return fmt.Errorf("expected cqrs.ErrNotFound, got: %w", c.err)
	}
	return nil
}

// RegisterSteps wires the step definitions and rebuilds the CQRS loop before
// each scenario, subscribing the projection to the event store.
func RegisterSteps(ctx *godog.ScenarioContext) {
	tc := &getExampleTestContext{}

	ctx.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		store := db.NewInMemoryStore[domain.ID]()
		tc.repo = infrastructure.NewRepository(store)
		tc.proj = projection.New()
		if err := store.Subscribe(ctx, tc.proj.Apply); err != nil {
			return ctx, err
		}
		tc.handler = NewHandler(tc.proj)
		tc.id = ""
		tc.result = Result{}
		tc.err = nil
		return ctx, nil
	})

	ctx.Step(`^an Example named "([^"]*)" was created$`, tc.anExampleNamedWasCreated)
	ctx.Step(`^it was renamed to "([^"]*)"$`, tc.itWasRenamedTo)
	ctx.Step(`^I fetch it by its ID$`, tc.iFetchItByItsID)
	ctx.Step(`^I fetch an unknown Example$`, tc.iFetchAnUnknownExample)
	ctx.Step(`^the query should succeed$`, tc.theQueryShouldSucceed)
	ctx.Step(`^the returned Example should have name "([^"]*)"$`, tc.theReturnedExampleShouldHaveName)
	ctx.Step(`^the query should fail with a not-found error$`, tc.theQueryShouldFailWithANotFoundError)
}
