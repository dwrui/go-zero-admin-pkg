package gutil

import (
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/deepcopy"
)

// Copy 返回 v 的一个深度拷贝。
//
// Copy 无法复制结构体中未导出的字段（字段名为小写）。
// 未导出的字段无法被Go运行时反射，因此
// 他们无法执行任何数据复制。
func Copy(src interface{}) (dst interface{}) {
	return deepcopy.Copy(src)
}
