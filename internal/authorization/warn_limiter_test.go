package authorization

import (
	"testing"
	"time"
)

func TestWarnLimiter_FirstCallAllowed(t *testing.T) {
	l := newWarnLimiter(time.Minute)
	if !l.allow("k1") {
		t.Fatal("first call for a key must be allowed")
	}
}

func TestWarnLimiter_SecondCallSameKeyBlocked(t *testing.T) {
	l := newWarnLimiter(time.Minute)
	_ = l.allow("k1")
	if l.allow("k1") {
		t.Fatal("second call for the same key within window must be blocked")
	}
}

func TestWarnLimiter_DifferentKeysIndependent(t *testing.T) {
	l := newWarnLimiter(time.Minute)
	if !l.allow("k1") || !l.allow("k2") {
		t.Fatal("distinct keys must each be allowed independently")
	}
}

func TestWarnLimiter_ExpiryLetsThroughAgain(t *testing.T) {
	l := newWarnLimiter(10 * time.Millisecond)
	_ = l.allow("k1")
	time.Sleep(15 * time.Millisecond)
	if !l.allow("k1") {
		t.Fatal("after window expiry the key must be allowed again")
	}
}

func TestWarnLimiter_ZeroWindowAlwaysAllows(t *testing.T) {
	l := newWarnLimiter(0)
	if !l.allow("k1") || !l.allow("k1") {
		t.Fatal("zero window means rate-limiting disabled; every call must pass")
	}
}
