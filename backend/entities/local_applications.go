/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package entities

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/models"
)

// LocalApplicationRepository implements repository for applications
type LocalApplicationRepository struct {
	db *sql.DB
}

// NewLocalApplicationRepository creates a new local application repository
func NewLocalApplicationRepository() *LocalApplicationRepository {
	return &LocalApplicationRepository{
		db: database.DB,
	}
}

// GetByID retrieves a specific application by ID
func (r *LocalApplicationRepository) GetByID(id string) (*models.Application, error) {
	query := `
		SELECT a.id, a.system_id, a.module_id, a.instance_of, a.display_name, a.node_id, a.node_label,
		       a.version, a.organization_id, a.organization_type, a.status, a.inventory_data,
		       a.backup_data, a.services_data, a.url, a.notes, a.is_user_facing,
		       a.created_at, a.updated_at, a.first_seen_at, a.last_inventory_at, a.deleted_at,
		       s.name as system_name,
		       COALESCE(d.name, re.name, c.name, 'Owner') as organization_name,
		       COALESCE(d.id::text, re.id::text, c.id::text) as organization_db_id
		FROM applications a
		LEFT JOIN systems s ON a.system_id = s.id
		LEFT JOIN distributors d ON (a.organization_id = d.logto_id OR a.organization_id = d.id::text) AND d.deleted_at IS NULL
		LEFT JOIN resellers re ON (a.organization_id = re.logto_id OR a.organization_id = re.id::text) AND re.deleted_at IS NULL
		LEFT JOIN customers c ON (a.organization_id = c.logto_id OR a.organization_id = c.id::text) AND c.deleted_at IS NULL
		WHERE a.id = $1 AND a.deleted_at IS NULL
	`

	app := &models.Application{}
	var displayName, nodeLabel, version, orgID, orgType, url, notes sql.NullString
	var nodeID sql.NullInt32
	var lastInventoryAt, deletedAt sql.NullTime
	var systemName, orgName, orgDbID sql.NullString
	var inventoryData, backupData, servicesData []byte

	err := r.db.QueryRow(query, id).Scan(
		&app.ID, &app.SystemID, &app.ModuleID, &app.InstanceOf, &displayName, &nodeID, &nodeLabel,
		&version, &orgID, &orgType, &app.Status, &inventoryData,
		&backupData, &servicesData, &url, &notes, &app.IsUserFacing,
		&app.CreatedAt, &app.UpdatedAt, &app.FirstSeenAt, &lastInventoryAt, &deletedAt,
		&systemName, &orgName, &orgDbID,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("application not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query application: %w", err)
	}

	// Convert nullable fields
	if displayName.Valid {
		app.DisplayName = &displayName.String
	}
	if nodeID.Valid {
		nodeIDInt := int(nodeID.Int32)
		app.NodeID = &nodeIDInt
	}
	if nodeLabel.Valid {
		app.NodeLabel = &nodeLabel.String
	}
	if version.Valid {
		app.Version = &version.String
	}
	if orgID.Valid {
		app.OrganizationID = &orgID.String
	}
	if orgType.Valid {
		app.OrganizationType = &orgType.String
	}
	if url.Valid {
		app.URL = &url.String
	}
	if notes.Valid {
		app.Notes = &notes.String
	}
	if lastInventoryAt.Valid {
		app.LastInventoryAt = &lastInventoryAt.Time
	}
	if deletedAt.Valid {
		app.DeletedAt = &deletedAt.Time
	}

	// Set JSONB fields
	app.InventoryData = inventoryData
	app.BackupData = backupData
	app.ServicesData = servicesData

	// Set system summary
	if systemName.Valid {
		app.System = &models.SystemSummary{
			ID:   app.SystemID,
			Name: systemName.String,
		}
	}

	// Set organization summary
	if orgID.Valid && orgName.Valid {
		app.Organization = &models.OrganizationSummary{
			ID:      orgDbID.String,
			LogtoID: orgID.String,
			Name:    orgName.String,
			Type:    orgType.String,
		}
	}

	return app, nil
}

// GetBySystemAndModuleID retrieves an application by system_id and module_id
func (r *LocalApplicationRepository) GetBySystemAndModuleID(systemID, moduleID string) (*models.Application, error) {
	query := `
		SELECT id FROM applications
		WHERE system_id = $1 AND module_id = $2 AND deleted_at IS NULL
	`
	var id string
	err := r.db.QueryRow(query, systemID, moduleID).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query application: %w", err)
	}
	return r.GetByID(id)
}

