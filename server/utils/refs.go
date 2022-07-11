package utils

// NewStringRef returns a reference to a string with given value
func NewStringRef(v string) *string {
	return &v
}

// StringValue returns the value of the given string ref
func StringValue(r *string, defaultValue ...string) string {
	if r != nil {
		return *r
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return ""
}

// NewBoolRef returns a reference to a bool with given value
func NewBoolRef(v bool) *bool {
	return &v
}

// BoolValue returns the value of the given bool ref
func BoolValue(r *bool) bool {
	if r == nil {
		return false
	}
	return *r
}
