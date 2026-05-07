/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Registry is the on-disk catalog of orgs and users created by apitool.
// It also stores OIDC config and the owner credentials used for any subsequent
// privileged action. The file is gitignored and lives in the backend dir.
type Registry struct {
	Config Config          `json:"config"`
	Owner  OwnerCreds      `json:"owner"`
	Orgs   map[string]Org  `json:"orgs"`
	Users  map[string]User `json:"users"`
}

type Config struct {
	LogtoEndpoint string `json:"logto_endpoint"`
	LogtoAppID    string `json:"logto_app_id"`
	AuthBaseURL   string `json:"auth_base_url"`
	BackendURL    string `json:"backend_url"`
}

type OwnerCreds struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Org struct {
	Type      string    `json:"type"`
	LogtoID   string    `json:"logto_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type User struct {
	Email     string    `json:"email"`
	Username  string    `json:"username,omitempty"`
	Password  string    `json:"password"`
	LogtoID   string    `json:"logto_id"`
	OrgRole   string    `json:"org_role"`
	OrgID     string    `json:"org_id"`
	OrgName   string    `json:"org_name"`
	CreatedAt time.Time `json:"created_at"`
}

const registryFilename = ".api-registry.json"

// registryPath finds the registry file. Looks in cwd first, then in backend/
// (so the tool works whether invoked from repo root or backend/). When the
// file does not exist, returns the cwd path so saves create it there.
func registryPath() string {
	if _, err := os.Stat(registryFilename); err == nil {
		return registryFilename
	}
	if _, err := os.Stat(filepath.Join("backend", registryFilename)); err == nil {
		return filepath.Join("backend", registryFilename)
	}
	return registryFilename
}

func LoadRegistry() (*Registry, error) {
	path := registryPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Registry{
				Orgs:  map[string]Org{},
				Users: map[string]User{},
			}, nil
		}
		return nil, fmt.Errorf("reading registry: %w", err)
	}
	var r Registry
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parsing registry: %w", err)
	}
	if r.Orgs == nil {
		r.Orgs = map[string]Org{}
	}
	if r.Users == nil {
		r.Users = map[string]User{}
	}
	return &r, nil
}

func (r *Registry) Save() error {
	path := registryPath()
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding registry: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing registry: %w", err)
	}
	return nil
}
