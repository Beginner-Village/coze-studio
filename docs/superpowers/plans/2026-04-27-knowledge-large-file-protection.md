# Knowledge Large File Protection — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 防止大文件压垮知识库服务（P0），加可观测性能事后排查（P1），文本类 parser 改真流式 + 图片/PDF 隔离 worker pool（P2）。

**Architecture:** 上传入口加 size 限制 → parser 入口包 LimitReader → 文本类用 bufio.Scanner / json.Decoder 流式 chunk，不 ReadAll → 图片/PDF 走有界并发 worker pool → DocumentReaper 兜底卡死文档 → Prometheus 指标全程上报。

**Tech Stack:** Go 1.22+, GORM, bufio, encoding/json, prometheus/client_golang, testcontainers-go

**Spec:** [docs/superpowers/specs/2026-04-27-knowledge-large-file-protection-design.md](docs/superpowers/specs/2026-04-27-knowledge-large-file-protection-design.md)

---

## Phase P0: 止血（防 OOM）

### Task 1: 添加 size 限制常量和错误码

**Files:**
- Create: `backend/domain/knowledge/entity/limits.go`
- Modify: `backend/types/errno/knowledge.go` (在文件末尾追加)

- [ ] **Step 1: 创建 limits.go**

写入 `backend/domain/knowledge/entity/limits.go`:
```go
package entity

import "strings"

const (
	MaxImageFileSize = 10 * 1024 * 1024   // 10 MB
	MaxOtherFileSize = 100 * 1024 * 1024  // 100 MB
)

var imageExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
	".bmp":  true,
	".tiff": true,
}

// IsImageFile 根据文件名扩展名判断是否图片类型
func IsImageFile(filename string) bool {
	idx := strings.LastIndex(filename, ".")
	if idx == -1 {
		return false
	}
	ext := strings.ToLower(filename[idx:])
	return imageExtensions[ext]
}

// MaxFileSizeFor 返回该文件应用的大小上限
func MaxFileSizeFor(filename string) int64 {
	if IsImageFile(filename) {
		return MaxImageFileSize
	}
	return MaxOtherFileSize
}
```

- [ ] **Step 2: 追加错误码**

在 `backend/types/errno/knowledge.go` 末尾追加：
```go
	ErrKnowledgeFileTooLargeCode = 105000038
	ErrKnowledgeSystemBusyCode   = 105000039
```

- [ ] **Step 3: 验证编译**

Run: `cd /Users/luzhipeng/projects/ynet/coze-studio && go build ./backend/...`
Expected: 编译成功，无错误

- [ ] **Step 4: 提交**

```bash
git add backend/domain/knowledge/entity/limits.go backend/types/errno/knowledge.go
git commit -m "feat(knowledge): add file size limits and error codes"
```

---

### Task 2: 文件大小检查单元测试 + UploadGuard

**Files:**
- Create: `backend/domain/knowledge/entity/limits_test.go`
- Modify: `backend/api/handler/coze/knowledge_service.go` (在 upload handler 入口加 size 检查)

- [ ] **Step 1: 写失败的单元测试**

写入 `backend/domain/knowledge/entity/limits_test.go`:
```go
package entity

import "testing"

func TestIsImageFile(t *testing.T) {
	tests := []struct {
		filename string
		want     bool
	}{
		{"photo.jpg", true},
		{"photo.JPG", true},
		{"photo.JPEG", true},
		{"image.png", true},
		{"animation.gif", true},
		{"document.pdf", false},
		{"text.txt", false},
		{"noext", false},
	}
	for _, tt := range tests {
		if got := IsImageFile(tt.filename); got != tt.want {
			t.Errorf("IsImageFile(%q) = %v, want %v", tt.filename, got, tt.want)
		}
	}
}

func TestMaxFileSizeFor(t *testing.T) {
	if MaxFileSizeFor("photo.jpg") != MaxImageFileSize {
		t.Error("image should use MaxImageFileSize")
	}
	if MaxFileSizeFor("doc.pdf") != MaxOtherFileSize {
		t.Error("non-image should use MaxOtherFileSize")
	}
}
```

- [ ] **Step 2: 跑测试，验证 PASS**

Run: `cd /Users/luzhipeng/projects/ynet/coze-studio && go test ./backend/domain/knowledge/entity/ -run "TestIsImageFile|TestMaxFileSizeFor" -v`
Expected: PASS（因为函数已在 Task 1 实现）

- [ ] **Step 3: 在 handler 入口加 size 检查**

先用 grep 找到所有 upload 入口：
```bash
grep -rn "func.*Upload\|CreateDocument\|ProcessFile" backend/api/handler/coze/knowledge_service.go | head -10
```

在每个接受文件上传的 handler（典型如 `CreateDocument`）的入口处，识别到 `file_size`（或类似字段）后立即检查：

```go
// 文件参数解析后立即检查
if err := checkUploadSize(filename, fileSize); err != nil {
	c.JSON(consts.StatusBadRequest, errResp(err))
	return
}
```

并在文件中添加辅助函数（如未存在则在 handler 文件末尾新增）：
```go
func checkUploadSize(filename string, size int64) error {
	limit := entity.MaxFileSizeFor(filename)
	if size > limit {
		return errorx.New(errno.ErrKnowledgeFileTooLargeCode,
			errorx.KVf("msg", "file %q size %d bytes exceeds limit %d bytes", filename, size, limit))
	}
	return nil
}
```

如果 handler 不在入口拿到 size（比如来自 multipart），用 `request.ContentLength` 或在解析后立即检查 `len(fileBytes)`。

