package knowledge

import "errors"

var ErrKnowledgeObjectNotFound = errors.New("knowledge object not found")

type SaveObjectInput struct {
	Key         string
	ContentType string
	Content     []byte
}

type StoredObject struct {
	Key         string
	ContentType string
	Content     []byte
}

type ObjectStorage interface {
	Save(input SaveObjectInput) error
	Load(key string) (StoredObject, error)
}
