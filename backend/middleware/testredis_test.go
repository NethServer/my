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
	"os"
	"sync"

	"github.com/alicebob/miniredis/v2"
)

// The JWT middleware fails closed when the revocation blacklist cannot be
// read, so every test that drives it needs a Redis that answers. CI has none,
// and a developer's local one must not be what makes the suite pass: an
// in-memory Redis is started once per package and REDIS_URL points at it
// before the configuration is loaded.
var (
	testRedisOnce sync.Once
	testRedisURL  string
)

func testRedisEnv() {
	testRedisOnce.Do(func() {
		srv, err := miniredis.Run()
		if err != nil {
			panic("cannot start the in-memory redis for tests: " + err.Error())
		}
		testRedisURL = "redis://" + srv.Addr()
	})
	_ = os.Setenv("REDIS_URL", testRedisURL)
}
