/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/entities"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/local"
)

// favoriteUserID is the key system_favorites rows are stored under: the Logto
// ID, which every principal has (the bootstrap Owner has no local users row).
func favoriteUserID(user *models.User) string {
	if user == nil {
		return ""
	}
	if user.LogtoID != nil && *user.LogtoID != "" {
		return *user.LogtoID
	}
	return user.ID
}

// applySystemFavoriteFilters scopes the favorite flag, filter and sort of a
// systems list to the caller. With an impersonation token the caller is the
// target, so the impersonator sees the target's favorites.
func applySystemFavoriteFilters(c *gin.Context, filters *models.SystemListFilters) {
	if value, exists := c.Get("user"); exists {
		if user, ok := value.(*models.User); ok {
			filters.FavoritesOf = favoriteUserID(user)
		}
	}
	filters.FavoritesOnly = c.Query("favorite") == "true"
}

// AddSystemFavorite handles PUT /api/systems/:id/favorite - stars a system for
// the caller. Starring is personal and changes nothing on the system, so it
// needs only read access to it.
func AddSystemFavorite(c *gin.Context) {
	setSystemFavorite(c, true)
}

// RemoveSystemFavorite handles DELETE /api/systems/:id/favorite - unstars a
// system for the caller. It skips the access check: it only deletes the
// caller's own row, and a system that left the caller's scope (or was
// archived) must still be removable from their favorites.
func RemoveSystemFavorite(c *gin.Context) {
	setSystemFavorite(c, false)
}

func setSystemFavorite(c *gin.Context, favorite bool) {
	systemID := c.Param("id")
	if systemID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system ID required", nil))
		return
	}

	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}
	userKey := favoriteUserID(user)

	operation := "unfavorite"
	if favorite {
		operation = "favorite"
		if _, err := local.NewSystemsService().GetSystem(systemID, user.OrgRole, user.OrganizationID); helpers.HandleAccessError(c, err, "system", systemID) {
			return
		}
	}

	repo := entities.NewLocalSystemFavoriteRepository()
	var err error
	if favorite {
		err = repo.Add(userKey, systemID)
	} else {
		err = repo.Remove(userKey, systemID)
	}
	if err != nil {
		logger.LogBusinessOperation(c, "systems", operation, "system", systemID, false, err)
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to update favorite system", nil))
		return
	}

	logger.LogBusinessOperation(c, "systems", operation, "system", systemID, true, nil)

	message := "system removed from favorites"
	if favorite {
		message = "system added to favorites"
	}
	c.JSON(http.StatusOK, response.OK(message, gin.H{
		"system_id":   systemID,
		"is_favorite": favorite,
	}))
}
