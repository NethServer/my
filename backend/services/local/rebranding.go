/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package local

import (
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/lib/pq"

	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/models"
)

// Sentinel errors so callers can tell "there was nothing to delete" and "you
// asked for an asset that does not exist" apart from a real storage failure.
// Without them a delete on an organization with no assets is indistinguishable
// from a database outage, and the handlers have to guess a status code.
var (
	ErrRebrandingAssetsNotFound = errors.New("no rebranding assets found for this organization and product")
	ErrInvalidRebrandingAsset   = errors.New("invalid rebranding asset name")
	ErrInvalidRebrandingProduct = errors.New("unknown rebranding product")
	ErrNoRebrandingProducts     = errors.New("at least one product is required")
)

// RebrandingFieldError names the request field and the value that was wrong, so
// the handler can answer in the project's validation shape (key/message/value)
// instead of a bare sentence. errors.Is still resolves the sentinel underneath.
type RebrandingFieldError struct {
	Field string
	Value string
	Err   error
}

func (e *RebrandingFieldError) Error() string { return fmt.Sprintf("%v: %s", e.Err, e.Value) }
func (e *RebrandingFieldError) Unwrap() error { return e.Err }

// The asset columns of rebranding_assets, in the order the configuration form
// presents them. Every query that walks the assets derives its column list from
// here, so adding an asset is one entry plus a migration.
var rebrandingAssetNames = []string{
	"logo_light_rect",
	"logo_dark_rect",
	"logo_light_square",
	"logo_dark_square",
	"favicon",
	"background_image",
}

// RebrandingService handles rebranding operations
type RebrandingService struct{}

// NewRebrandingService creates a new rebranding service
func NewRebrandingService() *RebrandingService {
	return &RebrandingService{}
}

// ListProducts returns all rebrandable products
func (s *RebrandingService) ListProducts() ([]models.RebrandableProduct, error) {
	query := `SELECT id, display_name, type, created_at FROM rebrandable_products ORDER BY type, display_name`
	rows, err := database.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query rebrandable products: %w", err)
	}
	defer func() { _ = rows.Close() }()

	products := make([]models.RebrandableProduct, 0)
	for rows.Next() {
		var p models.RebrandableProduct
		if err := rows.Scan(&p.ID, &p.DisplayName, &p.Type, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan rebrandable product: %w", err)
		}
		products = append(products, p)
	}
	return products, nil
}

// EnableRebranding enables rebranding for an organization
func (s *RebrandingService) EnableRebranding(orgID, orgType string) error {
	query := `
		INSERT INTO rebranding_enabled (organization_id, organization_type)
		VALUES ($1, $2)
		ON CONFLICT (organization_id) DO UPDATE SET organization_type = $2, enabled_at = NOW()
	`
	_, err := database.DB.Exec(query, orgID, orgType)
	if err != nil {
		return fmt.Errorf("failed to enable rebranding: %w", err)
	}
	return nil
}

// EnableRebrandingBulk adds several organizations at once, the way the picker
// submits them. Organizations that cannot be resolved are reported back and
// nothing is written: a partial enablement would leave the caller with no
// coherent state to show. Returns how many were newly enabled.
func (s *RebrandingService) EnableRebrandingBulk(orgIDs []string) (int, []string, error) {
	unique := make([]string, 0, len(orgIDs))
	seen := make(map[string]bool)
	for _, id := range orgIDs {
		if id != "" && !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}
	if len(unique) == 0 {
		return 0, nil, nil
	}

	rows, err := database.DB.Query(
		`SELECT logto_id, org_type FROM unified_organizations WHERE logto_id = ANY($1)`,
		pq.Array(unique),
	)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to resolve organizations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	types := make(map[string]string)
	for rows.Next() {
		var logtoID, orgType string
		if err := rows.Scan(&logtoID, &orgType); err != nil {
			return 0, nil, fmt.Errorf("failed to resolve organizations: %w", err)
		}
		types[logtoID] = orgType
	}

	// The owner organization is in none of the three tables behind the view, so
	// it lands here rather than needing a case of its own.
	invalid := make([]string, 0)
	ids := make([]string, 0, len(unique))
	orgTypes := make([]string, 0, len(unique))
	for _, id := range unique {
		orgType, ok := types[id]
		if !ok {
			invalid = append(invalid, id)
			continue
		}
		ids = append(ids, id)
		orgTypes = append(orgTypes, orgType)
	}
	if len(invalid) > 0 {
		return 0, invalid, nil
	}

	result, err := database.DB.Exec(`
		INSERT INTO rebranding_enabled (organization_id, organization_type)
		SELECT * FROM unnest($1::varchar[], $2::varchar[])
		ON CONFLICT (organization_id) DO NOTHING`,
		pq.Array(ids), pq.Array(orgTypes),
	)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to enable rebranding: %w", err)
	}

	enabled, _ := result.RowsAffected()
	return int(enabled), nil, nil
}

// DisableRebranding removes an organization from rebranding. The assets go with
// it: rebranding_assets references this row with ON DELETE CASCADE, so nothing
// survives to be restored the next time the organization is added back.
func (s *RebrandingService) DisableRebranding(orgID string) error {
	query := `DELETE FROM rebranding_enabled WHERE organization_id = $1`
	_, err := database.DB.Exec(query, orgID)
	if err != nil {
		return fmt.Errorf("failed to disable rebranding: %w", err)
	}
	return nil
}

// IsRebrandingEnabled checks if rebranding is enabled for an organization
func (s *RebrandingService) IsRebrandingEnabled(orgID string) (bool, error) {
	query := `SELECT COUNT(*) FROM rebranding_enabled WHERE organization_id = $1`
	var count int
	err := database.DB.QueryRow(query, orgID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check rebranding status: %w", err)
	}
	return count > 0, nil
}