- [ ] **Step 4: 验证编译**

Run: `go build ./backend/...`
Expected: 成功

- [ ] **Step 5: 提交**

```bash
git add backend/domain/knowledge/entity/limits_test.go backend/api/handler/coze/knowledge_service.go
git commit -m "feat(knowledge): enforce upload size limits at HTTP entry"
```

---

### Task 3: parser 入口包 LimitReader（builtin 文本类）

**Files:**
- Modify: `backend/infra/impl/document/parser/builtin/parse_text.go`
- Modify: `backend/infra/impl/document/parser/builtin/parse_markdown.go`
- Modify: `backend/infra/impl/document/parser/builtin/parse_qa.go`
- Modify: `backend/infra/impl/document/parser/builtin/parse_json.go`
- Modify: `backend/infra/impl/document/parser/builtin/parse_json_maps.go`

**目标**：`io.ReadAll(reader)` 全部改成 `io.ReadAll(io.LimitReader(reader, MaxOtherFileSize+1))`，并在 read 后检查是否触顶。

- [ ] **Step 1: 改 parse_text.go**

```go
import (
    // ... 已有 imports
    "github.com/ynet-dev/ynet-studio/backend/domain/knowledge/entity"
    "github.com/ynet-dev/ynet-studio/backend/types/errno"
    "github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
)

func ParseText(config *contract.Config) ParseFn {
	return func(ctx context.Context, reader io.Reader, opts ...parser.Option) (docs []*schema.Document, err error) {
		limited := io.LimitReader(reader, entity.MaxOtherFileSize+1)
		content, err := io.ReadAll(limited)
		if err != nil {
			return nil, err
		}
		if int64(len(content)) > entity.MaxOtherFileSize {
			return nil, errorx.New(errno.ErrKnowledgeFileTooLargeCode,
				errorx.KVf("msg", "file size exceeds %d bytes", entity.MaxOtherFileSize))
		}
		// ... 后续不变
	}
}
```

- [ ] **Step 2: 改 parse_markdown.go**

同样在每处 `io.ReadAll(reader)` 和 `io.ReadAll(resp.Body)` 前面加 LimitReader 包装并检查。

- [ ] **Step 3: 改 parse_qa.go**

同上。

- [ ] **Step 4: 改 parse_json.go 和 parse_json_maps.go**

同上。

- [ ] **Step 5: 验证编译**

Run: `go build ./backend/...`
Expected: 成功

- [ ] **Step 6: 提交**

```bash
git add backend/infra/impl/document/parser/builtin/
git commit -m "feat(parser): enforce size limit in builtin text parsers"
```

---

### Task 4: parser 入口包 LimitReader（图片 + PDF）

**Files:**
- Modify: `backend/infra/impl/document/parser/builtin/parse_image.go`
- Modify: `backend/infra/impl/document/parser/builtin/parse_csv.go` (csv.NewReader 流式但仍需 size 限制)
- Modify: `backend/infra/impl/document/parser/ppstructure/parser.go`

- [ ] **Step 1: 改 parse_image.go**

图片用 `entity.MaxImageFileSize`：
```go
limited := io.LimitReader(reader, entity.MaxImageFileSize+1)
bytes, err := io.ReadAll(limited)
if err != nil {
    return nil, err
}
if int64(len(bytes)) > entity.MaxImageFileSize {
    return nil, errorx.New(errno.ErrKnowledgeFileTooLargeCode,
        errorx.KVf("msg", "image size exceeds %d bytes", entity.MaxImageFileSize))
}
```

- [ ] **Step 2: 改 parse_csv.go**

CSV 已经流式（`csv.NewReader`），但仍要包 LimitReader 防止单条超大行：
```go
limited := io.LimitReader(reader, entity.MaxOtherFileSize+1)
iter := &csvIterator{csv.NewReader(utfbom.SkipOnly(limited))}
```

注意：CSV 的 size 在读取过程中触顶时 `csv.Read` 会返回 `unexpected EOF`。需要在 iterator 错误处理中识别这种情况转换为 `ErrFileTooLarge`。简化做法：先用 `io.ReadAll(io.LimitReader(reader, MaxOtherFileSize+1))` 一次性读到内存（CSV 一般不会超过 100MB），再用 `csv.NewReader(bytes.NewReader(buf))` 处理。

- [ ] **Step 3: 改 ppstructure/parser.go**

PDF 用 `entity.MaxOtherFileSize`：
```go
limited := io.LimitReader(reader, entity.MaxOtherFileSize+1)
fileBytes, err := io.ReadAll(limited)
// ... 检查
```

也包装 `resp.Body` 的 ReadAll（防止远端服务返回过大响应耗光内存）。

- [ ] **Step 4: 验证编译**

Run: `go build ./backend/...`
Expected: 成功

- [ ] **Step 5: 提交**

```bash
git add backend/infra/impl/document/parser/
git commit -m "feat(parser): enforce size limits in image, csv, and ppstructure parsers"
```

---

## Phase P1: 可观测性

### Task 5: Prometheus metrics

**Files:**
- Create: `backend/domain/knowledge/service/metrics.go`

- [ ] **Step 1: 创建 metrics.go**

