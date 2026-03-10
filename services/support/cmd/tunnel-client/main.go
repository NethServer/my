/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

// tunnel-client connects to the support service WebSocket tunnel, advertises
// available services via a manifest, and handles incoming CONNECT requests
// by proxying traffic to local targets.
//
// Usage:
//
//	tunnel-client --url ws://support:8082/api/tunnel --key NETH-XXXX --secret my_xxx.yyy \
//	  [--static-services cluster-admin=localhost:9090] \
//	  [--redis-addr localhost:6379]
package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"
	"github.com/redis/go-redis/v9"
	"gopkg.in/yaml.v3"
)

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

// tunnelClientConfig is the YAML configuration file for the tunnel client
type tunnelClientConfig struct {
	Exclude []string `yaml:"exclude"`
}

// Defaults — all overridable via CLI flags or environment variables
const (
	defaultRedisAddr         = "127.0.0.1:6379"
	defaultReconnectDelay    = 5 * time.Second
	defaultMaxReconnect      = 5 * time.Minute
	defaultDiscoveryInterval = 5 * time.Minute
	defaultShell             = "/bin/bash"
	defaultTermEnv           = "TERM=xterm-256color"
	defaultYamuxKeepAlive    = 30 // seconds
	defaultRemoteHTTPSPort   = "443"
	defaultNethSecUIPort     = "443"
	maxFrameSize             = 1024 * 1024 // 1 MB
	maxLineLength            = 1024
	redisPingTimeout         = 2 * time.Second

	// NethSecurity detection paths
	nethSecUIPath    = "/www-ns/index.html"
	nethSecNginxConf = "/etc/nginx/conf.d/ns-ui.conf"
	ns8NodeEnvFile   = "/var/lib/nethserver/node/state/environment"
)

// defaultExclude filters out backend API routes that are not useful for
// support operators. Only UI-facing services (cluster-admin, *-ui, *-wizard,
// *-reports-ui, *-amld, *_grafana, n8n*, nethsecurity-controller*) are kept.
var defaultExclude = []string{
	"*-cti-server-api",
	"*-janus",
	"*-middleware-*",
	"*-provisioning",
	"*-reports-api",
	"*-server-api",
	"*-server-websocket",
	"*-tancredi",
	"*_loki",
	"*_prometheus",
}

