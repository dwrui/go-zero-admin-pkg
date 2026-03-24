package aliyun

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/dwrui/go-zero-admin-pkg/uploads/types"
)

type AliyunStorage struct {
	client *oss.Client
	bucket *oss.Bucket
	config *types.AliyunConfig
}

func NewAliyunStorage(config *types.AliyunConfig) (*AliyunStorage, error) {
	if config.Endpoint == "" {
		return nil, errors.New("aliyun oss endpoint is required")
	}
	if config.AccessKeyID == "" {
		return nil, errors.New("aliyun oss access key id is required")
	}
	if config.AccessKeySecret == "" {
		return nil, errors.New("aliyun oss access key secret is required")
	}
	if config.BucketName == "" {
		return nil, errors.New("aliyun oss bucket name is required")
	}

	client, err := oss.New(config.Endpoint, config.AccessKeyID, config.AccessKeySecret)
	if err != nil {
		return nil, fmt.Errorf("failed to create aliyun oss client: %w", err)
	}

	bucket, err := client.Bucket(config.BucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket: %w", err)
	}

	return &AliyunStorage{
		client: client,
		bucket: bucket,
		config: config,
	}, nil
}

func (a *AliyunStorage) Upload(reader io.Reader, destPath string, opts ...types.UploadOption) (*types.UploadResult, error) {
	options := &types.UploadOptions{}
	for _, opt := range opts {
		opt(options)
	}

	objectKey := a.normalizePath(destPath)

	ossOptions := make([]oss.Option, 0)
	if options.ContentType != "" {
		ossOptions = append(ossOptions, oss.ContentType(options.ContentType))
	}
	if options.ACL != "" {
		ossOptions = append(ossOptions, oss.ObjectACL(oss.ACLType(options.ACL)))
	}
	for k, v := range options.Metadata {
		ossOptions = append(ossOptions, oss.Meta(k, v))
	}

	err := a.bucket.PutObject(objectKey, reader, ossOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to upload object: %w", err)
	}

	meta, err := a.bucket.GetObjectDetailedMeta(objectKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get object meta: %w", err)
	}

	size := int64(0)
	if contentLength := meta.Get("Content-Length"); contentLength != "" {
		fmt.Sscanf(contentLength, "%d", &size)
	}

	contentType := options.ContentType
	if contentType == "" {
		contentType = meta.Get("Content-Type")
	}

	return &types.UploadResult{
		Name:        filepath.Base(destPath),
		Path:        destPath,
		URL:         a.GetURL(destPath),
		Size:        size,
		ContentType: contentType,
		ETag:        meta.Get("ETag"),
	}, nil
}

