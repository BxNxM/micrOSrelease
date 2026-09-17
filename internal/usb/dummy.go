package usb

import (
	"context"
	"time"
)

func (m DummyManager) delay() time.Duration {
	if m.Delay <= 0 {
		return 1200 * time.Millisecond
	}
	return m.Delay
}

func wait(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
