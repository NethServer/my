/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package local

import (
	"fmt"
	"strings"

	"github.com/lib/pq"
	"github.com/nethesis/my/backend/cache"
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

	cache.GetAppsCache().InvalidateAll()

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

	cache.GetAppsCache().InvalidateAll()

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

	cache.GetAppsCache().InvalidateAll()

	logger.Info().
		Str("application_id", id).
		Str("module_id", app.ModuleID).
		Str("deleted_by_org", userOrgID).
		Msg("Application deleted successfully")

	return nil
}

// GetApplicationTotals returns statistics for applications
func (s *LocalApplicationsService) GetApplicationTotals(userOrgRole, userOrgID string) (*models.ApplicationTotals, error) {
	// Check cache
	ac := cache.GetAppsCache()
	var cached models.ApplicationTotals
	if ac.Get("totals", userOrgRole, userOrgID, &cached) {
		return &cached, nil
	}

	allowedSystemIDs, err := s.getAllowedSystemIDs(userOrgRole, userOrgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get allowed systems: %w", err)
	}

	result, err := s.repo.GetTotals(allowedSystemIDs, true) // userFacingOnly
	if err != nil {
		return nil, err
	}

	ac.Set("totals", userOrgRole, userOrgID, result)
	return result, nil
}

// GetApplicationTypes returns distinct application types
func (s *LocalApplicationsService) GetApplicationTypes(userOrgRole, userOrgID string) ([]models.ApplicationType, error) {
	ac := cache.GetAppsCache()
	var cached []models.ApplicationType
	if ac.Get("types", userOrgRole, userOrgID, &cached) {
		return cached, nil
	}

	allowedSystemIDs, err := s.getAllowedSystemIDs(userOrgRole, userOrgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get allowed systems: %w", err)
	}

	result, err := s.repo.GetDistinctTypes(allowedSystemIDs, true)
	if err != nil {
		return nil, err
	}

	ac.Set("types", userOrgRole, userOrgID, result)
	return result, nil
}

// GetApplicationVersions returns distinct application versions grouped by instance_of
func (s *LocalApplicationsService) GetApplicationVersions(userOrgRole, userOrgID string) (map[string]entities.ApplicationVersionGroup, error) {
	ac := cache.GetAppsCache()
	var cached map[string]entities.ApplicationVersionGroup
	if ac.Get("versions", userOrgRole, userOrgID, &cached) {
		return cached, nil
	}

	allowedSystemIDs, err := s.getAllowedSystemIDs(userOrgRole, userOrgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get allowed systems: %w", err)
	}

	result, err := s.repo.GetDistinctVersions(allowedSystemIDs, true)
	if err != nil {
		return nil, err
	}

	ac.Set("versions", userOrgRole, userOrgID, result)
	return result, nil
}

// GetApplicationTotalsWithIDs returns statistics using pre-computed system IDs (avoids re-resolving RBAC)
func (s *LocalApplicationsService) GetApplicationTotalsWithIDs(allowedSystemIDs []string) (*models.ApplicationTotals, error) {
	return s.repo.GetTotals(allowedSystemIDs, true)
}

// GetApplicationTypesWithIDs returns distinct types using pre-computed system IDs
func (s *LocalApplicationsService) GetApplicationTypesWithIDs(allowedSystemIDs []string) ([]models.ApplicationType, error) {
	return s.repo.GetDistinctTypes(allowedSystemIDs, true)
}

// GetApplicationVersionsWithIDs returns distinct versions using pre-computed system IDs
func (s *LocalApplicationsService) GetApplicationVersionsWithIDs(allowedSystemIDs []string) (map[string]entities.ApplicationVersionGroup, error) {
	return s.repo.GetDistinctVersions(allowedSystemIDs, true)
}

// GetAvailableOrganizationsWithIDs returns available organizations using pre-computed allowed org IDs
func (s *LocalApplicationsService) GetAvailableOrganizationsWithIDs(allowedOrgIDs []string) ([]models.OrganizationSummary, error) {
	return s.getAvailableOrganizationsFromIDs(allowedOrgIDs)
}

