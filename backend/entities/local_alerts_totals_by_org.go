/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package entities

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"

	"github.com/nethesis/my/backend/database"
)

// LocalAlertsTotalsByOrgRepository serves the pre-aggregated active alert
// counts maintained by collect's AlertsTotalsRefresher cron. /api/alerts/totals
// reads here instead of fanning out to Mimir per tenant on every request.
type LocalAlertsTotalsByOrgRepository struct {
	db *sql.DB
}

// NewLocalAlertsTotalsByOrgRepository creates a new repository instance.
func NewLocalAlertsTotalsByOrgRepository() *LocalAlertsTotalsByOrgRepository {
	return &LocalAlertsTotalsByOrgRepository{db: database.DB}
}

// AlertsTotalsSum is the aggregated answer for one /totals call. OldestUpdate
// is the MIN(updated_at) across the queried rows — the caller can compare it
// against now() to surface staleness when the refresher is lagging or down.
// It is the zero time when no rows matched (e.g. brand-new tenant, refresher
// has not yet inserted any rows).
type AlertsTotalsSum struct {
	Active       int
	Critical     int
	Warning      int
	Info         int
	Muted        int
	OldestUpdate time.Time
}

// SumByOrgIDs returns the per-severity and muted totals summed across the
// given organization IDs. An empty input returns a zero-valued struct with no
// error (caller has no orgs in scope). Tenants that have never been touched
// by the refresher are simply missing from the table and contribute zero.
func (r *LocalAlertsTotalsByOrgRepository) SumByOrgIDs(orgIDs []string) (AlertsTotalsSum, error) {
	var out AlertsTotalsSum
	if len(orgIDs) == 0 {
		return out, nil
	}

	var oldest sql.NullTime
	err := r.db.QueryRow(`
		SELECT
			COALESCE(SUM(active),   0),
			COALESCE(SUM(critical), 0),
			COALESCE(SUM(warning),  0),
			COALESCE(SUM(info),     0),
			COALESCE(SUM(muted),    0),
			MIN(updated_at)
		FROM alerts_totals_by_org
		WHERE organization_id = ANY($1)
	`, pq.Array(orgIDs)).Scan(
		&out.Active,
		&out.Critical,
		&out.Warning,
		&out.Info,
		&out.Muted,
		&oldest,
	)
	if err != nil {
		return AlertsTotalsSum{}, fmt.Errorf("sum alerts_totals_by_org: %w", err)
	}
	if oldest.Valid {
		out.OldestUpdate = oldest.Time
	}
	return out, nil
}
