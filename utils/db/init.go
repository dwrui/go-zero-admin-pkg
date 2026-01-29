package db

import (
	"sync"

	"github.com/zeromicro/go-zero/core/logx"
)

var (
	dbManager *DBManager
	once      sync.Once
)

// InitDB 初始化数据库
func InitDB(config DBConfig) {
	once.Do(func() {
		dbManager = NewDBManagerFromConfig(config)
		logx.Info("数据库连接初始化成功")
	})
}

// GetDB 获取数据库管理器
func GetDB() *DBManager {
	if dbManager == nil {
		logx.Error("数据库未初始化，请先调用InitDB")
		return nil
	}
	return dbManager
}
