/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package email

import (
	"crypto/tls"
	"fmt"
	"net/smtp"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/logger"
)

// EmailService handles SMTP email sending
type EmailService struct {
	host     string
	port     int
	username string
	password string
	from     string
	fromName string
	useTLS   bool
}

// NewEmailService creates a new email service instance
func NewEmailService() *EmailService {
	return &EmailService{
		host:     configuration.Config.SMTPHost,
		port:     configuration.Config.SMTPPort,
		username: configuration.Config.SMTPUsername,
		password: configuration.Config.SMTPPassword,
		from:     configuration.Config.SMTPFrom,
		fromName: configuration.Config.SMTPFromName,
		useTLS:   configuration.Config.SMTPTLS,
	}
}

// EmailData contains all data needed for sending emails
type EmailData struct {
	To       string
	Subject  string
	HTMLBody string
	TextBody string
}

// =============================================================================
// PUBLIC METHODS
// =============================================================================

// SendEmail sends an email using SMTP
func (e *EmailService) SendEmail(data EmailData) error {
	// Validate configuration
	if e.host == "" || e.from == "" {
		return fmt.Errorf("SMTP configuration incomplete: host and from address are required")
	}

	// Prepare message
	from := fmt.Sprintf("%s <%s>", e.fromName, e.from)

	headers := make(map[string]string)
	headers["From"] = from
	headers["To"] = data.To
	headers["Subject"] = data.Subject
	headers["MIME-Version"] = "1.0"

	// Multi-part message with HTML and text
	boundary := "boundary-nethesis-email"
	headers["Content-Type"] = fmt.Sprintf("multipart/alternative; boundary=%s", boundary)

	// Build message
	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n"

	// Text part
	if data.TextBody != "" {
		message += fmt.Sprintf("--%s\r\n", boundary)
		message += "Content-Type: text/plain; charset=UTF-8\r\n"
		message += "Content-Transfer-Encoding: 7bit\r\n\r\n"
		message += data.TextBody + "\r\n\r\n"
	}

	// HTML part
	if data.HTMLBody != "" {
		message += fmt.Sprintf("--%s\r\n", boundary)
		message += "Content-Type: text/html; charset=UTF-8\r\n"
		message += "Content-Transfer-Encoding: 7bit\r\n\r\n"
		message += data.HTMLBody + "\r\n\r\n"
	}

	message += fmt.Sprintf("--%s--\r\n", boundary)

	// Send email
	err := e.sendSMTP([]string{data.To}, []byte(message))
	if err != nil {
		logger.Error().
			Err(err).
			Str("to", data.To).
			Str("subject", data.Subject).
			Str("smtp_host", e.host).
			Int("smtp_port", e.port).
			Msg("Failed to send email")
		return fmt.Errorf("failed to send email: %w", err)
	}

	logger.Info().
		Str("to", data.To).
		Str("subject", data.Subject).
		Str("smtp_host", e.host).
		Int("smtp_port", e.port).
		Msg("Email sent successfully")

	return nil
}

// IsConfigured checks if SMTP is properly configured
func (e *EmailService) IsConfigured() bool {
	return e.host != "" && e.from != ""
}

// =============================================================================
// PRIVATE METHODS
// =============================================================================

// sendSMTP handles the actual SMTP sending
func (e *EmailService) sendSMTP(to []string, body []byte) error {
	addr := fmt.Sprintf("%s:%d", e.host, e.port)

	// Create connection
	conn, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer func() { _ = conn.Close() }()

	// Start TLS if enabled
	if e.useTLS {
		tlsConfig := &tls.Config{
			ServerName: e.host,
		}
		if err := conn.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("failed to start TLS: %w", err)
		}
	}

	// Authenticate if credentials provided
	if e.username != "" && e.password != "" {
		auth := smtp.PlainAuth("", e.username, e.password, e.host)
		if err := conn.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	// Set sender
	if err := conn.Mail(e.from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients
	for _, recipient := range to {
		if err := conn.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	// Send body
	w, err := conn.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	_, err = w.Write(body)
	if err != nil {
		return fmt.Errorf("failed to write email body: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	return nil
}
