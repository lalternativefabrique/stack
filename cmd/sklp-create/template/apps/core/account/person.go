package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// ResolvePerson names the local account for a person urbangate signed. One
// urbangate meets for the first time is opened under the identity id itself,
// so "user".id and the identity id are the same value from day one.
func (s *Service) ResolvePerson(ctx context.Context, identityID, email, name string) (string, error) {
	if id, ok := s.people.Load(identityID); ok {
		return id.(string), nil
	}
	id, err := s.resolvePerson(ctx, identityID, email, name)
	if err != nil {
		return "", err
	}
	s.people.Store(identityID, id)
	return id, nil
}

func (s *Service) resolvePerson(ctx context.Context, identityID, email, name string) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `SELECT id FROM "user" WHERE identity_id = $1`, identityID).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("resolve person: %w", err)
	}
	if name == "" {
		name = email
	}
	err = s.pool.QueryRow(ctx, `
		INSERT INTO "user" (id, identity_id, email, name)
		VALUES ($1, $1, $2, $3)
		ON CONFLICT (id) DO UPDATE SET email = EXCLUDED.email, updated_at = NOW()
		RETURNING id`, identityID, email, name).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("open account: %w", err)
	}
	return id, nil
}
