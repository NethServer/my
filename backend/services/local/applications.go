/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package local

import (
	"fmt"
	"strings"

	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/entities"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
)

// LocalApplicationsService handles business logic for applications management
type LocalApplicationsService struct {
	repo *entities.LocalApplicationRepository
}

// NewApplicationsService creates a new applications service
func NewApplicationsService() *LocalApplicationsService {
	return &LocalApplicationsService{
		repo: entities.NewLocalApplicationRepository(),
	}
}

// GetApplications retrieves paginated list of applications with filters
func (s *LocalApplicationsService) GetApplications(
	userOrgRole, userOrgID string,
	page, pageSize int,
	search, sortBy, sortDirection string,
	filterTypes, filterVersions, filterSystemIDs, filterOrgIDs, filterStatuses []string,
) ([]*models.Application, int, error) {
	// Get allowed system IDs based on user's organization hierarchy
	allowedSystemIDs, err := s.getAllowedSystemIDs(userOrgRole, userOrgID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get allowed systems: %w", err)
	}

	// Only show user-facing applications
	return s.repo.List(
		allowedSystemIDs,
		page, pageSize,
		search, sortBy, sortDirection,
		filterTypes, filterVersions, filterSystemIDs, filterOrgIDs, filterStatuses,
		true, // userFacingOnly
	)
}

// GetApplication retrieves a single application by ID with access validation
func (s *LocalApplicationsService) GetApplication(id, userOrgRole, userOrgID string) (*models.Application, error) {
	app, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Validate user has access to the system this application belongs to
	if !s.canAccessSystem(app.SystemID, userOrgRole, userOrgID) {
		return nil, fmt.Errorf("access denied: user cannot access this application")
	}

	return app, nil
}

// UpdateApplication updates an application's display name, notes, or URL
func (s *LocalApplicationsService) UpdateApplication(id string, req *models.UpdateApplicationRequest, userOrgRole, userOrgID string) error {
	// Validate access
	app, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	if !s.canAccessSystem(app.SystemID, userOrgRole, userOrgID) {
		return fmt.Errorf("access denied: user cannot modify this application")
	}

	err = s.repo.Update(id, req)
	if err != nil {
		return err
	}

	logger.Info().
		Str("application_id", id).
		Str("module_id", app.ModuleID).
		Str("user_org_id", userOrgID).
		Msg("Application updated successfully")

	return nil
}

// AssignOrganization assigns an organization to an application
func (s *LocalApplicationsService) AssignOrganization(id string, req *models.AssignApplicationRequest, userOrgRole, userOrgID string) error {
	// Validate access
	app, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	if !s.canAccessSystem(app.SystemID, userOrgRole, userOrgID) {
		return fmt.Errorf("access denied: user cannot modify this application")
	}

	// Validate that user can assign to the target organization
	if !s.canAssignToOrganization(userOrgRole, userOrgID, req.OrganizationID) {
		return fmt.Errorf("access denied: user cannot assign application to this organization")
	}

	// Get organization type
	orgType, err := s.getOrganizationType(req.OrganizationID)
	if err != nil {
		return fmt.Errorf("failed to get organization type: %w", err)
	}

	err = s.repo.AssignOrganization(id, req.OrganizationID, orgType)
	if err != nil {
		return err
	}

	logger.Info().
		Str("application_id", id).
		Str("module_id", app.ModuleID).
		Str("organization_id", req.OrganizationID).
		Str("organization_type", orgType).
		Str("assigned_by_org", userOrgID).
		Msg("Application assigned to organization successfully")

	return nil
}

// UnassignOrganization removes organization assignment from an application
func (s *LocalApplicationsService) UnassignOrganization(id, userOrgRole, userOrgID string) error {
	// Validate access
	app, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	if !s.canAccessSystem(app.SystemID, userOrgRole, userOrgID) {
		return fmt.Errorf("access denied: user cannot modify this application")
	}

	err = s.repo.UnassignOrganization(id)
	if err != nil {
		return err
	}

	logger.Info().
		Str("application_id", id).
		Str("module_id", app.ModuleID).
		Str("unassigned_by_org", userOrgID).
		Msg("Application unassigned from organization successfully")

	return nil
}

