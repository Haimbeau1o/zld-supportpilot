package knowledge

import "sync"

// MemoryObjectStorage 使用内存保存原始文件内容，当前阶段用于先稳定上传入口和存储边界。
type MemoryObjectStorage struct {
	mutex   sync.RWMutex
	objects map[string]StoredObject
}

func NewMemoryObjectStorage() *MemoryObjectStorage {
	return &MemoryObjectStorage{
		objects: make(map[string]StoredObject),
	}
}

func (storage *MemoryObjectStorage) Save(input SaveObjectInput) error {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	contentCopy := make([]byte, len(input.Content))
	copy(contentCopy, input.Content)
	storage.objects[input.Key] = StoredObject{
		Key:         input.Key,
		ContentType: input.ContentType,
		Content:     contentCopy,
	}
	return nil
}

func (storage *MemoryObjectStorage) Load(key string) (StoredObject, error) {
	storage.mutex.RLock()
	defer storage.mutex.RUnlock()

	object, ok := storage.objects[key]
	if !ok {
		return StoredObject{}, ErrKnowledgeObjectNotFound
	}

	contentCopy := make([]byte, len(object.Content))
	copy(contentCopy, object.Content)
	return StoredObject{
		Key:         object.Key,
		ContentType: object.ContentType,
		Content:     contentCopy,
	}, nil
}
