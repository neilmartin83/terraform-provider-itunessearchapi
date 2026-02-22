// Copyright Neil Martin 2026
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"testing"
)

func TestChunkInt64_Empty(t *testing.T) {
	result := ChunkInt64(nil, 200)
	if len(result) != 0 {
		t.Errorf("expected 0 batches, got %d", len(result))
	}
}

func TestChunkInt64_SmallerThanBatch(t *testing.T) {
	ids := []int64{1, 2, 3}
	result := ChunkInt64(ids, 200)
	if len(result) != 1 {
		t.Fatalf("expected 1 batch, got %d", len(result))
	}
	if len(result[0]) != 3 {
		t.Errorf("expected 3 items in batch, got %d", len(result[0]))
	}
}

func TestChunkInt64_ExactBatch(t *testing.T) {
	ids := make([]int64, 200)
	for i := range ids {
		ids[i] = int64(i)
	}
	result := ChunkInt64(ids, 200)
	if len(result) != 1 {
		t.Fatalf("expected 1 batch, got %d", len(result))
	}
	if len(result[0]) != 200 {
		t.Errorf("expected 200 items in batch, got %d", len(result[0]))
	}
}

func TestChunkInt64_MultipleBatches(t *testing.T) {
	ids := make([]int64, 450)
	for i := range ids {
		ids[i] = int64(i)
	}
	result := ChunkInt64(ids, 200)
	if len(result) != 3 {
		t.Fatalf("expected 3 batches, got %d", len(result))
	}
	if len(result[0]) != 200 {
		t.Errorf("expected 200 items in first batch, got %d", len(result[0]))
	}
	if len(result[1]) != 200 {
		t.Errorf("expected 200 items in second batch, got %d", len(result[1]))
	}
	if len(result[2]) != 50 {
		t.Errorf("expected 50 items in third batch, got %d", len(result[2]))
	}
}

func TestChunkStrings_Empty(t *testing.T) {
	result := ChunkStrings(nil, 200)
	if len(result) != 0 {
		t.Errorf("expected 0 batches, got %d", len(result))
	}
}

func TestChunkStrings_SmallerThanBatch(t *testing.T) {
	values := []string{"a", "b"}
	result := ChunkStrings(values, 200)
	if len(result) != 1 {
		t.Fatalf("expected 1 batch, got %d", len(result))
	}
	if len(result[0]) != 2 {
		t.Errorf("expected 2 items in batch, got %d", len(result[0]))
	}
}

func TestChunkStrings_MultipleBatches(t *testing.T) {
	values := make([]string, 350)
	for i := range values {
		values[i] = "item"
	}
	result := ChunkStrings(values, 200)
	if len(result) != 2 {
		t.Fatalf("expected 2 batches, got %d", len(result))
	}
	if len(result[0]) != 200 {
		t.Errorf("expected 200 items in first batch, got %d", len(result[0]))
	}
	if len(result[1]) != 150 {
		t.Errorf("expected 150 items in second batch, got %d", len(result[1]))
	}
}