func main() {
	var (
		urlFlag           = flag.StringP("url", "u", envWithDefault("SUPPORT_URL", ""), "WebSocket tunnel URL (env: SUPPORT_URL)")
		keyFlag           = flag.StringP("key", "k", envWithDefault("SYSTEM_KEY", ""), "System key (env: SYSTEM_KEY)")
		secretFlag        = flag.StringP("secret", "s", envWithDefault("SYSTEM_SECRET", ""), "System secret (env: SYSTEM_SECRET)")
		nodeIDFlag        = flag.StringP("node-id", "n", envWithDefault("NODE_ID", ""), "Cluster node ID, auto-detected on NS8 (env: NODE_ID)")
		redisAddr         = flag.StringP("redis-addr", "r", envWithDefault("REDIS_ADDR", ""), "Redis address, auto-detected on NS8 (env: REDIS_ADDR)")
		staticServices    = flag.String("static-services", envWithDefault("STATIC_SERVICES", ""), "Static services name=host:port[:tls],… (env: STATIC_SERVICES)")
		configFile        = flag.StringP("config", "c", envWithDefault("TUNNEL_CONFIG", ""), "YAML config file for exclusions (env: TUNNEL_CONFIG)")
		reconnectDelay    = flag.Duration("reconnect-delay", parseDurationDefault(envWithDefault("RECONNECT_DELAY", ""), defaultReconnectDelay), "Base reconnect delay (env: RECONNECT_DELAY)")
		maxReconnectDelay = flag.Duration("max-reconnect-delay", parseDurationDefault(envWithDefault("MAX_RECONNECT_DELAY", ""), defaultMaxReconnect), "Max reconnect delay (env: MAX_RECONNECT_DELAY)")
		discoveryInterval = flag.Duration("discovery-interval", parseDurationDefault(envWithDefault("DISCOVERY_INTERVAL", ""), defaultDiscoveryInterval), "Service re-discovery interval (env: DISCOVERY_INTERVAL)")
		tlsInsecure       = flag.Bool("tls-insecure", envWithDefault("TLS_INSECURE", "") == "true", "Skip TLS verification (env: TLS_INSECURE)")
	)
	flag.Parse()

	if *urlFlag == "" || *keyFlag == "" || *secretFlag == "" {
		fmt.Fprintln(os.Stderr, "Usage: tunnel-client --url URL --key KEY --secret SECRET [options]")
		fmt.Fprintln(os.Stderr, "  Required: --url, --key, --secret (or SUPPORT_URL, SYSTEM_KEY, SYSTEM_SECRET env vars)")
		os.Exit(1)
	}

	// Auto-detect Redis on localhost if not explicitly specified
	if *redisAddr == "" {
		rdb := redis.NewClient(&redis.Options{Addr: defaultRedisAddr})
		ctx, cancel := context.WithTimeout(context.Background(), redisPingTimeout)
		if err := rdb.Ping(ctx).Err(); err == nil {
			log.Printf("Redis detected at %s, enabling NS8 auto-discovery", defaultRedisAddr)
			*redisAddr = defaultRedisAddr
		} else {
			log.Printf("No Redis at %s, skipping NS8 auto-discovery (use -redis-addr to specify)", defaultRedisAddr)
		}
		cancel()
		_ = rdb.Close()
	}

	// Auto-detect node ID from NS8 environment if not explicitly specified
	if *nodeIDFlag == "" && *redisAddr != "" {
		if nid := readNodeID(); nid != "" {
			log.Printf("Auto-detected node ID: %s", nid)
			*nodeIDFlag = nid
		}
	}

	// Build exclusion list: start with defaults, add config file overrides
	exclude := append([]string{}, defaultExclude...)
	if *configFile != "" {
		if tc, err := loadConfig(*configFile); err != nil {
			log.Printf("Warning: cannot load config %s: %v", *configFile, err)
		} else if len(tc.Exclude) > 0 {
			exclude = append(exclude, tc.Exclude...)
		}
	}
	log.Printf("Excluding %d service patterns: %v", len(exclude), exclude)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Shutting down...")
		cancel()
	}()

	config := &clientConfig{
		url:               *urlFlag,
		key:               *keyFlag,
		secret:            *secretFlag,
		nodeID:            *nodeIDFlag,
		redisAddr:         *redisAddr,
		staticServices:    *staticServices,
		configFile:        *configFile,
		exclude:           exclude,
		reconnectDelay:    *reconnectDelay,
		maxReconnectDelay: *maxReconnectDelay,
		discoveryInterval: *discoveryInterval,
		tlsInsecure:       *tlsInsecure,
	}

	runWithReconnect(ctx, config)
}

type clientConfig struct {
	url               string
	key               string
	secret            string
	nodeID            string
	redisAddr         string
	staticServices    string
	configFile        string
	reconnectDelay    time.Duration
	maxReconnectDelay time.Duration
	discoveryInterval time.Duration
	tlsInsecure       bool
	exclude           []string // loaded from config file
}

// closeCodeSessionClosed matches the server's CloseCodeSessionClosed.
// When the operator closes a session, the server sends this code
// to tell the client to exit without reconnecting.
const closeCodeSessionClosed = 4000

func runWithReconnect(ctx context.Context, cfg *clientConfig) {
	delay := cfg.reconnectDelay

	for {
		start := time.Now()
		err := connect(ctx, cfg)
		if ctx.Err() != nil {
			return // context cancelled, clean shutdown
		}

		// Check if the server sent a "session closed" close frame
		if websocket.IsCloseError(err, closeCodeSessionClosed) {
			log.Println("Session closed by operator. Exiting.")
			os.Exit(0)
		}

		log.Printf("Connection lost: %v", err)

		// Reset backoff if connection lasted longer than 60 seconds
		if time.Since(start) > 60*time.Second {
			delay = cfg.reconnectDelay
		}

		log.Printf("Reconnecting in %v...", delay)

		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}

		// Exponential backoff
		delay = delay * 2
		if delay > cfg.maxReconnectDelay {
			delay = cfg.maxReconnectDelay
		}
	}
}

