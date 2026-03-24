package service

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/dwrui/go-zero-admin-pkg/uploads/types"
)

type Manager interface {
	Get(name ...string) (types.Storage, error)
	GetDefault() (types.Storage, error)
	AddFromConfig(name string, cfg *types.Config, isDefault ...bool) error
}

type UploadService struct {
	manager Manager
}

func NewUploadService(manager Manager) *UploadService {
	return &UploadService{
		manager: manager,
	}
}

type UploadRequest struct {
	File        io.Reader
	Filename    string
	DestPath    string
	ContentType string
	StorageName string
	Overwrite   bool
}

type UploadResponse struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	URL         string `json:"url"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
	ETag        string `json:"etag"`
	StorageType string `json:"storage_type"`
}

type DownloadRequest struct {
	Path        string
	StorageName string
}

type DownloadResponse struct {
	Reader      io.ReadCloser
	Name        string
	Size        int64
	ContentType string
}

type DeleteRequest struct {
	Path        string
	StorageName string
}

type DeleteMultiRequest struct {
	Paths       []string
	StorageName string
}

type MoveRequest struct {
	SrcPath     string
	DestPath    string
	StorageName string
}

type CopyRequest struct {
	SrcPath     string
	DestPath    string
	StorageName string
	Overwrite   bool
}

type ListRequest struct {
	Prefix      string
	Limit       int
	Marker      string
	StorageName string
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

type ListResponse struct {
	Files       []FileInfo `json:"files"`
	NextMarker  string     `json:"next_marker"`
	IsTruncated bool       `json:"is_truncated"`
}

type SignedURLRequest struct {
	Path        string
	Expire      time.Duration
	StorageName string
}

type SignedURLResponse struct {
	URL string `json:"url"`
}

func (s *UploadService) getStorage(name string) (types.Storage, error) {
	if name == "" {
		return s.manager.GetDefault()
	}
	return s.manager.Get(name)
}

func (s *UploadService) Upload(ctx context.Context, req *UploadRequest) (*UploadResponse, error) {
	storage, err := s.getStorage(req.StorageName)
	if err != nil {
		return nil, err
	}

	destPath := req.DestPath
	if destPath == "" {
		destPath = generatePath(req.Filename, "uploads")
	}

	contentType := req.ContentType
	if contentType == "" {
		contentType = detectContentType(req.Filename, nil)
	}

	opts := []types.UploadOption{
		types.WithContentType(contentType),
		types.WithOverwrite(req.Overwrite),
	}

	result, err := storage.Upload(req.File, destPath, opts...)
	if err != nil {
		return nil, fmt.Errorf("upload failed: %w", err)
	}

	return &UploadResponse{
		Name:        result.Name,
		Path:        result.Path,
		URL:         result.URL,
		Size:        result.Size,
		ContentType: result.ContentType,
		ETag:        result.ETag,
		StorageType: string(storage.GetType()),
	}, nil
}

func (s *UploadService) UploadFile(ctx context.Context, localPath string, destPath string, storageName string, overwrite bool) (*UploadResponse, error) {
	storage, err := s.getStorage(storageName)
	if err != nil {
		return nil, err
	}

	if destPath == "" {
		destPath = generatePath(filepath.Base(localPath), "uploads")
	}

	result, err := storage.UploadFile(localPath, destPath, types.WithOverwrite(overwrite))
	if err != nil {
		return nil, fmt.Errorf("upload file failed: %w", err)
	}

	return &UploadResponse{
		Name:        result.Name,
		Path:        result.Path,
		URL:         result.URL,
		Size:        result.Size,
		ContentType: result.ContentType,
		ETag:        result.ETag,
		StorageType: string(storage.GetType()),
	}, nil
}

func (s *UploadService) UploadMultipartFile(ctx context.Context, file *multipart.FileHeader, destPath string, storageName string, overwrite bool) (*UploadResponse, error) {
	f, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open multipart file: %w", err)
	}
	defer f.Close()

	return s.Upload(ctx, &UploadRequest{
		File:        f,
		Filename:    file.Filename,
		DestPath:    destPath,
		ContentType: file.Header.Get("Content-Type"),
		StorageName: storageName,
		Overwrite:   overwrite,
	})
}

func (s *UploadService) Download(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
	storage, err := s.getStorage(req.StorageName)
	if err != nil {
		return nil, err
	}

	reader, info, err := storage.Download(req.Path)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}

	return &DownloadResponse{
		Reader:      reader,
		Name:        info.Name,
		Size:        info.Size,
		ContentType: info.ContentType,
	}, nil
}

func (s *UploadService) DownloadToFile(ctx context.Context, path string, localPath string, storageName string) error {
	storage, err := s.getStorage(storageName)
	if err != nil {
		return err
	}

	return storage.DownloadToFile(path, localPath)
}

func (s *UploadService) Delete(ctx context.Context, req *DeleteRequest) error {
	storage, err := s.getStorage(req.StorageName)
	if err != nil {
		return err
	}

	return storage.Delete(req.Path)
}

func (s *UploadService) DeleteMulti(ctx context.Context, req *DeleteMultiRequest) error {
	storage, err := s.getStorage(req.StorageName)
	if err != nil {
		return err
	}

	return storage.DeleteMulti(req.Paths)
}

func (s *UploadService) Move(ctx context.Context, req *MoveRequest) error {
	storage, err := s.getStorage(req.StorageName)
	if err != nil {
		return err
	}

	return storage.Move(req.SrcPath, req.DestPath)
}

func (s *UploadService) Copy(ctx context.Context, req *CopyRequest) error {
	storage, err := s.getStorage(req.StorageName)
	if err != nil {
		return err
	}

	return storage.Copy(req.SrcPath, req.DestPath, types.WithCopyOverwrite(req.Overwrite))
}

func (s *UploadService) Exists(ctx context.Context, path string, storageName string) (bool, error) {
	storage, err := s.getStorage(storageName)
	if err != nil {
		return false, err
	}

	return storage.Exists(path)
}

func (s *UploadService) GetInfo(ctx context.Context, path string, storageName string) (*FileInfo, error) {
	storage, err := s.getStorage(storageName)
	if err != nil {
		return nil, err
	}

	info, err := storage.GetInfo(path)
	if err != nil {
		return nil, err
	}

	return &FileInfo{
		Name:        info.Name,
		Path:        info.Path,
		Size:        info.Size,
		IsDir:       info.IsDir,
		ModTime:     info.ModTime,
		ContentType: info.ContentType,
		ETag:        info.ETag,
		URL:         info.URL,
		StorageType: info.StorageType,
	}, nil
}

func (s *UploadService) List(ctx context.Context, req *ListRequest) (*ListResponse, error) {
	storage, err := s.getStorage(req.StorageName)
	if err != nil {
		return nil, err
	}

	result, err := storage.List(req.Prefix, req.Limit, req.Marker)
	if err != nil {
		return nil, err
	}

	files := make([]FileInfo, len(result.Files))
	for i, f := range result.Files {
		files[i] = FileInfo{
			Name:        f.Name,
			Path:        f.Path,
			Size:        f.Size,
			IsDir:       f.IsDir,
			ModTime:     f.ModTime,
			ContentType: f.ContentType,
			ETag:        f.ETag,
			URL:         f.URL,
			StorageType: f.StorageType,
		}
	}

	return &ListResponse{
		Files:       files,
		NextMarker:  result.NextMarker,
		IsTruncated: result.IsTruncated,
	}, nil
}

func (s *UploadService) CreateDir(ctx context.Context, path string, storageName string) error {
	storage, err := s.getStorage(storageName)
	if err != nil {
		return err
	}

	return storage.CreateDir(path)
}

func (s *UploadService) GetURL(ctx context.Context, path string, storageName string) (string, error) {
	storage, err := s.getStorage(storageName)
	if err != nil {
		return "", err
	}

	return storage.GetURL(path), nil
}

func (s *UploadService) GetSignedURL(ctx context.Context, req *SignedURLRequest) (*SignedURLResponse, error) {
	storage, err := s.getStorage(req.StorageName)
	if err != nil {
		return nil, err
	}

	if req.Expire == 0 {
		req.Expire = time.Hour
	}

	url, err := storage.GetSignedURL(req.Path, req.Expire)
	if err != nil {
		return nil, err
	}

	return &SignedURLResponse{URL: url}, nil
}

func (s *UploadService) GetStorageType(ctx context.Context, storageName string) (types.StorageType, error) {
	storage, err := s.getStorage(storageName)
	if err != nil {
		return "", err
	}

	return storage.GetType(), nil
}

func (s *UploadService) ValidateUpload(contentType string, allowedTypes []string) error {
	if len(allowedTypes) > 0 {
		allowed := false
		for _, t := range allowedTypes {
			if strings.HasPrefix(contentType, t) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("file type '%s' is not allowed", contentType)
		}
	}
	return nil
}

func generatePath(filename string, basePath string) string {
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)
	name = sanitizeFilename(name)

	timestamp := time.Now().UnixNano()
	hash := md5.Sum([]byte(fmt.Sprintf("%s%d", name, timestamp)))
	hashStr := hex.EncodeToString(hash[:])[:8]

	datePath := time.Now().Format("2006/01/02")

	return fmt.Sprintf("%s/%s/%s%s", basePath, datePath, hashStr, ext)
}

func sanitizeFilename(filename string) string {
	filename = filepath.Base(filename)

	replacer := strings.NewReplacer(
		" ", "_",
		"\t", "_",
		"\n", "_",
		"\r", "_",
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	filename = replacer.Replace(filename)

	if len(filename) > 200 {
		ext := filepath.Ext(filename)
		name := strings.TrimSuffix(filename, ext)
		filename = name[:200-len(ext)] + ext
	}

	return filename
}

func detectContentType(filename string, data []byte) string {
	ext := strings.ToLower(filepath.Ext(filename))

	contentTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".svg":  "image/svg+xml",
		".ico":  "image/x-icon",
		".bmp":  "image/bmp",
		".mp4":  "video/mp4",
		".avi":  "video/x-msvideo",
		".mov":  "video/quicktime",
		".wmv":  "video/x-ms-wmv",
		".flv":  "video/x-flv",
		".mkv":  "video/x-matroska",
		".mp3":  "audio/mpeg",
		".wav":  "audio/wav",
		".ogg":  "audio/ogg",
		".flac": "audio/flac",
		".aac":  "audio/aac",
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".ppt":  "application/vnd.ms-powerpoint",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		".zip":  "application/zip",
		".rar":  "application/vnd.rar",
		".7z":   "application/x-7z-compressed",
		".tar":  "application/x-tar",
		".gz":   "application/gzip",
		".txt":  "text/plain",
		".html": "text/html",
		".css":  "text/css",
		".js":   "application/javascript",
		".json": "application/json",
		".xml":  "application/xml",
	}

	if ct, ok := contentTypes[ext]; ok {
		return ct
	}

	if len(data) >= 512 {
		return http.DetectContentType(data[:512])
	}

	if len(data) > 0 {
		return http.DetectContentType(data)
	}

	return "application/octet-stream"
}