```go
package service

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	ParseFileSizeBytes = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "knowledge_parse_file_size_bytes",
			Help:    "File size in bytes processed by knowledge parser.",
			Buckets: prometheus.ExponentialBuckets(1024, 4, 12), // 1KB ~ 16GB
		},
		[]string{"file_type"},
	)

	ParseDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "knowledge_parse_duration_seconds",
			Help:    "Knowledge parse duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"file_type", "outcome"},
	)

	ParseFailedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "knowledge_parse_failed_total",
			Help: "Total number of failed knowledge parses, by reason.",
		},
		[]string{"file_type", "reason"}, // reason: too_large/timeout/parse_error/system_busy/panic
	)

	LargeFileWorkerActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "knowledge_large_file_worker_active",
			Help: "Number of currently active large-file parse workers.",
		},
	)

	LargeFileWorkerQueueDepth = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "knowledge_large_file_worker_queue_depth",
			Help: "Number of large-file parse tasks waiting in queue.",
		},
	)

	DocumentReaperCleanedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "knowledge_reaper_cleaned_total",
			Help: "Total documents cleaned up by the document reaper.",
		},
	)
)

// FileTypeForLabel 把 filename 转为 metric label
func FileTypeForLabel(filename string) string {
	idx := -1
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			idx = i
			break
		}
	}
	if idx == -1 {
		return "unknown"
	}
	ext := filename[idx+1:]
	if ext == "" {
		return "unknown"
	}
	// 限制 cardinality
	switch ext {
	case "txt", "md", "json", "csv", "pdf", "docx", "doc", "ppt", "pptx", "jpg", "jpeg", "png", "gif", "webp", "bmp":
		return ext
	}
	return "other"
}
```

- [ ] **Step 2: 验证编译**

Run: `go build ./backend/...`
Expected: 成功

- [ ] **Step 3: 提交**

```bash
git add backend/domain/knowledge/service/metrics.go
git commit -m "feat(knowledge): add Prometheus metrics for parser observability"
```

---

### Task 6: DocumentReaper（卡死文档清理）

**Files:**
- Create: `backend/domain/knowledge/service/reaper.go`
- Create: `backend/domain/knowledge/service/reaper_test.go`

- [ ] **Step 1: 写失败的测试**

写入 `backend/domain/knowledge/service/reaper_test.go`:
```go
package service

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	// 假设 mock_repository 在测试包内
)

func TestDocumentReaper_Sweep(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := NewMockDocumentRepo(ctrl) // 项目应已有 gomock 生成的 mock
	threshold := 30 * time.Minute

	// 期望：扫到 2 条卡死文档
	repo.EXPECT().FindStuckChunking(gomock.Any(), threshold).Return([]int64{101, 102}, nil)
	repo.EXPECT().SetStatus(gomock.Any(), int64(101), int32(2), "stuck in chunking - reaper cleanup").Return(nil)
	repo.EXPECT().SetStatus(gomock.Any(), int64(102), int32(2), "stuck in chunking - reaper cleanup").Return(nil)

	r := &DocumentReaper{
		repo:      repo,
		interval:  100 * time.Millisecond,
		threshold: threshold,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	r.sweep(ctx)
	// sweep 应已调用过 SetStatus 两次（mock 验证）
	assert.True(t, true)
}
```

- [ ] **Step 2: 跑测试，验证 FAIL（结构未实现）**

Run: `go test ./backend/domain/knowledge/service/ -run TestDocumentReaper_Sweep -v`
Expected: 编译错误或 fail

- [ ] **Step 3: 创建 reaper.go**

```go
package service

import (
	"context"
	"time"

	"github.com/coze-dev/coze-studio/backend/pkg/logs"
)

const (
	defaultReaperInterval  = 5 * time.Minute
	defaultReaperThreshold = 30 * time.Minute
	stuckCleanupReason     = "stuck in chunking - reaper cleanup"
	documentStatusFailed   = int32(2) // entity.DocumentStatusFailed
)

// DocumentRepo 反映 reaper 用到的 repo 接口子集（可与 entity 的 DocumentRepo 兼容/扩展）
type DocumentRepo interface {
	FindStuckChunking(ctx context.Context, threshold time.Duration) ([]int64, error)
	SetStatus(ctx context.Context, id int64, status int32, reason string) error
}

type DocumentReaper struct {
	repo      DocumentRepo
	interval  time.Duration
	threshold time.Duration
}

func NewDocumentReaper(repo DocumentRepo) *DocumentReaper {
	return &DocumentReaper{
		repo:      repo,
		interval:  defaultReaperInterval,
		threshold: defaultReaperThreshold,
	}
}

func (r *DocumentReaper) Start(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	logs.CtxInfof(ctx, "[reaper] started, interval=%v threshold=%v", r.interval, r.threshold)

	for {
		select {
		case <-ctx.Done():
			logs.CtxInfof(ctx, "[reaper] shutting down")
			return
		case <-ticker.C:
			r.sweep(ctx)
		}
	}
}

func (r *DocumentReaper) sweep(ctx context.Context) {
	ids, err := r.repo.FindStuckChunking(ctx, r.threshold)
	if err != nil {
		logs.CtxErrorf(ctx, "[reaper] FindStuckChunking failed: %v", err)
		return
	}
	if len(ids) == 0 {
		return
	}
	logs.CtxInfof(ctx, "[reaper] cleaning %d stuck documents", len(ids))
	for _, id := range ids {
		if err := r.repo.SetStatus(ctx, id, documentStatusFailed, stuckCleanupReason); err != nil {
			logs.CtxErrorf(ctx, "[reaper] SetStatus(%d) failed: %v", id, err)
			continue
		}
		DocumentReaperCleanedTotal.Inc()
	}
}
```

- [ ] **Step 4: 实现 repo 端 FindStuckChunking**