// DeleteApplication soft-deletes an application
func (s *LocalApplicationsService) DeleteApplication(id, userOrgRole, userOrgID string) error {
	// Validate access
	app, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	if !s.canAccessSystem(app.SystemID, userOrgRole, userOrgID) {
		return fmt.Errorf("access denied: user cannot delete this application")
	}

	err = s.repo.Delete(id)
	if err != nil {
		return err
	}

	logger.Info().
		Str("application_id", id).
		Str("module_id", app.ModuleID).
		Str("deleted_by_org", userOrgID).
		Msg("Application deleted successfully")

	return nil
}

// GetApplicationTotals returns statistics for applications
func (s *LocalApplicationsService) GetApplicationTotals(userOrgRole, userOrgID string) (*models.ApplicationTotals, error) {
	allowedSystemIDs, err := s.getAllowedSystemIDs(userOrgRole, userOrgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get allowed systems: %w", err)
	}

	return s.repo.GetTotals(allowedSystemIDs, true) // userFacingOnly
}

// GetApplicationTypes returns distinct application types
func (s *LocalApplicationsService) GetApplicationTypes(userOrgRole, userOrgID string) ([]models.ApplicationType, error) {
	allowedSystemIDs, err := s.getAllowedSystemIDs(userOrgRole, userOrgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get allowed systems: %w", err)
	}

	return s.repo.GetDistinctTypes(allowedSystemIDs, true)
}

// GetApplicationVersions returns distinct application versions grouped by instance_of
func (s *LocalApplicationsService) GetApplicationVersions(userOrgRole, userOrgID string) (map[string]entities.ApplicationVersionGroup, error) {
	allowedSystemIDs, err := s.getAllowedSystemIDs(userOrgRole, userOrgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get allowed systems: %w", err)
	}

	return s.repo.GetDistinctVersions(allowedSystemIDs, true)
}

// GetApplicationsTrend returns trend data for applications over a specified period
func (s *LocalApplicationsService) GetApplicationsTrend(userOrgRole, userOrgID string, period int) ([]struct {
	Date  string
	Count int
}, int, int, error) {
	allowedSystemIDs, err := s.getAllowedSystemIDs(userOrgRole, userOrgID)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to get allowed systems: %w", err)
	}

	return s.repo.GetTrend(allowedSystemIDs, period)
}

// =============================================================================
// PRIVATE HELPER METHODS
// =============================================================================

// getAllowedSystemIDs returns list of system IDs the user can access based on hierarchy
func (s *LocalApplicationsService) getAllowedSystemIDs(userOrgRole, userOrgID string) ([]string, error) {
	// Get allowed organization IDs based on hierarchy
	allowedOrgIDs, err := s.getAllowedOrganizationIDs(userOrgRole, userOrgID)
	if err != nil {
		return nil, err
	}

	if len(allowedOrgIDs) == 0 {
		return []string{}, nil
	}

	// Build query to get system IDs for allowed organizations
	placeholders := make([]string, len(allowedOrgIDs))
	args := make([]interface{}, len(allowedOrgIDs))
	for i, orgID := range allowedOrgIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = orgID
	}

	query := fmt.Sprintf(`
		SELECT id FROM systems
		WHERE deleted_at IS NULL AND created_by ->> 'organization_id' IN (%s)
	`, strings.Join(placeholders, ","))

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query systems: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var systemIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan system ID: %w", err)
		}
		systemIDs = append(systemIDs, id)
	}

	return systemIDs, nil
}

