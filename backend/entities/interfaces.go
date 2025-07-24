/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package entities

import (
	"time"

	"github.com/nethesis/my/backend/models"
)

// DistributorRepository defines interface for distributor data operations
type DistributorRepository interface {
	Create(req *models.CreateLocalDistributorRequest) (*models.LocalDistributor, error)
	GetByID(id string) (*models.LocalDistributor, error)
	Update(id string, req *models.UpdateLocalDistributorRequest) (*models.LocalDistributor, error)
	Delete(id string) error
	List(userOrgRole, userOrgID string, page, pageSize int) ([]*models.LocalDistributor, int, error)
	GetTotals(userOrgRole, userOrgID string) (int, error)
}

// ResellerRepository defines interface for reseller data operations
type ResellerRepository interface {
	Create(req *models.CreateLocalResellerRequest) (*models.LocalReseller, error)
	GetByID(id string) (*models.LocalReseller, error)
	Update(id string, req *models.UpdateLocalResellerRequest) (*models.LocalReseller, error)
	Delete(id string) error
	List(userOrgRole, userOrgID string, page, pageSize int) ([]*models.LocalReseller, int, error)
	GetTotals(userOrgRole, userOrgID string) (int, error)
}

// CustomerRepository defines interface for customer data operations
type CustomerRepository interface {
	Create(req *models.CreateLocalCustomerRequest) (*models.LocalCustomer, error)
	GetByID(id string) (*models.LocalCustomer, error)
	Update(id string, req *models.UpdateLocalCustomerRequest) (*models.LocalCustomer, error)
	Delete(id string) error
	List(userOrgRole, userOrgID string, page, pageSize int) ([]*models.LocalCustomer, int, error)
	GetTotals(userOrgRole, userOrgID string) (int, error)
}

// UserRepository defines interface for user data operations
type UserRepository interface {
	Create(req *models.CreateLocalUserRequest) (*models.LocalUser, error)
	GetByID(id string) (*models.LocalUser, error)
	Update(id string, req *models.UpdateLocalUserRequest) (*models.LocalUser, error)
	Delete(id string) error
	List(userOrgRole, userOrgID string, page, pageSize int) ([]*models.LocalUser, int, error)
	GetTotals(userOrgRole, userOrgID string) (int, error)
}

// SystemRepository defines interface for system data operations
type SystemRepository interface {
	Create(req *models.CreateSystemRequest) (*models.System, error)
	GetByID(id string) (*models.System, error)
	Update(id string, req *models.UpdateSystemRequest) (*models.System, error)
	Delete(id string) error
	List(userOrgRole, userOrgID string, page, pageSize int) ([]*models.System, int, error)
	GetTotals(userOrgRole, userOrgID string) (*models.SystemTotals, error)
}

// InventoryRepository defines interface for inventory data operations
type InventoryRepository interface {
	GetLatestInventory(systemID string) (*models.InventoryRecord, error)
	GetInventoryHistory(systemID string, page, pageSize int, fromDate, toDate *time.Time) ([]models.InventoryRecord, int, error)
	GetInventoryDiffs(systemID string, page, pageSize int, severity, category, diffType string, fromDate, toDate *time.Time) ([]models.InventoryDiff, int, error)
	GetLatestInventoryDiffs(systemID string) ([]models.InventoryDiff, error)
}
