package api

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/intivai/backend/internal/iam/application"
	sharederr "github.com/intivai/backend/internal/shared/errors"
	"github.com/intivai/backend/internal/shared/httpapi"
)

type AuthHandler struct {
	registerOrg  *application.RegisterOrg
	authenticate *application.Authenticate
	createUser   *application.CreateUser
}

func NewAuthHandler(
	registerOrg *application.RegisterOrg,
	authenticate *application.Authenticate,
	createUser *application.CreateUser,
) *AuthHandler {
	return &AuthHandler{registerOrg: registerOrg, authenticate: authenticate, createUser: createUser}
}

type registerRequest struct {
	Name          string `json:"name"`
	Slug          string `json:"slug"`
	AdminEmail    string `json:"admin_email"`
	AdminPassword string `json:"admin_password"`
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req registerRequest
	if err := c.BodyParser(&req); err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "invalid body"))
	}
	result, err := h.registerOrg.Execute(c.UserContext(), application.RegisterOrgCommand{
		Name:          req.Name,
		Slug:          req.Slug,
		AdminEmail:    req.AdminEmail,
		AdminPassword: req.AdminPassword,
	})
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.Created(c, result)
}

type loginRequest struct {
	OrgSlug  string `json:"org_slug"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "invalid body"))
	}
	result, err := h.authenticate.Execute(c.UserContext(), application.AuthenticateCommand{
		OrgSlug:  req.OrgSlug,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.OK(c, result)
}

func (h *AuthHandler) Me(c *fiber.Ctx) error {
	actor, err := RequireActor(c)
	if err != nil {
		return err
	}
	return httpapi.OK(c, fiber.Map{
		"user_id": actor.UserID,
		"org_id":  actor.OrgID,
		"role":    actor.Role,
		"time":    time.Now().UTC(),
	})
}

type createUserRequest struct {
	Email    string `json:"email"`
	Role     string `json:"role"`
	Password string `json:"password"`
}

func (h *AuthHandler) CreateUser(c *fiber.Ctx) error {
	actor, err := RequireActor(c)
	if err != nil {
		return err
	}
	var req createUserRequest
	if err := c.BodyParser(&req); err != nil {
		return httpapi.Error(c, sharederr.NewDomainError("BAD_REQUEST", "invalid body"))
	}
	result, err := h.createUser.Execute(c.UserContext(), actor, application.CreateUserCommand{
		OrgID:    actor.OrgID,
		Email:    req.Email,
		Role:     req.Role,
		Password: req.Password,
	})
	if err != nil {
		return httpapi.Error(c, err)
	}
	return httpapi.Created(c, result)
}
