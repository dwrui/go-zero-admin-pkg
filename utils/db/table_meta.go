package db

import "context"

// tableMeta 单表元数据，由 getTableMeta 从 information_schema 加载并缓存在 DBManager 中。
type tableMeta struct {
	hasDeleteTime bool   // 是否存在 delete_time 列（决定是否启用软删除）
	primaryKey    string // 主键列名，默认 "id"
}

// getTableMeta 获取表元数据（双检锁 + 进程内缓存，避免重复查 information_schema）
func (db *DBManager) getTableMeta(ctx context.Context, table string) tableMeta {
	db.tableMetaMu.RLock()
	if meta, ok := db.tableMeta[table]; ok {
		db.tableMetaMu.RUnlock()
		return meta
	}
	db.tableMetaMu.RUnlock()

	db.tableMetaMu.Lock()
	defer db.tableMetaMu.Unlock()
	if meta, ok := db.tableMeta[table]; ok {
		return meta
	}

	meta := tableMeta{primaryKey: "id"}

	deleteSQL := `SELECT COUNT(*) FROM information_schema.COLUMNS 
				 WHERE TABLE_SCHEMA = DATABASE() 
				 AND TABLE_NAME = ? 
				 AND COLUMN_NAME = 'delete_time'`
	var deleteCount int
	if err := db.QueryRow(ctx, &deleteCount, deleteSQL, table); err == nil {
		meta.hasDeleteTime = deleteCount > 0
	}

	pkSQL := `SELECT COLUMN_NAME FROM information_schema.KEY_COLUMN_USAGE 
			  WHERE TABLE_SCHEMA = DATABASE() 
			  AND TABLE_NAME = ? 
			  AND CONSTRAINT_NAME = 'PRIMARY' 
			  LIMIT 1`
	var pk string
	if err := db.QueryRow(ctx, &pk, pkSQL, table); err == nil && pk != "" {
		meta.primaryKey = pk
	}

	if db.tableMeta == nil {
		db.tableMeta = make(map[string]tableMeta)
	}
	db.tableMeta[table] = meta
	return meta
}

// tableHasDeleteTime 表是否含 delete_time 列
func (db *DBManager) tableHasDeleteTime(ctx context.Context, table string) bool {
	return db.getTableMeta(ctx, table).hasDeleteTime
}

// tablePrimaryKey 表主键列名
func (db *DBManager) tablePrimaryKey(ctx context.Context, table string) string {
	return db.getTableMeta(ctx, table).primaryKey
}