// List returns paginated list of applications with filters
func (r *LocalApplicationRepository) List(
	allowedSystemIDs []string,
	page, pageSize int,
	search, sortBy, sortDirection string,
	filterTypes, filterVersions, filterSystemIDs, filterOrgIDs, filterStatuses []string,
	userFacingOnly bool,
) ([]*models.Application, int, error) {
	if len(allowedSystemIDs) == 0 {
		return []*models.Application{}, 0, nil
	}

	offset := (page - 1) * pageSize

	// Build placeholders for allowed systems
	placeholders := make([]string, len(allowedSystemIDs))
	baseArgs := make([]interface{}, len(allowedSystemIDs))
	for i, sysID := range allowedSystemIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		baseArgs[i] = sysID
	}
	placeholdersStr := strings.Join(placeholders, ",")

	// Build WHERE clause
	whereClause := fmt.Sprintf("a.deleted_at IS NULL AND a.system_id IN (%s)", placeholdersStr)
	args := make([]interface{}, len(baseArgs))
	copy(args, baseArgs)

	// User-facing filter
	if userFacingOnly {
		whereClause += " AND a.is_user_facing = TRUE"
	}

	// Search condition
	if search != "" {
		searchPattern := "%" + search + "%"
		whereClause += fmt.Sprintf(" AND (a.module_id ILIKE $%d OR a.display_name ILIKE $%d OR a.instance_of ILIKE $%d OR s.name ILIKE $%d)",
			len(args)+1, len(args)+2, len(args)+3, len(args)+4)
		args = append(args, searchPattern, searchPattern, searchPattern, searchPattern)
	}

	// Filter by types (instance_of)
	if len(filterTypes) > 0 {
		typePlaceholders := make([]string, len(filterTypes))
		baseIndex := len(args)
		for i, t := range filterTypes {
			typePlaceholders[i] = fmt.Sprintf("$%d", baseIndex+i+1)
			args = append(args, t)
		}
		whereClause += fmt.Sprintf(" AND a.instance_of IN (%s)", strings.Join(typePlaceholders, ","))
	}

	// Filter by versions
	if len(filterVersions) > 0 {
		versionPlaceholders := make([]string, len(filterVersions))
		baseIndex := len(args)
		for i, v := range filterVersions {
			versionPlaceholders[i] = fmt.Sprintf("$%d", baseIndex+i+1)
			args = append(args, v)
		}
		whereClause += fmt.Sprintf(" AND a.version IN (%s)", strings.Join(versionPlaceholders, ","))
	}

	// Filter by system IDs (additional filter within allowed)
	if len(filterSystemIDs) > 0 {
		sysPlaceholders := make([]string, len(filterSystemIDs))
		baseIndex := len(args)
		for i, sid := range filterSystemIDs {
			sysPlaceholders[i] = fmt.Sprintf("$%d", baseIndex+i+1)
			args = append(args, sid)
		}
		whereClause += fmt.Sprintf(" AND a.system_id IN (%s)", strings.Join(sysPlaceholders, ","))
	}

	// Filter by organization IDs (handle "null" for unassigned)
	if len(filterOrgIDs) > 0 {
		var orgConditions []string
		var hasNull bool
		var nonNullOrgIDs []string

		for _, orgID := range filterOrgIDs {
			if orgID == "null" || orgID == "" {
				hasNull = true
			} else {
				nonNullOrgIDs = append(nonNullOrgIDs, orgID)
			}
		}

		if hasNull {
			orgConditions = append(orgConditions, "a.organization_id IS NULL")
		}

		if len(nonNullOrgIDs) > 0 {
			orgPlaceholders := make([]string, len(nonNullOrgIDs))
			baseIndex := len(args)
			for i, oid := range nonNullOrgIDs {
				orgPlaceholders[i] = fmt.Sprintf("$%d", baseIndex+i+1)
				args = append(args, oid)
			}
			orgConditions = append(orgConditions, fmt.Sprintf("a.organization_id IN (%s)", strings.Join(orgPlaceholders, ",")))
		}

		if len(orgConditions) > 0 {
			whereClause += fmt.Sprintf(" AND (%s)", strings.Join(orgConditions, " OR "))
		}
	}

	// Filter by statuses
	if len(filterStatuses) > 0 {
		statusPlaceholders := make([]string, len(filterStatuses))
		baseIndex := len(args)
		for i, st := range filterStatuses {
			statusPlaceholders[i] = fmt.Sprintf("$%d", baseIndex+i+1)
			args = append(args, st)
		}
		whereClause += fmt.Sprintf(" AND a.status IN (%s)", strings.Join(statusPlaceholders, ","))
	}

	// Get total count
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM applications a
		LEFT JOIN systems s ON a.system_id = s.id
		WHERE %s`, whereClause)

	var totalCount int
	err := r.db.QueryRow(countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get applications count: %w", err)
	}

	// Build ORDER BY clause
	orderBy := "a.created_at DESC"
	if sortBy != "" {
		columnMap := map[string]string{
			"display_name":      "LOWER(COALESCE(NULLIF(TRIM(a.display_name), ''), a.module_id))",
			"module_id":         "LOWER(a.module_id)",
			"instance_of":       "LOWER(a.instance_of)",
			"version":           "a.version",
			"status":            "a.status",
			"system_name":       "LOWER(s.name)",
			"organization_name": "LOWER(COALESCE(d.name, re.name, c.name))",
			"created_at":        "a.created_at",
			"updated_at":        "a.updated_at",
			"last_inventory_at": "a.last_inventory_at",
		}

		if column, exists := columnMap[sortBy]; exists {
			direction := "ASC"
			if sortDirection == "desc" {
				direction = "DESC"
			}
			orderBy = fmt.Sprintf("%s %s", column, direction)
		}
	}

	// Build main query
	query := fmt.Sprintf(`
		SELECT a.id, a.system_id, a.module_id, a.instance_of, a.display_name, a.node_id, a.node_label,
		       a.version, a.organization_id, a.organization_type, a.status, a.inventory_data,
		       a.backup_data, a.services_data, a.url, a.notes, a.is_user_facing,
		       a.created_at, a.updated_at, a.first_seen_at, a.last_inventory_at,
		       s.name as system_name,
		       COALESCE(d.name, re.name, c.name, 'Owner') as organization_name,
		       COALESCE(d.id::text, re.id::text, c.id::text) as organization_db_id
		FROM applications a
		LEFT JOIN systems s ON a.system_id = s.id
		LEFT JOIN distributors d ON (a.organization_id = d.logto_id OR a.organization_id = d.id::text) AND d.deleted_at IS NULL
		LEFT JOIN resellers re ON (a.organization_id = re.logto_id OR a.organization_id = re.id::text) AND re.deleted_at IS NULL
		LEFT JOIN customers c ON (a.organization_id = c.logto_id OR a.organization_id = c.id::text) AND c.deleted_at IS NULL
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, len(args)+1, len(args)+2)

	listArgs := make([]interface{}, len(args)+2)
	copy(listArgs, args)
	listArgs[len(args)] = pageSize
	listArgs[len(args)+1] = offset

	rows, err := r.db.Query(query, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query applications: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var apps []*models.Application
	for rows.Next() {
		app := &models.Application{}
		var displayName, nodeLabel, version, orgID, orgType, url, notes sql.NullString
		var nodeID sql.NullInt32
		var lastInventoryAt sql.NullTime
		var systemName, orgName, orgDbID sql.NullString
		var inventoryData, backupData, servicesData []byte

		err := rows.Scan(
			&app.ID, &app.SystemID, &app.ModuleID, &app.InstanceOf, &displayName, &nodeID, &nodeLabel,
			&version, &orgID, &orgType, &app.Status, &inventoryData,
			&backupData, &servicesData, &url, &notes, &app.IsUserFacing,
			&app.CreatedAt, &app.UpdatedAt, &app.FirstSeenAt, &lastInventoryAt,
			&systemName, &orgName, &orgDbID,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan application: %w", err)
		}

		// Convert nullable fields
		if displayName.Valid {
			app.DisplayName = &displayName.String
		}
		if nodeID.Valid {
			nodeIDInt := int(nodeID.Int32)
			app.NodeID = &nodeIDInt
		}
		if nodeLabel.Valid {
			app.NodeLabel = &nodeLabel.String
		}
		if version.Valid {
			app.Version = &version.String
		}
		if orgID.Valid {
			app.OrganizationID = &orgID.String
		}
		if orgType.Valid {
			app.OrganizationType = &orgType.String
		}
		if url.Valid {
			app.URL = &url.String
		}
		if notes.Valid {
			app.Notes = &notes.String
		}
		if lastInventoryAt.Valid {
			app.LastInventoryAt = &lastInventoryAt.Time
		}

		app.InventoryData = inventoryData
		app.BackupData = backupData
		app.ServicesData = servicesData

		// Set system summary
		if systemName.Valid {
			app.System = &models.SystemSummary{
				ID:   app.SystemID,
				Name: systemName.String,
			}
		}

		// Set organization summary
		if orgID.Valid && orgName.Valid {
			app.Organization = &models.OrganizationSummary{
				ID:      orgDbID.String,
				LogtoID: orgID.String,
				Name:    orgName.String,
				Type:    orgType.String,
			}
		}

		apps = append(apps, app)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating applications: %w", err)
	}

	return apps, totalCount, nil
}

