package poll

import (
	"context"
	"testing"
	"time"
)

func TestWaitCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	Wait(ctx, time.Second)
	if time.Since(start) > 200*time.Millisecond {
		t.Fatal("wait should return immediately when cancelled")
	}
}

func TestWaitInterval(t *testing.T) {
	ctx := context.Background()
	start := time.Now()
	Wait(ctx, 20*time.Millisecond)
	if time.Since(start) < 15*time.Millisecond {
		t.Fatal("wait returned too soon")
	}
}
