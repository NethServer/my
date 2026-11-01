/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package methods

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/nethesis/my/services/support/logger"
	"github.com/nethesis/my/services/support/models"
	"github.com/nethesis/my/services/support/session"
)

// Routes behind the NS8 cluster login answer with an HTTP Basic challenge: in
// a browser that is a native dialog that blocks the whole support window. The
// proxy answers it with the cluster admin the appliance created for the
// session, so the credentials never go through the browser.

const clusterAdminCacheTTL = time.Minute

type clusterAdminEntry struct {
	cred      *models.UserCredential
	expiresAt time.Time
}

var (
	clusterAdminMu    sync.Mutex
	clusterAdminCache = map[string]clusterAdminEntry{}
	// swappable in tests
	lookupClusterAdmin = session.ClusterAdminForSession
)

func clusterAdminFor(sessionID string) *models.UserCredential {
	now := time.Now()
	clusterAdminMu.Lock()
	for id, e := range clusterAdminCache {
		if now.After(e.expiresAt) {
			delete(clusterAdminCache, id)
		}
	}
	entry, ok := clusterAdminCache[sessionID]
	clusterAdminMu.Unlock()
	if ok {
		return entry.cred
	}

	cred, err := lookupClusterAdmin(sessionID)
	if err != nil {
		logger.ComponentLogger("proxy").Warn().Err(err).Str("session_id", sessionID).Msg("cannot load the cluster admin for the cluster login")
		return nil
	}
	if cred == nil {
		// not reported yet: ask again on the next request
		return nil
	}

	clusterAdminMu.Lock()
	clusterAdminCache[sessionID] = clusterAdminEntry{cred: cred, expiresAt: now.Add(clusterAdminCacheTTL)}
	clusterAdminMu.Unlock()
	return cred
}

// ForgetClusterAdmin drops the cached credentials of a session that ended
func ForgetClusterAdmin(sessionID string) {
	clusterAdminMu.Lock()
	delete(clusterAdminCache, sessionID)
	clusterAdminMu.Unlock()
}

// applyClusterLogin signs a request to a cluster-login route with the session's
// cluster admin, unless the browser already sent credentials of its own. Only
// the page's own requests and navigations get them: a fetch from another
// service of the session (CORS between siblings is allowed) must not act as
// cluster admin. Sec-Fetch-* are set by the browser, a page cannot forge them.
// Reports whether it did.
func applyClusterLogin(req *http.Request, auth, sessionID string) bool {
	if auth != "cluster" || req.Header.Get("Authorization") != "" {
		return false
	}
	site := strings.ToLower(req.Header.Get("Sec-Fetch-Site"))
	mode := strings.ToLower(req.Header.Get("Sec-Fetch-Mode"))
	if site != "same-origin" && site != "none" && mode != "navigate" {
		return false
	}
	if cred := clusterAdminFor(sessionID); cred != nil && cred.Username != "" {
		req.SetBasicAuth(cred.Username, cred.Password)
		return true
	}
	return false
}
