package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/intivai/backend/internal/iam/application"
)

// JWTProvider issues and parses HS256 tokens. Same provider issues short-lived
// WS tickets (M3 candidate flow) via TokenTypeWSTicket + Extra claims.
type JWTProvider struct {
	secret []byte
}

func NewJWTProvider(secret string) *JWTProvider {
	return &JWTProvider{secret: []byte(secret)}
}

type claims struct {
	OrgID string         `json:"org_id"`
	Role  string         `json:"role"`
	Type  string         `json:"type"`
	Extra map[string]any `json:"extra,omitempty"`
	jwt.RegisteredClaims
}

func (p *JWTProvider) Issue(subject, orgID uuid.UUID, role, tokenType string, ttl time.Duration, extra map[string]any) (string, error) {
	if tokenType == "" {
		tokenType = application.TokenTypeAuth
	}
	now := time.Now().UTC()
	c := claims{
		OrgID: orgID.String(),
		Role:  role,
		Type:  tokenType,
		Extra: extra,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			Issuer:    "intivai",
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(p.secret)
}

func (p *JWTProvider) Parse(token string) (*application.Claims, error) {
	c := claims{}
	parsed, err := jwt.ParseWithClaims(token, &c, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return p.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, err
	}
	if !parsed.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	sub, err := uuid.Parse(c.Subject)
	if err != nil {
		return nil, fmt.Errorf("invalid subject")
	}
	orgID, err := uuid.Parse(c.OrgID)
	if err != nil {
		return nil, fmt.Errorf("invalid org_id")
	}
	return &application.Claims{Subject: sub, OrgID: orgID, Role: c.Role, Type: c.Type, Extra: c.Extra}, nil
}
