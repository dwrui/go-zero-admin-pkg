package uploads

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/dwrui/go-zero-admin-pkg/uploads/types"
)

type FileManager struct {
	storage types.Storage
}

func NewFileManager(storage types.Storage) *FileManager {
	return &FileManager{
		storage: storage,
	}
}

func (fm *FileManager) GetStorage() types.Storage {
	return fm.storage
}

func (fm *FileManager) UploadFromReader(reader io.Reader, destPath string, opts ...types.UploadOption) (*types.UploadResult, error) {
	return fm.storage.Upload(reader, destPath, opts...)
}

func (fm *FileManager) UploadFromBytes(data []byte, destPath string, opts ...types.UploadOption) (*types.UploadResult, error) {
	return fm.storage.Upload(strings.NewReader(string(data)), destPath, opts...)
}

func (fm *FileManager) UploadFromFile(localPath string, destPath string, opts ...types.UploadOption) (*types.UploadResult, error) {
	return fm.storage.UploadFile(localPath, destPath, opts...)
}

func (fm *FileManager) UploadFromMultipartFile(file *multipart.FileHeader, destPath string, opts ...types.UploadOption) (*types.UploadResult, error) {
	f, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open multipart file: %w", err)
	}
	defer f.Close()

	options := &types.UploadOptions{}
	for _, opt := range opts {
		opt(options)
	}

	if options.ContentType == "" {
		options.ContentType = file.Header.Get("Content-Type")
		if options.ContentType == "" {
			options.ContentType = DetectContentType(file.Filename, nil)
		}
	}

	return fm.storage.Upload(f, destPath, types.WithContentType(options.ContentType))
}

func (fm *FileManager) Download(path string) (io.ReadCloser, *types.FileInfo, error) {
	return fm.storage.Download(path)
}

func (fm *FileManager) DownloadToBytes(path string) ([]byte, error) {
	reader, _, err := fm.storage.Download(path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

func (fm *FileManager) DownloadToFile(path string, localPath string) error {
	return fm.storage.DownloadToFile(path, localPath)
}

func (fm *FileManager) Delete(path string) error {
	return fm.storage.Delete(path)
}

func (fm *FileManager) DeleteMulti(paths []string) error {
	return fm.storage.DeleteMulti(paths)
}

func (fm *FileManager) Copy(srcPath string, destPath string, opts ...types.CopyOption) error {
	return fm.storage.Copy(srcPath, destPath, opts...)
}

func (fm *FileManager) Move(srcPath string, destPath string) error {
	return fm.storage.Move(srcPath, destPath)
}

func (fm *FileManager) Exists(path string) (bool, error) {
	return fm.storage.Exists(path)
}

func (fm *FileManager) GetInfo(path string) (*types.FileInfo, error) {
	return fm.storage.GetInfo(path)
}

func (fm *FileManager) List(prefix string, limit int, marker string) (*types.ListResult, error) {
	return fm.storage.List(prefix, limit, marker)
}

func (fm *FileManager) CreateDir(path string) error {
	return fm.storage.CreateDir(path)
}

func (fm *FileManager) GetURL(path string) string {
	return fm.storage.GetURL(path)
}

func (fm *FileManager) GetSignedURL(path string, expire time.Duration) (string, error) {
	return fm.storage.GetSignedURL(path, expire)
}

func (fm *FileManager) GetType() types.StorageType {
	return fm.storage.GetType()
}

func GeneratePath(filename string, basePath string, storage string) string {
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)
	name = SanitizeFilename(name)

	timestamp := time.Now().UnixNano()
	hash := md5.Sum([]byte(fmt.Sprintf("%s%d", name, timestamp)))
	hashStr := storage + hex.EncodeToString(hash[:])[:8]

	datePath := time.Now().Format("2006/01/02")

	return fmt.Sprintf("%s/%s/%s%s", basePath, datePath, hashStr, ext)
}

func GenerateUniquePath(filename string, basePath string, storage types.Storage) (string, error) {
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)
	name = SanitizeFilename(name)

	datePath := time.Now().Format("2006/01/02")
	basePath = fmt.Sprintf("%s/%s/%s", basePath, datePath, name)

	for i := 0; i < 1000; i++ {
		var path string
		if i == 0 {
			path = basePath + ext
		} else {
			path = fmt.Sprintf("%s_%d%s", basePath, i, ext)
		}

		exists, err := storage.Exists(path)
		if err != nil {
			return "", err
		}
		if !exists {
			return path, nil
		}
	}

	return "", errors.New("failed to generate unique path")
}

func SanitizeFilename(filename string) string {
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

func DetectContentType(filename string, data []byte) string {
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

func IsImage(contentType string) bool {
	return strings.HasPrefix(contentType, "image/")
}

func IsVideo(contentType string) bool {
	return strings.HasPrefix(contentType, "video/")
}

func IsAudio(contentType string) bool {
	return strings.HasPrefix(contentType, "audio/")
}

func IsDocument(contentType string) bool {
	docTypes := []string{
		"application/pdf",
		"application/msword",
		"application/vnd.openxmlformats-officedocument",
		"application/vnd.ms-",
		"application/zip",
		"application/vnd.rar",
		"application/x-7z-compressed",
		"text/plain",
		"text/html",
		"text/css",
		"application/javascript",
		"application/json",
		"application/xml",
	}
	for _, t := range docTypes {
		if strings.HasPrefix(contentType, t) {
			return true
		}
	}
	return false
}

func GetFileType(contentType string) string {
	if IsImage(contentType) {
		return "image"
	}
	if IsVideo(contentType) {
		return "video"
	}
	if IsAudio(contentType) {
		return "audio"
	}
	if IsDocument(contentType) {
		return "document"
	}
	return "other"
}

func GetFileTypeInt(contentType string) int8 {
	switch GetFileType(contentType) {
	case "image":
		return 0
	case "folder":
		return 1
	case "video":
		return 2
	case "audio":
		return 3
	case "document":
		return 4
	default:
		return 5
	}
}

func FormatFileSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case size >= TB:
		return fmt.Sprintf("%.2f TB", float64(size)/TB)
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/GB)
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/KB)
	default:
		return fmt.Sprintf("%d B", size)
	}
}
