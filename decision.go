package ratelimit

import "time"

type Decision struct {
	Allowed    bool
	Remaining  int
	RetryAfter time.Duration
}