打开 `backend/domain/knowledge/internal/dal/dao/knowledge_document.go`，加方法：
```go
func (k *knowledgeDocumentDAO) FindStuckChunking(ctx context.Context, threshold time.Duration) ([]int64, error) {
	cutoff := time.Now().Add(-threshold).UnixMilli()
	var ids []int64
	err := k.client.WithContext(ctx).Table("knowledge_document").
		Select("id").
		Where("status = ?", entity.DocumentStatusChunking).
		Where("updated_at < ?", cutoff).
		Pluck("id", &ids).Error
	return ids, err
}
```

并在对应 repository 接口添加这个方法，让 mock 重新生成。

- [ ] **Step 5: 跑测试，验证 PASS**

Run: `go test ./backend/domain/knowledge/service/ -run TestDocumentReaper -v`
Expected: PASS

- [ ] **Step 6: 提交**

```bash
git add backend/domain/knowledge/service/reaper.go backend/domain/knowledge/service/reaper_test.go backend/domain/knowledge/internal/dal/dao/knowledge_document.go
git commit -m "feat(knowledge): add DocumentReaper to clean stuck chunking documents"
```

---

### Task 7: Reaper 接入服务启动

**Files:**
- Modify: `backend/application/knowledge/service.go`（或对应的服务启动文件）

- [ ] **Step 1: 找到启动入口**

```bash
grep -rn "knowledge.*Service.*New\|RegisterService\|InitService" backend/application/knowledge/ | head
```

- [ ] **Step 2: 在启动入口启动 reaper goroutine**

```go
import "github.com/coze-dev/coze-studio/backend/domain/knowledge/service"

func InitKnowledgeService(ctx context.Context, ...) {
    // ... 现有初始化
    reaper := service.NewDocumentReaper(documentRepo)
    go reaper.Start(ctx)
}
```

- [ ] **Step 3: 编译 + 验证**

Run: `go build ./backend/...`
Expected: 成功

- [ ] **Step 4: 提交**

```bash
git add backend/application/knowledge/
git commit -m "feat(knowledge): start DocumentReaper at service init"
```

---

### Task 8: parse 失败/成功上报指标

**Files:**
- Modify: `backend/domain/knowledge/service/event_handle.go`

- [ ] **Step 1: 定位 indexDocument 函数**

它从 line 160 附近开始，已经有 panic recover。

- [ ] **Step 2: 在函数入口和退出处加 metric 上报**

```go
func (k *knowledgeImpl) indexDocument(ctx context.Context, event *Event) (err error) {
    doc := event.Document
    fileType := FileTypeForLabel(doc.Name)
    start := time.Now()
    
    if doc.Size > 0 {
        ParseFileSizeBytes.WithLabelValues(fileType).Observe(float64(doc.Size))
    }
    
    defer func() {
        outcome := "success"
        if err != nil {
            outcome = "failure"
            // 提取 reason
            reason := "parse_error"
            if errors.Is(err, ErrFileTooLarge) || strings.Contains(err.Error(), "exceeds") {
                reason = "too_large"
            } else if strings.Contains(err.Error(), "panic") {
                reason = "panic"
            } else if strings.Contains(err.Error(), "system busy") {
                reason = "system_busy"
            }
            ParseFailedTotal.WithLabelValues(fileType, reason).Inc()
        }
        ParseDurationSeconds.WithLabelValues(fileType, outcome).Observe(time.Since(start).Seconds())
    }()
    
    // ... 现有逻辑
}
```

- [ ] **Step 3: 编译**

Run: `go build ./backend/...`
Expected: 成功

- [ ] **Step 4: 提交**

```bash
git add backend/domain/knowledge/service/event_handle.go
git commit -m "feat(knowledge): emit Prometheus metrics for parse outcomes"
```

---

## Phase P2: 流式 Parser 重构

### Task 9: StreamingTextChunker

**Files:**
- Create: `backend/infra/impl/document/parser/builtin/streaming_text.go`
- Create: `backend/infra/impl/document/parser/builtin/streaming_text_test.go`

- [ ] **Step 1: 写失败的测试**

写入 `streaming_text_test.go`:
```go
package builtin

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamingTextChunker_ShortText(t *testing.T) {
	chunker := &streamingTextChunker{chunkSize: 100, overlap: 10}
	docs, err := chunker.Chunk(nil, strings.NewReader("hello world"))
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, "hello world", docs[0].Content)
}

func TestStreamingTextChunker_LongText(t *testing.T) {
	chunker := &streamingTextChunker{chunkSize: 50, overlap: 0}
	input := strings.Repeat("abc ", 100) // 400 chars
	docs, err := chunker.Chunk(nil, strings.NewReader(input))
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(docs), 8)
}

func TestStreamingTextChunker_Empty(t *testing.T) {
	chunker := &streamingTextChunker{chunkSize: 100, overlap: 0}
	docs, err := chunker.Chunk(nil, strings.NewReader(""))
	require.NoError(t, err)
	assert.Len(t, docs, 0)
}
```

- [ ] **Step 2: 跑测试，验证 FAIL**

Run: `go test ./backend/infra/impl/document/parser/builtin/ -run TestStreamingTextChunker -v`
Expected: FAIL（streamingTextChunker 未定义）

- [ ] **Step 3: 实现 streaming_text.go**

