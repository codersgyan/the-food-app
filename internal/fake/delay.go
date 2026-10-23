// Package fake stands in for the outside services the app talks to: the OTP
// service that texts login codes, and the payment provider. Each one waits a
// little before it answers, like a real service over the network.
package fake

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"
)

// Delay is a random wait between Min and Max.
type Delay struct {
	Min, Max time.Duration
}

// ParseDelay reads a delay like "1s-2s" (a random wait in that range) or "300ms" (always that long).
func ParseDelay(s string) (Delay, error) {
	lo, hi, isRange := strings.Cut(s, "-")
	min, err := time.ParseDuration(lo)
	if err != nil {
		return Delay{}, fmt.Errorf("delay %q: %w", s, err)
	}
	max := min
	if isRange {
		if max, err = time.ParseDuration(hi); err != nil {
			return Delay{}, fmt.Errorf("delay %q: %w", s, err)
		}
	}
	if min < 0 || max < min {
		return Delay{}, fmt.Errorf("delay %q: want min-max with 0 <= min <= max", s)
	}
	return Delay{Min: min, Max: max}, nil
}

// wait sleeps for a random time between Min and Max, or until ctx is done.
func (d Delay) wait(ctx context.Context) error {
	t := d.Min
	if d.Max > d.Min {
		t += rand.N(d.Max - d.Min)
	}
	timer := time.NewTimer(t)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