// RebrandingOrganizationFilters narrows the organizations-with-rebranding list.
type RebrandingOrganizationFilters struct {
	Search        string
	Types         []string
	ProductIDs    []string
	Page          int
	PageSize      int
	SortBy        string
	SortDirection string
}

// Sortable columns of the organizations list, mapped to their SQL expression.
var rebrandingSortColumns = map[string]string{
	"name":              "uo.name",
	"organization_type": "re.organization_type",
	"enabled_at":        "re.enabled_at",
	"updated_at":        "updated_at",
}

// scopeRebrandingOrgs restricts a query to the organizations the caller may
// see. The owner sees every enabled organization; everyone else sees their own
// subtree, so a partner reading the list cannot enumerate another partner's.
func scopeRebrandingOrgs(where, column, userOrgRole, userOrgID string, args []interface{}, nextArg int) (string, []interface{}, int) {
	if strings.ToLower(userOrgRole) == "owner" {
		return where, args, nextArg
	}
	allowed := helpers.GetAllowedOrgIDsForFilter(strings.ToLower(userOrgRole), userOrgID)
	where += fmt.Sprintf(" AND %s = ANY($%d)", column, nextArg)
	args = append(args, pq.Array(allowed))
	return where, args, nextArg + 1
}

// ListOrganizations returns the organizations that have rebranding enabled,
// each with the products it has actually branded.
func (s *RebrandingService) ListOrganizations(userOrgRole, userOrgID string, f RebrandingOrganizationFilters) ([]models.RebrandingOrganization, int, error) {
	from := `
		FROM rebranding_enabled re
		JOIN unified_organizations uo ON uo.logto_id = re.organization_id`

	where := " WHERE 1=1"
	args := []interface{}{}
	nextArg := 1

	if f.Search != "" {
		where += fmt.Sprintf(" AND uo.name ILIKE $%d", nextArg)
		args = append(args, "%"+f.Search+"%")
		nextArg++
	}
	if len(f.Types) > 0 {
		where += fmt.Sprintf(" AND re.organization_type = ANY($%d)", nextArg)
		args = append(args, pq.Array(f.Types))
		nextArg++
	}
	if len(f.ProductIDs) > 0 {
		where += fmt.Sprintf(` AND EXISTS (
			SELECT 1 FROM rebranding_assets ra
			WHERE ra.organization_id = re.organization_id AND ra.product_id = ANY($%d))`, nextArg)
		args = append(args, pq.Array(f.ProductIDs))
		nextArg++
	}
	where, args, nextArg = scopeRebrandingOrgs(where, "re.organization_id", userOrgRole, userOrgID, args, nextArg)

	var totalCount int
	if err := database.DB.QueryRow(`SELECT COUNT(*)`+from+where, args...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to count rebranding organizations: %w", err)
	}

	sortColumn, ok := rebrandingSortColumns[f.SortBy]
	if !ok {
		sortColumn = "uo.name"
	}
	sortDirection := "ASC"
	if strings.EqualFold(f.SortDirection, "desc") {
		sortDirection = "DESC"
	}

	page, pageSize := f.Page, f.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	query := `
		SELECT re.organization_id, uo.db_id, uo.name, re.organization_type, re.enabled_at,
			(SELECT MAX(ra.updated_at) FROM rebranding_assets ra
			 WHERE ra.organization_id = re.organization_id) AS updated_at` + from + where +
		fmt.Sprintf(" ORDER BY %s %s NULLS LAST, uo.name ASC LIMIT $%d OFFSET $%d", sortColumn, sortDirection, nextArg, nextArg+1)
	args = append(args, pageSize, (page-1)*pageSize)

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list rebranding organizations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	organizations := make([]models.RebrandingOrganization, 0)
	orgIDs := make([]string, 0)
	for rows.Next() {
		var org models.RebrandingOrganization
		var updatedAt sql.NullTime
		if err := rows.Scan(&org.LogtoID, &org.ID, &org.Name, &org.OrganizationType, &org.EnabledAt, &updatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan rebranding organization: %w", err)
		}
		if updatedAt.Valid {
			ts := updatedAt.Time
			org.UpdatedAt = &ts
		}
		org.Products = []models.RebrandingOrganizationProduct{}
		organizations = append(organizations, org)
		orgIDs = append(orgIDs, org.LogtoID)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to read rebranding organizations: %w", err)
	}

	byOrg, err := s.brandedProducts(orgIDs)
	if err != nil {
		return nil, 0, err
	}
	for i := range organizations {
		if products, ok := byOrg[organizations[i].LogtoID]; ok {
			organizations[i].Products = products
		}
	}

	return organizations, totalCount, nil
}

