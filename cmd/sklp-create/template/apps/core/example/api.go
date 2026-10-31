package example

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/lalternativefabrique/eventkit/pkg/cqrs"

	"app/core/example/application/create_example"
	"app/core/example/application/get_example"
	"app/core/example/application/rename_example"
)

// Create godoc
// @Summary  Create an example
// @Tags     examples
// @Accept   json
// @Produce  json
// @Param    body  body      CreateRequest  true  "Example to create"
// @Success  201   {object}  ExampleDTO
// @Failure  400   {object}  echo.HTTPError
// @Security BearerAuth
// @Router   /examples [post]
// @ID       createExample
func (s *Service) Create(c echo.Context) error {
	var req CreateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	res, err := cqrs.Execute[create_example.Command, create_example.Result](
		c.Request().Context(), s.commands, create_example.Command{Name: req.Name},
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(http.StatusCreated, ExampleDTO{ID: res.ID, Name: req.Name})
}

// Get godoc
// @Summary  Get an example by id
// @Tags     examples
// @Produce  json
// @Param    id   path      string  true  "Example id"
// @Success  200  {object}  ExampleDTO
// @Failure  404  {object}  echo.HTTPError
// @Security BearerAuth
// @Router   /examples/{id} [get]
// @ID       getExample
func (s *Service) Get(c echo.Context) error {
	res, err := cqrs.Ask[get_example.Query, get_example.Result](
		c.Request().Context(), s.queries, get_example.Query{ID: c.Param("id")},
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}
	return c.JSON(http.StatusOK, ExampleDTO{ID: res.ID, Name: res.Name})
}

// Rename godoc
// @Summary  Rename an example
// @Tags     examples
// @Accept   json
// @Produce  json
// @Param    id    path      string         true  "Example id"
// @Param    body  body      RenameRequest  true  "New name"
// @Success  204
// @Failure  400   {object}  echo.HTTPError
// @Failure  404   {object}  echo.HTTPError
// @Security BearerAuth
// @Router   /examples/{id} [patch]
// @ID       renameExample
func (s *Service) Rename(c echo.Context) error {
	var req RenameRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if _, err := cqrs.Execute[rename_example.Command, rename_example.Result](
		c.Request().Context(), s.commands, rename_example.Command{ID: c.Param("id"), Name: req.Name},
	); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}