// GetTotals returns statistics for applications
func (r *LocalApplicationRepository) GetTotals(allowedSystemIDs []string, userFacingOnly bool) (*models.ApplicationTotals, error) {
	if len(allowedSystemIDs) == 0 {
		return &models.ApplicationTotals{
			Total:      0,
			Unassigned: 0,
			Assigned:   0,
			WithErrors: 0,
			ByType:     make(map[string]int64),
			ByStatus:   make(map[string]int64),
		}, nil
	}

	// Build placeholders
	placeholders := make([]string, len(allowedSystemIDs))
	args := make([]interface{}, len(allowedSystemIDs))
	for i, sysID := range allowedSystemIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = sysID
	}
	placeholdersStr := strings.Join(placeholders, ",")

	userFacingClause := ""
	if userFacingOnly {
		userFacingClause = " AND is_user_facing = TRUE"
	}

	// Get main counts
	query := fmt.Sprintf(`
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE organization_id IS NULL) as unassigned,
			COUNT(*) FILTER (WHERE organization_id IS NOT NULL) as assigned,
			COUNT(*) FILTER (WHERE services_data->>'has_errors' = 'true') as with_errors
		FROM applications
		WHERE deleted_at IS NULL AND system_id IN (%s)%s
	`, placeholdersStr, userFacingClause)

	totals := &models.ApplicationTotals{
		ByType:   make(map[string]int64),
		ByStatus: make(map[string]int64),
	}

	err := r.db.QueryRow(query, args...).Scan(
		&totals.Total, &totals.Unassigned, &totals.Assigned, &totals.WithErrors,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get applications totals: %w", err)
	}

	// Get counts by type
	typeQuery := fmt.Sprintf(`
		SELECT instance_of, COUNT(*) as count
		FROM applications
		WHERE deleted_at IS NULL AND system_id IN (%s)%s
		GROUP BY instance_of
		ORDER BY count DESC
	`, placeholdersStr, userFacingClause)

	typeRows, err := r.db.Query(typeQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get applications by type: %w", err)
	}
	defer func() { _ = typeRows.Close() }()

	for typeRows.Next() {
		var instanceOf string
		var count int64
		if err := typeRows.Scan(&instanceOf, &count); err != nil {
			return nil, fmt.Errorf("failed to scan type count: %w", err)
		}
		totals.ByType[instanceOf] = count
	}

	// Get counts by status
	statusQuery := fmt.Sprintf(`
		SELECT status, COUNT(*) as count
		FROM applications
		WHERE deleted_at IS NULL AND system_id IN (%s)%s
		GROUP BY status
	`, placeholdersStr, userFacingClause)

	statusRows, err := r.db.Query(statusQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get applications by status: %w", err)
	}
	defer func() { _ = statusRows.Close() }()

	for statusRows.Next() {
		var status string
		var count int64
		if err := statusRows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan status count: %w", err)
		}
		totals.ByStatus[status] = count
	}

	return totals, nil
}

