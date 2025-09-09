/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package models

// ThirdPartyApplication represents a third-party application from Logto
type ThirdPartyApplication struct {
	ID                     string               `json:"id"`
	Name                   string               `json:"name"`
	Description            string               `json:"description"`
	RedirectUris           []string             `json:"redirect_uris"`
	PostLogoutRedirectUris []string             `json:"post_logout_redirect_uris"`
	LoginURL               string               `json:"login_url"`
	Branding               *ApplicationBranding `json:"branding,omitempty"`
}

// ApplicationBranding represents branding information for an application
type ApplicationBranding struct {
	DisplayName string `json:"display_name"`
	LogoURL     string `json:"logo_url,omitempty"`
	DarkLogoURL string `json:"dark_logo_url,omitempty"`
}

// AccessControl defines which roles and organizations can access a third-party application
type AccessControl struct {
	OrganizationRoles []string `json:"organization_roles,omitempty"`
	UserRoles         []string `json:"user_roles,omitempty"`
	UserRoleIDs       []string `json:"user_role_ids,omitempty"`
	OrganizationIDs   []string `json:"organization_ids,omitempty"`
}

// LogtoThirdPartyApp represents the raw application data from Logto API
type LogtoThirdPartyApp struct {
	ID                 string                 `json:"id"`
	Name               string                 `json:"name"`
	Description        string                 `json:"description"`
	Type               string                 `json:"type"`
	IsThirdParty       bool                   `json:"isThirdParty"`
	CustomData         map[string]interface{} `json:"customData,omitempty"`
	OidcClientMetadata *OidcClientMetadata    `json:"oidcClientMetadata,omitempty"`
}

// OidcClientMetadata represents OIDC client metadata from Logto
type OidcClientMetadata struct {
	RedirectUris           []string `json:"redirectUris,omitempty"`
	PostLogoutRedirectUris []string `json:"postLogoutRedirectUris,omitempty"`
}

// ApplicationSignInExperience represents application branding from Logto
type ApplicationSignInExperience struct {
	DisplayName string                    `json:"displayName"`
	Branding    *LogtoApplicationBranding `json:"branding,omitempty"`
}

// LogtoApplicationBranding represents branding data from Logto API
type LogtoApplicationBranding struct {
	LogoURL     string `json:"logoUrl,omitempty"`
	DarkLogoURL string `json:"darkLogoUrl,omitempty"`
}

// ToThirdPartyApplication converts a LogtoThirdPartyApp to a ThirdPartyApplication
func (l *LogtoThirdPartyApp) ToThirdPartyApplication(branding *ApplicationSignInExperience, scopes []string, loginURLGenerator func(string, string, []string, bool) string, isValidDomain bool) *ThirdPartyApplication {
	app := &ThirdPartyApplication{
		ID:          l.ID,
		Name:        l.Name,
		Description: l.Description,
	}

	// Set branding information
	if branding != nil {
		// Include branding details if available
		if branding.Branding != nil {
			app.Branding = &ApplicationBranding{
				DisplayName: branding.DisplayName,
				LogoURL:     branding.Branding.LogoURL,
				DarkLogoURL: branding.Branding.DarkLogoURL,
			}
		} else {
			// Create basic branding with just display name
			app.Branding = &ApplicationBranding{
				DisplayName: branding.DisplayName,
			}
		}
	}

	// Extract OIDC metadata
	if l.OidcClientMetadata != nil {
		app.RedirectUris = l.OidcClientMetadata.RedirectUris
		app.PostLogoutRedirectUris = l.OidcClientMetadata.PostLogoutRedirectUris
	}

	// Use login URL from custom_data if available, otherwise generate it using redirect URI
	if l.CustomData != nil {
		if loginURLData, exists := l.CustomData["login_url"]; exists {
			if loginURLStr, ok := loginURLData.(string); ok && loginURLStr != "" {
				app.LoginURL = loginURLStr
			}
		}
	}

	// Fallback: Generate login URL using the first redirect URI if not provided in custom_data
	if app.LoginURL == "" && len(app.RedirectUris) > 0 && loginURLGenerator != nil {
		app.LoginURL = loginURLGenerator(l.ID, app.RedirectUris[0], scopes, isValidDomain)
	}

	return app
}

// ExtractAccessControlFromCustomData extracts access control configuration from Logto custom data
func (l *LogtoThirdPartyApp) ExtractAccessControlFromCustomData() *AccessControl {
	if l.CustomData == nil {
		return nil
	}

	accessControlData, exists := l.CustomData["access_control"]
	if !exists {
		return nil
	}

	accessControlMap, ok := accessControlData.(map[string]interface{})
	if !ok {
		return nil
	}

	accessControl := &AccessControl{}

	if orgRoles, exists := accessControlMap["organization_roles"]; exists {
		if orgRolesList, ok := orgRoles.([]interface{}); ok {
			accessControl.OrganizationRoles = make([]string, 0, len(orgRolesList))
			for _, role := range orgRolesList {
				if roleStr, ok := role.(string); ok {
					accessControl.OrganizationRoles = append(accessControl.OrganizationRoles, roleStr)
				}
			}
		}
	}

	if userRoles, exists := accessControlMap["user_roles"]; exists {
		if userRolesList, ok := userRoles.([]interface{}); ok {
			accessControl.UserRoles = make([]string, 0, len(userRolesList))
			for _, role := range userRolesList {
				if roleStr, ok := role.(string); ok {
					accessControl.UserRoles = append(accessControl.UserRoles, roleStr)
				}
			}
		}
	}

	if userRoleIDs, exists := accessControlMap["user_role_ids"]; exists {
		if userRoleIDsList, ok := userRoleIDs.([]interface{}); ok {
			accessControl.UserRoleIDs = make([]string, 0, len(userRoleIDsList))
			for _, roleID := range userRoleIDsList {
				if roleIDStr, ok := roleID.(string); ok {
					accessControl.UserRoleIDs = append(accessControl.UserRoleIDs, roleIDStr)
				}
			}
		}
	}

	if orgIDs, exists := accessControlMap["organization_ids"]; exists {
		if orgIDsList, ok := orgIDs.([]interface{}); ok {
			accessControl.OrganizationIDs = make([]string, 0, len(orgIDsList))
			for _, orgID := range orgIDsList {
				if orgIDStr, ok := orgID.(string); ok {
					accessControl.OrganizationIDs = append(accessControl.OrganizationIDs, orgIDStr)
				}
			}
		}
	}

	return accessControl
}
