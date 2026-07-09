package excel

import "strings"

// EnrichDicColumns 补全字典列 field_type / on_not_found（兼容旧模板 field_type 被误存为 text）
func EnrichDicColumns(mapping *ImportMapping) {
	if mapping == nil {
		return
	}
	for i := range mapping.Columns {
		col := &mapping.Columns[i]
		if col.DicGroupID > 0 && !strings.EqualFold(col.FieldType, "belongDic") {
			col.FieldType = "belongDic"
		}
		if strings.EqualFold(col.FieldType, "belongDic") || col.DicGroupID > 0 {
			col.FieldType = "belongDic"
			if col.OnNotFound == "" {
				col.OnNotFound = "error"
			}
			continue
		}
		if hasEnumOptionValue(col.OptionValue) && isPlainImportFieldType(col.FieldType) {
			col.FieldType = "radio"
		}
	}
}

func isPlainImportFieldType(fieldType string) bool {
	switch strings.ToLower(strings.TrimSpace(fieldType)) {
	case "", "text", "textarea", "dic":
		return true
	default:
		return false
	}
}

// MergeSystemImportColumns 租户模板缺失字典/关联元数据时，从系统模板补全
func MergeSystemImportColumns(tenant *ImportMapping, systemRaw string) error {
	if tenant == nil || strings.TrimSpace(systemRaw) == "" {
		return nil
	}
	sys, err := ParseImportMapping(systemRaw)
	if err != nil {
		return err
	}
	sysCol := make(map[string]ImportColumn, len(sys.Columns))
	for _, c := range sys.Columns {
		sysCol[c.DbField] = c
	}
	for i, tc := range tenant.Columns {
		sc, ok := sysCol[tc.DbField]
		if !ok {
			continue
		}
		if !sc.TenantEditable {
			tenant.Columns[i].FieldType = sc.FieldType
			tenant.Columns[i].RefTable = sc.RefTable
			tenant.Columns[i].RefMatchFields = sc.RefMatchFields
			tenant.Columns[i].RefDisplayField = sc.RefDisplayField
			tenant.Columns[i].OnNotFound = sc.OnNotFound
			tenant.Columns[i].DicGroupID = sc.DicGroupID
			tenant.Columns[i].RegionLevel = sc.RegionLevel
			tenant.Columns[i].ParentField = sc.ParentField
			if sc.OptionValue != "" {
				tenant.Columns[i].OptionValue = sc.OptionValue
			}
			continue
		}
		mergeSystemColumnMeta(&tenant.Columns[i], sc)
	}
	EnrichRegionColumns(tenant)
	EnrichDicColumns(tenant)
	return nil
}

func mergeSystemColumnMeta(tc *ImportColumn, sc ImportColumn) {
	if strings.TrimSpace(tc.OptionValue) == "" && sc.OptionValue != "" {
		tc.OptionValue = sc.OptionValue
	}
	if tc.DicGroupID <= 0 && sc.DicGroupID > 0 {
		tc.DicGroupID = sc.DicGroupID
	}
	if isEnumImportFieldType(sc.FieldType) && isPlainImportFieldType(tc.FieldType) {
		tc.FieldType = sc.FieldType
	}
	if sc.OnNotFound != "" && tc.OnNotFound == "" && isEnumImportFieldType(sc.FieldType) {
		tc.OnNotFound = sc.OnNotFound
	}
}

func isEnumImportFieldType(fieldType string) bool {
	switch normalizeColumnFieldType(fieldType) {
	case "radio", "select", "switch", "checkbox", "belongDic":
		return true
	default:
		return false
	}
}

func normalizeColumnFieldType(fieldType string) string {
	return normalizeImportFieldType(fieldType)
}
