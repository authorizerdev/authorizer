package rate_limit

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/authorizerdev/authorizer/internal/config"
)

// Compile-time interface guard
var _ Provider = (*inMemoryProvider)(nil)

type entry struct {
	limiter  *rate.Limiter
	lastSeen time.Time // benign data race: only used for cleanup staleness heuristic
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
	v, loaded := p.visitors.LoadOrStore(ip, &entry{
		limiter:  rate.NewLimiter(p.rps, p.burst),
		lastSeen: time.Now(),
	})
	e := v.(*entry)
	e.lastSeen = time.Now()
	if loaded {
		p.visitors.Store(ip, e)
	}
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
				if time.Since(e.lastSeen) > 10*time.Minute {
					p.visitors.Delete(key)
				}
				return true
			})
		}
	}
}
