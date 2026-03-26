/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package diagnostics

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const defaultPluginTimeout = 10 * time.Second

// statusRank maps a DiagnosticStatus to a numeric severity (higher = worse).
// critical > warning > ok > error > timeout
func statusRank(s DiagnosticStatus) int {
	switch s {
	case StatusCritical:
		return 5
	case StatusWarning:
		return 4
	case StatusOK:
		return 3
	case StatusError:
		return 2
	case StatusTimeout:
		return 1
	default:
		return 0
	}
}

// worstStatus returns the more severe of two statuses.
func worstStatus(a, b DiagnosticStatus) DiagnosticStatus {
	if statusRank(a) >= statusRank(b) {
		return a
	}
	return b
}

// runBuiltinSystem collects basic OS and resource metrics from /proc and syscall.
func runBuiltinSystem() PluginResult {
	result := PluginResult{
		ID:     "system",
		Name:   "System",
		Status: StatusOK,
	}

	overallStatus := StatusOK
	var checks []DiagnosticCheck

	// --- OS info ---
	osName := "unknown"
	osVersion := "unknown"
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		scanner := bufio.NewScanner(bytes.NewReader(data))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "NAME=") {
				osName = strings.Trim(strings.TrimPrefix(line, "NAME="), `"`)
			}
			if strings.HasPrefix(line, "VERSION_ID=") {
				osVersion = strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), `"`)
			}
		}
	}
	checks = append(checks, DiagnosticCheck{
		Name:   "os",
		Status: StatusOK,
		Value:  fmt.Sprintf("%s %s", osName, osVersion),
	})

	// --- CPU load averages ---
	var load1m, load5m, load15m float64
	if data, err := os.ReadFile("/proc/loadavg"); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) >= 3 {
			load1m, _ = strconv.ParseFloat(fields[0], 64)
			load5m, _ = strconv.ParseFloat(fields[1], 64)
			load15m, _ = strconv.ParseFloat(fields[2], 64)
		}
	}
	cpuCount := runtime.NumCPU()
	loadStatus := StatusOK
	if load1m > float64(cpuCount)*2 {
		loadStatus = StatusCritical
	} else if load1m > float64(cpuCount) {
		loadStatus = StatusWarning
	}
	overallStatus = worstStatus(overallStatus, loadStatus)
	loadDetails, _ := json.Marshal(map[string]interface{}{
		"load_1m": load1m, "load_5m": load5m, "load_15m": load15m, "cpu_count": cpuCount,
	})
	checks = append(checks, DiagnosticCheck{
		Name:    "load_average",
		Status:  loadStatus,
		Value:   fmt.Sprintf("%.2f %.2f %.2f", load1m, load5m, load15m),
		Details: string(loadDetails),
	})

	// --- RAM usage ---
	var memTotal, memAvailable uint64
	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		scanner := bufio.NewScanner(bytes.NewReader(data))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "MemTotal:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					memTotal, _ = strconv.ParseUint(fields[1], 10, 64)
				}
			}
			if strings.HasPrefix(line, "MemAvailable:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					memAvailable, _ = strconv.ParseUint(fields[1], 10, 64)
				}
			}
		}
	}
	ramStatus := StatusOK
	var ramUsagePct float64
	if memTotal > 0 {
		ramUsagePct = float64(memTotal-memAvailable) / float64(memTotal) * 100
		if ramUsagePct > 95 {
			ramStatus = StatusCritical
		} else if ramUsagePct > 85 {
			ramStatus = StatusWarning
		}
	}
	overallStatus = worstStatus(overallStatus, ramStatus)
	ramDetails, _ := json.Marshal(map[string]interface{}{
		"total_kb": memTotal, "available_kb": memAvailable, "used_pct": ramUsagePct,
	})
	checks = append(checks, DiagnosticCheck{
		Name:    "ram_usage",
		Status:  ramStatus,
		Value:   fmt.Sprintf("%.1f%%", ramUsagePct),
		Details: string(ramDetails),
	})

	// --- Root filesystem usage ---
	var stat syscall.Statfs_t
	diskStatus := StatusOK
	var diskUsagePct float64
	if err := syscall.Statfs("/", &stat); err == nil {
		total := stat.Blocks * uint64(stat.Bsize)
		available := stat.Bavail * uint64(stat.Bsize)
		if total > 0 {
			diskUsagePct = float64(total-available) / float64(total) * 100
			if diskUsagePct > 95 {
				diskStatus = StatusCritical
			} else if diskUsagePct > 85 {
				diskStatus = StatusWarning
			}
		}
		overallStatus = worstStatus(overallStatus, diskStatus)
		diskDetails, _ := json.Marshal(map[string]interface{}{
			"total_bytes": total, "available_bytes": available, "used_pct": diskUsagePct,
		})
		checks = append(checks, DiagnosticCheck{
			Name:    "disk_usage",
			Status:  diskStatus,
			Value:   fmt.Sprintf("%.1f%%", diskUsagePct),
			Details: string(diskDetails),
		})
	} else {
		checks = append(checks, DiagnosticCheck{
			Name:    "disk_usage",
			Status:  StatusError,
			Value:   "unavailable",
			Details: err.Error(),
		})
	}

	// --- Uptime ---
	if data, err := os.ReadFile("/proc/uptime"); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) >= 1 {
			if secs, parseErr := strconv.ParseFloat(fields[0], 64); parseErr == nil {
				totalSecs := int64(secs)
				days := totalSecs / 86400
				hours := (totalSecs % 86400) / 3600
				minutes := (totalSecs % 3600) / 60
				uptimeDetails, _ := json.Marshal(map[string]interface{}{
					"days": days, "hours": hours, "minutes": minutes, "total_seconds": totalSecs,
				})
				checks = append(checks, DiagnosticCheck{
					Name:    "uptime",
					Status:  StatusOK,
					Value:   fmt.Sprintf("%d days %d hours %d minutes", days, hours, minutes),
					Details: string(uptimeDetails),
				})
			}
		}
	} else {
		checks = append(checks, DiagnosticCheck{
			Name:    "uptime",
			Status:  StatusError,
			Value:   "unavailable",
			Details: err.Error(),
		})
	}

	result.Status = overallStatus
	result.Checks = checks
	return result
}

