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

func TestAppendSystemKeyFilter(t *testing.T) {
	tests := []struct {
		name      string
		rawQuery  string
		systemKey string
		expected  string
	}{
		{
			name:      "empty query",
			rawQuery:  "",
			systemKey: "SYS-001",
			expected:  `filter=system_key%3D%22SYS-001%22`,
		},
		{
			name:      "existing query params",
			rawQuery:  "active=true",
			systemKey: "SYS-001",
			expected:  `active=true&filter=system_key%3D%22SYS-001%22`,
		},
		{
			name:      "special characters in key",
			rawQuery:  "",
			systemKey: "sys:key-01.test",
			expected:  `filter=system_key%3D%22sys%3Akey-01.test%22`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := appendSystemKeyFilter(tt.rawQuery, tt.systemKey)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInjectSilenceMatcher(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		systemKey string
		wantKey   string
	}{
		{
			name:      "adds system_key when missing",
			body:      `{"matchers":[{"name":"alertname","value":"DiskFull","isRegex":false,"isEqual":true}],"startsAt":"2024-01-01T00:00:00Z","endsAt":"2024-01-01T01:00:00Z","createdBy":"system","comment":"test"}`,
			systemKey: "SYS-001",
			wantKey:   "SYS-001",
		},
		{
			name:      "overwrites existing system_key",
			body:      `{"matchers":[{"name":"system_key","value":"ATTACKER","isRegex":false,"isEqual":true},{"name":"alertname","value":"Foo","isRegex":false,"isEqual":true}],"startsAt":"2024-01-01T00:00:00Z","endsAt":"2024-01-01T01:00:00Z","createdBy":"sys","comment":"x"}`,
			systemKey: "SYS-001",
			wantKey:   "SYS-001",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := injectSilenceMatcher([]byte(tt.body), tt.systemKey)

			var silence map[string]interface{}
			err := json.Unmarshal(result, &silence)
			assert.NoError(t, err)

			matchers, ok := silence["matchers"].([]interface{})
			assert.True(t, ok)

			found := false
			count := 0
			for _, m := range matchers {
				mm := m.(map[string]interface{})
				if mm["name"] == "system_key" {
					count++
					assert.Equal(t, tt.wantKey, mm["value"])
					assert.Equal(t, false, mm["isRegex"])
					assert.Equal(t, true, mm["isEqual"])
					found = true
				}
			}
			assert.True(t, found, "system_key matcher not found")
			assert.Equal(t, 1, count, "exactly one system_key matcher expected")
		})
	}
}

func TestInjectSilenceMatcher_InvalidJSON(t *testing.T) {
	body := []byte("not json")
	result := injectSilenceMatcher(body, "SYS-001")
	assert.Equal(t, body, result, "invalid JSON must be returned unchanged")
}

func TestInjectSilenceMatcher_EmptyBody(t *testing.T) {
	result := injectSilenceMatcher([]byte{}, "SYS-001")
	assert.Empty(t, result)
}

func TestSilenceHasSystemKeyMatcher(t *testing.T) {
	boolPtr := func(b bool) *bool { return &b }

	tests := []struct {
		name      string
		body      string
		systemKey string
		want      bool
	}{
		{
			name:      "exact match",
			body:      `{"matchers":[{"name":"system_key","value":"SYS-001","isRegex":false,"isEqual":true}]}`,
			systemKey: "SYS-001",
			want:      true,
		},
		{
			name:      "isEqual absent defaults to true",
			body:      `{"matchers":[{"name":"system_key","value":"SYS-001","isRegex":false}]}`,
			systemKey: "SYS-001",
			want:      true,
		},
		{
			name:      "wrong value",
			body:      `{"matchers":[{"name":"system_key","value":"SYS-999","isRegex":false,"isEqual":true}]}`,
			systemKey: "SYS-001",
			want:      false,
		},
		{
			name:      "regex matcher rejected",
			body:      `{"matchers":[{"name":"system_key","value":"SYS-001","isRegex":true,"isEqual":true}]}`,
			systemKey: "SYS-001",
			want:      false,
		},
		{
			name:      "isEqual false rejected",
			body:      `{"matchers":[{"name":"system_key","value":"SYS-001","isRegex":false,"isEqual":false}]}`,
			systemKey: "SYS-001",
			want:      false,
		},
		{
			name:      "no system_key matcher",
			body:      `{"matchers":[{"name":"alertname","value":"DiskFull","isRegex":false,"isEqual":true}]}`,
			systemKey: "SYS-001",
			want:      false,
		},
		{
			name:      "invalid JSON",
			body:      `not json`,
			systemKey: "SYS-001",
			want:      false,
		},
	}

	_ = boolPtr // used in table for documentation clarity
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := silenceHasSystemKeyMatcher([]byte(tt.body), tt.systemKey)
			assert.Equal(t, tt.want, got)
		})
	}
}
