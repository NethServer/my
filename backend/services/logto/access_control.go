/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package logto

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ApplicationAccessControl is the rule set of Logto app-level access control
// for one application. Logto evaluates the four lists with OR: a user signs in
// when it matches any of them. The backend only ever fills organizationIds.
type ApplicationAccessControl struct {
	UserIDs               []string                          `json:"userIds"`
	UserRoleIDs           []string                          `json:"userRoleIds"`
	OrganizationIDs       []string                          `json:"organizationIds"`
	OrganizationRoleRules []ApplicationOrganizationRoleRule `json:"organizationRoleRules"`
}

// ApplicationOrganizationRoleRule admits the members of one organization that
// hold one of the given organization roles.
type ApplicationOrganizationRoleRule struct {
	OrganizationID      string   `json:"organizationId"`
	OrganizationRoleIDs []string `json:"organizationRoleIds"`
}

// GetApplicationAccessControl reads the current rule set of an application.
func (c *LogtoManagementClient) GetApplicationAccessControl(appID string) (*ApplicationAccessControl, error) {
	resp, err := c.makeRequest("GET", "/applications/"+appID+"/access-control", nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	return decodeResponse[ApplicationAccessControl](resp, []int{http.StatusOK}, "get application access control")
}

// ReplaceApplicationAccessControl replaces the whole rule set of an
// application (Logto's PUT has replace semantics, not merge).
func (c *LogtoManagementClient) ReplaceApplicationAccessControl(appID string, accessControl ApplicationAccessControl) error {
	// Logto wants every list present as an array, never null.
	if accessControl.UserIDs == nil {
		accessControl.UserIDs = []string{}
	}
	if accessControl.UserRoleIDs == nil {
		accessControl.UserRoleIDs = []string{}
	}
	if accessControl.OrganizationIDs == nil {
		accessControl.OrganizationIDs = []string{}
	}
	if accessControl.OrganizationRoleRules == nil {
		accessControl.OrganizationRoleRules = []ApplicationOrganizationRoleRule{}
	}

	body, err := json.Marshal(accessControl)
	if err != nil {
		return fmt.Errorf("failed to marshal access control: %w", err)
	}

	resp, err := c.makeRequest("PUT", "/applications/"+appID+"/access-control", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to replace application access control, status %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// SetApplicationAccessControlEnabled turns Logto app-level access control on
// or off for an application. With it off Logto ignores the rule set and every
// registered user may sign in to the application.
func (c *LogtoManagementClient) SetApplicationAccessControlEnabled(appID string, enabled bool) error {
	body, err := json.Marshal(map[string]bool{"appLevelAccessControlEnabled": enabled})
	if err != nil {
		return fmt.Errorf("failed to marshal application patch: %w", err)
	}

	resp, err := c.makeRequest("PATCH", "/applications/"+appID, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to set application access control enabled=%t, status %d: %s", enabled, resp.StatusCode, string(respBody))
	}
	return nil
}
