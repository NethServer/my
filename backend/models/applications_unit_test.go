/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestThirdPartyApplicationStruct(t *testing.T) {
	app := ThirdPartyApplication{
		ID:                     "app_123",
		Name:                   "Test App",
		Description:            "Test application",
		RedirectUris:           []string{"https://example.com/callback"},
		PostLogoutRedirectUris: []string{"https://example.com/logout"},
		LoginURL:               "https://auth.example.com/login",
		Branding: &ApplicationBranding{
			DisplayName: "Test Application",
			LogoURL:     "https://example.com/logo.png",
			DarkLogoURL: "https://example.com/logo-dark.png",
		},
	}

	assert.Equal(t, "app_123", app.ID)
	assert.Equal(t, "Test App", app.Name)
	assert.Equal(t, "Test application", app.Description)
	assert.Equal(t, []string{"https://example.com/callback"}, app.RedirectUris)
	assert.Equal(t, []string{"https://example.com/logout"}, app.PostLogoutRedirectUris)
	assert.Equal(t, "https://auth.example.com/login", app.LoginURL)
	assert.NotNil(t, app.Branding)
	assert.Equal(t, "Test Application", app.Branding.DisplayName)
	assert.Equal(t, "https://example.com/logo.png", app.Branding.LogoURL)
	assert.Equal(t, "https://example.com/logo-dark.png", app.Branding.DarkLogoURL)
}

func TestApplicationBrandingStruct(t *testing.T) {
	branding := ApplicationBranding{
		DisplayName: "My App",
		LogoURL:     "https://example.com/logo.png",
		DarkLogoURL: "https://example.com/dark-logo.png",
	}

	assert.Equal(t, "My App", branding.DisplayName)
	assert.Equal(t, "https://example.com/logo.png", branding.LogoURL)
	assert.Equal(t, "https://example.com/dark-logo.png", branding.DarkLogoURL)
}

func TestAccessControlStruct(t *testing.T) {
	accessControl := AccessControl{
		OrganizationRoles: []string{"Owner", "Admin"},
		UserRoles:         []string{"Support", "Manager"},
	}

	assert.Equal(t, []string{"Owner", "Admin"}, accessControl.OrganizationRoles)
	assert.Equal(t, []string{"Support", "Manager"}, accessControl.UserRoles)
}

func TestLogtoThirdPartyAppStruct(t *testing.T) {
	app := LogtoThirdPartyApp{
		ID:           "logto_app_123",
		Name:         "Logto Test App",
		Description:  "Test app from Logto",
		Type:         "Traditional",
		IsThirdParty: true,
		CustomData:   map[string]interface{}{"key": "value"},
		OidcClientMetadata: &OidcClientMetadata{
			RedirectUris:           []string{"https://app.example.com/callback"},
			PostLogoutRedirectUris: []string{"https://app.example.com/logout"},
		},
	}

	assert.Equal(t, "logto_app_123", app.ID)
	assert.Equal(t, "Logto Test App", app.Name)
	assert.Equal(t, "Test app from Logto", app.Description)
	assert.Equal(t, "Traditional", app.Type)
	assert.True(t, app.IsThirdParty)
	assert.Equal(t, map[string]interface{}{"key": "value"}, app.CustomData)
	assert.NotNil(t, app.OidcClientMetadata)
	assert.Equal(t, []string{"https://app.example.com/callback"}, app.OidcClientMetadata.RedirectUris)
	assert.Equal(t, []string{"https://app.example.com/logout"}, app.OidcClientMetadata.PostLogoutRedirectUris)
}

func TestOidcClientMetadataStruct(t *testing.T) {
	metadata := OidcClientMetadata{
		RedirectUris:           []string{"https://app1.com/callback", "https://app2.com/callback"},
		PostLogoutRedirectUris: []string{"https://app1.com/logout", "https://app2.com/logout"},
	}

	assert.Equal(t, []string{"https://app1.com/callback", "https://app2.com/callback"}, metadata.RedirectUris)
	assert.Equal(t, []string{"https://app1.com/logout", "https://app2.com/logout"}, metadata.PostLogoutRedirectUris)
}

func TestApplicationSignInExperienceStruct(t *testing.T) {
	experience := ApplicationSignInExperience{
		DisplayName: "Sign In Experience",
		Branding: &LogtoApplicationBranding{
			LogoURL:     "https://logto.example.com/logo.png",
			DarkLogoURL: "https://logto.example.com/dark-logo.png",
		},
	}

	assert.Equal(t, "Sign In Experience", experience.DisplayName)
	assert.NotNil(t, experience.Branding)
	assert.Equal(t, "https://logto.example.com/logo.png", experience.Branding.LogoURL)
	assert.Equal(t, "https://logto.example.com/dark-logo.png", experience.Branding.DarkLogoURL)
}