// getAllowedOrganizationIDs returns list of organization IDs the user can access
func (s *LocalApplicationsService) getAllowedOrganizationIDs(userOrgRole, userOrgID string) ([]string, error) {
	var allowedOrgIDs []string

	// Normalize role to lowercase for comparison (JWT contains "Owner", "Distributor", etc.)
	switch strings.ToLower(userOrgRole) {
	case "owner":
		// Owner can access all organizations
		// Get all distributor, reseller, customer logto_ids
		query := `
			SELECT logto_id FROM distributors WHERE deleted_at IS NULL AND logto_id IS NOT NULL
			UNION
			SELECT logto_id FROM resellers WHERE deleted_at IS NULL AND logto_id IS NOT NULL
			UNION
			SELECT logto_id FROM customers WHERE deleted_at IS NULL AND logto_id IS NOT NULL
		`
		rows, err := database.DB.Query(query)
		if err != nil {
			return nil, fmt.Errorf("failed to query organizations: %w", err)
		}
		defer func() { _ = rows.Close() }()

		for rows.Next() {
			var orgID string
			if err := rows.Scan(&orgID); err != nil {
				return nil, fmt.Errorf("failed to scan org ID: %w", err)
			}
			allowedOrgIDs = append(allowedOrgIDs, orgID)
		}
		// Also include owner's own org ID
		allowedOrgIDs = append(allowedOrgIDs, userOrgID)

	case "distributor":
		// Distributor can access own org and child resellers/customers
		allowedOrgIDs = append(allowedOrgIDs, userOrgID)

		// Get child resellers
		resellerQuery := `
			SELECT logto_id FROM resellers
			WHERE deleted_at IS NULL AND logto_id IS NOT NULL
			AND custom_data->>'createdBy' = $1
		`
		resellerRows, err := database.DB.Query(resellerQuery, userOrgID)
		if err != nil {
			return nil, fmt.Errorf("failed to query resellers: %w", err)
		}
		defer func() { _ = resellerRows.Close() }()

		var resellerIDs []string
		for resellerRows.Next() {
			var resID string
			if err := resellerRows.Scan(&resID); err != nil {
				return nil, fmt.Errorf("failed to scan reseller ID: %w", err)
			}
			resellerIDs = append(resellerIDs, resID)
			allowedOrgIDs = append(allowedOrgIDs, resID)
		}

		// Get child customers (direct and through resellers)
		if len(resellerIDs) > 0 {
			placeholders := make([]string, len(resellerIDs)+1)
			args := make([]interface{}, len(resellerIDs)+1)
			args[0] = userOrgID
			placeholders[0] = "$1"
			for i, rid := range resellerIDs {
				placeholders[i+1] = fmt.Sprintf("$%d", i+2)
				args[i+1] = rid
			}

			customerQuery := fmt.Sprintf(`
				SELECT logto_id FROM customers
				WHERE deleted_at IS NULL AND logto_id IS NOT NULL
				AND (custom_data->>'createdBy' = $1 OR custom_data->>'createdBy' IN (%s))
			`, strings.Join(placeholders[1:], ","))

			customerRows, err := database.DB.Query(customerQuery, args...)
			if err != nil {
				return nil, fmt.Errorf("failed to query customers: %w", err)
			}
			defer func() { _ = customerRows.Close() }()

			for customerRows.Next() {
				var custID string
				if err := customerRows.Scan(&custID); err != nil {
					return nil, fmt.Errorf("failed to scan customer ID: %w", err)
				}
				allowedOrgIDs = append(allowedOrgIDs, custID)
			}
		} else {
			// Only direct customers
			customerQuery := `
				SELECT logto_id FROM customers
				WHERE deleted_at IS NULL AND logto_id IS NOT NULL
				AND custom_data->>'createdBy' = $1
			`
			customerRows, err := database.DB.Query(customerQuery, userOrgID)
			if err != nil {
				return nil, fmt.Errorf("failed to query customers: %w", err)
			}
			defer func() { _ = customerRows.Close() }()

			for customerRows.Next() {
				var custID string
				if err := customerRows.Scan(&custID); err != nil {
					return nil, fmt.Errorf("failed to scan customer ID: %w", err)
				}
				allowedOrgIDs = append(allowedOrgIDs, custID)
			}
		}

	case "reseller":
		// Reseller can access own org and child customers
		allowedOrgIDs = append(allowedOrgIDs, userOrgID)

		customerQuery := `
			SELECT logto_id FROM customers
			WHERE deleted_at IS NULL AND logto_id IS NOT NULL
			AND custom_data->>'createdBy' = $1
		`
		customerRows, err := database.DB.Query(customerQuery, userOrgID)
		if err != nil {
			return nil, fmt.Errorf("failed to query customers: %w", err)
		}
		defer func() { _ = customerRows.Close() }()

		for customerRows.Next() {
			var custID string
			if err := customerRows.Scan(&custID); err != nil {
				return nil, fmt.Errorf("failed to scan customer ID: %w", err)
			}
			allowedOrgIDs = append(allowedOrgIDs, custID)
		}

	case "customer":
		// Customer can only access own org
		allowedOrgIDs = append(allowedOrgIDs, userOrgID)

	default:
		return nil, fmt.Errorf("unknown organization role: %s", userOrgRole)
	}

	return allowedOrgIDs, nil
}

