/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInjectLabels(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		toInject map[string]string
		expected map[string]string
	}{
		{
			name:     "adds all labels when missing",
			body:     `[{"labels":{"alertname":"DiskFull","severity":"critical"}}]`,
			toInject: map[string]string{"system_key": "SYS-001", "system_id": "uuid-001"},
			expected: map[string]string{
				"alertname":  "DiskFull",
				"severity":   "critical",
				"system_key": "SYS-001",
				"system_id":  "uuid-001",
			},
		},
		{
			name:     "overrides existing system_key",
			body:     `[{"labels":{"alertname":"DiskFull","system_key":"EXISTING"}}]`,
			toInject: map[string]string{"system_key": "SYS-001", "system_id": "uuid-001"},
			expected: map[string]string{
				"alertname":  "DiskFull",
				"system_key": "SYS-001",
				"system_id":  "uuid-001",
			},
		},
		{
			name:     "handles missing labels object",
			body:     `[{"annotations":{"summary":"test"}}]`,
			toInject: map[string]string{"system_key": "SYS-KEY-003", "system_id": "uuid-003"},
			expected: map[string]string{
				"system_key": "SYS-KEY-003",
				"system_id":  "uuid-003",
			},
		},
		{
			name: "injects all context labels",
			body: `[{"labels":{"alertname":"Test","severity":"warning"}}]`,
			toInject: map[string]string{
				"system_key":        "SYS-001",
				"system_id":         "uuid-001",
				"system_name":       "web-01",
				"organization_name": "Acme Corp",
				"organization_vat":  "IT00000000001",
			},
			expected: map[string]string{
				"alertname":         "Test",
				"severity":          "warning",
				"system_key":        "SYS-001",
				"system_id":         "uuid-001",
				"system_name":       "web-01",
				"organization_name": "Acme Corp",
				"organization_vat":  "IT00000000001",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := injectLabels([]byte(tt.body), tt.toInject)

			var alerts []map[string]interface{}
			err := json.Unmarshal(result, &alerts)
			assert.NoError(t, err)
			assert.NotEmpty(t, alerts)

			labels, ok := alerts[0]["labels"].(map[string]interface{})
			assert.True(t, ok)

			assert.Equal(t, len(tt.expected), len(labels), "unexpected label count")
			for key, wantVal := range tt.expected {
				assert.Equal(t, wantVal, labels[key], "label %s mismatch", key)
			}
		})
	}
}

func TestInjectLabels_InvalidJSON(t *testing.T) {
	body := []byte("not json")
	result := injectLabels(body, map[string]string{"system_key": "SYS-001"})
	assert.Equal(t, body, result, "invalid JSON must be returned unchanged")
}

func TestInjectLabels_EmptyInjection(t *testing.T) {
	body := []byte(`[{"labels":{"alertname":"Test"}}]`)
	result := injectLabels(body, map[string]string{})
	assert.Equal(t, body, result, "empty injection must return body unchanged")
}
