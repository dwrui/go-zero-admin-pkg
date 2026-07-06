package excel

import "errors"

// ErrImportRefSkip 关联未找到且 on_not_found=skip 时跳过该字段
var ErrImportRefSkip = errors.New("import ref skip")

// BelongToResolver belongto 关联查表转 id
type BelongToResolver interface {
	ResolveBelongTo(col ImportColumn, displayValue string) (int64, error)
}

// DicResolver belongDic 字典项解析（keyname/keyvalue → 存库 keyvalue）
type DicResolver interface {
	ResolveBelongDic(col ImportColumn, displayValue string) (string, error)
}
