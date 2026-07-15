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
	valid_from, valid_until, revoked_at, created_by, created_at, updated_at,
	(revoked_at IS NULL AND (valid_until IS NULL OR valid_until > NOW())) AS active`

func scanEntitlement(scanner interface{ Scan(...interface{}) error }) (*models.SystemEntitlement, error) {
	var e models.SystemEntitlement
	var validUntil, revokedAt sql.NullTime
	var createdBy []byte

	err := scanner.Scan(
		&e.ID, &e.SystemID, &e.Entitlement, &e.Scope, &e.Source, &e.SourceRef,
		&e.ValidFrom, &validUntil, &revokedAt, &createdBy, &e.CreatedAt, &e.UpdatedAt,
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
	if len(createdBy) > 0 {
		_ = json.Unmarshal(createdBy, &e.CreatedBy)
	}

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
func (r *LocalSystemEntitlementRepository) Create(systemID, entitlement, scope, source, sourceRef string, validUntil *time.Time, createdBy map[string]interface{}) (*models.SystemEntitlement, error) {
	var createdByJSON []byte
	if createdBy != nil {
		createdByJSON, _ = json.Marshal(createdBy)
	}

	e, err := scanEntitlement(r.db.QueryRow(
		`INSERT INTO system_entitlements (system_id, entitlement, scope, source, source_ref, valid_until, created_by)
		 VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7)
		 RETURNING `+entitlementColumns,
		systemID, entitlement, scope, source, sourceRef, validUntil, createdByJSON))
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
// to the given value, possibly NULL" (true).
func (r *LocalSystemEntitlementRepository) Update(systemID, entitlement, scope string, setValidUntil bool, validUntil *time.Time, revoked *bool) (*models.SystemEntitlement, error) {
	e, err := scanEntitlement(r.db.QueryRow(
		`UPDATE system_entitlements SET
		     valid_until = CASE WHEN $4 THEN $5 ELSE valid_until END,
		     revoked_at  = CASE
		                       WHEN $6::boolean IS NULL THEN revoked_at
		                       WHEN $6 THEN COALESCE(revoked_at, NOW())
		                       ELSE NULL
		                   END,
		     updated_at  = NOW()
		 WHERE system_id = $1 AND entitlement = $2 AND scope = $3
		 RETURNING `+entitlementColumns,
		systemID, entitlement, scope, setValidUntil, validUntil, revoked))
	if err == sql.ErrNoRows {
		return nil, ErrEntitlementNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update entitlement: %w", err)
	}
	return e, nil
}

// Revoke marks the entitlement as revoked (idempotent, row kept for audit).
func (r *LocalSystemEntitlementRepository) Revoke(systemID, entitlement, scope string) (*models.SystemEntitlement, error) {
	revoked := true
	return r.Update(systemID, entitlement, scope, false, nil, &revoked)
}

// Upsert creates the grant or, when the (system, entitlement, scope) row
// already exists, renews it in place: new expiry, revocation cleared,
// source/source_ref refreshed. This is the shop-webhook semantics —
// activation and renewal are the same idempotent call (safe on retries).
func (r *LocalSystemEntitlementRepository) Upsert(systemID, entitlement, scope, source, sourceRef string, validUntil *time.Time, createdBy map[string]interface{}) (*models.SystemEntitlement, error) {
	var createdByJSON []byte
	if createdBy != nil {
		createdByJSON, _ = json.Marshal(createdBy)
	}

	e, err := scanEntitlement(r.db.QueryRow(
		`INSERT INTO system_entitlements (system_id, entitlement, scope, source, source_ref, valid_until, created_by)
		 VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7)
		 ON CONFLICT (system_id, entitlement, scope) DO UPDATE SET
		     valid_until = EXCLUDED.valid_until,
		     revoked_at  = NULL,
		     source      = EXCLUDED.source,
		     source_ref  = COALESCE(EXCLUDED.source_ref, system_entitlements.source_ref),
		     updated_at  = NOW()
		 RETURNING `+entitlementColumns,
		systemID, entitlement, scope, source, sourceRef, validUntil, createdByJSON))
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23503" {
			return nil, ErrCatalogItemNotFound
		}
		return nil, fmt.Errorf("failed to upsert entitlement: %w", err)
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
		        e.valid_from, e.valid_until, e.revoked_at, e.created_by, e.created_at, e.updated_at,
		        (e.revoked_at IS NULL AND (e.valid_until IS NULL OR e.valid_until > NOW())) AS active,
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
		var validUntil, revokedAt sql.NullTime
		var createdBy []byte
		err := rows.Scan(
			&row.ID, &row.SystemID, &row.Entitlement, &row.Scope, &row.Source, &row.SourceRef,
			&row.ValidFrom, &validUntil, &revokedAt, &createdBy, &row.CreatedAt, &row.UpdatedAt,
			&row.Active,
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
		if len(createdBy) > 0 {
			_ = json.Unmarshal(createdBy, &row.CreatedBy)
		}
		out = append(out, &row)
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
