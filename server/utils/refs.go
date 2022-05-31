package utils

// NewStringRef returns a reference to a string with given value
func NewStringRef(v string) *string {
	return &v
}
