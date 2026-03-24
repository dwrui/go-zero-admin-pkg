package tencent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/dwrui/go-zero-admin-pkg/uploads/types"
	"github.com/tencentyun/cos-go-sdk-v5"
)

type TencentStorage struct {
	client *cos.Client
	config *types.TencentConfig
}

func NewTencentStorage(config *types.TencentConfig) (*TencentStorage, error) {
	if config.Region == "" {
		return nil, errors.New("tencent cos region is required")
	}
	if config.SecretID == "" {
		return nil, errors.New("tencent cos secret id is required")
	}
	if config.SecretKey == "" {
		return nil, errors.New("tencent cos secret key is required")
	}
	if config.Bucket == "" {
		return nil, errors.New("tencent cos bucket is required")
	}

	bucketURL, err := url.Parse(fmt.Sprintf("https://%s.cos.%s.myqcloud.com", config.Bucket, config.Region))
	if err != nil {
		return nil, fmt.Errorf("failed to parse bucket url: %w", err)
	}

	client := cos.NewClient(&cos.BaseURL{BucketURL: bucketURL}, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  config.SecretID,
			SecretKey: config.SecretKey,
		},
	})

	return &TencentStorage{
		client: client,
		config: config,
	}, nil
}

func (t *TencentStorage) Upload(reader io.Reader, destPath string, opts ...types.UploadOption) (*types.UploadResult, error) {
	options := &types.UploadOptions{}
	for _, opt := range opts {
		opt(options)
	}

	objectKey := t.normalizePath(destPath)

	optHeader := &cos.ObjectPutHeaderOptions{}
	if options.ContentType != "" {
		optHeader.ContentType = options.ContentType
	}

	optPut := &cos.ObjectPutOptions{
		ObjectPutHeaderOptions: optHeader,
	}

	resp, err := t.client.Object.Put(context.Background(), objectKey, reader, optPut)
	if err != nil {
		return nil, fmt.Errorf("failed to upload object: %w", err)
	}

	return &types.UploadResult{
		Name:        filepath.Base(destPath),
		Path:        destPath,
		URL:         t.GetURL(destPath),
		ContentType: options.ContentType,
		ETag:        resp.Header.Get("ETag"),
	}, nil
}

func (t *TencentStorage) UploadFile(localPath string, destPath string, opts ...types.UploadOption) (*types.UploadResult, error) {
	options := &types.UploadOptions{}
	for _, opt := range opts {
		opt(options)
	}

	objectKey := t.normalizePath(destPath)

	optHeader := &cos.ObjectPutHeaderOptions{}
	if options.ContentType != "" {
		optHeader.ContentType = options.ContentType
	}

	optPut := &cos.ObjectPutOptions{
		ObjectPutHeaderOptions: optHeader,
	}

	resp, err := t.client.Object.PutFromFile(context.Background(), objectKey, localPath, optPut)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	return &types.UploadResult{
		Name:        filepath.Base(destPath),
		Path:        destPath,
		URL:         t.GetURL(destPath),
		ContentType: options.ContentType,
		ETag:        resp.Header.Get("ETag"),
	}, nil
}

func (t *TencentStorage) Download(path string) (io.ReadCloser, *types.FileInfo, error) {
	objectKey := t.normalizePath(path)

	resp, err := t.client.Object.Get(context.Background(), objectKey, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to download object: %w", err)
	}

	info, err := t.GetInfo(path)
	if err != nil {
		resp.Body.Close()
		return nil, nil, err
	}

	return resp.Body, info, nil
}

func (t *TencentStorage) DownloadToFile(path string, localPath string) error {
	objectKey := t.normalizePath(path)

	_, err := t.client.Object.GetToFile(context.Background(), objectKey, localPath, nil)
	if err != nil {
		return fmt.Errorf("failed to download object to file: %w", err)
	}

	return nil
}

func (t *TencentStorage) Delete(path string) error {
	objectKey := t.normalizePath(path)

	_, err := t.client.Object.Delete(context.Background(), objectKey)
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}

func (t *TencentStorage) DeleteMulti(paths []string) error {
	objects := make([]cos.Object, len(paths))
	for i, path := range paths {
		objects[i] = cos.Object{Key: t.normalizePath(path)}
	}

	_, _, err := t.client.Object.DeleteMulti(context.Background(), &cos.ObjectDeleteMultiOptions{
		Objects: objects,
		Quiet:   true,
	})
	if err != nil {
		return fmt.Errorf("failed to delete objects: %w", err)
	}

	return nil
}

