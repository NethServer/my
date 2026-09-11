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
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nethesis/my/backend/cache"
	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/services/logto"
)

const userProfileCacheKeyPrefix = "user_profile:"

// ErrUserInactive is returned when the account is suspended or soft-deleted,
// locally or in Logto. Exchange, refresh, API-key and impersonation paths all
// resolve through here, so this is where a lifecycle change becomes a refusal.
var ErrUserInactive = errors.New("user account is suspended or deleted")

// ErrUserNotProvisioned is returned for a Logto account with no local users
// row outside the Owner organization: every partner account is created
// through my and has one, so a missing row is an account my never
// provisioned (created in the Logto console, or left over from a sync race),
// not one to serve. It wraps ErrUserInactive so callers refuse it the same way.
var ErrUserNotProvisioned = fmt.Errorf("%w: no local account", ErrUserInactive)

// InvalidateUserProfileCache drops cached profiles so the next resolve reads
// roles, permissions and organization live instead of serving the cached copy
// for the rest of the TTL.
func InvalidateUserProfileCache(logtoIDs ...string) {
	rc := cache.GetRedisClient()
	if rc == nil {
		return
	}
	for _, logtoID := range logtoIDs {
		if logtoID == "" {
			continue
		}
		_ = rc.Delete(userProfileCacheKeyPrefix + logtoID)
	}
}

// ResolveUserByLogtoID builds a fully enriched User (roles, permissions, org)
// for a Logto ID, reusing the 10-minute Redis profile cache. On a cache hit it
// performs no live Logto call; on a miss it fetches the profile and enriches
// roles/permissions, then caches the result.
//
// It is shared by the token exchange and by API-key authentication: both need
// the owner's effective permissions without a per-request Logto round-trip.
func ResolveUserByLogtoID(logtoID string) (*models.User, error) {
	// Lifecycle first, before any cache: a suspended or soft-deleted account is
	// refused here whatever a cached profile says, so revocation does not
	// depend on every writer remembering to invalidate the cache. Accounts
	// with no local row (the bootstrap owner) have no lifecycle to check.
	if database.DB != nil {
		active, err := NewUserService().IsUserActive(logtoID)
		if err != nil {
			return nil, fmt.Errorf("failed to verify user lifecycle: %w", err)
		}
		if !active {
			return nil, ErrUserInactive
		}
	}

	cacheKey := userProfileCacheKeyPrefix + logtoID
	rc := cache.GetRedisClient()
	if rc != nil {
		var cached models.User
		if err := rc.Get(cacheKey, &cached); err == nil {
			// Require a resolved local ID: a cached entry with an empty ID
			// (user not yet present in the local DB when it was cached) would
			// otherwise mask the now-existing local row for the whole TTL and
			// break anything keyed on user.ID (e.g. listing API keys).
			// The bootstrap owner account is the exception: it never gains a
			// local users row, so its empty ID is permanent and caching it is
			// safe (and needed — owner API keys resolve here on every request).
			if cached.Username != "" && cached.Email != "" && (cached.ID != "" || strings.EqualFold(cached.OrgRole, "owner")) {
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
		user = models.User{ID: localUser.ID, LogtoID: localUser.LogtoID, HasAvatar: localUser.HasAvatar}
	} else {
		user = models.User{ID: "", LogtoID: &logtoID}
	}

	if userProfile != nil {
		// Suspended in Logto (console, or an org cascade that reached Logto but
		// not the local row): the IdP refuses sign-in, my must refuse tokens.
		if userProfile.IsSuspended {
			return nil, ErrUserInactive
		}
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

	// Only the bootstrap owner legitimately lacks a local row.
	if user.ID == "" && !strings.EqualFold(user.OrgRole, "owner") {
		return nil, ErrUserNotProvisioned
	}

	// Cache only complete profiles with a resolved local ID, to avoid
	// persisting transient Logto failures or a not-yet-synced user (empty ID)
	// for the full TTL. The bootstrap owner account never gets a local ID and
	// is cached anyway (see the read-side guard above), but with a shorter
	// TTL: an owner API key rechecks the org role from this entry, and that
	// account is managed directly in Logto with no lifecycle hook to
	// invalidate it, so a role downgrade must not linger the full window.
	// Staff users of the Owner organization have a local row and lifecycle
	// invalidation, so they take the regular TTL.
	if rc != nil && userProfile != nil && user.Username != "" && (user.ID != "" || strings.EqualFold(user.OrgRole, "owner")) {
		ttl := 10 * time.Minute
		if user.ID == "" && strings.EqualFold(user.OrgRole, "owner") {
			ttl = 1 * time.Minute
		}
		_ = rc.Set(cacheKey, user, ttl)
	}

	return &user, nil
}
