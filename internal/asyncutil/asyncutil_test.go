package asyncutil

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestGoWaitRunsAndDrains(t *testing.T) {
	var ran atomic.Bool
	Go(nil, func() {
		time.Sleep(20 * time.Millisecond)
		ran.Store(true)
	})
	Wait(zerolog.Nop())
	if !ran.Load() {
		t.Fatal("Wait returned before the goroutine finished")
	}
}

func TestGoRecoversPanic(t *testing.T) {
	log := zerolog.Nop()
	Go(&log, func() {
		panic("boom")
	})
	Wait(log) // must not propagate the panic to the test goroutine
}