// brandedProducts fetches the products of a whole page of organizations in one
// query, so the list does not run a query per row.
func (s *RebrandingService) brandedProducts(orgIDs []string) (map[string][]models.RebrandingOrganizationProduct, error) {
	result := make(map[string][]models.RebrandingOrganizationProduct)
	if len(orgIDs) == 0 {
		return result, nil
	}

	rows, err := database.DB.Query(`
		SELECT ra.organization_id, ra.product_id, rp.display_name, ra.product_name
		FROM rebranding_assets ra
		JOIN rebrandable_products rp ON rp.id = ra.product_id
		WHERE ra.organization_id = ANY($1)
		ORDER BY rp.type, rp.display_name`, pq.Array(orgIDs))
	if err != nil {
		return nil, fmt.Errorf("failed to query branded products: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var orgID string
		var product models.RebrandingOrganizationProduct
		var productName sql.NullString
		if err := rows.Scan(&orgID, &product.ProductID, &product.ProductDisplayName, &productName); err != nil {
			return nil, fmt.Errorf("failed to scan branded product: %w", err)
		}
		if productName.Valid {
			name := productName.String
			product.ProductName = &name
		}
		result[orgID] = append(result[orgID], product)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read branded products: %w", err)
	}

	return result, nil
}

// Summary counts the enabled organizations by type for the counters above the
// list. It walks the same join as the list, so a soft-deleted organization is
// absent from both and the counters match the rows.
func (s *RebrandingService) Summary(userOrgRole, userOrgID string) (*models.RebrandingSummary, error) {
	where := " WHERE 1=1"
	args := []interface{}{}
	where, args, _ = scopeRebrandingOrgs(where, "re.organization_id", userOrgRole, userOrgID, args, 1)

	query := `
		SELECT re.organization_type, COUNT(*)
		FROM rebranding_enabled re
		JOIN unified_organizations uo ON uo.logto_id = re.organization_id` + where + `
		GROUP BY re.organization_type`

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to summarize rebranding organizations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	summary := &models.RebrandingSummary{}
	for rows.Next() {
		var orgType string
		var count int
		if err := rows.Scan(&orgType, &count); err != nil {
			return nil, fmt.Errorf("failed to scan rebranding summary: %w", err)
		}
		switch orgType {
		case "distributor":
			summary.Distributors = count
		case "reseller":
			summary.Resellers = count
		case "customer":
			summary.Customers = count
		}
		summary.Total += count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read rebranding summary: %w", err)
	}

	return summary, nil
}

// ListAvailableOrganizations returns the organizations that can still be added
// to rebranding: they exist, they are not enabled yet, and they are within the
// caller's scope.
func (s *RebrandingService) ListAvailableOrganizations(userOrgRole, userOrgID, search string, limit int) ([]models.RebrandingAvailableOrganization, error) {
	where := ` WHERE NOT EXISTS (
		SELECT 1 FROM rebranding_enabled re WHERE re.organization_id = uo.logto_id)`
	args := []interface{}{}
	nextArg := 1

	if search != "" {
		where += fmt.Sprintf(" AND uo.name ILIKE $%d", nextArg)
		args = append(args, "%"+search+"%")
		nextArg++
	}
	where, args, nextArg = scopeRebrandingOrgs(where, "uo.logto_id", userOrgRole, userOrgID, args, nextArg)

	if limit < 1 {
		limit = 50
	}

	query := `SELECT uo.logto_id, uo.db_id, uo.name, uo.org_type FROM unified_organizations uo` + where +
		fmt.Sprintf(" ORDER BY uo.org_type, uo.name LIMIT $%d", nextArg)
	args = append(args, limit)

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list available organizations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	organizations := make([]models.RebrandingAvailableOrganization, 0)
	for rows.Next() {
		var org models.RebrandingAvailableOrganization
		if err := rows.Scan(&org.LogtoID, &org.ID, &org.Name, &org.OrganizationType); err != nil {
			return nil, fmt.Errorf("failed to scan available organization: %w", err)
		}
		organizations = append(organizations, org)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read available organizations: %w", err)
	}

	return organizations, nil
}

// CountInheritingOrganizations counts the organizations downstream that display
// this branding, which is what the save confirmation reports. It mirrors the
// resolution in resolveRebrandingOrg: an organization with its own rebranding
// enabled shows its own, and shields the ones below it.
func (s *RebrandingService) CountInheritingOrganizations(orgID string) (int, error) {
	var orgType string
	err := database.DB.QueryRow(`SELECT org_type FROM unified_organizations WHERE logto_id = $1`, orgID).Scan(&orgType)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to resolve organization type: %w", err)
	}

	switch orgType {
	case "distributor":
		var count int
		err = database.DB.QueryRow(`
			WITH inheriting_resellers AS (
				SELECT r.logto_id FROM resellers r
				WHERE r.custom_data->>'createdBy' = $1 AND r.deleted_at IS NULL
				  AND NOT EXISTS (SELECT 1 FROM rebranding_enabled e WHERE e.organization_id = r.logto_id)
			)
			SELECT (SELECT COUNT(*) FROM inheriting_resellers)
			     + (SELECT COUNT(*) FROM customers c
			        WHERE c.deleted_at IS NULL
			          AND NOT EXISTS (SELECT 1 FROM rebranding_enabled e WHERE e.organization_id = c.logto_id)
			          AND (c.custom_data->>'createdBy' = $1
			               OR c.custom_data->>'createdBy' IN (SELECT logto_id FROM inheriting_resellers)))`,
			orgID).Scan(&count)
		if err != nil {
			return 0, fmt.Errorf("failed to count inheriting organizations: %w", err)
		}
		return count, nil

	case "reseller":
		var count int
		err = database.DB.QueryRow(`
			SELECT COUNT(*) FROM customers c
			WHERE c.deleted_at IS NULL
			  AND c.custom_data->>'createdBy' = $1
			  AND NOT EXISTS (SELECT 1 FROM rebranding_enabled e WHERE e.organization_id = c.logto_id)`,
			orgID).Scan(&count)
		if err != nil {
			return 0, fmt.Errorf("failed to count inheriting organizations: %w", err)
		}
		return count, nil
	}

	return 0, nil
}

