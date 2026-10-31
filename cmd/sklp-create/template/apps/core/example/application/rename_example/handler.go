// Package rename_example is the write-side use case: rename an Example.
package rename_example

import (
	"context"
	"fmt"

	"github.com/lalternativefabrique/eventkit/pkg/cqrs"

	"app/core/example/domain"
	"app/core/example/infrastructure"
)

// Command is the rename-example command.
type Command struct {
	ID   domain.ID
	Name string
}

// Result is empty (the rename has no payload to return).
type Result struct{}

// Handler executes the command: load → mutate → save.
type Handler struct {
	repo *infrastructure.Repository
}

func NewHandler(repo *infrastructure.Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) Handle(ctx context.Context, cmd Command) (Result, error) {
	e, err := h.repo.Load(ctx, cmd.ID)
	if err != nil {
		return Result{}, fmt.Errorf("%w: %s", cqrs.ErrNotFound, cmd.ID)
	}
	if err := e.Rename(cmd.Name); err != nil {
		return Result{}, err
	}
	if err := h.repo.Save(ctx, e); err != nil {
		return Result{}, err
	}
	return Result{}, nil
}
