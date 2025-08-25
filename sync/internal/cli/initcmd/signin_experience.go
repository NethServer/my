/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package initcmd

import (
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/nethesis/my/sync/internal/client"
	"github.com/nethesis/my/sync/internal/logger"
)

// SignInExperienceFiles represents local files and colors for sign-in experience customization
type SignInExperienceFiles struct {
	LogoPath        string
	DarkLogoPath    string
	FaviconPath     string
	DarkFaviconPath string
	CustomCSSPath   string
	BrandColor      string
	BrandColorDark  string
}

// SignInExperienceAssets represents the processed assets for sign-in experience
type SignInExperienceAssets struct {
	Logo        string
	DarkLogo    string
	Favicon     string
	DarkFavicon string
	CustomCSS   string
}

// loadFileAsDataURL loads a file and converts it to a data URL
func loadFileAsDataURL(filePath string) (string, error) {
	if filePath == "" {
		return "", nil
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		logger.Debug("File %s does not exist, skipping", filePath)
		return "", nil
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
func loadTextFile(filePath string) (string, error) {
	if filePath == "" {
		return "", nil
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		logger.Debug("File %s does not exist, skipping", filePath)
		return "", nil
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

// LoadSignInExperienceAssets loads and processes all sign-in experience assets
func LoadSignInExperienceAssets(files SignInExperienceFiles) (*SignInExperienceAssets, error) {
	assets := &SignInExperienceAssets{}

	// Load image assets as data URLs
	var err error

	assets.Logo, err = loadFileAsDataURL(files.LogoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load logo: %w", err)
	}

	assets.DarkLogo, err = loadFileAsDataURL(files.DarkLogoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load dark logo: %w", err)
	}

	assets.Favicon, err = loadFileAsDataURL(files.FaviconPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load favicon: %w", err)
	}

	assets.DarkFavicon, err = loadFileAsDataURL(files.DarkFaviconPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load dark favicon: %w", err)
	}

	// Load CSS as text
	assets.CustomCSS, err = loadTextFile(files.CustomCSSPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load custom CSS: %w", err)
	}

	return assets, nil
}

// ConfigureSignInExperience configures the default sign-in experience
func ConfigureSignInExperience(logtoClient *client.LogtoClient, files SignInExperienceFiles) error {
	logger.Info("Configuring default sign-in experience...")

	// Load assets from local files
	assets, err := LoadSignInExperienceAssets(files)
	if err != nil {
		return fmt.Errorf("failed to load sign-in experience assets: %w", err)
	}

	// Create sign-in experience configuration
	config := client.SignInExperienceConfig{
		Color: &client.SignInExperienceColor{
			PrimaryColor:      files.BrandColor,
			IsDarkModeEnabled: true,
			DarkPrimaryColor:  files.BrandColorDark,
		},
		LanguageInfo: &client.SignInExperienceLanguageInfo{
			AutoDetect:       true,
			FallbackLanguage: "en",
		},
		SignIn: &client.SignInExperienceSignIn{
			Methods: []client.SignInExperienceSignInMethod{
				{
					Identifier:        "email",
					Password:          true,
					VerificationCode:  false,
					IsPasswordPrimary: true,
				},
			},
		},
		SignUp: &client.SignInExperienceSignUp{
			Identifiers:          []string{},
			Password:             false,
			Verify:               false,
			SecondaryIdentifiers: []string{},
		},
		SocialSignIn: map[string]interface{}{},
	}

	// Add branding if assets are available
	if assets.Logo != "" || assets.DarkLogo != "" || assets.Favicon != "" || assets.DarkFavicon != "" {
		config.Branding = &client.SignInExperienceBranding{}

		if assets.Logo != "" {
			config.Branding.LogoURL = assets.Logo
		}
		if assets.DarkLogo != "" {
			config.Branding.DarkLogoURL = assets.DarkLogo
		}
		if assets.Favicon != "" {
			config.Branding.Favicon = assets.Favicon
		}
		if assets.DarkFavicon != "" {
			config.Branding.DarkFavicon = assets.DarkFavicon
		}
	}

	// Add custom CSS if available
	if assets.CustomCSS != "" {
		config.CustomCSS = assets.CustomCSS
	}

	// Update sign-in experience
	if err := logtoClient.UpdateSignInExperience(config); err != nil {
		return fmt.Errorf("failed to update sign-in experience: %w", err)
	}

	logger.Info("Sign-in experience configured successfully")
	return nil
}
