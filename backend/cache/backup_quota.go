/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package cache

import (
	"context"
	"fmt"

	"github.com/nethesis/my/backend/logger"
)

// InvalidateOrgBackupUsage drops the cached aggregate-usage counter for the
// given organization on the shared Redis DB. collect maintains the counter
// incrementally on its own writes; backend mutates the same prefix when an
// admin deletes a backup, when DestroySystem purges, or when a cross-org
// reassignment moves bytes around — none of which collect's counter sees.
// Invalidating the key forces collect's next quota check to recompute from
// S3, restoring consistency.
//
// The counter lives on the SHARED Redis DB (configured via REDIS_DB_SHARED,
// default 1, matching collect's default REDIS_DB) so this DEL must also use
// that DB; backend's own client (DB 0 by default) cannot reach it.
//
// The cache key shape MUST match collect/methods/backups.go::orgUsageCacheKey.
// Best-effort: a Redis hiccup is logged but does not block the caller.
func InvalidateOrgBackupUsage(ctx context.Context, orgID string) {
	if orgID == "" {
		return
	}
	client := GetSharedRedisClient()
	if client == nil {
		return
	}
	key := fmt.Sprintf("backup:org_usage:%s", orgID)
	if err := client.Del(ctx, key).Err(); err != nil {
		logger.Warn().Err(err).Str("org_id", orgID).Msg("failed to invalidate org backup usage counter")
	}
}
