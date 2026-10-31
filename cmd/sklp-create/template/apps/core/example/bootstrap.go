// Package example wires the Example bounded context: DI container, event store,
// repository, projection, and the typed CQRS command/query routers. The HTTP
// surface lives in api.go. Modelled on go/eda/examples/banking.
package example

import (
	"context"
	"log/slog"

	"github.com/labstack/echo/v4"

	"github.com/lalternativefabrique/eventkit/pkg/cqrs"
	"github.com/lalternativefabrique/eventkit/pkg/db"
	"github.com/lalternativefabrique/eventkit/pkg/di"
	"github.com/lalternativefabrique/eventkit/pkg/logger"
	"github.com/lalternativefabrique/eventkit/pkg/obs"

	"app/core/example/application/create_example"
	"app/core/example/application/get_example"
	"app/core/example/application/rename_example"
	"app/core/example/domain"
	"app/core/example/infrastructure"
	"app/core/example/projection"
)

// Service is the context's HTTP-facing facade. It holds the CQRS routers; the
// Echo handlers in api.go dispatch through cqrs.Execute / cqrs.Ask.
type Service struct {
	commands *cqrs.CommandRouter
	queries  *cqrs.QueryRouter
}

// NewService builds the DI registry, starts it, wires the projection to the
// event stream, and returns the ready-to-serve facade.
func NewService(ctx context.Context) (*Service, error) {
	reg := buildRegistry()
	if err := reg.Start(ctx); err != nil {
		return nil, err
	}

	store := di.MustResolve[*db.InMemoryStore[domain.ID]](reg)
	proj := di.MustResolve[*projection.ExampleProjection](reg)
	// Feed the read model from the event stream.
	if err := store.Subscribe(ctx, proj.Apply); err != nil {
		return nil, err
	}

	return &Service{
		commands: di.MustResolve[*cqrs.CommandRouter](reg),
		queries:  di.MustResolve[*cqrs.QueryRouter](reg),
	}, nil
}

// RegisterRoutes mounts the context's HTTP endpoints on the given group.
func (s *Service) RegisterRoutes(g *echo.Group) {
	g.POST("/examples", s.Create)
	g.GET("/examples/:id", s.Get)
	g.PATCH("/examples/:id", s.Rename)
}

// buildRegistry declares every dependency of the context.
func buildRegistry() *di.Registry {
	r := di.New()

	di.Provide[logger.Logger](r, func(_ *di.Resolver) (logger.Logger, error) {
		return logger.NewJSONSlogLogger(slog.LevelInfo), nil
	})

	di.Provide(r, func(_ *di.Resolver) (*db.InMemoryStore[domain.ID], error) {
		return db.NewInMemoryStore[domain.ID](), nil
	})

	di.Provide(r, func(rv *di.Resolver) (*infrastructure.Repository, error) {
		store, err := di.From[*db.InMemoryStore[domain.ID]](rv)
		if err != nil {
			return nil, err
		}
		return infrastructure.NewRepository(store), nil
	})

	di.Provide(r, func(_ *di.Resolver) (*projection.ExampleProjection, error) {
		return projection.New(), nil
	})

	di.Provide(r, func(rv *di.Resolver) (*cqrs.CommandRouter, error) {
		repo := di.MustFrom[*infrastructure.Repository](rv)
		log := di.MustFrom[logger.Logger](rv)
		router := cqrs.NewCommandRouter()

		createMW := cqrs.Chain(
			cqrs.RecoveryMiddleware[create_example.Command, create_example.Result](),
			obs.LoggingMiddleware[create_example.Command, create_example.Result](log),
		)
		cqrs.RegisterCommandHandler[create_example.Command, create_example.Result](router,
			cqrs.TypedCommandHandlerFunc[create_example.Command, create_example.Result](
				createMW(create_example.NewHandler(repo).Handle),
			),
		)

		renameMW := cqrs.Chain(
			cqrs.RecoveryMiddleware[rename_example.Command, rename_example.Result](),
			obs.LoggingMiddleware[rename_example.Command, rename_example.Result](log),
		)
		cqrs.RegisterCommandHandler[rename_example.Command, rename_example.Result](router,
			cqrs.TypedCommandHandlerFunc[rename_example.Command, rename_example.Result](
				renameMW(rename_example.NewHandler(repo).Handle),
			),
		)

		return router, nil
	})

	di.Provide(r, func(rv *di.Resolver) (*cqrs.QueryRouter, error) {
		proj := di.MustFrom[*projection.ExampleProjection](rv)
		router := cqrs.NewQueryRouter()
		cqrs.RegisterQueryHandler[get_example.Query, get_example.Result](router,
			get_example.NewHandler(proj),
		)
		return router, nil
	})

	return r
}
