/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package local

import (
	"database/sql"
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

// GetApplicationVersions returns distinct application versions
func (s *LocalApplicationsService) GetApplicationVersions(userOrgRole, userOrgID string) ([]string, error) {
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
			AND custom_data->>'distributor_id' = $1
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
				AND (custom_data->>'distributor_id' = $1 OR custom_data->>'reseller_id' IN (%s))
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
				AND custom_data->>'distributor_id' = $1
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
			AND custom_data->>'reseller_id' = $1
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

// GetAvailableSystems returns list of systems user can see (for filter dropdown)
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

	query := fmt.Sprintf(`
		SELECT id, name FROM systems
		WHERE id IN (%s) AND deleted_at IS NULL
		ORDER BY name
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

// GetAvailableOrganizations returns list of organizations user can assign to (for filter/assign dropdown)
func (s *LocalApplicationsService) GetAvailableOrganizations(userOrgRole, userOrgID string) ([]models.OrganizationSummary, error) {
	allowedOrgIDs, err := s.getAllowedOrganizationIDs(userOrgRole, userOrgID)
	if err != nil {
		return nil, err
	}

	if len(allowedOrgIDs) == 0 {
		return []models.OrganizationSummary{}, nil
	}

	var orgs []models.OrganizationSummary

	// Query each organization table
	for _, logtoID := range allowedOrgIDs {
		var dbID, name string

		// Try distributors
		err := database.DB.QueryRow(`
			SELECT id, name FROM distributors WHERE logto_id = $1 AND deleted_at IS NULL
		`, logtoID).Scan(&dbID, &name)
		if err == nil {
			orgs = append(orgs, models.OrganizationSummary{ID: dbID, LogtoID: logtoID, Name: name, Type: "distributor"})
			continue
		}
		if err != sql.ErrNoRows {
			return nil, err
		}

		// Try resellers
		err = database.DB.QueryRow(`
			SELECT id, name FROM resellers WHERE logto_id = $1 AND deleted_at IS NULL
		`, logtoID).Scan(&dbID, &name)
		if err == nil {
			orgs = append(orgs, models.OrganizationSummary{ID: dbID, LogtoID: logtoID, Name: name, Type: "reseller"})
			continue
		}
		if err != sql.ErrNoRows {
			return nil, err
		}

		// Try customers
		err = database.DB.QueryRow(`
			SELECT id, name FROM customers WHERE logto_id = $1 AND deleted_at IS NULL
		`, logtoID).Scan(&dbID, &name)
		if err == nil {
			orgs = append(orgs, models.OrganizationSummary{ID: dbID, LogtoID: logtoID, Name: name, Type: "customer"})
			continue
		}
		if err != sql.ErrNoRows {
			return nil, err
		}

		// Check if it's owner org (owner has no DB entry, use logto_id for both)
		if logtoID == userOrgID && strings.ToLower(userOrgRole) == "owner" {
			orgs = append(orgs, models.OrganizationSummary{ID: logtoID, LogtoID: logtoID, Name: "Owner", Type: "owner"})
		}
	}

	return orgs, nil
}
