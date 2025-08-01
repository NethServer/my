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
	"fmt"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/logger"
)

// WelcomeEmailService handles sending welcome emails to new users
type WelcomeEmailService struct {
	emailService    *EmailService
	templateService *TemplateService
}

// NewWelcomeEmailService creates a new welcome email service
func NewWelcomeEmailService() *WelcomeEmailService {
	return &WelcomeEmailService{
		emailService:    NewEmailService(),
		templateService: NewTemplateService(),
	}
}

// =============================================================================
// PUBLIC METHODS
// =============================================================================

// SendWelcomeEmail sends a welcome email with organization and user roles information
func (w *WelcomeEmailService) SendWelcomeEmail(userEmail, userName, organizationName, organizationType string, userRoles []string, tempPassword string) error {
	// Check if email service is configured
	if !w.emailService.IsConfigured() {
		logger.Warn().
			Str("user_email", userEmail).
			Msg("SMTP not configured, skipping welcome email")
		return nil // Don't fail user creation if email is not configured
	}

	// Validate templates
	if err := w.templateService.ValidateTemplates(); err != nil {
		logger.Error().
			Err(err).
			Str("user_email", userEmail).
			Msg("Email templates validation failed")
		return fmt.Errorf("email templates validation failed: %w", err)
	}

	// Prepare template data
	templateData := WelcomeEmailData{
		UserName:         userName,
		UserEmail:        userEmail,
		OrganizationName: organizationName,
		OrganizationType: organizationType,
		UserRoles:        userRoles,
		TempPassword:     tempPassword,
		LoginURL:         w.getLoginURL(),
		SupportEmail:     w.getSupportEmail(),
		CompanyName:      w.getCompanyName(),
	}

	// Generate email content
	htmlBody, textBody, err := w.templateService.GenerateWelcomeEmail(templateData)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_email", userEmail).
			Msg("Failed to generate email content")
		return fmt.Errorf("failed to generate email content: %w", err)
	}

	// Prepare email data
	emailData := EmailData{
		To:       userEmail,
		Subject:  fmt.Sprintf("Welcome to %s - Account Created", organizationName),
		HTMLBody: htmlBody,
		TextBody: textBody,
	}

	// Send email
	if err := w.emailService.SendEmail(emailData); err != nil {
		logger.Error().
			Err(err).
			Str("user_email", userEmail).
			Str("organization_name", organizationName).
			Msg("Failed to send welcome email")
		return fmt.Errorf("failed to send welcome email: %w", err)
	}

	logger.Info().
		Str("user_email", userEmail).
		Str("user_name", userName).
		Str("organization_name", organizationName).
		Str("organization_type", organizationType).
		Strs("user_roles", userRoles).
		Msg("Welcome email sent successfully")

	return nil
}

// =============================================================================
// PRIVATE METHODS
// =============================================================================

// getLoginURL returns the login URL for the application
func (w *WelcomeEmailService) getLoginURL() string {
	// Try to get from tenant domain configuration
	if configuration.Config.TenantDomain != "" {
		return fmt.Sprintf("https://%s/account?changePassword=true", "qa.my.nethesis.it")
	}

	// Fallback to tenant ID based URL
	if configuration.Config.TenantID != "" {
		return fmt.Sprintf("https://%s.logto.app/account?changePassword=true", configuration.Config.TenantID)
	}

	// Final fallback
	return "https://localhost:3000/account?changePassword=true"
}

// getSupportEmail returns the support email address
func (w *WelcomeEmailService) getSupportEmail() string {
	// Try to get from SMTP configuration
	if configuration.Config.SMTPFrom != "" {
		return configuration.Config.SMTPFrom
	}

	// Fallback
	return "support@example.com"
}

// getCompanyName returns the company name
func (w *WelcomeEmailService) getCompanyName() string {
	// Try to get from SMTP configuration
	if configuration.Config.SMTPFromName != "" {
		return configuration.Config.SMTPFromName
	}

	// Fallback
	return "Nethesis Operation Center"
}
