package cache

const (
	// unlockScript 解锁脚本：防止误删别人的锁
	unlockScript = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("DEL", KEYS[1])
end
return 0
`

	// nullValue 表示空值缓存的标记
	nullValue = `"__NULL__"`
)