// GetOrgStatus returns the full rebranding status for an organization
func (s *RebrandingService) GetOrgStatus(orgID string) (*models.RebrandingOrgStatus, error) {
	enabled, err := s.IsRebrandingEnabled(orgID)
	if err != nil {
		return nil, err
	}

	products, err := s.ListProducts()
	if err != nil {
		return nil, err
	}

	configured, err := s.assetsByProduct(orgID)
	if err != nil {
		return nil, err
	}

	productStatuses := make([]models.RebrandingProductStatus, 0, len(products))
	for _, p := range products {
		ps := models.RebrandingProductStatus{
			ProductID:          p.ID,
			ProductDisplayName: p.DisplayName,
			ProductType:        p.Type,
			Assets:             []models.RebrandingAssetInfo{},
		}
		if info, ok := configured[p.ID]; ok {
			ps.ProductName = info.productName
			ps.Assets = info.assets
			updatedAt := info.updatedAt
			ps.UpdatedAt = &updatedAt
		}
		productStatuses = append(productStatuses, ps)
	}

	return &models.RebrandingOrgStatus{
		Enabled:  enabled,
		Products: productStatuses,
	}, nil
}

type rebrandingProductAssets struct {
	productName *string
	updatedAt   time.Time
	assets      []models.RebrandingAssetInfo
}

// assetsByProduct reads what an organization has uploaded, per product. Only
// the metadata travels: octet_length() gives the size without pulling megabytes
// of binary through the connection, and the caller gets enough to render the
// upload form (name, type, size) and to cache-bust the binary endpoint.
func (s *RebrandingService) assetsByProduct(orgID string) (map[string]rebrandingProductAssets, error) {
	columns := make([]string, 0, len(rebrandingAssetNames)*3)
	for _, asset := range rebrandingAssetNames {
		columns = append(columns,
			"octet_length(ra."+asset+")",
			"ra."+asset+"_mime",
			"ra."+asset+"_filename",
		)
	}

	query := `SELECT ra.product_id, ra.product_name, ra.updated_at, ` + strings.Join(columns, ", ") + `
		FROM rebranding_assets ra
		WHERE ra.organization_id = $1`

	rows, err := database.DB.Query(query, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to query rebranding assets: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make(map[string]rebrandingProductAssets)
	for rows.Next() {
		var productID string
		var productName sql.NullString
		var updatedAt time.Time

		sizes := make([]sql.NullInt64, len(rebrandingAssetNames))
		mimes := make([]sql.NullString, len(rebrandingAssetNames))
		filenames := make([]sql.NullString, len(rebrandingAssetNames))

		dest := []interface{}{&productID, &productName, &updatedAt}
		for i := range rebrandingAssetNames {
			dest = append(dest, &sizes[i], &mimes[i], &filenames[i])
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("failed to scan rebranding asset: %w", err)
		}

		assets := make([]models.RebrandingAssetInfo, 0, len(rebrandingAssetNames))
		for i, asset := range rebrandingAssetNames {
			if !sizes[i].Valid {
				continue
			}
			info := models.RebrandingAssetInfo{
				Name:      asset,
				Size:      sizes[i].Int64,
				MimeType:  "application/octet-stream",
				UpdatedAt: updatedAt,
			}
			if mimes[i].Valid {
				info.MimeType = mimes[i].String
			}
			if filenames[i].Valid {
				filename := filenames[i].String
				info.Filename = &filename
			}
			assets = append(assets, info)
		}

		entry := rebrandingProductAssets{updatedAt: updatedAt, assets: assets}
		if productName.Valid {
			name := productName.String
			entry.productName = &name
		}
		result[productID] = entry
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read rebranding assets: %w", err)
	}

	return result, nil
}

// UpsertAssets writes the assets of one organization+product. Assets not named
// in uploads or clear keep the value they already have, so a form that touches
// one logo does not blank the others.
func (s *RebrandingService) UpsertAssets(orgID, productID string, productName *string, uploads map[string]models.RebrandingUpload, clear []string) error {
	if err := s.validateProducts([]string{productID}); err != nil {
		return err
	}
	for _, asset := range clear {
		if !slices.Contains(rebrandingAssetNames, asset) {
			return &RebrandingFieldError{Field: "clear", Value: asset, Err: ErrInvalidRebrandingAsset}
		}
	}
	return upsertRebrandingAssets(database.DB, orgID, productID, productName, uploads, clear)
}

// SaveConfig writes a whole configuration form in one transaction: the selected
// products receive the brand name and the uploaded assets, the assets listed in
// clear are emptied, and the products left out of the selection lose their
// configuration. One Save is one atomic write, so a form that replaces a logo
// and clears another cannot land half-applied.
func (s *RebrandingService) SaveConfig(orgID string, cfg models.RebrandingConfig) error {
	if len(cfg.Products) == 0 {
		return ErrNoRebrandingProducts
	}
	if err := s.validateProducts(cfg.Products); err != nil {
		return err
	}
	for _, asset := range cfg.Clear {
		if !slices.Contains(rebrandingAssetNames, asset) {
			return &RebrandingFieldError{Field: "clear", Value: asset, Err: ErrInvalidRebrandingAsset}
		}
	}

	tx, err := database.DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(
		`DELETE FROM rebranding_assets WHERE organization_id = $1 AND product_id <> ALL($2)`,
		orgID, pq.Array(cfg.Products),
	); err != nil {
		return fmt.Errorf("failed to remove deselected products: %w", err)
	}

	for _, productID := range cfg.Products {
		if err := upsertRebrandingAssets(tx, orgID, productID, cfg.BrandName, cfg.Uploads, cfg.Clear); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to save rebranding configuration: %w", err)
	}
	return nil
}

// rebrandingExecer is satisfied by both the pool and a transaction, so the
// single-product and whole-form paths share one upsert.
type rebrandingExecer interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}

func upsertRebrandingAssets(db rebrandingExecer, orgID, productID string, productName *string, uploads map[string]models.RebrandingUpload, clear []string) error {
	query, args := buildRebrandingUpsert(orgID, productID, productName, uploads, clear)
	if _, err := db.Exec(query, args...); err != nil {
		return fmt.Errorf("failed to upsert rebranding assets: %w", err)
	}
	return nil
}