func connect(ctx context.Context, cfg *clientConfig) error {
	// Build Basic Auth header
	creds := base64.StdEncoding.EncodeToString([]byte(cfg.key + ":" + cfg.secret))
	header := http.Header{}
	header.Set("Authorization", "Basic "+creds)

	// Append node_id query parameter for multi-node clusters
	connectURL := cfg.url
	if cfg.nodeID != "" {
		parsed, err := url.Parse(connectURL)
		if err != nil {
			return fmt.Errorf("invalid URL: %w", err)
		}
		q := parsed.Query()
		q.Set("node_id", cfg.nodeID)
		parsed.RawQuery = q.Encode()
		connectURL = parsed.String()
	}

	log.Printf("Connecting to %s ...", connectURL)

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: cfg.tlsInsecure, //nolint:gosec // Configurable: disabled by default, enable for dev/self-signed certs
		},
	}
	wsConn, _, err := dialer.Dial(connectURL, header)
	if err != nil {
		return fmt.Errorf("websocket dial failed: %w", err)
	}
	log.Println("WebSocket connected")

	// Wrap as net.Conn
	netConn := &wsNetConn{conn: wsConn}

	// Create yamux client session
	yamuxCfg := yamux.DefaultConfig()
	yamuxCfg.EnableKeepAlive = true
	yamuxCfg.KeepAliveInterval = defaultYamuxKeepAlive
	yamuxCfg.LogOutput = io.Discard

	session, err := yamux.Client(netConn, yamuxCfg)
	if err != nil {
		_ = wsConn.Close()
		return fmt.Errorf("yamux client creation failed: %w", err)
	}
	log.Println("yamux session established")

	// Discover services
	services := discoverServices(ctx, cfg)

	// Send initial manifest
	if err := sendManifest(session, services); err != nil {
		_ = session.Close()
		return fmt.Errorf("failed to send manifest: %w", err)
	}

	// Start periodic re-discovery
	go func() {
		ticker := time.NewTicker(cfg.discoveryInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-session.CloseChan():
				return
			case <-ticker.C:
				newServices := discoverServices(ctx, cfg)
				if len(newServices) > 0 {
					if err := sendManifest(session, newServices); err != nil {
						log.Printf("Failed to send updated manifest: %v", err)
					} else {
						services = newServices
						log.Printf("Manifest updated with %d services", len(services))
					}
				}
			}
		}
	}()

	// Close session when context is cancelled to unblock Accept()
	go func() {
		<-ctx.Done()
		_ = session.Close()
	}()

	// Accept incoming streams
	for {
		stream, err := session.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			// If the underlying WebSocket received a close frame, return that error
			// so the reconnect loop can inspect the close code
			netConn.mu.Lock()
			closeErr := netConn.closeErr
			netConn.mu.Unlock()
			if closeErr != nil {
				return closeErr
			}
			return fmt.Errorf("stream accept error: %w", err)
		}
		go handleStream(stream, services)
	}
}

