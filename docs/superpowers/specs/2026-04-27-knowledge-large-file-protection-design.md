# 知识库大文件防护与流式处理 — 设计文档

**Date:** 2026-04-27
**Status:** Approved (awaiting plan & implementation)
**Topic:** knowledge-large-file-protection

---

## 背景

用户反馈：知识库导入大文件时，整个系统崩溃且无法排错。

代码审计发现根因：

1. **所有文件 parser 用 `io.ReadAll` 把整个文件加载进内存**
   - [`parse_text.go:32`](backend/infra/impl/document/parser/builtin/parse_text.go:32)
   - [`parse_markdown.go:45,121`](backend/infra/impl/document/parser/builtin/parse_markdown.go)
   - [`parse_qa.go:135`](backend/infra/impl/document/parser/builtin/parse_qa.go:135)
   - [`parse_json.go:33`](backend/infra/impl/document/parser/builtin/parse_json.go:33)
   - [`parse_json_maps.go:34`](backend/infra/impl/document/parser/builtin/parse_json_maps.go:34)
   - [`parse_image.go:50`](backend/infra/impl/document/parser/builtin/parse_image.go:50)
   - [`ppstructure/parser.go:108,138`](backend/infra/impl/document/parser/ppstructure/parser.go)

2. **没有任何 file size 硬限制**（grep `MaxFileSize` 后端零结果）

3. **OOM 后无法排错**
   - [`event_handle.go:180`](backend/domain/knowledge/service/event_handle.go:180) 的 `recover` 只能处理 Go panic
   - Linux OOMKiller SIGKILL 整个进程，recover 不会执行
   - K8s 重启后 `chunking` 状态文档卡死，没有 reaper 清理
   - 没有 metrics 记录文件大小、处理耗时、失败原因

---

## 目标

1. **止血（P0）**：永远不让大文件压垮服务
2. **可排错（P1）**：出问题能查得清楚发生了什么
3. **架构升级（P2）**：文本类 parser 真流式，不再 ReadAll；图片/PDF 用隔离 worker 限制并发

---

## 范围与约束

### 文件大小硬限制

| 类型 | 限制 |
|------|------|
| 图片（jpg/png/gif/webp 等，OCR 识别） | **10 MB** |
| 其他所有类型（text/markdown/json/csv/qa/pdf/ppt/doc 等） | **100 MB** |

理由：图片走 OCR，下游服务（PaddleOCR）对内存压力大，且实际业务场景图片 10MB 已经覆盖典型截图/扫描文档；其他文件类型用户提交几十 MB 的 PDF/文档是常见的，限制 100MB 是业界惯例（Notion、飞书量级）。

### 流式策略（分层处理）

| 文件类型 | 策略 |
|----------|------|
| text / markdown / qa | **真流式**：用 `bufio.Scanner` 按段落/行 chunk |
| json / json_maps | **真流式**：用 `json.Decoder` token 流处理数组/maps 的元素 |
| image / pdf / ppt | **隔离 worker**：硬限制 + 独立 worker pool（限并发 N=2） |

### 兼容性

- 只改 `infra/impl/document/parser/builtin/*` 和 `infra/impl/document/parser/ppstructure/*` 内部实现
- 对 `infra/contract/document/parser/parser.go` 的接口不破坏
- 现有 chunk 行为（chunk 大小、重叠、分隔符规则）保持一致

---

## 架构

### 高层架构图

```
┌─────────────────────────────────────────────────────────────┐
│  HTTP 上传入口 (knowledge_service.go)                        │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ UploadGuard.CheckSize(file)                          │  │
│  │ 图片 > 10MB → 400 ErrFileTooLarge                    │  │
│  │ 其他 > 100MB → 400 ErrFileTooLarge                   │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼ (上传成功)
┌─────────────────────────────────────────────────────────────┐
│  对象存储 (MinIO/S3)                                        │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼ (异步事件)
┌─────────────────────────────────────────────────────────────┐
│  event_handle.go: indexDocument                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ ParserLimiter.Wrap(reader)  // io.LimitReader        │  │
│  │ 文件类型分发:                                         │  │
│  │   ├─ 文本类 → StreamingTextParser                    │  │
│  │   └─ 图片/PDF → LargeFileWorker.Submit               │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│  Metrics + Logs                                              │
│  - knowledge_parse_file_size_bytes (histogram)              │
│  - knowledge_parse_duration_seconds (histogram)             │
│  - knowledge_parse_failed_total{reason} (counter)           │
│  - knowledge_parse_concurrent_workers (gauge)               │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│  DocumentReaper (常驻 goroutine)                            │
│  每 5 分钟扫: status=chunking AND updated_at < now-30min     │
│  → SetStatus(Failed, "stuck on legacy / OOM recovery")      │
└─────────────────────────────────────────────────────────────┘
```

