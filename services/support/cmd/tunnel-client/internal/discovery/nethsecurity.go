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
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"github.com/nethesis/my/services/support/cmd/tunnel-client/internal/models"
)

const (
	// NethSecurity detection paths
	nethSecUIPath        = "/www-ns/index.html"
	nethSecNginxConf     = "/etc/nginx/conf.d/ns-ui.conf"
	defaultNethSecUIPort = "443"
)

// DiscoverNethSecurityServices detects NethSecurity (OpenWrt-based firewall)
// by checking for its web UI files and registers the main HTTPS service.
// NethSecurity runs nginx with the UI on a configurable port:
//   - Port from /etc/nginx/conf.d/ns-ui.conf (dedicated UI server block)
//   - Port 443 (when 00ns.locations is active, UI is on the default server)
func DiscoverNethSecurityServices() map[string]models.ServiceInfo {
	services := make(map[string]models.ServiceInfo)

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

	services["nethsecurity-ui"] = models.ServiceInfo{
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
