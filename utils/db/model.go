package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// Model 链式查询构建器
type Model struct {
	db       *DBManager
	table    string
	alias    string
	joins    []joinClause
	where    []whereClause
	groupBy  []string
	having   []whereClause
	orderBy  []orderClause
	limit    int
	offset   int
	page     int
	pageSize int
	lockMode string
	distinct bool
	fields   []string
	sqlFetch bool // 是否只输出SQL不执行查询
}

// joinClause 关联查询结构
type joinClause struct {
	joinType string // LEFT, RIGHT, INNER
	table    string
	alias    string
	on       string
	args     []interface{}
}

// whereClause 条件结构
type whereClause struct {
	operator string // AND, OR
	field    string
	cond     string
	args     []interface{}
}

// orderClause 排序结构
type orderClause struct {
	field string
	dir   string // ASC, DESC
}

// QueryResult 查询结果包装器
type QueryResult struct {
	data  interface{}
	err   error
	query string
	args  []interface{}
}

// IsEmpty 判断查询结果是否为空
func (r *QueryResult) IsEmpty() bool {
	if r.err != nil {
		return true
	}

	switch v := r.data.(type) {
	case []interface{}:
		return len(v) == 0
	case nil:
		return true
	default:
		// 使用反射检查是否为空切片
		return r.isSliceEmpty(v)
	}
}

// IsNotEmpty 判断查询结果是否不为空
func (r *QueryResult) IsNotEmpty() bool {
	return !r.IsEmpty()
}

// GetError 获取错误信息
func (r *QueryResult) GetError() error {
	return r.err
}

// GetSQL 获取执行的SQL语句（调试用）
func (r *QueryResult) GetSQL() string {
	return r.query
}

// GetArgs 获取SQL参数（调试用）
func (r *QueryResult) GetArgs() []interface{} {
	return r.args
}

// SQLFetch 设置是否只输出SQL不执行查询
func (qb *Model) SQLFetch(fetch bool) *Model {
	qb.sqlFetch = fetch
	return qb
}

// Alias 设置表别名
func (qb *Model) Alias(alias string) *Model {
	qb.alias = alias
	return qb
}

// Fields 设置查询字段
func (qb *Model) Fields(fields ...string) *Model {
	if len(fields) == 0 {
		return qb
	}

	// 如果只传了一个参数且包含逗号，则按逗号分割
	if len(fields) == 1 && strings.Contains(fields[0], ",") {
		// 按逗号分割并去除空格
		fieldList := strings.Split(fields[0], ",")
		for i, field := range fieldList {
			fieldList[i] = strings.TrimSpace(field)
		}
		qb.fields = fieldList
	} else {
		// 多个参数形式，直接使用
		qb.fields = fields
	}
	return qb
}

// Distinct 设置DISTINCT
func (qb *Model) Distinct() *Model {
	qb.distinct = true
	return qb
}

// LeftJoin 左关联
func (qb *Model) LeftJoin(table, alias, on string, args ...interface{}) *Model {
	qb.joins = append(qb.joins, joinClause{
		joinType: "LEFT",
		table:    qb.db.formatTableName(table), // 格式化关联表名
		alias:    alias,
		on:       on,
		args:     args,
	})
	return qb
}

// RightJoin 右关联
func (qb *Model) RightJoin(table, alias, on string, args ...interface{}) *Model {
	qb.joins = append(qb.joins, joinClause{
		joinType: "RIGHT",
		table:    qb.db.formatTableName(table), // 格式化关联表名
		alias:    alias,
		on:       on,
		args:     args,
	})
	return qb
}

// Join 内关联
func (qb *Model) Join(table, alias, on string, args ...interface{}) *Model {
	qb.joins = append(qb.joins, joinClause{
		joinType: "INNER",
		table:    qb.db.formatTableName(table), // 格式化关联表名
		alias:    alias,
		on:       on,
		args:     args,
	})
	return qb
}

