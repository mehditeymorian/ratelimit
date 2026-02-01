// window.go
package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"strings"
	"time"
)

type WindowOptions struct {
	Limit  int           // e.g., 100
	Window time.Duration // e.g., 10 * time.Second
	TTL    time.Duration // e.g., same as Window or a bit longer
	Prefix string
}

type Window struct {
	redis func(ctx context.Context) redis.UniversalClient
	sha   string

	prefix string
	limit  int
	window time.Duration
	ttl    time.Duration
}

func NewWindowWithDynamicCtx(ctx context.Context, redisFunc func(context.Context) redis.UniversalClient, opt WindowOptions) (RateLimiter, error) {
	if opt.Limit <= 0 || opt.Window <= 0 {
		return nil, errors.New("invalid WindowOptions")
	}

	w := &Window{
		redis:  redisFunc,
		prefix: opt.Prefix,
		limit:  opt.Limit,
		window: opt.Window,
		ttl:    opt.TTL,
	}
	if w.ttl <= 0 {
		w.ttl = opt.Window
	}

	sha, err := redisFunc(ctx).ScriptLoad(ctx, luaTokenWindowScript).Result()
	if err != nil {
		return nil, err
	}
	w.sha = sha
	return w, nil
}

func NewWindow(ctx context.Context, r redis.UniversalClient, opt WindowOptions) (RateLimiter, error) {
	return NewWindowWithDynamicCtx(ctx, func(ctx context.Context) redis.UniversalClient {
		return r
	}, opt)
}

func (w *Window) Allow(ctx context.Context, id string) (Decision, error) {
	return w.AllowN(ctx, id, 1)
}

func (w *Window) AllowN(ctx context.Context, id string, cost int) (Decision, error) {
	if cost <= 0 {
		return Decision{}, errors.New("cost must be >= 1")
	}

	key := ZKey(w.prefix, id)
	nowMs := time.Now().UnixMilli()
	args := []any{
		nowMs,
		w.window.Milliseconds(),
		w.limit,
		w.ttl.Milliseconds(),
		cost,
	}

	client := w.redis(ctx)

	res, err := client.EvalSha(ctx, w.sha, []string{key}, args...).Slice()
	if isNoScript(err) {
		if w.sha, err = client.ScriptLoad(ctx, luaTokenWindowScript).Result(); err == nil {
			res, err = client.EvalSha(ctx, w.sha, []string{key}, args...).Slice()
			if err != nil {
				return Decision{}, fmt.Errorf("failed to re-execute script after NOSCRIPT error: %w", err)
			}
		}
	}
	if err != nil {
		return Decision{}, err
	}

	allowed := toInt64(res[0]) == 1
	remaining := int(toInt64(res[1]))
	retry := time.Duration(toInt64(res[2])) * time.Millisecond
	return Decision{Allowed: allowed, Remaining: remaining, RetryAfter: retry}, nil
}

func isNoScript(err error) bool {
	return err != nil && strings.Contains(err.Error(), "NOSCRIPT")
}

func ZKey(prefix, id string) string {
	return fmt.Sprintf("%s:%s", prefix, id)
}