// buildRebrandingUpsert writes the statement for one product: uploaded assets
// are set, cleared ones are nulled, and the rest are left out of the SET clause
// so they keep their value.
func buildRebrandingUpsert(orgID, productID string, productName *string, uploads map[string]models.RebrandingUpload, clear []string) (string, []interface{}) {
	// A brand name sent empty means "clear it": the column goes to NULL and the
	// product falls back to its own name. A nil pointer means the form did not
	// send the field at all, and the stored name is left alone.
	var nameArg interface{}
	if productName != nil && *productName != "" {
		nameArg = *productName
	}

	columns := []string{"organization_id", "product_id", "product_name"}
	values := []string{"$1", "$2", "$3"}
	args := []interface{}{orgID, productID, nameArg}
	updates := []string{}

	if productName != nil {
		updates = append(updates, "product_name = $3")
	}

	next := 4
	for _, asset := range rebrandingAssetNames {
		dataCol, mimeCol, nameCol := asset, asset+"_mime", asset+"_filename"

		if upload, uploaded := uploads[asset]; uploaded {
			columns = append(columns, dataCol, mimeCol, nameCol)
			values = append(values,
				fmt.Sprintf("$%d", next), fmt.Sprintf("$%d", next+1), fmt.Sprintf("$%d", next+2))
			updates = append(updates,
				fmt.Sprintf("%s = $%d", dataCol, next),
				fmt.Sprintf("%s = $%d", mimeCol, next+1),
				fmt.Sprintf("%s = $%d", nameCol, next+2))
			args = append(args, upload.Data, upload.MimeType, nilIfEmptyStr(upload.Filename))
			next += 3
			continue
		}

		// An asset the form did not send is left alone; one the form emptied is
		// nulled explicitly. Columns absent from the SET clause keep their value.
		if slices.Contains(clear, asset) {
			updates = append(updates, dataCol+" = NULL", mimeCol+" = NULL", nameCol+" = NULL")
		}
	}

	updates = append(updates, "updated_at = NOW()")

	query := fmt.Sprintf(`
		INSERT INTO rebranding_assets (%s) VALUES (%s)
		ON CONFLICT (organization_id, product_id) DO UPDATE SET %s`,
		strings.Join(columns, ", "), strings.Join(values, ", "), strings.Join(updates, ", "))

	return query, args
}

// validateProducts rejects a product id that is not in the catalogue before any
// write happens, so an unknown id is a 400 and not a foreign key violation.
func (s *RebrandingService) validateProducts(productIDs []string) error {
	rows, err := database.DB.Query(`SELECT id FROM rebrandable_products WHERE id = ANY($1)`, pq.Array(productIDs))
	if err != nil {
		return fmt.Errorf("failed to validate products: %w", err)
	}
	defer func() { _ = rows.Close() }()

	known := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("failed to validate products: %w", err)
		}
		known[id] = true
	}

	for _, id := range productIDs {
		if !known[id] {
			return &RebrandingFieldError{Field: "products", Value: id, Err: ErrInvalidRebrandingProduct}
		}
	}
	return nil
}