---

## 组件设计

### C1. UploadGuard（HTTP 入口大小检查）

**位置**：`backend/api/handler/coze/knowledge_service.go`（修改）

**职责**：
- 在 `UploadDocument` / `CreateDocument` 等 handler 入口检查文件大小
- 根据 file extension 区分图片 vs 其他

**接口**：
```go
// CheckUploadSize 检查上传文件大小是否超出限制
// 返回 ErrFileTooLarge 错误（含具体 size、limit、type）
func CheckUploadSize(filename string, size int64) error
```

**配置常量**（新建 `backend/domain/knowledge/entity/limits.go`）：
```go
const (
    MaxImageFileSize = 10 * 1024 * 1024   // 10 MB
    MaxOtherFileSize = 100 * 1024 * 1024  // 100 MB
)

var imageExtensions = map[string]bool{
    ".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
    ".webp": true, ".bmp": true, ".tiff": true,
}
```

### C2. ParserLimiter（parser 入口包 LimitReader）

**位置**：每个 `backend/infra/impl/document/parser/builtin/*.go`（修改）

**职责**：
- 在每个 ParseFn 入口包一层 `io.LimitReader`
- 超限时 `bytes read >= limit` → 返回 `ErrFileTooLarge`

**实现模式**：
```go
func ParseText(config *contract.Config) ParseFn {
    return func(ctx context.Context, reader io.Reader, opts ...parser.Option) (...)  {
        limited := &io.LimitedReader{R: reader, N: MaxOtherFileSize + 1}

        // 文本类: 不再 ReadAll，改用 StreamingScanner
        return streamingChunk(ctx, limited, config, opts...)
    }
}
```

### C3. StreamingTextParser（文本类真流式）

**位置**：新建 `backend/infra/impl/document/parser/builtin/streaming_text.go`

**职责**：
- 用 `bufio.Scanner` 按 chunk 配置（按行/按段/按定长）流式读取
- 每读够一个 chunk 就 yield 一个 `*schema.Document`
- 不在内存中持有整个文件

**接口**：
```go
type StreamingTextChunker struct {
    config   *contract.Config
    maxBytes int64
}

func (s *StreamingTextChunker) Chunk(ctx context.Context, reader io.Reader) ([]*schema.Document, error)
```

**复用范围**：`parse_text.go`、`parse_markdown.go`、`parse_qa.go`

### C4. StreamingJSONParser（JSON 流式）

**位置**：新建 `backend/infra/impl/document/parser/builtin/streaming_json.go`

**职责**：
- 用 `encoding/json.Decoder` 边读边 decode
- 对数组：逐个 element decode，每 N 个 element batch 成一个 chunk
- 对 maps：逐个 key-value decode
- 顶层巨大对象（不是数组/maps）：依然要 ReadAll，但有 LimitReader 兜底

**复用范围**：`parse_json.go`、`parse_json_maps.go`

### C5. LargeFileWorker（图片/PDF 隔离 worker pool）

**位置**：新建 `backend/domain/knowledge/service/large_file_worker.go`

**职责**：
- 提供有界并发的 worker pool（默认 N=2）
- 图片/PDF parse 通过 `Submit` 提交任务
- worker 满时排队，超过队列上限返回 `ErrSystemBusy`
- 每个 worker goroutine 独立处理，panic 不影响其他

**接口**：
```go
type LargeFileWorker struct {
    sem    chan struct{}  // 限并发
    queue  chan task      // 排队
}

type task struct {
    ctx    context.Context
    fn     func() error
    result chan error
}

func NewLargeFileWorker(maxConcurrent, maxQueue int) *LargeFileWorker
func (w *LargeFileWorker) Submit(ctx context.Context, fn func() error) error
```

**配置**：
- `maxConcurrent = 2`（同时最多 2 个大文件解析）
- `maxQueue = 10`（队列上限，超过返回 busy）
- 配置项放 `backend/conf/model/knowledge.yaml`

### C6. DocumentReaper（卡死文档清理）

**位置**：新建 `backend/domain/knowledge/service/reaper.go`

**职责**：
- 服务启动时启动一个常驻 goroutine
- 每 5 分钟扫描：`SELECT id FROM knowledge_document WHERE status = ? AND updated_at < ?` (chunking, now - 30min)
- 命中的文档 → `SetStatus(Failed, "stuck in chunking - reaper cleanup")`
- 优雅退出：监听 ctx.Done()

**接口**：
```go
type DocumentReaper struct {
    repo     repository.DocumentRepo
    interval time.Duration
    threshold time.Duration
}

func NewDocumentReaper(repo repository.DocumentRepo) *DocumentReaper
func (r *DocumentReaper) Start(ctx context.Context)
```

