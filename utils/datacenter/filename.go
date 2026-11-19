package datacenter

import (
	"fmt"
	"strings"
	"time"
)

// FilenameMeta 导出/错误文件命名占位符
type FilenameMeta struct {
	BizType   string
	BizName   string
	TplCode   string
	TplName   string
	TaskNo    string
	UserID    uint64
	UserName  string
	Ext       string
}

// DefaultExportFilenamePattern 与配置中心 export_filename_pattern 默认一致
const DefaultExportFilenamePattern = "{biz_type}_{tpl_code}_{date}_{task_no}"

// BuildFilename 按 pattern 生成文件名（未配置 pattern 时用默认）
func BuildFilename(pattern string, meta FilenameMeta) string {
	if strings.TrimSpace(pattern) == "" {
		pattern = DefaultExportFilenamePattern
	}
	ext := meta.Ext
	if ext == "" {
		ext = "xlsx"
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	date := time.Now().Format("20060102")
	datetime := time.Now().Format("20060102150405")
	replacer := strings.NewReplacer(
		"{biz_type}", safeName(meta.BizType),
		"{biz_name}", safeName(meta.BizName),
		"{tpl_code}", safeName(meta.TplCode),
		"{tpl_name}", safeName(meta.TplName),
		"{task_no}", safeName(meta.TaskNo),
		"{date}", date,
		"{datetime}", datetime,
		"{user_id}", fmt.Sprintf("%d", meta.UserID),
		"{user_name}", safeName(meta.UserName),
	)
	name := replacer.Replace(pattern)
	name = strings.Trim(name, " _-")
	if name == "" {
		name = safeName(meta.BizType) + "_" + date
	}
	if !strings.HasSuffix(strings.ToLower(name), ext) {
		name += ext
	}
	return name
}

func safeName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "file"
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		case r == '.':
			b.WriteRune('_')
		}
	}
	out := b.String()
	if out == "" {
		return "file"
	}
	return out
}
