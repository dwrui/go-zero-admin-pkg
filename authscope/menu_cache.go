package authscope

import "fmt"

// MenuUserCacheKey GetMenu 用户菜单缓存 key。
// 示例：menu:superadmin:user:0:1、menu:admin:user:100:5、menu:biz:user:0:1
func MenuUserCacheKey(scope string, businessId, userId uint64) string {
	return fmt.Sprintf("menu:%s:user:%d:%d", scope, businessId, userId)
}

// MenuRouteCacheKey GetMenu 路由菜单缓存 key。
func MenuRouteCacheKey(scope string, businessId, routeId uint64) string {
	return fmt.Sprintf("menu:%s:route:%d:%d", scope, businessId, routeId)
}

// MenuCacheClearPatterns 菜单增删改后批量清理用的 Redis 匹配模式。
// 末尾 * 为通配符，会命中该 scope 下所有 businessId/userId（或 routeId）的缓存。
func MenuCacheClearPatterns(scope string) []string {
	return []string{
		fmt.Sprintf("menu:%s:user:*", scope),
		fmt.Sprintf("menu:%s:route:*", scope),
	}
}
