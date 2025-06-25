/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: GPL-2.0-only
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

// Package services provides Logto integration functionality.
//
// This file serves as a compatibility layer that re-exports all types and functions
// from the modular Logto services files. The original monolithic logto.go file
// has been refactored into several focused modules:
//
// - logto_models.go: Data structures and type definitions
// - logto_client.go: Base Management API client
// - logto_auth.go: Authentication functions
// - logto_roles.go: Role and permission management
// - logto_organizations.go: Organization management
// - logto_users.go: User/account management
//
// All existing imports and function calls will continue to work unchanged.

package services

// This file intentionally left minimal as all functionality has been moved
// to the specialized modules. All types and functions are automatically
// available through Go's package-level exports from the other files.
