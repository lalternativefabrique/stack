// Package middleware names the person behind a protected route: a token
// urbangate issued for them (urbangate ADR 0009), read by go/websession, then
// mapped to the local account row everything in this core is keyed on.
package middleware

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/lalternative/packages/go/websession"
)

type User struct {
	ID         string
	IdentityID string
	Email      string
	Name       string
	Role       string
}

// PersonResolver turns a raw token into the person urbangate signed it for.
// *websession.Guard's Resolve is the one main wires.
type PersonResolver func(ctx context.Context, raw string) (websession.User, error)

// AccountResolver names the local account a person is scoped by.
// account.Service.ResolvePerson is the one main wires.
type AccountResolver func(ctx context.Context, identityID, email, name string) (string, error)

// NewGuard trusts urbangate for the people it signs, from OIDC_ISSUER_URL and
// OIDC_AUDIENCE. It exits rather than booting unable to verify: such a core
// would refuse every signed-in person anyway, and failing at boot names the
// missing variable instead of 401-ing one request at a time.
func NewGuard() *websession.Guard {
	issuer := strings.TrimRight(os.Getenv("OIDC_ISSUER_URL"), "/")
	if issuer == "" {
		issuer = "https://id.urbangate.dev"
	}
	audience := os.Getenv("OIDC_AUDIENCE")
	if audience == "" {
		audience = "__APP_NAME__"
	}
	g, err := websession.New(websession.Config{Product: audience, Urbangate: issuer})
	if err != nil {
		log.Fatalf("auth: %v", err)
	}
	return g
}

// RequireAuth verifies the bearer token (or the `<product>_token` cookie
// @lalternative/auth sets) as one urbangate issued for a person, and stores
// the resolved User in the echo context.
func RequireAuth(resolvePerson PersonResolver, resolveAccount AccountResolver) echo.MiddlewareFunc {
	product := os.Getenv("OIDC_AUDIENCE")
	if product == "" {
		product = "__APP_NAME__"
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			raw := tokenFromRequest(c, product)
			if raw == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
			}
			ctx := c.Request().Context()
			person, err := resolvePerson(ctx, raw)
			if errors.Is(err, websession.ErrUnavailable) {
				return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "identity_provider_unavailable"})
			}
			if err != nil || person.IdentityID == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "unauthenticated")
			}
			id, err := resolveAccount(ctx, person.IdentityID, person.Email, person.Name)
			if err != nil {
				return echo.NewHTTPError(http.StatusServiceUnavailable, "account unavailable")
			}
			c.Set("user", User{
				ID:         id,
				IdentityID: person.IdentityID,
				Email:      person.Email,
				Name:       person.Name,
				Role:       person.Role,
			})
			return next(c)
		}
	}
}

// RequireRole refuses everyone but this product's holders of role, 403.
func RequireRole(role string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			u, ok := GetUser(c)
			if !ok || u.Role != role {
				return echo.NewHTTPError(http.StatusForbidden, "forbidden")
			}
			return next(c)
		}
	}
}

func tokenFromRequest(c echo.Context, product string) string {
	if auth := c.Request().Header.Get("Authorization"); len(auth) > 7 && auth[:7] == "Bearer " {
		return auth[7:]
	}
	if cookie, err := c.Cookie(product + "_token"); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	return ""
}

func GetUser(c echo.Context) (User, bool) {
	v := c.Get("user")
	if v == nil {
		return User{}, false
	}
	u, ok := v.(User)
	return u, ok
}
