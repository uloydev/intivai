package domain

import (
	"errors"
	"net"
	"testing"

	"github.com/google/uuid"
)

func TestValidateWebhookURL(t *testing.T) {
	previousLookup := lookupWebhookHostIP
	lookupWebhookHostIP = func(host string) ([]net.IP, error) {
		if host == "example.com" {
			return []net.IP{net.ParseIP("93.184.216.34")}, nil
		}
		return nil, errors.New("host not found")
	}
	defer func() { lookupWebhookHostIP = previousLookup }()
	cases := []struct {
		name string
		url  string
		want bool // true = valid
	}{
		{"empty", "", false},
		{"not a url", "not-a-url", false},
		{"ftp scheme", "ftp://example.com/hook", false},
		{"no scheme", "example.com/hook", false},
		{"loopback ip", "http://127.0.0.1/hook", false},
		{"private ip", "http://10.0.0.5/hook", false},
		{"link local", "http://169.254.169.254/latest/meta-data", false},
		{"local hostname", "http://localhost/hook", false},
		{"sub local hostname", "http://api.localhost/hook", false},
		{"mDNS", "http://printer.local/hook", false},
		{"public ip", "http://93.184.216.34/hook", true},
		{"public hostname", "https://example.com/cb", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateWebhookURL(c.url)
			if c.want && err != nil {
				t.Fatalf("want valid, got %v", err)
			}
			if !c.want && err == nil {
				t.Fatal("want invalid, got nil")
			}
		})
	}
}

func TestNewWebhookConfigValidatesURL(t *testing.T) {
	cfg, err := NewWebhookConfig(uuid.New(), "http://10.0.0.1/hook", []WebhookEvent{EventInterviewCompleted}, "s")
	if err == nil {
		t.Fatal("want error for private-IP URL")
	}
	if cfg != nil {
		t.Fatal("want nil config on invalid URL")
	}
}

func TestNewWebhookConfigValidEvents(t *testing.T) {
	if _, err := NewWebhookConfig(uuid.New(), "https://example.com/hook", []WebhookEvent{"bogus.event"}, "s"); err == nil {
		t.Fatal("want error for unknown event")
	}
	if _, err := NewWebhookConfig(uuid.New(), "https://example.com/hook", nil, "s"); err == nil {
		t.Fatal("want error for empty events")
	}
}

func TestNewWebhookConfigPublicURL(t *testing.T) {
	cfg, err := NewWebhookConfig(uuid.New(), "https://93.184.216.34/cb", []WebhookEvent{EventInterviewCompleted}, "secret")
	if err != nil {
		t.Fatalf("want valid config, got %v", err)
	}
	if !cfg.Active {
		t.Fatal("want active=true by default")
	}
}

func TestValidateWebhookURLRejectsPrivateDNSResult(t *testing.T) {
	previousLookup := lookupWebhookHostIP
	lookupWebhookHostIP = func(host string) ([]net.IP, error) {
		if host == "internal.test" {
			return []net.IP{net.ParseIP("192.168.1.10")}, nil
		}
		return []net.IP{net.ParseIP("93.184.216.34")}, nil
	}
	defer func() { lookupWebhookHostIP = previousLookup }()

	if err := ValidateWebhookURL("https://internal.test/hook"); err == nil {
		t.Fatal("private DNS result accepted")
	}
}
