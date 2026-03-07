package knowledge

type SaveObjectInput struct {
	Key         string
	ContentType string
	Content     []byte
}

type ObjectStorage interface {
	Save(input SaveObjectInput) error
}
