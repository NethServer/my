/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package users

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"syscall"
	"time"

	"github.com/nethesis/my/services/support/cmd/tunnel-client/internal/models"
)

const defaultPluginTimeout = 15 * time.Second

// moduleBaseNameRegex extracts the base name from a module ID by stripping the trailing instance number.
// e.g., "nethvoice103" → "nethvoice", "n8n2" → "n8n", "nethsecurity-controller4" → "nethsecurity-controller"
var moduleBaseNameRegex = regexp.MustCompile(`^(.+?)\d+$`)

// ExtractModuleBaseName returns the base name of a module ID (without trailing digits).
func ExtractModuleBaseName(moduleID string) string {
	m := moduleBaseNameRegex.FindStringSubmatch(moduleID)
	if len(m) > 1 {
		return m[1]
	}
	return moduleID
}

// RunSetup discovers and executes plugin scripts in usersDir with the "setup" action.
// For plugins that match a discovered module (by base name), the module's instances
// are passed via --instances-file. Other plugins run without instance context.
func RunSetup(ctx context.Context, usersDir string, users *SessionUsers, services map[string]models.ServiceInfo, redisAddr string, pluginTimeout time.Duration) ([]AppConfig, []PluginError) {
	if pluginTimeout <= 0 {
		pluginTimeout = defaultPluginTimeout
	}

	plugins := discoverPlugins(usersDir)
	if len(plugins) == 0 {
		return nil, nil
	}

	// Write users data to a temp file for plugins to read
	usersFile, err := writeUsersFile(users)
	if err != nil {
		log.Printf("Users configurator: cannot write temp users file: %v", err)
		return nil, nil
	}
	defer func() { _ = os.Remove(usersFile) }()

	// Build module contexts from discovered services
	moduleContexts := buildModuleContexts(services, redisAddr)

	// Populate module_domains mapping on the session users so the frontend
	// can display per-module domain credentials inside each accordion.
	if users.ModuleDomains == nil {
		users.ModuleDomains = make(map[string]string)
	}
	for _, mc := range moduleContexts {
		for _, inst := range mc.Instances {
			if inst.Domain != "" {
				users.ModuleDomains[inst.ID] = inst.Domain
			}
		}
	}

	var apps []AppConfig
	var errors []PluginError

	for _, pluginPath := range plugins {
		pluginName := filepath.Base(pluginPath)

		// Check if this plugin matches a discovered module
		var instancesFile string
		if mc, ok := moduleContexts[pluginName]; ok && len(mc.Instances) > 0 {
			var writeErr error
			instancesFile, writeErr = writeTempJSON("my-instances-*.json", mc)
			if writeErr != nil {
				log.Printf("Users configurator: cannot write instances file for %q: %v", pluginName, writeErr)
			} else {
				defer func(f string) { _ = os.Remove(f) }(instancesFile)
			}
		}

		results, err := runPlugin(ctx, pluginPath, "setup", usersFile, instancesFile, pluginTimeout)
		if err != nil {
			log.Printf("Users configurator: plugin %q setup failed: %v", pluginName, err)
			errors = append(errors, PluginError{
				ID:      pluginName,
				Message: err.Error(),
			})
			continue
		}
		for _, app := range results {
			apps = append(apps, app)
			log.Printf("Users configurator: plugin %q configured app %q", pluginName, app.Name)
		}
	}

	return apps, errors
}

// RunTeardown executes plugin scripts with the "teardown" action to undo app configurations.
func RunTeardown(ctx context.Context, usersDir string, users *SessionUsers, services map[string]models.ServiceInfo, redisAddr string, pluginTimeout time.Duration) {
	if pluginTimeout <= 0 {
		pluginTimeout = defaultPluginTimeout
	}

	plugins := discoverPlugins(usersDir)
	if len(plugins) == 0 {
		return
	}

	usersFile, err := writeUsersFile(users)
	if err != nil {
		log.Printf("Users configurator: cannot write temp users file for teardown: %v", err)
		return
	}
	defer func() { _ = os.Remove(usersFile) }()

	moduleContexts := buildModuleContexts(services, redisAddr)

	// Run teardown in reverse order
	for i := len(plugins) - 1; i >= 0; i-- {
		pluginName := filepath.Base(plugins[i])

		var instancesFile string
		if mc, ok := moduleContexts[pluginName]; ok && len(mc.Instances) > 0 {
			var writeErr error
			instancesFile, writeErr = writeTempJSON("my-instances-*.json", mc)
			if writeErr == nil {
				defer func(f string) { _ = os.Remove(f) }(instancesFile)
			}
		}

		if _, err := runPlugin(ctx, plugins[i], "teardown", usersFile, instancesFile, pluginTimeout); err != nil {
			log.Printf("Users configurator: plugin %q teardown failed: %v", pluginName, err)
		} else {
			log.Printf("Users configurator: plugin %q teardown complete", pluginName)
		}
	}
}

