package datacenter

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// FileKind 任务关联文件类型
type FileKind string

const (
	FileKindSource FileKind = "source"
	FileKindResult FileKind = "result"
	FileKindError  FileKind = "error"
)

// Backend 存储后端
type Backend string

const (
	BackendRedis Backend = "redis"
	BackendOSS   Backend = "oss"
)

// FileRef 写入 task 表 *_file_url 字段的 JSON（兼容后续 OSS 迁移）
type FileRef struct {
	Backend  Backend `json:"backend"`
	Key      string  `json:"key,omitempty"`
	URL      string  `json:"url,omitempty"`
	FileID   uint64  `json:"file_id,omitempty"`
	Filename string  `json:"filename,omitempty"`
	ExpireAt string  `json:"expire_at,omitempty"`
	Kind     FileKind `json:"kind,omitempty"`
}

// UploadParams OSS 上传参数（由 FileUploader 实现方调用 configcenter FilesService）
type UploadParams struct {
	BusinessID  uint64
	UserID      uint64
	Filename    string
	ContentType string
	Data        []byte
	Kind        FileKind
	StorageName string
}

// FileUploader 对象存储上传（admin-api 注入 configcenter FilesService；未配置时为 nil）
type FileUploader interface {
	Upload(ctx context.Context, p UploadParams) (FileRef, error)
}

// FileOpener 按 URL 拉取 OSS 文件（可选；未配置时仅支持 redis backend）
type FileOpener interface {
	OpenURL(ctx context.Context, url string) ([]byte, error)
}

// ObjectStore 任务文件读写（Redis 兜底 + OSS 优先）
type ObjectStore struct {
	Redis    *redis.Redis
	Settings Settings
	Upload   FileUploader
	Opener   FileOpener
}

// EncodeFileRef 序列化为 DB 字段值
func EncodeFileRef(ref FileRef) string {
	b, err := json.Marshal(ref)
	if err != nil {
		return ""
	}
	return string(b)
}

// ParseFileRef 解析 DB 字段；纯 URL 字符串视为 OSS
func ParseFileRef(raw string) (FileRef, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return FileRef{}, false
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return FileRef{Backend: BackendOSS, URL: raw}, true
	}
	var ref FileRef
	if err := json.Unmarshal([]byte(raw), &ref); err != nil {
		return FileRef{}, false
	}
	if ref.Backend == "" {
		if ref.URL != "" {
			ref.Backend = BackendOSS
		} else if ref.Key != "" {
			ref.Backend = BackendRedis
		}
	}
	return ref, ref.Backend != ""
}

// Save 保存任务文件；OSS 未启用或上传失败时回落 Redis
func (s *ObjectStore) Save(ctx context.Context, kind FileKind, taskID uint64, businessID, userID uint64, filename string, data []byte) (FileRef, error) {
	if len(data) == 0 {
		return FileRef{}, fmt.Errorf("文件内容为空")
	}
	expireAt := time.Now().Add(time.Duration(s.Settings.ResultExpireDays) * 24 * time.Hour).Format("2006-01-02 15:04:05")
	contentType := "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	if strings.HasSuffix(strings.ToLower(filename), ".csv") {
		contentType = "text/csv"
	}

	if s.Settings.OssEnabled && s.Upload != nil {
		ref, err := s.Upload.Upload(ctx, UploadParams{
			BusinessID:  businessID,
			UserID:      userID,
			Filename:    filename,
			ContentType: contentType,
			Data:        data,
			Kind:        kind,
			StorageName: s.Settings.OssStorageName,
		})
		if err == nil {
			ref.Kind = kind
			if ref.Filename == "" {
				ref.Filename = filename
			}
			if ref.ExpireAt == "" {
				ref.ExpireAt = expireAt
			}
			return ref, nil
		}
		logx.WithContext(ctx).Errorf("datacenter OSS upload fallback redis: %v", err)
	}

	if s.Redis == nil {
		return FileRef{}, fmt.Errorf("Redis 不可用且 OSS 未配置")
	}
	key := taskFileRedisKey(kind, taskID)
	ttl := s.Settings.ResultExpireDays * 86400
	if ttl <= 0 {
		ttl = ExportResultTTL
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	if err := s.Redis.Setex(key, encoded, ttl); err != nil {
		return FileRef{}, err
	}
	return FileRef{
		Backend:  BackendRedis,
		Key:      key,
		Filename: filename,
		ExpireAt: expireAt,
		Kind:     kind,
	}, nil
}

// Open 读取任务文件
func (s *ObjectStore) Open(ctx context.Context, ref FileRef) ([]byte, error) {
	if ref.Backend == BackendOSS {
		if ref.URL == "" {
			return nil, fmt.Errorf("OSS 文件 URL 为空")
		}
		if s.Opener != nil {
			return s.Opener.OpenURL(ctx, ref.URL)
		}
		return nil, fmt.Errorf("OSS 文件暂不支持直读，请配置对象存储或使用下载链接")
	}
	if s.Redis == nil {
		return nil, fmt.Errorf("Redis 不可用")
	}
	key := ref.Key
	if key == "" {
		return nil, fmt.Errorf("Redis 文件 key 为空")
	}
	val, err := s.Redis.Get(key)
	if err != nil || val == "" {
		return nil, fmt.Errorf("文件已过期或不存在")
	}
	return base64.StdEncoding.DecodeString(val)
}

// OpenFromField 从 task 表字段读取
func (s *ObjectStore) OpenFromField(ctx context.Context, raw string) ([]byte, string, error) {
	ref, ok := ParseFileRef(raw)
	if !ok {
		return nil, "", fmt.Errorf("文件引用无效")
	}
	data, err := s.Open(ctx, ref)
	if err != nil {
		return nil, "", err
	}
	name := ref.Filename
	if name == "" {
		name = "download.xlsx"
	}
	return data, name, nil
}

func taskFileRedisKey(kind FileKind, taskID uint64) string {
	switch kind {
	case FileKindSource:
		if taskID > 0 {
			return fmt.Sprintf("datacenter:import:source:%d", taskID)
		}
		return ""
	case FileKindError:
		return ImportErrorKeyPrefix + fmt.Sprintf("%d", taskID)
	default:
		if kind == "" {
			kind = FileKindResult
		}
		return ExportResultKeyPrefix + fmt.Sprintf("%d", taskID)
	}
}

// SaveToRedisKey 直接写入指定 Redis key（导入源文件 parse 阶段使用）
func (s *ObjectStore) SaveToRedisKey(key string, data []byte, ttlSec int) error {
	if s.Redis == nil {
		return fmt.Errorf("Redis 不可用")
	}
	if ttlSec <= 0 {
		ttlSec = ExportResultTTL
	}
	return s.Redis.Setex(key, base64.StdEncoding.EncodeToString(data), ttlSec)
}
