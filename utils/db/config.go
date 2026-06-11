package db

import (
	"fmt"
	"sync"
	"time"
)

// DBPoolConfig 数据库连接池配置
type DBPoolConfig struct {
	MaxOpenConns    int           `json:"max_open_conns" yaml:"max_open_conns"`         // 最大打开连接数，默认20
	MaxIdleConns    int           `json:"max_idle_conns" yaml:"max_idle_conns"`         // 最大空闲连接数，默认10
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime" yaml:"conn_max_lifetime"`   // 连接最大生命周期，默认30分钟
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time" yaml:"conn_max_idle_time"` // 空闲连接超时，默认5分钟
}

// DBConfig 数据库配置
type DBConfig struct {
	Host        string       `json:"host" yaml:"host"`
	Port        int          `json:"port" yaml:"port"`
	Database    string       `json:"database" yaml:"database"`
	Username    string       `json:"username" yaml:"username"`
	Password    string       `json:"password" yaml:"password"`
	Charset     string       `json:"charset" yaml:"charset"`
	TablePrefix string       `json:"table_prefix" yaml:"table_prefix"` // 表前缀配置
	Pool        DBPoolConfig `json:"pool" yaml:"pool"`                 // 连接池配置
}

// 生产环境默认连接池（13个微服务 × 20 ≈ 260，需小于 MySQL max_connections）
var (
	defaultPoolConfig = DBPoolConfig{
		MaxOpenConns:    20,
		MaxIdleConns:    10,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
	}
	poolConfigMutex sync.RWMutex
)

// SetDefaultPoolConfig 设置全局默认连接池配置
// 可以在程序启动时调用，所有数据库连接将使用此配置
func SetDefaultPoolConfig(config DBPoolConfig) {
	poolConfigMutex.Lock()
	defer poolConfigMutex.Unlock()

	if config.MaxOpenConns > 0 {
		defaultPoolConfig.MaxOpenConns = config.MaxOpenConns
	}
	if config.MaxIdleConns > 0 {
		defaultPoolConfig.MaxIdleConns = config.MaxIdleConns
	}
	if config.ConnMaxLifetime > 0 {
		defaultPoolConfig.ConnMaxLifetime = config.ConnMaxLifetime
	}
	if config.ConnMaxIdleTime > 0 {
		defaultPoolConfig.ConnMaxIdleTime = config.ConnMaxIdleTime
	}
	normalizePoolConfig(&defaultPoolConfig)
}

// GetDefaultPoolConfig 获取全局默认连接池配置
func GetDefaultPoolConfig() DBPoolConfig {
	poolConfigMutex.RLock()
	defer poolConfigMutex.RUnlock()
	return defaultPoolConfig
}

// GetDataSource 获取数据源字符串
func (c *DBConfig) GetDataSource() string {
	charset := c.Charset
	if charset == "" {
		charset = "utf8mb4"
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=true&loc=Local&timeout=10s&readTimeout=30s&writeTimeout=30s",
		c.Username, c.Password, c.Host, c.Port, c.Database, charset)
}

// GetTablePrefix 获取表前缀
func (c *DBConfig) GetTablePrefix() string {
	return c.TablePrefix
}

// normalizePoolConfig 校正连接池参数，避免 Idle 大于 Open
func normalizePoolConfig(pool *DBPoolConfig) {
	if pool.MaxIdleConns > pool.MaxOpenConns {
		pool.MaxIdleConns = pool.MaxOpenConns
	}
}

// GetPoolConfig 获取连接池配置（带默认值）
// 如果未配置，则使用全局默认配置
func (c *DBConfig) GetPoolConfig() DBPoolConfig {
	pool := c.Pool
	defaultConfig := GetDefaultPoolConfig()

	// 如果没有配置任何连接池参数，使用全局默认配置
	if pool.MaxOpenConns <= 0 && pool.MaxIdleConns <= 0 &&
		pool.ConnMaxLifetime <= 0 && pool.ConnMaxIdleTime <= 0 {
		return defaultConfig
	}

	// 部分配置使用默认值填充
	if pool.MaxOpenConns <= 0 {
		pool.MaxOpenConns = defaultConfig.MaxOpenConns
	}
	if pool.MaxIdleConns <= 0 {
		pool.MaxIdleConns = defaultConfig.MaxIdleConns
	}
	if pool.ConnMaxLifetime <= 0 {
		pool.ConnMaxLifetime = defaultConfig.ConnMaxLifetime
	}
	if pool.ConnMaxIdleTime <= 0 {
		pool.ConnMaxIdleTime = defaultConfig.ConnMaxIdleTime
	}

	normalizePoolConfig(&pool)
	return pool
}

// NewDBManagerFromConfig 根据配置创建数据库管理器
func NewDBManagerFromConfig(config DBConfig) (*DBManager, error) {
	manager := NewDBManager(config.GetDataSource()).SetTablePrefix(config.GetTablePrefix())
	if err := manager.SetPoolConfig(config.GetPoolConfig()); err != nil {
		return nil, err
	}
	return manager, nil
}
