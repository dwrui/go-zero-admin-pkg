package local

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dwrui/go-zero-admin-pkg/uploads/types"
)

type LocalStorage struct {
	config  *types.LocalConfig
	rootDir string
}

func NewLocalStorage(config *types.LocalConfig) (*LocalStorage, error) {
	if config.RootPath == "" {
		return nil, errors.New("local storage root path is required")
	}

	rootPath := filepath.Clean(config.RootPath)

	if err := os.MkdirAll(rootPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create root directory: %w", err)
	}

	return &LocalStorage{
		config:  config,
		rootDir: rootPath,
	}, nil
}

func (l *LocalStorage) Upload(reader io.Reader, destPath string, opts ...types.UploadOption) (*types.UploadResult, error) {
	options := &types.UploadOptions{}
	for _, opt := range opts {
		opt(options)
	}

	fullPath := l.getFullPath(destPath)
	dir := filepath.Dir(fullPath)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	if !options.Overwrite {
		if _, err := os.Stat(fullPath); err == nil {
			return nil, fmt.Errorf("file already exists: %s", destPath)
		}
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	var written int64
	if options.ProgressFunc != nil {
		buf := make([]byte, 32*1024)
		written, err = io.CopyBuffer(file, reader, buf)
		options.ProgressFunc(written, written)
	} else {
		written, err = io.Copy(file, reader)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	contentType := options.ContentType
	if contentType == "" {
		contentType = l.getContentType(fullPath)
	}

	return &types.UploadResult{
		Name:        filepath.Base(destPath),
		Path:        destPath,
		URL:         l.GetURL(destPath),
		Size:        written,
		ContentType: contentType,
	}, nil
}

func (l *LocalStorage) UploadFile(localPath string, destPath string, opts ...types.UploadOption) (*types.UploadResult, error) {
	file, err := os.Open(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open local file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	options := &types.UploadOptions{}
	for _, opt := range opts {
		opt(options)
	}

	if options.ProgressFunc != nil {
		options.ProgressFunc(0, stat.Size())
	}

	result, err := l.Upload(file, destPath, opts...)
	if err != nil {
		return nil, err
	}

	if options.ProgressFunc != nil {
		options.ProgressFunc(stat.Size(), stat.Size())
	}

	return result, nil
}

func (l *LocalStorage) Download(path string) (io.ReadCloser, *types.FileInfo, error) {
	fullPath := l.getFullPath(path)

	file, err := os.Open(fullPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open file: %w", err)
	}

	info, err := l.GetInfo(path)
	if err != nil {
		file.Close()
		return nil, nil, err
	}

	return file, info, nil
}

func (l *LocalStorage) DownloadToFile(path string, localPath string) error {
	fullPath := l.getFullPath(path)

	srcFile, err := os.Open(fullPath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	dstFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

func (l *LocalStorage) Delete(path string) error {
	fullPath := l.getFullPath(path)

	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

func (l *LocalStorage) DeleteMulti(paths []string) error {
	for _, path := range paths {
		if err := l.Delete(path); err != nil {
			return err
		}
	}
	return nil
}

func (l *LocalStorage) Copy(srcPath string, destPath string, opts ...types.CopyOption) error {
	options := &types.CopyOptions{}
	for _, opt := range opts {
		opt(options)
	}

	srcFullPath := l.getFullPath(srcPath)
	destFullPath := l.getFullPath(destPath)

	if !options.Overwrite {
		if _, err := os.Stat(destFullPath); err == nil {
			return fmt.Errorf("destination file already exists: %s", destPath)
		}
	}

	dir := filepath.Dir(destFullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	srcFile, err := os.Open(srcFullPath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(destFullPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

func (l *LocalStorage) Move(srcPath string, destPath string) error {
	srcFullPath := l.getFullPath(srcPath)
	destFullPath := l.getFullPath(destPath)

	dir := filepath.Dir(destFullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.Rename(srcFullPath, destFullPath); err != nil {
		return fmt.Errorf("failed to move file: %w", err)
	}

	return nil
}

func (l *LocalStorage) Exists(path string) (bool, error) {
	fullPath := l.getFullPath(path)

	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (l *LocalStorage) GetInfo(path string) (*types.FileInfo, error) {
	fullPath := l.getFullPath(path)

	stat, err := os.Stat(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	return &types.FileInfo{
		Name:        stat.Name(),
		Path:        path,
		Size:        stat.Size(),
		IsDir:       stat.IsDir(),
		ModTime:     stat.ModTime(),
		ContentType: l.getContentType(fullPath),
		URL:         l.GetURL(path),
		StorageType: string(types.StorageLocal),
	}, nil
}

func (l *LocalStorage) List(prefix string, limit int, marker string) (*types.ListResult, error) {
	fullPath := l.getFullPath(prefix)

	var files []types.FileInfo

	err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(l.rootDir, path)
		if err != nil {
			return err
		}

		relPath = filepath.ToSlash(relPath)

		if marker != "" && relPath <= marker {
			return nil
		}

		if limit > 0 && len(files) >= limit {
			return filepath.SkipAll
		}

		files = append(files, types.FileInfo{
			Name:        info.Name(),
			Path:        relPath,
			Size:        info.Size(),
			IsDir:       info.IsDir(),
			ModTime:     info.ModTime(),
			ContentType: l.getContentType(path),
			URL:         l.GetURL(relPath),
			StorageType: string(types.StorageLocal),
		})

		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	result := &types.ListResult{
		Files:       files,
		IsTruncated: limit > 0 && len(files) >= limit,
	}

	if result.IsTruncated && len(files) > 0 {
		result.NextMarker = files[len(files)-1].Path
	}

	return result, nil
}

func (l *LocalStorage) CreateDir(path string) error {
	fullPath := l.getFullPath(path)

	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return nil
}

func (l *LocalStorage) GetURL(path string) string {
	if l.config.BaseURL != "" {
		return strings.TrimRight(l.config.BaseURL, "/") + "/" + strings.TrimLeft(path, "/")
	}
	return path
}

func (l *LocalStorage) GetSignedURL(path string, expire time.Duration) (string, error) {
	return l.GetURL(path), nil
}

func (l *LocalStorage) GetType() types.StorageType {
	return types.StorageLocal
}

func (l *LocalStorage) getFullPath(path string) string {
	cleanPath := filepath.Clean(path)
	if strings.HasPrefix(cleanPath, "..") {
		cleanPath = strings.TrimPrefix(cleanPath, "..")
	}
	return filepath.Join(l.rootDir, cleanPath)
}

func (l *LocalStorage) getContentType(path string) string {
	ext := filepath.Ext(path)
	if ext != "" {
		contentType := mime.TypeByExtension(ext)
		if contentType != "" {
			return contentType
		}
	}

	file, err := os.Open(path)
	if err != nil {
		return "application/octet-stream"
	}
	defer file.Close()

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil {
		return "application/octet-stream"
	}

	return http.DetectContentType(buffer[:n])
}

func (l *LocalStorage) getETag(path string) string {
	stat, err := os.Stat(path)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%x-%x", stat.Size(), stat.ModTime().UnixNano())
}
