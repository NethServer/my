/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package entities

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lib/pq"

	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/models"
)

// ErrEntitlementExists is returned by Create when the (system, entitlement,
// scope) tuple already has a row — renewals go through Update, never
// duplicate rows.
var ErrEntitlementExists = fmt.Errorf("entitlement already exists for this system")

// ErrEntitlementNotFound is returned when the (system, entitlement, scope)
// tuple has no row.
var ErrEntitlementNotFound = fmt.Errorf("entitlement not found for this system")

// ErrCatalogItemExists / ErrCatalogItemNotFound / ErrCatalogItemInUse are the
// catalog counterparts.
var ErrCatalogItemExists = fmt.Errorf("catalog item already exists")
var ErrCatalogItemNotFound = fmt.Errorf("catalog item not found")
var ErrCatalogItemInUse = fmt.Errorf("catalog item is referenced by existing grants")

// LocalEntitlementCatalogRepository reads / writes entitlement_catalog.
type LocalEntitlementCatalogRepository struct {
	db *sql.DB
}

func NewLocalEntitlementCatalogRepository() *LocalEntitlementCatalogRepository {
	return &LocalEntitlementCatalogRepository{db: database.DB}
}

func scanCatalogItem(scanner interface{ Scan(...interface{}) error }) (*models.EntitlementCatalogItem, error) {
	var item models.EntitlementCatalogItem
	err := scanner.Scan(&item.ID, &item.DisplayName, &item.Description, &item.Scoped, &item.Kind, &item.SystemType, &item.LegacyAlias, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// List returns the whole catalog ordered by id.
func (r *LocalEntitlementCatalogRepository) List() ([]*models.EntitlementCatalogItem, error) {
	rows, err := r.db.Query(
		`SELECT id, display_name, description, scoped, kind, system_type, legacy_alias, created_at, updated_at
		 FROM entitlement_catalog ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("failed to list entitlement catalog: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := []*models.EntitlementCatalogItem{}
	for rows.Next() {
		item, err := scanCatalogItem(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan catalog item: %w", err)
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// Get returns one catalog item by id.
func (r *LocalEntitlementCatalogRepository) Get(id string) (*models.EntitlementCatalogItem, error) {
	item, err := scanCatalogItem(r.db.QueryRow(
		`SELECT id, display_name, description, scoped, kind, system_type, legacy_alias, created_at, updated_at
		 FROM entitlement_catalog WHERE id = $1`, id))
	if err == sql.ErrNoRows {
		return nil, ErrCatalogItemNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get catalog item: %w", err)
	}
	return item, nil
}

// Resolve returns the catalog item matching the given id, accepting either
// the canonical id or the legacy wire alias (e.g. ng-blacklist for
// nsec-blacklist).
func (r *LocalEntitlementCatalogRepository) Resolve(idOrAlias string) (*models.EntitlementCatalogItem, error) {
	item, err := scanCatalogItem(r.db.QueryRow(
		`SELECT id, display_name, description, scoped, kind, system_type, legacy_alias, created_at, updated_at
		 FROM entitlement_catalog WHERE id = $1 OR (legacy_alias <> '' AND legacy_alias = $1)`, idOrAlias))
	if err == sql.ErrNoRows {
		return nil, ErrCatalogItemNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to resolve catalog item: %w", err)
	}
	return item, nil
}

// Create adds a new add-on type.
func (r *LocalEntitlementCatalogRepository) Create(req *models.CreateEntitlementCatalogRequest) (*models.EntitlementCatalogItem, error) {
	item, err := scanCatalogItem(r.db.QueryRow(
		`INSERT INTO entitlement_catalog (id, display_name, description, scoped, kind, system_type, legacy_alias)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, display_name, description, scoped, kind, system_type, legacy_alias, created_at, updated_at`,
		req.ID, req.DisplayName, req.Description, req.Scoped, req.Kind, req.SystemType, req.LegacyAlias))
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return nil, ErrCatalogItemExists
		}
		return nil, fmt.Errorf("failed to create catalog item: %w", err)
	}
	return item, nil
}

// Update changes the display fields of a catalog item (id and scoped are
// immutable).
func (r *LocalEntitlementCatalogRepository) Update(id string, displayName, description *string) (*models.EntitlementCatalogItem, error) {
	item, err := scanCatalogItem(r.db.QueryRow(
		`UPDATE entitlement_catalog SET
		     display_name = COALESCE($2, display_name),
		     description  = COALESCE($3, description),
		     updated_at   = NOW()
		 WHERE id = $1
		 RETURNING id, display_name, description, scoped, kind, system_type, legacy_alias, created_at, updated_at`,
		id, displayName, description))
	if err == sql.ErrNoRows {
		return nil, ErrCatalogItemNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update catalog item: %w", err)
	}
	return item, nil
}

// Delete removes a catalog item; refused while grants reference it (FK).
func (r *LocalEntitlementCatalogRepository) Delete(id string) error {
	res, err := r.db.Exec(`DELETE FROM entitlement_catalog WHERE id = $1`, id)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23503" {
			return ErrCatalogItemInUse
		}
		return fmt.Errorf("failed to delete catalog item: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrCatalogItemNotFound
	}
	return nil
}

// ErrAvailabilityExists / ErrAvailabilityNotFound are the availability
// counterparts.
var ErrAvailabilityExists = fmt.Errorf("availability rule already exists")
var ErrAvailabilityNotFound = fmt.Errorf("availability rule not found")

// LocalEntitlementAvailabilityRepository reads / writes
// entitlement_availability (commercial unlocks).
type LocalEntitlementAvailabilityRepository struct {
	db *sql.DB
}

func NewLocalEntitlementAvailabilityRepository() *LocalEntitlementAvailabilityRepository {
	return &LocalEntitlementAvailabilityRepository{db: database.DB}
}

func scanAvailability(scanner interface{ Scan(...interface{}) error }) (*models.EntitlementAvailability, error) {
	var a models.EntitlementAvailability
	var createdBy []byte
	err := scanner.Scan(&a.ID, &a.Entitlement, &a.OrgRole, &a.OrganizationID, &createdBy, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	if len(createdBy) > 0 {
		_ = json.Unmarshal(createdBy, &a.CreatedBy)
	}
	return &a, nil
}

// ListByEntitlement returns the unlock rules of one catalog item.
func (r *LocalEntitlementAvailabilityRepository) ListByEntitlement(entitlement string) ([]*models.EntitlementAvailability, error) {
	rows, err := r.db.Query(
		`SELECT id, entitlement, org_role, organization_id, created_by, created_at
		 FROM entitlement_availability WHERE entitlement = $1
		 ORDER BY org_role, organization_id`, entitlement)
	if err != nil {
		return nil, fmt.Errorf("failed to list availability: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := []*models.EntitlementAvailability{}
	for rows.Next() {
		a, err := scanAvailability(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan availability: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ListAvailableFor returns the sellable catalog items (services and
// modules; apps are enablement-only) available to an organization. A newly
// created item is available to EVERYONE by default; availability rules, when
// present for an item, RESTRICT it to the matching role/orgs.
func (r *LocalEntitlementAvailabilityRepository) ListAvailableFor(orgRole, orgID string) ([]*models.EntitlementCatalogItem, error) {
	rows, err := r.db.Query(
		`SELECT c.id, c.display_name, c.description, c.scoped, c.kind, c.system_type, c.legacy_alias, c.created_at, c.updated_at
		 FROM entitlement_catalog c
		 WHERE c.kind IN ('service', 'module')
		   AND (NOT EXISTS (SELECT 1 FROM entitlement_availability a WHERE a.entitlement = c.id)
		        OR EXISTS (SELECT 1 FROM entitlement_availability a
		                   WHERE a.entitlement = c.id
		                     AND (a.org_role = LOWER($1) OR a.organization_id = $2)))
		 ORDER BY c.id`, orgRole, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list available entitlements: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := []*models.EntitlementCatalogItem{}
	for rows.Next() {
		item, err := scanCatalogItem(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan catalog item: %w", err)
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// Add creates one unlock rule (role-wide or org-specific).
func (r *LocalEntitlementAvailabilityRepository) Add(entitlement, orgRole, orgID string, createdBy map[string]interface{}) (*models.EntitlementAvailability, error) {
	var createdByJSON []byte
	if createdBy != nil {
		createdByJSON, _ = json.Marshal(createdBy)
	}
	a, err := scanAvailability(r.db.QueryRow(
		`INSERT INTO entitlement_availability (entitlement, org_role, organization_id, created_by)
		 VALUES ($1, LOWER($2), $3, $4)
		 RETURNING id, entitlement, org_role, organization_id, created_by, created_at`,
		entitlement, orgRole, orgID, createdByJSON))
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code {
			case "23505":
				return nil, ErrAvailabilityExists
			case "23503":
				return nil, ErrCatalogItemNotFound
			}
		}
		return nil, fmt.Errorf("failed to add availability: %w", err)
	}
	return a, nil
}

// Remove deletes one unlock rule by id (scoped to the entitlement for path
// consistency).
func (r *LocalEntitlementAvailabilityRepository) Remove(entitlement, ruleID string) error {
	res, err := r.db.Exec(
		`DELETE FROM entitlement_availability WHERE id = $1 AND entitlement = $2`, ruleID, entitlement)
	if err != nil {
		return fmt.Errorf("failed to remove availability: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrAvailabilityNotFound
	}
	return nil
}

// LocalSystemEntitlementRepository reads / writes system_entitlements.
type LocalSystemEntitlementRepository struct {
	db *sql.DB
}

func NewLocalSystemEntitlementRepository() *LocalSystemEntitlementRepository {
	return &LocalSystemEntitlementRepository{db: database.DB}
}

const entitlementColumns = `
	id, system_id, entitlement, scope, source, COALESCE(source_ref, ''),
	valid_from, valid_until, revoked_at, COALESCE(revoked_source, ''),
	COALESCE(pending_ref, ''), pending_since, created_by, purchased_by, variant, renewal_count, created_at, updated_at,
	(revoked_at IS NULL AND (valid_until IS NULL OR valid_until > NOW())) AS active`

func scanEntitlement(scanner interface{ Scan(...interface{}) error }) (*models.SystemEntitlement, error) {
	var e models.SystemEntitlement
	var validUntil, revokedAt, pendingSince sql.NullTime
	var createdBy, purchasedBy, variant []byte

	err := scanner.Scan(
		&e.ID, &e.SystemID, &e.Entitlement, &e.Scope, &e.Source, &e.SourceRef,
		&e.ValidFrom, &validUntil, &revokedAt, &e.RevokedSource,
		&e.PendingRef, &pendingSince, &createdBy, &purchasedBy, &variant, &e.RenewalCount, &e.CreatedAt, &e.UpdatedAt,
		&e.Active,
	)
	if err != nil {
		return nil, err
	}

	if validUntil.Valid {
		t := validUntil.Time
		e.ValidUntil = &t
	}
	if revokedAt.Valid {
		t := revokedAt.Time
		e.RevokedAt = &t
	}
	if pendingSince.Valid {
		t := pendingSince.Time
		e.PendingSince = &t
	}
	if len(createdBy) > 0 {
		_ = json.Unmarshal(createdBy, &e.CreatedBy)
	}
	if len(purchasedBy) > 0 {
		_ = json.Unmarshal(purchasedBy, &e.PurchasedBy)
	}
	if len(variant) > 0 {
		_ = json.Unmarshal(variant, &e.Variant)
	}

	// Grant-level status; callers that know the owning system's lifecycle
	// re-derive it with the suspension overlay (models.EntitlementStatus).
	e.Status = models.EntitlementStatus(e.Active, e.RevokedAt, false, e.PendingRef)

	return &e, nil
}

// ListBySystem returns every entitlement row (active or not) for a system.
func (r *LocalSystemEntitlementRepository) ListBySystem(systemID string) ([]*models.SystemEntitlement, error) {
	rows, err := r.db.Query(
		`SELECT `+entitlementColumns+`
		 FROM system_entitlements
		 WHERE system_id = $1
		 ORDER BY entitlement, scope`, systemID)
	if err != nil {
		return nil, fmt.Errorf("failed to list entitlements: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := []*models.SystemEntitlement{}
	for rows.Next() {
		e, err := scanEntitlement(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan entitlement: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// Get returns one entitlement row by (system, entitlement id, scope).
func (r *LocalSystemEntitlementRepository) Get(systemID, entitlement, scope string) (*models.SystemEntitlement, error) {
	e, err := scanEntitlement(r.db.QueryRow(
		`SELECT `+entitlementColumns+`
		 FROM system_entitlements
		 WHERE system_id = $1 AND entitlement = $2 AND scope = $3`, systemID, entitlement, scope))
	if err == sql.ErrNoRows {
		return nil, ErrEntitlementNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get entitlement: %w", err)
	}
	return e, nil
}

// Create grants an entitlement to a system. Duplicate (system, entitlement,
// scope) tuples return ErrEntitlementExists — renewals must Update the
// existing row.
func (r *LocalSystemEntitlementRepository) Create(systemID, entitlement, scope, source, sourceRef string, validFrom, validUntil *time.Time, createdBy map[string]interface{}) (*models.SystemEntitlement, error) {
	var createdByJSON []byte
	if createdBy != nil {
		createdByJSON, _ = json.Marshal(createdBy)
	}

	// validFrom NULL → DB default now(); set for legacy imports to preserve
	// the original order date.
	e, err := scanEntitlement(r.db.QueryRow(
		`INSERT INTO system_entitlements (system_id, entitlement, scope, source, source_ref, valid_from, valid_until, created_by)
		 VALUES ($1, $2, $3, $4, NULLIF($5, ''), COALESCE($6, now()), $7, $8)
		 RETURNING `+entitlementColumns,
		systemID, entitlement, scope, source, sourceRef, validFrom, validUntil, createdByJSON))
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code {
			case "23505":
				return nil, ErrEntitlementExists
			case "23503":
				// FK on entitlement_catalog — unknown entitlement id
				return nil, ErrCatalogItemNotFound
			}
		}
		return nil, fmt.Errorf("failed to create entitlement: %w", err)
	}
	return e, nil
}

// Update applies expiry and/or revocation changes to an existing row.
// setValidUntil distinguishes "leave valid_until alone" (false) from "set it
// to the given value, possibly NULL" (true). revokedSource records who is
// revoking (manual|shop): stamped only on the transition to revoked (an
// idempotent re-revoke keeps the original), cleared on unrevoke.
func (r *LocalSystemEntitlementRepository) Update(systemID, entitlement, scope string, setValidUntil bool, validUntil *time.Time, revoked *bool, revokedSource string) (*models.SystemEntitlement, error) {
	e, err := scanEntitlement(r.db.QueryRow(
		`UPDATE system_entitlements SET
		     valid_until = CASE WHEN $4 THEN $5 ELSE valid_until END,
		     revoked_source = CASE
		                       WHEN $6::boolean IS NULL THEN revoked_source
		                       WHEN $6 THEN CASE WHEN revoked_at IS NULL THEN NULLIF($7, '') ELSE revoked_source END
		                       ELSE NULL
		                   END,
		     revoked_at  = CASE
		                       WHEN $6::boolean IS NULL THEN revoked_at
		                       WHEN $6 THEN COALESCE(revoked_at, NOW())
		                       ELSE NULL
		                   END,
		     pending_ref   = CASE WHEN $6::boolean IS TRUE THEN NULL ELSE pending_ref END,
		     pending_since = CASE WHEN $6::boolean IS TRUE THEN NULL ELSE pending_since END,
		     updated_at  = NOW()
		 WHERE system_id = $1 AND entitlement = $2 AND scope = $3
		 RETURNING `+entitlementColumns,
		systemID, entitlement, scope, setValidUntil, validUntil, revoked, revokedSource))
	if err == sql.ErrNoRows {
		return nil, ErrEntitlementNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update entitlement: %w", err)
	}
	return e, nil
}

// Revoke marks the entitlement as revoked (idempotent, row kept for audit).
// source records who revoked: manual (admin) or shop (deactivate webhook).
func (r *LocalSystemEntitlementRepository) Revoke(systemID, entitlement, scope, source string) (*models.SystemEntitlement, error) {
	revoked := true
	return r.Update(systemID, entitlement, scope, false, nil, &revoked, source)
}

// Upsert creates the grant or, when the (system, entitlement, scope) row
// already exists, renews it in place: new expiry, revocation cleared,
// source/source_ref refreshed. This is the shop-webhook semantics —
// activation and renewal are the same idempotent call (safe on retries).
// purchasedBy (the buyer audit snapshot) and variant (the shop tier of the
// order line) replace the stored ones only when provided — an activation
// without them (stamped legacy order) must not wipe what a previous
// purchase recorded.
func (r *LocalSystemEntitlementRepository) Upsert(systemID, entitlement, scope, source, sourceRef string, validUntil *time.Time, createdBy, purchasedBy, variant map[string]interface{}) (*models.SystemEntitlement, error) {
	var createdByJSON []byte
	if createdBy != nil {
		createdByJSON, _ = json.Marshal(createdBy)
	}
	// interface{} nil → SQL NULL: lib/pq encodes a nil []byte as an empty
	// string, which jsonb rejects.
	var purchasedByJSON interface{}
	if purchasedBy != nil {
		purchasedByJSON, _ = json.Marshal(purchasedBy)
	}
	var variantJSON interface{}
	if variant != nil {
		variantJSON, _ = json.Marshal(variant)
	}

	e, err := scanEntitlement(r.db.QueryRow(
		`INSERT INTO system_entitlements (system_id, entitlement, scope, source, source_ref, valid_until, created_by, purchased_by, variant)
		 VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7, $8, $9)
		 ON CONFLICT (system_id, entitlement, scope) DO UPDATE SET
		     valid_until = EXCLUDED.valid_until,
		     valid_from  = CASE WHEN system_entitlements.valid_until = system_entitlements.valid_from
		                        THEN NOW() ELSE system_entitlements.valid_from END,
		     revoked_at  = NULL,
		     revoked_source = NULL,
		     pending_ref = NULL,
		     pending_since = NULL,
		     source      = EXCLUDED.source,
		     source_ref  = COALESCE(EXCLUDED.source_ref, system_entitlements.source_ref),
		     purchased_by = COALESCE(EXCLUDED.purchased_by, system_entitlements.purchased_by),
		     variant     = COALESCE(EXCLUDED.variant, system_entitlements.variant),
		     -- a renewal is an activation paid by a DIFFERENT order: webhook
		     -- retries on the same order never double-count
		     renewal_count = system_entitlements.renewal_count +
		                     CASE WHEN EXCLUDED.source_ref IS NOT NULL
		                               AND EXCLUDED.source_ref IS DISTINCT FROM system_entitlements.source_ref
		                          THEN 1 ELSE 0 END,
		     updated_at  = NOW()
		 RETURNING `+entitlementColumns,
		systemID, entitlement, scope, source, sourceRef, validUntil, createdByJSON, purchasedByJSON, variantJSON))
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23503" {
			return nil, ErrCatalogItemNotFound
		}
		return nil, fmt.Errorf("failed to upsert entitlement: %w", err)
	}
	return e, nil
}

// MarkPending records a shop order placed at checkout and not yet paid.
// Display-only: on a fresh purchase it creates a NON-active stub row
// (valid_until = valid_from → enforcement keeps answering 403); on an
// existing row (renewal, re-buy after revoke/expiry) it only stamps the
// pending marker, leaving validity and revocation untouched. Idempotent.
// purchasedBy and variant are stamped on the fresh stub only: a pending
// renewal must not overwrite who bought — or which tier — the grant the
// customer currently has; activate stamps both when the payment lands.
func (r *LocalSystemEntitlementRepository) MarkPending(systemID, entitlement, scope, ref string, createdBy, purchasedBy, variant map[string]interface{}) (*models.SystemEntitlement, error) {
	var createdByJSON []byte
	if createdBy != nil {
		createdByJSON, _ = json.Marshal(createdBy)
	}
	// interface{} nil → SQL NULL: lib/pq encodes a nil []byte as an empty
	// string, which jsonb rejects.
	var purchasedByJSON interface{}
	if purchasedBy != nil {
		purchasedByJSON, _ = json.Marshal(purchasedBy)
	}
	var variantJSON interface{}
	if variant != nil {
		variantJSON, _ = json.Marshal(variant)
	}

	e, err := scanEntitlement(r.db.QueryRow(
		`INSERT INTO system_entitlements (system_id, entitlement, scope, source, source_ref, valid_from, valid_until, pending_ref, pending_since, created_by, purchased_by, variant)
		 VALUES ($1, $2, $3, 'shop', $4, NOW(), NOW(), $4, NOW(), $5, $6, $7)
		 ON CONFLICT (system_id, entitlement, scope) DO UPDATE SET
		     pending_ref   = EXCLUDED.pending_ref,
		     pending_since = NOW(),
		     updated_at    = NOW()
		 RETURNING `+entitlementColumns,
		systemID, entitlement, scope, ref, createdByJSON, purchasedByJSON, variantJSON))
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23503" {
			return nil, ErrCatalogItemNotFound
		}
		return nil, fmt.Errorf("failed to mark entitlement pending: %w", err)
	}
	return e, nil
}

// ClearPending removes the pending marker (order cancelled/failed before
// payment). A never-activated stub (valid_until = valid_from, not revoked)
// is deleted outright so the UI goes back to "not purchased"; in that case
// (nil, nil) is returned. ErrEntitlementNotFound when no row matches.
func (r *LocalSystemEntitlementRepository) ClearPending(systemID, entitlement, scope string) (*models.SystemEntitlement, error) {
	res, err := r.db.Exec(
		`DELETE FROM system_entitlements
		 WHERE system_id = $1 AND entitlement = $2 AND scope = $3
		   AND pending_ref IS NOT NULL AND revoked_at IS NULL AND valid_until = valid_from`,
		systemID, entitlement, scope)
	if err != nil {
		return nil, fmt.Errorf("failed to clear pending entitlement: %w", err)
	}
	if n, _ := res.RowsAffected(); n > 0 {
		return nil, nil
	}

	e, err := scanEntitlement(r.db.QueryRow(
		`UPDATE system_entitlements SET pending_ref = NULL, pending_since = NULL, updated_at = NOW()
		 WHERE system_id = $1 AND entitlement = $2 AND scope = $3
		 RETURNING `+entitlementColumns,
		systemID, entitlement, scope))
	if err == sql.ErrNoRows {
		return nil, ErrEntitlementNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to clear pending entitlement: %w", err)
	}
	return e, nil
}

// FindSystemIDByKey resolves a system_key (NETH-...) to the internal system
// id; deleted systems are excluded.
func (r *LocalSystemEntitlementRepository) FindSystemIDByKey(systemKey string) (id, systemType string, err error) {
	var sysType sql.NullString
	err = r.db.QueryRow(
		`SELECT id, type FROM systems WHERE system_key = $1 AND deleted_at IS NULL`, systemKey).Scan(&id, &sysType)
	if err == sql.ErrNoRows {
		return "", "", ErrEntitlementNotFound
	}
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve system key: %w", err)
	}
	return id, sysType.String, nil
}

// GrantsReportFilter narrows the fleet-wide grants report. OrgScope nil =
// no restriction (owner/Super Admin); otherwise only systems whose
// organization_id is in the set are returned (caller's hierarchy).
type GrantsReportFilter struct {
	Entitlement    string
	OrganizationID string
	Source         string
	ActiveOnly     bool
	ExpiringBefore *time.Time
	OrgScope       []string
}

const grantsReportJoin = `
	FROM system_entitlements e
	JOIN systems s ON s.id = e.system_id
	LEFT JOIN distributors d ON (s.organization_id = d.logto_id) AND d.deleted_at IS NULL
	LEFT JOIN resellers   r ON (s.organization_id = r.logto_id) AND r.deleted_at IS NULL
	LEFT JOIN customers   c ON (s.organization_id = c.logto_id) AND c.deleted_at IS NULL`

func (f *GrantsReportFilter) whereClause(args *[]interface{}) string {
	where := "WHERE s.deleted_at IS NULL"
	add := func(clause string, v interface{}) {
		*args = append(*args, v)
		where += fmt.Sprintf(" AND "+clause, len(*args))
	}
	if f.Entitlement != "" {
		add("e.entitlement = $%d", f.Entitlement)
	}
	if f.OrganizationID != "" {
		add("s.organization_id = $%d", f.OrganizationID)
	}
	if f.Source != "" {
		add("e.source = $%d", f.Source)
	}
	if f.ExpiringBefore != nil {
		add("e.valid_until IS NOT NULL AND e.valid_until <= $%d", *f.ExpiringBefore)
	}
	if f.ActiveOnly {
		where += " AND e.revoked_at IS NULL AND (e.valid_until IS NULL OR e.valid_until > NOW())"
	}
	if f.OrgScope != nil {
		add("s.organization_id = ANY($%d)", pq.Array(f.OrgScope))
	}
	return where
}

// ListGrants returns the grants report (grant + system + org identity),
// newest first, with total count for pagination.
func (r *LocalSystemEntitlementRepository) ListGrants(f GrantsReportFilter, limit, offset int) ([]*models.EntitlementGrantReportRow, int, error) {
	args := []interface{}{}
	where := f.whereClause(&args)

	var total int
	if err := r.db.QueryRow(`SELECT COUNT(*) `+grantsReportJoin+` `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count grants: %w", err)
	}

	args = append(args, limit, offset)
	rows, err := r.db.Query(
		`SELECT e.id, e.system_id, e.entitlement, e.scope, e.source, COALESCE(e.source_ref, ''),
		        e.valid_from, e.valid_until, e.revoked_at, COALESCE(e.revoked_source, ''),
		        COALESCE(e.pending_ref, ''), e.pending_since, e.created_by, e.purchased_by, e.variant, e.renewal_count, e.created_at, e.updated_at,
		        (e.revoked_at IS NULL AND (e.valid_until IS NULL OR e.valid_until > NOW())) AS active,
		        (s.suspended_at IS NOT NULL) AS system_suspended,
		        s.name, COALESCE(s.system_key, ''), s.organization_id,
		        COALESCE(d.name, r.name, c.name, 'Owner') AS organization_name
		 `+grantsReportJoin+`
		 `+where+`
		 ORDER BY e.created_at DESC
		 LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)),
		args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list grants: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := []*models.EntitlementGrantReportRow{}
	for rows.Next() {
		var row models.EntitlementGrantReportRow
		var validUntil, revokedAt, pendingSince sql.NullTime
		var createdBy, purchasedBy, variant []byte
		var systemSuspended bool
		err := rows.Scan(
			&row.ID, &row.SystemID, &row.Entitlement, &row.Scope, &row.Source, &row.SourceRef,
			&row.ValidFrom, &validUntil, &revokedAt, &row.RevokedSource,
			&row.PendingRef, &pendingSince, &createdBy, &purchasedBy, &variant, &row.RenewalCount, &row.CreatedAt, &row.UpdatedAt,
			&row.Active, &systemSuspended,
			&row.SystemName, &row.SystemKey, &row.OrganizationID, &row.OrganizationName,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan grant row: %w", err)
		}
		if validUntil.Valid {
			t := validUntil.Time
			row.ValidUntil = &t
		}
		if revokedAt.Valid {
			t := revokedAt.Time
			row.RevokedAt = &t
		}
		if pendingSince.Valid {
			t := pendingSince.Time
			row.PendingSince = &t
		}
		if len(createdBy) > 0 {
			_ = json.Unmarshal(createdBy, &row.CreatedBy)
		}
		if len(purchasedBy) > 0 {
			_ = json.Unmarshal(purchasedBy, &row.PurchasedBy)
		}
		if len(variant) > 0 {
			_ = json.Unmarshal(variant, &row.Variant)
		}
		// Deleted systems are filtered out by the WHERE, so the overlay here
		// is suspension only.
		row.Status = models.EntitlementStatus(row.Active, row.RevokedAt, systemSuspended, row.PendingRef)
		out = append(out, &row)
	}
	return out, total, rows.Err()
}