// Where 设置条件 (支持map和map切片)
func (qb *Model) Where(conditions interface{}, args ...interface{}) *Model {
	switch cond := conditions.(type) {
	case map[string]interface{}:
		// 处理map类型条件
		for field, value := range cond {
			qb.where = append(qb.where, whereClause{
				operator: "AND",
				field:    field,
				cond:     "= ?",
				args:     []interface{}{value},
			})
		}
	case []map[string]interface{}:
		// 处理map切片类型条件
		for i, condition := range cond {
			for field, value := range condition {
				operator := "AND"
				if i == 0 && len(qb.where) == 0 {
					operator = "" // 第一个条件不加AND
				}
				qb.where = append(qb.where, whereClause{
					operator: operator,
					field:    field,
					cond:     "= ?",
					args:     []interface{}{value},
				})
			}
		}
	case string:
		// 处理字符串条件
		operator := "AND"
		if len(qb.where) == 0 {
			operator = ""
		}
		qb.where = append(qb.where, whereClause{
			operator: operator,
			field:    cond,
			cond:     cond,
			args:     args,
		})
	}
	return qb
}

// WhereOr 设置OR条件
func (qb *Model) WhereOr(field string, args ...interface{}) *Model {
	operator := "OR"
	if len(qb.where) == 0 {
		operator = ""
	}
	qb.where = append(qb.where, whereClause{
		operator: operator,
		field:    field,
		cond:     fmt.Sprintf("%s = ?", field),
		args:     args,
	})
	return qb
}

// WhereIn 设置IN条件
func (qb *Model) WhereIn(field string, values []interface{}) *Model {
	if len(values) == 0 {
		return qb
	}

	placeholders := make([]string, len(values))
	for i := range values {
		placeholders[i] = "?"
	}

	operator := "AND"
	if len(qb.where) == 0 {
		operator = ""
	}

	qb.where = append(qb.where, whereClause{
		operator: operator,
		field:    field,
		cond:     fmt.Sprintf("IN (%s)", strings.Join(placeholders, ",")),
		args:     values,
	})
	return qb
}

// WhereNotIn 设置NOT IN条件
func (qb *Model) WhereNotIn(field string, values []interface{}) *Model {
	if len(values) == 0 {
		return qb
	}

	placeholders := make([]string, len(values))
	for i := range values {
		placeholders[i] = "?"
	}

	operator := "AND"
	if len(qb.where) == 0 {
		operator = ""
	}

	qb.where = append(qb.where, whereClause{
		operator: operator,
		field:    field,
		cond:     fmt.Sprintf("NOT IN (%s)", strings.Join(placeholders, ",")),
		args:     values,
	})
	return qb
}

// WhereBetween 设置BETWEEN条件
func (qb *Model) WhereBetween(field string, start, end interface{}) *Model {
	operator := "AND"
	if len(qb.where) == 0 {
		operator = ""
	}

	qb.where = append(qb.where, whereClause{
		operator: operator,
		field:    field,
		cond:     "BETWEEN ? AND ?",
		args:     []interface{}{start, end},
	})
	return qb
}

// WhereNull 设置IS NULL条件
func (qb *Model) WhereNull(field string) *Model {
	operator := "AND"
	if len(qb.where) == 0 {
		operator = ""
	}

	qb.where = append(qb.where, whereClause{
		operator: operator,
		field:    field,
		cond:     "IS NULL",
		args:     []interface{}{},
	})
	return qb
}

// WhereNotNull 设置IS NOT NULL条件
func (qb *Model) WhereNotNull(field string) *Model {
	operator := "AND"
	if len(qb.where) == 0 {
		operator = ""
	}

	qb.where = append(qb.where, whereClause{
		operator: operator,
		field:    field,
		cond:     "IS NOT NULL",
		args:     []interface{}{},
	})
	return qb
}

// GroupBy 设置分组
func (qb *Model) Group(fields ...string) *Model {
	qb.groupBy = append(qb.groupBy, fields...)
	return qb
}

// Having 设置HAVING条件
func (qb *Model) Having(condition string, args ...interface{}) *Model {
	qb.having = append(qb.having, whereClause{
		operator: "AND",
		field:    condition,
		cond:     condition,
		args:     args,
	})
	return qb
}

// Order 设置排序
func (qb *Model) Order(field, direction string) *Model {
	qb.orderBy = append(qb.orderBy, orderClause{
		field: field,
		dir:   strings.ToUpper(direction),
	})
	return qb
}

// OrderBy 设置排序（升序）
func (qb *Model) OrderBy(field string) *Model {
	return qb.Order(field, "ASC")
}

// OrderByDesc 设置排序（降序）
func (qb *Model) OrderByDesc(field string) *Model {
	return qb.Order(field, "DESC")
}