// runPlugin executes an external diagnostic plugin and parses its output.
func runPlugin(path string, timeout time.Duration) PluginResult {
	// Derive id and name from the filename (strip extension)
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	pluginID := strings.TrimSuffix(base, ext)
	pluginName := pluginID

	result := PluginResult{
		ID:     pluginID,
		Name:   pluginName,
		Status: StatusError,
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, path) //nolint:gosec // path comes from a configured directory, not user input

	// Fix #6: run plugins with a minimal environment to prevent credential leakage.
	// The inherited environment may contain SYSTEM_KEY, SYSTEM_SECRET, SUPPORT_URL,
	// and other sensitive values that plugins should not access.
	cmd.Env = []string{
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		result.Summary = fmt.Sprintf("failed to create stdout pipe: %v", err)
		return result
	}

	if startErr := cmd.Start(); startErr != nil {
		result.Summary = fmt.Sprintf("failed to start plugin: %v", startErr)
		return result
	}

	// Read at most 512 KB from stdout
	var stdout bytes.Buffer
	if _, readErr := io.Copy(&stdout, io.LimitReader(stdoutPipe, 512*1024)); readErr != nil {
		// Partial reads are acceptable; continue to Wait
		log.Printf("Partial read from plugin %q: %v", path, readErr)
	}

	runErr := cmd.Wait()

	// Determine exit-code-derived status
	var exitStatus DiagnosticStatus
	if ctx.Err() != nil {
		exitStatus = StatusTimeout
	} else if runErr == nil {
		exitStatus = StatusOK
	} else {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			switch exitErr.ExitCode() {
			case 1:
				exitStatus = StatusWarning
			case 2:
				exitStatus = StatusCritical
			default:
				exitStatus = StatusError
			}
		} else {
			result.Summary = runErr.Error()
			return result
		}
	}

	// Try to parse stdout as JSON into a PluginResult
	var parsed PluginResult
	if jsonErr := json.Unmarshal(stdout.Bytes(), &parsed); jsonErr == nil {
		// Fill missing id/name from filename
		if parsed.ID == "" {
			parsed.ID = pluginID
		}
		if parsed.Name == "" {
			parsed.Name = pluginName
		}
		// Use exit-code-derived status if it's worse than what the plugin reported
		if parsed.Status == "" {
			parsed.Status = exitStatus
		} else {
			parsed.Status = worstStatus(parsed.Status, exitStatus)
		}
		return parsed
	}

	// JSON parse failed — use raw stdout as summary
	result.Status = exitStatus
	summary := strings.TrimSpace(stdout.String())
	if len(summary) > 256 {
		summary = summary[:256]
	}
	result.Summary = summary
	return result
}

