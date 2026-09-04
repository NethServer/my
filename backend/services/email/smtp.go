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
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

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

	message, err := e.buildMessage(data, time.Now())
	if err != nil {
		logger.Error().
			Err(err).
			Str("to", data.To).
			Str("subject", data.Subject).
			Msg("Failed to build email message")
		return fmt.Errorf("failed to build email message: %w", err)
	}

	// Send email
	err = e.sendSMTP([]string{data.To}, message)
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

// buildMessage renders the RFC 5322 message.
//
// Bodies are UTF-8 and go out as quoted-printable: the templates carry accented
// text and emoji, which a 7bit declaration would misrepresent (mojibake for the
// recipient, and a spam signal for the filters). Subject goes through RFC 2047
// for the same reason, which also neutralizes header injection via the
// organization name: control characters force the whole word to be encoded.
func (e *EmailService) buildMessage(data EmailData, now time.Time) ([]byte, error) {
	boundary := "nethesis-" + rand.Text()

	headers := [][2]string{
		{"Date", now.Format(time.RFC1123Z)},
		{"Message-ID", e.messageID()},
		{"From", (&mail.Address{Name: e.fromName, Address: e.from}).String()},
		{"To", (&mail.Address{Address: data.To}).String()},
		{"Subject", mime.QEncoding.Encode("UTF-8", data.Subject)},
		{"MIME-Version", "1.0"},
		{"Content-Type", fmt.Sprintf("multipart/alternative; boundary=%q", boundary)},
	}

	var message strings.Builder
	for _, header := range headers {
		message.WriteString(header[0] + ": " + header[1] + "\r\n")
	}
	message.WriteString("\r\n")

	if data.TextBody != "" {
		if err := writeBodyPart(&message, boundary, "text/plain", data.TextBody); err != nil {
			return nil, err
		}
	}

	if data.HTMLBody != "" {
		if err := writeBodyPart(&message, boundary, "text/html", data.HTMLBody); err != nil {
			return nil, err
		}
	}

	message.WriteString("--" + boundary + "--\r\n")

	return []byte(message.String()), nil
}

// messageID builds a Message-ID anchored on the sender domain
func (e *EmailService) messageID() string {
	domain := e.from
	if at := strings.LastIndex(e.from, "@"); at != -1 {
		domain = e.from[at+1:]
	}

	return "<" + rand.Text() + "@" + domain + ">"
}

// writeBodyPart appends a quoted-printable UTF-8 part to the multipart body
func writeBodyPart(message *strings.Builder, boundary, contentType, body string) error {
	message.WriteString("--" + boundary + "\r\n")
	message.WriteString("Content-Type: " + contentType + "; charset=UTF-8\r\n")
	message.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")

	encoder := quotedprintable.NewWriter(message)
	if _, err := encoder.Write([]byte(body)); err != nil {
		return fmt.Errorf("failed to encode %s part: %w", contentType, err)
	}
	if err := encoder.Close(); err != nil {
		return fmt.Errorf("failed to encode %s part: %w", contentType, err)
	}
	message.WriteString("\r\n")

	return nil
}

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
