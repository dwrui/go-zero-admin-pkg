package ga

import (
	"context"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/empty"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gcache"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gmd5"
	"time"
)

var (
	cache = gcache.New()
)

// Md5 encryption
func Md5(str string) string {
	mdsecret, _ := gmd5.Encrypt(str)
	return mdsecret
}

// md5hex编码加密
func Md5Str(origin string) string {
	return gmd5.Md5StrHex(origin)
}

// 把验证码保存在本地，用GetVerifyCode获取key对应缓存
func SetVerifyCode(key, code string) (err error) {
	ctx := context.Background()
	err = cache.Set(ctx, key, code, time.Second*60)
	return
}

// 获取本地保存的验证码，使用SetVerifyCode保存可以对应数据
func GetVerifyCode(key string) (code int, err error) {
	ctx := context.Background()
	val, err := cache.Get(ctx, key)
	if err == nil {
		code = Int(val)
	}
	return
}

// IsNil checks whether given `value` is nil.
func IsNil(value interface{}, traceSource ...bool) bool {
	return empty.IsNil(value, traceSource...)
}

// IsEmpty checks whether given `value` empty.
// It returns true if `value` is in: 0, nil, false, "", len(slice/map/chan) == 0.
// Or else it returns true.
func IsEmpty(value interface{}, traceSource ...bool) bool {
	return empty.IsEmpty(value, traceSource...)
}
