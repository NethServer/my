/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// MaxUIURLLength matches the systems.ui_url column.
const MaxUIURLLength = 2048

// ErrInvalidUIURL is returned for a web interface address that is not an
// absolute http(s) URL with a host.
var ErrInvalidUIURL = errors.New("invalid")

// DefaultUIURL is the address the web interface of a system answers on when
// nobody set another one: it follows the FQDN of the latest inventory. Empty
// when the type or the FQDN is not known yet. The frontend derives the same
// address in getDefaultUiUrl, for the helper text of the edit drawer.
func DefaultUIURL(systemType *string, fqdn string) string {
	if systemType == nil || fqdn == "" {
		return ""
	}
	switch *systemType {
	case "ns8":
		return fmt.Sprintf("https://%s/cluster-admin", fqdn)
	case "nsec":
		return fmt.Sprintf("https://%s:9090", fqdn)
	}
	return ""
}

// UIURLCandidates lists the addresses the reachability check tries, in order.
// A custom address is the only one: the admin set it because the default does
// not open the system. Without one, NethSecurity also tries the standard HTTPS
// port, where some installations serve their web interface.
func UIURLCandidates(customURL *string, systemType *string, fqdn string) []string {
	if customURL != nil && *customURL != "" {
		return []string{*customURL}
	}
	defaultURL := DefaultUIURL(systemType, fqdn)
	if defaultURL == "" {
		return nil
	}
	if *systemType == "nsec" {
		return []string{defaultURL, fmt.Sprintf("https://%s:443", fqdn)}
	}
	return []string{defaultURL}
}

// NormalizeUIURL validates the address an admin entered for the web interface
// and returns what to store: nil for an empty value or for the default
// address, so that the system keeps following its FQDN.
func NormalizeUIURL(raw string, systemType *string, fqdn string) (*string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, nil
	}
	if len(value) > MaxUIURLLength {
		return nil, ErrInvalidUIURL
	}
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Hostname() == "" || parsed.User != nil {
		return nil, ErrInvalidUIURL
	}
	if value == DefaultUIURL(systemType, fqdn) {
		return nil, nil
	}
	return &value, nil
}
