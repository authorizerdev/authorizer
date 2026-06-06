package refs

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
