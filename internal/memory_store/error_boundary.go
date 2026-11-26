package memory_store

import "github.com/authorizerdev/authorizer/internal/internalerr"

// errorSource is the internalerr source recorded for memory-store faults.
const errorSource = "memory_store"

// errorBoundary decorates a memory-store Provider so every error it returns is
// marked internal (internalerr.Wrap). Redis and in-memory errors otherwise
// reach API clients verbatim, while the database-backed store's errors were
// already masked as storage errors — this keeps the three backends at parity.
// It is applied once, in New; the compile-time assertion below forces every new
// Provider method to be added here as well.
//
// Wrapping is transparent to errors.Is / errors.As, so callers that compare
// against a driver sentinel keep working.
type errorBoundary struct {
	next Provider
}

var _ Provider = (*errorBoundary)(nil)

func wrapErr(err error) error { return internalerr.Wrap(errorSource, err) }

func (p *errorBoundary) SetUserSession(userId, key, token string, expiration int64) error {
	return wrapErr(p.next.SetUserSession(userId, key, token, expiration))
}

func (p *errorBoundary) GetUserSession(userId, key string) (string, error) {
	r0, err := p.next.GetUserSession(userId, key)
	return r0, wrapErr(err)
}

func (p *errorBoundary) DeleteUserSession(userId, key string) error {
	return wrapErr(p.next.DeleteUserSession(userId, key))
}

func (p *errorBoundary) ClaimRefreshToken(userId, nonce string) (bool, error) {
	r0, err := p.next.ClaimRefreshToken(userId, nonce)
	return r0, wrapErr(err)
}

func (p *errorBoundary) DeleteAllUserSessions(userId string) error {
	return wrapErr(p.next.DeleteAllUserSessions(userId))
}

func (p *errorBoundary) DeleteSessionForNamespace(namespace string) error {
	return wrapErr(p.next.DeleteSessionForNamespace(namespace))
}

func (p *errorBoundary) SetMfaSession(userId, key, purpose string, expiration int64) error {
	return wrapErr(p.next.SetMfaSession(userId, key, purpose, expiration))
}

func (p *errorBoundary) GetMfaSession(userId, key string) (string, error) {
	r0, err := p.next.GetMfaSession(userId, key)
	return r0, wrapErr(err)
}

func (p *errorBoundary) GetAllMfaSessions(userId string) ([]string, error) {
	r0, err := p.next.GetAllMfaSessions(userId)
	return r0, wrapErr(err)
}

func (p *errorBoundary) DeleteMfaSession(userId, key string) error {
	return wrapErr(p.next.DeleteMfaSession(userId, key))
}

func (p *errorBoundary) GetMfaSessionOwner(key string) (string, string, error) {
	r0, r1, err := p.next.GetMfaSessionOwner(key)
	return r0, r1, wrapErr(err)
}

func (p *errorBoundary) SetState(key, state string) error {
	return wrapErr(p.next.SetState(key, state))
}

func (p *errorBoundary) GetState(key string) (string, error) {
	r0, err := p.next.GetState(key)
	return r0, wrapErr(err)
}

func (p *errorBoundary) RemoveState(key string) error {
	return wrapErr(p.next.RemoveState(key))
}

func (p *errorBoundary) GetAndRemoveState(key string) (string, error) {
	r0, err := p.next.GetAndRemoveState(key)
	return r0, wrapErr(err)
}

func (p *errorBoundary) SetCache(key string, value string, ttlSeconds int64) error {
	return wrapErr(p.next.SetCache(key, value, ttlSeconds))
}

func (p *errorBoundary) SetCacheNX(key string, value string, ttlSeconds int64) (bool, error) {
	r0, err := p.next.SetCacheNX(key, value, ttlSeconds)
	return r0, wrapErr(err)
}

func (p *errorBoundary) GetCache(key string) (string, error) {
	r0, err := p.next.GetCache(key)
	return r0, wrapErr(err)
}

func (p *errorBoundary) DeleteCacheByPrefix(prefix string) error {
	return wrapErr(p.next.DeleteCacheByPrefix(prefix))
}

func (p *errorBoundary) IncrementCache(key string, ttlSeconds int64) (int64, error) {
	r0, err := p.next.IncrementCache(key, ttlSeconds)
	return r0, wrapErr(err)
}

func (p *errorBoundary) GetAllData() (map[string]string, error) {
	r0, err := p.next.GetAllData()
	return r0, wrapErr(err)
}