// buildModuleContexts groups discovered services by module base name.
func buildModuleContexts(services map[string]models.ServiceInfo, redisAddr string) map[string]*ModuleContext {
	if len(services) == 0 {
		return nil
	}

	// Group services by moduleID
	type moduleInfo struct {
		nodeID   string
		label    string
		services map[string]ModuleServiceInfo
	}
	moduleMap := make(map[string]*moduleInfo)

	for serviceName, svc := range services {
		if svc.ModuleID == "" {
			continue
		}
		mi, ok := moduleMap[svc.ModuleID]
		if !ok {
			mi = &moduleInfo{
				nodeID:   svc.NodeID,
				label:    svc.Label,
				services: make(map[string]ModuleServiceInfo),
			}
			moduleMap[svc.ModuleID] = mi
		}
		if mi.label == "" && svc.Label != "" {
			mi.label = svc.Label
		}
		mi.services[serviceName] = ModuleServiceInfo{
			Host:       svc.Host,
			Path:       svc.Path,
			PathPrefix: svc.PathPrefix,
			TLS:        svc.TLS,
		}
	}

	// Fetch USER_DOMAIN for each module instance (NS8 only)
	domains := make(map[string]string)
	if redisAddr != "" {
		for moduleID := range moduleMap {
			domain := fetchModuleDomain(moduleID)
			if domain != "" {
				domains[moduleID] = domain
			}
		}
	}

	// Group module instances by base name
	contexts := make(map[string]*ModuleContext)
	for moduleID, mi := range moduleMap {
		baseName := ExtractModuleBaseName(moduleID)
		mc, ok := contexts[baseName]
		if !ok {
			mc = &ModuleContext{Module: baseName}
			contexts[baseName] = mc
		}
		mc.Instances = append(mc.Instances, ModuleInstance{
			ID:       moduleID,
			NodeID:   mi.nodeID,
			Label:    mi.label,
			Domain:   domains[moduleID],
			Services: mi.services,
		})
	}

	// Sort instances within each context
	for _, mc := range contexts {
		sort.Slice(mc.Instances, func(i, j int) bool {
			return mc.Instances[i].ID < mc.Instances[j].ID
		})
	}

	return contexts
}

// fetchModuleDomain reads USER_DOMAIN from Redis for a module instance.
func fetchModuleDomain(moduleID string) string {
	cmd := exec.Command("redis-cli", "HGET", fmt.Sprintf("module/%s/environment", moduleID), "USER_DOMAIN") //nolint:gosec // redis-cli is trusted
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	domain := string(bytes.TrimSpace(output))
	if domain == "" || domain == "(nil)" {
		return ""
	}
	return domain
}

// discoverPlugins scans usersDir for valid executable plugin files.
// Uses the same security checks as diagnostics: ownership, permissions.
func discoverPlugins(usersDir string) []string {
	if usersDir == "" {
		return nil
	}

	entries, err := os.ReadDir(usersDir)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Users configurator: failed to read directory %q: %v", usersDir, err)
		}
		return nil
	}

	currentUID := os.Getuid()
	var paths []string

	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			continue
		}
		// Must be executable
		if info.Mode()&0o111 == 0 {
			continue
		}

		pluginPath := filepath.Join(usersDir, entry.Name())

		// Ownership check: only root or current process user
		if sysInfo, ok := info.Sys().(*syscall.Stat_t); ok {
			ownerUID := int(sysInfo.Uid)
			if ownerUID != 0 && ownerUID != currentUID {
				log.Printf("Users configurator: skipping %q: owned by UID %d (must be root or UID %d)", pluginPath, ownerUID, currentUID)
				continue
			}
		}

		// Reject group-writable or world-writable
		if info.Mode().Perm()&0o022 != 0 {
			log.Printf("Users configurator: skipping %q: group- or world-writable (mode=%04o)", pluginPath, info.Mode().Perm())
			continue
		}

		paths = append(paths, pluginPath)
	}

	sort.Strings(paths)
	return paths
}

// runPlugin executes a single plugin with the given action and context files.
// For "setup", parses stdout as a single AppConfig or an array of AppConfig.
// For "teardown", ignores stdout. instancesFile may be empty for generic plugins.
func runPlugin(ctx context.Context, path, action, usersFile, instancesFile string, timeout time.Duration) ([]AppConfig, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	args := []string{action, "--users-file", usersFile}
	if instancesFile != "" {
		args = append(args, "--instances-file", instancesFile)
	}

	cmd := exec.CommandContext(ctx, path, args...) //nolint:gosec // path comes from a configured directory
	// Minimal environment to prevent credential leakage
	cmd.Env = []string{
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start: %w", err)
	}

	var stdout bytes.Buffer
	if _, readErr := io.Copy(&stdout, io.LimitReader(stdoutPipe, 64*1024)); readErr != nil {
		log.Printf("Users configurator: partial read from %q: %v", path, readErr)
	}

	runErr := cmd.Wait()
	if ctx.Err() != nil {
		return nil, fmt.Errorf("timed out after %v", timeout)
	}
	if runErr != nil {
		return nil, fmt.Errorf("exit error: %w", runErr)
	}

	if action != "setup" || stdout.Len() == 0 {
		return nil, nil
	}

	raw := bytes.TrimSpace(stdout.Bytes())
	pluginName := filepath.Base(path)

	// Try parsing as array first, then as single object
	if len(raw) > 0 && raw[0] == '[' {
		var apps []AppConfig
		if err := json.Unmarshal(raw, &apps); err != nil {
			return nil, fmt.Errorf("invalid JSON array output: %w", err)
		}
		for i := range apps {
			if apps[i].ID == "" {
				apps[i].ID = pluginName
			}
			if apps[i].Name == "" {
				apps[i].Name = apps[i].ID
			}
		}
		return apps, nil
	}

	var app AppConfig
	if err := json.Unmarshal(raw, &app); err != nil {
		return nil, fmt.Errorf("invalid JSON output: %w", err)
	}
	if app.ID == "" {
		app.ID = pluginName
	}
	if app.Name == "" {
		app.Name = app.ID
	}
	return []AppConfig{app}, nil
}

// writeUsersFile writes SessionUsers data to a temporary file and returns its path.
func writeUsersFile(users *SessionUsers) (string, error) {
	return writeTempJSON("my-support-users-*.json", users)
}

// writeTempJSON writes data as JSON to a temporary file and returns its path.
func writeTempJSON(pattern string, data interface{}) (string, error) {
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", err
	}

	if err := json.NewEncoder(f).Encode(data); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return "", err
	}

	_ = f.Close()
	return f.Name(), nil
}
