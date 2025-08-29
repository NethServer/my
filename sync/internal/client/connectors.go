/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package client

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/nethesis/my/sync/internal/config"
	"github.com/nethesis/my/sync/internal/logger"
)

// ConnectorConfig represents a connector configuration for Logto API
type ConnectorConfig struct {
	Config map[string]interface{} `json:"config"`
}

// GetConnectors retrieves all connectors
func (c *LogtoClient) GetConnectors() ([]map[string]interface{}, error) {
	resp, err := c.makeRequest("GET", "/api/connectors", nil)
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	return result, c.handlePaginatedResponse(resp, &result)
}

// CreateConnector creates a new connector
func (c *LogtoClient) CreateConnector(connectorType string, config ConnectorConfig) (map[string]interface{}, error) {
	logger.Debug("Creating connector of type %s", connectorType)

	payload := map[string]interface{}{
		"connectorId": connectorType,
		"config":      config.Config,
	}

	resp, err := c.makeRequest("POST", "/api/connectors", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create connector: %w", err)
	}

	var result map[string]interface{}
	return result, c.handleCreationResponse(resp, &result)
}

// UpdateConnector updates an existing connector
func (c *LogtoClient) UpdateConnector(connectorID string, config ConnectorConfig) error {
	logger.Debug("Updating connector %s", connectorID)

	resp, err := c.makeRequest("PATCH", "/api/connectors/"+connectorID, config)
	if err != nil {
		return fmt.Errorf("failed to update connector: %w", err)
	}

	return c.handleResponse(resp, http.StatusOK, nil)
}

// SyncSMTPConnector synchronizes SMTP connector configuration with Logto
func (c *LogtoClient) SyncSMTPConnector(smtpConfig *config.SMTPConnector) error {
	if smtpConfig == nil {
		logger.Info("No SMTP configuration provided, skipping connector sync")
		return nil
	}

	logger.Info("Syncing SMTP connector configuration")

	// Load email templates
	templates, err := c.loadEmailTemplates(smtpConfig.TemplateSettings)
	if err != nil {
		return fmt.Errorf("failed to load email templates: %w", err)
	}

	// Build the connector configuration payload (matching the structure from UI)
	connectorConfig := ConnectorConfig{
		Config: map[string]interface{}{
			"host": smtpConfig.Host,
			"port": c.getPortOrDefault(smtpConfig.Port),
			"auth": map[string]interface{}{
				"type": "login",
				"user": smtpConfig.Username,
				"pass": smtpConfig.Password,
			},
			"fromEmail":         c.getFormattedFromEmail(smtpConfig),
			"fromName":          smtpConfig.FromName,
			"templates":         templates,
			"logger":            c.getBoolOrDefault(smtpConfig.Logger, true),
			"debug":             smtpConfig.Debug,
			"disableFileAccess": c.getBoolOrDefault(smtpConfig.DisableFileAccess, true),
			"disableUrlAccess":  c.getBoolOrDefault(smtpConfig.DisableUrlAccess, true),
			"secure":            smtpConfig.Secure,
			"tls":               map[string]interface{}{},
			"requireTLS":        smtpConfig.TLS,
			"customHeaders":     c.getCustomHeadersOrDefault(smtpConfig.CustomHeaders),
		},
	}

	// Find existing SMTP connector or use default connector ID
	connectors, err := c.GetConnectors()
	if err != nil {
		return fmt.Errorf("failed to get existing connectors: %w", err)
	}

	logger.Debug("Found %d existing connectors", len(connectors))
	for i, connector := range connectors {
		if connectorType, ok := connector["connectorId"].(string); ok {
			logger.Debug("Connector %d: ID=%s, connectorId=%s", i, connector["id"], connectorType)
		}
	}

	smtpConnectorID := ""
	for _, connector := range connectors {
		if connectorType, ok := connector["connectorId"].(string); ok {
			// Check for SMTP connector
			if connectorType == "simple-mail-transfer-protocol" {
				if id, ok := connector["id"].(string); ok {
					smtpConnectorID = id
					logger.Debug("Found existing SMTP connector with ID: %s", smtpConnectorID)
					break
				}
			}
		}
	}

	// If no SMTP connector found, create one
	if smtpConnectorID == "" {
		logger.Info("No SMTP connector found, creating new one")

		connector, err := c.CreateConnector("simple-mail-transfer-protocol", connectorConfig)
		if err != nil {
			return fmt.Errorf("failed to create SMTP connector: %w", err)
		}

		if id, ok := connector["id"].(string); ok {
			smtpConnectorID = id
			logger.Info("Created SMTP connector with ID: %s", smtpConnectorID)
		} else {
			return fmt.Errorf("failed to get ID from created SMTP connector")
		}
	} else {
		// Update existing connector
		err = c.UpdateConnector(smtpConnectorID, connectorConfig)
		if err != nil {
			return fmt.Errorf("failed to update SMTP connector: %w", err)
		}
		logger.Info("Updated SMTP connector with ID: %s", smtpConnectorID)
	}

	logger.Info("SMTP connector synchronized successfully")
	return nil
}

