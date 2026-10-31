// Package infrastructure holds the write-side adapter for the Example context:
// an event-sourced repository on top of go/eda's in-memory event store.
//
// This is an EXAMPLE store (state is lost on restart). To persist, swap
// db.InMemoryStore for a real db.Store backed by Postgres — the repository API
// (Load/Save with optimistic concurrency) stays the same.
package infrastructure

import (
	"context"

	"github.com/lalternativefabrique/eventkit/pkg/db"
	"github.com/lalternativefabrique/eventkit/pkg/ddd"

	"app/core/example/domain"
)

// Repository loads and saves Example aggregates as event streams.
type Repository struct {
	store *db.InMemoryStore[domain.ID]
}

// NewRepository wires the repository to an event store.
func NewRepository(store *db.InMemoryStore[domain.ID]) *Repository {
	return &Repository{store: store}
}

// Load rebuilds an Example from its event history.
func (r *Repository) Load(ctx context.Context, id domain.ID) (*domain.Example, error) {
	history, err := r.store.Load(ctx, id)
	if err != nil {
		return nil, err
	}
	e := domain.New(id)
	if err := ddd.LoadFromHistory[domain.ID, *domain.Example](e, &e.BaseAggregateRoot, history); err != nil {
		return nil, err
	}
	return e, nil
}

// Save appends the aggregate's uncommitted events with optimistic concurrency.
func (r *Repository) Save(ctx context.Context, e *domain.Example) error {
	pending := e.Uncommitted()
	if len(pending) == 0 {
		return nil
	}
	expected := e.Version() - len(pending)
	if err := r.store.Save(ctx, e.ID(), expected, pending); err != nil {
		return err
	}
	e.MarkCommitted()
	return nil
}
