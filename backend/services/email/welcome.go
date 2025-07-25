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
	"fmt"
	"math/big"
	"strings"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/services/logto"
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

// SendWelcomeEmail sends a welcome email to a newly created user (legacy method)
func (w *WelcomeEmailService) SendWelcomeEmail(userEmail, userName, organizationName, userRole, tempPassword string) error {
	// Convert to new format for backward compatibility
	userRoles := []string{}
	if userRole != "" {
		userRoles = append(userRoles, userRole)
	}
	return w.SendWelcomeEmailWithRoles(userEmail, userName, organizationName, "", userRoles, tempPassword)
}

// SendWelcomeEmailWithRoles sends a welcome email with support for multiple user roles
func (w *WelcomeEmailService) SendWelcomeEmailWithRoles(userEmail, userName, organizationName, organizationType string, userRoles []string, tempPassword string) error {
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

// GenerateValidTemporaryPassword generates a secure temporary password that passes our validation
func (w *WelcomeEmailService) GenerateValidTemporaryPassword() (string, error) {
	const maxAttempts = 10

	for attempt := 0; attempt < maxAttempts; attempt++ {
		password, err := w.generatePassword()
		if err != nil {
			return "", fmt.Errorf("failed to generate password: %w", err)
		}

		// Validate using our existing validator
		isValid, errors := helpers.ValidatePasswordStrength(password)
		if isValid {
			logger.Debug().
				Int("attempt", attempt+1).
				Msg("Generated valid temporary password")
			return password, nil
		}

		logger.Debug().
			Int("attempt", attempt+1).
			Strs("validation_errors", errors).
			Msg("Generated password failed validation, retrying")
	}

	return "", fmt.Errorf("failed to generate valid password after %d attempts", maxAttempts)
}

// generatePassword creates a password that should meet our validation requirements
func (w *WelcomeEmailService) generatePassword() (string, error) {
	// Generate a 14-character password (exceeds minimum requirement of 12)
	const length = 14

	// Character sets that match our validator requirements
	const (
		lowerChars  = "abcdefghijklmnopqrstuvwxyz"
		upperChars  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		numberChars = "0123456789"
		// Use only safe special characters that are in our validator regex
		specialChars = "!@#$%^&*()_+-=[]{}|;:,.<>?~"
	)

	password := make([]byte, length)

	// Ensure at least 2 characters from each required category (for better distribution)
	requirements := []struct {
		chars string
		count int
	}{
		{lowerChars, 3},   // At least 3 lowercase
		{upperChars, 3},   // At least 3 uppercase
		{numberChars, 2},  // At least 2 digits
		{specialChars, 2}, // At least 2 special chars
	}

	pos := 0

	// Fill required characters
	for _, req := range requirements {
		for i := 0; i < req.count && pos < length; i++ {
			char, err := w.randomChar(req.chars)
			if err != nil {
				return "", err
			}
			password[pos] = char
			pos++
		}
	}

	// Fill remaining positions with random mix
	allChars := lowerChars + upperChars + numberChars + specialChars
	for pos < length {
		char, err := w.randomChar(allChars)
		if err != nil {
			return "", err
		}
		password[pos] = char
		pos++
	}

	// Shuffle the password to avoid predictable patterns
	for i := range password {
		j, err := rand.Int(rand.Reader, big.NewInt(int64(len(password))))
		if err != nil {
			return "", fmt.Errorf("failed to shuffle password: %w", err)
		}
		password[i], password[j.Int64()] = password[j.Int64()], password[i]
	}

	return string(password), nil
}

// randomChar returns a random character from the given character set
func (w *WelcomeEmailService) randomChar(charSet string) (byte, error) {
	if len(charSet) == 0 {
		return 0, fmt.Errorf("character set cannot be empty")
	}

	index, err := rand.Int(rand.Reader, big.NewInt(int64(len(charSet))))
	if err != nil {
		return 0, err
	}

	return charSet[index.Int64()], nil
}

// getLoginURL returns the login URL for the application
func (w *WelcomeEmailService) getLoginURL() string {
	// Try to get the frontend application's redirect URI from Logto
	if frontendURL := w.getFrontendRedirectURI(); frontendURL != "" {
		return frontendURL
	}

	// Try to get from tenant domain configuration
	if configuration.Config.TenantDomain != "" {
		return fmt.Sprintf("https://%s/account?change-password=true", configuration.Config.TenantDomain)
	}

	// Fallback to tenant ID based URL
	if configuration.Config.TenantID != "" {
		return fmt.Sprintf("https://%s.logto.app/account?change-password=true", configuration.Config.TenantID)
	}

	// Final fallback
	return "https://localhost:3000/account?change-password=true"
}

// getFrontendRedirectURI gets the frontend application's redirect URI from Logto
func (w *WelcomeEmailService) getFrontendRedirectURI() string {
	// Create Logto client
	client := logto.NewManagementClient()

	// Get all applications
	apps, err := client.GetApplications()
	if err != nil {
		logger.Debug().
			Err(err).
			Msg("Failed to get applications from Logto for redirect URI lookup")
		return ""
	}

	// Look for the frontend application
	for _, app := range apps {
		if strings.ToLower(app.Name) == "frontend" && app.Type == "SPA" {
			// Get the first post logout redirect URI
			if len(app.OidcClientMetadata.PostLogoutRedirectUris) > 0 {
				baseURL := app.OidcClientMetadata.PostLogoutRedirectUris[0]
				// Remove /login if present and add /account?change-password=true
				baseURL = strings.TrimSuffix(baseURL, "/login")
				baseURL = strings.TrimSuffix(baseURL, "/")
				return baseURL + "/account?change-password=true"
			}
			// Fallback to redirect URIs if post logout not available
			if len(app.OidcClientMetadata.RedirectUris) > 0 {
				baseURL := app.OidcClientMetadata.RedirectUris[0]
				// Remove /login if present and add /account?change-password=true
				baseURL = strings.TrimSuffix(baseURL, "/login")
				baseURL = strings.TrimSuffix(baseURL, "/")
				return baseURL + "/account?change-password=true"
			}
		}
	}

	logger.Debug().
		Msg("Frontend application not found in Logto applications")
	return ""
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

// IsConfigured checks if the welcome email service is properly configured
func (w *WelcomeEmailService) IsConfigured() bool {
	return w.emailService.IsConfigured()
}

// TestConfiguration tests the email service configuration
func (w *WelcomeEmailService) TestConfiguration() error {
	// Test SMTP connection
	if err := w.emailService.TestConnection(); err != nil {
		return fmt.Errorf("SMTP connection test failed: %w", err)
	}

	// Test template validation
	if err := w.templateService.ValidateTemplates(); err != nil {
		return fmt.Errorf("template validation failed: %w", err)
	}

	// Test password generation
	_, err := w.GenerateValidTemporaryPassword()
	if err != nil {
		return fmt.Errorf("password generation test failed: %w", err)
	}

	logger.Info().Msg("Welcome email service configuration test successful")
	return nil
}