// Helper methods for configuration defaults
func (c *LogtoClient) getPortOrDefault(port int) int {
	if port == 0 {
		return 587 // Default SMTP port
	}
	return port
}

func (c *LogtoClient) getBoolOrDefault(value bool, defaultValue bool) bool {
	return value || defaultValue
}

func (c *LogtoClient) getCustomHeadersOrDefault(headers map[string]string) map[string]interface{} {
	if headers == nil {
		return map[string]interface{}{}
	}
	result := make(map[string]interface{})
	for k, v := range headers {
		result[k] = v
	}
	return result
}

// loadEmailTemplates loads email templates from the configs/connectors directory
func (c *LogtoClient) loadEmailTemplates(templateSettings *config.SMTPTemplateSettings) ([]map[string]interface{}, error) {
	templates := []map[string]interface{}{}

	// Define template configurations
	templateConfigs := []struct {
		usageType string
		subject   string
		htmlFile  string
	}{
		{
			usageType: "SignIn",
			subject:   "Sign In Verification Code",
			htmlFile:  "",
		},
		{
			usageType: "Register",
			subject:   "Registration Verification Code",
			htmlFile:  "",
		},
		{
			usageType: "ForgotPassword",
			subject:   "Password Reset Verification Code",
			htmlFile:  "forgot-password.html",
		},
		{
			usageType: "Generic",
			subject:   "Verification Code",
			htmlFile:  "",
		},
	}

	for _, tc := range templateConfigs {
		// For ForgotPassword, load HTML template from file
		if tc.usageType == "ForgotPassword" && tc.htmlFile != "" {
			htmlContent, err := c.loadTemplateFile(tc.htmlFile, templateSettings)
			if err != nil {
				return nil, fmt.Errorf("failed to load HTML template %s: %w", tc.htmlFile, err)
			}

			// Use HTML template for ForgotPassword
			templates = append(templates, map[string]interface{}{
				"content":     htmlContent,
				"subject":     tc.subject,
				"usageType":   tc.usageType,
				"contentType": "text/html",
			})
		} else {
			// For other templates, use default plain text content
			content := "Your verification code is {{code}}. The code will remain active for 10 minutes."
			templates = append(templates, map[string]interface{}{
				"content":     content,
				"subject":     tc.subject,
				"usageType":   tc.usageType,
				"contentType": "text/plain",
			})
		}
	}

	logger.Debug("Loaded %d email templates", len(templates))
	return templates, nil
}

// loadTemplateFile loads a template file from configs/connectors directory
func (c *LogtoClient) loadTemplateFile(filename string, templateSettings *config.SMTPTemplateSettings) (string, error) {
	// Construct path relative to sync directory
	templatePath := filepath.Join("configs", "connectors", filename)

	// Check if file exists
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return "", fmt.Errorf("template file not found: %s", templatePath)
	}

	// Read file content
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template file %s: %w", templatePath, err)
	}

	// Process template content to replace variables with configuration values or defaults
	templateContent := string(content)

	// Get values from template settings or use defaults
	supportEmail := "support@nethesis.it"
	companyName := "Nethesis S.r.l."

	if templateSettings != nil {
		if templateSettings.SupportEmail != "" {
			supportEmail = templateSettings.SupportEmail
		}
		if templateSettings.CompanyName != "" {
			companyName = templateSettings.CompanyName
		}
	}

	templateContent = strings.ReplaceAll(templateContent, "{{.SupportEmail}}", supportEmail)
	templateContent = strings.ReplaceAll(templateContent, "{{.CompanyName}}", companyName)

	logger.Debug("Loaded template from %s (%d characters)", templatePath, len(templateContent))
	return templateContent, nil
}

// getFormattedFromEmail formats the from email with display name if provided
func (c *LogtoClient) getFormattedFromEmail(smtpConfig *config.SMTPConnector) string {
	if smtpConfig.FromName != "" {
		// Format as "Display Name <email@domain.com>"
		return fmt.Sprintf("%s <%s>", smtpConfig.FromName, smtpConfig.FromEmail)
	}
	// Return just the email if no name is provided
	return smtpConfig.FromEmail
}
