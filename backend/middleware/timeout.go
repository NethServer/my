/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/logger"
)

// ExtendDeadline raises the per-connection read/write deadline for a single slow,
// long-running endpoint (bulk CSV import confirm creates one organization per row
// in Logto, self-throttled by 429 backoff, so a few hundred rows take tens of
// seconds). The global http.Server ReadTimeout/WriteTimeout stay low (slowloris
// protection) for every other route; net/http re-applies them to the next request
// on the connection, so this override is scoped to this request only.
//
// It relies on the ResponseWriter exposing SetWriteDeadline, which gin surfaces
// through Unwrap() (gin >= 1.9). If the writer does not support it the request
// simply keeps the server default instead of failing.
func ExtendDeadline(d time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		rc := http.NewResponseController(c.Writer)
		deadline := time.Now().Add(d)
		if err := rc.SetWriteDeadline(deadline); err != nil {
			logger.Warn().Err(err).Str("path", c.FullPath()).Msg("could not extend write deadline; keeping server WriteTimeout")
		}
		if err := rc.SetReadDeadline(deadline); err != nil {
			logger.Warn().Err(err).Str("path", c.FullPath()).Msg("could not extend read deadline; keeping server ReadTimeout")
		}
		c.Next()
	}
}
