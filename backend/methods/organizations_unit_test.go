/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/nethesis/my/backend/models"
)

func TestOwnerEntryMatchesStatusFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	// the Owner organization is never suspended
	assert.True(t, ownerEntryMatchesFilters(c, models.OrganizationFilters{}))
	assert.True(t, ownerEntryMatchesFilters(c, models.OrganizationFilters{Statuses: []string{"enabled"}}))
	assert.True(t, ownerEntryMatchesFilters(c, models.OrganizationFilters{Statuses: []string{"enabled", "suspended"}}))
	assert.False(t, ownerEntryMatchesFilters(c, models.OrganizationFilters{Statuses: []string{"suspended"}}))
}