**配置**：
- `interval = 5 * time.Minute`
- `threshold = 30 * time.Minute`

### C7. Metrics（Prometheus 指标）

**位置**：新建 `backend/domain/knowledge/service/metrics.go`

**指标**：
```go
var (
    ParseFileSizeBytes = promauto.NewHistogramVec(...,
        []string{"file_type"})

    ParseDurationSeconds = promauto.NewHistogramVec(...,
        []string{"file_type", "outcome"})

    ParseFailedTotal = promauto.NewCounterVec(...,
        []string{"file_type", "reason"})  // reason: too_large/timeout/parse_error/system_busy/oom

    LargeFileWorkerActive = promauto.NewGauge(...)
    LargeFileWorkerQueue = promauto.NewGauge(...)
    DocumentReaperCleaned = promauto.NewCounter(...)
)
```

---

## 数据流

### 成功路径（文本类）
```
HTTP POST /api/knowledge/document
  ↓ UploadGuard 通过
  ↓ 存对象存储
  ↓ Kafka/事件 → indexDocument(doc)
  ↓ ParserLimiter 包装 reader (LimitReader 100MB)
  ↓ StreamingTextParser.Chunk
  ↓ for chunk := range scanner: emit *schema.Document
  ↓ batch insert knowledge_document_slice
  ↓ SetStatus(Indexed)
  ↓ Metrics: file_size, duration, success
```

### 成功路径（图片/PDF）
```
HTTP POST /api/knowledge/document
  ↓ UploadGuard 通过
  ↓ 存对象存储
  ↓ Kafka/事件 → indexDocument(doc)
  ↓ LargeFileWorker.Submit(parseTask)
  ↓ (排队等可用 worker)
  ↓ worker 调远程 OCR/PPStructure 服务
  ↓ 返回 chunks → batch insert
  ↓ SetStatus(Indexed)
  ↓ Metrics: 上报
```

### 失败路径

| 场景 | 行为 |
|------|------|
| 上传超大 | HTTP 400 `ErrFileTooLarge`，含 size/limit/type |
| parser ReadAll 触发 LimitReader | 返回 `ErrFileTooLarge`，状态 Failed |
| LargeFileWorker 队列满 | 返回 `ErrSystemBusy`，状态保持 Pending（可重试） |
| Go panic | recover → SetStatus(Failed, "panic: %v") |
| OOM（极端，本设计应避免） | 进程被 SIGKILL，DocumentReaper 5 分钟内清理卡死文档 |
| 远程 OCR 服务 5xx | 重试逻辑保持现状（依赖 retry queue） |

---

## 文件结构

### 新建文件
```
backend/domain/knowledge/entity/limits.go              # size 常量
backend/domain/knowledge/service/large_file_worker.go  # worker pool
backend/domain/knowledge/service/reaper.go             # 卡死清理
backend/domain/knowledge/service/metrics.go            # Prometheus 指标
backend/infra/impl/document/parser/builtin/streaming_text.go  # 文本流式
backend/infra/impl/document/parser/builtin/streaming_json.go  # JSON 流式

# 测试
backend/domain/knowledge/service/large_file_worker_test.go
backend/domain/knowledge/service/reaper_test.go
backend/infra/impl/document/parser/builtin/streaming_text_test.go
backend/infra/impl/document/parser/builtin/streaming_json_test.go
backend/domain/knowledge/service/integration_large_file_test.go  # 集成测试
```

### 修改文件
```
backend/api/handler/coze/knowledge_service.go         # 接 UploadGuard
backend/infra/impl/document/parser/builtin/parse_text.go      # 改流式
backend/infra/impl/document/parser/builtin/parse_markdown.go  # 改流式
backend/infra/impl/document/parser/builtin/parse_qa.go        # 改流式
backend/infra/impl/document/parser/builtin/parse_json.go      # 改流式
backend/infra/impl/document/parser/builtin/parse_json_maps.go # 改流式
backend/infra/impl/document/parser/builtin/parse_image.go     # 包 LimitReader（不改流式）
backend/infra/impl/document/parser/ppstructure/parser.go      # 包 LimitReader（不改流式）
backend/infra/impl/document/parser/builtin/parse_csv.go       # 包 LimitReader（已用 csv.NewReader 流式，无需改 chunk 逻辑）
backend/domain/knowledge/service/event_handle.go      # 接 LargeFileWorker
backend/conf/model/knowledge.yaml                     # worker 配置
```

---

## 错误处理

### 错误码（新增）

