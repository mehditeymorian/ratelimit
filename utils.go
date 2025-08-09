package ratelimit

import "strconv"

func toInt64(v any) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case float64:
		return int64(t)
	case string:
		i, _ := strconv.ParseInt(t, 10, 64)
		return i
	default:
		return 0
	}
}
