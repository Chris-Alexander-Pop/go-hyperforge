package handler

import (
	"context"
	"net/http"

	"github.com/chris-alexander-pop/system-design-library/pkg/events"
	"github.com/labstack/echo/v4"
)

type Handler struct {
	events events.EventStore
}

func New(evt events.EventStore) *Handler {
	return &Handler{events: evt}
}

func (h *Handler) Register(e *echo.Echo) {
	e.POST("/users", h.CreateUser)
}

func (h *Handler) CreateUser(c echo.Context) error {
	userID := "user-123"

	err := h.events.Publish(context.Background(), "user.registered", []byte(`{"user_id":"`+userID+`"}`))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]string{"id": userID})
}
