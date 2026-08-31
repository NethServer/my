/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package models

import "time"

// UnregisterResponse acknowledges that a system gave up its credentials.
type UnregisterResponse struct {
	SystemKey      string    `json:"system_key"`
	UnregisteredAt time.Time `json:"unregistered_at"`
}
