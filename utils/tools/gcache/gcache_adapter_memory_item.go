package gcache

import (
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gtime"
)

// IsExpired 检查 `item` 是否过期。
func (item *memoryDataItem) IsExpired() bool {
	// Note that it should use greater than or equal judgement here
	// imagining that the cache time is only 1 millisecond.
	return item.e < gtime.TimestampMilli()
}
