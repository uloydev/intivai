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
	SendCandidateReview(ctx context.Context, to, candidateName, inviteURL string) error
	// SendCandidateDecision — recruiter decision (offer extended / rejected)
	// notification to the candidate, with a portal link to their status.
	SendCandidateDecision(ctx context.Context, to, name, jobTitle, decision, portalURL string) error
	// SendCandidatePortalAccess — one-time portal magic link for a
	// just-applied candidate (apply flow: the link is delivered by email
	// only, never in the HTTP response).
	SendCandidatePortalAccess(ctx context.Context, to, name, portalURL string) error
	SendHumanRequestNotification(ctx context.Context, to, candidateName, jobTitle, reportURL string) error
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
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{.BannerTitle}}</title>
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; line-height: 1.6; color: #1e293b; background-color: #f8fafc; margin: 0; padding: 24px;">
  <div style="max-width: 580px; margin: 0 auto; background: #ffffff; border-radius: 16px; overflow: hidden; box-shadow: 0 4px 12px rgba(0, 0, 0, 0.05); border: 1px solid #e2e8f0;">
    <div style="background: {{.BannerBg}}; padding: 28px 24px; text-align: center; color: #ffffff;">
      <h1 style="margin: 0; font-size: 22px; font-weight: 700; letter-spacing: -0.025em; color: #ffffff;">{{.BannerTitle}}</h1>
    </div>
    <div style="padding: 28px 24px; font-size: 15px; color: #334155;">
      {{.Body}}
    </div>
    <div style="padding: 16px 24px; background: #f8fafc; border-top: 1px solid #f1f5f9; text-align: center; font-size: 12px; color: #94a3b8;">
      &copy; Intivai AI Platform. Automated Notification.
    </div>
  </div>
