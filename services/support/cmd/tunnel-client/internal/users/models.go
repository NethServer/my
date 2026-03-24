/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package users

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

// SessionUsers is the full user provisioning result for a support session
type SessionUsers struct {
	SessionID     string            `json:"session_id"`
	Platform      string            `json:"platform"`
	ClusterAdmin  *UserCredential   `json:"cluster_admin,omitempty"`
	DomainUsers   []DomainUser      `json:"domain_users,omitempty"`
	LocalUsers    []UserCredential  `json:"local_users,omitempty"`
	Apps          []AppConfig       `json:"apps,omitempty"`
	Errors        []PluginError     `json:"errors,omitempty"`
	ModuleDomains map[string]string `json:"module_domains,omitempty"` // moduleID → user domain (e.g., "nethvoice103" → "sf.nethserver.net")
	CreatedAt     time.Time         `json:"created_at"`
}

// UsersReport is sent to the support service via the USERS yamux stream
type UsersReport struct {
	CreatedAt  time.Time    `json:"created_at"`
	DurationMs int64        `json:"duration_ms"`
	Users      SessionUsers `json:"users"`
}

// Provisioner creates and deletes ephemeral support users.
// Implementations are platform-specific (NS8 vs NethSecurity).
type Provisioner interface {
	Create(sessionID string) (*SessionUsers, error)
	Delete(users *SessionUsers) error
}

// ModuleServiceInfo describes a single service route for a module instance
type ModuleServiceInfo struct {
	Host       string `json:"host"`
	Path       string `json:"path,omitempty"`
	PathPrefix string `json:"path_prefix,omitempty"`
	TLS        bool   `json:"tls,omitempty"`
}

// ModuleInstance describes a single instance of a module (e.g., nethvoice103)
type ModuleInstance struct {
	ID       string                       `json:"id"`
	NodeID   string                       `json:"node_id,omitempty"`
	Label    string                       `json:"label,omitempty"`
	Domain   string                       `json:"domain,omitempty"`
	Services map[string]ModuleServiceInfo `json:"services"`
}

// ModuleContext is the context passed to a users.d plugin via --instances-file.
// It contains all instances of the module that the plugin manages.
type ModuleContext struct {
	Module    string           `json:"module"`
	Instances []ModuleInstance `json:"instances"`
}
