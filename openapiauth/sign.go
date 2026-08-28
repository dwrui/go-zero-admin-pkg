package openapiauth

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	HeaderAccessKey = "X-Access-Key"
	HeaderTimestamp = "X-Timestamp"
	HeaderSignature = "X-Signature"
)

// SignPayload 计算签名原文：timestamp\nMETHOD\npath\nquery\nbodySha256Hex
func SignPayload(timestamp, method, path, query, bodySHA256Hex string) string {
	return strings.Join([]string{
		timestamp,
		strings.ToUpper(method),
		path,
		query,
		bodySHA256Hex,
	}, "\n")
}

// BodySHA256Hex 计算 body 的 SHA256 hex；空 body 为固定空哈希
func BodySHA256Hex(body []byte) string {
	if len(body) == 0 {
		return "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// HMACSHA256Hex 使用 Secret 对原文做 HMAC-SHA256，返回 hex
func HMACSHA256Hex(secret, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = io.WriteString(mac, payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// BuildSignature 客户端/服务端共用：生成 X-Signature
func BuildSignature(secret, timestamp, method, path, query string, body []byte) string {
	payload := SignPayload(timestamp, method, path, query, BodySHA256Hex(body))
	return HMACSHA256Hex(secret, payload)
}

// VerifySignature 校验签名（恒定时间比较）
func VerifySignature(secret, timestamp, method, path, query string, body []byte, gotSignature string) bool {
	expect := BuildSignature(secret, timestamp, method, path, query, body)
	return hmac.Equal([]byte(strings.ToLower(expect)), []byte(strings.ToLower(strings.TrimSpace(gotSignature))))
}

// CheckTimestamp 校验时间窗（秒）
func CheckTimestamp(ts string, skewSec int64) error {
	if skewSec <= 0 {
		skewSec = 300
	}
	sec, err := strconv.ParseInt(strings.TrimSpace(ts), 10, 64)
	if err != nil {
		return fmt.Errorf("无效的时间戳")
	}
	now := time.Now().Unix()
	diff := now - sec
	if diff < 0 {
		diff = -diff
	}
	if diff > skewSec {
		return fmt.Errorf("请求已过期或时间不同步")
	}
	return nil
}

// ReadBodyBytes 读取并还原 Request.Body，供验签使用
func ReadBodyBytes(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	defer r.Body.Close()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	r.Body = io.NopCloser(bytes.NewReader(data))
	return data, nil
}
