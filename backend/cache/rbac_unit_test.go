/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRBACCacheSetAndGet(t *testing.T) {
	c := GetRBACCache()
	c.InvalidateAll()

	c.SetOrgIDs("distributor", "org-test-1", []string{"orgA", "orgB"})
	c.SetSystemIDs("distributor", "org-test-1", []string{"sys1"})

	orgIDs, ok := c.GetOrgIDs("distributor", "org-test-1")
	assert.True(t, ok)
	assert.Equal(t, []string{"orgA", "orgB"}, orgIDs)

	systemIDs, ok := c.GetSystemIDs("distributor", "org-test-1")
	assert.True(t, ok)
	assert.Equal(t, []string{"sys1"}, systemIDs)
}

// A refresh of the org-ID list must not extend the lifetime of a stale
// system-ID list sharing the same cache entry (and vice versa): a system
// created after the lists were cached would otherwise stay invisible far
// beyond the TTL on busy deployments where org lookups win every refresh.
func TestRBACCacheIndependentTimestamps(t *testing.T) {
	c := GetRBACCache()
	c.InvalidateAll()

	c.SetOrgIDs("distributor", "org-test-2", []string{"orgA"})
	c.SetSystemIDs("distributor", "org-test-2", []string{"sys1"})

	// Expire only the system IDs
	key := rbacKey("distributor", "org-test-2")
	c.mutex.Lock()
	c.memCache[key].SystemIDsCachedAt = time.Now().Add(-rbacCacheTTL - time.Minute)
	c.mutex.Unlock()

	// Refreshing org IDs must not revive the expired system IDs
	c.SetOrgIDs("distributor", "org-test-2", []string{"orgA", "orgB"})

	_, ok := c.GetSystemIDs("distributor", "org-test-2")
	assert.False(t, ok, "expired SystemIDs must not be revived by an org-ID refresh")

	orgIDs, ok := c.GetOrgIDs("distributor", "org-test-2")
	assert.True(t, ok)
	assert.Equal(t, []string{"orgA", "orgB"}, orgIDs)

	// Symmetric case: a system-ID refresh must not revive expired org IDs
	c.mutex.Lock()
	c.memCache[key].OrgIDsCachedAt = time.Now().Add(-rbacCacheTTL - time.Minute)
	c.mutex.Unlock()

	c.SetSystemIDs("distributor", "org-test-2", []string{"sys1", "sys2"})

	_, ok = c.GetOrgIDs("distributor", "org-test-2")
	assert.False(t, ok, "expired OrgIDs must not be revived by a system-ID refresh")
}

func TestRBACCacheInvalidate(t *testing.T) {
	c := GetRBACCache()
	c.InvalidateAll()

	c.SetOrgIDs("distributor", "org-test-3", []string{"orgA"})
	c.SetSystemIDs("distributor", "org-test-3", []string{"sys1"})

	c.Invalidate("distributor", "org-test-3")

	_, ok := c.GetOrgIDs("distributor", "org-test-3")
	assert.False(t, ok)
	_, ok = c.GetSystemIDs("distributor", "org-test-3")
	assert.False(t, ok)
}
