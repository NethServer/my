/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package discovery

import (
	"context"
	"log"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nethesis/my/services/support/cmd/tunnel-client/internal/config"
	"github.com/nethesis/my/services/support/cmd/tunnel-client/internal/models"
)

// DiscoverServices discovers all available services from static configuration,
// Traefik routes (NS8), and NethSecurity detection.
func DiscoverServices(ctx context.Context, cfg *config.ClientConfig) map[string]models.ServiceInfo {
	services := make(map[string]models.ServiceInfo)

	// Parse static services
	if cfg.StaticServices != "" {
		for _, entry := range strings.Split(cfg.StaticServices, ",") {
			entry = strings.TrimSpace(entry)
			if entry == "" {
				continue
			}
			parts := strings.SplitN(entry, "=", 2)
			if len(parts) != 2 {
				log.Printf("Invalid static service entry: %s", entry)
				continue
			}
			name := parts[0]
			if IsExcluded(name, cfg.Exclude) {
				continue
			}
			target := parts[1]

			svc := models.ServiceInfo{Label: name}

			// Check for :tls suffix
			if strings.HasSuffix(target, ":tls") {
				svc.TLS = true
				target = strings.TrimSuffix(target, ":tls")
			}

			// Check for host override: name=target:port:host=hostname
			if idx := strings.Index(target, ":host="); idx != -1 {
				svc.Host = target[idx+6:]
				target = target[:idx]
			}

			svc.Target = target
			services[name] = svc
		}
	}

	// NethServer auto-discovery via api-cli and Redis
	if cfg.RedisAddr != "" {
		discovered := DiscoverNethServerServices(ctx, cfg.RedisAddr)
		for name, svc := range discovered {
			if IsExcluded(name, cfg.Exclude) {
				continue
			}
			services[name] = svc
		}
	}

	// NethSecurity auto-discovery (OpenWrt-based, no Redis/Traefik)
	if cfg.RedisAddr == "" {
		discovered := DiscoverNethSecurityServices()
		for name, svc := range discovered {
			if IsExcluded(name, cfg.Exclude) {
				continue
			}
			services[name] = svc
		}
	}

	LogDiscoveredServices(services)

	return services
}

// LogDiscoveredServices prints a structured summary grouped by node -> module -> service
func LogDiscoveredServices(services map[string]models.ServiceInfo) {
	type moduleGroup struct {
		label    string
		services map[string]models.ServiceInfo
	}
	type nodeGroup struct {
		modules   map[string]*moduleGroup
		ungrouped []string // service keys without moduleID
	}

	nodes := make(map[string]*nodeGroup) // keyed by nodeID ("" for non-node services)

	for name, svc := range services {
		nid := svc.NodeID
		ng, ok := nodes[nid]
		if !ok {
			ng = &nodeGroup{modules: make(map[string]*moduleGroup)}
			nodes[nid] = ng
		}

		if svc.ModuleID == "" {
			ng.ungrouped = append(ng.ungrouped, name)
			continue
		}

		mg, ok := ng.modules[svc.ModuleID]
		if !ok {
			mg = &moduleGroup{services: make(map[string]models.ServiceInfo)}
			ng.modules[svc.ModuleID] = mg
		}
		if mg.label == "" && svc.Label != "" {
			mg.label = svc.Label
		}
		mg.services[name] = svc
	}

	log.Printf("Discovered %d services across %d node(s)", len(services), len(nodes))

	// Sort node IDs (empty string = non-node services, printed last)
	nodeIDs := make([]string, 0, len(nodes))
	for nid := range nodes {
		nodeIDs = append(nodeIDs, nid)
	}
	sort.Strings(nodeIDs)

	for _, nid := range nodeIDs {
		ng := nodes[nid]

		if nid != "" {
			log.Printf("  Node %s:", nid)
		}

		indent := "  "
		if nid != "" {
			indent = "    "
		}

		// Print modules (sorted)
		moduleIDs := make([]string, 0, len(ng.modules))
		for id := range ng.modules {
			moduleIDs = append(moduleIDs, id)
		}
		sort.Strings(moduleIDs)

		for _, moduleID := range moduleIDs {
			mg := ng.modules[moduleID]
			if mg.label != "" {
				log.Printf("%s%s (%s)", indent, moduleID, mg.label)
			} else {
				log.Printf("%s%s", indent, moduleID)
			}
			names := make([]string, 0, len(mg.services))
			for name := range mg.services {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				svc := mg.services[name]
				route := svc.Host
				if svc.Path != "" && svc.Path != "/" {
					route += svc.Path
				}
				log.Printf("%s  - %s -> %s", indent, name, route)
			}
		}

		// Print ungrouped services (static, cluster-admin)
		sort.Strings(ng.ungrouped)
		for _, name := range ng.ungrouped {
			svc := services[name]
			log.Printf("%s%s -> %s", indent, name, svc.Target)
		}
	}
}

// IsExcluded checks if a service name matches any of the exclusion patterns.
func IsExcluded(name string, patterns []string) bool {
	for _, pattern := range patterns {
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}
	return false
}
