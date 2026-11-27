package email

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"

	"github.com/pkg/errors"
	"github.com/resend/resend-go/v2"
	"github.com/rs/zerolog"
	"github.com/tutorgrid/tutorgrid/internal/config"
)

type Client struct {
	client    *resend.Client
	fromName  string
	fromEmail string
	logger    *zerolog.Logger
}

func NewClient(cfg *config.Config, logger *zerolog.Logger) *Client {
	fromName := strings.TrimSpace(cfg.Integration.FromName)
	if fromName == "" {
		fromName = "TutorGrid Circle"
	}
	fromEmail := strings.TrimSpace(cfg.Integration.FromEmail)
	if fromEmail == "" {
		fromEmail = "onboarding@resend.dev"
	}
	return &Client{
		client:    resend.NewClient(cfg.Integration.ResendAPIKey),
		fromName:  fromName,
		fromEmail: fromEmail,
		logger:    logger,
	}
}

func (c *Client) SendEmail(to, subject string, templateName Template, data map[string]string) error {
	tmplPath := fmt.Sprintf("%s/%s.html", "templates/emails", templateName)

	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		return errors.Wrapf(err, "failed to parse email template %s", templateName)
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return errors.Wrapf(err, "failed to execute email template %s", templateName)
	}

	params := &resend.SendEmailRequest{
		From:    fmt.Sprintf("%s <%s>", c.fromName, c.fromEmail),
		To:      []string{to},
		Subject: subject,
		Html:    body.String(),
	}

	_, err = c.client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
