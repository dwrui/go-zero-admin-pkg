package openapiauth

import (
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/dwrui/go-zero-admin-pkg/utils/ga"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// Config 验签中间件配置
type Config struct {
	MasterKey     string // 解密 secret_enc
	TimestampSkew int64  // 秒，默认 300
	SkipPaths     []string
}

// Middleware 返回 go-zero 可用的 HTTP 中间件
func Middleware(store Store, cfg Config) func(http.HandlerFunc) http.HandlerFunc {
	if cfg.TimestampSkew <= 0 {
		cfg.TimestampSkew = 300
	}
	skip := map[string]bool{}
	for _, p := range cfg.SkipPaths {
		skip[p] = true
	}
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if skip[path] || strings.HasSuffix(path, "/healthz") || strings.HasSuffix(path, "/readyz") {
				next(w, r)
				return
			}

			accessKey := strings.TrimSpace(r.Header.Get(HeaderAccessKey))
			timestamp := strings.TrimSpace(r.Header.Get(HeaderTimestamp))
			signature := strings.TrimSpace(r.Header.Get(HeaderSignature))
			if accessKey == "" || timestamp == "" || signature == "" {
				fail(w, r, "缺少签名请求头(X-Access-Key/X-Timestamp/X-Signature)")
				return
			}
			if err := CheckTimestamp(timestamp, cfg.TimestampSkew); err != nil {
				fail(w, r, err.Error())
				return
			}

			if store == nil {
				fail(w, r, "验签服务未就绪")
				return
			}
			cred, err := store.FindByAccessKey(r.Context(), accessKey)
			if err != nil || cred == nil {
				fail(w, r, "无效的 AccessKey")
				return
			}
			if cred.Status != 1 {
				fail(w, r, "密钥已禁用")
				return
			}
			if cred.ExpireTime != nil && time.Now().After(*cred.ExpireTime) {
				fail(w, r, "密钥已过期")
				return
			}
			if !ipAllowed(clientIP(r), cred.IPWhitelist) {
				fail(w, r, "IP 不在白名单")
				return
			}
			if cred.SecretEnc == "" {
				fail(w, r, "密钥未就绪，请重置 Secret")
				return
			}
			secret, err := DecryptSecret(cfg.MasterKey, cred.SecretEnc)
			if err != nil {
				fail(w, r, "密钥解密失败")
				return
			}

			body, err := ReadBodyBytes(r)
			if err != nil {
				fail(w, r, "读取请求体失败")
				return
			}
			query := r.URL.RawQuery
			if !VerifySignature(secret, timestamp, r.Method, path, query, body, signature) {
				fail(w, r, "签名校验失败")
				return
			}

			_ = store.TouchLastUsed(r.Context(), cred.ID)
			next(w, r.WithContext(WithCredential(r.Context(), cred)))
		}
	}
}

func fail(w http.ResponseWriter, r *http.Request, msg string) {
	httpx.WriteJsonCtx(r.Context(), w, http.StatusOK, ga.Failed().SetMsg(msg).SetCode(401))
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xr := r.Header.Get("X-Real-IP"); xr != "" {
		return strings.TrimSpace(xr)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func ipAllowed(ip, whitelist string) bool {
	whitelist = strings.TrimSpace(whitelist)
	if whitelist == "" {
		return true
	}
	for _, item := range strings.Split(whitelist, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if item == ip {
			return true
		}
		if strings.Contains(item, "/") {
			if _, cidr, err := net.ParseCIDR(item); err == nil {
				if parsed := net.ParseIP(ip); parsed != nil && cidr.Contains(parsed) {
					return true
				}
			}
		}
	}
	return false
}
