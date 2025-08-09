# 🚦 Go Sliding Window Rate Limiter (Redis + Lua)

A **high-performance, distributed rate limiter** for Go using **Redis ZSET** and **Lua scripting**.  
Implements a **strict sliding window algorithm** with atomic operations for accuracy and race-safety in distributed environments.

> Ideal for APIs, background jobs, and any service where you need precise "N requests in the last W seconds" limits.

---

## ✨ Features

- ⚡ **Strict sliding window** – never exceeds `limit` requests in `(now - window, now]`.
- 🛡 **Atomic with Lua** – safe in high concurrency, no race conditions.
- 🎯 **Cost-based limiting** – charge more than 1 token per request.
- 🔧 **Configurable TTL** – keeps memory bounded by auto-expiring idle keys.
- 🛠 **Universal Redis client** – works with standalone, sentinel, or cluster setups.

---

## 📦 Installation

```bash
go get github.com/mehditeymorian/ratelimit
```

## 🛠 Usage
```go
import (
    "context"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
    "github.com/your/repo/ratelimit"
)

func main() {
    ctx := context.Background()

    // Works with standalone, sentinel, or cluster
    rdb := redis.NewUniversalClient(&redis.UniversalOptions{
        Addrs: []string{"127.0.0.1:6379"},
        // Sentinel: MasterName, SentinelAddrs: [...]
        // Cluster:  Addrs: [...]
    })


    limiter, err := ratelimit.NewWindow(ctx, rdb, ratelimit.WindowOptions{
        Limit:     100,                 // max 100 requests
        Window:    10 * time.Second,    // in a 10-second window
        TTL:       12 * time.Second,    // expire keys after idle
        Prefix: "rl",                // Redis key prefix
    })
    if err != nil {
        panic(err)
    }

    dec, err := limiter.Allow(ctx, "user:42")
    if err != nil {
        panic(err)
    }
    if dec.Allowed {
        fmt.Println("Request allowed")
    } else {
        fmt.Printf("Rate limit exceeded. Retry after %v\n", dec.RetryAfter)
    }
}

```
