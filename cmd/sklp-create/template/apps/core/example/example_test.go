package example_test

import (
	"context"
	"testing"

	"github.com/lalternativefabrique/eventkit/pkg/db"

	"app/core/example/application/create_example"
	"app/core/example/application/get_example"
	"app/core/example/domain"
	"app/core/example/infrastructure"
	"app/core/example/projection"
)

// End-to-end CQRS loop, wired by hand (mirrors bootstrap.go without DI):
// command → aggregate → event → projection → query.
func TestCQRSLoop_CreateThenGet(t *testing.T) {
	ctx := context.Background()

	store := db.NewInMemoryStore[domain.ID]()
	repo := infrastructure.NewRepository(store)
	proj := projection.New()
	if err := store.Subscribe(ctx, proj.Apply); err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	// Write side: create.
	created, err := create_example.NewHandler(repo).Handle(ctx, create_example.Command{Name: "alpha"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Read side: the projection has been updated from the event stream.
	got, err := get_example.NewHandler(proj).Handle(ctx, get_example.Query{ID: created.ID})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "alpha" {
		t.Fatalf("projected name = %q, want alpha", got.Name)
	}
}