// GetTrend returns trend data for applications over a specified period
func (r *LocalApplicationRepository) GetTrend(allowedSystemIDs []string, period int) ([]struct {
	Date  string
	Count int
}, int, int, error) {
	// If no allowed systems, return empty data
	if len(allowedSystemIDs) == 0 {
		return []struct {
			Date  string
			Count int
		}{}, 0, 0, nil
	}

	// Determine interval for date series based on period
	var interval string
	switch period {
	case 7, 30:
		interval = "1 day"
	case 180:
		interval = "1 week"
	case 365:
		interval = "1 month"
	default:
		return nil, 0, 0, fmt.Errorf("invalid period: %d", period)
	}

	// Build placeholders for allowed system IDs
	placeholders := make([]string, len(allowedSystemIDs))
	args := make([]interface{}, len(allowedSystemIDs))
	for i, sysID := range allowedSystemIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = sysID
	}
	placeholdersStr := strings.Join(placeholders, ",")

	// Query to get cumulative count for each date in the period
	query := fmt.Sprintf(`
		WITH date_series AS (
			SELECT generate_series(
				CURRENT_DATE - INTERVAL '%d days',
				CURRENT_DATE,
				INTERVAL '%s'
			)::date AS date
		)
		SELECT
			ds.date::text,
			COALESCE((
				SELECT COUNT(*)
				FROM applications
				WHERE deleted_at IS NULL
				  AND system_id IN (%s)
				  AND created_at::date <= ds.date
			), 0) AS count
		FROM date_series ds
		ORDER BY ds.date
	`, period, interval, placeholdersStr)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to query applications trend data: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var dataPoints []struct {
		Date  string
		Count int
	}

	for rows.Next() {
		var date string
		var count int
		if err := rows.Scan(&date, &count); err != nil {
			return nil, 0, 0, fmt.Errorf("failed to scan applications trend data: %w", err)
		}
		dataPoints = append(dataPoints, struct {
			Date  string
			Count int
		}{Date: date, Count: count})
	}

	if err := rows.Err(); err != nil {
		return nil, 0, 0, fmt.Errorf("error iterating applications trend data: %w", err)
	}

	// Calculate current and previous totals
	var currentTotal, previousTotal int
	if len(dataPoints) > 0 {
		currentTotal = dataPoints[len(dataPoints)-1].Count
		previousTotal = dataPoints[0].Count
	}

	return dataPoints, currentTotal, previousTotal, nil
}

