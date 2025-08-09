package ratelimit

import "time"

// Options are static parameters of a bucket.
type Options struct {
	Rate    float64             // tokens per second (e.g., 10)
	Burst   int                 // bucket capacity (e.g., 20)
	TTL     time.Duration       // key GC (PX on SET), e.g., 10 * time.Minute
	Prefix  string              // key namespace/prefix (no trailing ':')
	HashTag func(string) string // choose cluster hash-tag; nil => use id
}