```go
package builtin

import (
	"bufio"
	"context"
	"io"

	"github.com/cloudwego/eino/schema"
)

// streamingTextChunker 用 bufio.Scanner 按定长 chunk 流式读取
type streamingTextChunker struct {
	chunkSize int
	overlap   int
}

func (c *streamingTextChunker) Chunk(ctx context.Context, reader io.Reader) ([]*schema.Document, error) {
	scanner := bufio.NewScanner(reader)
	// 设大 buffer，避免单行超过默认 64KB 失败
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	scanner.Split(splitByChunkSize(c.chunkSize, c.overlap))

	var docs []*schema.Document
	for scanner.Scan() {
		text := scanner.Text()
		if text == "" {
			continue
		}
		docs = append(docs, &schema.Document{
			Content: text,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return docs, nil
}

// splitByChunkSize 返回一个 SplitFunc，按 chunkSize 切分（不切 UTF-8 字符中间）
func splitByChunkSize(chunkSize, overlap int) bufio.SplitFunc {
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if len(data) >= chunkSize {
			// 截断到 chunkSize，但确保不切到 UTF-8 字符中间
			end := chunkSize
			for end > 0 && (data[end]&0xC0) == 0x80 {
				end--
			}
			advance = end
			if overlap > 0 && advance > overlap {
				advance -= overlap
			}
			return advance, data[:end], nil
		}
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	}
}
```

- [ ] **Step 4: 跑测试，验证 PASS**

Run: `go test ./backend/infra/impl/document/parser/builtin/ -run TestStreamingTextChunker -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add backend/infra/impl/document/parser/builtin/streaming_text.go backend/infra/impl/document/parser/builtin/streaming_text_test.go
git commit -m "feat(parser): add streamingTextChunker for byte-bounded chunking"
```

---

### Task 10: 把 parse_text 切到 streaming（保留 LimitReader）

**Files:**
- Modify: `backend/infra/impl/document/parser/builtin/parse_text.go`
- Modify: `backend/infra/impl/document/parser/builtin/parse_markdown.go`
- Modify: `backend/infra/impl/document/parser/builtin/parse_qa.go`

**注意**：现有 chunk 逻辑（[chunk_custom.go](backend/infra/impl/document/parser/builtin/chunk_custom.go)）有自己的 chunk 大小、重叠、分隔符规则。**保持现有 chunk 行为不变**，只把"读取整文件 → 给 ChunkCustom"改成"分批读取 → 累积成单个 string → ChunkCustom"。

如果 ChunkCustom 接受 io.Reader 参数那就直接传 LimitReader。否则需要包一层：read 到 strings.Builder 但用 LimitReader 兜底。

- [ ] **Step 1: 检查 ChunkCustom 签名**

```bash
grep -A 5 "func ChunkCustom" backend/infra/impl/document/parser/builtin/chunk_custom.go | head -10
```

- [ ] **Step 2: 改 parse_text.go**

如果 ChunkCustom 已经接受 io.Reader，最小改动版：
```go
func ParseText(config *contract.Config) ParseFn {
	return func(ctx context.Context, reader io.Reader, opts ...parser.Option) (docs []*schema.Document, err error) {
		limited := io.LimitReader(reader, entity.MaxOtherFileSize+1)
		switch config.ChunkingStrategy.ChunkType {
		case contract.ChunkTypeCustom, contract.ChunkTypeDefault:
			docs, err = ChunkCustom(ctx, limited, config, opts...)
		default:
			return nil, fmt.Errorf("[ParseText] chunk type not support, type=%d", config.ChunkingStrategy.ChunkType)
		}
		return docs, err
	}
}
```

如果 ChunkCustom 接受 string，需要改 ChunkCustom 内部用 streamingTextChunker，或保留当前接口但用 streaming 读取后传 string。**保留原 chunk 行为优先**。

- [ ] **Step 3: 类似方式改 parse_markdown.go 和 parse_qa.go**

注意 markdown parser 还有"远程 fetch URL"路径（line 121），那部分也保持 LimitReader。

- [ ] **Step 4: 验证现有单元测试通过（chunk 行为不变）**

Run: `go test ./backend/infra/impl/document/parser/builtin/ -v`
Expected: 所有原有 parse_markdown_test、parse_qa_test 等通过

- [ ] **Step 5: 提交**

```bash
git add backend/infra/impl/document/parser/builtin/parse_text.go backend/infra/impl/document/parser/builtin/parse_markdown.go backend/infra/impl/document/parser/builtin/parse_qa.go
git commit -m "refactor(parser): use bounded reader for text-class parsers"
```

---

### Task 11: StreamingJSONParser

**Files:**
- Create: `backend/infra/impl/document/parser/builtin/streaming_json.go`
- Create: `backend/infra/impl/document/parser/builtin/streaming_json_test.go`

- [ ] **Step 1: 写失败的测试**

```go
package builtin

import (
	"strings"
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamJSONArray(t *testing.T) {
	input := `[{"a":1},{"a":2},{"a":3}]`
	items, err := streamJSONArray(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, items, 3)
	assert.Equal(t, `{"a":1}`, items[0])
}

func TestStreamJSONArray_Empty(t *testing.T) {
	items, err := streamJSONArray(strings.NewReader(`[]`))
	require.NoError(t, err)
	assert.Len(t, items, 0)
}

func TestStreamJSONArray_NotArray(t *testing.T) {
	_, err := streamJSONArray(strings.NewReader(`{"x": 1}`))
	assert.Error(t, err)
}
```

- [ ] **Step 2: 跑测试，验证 FAIL**

Run: `go test ./backend/infra/impl/document/parser/builtin/ -run TestStreamJSONArray -v`
Expected: FAIL

- [ ] **Step 3: 实现 streaming_json.go**

