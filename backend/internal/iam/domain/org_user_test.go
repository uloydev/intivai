package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewOrgValidation(t *testing.T) {
	if _, err := NewOrg("", "slug"); err == nil {
		t.Fatal("empty name accepted")
	}
	if _, err := NewOrg("Name", ""); err == nil {
		t.Fatal("empty slug accepted")
	}
	org, err := NewOrg("Acme", "acme")
	if err != nil {
		t.Fatal(err)
	}
	if org.Plan != "free" {
		t.Fatalf("plan = %q, want free", org.Plan)
	}
}

func TestNewUserValidation(t *testing.T) {
	orgID := uuid.New()
	if _, err := NewUser(orgID, "", RoleAdmin, "h", "password"); err == nil {
		t.Fatal("empty email accepted")
	}
	if _, err := NewUser(orgID, "a@b.io", Role("bogus"), "h", "password"); err == nil {
		t.Fatal("invalid role accepted")
	}
	user, err := NewUser(orgID, "a@b.io", RoleRecruiter, "h", "")
	if err != nil {
		t.Fatal(err)
	}
	if user.AuthProvider != "password" {
		t.Fatalf("auth provider = %q, want default password", user.AuthProvider)
	}
	if !user.HasRole(RoleRecruiter) || user.HasRole(RoleAdmin) {
		t.Fatal("HasRole wrong")
	}
}

func TestSetScoringWeightsOrg(t *testing.T) {
	org, _ := NewOrg("A", "a")
	if err := org.SetScoringWeights(map[string]float64{"skills_match": 0.9}); err != nil {
		t.Fatal(err)
	}
	if err := org.SetScoringWeights(map[string]float64{"skills_match": -0.1}); err == nil {
		t.Fatal("negative weight accepted")
	}
}

func TestRoleValid(t *testing.T) {
	for _, r := range []Role{RoleAdmin, RoleRecruiter, RoleInterviewer, RoleMember} {
		if !r.Valid() {
			t.Fatalf("role %q should be valid", r)
		}
	}
	if Role("bogus").Valid() {
		t.Fatal("bogus role valid")
	}
}
