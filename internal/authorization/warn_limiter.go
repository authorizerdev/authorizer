package authorization

import (
	"sync"
	"time"
)

// warnLimiter is a per-key rate limiter. It permits a key to "fire" at most once
// per configured window. Used to tame the authz.unmatched warn log so that a
// permissive-mode deployment does not emit one line per request.
//
// A window of zero disables rate limiting (every call returns true).
type warnLimiter struct {
	window time.Duration
	last   sync.Map // key string -> time.Time of last allow
}

// newWarnLimiter constructs a warnLimiter with the given window. A window <= 0
// disables rate limiting.
func newWarnLimiter(window time.Duration) *warnLimiter {
	return &warnLimiter{window: window}
}

// allow returns true if the key has not fired within the last window.
// It records the current time for the key on success.
func (w *warnLimiter) allow(key string) bool {
	if w.window <= 0 {
		return true
	}
	now := time.Now()
	if prev, ok := w.last.Load(key); ok {
		if now.Sub(prev.(time.Time)) < w.window {
			return false
		}
	}
	w.last.Store(key, now)
	return true
}