// getAvailableOrganizationsFromIDs is the internal implementation that accepts pre-computed org IDs
func (s *LocalApplicationsService) getAvailableOrganizationsFromIDs(allowedOrgIDs []string) ([]models.OrganizationSummary, error) {
	var orgs []models.OrganizationSummary

	// Check if there are applications without organization
	var hasUnassigned bool
	err := database.DB.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM applications WHERE (organization_id IS NULL OR organization_id = '') AND deleted_at IS NULL AND is_user_facing = TRUE AND (inventory_data->>'certification_level')::int IN (4, 5))
	`).Scan(&hasUnassigned)
	if err != nil {
		return nil, fmt.Errorf("failed to check unassigned applications: %w", err)
	}

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

	query := `
		WITH all_orgs AS (
			SELECT id::text, logto_id, name, 'distributor' AS type FROM distributors WHERE deleted_at IS NULL AND logto_id = ANY($1::text[])
			UNION ALL
			SELECT id::text, logto_id, name, 'reseller' AS type FROM resellers WHERE deleted_at IS NULL AND logto_id = ANY($1::text[])
			UNION ALL
			SELECT id::text, logto_id, name, 'customer' AS type FROM customers WHERE deleted_at IS NULL AND logto_id = ANY($1::text[])
		)
		SELECT DISTINCT o.id, o.logto_id, o.name, o.type
		FROM all_orgs o
		INNER JOIN applications a ON a.organization_id = o.logto_id
		WHERE a.deleted_at IS NULL AND a.is_user_facing = TRUE
		  AND (a.inventory_data->>'certification_level')::int IN (4, 5)
		ORDER BY o.name
	`

	rows, err := database.DB.Query(query, pq.Array(allowedOrgIDs))
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

// GetAllowedSystemIDs returns list of system IDs the user can access based on hierarchy.
// Results are cached for performance (5-minute TTL).
func (s *LocalApplicationsService) GetAllowedSystemIDs(userOrgRole, userOrgID string) ([]string, error) {
	normalizedRole := strings.ToLower(userOrgRole)

	// Check cache first
	rbac := cache.GetRBACCache()
	if cached, ok := rbac.GetSystemIDs(normalizedRole, userOrgID); ok {
		return cached, nil
	}

	// Get allowed organization IDs based on hierarchy
	allowedOrgIDs, err := s.GetAllowedOrganizationIDs(userOrgRole, userOrgID)
	if err != nil {
		return nil, err
	}

	if len(allowedOrgIDs) == 0 {
		rbac.SetSystemIDs(normalizedRole, userOrgID, []string{})
		return []string{}, nil
	}

	query := `
		SELECT id FROM systems
		WHERE deleted_at IS NULL AND created_by ->> 'organization_id' = ANY($1::text[])
	`

	rows, err := database.DB.Query(query, pq.Array(allowedOrgIDs))
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

	// Cache result
	rbac.SetSystemIDs(normalizedRole, userOrgID, systemIDs)

	return systemIDs, nil
}

// getAllowedSystemIDs is a private alias for backward compatibility within the service
func (s *LocalApplicationsService) getAllowedSystemIDs(userOrgRole, userOrgID string) ([]string, error) {
	return s.GetAllowedSystemIDs(userOrgRole, userOrgID)
}

// GetAllowedOrganizationIDs returns list of organization IDs the user can access.
// Results are cached for performance (5-minute TTL).
func (s *LocalApplicationsService) GetAllowedOrganizationIDs(userOrgRole, userOrgID string) ([]string, error) {
	normalizedRole := strings.ToLower(userOrgRole)

	// Check cache first
	rbac := cache.GetRBACCache()
	if cached, ok := rbac.GetOrgIDs(normalizedRole, userOrgID); ok {
		return cached, nil
	}

	allowedOrgIDs, err := s.computeAllowedOrganizationIDs(normalizedRole, userOrgID)
	if err != nil {
		return nil, err
	}

	// Cache result
	rbac.SetOrgIDs(normalizedRole, userOrgID, allowedOrgIDs)

	return allowedOrgIDs, nil
}

// getAllowedOrganizationIDs is a private alias for backward compatibility within the service
func (s *LocalApplicationsService) getAllowedOrganizationIDs(userOrgRole, userOrgID string) ([]string, error) {
	return s.GetAllowedOrganizationIDs(userOrgRole, userOrgID)
}

// computeAllowedOrganizationIDs performs the actual DB queries for allowed org IDs
func (s *LocalApplicationsService) computeAllowedOrganizationIDs(normalizedRole, userOrgID string) ([]string, error) {
	var allowedOrgIDs []string

	switch normalizedRole {
	case "owner":
		// Owner can access all organizations - single UNION query
		query := `
			SELECT logto_id FROM distributors WHERE deleted_at IS NULL AND logto_id IS NOT NULL
			UNION ALL
			SELECT logto_id FROM resellers WHERE deleted_at IS NULL AND logto_id IS NOT NULL
			UNION ALL
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
			createdByIDs := append([]string{userOrgID}, resellerIDs...)

			customerQuery := `
				SELECT logto_id FROM customers
				WHERE deleted_at IS NULL AND logto_id IS NOT NULL
				AND custom_data->>'createdBy' = ANY($1::text[])
			`

			customerRows, err := database.DB.Query(customerQuery, pq.Array(createdByIDs))
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
		return nil, fmt.Errorf("unknown organization role: %s", normalizedRole)
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

	allowedMap := make(map[string]bool, len(allowedOrgIDs))
	for _, id := range allowedOrgIDs {
		allowedMap[id] = true
	}
	return allowedMap[creatorOrgID]
}

