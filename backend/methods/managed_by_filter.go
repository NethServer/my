/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package methods

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/local"
)

// managedByOrganizationIDs reads the parent_organization_id list filter ("Managed
// by") and expands each organization to itself plus its whole subtree, the same
// set the hierarchy counters of an organization's detail page count over. The
// list RBAC scope still applies on top, so the expansion never widens
// visibility. It writes a 500 and returns ok=false when the expansion fails.
func managedByOrganizationIDs(c *gin.Context) (ids []string, ok bool) {
	parents := c.QueryArray("parent_organization_id")
	if len(parents) == 0 {
		return nil, true
	}
	expanded, err := local.NewOrganizationService().ExpandOrganizationIDs(parents)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to expand organization hierarchy", nil))
		return nil, false
	}
	return expanded, true
}