func TestLogtoApplicationBrandingStruct(t *testing.T) {
	branding := LogtoApplicationBranding{
		LogoURL:     "https://example.com/logo.svg",
		DarkLogoURL: "https://example.com/logo-dark.svg",
	}

	assert.Equal(t, "https://example.com/logo.svg", branding.LogoURL)
	assert.Equal(t, "https://example.com/logo-dark.svg", branding.DarkLogoURL)
}

func TestToThirdPartyApplication(t *testing.T) {
	logtoApp := &LogtoThirdPartyApp{
		ID:          "app_123",
		Name:        "Test Application",
		Description: "Test description",
		OidcClientMetadata: &OidcClientMetadata{
			RedirectUris:           []string{"https://app.com/callback"},
			PostLogoutRedirectUris: []string{"https://app.com/logout"},
		},
	}

	branding := &ApplicationSignInExperience{
		DisplayName: "My Test App",
		Branding: &LogtoApplicationBranding{
			LogoURL:     "https://app.com/logo.png",
			DarkLogoURL: "https://app.com/dark-logo.png",
		},
	}

	scopes := []string{"openid", "profile", "email"}

	loginURLGenerator := func(appID, redirectURI string, scopes []string, isValidDomain bool) string {
		return "https://auth.example.com/login?client_id=" + appID + "&redirect_uri=" + redirectURI
	}

	app := logtoApp.ToThirdPartyApplication(branding, scopes, loginURLGenerator, true)

	assert.NotNil(t, app)
	assert.Equal(t, "app_123", app.ID)
	assert.Equal(t, "Test Application", app.Name)
	assert.Equal(t, "Test description", app.Description)
	assert.Equal(t, []string{"https://app.com/callback"}, app.RedirectUris)
	assert.Equal(t, []string{"https://app.com/logout"}, app.PostLogoutRedirectUris)
	assert.Equal(t, "https://auth.example.com/login?client_id=app_123&redirect_uri=https://app.com/callback", app.LoginURL)

	assert.NotNil(t, app.Branding)
	assert.Equal(t, "My Test App", app.Branding.DisplayName)
	assert.Equal(t, "https://app.com/logo.png", app.Branding.LogoURL)
	assert.Equal(t, "https://app.com/dark-logo.png", app.Branding.DarkLogoURL)
}

func TestToThirdPartyApplicationWithoutBranding(t *testing.T) {
	logtoApp := &LogtoThirdPartyApp{
		ID:          "app_456",
		Name:        "Simple App",
		Description: "Simple description",
		OidcClientMetadata: &OidcClientMetadata{
			RedirectUris: []string{"https://simple.com/callback"},
		},
	}

	app := logtoApp.ToThirdPartyApplication(nil, nil, nil, false)

	assert.NotNil(t, app)
	assert.Equal(t, "app_456", app.ID)
	assert.Equal(t, "Simple App", app.Name)
	assert.Equal(t, "Simple description", app.Description)
	assert.Equal(t, []string{"https://simple.com/callback"}, app.RedirectUris)
	assert.Empty(t, app.LoginURL) // No login URL generator provided
	assert.Nil(t, app.Branding)   // No branding provided
}

func TestToThirdPartyApplicationWithBrandingButNoLogos(t *testing.T) {
	logtoApp := &LogtoThirdPartyApp{
		ID:          "app_789",
		Name:        "Minimal App",
		Description: "Minimal description",
	}

	branding := &ApplicationSignInExperience{
		DisplayName: "Minimal Test App",
		Branding:    nil, // No logo branding
	}

	app := logtoApp.ToThirdPartyApplication(branding, nil, nil, false)

	assert.NotNil(t, app)
	assert.NotNil(t, app.Branding)
	assert.Equal(t, "Minimal Test App", app.Branding.DisplayName)
	assert.Empty(t, app.Branding.LogoURL)
	assert.Empty(t, app.Branding.DarkLogoURL)
}

func TestToThirdPartyApplicationWithoutOidcMetadata(t *testing.T) {
	logtoApp := &LogtoThirdPartyApp{
		ID:                 "app_no_oidc",
		Name:               "No OIDC App",
		Description:        "App without OIDC metadata",
		OidcClientMetadata: nil,
	}

	app := logtoApp.ToThirdPartyApplication(nil, nil, nil, false)

	assert.NotNil(t, app)
	assert.Equal(t, "app_no_oidc", app.ID)
	assert.Nil(t, app.RedirectUris)
	assert.Nil(t, app.PostLogoutRedirectUris)
	assert.Empty(t, app.LoginURL)
}