// GetDistinctTypes returns distinct application types with is_user_facing from database
func (r *LocalApplicationRepository) GetDistinctTypes(allowedSystemIDs []string, userFacingOnly bool) ([]models.ApplicationType, error) {
	if len(allowedSystemIDs) == 0 {
		return []models.ApplicationType{}, nil
	}

	placeholders := make([]string, len(allowedSystemIDs))
	args := make([]interface{}, len(allowedSystemIDs))
	for i, sysID := range allowedSystemIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = sysID
	}
	placeholdersStr := strings.Join(placeholders, ",")

	userFacingClause := ""
	if userFacingOnly {
		userFacingClause = " AND is_user_facing = TRUE"
	}

	query := fmt.Sprintf(`
		SELECT instance_of, is_user_facing, COUNT(*) as count
		FROM applications
		WHERE deleted_at IS NULL AND system_id IN (%s)%s
		GROUP BY instance_of, is_user_facing
		ORDER BY instance_of
	`, placeholdersStr, userFacingClause)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get distinct types: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var types []models.ApplicationType
	for rows.Next() {
		var t models.ApplicationType
		if err := rows.Scan(&t.InstanceOf, &t.IsUserFacing, &t.Count); err != nil {
			return nil, fmt.Errorf("failed to scan type: %w", err)
		}
		types = append(types, t)
	}

	return types, nil
}

// GetDistinctVersions returns distinct application versions
func (r *LocalApplicationRepository) GetDistinctVersions(allowedSystemIDs []string, userFacingOnly bool) ([]string, error) {
	if len(allowedSystemIDs) == 0 {
		return []string{}, nil
	}

	placeholders := make([]string, len(allowedSystemIDs))
	args := make([]interface{}, len(allowedSystemIDs))
	for i, sysID := range allowedSystemIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = sysID
	}
	placeholdersStr := strings.Join(placeholders, ",")

	userFacingClause := ""
	if userFacingOnly {
		userFacingClause = " AND is_user_facing = TRUE"
	}

	query := fmt.Sprintf(`
		SELECT DISTINCT version
		FROM applications
		WHERE deleted_at IS NULL AND version IS NOT NULL AND system_id IN (%s)%s
		ORDER BY version DESC
	`, placeholdersStr, userFacingClause)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get distinct versions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var versions []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("failed to scan version: %w", err)
		}
		versions = append(versions, v)
	}

	return versions, nil
}

// Create creates a new application
func (r *LocalApplicationRepository) Create(app *models.Application) error {
	query := `
		INSERT INTO applications (
			id, system_id, module_id, instance_of, display_name, node_id, node_label,
			version, organization_id, organization_type, status, inventory_data,
			backup_data, services_data, url, notes, is_user_facing,
			created_at, updated_at, first_seen_at, last_inventory_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21
		)
	`

	_, err := r.db.Exec(query,
		app.ID, app.SystemID, app.ModuleID, app.InstanceOf, app.DisplayName, app.NodeID, app.NodeLabel,
		app.Version, app.OrganizationID, app.OrganizationType, app.Status, app.InventoryData,
		app.BackupData, app.ServicesData, app.URL, app.Notes, app.IsUserFacing,
		app.CreatedAt, app.UpdatedAt, app.FirstSeenAt, app.LastInventoryAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create application: %w", err)
	}

	return nil
}

