/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package client

import "strings"

// UserProfileFromName mirrors backend/models.UserProfileFromName: the last
// word is the family name
func UserProfileFromName(name string) map[string]interface{} {
	words := strings.Fields(name)
	switch len(words) {
	case 0:
		return nil
	case 1:
		return map[string]interface{}{"givenName": words[0]}
	default:
		return map[string]interface{}{
			"givenName":  strings.Join(words[:len(words)-1], " "),
			"familyName": words[len(words)-1],
		}
	}
}
