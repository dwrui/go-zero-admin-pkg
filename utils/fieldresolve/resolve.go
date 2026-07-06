package fieldresolve

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/dwrui/go-zero-admin-pkg/utils/db"
	"github.com/dwrui/go-zero-admin-pkg/utils/ga"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

const (
	cacheTTLTableSec = 300
	cacheTTLDicSec   = 600
	dicDataTable     = "common_dictionary_data"
)

// DicItem 字典项展示结构（列表 dic 槽、详情名称等）
type DicItem struct {
	Keyname  string `json:"keyname"`
	Tagcolor string `json:"tagcolor"`
	Keyvalue string `json:"keyvalue,omitempty"`
}

type dicDataRow struct {
	Keyname  string `db:"keyname"`
	Tagcolor string `db:"tagcolor"`
	Keyvalue string `db:"keyvalue"`
}

func tableCacheKey(tableName, fieldName string, id int64) string {
	return fmt.Sprintf("fieldresolve:table:%s:%d:%s", tableName, id, fieldName)
}

func dicCacheKey(groupId int64, keyvalue string) string {
	return fmt.Sprintf("fieldresolve:dic:%d:%s", groupId, keyvalue)
}

func cacheGet(rds *redis.Redis, key string) (string, bool) {
	if rds == nil || key == "" {
		return "", false
	}
	val, err := rds.Get(key)
	return val, err == nil && val != ""
}

func cacheSet(rds *redis.Redis, key, val string, ttlSec int) {
	if rds == nil || key == "" || val == "" {
		return
	}
	_ = rds.Setex(key, val, ttlSec)
}

// GetTableFieldVal 按主键解析关联表显示字段（Redis 缓存）
func GetTableFieldVal(ctx context.Context, dbm *db.DBManager, rds *redis.Redis, tableName, fieldName string, id int64) string {
	if dbm == nil || id <= 0 || tableName == "" || fieldName == "" {
		return ""
	}
	key := tableCacheKey(tableName, fieldName, id)
	if cached, ok := cacheGet(rds, key); ok {
		return cached
	}
	resp := dbm.Model(tableName).Where("id", id).Value(ctx, fieldName)
	if resp.GetError() != nil || resp.IsEmpty() {
		return ""
	}
	val := ga.String(resp.GetData())
	cacheSet(rds, key, val, cacheTTLTableSec)
	return val
}

// BatchGetTableFieldVal 批量解析关联表字段，减少 N+1 查询
func BatchGetTableFieldVal(ctx context.Context, dbm *db.DBManager, rds *redis.Redis, tableName, fieldName string, ids []int64) map[int64]string {
	out := make(map[int64]string)
	if dbm == nil || tableName == "" || fieldName == "" || len(ids) == 0 {
		return out
	}
	uniq := make([]int64, 0, len(ids))
	seen := map[int64]struct{}{}
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		if cached, ok := cacheGet(rds, tableCacheKey(tableName, fieldName, id)); ok {
			out[id] = cached
			continue
		}
		uniq = append(uniq, id)
	}
	if len(uniq) == 0 {
		return out
	}
	type row struct {
		ID    int64  `db:"id"`
		Label string `db:"label"`
	}
	var rows []row
	resp := dbm.Model(tableName).
		WhereIn("id", uniq).
		Fields("id," + fieldName + " as label").
		Select(ctx, &rows)
	if resp.GetError() != nil {
		return out
	}
	for _, r := range rows {
		if r.ID <= 0 {
			continue
		}
		out[r.ID] = r.Label
		cacheSet(rds, tableCacheKey(tableName, fieldName, r.ID), r.Label, cacheTTLTableSec)
	}
	return out
}

