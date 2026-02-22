// Copyright Neil Martin 2026
// SPDX-License-Identifier: MPL-2.0

package common

// ChunkInt64 splits a slice of int64 values into batches of the given size.
func ChunkInt64(ids []int64, batchSize int) [][]int64 {
	var batches [][]int64
	for i := 0; i < len(ids); i += batchSize {
		end := min(i+batchSize, len(ids))
		batches = append(batches, ids[i:end])
	}
	return batches
}

// ChunkStrings splits a slice of strings into batches of the given size.
func ChunkStrings(values []string, batchSize int) [][]string {
	var batches [][]string
	for i := 0; i < len(values); i += batchSize {
		end := min(i+batchSize, len(values))
		batches = append(batches, values[i:end])
	}
	return batches
}
