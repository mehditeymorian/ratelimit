package ratelimit

const luaTokenWindowScript = `
-- KEYS[1]   = zset key (events within the window)
-- ARGV[1]   = now_ms
-- ARGV[2]   = window_ms
-- ARGV[3]   = limit (max events allowed in the window)
-- ARGV[4]   = ttl_ms (PEXPIRE to GC idle keys)
-- ARGV[5]   = cost (optional, >=1; default 1; clamped to 100 to avoid abuse)

local zkey       = KEYS[1]
local now        = tonumber(ARGV[1])
local window_ms  = tonumber(ARGV[2])
local limit      = tonumber(ARGV[3])
local ttl        = tonumber(ARGV[4])
local cost       = tonumber(ARGV[5] or "1")

if cost < 1 then cost = 1 end
if cost > 100 then cost = 100 end -- safety cap

-- 1) drop items outside the window
redis.call("ZREMRANGEBYSCORE", zkey, 0, now - window_ms)

-- 2) count current items
local count = tonumber(redis.call("ZCARD", zkey))

-- 3) allow?
if (count + cost) <= limit then
  -- add 'cost' members at now (unique members so they don't collide)
  for i = 1, cost do
	local time = redis.call("TIME")
	local member = string.format("%d:%d:%d", time[1], time[2],i)
    redis.call("ZADD", zkey, now, member)
  end
  redis.call("PEXPIRE", zkey, ttl)
  local remaining = limit - (count + cost)
  return {1, remaining, 0}
end

-- 4) deny: compute retry_after from the oldest score remaining
local oldest = redis.call("ZRANGE", zkey, 0, 0, "WITHSCORES")
local oldestScore = now
if oldest and #oldest >= 2 then
  oldestScore = tonumber(oldest[2])
end
local retry = (oldestScore + window_ms) - now
if retry < 0 then retry = 0 end
local remaining = math.max(0, limit - count)
return {0, remaining, retry}
`