</body>
</html>`

var shellTmpl = template.Must(template.New("shell").Parse(shell))

type shellData struct {
	BannerBg    template.CSS
	BannerTitle string
	Body        template.HTML
}

func renderShell(bannerBg, bannerTitle, body string) (string, error) {
	var sb strings.Builder
	err := shellTmpl.Execute(&sb, shellData{
		//nolint:gosec // G203: intentional raw HTML/CSS — body built server-side from trusted templates;
		// bannerBg is tenant-admin config scoped to that org's own outbound emails.
		BannerBg:    template.CSS(bannerBg),
		BannerTitle: bannerTitle,
		Body:        template.HTML(body), //nolint:gosec // G203: see above
	})
	if err != nil {
		return "", err
	}
	return sb.String(), nil
}

// Body templates — parsed once at package init instead of per send call.
var (
	applicationConfirmationBody = template.Must(template.New("body").Parse(`
    <h2 style="margin-top: 0; color: #111;">Hello {{.Name}},</h2>
    <p>We have received your application for the <strong>{{.JobTitle}}</strong> position.</p>
    <p>Our autonomous AI engine is currently analyzing your technical background and competencies. If shortlisted, you will receive an invitation to take your interactive technical interview.</p>
    <p style="color: #666; font-size: 14px;">Thank you for your interest in joining our team!</p>`))

	interviewInvitationBody = template.Must(template.New("body").Parse(`
    <h2 style="margin-top: 0; color: #111;">Congratulations {{.Name}}!</h2>
    <p>Your profile passed our initial screening for <strong>{{.JobTitle}}</strong>. You are invited to complete your interactive AI technical interview.</p>
    <div style="text-align: center; margin: 30px 0;">
      <a href="{{.MagicLink}}" style="background: #4f46e5; color: white; padding: 12px 28px; text-decoration: none; border-radius: 8px; font-weight: bold; display: inline-block;">Start Interview Instantly</a>
    </div>
    <p style="font-size: 13px; color: #666;">This link starts your interview immediately — no login required. You can also use the standard link below if you prefer to review the details first:</p>
    <div style="text-align: center; margin: 16px 0;">
      <a href="{{.InviteURL}}" style="color: #4f46e5; text-decoration: underline; font-size: 13px;">Review &amp; start manually</a>
    </div>
    <p style="font-size: 13px; color: #666;">This interview takes approx. 15-20 minutes. You can take it at any time from a quiet room with a stable internet connection.</p>`))

	scorecardReadyBody = template.Must(template.New("body").Parse(`
    <p>Candidate <strong>{{.CandidateName}}</strong> has completed their interview for <strong>{{.JobTitle}}</strong>.</p>
    <div style="background: white; border: 1px solid #e5e7eb; padding: 16px; border-radius: 8px; margin: 16px 0;">
      <p style="margin: 0; font-size: 14px;"><strong>Overall Score:</strong> {{.Score}} / 100</p>
      <p style="margin: 4px 0 0; font-size: 14px;"><strong>Hiring Verdict:</strong> {{.Recommendation}}</p>
    </div>
    <div style="text-align: center; margin: 24px 0;">
      <a href="{{.ReportURL}}" style="background: #2563eb; color: white; padding: 10px 24px; text-decoration: none; border-radius: 6px; font-weight: bold; display: inline-block;">View Full Scorecard & PDF</a>
    </div>`))

	candidateOTPBody = template.Must(template.New("body").Parse(`
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
    <p style="font-size: 12px; color: #888; text-align: center; margin-top: 24px;">This code and magic link will expire in {{.ExpiryWindow}}. If you did not request this login, you can safely ignore this email.</p>`))

	candidateReviewBody = template.Must(template.New("body").Parse(`
    <h2 style="margin-top: 0; color: #111;">Hello {{.CandidateName}},</h2>
    <p>Your AI-extracted profile is ready for review.</p>
    <div style="text-align: center; margin: 30px 0;">
      <a href="{{.InviteURL}}" style="background: #4f46e5; color: white; padding: 12px 28px; text-decoration: none; border-radius: 8px; font-weight: bold; display: inline-block;">Review Profile</a>
    </div>
    <p style="font-size: 13px; color: #666;">Please review your profile to proceed with the application.</p>`))

	candidateDecisionBody = template.Must(template.New("body").Parse(`
    <h2 style="margin-top: 0; color: #111;">Hello {{.Name}},</h2>
    <p>An update on your application for <strong>{{.JobTitle}}</strong>:</p>
    <div style="background: white; border: 1px solid #e5e7eb; padding: 16px; border-radius: 8px; margin: 16px 0;">
      <p style="margin: 0; font-size: 15px;"><strong>{{.Decision}}</strong></p>
    </div>
    <div style="text-align: center; margin: 24px 0;">
      <a href="{{.PortalURL}}" style="background: #4f46e5; color: white; padding: 10px 24px; text-decoration: none; border-radius: 6px; font-weight: bold; display: inline-block; font-size: 14px;">View your application status</a>
    </div>
    <p style="font-size: 12px; color: #888; text-align: center; margin-top: 24px;">You can track your application anytime in the Intivai candidate portal.</p>`))

	candidatePortalAccessBody = template.Must(template.New("body").Parse(`
    <h2 style="margin-top: 0; color: #111;">Hello {{.Name}},</h2>
    <p>Your application was received and is being analyzed by our AI engine.</p>
    <div style="text-align: center; margin: 24px 0;">
      <a href="{{.MagicLink}}" style="background: #4f46e5; color: white; padding: 10px 24px; text-decoration: none; border-radius: 6px; font-weight: bold; display: inline-block; font-size: 14px;">Open Your Candidate Portal</a>
    </div>
    <p style="font-size: 12px; color: #888; text-align: center; margin-top: 24px;">This one-time link expires in 24 hours. If you did not apply, you can safely ignore this email.</p>`))
)

// otpExpiryWindow — the OTP validity window quoted in the login email.
const otpExpiryWindow = "10 minutes"

func (m *SMTPMailer) SendApplicationConfirmation(ctx context.Context, to, name, jobTitle string) error {
	subject := sanitizeHeader(fmt.Sprintf("Application Received: %s at Intivai", jobTitle))

	var sb strings.Builder
	if err := applicationConfirmationBody.Execute(&sb, map[string]string{"Name": name, "JobTitle": jobTitle}); err != nil {
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

	magicLink := inviteURL + "&auto=1"

	var sb strings.Builder
	if err := interviewInvitationBody.Execute(&sb, map[string]string{"Name": name, "JobTitle": jobTitle, "InviteURL": inviteURL, "MagicLink": magicLink}); err != nil {
		return err
	}
	html, err := renderShell("linear-gradient(135deg, #4f46e5 0%, #3b82f6 100%)", "Interview Invitation", sb.String())
	if err != nil {
		return err
	}

	text := fmt.Sprintf("Congratulations %s!\n\nYou have been invited to interview for %s.\n\nStart your interview instantly (no login required): %s\n\nOr review and start manually: %s\n\nGood luck!", name, jobTitle, magicLink, inviteURL)
	return m.SendEmail(ctx, to, subject, html, text)
}

func (m *SMTPMailer) SendScorecardReady(ctx context.Context, to, candidateName, jobTitle string, score float64, recommendation, reportURL string) error {
	subject := sanitizeHeader(fmt.Sprintf("Scorecard Ready: %s (%v/100 - %s)", candidateName, score, recommendation))

	var sb strings.Builder
	if err := scorecardReadyBody.Execute(&sb, map[string]string{
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

	var sb strings.Builder
	if err := candidateOTPBody.Execute(&sb, map[string]string{"OTPCode": otpCode, "MagicLink": magicLink, "ExpiryWindow": otpExpiryWindow}); err != nil {
		return err
	}
	html, err := renderShell("linear-gradient(135deg, #4f46e5 0%, #7c3aed 100%)", "Intivai Candidate Portal", sb.String())
	if err != nil {
		return err
	}

	text := fmt.Sprintf("Your Intivai Candidate Portal Login Code is: %s\n\nOr click this magic link to log in directly: %s\n\nThis code expires in %s.", otpCode, magicLink, otpExpiryWindow)
	return m.SendEmail(ctx, to, subject, html, text)
}

func (m *SMTPMailer) SendCandidateReview(ctx context.Context, to, candidateName, inviteURL string) error {
	subject := sanitizeHeader("Review Your Intivai AI Profile")

	var sb strings.Builder
	if err := candidateReviewBody.Execute(&sb, map[string]string{"CandidateName": candidateName, "InviteURL": inviteURL}); err != nil {
		return err
	}
	html, err := renderShell("linear-gradient(135deg, #4f46e5 0%, #3b82f6 100%)", "Profile Review", sb.String())
	if err != nil {
		return err
	}

	text := fmt.Sprintf("Hello %s,\n\nYour AI-extracted profile is ready for review.\n\nReview your profile here: %s\n\nThank you!", candidateName, inviteURL)
	return m.SendEmail(ctx, to, subject, html, text)
}

func (m *SMTPMailer) SendCandidateDecision(ctx context.Context, to, name, jobTitle, decision, portalURL string) error {
	subject := sanitizeHeader(fmt.Sprintf("Application Update: %s", jobTitle))

	var sb strings.Builder
	if err := candidateDecisionBody.Execute(&sb, map[string]string{
		"Name": name, "JobTitle": jobTitle, "Decision": decision, "PortalURL": portalURL,
	}); err != nil {
		return err
	}
	html, err := renderShell("linear-gradient(135deg, #4f46e5 0%, #7c3aed 100%)", "Application Update", sb.String())
	if err != nil {
		return err
	}

	text := fmt.Sprintf("Hello %s,\n\nAn update on your application for %s: %s\n\nTrack your application in the portal: %s",
		name, jobTitle, decision, portalURL)
	return m.SendEmail(ctx, to, subject, html, text)
}

func (m *SMTPMailer) SendCandidatePortalAccess(ctx context.Context, to, name, portalURL string) error {
	subject := sanitizeHeader("Access Your Intivai Candidate Portal")

	var sb strings.Builder
	if err := candidatePortalAccessBody.Execute(&sb, map[string]string{"Name": name, "MagicLink": portalURL}); err != nil {
		return err
	}
	html, err := renderShell("linear-gradient(135deg, #4f46e5 0%, #7c3aed 100%)", "Intivai Candidate Portal", sb.String())
	if err != nil {
		return err
	}

	text := fmt.Sprintf("Hello %s,\n\nYour application was received. Open your one-time candidate portal link: %s\n\nThe link expires in 24 hours.", name, portalURL)
	return m.SendEmail(ctx, to, subject, html, text)
}

var humanRequestBody = template.Must(template.New("body").Parse(`
    <h2 style="margin-top: 0; color: #111;">Candidate Requested Human Interviewer</h2>
    <p><strong>{{.CandidateName}}</strong> has requested a human interviewer for <strong>{{.JobTitle}}</strong>.</p>
    <p style="color: #666;">The AI interview has been paused. Please follow up with the candidate.</p>
    <div style="text-align: center; margin: 30px 0;">
      <a href="{{.ReportURL}}" style="background: #dc2626; color: white; padding: 12px 28px; text-decoration: none; border-radius: 8px; font-weight: bold; display: inline-block;">Review Candidate</a>
    </div>`))

func (m *SMTPMailer) SendHumanRequestNotification(ctx context.Context, to, candidateName, jobTitle, reportURL string) error {
	subject := sanitizeHeader(fmt.Sprintf("Human Interview Request: %s", candidateName))
	var sb strings.Builder
	if err := humanRequestBody.Execute(&sb, map[string]string{"CandidateName": candidateName, "JobTitle": jobTitle, "ReportURL": reportURL}); err != nil {
		return err
	}
	html, err := renderShell("linear-gradient(135deg, #dc2626 0%, #b91c1c 100%)", "Human Interview Request", sb.String())
	if err != nil {
		return err
	}
	text := fmt.Sprintf("Candidate %s has requested a human interviewer for %s.\n\nReview: %s", candidateName, jobTitle, reportURL)
	return m.SendEmail(ctx, to, subject, html, text)
}
