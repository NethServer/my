/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package local

import (
	"time"

	"github.com/nethesis/my/backend/cache"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/services/logto"
)

// ResolveUserByLogtoID builds a fully enriched User (roles, permissions, org)
// for a Logto ID, reusing the 10-minute Redis profile cache. On a cache hit it
// performs no live Logto call; on a miss it fetches the profile and enriches
// roles/permissions, then caches the result.
//
// It is shared by the token exchange and by API-key authentication: both need
// the owner's effective permissions without a per-request Logto round-trip.
func ResolveUserByLogtoID(logtoID string) (*models.User, error) {
	cacheKey := "user_profile:" + logtoID
	rc := cache.GetRedisClient()
	if rc != nil {
		var cached models.User
		if err := rc.Get(cacheKey, &cached); err == nil {
			// Require a resolved local ID: a cached entry with an empty ID
			// (user not yet present in the local DB when it was cached) would
			// otherwise mask the now-existing local row for the whole TTL and
			// break anything keyed on user.ID (e.g. listing API keys).
			if cached.Username != "" && cached.Email != "" && cached.ID != "" {
				return &cached, nil
			}
			_ = rc.Delete(cacheKey)
		}
	}

	userProfile, err := logto.GetUserProfileFromLogto(logtoID)
	if err != nil {
		logger.Logger.Warn().
			Err(err).
			Str("operation", "get_profile").
			Str("logto_id", logtoID).
			Msg("Failed to get user profile from Logto")
		userProfile = nil
	}

	var user models.User
	if localUser, err := NewUserService().GetUserByLogtoID(logtoID); err == nil {
		user = models.User{ID: localUser.ID, LogtoID: localUser.LogtoID}
	} else {
		user = models.User{ID: "", LogtoID: &logtoID}
	}

	if userProfile != nil {
		user.Username = userProfile.Username
		user.Email = userProfile.PrimaryEmail
		user.Name = userProfile.Name
		if userProfile.PrimaryPhone != "" {
			user.Phone = &userProfile.PrimaryPhone
		}
	}

	enriched, err := logto.EnrichUserWithRolesAndPermissions(logtoID)
	if err != nil {
		return nil, err
	}
	user.UserRoles = enriched.UserRoles
	user.UserRoleIDs = enriched.UserRoleIDs
	user.UserPermissions = enriched.UserPermissions
	user.OrgRole = enriched.OrgRole
	user.OrgRoleID = enriched.OrgRoleID
	user.OrgPermissions = enriched.OrgPermissions
	user.OrganizationID = enriched.OrganizationID
	user.OrganizationName = enriched.OrganizationName

	// Cache only complete profiles with a resolved local ID, to avoid
	// persisting transient Logto failures or a not-yet-synced user (empty ID)
	// for the full TTL.
	if rc != nil && userProfile != nil && user.Username != "" && user.ID != "" {
		_ = rc.Set(cacheKey, user, 10*time.Minute)
	}

	return &user, nil
}
