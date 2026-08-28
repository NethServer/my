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

	"github.com/nethesis/my/backend/models"
)

// canWriteRebranding is the write-side gate of the whole rebranding surface:
// the configuration form, the per-product upload and both deletes go through
// it. Branding is inherited downwards, never written downwards — a partner
// configures its own organization and the ones below display it. So holding
// manage:rebranding is not a licence to write into another organization, not
// even one of your own children: that would silently override a configuration
// its own administrators own.
func TestCanWriteRebranding(t *testing.T) {
	gin.SetMode(gin.TestMode)

	newContext := func(user *models.User) (*gin.Context, *httptest.ResponseRecorder) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPut, "/", nil)
		c.Set("user", user)
		return c, rec
	}

	user := func(orgRole, orgID string) *models.User {
		return &models.User{
			ID:              "u1",
			OrganizationID:  orgID,
			OrgRole:         orgRole,
			UserRoles:       []string{"Admin"},
			UserPermissions: []string{"read:rebranding", "manage:rebranding"},
		}
	}

	t.Run("the owner writes any organization", func(t *testing.T) {
		c, rec := newContext(user("Owner", "owner-org"))
		assert.True(t, canWriteRebranding(c, "reseller-a"))
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("a reseller writes its own organization", func(t *testing.T) {
		c, rec := newContext(user("Reseller", "reseller-a"))
		assert.True(t, canWriteRebranding(c, "reseller-a"))
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("a reseller cannot write another reseller", func(t *testing.T) {
		c, rec := newContext(user("Reseller", "reseller-a"))
		assert.False(t, canWriteRebranding(c, "reseller-b"))
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("a distributor cannot write a reseller below it", func(t *testing.T) {
		c, rec := newContext(user("Distributor", "distributor-a"))
		assert.False(t, canWriteRebranding(c, "reseller-a"))
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("a customer cannot write the organization it inherits from", func(t *testing.T) {
		c, rec := newContext(user("Customer", "customer-a"))
		assert.False(t, canWriteRebranding(c, "reseller-a"))
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("the org role is matched case-insensitively", func(t *testing.T) {
		c, _ := newContext(user("owner", "owner-org"))
		assert.True(t, canWriteRebranding(c, "reseller-a"))
	})
}
