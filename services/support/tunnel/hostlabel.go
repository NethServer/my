/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package tunnel

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// ServiceHostLabelMaxPrefix bounds the readable part of a service label so
// that {label}--{32-hex session} always fits the 63-char DNS label limit.
const ServiceHostLabelMaxPrefix = 20

// ServiceHostLabel maps a service name to the DNS-safe label used in
// {label}--{session}.support.{domain}: a readable prefix plus 8 hex chars of
// sha256(name), at most 29 chars. Must stay identical to the backend's
// helpers.ServiceHostLabel.
func ServiceHostLabel(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	prefix := b.String()
	if len(prefix) > ServiceHostLabelMaxPrefix {
		prefix = prefix[:ServiceHostLabelMaxPrefix]
	}
	prefix = strings.TrimRight(prefix, "-")
	if prefix == "" {
		prefix = "svc"
	}
	sum := sha256.Sum256([]byte(name))
	return prefix + "-" + hex.EncodeToString(sum[:4])
}
