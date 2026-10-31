// Package get_example is the read-side use case: fetch an Example from the
// projection (read model), never from the aggregate.
package get_example

import (
	"context"
	"fmt"

	"github.com/lalternativefabrique/eventkit/pkg/cqrs"

	"app/core/example/domain"
	"app/core/example/projection"
)

// Query asks for one Example by id.
type Query struct {
	ID domain.ID
}

// Result is the read-model view.
type Result struct {
	ID   domain.ID
	Name string
}

// Handler answers the query from the projection.
type Handler struct {
	proj *projection.ExampleProjection
}

func NewHandler(proj *projection.ExampleProjection) *Handler {
	return &Handler{proj: proj}
}

func (h *Handler) Handle(_ context.Context, q Query) (Result, error) {
	v, ok := h.proj.Get(q.ID)
	if !ok {
		return Result{}, fmt.Errorf("%w: %s", cqrs.ErrNotFound, q.ID)
	}
	return Result{ID: v.ID, Name: v.Name}, nil
}
