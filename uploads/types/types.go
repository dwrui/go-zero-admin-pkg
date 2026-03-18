package types

import (
	"io"
	"time"
)

type StorageType string

const (
	StorageLocal   StorageType = "local"
	StorageAliyun  StorageType = "aliyun"
	StorageTencent StorageType = "tencent"
	StorageQiniu   StorageType = "qiniu"
)

type Config struct {
	Type StorageType `json:"type" yaml:"type"`

	Local   *LocalConfig   `json:"local,omitempty" yaml:"local,omitempty"`
	Aliyun  *AliyunConfig  `json:"aliyun,omitempty" yaml:"aliyun,omitempty"`
	Tencent *TencentConfig `json:"tencent,omitempty" yaml:"tencent,omitempty"`
	Qiniu   *QiniuConfig   `json:"qiniu,omitempty" yaml:"qiniu,omitempty"`
}

type LocalConfig struct {
	RootPath string `json:"root_path" yaml:"root_path"`
	BaseURL  string `json:"base_url" yaml:"base_url"`
}

type AliyunConfig struct {
	Endpoint        string `json:"endpoint" yaml:"endpoint"`
	AccessKeyID     string `json:"access_key_id" yaml:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret" yaml:"access_key_secret"`
	BucketName      string `json:"bucket_name" yaml:"bucket_name"`
	BaseURL         string `json:"base_url" yaml:"base_url"`
}

type TencentConfig struct {
	Region    string `json:"region" yaml:"region"`
	SecretID  string `json:"secret_id" yaml:"secret_id"`
	SecretKey string `json:"secret_key" yaml:"secret_key"`
	Bucket    string `json:"bucket" yaml:"bucket"`
	BaseURL   string `json:"base_url" yaml:"base_url"`
}

type QiniuConfig struct {
	AccessKey string `json:"access_key" yaml:"access_key"`
	SecretKey string `json:"secret_key" yaml:"secret_key"`
	Bucket    string `json:"bucket" yaml:"bucket"`
	Domain    string `json:"domain" yaml:"domain"`
	Region    string `json:"region" yaml:"region"`
}

type FileInfo struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Size        int64     `json:"size"`
	IsDir       bool      `json:"is_dir"`
	ModTime     time.Time `json:"mod_time"`
	ContentType string    `json:"content_type"`
	ETag        string    `json:"etag"`
	URL         string    `json:"url"`
	StorageType string    `json:"storage_type"`
}

type UploadResult struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	URL         string `json:"url"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
	ETag        string `json:"etag"`
}

type ListResult struct {
	Files       []FileInfo `json:"files"`
	NextMarker  string     `json:"next_marker"`
	IsTruncated bool       `json:"is_truncated"`
}

type UploadOption func(*UploadOptions)

type UploadOptions struct {
	ContentType  string
	ACL          string
	Metadata     map[string]string
	Overwrite    bool
	ProgressFunc func(readBytes int64, totalBytes int64)
}

func WithContentType(contentType string) UploadOption {
	return func(o *UploadOptions) {
		o.ContentType = contentType
	}
}

func WithACL(acl string) UploadOption {
	return func(o *UploadOptions) {
		o.ACL = acl
	}
}

func WithMetadata(metadata map[string]string) UploadOption {
	return func(o *UploadOptions) {
		o.Metadata = metadata
	}
}

func WithOverwrite(overwrite bool) UploadOption {
	return func(o *UploadOptions) {
		o.Overwrite = overwrite
	}
}

func WithProgress(fn func(readBytes int64, totalBytes int64)) UploadOption {
	return func(o *UploadOptions) {
		o.ProgressFunc = fn
	}
}

type CopyOption func(*CopyOptions)

type CopyOptions struct {
	Overwrite   bool
	Metadata    map[string]string
	ContentType string
}

func WithCopyOverwrite(overwrite bool) CopyOption {
	return func(o *CopyOptions) {
		o.Overwrite = overwrite
	}
}

func WithCopyMetadata(metadata map[string]string) CopyOption {
	return func(o *CopyOptions) {
		o.Metadata = metadata
	}
}

func WithCopyContentType(contentType string) CopyOption {
	return func(o *CopyOptions) {
		o.ContentType = contentType
	}
}

type Storage interface {
	Upload(reader io.Reader, destPath string, opts ...UploadOption) (*UploadResult, error)
	UploadFile(localPath string, destPath string, opts ...UploadOption) (*UploadResult, error)
	Download(path string) (io.ReadCloser, *FileInfo, error)
	DownloadToFile(path string, localPath string) error
	Delete(path string) error
	DeleteMulti(paths []string) error
	Copy(srcPath string, destPath string, opts ...CopyOption) error
	Move(srcPath string, destPath string) error
	Exists(path string) (bool, error)
	GetInfo(path string) (*FileInfo, error)
	List(prefix string, limit int, marker string) (*ListResult, error)
	CreateDir(path string) error
	GetURL(path string) string
	GetSignedURL(path string, expire time.Duration) (string, error)
	GetType() StorageType
}
