/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRejectDeletedStatusOutsideOwnerOrg(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		orgRole  string
		statuses []string
		rejected bool
	}{
		{"owner organization asks for deleted", "owner", []string{"deleted"}, false},
		{"owner organization in another case", "Owner", []string{"enabled", "deleted"}, false},
		{"distributor asks for deleted", "distributor", []string{"deleted"}, true},
		{"reseller asks for deleted among others", "reseller", []string{"enabled", "DELETED"}, true},
		{"customer asks for active systems", "customer", []string{"active", "suspended"}, false},
		{"customer asks for no status", "customer", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			rejected := rejectDeletedStatusOutsideOwnerOrg(c, tt.orgRole, tt.statuses)

			assert.Equal(t, tt.rejected, rejected)
			if tt.rejected {
				assert.Equal(t, http.StatusForbidden, w.Code)
			} else {
				assert.False(t, c.Writer.Written())
			}
		})
	}
}
