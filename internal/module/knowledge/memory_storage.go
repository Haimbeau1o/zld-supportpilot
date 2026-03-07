package knowledge

import "sync"

// MemoryObjectStorage 使用内存保存原始文件内容，当前阶段用于先稳定上传入口和存储边界。
type MemoryObjectStorage struct {
	mutex   sync.RWMutex
	objects map[string][]byte
}

func NewMemoryObjectStorage() *MemoryObjectStorage {
	return &MemoryObjectStorage{
		objects: make(map[string][]byte),
	}
}

func (storage *MemoryObjectStorage) Save(input SaveObjectInput) error {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	contentCopy := make([]byte, len(input.Content))
	copy(contentCopy, input.Content)
	storage.objects[input.Key] = contentCopy
	return nil
}