func discoverServices(ctx context.Context, cfg *clientConfig) map[string]ServiceInfo {
	services := make(map[string]ServiceInfo)

	// Parse static services
	if cfg.staticServices != "" {
		for _, entry := range strings.Split(cfg.staticServices, ",") {
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
			if isExcluded(name, cfg.exclude) {
				continue
			}
			target := parts[1]

			svc := ServiceInfo{Label: name}

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

	// NS8 auto-discovery from Traefik config files
	if cfg.redisAddr != "" {
		discovered := discoverTraefikRoutes(ctx, cfg.redisAddr)
		for name, svc := range discovered {
			if isExcluded(name, cfg.exclude) {
				continue
			}
			services[name] = svc
		}
	}

	// NethSecurity auto-discovery (OpenWrt-based, no Redis/Traefik)
	if cfg.redisAddr == "" {
		discovered := discoverNethSecurityServices()
		for name, svc := range discovered {
			if isExcluded(name, cfg.exclude) {
				continue
			}
			services[name] = svc
		}
	}

	logDiscoveredServices(services)

	return services
}

// logDiscoveredServices prints a structured summary grouped by node → module → service
func logDiscoveredServices(services map[string]ServiceInfo) {
	type moduleGroup struct {
		label    string
		services map[string]ServiceInfo
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
			mg = &moduleGroup{services: make(map[string]ServiceInfo)}
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

func loadConfig(path string) (*tunnelClientConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg tunnelClientConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func isExcluded(name string, patterns []string) bool {
	for _, pattern := range patterns {
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}
	return false
}

// apiCliRoute represents a single route returned by api-cli list-routes with expand_list
type apiCliRoute struct {
	Instance      string `json:"instance"`
	Host          string `json:"host"`
	Path          string `json:"path"`
	URL           string `json:"url"`
	StripPrefix   bool   `json:"strip_prefix"`
	SkipCertVerif bool   `json:"skip_cert_verify"`
}

// discoverTraefikRoutes uses api-cli to discover routes from ALL cluster nodes.
func discoverTraefikRoutes(ctx context.Context, redisAddr string) map[string]ServiceInfo {
	services := make(map[string]ServiceInfo)

	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	defer func() { _ = rdb.Close() }()

	// Discover all node IDs by scanning Redis keys
	nodeIDs := discoverNodeIDs(ctx, rdb)
	if len(nodeIDs) == 0 {
		log.Println("Traefik discovery: no nodes found, skipping")
		return services
	}

	// Read local NODE_ID to distinguish local vs remote nodes
	localNodeID := readNodeID()
	log.Printf("Traefik discovery: found %d node(s): %v (local: %s)", len(nodeIDs), nodeIDs, localNodeID)

	// Build a map of remote node IPs from Redis VPN config
	nodeIPs := make(map[string]string)
	for _, nid := range nodeIDs {
		if nid == localNodeID {
			continue
		}
		ip, err := rdb.HGet(ctx, fmt.Sprintf("node/%s/vpn", nid), "ip_address").Result()
		if err != nil {
			log.Printf("Traefik discovery: cannot get IP for node %s: %v", nid, err)
			continue
		}
		nodeIPs[nid] = ip
		log.Printf("Traefik discovery: node %s -> %s", nid, ip)
	}

	for _, nodeID := range nodeIDs {
		nodeServices := discoverNodeRoutes(ctx, rdb, nodeID)

		// For remote nodes, rewrite targets to go through the node's Traefik (HTTPS).
		// Traefik on the remote node handles TLS termination, Host-based routing,
		// and PathPrefix stripping, so we clear PathPrefix to avoid double-stripping.
		if nodeID != localNodeID {
			remoteIP, ok := nodeIPs[nodeID]
			if !ok {
				log.Printf("Traefik discovery: skipping node %s (no IP)", nodeID)
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

// readNodeID reads NODE_ID from the NS8 node environment file.
func readNodeID() string {
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
			log.Printf("Traefik discovery: Redis SCAN error: %v", err)
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
func discoverNodeRoutes(ctx context.Context, rdb *redis.Client, nodeID string) map[string]ServiceInfo {
	services := make(map[string]ServiceInfo)

	// Get the traefik instance name from Redis
	traefikInstance, err := rdb.Get(ctx, fmt.Sprintf("node/%s/default_instance/traefik", nodeID)).Result()
	if err != nil {
		log.Printf("Traefik discovery: cannot get traefik instance for node %s: %v", nodeID, err)
		return services
	}

	// Call api-cli to get all routes with details
	cmd := exec.CommandContext(ctx, "api-cli", "run",
		fmt.Sprintf("module/%s/list-routes", traefikInstance),
		"--data", `{"expand_list": true}`)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Traefik discovery: api-cli failed for %s (node %s): %v", traefikInstance, nodeID, err)
		return services
	}

	var routes []apiCliRoute
	if err := json.Unmarshal(output, &routes); err != nil {
		log.Printf("Traefik discovery: cannot parse api-cli output for %s: %v", traefikInstance, err)
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

		services[serviceKey] = ServiceInfo{
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

// moduleIDRegex matches NS8 module IDs (compiled once at package level)
var moduleIDRegex = regexp.MustCompile(`^(.+\d+)(?:[-_]|$)`)

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

// discoverNethSecurityServices detects NethSecurity (OpenWrt-based firewall)
// by checking for its web UI files and registers the main HTTPS service.
// NethSecurity runs nginx with the UI on a configurable port:
//   - Port from /etc/nginx/conf.d/ns-ui.conf (dedicated UI server block)
//   - Port 443 (when 00ns.locations is active, UI is on the default server)
func discoverNethSecurityServices() map[string]ServiceInfo {
	services := make(map[string]ServiceInfo)

	// Detect NethSecurity by checking for its UI directory
	if _, err := os.Stat(nethSecUIPath); err != nil {
		return services
	}

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "NethSecurity"
	}

	port := detectNethSecurityUIPort()

	log.Printf("NethSecurity detected (hostname: %s, UI port: %s), registering web UI service", hostname, port)

	services["nethsecurity-ui"] = ServiceInfo{
		Target: net.JoinHostPort("127.0.0.1", port),
		Host:   "127.0.0.1",
		TLS:    true,
		Label:  hostname,
		Path:   "/",
	}

	return services
}

// detectNethSecurityUIPort determines the HTTPS port serving the NethSecurity UI.
// It checks ns-ui.conf for a dedicated server block (e.g., port 9090), and
// falls back to 443 when the UI locations are on the default server.
func detectNethSecurityUIPort() string {
	// Check for dedicated UI server block (ns-ui.conf)
	data, err := os.ReadFile(nethSecNginxConf)
	if err == nil {
		// Parse "listen <port> ssl" directive
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "listen") && strings.Contains(line, "ssl") && !strings.Contains(line, "[::]:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					port := fields[1]
					// Validate it looks like a port number
					if _, err := fmt.Sscanf(port, "%d", new(int)); err == nil {
						return port
					}
				}
			}
		}
	}

	// Default: UI on the main server
	return defaultNethSecUIPort
}

func sendManifest(session *yamux.Session, services map[string]ServiceInfo) error {
	stream, err := session.Open()
	if err != nil {
		return fmt.Errorf("failed to open control stream: %w", err)
	}
	defer func() { _ = stream.Close() }()

	manifest := ServiceManifest{
		Version:  1,
		Services: services,
	}

	if err := json.NewEncoder(stream).Encode(manifest); err != nil {
		return fmt.Errorf("failed to encode manifest: %w", err)
	}

	log.Printf("Manifest sent with %d services", len(services))
	return nil
}

func handleStream(stream net.Conn, services map[string]ServiceInfo) {
	defer func() { _ = stream.Close() }()

	// Read CONNECT header
	serviceName, err := readConnectHeader(stream)
	if err != nil {
		log.Printf("Failed to read CONNECT header: %v", err)
		return
	}

	// Built-in terminal service: spawn a PTY instead of dialing TCP
	if serviceName == "terminal" {
		if err := writeConnectResponse(stream, nil); err != nil {
			return
		}
		log.Println("CONNECT terminal -> PTY")
		handleTerminal(stream)
		return
	}

	// Look up service
	svc, ok := services[serviceName]
	if !ok {
		_ = writeConnectResponse(stream, fmt.Errorf("service not found: %s", serviceName))
		return
	}

	// Connect to local target
	var targetConn net.Conn
	if svc.TLS {
		targetConn, err = tls.Dial("tcp", svc.Target, &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // Local services use self-signed certs
		})
	} else {
		targetConn, err = net.DialTimeout("tcp", svc.Target, 10*time.Second)
	}
	if err != nil {
		_ = writeConnectResponse(stream, fmt.Errorf("failed to connect to %s: %v", svc.Target, err))
		return
	}

	// Send OK response
	if err := writeConnectResponse(stream, nil); err != nil {
		_ = targetConn.Close()
		return
	}

	log.Printf("CONNECT %s -> %s", serviceName, svc.Target)

	// Bidirectional copy with proper cleanup to prevent goroutine leaks
	var once sync.Once
	done := make(chan struct{})
	closeBoth := func() {
		once.Do(func() {
			close(done)
			_ = targetConn.Close()
			_ = stream.Close()
		})
	}

	go func() {
		defer closeBoth()
		_, _ = io.Copy(targetConn, stream)
	}()

	go func() {
		defer closeBoth()
		_, _ = io.Copy(stream, targetConn)
	}()

	<-done
}

// readConnectHeader reads "CONNECT <service>\n" from the stream byte-by-byte
func readConnectHeader(r io.Reader) (string, error) {
	line, err := readLine(r)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(line, "CONNECT ") {
		return "", fmt.Errorf("invalid CONNECT header: %q", line)
	}
	name := strings.TrimPrefix(line, "CONNECT ")
	if name == "" {
		return "", fmt.Errorf("empty service name")
	}
	return name, nil
}

func writeConnectResponse(w io.Writer, err error) error {
	if err == nil {
		_, writeErr := fmt.Fprint(w, "OK\n")
		return writeErr
	}
	_, writeErr := fmt.Fprintf(w, "ERROR %s\n", err.Error())
	return writeErr
}

func readLine(r io.Reader) (string, error) {
	var buf []byte
	b := make([]byte, 1)
	for {
		n, err := r.Read(b)
		if n > 0 {
			if b[0] == '\n' {
				return string(buf), nil
			}
			buf = append(buf, b[0])
			if len(buf) > maxLineLength {
				return "", fmt.Errorf("line too long")
			}
		}
		if err != nil {
			if err == io.EOF && len(buf) > 0 {
				return string(buf), nil
			}
			return "", err
		}
	}
}

// wsNetConn wraps gorilla/websocket.Conn as net.Conn for yamux.
// It captures WebSocket close errors so the reconnect loop can inspect the close code.
type wsNetConn struct {
	conn     *websocket.Conn
	reader   io.Reader
	mu       sync.Mutex
	closeErr error // stores the WebSocket close error if received
}

func (w *wsNetConn) Read(b []byte) (int, error) {
	for {
		if w.reader == nil {
			_, reader, err := w.conn.NextReader()
			if err != nil {
				w.mu.Lock()
				w.closeErr = err
				w.mu.Unlock()
				return 0, err
			}
			w.reader = reader
		}
		n, err := w.reader.Read(b)
		if err == io.EOF {
			w.reader = nil
			if n > 0 {
				return n, nil
			}
			continue
		}
		return n, err
	}
}

func (w *wsNetConn) Write(b []byte) (int, error) {
	err := w.conn.WriteMessage(websocket.BinaryMessage, b)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

func (w *wsNetConn) Close() error         { return w.conn.Close() }
func (w *wsNetConn) LocalAddr() net.Addr  { return w.conn.LocalAddr() }
func (w *wsNetConn) RemoteAddr() net.Addr { return w.conn.RemoteAddr() }
func (w *wsNetConn) SetDeadline(t time.Time) error {
	if err := w.conn.SetReadDeadline(t); err != nil {
		return err
	}
	return w.conn.SetWriteDeadline(t)
}
func (w *wsNetConn) SetReadDeadline(t time.Time) error  { return w.conn.SetReadDeadline(t) }
func (w *wsNetConn) SetWriteDeadline(t time.Time) error { return w.conn.SetWriteDeadline(t) }

// sensitiveEnvPrefixes lists environment variable prefixes that are stripped
// from the PTY shell to prevent operators from extracting credentials (#8).
var sensitiveEnvPrefixes = []string{
	"SYSTEM_KEY=",
	"SYSTEM_SECRET=",
	"SUPPORT_URL=",
	"DATABASE_URL=",
	"REDIS_ADDR=",
	"REDIS_PASSWORD=",
	"REDIS_URL=",
	"INTERNAL_SECRET=",
	"TUNNEL_CONFIG=",
}

// sanitizeEnv filters out sensitive environment variables before spawning a shell
func sanitizeEnv(env []string) []string {
	filtered := make([]string, 0, len(env))
	for _, e := range env {
		sensitive := false
		for _, prefix := range sensitiveEnvPrefixes {
			if strings.HasPrefix(e, prefix) {
				sensitive = true
				break
			}
		}
		if !sensitive {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

func envWithDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func parseDurationDefault(s string, d time.Duration) time.Duration {
	if s == "" {
		return d
	}
	if v, err := time.ParseDuration(s); err == nil {
		return v
	}
	return d
}

// handleTerminal spawns a shell with a PTY and bridges it to the yamux stream
// using length-prefixed binary frames:
//   - Type 0 (data): raw terminal bytes (bidirectional)
//   - Type 1 (resize): JSON {"cols": N, "rows": N} (stream → PTY)
func handleTerminal(stream net.Conn) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = defaultShell
	}

	cmd := exec.Command(shell)
	cmd.Env = append(sanitizeEnv(os.Environ()), defaultTermEnv)

	ptmx, err := pty.Start(cmd)
	if err != nil {
		log.Printf("Failed to start PTY: %v", err)
		return
	}
	var once sync.Once
	done := make(chan struct{})
	closeAll := func() {
		once.Do(func() {
			close(done)
			_ = ptmx.Close()
			_ = stream.Close()
		})
	}
	defer func() {
		closeAll()
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	// PTY → stream: read from PTY, send as type-0 length-prefixed frames
	go func() {
		defer closeAll()
		buf := make([]byte, 4096)
		for {
			n, readErr := ptmx.Read(buf)
			if n > 0 {
				frame := make([]byte, 1+n)
				frame[0] = 0 // data frame
				copy(frame[1:], buf[:n])
				if writeErr := writeFrame(stream, frame); writeErr != nil {
					return
				}
			}
			if readErr != nil {
				return
			}
		}
	}()

	// Stream → PTY: read length-prefixed frames, dispatch by type
	go func() {
		defer closeAll()
		for {
			frame, readErr := readFrame(stream)
			if readErr != nil {
				return
			}
			if len(frame) < 1 {
				continue
			}

			frameType := frame[0]
			payload := frame[1:]

			switch frameType {
			case 0: // data → write to PTY
				if _, writeErr := ptmx.Write(payload); writeErr != nil {
					return
				}
			case 1: // resize → set PTY window size
				var size struct {
					Cols int `json:"cols"`
					Rows int `json:"rows"`
				}
				if jsonErr := json.Unmarshal(payload, &size); jsonErr != nil {
					continue
				}
				if size.Cols > 0 && size.Rows > 0 {
					_ = pty.Setsize(ptmx, &pty.Winsize{
						Rows: uint16(size.Rows),
						Cols: uint16(size.Cols),
					})
				}
			}
		}
	}()

	<-done
}

// writeFrame writes a length-prefixed frame: [4 bytes big-endian length][payload]
func writeFrame(w io.Writer, data []byte) error {
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, uint32(len(data)))
	if _, err := w.Write(header); err != nil {
		return err
	}
	_, err := w.Write(data)
	return err
}

// readFrame reads a length-prefixed frame: [4 bytes big-endian length][payload]
func readFrame(r io.Reader) ([]byte, error) {
	header := make([]byte, 4)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(header)
	if length > maxFrameSize {
		return nil, fmt.Errorf("frame too large: %d", length)
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}
	return data, nil
}
