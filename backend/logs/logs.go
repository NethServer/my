/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package logs

import (
	"log"
	"os"
)

var Logs *log.Logger

func Init(name string) {
	logger := log.New(os.Stderr, name+" ", log.Ldate|log.Ltime|log.Lshortfile)
	Logs = logger
}