// reportStatusExpr derives the grant lifecycle status in SQL with the same
// precedence as models.EntitlementStatus (suspended > active > pending >
// revoked > expired); `active` mirrors the usual validity expression.
const reportStatusExpr = `
	CASE
	    WHEN (e.revoked_at IS NULL AND (e.valid_until IS NULL OR e.valid_until > NOW()))
	         AND s.suspended_at IS NOT NULL THEN 'suspended'
	    WHEN (e.revoked_at IS NULL AND (e.valid_until IS NULL OR e.valid_until > NOW())) THEN 'active'
	    WHEN e.pending_ref IS NOT NULL THEN 'pending'
	    WHEN e.revoked_at IS NOT NULL THEN 'revoked'
	    ELSE 'expired'
	END`

// Report builds the fleet-wide add-on analytics for the owner/Super Admin
// view: lifecycle totals, per-type / per-org / per-tier breakdowns, renewal
// distribution and a 12-month activation trend. Deleted systems excluded.
func (r *LocalSystemEntitlementRepository) Report() (*models.EntitlementReport, error) {
	report := &models.EntitlementReport{
		ByEntitlement: []models.EntitlementReportByType{},
		Trend:         []models.EntitlementReportTrendRow{},
	}

	statusJoin := `FROM system_entitlements e JOIN systems s ON s.id = e.system_id WHERE s.deleted_at IS NULL`

	// Totals: lifecycle counts, expiry buckets, coverage (with the org-type
	// breakdown of who is buying) and renewals in one pass.
	err := r.db.QueryRow(`
		SELECT COUNT(*),
		       COUNT(*) FILTER (WHERE `+reportStatusExpr+` = 'active'),
		       COUNT(*) FILTER (WHERE `+reportStatusExpr+` = 'expired'),
		       COUNT(*) FILTER (WHERE `+reportStatusExpr+` = 'revoked'),
		       COUNT(*) FILTER (WHERE `+reportStatusExpr+` = 'pending'),
		       COUNT(*) FILTER (WHERE `+reportStatusExpr+` = 'suspended'),
		       COUNT(*) FILTER (WHERE e.revoked_at IS NULL AND e.valid_until IS NULL),
		       COUNT(*) FILTER (WHERE e.revoked_at IS NULL AND e.valid_until > NOW() AND e.valid_until <= NOW() + interval '30 days'),
		       COUNT(*) FILTER (WHERE e.revoked_at IS NULL AND e.valid_until > NOW() AND e.valid_until <= NOW() + interval '60 days'),
		       COUNT(*) FILTER (WHERE e.revoked_at IS NULL AND e.valid_until > NOW() AND e.valid_until <= NOW() + interval '90 days'),
		       COUNT(DISTINCT e.system_id),
		       COUNT(DISTINCT s.organization_id),
		       COUNT(DISTINCT e.system_id) FILTER (WHERE d.logto_id IS NOT NULL),
		       COUNT(DISTINCT e.system_id) FILTER (WHERE rs.logto_id IS NOT NULL),
		       COUNT(DISTINCT e.system_id) FILTER (WHERE c.logto_id IS NOT NULL),
		       COUNT(DISTINCT e.system_id) FILTER (WHERE d.logto_id IS NULL AND rs.logto_id IS NULL AND c.logto_id IS NULL),
		       COALESCE(SUM(e.renewal_count), 0)
		FROM system_entitlements e
		JOIN systems s ON s.id = e.system_id
		LEFT JOIN distributors d ON s.organization_id = d.logto_id AND d.deleted_at IS NULL
		LEFT JOIN resellers   rs ON s.organization_id = rs.logto_id AND rs.deleted_at IS NULL
		LEFT JOIN customers    c ON s.organization_id = c.logto_id AND c.deleted_at IS NULL
		WHERE s.deleted_at IS NULL`,
	).Scan(
		&report.Totals.Total, &report.Totals.Active, &report.Totals.Expired,
		&report.Totals.Revoked, &report.Totals.Pending, &report.Totals.Suspended,
		&report.Totals.Perpetual,
		&report.Totals.ExpiringIn30d, &report.Totals.ExpiringIn60d, &report.Totals.ExpiringIn90d,
		&report.Totals.Systems, &report.Totals.Organizations,
		&report.Totals.DistributorSystems, &report.Totals.ResellerSystems, &report.Totals.CustomerSystems,
		&report.Totals.OwnerSystems,
		&report.Totals.TotalRenewals,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to compute report totals: %w", err)
	}

	// Renewal distribution.
	err = r.db.QueryRow(`
		SELECT COUNT(*) FILTER (WHERE e.renewal_count = 0),
		       COUNT(*) FILTER (WHERE e.renewal_count = 1),
		       COUNT(*) FILTER (WHERE e.renewal_count = 2),
		       COUNT(*) FILTER (WHERE e.renewal_count >= 3)
		`+statusJoin,
	).Scan(&report.Renewals.Never, &report.Renewals.Once, &report.Renewals.Twice, &report.Renewals.ThreePlus)
	if err != nil {
		return nil, fmt.Errorf("failed to compute renewal distribution: %w", err)
	}

	// Per add-on type.
	rows, err := r.db.Query(`
		SELECT e.entitlement, COALESCE(cat.display_name, e.entitlement),
		       COUNT(*) FILTER (WHERE ` + reportStatusExpr + ` = 'active'),
		       COUNT(*) FILTER (WHERE ` + reportStatusExpr + ` = 'expired'),
		       COUNT(*) FILTER (WHERE ` + reportStatusExpr + ` = 'revoked'),
		       COUNT(*) FILTER (WHERE ` + reportStatusExpr + ` = 'pending'),
		       COUNT(*) FILTER (WHERE ` + reportStatusExpr + ` = 'suspended'),
		       COUNT(*)
		FROM system_entitlements e
		JOIN systems s ON s.id = e.system_id
		LEFT JOIN entitlement_catalog cat ON cat.id = e.entitlement
		WHERE s.deleted_at IS NULL
		GROUP BY e.entitlement, cat.display_name
		ORDER BY COUNT(*) DESC, e.entitlement`)
	if err != nil {
		return nil, fmt.Errorf("failed to compute per-type report: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var row models.EntitlementReportByType
		if err := rows.Scan(&row.Entitlement, &row.DisplayName, &row.Active, &row.Expired, &row.Revoked, &row.Pending, &row.Suspended, &row.Total); err != nil {
			return nil, fmt.Errorf("failed to scan per-type row: %w", err)
		}
		report.ByEntitlement = append(report.ByEntitlement, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Activation trend: grants created per month, last 12 months.
	trendRows, err := r.db.Query(`
		SELECT to_char(date_trunc('month', e.created_at), 'YYYY-MM'), COUNT(*)
		` + statusJoin + ` AND e.created_at >= date_trunc('month', NOW()) - interval '11 months'
		GROUP BY 1 ORDER BY 1`)
	if err != nil {
		return nil, fmt.Errorf("failed to compute activation trend: %w", err)
	}
	defer func() { _ = trendRows.Close() }()
	for trendRows.Next() {
		var row models.EntitlementReportTrendRow
		if err := trendRows.Scan(&row.Month, &row.Activations); err != nil {
			return nil, fmt.Errorf("failed to scan trend row: %w", err)
		}
		report.Trend = append(report.Trend, row)
	}
	if err := trendRows.Err(); err != nil {
		return nil, err
	}

	return report, nil
}

// ReportOrganizations is the paginated + searchable per-organization slice
// of the add-on report (orgs can be hundreds on the real fleet): grants
// grouped by the systems' owning organization, most active first. search
// filters by organization name.
func (r *LocalSystemEntitlementRepository) ReportOrganizations(search string, limit, offset int) ([]models.EntitlementReportByOrg, int, error) {
	base := `
		FROM system_entitlements e
		JOIN systems s ON s.id = e.system_id
		LEFT JOIN distributors d ON s.organization_id = d.logto_id AND d.deleted_at IS NULL
		LEFT JOIN resellers   rs ON s.organization_id = rs.logto_id AND rs.deleted_at IS NULL
		LEFT JOIN customers    c ON s.organization_id = c.logto_id AND c.deleted_at IS NULL
		WHERE s.deleted_at IS NULL
		  AND ($1 = '' OR COALESCE(d.name, rs.name, c.name, 'Owner') ILIKE '%' || $1 || '%')
		GROUP BY s.organization_id, d.logto_id, rs.logto_id, c.logto_id, d.name, rs.name, c.name`

	var total int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM (SELECT 1 `+base+`) g`, search).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count report organizations: %w", err)
	}

	rows, err := r.db.Query(`
		SELECT s.organization_id,
		       COALESCE(d.name, rs.name, c.name, 'Owner'),
		       COALESCE(
		           CASE WHEN d.logto_id IS NOT NULL THEN 'distributor'
		                WHEN rs.logto_id IS NOT NULL THEN 'reseller'
		                WHEN c.logto_id IS NOT NULL THEN 'customer'
		           END, 'owner'),
		       COUNT(DISTINCT e.system_id),
		       COUNT(*) FILTER (WHERE `+reportStatusExpr+` = 'active'),
		       COUNT(*)
		`+base+`
		ORDER BY COUNT(*) FILTER (WHERE `+reportStatusExpr+` = 'active') DESC, COUNT(*) DESC
		LIMIT $2 OFFSET $3`, search, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to compute per-org report: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := []models.EntitlementReportByOrg{}
	for rows.Next() {
		var row models.EntitlementReportByOrg
		if err := rows.Scan(&row.OrganizationID, &row.OrganizationName, &row.OrgType, &row.Systems, &row.Active, &row.Total); err != nil {
			return nil, 0, fmt.Errorf("failed to scan per-org row: %w", err)
		}
		out = append(out, row)
	}
	return out, total, rows.Err()
}

// ReportVariants is the paginated + searchable per-tier slice of the add-on
// report (variable products only). search filters by add-on id or tier label.
func (r *LocalSystemEntitlementRepository) ReportVariants(search string, limit, offset int) ([]models.EntitlementReportByVariant, int, error) {
	where := `
		FROM system_entitlements e
		JOIN systems s ON s.id = e.system_id
		WHERE s.deleted_at IS NULL AND e.variant->>'label' IS NOT NULL
		  AND ($1 = '' OR e.entitlement ILIKE '%' || $1 || '%' OR e.variant->>'label' ILIKE '%' || $1 || '%')
		GROUP BY e.entitlement, e.variant->>'label'`

	var total int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM (SELECT 1 `+where+`) g`, search).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count report tiers: %w", err)
	}

	rows, err := r.db.Query(`
		SELECT e.entitlement, e.variant->>'label', COUNT(*)
		`+where+`
		ORDER BY e.entitlement, COUNT(*) DESC
		LIMIT $2 OFFSET $3`, search, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to compute per-variant report: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := []models.EntitlementReportByVariant{}
	for rows.Next() {
		var row models.EntitlementReportByVariant
		if err := rows.Scan(&row.Entitlement, &row.Label, &row.Count); err != nil {
			return nil, 0, fmt.Errorf("failed to scan per-variant row: %w", err)
		}
		out = append(out, row)
	}
	return out, total, rows.Err()
}

// Stats aggregates ACTIVE grants per entitlement per organization, within
// the caller's scope (nil = everything).
func (r *LocalSystemEntitlementRepository) Stats(orgScope []string) ([]*models.EntitlementStatsRow, error) {
	f := GrantsReportFilter{ActiveOnly: true, OrgScope: orgScope}
	args := []interface{}{}
	where := f.whereClause(&args)

	rows, err := r.db.Query(
		`SELECT e.entitlement, s.organization_id,
		        COALESCE(d.name, r.name, c.name, 'Owner') AS organization_name,
		        COUNT(*) AS active_grants
		 `+grantsReportJoin+`
		 `+where+`
		 GROUP BY e.entitlement, s.organization_id, organization_name
		 ORDER BY e.entitlement, active_grants DESC`,
		args...)
	if err != nil {
		return nil, fmt.Errorf("failed to compute entitlement stats: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := []*models.EntitlementStatsRow{}
	for rows.Next() {
		var row models.EntitlementStatsRow
		if err := rows.Scan(&row.Entitlement, &row.OrganizationID, &row.OrganizationName, &row.ActiveGrants); err != nil {
			return nil, fmt.Errorf("failed to scan stats row: %w", err)
		}
		out = append(out, &row)
	}
	return out, rows.Err()
}