// Limit 设置限制条数
func (qb *Model) Limit(limit int) *Model {
	qb.limit = limit
	return qb
}

// Offset 设置偏移量
func (qb *Model) Offset(offset int) *Model {
	qb.offset = offset
	return qb
}

// Page 设置分页
func (qb *Model) Page(page, pageSize int) *Model {
	qb.page = page
	qb.pageSize = pageSize
	if page > 0 && pageSize > 0 {
		qb.offset = (page - 1) * pageSize
		qb.limit = pageSize
	}
	return qb
}

// ForUpdate 设置FOR UPDATE锁
func (qb *Model) ForUpdate() *Model {
	qb.lockMode = "FOR UPDATE"
	return qb
}

// LockInShareMode 设置LOCK IN SHARE MODE锁
func (qb *Model) LockInShareMode() *Model {
	qb.lockMode = "LOCK IN SHARE MODE"
	return qb
}

// Find 执行查询
func (qb *Model) Find(ctx context.Context, dest interface{}) *QueryResult {
	query, args := qb.buildQuery()

	// 如果设置了SQLFetch，只输出SQL不执行查询
	if qb.sqlFetch {
		fmt.Printf("SQL: %s\nArgs: %v\n", query, args)
		return &QueryResult{
			data:  dest,
			err:   nil,
			query: query,
			args:  args,
		}
	}

	err := qb.db.Query(ctx, dest, query, args...)
	return &QueryResult{
		data:  dest,
		err:   err,
		query: query,
		args:  args,
	}
}

// FindOne 执行单条查询
func (qb *Model) FindOne(ctx context.Context, dest interface{}) *QueryResult {
	qb.Limit(1)
	query, args := qb.buildQuery()

	// 如果设置了SQLFetch，只输出SQL不执行查询
	if qb.sqlFetch {
		fmt.Printf("SQL: %s\nArgs: %v\n", query, args)
		return &QueryResult{
			data:  dest,
			err:   nil,
			query: query,
			args:  args,
		}
	}

	err := qb.db.QueryRow(ctx, dest, query, args...)
	return &QueryResult{
		data:  dest,
		err:   err,
		query: query,
		args:  args,
	}
}

// Count 统计数量
func (qb *Model) Count(ctx context.Context) *QueryResult {
	qb.fields = []string{"COUNT(*)"}
	query, args := qb.buildQuery()

	// 如果设置了SQLFetch，只输出SQL不执行查询
	if qb.sqlFetch {
		fmt.Printf("SQL: %s\nArgs: %v\n", query, args)
		return &QueryResult{
			data:  int64(0),
			err:   nil,
			query: query,
			args:  args,
		}
	}

	var count int64
	err := qb.db.QueryRow(ctx, &count, query, args...)
	return &QueryResult{
		data:  count,
		err:   err,
		query: query,
		args:  args,
	}
}

// Exists 检查是否存在
func (qb *Model) Exists(ctx context.Context) *QueryResult {
	result := qb.Count(ctx)
	if result.err != nil {
		return result
	}

	// 如果设置了SQLFetch，直接返回结果
	if qb.sqlFetch {
		return result
	}

	count, ok := result.data.(int64)
	if !ok {
		return &QueryResult{
			data:  false,
			err:   fmt.Errorf("count result type error"),
			query: result.query,
			args:  result.args,
		}
	}

	return &QueryResult{
		data:  count > 0,
		err:   nil,
		query: result.query,
		args:  result.args,
	}
}

// Sum 查询指定字段的合计数
func (qb *Model) Sum(ctx context.Context, field string) *QueryResult {
	qb.fields = []string{fmt.Sprintf("SUM(%s)", field)}
	query, args := qb.buildQuery()

	// 如果设置了SQLFetch，只输出SQL不执行查询
	if qb.sqlFetch {
		fmt.Printf("SQL: %s\nArgs: %v\n", query, args)
		return &QueryResult{
			data:  float64(0),
			err:   nil,
			query: query,
			args:  args,
		}
	}

	var sum sql.NullFloat64
	err := qb.db.QueryRow(ctx, &sum, query, args...)

	var result float64
	if err == nil && sum.Valid {
		result = sum.Float64
	}

	return &QueryResult{
		data:  result,
		err:   err,
		query: query,
		args:  args,
	}
}

