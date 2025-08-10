package ratelimit

import "context"

type RateLimiter interface {
	Allow(ctx context.Context, id string) (Decision, error)
	AllowN(ctx context.Context, id string, cost int) (Decision, error)
}