// canAssignToOrganization checks if user can assign application to target organization
func (s *LocalApplicationsService) canAssignToOrganization(userOrgRole, userOrgID, targetOrgID string) bool {
	// Get allowed organizations
	allowedOrgIDs, err := s.getAllowedOrganizationIDs(userOrgRole, userOrgID)
	if err != nil {
		return false
	}

	allowedMap := make(map[string]bool, len(allowedOrgIDs))
	for _, id := range allowedOrgIDs {
		allowedMap[id] = true
	}
	return allowedMap[targetOrgID]
}

// getOrganizationType returns the organization type for a given organization ID
func (s *LocalApplicationsService) getOrganizationType(orgID string) (string, error) {
	var orgType string
	err := database.DB.QueryRow(`
		SELECT CASE
			WHEN EXISTS (SELECT 1 FROM distributors WHERE logto_id = $1 AND deleted_at IS NULL) THEN 'distributor'
			WHEN EXISTS (SELECT 1 FROM resellers WHERE logto_id = $1 AND deleted_at IS NULL) THEN 'reseller'
			WHEN EXISTS (SELECT 1 FROM customers WHERE logto_id = $1 AND deleted_at IS NULL) THEN 'customer'
			ELSE 'owner'
		END
	`, orgID).Scan(&orgType)
	if err != nil {
		return "", err
	}
	return orgType, nil
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
			customerRows, err := database.DB.Query(`
				SELECT logto_id FROM customers
				WHERE deleted_at IS NULL AND logto_id IS NOT NULL
				AND custom_data->>'createdBy' = ANY($1::text[])
			`, pq.Array(createdByIDs))
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
	ac := cache.GetAppsCache()
	var cached []models.SystemSummary
	if ac.Get("systems", userOrgRole, userOrgID, &cached) {
		return cached, nil
	}

	allowedSystemIDs, err := s.getAllowedSystemIDs(userOrgRole, userOrgID)
	if err != nil {
		return nil, err
	}

	if len(allowedSystemIDs) == 0 {
		return []models.SystemSummary{}, nil
	}

	// Only return systems that have at least one application with certification level 4 or 5
	query := `
		SELECT DISTINCT s.id, s.name FROM systems s
		INNER JOIN applications a ON s.id = a.system_id
		WHERE s.id = ANY($1::text[]) AND s.deleted_at IS NULL
		  AND a.deleted_at IS NULL AND a.is_user_facing = TRUE
		  AND (a.inventory_data->>'certification_level')::int IN (4, 5)
		ORDER BY s.name
	`

	rows, err := database.DB.Query(query, pq.Array(allowedSystemIDs))
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

	ac.Set("systems", userOrgRole, userOrgID, systems)
	return systems, nil
}

// GetAvailableOrganizations returns list of organizations that have applications (for filter dropdown)
func (s *LocalApplicationsService) GetAvailableOrganizations(userOrgRole, userOrgID string) ([]models.OrganizationSummary, error) {
	ac := cache.GetAppsCache()
	var cached []models.OrganizationSummary
	if ac.Get("orgs", userOrgRole, userOrgID, &cached) {
		return cached, nil
	}

	allowedOrgIDs, err := s.getAllowedOrganizationIDs(userOrgRole, userOrgID)
	if err != nil {
		return nil, err
	}

	result, err := s.getAvailableOrganizationsFromIDs(allowedOrgIDs)
	if err != nil {
		return nil, err
	}

	ac.Set("orgs", userOrgRole, userOrgID, result)
	return result, nil
}
