/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultUIURL(t *testing.T) {
	assert.Equal(t, "https://ns.example.com/cluster-admin", DefaultUIURL(strPtr("ns8"), "ns.example.com"))
	assert.Equal(t, "https://fw.example.com:9090", DefaultUIURL(strPtr("nsec"), "fw.example.com"))
	assert.Equal(t, "", DefaultUIURL(nil, "fw.example.com"))
	assert.Equal(t, "", DefaultUIURL(strPtr("nsec"), ""))
	assert.Equal(t, "", DefaultUIURL(strPtr("other"), "x.example.com"))
}

func TestUIURLCandidates(t *testing.T) {
	assert.Equal(t, []string{"https://ns.example.com/cluster-admin"}, UIURLCandidates(nil, strPtr("ns8"), "ns.example.com"))
	assert.Equal(t, []string{"https://fw.example.com:9090", "https://fw.example.com:443"}, UIURLCandidates(nil, strPtr("nsec"), "fw.example.com"))
	assert.Equal(t, []string{"https://custom.example.org:8443"}, UIURLCandidates(strPtr("https://custom.example.org:8443"), strPtr("nsec"), "fw.example.com"))
	assert.Equal(t, []string{"https://custom.example.org"}, UIURLCandidates(strPtr("https://custom.example.org"), nil, ""))
	assert.Nil(t, UIURLCandidates(nil, nil, ""))
}

func TestNormalizeUIURL(t *testing.T) {
	ns8 := strPtr("ns8")
	tests := []struct {
		name    string
		raw     string
		want    *string
		wantErr bool
	}{
		{"empty goes back to the default", "  ", nil, false},
		{"the default is not stored", "https://ns.example.com/cluster-admin", nil, false},
		{"a custom https address", " https://ns.example.org:8443/cluster-admin ", strPtr("https://ns.example.org:8443/cluster-admin"), false},
		{"http is accepted", "http://192.0.2.10:9090", strPtr("http://192.0.2.10:9090"), false},
		{"no scheme", "ns.example.org/cluster-admin", nil, true},
		{"another scheme", "javascript:alert(1)", nil, true},
		{"no host", "https:///cluster-admin", nil, true},
		{"credentials", "https://user:pass@ns.example.org", nil, true},
		{"too long", "https://ns.example.org/" + strings.Repeat("a", MaxUIURLLength), nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeUIURL(tt.raw, ns8, "ns.example.com")
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrInvalidUIURL)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
