package openapiauth

import (
	"context"
	"time"
)

// Credential 开放接口凭证（验签用）
type Credential struct {
	ID          uint64
	BusinessID  uint64
	AccountID   uint64
	AccessKey   string
	SecretEnc   string
	SecretHash  string
	SignType    string
	Status      int64
	ExpireTime  *time.Time
	IPWhitelist string
	RateLimit   int64
}

// Store 凭证查询接口（可由 DB / Redis 实现）
type Store interface {
	FindByAccessKey(ctx context.Context, accessKey string) (*Credential, error)
	TouchLastUsed(ctx context.Context, id uint64) error
}

type ctxKey int

const (
	ctxCredentialID ctxKey = iota + 1
	ctxAccessKey
	ctxBusinessID
	ctxAccountID
)

func WithCredential(ctx context.Context, c *Credential) context.Context {
	ctx = context.WithValue(ctx, ctxCredentialID, c.ID)
	ctx = context.WithValue(ctx, ctxAccessKey, c.AccessKey)
	ctx = context.WithValue(ctx, ctxBusinessID, c.BusinessID)
	ctx = context.WithValue(ctx, ctxAccountID, c.AccountID)
	// 兼容业务代码常用 key
	ctx = context.WithValue(ctx, "business_id", c.BusinessID)
	ctx = context.WithValue(ctx, "user_id", c.AccountID)
	ctx = context.WithValue(ctx, "account_id", c.AccountID)
	return ctx
}

func CredentialIDFromContext(ctx context.Context) uint64 {
	v, _ := ctx.Value(ctxCredentialID).(uint64)
	return v
}

func AccessKeyFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxAccessKey).(string)
	return v
}

func BusinessIDFromContext(ctx context.Context) uint64 {
	if v, ok := ctx.Value(ctxBusinessID).(uint64); ok {
		return v
	}
	if v, ok := ctx.Value("business_id").(uint64); ok {
		return v
	}
	return 0
}

func AccountIDFromContext(ctx context.Context) uint64 {
	if v, ok := ctx.Value(ctxAccountID).(uint64); ok {
		return v
	}
	if v, ok := ctx.Value("account_id").(uint64); ok {
		return v
	}
	if v, ok := ctx.Value("user_id").(uint64); ok {
		return v
	}
	return 0
}