```go
package builtin

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

// streamJSONArray 流式读取 JSON 数组的每个元素，返回 raw JSON 字符串切片
func streamJSONArray(r io.Reader) ([]string, error) {
	dec := json.NewDecoder(r)
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if d, ok := tok.(json.Delim); !ok || d != '[' {
		return nil, errors.New("streamJSONArray: input is not a JSON array")
	}

	var items []string
	for dec.More() {
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, err
		}
		// compact 一下避免不必要空白
		var buf bytes.Buffer
		if err := json.Compact(&buf, raw); err != nil {
			return nil, err
		}
		items = append(items, buf.String())
	}
	return items, nil
}
```

- [ ] **Step 4: 跑测试，验证 PASS**

Run: `go test ./backend/infra/impl/document/parser/builtin/ -run TestStreamJSONArray -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add backend/infra/impl/document/parser/builtin/streaming_json.go backend/infra/impl/document/parser/builtin/streaming_json_test.go
git commit -m "feat(parser): add streaming JSON array decoder"
```

---

### Task 12: 集成 streamJSONArray 到 parse_json/parse_json_maps

**Files:**
- Modify: `backend/infra/impl/document/parser/builtin/parse_json.go`
- Modify: `backend/infra/impl/document/parser/builtin/parse_json_maps.go`

- [ ] **Step 1: 改 parse_json.go**

如果原本是把整个 JSON read 进来再 unmarshal 成 `[]map`，改用 streamJSONArray 一次只 decode 一个 element。

```go
func ParseJSON(config *contract.Config) ParseFn {
	return func(ctx context.Context, reader io.Reader, opts ...parser.Option) (docs []*schema.Document, err error) {
		limited := io.LimitReader(reader, entity.MaxOtherFileSize+1)

		// 流式 decode 数组元素
		items, err := streamJSONArray(limited)
		if err != nil {
			// 不是数组，fallback 到原 ReadAll 行为（顶层对象）
			// 注意：limited 已经被消费一部分，这里重新构造或直接报错
			return nil, fmt.Errorf("[ParseJSON] streaming decode failed: %w", err)
		}
		// 把 items 转成 schema.Document
		for _, item := range items {
			docs = append(docs, &schema.Document{Content: item})
		}
		return docs, nil
	}
}
```

如果原 parse_json 有更多业务逻辑（结构转换、字段过滤），保留这部分逻辑，只把"如何拿到每条 record"改成 streamJSONArray。

- [ ] **Step 2: 改 parse_json_maps.go**

类似处理。

- [ ] **Step 3: 跑测试**

Run: `go test ./backend/infra/impl/document/parser/builtin/ -v`
Expected: 通过（包括 parse_json_maps_test 和新增的 streamJSON 测试）

- [ ] **Step 4: 提交**

```bash
git add backend/infra/impl/document/parser/builtin/parse_json.go backend/infra/impl/document/parser/builtin/parse_json_maps.go
git commit -m "refactor(parser): stream JSON parsing instead of full ReadAll"
```

---

### Task 13: LargeFileWorker

**Files:**
- Create: `backend/domain/knowledge/service/large_file_worker.go`
- Create: `backend/domain/knowledge/service/large_file_worker_test.go`

- [ ] **Step 1: 写失败的测试**

```go
package service

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLargeFileWorker_RespectsConcurrency(t *testing.T) {
	w := NewLargeFileWorker(2, 10)
	defer w.Close()

	var active int64
	var maxActive int64
	done := make(chan struct{}, 5)

	for i := 0; i < 5; i++ {
		go func() {
			err := w.Submit(context.Background(), func() error {
				cur := atomic.AddInt64(&active, 1)
				if cur > atomic.LoadInt64(&maxActive) {
					atomic.StoreInt64(&maxActive, cur)
				}
				time.Sleep(50 * time.Millisecond)
				atomic.AddInt64(&active, -1)
				return nil
			})
			require.NoError(t, err)
			done <- struct{}{}
		}()
	}

	for i := 0; i < 5; i++ {
		<-done
	}
	assert.LessOrEqual(t, atomic.LoadInt64(&maxActive), int64(2))
}

func TestLargeFileWorker_QueueFullReturnsBusy(t *testing.T) {
	w := NewLargeFileWorker(1, 1)
	defer w.Close()

	// 占满 worker
	go w.Submit(context.Background(), func() error {
		time.Sleep(200 * time.Millisecond)
		return nil
	})
	time.Sleep(10 * time.Millisecond)

	// 占满队列
	go w.Submit(context.Background(), func() error {
		return nil
	})
	time.Sleep(10 * time.Millisecond)

	// 第三个应被拒绝
	err := w.Submit(context.Background(), func() error { return nil })
	assert.True(t, errors.Is(err, ErrSystemBusy))
}

func TestLargeFileWorker_RecoverPanic(t *testing.T) {
	w := NewLargeFileWorker(1, 5)
	defer w.Close()

	err := w.Submit(context.Background(), func() error {
		panic("boom")
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "panic")
}
```

- [ ] **Step 2: 跑测试，验证 FAIL**

- [ ] **Step 3: 实现 large_file_worker.go**