// Update updates an existing application (only notes is editable, other fields come from inventory)
func (r *LocalApplicationRepository) Update(id string, req *models.UpdateApplicationRequest) error {
	query := `
		UPDATE applications
		SET notes = COALESCE($2, notes),
		    updated_at = $3
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.Exec(query, id, req.Notes, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update application: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("application not found")
	}

	return nil
}

// AssignOrganization assigns an organization to an application
func (r *LocalApplicationRepository) AssignOrganization(id, organizationID, organizationType string) error {
	query := `
		UPDATE applications
		SET organization_id = $2,
		    organization_type = $3,
		    status = 'assigned',
		    updated_at = $4
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.Exec(query, id, organizationID, organizationType, time.Now())
	if err != nil {
		return fmt.Errorf("failed to assign organization: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("application not found")
	}

	return nil
}

// UnassignOrganization removes organization assignment from an application
func (r *LocalApplicationRepository) UnassignOrganization(id string) error {
	query := `
		UPDATE applications
		SET organization_id = NULL,
		    organization_type = NULL,
		    status = 'unassigned',
		    updated_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.Exec(query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to unassign organization: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("application not found")
	}

	return nil
}

// Delete soft-deletes an application
func (r *LocalApplicationRepository) Delete(id string) error {
	query := `
		UPDATE applications
		SET deleted_at = $2, updated_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`

	now := time.Now()
	result, err := r.db.Exec(query, id, now)
	if err != nil {
		return fmt.Errorf("failed to delete application: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("application not found")
	}

	return nil
}

// UpdateFromInventory updates application data from inventory
func (r *LocalApplicationRepository) UpdateFromInventory(
	systemID, moduleID string,
	nodeID *int,
	nodeLabel, version *string,
	inventoryData json.RawMessage,
	isUserFacing bool,
) error {
	query := `
		UPDATE applications
		SET node_id = $3,
		    node_label = $4,
		    version = $5,
		    inventory_data = $6,
		    is_user_facing = $7,
		    last_inventory_at = $8,
		    updated_at = $8
		WHERE system_id = $1 AND module_id = $2 AND deleted_at IS NULL
	`

	now := time.Now()
	result, err := r.db.Exec(query, systemID, moduleID, nodeID, nodeLabel, version, inventoryData, isUserFacing, now)
	if err != nil {
		return fmt.Errorf("failed to update application from inventory: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("application not found for system %s, module %s", systemID, moduleID)
	}

	return nil
}

// UpsertFromInventory creates or updates application from inventory data
func (r *LocalApplicationRepository) UpsertFromInventory(
	id, systemID, moduleID, instanceOf string,
	nodeID *int,
	nodeLabel, version *string,
	inventoryData json.RawMessage,
	isUserFacing bool,
) error {
	query := `
		INSERT INTO applications (
			id, system_id, module_id, instance_of, node_id, node_label, version,
			inventory_data, is_user_facing, status,
			created_at, updated_at, first_seen_at, last_inventory_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, 'unassigned', $10, $10, $10, $10
		)
		ON CONFLICT (system_id, module_id) WHERE deleted_at IS NULL
		DO UPDATE SET
			node_id = EXCLUDED.node_id,
			node_label = EXCLUDED.node_label,
			version = EXCLUDED.version,
			inventory_data = EXCLUDED.inventory_data,
			is_user_facing = EXCLUDED.is_user_facing,
			last_inventory_at = EXCLUDED.last_inventory_at,
			updated_at = EXCLUDED.updated_at
	`

	now := time.Now()
	_, err := r.db.Exec(query, id, systemID, moduleID, instanceOf, nodeID, nodeLabel, version, inventoryData, isUserFacing, now)
	if err != nil {
		return fmt.Errorf("failed to upsert application: %w", err)
	}

	return nil
}

// GetSystemIDForApplication returns the system_id for a given application
func (r *LocalApplicationRepository) GetSystemIDForApplication(appID string) (string, error) {
	var systemID string
	err := r.db.QueryRow("SELECT system_id FROM applications WHERE id = $1 AND deleted_at IS NULL", appID).Scan(&systemID)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("application not found")
	}
	if err != nil {
		return "", fmt.Errorf("failed to get system ID: %w", err)
	}
	return systemID, nil
}
