/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package entities

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nethesis/my/backend/database"
)

// AlertActivityAction enumerates the event kinds written to alert_activity.
// New values can be added without a schema change. Note edits are not
// represented as their own action: the operator note IS the silence comment,
// so a comment change shows up as silence_updated.
const (
	AlertActivitySilenced       = "silenced"
	AlertActivitySilenceUpdated = "silence_updated"
	AlertActivityUnsilenced     = "unsilenced"
)

// AlertActivityEntry is one row of the per-alert audit timeline.
type AlertActivityEntry struct {
	ID             int64                  `json:"id"`
	OrganizationID string                 `json:"organization_id"`
	Fingerprint    string                 `json:"fingerprint"`
	Action         string                 `json:"action"`
	ActorUserID    *string                `json:"actor_user_id,omitempty"`
	ActorName      *string                `json:"actor_name,omitempty"`
	SilenceID      *string                `json:"silence_id,omitempty"`
	Details        map[string]interface{} `json:"details"`
	CreatedAt      time.Time              `json:"created_at"`
}

// LocalAlertActivityRepository writes / reads the alert_activity timeline.
type LocalAlertActivityRepository struct {
	db *sql.DB
}

func NewLocalAlertActivityRepository() *LocalAlertActivityRepository {
	return &LocalAlertActivityRepository{db: database.DB}
}

// Log appends a single event to the activity timeline. Best-effort: callers
// that only want to record audit info should not fail their primary action
// when this returns an error — wrap the call site with a warn-level log.
func (r *LocalAlertActivityRepository) Log(orgID, fingerprint, action, actorUserID, actorName, silenceID string, details map[string]interface{}) error {
	if details == nil {
		details = map[string]interface{}{}
	}
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("encode details: %w", err)
	}
	_, err = r.db.Exec(
		`INSERT INTO alert_activity (organization_id, fingerprint, action, actor_user_id, actor_name, silence_id, details)
		 VALUES ($1, $2, $3, NULLIF($4,''), NULLIF($5,''), NULLIF($6,''), $7::jsonb)`,
		orgID, fingerprint, action, actorUserID, actorName, silenceID, string(detailsJSON),
	)
	if err != nil {
		return fmt.Errorf("insert alert_activity: %w", err)
	}
	return nil
}

// ListByFingerprint returns the timeline for one alert, most recent first.
// limit caps the number of rows; values <=0 fall back to 100.
func (r *LocalAlertActivityRepository) ListByFingerprint(orgID, fingerprint string, limit int) ([]AlertActivityEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.Query(
		`SELECT id, organization_id, fingerprint, action, actor_user_id, actor_name, silence_id, details, created_at
		 FROM alert_activity
		 WHERE organization_id = $1 AND fingerprint = $2
		 ORDER BY created_at DESC, id DESC
		 LIMIT $3`,
		orgID, fingerprint, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query alert_activity: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]AlertActivityEntry, 0)
	for rows.Next() {
		var e AlertActivityEntry
		var actorUserID, actorName, silenceID sql.NullString
		var detailsRaw []byte
		if err := rows.Scan(&e.ID, &e.OrganizationID, &e.Fingerprint, &e.Action, &actorUserID, &actorName, &silenceID, &detailsRaw, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan alert_activity: %w", err)
		}
		if actorUserID.Valid {
			e.ActorUserID = &actorUserID.String
		}
		if actorName.Valid {
			e.ActorName = &actorName.String
		}
		if silenceID.Valid {
			e.SilenceID = &silenceID.String
		}
		if len(detailsRaw) > 0 {
			if err := json.Unmarshal(detailsRaw, &e.Details); err != nil {
				e.Details = map[string]interface{}{}
			}
		} else {
			e.Details = map[string]interface{}{}
		}
		out = append(out, e)
	}
	return out, nil
}

// FindFingerprintBySilenceID returns the fingerprint of the alert that the
// given silence was created against, or empty string if no record exists.
// Used by DeleteSystemAlertSilence to log the unsilence event under the
// correct alert without requiring the caller to pass the fingerprint.
func (r *LocalAlertActivityRepository) FindFingerprintBySilenceID(orgID, silenceID string) (string, error) {
	var fp string
	err := r.db.QueryRow(
		`SELECT fingerprint FROM alert_activity
		 WHERE organization_id = $1 AND silence_id = $2 AND action = $3
		 ORDER BY created_at DESC LIMIT 1`,
		orgID, silenceID, AlertActivitySilenced,
	).Scan(&fp)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("lookup fingerprint by silence_id: %w", err)
	}
	return fp, nil
}
