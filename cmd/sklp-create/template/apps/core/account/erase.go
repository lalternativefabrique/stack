package account

import (
	"context"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"app/core/middleware"
)

// Erase deletes everything the account holds. Rows of other contexts keyed on
// "user".id go with it through ON DELETE CASCADE; a context whose rows are
// keyed without a foreign key deletes them here first. It is idempotent: the
// web retries the whole deletion when a later step failed.
func (s *Service) Erase(ctx context.Context, userID string) error {
	if _, err := s.pool.Exec(ctx, `DELETE FROM "user" WHERE id = $1`, userID); err != nil {
		return fmt.Errorf("erase account: %w", err)
	}
	s.people.Range(func(key, value any) bool {
		if value == userID {
			s.people.Delete(key)
		}
		return true
	})
	return nil
}

// DeleteAccount godoc
// @Summary  Delete the caller's account and everything it holds
// @Tags     account
// @Success  204
// @Failure  401  {object}  map[string]string
// @Router   /account [delete]
// @ID       deleteAccount
func (s *Service) DeleteAccount(c echo.Context) error {
	user, ok := middleware.GetUser(c)
	if !ok || user.ID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	if err := s.Erase(c.Request().Context(), user.ID); err != nil {
		c.Logger().Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "deletion failed"})
	}
	return c.NoContent(http.StatusNoContent)
}

// Me godoc
// @Summary  The signed-in person
// @Tags     account
// @Produce  json
// @Success  200  {object}  MeDTO
// @Router   /me [get]
// @ID       me
func (s *Service) Me(c echo.Context) error {
	u, _ := middleware.GetUser(c)
	return c.JSON(http.StatusOK, MeDTO{UserID: u.ID, Email: u.Email, Name: u.Name, Role: u.Role})
}

type MeDTO struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	Role   string `json:"role"`
}

// RegisterRoutes mounts what only the signed-in person may do.
func (s *Service) RegisterRoutes(g *echo.Group) {
	g.GET("/me", s.Me)
	g.DELETE("/account", s.DeleteAccount)
}
