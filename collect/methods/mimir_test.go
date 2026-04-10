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

func TestInjectSystemLabels(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		systemKey  string
		systemID   string
		wantKeyVal string
		wantIDVal  string
	}{
		{
			name:       "adds system_key and system_id when missing",
			body:       `[{"labels":{"alertname":"DiskFull","severity":"critical"}}]`,
			systemKey:  "SYS-KEY-001",
			systemID:   "uuid-001",
			wantKeyVal: "SYS-KEY-001",
			wantIDVal:  "uuid-001",
		},
		{
			name:       "preserves existing system_key",
			body:       `[{"labels":{"alertname":"DiskFull","system_key":"EXISTING"}}]`,
			systemKey:  "SYS-KEY-001",
			systemID:   "uuid-001",
			wantKeyVal: "EXISTING",
			wantIDVal:  "uuid-001",
		},
		{
			name:       "preserves existing system_id",
			body:       `[{"labels":{"alertname":"DiskFull","system_id":"EXISTING-UUID"}}]`,
			systemKey:  "SYS-KEY-001",
			systemID:   "uuid-001",
			wantKeyVal: "SYS-KEY-001",
			wantIDVal:  "EXISTING-UUID",
		},
		{
			name:       "handles missing labels object",
			body:       `[{"annotations":{"summary":"test"}}]`,
			systemKey:  "SYS-KEY-003",
			systemID:   "uuid-003",
			wantKeyVal: "SYS-KEY-003",
			wantIDVal:  "uuid-003",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := injectSystemLabels([]byte(tt.body), tt.systemKey, tt.systemID)

			var alerts []map[string]interface{}
			err := json.Unmarshal(result, &alerts)
			assert.NoError(t, err)
			assert.NotEmpty(t, alerts)

			labels, ok := alerts[0]["labels"].(map[string]interface{})
			assert.True(t, ok)
			assert.Equal(t, tt.wantKeyVal, labels["system_key"])
			assert.Equal(t, tt.wantIDVal, labels["system_id"])
		})
	}
}

func TestInjectSystemLabels_InvalidJSON(t *testing.T) {
	body := []byte("not json")
	result := injectSystemLabels(body, "SYS-001", "uuid-001")
	assert.Equal(t, body, result, "invalid JSON must be returned unchanged")
}

func TestInjectSystemLabels_EmptyArray(t *testing.T) {
	body := []byte("[]")
	result := injectSystemLabels(body, "SYS-001", "uuid-001")
	assert.Equal(t, body, result, "empty array must be returned unchanged")
}
