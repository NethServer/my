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
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"runtime"
)

// WelcomeEmailData contains data for welcome email template
type WelcomeEmailData struct {
	UserName         string
	UserEmail        string
	OrganizationName string
	OrganizationType string
	UserRoles        []string
	TempPassword     string
	LoginURL         string
	SupportEmail     string
	CompanyName      string
}

// TemplateService handles email template rendering
type TemplateService struct {
	templateDir string
}

// NewTemplateService creates a new template service
func NewTemplateService() *TemplateService {
	// Get the current directory (services/email)
	_, filename, _, _ := runtime.Caller(0)
	currentDir := filepath.Dir(filename)
	templateDir := filepath.Join(currentDir, "templates")

	return &TemplateService{
		templateDir: templateDir,
	}
}

// =============================================================================
// PUBLIC METHODS
// =============================================================================

// GenerateWelcomeEmail generates HTML and text versions of welcome email
func (ts *TemplateService) GenerateWelcomeEmail(data WelcomeEmailData) (htmlBody, textBody string, err error) {
	// Generate HTML version
	htmlBody, err = ts.renderTemplate("welcome.html", data)
	if err != nil {
		return "", "", fmt.Errorf("failed to render HTML template: %w", err)
	}

	// Generate text version
	textBody, err = ts.renderTemplate("welcome.txt", data)
	if err != nil {
		return "", "", fmt.Errorf("failed to render text template: %w", err)
	}

	return htmlBody, textBody, nil
}

// ValidateTemplates checks if required template files exist
func (ts *TemplateService) ValidateTemplates() error {
	requiredTemplates := []string{
		"welcome.html",
		"welcome.txt",
	}

	for _, templateName := range requiredTemplates {
		templatePath := filepath.Join(ts.templateDir, templateName)
		if _, err := os.Stat(templatePath); os.IsNotExist(err) {
			return fmt.Errorf("required template file not found: %s", templatePath)
		}
	}

	return nil
}

// =============================================================================
// PRIVATE METHODS
// =============================================================================

// renderTemplate loads and renders a template file
func (ts *TemplateService) renderTemplate(templateName string, data interface{}) (string, error) {
	// Load template file
	templatePath := filepath.Join(ts.templateDir, templateName)
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template file %s: %w", templatePath, err)
	}

	// Parse template
	tmpl, err := template.New(templateName).Parse(string(templateContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse template %s: %w", templateName, err)
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", templateName, err)
	}

	return buf.String(), nil
}
