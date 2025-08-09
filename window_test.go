package ratelimit_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/mehditeymorian/ratelimit"
	"github.com/redis/go-redis/v9"
)

func redisClientFromEnv() redis.UniversalClient {
	return redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs: []string{"redis.test.svc.cluster.local:6379"},
		// You can also pass Sentinel/Cluster options via env in CI.
	})
}

func TestSlidingWindow_Integration(t *testing.T) {
	ctx := context.Background()
	rdb := redisClientFromEnv()

	idBase := "user-"

	tests := []struct {
		name   string
		limit  int
		window time.Duration
		step   func(t *testing.T, w *ratelimit.Window, id string)
	}{
		{
			name:   "allow up to limit then deny",
			limit:  5,
			window: 3 * time.Second,
			step: func(t *testing.T, w *ratelimit.Window, id string) {
				for i := 0; i < 5; i++ {
					dec, err := w.Allow(context.Background(), id)
					if err != nil {
						t.Fatalf("err: %v", err)
					}
					if !dec.Allowed {
						t.Fatalf("unexpected deny at i=%d", i)
					}

				}
				dec, err := w.Allow(context.Background(), id)
				if err != nil {
					t.Fatalf("err: %v", err)
				}
				if dec.Allowed {
					t.Fatalf("expected deny after hitting limit")
				}
				if dec.RetryAfter <= 0 {
					t.Fatalf("expected positive retry-after, got %v", dec.RetryAfter)
				}
			},
		},
		{
			name:   "window expiry permits again",
			limit:  3,
			window: 1500 * time.Millisecond,
			step: func(t *testing.T, w *ratelimit.Window, id string) {
				for i := 0; i < 3; i++ {
					dec, err := w.Allow(context.Background(), id)
					if err != nil || !dec.Allowed {
						t.Fatalf("allow %d failed: dec=%+v err=%v", i, dec, err)
					}
				}
				dec, _ := w.Allow(context.Background(), id)
				if dec.Allowed {
					t.Fatalf("should be denied before expiry")
				}

				// advance past window
				time.Sleep(1600 * time.Millisecond)

				dec2, err := w.Allow(context.Background(), id)
				if err != nil {
					t.Fatalf("err: %v", err)
				}
				if !dec2.Allowed {
					t.Fatalf("expected allow after expiry; got %+v", dec2)
				}
			},
		},
		{
			name:   "cost-based allowance",
			limit:  10,
			window: 5 * time.Second,
			step: func(t *testing.T, w *ratelimit.Window, id string) {
				dec, err := w.AllowN(context.Background(), id, 7)
				fmt.Printf("dec=%+v\n", dec)
				if err != nil {
					t.Fatalf("err: %v", err)
				}
				if !dec.Allowed || dec.Remaining != 3 {
					t.Fatalf("expected allowed with remaining=3, got %+v", dec)
				}
				dec2, err := w.AllowN(context.Background(), id, 4) // 7+4 > 10
				fmt.Printf("dec=%+v\n", dec)
				if err != nil {
					t.Fatalf("err: %v", err)
				}
				if dec2.Allowed {
					t.Fatalf("expected deny on cost overflow")
				}
				if dec2.RetryAfter <= 0 {
					t.Fatalf("expected positive retry-after")
				}
			},
		},
		{
			name:   "burst many small then deny",
			limit:  3,
			window: 1 * time.Second,
			step: func(t *testing.T, w *ratelimit.Window, id string) {
				for i := 0; i < 3; i++ {
					if dec, err := w.Allow(context.Background(), id); err != nil || !dec.Allowed {
						t.Fatalf("i=%d allow failed: %+v err=%v", i, dec, err)
					}
				}
				dec, _ := w.Allow(context.Background(), id)
				if dec.Allowed {
					t.Fatalf("expected deny at 4th")
				}
			},
		},
	}

	for i, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			id := idBase + strconv.Itoa(i)
			opts := ratelimit.WindowOptions{
				Prefix: "rate",
				Limit:  tc.limit,
				Window: tc.window,
				TTL:    tc.window + 2*time.Second,
			}
			w, err := ratelimit.NewWindow(ctx, rdb, opts)
			if err != nil {
				t.Fatalf("NewWindow error: %v", err)
			}

			tc.step(t, w, id)
		})
	}
}
