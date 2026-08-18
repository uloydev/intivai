package mailer

import (
	"context"
	"fmt"
	"html/template"
	"net/smtp"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

type Mailer interface {
	SendEmail(ctx context.Context, to, subject, htmlBody, textBody string) error
	SendApplicationConfirmation(ctx context.Context, to, name, jobTitle string) error
	SendInterviewInvitation(ctx context.Context, to, name, jobTitle, inviteURL string) error
	SendScorecardReady(ctx context.Context, to, candidateName, jobTitle string, score float64, recommendation, reportURL string) error
	SendCandidateLoginOTP(ctx context.Context, to, otpCode, magicLink string) error
}

type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

type SMTPMailer struct {
	cfg    Config
	logger zerolog.Logger
}

func NewSMTPMailer(cfg Config, logger zerolog.Logger) *SMTPMailer {
	if cfg.From == "" {
		cfg.From = "Intivai Talent <no-reply@intivai.com>"
	}
	return &SMTPMailer{cfg: cfg, logger: logger}
}

// sanitizeHeader strips CR/LF from header values (subject) to prevent
// header-injection when candidate-controlled strings reach the subject line.
func sanitizeHeader(s string) string {
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}

func (m *SMTPMailer) SendEmail(ctx context.Context, to, subject, htmlBody, textBody string) error {
	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)
	from := m.cfg.From

	var auth smtp.Auth
	if m.cfg.Username != "" && m.cfg.Password != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}

	headers := make(map[string]string)
	headers["From"] = from
	headers["To"] = to
	headers["Subject"] = sanitizeHeader(subject)
	headers["MIME-Version"] = "1.0"
	headers["Date"] = time.Now().Format(time.RFC1123Z)

	var msgBuilder strings.Builder
	if htmlBody != "" {
		headers["Content-Type"] = `text/html; charset="UTF-8"`
		for k, v := range headers {
			fmt.Fprintf(&msgBuilder, "%s: %s\r\n", k, v)
		}
		msgBuilder.WriteString("\r\n")
		msgBuilder.WriteString(htmlBody)
	} else {
		headers["Content-Type"] = `text/plain; charset="UTF-8"`
		for k, v := range headers {
			fmt.Fprintf(&msgBuilder, "%s: %s\r\n", k, v)
		}
		msgBuilder.WriteString("\r\n")
		msgBuilder.WriteString(textBody)
	}

	fromAddr := from
	if start := strings.Index(from, "<"); start != -1 {
		if end := strings.Index(from[start:], ">"); end != -1 {
			fromAddr = from[start+1 : start+end]
		}
	}
	fromAddr = strings.TrimSpace(fromAddr)
	if fromAddr == "" {
		fromAddr = "no-reply@intivai.com"
	}

	// Try sending via SMTP
	err := smtp.SendMail(addr, auth, fromAddr, []string{to}, []byte(msgBuilder.String()))
	if err != nil {
		m.logger.Warn().Err(err).Str("to", to).Str("subject", subject).Msg("smtp send failed, logging email")
		return err
	}

	m.logger.Info().Str("to", to).Str("subject", subject).Msg("email dispatched via SMTP")
	return nil
}

const shell = `<!DOCTYPE html>
<html>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
  <div style="background: {{.BannerBg}}; padding: 24px; border-radius: 12px; text-align: center; color: white;">
    <h1 style="margin: 0; font-size: 24px;">{{.BannerTitle}}</h1>
  </div>
  <div style="padding: 24px; background: #fafafa; border-radius: 12px; margin-top: 16px; border: 1px solid #eaeaea;">
    {{.Body}}
  </div>
</body>
</html>`

var shellTmpl = template.Must(template.New("shell").Parse(shell))

func renderShell(bannerBg, bannerTitle, body string) (string, error) {
	var sb strings.Builder
	err := shellTmpl.Execute(&sb, map[string]string{"BannerBg": bannerBg, "BannerTitle": bannerTitle, "Body": body})
	if err != nil {
		return "", err
	}
	return sb.String(), nil
}

func (m *SMTPMailer) SendApplicationConfirmation(ctx context.Context, to, name, jobTitle string) error {
	subject := sanitizeHeader(fmt.Sprintf("Application Received: %s at Intivai", jobTitle))
	body := `
    <h2 style="margin-top: 0; color: #111;">Hello {{.Name}},</h2>
    <p>We have received your application for the <strong>{{.JobTitle}}</strong> position.</p>
    <p>Our autonomous AI engine is currently analyzing your technical background and competencies. If shortlisted, you will receive an invitation to take your interactive technical interview.</p>
    <p style="color: #666; font-size: 14px;">Thank you for your interest in joining our team!</p>`

	var sb strings.Builder
	if err := template.Must(template.New("body").Parse(body)).Execute(&sb, map[string]string{"Name": name, "JobTitle": jobTitle}); err != nil {
		return err
	}
	html, err := renderShell("linear-gradient(135deg, #4f46e5 0%, #3b82f6 100%)", "Intivai Recruitment", sb.String())
	if err != nil {
		return err
	}

	text := fmt.Sprintf("Hello %s,\n\nWe have received your application for %s. Our AI engine is analyzing your background. You will receive an interview invitation if shortlisted.\n\nThank you!", name, jobTitle)
	return m.SendEmail(ctx, to, subject, html, text)
}

