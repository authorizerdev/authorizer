// Package memerr holds the memory-store miss sentinel. It is a leaf package
// so the in-memory, DB-backed and Redis backends (subpackages of memory_store,
// which memory_store itself imports) can share it without an import cycle.
// Callers use memory_store.IsNotFound rather than this package directly.
package memerr

import "errors"

// ErrNotFound reports that the requested entry is absent (or expired). Its
// text is the "not found" every backend returned before the sentinel existed,
// so log lines are unchanged.
var ErrNotFound = errors.New("not found")