// Collect runs the built-in system plugin and any executable plugins found in
// pluginsDir, aggregates the results, and returns a DiagnosticsReport.
// If pluginsDir is empty or does not exist, only the built-in plugin runs.
// This function never panics.
func Collect(pluginsDir string, pluginTimeout time.Duration) DiagnosticsReport {
	start := time.Now()

	if pluginTimeout <= 0 {
		pluginTimeout = defaultPluginTimeout
	}

	var plugins []PluginResult

	// Always run the built-in system plugin
	func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panic in built-in system plugin: %v", r)
				plugins = append(plugins, PluginResult{
					ID:      "system",
					Name:    "System",
					Status:  StatusError,
					Summary: fmt.Sprintf("panic: %v", r),
				})
			}
		}()
		plugins = append(plugins, runBuiltinSystem())
	}()

	// Scan pluginsDir for executable files if specified
	if pluginsDir != "" {
		entries, err := os.ReadDir(pluginsDir)
		if err != nil {
			if !os.IsNotExist(err) {
				log.Printf("Failed to read diagnostics directory %q: %v", pluginsDir, err)
			}
			// Silently skip if directory does not exist
		} else {
			// Collect and sort plugin paths
			currentUID := os.Getuid()
			var pluginPaths []string
			for _, entry := range entries {
				if !entry.Type().IsRegular() {
					continue
				}
				info, infoErr := entry.Info()
				if infoErr != nil {
					continue
				}
				// Check executable bit
				if info.Mode()&0o111 == 0 {
					continue
				}
				pluginPath := filepath.Join(pluginsDir, entry.Name())
				// Fix #6: only run plugins owned by root (UID 0) or the current process user.
				// Prevents privilege escalation if a less-privileged process can write to
				// the plugins directory.
				if sysInfo, ok := info.Sys().(*syscall.Stat_t); ok {
					ownerUID := int(sysInfo.Uid)
					if ownerUID != 0 && ownerUID != currentUID {
						log.Printf("Skipping plugin %q: owned by UID %d (must be root or UID %d)", pluginPath, ownerUID, currentUID)
						continue
					}
				}
				// Fix #6: reject group-writable or world-writable plugins to prevent tampering.
				if info.Mode().Perm()&0o022 != 0 {
					log.Printf("Skipping plugin %q: file is group- or world-writable (mode=%04o)", pluginPath, info.Mode().Perm())
					continue
				}
				pluginPaths = append(pluginPaths, pluginPath)
			}
			sort.Strings(pluginPaths)

			for _, path := range pluginPaths {
				p := path // capture for closure
				func() {
					defer func() {
						if r := recover(); r != nil {
							log.Printf("Panic in plugin %q: %v", p, r)
							plugins = append(plugins, PluginResult{
								ID:      filepath.Base(p),
								Name:    filepath.Base(p),
								Status:  StatusError,
								Summary: fmt.Sprintf("panic: %v", r),
							})
						}
					}()
					plugins = append(plugins, runPlugin(p, pluginTimeout))
				}()
			}
		}
	}

	// Compute overall status as the worst of all plugin statuses
	overall := StatusOK
	for _, p := range plugins {
		overall = worstStatus(overall, p.Status)
	}

	return DiagnosticsReport{
		CollectedAt:   start,
		DurationMs:    time.Since(start).Milliseconds(),
		OverallStatus: overall,
		Plugins:       plugins,
	}
}
