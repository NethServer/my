/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package models

import "time"

// UserCredential holds username and password for an ephemeral support user
type UserCredential struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// DomainUser represents a user created on a specific LDAP/Samba domain
type DomainUser struct {
	Domain   string `json:"domain"`
	Module   string `json:"module"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// AppConfig describes an application configured for the support user
type AppConfig struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	URL   string `json:"url,omitempty"`
	Notes string `json:"notes,omitempty"`
}

// PluginError records a users.d plugin that failed during setup
type PluginError struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

// SessionUsersData is the user provisioning data for a support session
type SessionUsersData struct {
	SessionID     string            `json:"session_id"`
	Platform      string            `json:"platform"`
	ClusterAdmin  *UserCredential   `json:"cluster_admin,omitempty"`
	DomainUsers   []DomainUser      `json:"domain_users,omitempty"`
	LocalUsers    []UserCredential  `json:"local_users,omitempty"`
	Apps          []AppConfig       `json:"apps,omitempty"`
	Errors        []PluginError     `json:"errors,omitempty"`
	ModuleDomains map[string]string `json:"module_domains,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
}

// UsersReport is the report sent by the tunnel-client via the USERS yamux stream
type UsersReport struct {
	CreatedAt  time.Time        `json:"created_at"`
	DurationMs int64            `json:"duration_ms"`
	Users      SessionUsersData `json:"users"`
}
