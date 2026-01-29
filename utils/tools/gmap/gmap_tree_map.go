package gmap

import (
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gtree"
)

// 基于红黑树的树状图，RedBlackTree的别名。
type TreeMap = gtree.RedBlackTree

// NewTreeMap 实例化一个基于自定义比较器的树状图。
// 参数 `safe` 用于指定是否在并发安全模式下使用树状图，默认值为 false。
func NewTreeMap(comparator func(v1, v2 interface{}) int, safe ...bool) *TreeMap {
	return gtree.NewRedBlackTree(comparator, safe...)
}

// NewTreeMapFrom 实例化一个基于自定义比较器的树状图，并使用 `data` 映射初始化树状图。
// 注意，参数 `data` 映射将被设置为底层数据映射（无深拷贝），
// 因此在外部修改映射时可能会存在并发安全问题。
// 参数 `safe` 用于指定是否在并发安全模式下使用树状图，默认值为 false。
func NewTreeMapFrom(comparator func(v1, v2 interface{}) int, data map[interface{}]interface{}, safe ...bool) *TreeMap {
	return gtree.NewRedBlackTreeFrom(comparator, data, safe...)
}