// canAccessSystem checks if user can access a specific system
func (s *LocalApplicationsService) canAccessSystem(systemID, userOrgRole, userOrgID string) bool {
	// Get the system's created_by organization
	var creatorOrgID string
	err := database.DB.QueryRow(`
		SELECT created_by->>'organization_id' FROM systems WHERE id = $1 AND deleted_at IS NULL
	`, systemID).Scan(&creatorOrgID)
	if err != nil {
		return false
	}

	// Check if creator org is in user's allowed orgs
	allowedOrgIDs, err := s.getAllowedOrganizationIDs(userOrgRole, userOrgID)
	if err != nil {
		return false
	}

	for _, allowedID := range allowedOrgIDs {
		if allowedID == creatorOrgID {
			return true
		}
	}

	return false
}

// canAssignToOrganization checks if user can assign application to target organization
func (s *LocalApplicationsService) canAssignToOrganization(userOrgRole, userOrgID, targetOrgID string) bool {
	// Get allowed organizations
	allowedOrgIDs, err := s.getAllowedOrganizationIDs(userOrgRole, userOrgID)
	if err != nil {
		return false
	}

	for _, allowedID := range allowedOrgIDs {
		if allowedID == targetOrgID {
			return true
		}
	}

	return false
}

// getOrganizationType returns the organization type for a given organization ID
func (s *LocalApplicationsService) getOrganizationType(orgID string) (string, error) {
	// Check distributors
	var exists bool
	err := database.DB.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM distributors WHERE logto_id = $1 AND deleted_at IS NULL)
	`, orgID).Scan(&exists)
	if err != nil {
		return "", err
	}
	if exists {
		return "distributor", nil
	}

	// Check resellers
	err = database.DB.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM resellers WHERE logto_id = $1 AND deleted_at IS NULL)
	`, orgID).Scan(&exists)
	if err != nil {
		return "", err
	}
	if exists {
		return "reseller", nil
	}

	// Check customers
	err = database.DB.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM customers WHERE logto_id = $1 AND deleted_at IS NULL)
	`, orgID).Scan(&exists)
	if err != nil {
		return "", err
	}
	if exists {
		return "customer", nil
	}

	// Default to owner if not found in other tables
	return "owner", nil
}

// GetApplicationTypeSummary returns applications grouped by type, optionally filtered by organization
func (s *LocalApplicationsService) GetApplicationTypeSummary(userOrgRole, userOrgID, organizationID, systemID string, includeHierarchy bool, page, pageSize int, sortBy, sortDirection string) (*models.ApplicationTypeSummary, error) {
	// Get allowed system IDs based on user's hierarchy (always enforced)
	allowedSystemIDs, err := s.getAllowedSystemIDs(userOrgRole, userOrgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get allowed systems: %w", err)
	}

	// If a specific system_id is requested, validate it's in the allowed set and restrict to it
	if systemID != "" {
		found := false
		for _, id := range allowedSystemIDs {
			if id == systemID {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("access denied: system not in user hierarchy")
		}
		allowedSystemIDs = []string{systemID}
	}

	var orgIDsToFilter []string

	if organizationID != "" {
		// Validate that the requested organization is within the user's hierarchy
		allowedOrgIDs, err := s.getAllowedOrganizationIDs(userOrgRole, userOrgID)
		if err != nil {
			return nil, fmt.Errorf("failed to get allowed organizations: %w", err)
		}

		found := false
		for _, id := range allowedOrgIDs {
			if id == organizationID {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("access denied: organization not in user hierarchy")
		}

		if includeHierarchy {
			// Get the target org + all its children
			childIDs, err := s.getChildOrganizationIDs(organizationID)
			if err != nil {
				return nil, fmt.Errorf("failed to get child organizations: %w", err)
			}

			// Intersect with allowed org IDs for safety
			allowedSet := make(map[string]bool, len(allowedOrgIDs))
			for _, id := range allowedOrgIDs {
				allowedSet[id] = true
			}
			for _, id := range childIDs {
				if allowedSet[id] {
					orgIDsToFilter = append(orgIDsToFilter, id)
				}
			}
		} else {
			orgIDsToFilter = []string{organizationID}
		}
	}
	// If organizationID is empty, orgIDsToFilter stays nil -> no org filter, all apps on allowed systems

	return s.repo.GetTypeSummary(allowedSystemIDs, orgIDsToFilter, true, page, pageSize, sortBy, sortDirection) // userFacingOnly
}

// getChildOrganizationIDs returns the given org plus all its children in the hierarchy
func (s *LocalApplicationsService) getChildOrganizationIDs(orgID string) ([]string, error) {
	orgType, err := s.getOrganizationType(orgID)
	if err != nil {
		return nil, err
	}

	result := []string{orgID}

	switch orgType {
	case "distributor":
		// Get child resellers
		resellerRows, err := database.DB.Query(`
			SELECT logto_id FROM resellers
			WHERE deleted_at IS NULL AND logto_id IS NOT NULL
			AND custom_data->>'createdBy' = $1
		`, orgID)
		if err != nil {
			return nil, fmt.Errorf("failed to query child resellers: %w", err)
		}
		defer func() { _ = resellerRows.Close() }()

		var resellerIDs []string
		for resellerRows.Next() {
			var id string
			if err := resellerRows.Scan(&id); err != nil {
				return nil, fmt.Errorf("failed to scan reseller ID: %w", err)
			}
			resellerIDs = append(resellerIDs, id)
			result = append(result, id)
		}

		// Get child customers (direct + through resellers)
		createdByIDs := append([]string{orgID}, resellerIDs...)
		if len(createdByIDs) > 0 {
			placeholders := make([]string, len(createdByIDs))
			args := make([]interface{}, len(createdByIDs))
			for i, id := range createdByIDs {
				placeholders[i] = fmt.Sprintf("$%d", i+1)
				args[i] = id
			}

			customerRows, err := database.DB.Query(fmt.Sprintf(`
				SELECT logto_id FROM customers
				WHERE deleted_at IS NULL AND logto_id IS NOT NULL
				AND custom_data->>'createdBy' IN (%s)
			`, strings.Join(placeholders, ",")), args...)
			if err != nil {
				return nil, fmt.Errorf("failed to query child customers: %w", err)
			}
			defer func() { _ = customerRows.Close() }()

			for customerRows.Next() {
				var id string
				if err := customerRows.Scan(&id); err != nil {
					return nil, fmt.Errorf("failed to scan customer ID: %w", err)
				}
				result = append(result, id)
			}
		}

	case "reseller":
		// Get child customers
		customerRows, err := database.DB.Query(`
			SELECT logto_id FROM customers
			WHERE deleted_at IS NULL AND logto_id IS NOT NULL
			AND custom_data->>'createdBy' = $1
		`, orgID)
		if err != nil {
			return nil, fmt.Errorf("failed to query child customers: %w", err)
		}
		defer func() { _ = customerRows.Close() }()

		for customerRows.Next() {
			var id string
			if err := customerRows.Scan(&id); err != nil {
				return nil, fmt.Errorf("failed to scan customer ID: %w", err)
			}
			result = append(result, id)
		}

	case "customer":
		// No children, just the org itself
	}

	return result, nil
}

// GetAvailableSystems returns list of systems that have applications (for filter dropdown)
func (s *LocalApplicationsService) GetAvailableSystems(userOrgRole, userOrgID string) ([]models.SystemSummary, error) {
	allowedSystemIDs, err := s.getAllowedSystemIDs(userOrgRole, userOrgID)
	if err != nil {
		return nil, err
	}

	if len(allowedSystemIDs) == 0 {
		return []models.SystemSummary{}, nil
	}

	placeholders := make([]string, len(allowedSystemIDs))
	args := make([]interface{}, len(allowedSystemIDs))
	for i, sysID := range allowedSystemIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = sysID
	}

	// Only return systems that have at least one application with certification level 4 or 5
	query := fmt.Sprintf(`
		SELECT DISTINCT s.id, s.name FROM systems s
		INNER JOIN applications a ON s.id = a.system_id
		WHERE s.id IN (%s) AND s.deleted_at IS NULL
		  AND a.deleted_at IS NULL AND a.is_user_facing = TRUE
		  AND (a.inventory_data->>'certification_level')::int IN (4, 5)
		ORDER BY s.name
	`, strings.Join(placeholders, ","))

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query systems: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var systems []models.SystemSummary
	for rows.Next() {
		var sys models.SystemSummary
		if err := rows.Scan(&sys.ID, &sys.Name); err != nil {
			return nil, fmt.Errorf("failed to scan system: %w", err)
		}
		systems = append(systems, sys)
	}

	return systems, nil
}

// GetAvailableOrganizations returns list of organizations that have applications (for filter dropdown)
func (s *LocalApplicationsService) GetAvailableOrganizations(userOrgRole, userOrgID string) ([]models.OrganizationSummary, error) {
	allowedOrgIDs, err := s.getAllowedOrganizationIDs(userOrgRole, userOrgID)
	if err != nil {
		return nil, err
	}

	var orgs []models.OrganizationSummary

	// Check if there are applications without organization (for "No organization" option)
	var hasUnassigned bool
	err = database.DB.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM applications WHERE (organization_id IS NULL OR organization_id = '') AND deleted_at IS NULL AND is_user_facing = TRUE AND (inventory_data->>'certification_level')::int IN (4, 5))
	`).Scan(&hasUnassigned)
	if err != nil {
		return nil, fmt.Errorf("failed to check unassigned applications: %w", err)
	}

	// Add "No organization" option first if there are unassigned applications
	if hasUnassigned {
		orgs = append(orgs, models.OrganizationSummary{
			ID:      "no_org",
			LogtoID: "no_org",
			Name:    "No organization",
			Type:    "unassigned",
		})
	}

	if len(allowedOrgIDs) == 0 {
		return orgs, nil
	}

	// Build placeholders for allowed org IDs - need different placeholders for each UNION part
	n := len(allowedOrgIDs)
	placeholders1 := make([]string, n)
	placeholders2 := make([]string, n)
	placeholders3 := make([]string, n)
	allArgs := make([]interface{}, 0, n*3)

	for i, orgID := range allowedOrgIDs {
		placeholders1[i] = fmt.Sprintf("$%d", i+1)
		placeholders2[i] = fmt.Sprintf("$%d", n+i+1)
		placeholders3[i] = fmt.Sprintf("$%d", 2*n+i+1)
		allArgs = append(allArgs, orgID)
	}
	for _, orgID := range allowedOrgIDs {
		allArgs = append(allArgs, orgID)
	}
	for _, orgID := range allowedOrgIDs {
		allArgs = append(allArgs, orgID)
	}

	// Query organizations that have at least one application assigned
	// UNION all organization tables, then INNER JOIN with applications
	query := fmt.Sprintf(`
		WITH all_orgs AS (
			SELECT id::text, logto_id, name, 'distributor' AS type FROM distributors WHERE deleted_at IS NULL AND logto_id IN (%s)
			UNION ALL
			SELECT id::text, logto_id, name, 'reseller' AS type FROM resellers WHERE deleted_at IS NULL AND logto_id IN (%s)
			UNION ALL
			SELECT id::text, logto_id, name, 'customer' AS type FROM customers WHERE deleted_at IS NULL AND logto_id IN (%s)
		)
		SELECT DISTINCT o.id, o.logto_id, o.name, o.type
		FROM all_orgs o
		INNER JOIN applications a ON a.organization_id = o.logto_id
		WHERE a.deleted_at IS NULL AND a.is_user_facing = TRUE
		  AND (a.inventory_data->>'certification_level')::int IN (4, 5)
		ORDER BY o.name
	`, strings.Join(placeholders1, ","), strings.Join(placeholders2, ","), strings.Join(placeholders3, ","))

	rows, err := database.DB.Query(query, allArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query organizations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var org models.OrganizationSummary
		if err := rows.Scan(&org.ID, &org.LogtoID, &org.Name, &org.Type); err != nil {
			return nil, fmt.Errorf("failed to scan organization: %w", err)
		}
		orgs = append(orgs, org)
	}

	return orgs, nil
}
