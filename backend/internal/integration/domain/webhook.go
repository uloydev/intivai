package domain

import (
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/intivai/backend/internal/shared/domain"
)

var lookupWebhookHostIP = net.LookupIP

type WebhookEvent string

const (
	EventInterviewCompleted WebhookEvent = "interview.completed"
	EventCandidateScreened  WebhookEvent = "candidate.screened"
	EventCandidateAdvanced  WebhookEvent = "candidate.advanced"
)

var ValidEvents = map[WebhookEvent]bool{
	EventInterviewCompleted: true,
	EventCandidateScreened:  true,
	EventCandidateAdvanced:  true,
}

type WebhookConfig struct {
	domain.Entity
	OrgID  uuid.UUID
	URL    string
	Events []WebhookEvent
	Secret string
	Active bool
}

func NewWebhookConfig(orgID uuid.UUID, url string, events []WebhookEvent, secret string) (*WebhookConfig, error) {
	if err := ValidateWebhookURL(url); err != nil {
		return nil, err
	}
	if len(events) == 0 {
		return nil, ErrNoEvents
	}
	for _, e := range events {
		if !ValidEvents[e] {
			return nil, ErrInvalidEvent
		}
	}
	return &WebhookConfig{
		Entity: domain.Entity{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		OrgID:  orgID,
		URL:    url,
		Events: events,
		Secret: secret,
		Active: true,
	}, nil
}

// ValidateWebhookURL guards against SSRF: only http/https URLs to public
// addresses. Loopback, private, link-local, and multicast ranges are
// rejected. Hostnames are resolved and every resulting address is checked.
func ValidateWebhookURL(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return ErrInvalidURL
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ErrInvalidURL
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ErrInvalidURL
	}
	host := u.Hostname()
	if host == "" {
		return ErrInvalidURL
	}
	lower := strings.ToLower(host)
	if lower == "localhost" || strings.HasSuffix(lower, ".localhost") || strings.HasSuffix(lower, ".local") {
		return ErrInvalidURL
	}
	if ip := net.ParseIP(host); ip != nil {
		if !isPublicIP(ip) {
			return ErrInvalidURL
		}
		return nil
	}
	ips, err := lookupWebhookHostIP(host)
	if err != nil || len(ips) == 0 {
		return ErrInvalidURL
	}
	for _, ip := range ips {
		if !isPublicIP(ip) {
			return ErrInvalidURL
		}
	}
	return nil
}

func isPublicIP(ip net.IP) bool {
	for _, cidr := range []string{
		"100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24",
		"198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4",
	} {
		_, network, err := net.ParseCIDR(cidr)
		if err == nil && network.Contains(ip) {
			return false
		}
	}
	return ip.IsGlobalUnicast() && !ip.IsLoopback() && !ip.IsPrivate() &&
		!ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() &&
		!ip.IsUnspecified() && !ip.IsMulticast()
}

type WebhookDelivery struct {
	domain.Entity
	OrgID        uuid.UUID
	WebhookID    uuid.UUID
	Event        WebhookEvent
	Payload      []byte
	StatusCode   int
	ResponseBody string
	Attempts     int
	NextRetryAt  *time.Time
	FinalStatus  string // "delivered", "failed", "pending"
}

// InterviewCompletedPayload — typed body of the interview.completed webhook
// event (strict types; no map[string]any).
type InterviewCompletedPayload struct {
	InterviewID    string  `json:"interview_id"`
	Score          float64 `json:"score"`
	Recommendation string  `json:"recommendation"`
	ReportURL      string  `json:"report_url"`
}
