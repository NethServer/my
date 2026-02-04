/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package helpers

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProcessInBatchesEmpty(t *testing.T) {
	var items []int
	called := false
	err := ProcessInBatches(items, 10, func(batch []int) error {
		called = true
		return nil
	})
	assert.NoError(t, err)
	assert.False(t, called, "processFn is not called for empty slice")
}

func TestProcessInBatchesSingleBatch(t *testing.T) {
	items := []int{1, 2, 3}
	var batches [][]int

	err := ProcessInBatches(items, 10, func(batch []int) error {
		batches = append(batches, batch)
		return nil
	})

	assert.NoError(t, err)
	assert.Len(t, batches, 1)
	assert.Equal(t, []int{1, 2, 3}, batches[0])
}

func TestProcessInBatchesMultipleBatches(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 6, 7}
	var batches [][]int

	err := ProcessInBatches(items, 3, func(batch []int) error {
		// Copy the batch to avoid slice aliasing
		b := make([]int, len(batch))
		copy(b, batch)
		batches = append(batches, b)
		return nil
	})

	assert.NoError(t, err)
	assert.Len(t, batches, 3)
	assert.Equal(t, []int{1, 2, 3}, batches[0])
	assert.Equal(t, []int{4, 5, 6}, batches[1])
	assert.Equal(t, []int{7}, batches[2])
}

func TestProcessInBatchesExactDivision(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 6}
	callCount := 0

	err := ProcessInBatches(items, 3, func(batch []int) error {
		callCount++
		assert.Len(t, batch, 3)
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 2, callCount)
}

func TestProcessInBatchesBatchSizeOne(t *testing.T) {
	items := []string{"a", "b", "c"}
	callCount := 0

	err := ProcessInBatches(items, 1, func(batch []string) error {
		assert.Len(t, batch, 1)
		callCount++
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 3, callCount)
}

func TestProcessInBatchesErrorStopsProcessing(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 6}
	callCount := 0
	expectedErr := fmt.Errorf("batch failure")

	err := ProcessInBatches(items, 2, func(batch []int) error {
		callCount++
		if callCount == 2 {
			return expectedErr
		}
		return nil
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "batch failure")
	assert.Equal(t, 2, callCount, "processing stops on first error")
}

func TestProcessInBatchesWithStructs(t *testing.T) {
	type item struct {
		ID   int
		Name string
	}

	items := []item{
		{1, "one"}, {2, "two"}, {3, "three"}, {4, "four"}, {5, "five"},
	}

	var processedIDs []int
	err := ProcessInBatches(items, 2, func(batch []item) error {
		for _, it := range batch {
			processedIDs = append(processedIDs, it.ID)
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3, 4, 5}, processedIDs)
}
