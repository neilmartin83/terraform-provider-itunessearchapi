// Copyright Neil Martin 2026
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestStringValue_Null(t *testing.T) {
	v := types.StringNull()
	if got := StringValue(v); got != "" {
		t.Errorf("expected empty string for null, got %q", got)
	}
}

func TestStringValue_Unknown(t *testing.T) {
	v := types.StringUnknown()
	if got := StringValue(v); got != "" {
		t.Errorf("expected empty string for unknown, got %q", got)
	}
}

func TestStringValue_Set(t *testing.T) {
	v := types.StringValue("hello")
	if got := StringValue(v); got != "hello" {
		t.Errorf("expected %q, got %q", "hello", got)
	}
}

func TestStringValue_Empty(t *testing.T) {
	v := types.StringValue("")
	if got := StringValue(v); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestInt64Value_Null(t *testing.T) {
	v := types.Int64Null()
	if got := Int64Value(v); got != 0 {
		t.Errorf("expected 0 for null, got %d", got)
	}
}

func TestInt64Value_Unknown(t *testing.T) {
	v := types.Int64Unknown()
	if got := Int64Value(v); got != 0 {
		t.Errorf("expected 0 for unknown, got %d", got)
	}
}

func TestInt64Value_Set(t *testing.T) {
	v := types.Int64Value(42)
	if got := Int64Value(v); got != 42 {
		t.Errorf("expected 42, got %d", got)
	}
}

func TestInt64Value_Zero(t *testing.T) {
	v := types.Int64Value(0)
	if got := Int64Value(v); got != 0 {
		t.Errorf("expected 0, got %d", got)
	}
}

func TestBoolPointer_Null(t *testing.T) {
	v := types.BoolNull()
	if got := BoolPointer(v); got != nil {
		t.Errorf("expected nil for null, got %v", *got)
	}
}

func TestBoolPointer_Unknown(t *testing.T) {
	v := types.BoolUnknown()
	if got := BoolPointer(v); got != nil {
		t.Errorf("expected nil for unknown, got %v", *got)
	}
}

func TestBoolPointer_True(t *testing.T) {
	v := types.BoolValue(true)
	got := BoolPointer(v)
	if got == nil {
		t.Fatal("expected non-nil pointer")
	}
	if !*got {
		t.Errorf("expected true, got false")
	}
}

func TestBoolPointer_False(t *testing.T) {
	v := types.BoolValue(false)
	got := BoolPointer(v)
	if got == nil {
		t.Fatal("expected non-nil pointer")
	}
	if *got {
		t.Errorf("expected false, got true")
	}
}
