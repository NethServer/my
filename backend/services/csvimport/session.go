/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package csvimport

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/nethesis/my/backend/cache"
	"github.com/nethesis/my/backend/models"
)

const (
	importSessionTTL    = 30 * time.Minute
	importSessionPrefix = "csv_import:"
)

// SaveImportSession stores validated import data in Redis for later confirmation.
// Returns the generated import_id.
func SaveImportSession(session *models.ImportSessionData) (string, error) {
	importID := uuid.New().String()
	key := importSessionPrefix + importID

	rc := cache.GetRedisClient()
	if rc == nil {
		return "", fmt.Errorf("redis not available")
	}

	err := rc.Set(key, session, importSessionTTL)
	if err != nil {
		return "", fmt.Errorf("failed to save import session: %w", err)
	}

	return importID, nil
}

// GetImportSession retrieves a validated import session from Redis.
func GetImportSession(importID string) (*models.ImportSessionData, error) {
	key := importSessionPrefix + importID

	rc := cache.GetRedisClient()
	if rc == nil {
		return nil, fmt.Errorf("redis not available")
	}

	var session models.ImportSessionData
	err := rc.Get(key, &session)
	if err != nil {
		return nil, fmt.Errorf("import session not found or expired")
	}

	return &session, nil
}

// DeleteImportSession removes an import session from Redis after it has been consumed.
func DeleteImportSession(importID string) {
	key := importSessionPrefix + importID
	rc := cache.GetRedisClient()
	if rc != nil {
		_ = rc.Delete(key)
	}
}
