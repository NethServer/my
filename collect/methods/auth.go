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

	"github.com/nethesis/my/collect/database"
	"github.com/nethesis/my/collect/logger"
	"github.com/nethesis/my/collect/response"
)

// Native /auth for the appliance enterprise feeds (distfeed, ns-signatures-
// proxy, blacklists). The feeds forwardAuth system_key:secret Basic
// credentials; BasicAuthMiddleware already rejects unknown, unregistered,
// suspended and deleted systems, so reaching a handler means the
// subscription is valid.
//
//   GET /auth                        -> 200 (valid subscription)
//   GET /auth/service/<id>[?scope=]  -> 200 if the system holds an active
//                                       entitlement for <id>, 403 otherwise.
//                                       ?scope=<app-instance> checks a grant
//                                       narrowed to that application instance
//                                       (e.g. nethvoice1); a system-wide
//                                       grant (empty scope) also covers every
//                                       instance (fallback).
//   GET /auth/product/<name>         -> 200 (valid subscription) — product-
//                                       level enforcement is deferred until
//                                       the legacy system_products data is
//                                       mapped; this mirrors the transitional
//                                       broker behaviour
//
// Same semantics as the legacy nethserver_basic_auth.php: 401 = bad or
// suspended credentials, 403 = valid system without the specific add-on.

// AuthCheck handles GET /auth — subscription-level check only.
func AuthCheck(c *gin.Context) {
	if _, ok := getAuthenticatedSystemID(c); !ok {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("authentication required", nil))
		return
	}
	c.JSON(http.StatusOK, response.OK("authorized", nil))
}

// AuthCheckService handles GET /auth/service/:id — grants access only when
// the system holds an active entitlement for the requested service id
// (active = not revoked, not expired; valid_until NULL = perpetual).
func AuthCheckService(c *gin.Context) {
	systemID, ok := getAuthenticatedSystemID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("authentication required", nil))
		return
	}

	serviceID := c.Param("id")
	if serviceID == "" {
		c.JSON(http.StatusForbidden, response.Forbidden("service id required", nil))
		return
	}

	// A system-wide grant (scope '') always qualifies; when the caller asks
	// for a specific application instance, an instance-narrowed grant does
	// too. Without ?scope only system-wide grants count.
	// The requested id may be a legacy wire alias (the feeds still call
	// ng-blacklist while the canonical catalog id is nsec-blacklist): the
	// join resolves both.
	scope := c.Query("scope")

	var active bool
	err := database.DB.QueryRow(
		`SELECT EXISTS (
		     SELECT 1 FROM system_entitlements e
		     JOIN entitlement_catalog cat ON cat.id = e.entitlement
		     WHERE e.system_id = $1
		       AND ($2 IN (cat.id, cat.legacy_alias))
		       AND e.scope IN ('', $3)
		       AND e.revoked_at IS NULL
		       AND (e.valid_until IS NULL OR e.valid_until > NOW())
		 )`, systemID, serviceID, scope).Scan(&active)
	if err != nil {
		logger.ComponentLogger("auth").Error().Err(err).
			Str("system_id", systemID).
			Str("service", serviceID).
			Msg("Entitlement lookup failed")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("entitlement lookup failed", nil))
		return
	}

	if !active {
		c.JSON(http.StatusForbidden, response.Forbidden("service not enabled for this system", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("authorized", nil))
}

// AuthCheckProduct handles GET /auth/product/:name — product-level checks are
// not enforced yet (fase 1): any system with a valid subscription passes,
// matching the transitional broker's behaviour for /auth/product/*.
func AuthCheckProduct(c *gin.Context) {
	if _, ok := getAuthenticatedSystemID(c); !ok {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("authentication required", nil))
		return
	}
	c.JSON(http.StatusOK, response.OK("authorized", nil))
}
