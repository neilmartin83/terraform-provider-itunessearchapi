// Copyright Neil Martin 2026
// SPDX-License-Identifier: MPL-2.0

package common

import "github.com/hashicorp/terraform-plugin-framework/types"

// StringValue returns the underlying string from a Terraform types.String,
// or an empty string if the value is null or unknown.
func StringValue(v types.String) string {
	if v.IsNull() || v.IsUnknown() {
		return ""
	}
	return v.ValueString()
}

// Int64Value returns the underlying int64 from a Terraform types.Int64,
// or zero if the value is null or unknown.
func Int64Value(v types.Int64) int64 {
	if v.IsNull() || v.IsUnknown() {
		return 0
	}
	return v.ValueInt64()
}

// BoolPointer returns a pointer to the underlying bool from a Terraform types.Bool,
// or nil if the value is null or unknown.
func BoolPointer(v types.Bool) *bool {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	val := v.ValueBool()
	return &val
}
