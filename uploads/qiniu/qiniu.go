package qiniu

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/dwrui/go-zero-admin-pkg/uploads/types"
	"github.com/qiniu/go-sdk/v7/auth/qbox"
	"github.com/qiniu/go-sdk/v7/storage"
)

type QiniuStorage struct {
	mac    *qbox.Mac
	bucket string
	domain string
	config *types.QiniuConfig
}

func NewQiniuStorage(config *types.QiniuConfig) (*QiniuStorage, error) {
	if config.AccessKey == "" {
		return nil, errors.New("qiniu access key is required")
	}
	if config.SecretKey == "" {
		return nil, errors.New("qiniu secret key is required")
	}
	if config.Bucket == "" {
		return nil, errors.New("qiniu bucket is required")
	}
	if config.Domain == "" {
		return nil, errors.New("qiniu domain is required")
	}

	mac := qbox.NewMac(config.AccessKey, config.SecretKey)

	return &QiniuStorage{
		mac:    mac,
		bucket: config.Bucket,
		domain: config.Domain,
		config: config,
	}, nil
}

func (q *QiniuStorage) Upload(reader io.Reader, destPath string, opts ...types.UploadOption) (*types.UploadResult, error) {
	options := &types.UploadOptions{}
	for _, opt := range opts {
		opt(options)
	}

	objectKey := q.normalizePath(destPath)

	putPolicy := storage.PutPolicy{
		Scope: q.bucket,
	}
	upToken := putPolicy.UploadToken(q.mac)

	cfg := q.getStorageConfig()
	formUploader := storage.NewFormUploader(cfg)

	var ret storage.PutRet
	putExtra := &storage.PutExtra{}
	if options.ContentType != "" {
		putExtra.MimeType = options.ContentType
	}

	err := formUploader.Put(context.Background(), &ret, upToken, objectKey, reader, -1, putExtra)
	if err != nil {
		return nil, fmt.Errorf("failed to upload object: %w", err)
	}

	return &types.UploadResult{
		Name:        filepath.Base(destPath),
		Path:        destPath,
		URL:         q.GetURL(destPath),
		ContentType: options.ContentType,
		ETag:        ret.Hash,
	}, nil
}

func (q *QiniuStorage) UploadFile(localPath string, destPath string, opts ...types.UploadOption) (*types.UploadResult, error) {
	options := &types.UploadOptions{}
	for _, opt := range opts {
		opt(options)
	}

	objectKey := q.normalizePath(destPath)

	putPolicy := storage.PutPolicy{
		Scope: q.bucket,
	}
	upToken := putPolicy.UploadToken(q.mac)

	cfg := q.getStorageConfig()
	formUploader := storage.NewFormUploader(cfg)

	var ret storage.PutRet
	putExtra := &storage.PutExtra{}
	if options.ContentType != "" {
		putExtra.MimeType = options.ContentType
	}

	err := formUploader.PutFile(context.Background(), &ret, upToken, objectKey, localPath, putExtra)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	return &types.UploadResult{
		Name:        filepath.Base(destPath),
		Path:        destPath,
		URL:         q.GetURL(destPath),
		ContentType: options.ContentType,
		ETag:        ret.Hash,
	}, nil
}

func (q *QiniuStorage) Download(path string) (io.ReadCloser, *types.FileInfo, error) {
	signedURL, err := q.GetSignedURL(path, time.Hour)
	if err != nil {
		return nil, nil, err
	}

	resp, err := http.Get(signedURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to download object: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, nil, fmt.Errorf("failed to download object: status code %d", resp.StatusCode)
	}

	info, err := q.GetInfo(path)
	if err != nil {
		resp.Body.Close()
		return nil, nil, err
	}

	return resp.Body, info, nil
}

func (q *QiniuStorage) DownloadToFile(path string, localPath string) error {
	signedURL, err := q.GetSignedURL(path, time.Hour)
	if err != nil {
		return err
	}

	resp, err := http.Get(signedURL)
	if err != nil {
		return fmt.Errorf("failed to download object: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download object: status code %d", resp.StatusCode)
	}

	return nil
}

func (q *QiniuStorage) Delete(path string) error {
	objectKey := q.normalizePath(path)

	cfg := q.getStorageConfig()
	bucketManager := storage.NewBucketManager(q.mac, cfg)

	err := bucketManager.Delete(q.bucket, objectKey)
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}

func (q *QiniuStorage) DeleteMulti(paths []string) error {
	cfg := q.getStorageConfig()
	bucketManager := storage.NewBucketManager(q.mac, cfg)

	deleteOps := make([]string, len(paths))
	for i, path := range paths {
		deleteOps[i] = storage.URIDelete(q.bucket, q.normalizePath(path))
	}

	_, err := bucketManager.Batch(deleteOps)
	if err != nil {
		return fmt.Errorf("failed to delete objects: %w", err)
	}

	return nil
}

func (q *QiniuStorage) Copy(srcPath string, destPath string, opts ...types.CopyOption) error {
	srcObjectKey := q.normalizePath(srcPath)
	destObjectKey := q.normalizePath(destPath)

	cfg := q.getStorageConfig()
	bucketManager := storage.NewBucketManager(q.mac, cfg)

	err := bucketManager.Copy(q.bucket, srcObjectKey, q.bucket, destObjectKey, true)
	if err != nil {
		return fmt.Errorf("failed to copy object: %w", err)
	}

	return nil
}

