// Package gmd5 提供了用于 MD5 加密算法的有用 API。
package gmd5

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gconv"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gerror"
	"io"
	"os"
)

// Encrypt 加密任意类型的变量使用 MD5 算法。
// 它使用 gconv 包将 `v` 转换为其字节类型。
func Encrypt(data interface{}) (encrypt string, err error) {
	return EncryptBytes(gconv.Bytes(data))
}

// md5加密
func Md5StrHex(origin string) string {
	m := md5.New()
	m.Write([]byte(origin))
	return hex.EncodeToString(m.Sum(nil))
}

// MustEncrypt 加密任意类型的变量使用 MD5 算法。
// 它使用 gconv 包将 `v` 转换为其字节类型。
// 如果发生任何错误，它会 panic。
func MustEncrypt(data interface{}) string {
	result, err := Encrypt(data)
	if err != nil {
		panic(err)
	}
	return result
}

// EncryptBytes 加密 `data` 使用 MD5 算法。
func EncryptBytes(data []byte) (encrypt string, err error) {
	h := md5.New()
	if _, err = h.Write(data); err != nil {
		err = gerror.Wrap(err, `hash.Write failed`)
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// MustEncryptBytes 加密 `data` 使用 MD5 算法。
// 如果发生任何错误，它会 panic。
func MustEncryptBytes(data []byte) string {
	result, err := EncryptBytes(data)
	if err != nil {
		panic(err)
	}
	return result
}

// EncryptString 加密字符串 `data` 使用 MD5 算法。
func EncryptString(data string) (encrypt string, err error) {
	return EncryptBytes([]byte(data))
}

// MustEncryptString 加密字符串 `data` 使用 MD5 算法。
// 如果发生任何错误，它会 panic。
func MustEncryptString(data string) string {
	result, err := EncryptString(data)
	if err != nil {
		panic(err)
	}
	return result
}

// EncryptFile 加密文件 `path` 的内容使用 MD5 算法。
func EncryptFile(path string) (encrypt string, err error) {
	f, err := os.Open(path)
	if err != nil {
		err = gerror.Wrapf(err, `os.Open failed for name "%s"`, path)
		return "", err
	}
	defer f.Close()
	h := md5.New()
	_, err = io.Copy(h, f)
	if err != nil {
		err = gerror.Wrap(err, `io.Copy failed`)
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// MustEncryptFile 加密文件 `path` 的内容使用 MD5 算法。
// 如果发生任何错误，它会 panic。
func MustEncryptFile(path string) string {
	result, err := EncryptFile(path)
	if err != nil {
		panic(err)
	}
	return result
}
