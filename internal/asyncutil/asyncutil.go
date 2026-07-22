// Package asyncutil tracks detached, fire-and-forget goroutines (email/SMS
// sends, webhook events, audit logs) that request handlers start and don't
// wait on. Go recovers panics so a background side-effect can never take
// down the process; Wait lets graceful shutdown drain them before exiting.
package asyncutil

import (
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
)

var counter atomic.Int64

// Go runs fn in a new goroutine, tracking it so Wait can block until it
// finishes and recovering any panic fn raises. log may be nil, in which
// case a recovered panic is dropped (all current call sites pass a logger).
func Go(log *zerolog.Logger, fn func()) {
	counter.Add(1)
	go func() {
		defer counter.Add(-1)
		defer func() {
			if r := recover(); r != nil && log != nil {
				log.Error().Interface("panic", r).Msg("recovered panic in background goroutine")
			}
		}()
		fn()
	}()
}

// Wait blocks until every goroutine started via Go has finished. Intended
// for use once during graceful shutdown, after listeners have stopped
// accepting new requests.
func Wait(log zerolog.Logger) {
	if n := counter.Load(); n > 0 {
		log.Info().Int64("active_goroutines", n).Msg("waiting for background goroutines to complete")
	}
	for counter.Load() > 0 {
		time.Sleep(100 * time.Millisecond)
	}
}
