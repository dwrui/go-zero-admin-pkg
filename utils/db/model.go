// Package db 提供链式 SQL 构建器 Model，支持查询、写入、分页、软删除与事务。
//
// 基本用法：
//
//	db.Model("ga_xxx").Where("id", 1).Find(ctx, &row)
//
// 事务用法（事务内须使用 s.Model，每次调用返回独立 builder，共享同一 session）：
//
//	db.Trans(ctx, func(ctx context.Context, s *db.Session) error {
//	    if err := s.Model("ga_xxx").Where("id", id).Find(ctx, &row).GetError(); err != nil {
//	        return err
//	    }
//	    return s.Model("ga_xxx").Where("id", id).Update(ctx, data).GetError()
//	})
//
// 软删除约定（表含 delete_time 字段时自动生效）：
//   - 未删除：delete_time = 0
//   - 已删除：delete_time = 毫秒时间戳
//   - Delete 写时间戳；Restore 置 0；ForceDelete 物理删除
package db

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dwrui/go-zero-admin-pkg/utils/ga"
	"github.com/dwrui/go-zero-admin-pkg/utils/tools/gconv"
	"github.com/dwrui/go-zero-admin-pkg/utils/tools/gmap"
	"github.com/dwrui/go-zero-admin-pkg/utils/tools/gvar"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// Model 链式 SQL 构建器。通过 DBManager.Model 或 Session.Model 创建；
// 同一 Session 下多次 Model() 彼此独立，但共享事务连接。
type Model struct {
	db      *DBManager
	session sqlx.Session // 非 nil 时读写均走事务 session
	table   string
	alias   string
	joins   []joinClause
	where   []whereClause
	groupBy []string
	having  []whereClause
	orderBy []orderClause
	limit   int
	offset  int
	page    int
	pageSize int
	lockMode string
	distinct bool
	fields   []string
	sqlFetch bool       // true 时仅打印 SQL，不执行（开发调试）
	data     interface{} // Insert/Update 等待写入数据
	withTrashed bool     // 包含已软删记录（不过滤 delete_time）
	onlyTrashed bool     // 仅操作已软删记录（delete_time > 0）
	updateData  map[string]interface{}
	incData     map[string]interface{}
	decData     map[string]interface{}
	hasDeleteTime *bool // 测试覆盖用；生产环境走 DBManager 表元数据缓存
	primaryKey    string // SetPrimaryKey 手动指定，否则从表元数据读取
	buildErr      error  // Where 等链式构建阶段的首个错误
}

// execCtx 执行SQL（支持事务）
func (qb *Model) execCtx(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if qb.session != nil {
		return qb.session.ExecCtx(ctx, query, args...)
	}
	return qb.db.Exec(ctx, query, args...)
}

// queryRowCtx 查询单行（支持事务）
func (qb *Model) queryRowCtx(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	if qb.session != nil {
		return qb.session.QueryRowPartialCtx(ctx, dest, query, args...)
	}
	return qb.db.QueryRow(ctx, dest, query, args...)
}

// queryRowsCtx 查询多行（支持事务）
func (qb *Model) queryRowsCtx(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	if qb.session != nil {
		return qb.session.QueryRowsPartialCtx(ctx, dest, query, args...)
	}
	return qb.db.Query(ctx, dest, query, args...)
}

// convertToMap 将任意类型转换为map[string]interface{}
func (qb *Model) convertToMap(data interface{}) (map[string]interface{}, error) {
	if data == nil {
		return nil, fmt.Errorf("data cannot be nil")
	}

	switch v := data.(type) {
	case map[string]interface{}:
		return v, nil
	case map[string]string:
		result := make(map[string]interface{})
		for k, val := range v {
			result[k] = val
		}
		return result, nil
	default:
		// 使用gconv转换结构体到map
		result := gconv.Map(data)
		if result == nil {
			return nil, fmt.Errorf("convert data to map failed: data type %T is not supported", data)
		}
		return result, nil
	}
}

