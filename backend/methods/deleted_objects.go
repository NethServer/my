/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
)

// rejectDeletedStatusOutsideOwnerOrg refuses a listing or an export filtered
// on soft-deleted objects to a caller outside the Owner organization. Seeing
// them, and restoring them, belongs to the Owner organization: for everyone
// else a Delete is final. It reports whether it wrote the refusal.
func rejectDeletedStatusOutsideOwnerOrg(c *gin.Context, orgRole string, statuses []string) bool {
	if models.IsGlobalOrgRole(orgRole) {
		return false
	}
	for _, status := range statuses {
		if strings.EqualFold(status, "deleted") {
			c.JSON(http.StatusForbidden, response.Forbidden("deleted objects are visible to the owner organization only", nil))
			return true
		}
	}
	return false
}
