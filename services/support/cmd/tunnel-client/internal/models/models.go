/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package models

// ServiceInfo matches the support service's tunnel.ServiceInfo
type ServiceInfo struct {
	Target     string `json:"target"`
	Host       string `json:"host"`
	TLS        bool   `json:"tls"`
	Label      string `json:"label"`
	Path       string `json:"path,omitempty"`
	PathPrefix string `json:"path_prefix,omitempty"`
	ModuleID   string `json:"module_id,omitempty"`
	NodeID     string `json:"node_id,omitempty"`
}

// ServiceManifest is the JSON manifest sent to the support service
type ServiceManifest struct {
	Version  int                    `json:"version"`
	Services map[string]ServiceInfo `json:"services"`
}

// ApiCliRoute represents a single route returned by api-cli list-routes with expand_list
type ApiCliRoute struct {
	Instance      string `json:"instance"`
	Host          string `json:"host"`
	Path          string `json:"path"`
	URL           string `json:"url"`
	StripPrefix   bool   `json:"strip_prefix"`
	SkipCertVerif bool   `json:"skip_cert_verify"`
}
