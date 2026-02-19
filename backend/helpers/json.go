/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package helpers

import "reflect"

// EnsureSlice ensures a nil slice is returned as an empty slice for proper JSON serialization.
// In Go, nil slices serialize to `null` in JSON, while empty slices serialize to `[]`.
// This function converts nil slices to empty slices of the same type.
func EnsureSlice(v interface{}) interface{} {
	if v == nil {
		return []interface{}{}
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Slice && rv.IsNil() {
		return reflect.MakeSlice(rv.Type(), 0, 0).Interface()
	}
	return v
}
