package mailer

import (
	"strings"
	"testing"
)

func TestEmailHTMLRenderingNotEscaped(t *testing.T) {

	// Test 1: Application confirmation
	body := `
    <h2 style="margin-top: 0; color: #111;">Hello Jane Doe,</h2>
    <p>We have received your application for the <strong>Senior Backend Engineer</strong> position.</p>`
	html, err := renderShell("linear-gradient(135deg, #4f46e5 0%, #3b82f6 100%)", "Intivai Recruitment", body)
	if err != nil {
		t.Fatalf("renderShell failed: %v", err)
	}

	// Verify HTML is unescaped
	if strings.Contains(html, "&lt;h2") || strings.Contains(html, "&lt;p&gt;") || strings.Contains(html, "&lt;strong&gt;") {
		t.Errorf("HTML tags were escaped in output: %s", html)
	}
	if !strings.Contains(html, "<h2 style=\"margin-top: 0; color: #111;\">Hello Jane Doe,</h2>") {
		t.Errorf("Expected unescaped h2 element in html: %s", html)
	}
	if !strings.Contains(html, "<strong>Senior Backend Engineer</strong>") {
		t.Errorf("Expected unescaped strong element in html: %s", html)
	}

	// Test 2: Interview invitation
	inviteBody := `
    <h2 style="margin-top: 0; color: #111;">Congratulations Alex!</h2>
    <p>You are invited to interview.</p>
    <div style="text-align: center; margin: 30px 0;">
      <a href="https://app.intivai.com/invite/tok123" style="background: #4f46e5;">Start AI Interview</a>
    </div>`
	inviteHTML, err := renderShell("linear-gradient(135deg, #4f46e5 0%, #3b82f6 100%)", "Interview Invitation", inviteBody)
	if err != nil {
		t.Fatalf("renderShell failed: %v", err)
	}
	if strings.Contains(inviteHTML, "&lt;a href=") || strings.Contains(inviteHTML, "&lt;div") {
		t.Errorf("Invitation HTML was escaped: %s", inviteHTML)
	}
	if !strings.Contains(inviteHTML, "<a href=\"https://app.intivai.com/invite/tok123\"") {
		t.Errorf("Expected unescaped link in invitation: %s", inviteHTML)
	}

	// Test 3: Candidate OTP
	otpBody := `
    <div style="display: inline-block; background: #111827;">
      123456
    </div>`
	otpHtml, err := renderShell("#111827", "Candidate Portal", otpBody)
	if err != nil {
		t.Fatalf("renderShell failed: %v", err)
	}
	if strings.Contains(otpHtml, "&lt;div") {
		t.Errorf("OTP HTML was escaped: %s", otpHtml)
	}
}

func TestSanitizeHeader(t *testing.T) {
	raw := "Subject with\r\nCRLF\ninjection\rtest"
	clean := sanitizeHeader(raw)
	if strings.Contains(clean, "\r") || strings.Contains(clean, "\n") {
		t.Fatalf("sanitizeHeader failed to strip CRLF: %q", clean)
	}
	if clean != "Subject with  CRLF injection test" {
		t.Fatalf("unexpected sanitized output: %q", clean)
	}
}
