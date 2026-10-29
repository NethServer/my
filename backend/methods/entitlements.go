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
// organization (Owner and Staff users — Nethesis).
func isEntitlementAdmin(u *models.User) bool {
	return IsOwnerOrgMember(u)
}

// canTransactEntitlements returns true for the TRANSACTIONAL surface — buy
// on the shop / cancel a subscription (activate/deactivate/pending). Its
// callers are the shop, server-to-server with the owner API key, and Nethesis
// staff: both live in the Owner organization, so membership there is required
// on top of manage:entitlements. The permission alone is not a gate: partner
// Admin and Backoffice roles carry it for the shop's purchase check, and a
// direct call from them would mint a grant nobody paid for.
func canTransactEntitlements(u *models.User) bool {
	if !IsOwnerOrgMember(u) {
		return false
	}
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
		c.JSON(http.StatusForbidden, response.Forbidden("only the Owner organization can manage grants directly", nil))
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

// maxSeats bounds the licenses of one add-on on one system: far above any
// shop tier, low enough to keep a typo from becoming a database error.
const maxSeats = 10000

// usageMetricError checks the usage metric of a catalog item: a known one,
// on an add-on without seats (seats are counted by their assignments), and
// the instances of an application only for an add-on that names it.
func usageMetricError(metric string, seated bool, appliesTo string) string {
	switch metric {
	case "":
		return ""
	case models.EntitlementUsageArpMacs, models.EntitlementUsageInstances:
	default:
		return "usage_metric must be one of: arp_macs, instances"
	}
	if seated {
		return "a seated add-on counts its seats, not a usage metric"
	}
	if metric == models.EntitlementUsageInstances && appliesTo == "" {
		return "the instances metric needs the application the add-on applies to"
	}
	return ""
}

// hasQuantity tells whether the grants of a catalog item carry a number:
// the seats of a seated add-on, or the tier limit of one with a usage metric.
func hasQuantity(item *models.EntitlementCatalogItem) bool {
	return item.Seated || item.UsageMetric != ""
}

// seatsFor validates the seats sent for a catalog item: required and positive
// for seated add-ons when required is set, positive when sent, and dropped for
// add-ons carrying no number. For an add-on with a usage metric they are its
// tier limit, never required: no limit is a tier too (an NS8 cluster). The
// message is empty when the value is acceptable.
func seatsFor(item *models.EntitlementCatalogItem, seats *int, required bool) (*int, string) {
	if !hasQuantity(item) {
		return nil, ""
	}
	if seats == nil {
		if required && item.Seated {
			return nil, "seats required for this add-on"
		}
		return nil, ""
	}
	if *seats <= 0 {
		return nil, "seats must be greater than zero"
	}
	if *seats > maxSeats {
		return nil, "seats must be at most " + strconv.Itoa(maxSeats)
	}
	return seats, ""
}

// resolvePurchaser builds the purchased_by audit snapshot from the order's
// customer email sent by the shop webhook (server-to-server, trusted). The
// email is resolved to a my user and snapshotted in full — identity, org and
// roles AT PURCHASE TIME, robust to later renames/moves like the other
// created_by/on_behalf_of snapshots. An address matching no my user is kept
// raw ({email} only); no email at all (stamped legacy order) → nil.
func resolvePurchaser(buyerEmail string) map[string]interface{} {
	if buyerEmail == "" {
		return nil
	}

	u, err := entities.NewLocalUserRepository().GetByEmail(buyerEmail)
	if err != nil {
		return map[string]interface{}{"email": buyerEmail}
	}

	snap := map[string]interface{}{
		"name":  u.Name,
		"email": u.Email,
	}
	if u.LogtoID != nil {
		snap["logto_id"] = *u.LogtoID
	}
	if u.Organization != nil {
		snap["organization_id"] = u.Organization.LogtoID
		snap["organization_name"] = u.Organization.Name
		snap["org_role"] = u.Organization.Type
	}
	roles := make([]string, 0, len(u.Roles))
	for _, role := range u.Roles {
		if role.Name != "" {
			roles = append(roles, role.Name)
		}
	}
	if len(roles) > 0 {
		snap["user_roles"] = roles
	}
	return snap
}

// redactPurchaser strips the buyer identity when the buyer's organization is
// outside the viewer's hierarchy (orgScope nil = no restriction, Owner-org
// Admin): a reseller must not learn who sits above them — the UI renders a
// generic "purchased by another organization" from the bare marker. Raw
// email-only snapshots have no organization and are admin-only too.
// buyerOutsideHierarchy reports whether the my user who bought a shop order
// cannot reach the system it activates: an order must not land on somebody
// else's system, where cancelling or refunding it would revoke an add-on that
// system's owner paid for. A buyer matching no my user (an old shop account,
// an imported order) cannot be placed and is let through; the Owner
// organization reaches every system.
func buyerOutsideHierarchy(purchaser map[string]interface{}, systemOrgID string) (bool, error) {
	orgID, _ := purchaser["organization_id"].(string)
	orgRole, _ := purchaser["org_role"].(string)
	if orgID == "" || strings.EqualFold(orgRole, "owner") {
		return false, nil
	}
	allowed, err := local.NewUserService().GetHierarchicalOrganizationIDs(strings.ToLower(orgRole), orgID)
	if err != nil {
		return false, err
	}
	return !slices.Contains(allowed, systemOrgID), nil
}

// checkBuyer refuses a shop call whose buyer cannot reach the system; it
// answers the request itself and returns false when it refuses or fails.
func checkBuyer(c *gin.Context, repo *entities.LocalSystemEntitlementRepository, systemID string, purchaser map[string]interface{}) bool {
	if purchaser == nil {
		return true
	}
	systemOrgID, err := repo.SystemOrganizationID(systemID)
	if err == nil {
		var outside bool
		outside, err = buyerOutsideHierarchy(purchaser, systemOrgID)
		if err == nil && outside {
			logger.RequestLogger(c, "entitlements").Warn().
				Str("system_id", systemID).
				Interface("buyer", purchaser["email"]).
				Msg("Shop order refused: the buyer cannot reach this system")
			c.JSON(http.StatusForbidden, response.Forbidden("the buyer of this order cannot purchase add-ons for this system", nil))
			return false
		}
	}
	if err != nil {
		logger.RequestLogger(c, "entitlements").Error().Err(err).Str("system_id", systemID).Msg("Failed to check the buyer")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to check the buyer", nil))
		return false
	}
	return true
}

// redactSeatAssigners hides who gave each license when that user sits outside
// the viewer's hierarchy, the same rule the purchaser follows: a customer sees
// that a license is assigned, not the name of the distributor or Nethesis user
// that assigned it.
func redactSeatAssigners(e *models.SystemEntitlement, orgScope []string) {
	if orgScope == nil {
		return
	}
	for i := range e.SeatAssignments {
		orgID, _ := e.SeatAssignments[i].AssignedBy["organization_id"].(string)
		if orgID == "" || !slices.Contains(orgScope, orgID) {
			e.SeatAssignments[i].AssignedBy = map[string]interface{}{"out_of_scope": true}
		}
	}
}

// redactUsage drops the measured usage for everyone outside Nethesis: going
// past a tier limit is followed up by Nethesis sales.
func redactUsage(e *models.SystemEntitlement, user *models.User) {
	if !IsOwnerOrgMember(user) {
		e.Usage, e.UsageAt = nil, nil
	}
}

func redactPurchaser(e *models.SystemEntitlement, orgScope []string) {
	if e.PurchasedBy == nil || orgScope == nil {
		return
	}
	orgID, _ := e.PurchasedBy["organization_id"].(string)
	if orgID == "" || !slices.Contains(orgScope, orgID) {
		e.PurchasedBy = map[string]interface{}{"out_of_scope": true}
	}
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
// organization): the catalog and its availability rules are Nethesis
// product management.
func catalogWriteGate(c *gin.Context) (*models.User, bool) {
	u, found := helpers.GetUserFromContext(c)
	if !found {
		return nil, false
	}
	if !isEntitlementAdmin(u) {
		c.JSON(http.StatusForbidden, response.Forbidden("only the Owner organization can manage the entitlement catalog", nil))
		return nil, false
	}
	return u, true
}

// transactGate rejects callers outside the Owner organization or without the
// manage:entitlements permission (buy/cancel surface: shop owner key, Staff,
// Owner).
func transactGate(c *gin.Context) (*models.User, bool) {
	u, found := helpers.GetUserFromContext(c)
	if !found {
		return nil, false
	}
	if !canTransactEntitlements(u) {
		c.JSON(http.StatusForbidden, response.Forbidden("only the Owner organization can activate or deactivate entitlements", nil))
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

	// The application a module belongs to is explicit; clients that predate
	// it fall back to the id prefix — the rule the column replaces — which is
	// exact except for hyphenated app names (nethvoice-proxy), the very case
	// the UI now sends applies_to for.
	if req.Kind == models.EntitlementKindModule {
		if req.AppliesTo == "" {
			req.AppliesTo, _, _ = strings.Cut(req.ID, "-")
		}
	} else {
		req.AppliesTo = ""
	}

	// Add-ons are bought for the whole system. On a cluster an add-on reaches
	// single application instances through seats, never by being narrowed to
	// one instance.
	if req.Scoped {
		c.JSON(http.StatusBadRequest, response.BadRequest("add-ons cannot be narrowed to one application: sell them by licenses", nil))
		return
	}
	if req.Seated && (req.Kind != models.EntitlementKindModule || req.SystemType != "ns8") {
		c.JSON(http.StatusBadRequest, response.BadRequest("a seated add-on must be an ns8 module", nil))
		return
	}
	if msg := usageMetricError(req.UsageMetric, req.Seated, req.AppliesTo); msg != "" {
		c.JSON(http.StatusBadRequest, response.BadRequest(msg, nil))
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
		Bool("seated", req.Seated).
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
	if req.UsageMetric != nil {
		current, err := repo.Get(c.Param("id"))
		if err == entities.ErrCatalogItemNotFound {
			c.JSON(http.StatusNotFound, response.NotFound("catalog item not found", nil))
			return
		}
		if err != nil {
			logger.RequestLogger(c, "entitlements").Error().Err(err).Str("catalog_id", c.Param("id")).Msg("Failed to read catalog item")
			c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to read catalog item", nil))
			return
		}
		if msg := usageMetricError(*req.UsageMetric, current.Seated, current.AppliesTo); msg != "" {
			c.JSON(http.StatusBadRequest, response.BadRequest(msg, nil))
			return
		}
	}
	item, err := repo.Update(c.Param("id"), req.DisplayName, req.Description, req.Purchasable, req.UsageMetric)
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
		"user_id":           actorLogtoID(user),
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
// my UI and, in fase 3, the shop). Owner-organization users see the whole
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
	seats, msg := seatsFor(item, req.Seats, true)
	if msg != "" {
		c.JSON(http.StatusBadRequest, response.BadRequest(msg, nil))
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

	purchaser := resolvePurchaser(req.BuyerEmail)
	if !checkBuyer(c, repo, systemID, purchaser) {
		return
	}

	createdBy := map[string]interface{}{
		"user_id":           actorLogtoID(user),
		"user_name":         user.Name,
		"organization_id":   user.OrganizationID,
		"organization_name": user.OrganizationName,
		"channel":           "shop",
	}

	// The tier of a measured add-on sets its limit, a tier without one (an
	// NS8 cluster) included; an activation that names no tier keeps it.
	limitFromTier := item.UsageMetric != "" && req.Variant != nil
	grant, err := repo.Upsert(systemID, item.ID, req.Scope, models.EntitlementSourceShop, req.SourceRef, req.ValidUntil, createdBy, purchaser, req.Variant, seats, limitFromTier)
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
	seats, msg := seatsFor(item, req.Seats, false)
	if msg != "" {
		c.JSON(http.StatusBadRequest, response.BadRequest(msg, nil))
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

	purchaser := resolvePurchaser(req.BuyerEmail)
	if !checkBuyer(c, repo, systemID, purchaser) {
		return
	}

	createdBy := map[string]interface{}{
		"user_id":           actorLogtoID(user),
		"user_name":         user.Name,
		"organization_id":   user.OrganizationID,
		"organization_name": user.OrganizationName,
		"channel":           "shop",
	}

	grant, err := repo.MarkPending(systemID, item.ID, req.Scope, req.SourceRef, createdBy, purchaser, req.Variant, seats)
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
// restriction) for the Owner organization, the caller's hierarchy
// otherwise — buyers see their own modules/expirations, owner sees the fleet.
func grantsOrgScope(u *models.User) ([]string, error) {
	if isEntitlementAdmin(u) {
		return nil, nil
	}
	return local.NewUserService().GetHierarchicalOrganizationIDs(u.OrgRole, u.OrganizationID)
}

// grantsFilter reads the grants report filters of the request, within the
// caller's visibility. The over-limit view is Nethesis only.
func grantsFilter(c *gin.Context) (*models.User, []string, entities.GrantsReportFilter, bool) {
	u, found := helpers.GetUserFromContext(c)
	if !found {
		return nil, nil, entities.GrantsReportFilter{}, false
	}

	scope, err := grantsOrgScope(u)
	if err != nil {
		logger.RequestLogger(c, "entitlements").Error().Err(err).Msg("Failed to resolve org scope")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to resolve organization scope", nil))
		return nil, nil, entities.GrantsReportFilter{}, false
	}

	filter := entities.GrantsReportFilter{
		Entitlement:    c.Query("entitlement"),
		OrganizationID: c.Query("organization_id"),
		Source:         c.Query("source"),
		ActiveOnly:     c.Query("active") == "true",
		OrgScope:       scope,
		OverLimit:      c.Query("over_limit") == "true" && IsOwnerOrgMember(u),
		Search:         strings.TrimSpace(c.Query("search")),
	}
	if v := c.Query("expiring_before"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			c.JSON(http.StatusBadRequest, response.BadRequest("expiring_before must be RFC3339", nil))
			return nil, nil, entities.GrantsReportFilter{}, false
		}
		filter.ExpiringBefore = &t
	}
	return u, scope, filter, true
}

// GetEntitlementGrants handles GET /api/entitlements/grants — fleet-wide (or
// hierarchy-wide) grants report with filters: entitlement, organization_id,
// source, active=true, expiring_before=RFC3339, page/page_size.
func GetEntitlementGrants(c *gin.Context) {
	u, scope, filter, ok := grantsFilter(c)
	if !ok {
		return
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

	// Buyer identity is only shown within the viewer's hierarchy.
	for _, row := range rows {
		redactPurchaser(&row.SystemEntitlement, scope)
		redactUsage(&row.SystemEntitlement, u)
	}

	c.JSON(http.StatusOK, response.OK("grants retrieved successfully", gin.H{
		"grants":    rows,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}))
}

// GetEntitlementReport handles GET /api/entitlements/report — the add-on
// analytics (lifecycle totals, per-type breakdown, renewal distribution,
// 12-month activation trend) within the caller's visibility: fleet-wide for
// the Owner organization, own hierarchy for everyone else. Same scope
// as /grants and /stats — the report is their aggregate, no wider.
func GetEntitlementReport(c *gin.Context) {
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

	// Add-ons nobody holds belong in the breakdown, but only the ones the
	// caller is entitled to see: the whole catalog for the Owner organization
	// Admins, whatever its own organization may buy for everyone else. The
	// counts themselves stay bound to the hierarchy scope either way.
	var catalogScope []string
	if !isEntitlementAdmin(u) {
		items, err := entities.NewLocalEntitlementAvailabilityRepository().ListAvailableFor(u.OrgRole, u.OrganizationID)
		if err != nil {
			logger.RequestLogger(c, "entitlements").Error().Err(err).Msg("Failed to resolve catalog visibility")
			c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to resolve catalog visibility", nil))
			return
		}
		catalogScope = make([]string, 0, len(items))
		for _, item := range items {
			catalogScope = append(catalogScope, item.ID)
		}
	}

	repo := entities.NewLocalSystemEntitlementRepository()
	report, err := repo.Report(scope, catalogScope)
	if err != nil {
		logger.RequestLogger(c, "entitlements").Error().Err(err).Msg("Failed to build entitlement report")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to build entitlement report", nil))
		return
	}
	if !IsOwnerOrgMember(u) {
		report.Totals.OverLimit = nil
	}

	c.JSON(http.StatusOK, response.OK("entitlement report retrieved successfully", report))
}

// reportQuery resolves the caller's org visibility (nil = fleet-wide) and
// parses the standard search/page/page_size query of the paginated report
// slices.
func reportQuery(c *gin.Context) (scope []string, search string, page, pageSize int, ok bool) {
	u, found := helpers.GetUserFromContext(c)
	if !found {
		return nil, "", 0, 0, false
	}

	scope, err := grantsOrgScope(u)
	if err != nil {
		logger.RequestLogger(c, "entitlements").Error().Err(err).Msg("Failed to resolve org scope")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to resolve organization scope", nil))
		return nil, "", 0, 0, false
	}

	search = c.Query("search")
	page, pageSize = 1, 10
	if v, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil && v > 0 {
		page = v
	}
	if v, err := strconv.Atoi(c.DefaultQuery("page_size", "10")); err == nil && v > 0 && v <= 200 {
		pageSize = v
	}
	return scope, search, page, pageSize, true
}

// GetEntitlementReportOrganizations handles GET /api/entitlements/report/organizations
// — the paginated + searchable per-organization slice of the add-on report,
// within the caller's hierarchy.
func GetEntitlementReportOrganizations(c *gin.Context) {
	scope, search, page, pageSize, ok := reportQuery(c)
	if !ok {
		return
	}

	repo := entities.NewLocalSystemEntitlementRepository()
	rows, total, err := repo.ReportOrganizations(search, scope, pageSize, (page-1)*pageSize)
	if err != nil {
		logger.RequestLogger(c, "entitlements").Error().Err(err).Msg("Failed to build per-organization report")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to build per-organization report", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("report organizations retrieved successfully", gin.H{
		"organizations": rows,
		"total":         total,
		"page":          page,
		"page_size":     pageSize,
	}))
}

// GetEntitlementReportTiers handles GET /api/entitlements/report/tiers — the
// paginated + searchable per-tier slice of the add-on report, within the
// caller's hierarchy.
func GetEntitlementReportTiers(c *gin.Context) {
	scope, search, page, pageSize, ok := reportQuery(c)
	if !ok {
		return
	}

	repo := entities.NewLocalSystemEntitlementRepository()
	rows, total, err := repo.ReportVariants(search, scope, pageSize, (page-1)*pageSize)
	if err != nil {
		logger.RequestLogger(c, "entitlements").Error().Err(err).Msg("Failed to build per-tier report")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to build per-tier report", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("report tiers retrieved successfully", gin.H{
		"tiers":     rows,
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
	system, user, ok := entitlementAccessCheck(c, false)
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

	// Buyer identity is only shown within the viewer's hierarchy.
	scope, err := grantsOrgScope(user)
	if err != nil {
		logger.RequestLogger(c, "entitlements").Error().Err(err).Msg("Failed to resolve org scope")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to resolve organization scope", nil))
		return
	}
	for _, e := range list {
		redactPurchaser(e, scope)
		redactUsage(e, user)
	}

	var seatedGrants []string
	for _, e := range list {
		if e.Seats != nil {
			seatedGrants = append(seatedGrants, e.ID)
		}
	}
	seats, err := entities.NewLocalEntitlementSeatRepository().ListByGrants(seatedGrants)
	if err != nil {
		logger.RequestLogger(c, "entitlements").Error().Err(err).
			Str("system_id", systemID).
			Msg("Failed to list seats")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to list entitlements", nil))
		return
	}
	for _, e := range list {
		if e.Seats != nil {
			e.SeatAssignments = seats[e.ID]
			if e.SeatAssignments == nil {
				e.SeatAssignments = []models.EntitlementSeat{}
			}
			redactSeatAssigners(e, scope)
		}
	}

	c.JSON(http.StatusOK, response.OK("entitlements retrieved successfully", gin.H{
		"entitlements": list,
	}))
}

// seatGrant resolves the seated add-on named in the path and its grant on
// the system. Whoever can buy add-ons (manage:entitlements, checked on the
// route) hands out the seats, within their own hierarchy.
func seatGrant(c *gin.Context) (*models.System, *models.User, *models.EntitlementCatalogItem, *models.SystemEntitlement, bool) {
	system, user, ok := entitlementAccessCheck(c, false)
	if !ok {
		return nil, nil, nil, nil, false
	}

	item, err := entities.NewLocalEntitlementCatalogRepository().Resolve(c.Param("entitlement"))
	if err == entities.ErrCatalogItemNotFound {
		c.JSON(http.StatusNotFound, response.NotFound("entitlement not found for this system", nil))
		return nil, nil, nil, nil, false
	}
	if err != nil {
		logger.RequestLogger(c, "entitlements").Error().Err(err).Msg("Failed to resolve entitlement")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to resolve entitlement", nil))
		return nil, nil, nil, nil, false
	}
	if !item.Seated {
		c.JSON(http.StatusBadRequest, response.BadRequest("this entitlement has no seats", nil))
		return nil, nil, nil, nil, false
	}

	grant, err := entities.NewLocalSystemEntitlementRepository().Get(system.ID, item.ID, "")
	if err == entities.ErrEntitlementNotFound {
		c.JSON(http.StatusNotFound, response.NotFound("entitlement not found for this system", nil))
		return nil, nil, nil, nil, false
	}
	if err != nil {
		logger.RequestLogger(c, "entitlements").Error().Err(err).Msg("Failed to read entitlement")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to read entitlement", nil))
		return nil, nil, nil, nil, false
	}

	return system, user, item, grant, true
}

// actorLogtoID is the id the actor snapshots carry: the Logto one, as the
// systems' created_by does, because the UI loads the avatar by it. The local
// id only stands in for a user not yet synced to the database.
func actorLogtoID(u *models.User) string {
	if u.LogtoID != nil && *u.LogtoID != "" {
		return *u.LogtoID
	}
	return u.ID
}

func seatActor(u *models.User) map[string]interface{} {
	return map[string]interface{}{
		"user_id":           actorLogtoID(u),
		"user_name":         u.Name,
		"organization_id":   u.OrganizationID,
		"organization_name": u.OrganizationName,
	}
}

// AssignEntitlementSeat handles POST /api/systems/:id/entitlements/:entitlement/seats
func AssignEntitlementSeat(c *gin.Context) {
	system, user, item, grant, ok := seatGrant(c)
	if !ok {
		return
	}

	var req models.AssignEntitlementSeatRequest
	if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
		return
	}

	if isSystemBlocked(system) {
		c.JSON(http.StatusConflict, response.Conflict("the system is suspended or deleted", nil))
		return
	}
	if !grant.Active {
		c.JSON(http.StatusConflict, response.Conflict("the add-on is not active on this system", nil))
		return
	}

	seat, err := entities.NewLocalEntitlementSeatRepository().Assign(grant.ID, system.ID, item.AppliesTo, req.ModuleID, seatActor(user))
	switch err {
	case nil:
	case entities.ErrInstanceNotFound:
		c.JSON(http.StatusBadRequest, response.BadRequest("no "+item.AppliesTo+" application "+req.ModuleID+" on this system", nil))
		return
	case entities.ErrSeatAlreadyHeld:
		c.JSON(http.StatusConflict, response.Conflict("the application already holds a license", nil))
		return
	case entities.ErrNoFreeSeat:
		c.JSON(http.StatusConflict, response.Conflict("no free license left: free one or buy more", nil))
		return
	default:
		logger.RequestLogger(c, "entitlements").Error().Err(err).
			Str("system_id", system.ID).
			Str("entitlement", item.ID).
			Str("module_id", req.ModuleID).
			Msg("Failed to assign seat")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to assign seat", nil))
		return
	}

	logger.RequestLogger(c, "entitlements").Info().
		Str("operation", "assign_entitlement_seat").
		Str("system_id", system.ID).
		Str("entitlement", item.ID).
		Str("module_id", req.ModuleID).
		Msg("Entitlement seat assigned")

	c.JSON(http.StatusCreated, response.Created("seat assigned successfully", seat))
}

// ReleaseEntitlementSeat handles DELETE /api/systems/:id/entitlements/:entitlement/seats/:module_id
func ReleaseEntitlementSeat(c *gin.Context) {
	system, user, item, grant, ok := seatGrant(c)
	if !ok {
		return
	}
	moduleID := c.Param("module_id")

	err := entities.NewLocalEntitlementSeatRepository().Release(grant.ID, moduleID, seatActor(user))
	if err == entities.ErrSeatNotFound {
		c.JSON(http.StatusNotFound, response.NotFound("the application holds no license", nil))
		return
	}
	if err != nil {
		logger.RequestLogger(c, "entitlements").Error().Err(err).
			Str("system_id", system.ID).
			Str("entitlement", item.ID).
			Str("module_id", moduleID).
			Msg("Failed to release seat")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to release seat", nil))
		return
	}

	logger.RequestLogger(c, "entitlements").Info().
		Str("operation", "release_entitlement_seat").
		Str("system_id", system.ID).
		Str("entitlement", item.ID).
		Str("module_id", moduleID).
		Msg("Entitlement seat released")

	c.JSON(http.StatusOK, response.OK("seat released successfully", nil))
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
	if req.Seats != nil && !hasQuantity(item) {
		c.JSON(http.StatusBadRequest, response.BadRequest("this entitlement has no seats", nil))
		return
	}
	seats, msg := seatsFor(item, req.Seats, true)
	if msg != "" {
		c.JSON(http.StatusBadRequest, response.BadRequest(msg, nil))
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
		"user_id":           actorLogtoID(user),
		"user_name":         user.Name,
		"organization_id":   user.OrganizationID,
		"organization_name": user.OrganizationName,
	}

	repo := entities.NewLocalSystemEntitlementRepository()
	entitlement, err := repo.Create(systemID, req.Entitlement, req.Scope, source, req.SourceRef, req.ValidFrom, req.ValidUntil, createdBy, resolvePurchaser(req.BuyerEmail), req.Variant, seats)
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

	if req.Seats != nil {
		item, err := entities.NewLocalEntitlementCatalogRepository().Get(entitlementID)
		if err == entities.ErrCatalogItemNotFound {
			c.JSON(http.StatusNotFound, response.NotFound("entitlement not found for this system", nil))
			return
		}
		if err != nil {
			logger.RequestLogger(c, "entitlements").Error().Err(err).Msg("Failed to read entitlement catalog")
			c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to read entitlement catalog", nil))
			return
		}
		if !hasQuantity(item) {
			c.JSON(http.StatusBadRequest, response.BadRequest("this entitlement has no seats", nil))
			return
		}
		if _, msg := seatsFor(item, req.Seats, true); msg != "" {
			c.JSON(http.StatusBadRequest, response.BadRequest(msg, nil))
			return
		}
	}

	repo := entities.NewLocalSystemEntitlementRepository()
	entitlement, err := repo.Update(systemID, entitlementID, scope, setValidUntil, validUntil, req.Revoked, models.EntitlementSourceManual, req.Seats)
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
