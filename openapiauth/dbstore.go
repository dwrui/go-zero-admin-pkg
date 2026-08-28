package openapiauth

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/dwrui/go-zero-admin-pkg/utils/db"
	"github.com/dwrui/go-zero-admin-pkg/utils/ga"
)

// DBStore 基于项目 ORM 的凭证仓库（表：open_api_credential → ga_open_api_credential）
type DBStore struct {
	DB *db.DBManager
}

type credentialRow struct {
	Id          uint64       `db:"id"`
	BusinessId  uint64       `db:"business_id"`
	AccountId   uint64       `db:"account_id"`
	AccessKey   string       `db:"access_key"`
	SecretEnc   string       `db:"secret_enc"`
	SecretHash  string       `db:"secret_hash"`
	SignType    string       `db:"sign_type"`
	Status      int64        `db:"status"`
	ExpireTime  sql.NullTime `db:"expire_time"`
	IpWhitelist string       `db:"ip_whitelist"`
	RateLimit   int64        `db:"rate_limit"`
	DeleteTime  int64        `db:"delete_time"`
}

func (s *DBStore) FindByAccessKey(ctx context.Context, accessKey string) (*Credential, error) {
	if s == nil || s.DB == nil {
		return nil, errors.New("db not ready")
	}
	var row credentialRow
	resp := s.DB.Model("open_api_credential").
		Where("access_key = ? AND delete_time = 0", accessKey).
		Find(ctx, &row)
	if resp.GetError() != nil {
		return nil, resp.GetError()
	}
	if resp.IsEmpty() || row.Id == 0 {
		return nil, errors.New("not found")
	}
	c := &Credential{
		ID:          row.Id,
		BusinessID:  row.BusinessId,
		AccountID:   row.AccountId,
		AccessKey:   row.AccessKey,
		SecretEnc:   row.SecretEnc,
		SecretHash:  row.SecretHash,
		SignType:    row.SignType,
		Status:      row.Status,
		IPWhitelist: row.IpWhitelist,
		RateLimit:   row.RateLimit,
	}
	if row.ExpireTime.Valid {
		t := row.ExpireTime.Time
		c.ExpireTime = &t
	}
	return c, nil
}

func (s *DBStore) TouchLastUsed(ctx context.Context, id uint64) error {
	if s == nil || s.DB == nil || id == 0 {
		return nil
	}
	_ = s.DB.Model("open_api_credential").Where("id = ?", id).Update(ctx, ga.Map{
		"last_used_at": time.Now().Format("2006-01-02 15:04:05"),
	})
	return nil
}