func (t *TencentStorage) Copy(srcPath string, destPath string, opts ...types.CopyOption) error {
	srcObjectKey := t.normalizePath(srcPath)
	destObjectKey := t.normalizePath(destPath)

	srcURL := fmt.Sprintf("%s/%s", t.config.Bucket, srcObjectKey)

	_, _, err := t.client.Object.Copy(context.Background(), destObjectKey, srcURL, nil)
	if err != nil {
		return fmt.Errorf("failed to copy object: %w", err)
	}

	return nil
}

func (t *TencentStorage) Move(srcPath string, destPath string) error {
	if err := t.Copy(srcPath, destPath); err != nil {
		return err
	}

	return t.Delete(srcPath)
}

func (t *TencentStorage) Exists(path string) (bool, error) {
	objectKey := t.normalizePath(path)

	_, err := t.client.Object.Head(context.Background(), objectKey, nil)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (t *TencentStorage) GetInfo(path string) (*types.FileInfo, error) {
	objectKey := t.normalizePath(path)

	resp, err := t.client.Object.Head(context.Background(), objectKey, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get object meta: %w", err)
	}

	size := int64(0)
	if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
		fmt.Sscanf(contentLength, "%d", &size)
	}

	var modTime time.Time
	if lastModified := resp.Header.Get("Last-Modified"); lastModified != "" {
		modTime, _ = http.ParseTime(lastModified)
	}

	return &types.FileInfo{
		Name:        filepath.Base(path),
		Path:        path,
		Size:        size,
		IsDir:       false,
		ModTime:     modTime,
		ContentType: resp.Header.Get("Content-Type"),
		ETag:        resp.Header.Get("ETag"),
		URL:         t.GetURL(path),
		StorageType: string(types.StorageTencent),
	}, nil
}

func (t *TencentStorage) List(prefix string, limit int, marker string) (*types.ListResult, error) {
	objectPrefix := t.normalizePath(prefix)

	opt := &cos.BucketGetOptions{
		Prefix: objectPrefix,
	}
	if limit > 0 {
		opt.MaxKeys = limit
	}
	if marker != "" {
		opt.Marker = marker
	}

	resp, _, err := t.client.Bucket.Get(context.Background(), opt)
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	var files []types.FileInfo
	for _, object := range resp.Contents {
		var modTime time.Time
		if object.LastModified != "" {
			modTime, _ = time.Parse(time.RFC3339, object.LastModified)
		}

		files = append(files, types.FileInfo{
			Name:        filepath.Base(object.Key),
			Path:        object.Key,
			Size:        int64(object.Size),
			IsDir:       strings.HasSuffix(object.Key, "/"),
			ModTime:     modTime,
			ETag:        object.ETag,
			URL:         t.GetURL(object.Key),
			StorageType: string(types.StorageTencent),
		})
	}

	return &types.ListResult{
		Files:       files,
		NextMarker:  resp.NextMarker,
		IsTruncated: resp.IsTruncated,
	}, nil
}

func (t *TencentStorage) CreateDir(path string) error {
	objectKey := t.normalizePath(path)
	if !strings.HasSuffix(objectKey, "/") {
		objectKey += "/"
	}

	_, err := t.client.Object.Put(context.Background(), objectKey, strings.NewReader(""), nil)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return nil
}

func (t *TencentStorage) GetURL(path string) string {
	objectKey := t.normalizePath(path)
	if t.config.BaseURL != "" {
		return strings.TrimRight(t.config.BaseURL, "/") + "/" + objectKey
	}
	return fmt.Sprintf("https://%s.cos.%s.myqcloud.com/%s", t.config.Bucket, t.config.Region, objectKey)
}

func (t *TencentStorage) GetSignedURL(path string, expire time.Duration) (string, error) {
	objectKey := t.normalizePath(path)
	presignedURL, err := t.client.Object.GetPresignedURL(context.Background(), http.MethodGet, objectKey, t.config.SecretID, t.config.SecretKey, expire, nil)
	if err != nil {
		return "", err
	}
	return presignedURL.String(), nil
}

func (t *TencentStorage) GetType() types.StorageType {
	return types.StorageTencent
}

func (t *TencentStorage) normalizePath(path string) string {
	return strings.TrimLeft(strings.TrimRight(path, "/"), "/")
}
