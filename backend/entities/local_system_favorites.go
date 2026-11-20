/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package entities

import (
	"database/sql"
	"fmt"

	"github.com/nethesis/my/backend/database"
)

// LocalSystemFavoriteRepository reads / writes system_favorites: the systems
// each user starred. Rows are keyed on the user's Logto ID.
type LocalSystemFavoriteRepository struct {
	db *sql.DB
}

func NewLocalSystemFavoriteRepository() *LocalSystemFavoriteRepository {
	return &LocalSystemFavoriteRepository{db: database.DB}
}

// Add stars the system for the user. Starring it again is a no-op.
func (r *LocalSystemFavoriteRepository) Add(userLogtoID, systemID string) error {
	_, err := r.db.Exec(
		`INSERT INTO system_favorites (user_logto_id, system_id) VALUES ($1, $2)
		 ON CONFLICT (user_logto_id, system_id) DO NOTHING`,
		userLogtoID, systemID,
	)
	if err != nil {
		return fmt.Errorf("failed to add system favorite: %w", err)
	}
	return nil
}

// Remove unstars the system for the user. Removing a missing row is a no-op.
func (r *LocalSystemFavoriteRepository) Remove(userLogtoID, systemID string) error {
	_, err := r.db.Exec(
		`DELETE FROM system_favorites WHERE user_logto_id = $1 AND system_id = $2`,
		userLogtoID, systemID,
	)
	if err != nil {
		return fmt.Errorf("failed to remove system favorite: %w", err)
	}
	return nil
}

// IsFavorite reports whether the user starred the system.
func (r *LocalSystemFavoriteRepository) IsFavorite(userLogtoID, systemID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(
		`SELECT EXISTS (SELECT 1 FROM system_favorites WHERE user_logto_id = $1 AND system_id = $2)`,
		userLogtoID, systemID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check system favorite: %w", err)
	}
	return exists, nil
}