`backend/types/errno/knowledge.go` 现有最大编号 `105000037`，新增码续编：
```go
const (
    ErrKnowledgeFileTooLargeCode = 105000038  // 文件超出大小限制
    ErrKnowledgeSystemBusyCode   = 105000039  // worker 队列已满
)
```

### 错误信息要求

`ErrFileTooLarge` 必须返回：
- 文件名
- 实际大小（人类可读，e.g., "150.3 MB"）
- 上限（e.g., "100 MB"）
- 文件类型（"image" / "other"）

### Panic 处理

`event_handle.go:180-188` 的 `recover` 保留，但增强：
- 记录 panic 时的文件大小、文件名到错误信息
- 上报 `ParseFailedTotal{reason="panic"}` metric

---

## 测试策略

### 单元测试

| 测试 | 覆盖 |
|------|------|
| `TestUploadGuard_CheckSize` | 图片 9MB OK / 11MB 拒绝；其他 99MB OK / 101MB 拒绝；空文件 OK |
| `TestStreamingTextChunker_Chunk` | 短文本不分 chunk；长文本按段分 chunk；空 reader 返回空；超限返回错误 |
| `TestStreamingJSONParser` | JSON 数组流式；JSON maps 流式；非数组顶层 fallback；malformed JSON 错误 |
| `TestLargeFileWorker_Submit` | 并发限制生效；队列满返回 busy；任务 panic 不影响其他；ctx cancel 中断 |
| `TestDocumentReaper_Sweep` | 30min 内不动；30min+ 转 Failed；扫描错误不阻塞下次 |

### 集成测试

| 测试 | 流程 |
|------|------|
| `TestIntegration_LargeTextFile` | 用 50MB 真实 text 文件走完整流程，断言：内存峰值 < 200MB（用 `runtime.MemStats`），slice 数量 > 0，状态 Indexed |
| `TestIntegration_RejectsOversized` | 上传 150MB 文件，断言：HTTP 400，错误信息含 size 和 limit |
| `TestIntegration_LargeImageWorker` | 同时上传 5 张 8MB 图片，断言：最多 2 并发，其他排队 |
| `TestIntegration_ReaperCleansStuck` | 手动插入一条 chunking 状态 35min 前的 doc，启动 reaper 后等 6min，断言：状态变 Failed |

---

## 部署与回滚

### 配置开关（feature flag）

新增配置（`backend/conf/model/knowledge.yaml`）：
```yaml
parser:
  streaming_enabled: true        # 是否启用流式 parser（false 回退到 ReadAll）
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

`streaming_enabled=false` 时回退到原 ReadAll 行为，但 size limit 始终生效。这给运维一个灰度/紧急回滚的开关。

### 部署顺序

1. P0 优先：UploadGuard + ParserLimiter + size 限制（半天）
2. P1：Reaper + Metrics + 增强日志（1-2 天）
3. P2：StreamingTextParser + StreamingJSONParser + LargeFileWorker（一周）

---

## 风险与缓解

| 风险 | 缓解 |
|------|------|
| 流式 chunk 边界与原 ReadAll 行为不一致 | 单元测试用相同输入对比新旧 chunk 结果，必要时保留原实现作 fallback |
| LargeFileWorker 死锁 | ctx cancellation 必须传透；超时上限保护（每个任务 max 5min） |
| Reaper 误杀正在处理的文档 | 阈值 30min 远大于正常处理时间（典型 < 1min），且只针对 chunking 状态 |
| OB 兼容性 | 单元测试用 MySQL，集成测试启动前 smoke test OB（已有 OB 测试镜像） |
| 现有 stuck 文档清理影响业务 | reaper 启动前先用 SQL 查询统计数量，必要时分批 |

---

## Out of Scope

- 不改 chunk 大小/重叠等业务参数
- 不改远程 OCR / PPStructure 服务自身的实现
- 不改 ES 索引/搜索逻辑
- 不重构 `event_handle.go` 整体（只接入 LargeFileWorker）
- 不动 `infra/contract/document/parser/parser.go` 接口

---

## 验收标准

1. ✅ 上传 11MB 图片返回 400，含明确错误信息
2. ✅ 上传 101MB 文件返回 400，含明确错误信息
3. ✅ 上传 50MB 文本文件成功，内存峰值 < 200MB（pprof 验证）
4. ✅ 同时上传 5 个 8MB 图片，确认最多 2 并发（metrics 验证）
5. ✅ 手动制造一条卡死文档，6min 内被 reaper 转为 Failed
6. ✅ Prometheus 能查到 `knowledge_parse_*` 系列指标
7. ✅ 错误码 `ErrFileTooLarge` 在前端可读，含 size/limit
8. ✅ `streaming_enabled=false` 能回退到原行为
9. ✅ 全部单元测试通过
10. ✅ 集成测试在 testcontainers 跑通
