//go:build qt

package qtapp

import (
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

const slowQtOperation = 150 * time.Millisecond

const maxQtPerformanceSamples = 80

type qtPerformanceSample struct {
	Operation string
	Elapsed   time.Duration
	At        time.Time
}

var qtPerformance = struct {
	mu      sync.Mutex
	samples []qtPerformanceSample
}{}

func traceQtOperation(name string) func() {
	started := time.Now()
	return func() {
		elapsed := time.Since(started)
		recordQtPerformance(name, elapsed, time.Now())
		event := log.Debug()
		if elapsed >= slowQtOperation {
			event = log.Info()
		}
		event.Str("operation", name).Int64("elapsedMs", elapsed.Milliseconds()).Msg("Qt performance")
	}
}

func recordQtPerformance(operation string, elapsed time.Duration, at time.Time) {
	qtPerformance.mu.Lock()
	defer qtPerformance.mu.Unlock()

	qtPerformance.samples = append(qtPerformance.samples, qtPerformanceSample{
		Operation: operation,
		Elapsed:   elapsed,
		At:        at,
	})
	if len(qtPerformance.samples) > maxQtPerformanceSamples {
		copy(qtPerformance.samples, qtPerformance.samples[len(qtPerformance.samples)-maxQtPerformanceSamples:])
		qtPerformance.samples = qtPerformance.samples[:maxQtPerformanceSamples]
	}
}

func snapshotQtPerformance() []qtPerformanceSample {
	qtPerformance.mu.Lock()
	defer qtPerformance.mu.Unlock()

	out := make([]qtPerformanceSample, len(qtPerformance.samples))
	copy(out, qtPerformance.samples)
	return out
}
