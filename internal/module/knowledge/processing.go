package knowledge

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

var ErrKnowledgeProcessingQueueUnavailable = errors.New("knowledge processing queue unavailable")

// ParsedDocument 表示解析器输出的中间文本结构；当前阶段只保留纯文本，为后续复杂文档解析预留演进点。
type ParsedDocument struct {
	PlainText string
}

// DocumentParser 负责把原始文件内容转成后续可切块的文本结构。
type DocumentParser interface {
	Parse(ctx context.Context, document Document, object StoredObject) (ParsedDocument, error)
}

// DocumentChunker 负责按照稳定策略把解析结果切成多个 chunk。
type DocumentChunker interface {
	Chunk(ctx context.Context, document Document, parsed ParsedDocument) ([]DocumentChunk, error)
}

// ChunkIndexer 负责把 chunk 写入索引侧；当前阶段先用内存实现，后续可替换成向量索引。
type ChunkIndexer interface {
	ReplaceDocumentChunks(ctx context.Context, document Document, chunks []DocumentChunk) error
}

// ProcessingTaskDispatcher 只暴露“入队”能力，避免服务层和具体队列实现直接耦合。
type ProcessingTaskDispatcher interface {
	Enqueue(taskID string) error
}

// ProcessingTaskSource 暴露后台 worker 需要消费的任务流。
type ProcessingTaskSource interface {
	Jobs() <-chan string
}

// ProcessingTaskQueue 同时具备入队、消费和关闭能力。
type ProcessingTaskQueue interface {
	ProcessingTaskDispatcher
	ProcessingTaskSource
	Close() error
}

// MemoryProcessingQueue 使用内存 channel 模拟最小可用任务队列。
type MemoryProcessingQueue struct {
	jobs   chan string
	closed chan struct{}
	once   sync.Once
}

func NewMemoryProcessingQueue(bufferSize int) *MemoryProcessingQueue {
	if bufferSize < 1 {
		bufferSize = 1
	}
	return &MemoryProcessingQueue{
		jobs:   make(chan string, bufferSize),
		closed: make(chan struct{}),
	}
}

func (queue *MemoryProcessingQueue) Enqueue(taskID string) error {
	select {
	case <-queue.closed:
		return ErrKnowledgeProcessingQueueUnavailable
	default:
	}

	select {
	case <-queue.closed:
		return ErrKnowledgeProcessingQueueUnavailable
	case queue.jobs <- taskID:
		return nil
	default:
		return ErrKnowledgeProcessingQueueUnavailable
	}
}

func (queue *MemoryProcessingQueue) Jobs() <-chan string {
	return queue.jobs
}

func (queue *MemoryProcessingQueue) Close() error {
	queue.once.Do(func() {
		close(queue.closed)
		close(queue.jobs)
	})
	return nil
}

// AsyncProcessor 把同步的 ProcessTask 包装成后台 worker，从而让上传 API 和耗时处理解耦。
type AsyncProcessor struct {
	queue       ProcessingTaskQueue
	workerCount int
	process     func(context.Context, string) (DocumentProcessingTask, error)
	ctx         context.Context
	cancel      context.CancelFunc
	startOnce   sync.Once
	closeOnce   sync.Once
	waitGroup   sync.WaitGroup
}

func NewAsyncProcessor(queue ProcessingTaskQueue, workerCount int, process func(context.Context, string) (DocumentProcessingTask, error)) *AsyncProcessor {
	if workerCount < 1 {
		workerCount = 1
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &AsyncProcessor{
		queue:       queue,
		workerCount: workerCount,
		process:     process,
		ctx:         ctx,
		cancel:      cancel,
	}
}

func (processor *AsyncProcessor) Start() {
	processor.startOnce.Do(func() {
		for workerIndex := 0; workerIndex < processor.workerCount; workerIndex++ {
			processor.waitGroup.Add(1)
			go processor.runWorker()
		}
	})
}

func (processor *AsyncProcessor) Close() error {
	processor.closeOnce.Do(func() {
		processor.cancel()
		_ = processor.queue.Close()
		processor.waitGroup.Wait()
	})
	return nil
}

func (processor *AsyncProcessor) runWorker() {
	defer processor.waitGroup.Done()

	for {
		select {
		case <-processor.ctx.Done():
			return
		case taskID, ok := <-processor.queue.Jobs():
			if !ok {
				return
			}
			if strings.TrimSpace(taskID) == "" {
				continue
			}
			_, _ = processor.process(processor.ctx, taskID)
		}
	}
}

// PlainTextDocumentParser 先用纯文本解析稳定接口边界；后续可在不改服务层的前提下替换为 PDF/OCR 解析器。
type PlainTextDocumentParser struct{}

func (PlainTextDocumentParser) Parse(_ context.Context, _ Document, object StoredObject) (ParsedDocument, error) {
	plainText := strings.TrimSpace(string(object.Content))
	if plainText == "" {
		return ParsedDocument{}, fmt.Errorf("document content is empty after parse")
	}
	return ParsedDocument{PlainText: plainText}, nil
}

// FixedSizeDocumentChunker 使用段落优先、长度兜底的简单切块策略，先把“切块边界”稳定下来。
type FixedSizeDocumentChunker struct {
	MaxCharacters int
}

func (chunker FixedSizeDocumentChunker) Chunk(_ context.Context, document Document, parsed ParsedDocument) ([]DocumentChunk, error) {
	maxCharacters := chunker.MaxCharacters
	if maxCharacters < 1 {
		maxCharacters = 200
	}

	segments := splitIntoSegments(parsed.PlainText, maxCharacters)
	if len(segments) == 0 {
		return nil, fmt.Errorf("no content to chunk")
	}

	chunks := make([]DocumentChunk, 0, len(segments))
	for index, segment := range segments {
		chunks = append(chunks, NewDocumentChunk(document, index+1, segment))
	}
	return chunks, nil
}

// MemoryChunkIndexer 先把 chunk 写入内存仓储，为后续检索实现保留稳定承接点。
type MemoryChunkIndexer struct {
	repository DocumentChunkRepository
}

func NewMemoryChunkIndexer(repository DocumentChunkRepository) *MemoryChunkIndexer {
	return &MemoryChunkIndexer{repository: repository}
}

func (indexer *MemoryChunkIndexer) ReplaceDocumentChunks(_ context.Context, document Document, chunks []DocumentChunk) error {
	if indexer.repository == nil {
		return fmt.Errorf("chunk repository is required")
	}
	indexer.repository.ReplaceByDocument(document.ID, chunks)
	return nil
}

func splitIntoSegments(content string, maxCharacters int) []string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return nil
	}

	paragraphs := strings.Split(trimmed, "\n\n")
	segments := make([]string, 0)
	for _, paragraph := range paragraphs {
		normalized := strings.TrimSpace(paragraph)
		if normalized == "" {
			continue
		}

		runes := []rune(normalized)
		if len(runes) <= maxCharacters {
			segments = append(segments, normalized)
			continue
		}

		for start := 0; start < len(runes); start += maxCharacters {
			end := start + maxCharacters
			if end > len(runes) {
				end = len(runes)
			}
			segments = append(segments, strings.TrimSpace(string(runes[start:end])))
		}
	}
	return segments
}
