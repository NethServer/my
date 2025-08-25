/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package sync

import (
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/nethesis/my/sync/internal/client"
	"github.com/nethesis/my/sync/internal/config"
	"github.com/nethesis/my/sync/internal/logger"
)

// syncSignInExperience synchronizes the sign-in experience configuration
func (e *Engine) syncSignInExperience(cfg *config.Config, result *Result) error {
	logger.Info("Synchronizing sign-in experience configuration")

	if cfg.SignInExperience == nil {
		logger.Debug("No sign-in experience configuration found, skipping")
		return nil
	}

	// Determine the base path for relative file paths
	// Use the directory containing the config file
	basePath := "configs" // Default fallback
	if e.options.ConfigFile != "" {
		basePath = filepath.Dir(e.options.ConfigFile)
		if basePath == "." {
			basePath = "configs"
		}
	}
	logger.Debug("Using base path for sign-in experience files: %s", basePath)

	// Build the sign-in experience configuration
	signInConfig, err := e.buildSignInExperienceConfig(cfg.SignInExperience, basePath)
	if err != nil {
		e.addOperation(result, "sign-in-experience", "build", "configuration", "Failed to build sign-in experience configuration", err)
		return fmt.Errorf("failed to build sign-in experience configuration: %w", err)
	}

	if e.options.DryRun {
		logger.Info("Would update sign-in experience configuration")
		e.addOperation(result, "sign-in-experience", "update", "configuration", "Would update sign-in experience configuration", nil)
		return nil
	}

	// Update the sign-in experience
	if err := e.client.UpdateSignInExperience(*signInConfig); err != nil {
		e.addOperation(result, "sign-in-experience", "update", "configuration", "Failed to update sign-in experience configuration", err)
		return fmt.Errorf("failed to update sign-in experience: %w", err)
	}

	logger.Info("Sign-in experience configuration updated successfully")
	e.addOperation(result, "sign-in-experience", "update", "configuration", "Updated sign-in experience configuration", nil)
	return nil
}

// buildSignInExperienceConfig builds the Logto sign-in experience configuration from config
func (e *Engine) buildSignInExperienceConfig(sie *config.SignInExperience, basePath string) (*client.SignInExperienceConfig, error) {
	signInConfig := &client.SignInExperienceConfig{}

	// Handle colors
	if sie.Colors != nil {
		signInConfig.Color = &client.SignInExperienceColor{
			PrimaryColor:      sie.Colors.PrimaryColor,
			IsDarkModeEnabled: sie.Colors.DarkModeEnabled,
			DarkPrimaryColor:  sie.Colors.PrimaryColorDark,
		}
	}

	// Handle branding (load files and convert to data URLs)
	if sie.Branding != nil {
		branding := &client.SignInExperienceBranding{}

		if sie.Branding.LogoPath != "" {
			fullPath := filepath.Join(basePath, sie.Branding.LogoPath)
			logoURL, err := e.loadFileAsDataURL(fullPath)
			if err != nil {
				logger.Debug("Failed to load logo from %s: %v", fullPath, err)
			} else {
				branding.LogoURL = logoURL
			}
		}

		if sie.Branding.LogoDarkPath != "" {
			fullPath := filepath.Join(basePath, sie.Branding.LogoDarkPath)
			darkLogoURL, err := e.loadFileAsDataURL(fullPath)
			if err != nil {
				logger.Debug("Failed to load dark logo from %s: %v", fullPath, err)
			} else {
				branding.DarkLogoURL = darkLogoURL
			}
		}

		if sie.Branding.FaviconPath != "" {
			fullPath := filepath.Join(basePath, sie.Branding.FaviconPath)
			faviconURL, err := e.loadFileAsDataURL(fullPath)
			if err != nil {
				logger.Debug("Failed to load favicon from %s: %v", fullPath, err)
			} else {
				branding.Favicon = faviconURL
			}
		}

		if sie.Branding.FaviconDarkPath != "" {
			fullPath := filepath.Join(basePath, sie.Branding.FaviconDarkPath)
			darkFaviconURL, err := e.loadFileAsDataURL(fullPath)
			if err != nil {
				logger.Debug("Failed to load dark favicon from %s: %v", fullPath, err)
			} else {
				branding.DarkFavicon = darkFaviconURL
			}
		}

		// Only set branding if at least one asset was loaded
		if branding.LogoURL != "" || branding.DarkLogoURL != "" || branding.Favicon != "" || branding.DarkFavicon != "" {
			signInConfig.Branding = branding
		}
	}

	// Handle custom CSS
	if sie.CustomCSSPath != "" {
		fullPath := filepath.Join(basePath, sie.CustomCSSPath)
		css, err := e.loadTextFile(fullPath)
		if err != nil {
			logger.Debug("Failed to load custom CSS from %s: %v", fullPath, err)
		} else {
			signInConfig.CustomCSS = css
		}
	}

	// Handle language
	if sie.Language != nil {
		signInConfig.LanguageInfo = &client.SignInExperienceLanguageInfo{
			AutoDetect:       sie.Language.AutoDetect,
			FallbackLanguage: sie.Language.FallbackLanguage,
		}
	}

	// Handle sign-in methods
	if sie.SignIn != nil && len(sie.SignIn.Methods) > 0 {
		methods := make([]client.SignInExperienceSignInMethod, len(sie.SignIn.Methods))
		for i, method := range sie.SignIn.Methods {
			methods[i] = client.SignInExperienceSignInMethod{
				Identifier:        method.Identifier,
				Password:          method.Password,
				VerificationCode:  method.VerificationCode,
				IsPasswordPrimary: method.IsPasswordPrimary,
			}
		}
		signInConfig.SignIn = &client.SignInExperienceSignIn{
			Methods: methods,
		}
	}

	// Handle sign-up
	if sie.SignUp != nil {
		signInConfig.SignUp = &client.SignInExperienceSignUp{
			Identifiers:          sie.SignUp.Identifiers,
			Password:             sie.SignUp.Password,
			Verify:               sie.SignUp.Verify,
			SecondaryIdentifiers: sie.SignUp.SecondaryIdentifiers,
		}
	}

	// Handle social sign-in
	if sie.SocialSignIn != nil {
		signInConfig.SocialSignIn = sie.SocialSignIn
	}

	return signInConfig, nil
}

// loadFileAsDataURL loads a file and converts it to a data URL
func (e *Engine) loadFileAsDataURL(filePath string) (string, error) {
	if filePath == "" {
		return "", nil
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("file does not exist: %s", filePath)
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Determine MIME type
	ext := strings.ToLower(filepath.Ext(filePath))
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		// Fallback for common types
		switch ext {
		case ".png":
			mimeType = "image/png"
		case ".jpg", ".jpeg":
			mimeType = "image/jpeg"
		case ".gif":
			mimeType = "image/gif"
		case ".svg":
			mimeType = "image/svg+xml"
		case ".ico":
			mimeType = "image/x-icon"
		default:
			mimeType = "application/octet-stream"
		}
	}

	// Convert to base64 data URL
	encoded := base64.StdEncoding.EncodeToString(content)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, encoded)

	logger.Debug("Loaded file %s as data URL (%d bytes)", filePath, len(content))
	return dataURL, nil
}

// loadTextFile loads a text file content
func (e *Engine) loadTextFile(filePath string) (string, error) {
	if filePath == "" {
		return "", nil
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("file does not exist: %s", filePath)
	}

	// Read file content
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer func() { _ = file.Close() }()

	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	logger.Debug("Loaded text file %s (%d bytes)", filePath, len(content))
	return string(content), nil
}