func (m *SMTPMailer) SendInterviewInvitation(ctx context.Context, to, name, jobTitle, inviteURL string) error {
	subject := sanitizeHeader(fmt.Sprintf("Invitation to Interview: %s", jobTitle))
	body := `
    <h2 style="margin-top: 0; color: #111;">Congratulations {{.Name}}!</h2>
    <p>Your profile passed our initial screening for <strong>{{.JobTitle}}</strong>. You are invited to complete your interactive AI technical interview.</p>
    <div style="text-align: center; margin: 30px 0;">
      <a href="{{.InviteURL}}" style="background: #4f46e5; color: white; padding: 12px 28px; text-decoration: none; border-radius: 8px; font-weight: bold; display: inline-block;">Start AI Interview</a>
    </div>
    <p style="font-size: 13px; color: #666;">This interview takes approx. 15-20 minutes. You can take it at any time from a quiet room with a stable internet connection.</p>`

	var sb strings.Builder
	if err := template.Must(template.New("body").Parse(body)).Execute(&sb, map[string]string{"Name": name, "JobTitle": jobTitle, "InviteURL": inviteURL}); err != nil {
		return err
	}
	html, err := renderShell("linear-gradient(135deg, #4f46e5 0%, #3b82f6 100%)", "Interview Invitation", sb.String())
	if err != nil {
		return err
	}

	text := fmt.Sprintf("Congratulations %s!\n\nYou have been invited to interview for %s.\n\nStart your interview here: %s\n\nGood luck!", name, jobTitle, inviteURL)
	return m.SendEmail(ctx, to, subject, html, text)
}

func (m *SMTPMailer) SendScorecardReady(ctx context.Context, to, candidateName, jobTitle string, score float64, recommendation, reportURL string) error {
	subject := sanitizeHeader(fmt.Sprintf("Scorecard Ready: %s (%v/100 - %s)", candidateName, score, recommendation))
	body := `
    <p>Candidate <strong>{{.CandidateName}}</strong> has completed their interview for <strong>{{.JobTitle}}</strong>.</p>
    <div style="background: white; border: 1px solid #e5e7eb; padding: 16px; border-radius: 8px; margin: 16px 0;">
      <p style="margin: 0; font-size: 14px;"><strong>Overall Score:</strong> {{.Score}} / 100</p>
      <p style="margin: 4px 0 0; font-size: 14px;"><strong>Hiring Verdict:</strong> {{.Recommendation}}</p>
    </div>
    <div style="text-align: center; margin: 24px 0;">
      <a href="{{.ReportURL}}" style="background: #2563eb; color: white; padding: 10px 24px; text-decoration: none; border-radius: 6px; font-weight: bold; display: inline-block;">View Full Scorecard & PDF</a>
    </div>`

	var sb strings.Builder
	if err := template.Must(template.New("body").Parse(body)).Execute(&sb, map[string]string{
		"CandidateName": candidateName, "JobTitle": jobTitle,
		"Score": fmt.Sprintf("%.1f", score), "Recommendation": recommendation, "ReportURL": reportURL,
	}); err != nil {
		return err
	}
	html, err := renderShell("#111827", "Evaluation Scorecard Synthesized", sb.String())
	if err != nil {
		return err
	}

	text := fmt.Sprintf("Interview Completed for %s (%s).\nScore: %.1f/100\nVerdict: %s\nView: %s", candidateName, jobTitle, score, recommendation, reportURL)
	return m.SendEmail(ctx, to, subject, html, text)
}

func (m *SMTPMailer) SendCandidateLoginOTP(ctx context.Context, to, otpCode, magicLink string) error {
	subject := sanitizeHeader(fmt.Sprintf("Your Intivai Candidate Portal Login Code: %s", otpCode))
	body := `
    <p style="margin-top: 0; font-size: 15px;">Use the 6-digit verification code below to log in to your candidate portal:</p>
    <div style="text-align: center; margin: 24px 0;">
      <div style="display: inline-block; background: #111827; color: #ffffff; font-size: 28px; font-weight: bold; letter-spacing: 8px; padding: 14px 28px; border-radius: 10px; font-family: monospace;">
        {{.OTPCode}}
      </div>
    </div>
    <div style="text-align: center; margin: 20px 0;">
      <p style="font-size: 13px; color: #666;">Or sign in instantly with one click:</p>
      <a href="{{.MagicLink}}" style="background: #4f46e5; color: white; padding: 10px 24px; text-decoration: none; border-radius: 6px; font-weight: bold; display: inline-block; font-size: 14px;">Log in to Candidate Portal</a>
    </div>
    <p style="font-size: 12px; color: #888; text-align: center; margin-top: 24px;">This code and magic link will expire in 10 minutes. If you did not request this login, you can safely ignore this email.</p>`

	var sb strings.Builder
	if err := template.Must(template.New("body").Parse(body)).Execute(&sb, map[string]string{"OTPCode": otpCode, "MagicLink": magicLink}); err != nil {
		return err
	}
	html, err := renderShell("linear-gradient(135deg, #4f46e5 0%, #7c3aed 100%)", "Intivai Candidate Portal", sb.String())
	if err != nil {
		return err
	}

	text := fmt.Sprintf("Your Intivai Candidate Portal Login Code is: %s\n\nOr click this magic link to log in directly: %s\n\nThis code expires in 10 minutes.", otpCode, magicLink)
	return m.SendEmail(ctx, to, subject, html, text)
}
