package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	iamapi "github.com/intivai/backend/internal/iam/api"
	notifapp "github.com/intivai/backend/internal/notification/application"
	sharederr "github.com/intivai/backend/internal/shared/errors"
	"github.com/intivai/backend/internal/shared/httpapi"
)

type NotificationHandler struct {
	svc *notifapp.NotificationService
}

func NewNotificationHandler(svc *notifapp.NotificationService) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

// List — GET /notifications
func (h *NotificationHandler) List(c *fiber.Ctx) error {
	actor, ok := iamapi.Actor(c)
	if !ok {
		return httpapi.Error(c, sharederr.NewDomainError("UNAUTHORIZED", "unauthorized"))
	}
	notifications, err := h.svc.List(c.UserContext(), actor.OrgID, 30)
	if err != nil {
		return httpapi.Error(c, err)
	}
	unread, err := h.svc.UnreadCount(c.UserContext(), actor.OrgID)
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.OK(c, fiber.Map{
		"notifications": notifications,
		"unread_count":  unread,
	})
}

// MarkRead — PATCH /notifications/:id/read
func (h *NotificationHandler) MarkRead(c *fiber.Ctx) error {
	actor, ok := iamapi.Actor(c)
	if !ok {
		return httpapi.Error(c, sharederr.NewDomainError("UNAUTHORIZED", "unauthorized"))
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("INVALID_INPUT", "invalid notification id"))
	}
	if err := h.svc.MarkRead(c.UserContext(), actor.OrgID, id); err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.OK(c, fiber.Map{"read": true})
}

// MarkAllRead — POST /notifications/read-all
func (h *NotificationHandler) MarkAllRead(c *fiber.Ctx) error {
	actor, ok := iamapi.Actor(c)
	if !ok {
		return httpapi.Error(c, sharederr.NewDomainError("UNAUTHORIZED", "unauthorized"))
	}
	if err := h.svc.MarkAllRead(c.UserContext(), actor.OrgID); err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.OK(c, fiber.Map{"marked_all_read": true})
}
