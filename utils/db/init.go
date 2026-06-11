package db

import (
	"context"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

var (
	dbManager *DBManager
	once      sync.Once
)

// InitDB 初始化数据库（启动时 Ping 失败则终止服务）
func InitDB(config DBConfig) {
	once.Do(func() {
		manager, err := NewDBManagerFromConfig(config)
		logx.Must(err)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		logx.Must(manager.Ping(ctx))

		dbManager = manager
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
