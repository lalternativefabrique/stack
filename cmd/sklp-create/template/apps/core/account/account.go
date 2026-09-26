// Package account is the core's account context. Authentication is
// urbangate's (see apps/core/middleware); what lives here is the local
// account row a person is scoped by, and its erasure.
//
// It is a plain service (no CQRS/event-sourcing like the example context) on
// purpose. Grow it into the DDD layout only if account state moves into the
// core.
package account

import (
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool   *pgxpool.Pool
	people sync.Map
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}