func loadDicItem(ctx context.Context, dbm *db.DBManager, groupId int64, keyvalue string) (DicItem, bool) {
	if dbm == nil || groupId <= 0 || keyvalue == "" {
		return DicItem{}, false
	}
	var row dicDataRow
	resp := dbm.Model(dicDataTable).
		Where("group_id", groupId).
		Where("keyvalue", keyvalue).
		Where("status", 1).
		Fields("keyname,tagcolor,keyvalue").
		Find(ctx, &row)
	if resp.GetError() != nil || row.Keyvalue == "" {
		return DicItem{}, false
	}
	return DicItem{
		Keyname:  row.Keyname,
		Tagcolor: row.Tagcolor,
		Keyvalue: row.Keyvalue,
	}, true
}

// GetDicFieldVal 按字典分组 + 字典值解析展示项（Redis 缓存）
func GetDicFieldVal(ctx context.Context, dbm *db.DBManager, rds *redis.Redis, groupId int64, keyvalue string) DicItem {
	keyvalue = strings.TrimSpace(keyvalue)
	if keyvalue == "" || groupId <= 0 {
		return DicItem{}
	}
	key := dicCacheKey(groupId, keyvalue)
	if cached, ok := cacheGet(rds, key); ok {
		var item DicItem
		if json.Unmarshal([]byte(cached), &item) == nil {
			return item
		}
	}
	item, ok := loadDicItem(ctx, dbm, groupId, keyvalue)
	if !ok {
		return DicItem{Keyvalue: keyvalue}
	}
	if raw, err := json.Marshal(item); err == nil {
		cacheSet(rds, key, string(raw), cacheTTLDicSec)
	}
	return item
}

// GetDicFieldName 仅返回字典显示名
func GetDicFieldName(ctx context.Context, dbm *db.DBManager, rds *redis.Redis, groupId int64, keyvalue string) string {
	return GetDicFieldVal(ctx, dbm, rds, groupId, keyvalue).Keyname
}

// BatchGetDicFieldVal 批量解析字典项
func BatchGetDicFieldVal(ctx context.Context, dbm *db.DBManager, rds *redis.Redis, groupId int64, keyvalues []string) map[string]DicItem {
	out := make(map[string]DicItem)
	if dbm == nil || groupId <= 0 || len(keyvalues) == 0 {
		return out
	}
	miss := make([]string, 0, len(keyvalues))
	seen := map[string]struct{}{}
	for _, kv := range keyvalues {
		kv = strings.TrimSpace(kv)
		if kv == "" {
			continue
		}
		if _, ok := seen[kv]; ok {
			continue
		}
		seen[kv] = struct{}{}
		if cached, ok := cacheGet(rds, dicCacheKey(groupId, kv)); ok {
			var item DicItem
			if json.Unmarshal([]byte(cached), &item) == nil {
				out[kv] = item
				continue
			}
		}
		miss = append(miss, kv)
	}
	if len(miss) == 0 {
		return out
	}
	var rows []dicDataRow
	resp := dbm.Model(dicDataTable).
		Where("group_id", groupId).
		WhereIn("keyvalue", miss).
		Where("status", 1).
		Fields("keyname,tagcolor,keyvalue").
		Select(ctx, &rows)
	if resp.GetError() != nil {
		return out
	}
	found := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		if row.Keyvalue == "" {
			continue
		}
		item := DicItem{
			Keyname:  row.Keyname,
			Tagcolor: row.Tagcolor,
			Keyvalue: row.Keyvalue,
		}
		out[row.Keyvalue] = item
		found[row.Keyvalue] = struct{}{}
		if raw, err := json.Marshal(item); err == nil {
			cacheSet(rds, dicCacheKey(groupId, row.Keyvalue), string(raw), cacheTTLDicSec)
		}
	}
	for _, kv := range miss {
		if _, ok := found[kv]; !ok {
			out[kv] = DicItem{Keyvalue: kv}
		}
	}
	return out
}

// FormatDicKeyvalue 将模型字段值格式化为字典 keyvalue 查询串
func FormatDicKeyvalue(v interface{}) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(x)
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case float64:
		if x == float64(int64(x)) {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	default:
		return strings.TrimSpace(fmt.Sprint(x))
	}
}
