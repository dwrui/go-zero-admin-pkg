package excel

import "strings"

const (
	DefaultRegionTable   = "ga_base_region"
	RegionRelKeyField    = "region_code"
	DefaultRegionDisplay = "name"
)

// IsRegionFieldType 是否省市区字段类型
func IsRegionFieldType(fieldType string) bool {
	switch strings.TrimSpace(fieldType) {
	case "region", "region_province", "region_city", "region_area":
		return true
	default:
		return false
	}
}

// RegionLevelFromFieldType 1=省 2=市 3=区/县
func RegionLevelFromFieldType(fieldType string) int {
	switch strings.TrimSpace(fieldType) {
	case "region_city":
		return 2
	case "region_area":
		return 3
	default:
		return 1
	}
}

// RegionLevelFromDbField 根据数据库字段名推断省市区层级
func RegionLevelFromDbField(dbField string) int {
	f := strings.ToLower(strings.TrimSpace(dbField))
	if f == "" {
		return 0
	}
	if strings.Contains(f, "area") || strings.Contains(f, "district") || strings.Contains(f, "county") {
		return 3
	}
	if strings.Contains(f, "city") {
		return 2
	}
	if strings.Contains(f, "province") {
		return 1
	}
	return 0
}

// EffectiveRegionLevel 模板列实际使用的地区层级（优先 field_type / region_level）
func EffectiveRegionLevel(col ImportColumn) int {
	if col.RegionLevel > 0 {
		return col.RegionLevel
	}
	ft := strings.TrimSpace(col.FieldType)
	switch ft {
	case "region_city":
		return 2
	case "region_area":
		return 3
	case "region_province":
		return 1
	case "region":
		if lvl := RegionLevelFromDbField(col.DbField); lvl > 0 {
			return lvl
		}
		return 1
	}
	if IsRegionFieldType(ft) {
		return RegionLevelFromFieldType(ft)
	}
	return 0
}

// EnrichRegionColumns 补全旧模板中缺失的地区解析元数据
func EnrichRegionColumns(mapping *ImportMapping) {
	if mapping == nil {
		return
	}
	for i := range mapping.Columns {
		col := &mapping.Columns[i]
		if !IsRegionFieldType(col.FieldType) {
			continue
		}
		if col.RefTable == "" {
			col.RefTable = DefaultRegionTable
		}
		if col.RefDisplayField == "" {
			col.RefDisplayField = DefaultRegionDisplay
		}
		if col.RegionLevel == 0 {
			col.RegionLevel = EffectiveRegionLevel(*col)
		}
		if col.ParentField == "" {
			col.ParentField = inferRegionParentField(*col, mapping.Columns)
		}
		if col.OnNotFound == "" {
			col.OnNotFound = "error"
		}
	}
}

func inferRegionParentField(col ImportColumn, cols []ImportColumn) string {
	level := EffectiveRegionLevel(col)
	if level <= 1 {
		return ""
	}
	wantParentLevel := level - 1
	for _, c := range cols {
		if !IsRegionFieldType(c.FieldType) {
			continue
		}
		if EffectiveRegionLevel(c) == wantParentLevel {
			return c.DbField
		}
	}
	return ""
}
