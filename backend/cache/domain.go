/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package cache

import (
	"sync"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/services/logto"
)

// DomainValidation provides in-memory access to domain validation status
type DomainValidation struct {
	isValid bool
	domain  string
	mutex   sync.RWMutex
	loaded  bool
}

var (
	domainValidation     *DomainValidation
	domainValidationOnce sync.Once
)

// GetDomainValidation returns a singleton instance of the domain validation store
func GetDomainValidation() *DomainValidation {
	domainValidationOnce.Do(func() {
		domainValidation = &DomainValidation{}
	})
	return domainValidation
}

// LoadDomainValidation validates the tenant domain with Logto and stores the result in memory
// This should be called at server startup
func (d *DomainValidation) LoadDomainValidation() error {
	tenantDomain := configuration.Config.TenantDomain

	logger.ComponentLogger("domain").Info().
		Str("domain", tenantDomain).
		Msg("Loading domain validation from Logto")

	client := logto.NewManagementClient()
	isValid := client.ValidateDomain(tenantDomain)

	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.domain = tenantDomain
	d.isValid = isValid
	d.loaded = true

	logger.ComponentLogger("domain").Info().
		Str("domain", tenantDomain).
		Bool("is_valid", isValid).
		Msg("Domain validation loaded successfully")

	return nil
}

// IsValid returns true if the cached domain is valid
func (d *DomainValidation) IsValid() bool {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	if !d.loaded {
		logger.ComponentLogger("domain").Warn().
			Msg("Domain validation not loaded yet, returning false")
		return false
	}

	return d.isValid
}

// IsLoaded returns true if domain validation has been loaded
func (d *DomainValidation) IsLoaded() bool {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.loaded
}

// GetDomain returns the cached domain
func (d *DomainValidation) GetDomain() string {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.domain
}
