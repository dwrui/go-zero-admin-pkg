package db

import (
	"context"
	"database/sql"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"strings"
)

// DBManager 数据库管理器
type DBManager struct {
	conn        sqlx.SqlConn
	tablePrefix string // 表前缀
}

// NewDBManager 创建数据库管理器
func NewDBManager(datasource string) *DBManager {
	return &DBManager{
		conn:        sqlx.NewSqlConn("mysql", datasource),
		tablePrefix: "", // 默认无前缀
	}
}

// SetTablePrefix 设置表前缀
func (db *DBManager) SetTablePrefix(prefix string) *DBManager {
	db.tablePrefix = prefix
	return db
}

// GetTablePrefix 获取表前缀
func (db *DBManager) GetTablePrefix() string {
	return db.tablePrefix
}

// formatTableName 格式化表名（自动添加前缀）
func (db *DBManager) formatTableName(table string) string {
	// 如果表名已经包含前缀，或者前缀为空，直接返回
	if db.tablePrefix == "" || strings.HasPrefix(table, db.tablePrefix) {
		return table
	}
	// 添加前缀
	return db.tablePrefix + table
}

// Model 创建链式查询构建器
func (db *DBManager) Model(table string) *Model {
	return &Model{
		db:       db,
		table:    db.formatTableName(table), // 格式化表名
		fields:   []string{"*"},
		where:    make([]whereClause, 0),
		joins:    make([]joinClause, 0),
		groupBy:  make([]string, 0),
		having:   make([]whereClause, 0),
		orderBy:  make([]orderClause, 0),
		page:     1,
		pageSize: 10,
	}
}

// Table 创建链式查询构建器（别名）
func (db *DBManager) Table(table string) *Model {
	return db.Model(table)
}

// Trans 执行事务
func (db *DBManager) Trans(ctx context.Context, fn func(context context.Context, session sqlx.Session) error) error {
	return db.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		return fn(ctx, session)
	})
}

// Exec 执行SQL语句
func (db *DBManager) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return db.conn.ExecCtx(ctx, query, args...)
}

// Query 查询多条记录
func (db *DBManager) Query(ctx context.Context, v interface{}, query string, args ...interface{}) error {
	return db.conn.QueryRowsCtx(ctx, v, query, args...)
}

// QueryRow 查询单条记录
func (db *DBManager) QueryRow(ctx context.Context, v interface{}, query string, args ...interface{}) error {
	return db.conn.QueryRowCtx(ctx, v, query, args...)
}
