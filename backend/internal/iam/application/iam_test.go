package application

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	iamdomain "github.com/intivai/backend/internal/iam/domain"
)

type memRepo struct {
	orgs  []*iamdomain.Org
	users []*iamdomain.User
}

func (m *memRepo) CreateOrg(_ context.Context, org *iamdomain.Org) error {
	m.orgs = append(m.orgs, org)
	return nil
}
func (m *memRepo) GetOrg(_ context.Context, id uuid.UUID) (*iamdomain.Org, error) {
	for _, o := range m.orgs {
		if o.ID == id {
			return o, nil
		}
	}
	return nil, iamdomain.ErrNotFound
}
func (m *memRepo) GetOrgBySlug(_ context.Context, slug string) (*iamdomain.Org, error) {
	for _, o := range m.orgs {
		if o.Slug == slug {
			return o, nil
		}
	}
	return nil, iamdomain.ErrNotFound
}
func (m *memRepo) CreateUser(_ context.Context, u *iamdomain.User) error {
	m.users = append(m.users, u)
	return nil
}
func (m *memRepo) GetUserByID(_ context.Context, id uuid.UUID) (*iamdomain.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, iamdomain.ErrNotFound
}
func (m *memRepo) GetUserByEmail(_ context.Context, orgID uuid.UUID, email string) (*iamdomain.User, error) {
	for _, u := range m.users {
		if u.OrgID == orgID && strings.EqualFold(u.Email, email) {
			return u, nil
		}
	}
	return nil, iamdomain.ErrNotFound
}
func (m *memRepo) ListUsers(_ context.Context, orgID uuid.UUID) ([]*iamdomain.User, error) {
	out := []*iamdomain.User{}
	for _, u := range m.users {
		if u.OrgID == orgID {
			out = append(out, u)
		}
	}
	return out, nil
}

func (m *memRepo) FindLoginIdentity(_ context.Context, orgSlug, email string) (*iamdomain.LoginIdentity, error) {
	for _, o := range m.orgs {
		if o.Slug != orgSlug {
			continue
		}
		for _, u := range m.users {
			if u.OrgID == o.ID && strings.EqualFold(u.Email, email) {
				hash := u.PasswordHash
				return &iamdomain.LoginIdentity{
					OrgID:        u.OrgID,
					UserID:       u.ID,
					Email:        u.Email,
					PasswordHash: &hash,
					Role:         u.Role,
					AuthProvider: u.AuthProvider,
					CreatedAt:    u.CreatedAt,
				}, nil
			}
		}
	}
	return nil, iamdomain.ErrNotFound
}

var _ iamdomain.IAMRepository = (*memRepo)(nil)

type fakeHasher struct{}

func (fakeHasher) Hash(plain string) (string, error) { return "h:" + plain, nil }
func (fakeHasher) Verify(hash, plain string) bool    { return hash == "h:"+plain }

type fakeTokens struct{}

func (fakeTokens) Issue(_ uuid.UUID, _ uuid.UUID, role, _ string, _ time.Duration, _ map[string]any) (string, error) {
	return "token-" + role, nil
}
func (fakeTokens) Parse(_ string) (*Claims, error) { return &Claims{}, nil }

type noopTx struct{}

func (noopTx) RunInTx(ctx context.Context, _ *uuid.UUID, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func TestRegisterOrgCreatesAdmin(t *testing.T) {
	repo := &memRepo{}
	uc := NewRegisterOrg(repo, fakeHasher{}, noopTx{})

	res, err := uc.Execute(context.Background(), RegisterOrgCommand{
		Name: "Acme", Slug: "acme", AdminEmail: "A@acme.io", AdminPassword: "secret123",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if res.Slug != "acme" {
		t.Errorf("slug = %q, want acme", res.Slug)
	}
	if len(repo.orgs) != 1 || len(repo.users) != 1 {
		t.Fatalf("expected 1 org + 1 user, got %d/%d", len(repo.orgs), len(repo.users))
	}
	if repo.users[0].Role != iamdomain.RoleAdmin {
		t.Errorf("role = %q, want admin", repo.users[0].Role)
	}
	if repo.users[0].Email != "a@acme.io" {
		t.Errorf("email = %q, want lowercase", repo.users[0].Email)
	}
}

func TestAuthenticateIssuesToken(t *testing.T) {
	repo := &memRepo{}
	reg := NewRegisterOrg(repo, fakeHasher{}, noopTx{})
	_, err := reg.Execute(context.Background(), RegisterOrgCommand{
		Name: "Acme", Slug: "acme", AdminEmail: "a@acme.io", AdminPassword: "secret123",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	auth := NewAuthenticate(repo, fakeHasher{}, fakeTokens{}, time.Hour)
	res, err := auth.Execute(context.Background(), AuthenticateCommand{OrgSlug: "acme", Email: "a@acme.io", Password: "secret123"})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if res.Token != "token-admin" {
		t.Errorf("token = %q", res.Token)
	}
}

func TestAuthenticateWrongPassword(t *testing.T) {
	repo := &memRepo{}
	reg := NewRegisterOrg(repo, fakeHasher{}, noopTx{})
	_, _ = reg.Execute(context.Background(), RegisterOrgCommand{
		Name: "Acme", Slug: "acme", AdminEmail: "a@acme.io", AdminPassword: "secret123",
	})

	auth := NewAuthenticate(repo, fakeHasher{}, fakeTokens{}, time.Hour)
	_, err := auth.Execute(context.Background(), AuthenticateCommand{OrgSlug: "acme", Email: "a@acme.io", Password: "wrong"})
	if err == nil {
		t.Fatal("expected auth failure")
	}
}

func TestCreateUserValidationAndAuthz(t *testing.T) {
	repo := &memRepo{}
	admin := AuthContext{UserID: uuid.New(), OrgID: uuid.New(), Role: string(iamdomain.RoleAdmin)}
	uc := NewCreateUser(repo, fakeHasher{})

	if _, err := uc.Execute(context.Background(), admin, CreateUserCommand{OrgID: admin.OrgID, Email: "r@x.io", Role: "recruiter", Password: "short"}); err == nil {
		t.Fatal("weak password accepted")
	}
	if _, err := uc.Execute(context.Background(), admin, CreateUserCommand{OrgID: admin.OrgID, Email: "r@x.io", Role: "bogus", Password: "secret123"}); err == nil {
		t.Fatal("invalid role accepted")
	}
	res, err := uc.Execute(context.Background(), admin, CreateUserCommand{OrgID: admin.OrgID, Email: "  R@X.IO ", Role: "recruiter", Password: "secret123"})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if res.Role != "recruiter" {
		t.Fatalf("role = %s", res.Role)
	}
	if repo.users[0].Email != "r@x.io" {
		t.Fatalf("email not normalized: %s", repo.users[0].Email)
	}

	// member role cannot create users
	member := AuthContext{UserID: uuid.New(), OrgID: admin.OrgID, Role: string(iamdomain.RoleMember)}
	if _, err := uc.Execute(context.Background(), member, CreateUserCommand{OrgID: admin.OrgID, Email: "x@x.io", Role: "member", Password: "secret123"}); err == nil {
		t.Fatal("member role created a user")
	}
}

func TestAuthorize(t *testing.T) {
	if err := Authorize(AuthContext{Role: string(iamdomain.RoleAdmin)}, iamdomain.RoleAdmin, iamdomain.RoleRecruiter); err != nil {
		t.Fatal(err)
	}
	if err := Authorize(AuthContext{Role: string(iamdomain.RoleMember)}, iamdomain.RoleAdmin); err == nil {
		t.Fatal("member authorized as admin")
	}
}
