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
//	SYSTEM_SECRET=my_xxx.yyy tunnel-client --url wss://support.example.com/api/tunnel --key NETH-XXXX \
//	  [--static-services cluster-admin=localhost:9090] \
//	  [--redis-addr localhost:6379]
//	tunnel-client --cleanup-only [--users-state-file PATH]
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/redis/go-redis/v9"

	"github.com/nethesis/my/services/support/cmd/tunnel-client/internal/config"
	"github.com/nethesis/my/services/support/cmd/tunnel-client/internal/connection"
	"github.com/nethesis/my/services/support/cmd/tunnel-client/internal/discovery"
	"github.com/nethesis/my/services/support/cmd/tunnel-client/internal/users"
)

func main() {
	var (
		urlFlag                  = flag.StringP("url", "u", config.EnvWithDefault("SUPPORT_URL", ""), "WebSocket tunnel URL (env: SUPPORT_URL)")
		keyFlag                  = flag.StringP("key", "k", config.EnvWithDefault("SYSTEM_KEY", ""), "System key (env: SYSTEM_KEY)")
		nodeIDFlag               = flag.StringP("node-id", "n", config.EnvWithDefault("NODE_ID", ""), "Cluster node ID, auto-detected on NS8 (env: NODE_ID)")
		redisAddr                = flag.StringP("redis-addr", "r", config.EnvWithDefault("REDIS_ADDR", ""), "Redis address, auto-detected on NS8 (env: REDIS_ADDR)")
		staticServices           = flag.String("static-services", config.EnvWithDefault("STATIC_SERVICES", ""), "Static services name=host:port[:tls],... (env: STATIC_SERVICES)")
		excludePatterns          = flag.String("exclude", config.EnvWithDefault("EXCLUDE_PATTERNS", ""), "Comma-separated glob patterns to exclude services (env: EXCLUDE_PATTERNS)")
		reconnectDelay           = flag.Duration("reconnect-delay", config.ParseDurationDefault(config.EnvWithDefault("RECONNECT_DELAY", ""), config.DefaultReconnectDelay), "Base reconnect delay (env: RECONNECT_DELAY)")
		maxReconnectDelay        = flag.Duration("max-reconnect-delay", config.ParseDurationDefault(config.EnvWithDefault("MAX_RECONNECT_DELAY", ""), config.DefaultMaxReconnect), "Max reconnect delay (env: MAX_RECONNECT_DELAY)")
		discoveryInterval        = flag.Duration("discovery-interval", config.ParseDurationDefault(config.EnvWithDefault("DISCOVERY_INTERVAL", ""), config.DefaultDiscoveryInterval), "Service re-discovery interval (env: DISCOVERY_INTERVAL)")
		tlsInsecure              = flag.Bool("tls-insecure", config.IsTruthy(os.Getenv("TLS_INSECURE")), "Skip TLS verification, requires --dev (env: TLS_INSECURE)")
		devMode                  = flag.Bool("dev", false, "Development mode: allow ws:// and --tls-insecure (flag only)")
		cleanupOnly              = flag.Bool("cleanup-only", config.IsTruthy(os.Getenv("CLEANUP_ONLY")), "Delete users recorded in the state file and exit, no network (env: CLEANUP_ONLY)")
		diagnosticsDir           = flag.String("diagnostics-dir", config.EnvWithDefault("DIAGNOSTICS_DIR", "/usr/share/my/diagnostics.d"), "Directory with diagnostic plugin scripts (env: DIAGNOSTICS_DIR)")
		diagnosticsPluginTimeout = flag.Duration("diagnostics-plugin-timeout", config.ParseDurationDefault(config.EnvWithDefault("DIAGNOSTICS_PLUGIN_TIMEOUT", ""), config.DefaultDiagnosticsPluginTimeout), "Timeout per diagnostic plugin (env: DIAGNOSTICS_PLUGIN_TIMEOUT)")
		diagnosticsTotalTimeout  = flag.Duration("diagnostics-total-timeout", config.ParseDurationDefault(config.EnvWithDefault("DIAGNOSTICS_TOTAL_TIMEOUT", ""), config.DefaultDiagnosticsTotalTimeout), "Max time to wait for all diagnostics (env: DIAGNOSTICS_TOTAL_TIMEOUT)")

		usersDir           = flag.String("users-dir", config.EnvWithDefault("USERS_DIR", config.DefaultUsersDir), "Directory with user configuration plugin scripts (env: USERS_DIR)")
		usersPluginTimeout = flag.Duration("users-plugin-timeout", config.ParseDurationDefault(config.EnvWithDefault("USERS_PLUGIN_TIMEOUT", ""), config.DefaultUsersPluginTimeout), "Timeout per user plugin (env: USERS_PLUGIN_TIMEOUT)")
		usersTotalTimeout  = flag.Duration("users-total-timeout", config.ParseDurationDefault(config.EnvWithDefault("USERS_TOTAL_TIMEOUT", ""), config.DefaultUsersTotalTimeout), "Max time to wait for user provisioning (env: USERS_TOTAL_TIMEOUT)")
		usersStateFile     = flag.String("users-state-file", config.EnvWithDefault("USERS_STATE_FILE", config.DefaultUsersStateFile), "State file for orphan user cleanup (env: USERS_STATE_FILE)")
	)
	flag.Parse()

	// The secret is read from the environment only, never from argv
	secret := os.Getenv("SYSTEM_SECRET")
	_ = os.Unsetenv("SYSTEM_SECRET")

	if *cleanupOnly {
		os.Exit(runCleanupOnly(*usersStateFile, *usersDir, *redisAddr, *usersPluginTimeout))
	}

	if *urlFlag == "" || *keyFlag == "" || secret == "" {
		fmt.Fprintln(os.Stderr, "Usage: SYSTEM_SECRET=... tunnel-client --url URL --key KEY [options]")
		fmt.Fprintln(os.Stderr, "  Required: --url, --key (or SUPPORT_URL, SYSTEM_KEY) and the SYSTEM_SECRET env var")
		os.Exit(1)
	}

	if err := config.ValidateTransport(*urlFlag, *tlsInsecure, *devMode); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if *devMode {
		log.Println("WARNING: DEVELOPMENT MODE: plaintext ws:// and unverified TLS are allowed. Never use --dev in production.")
		if *tlsInsecure {
			log.Println("WARNING: TLS CERTIFICATE VERIFICATION IS DISABLED")
		}
	}

	// Auto-detect Redis on localhost if not explicitly specified
	if *redisAddr == "" {
		rdb := redis.NewClient(&redis.Options{Addr: config.DefaultRedisAddr})
		ctx, cancel := context.WithTimeout(context.Background(), config.RedisPingTimeout)
		if err := rdb.Ping(ctx).Err(); err == nil {
			log.Printf("Redis detected at %s, enabling NS8 auto-discovery", config.DefaultRedisAddr)
			*redisAddr = config.DefaultRedisAddr
		} else {
			log.Printf("No Redis at %s, skipping NS8 auto-discovery (use -redis-addr to specify)", config.DefaultRedisAddr)
		}
		cancel()
		_ = rdb.Close()
	}

	// Auto-detect node ID from NS8 environment if not explicitly specified
	if *nodeIDFlag == "" && *redisAddr != "" {
		if nid := discovery.ReadNodeID(); nid != "" {
			log.Printf("Auto-detected node ID: %s", nid)
			*nodeIDFlag = nid
		}
	}

	// Build exclusion list (flag value already includes env fallback)
	exclude := config.ParseExcludePatterns(*excludePatterns)

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

	cfg := &config.ClientConfig{
		URL:                      *urlFlag,
		Key:                      *keyFlag,
		Secret:                   secret,
		NodeID:                   *nodeIDFlag,
		RedisAddr:                *redisAddr,
		StaticServices:           *staticServices,
		Exclude:                  exclude,
		ReconnectDelay:           *reconnectDelay,
		MaxReconnectDelay:        *maxReconnectDelay,
		DiscoveryInterval:        *discoveryInterval,
		TLSInsecure:              *tlsInsecure,
		Dev:                      *devMode,
		DiagnosticsDir:           *diagnosticsDir,
		DiagnosticsPluginTimeout: *diagnosticsPluginTimeout,
		DiagnosticsTotalTimeout:  *diagnosticsTotalTimeout,
		UsersDir:                 *usersDir,
		UsersPluginTimeout:       *usersPluginTimeout,
		UsersTotalTimeout:        *usersTotalTimeout,
		UsersStateFile:           *usersStateFile,
	}

	// Clean up orphaned support users from a previous run. Identities whose
	// deletion fails stay recorded and are deleted again with this session's.
	if err := cleanupState(ctx, cfg.UsersStateFile, cfg.UsersDir, cfg.RedisAddr, cfg.UsersPluginTimeout); err != nil {
		log.Printf("Warning: orphaned support users cleanup incomplete: %v", err)
	}
	leftover, loadErr := users.LoadState(cfg.UsersStateFile)
	if loadErr != nil {
		log.Printf("Warning: cannot read users state file: %v", loadErr)
	}

	code := connection.RunWithReconnect(ctx, cfg, leftover)
	cancel()
	os.Exit(code)
}

// cleanupState runs plugin teardown and deletes the identities recorded in
// stateFile, using the provisioner of the platform that created them.
func cleanupState(ctx context.Context, stateFile, usersDir, redisAddr string, pluginTimeout time.Duration) error {
	newProvisioner := func(platform string) users.Provisioner {
		return users.ProvisionerForPlatform(platform, redisAddr)
	}
	teardown := func(state *users.SessionUsers) {
		log.Printf("Found orphaned support users from session %s, cleaning up...", state.SessionID)
		addr := redisAddr
		if state.Platform == "ns8" && addr == "" {
			addr = config.DefaultRedisAddr
		}
		users.RunTeardown(ctx, usersDir, state, nil, addr, pluginTimeout)
	}
	return users.CleanupState(stateFile, newProvisioner, teardown)
}

// runCleanupOnly deletes the identities recorded in the state file without
// touching the network. Returns the process exit code.
func runCleanupOnly(stateFile, usersDir, redisAddr string, pluginTimeout time.Duration) int {
	if err := cleanupState(context.Background(), stateFile, usersDir, redisAddr, pluginTimeout); err != nil {
		log.Printf("Cleanup failed: %v", err)
		return 1
	}
	log.Println("Cleanup complete")
	return 0
}
