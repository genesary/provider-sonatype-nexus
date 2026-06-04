// Package utils provides simple helper functions used across all controllers.
// These utilities are designed to be easy to understand for Go beginners.
package utils

// Bool returns a pointer to the given bool value.
// Usage: Bool(true) returns *true.
func Bool(b bool) *bool { return &b }

// BoolValue safely dereferences a bool pointer.
// Returns false if the pointer is nil.
func BoolValue(b *bool) bool {
	if b == nil {
		return false
	}

	return *b
}

// BoolValueDefault safely dereferences a bool pointer with a custom default.
// Returns defaultVal if the pointer is nil.
func BoolValueDefault(b *bool, defaultVal bool) bool {
	if b == nil {
		return defaultVal
	}

	return *b
}

// String returns a pointer to the given string value.
// Usage: String("hello") returns *"hello".
func String(s string) *string { return &s }

// StringValue safely dereferences a string pointer.
// Returns empty string if the pointer is nil.
func StringValue(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}

// Int32 returns a pointer to the given int32 value.
// Usage: Int32(42) returns *42.
func Int32(i int32) *int32 { return &i }

// Int32Value safely dereferences an int32 pointer.
// Returns 0 if the pointer is nil.
func Int32Value(i *int32) int32 {
	if i == nil {
		return 0
	}

	return *i
}

// Int32ValueDefault safely dereferences an int32 pointer with a custom default.
// Returns defaultVal if the pointer is nil.
func Int32ValueDefault(i *int32, defaultVal int32) int32 {
	if i == nil {
		return defaultVal
	}

	return *i
}

// Int64 returns a pointer to the given int64 value.
// Usage: Int64(42) returns *42.
func Int64(i int64) *int64 { return &i }

// Int64Value safely dereferences an int64 pointer.
// Returns 0 if the pointer is nil.
func Int64Value(i *int64) int64 {
	if i == nil {
		return 0
	}

	return *i
}