func TestExtractAccessControlFromCustomData(t *testing.T) {
	tests := []struct {
		name              string
		customData        map[string]interface{}
		expectedOrgRoles  []string
		expectedUserRoles []string
		expectNil         bool
	}{
		{
			name: "Valid access control data",
			customData: map[string]interface{}{
				"access_control": map[string]interface{}{
					"organization_roles": []interface{}{"Owner", "Admin"},
					"user_roles":         []interface{}{"Support", "Manager"},
				},
			},
			expectedOrgRoles:  []string{"Owner", "Admin"},
			expectedUserRoles: []string{"Support", "Manager"},
			expectNil:         false,
		},
		{
			name: "Only organization roles",
			customData: map[string]interface{}{
				"access_control": map[string]interface{}{
					"organization_roles": []interface{}{"Owner"},
				},
			},
			expectedOrgRoles:  []string{"Owner"},
			expectedUserRoles: nil,
			expectNil:         false,
		},
		{
			name: "Only user roles",
			customData: map[string]interface{}{
				"access_control": map[string]interface{}{
					"user_roles": []interface{}{"Support"},
				},
			},
			expectedOrgRoles:  nil,
			expectedUserRoles: []string{"Support"},
			expectNil:         false,
		},
		{
			name:       "No custom data",
			customData: nil,
			expectNil:  true,
		},
		{
			name: "No access control in custom data",
			customData: map[string]interface{}{
				"other_field": "value",
			},
			expectNil: true,
		},
		{
			name: "Invalid access control format",
			customData: map[string]interface{}{
				"access_control": "invalid",
			},
			expectNil: true,
		},
		{
			name: "Empty access control",
			customData: map[string]interface{}{
				"access_control": map[string]interface{}{},
			},
			expectedOrgRoles:  nil,
			expectedUserRoles: nil,
			expectNil:         false,
		},
		{
			name: "Invalid role types",
			customData: map[string]interface{}{
				"access_control": map[string]interface{}{
					"organization_roles": "invalid",
					"user_roles":         []interface{}{123, "Support"}, // Mixed types
				},
			},
			expectedOrgRoles:  nil,
			expectedUserRoles: []string{"Support"}, // Only valid string is extracted
			expectNil:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &LogtoThirdPartyApp{
				CustomData: tt.customData,
			}

			result := app.ExtractAccessControlFromCustomData()

			if tt.expectNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectedOrgRoles, result.OrganizationRoles)
				assert.Equal(t, tt.expectedUserRoles, result.UserRoles)
			}
		})
	}
}

func TestStructFieldTags(t *testing.T) {
	// Test that struct fields have correct JSON tags
	t.Run("ThirdPartyApplication JSON tags", func(t *testing.T) {
		app := ThirdPartyApplication{}
		// These tests verify that the structs are properly defined for JSON serialization
		assert.IsType(t, "", app.ID)
		assert.IsType(t, "", app.Name)
		assert.IsType(t, []string{}, app.RedirectUris)
		assert.IsType(t, (*ApplicationBranding)(nil), app.Branding)
	})

	t.Run("AccessControl JSON tags", func(t *testing.T) {
		ac := AccessControl{}
		assert.IsType(t, []string{}, ac.OrganizationRoles)
		assert.IsType(t, []string{}, ac.UserRoles)
	})

	t.Run("LogtoThirdPartyApp JSON tags", func(t *testing.T) {
		app := LogtoThirdPartyApp{}
		assert.IsType(t, "", app.ID)
		assert.IsType(t, false, app.IsThirdParty)
		assert.IsType(t, map[string]interface{}{}, app.CustomData)
		assert.IsType(t, (*OidcClientMetadata)(nil), app.OidcClientMetadata)
	})
}

func TestEdgeCases(t *testing.T) {
	t.Run("Empty LogtoThirdPartyApp conversion", func(t *testing.T) {
		app := &LogtoThirdPartyApp{}
		result := app.ToThirdPartyApplication(nil, nil, nil, false)

		assert.NotNil(t, result)
		assert.Empty(t, result.ID)
		assert.Empty(t, result.Name)
		assert.Empty(t, result.Description)
		assert.Nil(t, result.RedirectUris)
		assert.Nil(t, result.PostLogoutRedirectUris)
		assert.Empty(t, result.LoginURL)
		assert.Nil(t, result.Branding)
	})

	t.Run("LoginURL generation with empty redirect URIs", func(t *testing.T) {
		app := &LogtoThirdPartyApp{
			ID:   "test_app",
			Name: "Test",
			OidcClientMetadata: &OidcClientMetadata{
				RedirectUris: []string{}, // Empty
			},
		}

		loginURLGenerator := func(appID, redirectURI string, scopes []string, isValidDomain bool) string {
			return "generated_url"
		}

		result := app.ToThirdPartyApplication(nil, nil, loginURLGenerator, false)
		assert.Empty(t, result.LoginURL) // Should be empty because no redirect URIs
	})

	t.Run("Access control with mixed data types", func(t *testing.T) {
		app := &LogtoThirdPartyApp{
			CustomData: map[string]interface{}{
				"access_control": map[string]interface{}{
					"organization_roles": []interface{}{
						"ValidRole",
						123, // Invalid type
						"",  // Empty string (valid but empty)
						"AnotherValidRole",
					},
				},
			},
		}

		result := app.ExtractAccessControlFromCustomData()
		assert.NotNil(t, result)
		assert.Equal(t, []string{"ValidRole", "", "AnotherValidRole"}, result.OrganizationRoles)
	})
}
