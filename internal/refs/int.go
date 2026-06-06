package refs

// NewInt64Ref returns a reference to a int64 with given value
func NewInt64Ref(v int64) *int64 {
	return &v
}

// Int64Value returns the value of the given bool ref
func Int64Value(r *int64) int64 {
	if r == nil {
		return 0
	}
	return *r
}
