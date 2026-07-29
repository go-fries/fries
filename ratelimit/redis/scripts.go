package redis

import goredis "github.com/redis/go-redis/v9"

var takeScript = goredis.NewScript(`
local current_time = redis.call("TIME")
local now = current_time[1] * 1000000 + current_time[2]
local interval = tonumber(ARGV[1])
local burst = tonumber(ARGV[2])
local cost = tonumber(ARGV[3])
local burst_offset = interval * burst
local increment = interval * cost

local tat = tonumber(redis.call("GET", KEYS[1])) or now
if tat < now then
    tat = now
end

local new_tat = tat + increment
local allow_at = new_tat - burst_offset
if allow_at > now then
    local remaining = math.floor((now + burst_offset - tat) / interval)
    if remaining < 0 then
        remaining = 0
    end
    return {0, remaining, allow_at - now, tat - now}
end

local remaining = math.floor((now + burst_offset - new_tat) / interval)
if remaining < 0 then
    remaining = 0
end
local reset_after = new_tat - now
local ttl = math.max(1, math.ceil(reset_after / 1000))
redis.call("SET", KEYS[1], string.format("%.0f", new_tat), "PX", ttl)
return {1, remaining, 0, reset_after}
`)
