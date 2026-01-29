package db

import (
	"fmt"
)

// DBConfig 数据库配置
type DBConfig struct {
	Host        string `json:"host" yaml:"host"`
	Port        int    `json:"port" yaml:"port"`
	Database    string `json:"database" yaml:"database"`
	Username    string `json:"username" yaml:"username"`
	Password    string `json:"password" yaml:"password"`
	Charset     string `json:"charset" yaml:"charset"`
	TablePrefix string `json:"table_prefix" yaml:"table_prefix"` // 表前缀配置
}

// GetDataSource 获取数据源字符串
func (c *DBConfig) GetDataSource() string {
	charset := c.Charset
	if charset == "" {
		charset = "utf8mb4"
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=true&loc=Local",
		c.Username, c.Password, c.Host, c.Port, c.Database, charset)
}

// GetTablePrefix 获取表前缀
func (c *DBConfig) GetTablePrefix() string {
	return c.TablePrefix
}

// NewDBManagerFromConfig 根据配置创建数据库管理器
func NewDBManagerFromConfig(config DBConfig) *DBManager {
	return NewDBManager(config.GetDataSource()).SetTablePrefix(config.GetTablePrefix())
}