// convertToMaps 将任意类型转换为[]map[string]interface{}
func (qb *Model) convertToMaps(data interface{}) ([]map[string]interface{}, error) {
	if data == nil {
		return nil, fmt.Errorf("data cannot be nil")
	}

	switch v := data.(type) {
	case []map[string]interface{}:
		return v, nil
	case []map[string]string:
		result := make([]map[string]interface{}, len(v))
		for i, m := range v {
			newMap := make(map[string]interface{})
			for k, val := range m {
				newMap[k] = val
			}
			result[i] = newMap
		}
		return result, nil
	default:
		// 使用反射处理结构体切片
		reflectValue := reflect.ValueOf(data)
		if reflectValue.Kind() != reflect.Slice {
			return nil, fmt.Errorf("data must be slice type")
		}

		result := make([]map[string]interface{}, reflectValue.Len())
		for i := 0; i < reflectValue.Len(); i++ {
			item := reflectValue.Index(i).Interface()
			itemMap, err := qb.convertToMap(item)
			if err != nil {
				return nil, fmt.Errorf("convert item %d to map failed: %v", i, err)
			}
			result[i] = itemMap
		}
		return result, nil
	}
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

// QueryResult 终结型查询/写入结果。Find/Select/Count/Insert/Update/Delete 等返回此类型，
// 不可继续链式调用；通过 GetError、GetData、GetLastId 等读取结果。
type QueryResult struct {
	data   interface{}
	err    error
	query  string
	args   []interface{}
	lastId string
}

// PaginateResult 分页查询结果
type PaginateResult struct {
	Items interface{}
	Page  int
	Size  int
	Total int64
	Error error
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
	case string:
		return v == ""
	case int, int8, int16, int32, int64:
		return v == 0
	case uint, uint8, uint16, uint32, uint64:
		return v == 0
	case float32, float64:
		return v == 0
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

// GetData 获取查询结果数据
func (r *QueryResult) GetData() interface{} {
	return r.data
}

// GetLastId 获取最后一次 Insert 的自增主键（Update 不填充此字段）
func (r *QueryResult) GetLastId() string {
	return r.lastId
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

// setBuildErr 记录链式构建错误（仅保留首个错误）
func (qb *Model) setBuildErr(err error) *Model {
	if err != nil && qb.buildErr == nil {
		qb.buildErr = err
	}
	return qb
}

func (qb *Model) whereOperator() string {
	if len(qb.where) == 0 {
		return ""
	}
	return "AND"
}

// appendRawWhere 追加完整 SQL 条件片段（不含占位符）
func (qb *Model) appendRawWhere(cond string) {
	cond = strings.TrimSpace(cond)
	if cond == "" {
		return
	}
	qb.where = append(qb.where, whereClause{
		operator: qb.whereOperator(),
		field:    "",
		cond:     cond,
		args:     nil,
	})
}

// appendStringWhere 处理 Where("...", args...) 字符串条件
func (qb *Model) appendStringWhere(cond string, args ...interface{}) *Model {
	if qb.buildErr != nil {
		return qb
	}
	cond = strings.TrimSpace(cond)
	if cond == "" {
		return qb
	}
	operator := qb.whereOperator()
	lowerCond := strings.ToLower(cond)

	if strings.Contains(lowerCond, "in(?)") {
		if len(args) == 0 {
			return qb.setBuildErr(fmt.Errorf("where条件缺少参数"))
		}
		field := cond
		if idx := strings.Index(lowerCond, "in(?)"); idx != -1 {
			field = strings.TrimSpace(cond[:idx])
		}
		var inArgs []interface{}
		if len(args) == 1 {
			v := reflect.ValueOf(args[0])
			if v.Kind() == reflect.Slice {
				inArgs = make([]interface{}, v.Len())
				for j := 0; j < v.Len(); j++ {
					inArgs[j] = v.Index(j).Interface()
				}
			} else {
				inArgs = []interface{}{args[0]}
			}
		} else {
			inArgs = args
		}
		if len(inArgs) == 0 {
			qb.where = append(qb.where, whereClause{
				operator: operator,
				field:    field,
				cond:     "IN (NULL)",
				args:     []interface{}{},
			})
			return qb
		}
		placeholders := make([]string, len(inArgs))
		for j := range placeholders {
			placeholders[j] = "?"
		}
		qb.where = append(qb.where, whereClause{
			operator: operator,
			field:    "",
			cond:     fmt.Sprintf("%s IN (%s)", field, strings.Join(placeholders, ", ")),
			args:     inArgs,
		})
		return qb
	}

	if strings.Contains(cond, "?") {
		if len(args) == 0 {
			return qb.setBuildErr(fmt.Errorf("where条件缺少参数"))
		}
		qb.where = append(qb.where, whereClause{
			operator: operator,
			field:    "",
			cond:     cond,
			args:     args,
		})
		return qb
	}

	if len(args) > 0 {
		qb.where = append(qb.where, whereClause{
			operator: operator,
			field:    cond,
			cond:     "= ?",
			args:     args,
		})
		return qb
	}

	qb.appendRawWhere(cond)
	return qb
}

// Where 设置条件 (支持map和map切片)
func (qb *Model) Where(conditions interface{}, args ...interface{}) *Model {
	if qb.buildErr != nil {
		return qb
	}
	switch cond := conditions.(type) {
	case map[string]interface{}:
		// 处理map类型条件
		i := 0
		for conditionStr, value := range cond {
			operator := "AND"
			if i == 0 && len(qb.where) == 0 {
				operator = "" // 第一个条件不加AND
			}
			i++

			// 检查是否包含 IN 关键字
			lowerCond := strings.ToLower(conditionStr)
			if strings.Contains(lowerCond, "in(?)") {
				// 提取字段名
				field := conditionStr
				if idx := strings.Index(lowerCond, "in(?)"); idx != -1 {
					field = strings.TrimSpace(conditionStr[:idx])
				}

				// 处理值，转换为 []interface{}
				var inArgs []interface{}
				if value != nil {
					v := reflect.ValueOf(value)
					if v.Kind() == reflect.Slice {
						// 是切片，转换为 []interface{}
						inArgs = make([]interface{}, v.Len())
						for j := 0; j < v.Len(); j++ {
							inArgs[j] = v.Index(j).Interface()
						}
					} else {
						// 不是切片，作为单个值处理
						inArgs = []interface{}{value}
					}
				}
				// 如果没有从value中获取到参数，尝试从args中获取
				if len(inArgs) == 0 && len(args) > 0 {
					// 检查args[0]是否为切片
					v := reflect.ValueOf(args[0])
					if v.Kind() == reflect.Slice {
						// 是切片，转换为 []interface{}
						inArgs = make([]interface{}, v.Len())
						for j := 0; j < v.Len(); j++ {
							inArgs[j] = v.Index(j).Interface()
						}
					} else {
						// 不是切片，直接使用args
						inArgs = args
					}
				}
				// 只有当有值时才添加 IN 条件
				if len(inArgs) > 0 {
					placeholders := make([]string, len(inArgs))
					for j := range placeholders {
						placeholders[j] = "?"
					}
					condPattern := fmt.Sprintf("%s IN (%s)", field, strings.Join(placeholders, ", "))

					qb.where = append(qb.where, whereClause{
						operator: operator,
						field:    "",
						cond:     condPattern,
						args:     inArgs,
					})
				} else {
					// 没有值，添加一个永远为假的条件（避免查询出全部数据）
					qb.where = append(qb.where, whereClause{
						operator: operator,
						field:    field,       // 保留字段名
						cond:     "IN (NULL)", // 添加一个永远为假的条件
						args:     []interface{}{},
					})
				}
			} else if strings.Contains(conditionStr, "?") {
				// 其他带问号的条件
				qb.where = append(qb.where, whereClause{
					operator: operator,
					field:    "",
					cond:     conditionStr,
					args:     getConditionArgs(value),
				})
			} else {
				// 简单条件：默认等于
				qb.where = append(qb.where, whereClause{
					operator: operator,
					field:    conditionStr,
					cond:     "= ?",
					args:     []interface{}{value},
				})
			}
		}
	case *gmap.Map:
		// 处理gmap.Map类型条件，与map[string]interface{}处理逻辑相同
		mapData := cond.MapStrAny()
		i := 0
		for conditionStr, value := range mapData {
			operator := "AND"
			if i == 0 && len(qb.where) == 0 {
				operator = "" // 第一个条件不加AND
			}
			i++

			// 检查是否包含 IN 关键字
			lowerCond := strings.ToLower(conditionStr)
			if strings.Contains(lowerCond, "in(?)") {
				// 提取字段名
				field := conditionStr
				if idx := strings.Index(lowerCond, "in(?)"); idx != -1 {
					field = strings.TrimSpace(conditionStr[:idx])
				}

				// 处理值，转换为 []interface{}
				var inArgs []interface{}
				if value != nil {
					v := reflect.ValueOf(value)
					if v.Kind() == reflect.Slice {
						// 是切片，转换为 []interface{}
						inArgs = make([]interface{}, v.Len())
						for j := 0; j < v.Len(); j++ {
							inArgs[j] = v.Index(j).Interface()
						}
					} else {
						// 不是切片，作为单个值处理
						inArgs = []interface{}{value}
					}
				}
				// 只有当有值时才添加 IN 条件
				if len(inArgs) > 0 {
					placeholders := make([]string, len(inArgs))
					for j := range placeholders {
						placeholders[j] = "?"
					}
					condPattern := fmt.Sprintf("%s IN (%s)", field, strings.Join(placeholders, ", "))

					qb.where = append(qb.where, whereClause{
						operator: operator,
						field:    "",
						cond:     condPattern,
						args:     inArgs,
					})
				} else {
					qb.where = append(qb.where, whereClause{
						operator: operator,
						field:    field,       // 保留字段名
						cond:     "IN (NULL)", // 添加一个永远为假的条件
						args:     []interface{}{},
					})
				}
			} else if strings.Contains(conditionStr, "?") {
				qb.where = append(qb.where, whereClause{
					operator: operator,
					field:    "",
					cond:     conditionStr,
					args:     getConditionArgs(value),
				})
			} else {
				// 简单条件：默认等于
				qb.where = append(qb.where, whereClause{
					operator: operator,
					field:    conditionStr,
					cond:     "= ?",
					args:     getConditionArgs(value),
				})
			}
		}
	case []map[string]interface{}:
		// 处理map切片类型条件
		for i, condition := range cond {

			for conditionStr, value := range condition {
				operator := "AND"
				if i == 0 && len(qb.where) == 0 {
					operator = "" // 第一个条件不加AND
				}

				// 检查是否包含 IN 关键字
				lowerCond := strings.ToLower(conditionStr)
				if strings.Contains(lowerCond, "in(?)") {
					// 提取字段名
					field := conditionStr
					if idx := strings.Index(lowerCond, "in(?)"); idx != -1 {
						field = strings.TrimSpace(conditionStr[:idx])
					}

					// 处理值，转换为 []interface{}
					var inArgs []interface{}
					if value != nil {
						v := reflect.ValueOf(value)
						if v.Kind() == reflect.Slice {
							// 是切片，转换为 []interface{}
							inArgs = make([]interface{}, v.Len())
							for j := 0; j < v.Len(); j++ {
								inArgs[j] = v.Index(j).Interface()
							}
						} else {
							// 不是切片，作为单个值处理
							inArgs = []interface{}{value}
						}
					}
					// 只有当有值时才添加 IN 条件
					if len(inArgs) > 0 {
						placeholders := make([]string, len(inArgs))
						for j := range placeholders {
							placeholders[j] = "?"
						}
						condPattern := fmt.Sprintf("%s IN (%s)", field, strings.Join(placeholders, ", "))

						qb.where = append(qb.where, whereClause{
							operator: operator,
							field:    "",
							cond:     condPattern,
							args:     inArgs,
						})
					} else {
						qb.where = append(qb.where, whereClause{
							operator: operator,
							field:    field,       // 保留字段名
							cond:     "IN (NULL)", // 添加一个永远为假的条件
							args:     []interface{}{},
						})
					}
				} else if strings.Contains(conditionStr, "?") {
					// 其他带问号的条件
					qb.where = append(qb.where, whereClause{
						operator: operator,
						field:    "",
						cond:     conditionStr,
						args:     getConditionArgs(value),
					})
				} else {
					// 简单条件：默认等于
					qb.where = append(qb.where, whereClause{
						operator: operator,
						field:    conditionStr,
						cond:     "= ?",
						args:     []interface{}{value},
					})
				}
			}
		}
	case string:
		return qb.appendStringWhere(cond, args...)
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
func (qb *Model) WhereIn(field string, values interface{}) *Model {
	// 将任意类型的切片转换为[]interface{}
	interfaceValues := convertToInterfaceSlice(values)
	if len(interfaceValues) == 0 {
		return qb
	}

	placeholders := make([]string, len(interfaceValues))
	for i := range interfaceValues {
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
		args:     interfaceValues,
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

// Find 查询单条记录（自动 Limit 1）。返回 *QueryResult，不可再链式 Update。
func (qb *Model) Find(ctx context.Context, dest interface{}) *QueryResult {
	if qb.buildErr != nil {
		return &QueryResult{data: dest, err: qb.buildErr}
	}
	qb.Limit(1)
	query, args := qb.buildQuery(ctx)

	// 如果设置了SQLFetch，只输出SQL不执行查询
	if qb.sqlFetch {
		completeSQL := buildCompleteSQL(query, args)
		fmt.Printf("完整SQL: %s\n原始SQL: %s\n参数: %v\n", completeSQL, query, args)
		return &QueryResult{
			data:  dest,
			err:   nil,
			query: query,
			args:  args,
		}
	}

	err := qb.queryRowCtx(ctx, dest, query, args...)

	if err != nil && err == sql.ErrNoRows {
		err = nil
		dest = nil
	}
	return &QueryResult{
		data:  dest,
		err:   err,
		query: query,
		args:  args,
	}
}

// Select 查询多条记录
func (qb *Model) Select(ctx context.Context, dest interface{}) *QueryResult {
	if qb.buildErr != nil {
		return &QueryResult{data: dest, err: qb.buildErr}
	}
	query, args := qb.buildQuery(ctx)
	// 如果设置了SQLFetch，只输出SQL不执行查询
	if qb.sqlFetch {
		completeSQL := buildCompleteSQL(query, args)
		fmt.Printf("完整SQL: %s\n原始SQL: %s\n参数: %v\n", completeSQL, query, args)
		return &QueryResult{
			data:  dest,
			err:   nil,
			query: query,
			args:  args,
		}
	}
	err := qb.queryRowsCtx(ctx, dest, query, args...)
	if err != nil && err == sql.ErrNoRows {
		err = nil
		dest = nil
	}
	return &QueryResult{
		data:  dest,
		err:   err,
		query: query,
		args:  args,
	}
}

// Get 查询单条记录（与Find相同）
func (qb *Model) Get(ctx context.Context, dest interface{}) *QueryResult {
	return qb.Find(ctx, dest)
}

// All 查询多条记录（与Select相同）
func (qb *Model) All(ctx context.Context, dest interface{}) *QueryResult {
	return qb.Select(ctx, dest)
}

// SelectMaps 查询多条记录为 []map[string]interface{}。
// go-zero sqlx 的 QueryRowsPartialCtx 不支持 slice/map，需走原生 Rows 扫描。
func (qb *Model) SelectMaps(ctx context.Context) *QueryResult {
	if qb.buildErr != nil {
		return &QueryResult{err: qb.buildErr}
	}
	query, args := qb.buildQuery(ctx)
	if qb.sqlFetch {
		completeSQL := buildCompleteSQL(query, args)
		fmt.Printf("完整SQL: %s\n原始SQL: %s\n参数: %v\n", completeSQL, query, args)
		return &QueryResult{data: []map[string]interface{}{}, err: nil, query: query, args: args}
	}
	rows, err := qb.queryRawCtx(ctx, query, args...)
	if err != nil {
		return &QueryResult{err: err, query: query, args: args}
	}
	defer rows.Close()
	list, scanErr := scanRowsToMaps(rows)
	return &QueryResult{data: list, err: scanErr, query: query, args: args}
}

func (qb *Model) queryRawCtx(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if qb.session != nil {
		rawDB, err := qb.db.conn.RawDB()
		if err != nil {
			return nil, err
		}
		return rawDB.QueryContext(ctx, query, args...)
	}
	return qb.db.QueryRaw(ctx, query, args...)
}

func scanRowsToMaps(rows *sql.Rows) ([]map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	list := make([]map[string]interface{}, 0)
	for rows.Next() {
		values := make([]interface{}, len(columns))
		ptrs := make([]interface{}, len(columns))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make(map[string]interface{}, len(columns))
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		list = append(list, row)
	}
	return list, rows.Err()
}

// Paginate 分页查询方法
// page: 第几页（从1开始）
// pageSize: 每页显示多少行
// 返回: 包含items数据、page页码、size每页行数、total总数的分页结果
func (qb *Model) Paginate(ctx context.Context, page, pageSize int, dest interface{}) *PaginateResult {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	// 保存原始字段设置，避免Count操作影响后续查询
	originalFields := make([]string, len(qb.fields))
	copy(originalFields, qb.fields)

	countResult := qb.Count(ctx)
	// 恢复原始字段设置
	qb.fields = originalFields

	// 设置分页参数
	qb.Page(page, pageSize)

	// 手动执行数据查询的SQL打印（当处于SQLFetch模式时）
	if qb.sqlFetch {
		// 构建并打印数据查询的SQL
		query, args := qb.buildQuery(ctx)
		completeSQL := buildCompleteSQL(query, args)
		fmt.Printf("数据查询SQL: %s\n原始SQL: %s\n参数: %v\n", completeSQL, query, args)

		// 返回空结果
		return &PaginateResult{
			Items: []interface{}{},
			Page:  page,
			Size:  pageSize,
			Total: 0,
			Error: nil,
		}
	}

	total, ok := countResult.data.(int64)
	if !ok {
		return &PaginateResult{
			Items: []interface{}{},
			Page:  page,
			Size:  pageSize,
			Total: 0,
			Error: fmt.Errorf("count result type error"),
		}
	}
	// 如果总数为0，直接返回空结果
	if total == 0 {
		return &PaginateResult{
			Items: []interface{}{},
			Page:  page,
			Size:  pageSize,
			Total: 0,
			Error: nil,
		}
	}

	// 查询当前页数据
	selectResult := qb.Select(ctx, dest)
	if selectResult.err != nil {
		return &PaginateResult{
			Items: []interface{}{},
			Page:  page,
			Size:  pageSize,
			Total: total,
			Error: selectResult.err,
		}
	}

	return &PaginateResult{
		Items: dest,
		Page:  page,
		Size:  pageSize,
		Total: total,
		Error: nil,
	}
}

// Count 统计数量
func (qb *Model) Count(ctx context.Context) *QueryResult {
	if qb.buildErr != nil {
		return &QueryResult{data: int64(0), err: qb.buildErr}
	}
	qb.fields = []string{"COUNT(*)"}
	query, args := qb.buildQuery(ctx)

	// 如果设置了SQLFetch，只输出SQL不执行查询
	if qb.sqlFetch {
		completeSQL := buildCompleteSQL(query, args)
		fmt.Printf("完整SQL: %s\n原始SQL: %s\n参数: %v\n", completeSQL, query, args)
		return &QueryResult{
			data:  int64(0),
			err:   nil,
			query: query,
			args:  args,
		}
	}
	var count int64
	err := qb.queryRowCtx(ctx, &count, query, args...)
	if err != nil && err == sql.ErrNoRows {
		err = nil
		count = 0
	}
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

// Raw 执行原生SQL查询
func (qb *Model) Raw(ctx context.Context, dest interface{}, query string, args ...interface{}) *QueryResult {
	// 如果设置了SQLFetch，只输出SQL不执行查询
	if qb.sqlFetch {
		completeSQL := buildCompleteSQL(query, args)
		fmt.Printf("完整SQL: %s\n原始SQL: %s\n参数: %v\n", completeSQL, query, args)
		return &QueryResult{
			data:  dest,
			err:   nil,
			query: query,
			args:  args,
		}
	}

	err := qb.queryRowsCtx(ctx, dest, query, args...)
	if err != nil && err == sql.ErrNoRows {
		err = nil
		dest = nil
	}

	return &QueryResult{
		data:  dest,
		err:   err,
		query: query,
		args:  args,
	}
}

// RawExec 执行原生SQL执行语句
func (qb *Model) RawExec(ctx context.Context, query string, args ...interface{}) *QueryResult {
	// 如果设置了SQLFetch，只输出SQL不执行查询
	if qb.sqlFetch {
		completeSQL := buildCompleteSQL(query, args)
		fmt.Printf("完整SQL: %s\n原始SQL: %s\n参数: %v\n", completeSQL, query, args)
		return &QueryResult{
			data:  nil,
			err:   nil,
			query: query,
			args:  args,
		}
	}

	result, err := qb.execCtx(ctx, query, args...)
	if err != nil {
		return &QueryResult{
			data:  nil,
			err:   err,
			query: query,
			args:  args,
		}
	}

	// 获取最后插入的ID
	lastId, err := result.LastInsertId()
	lastIdStr := ""
	if err == nil {
		lastIdStr = strconv.FormatInt(lastId, 10)
	}

	return &QueryResult{
		data:   result,
		err:    nil,
		query:  query,
		args:   args,
		lastId: lastIdStr,
	}
}

// Sum 查询指定字段的合计数
func (qb *Model) Sum(ctx context.Context, field string) *QueryResult {
	qb.fields = []string{fmt.Sprintf("SUM(%s)", field)}
	query, args := qb.buildQuery(ctx)

	// 如果设置了SQLFetch，只输出SQL不执行查询
	if qb.sqlFetch {
		completeSQL := buildCompleteSQL(query, args)
		fmt.Printf("完整SQL: %s\n原始SQL: %s\n参数: %v\n", completeSQL, query, args)
		return &QueryResult{
			data:  int64(0),
			err:   nil,
			query: query,
			args:  args,
		}
	}

	var sum sql.NullFloat64
	err := qb.queryRowCtx(ctx, &sum, query, args...)

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
	qb.fields = []string{quoteField(field)}
	qb.Limit(1)
	query, args := qb.buildQuery(ctx)

	// 如果设置了SQLFetch，只输出SQL不执行查询
	if qb.sqlFetch {
		completeSQL := buildCompleteSQL(query, args)
		fmt.Printf("完整SQL: %s\n原始SQL: %s\n参数: %v\n", completeSQL, query, args)
		return &QueryResult{
			data:  int64(0),
			err:   nil,
			query: query,
			args:  args,
		}
	}

	var value string
	err := qb.queryRowCtx(ctx, &value, query, args...)
	if err != nil && err == sql.ErrNoRows {
		err = nil
		value = ""
	}
	return &QueryResult{
		data:  value,
		err:   err,
		query: query,
		args:  args,
	}
}

// Column 获取单一字段的所有值 - 使用QueryRows处理多行数据
func (qb *Model) Column(ctx context.Context, field string, dest interface{}) *QueryResult {
	qb.fields = []string{quoteField(field)}
	query, args := qb.buildQuery(ctx)

	// 如果设置了SQLFetch，只输出SQL不执行查询
	if qb.sqlFetch {
		completeSQL := buildCompleteSQL(query, args)
		fmt.Printf("完整SQL: %s\n原始SQL: %s\n参数: %v\n", completeSQL, query, args)
		return &QueryResult{
			data:  []interface{}{},
			err:   nil,
			query: query,
			args:  args,
		}
	}
	var result interface{}
	var err error

	switch d := dest.(type) {
	case *[]string:
		var stringResult []string
		err = qb.queryRowsCtx(ctx, &stringResult, query, args...)
		if err == nil {
			*d = stringResult
			result = stringResult
		}
	case *[]int:
		var intResult []int
		err = qb.queryRowsCtx(ctx, &intResult, query, args...)
		if err == nil {
			*d = intResult
			result = intResult
		}
	case *[]int64:
		var int64Result []int64
		err = qb.queryRowsCtx(ctx, &int64Result, query, args...)
		if err == nil {
			*d = int64Result
			result = int64Result
		}
	case *[]uint:
		var intResult []uint
		err = qb.queryRowsCtx(ctx, &intResult, query, args...)
		if err == nil {
			*d = intResult
			result = intResult
		}
	case *[]uint64:
		var int64Result []uint64
		err = qb.queryRowsCtx(ctx, &int64Result, query, args...)
		if err == nil {
			*d = int64Result
			result = int64Result
		}
	case *[]float64:
		var float64Result []float64
		err = qb.queryRowsCtx(ctx, &float64Result, query, args...)
		if err == nil {
			*d = float64Result
			result = float64Result
		}
	case *[]bool:
		var boolResult []bool
		err = qb.queryRowsCtx(ctx, &boolResult, query, args...)
		if err == nil {
			*d = boolResult
			result = boolResult
		}
	case *[]interface{}:
		var interfaceResult []interface{}
		err = qb.queryRowsCtx(ctx, &interfaceResult, query, args...)
		if err == nil {
			*d = interfaceResult
			result = interfaceResult
		}
	default:
		// 默认情况下，使用[]string类型
		var stringResult []string
		err = qb.queryRowsCtx(ctx, &stringResult, query, args...)
		result = stringResult
	}
	// 如果查询出错，返回空结果
	if err != nil {
		if err == sql.ErrNoRows {
			err = nil
		}
		return &QueryResult{
			data:  []interface{}{},
			err:   err,
			query: query,
			args:  args,
		}
	}

	return &QueryResult{
		data:  result,
		err:   nil,
		query: query,
		args:  args,
	}
}

// Data 设置数据操作字段
func (qb *Model) Data(data interface{}) *Model {
	if data != nil {
		// 检查 data 是否是切片类型
		reflectValue := reflect.ValueOf(data)
		if reflectValue.Kind() == reflect.Slice {
			// 如果是切片类型，直接存储
			qb.data = data
		} else {
			// 如果不是切片类型，转换为 map
			dataMap, err := qb.convertToMap(data)
			if err == nil {
				qb.data = dataMap
			}
		}
	}
	return qb
}

// Insert 使用INSERT INTO语句进行数据库写入，如果写入的数据中存在主键或者唯一索引时，返回失败
func (qb *Model) Insert(ctx context.Context, data ...interface{}) *QueryResult {
	// 处理参数
	if len(data) > 0 && data[0] != nil {
		dataMap, err := qb.convertToMap(data[0])
		if err != nil {
			return &QueryResult{
				data:  nil,
				err:   err,
				query: "",
				args:  nil,
			}
		}
		qb.data = dataMap
	}
	// 如果没有数据，返回错误
	if qb.data == nil {
		return &QueryResult{
			data:  nil,
			err:   fmt.Errorf("no data to insert"),
			query: "",
			args:  nil,
		}
	}
	// 确保 qb.data 是一个 map
	dataMap, err := qb.convertToMap(qb.data)
	if err != nil {
		return &QueryResult{
			data:  nil,
			err:   err,
			query: "",
			args:  nil,
		}
	}
	var sql strings.Builder
	var args []interface{}

	sql.WriteString("INSERT INTO ")
	sql.WriteString(qb.table)
	sql.WriteString(" (")

	fields := make([]string, 0, len(dataMap))
	placeholders := make([]string, 0, len(dataMap))

	for _, field := range sortedMapKeys(dataMap) {
		fields = append(fields, quoteField(field))
		placeholders = append(placeholders, "?")
		args = append(args, dataMap[field])
	}

	sql.WriteString(strings.Join(fields, ", "))
	sql.WriteString(") VALUES (")
	sql.WriteString(strings.Join(placeholders, ", "))
	sql.WriteString(")")

	query := sql.String()

	// 如果设置了SQLFetch，只输出SQL不执行查询
	if qb.sqlFetch {
		completeSQL := buildCompleteSQL(query, args)
		fmt.Printf("完整SQL: %s\n原始SQL: %s\n参数: %v\n", completeSQL, query, args)
		return &QueryResult{
			data:  int64(0),
			err:   nil,
			query: query,
			args:  args,
		}
	}

	result, err := qb.execCtx(ctx, query, args...)
	if err != nil {
		return &QueryResult{
			data:  result,
			err:   err,
			query: query,
			args:  args,
		}
	}
	lastInsertID, err := result.LastInsertId()
	return &QueryResult{
		data:   result,
		err:    err,
		query:  query,
		args:   args,
		lastId: ga.String(lastInsertID),
	}
}

// Save 使用INSERT INTO语句进行数据库写入，如果写入的数据中存在主键或者唯一索引时，更新原有数据
func (qb *Model) Save(ctx context.Context, data ...interface{}) *QueryResult {
	// 处理参数
	if len(data) > 0 && data[0] != nil {
		dataMap, err := qb.convertToMap(data[0])
		if err != nil {
			return &QueryResult{
				data:  nil,
				err:   err,
				query: "",
				args:  nil,
			}
		}
		qb.data = dataMap
	}

	// 如果没有数据，返回错误
	if qb.data == nil {
		return &QueryResult{
			data:  nil,
			err:   fmt.Errorf("no data to save"),
			query: "",
			args:  nil,
		}
	}
	// 确保 qb.data 是一个 map
	dataMap, err := qb.convertToMap(qb.data)
	if err != nil {
		return &QueryResult{
			data:  nil,
			err:   err,
			query: "",
			args:  nil,
		}
	}
	// 如果没有数据，返回错误
	if len(dataMap) == 0 {
		return &QueryResult{
			data:  nil,
			err:   fmt.Errorf("no data to save"),
			query: "",
			args:  nil,
		}
	}
	var sql strings.Builder
	var args []interface{}

	sql.WriteString("INSERT INTO ")
	sql.WriteString(qb.table)
	sql.WriteString(" (")

	fields := make([]string, 0, len(dataMap))
	placeholders := make([]string, 0, len(dataMap))

	for _, field := range sortedMapKeys(dataMap) {
		fields = append(fields, quoteField(field))
		placeholders = append(placeholders, "?")
		args = append(args, dataMap[field])
	}
	sql.WriteString(strings.Join(fields, ", "))
	sql.WriteString(") VALUES (")
	sql.WriteString(strings.Join(placeholders, ", "))
	sql.WriteString(") ON DUPLICATE KEY UPDATE ")

	updates := make([]string, 0, len(dataMap))
	for _, field := range sortedMapKeys(dataMap) {
		qf := quoteField(field)
		updates = append(updates, fmt.Sprintf("%s = VALUES(%s)", qf, qf))
	}

	sql.WriteString(strings.Join(updates, ", "))

	query := sql.String()

	// 如果设置了SQLFetch，只输出SQL不执行查询
	if qb.sqlFetch {
		completeSQL := buildCompleteSQL(query, args)
		fmt.Printf("完整SQL: %s\n原始SQL: %s\n参数: %v\n", completeSQL, query, args)
		return &QueryResult{
			data:  int64(0),
			err:   nil,
			query: query,
			args:  args,
		}
	}

	result, err := qb.execCtx(ctx, query, args...)
	if err != nil {
		return &QueryResult{
			data:  result,
			err:   err,
			query: query,
			args:  args,
		}
	}
	lastInsertID, _ := result.LastInsertId()
	return &QueryResult{
		data:   result,
		err:    nil,
		query:  query,
		args:   args,
		lastId: ga.String(lastInsertID),
	}
}

// UpdateBatch 批量更新数据
// 使用 CASE WHEN 实现单条 SQL 批量更新
// 数据格式: []map[string]interface{}{{"id": 1, "field": value}, {"id": 2, "field": value}}
func (qb *Model) UpdateBatch(ctx context.Context, data ...interface{}) *QueryResult {
	var dataMaps []map[string]interface{}
	var err error

	if qb.data != nil {
		reflectValue := reflect.ValueOf(qb.data)
		if reflectValue.Kind() == reflect.Slice {
			dataMaps, err = qb.convertToMaps(qb.data)
			if err != nil {
				return &QueryResult{data: nil, err: err, query: "", args: nil}
			}
		} else {
			return &QueryResult{data: nil, err: fmt.Errorf("UpdateBatch requires slice data"), query: "", args: nil}
		}
	} else if len(data) > 0 && data[0] != nil {
		reflectValue := reflect.ValueOf(data[0])
		if reflectValue.Kind() == reflect.Slice {
			dataMaps, err = qb.convertToMaps(data[0])
		} else {
			dataMaps, err = qb.convertToMaps(data)
		}
		if err != nil {
			return &QueryResult{data: nil, err: err, query: "", args: nil}
		}
	}

	if len(dataMaps) == 0 {
		return &QueryResult{data: nil, err: fmt.Errorf("no data to update"), query: "", args: nil}
	}

	var sql strings.Builder
	var args []interface{}

	sql.WriteString("UPDATE ")
	sql.WriteString(qb.table)
	sql.WriteString(" SET ")

	ids := make([]interface{}, 0, len(dataMaps))
	caseWhenArgs := make([]interface{}, 0)

	for _, row := range dataMaps {
		if id, ok := row["id"]; ok {
			ids = append(ids, id)
		}
	}

	if len(ids) == 0 {
		return &QueryResult{data: nil, err: fmt.Errorf("id field is required for UpdateBatch"), query: "", args: nil}
	}

	fields := make([]string, 0)
	if len(dataMaps) > 0 {
		for field := range dataMaps[0] {
			if field != "id" {
				fields = append(fields, field)
			}
		}
	}

	if len(fields) == 0 {
		return &QueryResult{data: nil, err: fmt.Errorf("no fields to update"), query: "", args: nil}
	}

	for _, field := range fields {
		sql.WriteString(field)
		sql.WriteString(" = CASE id ")
		for _, row := range dataMaps {
			id := row["id"]
			value := row[field]
			sql.WriteString("WHEN ? THEN ? ")
			caseWhenArgs = append(caseWhenArgs, id, value)
		}
		sql.WriteString("END, ")
	}

	sqlStr := sql.String()
	sqlStr = strings.TrimRight(sqlStr, ", ")
	sqlStr += " WHERE id IN ("

	placeholders := make([]string, 0, len(ids))
	for range ids {
		placeholders = append(placeholders, "?")
	}
	sqlStr += strings.Join(placeholders, ", ")
	sqlStr += ")"
	if qb.hasDeleteTimeField(ctx) && !qb.withTrashed && !qb.onlyTrashed {
		sqlStr += " AND delete_time = 0"
	} else if qb.hasDeleteTimeField(ctx) && qb.onlyTrashed {
		sqlStr += " AND delete_time > 0"
	}

	query := sqlStr
	args = append(caseWhenArgs, ids...)

	if qb.sqlFetch {
		completeSQL := buildCompleteSQL(query, args)
		fmt.Printf("完整SQL: %s\n原始SQL: %s\n参数: %v\n", completeSQL, query, args)
		return &QueryResult{data: int64(0), err: nil, query: query, args: args}
	}

	result, err := qb.execCtx(ctx, query, args...)
	if err != nil {
		return &QueryResult{data: nil, err: err, query: query, args: args}
	}
	rowsAffected, _ := result.RowsAffected()
	return &QueryResult{data: rowsAffected, err: nil, query: query, args: args}
}

// InsertAll 批量插入
func (qb *Model) InsertAll(ctx context.Context, data ...interface{}) *QueryResult {
	var dataMaps []map[string]interface{}
	var err error

	// 优先使用 qb.data 中的数据
	if qb.data != nil {
		// 检查 qb.data 是否是一个切片类型
		reflectValue := reflect.ValueOf(qb.data)
		if reflectValue.Kind() == reflect.Slice {
			// 如果 qb.data 是切片类型，直接使用它
			dataMaps, err = qb.convertToMaps(qb.data)
			if err != nil {
				return &QueryResult{
					data:  nil,
					err:   err,
					query: "",
					args:  nil,
				}
			}
		} else {
			// 如果 qb.data 不是切片类型，将其转换为 map 后再添加到切片中
			dataMap, err := qb.convertToMap(qb.data)
			if err != nil {
				return &QueryResult{
					data:  nil,
					err:   err,
					query: "",
					args:  nil,
				}
			}
			dataMaps = []map[string]interface{}{dataMap}
		}
	} else if len(data) > 0 && data[0] != nil {
		// 如果 qb.data 为空，使用传递的参数
		// 检查 data[0] 是否是切片类型，如果是，直接使用它
		reflectValue := reflect.ValueOf(data[0])
		if reflectValue.Kind() == reflect.Slice {
			dataMaps, err = qb.convertToMaps(data[0])
		} else {
			dataMaps, err = qb.convertToMaps(data)
		}
		if err != nil {
			return &QueryResult{
				data:  nil,
				err:   err,
				query: "",
				args:  nil,
			}
		}
	}

	// 如果没有数据，返回错误
	if len(dataMaps) == 0 {
		return &QueryResult{
			data:  nil,
			err:   fmt.Errorf("no data to insert"),
			query: "",
			args:  nil,
		}
	}
	var sql strings.Builder
	var args []interface{}

	sql.WriteString("INSERT INTO ")
	sql.WriteString(qb.table)
	sql.WriteString(" (")

	// 获取所有字段名
	fields := sortedMapKeys(dataMaps[0])

	sql.WriteString(strings.Join(fields, ", "))
	sql.WriteString(") VALUES ")
	// 创建占位符
	placeholders := make([]string, len(fields))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	valuePlaceholders := make([]string, 0, len(dataMaps))
	for i, row := range dataMaps {
		if i > 0 {
			valuePlaceholders = append(valuePlaceholders, ", ")
		}
		valuePlaceholders = append(valuePlaceholders, "("+strings.Join(placeholders, ", ")+")")

		for _, field := range fields {
			args = append(args, row[field])
		}
	}

	sql.WriteString(strings.Join(valuePlaceholders, ""))

	query := sql.String()

	// 如果设置了 SQLFetch，只输出 SQL 不执行查询
	if qb.sqlFetch {
		completeSQL := buildCompleteSQL(query, args)
		fmt.Printf("完整 SQL: %s\n原始 SQL: %s\n参数：%v\n", completeSQL, query, args)
		return &QueryResult{
			data:  int64(0),
			err:   nil,
			query: query,
			args:  args,
		}
	}

	result, err := qb.execCtx(ctx, query, args...)
	return &QueryResult{
		data:  result,
		err:   err,
		query: query,
		args:  args,
	}
}

// Update 按 Where 条件更新；软删 Delete 内部亦走此方法。返回 *QueryResult，不含 LastInsertId。
func (qb *Model) Update(ctx context.Context, data ...interface{}) *QueryResult {
	if qb.buildErr != nil {
		return &QueryResult{data: nil, err: qb.buildErr, query: "", args: nil}
	}
	var err error
	// 如果Data没有设置数据，才使用Update参数中的数据
	if qb.data == nil && len(data) > 0 && data[0] != nil {
		dataMap, err := qb.convertToMap(data[0])
		if err != nil {
			return &QueryResult{
				data:  nil,
				err:   err,
				query: "",
				args:  nil,
			}
		}
		qb.data = dataMap
	}

	// 检查是否有更新数据
	hasData := false
	if qb.data != nil {
		var tempDataMap map[string]interface{}
		tempDataMap, err = qb.convertToMap(qb.data)
		if err != nil {
			return &QueryResult{
				data:  nil,
				err:   err,
				query: "",
				args:  nil,
			}
		}
		hasData = len(tempDataMap) > 0
	}

	// 检查是否有其他更新数据
	hasData = hasData || len(qb.updateData) > 0 || len(qb.incData) > 0 || len(qb.decData) > 0

	// 如果没有数据，返回错误
	if !hasData {
		return &QueryResult{
			data:  nil,
			err:   fmt.Errorf("no data to update"),
			query: "",
			args:  nil,
		}
	}

	// 确保 qb.data 是一个 map
	var dataMap map[string]interface{}
	if qb.data != nil {
		dataMap, err = qb.convertToMap(qb.data)
		if err != nil {
			return &QueryResult{
				data:  nil,
				err:   err,
				query: "",
				args:  nil,
			}
		}
	} else {
		dataMap = make(map[string]interface{})
	}
	// 合并所有更新数据
	updateData := make(map[string]interface{})

	// 添加普通更新数据
	if len(qb.updateData) > 0 {
		for k, v := range qb.updateData {
			updateData[k] = v
		}
	} else if len(dataMap) > 0 {
		for k, v := range dataMap {
			updateData[k] = v
		}
	}

	var sql strings.Builder
	var args []interface{}

	sql.WriteString("UPDATE ")
	sql.WriteString(qb.table)
	sql.WriteString(" SET ")

	sets := make([]string, 0, len(updateData)+len(qb.incData)+len(qb.decData))

	// 普通字段更新（支持表达式）
	for _, field := range sortedMapKeys(updateData) {
		value := updateData[field]
		qf := quoteField(field)
		if str, ok := value.(string); ok {
			// 支持MySQL函数表达式和字段表达式
			if str == "NOW()" || str == "CURRENT_TIMESTAMP" || str == softDeleteTimeExpr ||
				strings.Contains(str, field+" +") || strings.Contains(str, field+" -") ||
				strings.Contains(str, field+" *") || strings.Contains(str, field+" /") {
				sets = append(sets, fmt.Sprintf("%s = %s", qf, str))
			} else {
				sets = append(sets, fmt.Sprintf("%s = ?", qf))
				args = append(args, value)
			}
		} else {
			sets = append(sets, fmt.Sprintf("%s = ?", qf))
			args = append(args, value)
		}
	}

	sql.WriteString(strings.Join(sets, ", "))

	hasWhere := qb.buildWhereSQL(&sql, &args)
	qb.writeSoftDeleteWhere(ctx, &sql, hasWhere)

	query := sql.String()

	// 如果设置了SQLFetch，只输出SQL不执行查询
	if qb.sqlFetch {
		completeSQL := buildCompleteSQL(query, args)
		fmt.Printf("完整SQL: %s\n原始SQL: %s\n参数: %v\n", completeSQL, query, args)
		return &QueryResult{
			data:  int64(0),
			err:   nil,
			query: query,
			args:  args,
		}
	}

	result, err := qb.execCtx(ctx, query, args...)
	if err != nil {
		return &QueryResult{
			data:  result,
			err:   err,
			query: query,
			args:  args,
		}
	}
	return &QueryResult{
		data:  result,
		err:   err,
		query: query,
		args:  args,
	}
}

// Replace 使用REPLACE INTO语句进行数据库写入
func (qb *Model) Replace(ctx context.Context, data ...interface{}) *QueryResult {
	// 如果Data没有设置数据，才使用Replace参数中的数据
	if qb.data == nil && len(data) > 0 && data[0] != nil {
		dataMap, err := qb.convertToMap(data[0])
		if err != nil {
			return &QueryResult{
				data:  nil,
				err:   err,
				query: "",
				args:  nil,
			}
		}
		qb.data = dataMap
	}

	// 如果没有数据，返回错误
	if qb.data == nil {
		return &QueryResult{
			data:  nil,
			err:   fmt.Errorf("no data to replace"),
			query: "",
			args:  nil,
		}
	}
	// 确保 qb.data 是一个 map
	dataMap, err := qb.convertToMap(qb.data)
	if err != nil {
		return &QueryResult{
			data:  nil,
			err:   err,
			query: "",
			args:  nil,
		}
	}
	// 如果没有数据，返回错误
	if len(dataMap) == 0 {
		return &QueryResult{
			data:  nil,
			err:   fmt.Errorf("no data to replace"),
			query: "",
			args:  nil,
		}
	}

	var sql strings.Builder
	var args []interface{}

	sql.WriteString("REPLACE INTO ")
	sql.WriteString(qb.table)
	sql.WriteString(" (")

	fields := make([]string, 0, len(dataMap))
	placeholders := make([]string, 0, len(dataMap))

	for _, field := range sortedMapKeys(dataMap) {
		fields = append(fields, quoteField(field))
		placeholders = append(placeholders, "?")
		args = append(args, dataMap[field])
	}

	sql.WriteString(strings.Join(fields, ", "))
	sql.WriteString(") VALUES (")
	sql.WriteString(strings.Join(placeholders, ", "))
	sql.WriteString(")")

	query := sql.String()

	// 如果设置了SQLFetch，只输出SQL不执行查询
	if qb.sqlFetch {
		completeSQL := buildCompleteSQL(query, args)
		fmt.Printf("完整SQL: %s\n原始SQL: %s\n参数: %v\n", completeSQL, query, args)
		return &QueryResult{
			data:  int64(0),
			err:   nil,
			query: query,
			args:  args,
		}
	}

	result, err := qb.execCtx(ctx, query, args...)
	return &QueryResult{
		data:  result,
		err:   err,
		query: query,
		args:  args,
	}
}

// Inc 指定字段的自增操作
func (qb *Model) Inc(ctx context.Context, field string, value interface{}) *QueryResult {
	var sql strings.Builder
	var args []interface{}

	sql.WriteString("UPDATE ")
	sql.WriteString(qb.table)
	sql.WriteString(" SET ")
	sql.WriteString(field)
	sql.WriteString(" = ")
	sql.WriteString(field)
	sql.WriteString(" + ?")
	args = append(args, value)

	hasWhere := qb.buildWhereSQL(&sql, &args)
	qb.writeSoftDeleteWhere(ctx, &sql, hasWhere)

	query := sql.String()

	// 如果设置了SQLFetch，只输出SQL不执行查询
	if qb.sqlFetch {
		completeSQL := buildCompleteSQL(query, args)
		fmt.Printf("完整SQL: %s\n原始SQL: %s\n参数: %v\n", completeSQL, query, args)
		return &QueryResult{
			data:  int64(0),
			err:   nil,
			query: query,
			args:  args,
		}
	}

	result, err := qb.execCtx(ctx, query, args...)
	return &QueryResult{
		data:  result,
		err:   err,
		query: query,
		args:  args,
	}
}

// Dec 指定字段的自减操作（直接执行，不需要接Update）
func (qb *Model) Dec(ctx context.Context, field string, value interface{}) *QueryResult {
	var sql strings.Builder
	var args []interface{}

	sql.WriteString("UPDATE ")
	sql.WriteString(qb.table)
	sql.WriteString(" SET ")
	sql.WriteString(field)
	sql.WriteString(" = ")
	sql.WriteString(field)
	sql.WriteString(" - ?")
	args = append(args, value)

	hasWhere := qb.buildWhereSQL(&sql, &args)
	qb.writeSoftDeleteWhere(ctx, &sql, hasWhere)

	query := sql.String()

	// 如果设置了SQLFetch，只输出SQL不执行查询
	if qb.sqlFetch {
		completeSQL := buildCompleteSQL(query, args)
		fmt.Printf("完整SQL: %s\n原始SQL: %s\n参数: %v\n", completeSQL, query, args)
		return &QueryResult{
			data:  int64(0),
			err:   nil,
			query: query,
			args:  args,
		}
	}

	result, err := qb.execCtx(ctx, query, args...)
	return &QueryResult{
		data:  result,
		err:   err,
		query: query,
		args:  args,
	}
}

// -------- 软删除 --------

// WithTrashed 查询/更新时包含未删与已删记录（不过滤 delete_time）
func (qb *Model) WithTrashed() *Model {
	qb.withTrashed = true
	qb.onlyTrashed = false
	return qb
}

// OnlyTrashed 只操作已软删记录（delete_time > 0）
func (qb *Model) OnlyTrashed() *Model {
	qb.onlyTrashed = true
	qb.withTrashed = false
	return qb
}

// Restore 恢复软删记录（delete_time 置 0，须带 Where 条件）
func (qb *Model) Restore(ctx context.Context) *QueryResult {
	if qb.buildErr != nil {
		return &QueryResult{data: nil, err: qb.buildErr, query: "", args: nil}
	}
	if !qb.hasDeleteTimeField(ctx) {
		return &QueryResult{data: nil, err: fmt.Errorf("table %s has no delete_time field", qb.table), query: "", args: nil}
	}
	if len(qb.where) == 0 {
		return &QueryResult{data: nil, err: fmt.Errorf("restore requires where conditions"), query: "", args: nil}
	}
	qb.onlyTrashed = true
	qb.withTrashed = false
	if qb.updateData == nil {
		qb.updateData = make(map[string]interface{})
	}
	qb.updateData["delete_time"] = int64(0)
	return qb.Update(ctx)
}

// ForceDelete 物理删除（无视 delete_time 软删标记）
func (qb *Model) ForceDelete(ctx context.Context) *QueryResult {
	if qb.buildErr != nil {
		return &QueryResult{data: nil, err: qb.buildErr, query: "", args: nil}
	}
	return qb.forceDelete(ctx)
}

// Delete 删除数据（有 delete_time 字段则软删写时间戳，否则物理删除）
func (qb *Model) Delete(ctx context.Context) *QueryResult {
	if qb.buildErr != nil {
		return &QueryResult{data: nil, err: qb.buildErr, query: "", args: nil}
	}
	if qb.hasDeleteTimeField(ctx) {
		if qb.updateData == nil {
			qb.updateData = make(map[string]interface{})
		}
		qb.updateData["delete_time"] = softDeleteTimeExpr
		return qb.Update(ctx)
	}
	return qb.forceDelete(ctx)
}

// forceDelete 执行物理 DELETE（不检查 delete_time，供 Delete/ForceDelete 调用）
func (qb *Model) forceDelete(ctx context.Context) *QueryResult {
	var sql strings.Builder
	var args []interface{}

	if qb.alias != "" {
		sql.WriteString("DELETE ")
		sql.WriteString(qb.alias)
		sql.WriteString(" FROM ")
		sql.WriteString(qb.table)
		sql.WriteString(" AS ")
		sql.WriteString(qb.alias)
	} else {
		sql.WriteString("DELETE FROM ")
		sql.WriteString(qb.table)
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
	qb.buildWhereSQL(&sql, &args)

	query := sql.String()

	// 如果设置了SQLFetch，只输出SQL不执行查询
	if qb.sqlFetch {
		completeSQL := buildCompleteSQL(query, args)
		fmt.Printf("完整SQL: %s\n原始SQL: %s\n参数: %v\n", completeSQL, query, args)
		return &QueryResult{
			data:  int64(0),
			err:   nil,
			query: query,
			args:  args,
		}
	}

	result, err := qb.execCtx(ctx, query, args...)
	return &QueryResult{
		data:  result,
		err:   err,
		query: query,
		args:  args,
	}
}

// buildQuery 构建 SELECT 查询（默认排除软删：delete_time = 0）
func (qb *Model) buildQuery(ctx context.Context) (string, []interface{}) {
	if qb.buildErr != nil {
		return "", nil
	}
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
	conditions := make([]string, 0)

	// 处理其他WHERE条件
	for i, where := range qb.where {
		if i > 0 || len(conditions) > 0 {
			conditions = append(conditions, " "+where.operator+" ")
		}
		// 如果field为空，表示这是完整条件，只使用cond
		if where.field == "" {
			conditions = append(conditions, where.cond)
		} else {
			// 简单条件，组合字段名和条件
			conditions = append(conditions, quoteField(where.field)+" "+where.cond)
		}
		args = append(args, where.args...)
	}

	conditions = qb.appendSoftDeleteConditions(ctx, conditions)

	// 如果有条件，添加WHERE子句
	if len(conditions) > 0 {
		sql.WriteString(" WHERE ")
		sql.WriteString(strings.Join(conditions, ""))
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
	rv := reflect.ValueOf(v)
	// 处理指针类型
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return true // 空指针视为空
		}
		rv = rv.Elem()
	}

	if rv.Kind() == reflect.Slice {
		return rv.Len() == 0
	}
	return false
}

// quoteField 为简单字段名加反引号，避免 MySQL 保留字解析问题（如 key、role_id）
func quoteField(field string) string {
	field = strings.TrimSpace(field)
	if field == "" {
		return field
	}
	if strings.HasPrefix(field, "`") && strings.HasSuffix(field, "`") {
		return field
	}
	if strings.ContainsAny(field, " (,.*") {
		return field
	}
	return "`" + strings.ReplaceAll(field, "`", "") + "`"
}

// quoteWhereCond 为完整 WHERE 片段中的字段名加反引号，如 key = ? -> `key` = ?
func quoteWhereCond(cond string) string {
	cond = strings.TrimSpace(cond)
	if cond == "" {
		return cond
	}
	if strings.Contains(cond, "`") || strings.Contains(strings.ToUpper(cond), " IN ") {
		return cond
	}
	parts := strings.Fields(cond)
	if len(parts) >= 3 && isWhereOperator(parts[1]) {
		return quoteField(parts[0]) + " " + strings.Join(parts[1:], " ")
	}
	if len(parts) >= 2 && isWhereOperator(parts[0]) {
		return cond
	}
	if len(parts) >= 2 {
		return quoteField(parts[0]) + " " + strings.Join(parts[1:], " ")
	}
	return cond
}

func isWhereOperator(token string) bool {
	switch token {
	case "=", "<>", "!=", ">", "<", ">=", "<=", "LIKE", "like":
		return true
	default:
		return false
	}
}

// formatSQLValue 格式化SQL参数值为可打印的字符串
func formatSQLValue(arg interface{}) string {
	if arg == nil {
		return "NULL"
	}

	switch v := arg.(type) {
	case string:
		// 转义单引号
		return fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
	case time.Time:
		return fmt.Sprintf("'%s'", v.Format("2006-01-02 15:04:05"))
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%v", v)
	case float32, float64:
		return fmt.Sprintf("%v", v)
	case bool:
		if v {
			return "1"
		}
		return "0"
	default:
		// 对于其他类型，使用反射处理
		rv := reflect.ValueOf(arg)
		switch rv.Kind() {
		case reflect.String:
			return fmt.Sprintf("'%s'", strings.ReplaceAll(rv.String(), "'", "''"))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return strconv.FormatInt(rv.Int(), 10)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return strconv.FormatUint(rv.Uint(), 10)
		case reflect.Float32, reflect.Float64:
			return strconv.FormatFloat(rv.Float(), 'f', -1, 64)
		case reflect.Bool:
			if rv.Bool() {
				return "1"
			}
			return "0"
		default:
			return fmt.Sprintf("'%s'", strings.ReplaceAll(fmt.Sprintf("%v", arg), "'", "''"))
		}
	}
}

// buildCompleteSQL 构建完整的SQL语句（将参数替换到占位符中）
func buildCompleteSQL(query string, args []interface{}) string {
	if len(args) == 0 {
		return query
	}

	result := query
	argIndex := 0

	// 替换所有问号占位符
	for argIndex < len(args) && strings.Contains(result, "?") {
		// 找到第一个问号
		index := strings.Index(result, "?")
		if index == -1 {
			break
		}

		// 替换问号
		if argIndex < len(args) {
			formattedValue := formatSQLValue(args[argIndex])
			result = result[:index] + formattedValue + result[index+1:]
			argIndex++
		} else {
			break
		}
	}

	return result
}

// softDeleteTimeExpr 软删除时写入的毫秒时间戳表达式
const softDeleteTimeExpr = "UNIX_TIMESTAMP(NOW(3)) * 1000"

// softDeleteScope 返回 delete_time 过滤片段；第二返回值表示表是否有 delete_time 字段
func (qb *Model) softDeleteScope(ctx context.Context) (cond string, hasField bool) {
	if !qb.hasDeleteTimeField(ctx) {
		return "", false
	}
	if qb.onlyTrashed {
		return "delete_time > 0", true
	}
	if qb.withTrashed {
		return "", true
	}
	return "delete_time = 0", true
}

// appendSoftDeleteConditions 将 delete_time 条件并入字符串条件切片（用于 COUNT 等子查询）
func (qb *Model) appendSoftDeleteConditions(ctx context.Context, conditions []string) []string {
	scope, hasField := qb.softDeleteScope(ctx)
	if !hasField || scope == "" {
		return conditions
	}
	if len(conditions) > 0 {
		return append(conditions, " AND "+scope)
	}
	return append(conditions, scope)
}

// writeSoftDeleteWhere 在已有或新建 WHERE 后追加 delete_time 过滤
func (qb *Model) writeSoftDeleteWhere(ctx context.Context, sql *strings.Builder, hasWhere bool) {
	scope, hasField := qb.softDeleteScope(ctx)
	if !hasField || scope == "" {
		return
	}
	if hasWhere {
		sql.WriteString(" AND ")
	} else {
		sql.WriteString(" WHERE ")
	}
	sql.WriteString(scope)
}

// buildWhereSQL 拼接 WHERE 子句，返回是否已有 WHERE
func (qb *Model) buildWhereSQL(sql *strings.Builder, args *[]interface{}) bool {
	if len(qb.where) == 0 {
		return false
	}
	sql.WriteString(" WHERE ")
	for i, where := range qb.where {
		if i > 0 || where.operator != "" {
			sql.WriteString(" ")
			sql.WriteString(where.operator)
			sql.WriteString(" ")
		}
		if where.field != "" {
			sql.WriteString(quoteField(where.field))
			sql.WriteString(" ")
			sql.WriteString(where.cond)
		} else {
			sql.WriteString(quoteWhereCond(where.cond))
		}
		*args = append(*args, where.args...)
	}
	return true
}

// sortedMapKeys 按字段名字典序返回 map 键，保证 INSERT/UPDATE 列顺序稳定
func sortedMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Clone 深拷贝当前构建器，用于在同一组条件上派生 Count/Select/Update 等，避免链式污染。
func (qb *Model) Clone() *Model {
	if qb == nil {
		return nil
	}
	clone := *qb
	clone.where = cloneWhereClauses(qb.where)
	clone.joins = append([]joinClause(nil), qb.joins...)
	clone.groupBy = append([]string(nil), qb.groupBy...)
	clone.having = cloneWhereClauses(qb.having)
	clone.orderBy = append([]orderClause(nil), qb.orderBy...)
	clone.fields = append([]string(nil), qb.fields...)
	clone.updateData = cloneStringAnyMap(qb.updateData)
	clone.incData = cloneStringAnyMap(qb.incData)
	clone.decData = cloneStringAnyMap(qb.decData)
	if qb.hasDeleteTime != nil {
		v := *qb.hasDeleteTime
		clone.hasDeleteTime = &v
	}
	return &clone
}

func cloneWhereClauses(src []whereClause) []whereClause {
	if len(src) == 0 {
		return nil
	}
	out := make([]whereClause, len(src))
	for i, w := range src {
		out[i] = whereClause{
			operator: w.operator,
			field:    w.field,
			cond:     w.cond,
			args:     append([]interface{}(nil), w.args...),
		}
	}
	return out
}

func cloneStringAnyMap(src map[string]interface{}) map[string]interface{} {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]interface{}, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

// hasDeleteTimeField 检测表是否有 delete_time 字段（优先测试覆盖值，否则查 DBManager 表元数据缓存）
func (qb *Model) hasDeleteTimeField(ctx context.Context) bool {
	if qb.hasDeleteTime != nil {
		return *qb.hasDeleteTime
	}
	if qb.db == nil {
		return false
	}
	return qb.db.tableHasDeleteTime(ctx, qb.table)
}

// convertToInterfaceSlice 将任意类型的切片转换为[]interface{}
func convertToInterfaceSlice(values interface{}) []interface{} {
	if values == nil {
		return []interface{}{}
	}

	// 尝试直接类型断言
	switch v := values.(type) {
	case []interface{}:
		return v
	case []string:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = val
		}
		return result
	case []int:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = val
		}
		return result
	case []int64:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = val
		}
		return result
	case []uint64:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = val
		}
		return result
	case []*uint64:
		result := make([]interface{}, len(v))
		for i, val := range v {
			if val != nil {
				result[i] = *val
			}
		}
		return result
	case []*gvar.Var:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = val
		}
		return result
	default:
		// 使用反射处理其他类型
		return reflectConvertToInterfaceSlice(values)
	}
}

// reflectConvertToInterfaceSlice 使用反射将任意类型的切片转换为[]interface{}
func reflectConvertToInterfaceSlice(values interface{}) []interface{} {
	v := reflect.ValueOf(values)
	if v.Kind() != reflect.Slice {
		return []interface{}{values}
	}

	// 先使用临时切片存储结果
	var tempResult []interface{}
	for i := 0; i < v.Len(); i++ {
		// 检查是否为指针类型
		elem := v.Index(i)
		if elem.Kind() == reflect.Ptr {
			// 是指针，解引用
			if elem.IsNil() {
				// 指针为nil，跳过
				continue
			}
			tempResult = append(tempResult, elem.Elem().Interface())

		} else {
			// 不是指针，直接使用
			tempResult = append(tempResult, elem.Interface())
		}
	}

	return tempResult
}

// getConditionArgs 处理条件值，转换为 []interface{}
func getConditionArgs(value interface{}) []interface{} {
	if value == nil {
		return []interface{}{}
	}

	return convertToInterfaceSlice(value)
}
// SetPrimaryKey 手动指定主键列名（默认从 information_schema 缓存读取，通常为 id）
func (qb *Model) SetPrimaryKey(key string) *Model {
	qb.primaryKey = key
	return qb
}

// getPrimaryKey 返回主键列名：SetPrimaryKey > 表元数据 > 默认 "id"
func (qb *Model) getPrimaryKey(ctx context.Context) string {
	if qb.primaryKey != "" {
		return qb.primaryKey
	}
	if qb.db == nil {
		return "id"
	}
	return qb.db.tablePrimaryKey(ctx, qb.table)
}

func (qb *Model) shouldUpdateByPk(pkValue interface{}) bool {
	if pkValue == nil {
		return false
	}
	switch v := pkValue.(type) {
	case int, int8, int16, int32, int64:
		return v != 0
	case uint, uint8, uint16, uint32, uint64:
		return v != 0
	case string:
		return v != ""
	default:
		return pkValue != nil
	}
}

func (qb *Model) SaveOrUpdate(ctx context.Context, data ...interface{}) *QueryResult {
	var dataMaps []map[string]interface{}
	var err error

	if qb.data != nil {
		reflectValue := reflect.ValueOf(qb.data)
		if reflectValue.Kind() == reflect.Slice {
			dataMaps, err = qb.convertToMaps(qb.data)
			if err != nil {
				return &QueryResult{data: nil, err: err, query: "", args: nil}
			}
		} else {
			dataMap, err := qb.convertToMap(qb.data)
			if err != nil {
				return &QueryResult{data: nil, err: err, query: "", args: nil}
			}
			dataMaps = []map[string]interface{}{dataMap}
		}
	} else if len(data) > 0 && data[0] != nil {
		reflectValue := reflect.ValueOf(data[0])
		if reflectValue.Kind() == reflect.Slice {
			dataMaps, err = qb.convertToMaps(data[0])
		} else {
			dataMaps, err = qb.convertToMaps(data)
		}
		if err != nil {
			return &QueryResult{data: nil, err: err, query: "", args: nil}
		}
	}

	if len(dataMaps) == 0 {
		return &QueryResult{data: nil, err: fmt.Errorf("no data to save"), query: "", args: nil}
	}

	pk := qb.getPrimaryKey(ctx)

	if len(dataMaps) == 1 {
		dataMap := dataMaps[0]
		pkValue, hasPk := dataMap[pk]

		if hasPk && qb.shouldUpdateByPk(pkValue) {
			newModel := &Model{
				db:         qb.db,
				table:      qb.table,
				primaryKey: pk,
			}
			return newModel.Where(pk, pkValue).Update(ctx, dataMap)
		}
		newModel := &Model{
			db:         qb.db,
			table:      qb.table,
			primaryKey: pk,
		}
		return newModel.Insert(ctx, dataMap)
	}

	var insertData []map[string]interface{}
	var updateData []map[string]interface{}

	for _, row := range dataMaps {
		pkValue, hasPk := row[pk]
		if hasPk && qb.shouldUpdateByPk(pkValue) {
			updateData = append(updateData, row)
		} else {
			insertData = append(insertData, row)
		}
	}

	var lastErr error
	var affectedRows int64
	var lastInsertId string

	if len(insertData) > 0 {
		qb.data = insertData
		result := qb.InsertAll(ctx)
		if result.err != nil {
			lastErr = result.err
		} else {
			if rows, ok := result.data.(int64); ok {
				affectedRows += rows
			}
			if result.lastId != "" {
				lastInsertId = result.lastId
			}
		}
	}

	if len(updateData) > 0 {
		for _, row := range updateData {
			pkValue := row[pk]
			newModel := &Model{
				db:         qb.db,
				table:      qb.table,
				primaryKey: pk,
			}
			result := newModel.Where(pk, pkValue).Update(ctx, row)
			if result.err != nil {
				lastErr = result.err
			} else {
				if rows, ok := result.data.(int64); ok {
					affectedRows += rows
				}
			}
		}
	}

	return &QueryResult{
		data:   affectedRows,
		err:    lastErr,
		lastId: lastInsertId,
	}
}