func (q *QiniuStorage) Move(srcPath string, destPath string) error {
	srcObjectKey := q.normalizePath(srcPath)
	destObjectKey := q.normalizePath(destPath)

	cfg := q.getStorageConfig()
	bucketManager := storage.NewBucketManager(q.mac, cfg)

	err := bucketManager.Move(q.bucket, srcObjectKey, q.bucket, destObjectKey, true)
	if err != nil {
		return fmt.Errorf("failed to move object: %w", err)
	}

	return nil
}

func (q *QiniuStorage) Exists(path string) (bool, error) {
	objectKey := q.normalizePath(path)

	cfg := q.getStorageConfig()
	bucketManager := storage.NewBucketManager(q.mac, cfg)

	_, err := bucketManager.Stat(q.bucket, objectKey)
	if err != nil {
		if strings.Contains(err.Error(), "no such file or directory") || strings.Contains(err.Error(), "612") {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (q *QiniuStorage) GetInfo(path string) (*types.FileInfo, error) {
	objectKey := q.normalizePath(path)

	cfg := q.getStorageConfig()
	bucketManager := storage.NewBucketManager(q.mac, cfg)

	fileInfo, err := bucketManager.Stat(q.bucket, objectKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get object info: %w", err)
	}

	return &types.FileInfo{
		Name:        filepath.Base(path),
		Path:        path,
		Size:        fileInfo.Fsize,
		IsDir:       false,
		ModTime:     time.Unix(fileInfo.PutTime/10000000, 0),
		ContentType: fileInfo.MimeType,
		ETag:        fileInfo.Hash,
		URL:         q.GetURL(path),
		StorageType: string(types.StorageQiniu),
	}, nil
}

func (q *QiniuStorage) List(prefix string, limit int, marker string) (*types.ListResult, error) {
	objectPrefix := q.normalizePath(prefix)

	cfg := q.getStorageConfig()
	bucketManager := storage.NewBucketManager(q.mac, cfg)

	entries, commonPrefixes, nextMarker, hasNext, err := bucketManager.ListFiles(q.bucket, objectPrefix, "", marker, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	var files []types.FileInfo
	for _, entry := range entries {
		files = append(files, types.FileInfo{
			Name:        filepath.Base(entry.Key),
			Path:        entry.Key,
			Size:        entry.Fsize,
			IsDir:       strings.HasSuffix(entry.Key, "/"),
			ModTime:     time.Unix(entry.PutTime/10000000, 0),
			ContentType: entry.MimeType,
			ETag:        entry.Hash,
			URL:         q.GetURL(entry.Key),
			StorageType: string(types.StorageQiniu),
		})
	}

	for _, prefix := range commonPrefixes {
		files = append(files, types.FileInfo{
			Name:        filepath.Base(prefix),
			Path:        prefix,
			IsDir:       true,
			StorageType: string(types.StorageQiniu),
		})
	}

	return &types.ListResult{
		Files:       files,
		NextMarker:  nextMarker,
		IsTruncated: hasNext,
	}, nil
}

func (q *QiniuStorage) CreateDir(path string) error {
	objectKey := q.normalizePath(path)
	if !strings.HasSuffix(objectKey, "/") {
		objectKey += "/"
	}

	putPolicy := storage.PutPolicy{
		Scope: q.bucket,
	}
	upToken := putPolicy.UploadToken(q.mac)

	cfg := q.getStorageConfig()
	formUploader := storage.NewFormUploader(cfg)

	var ret storage.PutRet
	err := formUploader.Put(context.Background(), &ret, upToken, objectKey, nil, 0, nil)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return nil
}

func (q *QiniuStorage) GetURL(path string) string {
	objectKey := q.normalizePath(path)
	return strings.TrimRight(q.domain, "/") + "/" + objectKey
}

func (q *QiniuStorage) GetSignedURL(path string, expire time.Duration) (string, error) {
	objectKey := q.normalizePath(path)
	deadline := time.Now().Add(expire).Unix()
	return storage.MakePrivateURL(q.mac, q.domain, objectKey, deadline), nil
}

func (q *QiniuStorage) GetType() types.StorageType {
	return types.StorageQiniu
}

func (q *QiniuStorage) getStorageConfig() *storage.Config {
	cfg := &storage.Config{
		UseHTTPS: true,
	}

	switch q.config.Region {
	case "z0":
		cfg.Zone = &storage.ZoneHuadong
	case "z1":
		cfg.Zone = &storage.ZoneHuabei
	case "z2":
		cfg.Zone = &storage.ZoneHuanan
	case "na0":
		cfg.Zone = &storage.ZoneBeimei
	case "as0":
		cfg.Zone = &storage.ZoneXinjiapo
	default:
		cfg.Zone = &storage.ZoneHuadong
	}

	return cfg
}

func (q *QiniuStorage) normalizePath(path string) string {
	return strings.TrimLeft(strings.TrimRight(path, "/"), "/")
}
