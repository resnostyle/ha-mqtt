// Package poll provides an interruptible wait used by long-running services.
package poll

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func NotifyContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}

// Wait sleeps until interval elapses or ctx is cancelled.
func Wait(ctx context.Context, interval time.Duration) {
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}
