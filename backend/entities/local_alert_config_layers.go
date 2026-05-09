/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package entities

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"

	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/models"
)

// AlertConfigLayerRecord is one organization's saved alerting layer plus the
// audit metadata we render in the UI ("set by Mario, 5 minutes ago").
type AlertConfigLayerRecord struct {
	OrganizationID  string                     `json:"organization_id"`
	Config          models.AlertingConfigLayer `json:"config"`
	UpdatedByUserID *string                    `json:"updated_by_user_id,omitempty"`
	UpdatedByName   *string                    `json:"updated_by_name,omitempty"`
	UpdatedAt       time.Time                  `json:"updated_at"`
	CreatedAt       time.Time                  `json:"created_at"`
}

// LocalAlertConfigLayersRepository persists hierarchical alerting config
// layers (one row per organization).
type LocalAlertConfigLayersRepository struct {
	db *sql.DB
}

func NewLocalAlertConfigLayersRepository() *LocalAlertConfigLayersRepository {
	return &LocalAlertConfigLayersRepository{db: database.DB}
}

// ErrAlertConfigLayerNotFound is returned by Get when no layer has ever been
// saved for the given org. Callers translate this to "empty layer / inherit
// from above" rather than a hard 404.
var ErrAlertConfigLayerNotFound = errors.New("alert config layer not found")

// Get returns the saved layer for the given org, or ErrAlertConfigLayerNotFound
// when no row exists. Callers walking the hierarchy treat the missing-row case
// as "empty layer — inherit only".
func (r *LocalAlertConfigLayersRepository) Get(orgID string) (*AlertConfigLayerRecord, error) {
	row := r.db.QueryRow(
		`SELECT organization_id, config_json, updated_by_user_id, updated_by_name, updated_at, created_at
		 FROM alert_config_layers WHERE organization_id = $1`,
		orgID,
	)
	rec := &AlertConfigLayerRecord{}
	var raw []byte
	var by, byName sql.NullString
	if err := row.Scan(&rec.OrganizationID, &raw, &by, &byName, &rec.UpdatedAt, &rec.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAlertConfigLayerNotFound
		}
		return nil, fmt.Errorf("get alert config layer: %w", err)
	}
	if err := json.Unmarshal(raw, &rec.Config); err != nil {
		return nil, fmt.Errorf("decode alert config layer: %w", err)
	}
	if by.Valid {
		rec.UpdatedByUserID = &by.String
	}
	if byName.Valid {
		rec.UpdatedByName = &byName.String
	}
	return rec, nil
}

// Upsert writes or replaces the layer for the given org. updated_at is
// refreshed; created_at is preserved by the ON CONFLICT path.
//
// Calls cfg.Validate() before writing as a defense-in-depth backstop:
// any write path bypassing the HTTP handler (admin tooling, future
// endpoints, migrations) still gets the same regex / format checks as
// the handler. DNS-aware webhook URL checks remain at the handler.
func (r *LocalAlertConfigLayersRepository) Upsert(orgID string, cfg models.AlertingConfigLayer, byUserID, byName string) (*AlertConfigLayerRecord, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate layer: %w", err)
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("encode alert config layer: %w", err)
	}
	row := r.db.QueryRow(
		`INSERT INTO alert_config_layers (organization_id, config_json, updated_by_user_id, updated_by_name, updated_at, created_at)
		 VALUES ($1, $2::jsonb, NULLIF($3,''), NULLIF($4,''), NOW(), NOW())
		 ON CONFLICT (organization_id) DO UPDATE
		   SET config_json = EXCLUDED.config_json,
		       updated_by_user_id = EXCLUDED.updated_by_user_id,
		       updated_by_name = EXCLUDED.updated_by_name,
		       updated_at = NOW()
		 RETURNING organization_id, config_json, updated_by_user_id, updated_by_name, updated_at, created_at`,
		orgID, string(raw), byUserID, byName,
	)
	rec := &AlertConfigLayerRecord{}
	var rawOut []byte
	var by, byNameOut sql.NullString
	if err := row.Scan(&rec.OrganizationID, &rawOut, &by, &byNameOut, &rec.UpdatedAt, &rec.CreatedAt); err != nil {
		return nil, fmt.Errorf("upsert alert config layer: %w", err)
	}
	if err := json.Unmarshal(rawOut, &rec.Config); err != nil {
		return nil, fmt.Errorf("decode upsert response: %w", err)
	}
	if by.Valid {
		rec.UpdatedByUserID = &by.String
	}
	if byNameOut.Valid {
		rec.UpdatedByName = &byNameOut.String
	}
	return rec, nil
}

// Delete removes the layer for the given org. Idempotent: returns nil even
// when no row matched. Used by the org-deletion cascade.
func (r *LocalAlertConfigLayersRepository) Delete(orgID string) error {
	_, err := r.db.Exec(`DELETE FROM alert_config_layers WHERE organization_id = $1`, orgID)
	if err != nil {
		return fmt.Errorf("delete alert config layer: %w", err)
	}
	return nil
}

// GetByOrgIDs fetches the layers for a list of orgs in a single query,
// returned as a map keyed by org_id. Orgs without a row are simply absent
// from the map (treated as "empty layer" by callers).
//
// Used by the merge path: when computing the effective config for tenant T,
// we resolve T's hierarchy chain (Owner→...→T) and bulk-fetch the layers in
// one round-trip.
func (r *LocalAlertConfigLayersRepository) GetByOrgIDs(orgIDs []string) (map[string]*AlertConfigLayerRecord, error) {
	if len(orgIDs) == 0 {
		return map[string]*AlertConfigLayerRecord{}, nil
	}
	rows, err := r.db.Query(
		`SELECT organization_id, config_json, updated_by_user_id, updated_by_name, updated_at, created_at
		 FROM alert_config_layers WHERE organization_id = ANY($1)`,
		pq.Array(orgIDs),
	)
	if err != nil {
		return nil, fmt.Errorf("get layers by org_ids: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := make(map[string]*AlertConfigLayerRecord, len(orgIDs))
	for rows.Next() {
		rec := &AlertConfigLayerRecord{}
		var raw []byte
		var by, byName sql.NullString
		if err := rows.Scan(&rec.OrganizationID, &raw, &by, &byName, &rec.UpdatedAt, &rec.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan layer row: %w", err)
		}
		if err := json.Unmarshal(raw, &rec.Config); err != nil {
			return nil, fmt.Errorf("decode layer row: %w", err)
		}
		if by.Valid {
			rec.UpdatedByUserID = &by.String
		}
		if byName.Valid {
			rec.UpdatedByName = &byName.String
		}
		out[rec.OrganizationID] = rec
	}
	return out, nil
}
