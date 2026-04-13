package db

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/dwrui/go-zero-admin-pkg/utils/tools/gcache"
)

type CacheConfig struct {
	Enabled     bool          `json:"enabled" yaml:"enabled"`
	DefaultTTL  time.Duration `json:"default_ttl" yaml:"default_ttl"`
	Prefix      string        `json:"prefix" yaml:"prefix"`
	CacheOnRead bool          `json:"cache_on_read" yaml:"cache_on_read"`
}

var defaultCacheConfig = &CacheConfig{
	Enabled:     true,
	DefaultTTL:  15 * time.Minute,
	Prefix:      "db_cache:",
	CacheOnRead: true,
}

func SetDefaultCacheConfig(config *CacheConfig) {
	if config != nil {
		defaultCacheConfig = config
	}
}

func GetDefaultCacheConfig() *CacheConfig {
	return defaultCacheConfig
}

type CacheOption func(*CacheOptions)

type CacheOptions struct {
	TTL     time.Duration
	Prefix  string
	Skip    bool
	Refresh bool
}

func WithCacheTTL(ttl time.Duration) CacheOption {
	return func(o *CacheOptions) {
		o.TTL = ttl
	}
}

func WithCachePrefix(prefix string) CacheOption {
	return func(o *CacheOptions) {
		o.Prefix = prefix
	}
}

func WithSkipCache() CacheOption {
	return func(o *CacheOptions) {
		o.Skip = true
	}
}

func WithCacheRefresh() CacheOption {
	return func(o *CacheOptions) {
		o.Refresh = true
	}
}

func (qb *Model) generateCacheKey(query string, args ...interface{}) string {
	data := fmt.Sprintf("%s|%v", query, args)
	hash := sha256.Sum256([]byte(data))
	return qb.getCachePrefix() + hex.EncodeToString(hash[:])
}

func (qb *Model) getCacheTTL() time.Duration {
	if qb.cacheTTL > 0 {
		return qb.cacheTTL
	}
	return defaultCacheConfig.DefaultTTL
}

func (qb *Model) getCachePrefix() string {
	if qb.cachePrefix != "" {
		return qb.cachePrefix
	}
	return defaultCacheConfig.Prefix
}

func (qb *Model) Cache(ttl time.Duration) *Model {
	qb.cacheEnabled = true
	qb.cacheTTL = ttl
	return qb
}

func (qb *Model) CacheWithPrefix(prefix string) *Model {
	qb.cacheEnabled = true
	qb.cachePrefix = prefix
	return qb
}

func (qb *Model) SkipCache() *Model {
	qb.skipCache = true
	return qb
}

func (qb *Model) ClearCache(ctx context.Context) error {
	if !qb.cacheEnabled && !defaultCacheConfig.Enabled {
		return nil
	}

	pattern := qb.getCachePrefix() + qb.table + ":"
	return gcache.Removes(ctx, []interface{}{pattern})
}

func (qb *Model) clearTableCache(ctx context.Context) error {
	if !qb.cacheEnabled && !defaultCacheConfig.Enabled {
		return nil
	}

	tablePrefix := qb.getCachePrefix() + qb.table + ":"
	return gcache.Removes(ctx, []interface{}{tablePrefix})
}

func (qb *Model) CacheFind(ctx context.Context, dest interface{}, opts ...CacheOption) *QueryResult {
	options := &CacheOptions{
		TTL: qb.getCacheTTL(),
	}
	for _, opt := range opts {
		opt(options)
	}

	if options.Skip || qb.skipCache || (!qb.cacheEnabled && !defaultCacheConfig.Enabled) {
		return qb.Find(ctx, dest)
	}

	if options.Refresh {
		return qb.refreshAndCache(ctx, dest, options.TTL)
	}

	query, args := qb.buildQuery(ctx)
	cacheKey := qb.generateCacheKey(query, args...)

	cached, err := gcache.Get(ctx, cacheKey)
	if err == nil && cached != nil && !cached.IsNil() && !cached.IsEmpty() {
		if err := cached.Scan(dest); err == nil {
			return &QueryResult{
				data:  dest,
				err:   nil,
				query: query,
				args:  args,
			}
		}
	}

	return qb.refreshAndCache(ctx, dest, options.TTL)
}

