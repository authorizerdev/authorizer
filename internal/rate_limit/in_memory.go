package rate_limit

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"

	"github.com/authorizerdev/authorizer/internal/config"
)

// Compile-time interface guard
var _ Provider = (*inMemoryProvider)(nil)

// entry stores a per-IP rate limiter together with the last-seen timestamp
// used by the cleanup goroutine. lastSeen is accessed concurrently by
// Allow() and cleanup(), so it MUST be touched only via the atomic helpers
// below — using a plain time.Time field would trip the race detector and
// (more importantly) drop updates non-deterministically.
type entry struct {
	limiter  *rate.Limiter
	lastSeen atomic.Int64 // unix nanoseconds
}

func (e *entry) touch() {
	e.lastSeen.Store(time.Now().UnixNano())
}

func (e *entry) lastSeenTime() time.Time {
	return time.Unix(0, e.lastSeen.Load())
}

type inMemoryProvider struct {
	visitors sync.Map
	rps      rate.Limit
	burst    int
	cancel   context.CancelFunc
}

func newInMemoryProvider(cfg *config.Config, deps *Dependencies) (*inMemoryProvider, error) {
	ctx, cancel := context.WithCancel(context.Background())
	p := &inMemoryProvider{
		rps:    rate.Limit(cfg.RateLimitRPS),
		burst:  cfg.RateLimitBurst,
		cancel: cancel,
	}
	go p.cleanup(ctx)
	return p, nil
}

// Allow checks if a request from the given IP is allowed
func (p *inMemoryProvider) Allow(_ context.Context, ip string) (bool, error) {
	newEntry := &entry{limiter: rate.NewLimiter(p.rps, p.burst)}
	newEntry.touch()
	v, _ := p.visitors.LoadOrStore(ip, newEntry)
	e := v.(*entry)
	e.touch()
	return e.limiter.Allow(), nil
}

// Close stops the cleanup goroutine
func (p *inMemoryProvider) Close() error {
	p.cancel()
	return nil
}

// cleanup removes stale entries every 5 minutes
func (p *inMemoryProvider) cleanup(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.visitors.Range(func(key, value any) bool {
				e := value.(*entry)
				if time.Since(e.lastSeenTime()) > 10*time.Minute {
					p.visitors.Delete(key)
				}
				return true
			})
		}
	}
}