```go
package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

var ErrSystemBusy = errors.New("knowledge: large file worker busy")

type LargeFileWorker struct {
	sem    chan struct{}
	queue  chan task
	wg     sync.WaitGroup
	closed chan struct{}
}

type task struct {
	ctx    context.Context
	fn     func() error
	result chan error
}

func NewLargeFileWorker(maxConcurrent, maxQueue int) *LargeFileWorker {
	w := &LargeFileWorker{
		sem:    make(chan struct{}, maxConcurrent),
		queue:  make(chan task, maxQueue),
		closed: make(chan struct{}),
	}
	w.wg.Add(1)
	go w.run()
	return w
}

func (w *LargeFileWorker) run() {
	defer w.wg.Done()
	for {
		select {
		case <-w.closed:
			return
		case t := <-w.queue:
			w.sem <- struct{}{}
			LargeFileWorkerActive.Inc()
			LargeFileWorkerQueueDepth.Set(float64(len(w.queue)))
			go w.process(t)
		}
	}
}

func (w *LargeFileWorker) process(t task) {
	defer func() {
		<-w.sem
		LargeFileWorkerActive.Dec()
	}()
	defer func() {
		if r := recover(); r != nil {
			t.result <- fmt.Errorf("large file worker panic: %v", r)
		}
	}()
	t.result <- t.fn()
}

func (w *LargeFileWorker) Submit(ctx context.Context, fn func() error) error {
	t := task{ctx: ctx, fn: fn, result: make(chan error, 1)}
	select {
	case w.queue <- t:
		LargeFileWorkerQueueDepth.Set(float64(len(w.queue)))
	default:
		return ErrSystemBusy
	}
	select {
	case err := <-t.result:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *LargeFileWorker) Close() {
	close(w.closed)
	w.wg.Wait()
}
```

- [ ] **Step 4: 跑测试**

Run: `go test ./backend/domain/knowledge/service/ -run TestLargeFileWorker -v -timeout 5s`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add backend/domain/knowledge/service/large_file_worker.go backend/domain/knowledge/service/large_file_worker_test.go
git commit -m "feat(knowledge): add LargeFileWorker for bounded concurrency parsing"
```

---

### Task 14: 集成 LargeFileWorker 到 event_handle

**Files:**
- Modify: `backend/domain/knowledge/service/event_handle.go`
- Modify: `backend/domain/knowledge/service/knowledge.go`（service struct 加 worker 字段）

- [ ] **Step 1: 在 knowledgeImpl 加 worker 字段**

```go
type knowledgeImpl struct {
    // ... 已有字段
    largeFileWorker *LargeFileWorker
}
```

构造函数初始化：
```go
func NewKnowledge(...) *knowledgeImpl {
    impl := &knowledgeImpl{ /* ... */ }
    impl.largeFileWorker = NewLargeFileWorker(2, 10)
    return impl
}
```

- [ ] **Step 2: 在 indexDocument 中根据文件类型分发**

定位到 parse 阶段（约 line 240-280）。改造为：
```go
fileExt := strings.ToLower(filepath.Ext(doc.Name))
isLargeFileType := isLargeFileExt(fileExt)

parseFunc := func() error {
    return k.parseAndIndex(ctx, doc, ...) // 抽出原 parse 逻辑
}

if isLargeFileType {
    if err := k.largeFileWorker.Submit(ctx, parseFunc); err != nil {
        return errorx.WrapByCode(err, errno.ErrKnowledgeSystemBusyCode,
            errorx.KV("msg", "large file worker busy"))
    }
} else {
    if err := parseFunc(); err != nil {
        return err
    }
}
```

helper：
```go
func isLargeFileExt(ext string) bool {
    switch ext {
    case ".pdf", ".doc", ".docx", ".ppt", ".pptx",
         ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".tiff":
        return true
    }
    return false
}
```

- [ ] **Step 3: 验证编译**

Run: `go build ./backend/...`
Expected: 成功

- [ ] **Step 4: 验证现有单元测试通过**

Run: `go test ./backend/domain/knowledge/...`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add backend/domain/knowledge/service/event_handle.go backend/domain/knowledge/service/knowledge.go
git commit -m "feat(knowledge): route large file parses through LargeFileWorker"
```

---

### Task 15: 配置项

**Files:**
- Create or Modify: `backend/conf/model/knowledge.yaml`
- Modify: `backend/domain/knowledge/service/knowledge.go`（读 config）

- [ ] **Step 1: 添加配置**

如果文件不存在创建：
```yaml
parser:
  streaming_enabled: true
  max_image_size_bytes: 10485760
  max_other_size_bytes: 104857600
  large_file_worker:
    max_concurrent: 2
    max_queue: 10
  reaper:
    enabled: true
    interval_seconds: 300
    threshold_seconds: 1800
```

- [ ] **Step 2: 在初始化时读取并应用**

如果项目已有 config 加载机制，注入到 reaper / worker 构造参数。如果时间紧，先用代码常量保持现状，留 TODO 注释指向配置文件位置。

- [ ] **Step 3: 提交**

```bash
git add backend/conf/model/knowledge.yaml backend/domain/knowledge/service/
git commit -m "feat(knowledge): add config for parser, worker, and reaper"
```

---

## Phase Test: 集成测试

### Task 16: 集成测试 - 拒绝超大文件

**Files:**
- Create: `backend/domain/knowledge/service/integration_large_file_test.go`

- [ ] **Step 1: 写测试**

```go
//go:build integration

package service

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIntegration_RejectsOversizedUpload 验证 HTTP 入口拒绝超大文件
func TestIntegration_RejectsOversizedUpload(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	// 假设有 router 启动 helper（项目应有）
	server := setupTestServer(t)
	defer server.Close()

	// 构造一个 200MB 假上传
	bigPayload := bytes.Repeat([]byte("x"), 200*1024*1024)
	body := strings.NewReader(string(bigPayload))
	req, _ := http.NewRequest("POST", server.URL+"/api/knowledge/document", body)
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("X-File-Name", "huge.txt")
	req.ContentLength = int64(len(bigPayload))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
```