// DeleteProductAssets deletes all rebranding assets for an organization+product
func (s *RebrandingService) DeleteProductAssets(orgID, productID string) error {
	query := `DELETE FROM rebranding_assets WHERE organization_id = $1 AND product_id = $2`
	result, err := database.DB.Exec(query, orgID, productID)
	if err != nil {
		return fmt.Errorf("failed to delete rebranding assets: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("%w: organization %s product %s", ErrRebrandingAssetsNotFound, orgID, productID)
	}
	return nil
}

// DeleteSingleAsset removes a single asset field for an organization+product
func (s *RebrandingService) DeleteSingleAsset(orgID, productID, assetName string) error {
	if !slices.Contains(rebrandingAssetNames, assetName) {
		return fmt.Errorf("%w: %s", ErrInvalidRebrandingAsset, assetName)
	}

	// The asset has to be there to be deleted: without the IS NOT NULL the
	// statement matches the row whatever it holds, and clearing an asset that is
	// already empty answers 200 while the documented contract — and the caller's
	// stale view of the form — expect a 404.
	query := fmt.Sprintf(
		`UPDATE rebranding_assets SET %s = NULL, %s = NULL, %s = NULL, updated_at = NOW()
		 WHERE organization_id = $1 AND product_id = $2 AND %s IS NOT NULL`,
		assetName, assetName+"_mime", assetName+"_filename", assetName,
	)
	result, err := database.DB.Exec(query, orgID, productID)
	if err != nil {
		return fmt.Errorf("failed to delete asset %s: %w", assetName, err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("%w: organization %s product %s", ErrRebrandingAssetsNotFound, orgID, productID)
	}
	return nil
}

// GetAssetBinary retrieves a single asset binary, its mime type and the time it
// was written. The timestamp becomes the ETag of the response, so a preview
// that reloads all six assets after every save gets six 304s instead.
func (s *RebrandingService) GetAssetBinary(orgID, productID, assetName string) ([]byte, string, time.Time, error) {
	if !slices.Contains(rebrandingAssetNames, assetName) {
		return nil, "", time.Time{}, fmt.Errorf("%w: %s", ErrInvalidRebrandingAsset, assetName)
	}

	query := fmt.Sprintf(
		`SELECT %s, %s, updated_at FROM rebranding_assets WHERE organization_id = $1 AND product_id = $2`,
		assetName, assetName+"_mime",
	)

	var data []byte
	var mime sql.NullString
	var updatedAt time.Time
	err := database.DB.QueryRow(query, orgID, productID).Scan(&data, &mime, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", time.Time{}, ErrRebrandingAssetsNotFound
		}
		return nil, "", time.Time{}, fmt.Errorf("failed to get asset: %w", err)
	}

	if data == nil {
		return nil, "", time.Time{}, ErrRebrandingAssetsNotFound
	}

	mimeType := "application/octet-stream"
	if mime.Valid {
		mimeType = mime.String
	}

	return data, mimeType, updatedAt, nil
}

// GetSystemRebranding returns rebranding data for a system, resolving hierarchy inheritance
func (s *RebrandingService) GetSystemRebranding(systemID string) (*models.SystemRebrandingResponse, error) {
	// Get system's organization_id
	var orgID string
	err := database.DB.QueryRow(`SELECT organization_id FROM systems WHERE id = $1 AND deleted_at IS NULL`, systemID).Scan(&orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get system organization: %w", err)
	}

	// Try to resolve rebranding through hierarchy
	resolvedOrgID, inheritedFrom, err := s.resolveRebrandingOrg(orgID)
	if err != nil {
		return nil, err
	}

	if resolvedOrgID == "" {
		return &models.SystemRebrandingResponse{
			Enabled:      false,
			System:       []models.SystemRebrandingProduct{},
			Applications: []models.SystemRebrandingProduct{},
		}, nil
	}

	// Get all assets for the resolved org
	query := `
		SELECT ra.product_id, ra.product_name, rp.type,
			ra.logo_light_rect IS NOT NULL AS has_logo_light_rect,
			ra.logo_dark_rect IS NOT NULL AS has_logo_dark_rect,
			ra.logo_light_square IS NOT NULL AS has_logo_light_square,
			ra.logo_dark_square IS NOT NULL AS has_logo_dark_square,
			ra.favicon IS NOT NULL AS has_favicon,
			ra.background_image IS NOT NULL AS has_background_image
		FROM rebranding_assets ra
		JOIN rebrandable_products rp ON rp.id = ra.product_id
		WHERE ra.organization_id = $1
	`
	rows, err := database.DB.Query(query, resolvedOrgID)
	if err != nil {
		return nil, fmt.Errorf("failed to query rebranding assets: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var systemProducts []models.SystemRebrandingProduct
	var appProducts []models.SystemRebrandingProduct

	for rows.Next() {
		var productID string
		var productName *string
		var productType string
		var hasLLR, hasLDR, hasLLS, hasLDS, hasFav, hasBG bool

		if err := rows.Scan(&productID, &productName, &productType, &hasLLR, &hasLDR, &hasLLS, &hasLDS, &hasFav, &hasBG); err != nil {
			return nil, fmt.Errorf("failed to scan rebranding asset: %w", err)
		}

		assets := make(map[string]string)
		if hasLLR {
			assets["logo_light_rect"] = fmt.Sprintf("/api/systems/rebranding/%s/logo_light_rect", productID)
		}
		if hasLDR {
			assets["logo_dark_rect"] = fmt.Sprintf("/api/systems/rebranding/%s/logo_dark_rect", productID)
		}
		if hasLLS {
			assets["logo_light_square"] = fmt.Sprintf("/api/systems/rebranding/%s/logo_light_square", productID)
		}
		if hasLDS {
			assets["logo_dark_square"] = fmt.Sprintf("/api/systems/rebranding/%s/logo_dark_square", productID)
		}
		if hasFav {
			assets["favicon"] = fmt.Sprintf("/api/systems/rebranding/%s/favicon", productID)
		}
		if hasBG {
			assets["background_image"] = fmt.Sprintf("/api/systems/rebranding/%s/background_image", productID)
		}

		// Only include products that have at least one asset or a product name
		if len(assets) == 0 && productName == nil {
			continue
		}

		product := models.SystemRebrandingProduct{
			ProductID:   productID,
			ProductName: productName,
			Assets:      assets,
		}

		if productType == "system" {
			systemProducts = append(systemProducts, product)
		} else {
			appProducts = append(appProducts, product)
		}
	}

	if systemProducts == nil {
		systemProducts = []models.SystemRebrandingProduct{}
	}
	if appProducts == nil {
		appProducts = []models.SystemRebrandingProduct{}
	}

	return &models.SystemRebrandingResponse{
		Enabled:       true,
		InheritedFrom: inheritedFrom,
		System:        systemProducts,
		Applications:  appProducts,
	}, nil
}

// GetSystemAssetBinary retrieves a rebranding asset for a system, resolving hierarchy
func (s *RebrandingService) GetSystemAssetBinary(systemID, productID, assetName string) ([]byte, string, time.Time, error) {
	// Get system's organization_id
	var orgID string
	err := database.DB.QueryRow(`SELECT organization_id FROM systems WHERE id = $1 AND deleted_at IS NULL`, systemID).Scan(&orgID)
	if err != nil {
		return nil, "", time.Time{}, fmt.Errorf("failed to get system organization: %w", err)
	}

	// Resolve through hierarchy
	resolvedOrgID, _, err := s.resolveRebrandingOrg(orgID)
	if err != nil {
		return nil, "", time.Time{}, err
	}
	if resolvedOrgID == "" {
		return nil, "", time.Time{}, fmt.Errorf("rebranding not enabled for this system's organization")
	}

	return s.GetAssetBinary(resolvedOrgID, productID, assetName)
}

// ResolveRebranding checks if rebranding is active for an organization (directly or inherited)
// and returns the organization ID that provides the rebranding assets
func (s *RebrandingService) ResolveRebranding(orgID string) (bool, string, error) {
	resolvedOrgID, _, err := s.resolveRebrandingOrg(orgID)
	if err != nil {
		return false, "", err
	}
	if resolvedOrgID == "" {
		return false, "", nil
	}
	return true, resolvedOrgID, nil
}

// BatchResolveRebranding checks rebranding status for multiple organization IDs at once.
// Returns a map of orgID -> (enabled, resolvedOrgID).
// This eliminates N+1 queries when resolving rebranding for a page of applications.
func (s *RebrandingService) BatchResolveRebranding(orgIDs []string) map[string]struct {
	Enabled       bool
	ResolvedOrgID string
} {
	result := make(map[string]struct {
		Enabled       bool
		ResolvedOrgID string
	})

	if len(orgIDs) == 0 {
		return result
	}

	// Deduplicate org IDs
	uniqueMap := make(map[string]bool)
	var unique []string
	for _, id := range orgIDs {
		if id != "" && !uniqueMap[id] {
			uniqueMap[id] = true
			unique = append(unique, id)
		}
	}

	if len(unique) == 0 {
		return result
	}

	// Step 1: Check which orgs have rebranding directly enabled
	placeholders := make([]string, len(unique))
	args := make([]interface{}, len(unique))
	for i, id := range unique {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	directEnabled := make(map[string]bool)
	query := fmt.Sprintf(`SELECT organization_id FROM rebranding_enabled WHERE organization_id IN (%s)`,
		strings.Join(placeholders, ","))
	rows, err := database.DB.Query(query, args...)
	if err == nil {
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var oid string
			if rows.Scan(&oid) == nil {
				directEnabled[oid] = true
				result[oid] = struct {
					Enabled       bool
					ResolvedOrgID string
				}{Enabled: true, ResolvedOrgID: oid}
			}
		}
	}

	// Step 2: For orgs not directly enabled, find their parents
	var needParent []string
	for _, id := range unique {
		if !directEnabled[id] {
			needParent = append(needParent, id)
		}
	}

	if len(needParent) == 0 {
		return result
	}

	// Build parent lookup: check customers first, then resellers
	parentMap := make(map[string]string) // orgID -> parentOrgID

	// Batch lookup customer parents
	placeholders2 := make([]string, len(needParent))
	args2 := make([]interface{}, len(needParent))
	for i, id := range needParent {
		placeholders2[i] = fmt.Sprintf("$%d", i+1)
		args2[i] = id
	}

	custQuery := fmt.Sprintf(`SELECT logto_id, custom_data->>'createdBy' FROM customers WHERE logto_id IN (%s) AND deleted_at IS NULL AND custom_data->>'createdBy' IS NOT NULL`,
		strings.Join(placeholders2, ","))
	custRows, err := database.DB.Query(custQuery, args2...)
	if err == nil {
		defer func() { _ = custRows.Close() }()
		for custRows.Next() {
			var childID, parentID string
			if custRows.Scan(&childID, &parentID) == nil && parentID != "" {
				parentMap[childID] = parentID
			}
		}
	}

	// Batch lookup reseller parents (for orgs not found as customers)
	var resellerCheck []string
	for _, id := range needParent {
		if _, found := parentMap[id]; !found {
			resellerCheck = append(resellerCheck, id)
		}
	}

	if len(resellerCheck) > 0 {
		placeholders3 := make([]string, len(resellerCheck))
		args3 := make([]interface{}, len(resellerCheck))
		for i, id := range resellerCheck {
			placeholders3[i] = fmt.Sprintf("$%d", i+1)
			args3[i] = id
		}

		resQuery := fmt.Sprintf(`SELECT logto_id, custom_data->>'createdBy' FROM resellers WHERE logto_id IN (%s) AND deleted_at IS NULL AND custom_data->>'createdBy' IS NOT NULL`,
			strings.Join(placeholders3, ","))
		resRows, err := database.DB.Query(resQuery, args3...)
		if err == nil {
			defer func() { _ = resRows.Close() }()
			for resRows.Next() {
				var childID, parentID string
				if resRows.Scan(&childID, &parentID) == nil && parentID != "" {
					parentMap[childID] = parentID
				}
			}
		}
	}

	// Step 3: Check if parent orgs have rebranding enabled
	var parentIDs []string
	parentIDSet := make(map[string]bool)
	for _, pid := range parentMap {
		if !parentIDSet[pid] {
			parentIDSet[pid] = true
			parentIDs = append(parentIDs, pid)
		}
	}

	// Also collect grandparent IDs (for customer -> reseller -> distributor chain)
	grandparentMap := make(map[string]string)
	if len(parentIDs) > 0 {
		placeholders4 := make([]string, len(parentIDs))
		args4 := make([]interface{}, len(parentIDs))
		for i, id := range parentIDs {
			placeholders4[i] = fmt.Sprintf("$%d", i+1)
			args4[i] = id
		}

		// Check parent rebranding
		parentEnabledQuery := fmt.Sprintf(`SELECT organization_id FROM rebranding_enabled WHERE organization_id IN (%s)`,
			strings.Join(placeholders4, ","))
		parentEnabledRows, err := database.DB.Query(parentEnabledQuery, args4...)
		parentEnabled := make(map[string]bool)
		if err == nil {
			defer func() { _ = parentEnabledRows.Close() }()
			for parentEnabledRows.Next() {
				var pid string
				if parentEnabledRows.Scan(&pid) == nil {
					parentEnabled[pid] = true
				}
			}
		}

		// Set results for orgs whose parent has rebranding
		for childID, parentID := range parentMap {
			if parentEnabled[parentID] {
				result[childID] = struct {
					Enabled       bool
					ResolvedOrgID string
				}{Enabled: true, ResolvedOrgID: parentID}
			}
		}

		// For parents that are resellers, look up their distributor (grandparent)
		var needGrandparent []string
		for childID, parentID := range parentMap {
			if _, already := result[childID]; !already && !parentEnabled[parentID] {
				needGrandparent = append(needGrandparent, parentID)
			}
		}

		if len(needGrandparent) > 0 {
			gpUnique := make(map[string]bool)
			var gpIDs []string
			for _, id := range needGrandparent {
				if !gpUnique[id] {
					gpUnique[id] = true
					gpIDs = append(gpIDs, id)
				}
			}

			placeholders5 := make([]string, len(gpIDs))
			args5 := make([]interface{}, len(gpIDs))
			for i, id := range gpIDs {
				placeholders5[i] = fmt.Sprintf("$%d", i+1)
				args5[i] = id
			}

			gpQuery := fmt.Sprintf(`SELECT logto_id, custom_data->>'createdBy' FROM resellers WHERE logto_id IN (%s) AND deleted_at IS NULL AND custom_data->>'createdBy' IS NOT NULL`,
				strings.Join(placeholders5, ","))
			gpRows, err := database.DB.Query(gpQuery, args5...)
			if err == nil {
				defer func() { _ = gpRows.Close() }()
				for gpRows.Next() {
					var resID, distID string
					if gpRows.Scan(&resID, &distID) == nil && distID != "" {
						grandparentMap[resID] = distID
					}
				}
			}

			// Check grandparent rebranding
			var gpCheckIDs []string
			gpCheckSet := make(map[string]bool)
			for _, gpID := range grandparentMap {
				if !gpCheckSet[gpID] {
					gpCheckSet[gpID] = true
					gpCheckIDs = append(gpCheckIDs, gpID)
				}
			}

			if len(gpCheckIDs) > 0 {
				placeholders6 := make([]string, len(gpCheckIDs))
				args6 := make([]interface{}, len(gpCheckIDs))
				for i, id := range gpCheckIDs {
					placeholders6[i] = fmt.Sprintf("$%d", i+1)
					args6[i] = id
				}

				gpEnabledQuery := fmt.Sprintf(`SELECT organization_id FROM rebranding_enabled WHERE organization_id IN (%s)`,
					strings.Join(placeholders6, ","))
				gpEnabledRows, err := database.DB.Query(gpEnabledQuery, args6...)
				gpEnabled := make(map[string]bool)
				if err == nil {
					defer func() { _ = gpEnabledRows.Close() }()
					for gpEnabledRows.Next() {
						var gpID string
						if gpEnabledRows.Scan(&gpID) == nil {
							gpEnabled[gpID] = true
						}
					}
				}

				// Set results for orgs whose grandparent has rebranding
				for childID, parentID := range parentMap {
					if _, already := result[childID]; already {
						continue
					}
					if gpID, ok := grandparentMap[parentID]; ok && gpEnabled[gpID] {
						result[childID] = struct {
							Enabled       bool
							ResolvedOrgID string
						}{Enabled: true, ResolvedOrgID: gpID}
					}
				}
			}
		}
	}

	return result
}

// resolveRebrandingOrg walks up the hierarchy to find the first org with rebranding enabled
// Returns (resolved_org_id, inherited_from_label, error)
func (s *RebrandingService) resolveRebrandingOrg(orgID string) (string, *string, error) {
	// Check if this org has rebranding enabled
	enabled, err := s.IsRebrandingEnabled(orgID)
	if err != nil {
		return "", nil, err
	}
	if enabled {
		return orgID, nil, nil // Own rebranding, no inheritance
	}

	// Determine org type and walk up hierarchy
	// Check if org is a customer
	var parentOrgID string
	err = database.DB.QueryRow(
		`SELECT custom_data->>'createdBy' FROM customers WHERE logto_id = $1 AND deleted_at IS NULL`, orgID,
	).Scan(&parentOrgID)
	if err == nil && parentOrgID != "" {
		// It's a customer - check parent (reseller or distributor)
		enabled, err = s.IsRebrandingEnabled(parentOrgID)
		if err != nil {
			return "", nil, err
		}
		if enabled {
			label := s.getOrgLabel(parentOrgID)
			return parentOrgID, &label, nil
		}

		// Check if parent is a reseller - go up to its distributor
		var grandparentOrgID string
		err = database.DB.QueryRow(
			`SELECT custom_data->>'createdBy' FROM resellers WHERE logto_id = $1 AND deleted_at IS NULL`, parentOrgID,
		).Scan(&grandparentOrgID)
		if err == nil && grandparentOrgID != "" {
			enabled, err = s.IsRebrandingEnabled(grandparentOrgID)
			if err != nil {
				return "", nil, err
			}
			if enabled {
				label := s.getOrgLabel(grandparentOrgID)
				return grandparentOrgID, &label, nil
			}
		}
		return "", nil, nil
	}

	// Check if org is a reseller
	err = database.DB.QueryRow(
		`SELECT custom_data->>'createdBy' FROM resellers WHERE logto_id = $1 AND deleted_at IS NULL`, orgID,
	).Scan(&parentOrgID)
	if err == nil && parentOrgID != "" {
		// It's a reseller - check parent distributor
		enabled, err = s.IsRebrandingEnabled(parentOrgID)
		if err != nil {
			return "", nil, err
		}
		if enabled {
			label := s.getOrgLabel(parentOrgID)
			return parentOrgID, &label, nil
		}
	}

	return "", nil, nil
}

// getOrgLabel returns a label like "distributor:org_id" for the inherited_from field
func (s *RebrandingService) getOrgLabel(orgID string) string {
	// Check type
	var count int
	err := database.DB.QueryRow(`SELECT COUNT(*) FROM distributors WHERE logto_id = $1 AND deleted_at IS NULL`, orgID).Scan(&count)
	if err == nil && count > 0 {
		return "distributor:" + orgID
	}
	err = database.DB.QueryRow(`SELECT COUNT(*) FROM resellers WHERE logto_id = $1 AND deleted_at IS NULL`, orgID).Scan(&count)
	if err == nil && count > 0 {
		return "reseller:" + orgID
	}
	return "unknown:" + orgID
}

// helper functions

func nilIfEmptyStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
