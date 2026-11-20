/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package connection

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/nethesis/my/services/support/cmd/tunnel-client/internal/models"
)

// dangerousHostnames mirrors the server-side list in tunnel/manager.go.
var dangerousHostnames = map[string]bool{
	"metadata.google.internal":     true,
	"metadata":                     true,
	"metadata.azure.internal":      true,
	"instance-data":                true,
	"metadata.platformequinix.com": true,
}

const resolveTimeout = 5 * time.Second

// lookupIP and localAddrs are variables so tests can stub them.
var lookupIP = func(ctx context.Context, host string) ([]net.IP, error) {
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	ips := make([]net.IP, 0, len(addrs))
	for _, a := range addrs {
		ips = append(ips, a.IP)
	}
	return ips, nil
}

var localAddrs = func() []net.IP {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil
	}
	var ips []net.IP
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok {
			ips = append(ips, ipnet.IP)
		}
	}
	return ips
}

// resolveInjectedTarget validates a server-injected host:port target and
// returns the ip:port to dial. The name is resolved once here and the result
// pinned, so a later DNS answer cannot redirect the dial (DNS rebinding).
// Addresses of this host (loopback or any local interface) are only allowed
// when a discovered service already exposes the same ip:port.
func resolveInjectedTarget(target string, discovered map[string]models.ServiceInfo) (string, error) {
	host, port, err := net.SplitHostPort(target)
	if err != nil || host == "" {
		return "", fmt.Errorf("target must be host:port")
	}
	if p, err := strconv.Atoi(port); err != nil || p < 1 || p > 65535 {
		return "", fmt.Errorf("invalid port %q", port)
	}
	if dangerousHostnames[strings.ToLower(strings.TrimSuffix(host, "."))] {
		return "", fmt.Errorf("cloud metadata hostname blocked: %s", host)
	}

	var ips []net.IP
	if ip := net.ParseIP(host); ip != nil {
		ips = []net.IP{ip}
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), resolveTimeout)
		defer cancel()
		ips, err = lookupIP(ctx, host)
		if err != nil {
			return "", fmt.Errorf("DNS resolution failed for %s: %w", host, err)
		}
		if len(ips) == 0 {
			return "", fmt.Errorf("DNS resolution returned no address for %s", host)
		}
	}

	local := localAddrs()
	for _, ip := range ips {
		if err := validateTargetIP(ip); err != nil {
			return "", fmt.Errorf("%s resolves to blocked address: %w", host, err)
		}
		if isLocalIP(ip, local) && !matchesDiscovered(ip, port, discovered) {
			return "", fmt.Errorf("%s resolves to an address of this host (%s) not exposed by discovery", host, ip)
		}
	}
	return net.JoinHostPort(ips[0].String(), port), nil
}

func validateTargetIP(ip net.IP) error {
	if ip.IsUnspecified() {
		return fmt.Errorf("unspecified address blocked: %s", ip)
	}
	if ip.To4() != nil {
		linkLocal := net.IPNet{IP: net.IPv4(169, 254, 0, 0), Mask: net.CIDRMask(16, 32)}
		if linkLocal.Contains(ip) {
			return fmt.Errorf("link-local/cloud metadata address blocked: %s", ip)
		}
		multicast := net.IPNet{IP: net.IPv4(224, 0, 0, 0), Mask: net.CIDRMask(4, 32)}
		if multicast.Contains(ip) {
			return fmt.Errorf("multicast address blocked: %s", ip)
		}
		if ip.Equal(net.IPv4bcast) {
			return fmt.Errorf("broadcast address blocked: %s", ip)
		}
	} else {
		if ip.IsLinkLocalUnicast() {
			return fmt.Errorf("IPv6 link-local address blocked: %s", ip)
		}
		if ip.IsMulticast() {
			return fmt.Errorf("IPv6 multicast address blocked: %s", ip)
		}
	}
	return nil
}

func isLocalIP(ip net.IP, local []net.IP) bool {
	if ip.IsLoopback() {
		return true
	}
	for _, l := range local {
		if l.Equal(ip) {
			return true
		}
	}
	return false
}

// matchesDiscovered reports whether a discovered service targets ip:port.
func matchesDiscovered(ip net.IP, port string, discovered map[string]models.ServiceInfo) bool {
	for _, svc := range discovered {
		host, p, err := net.SplitHostPort(svc.Target)
		if err != nil || p != port {
			continue
		}
		if strings.EqualFold(host, "localhost") && ip.IsLoopback() {
			return true
		}
		if dip := net.ParseIP(host); dip != nil && dip.Equal(ip) {
			return true
		}
	}
	return false
}
