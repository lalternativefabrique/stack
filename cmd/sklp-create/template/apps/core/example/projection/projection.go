// Package projection holds the Example read model. It is updated from the
// event stream (store.Subscribe → Apply) and queried by the query side. An
// in-memory map here; back it with SQL read tables in a real context.
package projection

import (
	"context"
	"sync"

	"github.com/lalternativefabrique/eventkit/pkg/ddd"

	"app/core/example/domain"
)

// View is the read-model shape returned to queries.
type View struct {
	ID   domain.ID
	Name string
}

// ExampleProjection is a tenant-agnostic read model of Examples by id.
type ExampleProjection struct {
	mu    sync.RWMutex
	views map[domain.ID]View
}

// New returns an empty projection.
func New() *ExampleProjection {
	return &ExampleProjection{views: make(map[domain.ID]View)}
}

// Apply folds one event into the read model. Wire it to the store's stream
// via store.Subscribe(ctx, proj.Apply).
func (p *ExampleProjection) Apply(_ context.Context, env ddd.EventEnvelope[domain.ID]) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	switch e := env.Payload.(type) {
	case domain.ExampleCreated:
		p.views[env.AggregateID] = View{ID: env.AggregateID, Name: e.Name}
	case domain.ExampleRenamed:
		v := p.views[env.AggregateID]
		v.ID = env.AggregateID
		v.Name = e.Name
		p.views[env.AggregateID] = v
	}
	return nil
}

// Get returns the view for an id and whether it exists.
func (p *ExampleProjection) Get(id domain.ID) (View, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	v, ok := p.views[id]
	return v, ok
}
