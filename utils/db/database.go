package db

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// DBManager 数据库管理器
type DBManager struct {
	conn        sqlx.SqlConn
	tablePrefix string
	poolConfig  DBPoolConfig
}

// Session 事务会话
type Session struct {
	session     sqlx.Session
	db          *DBManager
	tablePrefix string
}

// NewDBManager 创建数据库管理器
func NewDBManager(datasource string) *DBManager {
	return &DBManager{
		conn:        sqlx.NewSqlConn("mysql", datasource),
		tablePrefix: "",
		poolConfig: DBPoolConfig{
			MaxOpenConns:    50,
			MaxIdleConns:    25,
			ConnMaxLifetime: 30 * time.Minute,
			ConnMaxIdleTime: 10 * time.Minute,
		},
	}
}

// SetPoolConfig 设置连接池配置
func (db *DBManager) SetPoolConfig(config DBPoolConfig) *DBManager {
	db.poolConfig = config
	if rawDB, err := db.conn.RawDB(); err == nil {
		rawDB.SetMaxOpenConns(config.MaxOpenConns)
		rawDB.SetMaxIdleConns(config.MaxIdleConns)
		rawDB.SetConnMaxLifetime(config.ConnMaxLifetime)
		rawDB.SetConnMaxIdleTime(config.ConnMaxIdleTime)
	}
	return db
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

// formatTableName 格式化表名
func (db *DBManager) formatTableName(table string) string {
	if db.tablePrefix == "" || strings.HasPrefix(table, db.tablePrefix) {
		return table
	}
	return db.tablePrefix + table
}

// Model 创建链式查询构建器
func (db *DBManager) Model(table string) *Model {
	return &Model{
		db:       db,
		table:    db.formatTableName(table),
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
func (db *DBManager) Trans(ctx context.Context, fn func(context.Context, *Session) error) error {
	return db.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		sess := &Session{
			session:     session,
			db:          db,
			tablePrefix: db.tablePrefix,
		}
		return fn(ctx, sess)
	})
}

// Model 创建链式查询构建器（事务会话）
func (s *Session) Model(table string) *Model {
	var tableName string
	if s.tablePrefix == "" || strings.HasPrefix(table, s.tablePrefix) {
		tableName = table
	} else {
		tableName = s.tablePrefix + table
	}
	return &Model{
		db:       s.db,
		session:  s.session,
		table:    tableName,
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
func (s *Session) Table(table string) *Model {
	return s.Model(table)
}

// Exec 执行SQL语句
func (db *DBManager) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return db.conn.ExecCtx(ctx, query, args...)
}

// Query 查询多条记录
func (db *DBManager) Query(ctx context.Context, v interface{}, query string, args ...interface{}) error {
	return db.conn.QueryRowsPartialCtx(ctx, v, query, args...)
}

// QueryRows 查询多条记录
func (db *DBManager) QueryRows(ctx context.Context, v interface{}, query string, args ...interface{}) error {
	return db.conn.QueryRowsPartialCtx(ctx, v, query, args...)
}

// QueryRow 查询单条记录
func (db *DBManager) QueryRow(ctx context.Context, v interface{}, query string, args ...interface{}) error {
	return db.conn.QueryRowPartialCtx(ctx, v, query, args...)
}

// QueryRaw 执行原始查询返回*sql.Rows
func (db *DBManager) QueryRaw(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	rawDB, err := db.conn.RawDB()
	if err != nil {
		return nil, err
	}
	return rawDB.QueryContext(ctx, query, args...)
}