func (qb *Model) CacheSelect(ctx context.Context, dest interface{}, opts ...CacheOption) *QueryResult {
	options := &CacheOptions{
		TTL: qb.getCacheTTL(),
	}
	for _, opt := range opts {
		opt(options)
	}

	if options.Skip || qb.skipCache || (!qb.cacheEnabled && !defaultCacheConfig.Enabled) {
		return qb.Select(ctx, dest)
	}

	if options.Refresh {
		return qb.refreshAndCacheList(ctx, dest, options.TTL)
	}

	query, args := qb.buildQuery(ctx)
	cacheKey := qb.generateCacheKey(query, args...)

	cached, err := gcache.Get(ctx, cacheKey)
	if err == nil && cached != nil && !cached.IsNil() && !cached.IsEmpty() {
		if err := cached.Scan(dest); err == nil {
			return &QueryResult{
				data:  dest,
				err:   nil,
				query: query,
				args:  args,
			}
		}
	}

	return qb.refreshAndCacheList(ctx, dest, options.TTL)
}

func (qb *Model) refreshAndCache(ctx context.Context, dest interface{}, ttl time.Duration) *QueryResult {
	result := qb.Find(ctx, dest)
	if result.err != nil {
		return result
	}

	query, args := qb.buildQuery(ctx)
	cacheKey := qb.generateCacheKey(query, args...)

	_ = gcache.Set(ctx, cacheKey, dest, ttl)

	return result
}

func (qb *Model) refreshAndCacheList(ctx context.Context, dest interface{}, ttl time.Duration) *QueryResult {
	result := qb.Select(ctx, dest)
	if result.err != nil {
		return result
	}

	query, args := qb.buildQuery(ctx)
	cacheKey := qb.generateCacheKey(query, args...)

	_ = gcache.Set(ctx, cacheKey, dest, ttl)

	return result
}

func (qb *Model) CacheInsert(ctx context.Context, data ...interface{}) *QueryResult {
	result := qb.Insert(ctx, data...)
	if result.err == nil {
		_ = qb.clearTableCache(context.Background())
	}
	return result
}

func (qb *Model) CacheUpdate(ctx context.Context, data ...interface{}) *QueryResult {
	result := qb.Update(ctx, data...)
	if result.err == nil {
		_ = qb.clearTableCache(context.Background())
	}
	return result
}

func (qb *Model) CacheDelete(ctx context.Context) *QueryResult {
	result := qb.Delete(ctx)
	if result.err == nil {
		_ = qb.clearTableCache(context.Background())
	}
	return result
}

func (qb *Model) CacheSave(ctx context.Context, data ...interface{}) *QueryResult {
	result := qb.Save(ctx, data...)
	if result.err == nil {
		_ = qb.clearTableCache(context.Background())
	}
	return result
}

func (qb *Model) CachePaginate(ctx context.Context, page, pageSize int, dest interface{}, opts ...CacheOption) *PaginateResult {
	options := &CacheOptions{
		TTL: qb.getCacheTTL(),
	}
	for _, opt := range opts {
		opt(options)
	}

	if options.Skip || qb.skipCache || (!qb.cacheEnabled && !defaultCacheConfig.Enabled) {
		return qb.Paginate(ctx, page, pageSize, dest)
	}

	cacheKey := qb.generateCacheKey(fmt.Sprintf("paginate:%d:%d", page, pageSize))

	if !options.Refresh {
		cached, err := gcache.Get(ctx, cacheKey)
		if err == nil && cached != nil && !cached.IsNil() && !cached.IsEmpty() {
			var cachedResult PaginateResult
			if err := cached.Scan(&cachedResult); err == nil && cachedResult.Error == nil {
				itemsCached, itemsErr := gcache.Get(ctx, cacheKey+":items")
				if itemsErr == nil && itemsCached.Scan(dest) == nil {
					cachedResult.Items = dest
					return &cachedResult
				}
			}
		}
	}

	result := qb.Paginate(ctx, page, pageSize, dest)
	if result.Error == nil {
		_ = gcache.Set(ctx, cacheKey, result, options.TTL)
		_ = gcache.Set(ctx, cacheKey+":items", dest, options.TTL)
	}

	return result
}

func ClearAllCache(ctx context.Context) error {
	return gcache.Clear(ctx)
}

func (c *CacheConfig) Apply() {
	SetDefaultCacheConfig(c)
}