func (a *AliyunStorage) UploadFile(localPath string, destPath string, opts ...types.UploadOption) (*types.UploadResult, error) {
	options := &types.UploadOptions{}
	for _, opt := range opts {
		opt(options)
	}

	objectKey := a.normalizePath(destPath)

	ossOptions := make([]oss.Option, 0)
	if options.ContentType != "" {
		ossOptions = append(ossOptions, oss.ContentType(options.ContentType))
	}

	err := a.bucket.PutObjectFromFile(objectKey, localPath, ossOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	meta, err := a.bucket.GetObjectDetailedMeta(objectKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get object meta: %w", err)
	}

	size := int64(0)
	if contentLength := meta.Get("Content-Length"); contentLength != "" {
		fmt.Sscanf(contentLength, "%d", &size)
	}

	contentType := options.ContentType
	if contentType == "" {
		contentType = meta.Get("Content-Type")
	}

	return &types.UploadResult{
		Name:        filepath.Base(destPath),
		Path:        destPath,
		URL:         a.GetURL(destPath),
		Size:        size,
		ContentType: contentType,
		ETag:        meta.Get("ETag"),
	}, nil
}

func (a *AliyunStorage) Download(path string) (io.ReadCloser, *types.FileInfo, error) {
	objectKey := a.normalizePath(path)

	reader, err := a.bucket.GetObject(objectKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to download object: %w", err)
	}

	info, err := a.GetInfo(path)
	if err != nil {
		reader.Close()
		return nil, nil, err
	}

	return reader, info, nil
}

func (a *AliyunStorage) DownloadToFile(path string, localPath string) error {
	objectKey := a.normalizePath(path)

	err := a.bucket.GetObjectToFile(objectKey, localPath)
	if err != nil {
		return fmt.Errorf("failed to download object to file: %w", err)
	}

	return nil
}

func (a *AliyunStorage) Delete(path string) error {
	objectKey := a.normalizePath(path)

	err := a.bucket.DeleteObject(objectKey)
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}

func (a *AliyunStorage) DeleteMulti(paths []string) error {
	objectKeys := make([]string, len(paths))
	for i, path := range paths {
		objectKeys[i] = a.normalizePath(path)
	}

	_, err := a.bucket.DeleteObjects(objectKeys)
	if err != nil {
		return fmt.Errorf("failed to delete objects: %w", err)
	}

	return nil
}

func (a *AliyunStorage) Copy(srcPath string, destPath string, opts ...types.CopyOption) error {
	srcObjectKey := a.normalizePath(srcPath)
	destObjectKey := a.normalizePath(destPath)

	_, err := a.bucket.CopyObject(srcObjectKey, destObjectKey)
	if err != nil {
		return fmt.Errorf("failed to copy object: %w", err)
	}

	return nil
}

func (a *AliyunStorage) Move(srcPath string, destPath string) error {
	if err := a.Copy(srcPath, destPath); err != nil {
		return err
	}

	return a.Delete(srcPath)
}

func (a *AliyunStorage) Exists(path string) (bool, error) {
	objectKey := a.normalizePath(path)

	_, err := a.bucket.GetObjectMeta(objectKey)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchKey") {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (a *AliyunStorage) GetInfo(path string) (*types.FileInfo, error) {
	objectKey := a.normalizePath(path)

	meta, err := a.bucket.GetObjectDetailedMeta(objectKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get object meta: %w", err)
	}

	size := int64(0)
	if contentLength := meta.Get("Content-Length"); contentLength != "" {
		fmt.Sscanf(contentLength, "%d", &size)
	}

	var modTime time.Time
	if lastModified := meta.Get("Last-Modified"); lastModified != "" {
		modTime, _ = http.ParseTime(lastModified)
	}

	return &types.FileInfo{
		Name:        filepath.Base(path),
		Path:        path,
		Size:        size,
		IsDir:       false,
		ModTime:     modTime,
		ContentType: meta.Get("Content-Type"),
		ETag:        meta.Get("ETag"),
		URL:         a.GetURL(path),
		StorageType: string(types.StorageAliyun),
	}, nil
}

func (a *AliyunStorage) List(prefix string, limit int, marker string) (*types.ListResult, error) {
	objectPrefix := a.normalizePath(prefix)

	options := []oss.Option{oss.Prefix(objectPrefix)}
	if limit > 0 {
		options = append(options, oss.MaxKeys(limit))
	}
	if marker != "" {
		options = append(options, oss.Marker(marker))
	}

	result, err := a.bucket.ListObjects(options...)
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	var files []types.FileInfo
	for _, object := range result.Objects {
		files = append(files, types.FileInfo{
			Name:        filepath.Base(object.Key),
			Path:        object.Key,
			Size:        object.Size,
			IsDir:       strings.HasSuffix(object.Key, "/"),
			ModTime:     object.LastModified,
			ContentType: object.Type,
			ETag:        object.ETag,
			URL:         a.GetURL(object.Key),
			StorageType: string(types.StorageAliyun),
		})
	}

	return &types.ListResult{
		Files:       files,
		NextMarker:  result.NextMarker,
		IsTruncated: result.IsTruncated,
	}, nil
}

func (a *AliyunStorage) CreateDir(path string) error {
	objectKey := a.normalizePath(path)
	if !strings.HasSuffix(objectKey, "/") {
		objectKey += "/"
	}

	err := a.bucket.PutObject(objectKey, nil)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return nil
}

func (a *AliyunStorage) GetURL(path string) string {
	objectKey := a.normalizePath(path)
	if a.config.BaseURL != "" {
		return strings.TrimRight(a.config.BaseURL, "/") + "/" + objectKey
	}
	return fmt.Sprintf("https://%s.%s/%s", a.config.BucketName, a.config.Endpoint, objectKey)
}

func (a *AliyunStorage) GetSignedURL(path string, expire time.Duration) (string, error) {
	objectKey := a.normalizePath(path)
	return a.bucket.SignURL(objectKey, oss.HTTPGet, int64(expire.Seconds()))
}

func (a *AliyunStorage) GetType() types.StorageType {
	return types.StorageAliyun
}

func (a *AliyunStorage) normalizePath(path string) string {
	return strings.TrimLeft(strings.TrimRight(path, "/"), "/")
}