`setupTestServer` 需要项目集成测试 helper，按现有约定调用。

- [ ] **Step 2: 跑测试**

Run: `go test -tags integration ./backend/domain/knowledge/service/ -run TestIntegration_RejectsOversizedUpload -v`
Expected: PASS

- [ ] **Step 3: 提交**

```bash
git add backend/domain/knowledge/service/integration_large_file_test.go
git commit -m "test(knowledge): integration test for oversized upload rejection"
```

---

### Task 17: 集成测试 - 大文本流式不 OOM

**Files:**
- Modify: `backend/domain/knowledge/service/integration_large_file_test.go`

- [ ] **Step 1: 加测试**

```go
func TestIntegration_LargeTextStreaming(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	var memBefore, memAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	// 50MB 文本
	bigText := strings.Repeat("hello world\n", 50*1024*1024/12)

	parser := ParseText(&contract.Config{ChunkingStrategy: contract.ChunkingStrategy{ChunkType: contract.ChunkTypeDefault}})
	docs, err := parser(context.Background(), strings.NewReader(bigText))
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(docs), 100)

	runtime.GC()
	runtime.ReadMemStats(&memAfter)
	deltaMB := (memAfter.Alloc - memBefore.Alloc) / 1024 / 1024
	assert.LessOrEqual(t, deltaMB, uint64(200), "memory delta should be under 200MB, got %d MB", deltaMB)
}
```

- [ ] **Step 2: 运行**

Run: `go test -tags integration ./backend/domain/knowledge/service/ -run TestIntegration_LargeTextStreaming -v -timeout 60s`
Expected: PASS

- [ ] **Step 3: 提交**

```bash
git add backend/domain/knowledge/service/integration_large_file_test.go
git commit -m "test(knowledge): verify streaming text parser memory bounded"
```

---

### Task 18: 集成测试 - LargeFileWorker 并发

**Files:**
- Modify: `backend/domain/knowledge/service/large_file_worker_test.go`

加测试：

```go
func TestIntegration_WorkerConcurrencyMetric(t *testing.T) {
	w := NewLargeFileWorker(2, 10)
	defer w.Close()

	var sem sync.WaitGroup
	sem.Add(5)
	for i := 0; i < 5; i++ {
		go func() {
			defer sem.Done()
			w.Submit(context.Background(), func() error {
				time.Sleep(100 * time.Millisecond)
				return nil
			})
		}()
	}
	time.Sleep(50 * time.Millisecond)
	// 应该最多 2 个 active
	active := testutil.ToFloat64(LargeFileWorkerActive)
	assert.LessOrEqual(t, active, 2.0)
	sem.Wait()
}
```

(`testutil.ToFloat64` 来自 `github.com/prometheus/client_golang/prometheus/testutil`)

- [ ] 提交：
```bash
git commit -am "test(knowledge): verify LargeFileWorker max concurrency"
```

---

### Task 19: 集成测试 - Reaper 真实清理

**Files:**
- Create: `backend/domain/knowledge/service/reaper_integration_test.go`

```go
//go:build integration
package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIntegration_ReaperCleansStuck(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	// 1. 启 testcontainer MySQL
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// 2. 插入卡死文档
	stuckTime := time.Now().Add(-35 * time.Minute).UnixMilli()
	_, err := db.Exec(`INSERT INTO knowledge_document (id, knowledge_id, name, status, updated_at, created_at)
		VALUES (?, 1, 'stuck.txt', 1, ?, ?)`, 9001, stuckTime, stuckTime)
	require.NoError(t, err)

	// 3. 启 reaper（小 interval）
	repo := NewDocumentRepoFromDB(db)
	r := &DocumentReaper{repo: repo, interval: 100 * time.Millisecond, threshold: 30 * time.Minute}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	go r.Start(ctx)

	// 4. 等待 sweep
	time.Sleep(500 * time.Millisecond)

	// 5. 验证状态
	var status int32
	err = db.QueryRow("SELECT status FROM knowledge_document WHERE id = 9001").Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, int32(2), status, "should be marked Failed by reaper")
}
```

- [ ] **运行**

Run: `go test -tags integration ./backend/domain/knowledge/service/ -run TestIntegration_ReaperCleansStuck -v`
Expected: PASS

- [ ] **提交**

```bash
git add backend/domain/knowledge/service/reaper_integration_test.go
git commit -m "test(knowledge): integration test for DocumentReaper"
```

---

### Task 20: 全量验证 + 文档

- [ ] **Step 1: 全部单元测试**

Run: `go test ./backend/domain/knowledge/... ./backend/infra/impl/document/parser/...`
Expected: All PASS

- [ ] **Step 2: 全部集成测试**

Run: `go test -tags integration ./backend/domain/knowledge/...`
Expected: All PASS

- [ ] **Step 3: 编译产物 + docker build smoke**

Run: `go build -o /tmp/ynet-knowledge ./backend/cmd/...`
Expected: 成功

- [ ] **Step 4: 提交最终**

```bash
git status
git log --oneline -25
```

确认所有 task 都已 commit。

---

## 验收清单

- [ ] 上传 11MB 图片返回 400
- [ ] 上传 101MB 文件返回 400
- [ ] 50MB 文本文件成功，内存增量 < 200MB
- [ ] 5 个 8MB 图片同时上传，最多 2 并发
- [ ] 卡死 35min 的文档 6min 内被 reaper 清理
- [ ] Prometheus 能查到 `knowledge_parse_*` 指标
- [ ] 所有单元测试通过
- [ ] 所有集成测试通过
