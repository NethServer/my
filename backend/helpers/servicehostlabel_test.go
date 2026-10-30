/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package helpers

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServiceHostLabel(t *testing.T) {
	// Same vectors as services/support/tunnel/hostlabel_test.go
	assert.Equal(t, "cluster-admin-6e5263a7", ServiceHostLabel("cluster-admin"))
	assert.Equal(t, "nethsecurity-control-87b27f47", ServiceHostLabel("nethsecurity-controller4-prometheus"))
	assert.Equal(t, "svc-e3b0c442", ServiceHostLabel(""))

	label := regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*-[0-9a-f]{8}$`)
	for _, name := range []string{"cluster-admin", "nethvoice103-cti", "Web.Top_5", "a.b.c", "--x--", "nethsecurity-controller4-prometheus-very-long-route-name", "_"} {
		l := ServiceHostLabel(name)
		assert.LessOrEqual(t, len(l), 29, name)
		assert.Regexp(t, label, l, name)
		assert.LessOrEqual(t, len(l+"--"+"0123456789abcdef0123456789abcdef"), 63, name)
	}
	assert.NotEqual(t, ServiceHostLabel("webtop5"), ServiceHostLabel("webtop6"))
}
