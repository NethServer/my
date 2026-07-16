/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/nethesis/my/backend/entities"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/local"
)

// Catalog ids are kebab-case (convention: nsec-<service>, ns8-<app>,
// <app>-<module>; the ng-* ids are the legacy wire ids).
var entitlementIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,98}$`)

// isSystemBlocked reports whether the system cannot use its entitlements at
// all: suspended or deleted (directly, or via the org cascade). Grants stay
// untouched but are reported with status=suspended.
func isSystemBlocked(system *models.System) bool {
	return system.Status == "suspended" || system.Status == "deleted"
}

// isEntitlementAdmin returns true for the ADMINISTRATIVE surface — catalog
// management, manual grants via API, fleet-wide visibility: only the owner
// organization or a Super Admin user (Nethesis).
func isEntitlementAdmin(u *models.User) bool {
	return strings.EqualFold(u.OrgRole, "owner") || slices.Contains(u.UserRoles, "Super Admin")
}

// canTransactEntitlements returns true for the TRANSACTIONAL surface — buy
// on the shop / cancel a subscription (activate/deactivate): the dedicated
// manage:entitlements permission, held by the Backoffice and Super Admin
// user roles.
func canTransactEntitlements(u *models.User) bool {
	return slices.Contains(u.UserPermissions, "manage:entitlements") ||
		slices.Contains(u.OrgPermissions, "manage:entitlements")
}

// entitlementAccessCheck resolves the system with the caller's hierarchy
// scope (same validation as GET /systems/:id) and, for writes, restricts to
// entitlement managers (manage:entitlements). Managers bypass the hierarchy
// (they operate on any system): Nethesis licensing back-office staff live
// under the Nethesis Italia distributor but must manage licences fleet-wide.
func entitlementAccessCheck(c *gin.Context, write bool) (system *models.System, user *models.User, ok bool) {
	systemID := c.Param("id")
	if systemID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("system ID required", nil))
		return nil, nil, false
	}

	u, found := helpers.GetUserFromContext(c)
	if !found {
		return nil, nil, false
	}
	user = u

	isAdmin := isEntitlementAdmin(u)

	if write && !isAdmin {
		c.JSON(http.StatusForbidden, response.Forbidden("only the owner organization or a Super Admin can manage grants directly", nil))
		return nil, nil, false
	}

	effectiveOrgRole := u.OrgRole
	if isAdmin {
		effectiveOrgRole = "owner"
	}

	systemsService := local.NewSystemsService()
	sys, err := systemsService.GetSystem(systemID, effectiveOrgRole, u.OrganizationID)
	if helpers.HandleAccessError(c, err, "system", systemID) {
		return nil, nil, false
	}

	return sys, user, true
}

// systemTypeMismatch returns true when the catalog item is restricted to a
// system type and the system's known type differs (unknown type = allowed:
// the type is only learned from the first inventory).
func systemTypeMismatch(item *models.EntitlementCatalogItem, systemType string) bool {
	return item.SystemType != "" && systemType != "" && systemType != item.SystemType
}

// ===========================================
// ENTITLEMENT CATALOG
// ===========================================

// ListEntitlementCatalog handles GET /api/entitlements/catalog
func ListEntitlementCatalog(c *gin.Context) {
	repo := entities.NewLocalEntitlementCatalogRepository()
	items, err := repo.List()
	if err != nil {
		logger.RequestLogger(c, "entitlements").Error().Err(err).Msg("Failed to list entitlement catalog")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to list entitlement catalog", nil))
		return
	}
	c.JSON(http.StatusOK, response.OK("entitlement catalog retrieved successfully", gin.H{"catalog": items}))
}

// catalogWriteGate rejects callers that are not entitlement admins (owner
// org or Super Admin): the catalog and its availability rules are Nethesis
// product management.
func catalogWriteGate(c *gin.Context) (*models.User, bool) {
	u, found := helpers.GetUserFromContext(c)
	if !found {
		return nil, false
	}
	if !isEntitlementAdmin(u) {
		c.JSON(http.StatusForbidden, response.Forbidden("only the owner organization or a Super Admin can manage the entitlement catalog", nil))
		return nil, false
	}
	return u, true
}

// transactGate rejects callers without the manage:entitlements permission
// (buy/cancel surface: Backoffice, Super Admin, shop owner key).
func transactGate(c *gin.Context) (*models.User, bool) {
	u, found := helpers.GetUserFromContext(c)
	if !found {
		return nil, false
	}
	if !canTransactEntitlements(u) {
		c.JSON(http.StatusForbidden, response.Forbidden("manage:entitlements permission required", nil))
		return nil, false
	}
	return u, true
}

// CreateEntitlementCatalogItem handles POST /api/entitlements/catalog
func CreateEntitlementCatalogItem(c *gin.Context) {
	if _, ok := catalogWriteGate(c); !ok {
		return
	}

	var req models.CreateEntitlementCatalogRequest
	if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
		return
	}

	if !entitlementIDPattern.MatchString(req.ID) {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid entitlement id: lowercase kebab-case required (e.g. nsec-service, ns8-app, app-module)", nil))
		return
	}
	if req.LegacyAlias != "" && !entitlementIDPattern.MatchString(req.LegacyAlias) {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid legacy_alias: lowercase kebab-case required", nil))
		return
	}

	switch req.Kind {
	case "":
		req.Kind = models.EntitlementKindService
	case models.EntitlementKindService, models.EntitlementKindModule:
	default:
		c.JSON(http.StatusBadRequest, response.BadRequest("kind must be service or module", nil))
		return
	}
	switch req.SystemType {
	case "", "nsec", "ns8":
	default:
		c.JSON(http.StatusBadRequest, response.BadRequest("system_type must be nsec, ns8 or empty", nil))
		return
	}

	repo := entities.NewLocalEntitlementCatalogRepository()
	item, err := repo.Create(&req)
	if err == entities.ErrCatalogItemExists {
		c.JSON(http.StatusConflict, response.Conflict("catalog item already exists", nil))
		return
	}
	if err != nil {
		logger.RequestLogger(c, "entitlements").Error().Err(err).Str("catalog_id", req.ID).Msg("Failed to create catalog item")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to create catalog item", nil))
		return
	}

	logger.RequestLogger(c, "entitlements").Info().
		Str("operation", "create_catalog_item").
		Str("catalog_id", req.ID).
		Bool("scoped", req.Scoped).
		Msg("Entitlement catalog item created")

	c.JSON(http.StatusCreated, response.Created("catalog item created successfully", item))
}

// UpdateEntitlementCatalogItem handles PUT /api/entitlements/catalog/:id
func UpdateEntitlementCatalogItem(c *gin.Context) {
	if _, ok := catalogWriteGate(c); !ok {
		return
	}

	var req models.UpdateEntitlementCatalogRequest
	if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
		return
	}

	repo := entities.NewLocalEntitlementCatalogRepository()
	item, err := repo.Update(c.Param("id"), req.DisplayName, req.Description)
	if err == entities.ErrCatalogItemNotFound {
		c.JSON(http.StatusNotFound, response.NotFound("catalog item not found", nil))
		return
	}
	if err != nil {
		logger.RequestLogger(c, "entitlements").Error().Err(err).Str("catalog_id", c.Param("id")).Msg("Failed to update catalog item")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to update catalog item", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("catalog item updated successfully", item))
}

// DeleteEntitlementCatalogItem handles DELETE /api/entitlements/catalog/:id
// Refused while grants reference the item.
func DeleteEntitlementCatalogItem(c *gin.Context) {
	if _, ok := catalogWriteGate(c); !ok {
		return
	}

	repo := entities.NewLocalEntitlementCatalogRepository()
	err := repo.Delete(c.Param("id"))
	if err == entities.ErrCatalogItemNotFound {
		c.JSON(http.StatusNotFound, response.NotFound("catalog item not found", nil))
		return
	}
	if err == entities.ErrCatalogItemInUse {
		c.JSON(http.StatusConflict, response.Conflict("catalog item is referenced by existing grants", nil))
		return
	}
	if err != nil {
		logger.RequestLogger(c, "entitlements").Error().Err(err).Str("catalog_id", c.Param("id")).Msg("Failed to delete catalog item")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to delete catalog item", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("catalog item deleted successfully", nil))
}

// ===========================================
// AVAILABILITY (commercial unlocks)
// ===========================================

// ListEntitlementAvailability handles GET /api/entitlements/catalog/:id/availability
func ListEntitlementAvailability(c *gin.Context) {
	catalogRepo := entities.NewLocalEntitlementCatalogRepository()
	if _, err := catalogRepo.Get(c.Param("id")); err == entities.ErrCatalogItemNotFound {
		c.JSON(http.StatusNotFound, response.NotFound("catalog item not found", nil))
		return
	}

	repo := entities.NewLocalEntitlementAvailabilityRepository()
	rules, err := repo.ListByEntitlement(c.Param("id"))
	if err != nil {
		logger.RequestLogger(c, "entitlements").Error().Err(err).Msg("Failed to list availability")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to list availability", nil))
		return
	}
	c.JSON(http.StatusOK, response.OK("availability retrieved successfully", gin.H{"availability": rules}))
}

// CreateEntitlementAvailability handles POST /api/entitlements/catalog/:id/availability
// Unlocks the catalog item for a hierarchy role OR one organization.
func CreateEntitlementAvailability(c *gin.Context) {
	user, ok := catalogWriteGate(c)
	if !ok {
		return
	}

	var req models.CreateEntitlementAvailabilityRequest
	if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
		return
	}

	role := strings.ToLower(req.OrgRole)
	if (role == "") == (req.OrganizationID == "") {
		c.JSON(http.StatusBadRequest, response.BadRequest("set exactly one of org_role or organization_id", nil))
		return
	}
	if role != "" && role != "distributor" && role != "reseller" && role != "customer" {
		c.JSON(http.StatusBadRequest, response.BadRequest("org_role must be distributor, reseller or customer", nil))
		return
	}

	createdBy := map[string]interface{}{
		"user_id":           user.ID,
		"user_name":         user.Name,
		"organization_id":   user.OrganizationID,
		"organization_name": user.OrganizationName,
	}

	repo := entities.NewLocalEntitlementAvailabilityRepository()
	rule, err := repo.Add(c.Param("id"), role, req.OrganizationID, createdBy)
	if err == entities.ErrAvailabilityExists {
		c.JSON(http.StatusConflict, response.Conflict("availability rule already exists", nil))
		return
	}
	if err == entities.ErrCatalogItemNotFound {
		c.JSON(http.StatusNotFound, response.NotFound("catalog item not found", nil))
		return
	}
	if err != nil {
		logger.RequestLogger(c, "entitlements").Error().Err(err).Msg("Failed to add availability")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to add availability", nil))
		return
	}

	logger.RequestLogger(c, "entitlements").Info().
		Str("operation", "create_availability").
		Str("catalog_id", c.Param("id")).
		Str("org_role", role).
		Str("organization_id", req.OrganizationID).
		Msg("Entitlement availability rule created")

	c.JSON(http.StatusCreated, response.Created("availability rule created successfully", rule))
}

// DeleteEntitlementAvailability handles DELETE /api/entitlements/catalog/:id/availability/:rule_id
func DeleteEntitlementAvailability(c *gin.Context) {
	if _, ok := catalogWriteGate(c); !ok {
		return
	}

	repo := entities.NewLocalEntitlementAvailabilityRepository()
	err := repo.Remove(c.Param("id"), c.Param("rule_id"))
	if err == entities.ErrAvailabilityNotFound {
		c.JSON(http.StatusNotFound, response.NotFound("availability rule not found", nil))
		return
	}
	if err != nil {
		logger.RequestLogger(c, "entitlements").Error().Err(err).Msg("Failed to remove availability")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to remove availability", nil))
		return
	}
	c.JSON(http.StatusOK, response.OK("availability rule removed successfully", nil))
}

// ListAvailableEntitlements handles GET /api/entitlements/available — the
// catalog items the CALLER's organization may buy/self-activate (drives the
// my UI and, in fase 3, the shop). Owner and Super Admin see the whole
// catalog (they can grant anything manually anyway).
func ListAvailableEntitlements(c *gin.Context) {
	u, found := helpers.GetUserFromContext(c)
	if !found {
		return
	}

	if isEntitlementAdmin(u) {
		repo := entities.NewLocalEntitlementCatalogRepository()
		items, err := repo.List()
		if err != nil {
			logger.RequestLogger(c, "entitlements").Error().Err(err).Msg("Failed to list catalog")
			c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to list available entitlements", nil))
			return
		}
		c.JSON(http.StatusOK, response.OK("available entitlements retrieved successfully", gin.H{"available": items}))
		return
	}

	repo := entities.NewLocalEntitlementAvailabilityRepository()
	items, err := repo.ListAvailableFor(u.OrgRole, u.OrganizationID)
	if err != nil {
		logger.RequestLogger(c, "entitlements").Error().Err(err).Msg("Failed to list available entitlements")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to list available entitlements", nil))
		return
	}
	c.JSON(http.StatusOK, response.OK("available entitlements retrieved successfully", gin.H{"available": items}))
}

// ===========================================
// SHOP ACTIVATION (webhook-facing)
// ===========================================

// ActivateEntitlement handles POST /api/entitlements/activate — the shop
// webhook calls it after a purchase or subscription renewal with an owner
// API key. Addressed by system_key; idempotent (existing grant renewed in
// place, safe on webhook retries).
func ActivateEntitlement(c *gin.Context) {
	user, ok := transactGate(c)
	if !ok {
		return
	}

	var req models.ActivateEntitlementRequest
	if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
		return
	}

	catalogRepo := entities.NewLocalEntitlementCatalogRepository()
	item, err := catalogRepo.Resolve(req.Entitlement)
	if err == entities.ErrCatalogItemNotFound {
		c.JSON(http.StatusBadRequest, response.BadRequest("unknown entitlement", nil))
		return
	}
	if err != nil {
		logger.RequestLogger(c, "entitlements").Error().Err(err).Msg("Failed to resolve entitlement")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to resolve entitlement", nil))
		return
	}
	if req.Scope != "" && !item.Scoped {
		c.JSON(http.StatusBadRequest, response.BadRequest("this entitlement does not support per-application scope", nil))
		return
	}

	repo := entities.NewLocalSystemEntitlementRepository()
	systemID, systemType, err := repo.FindSystemIDByKey(req.SystemKey)
	if err == entities.ErrEntitlementNotFound {
		c.JSON(http.StatusNotFound, response.NotFound("system not found for this key", nil))
		return
	}
	if err != nil {
		logger.RequestLogger(c, "entitlements").Error().Err(err).Msg("Failed to resolve system key")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to resolve system key", nil))
		return
	}
	if systemTypeMismatch(item, systemType) {
		c.JSON(http.StatusBadRequest, response.BadRequest("this entitlement applies to "+item.SystemType+" systems only", nil))
		return
	}

	createdBy := map[string]interface{}{
		"user_id":           user.ID,
		"user_name":         user.Name,
		"organization_id":   user.OrganizationID,
		"organization_name": user.OrganizationName,
		"channel":           "shop",
	}

	grant, err := repo.Upsert(systemID, item.ID, req.Scope, models.EntitlementSourceShop, req.SourceRef, req.ValidUntil, createdBy)
	if err != nil {
		logger.RequestLogger(c, "entitlements").Error().Err(err).
			Str("system_key", req.SystemKey).
			Str("entitlement", item.ID).
			Msg("Failed to activate entitlement")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to activate entitlement", nil))
		return
	}

	logger.RequestLogger(c, "entitlements").Info().
		Str("operation", "activate_entitlement").
		Str("system_key", req.SystemKey).
		Str("entitlement", item.ID).
		Str("scope", req.Scope).
		Str("source_ref", req.SourceRef).
		Msg("Entitlement activated via shop")

	c.JSON(http.StatusOK, response.OK("entitlement activated successfully", grant))
}

// DeactivateEntitlement handles POST /api/entitlements/deactivate — the shop
// webhook calls it when a subscription is cancelled or expires. Idempotent.
func DeactivateEntitlement(c *gin.Context) {
	if _, ok := transactGate(c); !ok {
		return
	}

	var req models.DeactivateEntitlementRequest
	if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
		return
	}

	catalogRepo := entities.NewLocalEntitlementCatalogRepository()
	item, err := catalogRepo.Resolve(req.Entitlement)
	if err == entities.ErrCatalogItemNotFound {
		c.JSON(http.StatusBadRequest, response.BadRequest("unknown entitlement", nil))
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to resolve entitlement", nil))
		return
	}

	repo := entities.NewLocalSystemEntitlementRepository()
	systemID, _, err := repo.FindSystemIDByKey(req.SystemKey)
	if err == entities.ErrEntitlementNotFound {
		c.JSON(http.StatusNotFound, response.NotFound("system not found for this key", nil))
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to resolve system key", nil))
		return
	}

	existing, err := repo.Get(systemID, item.ID, req.Scope)
	if err == entities.ErrEntitlementNotFound {
		c.JSON(http.StatusNotFound, response.NotFound("entitlement not found for this system", nil))
		return
	}
	if err != nil {
		logger.RequestLogger(c, "entitlements").Error().Err(err).Msg("Failed to read entitlement")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to deactivate entitlement", nil))
		return
	}

	// Reference matching (when the shop sends one): a cancelled UNPAID order
	// only clears its own pending marker, and an order that neither created
	// the grant nor is pending on it must not revoke what another order paid
	// for.
	if req.SourceRef != "" {
		if existing.PendingRef == req.SourceRef {
			grant, err := repo.ClearPending(systemID, item.ID, req.Scope)
			if err != nil && err != entities.ErrEntitlementNotFound {
				logger.RequestLogger(c, "entitlements").Error().Err(err).Msg("Failed to clear pending entitlement")
				c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to deactivate entitlement", nil))
				return
			}
			logger.RequestLogger(c, "entitlements").Info().
				Str("operation", "clear_pending_entitlement").
				Str("system_key", req.SystemKey).
				Str("entitlement", item.ID).
				Str("scope", req.Scope).
				Str("source_ref", req.SourceRef).
				Msg("Pending entitlement activation cleared (order cancelled before payment)")
			c.JSON(http.StatusOK, response.OK("pending activation cleared", grant))
			return
		}
		if existing.SourceRef != req.SourceRef {
			logger.RequestLogger(c, "entitlements").Warn().
				Str("operation", "deactivate_entitlement").
				Str("system_key", req.SystemKey).
				Str("entitlement", item.ID).
				Str("scope", req.Scope).
				Str("source_ref", req.SourceRef).
				Str("grant_source_ref", existing.SourceRef).
				Msg("Deactivate reference does not match the grant; leaving it untouched")
			c.JSON(http.StatusOK, response.OK("reference does not match the current grant, nothing deactivated", existing))
			return
		}
	}

	grant, err := repo.Revoke(systemID, item.ID, req.Scope, models.EntitlementSourceShop)
	if err == entities.ErrEntitlementNotFound {
		c.JSON(http.StatusNotFound, response.NotFound("entitlement not found for this system", nil))
		return
	}
	if err != nil {
		logger.RequestLogger(c, "entitlements").Error().Err(err).Msg("Failed to deactivate entitlement")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to deactivate entitlement", nil))
		return
	}

	logger.RequestLogger(c, "entitlements").Info().
		Str("operation", "deactivate_entitlement").
		Str("system_key", req.SystemKey).
		Str("entitlement", item.ID).
		Str("scope", req.Scope).
		Str("source_ref", req.SourceRef).
		Msg("Entitlement deactivated via shop")

	c.JSON(http.StatusOK, response.OK("entitlement deactivated successfully", grant))
}

// PendingEntitlement handles POST /api/entitlements/pending — the shop calls
// it at checkout, when the order exists but the payment (bank transfer/RiBa)
// hasn't been confirmed yet. Display-only: the UI shows "pending" instead of
// offering another purchase; enforcement is not affected. Idempotent; the
// later activate (order completed) or deactivate (order cancelled) resolves
// the marker.
func PendingEntitlement(c *gin.Context) {
	user, ok := transactGate(c)
	if !ok {
		return
	}

	var req models.PendingEntitlementRequest
	if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
		return
	}

	catalogRepo := entities.NewLocalEntitlementCatalogRepository()
	item, err := catalogRepo.Resolve(req.Entitlement)
	if err == entities.ErrCatalogItemNotFound {
		c.JSON(http.StatusBadRequest, response.BadRequest("unknown entitlement", nil))
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to resolve entitlement", nil))
		return
	}
	if req.Scope != "" && !item.Scoped {
		c.JSON(http.StatusBadRequest, response.BadRequest("this entitlement does not support per-application scope", nil))
		return
	}

	repo := entities.NewLocalSystemEntitlementRepository()
	systemID, systemType, err := repo.FindSystemIDByKey(req.SystemKey)
	if err == entities.ErrEntitlementNotFound {
		c.JSON(http.StatusNotFound, response.NotFound("system not found for this key", nil))
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to resolve system key", nil))
		return
	}
	if systemTypeMismatch(item, systemType) {
		c.JSON(http.StatusBadRequest, response.BadRequest("this entitlement applies to "+item.SystemType+" systems only", nil))
		return
	}

	createdBy := map[string]interface{}{
		"user_id":           user.ID,
		"user_name":         user.Name,
		"organization_id":   user.OrganizationID,
		"organization_name": user.OrganizationName,
		"channel":           "shop",
	}

	grant, err := repo.MarkPending(systemID, item.ID, req.Scope, req.SourceRef, createdBy)
	if err != nil {
		logger.RequestLogger(c, "entitlements").Error().Err(err).
			Str("system_key", req.SystemKey).
			Str("entitlement", item.ID).
			Msg("Failed to mark entitlement pending")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to mark entitlement pending", nil))
		return
	}

	logger.RequestLogger(c, "entitlements").Info().
		Str("operation", "pending_entitlement").
		Str("system_key", req.SystemKey).
		Str("entitlement", item.ID).
		Str("scope", req.Scope).
		Str("source_ref", req.SourceRef).
		Msg("Entitlement activation marked pending (order awaiting payment)")

	c.JSON(http.StatusOK, response.OK("entitlement activation marked pending", grant))
}

// ===========================================
// REPORTING
// ===========================================

// grantsOrgScope computes the org visibility set for the caller: nil (no
// restriction) for owner org and Super Admin, the caller's hierarchy
// otherwise — buyers see their own modules/expirations, owner sees the fleet.
func grantsOrgScope(u *models.User) ([]string, error) {
	if isEntitlementAdmin(u) {
		return nil, nil
	}
	return local.NewUserService().GetHierarchicalOrganizationIDs(u.OrgRole, u.OrganizationID)
}

// GetEntitlementGrants handles GET /api/entitlements/grants — fleet-wide (or
// hierarchy-wide) grants report with filters: entitlement, organization_id,
// source, active=true, expiring_before=RFC3339, page/page_size.
func GetEntitlementGrants(c *gin.Context) {
	u, found := helpers.GetUserFromContext(c)
	if !found {
		return
	}

	scope, err := grantsOrgScope(u)
	if err != nil {
		logger.RequestLogger(c, "entitlements").Error().Err(err).Msg("Failed to resolve org scope")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to resolve organization scope", nil))
		return
	}

	filter := entities.GrantsReportFilter{
		Entitlement:    c.Query("entitlement"),
		OrganizationID: c.Query("organization_id"),
		Source:         c.Query("source"),
		ActiveOnly:     c.Query("active") == "true",
		OrgScope:       scope,
	}
	if v := c.Query("expiring_before"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			c.JSON(http.StatusBadRequest, response.BadRequest("expiring_before must be RFC3339", nil))
			return
		}
		filter.ExpiringBefore = &t
	}

	page, pageSize := 1, 50
	if v, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil && v > 0 {
		page = v
	}
	if v, err := strconv.Atoi(c.DefaultQuery("page_size", "50")); err == nil && v > 0 && v <= 200 {
		pageSize = v
	}

	repo := entities.NewLocalSystemEntitlementRepository()
	rows, total, err := repo.ListGrants(filter, pageSize, (page-1)*pageSize)
	if err != nil {
		logger.RequestLogger(c, "entitlements").Error().Err(err).Msg("Failed to list grants")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to list grants", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("grants retrieved successfully", gin.H{
		"grants":    rows,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}))
}

// GetEntitlementStats handles GET /api/entitlements/stats — active grants
// per entitlement per organization, within the caller's visibility.
func GetEntitlementStats(c *gin.Context) {
	u, found := helpers.GetUserFromContext(c)
	if !found {
		return
	}

	scope, err := grantsOrgScope(u)
	if err != nil {
		logger.RequestLogger(c, "entitlements").Error().Err(err).Msg("Failed to resolve org scope")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to resolve organization scope", nil))
		return
	}

	repo := entities.NewLocalSystemEntitlementRepository()
	stats, err := repo.Stats(scope)
	if err != nil {
		logger.RequestLogger(c, "entitlements").Error().Err(err).Msg("Failed to compute stats")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to compute entitlement stats", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("entitlement stats retrieved successfully", gin.H{"stats": stats}))
}

// ===========================================
// SYSTEM ENTITLEMENTS
// ===========================================

// ListSystemEntitlements handles GET /api/systems/:id/entitlements
func ListSystemEntitlements(c *gin.Context) {
	system, _, ok := entitlementAccessCheck(c, false)
	if !ok {
		return
	}
	systemID := system.ID

	repo := entities.NewLocalSystemEntitlementRepository()
	list, err := repo.ListBySystem(systemID)
	if err != nil {
		logger.RequestLogger(c, "entitlements").Error().Err(err).
			Str("system_id", systemID).
			Msg("Failed to list entitlements")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to list entitlements", nil))
		return
	}

	if isSystemBlocked(system) {
		for _, e := range list {
			e.Status = models.EntitlementStatus(e.Active, e.RevokedAt, true, e.PendingRef)
		}
	}

	c.JSON(http.StatusOK, response.OK("entitlements retrieved successfully", gin.H{
		"entitlements": list,
	}))
}

// CreateSystemEntitlement handles POST /api/systems/:id/entitlements
func CreateSystemEntitlement(c *gin.Context) {
	system, user, ok := entitlementAccessCheck(c, true)
	if !ok {
		return
	}
	systemID := system.ID

	var req models.CreateSystemEntitlementRequest
	if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
		return
	}

	catalogRepo := entities.NewLocalEntitlementCatalogRepository()
	item, err := catalogRepo.Get(req.Entitlement)
	if err == entities.ErrCatalogItemNotFound {
		c.JSON(http.StatusBadRequest, response.BadRequest("unknown entitlement", nil))
		return
	}
	if err != nil {
		logger.RequestLogger(c, "entitlements").Error().Err(err).Msg("Failed to read entitlement catalog")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to read entitlement catalog", nil))
		return
	}

	if req.Scope != "" && !item.Scoped {
		c.JSON(http.StatusBadRequest, response.BadRequest("this entitlement does not support per-application scope", nil))
		return
	}

	systemType := ""
	if system.Type != nil {
		systemType = *system.Type
	}
	if systemTypeMismatch(item, systemType) {
		c.JSON(http.StatusBadRequest, response.BadRequest("this entitlement applies to "+item.SystemType+" systems only", nil))
		return
	}

	source := req.Source
	switch source {
	case "":
		source = models.EntitlementSourceManual
	case models.EntitlementSourceManual, models.EntitlementSourceShop, models.EntitlementSourceLegacyImport:
	default:
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid source", nil))
		return
	}

	createdBy := map[string]interface{}{
		"user_id":           user.ID,
		"user_name":         user.Name,
		"organization_id":   user.OrganizationID,
		"organization_name": user.OrganizationName,
	}

	repo := entities.NewLocalSystemEntitlementRepository()
	entitlement, err := repo.Create(systemID, req.Entitlement, req.Scope, source, req.SourceRef, req.ValidFrom, req.ValidUntil, createdBy)
	if err == entities.ErrEntitlementExists {
		c.JSON(http.StatusConflict, response.Conflict("entitlement already exists for this system", nil))
		return
	}
	if err != nil {
		logger.RequestLogger(c, "entitlements").Error().Err(err).
			Str("system_id", systemID).
			Str("entitlement", req.Entitlement).
			Msg("Failed to create entitlement")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to create entitlement", nil))
		return
	}

	logger.RequestLogger(c, "entitlements").Info().
		Str("operation", "create_entitlement").
		Str("system_id", systemID).
		Str("entitlement", req.Entitlement).
		Str("scope", req.Scope).
		Str("source", source).
		Msg("Entitlement created")

	entitlement.Status = models.EntitlementStatus(entitlement.Active, entitlement.RevokedAt, isSystemBlocked(system), entitlement.PendingRef)
	c.JSON(http.StatusCreated, response.Created("entitlement created successfully", entitlement))
}

// UpdateSystemEntitlement handles PUT /api/systems/:id/entitlements/:entitlement[?scope=]
func UpdateSystemEntitlement(c *gin.Context) {
	system, _, ok := entitlementAccessCheck(c, true)
	if !ok {
		return
	}
	systemID := system.ID
	entitlementID := c.Param("entitlement")
	scope := c.Query("scope")

	var req models.UpdateSystemEntitlementRequest
	if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
		return
	}

	setValidUntil := req.ClearValidUntil || req.ValidUntil != nil
	validUntil := req.ValidUntil
	if req.ClearValidUntil {
		validUntil = nil
	}

	repo := entities.NewLocalSystemEntitlementRepository()
	entitlement, err := repo.Update(systemID, entitlementID, scope, setValidUntil, validUntil, req.Revoked, models.EntitlementSourceManual)
	if err == entities.ErrEntitlementNotFound {
		c.JSON(http.StatusNotFound, response.NotFound("entitlement not found for this system", nil))
		return
	}
	if err != nil {
		logger.RequestLogger(c, "entitlements").Error().Err(err).
			Str("system_id", systemID).
			Str("entitlement", entitlementID).
			Msg("Failed to update entitlement")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to update entitlement", nil))
		return
	}

	logger.RequestLogger(c, "entitlements").Info().
		Str("operation", "update_entitlement").
		Str("system_id", systemID).
		Str("entitlement", entitlementID).
		Str("scope", scope).
		Msg("Entitlement updated")

	entitlement.Status = models.EntitlementStatus(entitlement.Active, entitlement.RevokedAt, isSystemBlocked(system), entitlement.PendingRef)
	c.JSON(http.StatusOK, response.OK("entitlement updated successfully", entitlement))
}

// DeleteSystemEntitlement handles DELETE /api/systems/:id/entitlements/:entitlement[?scope=]
// It revokes the grant (sets revoked_at) keeping the row for audit; idempotent.
func DeleteSystemEntitlement(c *gin.Context) {
	system, _, ok := entitlementAccessCheck(c, true)
	if !ok {
		return
	}
	systemID := system.ID
	entitlementID := c.Param("entitlement")
	scope := c.Query("scope")

	repo := entities.NewLocalSystemEntitlementRepository()
	entitlement, err := repo.Revoke(systemID, entitlementID, scope, models.EntitlementSourceManual)
	if err == entities.ErrEntitlementNotFound {
		c.JSON(http.StatusNotFound, response.NotFound("entitlement not found for this system", nil))
		return
	}
	if err != nil {
		logger.RequestLogger(c, "entitlements").Error().Err(err).
			Str("system_id", systemID).
			Str("entitlement", entitlementID).
			Msg("Failed to revoke entitlement")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to revoke entitlement", nil))
		return
	}

	logger.RequestLogger(c, "entitlements").Info().
		Str("operation", "revoke_entitlement").
		Str("system_id", systemID).
		Str("entitlement", entitlementID).
		Str("scope", scope).
		Msg("Entitlement revoked")

	entitlement.Status = models.EntitlementStatus(entitlement.Active, entitlement.RevokedAt, isSystemBlocked(system), entitlement.PendingRef)
	c.JSON(http.StatusOK, response.OK("entitlement revoked successfully", entitlement))
}
