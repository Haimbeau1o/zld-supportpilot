package knowledge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileSystemObjectStorage 使用本地文件系统保存原始文档内容，便于在无对象存储依赖时完成持久化验证。
type FileSystemObjectStorage struct {
	root string
}

type fileSystemObjectMetadata struct {
	ContentType string `json:"content_type"`
}

func NewFileSystemObjectStorage(root string) *FileSystemObjectStorage {
	return &FileSystemObjectStorage{root: strings.TrimSpace(root)}
}

func (storage *FileSystemObjectStorage) Save(input SaveObjectInput) error {
	objectPath, metadataPath, err := storage.resolvePaths(input.Key)
	if err != nil {
		return err
	}

	// 使用 sidecar 元数据文件保存 Content-Type，避免把协议字段混入原始文档字节流。
	if err := os.MkdirAll(filepath.Dir(objectPath), 0o755); err != nil {
		return fmt.Errorf("create storage directories: %w", err)
	}

	contentCopy := make([]byte, len(input.Content))
	copy(contentCopy, input.Content)
	if err := os.WriteFile(objectPath, contentCopy, 0o644); err != nil {
		return fmt.Errorf("write object file: %w", err)
	}

	metadataBytes, err := json.Marshal(fileSystemObjectMetadata{ContentType: input.ContentType})
	if err != nil {
		return fmt.Errorf("marshal object metadata: %w", err)
	}
	if err := os.WriteFile(metadataPath, metadataBytes, 0o644); err != nil {
		return fmt.Errorf("write object metadata: %w", err)
	}

	return nil
}

func (storage *FileSystemObjectStorage) Load(key string) (StoredObject, error) {
	objectPath, metadataPath, err := storage.resolvePaths(key)
	if err != nil {
		return StoredObject{}, err
	}

	content, err := os.ReadFile(objectPath)
	if err != nil {
		if os.IsNotExist(err) {
			return StoredObject{}, ErrKnowledgeObjectNotFound
		}
		return StoredObject{}, fmt.Errorf("read object file: %w", err)
	}

	metadata, err := storage.loadMetadata(metadataPath)
	if err != nil {
		return StoredObject{}, err
	}

	contentCopy := make([]byte, len(content))
	copy(contentCopy, content)
	return StoredObject{
		Key:         key,
		ContentType: metadata.ContentType,
		Content:     contentCopy,
	}, nil
}

func (storage *FileSystemObjectStorage) loadMetadata(metadataPath string) (fileSystemObjectMetadata, error) {
	metadataBytes, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fileSystemObjectMetadata{}, ErrKnowledgeObjectNotFound
		}
		return fileSystemObjectMetadata{}, fmt.Errorf("read object metadata: %w", err)
	}

	metadata := fileSystemObjectMetadata{}
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return fileSystemObjectMetadata{}, fmt.Errorf("unmarshal object metadata: %w", err)
	}

	return metadata, nil
}

func (storage *FileSystemObjectStorage) resolvePaths(key string) (string, string, error) {
	relativeKey, err := normalizeObjectKey(key)
	if err != nil {
		return "", "", err
	}

	objectPath := filepath.Join(storage.root, relativeKey)
	return objectPath, objectPath + ".meta.json", nil
}

func normalizeObjectKey(key string) (string, error) {
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return "", fmt.Errorf("knowledge object key is required")
	}

	normalizedKey := filepath.Clean(filepath.FromSlash(trimmedKey))
	if normalizedKey == "." || normalizedKey == "" {
		return "", fmt.Errorf("knowledge object key is required")
	}
	if filepath.IsAbs(normalizedKey) || normalizedKey == ".." || strings.HasPrefix(normalizedKey, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("knowledge object key must be a relative path")
	}

	return normalizedKey, nil
}
