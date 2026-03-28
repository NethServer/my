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
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"github.com/redis/go-redis/v9"

	"github.com/nethesis/my/services/support/cmd/tunnel-client/internal/models"
)

const (
	defaultRemoteHTTPSPort = "443"
	ns8NodeEnvFile         = "/var/lib/nethserver/node/state/environment"
)

// moduleIDRegex matches NS8 module IDs (compiled once at package level)
var moduleIDRegex = regexp.MustCompile(`^(.+\d+)(?:[-_]|$)`)

// DiscoverNethServerServices uses api-cli to discover routes from ALL cluster nodes.
func DiscoverNethServerServices(ctx context.Context, redisAddr string) map[string]models.ServiceInfo {
	services := make(map[string]models.ServiceInfo)

	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	defer func() { _ = rdb.Close() }()

	// Discover all node IDs by scanning Redis keys
	nodeIDs := discoverNodeIDs(ctx, rdb)
	if len(nodeIDs) == 0 {
		log.Println("NethServer discovery: no nodes found, skipping")
		return services
	}

	// Read local NODE_ID to distinguish local vs remote nodes
	localNodeID := ReadNodeID()
	log.Printf("NethServer discovery: found %d node(s): %v (local: %s)", len(nodeIDs), nodeIDs, localNodeID)

	// Build a map of remote node IPs from Redis VPN config
	nodeIPs := make(map[string]string)
	for _, nid := range nodeIDs {
		if nid == localNodeID {
			continue
		}
		ip, err := rdb.HGet(ctx, fmt.Sprintf("node/%s/vpn", nid), "ip_address").Result()
		if err != nil {
			log.Printf("NethServer discovery: cannot get IP for node %s: %v", nid, err)
			continue
		}
		nodeIPs[nid] = ip
		log.Printf("NethServer discovery: node %s -> %s", nid, ip)
	}

	for _, nodeID := range nodeIDs {
		nodeServices := discoverNodeRoutes(ctx, rdb, nodeID)

		// For remote nodes, rewrite targets to go through the node's Traefik (HTTPS).
		// Traefik on the remote node handles TLS termination, Host-based routing,
		// and PathPrefix stripping, so we clear PathPrefix to avoid double-stripping.
		if nodeID != localNodeID {
			remoteIP, ok := nodeIPs[nodeID]
			if !ok {
				log.Printf("NethServer discovery: skipping node %s (no IP)", nodeID)
				continue
			}
			for name, svc := range nodeServices {
				svc.Target = remoteIP + ":" + defaultRemoteHTTPSPort
				svc.TLS = true
				svc.PathPrefix = ""
				nodeServices[name] = svc
			}
		}

		for name, svc := range nodeServices {
			services[name] = svc
		}
	}

	return services
}

// ReadNodeID reads NODE_ID from the NS8 node environment file.
func ReadNodeID() string {
	f, err := os.Open(ns8NodeEnvFile)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "NODE_ID=") {
			return strings.TrimPrefix(line, "NODE_ID=")
		}
	}
	return ""
}

// discoverNodeIDs finds all NS8 node IDs by scanning Redis keys.
func discoverNodeIDs(ctx context.Context, rdb *redis.Client) []string {
	var nodeIDs []string
	var cursor uint64

	for {
		keys, nextCursor, err := rdb.Scan(ctx, cursor, "node/*/default_instance/traefik", 100).Result()
		if err != nil {
			log.Printf("NethServer discovery: Redis SCAN error: %v", err)
			return nodeIDs
		}

		for _, key := range keys {
			// key format: node/{NODE_ID}/default_instance/traefik
			parts := strings.Split(key, "/")
			if len(parts) >= 2 {
				nodeIDs = append(nodeIDs, parts[1])
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	sort.Strings(nodeIDs)
	return nodeIDs
}

// discoverNodeRoutes uses api-cli to get all routes from a node's Traefik instance.
func discoverNodeRoutes(ctx context.Context, rdb *redis.Client, nodeID string) map[string]models.ServiceInfo {
	services := make(map[string]models.ServiceInfo)

	// Get the traefik instance name from Redis
	traefikInstance, err := rdb.Get(ctx, fmt.Sprintf("node/%s/default_instance/traefik", nodeID)).Result()
	if err != nil {
		log.Printf("NethServer discovery: cannot get traefik instance for node %s: %v", nodeID, err)
		return services
	}

	// Call api-cli to get all routes with details.
	// api-cli authenticates to Redis using AGENT_ID, REDIS_USER, REDIS_PASSWORD
	// environment variables injected by the start-tunnel-client node action.
	cmd := exec.CommandContext(ctx, "api-cli", "run",
		fmt.Sprintf("module/%s/list-routes", traefikInstance),
		"--data", `{"expand_list": true}`)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("NethServer discovery: api-cli failed for %s (node %s): %v: %s", traefikInstance, nodeID, err, strings.TrimSpace(string(output)))
		return services
	}

	var routes []models.ApiCliRoute
	if err := json.Unmarshal(output, &routes); err != nil {
		log.Printf("NethServer discovery: cannot parse api-cli output for %s: %v", traefikInstance, err)
		return services
	}

	for _, route := range routes {
		serviceKey := route.Instance

		// Parse target from URL
		parsed, err := url.Parse(route.URL)
		if err != nil {
			continue
		}
		target := parsed.Host
		useTLS := parsed.Scheme == "https"

		// Determine PathPrefix (only if strip_prefix is true)
		var pathPrefix string
		if route.Path != "" && route.StripPrefix {
			pathPrefix = route.Path
		}

		// Extract module ID and look up its ui_name from Redis
		moduleID := extractModuleID(serviceKey)
		var moduleLabel string
		if moduleID != "" {
			uiName, err := rdb.Get(ctx, "module/"+moduleID+"/ui_name").Result()
			if err == nil && uiName != "" {
				moduleLabel = uiName
			}
		}

		services[serviceKey] = models.ServiceInfo{
			Target:     target,
			Host:       route.Host,
			TLS:        useTLS,
			Label:      moduleLabel,
			Path:       route.Path,
			PathPrefix: pathPrefix,
			ModuleID:   moduleID,
			NodeID:     nodeID,
		}
	}

	return services
}

// extractModuleID extracts the module ID from a Traefik config filename.
// NS8 module IDs end with an instance number (e.g., "nethvoice103", "n8n2",
// "nethsecurity-controller4"). Route suffixes are separated by hyphen or
// underscore after the digits (e.g., "nethvoice103-ui", "metrics1_grafana").
func extractModuleID(name string) string {
	m := moduleIDRegex.FindStringSubmatch(name)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}
