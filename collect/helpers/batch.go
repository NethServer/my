/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package helpers

import "fmt"

// ProcessInBatches processes items in batches of the given size,
// calling processFn for each batch. Returns an error if any batch fails.
func ProcessInBatches[T any](items []T, batchSize int, processFn func(batch []T) error) error {
	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}

		if err := processFn(items[i:end]); err != nil {
			return fmt.Errorf("failed to process batch %d-%d: %w", i, end, err)
		}
	}
	return nil
}