// Value 获取指定字段的值（单条记录）
func (qb *Model) Value(ctx context.Context, field string) *QueryResult {
	qb.fields = []string{field}
	qb.Limit(1)
	query, args := qb.buildQuery()

	// 如果设置了SQLFetch，只输出SQL不执行查询
	if qb.sqlFetch {
		fmt.Printf("SQL: %s\nArgs: %v\n", query, args)
		return &QueryResult{
			data:  nil,
			err:   nil,
			query: query,
			args:  args,
		}
	}

	var value interface{}
	err := qb.db.QueryRow(ctx, &value, query, args...)
	return &QueryResult{
		data:  value,
		err:   err,
		query: query,
		args:  args,
	}
}

// Column 获取单一字段的所有值
func (qb *Model) Column(ctx context.Context, field string) *QueryResult {
	qb.fields = []string{field}
	query, args := qb.buildQuery()

	// 如果设置了SQLFetch，只输出SQL不执行查询
	if qb.sqlFetch {
		fmt.Printf("SQL: %s\nArgs: %v\n", query, args)
		return &QueryResult{
			data:  []interface{}{},
			err:   nil,
			query: query,
			args:  args,
		}
	}

	var results []interface{}
	err := qb.db.Query(ctx, &results, query, args...)
	return &QueryResult{
		data:  results,
		err:   err,
		query: query,
		args:  args,
	}
}

// buildQuery 构建SQL查询
func (qb *Model) buildQuery() (string, []interface{}) {
	var sql strings.Builder
	var args []interface{}

	// SELECT 子句
	sql.WriteString("SELECT ")
	if qb.distinct {
		sql.WriteString("DISTINCT ")
	}
	sql.WriteString(strings.Join(qb.fields, ", "))

	// FROM 子句
	sql.WriteString(" FROM ")
	sql.WriteString(qb.table)
	if qb.alias != "" {
		sql.WriteString(" AS ")
		sql.WriteString(qb.alias)
	}

	// JOIN 子句
	for _, join := range qb.joins {
		sql.WriteString(" ")
		sql.WriteString(join.joinType)
		sql.WriteString(" JOIN ")
		sql.WriteString(join.table)
		if join.alias != "" {
			sql.WriteString(" AS ")
			sql.WriteString(join.alias)
		}
		sql.WriteString(" ON ")
		sql.WriteString(join.on)
		args = append(args, join.args...)
	}

	// WHERE 子句
	if len(qb.where) > 0 {
		sql.WriteString(" WHERE ")
		for i, where := range qb.where {
			if i > 0 || where.operator != "" {
				sql.WriteString(" ")
				sql.WriteString(where.operator)
				sql.WriteString(" ")
			}
			sql.WriteString(where.field)
			sql.WriteString(" ")
			sql.WriteString(where.cond)
			args = append(args, where.args...)
		}
	}

	// GROUP BY 子句
	if len(qb.groupBy) > 0 {
		sql.WriteString(" GROUP BY ")
		sql.WriteString(strings.Join(qb.groupBy, ", "))
	}

	// HAVING 子句
	if len(qb.having) > 0 {
		sql.WriteString(" HAVING ")
		for i, having := range qb.having {
			if i > 0 {
				sql.WriteString(" AND ")
			}
			sql.WriteString(having.cond)
			args = append(args, having.args...)
		}
	}

	// ORDER BY 子句
	if len(qb.orderBy) > 0 {
		sql.WriteString(" ORDER BY ")
		for i, order := range qb.orderBy {
			if i > 0 {
				sql.WriteString(", ")
			}
			sql.WriteString(order.field)
			sql.WriteString(" ")
			sql.WriteString(order.dir)
		}
	}

	// LIMIT 子句
	if qb.limit > 0 {
		sql.WriteString(" LIMIT ")
		sql.WriteString(fmt.Sprintf("%d", qb.limit))
	}

	// OFFSET 子句
	if qb.offset > 0 {
		sql.WriteString(" OFFSET ")
		sql.WriteString(fmt.Sprintf("%d", qb.offset))
	}

	// 锁
	if qb.lockMode != "" {
		sql.WriteString(" ")
		sql.WriteString(qb.lockMode)
	}

	return sql.String(), args
}

// isSliceEmpty 辅助方法：判断切片是否为空
func (r *QueryResult) isSliceEmpty(v interface{}) bool {
	// 这里可以添加更多的反射逻辑来判断不同类型的空值
	// 简化实现，主要处理常见的切片类型
	return false
}
