# 空间级操作审计日志 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 ynet-studio 后端增加空间级操作审计日志：API 中间件按路由白名单采集写操作，异步落库 MySQL，提供仅 Owner/Admin 可查的查询接口与前端页面。

**Architecture:** 顺延项目 DDD 分层。`OperationLogMW` 在 `SessionAuthMW` 之后采集命中白名单的写请求，非阻塞投递到内存 channel；`domain/operationlog` 的后台 worker 批量落库并定时清理过期记录；`application/operationlog` 提供查询编排（权限校验 + operator_name 回查）；新增 `POST /api/operation_log/list` 接口与前端「操作日志」页。

**Tech Stack:** Go (Hertz, gorm, idgen 雪花 ID)、MySQL、前端 React + Rush monorepo (`packages/arch/api-schema`)。

**关键参考文件（已存在的同类实现，照抄其模式）：**
- 中间件链注册：`backend/main.go:146-155`
- 中间件读 body：`backend/api/middleware/log.go`（`AccessLogMW`，`ctx.Next` 后 `ctx.Request.Body()` 可读）
- 自定义功能全套模板：`backend/application/space/space_diagnose.go`（全局 SVC + 权限）、`backend/api/handler/space/space_diagnose_service.go`（handler）、`backend/api/router/space/space_diagnose.go`（router）、`backend/application/application.go:226`（init wiring）
- DAO 模板：`backend/domain/shortcutcmd/internal/dal/dao.go`
- 权限校验：`domain/user/service.User.CheckMemberPermission(ctx, spaceID, userID) (isMember bool, roleType int32, canInvite, canManage bool, err error)`，`canManage==true` 即 Owner/Admin
- uid：`backend/application/base/ctxutil.GetUIDFromCtx(ctx) *int64`
- errno：`backend/types/errno/space.go`（自定义码段 112xxxxxx）
- schema 自愈：`docker/volumes/mysql/schema.sql`（add-only）

**全部 backend 命令在 `cd backend` 下执行。测试：`go test ./...`。**

---

## 文件结构

**新建（backend）：**
- `backend/domain/operationlog/entity/operation_log.go` — 实体 `OperationLog`、采集事件 `Event`、查询过滤 `ListFilter`、状态/动作常量
- `backend/domain/operationlog/internal/dal/model/operation_log.go` — gorm PO（手写，含 `TableName()`）
- `backend/domain/operationlog/internal/dal/dao.go` — gorm DAO：`BatchCreate` / `List` / `DeleteBefore`
- `backend/domain/operationlog/repository/repository.go` — 仓储接口 + 实现（封装 DAO）
- `backend/domain/operationlog/service/operation_log.go` — service 接口
- `backend/domain/operationlog/service/operation_log_impl.go` — 实现：异步 worker、查询、清理循环
- `backend/domain/operationlog/service/operation_log_test.go` — service/worker/清理单测
- `backend/application/operationlog/operation_log.go` — 全局 SVC 单例、`Collect`、`Init`、`ListOperationLogs`（权限 + 回查名字）
- `backend/application/operationlog/operation_log_test.go` — 权限/回查单测
- `backend/api/model/data/operationlog/operation_log.go` — 请求/响应结构体
- `backend/api/handler/operationlog/operation_log_service.go` — handler `ListOperationLog`
- `backend/api/middleware/operationlog/route_registry.go` — 路由白名单 + 语义映射 + 字段抽取
- `backend/api/middleware/operationlog/route_registry_test.go` — 匹配/抽取单测
- `backend/api/middleware/operation_log.go` — `OperationLogMW()` 中间件
- `backend/api/middleware/operation_log_test.go` — 中间件单测
- `backend/api/router/operationlog/operation_log.go` — `Register(r)`

**修改（backend）：**
- `backend/main.go` — 注册 `OperationLogMW()`
- `backend/api/router/register.go` — 注册 operation_log 路由
- `backend/application/application.go` — `initComplexServices` 末尾 wiring `operationlog.Init(...)`
- `docker/volumes/mysql/schema.sql` — 追加 `operation_log` 建表 DDL

**新建/修改（frontend）：**
- `frontend/packages/arch/api-schema/src/idl/operation_log/` — idl（或直接写 ts 调用）
- 空间「操作日志」页面组件 + workspace 子菜单入口（具体包在 Task 14 定位）

---

## Task 1: DB schema DDL

**Files:**
- Modify: `docker/volumes/mysql/schema.sql`（在文件末尾追加）

- [ ] **Step 1: 追加建表 DDL**

在 `docker/volumes/mysql/schema.sql` 末尾追加：

```sql
-- 空间级操作审计日志
CREATE TABLE IF NOT EXISTS `operation_log` (
  `id`              bigint unsigned NOT NULL COMMENT '主键ID',
  `space_id`        bigint unsigned NOT NULL DEFAULT '0' COMMENT '空间ID,0=无空间',
  `operator_id`     bigint unsigned NOT NULL DEFAULT '0' COMMENT '操作者用户ID',
  `module`          varchar(64)  NOT NULL DEFAULT '' COMMENT '模块',
  `resource_type`   int          NOT NULL DEFAULT '0' COMMENT '资源类型枚举',
  `resource_id`     bigint unsigned NOT NULL DEFAULT '0' COMMENT '资源ID',
  `resource_name`   varchar(255) NOT NULL DEFAULT '' COMMENT '资源名称',
  `action`          varchar(32)  NOT NULL DEFAULT '' COMMENT '动作',
  `description`     varchar(512) NOT NULL DEFAULT '' COMMENT '中文语义描述',
  `method`          varchar(8)   NOT NULL DEFAULT '' COMMENT 'HTTP方法',
  `path`            varchar(255) NOT NULL DEFAULT '' COMMENT '请求路径',
  `request_summary` varchar(512) NOT NULL DEFAULT '' COMMENT '请求摘要',
  `status`          tinyint      NOT NULL DEFAULT '1' COMMENT '1=成功 2=失败',
  `error_code`      varchar(64)  NOT NULL DEFAULT '' COMMENT '业务错误码',
  `client_ip`       varchar(64)  NOT NULL DEFAULT '' COMMENT '客户端IP',
  `duration_ms`     int          NOT NULL DEFAULT '0' COMMENT '耗时毫秒',
  `log_id`          varchar(64)  NOT NULL DEFAULT '' COMMENT '链路logID',
  `created_at`      bigint       NOT NULL DEFAULT '0' COMMENT '创建时间(毫秒)',
  PRIMARY KEY (`id`),
  KEY `idx_space_created` (`space_id`, `created_at`),
  KEY `idx_space_operator` (`space_id`, `operator_id`),
  KEY `idx_space_restype` (`space_id`, `resource_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='空间级操作审计日志';
```

- [ ] **Step 2: Commit**

```bash
git add docker/volumes/mysql/schema.sql
git commit -m "feat(operationlog): add operation_log table DDL"
```

---

## Task 2: domain entity 与常量

**Files:**
- Create: `backend/domain/operationlog/entity/operation_log.go`

- [ ] **Step 1: 写实体与常量**

```go
package entity

// Status 操作结果
type Status int32

const (
	StatusSuccess Status = 1
	StatusFail    Status = 2
)

// OperationLog 一条审计记录（落库 + 查询返回）
type OperationLog struct {
	ID             int64
	SpaceID        int64
	OperatorID     int64
	Module         string
	ResourceType   int32
	ResourceID     int64
	ResourceName   string
	Action         string
	Description    string
	Method         string
	Path           string
	RequestSummary string
	Status         Status
	ErrorCode      string
	ClientIP       string
	DurationMs     int32
	LogID          string
	CreatedAt      int64 // 毫秒
}

// Event 中间件采集后投递给 worker 的事件（未分配 ID）
type Event = OperationLog

// ListFilter 查询过滤条件
type ListFilter struct {
	SpaceID      int64
	OperatorID   *int64
	ResourceType *int32
	Action       *string
	StartTime    *int64 // 毫秒
	EndTime      *int64 // 毫秒
	Keyword      *string
	Page         int32
	PageSize     int32
}
```

- [ ] **Step 2: 编译**

Run: `cd backend && go build ./domain/operationlog/...`
Expected: 编译通过（无引用错误）

- [ ] **Step 3: Commit**

```bash
git add backend/domain/operationlog/entity/operation_log.go
git commit -m "feat(operationlog): add domain entity and constants"
```

---

## Task 3: gorm PO model

**Files:**
- Create: `backend/domain/operationlog/internal/dal/model/operation_log.go`

- [ ] **Step 1: 写 PO**

```go
package model

// OperationLog 是 operation_log 表的 gorm 持久化对象。
type OperationLog struct {
	ID             int64  `gorm:"column:id;primaryKey" json:"id"`
	SpaceID        int64  `gorm:"column:space_id" json:"space_id"`
	OperatorID     int64  `gorm:"column:operator_id" json:"operator_id"`
	Module         string `gorm:"column:module" json:"module"`
	ResourceType   int32  `gorm:"column:resource_type" json:"resource_type"`
	ResourceID     int64  `gorm:"column:resource_id" json:"resource_id"`
	ResourceName   string `gorm:"column:resource_name" json:"resource_name"`
	Action         string `gorm:"column:action" json:"action"`
	Description    string `gorm:"column:description" json:"description"`
	Method         string `gorm:"column:method" json:"method"`
	Path           string `gorm:"column:path" json:"path"`
	RequestSummary string `gorm:"column:request_summary" json:"request_summary"`
	Status         int32  `gorm:"column:status" json:"status"`
	ErrorCode      string `gorm:"column:error_code" json:"error_code"`
	ClientIP       string `gorm:"column:client_ip" json:"client_ip"`
	DurationMs     int32  `gorm:"column:duration_ms" json:"duration_ms"`
	LogID          string `gorm:"column:log_id" json:"log_id"`
	CreatedAt      int64  `gorm:"column:created_at" json:"created_at"`
}

func (OperationLog) TableName() string {
	return "operation_log"
}
```

- [ ] **Step 2: 编译**

Run: `cd backend && go build ./domain/operationlog/...`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add backend/domain/operationlog/internal/dal/model/operation_log.go
git commit -m "feat(operationlog): add gorm PO model"
```

---

## Task 4: DAO

**Files:**
- Create: `backend/domain/operationlog/internal/dal/dao.go`

- [ ] **Step 1: 写 DAO**

```go
package dal

import (
	"context"

	"gorm.io/gorm"

	"github.com/ynet-dev/ynet-studio/backend/domain/operationlog/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/operationlog/internal/dal/model"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/idgen"
)

type OperationLogDAO struct {
	db    *gorm.DB
	idgen idgen.IDGenerator
}

func NewOperationLogDAO(db *gorm.DB, idgen idgen.IDGenerator) *OperationLogDAO {
	return &OperationLogDAO{db: db, idgen: idgen}
}

// BatchCreate 批量写入；为每条记录分配雪花 ID。
func (dao *OperationLogDAO) BatchCreate(ctx context.Context, logs []*entity.OperationLog) error {
	if len(logs) == 0 {
		return nil
	}
	ids, err := dao.idgen.GenMultiIDs(ctx, len(logs))
	if err != nil {
		return err
	}
	pos := make([]*model.OperationLog, 0, len(logs))
	for i, l := range logs {
		pos = append(pos, &model.OperationLog{
			ID:             ids[i],
			SpaceID:        l.SpaceID,
			OperatorID:     l.OperatorID,
			Module:         l.Module,
			ResourceType:   l.ResourceType,
			ResourceID:     l.ResourceID,
			ResourceName:   l.ResourceName,
			Action:         l.Action,
			Description:    l.Description,
			Method:         l.Method,
			Path:           l.Path,
			RequestSummary: l.RequestSummary,
			Status:         int32(l.Status),
			ErrorCode:      l.ErrorCode,
			ClientIP:       l.ClientIP,
			DurationMs:     l.DurationMs,
			LogID:          l.LogID,
			CreatedAt:      l.CreatedAt,
		})
	}
	return dao.db.WithContext(ctx).CreateInBatches(pos, 100).Error
}

// List 按过滤条件分页查询，按 created_at 倒序。
func (dao *OperationLogDAO) List(ctx context.Context, f *entity.ListFilter) ([]*entity.OperationLog, int64, error) {
	q := dao.db.WithContext(ctx).Model(&model.OperationLog{}).Where("space_id = ?", f.SpaceID)
	if f.OperatorID != nil {
		q = q.Where("operator_id = ?", *f.OperatorID)
	}
	if f.ResourceType != nil {
		q = q.Where("resource_type = ?", *f.ResourceType)
	}
	if f.Action != nil {
		q = q.Where("action = ?", *f.Action)
	}
	if f.StartTime != nil {
		q = q.Where("created_at >= ?", *f.StartTime)
	}
	if f.EndTime != nil {
		q = q.Where("created_at <= ?", *f.EndTime)
	}
	if f.Keyword != nil && *f.Keyword != "" {
		kw := "%" + *f.Keyword + "%"
		q = q.Where("description LIKE ? OR resource_name LIKE ?", kw, kw)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page, size := f.Page, f.PageSize
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}

	var pos []*model.OperationLog
	if err := q.Order("created_at DESC").
		Offset(int((page - 1) * size)).Limit(int(size)).
		Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	out := make([]*entity.OperationLog, 0, len(pos))
	for _, p := range pos {
		out = append(out, &entity.OperationLog{
			ID: p.ID, SpaceID: p.SpaceID, OperatorID: p.OperatorID,
			Module: p.Module, ResourceType: p.ResourceType, ResourceID: p.ResourceID,
			ResourceName: p.ResourceName, Action: p.Action, Description: p.Description,
			Method: p.Method, Path: p.Path, RequestSummary: p.RequestSummary,
			Status: entity.Status(p.Status), ErrorCode: p.ErrorCode, ClientIP: p.ClientIP,
			DurationMs: p.DurationMs, LogID: p.LogID, CreatedAt: p.CreatedAt,
		})
	}
	return out, total, nil
}

// DeleteBefore 删除 created_at 小于 ts 的记录，分批避免大事务。返回删除条数。
func (dao *OperationLogDAO) DeleteBefore(ctx context.Context, ts int64) (int64, error) {
	var totalDeleted int64
	for {
		res := dao.db.WithContext(ctx).
			Where("created_at < ?", ts).
			Limit(1000).
			Delete(&model.OperationLog{})
		if res.Error != nil {
			return totalDeleted, res.Error
		}
		totalDeleted += res.RowsAffected
		if res.RowsAffected < 1000 {
			break
		}
	}
	return totalDeleted, nil
}
```

- [ ] **Step 2: 编译**

Run: `cd backend && go build ./domain/operationlog/...`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add backend/domain/operationlog/internal/dal/dao.go
git commit -m "feat(operationlog): add gorm DAO (batch create / list / delete-before)"
```

---

## Task 5: repository

**Files:**
- Create: `backend/domain/operationlog/repository/repository.go`

- [ ] **Step 1: 写仓储接口 + 实现**

```go
package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/ynet-dev/ynet-studio/backend/domain/operationlog/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/operationlog/internal/dal"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/idgen"
)

type OperationLogRepository interface {
	BatchCreate(ctx context.Context, logs []*entity.OperationLog) error
	List(ctx context.Context, f *entity.ListFilter) ([]*entity.OperationLog, int64, error)
	DeleteBefore(ctx context.Context, ts int64) (int64, error)
}

type operationLogRepo struct {
	dao *dal.OperationLogDAO
}

func NewOperationLogRepository(db *gorm.DB, idgen idgen.IDGenerator) OperationLogRepository {
	return &operationLogRepo{dao: dal.NewOperationLogDAO(db, idgen)}
}

func (r *operationLogRepo) BatchCreate(ctx context.Context, logs []*entity.OperationLog) error {
	return r.dao.BatchCreate(ctx, logs)
}

func (r *operationLogRepo) List(ctx context.Context, f *entity.ListFilter) ([]*entity.OperationLog, int64, error) {
	return r.dao.List(ctx, f)
}

func (r *operationLogRepo) DeleteBefore(ctx context.Context, ts int64) (int64, error) {
	return r.dao.DeleteBefore(ctx, ts)
}
```

- [ ] **Step 2: 编译**

Run: `cd backend && go build ./domain/operationlog/...`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add backend/domain/operationlog/repository/repository.go
git commit -m "feat(operationlog): add repository"
```

---

## Task 6: domain service 接口

**Files:**
- Create: `backend/domain/operationlog/service/operation_log.go`

- [ ] **Step 1: 写接口**

```go
package service

import (
	"context"

	"github.com/ynet-dev/ynet-studio/backend/domain/operationlog/entity"
)

// OperationLog 是审计日志领域服务。
type OperationLog interface {
	// Collect 非阻塞投递一条采集事件；缓冲满时丢弃并计数，绝不阻塞调用方。
	Collect(event *entity.Event)
	// List 分页查询。
	List(ctx context.Context, f *entity.ListFilter) ([]*entity.OperationLog, int64, error)
	// Start 启动后台落库 worker 与清理循环。
	Start(ctx context.Context)
	// DroppedCount 返回因缓冲满被丢弃的事件数（用于可观测/测试）。
	DroppedCount() int64
}
```

- [ ] **Step 2: 编译**

Run: `cd backend && go build ./domain/operationlog/...`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add backend/domain/operationlog/service/operation_log.go
git commit -m "feat(operationlog): add domain service interface"
```

---

## Task 7: domain service 实现（异步 worker + 查询 + 清理）

**Files:**
- Create: `backend/domain/operationlog/service/operation_log_impl.go`
- Test: `backend/domain/operationlog/service/operation_log_test.go`

- [ ] **Step 1: 写实现**

```go
package service

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/ynet-dev/ynet-studio/backend/domain/operationlog/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/operationlog/repository"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
)

// Config 控制缓冲、批量与保留期。
type Config struct {
	BufferSize    int           // channel 容量
	BatchSize     int           // 单批落库最大条数
	FlushInterval time.Duration // 定时 flush 间隔
	RetentionDays int           // 保留天数 [30,90]
	CleanupEvery  time.Duration // 清理任务周期
}

type operationLogSvc struct {
	repo    repository.OperationLogRepository
	cfg     Config
	ch      chan *entity.Event
	dropped int64
}

func NewOperationLog(repo repository.OperationLogRepository, cfg Config) OperationLog {
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 4096
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = time.Second
	}
	if cfg.RetentionDays < 30 {
		cfg.RetentionDays = 30
	}
	if cfg.RetentionDays > 90 {
		cfg.RetentionDays = 90
	}
	if cfg.CleanupEvery <= 0 {
		cfg.CleanupEvery = 24 * time.Hour
	}
	return &operationLogSvc{
		repo: repo,
		cfg:  cfg,
		ch:   make(chan *entity.Event, cfg.BufferSize),
	}
}

func (s *operationLogSvc) Collect(event *entity.Event) {
	if event == nil {
		return
	}
	select {
	case s.ch <- event:
	default:
		atomic.AddInt64(&s.dropped, 1)
	}
}

func (s *operationLogSvc) DroppedCount() int64 {
	return atomic.LoadInt64(&s.dropped)
}

func (s *operationLogSvc) List(ctx context.Context, f *entity.ListFilter) ([]*entity.OperationLog, int64, error) {
	return s.repo.List(ctx, f)
}

func (s *operationLogSvc) Start(ctx context.Context) {
	go s.runWorker(ctx)
	go s.runCleanup(ctx)
}

func (s *operationLogSvc) runWorker(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.FlushInterval)
	defer ticker.Stop()
	buf := make([]*entity.Event, 0, s.cfg.BatchSize)

	flush := func() {
		if len(buf) == 0 {
			return
		}
		if err := s.repo.BatchCreate(ctx, buf); err != nil {
			logs.CtxErrorf(ctx, "[operationlog] batch create failed: %v", err)
		}
		buf = buf[:0]
	}

	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case ev := <-s.ch:
			buf = append(buf, ev)
			if len(buf) >= s.cfg.BatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func (s *operationLogSvc) runCleanup(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.CleanupEvery)
	defer ticker.Stop()
	s.cleanupOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.cleanupOnce(ctx)
		}
	}
}

func (s *operationLogSvc) cleanupOnce(ctx context.Context) {
	cutoff := time.Now().Add(-time.Duration(s.cfg.RetentionDays) * 24 * time.Hour).UnixMilli()
	n, err := s.repo.DeleteBefore(ctx, cutoff)
	if err != nil {
		logs.CtxErrorf(ctx, "[operationlog] cleanup failed: %v", err)
		return
	}
	if n > 0 {
		logs.CtxInfof(ctx, "[operationlog] cleanup deleted %d records before %d", n, cutoff)
	}
}
```

- [ ] **Step 2: 写失败测试（fake repo 验证缓冲/丢弃/落库）**

```go
package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ynet-dev/ynet-studio/backend/domain/operationlog/entity"
)

type fakeRepo struct {
	mu       sync.Mutex
	created  []*entity.OperationLog
	deleted  int64
	deleteTS int64
}

func (f *fakeRepo) BatchCreate(ctx context.Context, logs []*entity.OperationLog) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.created = append(f.created, logs...)
	return nil
}
func (f *fakeRepo) List(ctx context.Context, _ *entity.ListFilter) ([]*entity.OperationLog, int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.created, int64(len(f.created)), nil
}
func (f *fakeRepo) DeleteBefore(ctx context.Context, ts int64) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deleteTS = ts
	return f.deleted, nil
}
func (f *fakeRepo) createdLen() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.created)
}

func TestCollectAndFlush(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewOperationLog(repo, Config{BufferSize: 10, BatchSize: 2, FlushInterval: 20 * time.Millisecond})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.Start(ctx)

	for i := 0; i < 3; i++ {
		svc.Collect(&entity.Event{SpaceID: 1, Action: "create"})
	}
	time.Sleep(100 * time.Millisecond)
	if repo.createdLen() != 3 {
		t.Fatalf("want 3 created, got %d", repo.createdLen())
	}
}

func TestCollectDropsWhenFull(t *testing.T) {
	repo := &fakeRepo{}
	// 不 Start ⇒ channel 不被消费，容量 2，投递 5 条 ⇒ 丢 3 条
	svc := NewOperationLog(repo, Config{BufferSize: 2, BatchSize: 2})
	for i := 0; i < 5; i++ {
		svc.Collect(&entity.Event{SpaceID: 1})
	}
	if got := svc.DroppedCount(); got != 3 {
		t.Fatalf("want 3 dropped, got %d", got)
	}
}
```

- [ ] **Step 3: 跑测试看失败/通过**

Run: `cd backend && go test ./domain/operationlog/service/ -run 'TestCollect' -v`
Expected: 实现完成后 PASS（若实现有误先看到 FAIL）

- [ ] **Step 4: Commit**

```bash
git add backend/domain/operationlog/service/
git commit -m "feat(operationlog): add async worker, query and cleanup service"
```

---

## Task 8: application 层（单例 + Collect + 查询 + 权限 + 名字回查）

**Files:**
- Create: `backend/application/operationlog/operation_log.go`
- Test: `backend/application/operationlog/operation_log_test.go`

参考：`application/space/space_diagnose.go`（全局 SVC、权限）、`application/base/ctxutil`。

`domain/user/service.User` 提供 `CheckMemberPermission(ctx, spaceID, userID) (isMember bool, roleType int32, canInvite, canManage bool, err error)`，以及 `GetUserInfo`/批量查名字的方法（实现时确认具体方法名，见 Step 1 注释）。

- [ ] **Step 1: 写 application service**

```go
package operationlog

import (
	"context"

	"gorm.io/gorm"

	"github.com/ynet-dev/ynet-studio/backend/application/base/ctxutil"
	"github.com/ynet-dev/ynet-studio/backend/domain/operationlog/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/operationlog/repository"
	oplogsvc "github.com/ynet-dev/ynet-studio/backend/domain/operationlog/service"
	usersvc "github.com/ynet-dev/ynet-studio/backend/domain/user/service"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/idgen"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

// OperationLogApplicationService 是全局单例，供中间件采集与 handler 查询使用。
type OperationLogApplicationService struct {
	DomainSVC oplogsvc.OperationLog
	userSVC   usersvc.User
}

var OperationLogApplicationSVC = &OperationLogApplicationService{}

// Init 在 application.Init 中调用，构造 domain service 并启动后台 worker。
func Init(ctx context.Context, db *gorm.DB, idgenSVC idgen.IDGenerator, userSVC usersvc.User, cfg oplogsvc.Config) *OperationLogApplicationService {
	repo := repository.NewOperationLogRepository(db, idgenSVC)
	domainSVC := oplogsvc.NewOperationLog(repo, cfg)
	domainSVC.Start(ctx)
	OperationLogApplicationSVC.DomainSVC = domainSVC
	OperationLogApplicationSVC.userSVC = userSVC
	return OperationLogApplicationSVC
}

// Collect 供中间件调用；未初始化时安全 no-op。
func (s *OperationLogApplicationService) Collect(event *entity.Event) {
	if s == nil || s.DomainSVC == nil {
		return
	}
	s.DomainSVC.Collect(event)
}

// LogItem 是查询返回的单条（含 operator_name）。
type LogItem struct {
	*entity.OperationLog
	OperatorName string
}

// ListOperationLogs 校验权限（仅 Owner/Admin）后查询，并批量回查 operator 名字。
func (s *OperationLogApplicationService) ListOperationLogs(ctx context.Context, f *entity.ListFilter) ([]*LogItem, int64, error) {
	uidPtr := ctxutil.GetUIDFromCtx(ctx)
	if uidPtr == nil {
		return nil, 0, errorx.New(errno.ErrOperationLogPermissionCode, errorx.KV("msg", "not logged in"))
	}
	_, _, _, canManage, err := s.userSVC.CheckMemberPermission(ctx, f.SpaceID, *uidPtr)
	if err != nil {
		return nil, 0, err
	}
	if !canManage {
		return nil, 0, errorx.New(errno.ErrOperationLogPermissionCode, errorx.KV("msg", "only space owner/admin can view operation logs"))
	}

	records, total, err := s.DomainSVC.List(ctx, f)
	if err != nil {
		return nil, 0, err
	}

	// 批量回查 operator 名字。具体方法名以 usersvc.User 实际签名为准
	// （如 GetUserInfos / MGetUserBasicInfo）。这里以 names map 表示结果。
	names := s.resolveOperatorNames(ctx, records)

	items := make([]*LogItem, 0, len(records))
	for _, r := range records {
		items = append(items, &LogItem{OperationLog: r, OperatorName: names[r.OperatorID]})
	}
	return items, total, nil
}

// resolveOperatorNames 去重 operator_id 后批量查名字。
// 实现：收集 distinct operator_id → 调 userSVC 批量查询 → 组装 map[int64]string。
func (s *OperationLogApplicationService) resolveOperatorNames(ctx context.Context, records []*entity.OperationLog) map[int64]string {
	idset := make(map[int64]struct{})
	for _, r := range records {
		if r.OperatorID > 0 {
			idset[r.OperatorID] = struct{}{}
		}
	}
	ids := make([]int64, 0, len(idset))
	for id := range idset {
		ids = append(ids, id)
	}
	out := make(map[int64]string, len(ids))
	if len(ids) == 0 {
		return out
	}
	// TODO(实现时替换为 usersvc.User 的真实批量查询方法：
	//   infos, err := s.userSVC.MGetUserInfo(ctx, ids)
	//   for _, in := range infos { out[in.UserID] = in.Name }
	// 若 userSVC 无批量方法，则逐个 GetUserInfo。返回 map。
	return out
}
```

> 注：`resolveOperatorNames` 里的批量查询方法名必须在实现时对照 `domain/user/service/user.go` 的真实接口替换（grep `GetUserInfo`/`MGet`）。这是本任务唯一需要现场确认的点。

- [ ] **Step 2: 加 errno 码**

Modify: `backend/types/errno/` 下新建 `operation_log.go`：

```go
package errno

func init() {
	// 复用 errorx 注册机制（与 space.go 一致，确认 space.go 是否有 register/init）
}

const (
	ErrOperationLogPermissionCode  = 112100001
	ErrOperationLogInvalidParamCode = 112100002
)
```

> 实现时先看 `types/errno/space.go` 的完整写法（是否需要 `code.Register`/msg 映射），照抄相同结构，避免漏注册导致错误信息为空。

- [ ] **Step 3: 写权限测试**

```go
package operationlog

import (
	"context"
	"testing"

	"github.com/ynet-dev/ynet-studio/backend/domain/operationlog/entity"
)

// 用一个最小 fake userSVC + fake domainSVC 验证：Member(canManage=false) 被拒。
// （实现时按 usersvc.User 接口造 fake；此处给出断言意图）
func TestListRejectsNonAdmin(t *testing.T) {
	// 构造 svc，使 CheckMemberPermission 返回 canManage=false
	// 期望 ListOperationLogs 返回权限错误
	_ = context.Background()
	_ = &entity.ListFilter{SpaceID: 1}
	// 断言 err != nil 且为 ErrOperationLogPermissionCode
}
```

> 该测试需配合 fake userSVC。若 `usersvc.User` 接口较大，使用接口的最小嵌入式 fake（只实现 `CheckMemberPermission`，其余 panic）。

- [ ] **Step 4: 编译 + 测试**

Run: `cd backend && go build ./application/operationlog/... && go test ./application/operationlog/... -v`
Expected: 编译通过；测试 PASS

- [ ] **Step 5: Commit**

```bash
git add backend/application/operationlog/ backend/types/errno/operation_log.go
git commit -m "feat(operationlog): add application service with permission and name resolution"
```

---

## Task 9: 路由白名单 + 语义映射

**Files:**
- Create: `backend/api/middleware/operationlog/route_registry.go`
- Test: `backend/api/middleware/operationlog/route_registry_test.go`

- [ ] **Step 1: 写 registry**

```go
package operationlog

import "strings"

// RouteRule 描述一条被审计的写接口及其语义映射。
type RouteRule struct {
	Method       string // POST/PUT/DELETE/PATCH
	PathPattern  string // 形如 /api/workflow/:id/update;段以 ':' 开头为通配
	Module       string
	ResourceType int32
	Action       string
	DescTemplate string // 如 "更新了工作流" 或含 {resource_name}
	// 抽取来源："body:<field>" / "query:<field>" / "path:<seg>"
	ResourceIDFrom   string
	ResourceNameFrom string
}

// rules 第一期覆盖：workflow / agent(app) / knowledge / plugin / database / prompt / space member。
// 具体路径以 backend/api/router/* 实际注册为准，实现时逐条核对补全。
var rules = []RouteRule{
	{Method: "POST", PathPattern: "/api/workflow_api/create", Module: "workflow", ResourceType: 6, Action: "create", DescTemplate: "创建了工作流", ResourceNameFrom: "body:name"},
	{Method: "POST", PathPattern: "/api/workflow_api/update", Module: "workflow", ResourceType: 6, Action: "update", DescTemplate: "更新了工作流", ResourceIDFrom: "body:workflow_id"},
	{Method: "POST", PathPattern: "/api/workflow_api/delete", Module: "workflow", ResourceType: 6, Action: "delete", DescTemplate: "删除了工作流", ResourceIDFrom: "body:workflow_id"},
	// ... agent / knowledge / plugin / database / prompt / space member 同样补全
}

// Match 返回命中的规则（method 大写比较），未命中返回 nil。
func Match(method, path string) *RouteRule {
	method = strings.ToUpper(method)
	for i := range rules {
		r := &rules[i]
		if r.Method == method && matchPath(r.PathPattern, path) {
			return r
		}
	}
	return nil
}

// matchPath 支持以 ':' 开头的通配段。
func matchPath(pattern, path string) bool {
	ps := strings.Split(strings.Trim(pattern, "/"), "/")
	xs := strings.Split(strings.Trim(path, "/"), "/")
	if len(ps) != len(xs) {
		return false
	}
	for i := range ps {
		if strings.HasPrefix(ps[i], ":") {
			continue
		}
		if ps[i] != xs[i] {
			return false
		}
	}
	return true
}
```

- [ ] **Step 2: 写测试**

```go
package operationlog

import "testing"

func TestMatchExact(t *testing.T) {
	r := Match("POST", "/api/workflow_api/update")
	if r == nil || r.Action != "update" {
		t.Fatalf("want update rule, got %+v", r)
	}
}

func TestMatchMiss(t *testing.T) {
	if r := Match("GET", "/api/workflow_api/list"); r != nil {
		t.Fatalf("GET/read should not match, got %+v", r)
	}
}

func TestMatchWildcard(t *testing.T) {
	rules = append(rules, RouteRule{Method: "DELETE", PathPattern: "/api/res/:id", Action: "delete"})
	if r := Match("DELETE", "/api/res/123"); r == nil || r.Action != "delete" {
		t.Fatalf("wildcard should match, got %+v", r)
	}
}
```

- [ ] **Step 3: 跑测试**

Run: `cd backend && go test ./api/middleware/operationlog/ -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add backend/api/middleware/operationlog/
git commit -m "feat(operationlog): add route registry with semantic mapping"
```

---

## Task 10: 字段抽取工具

**Files:**
- Modify: `backend/api/middleware/operationlog/route_registry.go`（追加抽取函数）
- Modify: `backend/api/middleware/operationlog/route_registry_test.go`（追加测试）

- [ ] **Step 1: 追加抽取函数**

在 `route_registry.go` 末尾追加：

```go
import "encoding/json" // 合并到文件顶部 import 块

// ExtractSpaceID 依次从 query / form / body json 中取 space_id（字符串数字均兼容）。
func ExtractSpaceID(query map[string]string, form map[string]string, bodyJSON map[string]any) int64 {
	if v, ok := query["space_id"]; ok {
		return parseInt64(v)
	}
	if v, ok := form["space_id"]; ok {
		return parseInt64(v)
	}
	if v, ok := bodyJSON["space_id"]; ok {
		return anyToInt64(v)
	}
	return 0
}

// ExtractBySpec 按 "body:field"/"query:field"/"path:seg" 取值；返回字符串。
func ExtractBySpec(spec string, query, form map[string]string, bodyJSON map[string]any, pathSegs map[string]string) string {
	if spec == "" {
		return ""
	}
	kind, key, ok := splitSpec(spec)
	if !ok {
		return ""
	}
	switch kind {
	case "query":
		return query[key]
	case "form":
		return form[key]
	case "path":
		return pathSegs[key]
	case "body":
		if v, ok := bodyJSON[key]; ok {
			return anyToStr(v)
		}
	}
	return ""
}

func splitSpec(spec string) (kind, key string, ok bool) {
	parts := strings.SplitN(spec, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func parseInt64(s string) int64 {
	var n int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int64(c-'0')
	}
	return n
}

func anyToInt64(v any) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case string:
		return parseInt64(t)
	case json.Number:
		i, _ := t.Int64()
		return i
	}
	return 0
}

func anyToStr(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatInt(int64(t), 10)
	}
	return ""
}
```

> 实现时把 `encoding/json`、`strconv`、`strings` 合并进文件顶部唯一 import 块。

- [ ] **Step 2: 追加测试**

```go
func TestExtractSpaceIDFromBody(t *testing.T) {
	got := ExtractSpaceID(nil, nil, map[string]any{"space_id": "123"})
	if got != 123 {
		t.Fatalf("want 123, got %d", got)
	}
}

func TestExtractBySpecBody(t *testing.T) {
	got := ExtractBySpec("body:name", nil, nil, map[string]any{"name": "wf1"}, nil)
	if got != "wf1" {
		t.Fatalf("want wf1, got %q", got)
	}
}
```

- [ ] **Step 3: 跑测试**

Run: `cd backend && go test ./api/middleware/operationlog/ -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add backend/api/middleware/operationlog/
git commit -m "feat(operationlog): add request field extraction helpers"
```

---

## Task 11: OperationLogMW 中间件

**Files:**
- Create: `backend/api/middleware/operation_log.go`
- Test: `backend/api/middleware/operation_log_test.go`

参考：`api/middleware/log.go`（body 读取）、`api/middleware/session.go`（ctxcache/ctxutil 取 uid）。

- [ ] **Step 1: 写中间件**

```go
package middleware

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"golang.org/x/net/context"

	oplogmw "github.com/ynet-dev/ynet-studio/backend/api/middleware/operationlog"
	"github.com/ynet-dev/ynet-studio/backend/application/base/ctxutil"
	"github.com/ynet-dev/ynet-studio/backend/application/operationlog"
	"github.com/ynet-dev/ynet-studio/backend/domain/operationlog/entity"
)

const opLogMaxSummary = 512

// OperationLogMW 采集命中路由白名单的写操作，异步落库。注册于 SessionAuthMW 之后。
func OperationLogMW() app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		start := time.Now()
		method := string(ctx.Request.Header.Method())
		path := string(ctx.Request.URI().PathOriginal())

		rule := oplogmw.Match(method, stripQuery(path))
		if rule == nil {
			ctx.Next(c)
			return
		}

		// 在 Next 前抓 body（Next 后业务可能已读取，但 Hertz 已缓冲，两处皆可；
		// 这里在 Next 前 parse 一次以拿到入参）。
		bodyBytes := ctx.Request.Body()

		ctx.Next(c)

		uidPtr := ctxutil.GetUIDFromCtx(c)
		if uidPtr == nil {
			return // 未登录写操作不审计
		}

		// 解析 query / body
		query := map[string]string{}
		ctx.QueryArgs().VisitAll(func(k, v []byte) { query[string(k)] = string(v) })
		bodyJSON := map[string]any{}
		if len(bodyBytes) > 0 {
			_ = json.Unmarshal(bodyBytes, &bodyJSON)
		}

		spaceID := oplogmw.ExtractSpaceID(query, nil, bodyJSON)
		resourceIDStr := oplogmw.ExtractBySpec(rule.ResourceIDFrom, query, nil, bodyJSON, nil)
		resourceName := oplogmw.ExtractBySpec(rule.ResourceNameFrom, query, nil, bodyJSON, nil)

		status := entity.StatusSuccess
		if ctx.Response.StatusCode() >= 400 {
			status = entity.StatusFail
		}

		summary := string(bodyBytes)
		if len(summary) > opLogMaxSummary {
			summary = summary[:opLogMaxSummary]
		}

		ev := &entity.Event{
			SpaceID:        spaceID,
			OperatorID:     *uidPtr,
			Module:         rule.Module,
			ResourceType:   rule.ResourceType,
			ResourceID:     oplogmw.ParseInt64Public(resourceIDStr),
			ResourceName:   resourceName,
			Action:         rule.Action,
			Description:    rule.DescTemplate,
			Method:         method,
			Path:           stripQuery(path),
			RequestSummary: summary,
			Status:         status,
			ClientIP:       ctx.ClientIP(),
			DurationMs:     int32(time.Since(start).Milliseconds()),
			LogID:          getLogID(ctx),
			CreatedAt:      time.Now().UnixMilli(),
		}
		operationlog.OperationLogApplicationSVC.Collect(ev)
	}
}

func stripQuery(p string) string {
	if i := strings.IndexByte(p, '?'); i >= 0 {
		return p[:i]
	}
	return p
}

// getLogID 从已有 log id 机制读取（与 SetLogIDMW 对齐，实现时确认 key）。
func getLogID(ctx *app.RequestContext) string {
	return ""
}
```

> 实现细节：
> - `oplogmw.ParseInt64Public` 是 Task 10 `parseInt64` 的导出版；在 route_registry.go 加 `func ParseInt64Public(s string) int64 { return parseInt64(s) }`。
> - `getLogID`：对照 `api/middleware/log.go` 的 `SetLogIDMW` 找到 log id 在 ctx 的存放 key，读出返回。
> - import 用 `context`：项目中 Hertz handler 签名首参是 `context.Context`；按 `log.go` 实际 import 路径对齐（可能是标准库 `context`）。

- [ ] **Step 2: 写中间件测试（用 Hertz 测试工具或直接测组装逻辑）**

最小测试：构造一个命中规则的假请求，断言 Collect 被调用且字段正确。若 Hertz 端到端测试成本高，则把"组装 Event"逻辑抽成纯函数 `buildEvent(...)` 单测。建议抽函数：

```go
// 在 operation_log.go 抽出可测纯函数
// func buildEvent(rule *oplogmw.RouteRule, method, path string, uid int64, query map[string]string, bodyJSON map[string]any, statusCode int, clientIP string, durationMs int32, logID string, now int64) *entity.Event

func TestBuildEventFields(t *testing.T) {
	rule := &oplogmw.RouteRule{Module: "workflow", ResourceType: 6, Action: "update", DescTemplate: "更新了工作流", ResourceIDFrom: "body:workflow_id"}
	ev := buildEvent(rule, "POST", "/api/workflow_api/update", 999,
		nil, map[string]any{"space_id": "7", "workflow_id": "55"}, 200, "1.2.3.4", 12, "lg1", 1000)
	if ev.SpaceID != 7 || ev.ResourceID != 55 || ev.OperatorID != 999 || ev.Status != entity.StatusSuccess {
		t.Fatalf("bad event: %+v", ev)
	}
}
```

> 重构建议：把 Step 1 里"解析 + 组装 Event"的部分提取为 `buildEvent(...)` 纯函数，中间件主体负责 IO（读 body / Next / Collect）。这样既可单测又保持中间件薄。

- [ ] **Step 3: 跑测试**

Run: `cd backend && go test ./api/middleware/ -run 'TestBuildEvent' -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add backend/api/middleware/operation_log.go backend/api/middleware/operation_log_test.go backend/api/middleware/operationlog/route_registry.go
git commit -m "feat(operationlog): add OperationLogMW collection middleware"
```

---

## Task 12: API model + handler + router

**Files:**
- Create: `backend/api/model/data/operationlog/operation_log.go`
- Create: `backend/api/handler/operationlog/operation_log_service.go`
- Create: `backend/api/router/operationlog/operation_log.go`
- Modify: `backend/api/router/register.go`

参考：`api/model/data/space/diagnose.go`、`api/handler/space/space_diagnose_service.go`、`api/router/space/space_diagnose.go`、`register.go`。

- [ ] **Step 1: 写请求/响应 model**

```go
package operationlog

import "github.com/ynet-dev/ynet-studio/backend/api/model/base"

type ListRequest struct {
	SpaceID      int64      `form:"space_id" json:"space_id,string"`
	OperatorID   *int64     `form:"operator_id" json:"operator_id,string,omitempty"`
	ResourceType *int32     `form:"resource_type" json:"resource_type,omitempty"`
	Action       *string    `form:"action" json:"action,omitempty"`
	StartTime    *int64     `form:"start_time" json:"start_time,omitempty"`
	EndTime      *int64     `form:"end_time" json:"end_time,omitempty"`
	Keyword      *string    `form:"keyword" json:"keyword,omitempty"`
	Page         int32      `form:"page" json:"page"`
	PageSize     int32      `form:"page_size" json:"page_size"`
	Base         *base.Base `json:"Base,omitempty"`
}

type LogItemDTO struct {
	ID           int64  `json:"id,string"`
	OperatorID   int64  `json:"operator_id,string"`
	OperatorName string `json:"operator_name"`
	Module       string `json:"module"`
	ResourceType int32  `json:"resource_type"`
	ResourceID   int64  `json:"resource_id,string"`
	ResourceName string `json:"resource_name"`
	Action       string `json:"action"`
	Description  string `json:"description"`
	Status       int32  `json:"status"`
	ClientIP     string `json:"client_ip"`
	CreatedAt    int64  `json:"created_at"`
}

type ListResponse struct {
	Code int64        `json:"code"`
	Msg  string       `json:"msg"`
	Logs []LogItemDTO `json:"logs"`
	Total int64       `json:"total"`
}
```

> 确认 `api/model/base` 包路径与 diagnose.go 中 `base.Base` 一致。

- [ ] **Step 2: 写 handler**

```go
package operationlog

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"github.com/ynet-dev/ynet-studio/backend/api/internal/httputil"
	model "github.com/ynet-dev/ynet-studio/backend/api/model/data/operationlog"
	app_oplog "github.com/ynet-dev/ynet-studio/backend/application/operationlog"
	"github.com/ynet-dev/ynet-studio/backend/domain/operationlog/entity"
)

// ListOperationLog 查询空间操作日志，仅 Owner/Admin。
// @router /api/operation_log/list [POST]
func ListOperationLog(ctx context.Context, c *app.RequestContext) {
	var req model.ListRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	f := &entity.ListFilter{
		SpaceID:      req.SpaceID,
		OperatorID:   req.OperatorID,
		ResourceType: req.ResourceType,
		Action:       req.Action,
		StartTime:    req.StartTime,
		EndTime:      req.EndTime,
		Keyword:      req.Keyword,
		Page:         req.Page,
		PageSize:     req.PageSize,
	}

	items, total, err := app_oplog.OperationLogApplicationSVC.ListOperationLogs(ctx, f)
	if err != nil {
		httputil.InternalError(ctx, c, err)
		return
	}

	dtos := make([]model.LogItemDTO, 0, len(items))
	for _, it := range items {
		dtos = append(dtos, model.LogItemDTO{
			ID: it.ID, OperatorID: it.OperatorID, OperatorName: it.OperatorName,
			Module: it.Module, ResourceType: it.ResourceType, ResourceID: it.ResourceID,
			ResourceName: it.ResourceName, Action: it.Action, Description: it.Description,
			Status: int32(it.Status), ClientIP: it.ClientIP, CreatedAt: it.CreatedAt,
		})
	}

	c.JSON(consts.StatusOK, &model.ListResponse{Code: 0, Msg: "success", Logs: dtos, Total: total})
}
```

- [ ] **Step 3: 写 router**

```go
package operationlog

import (
	"github.com/cloudwego/hertz/pkg/app/server"
	handler "github.com/ynet-dev/ynet-studio/backend/api/handler/operationlog"
)

// Register wires the operation-log query endpoint.
//   POST /api/operation_log/list
func Register(r *server.Hertz) {
	g := r.Group("/api/operation_log")
	g.POST("/list", handler.ListOperationLog)
}
```

- [ ] **Step 4: 在 register.go 注册**

Modify `backend/api/router/register.go`：import 处加
```go
operationlog "github.com/ynet-dev/ynet-studio/backend/api/router/operationlog"
```
在与 `space.Register(r)` 同区域加：
```go
operationlog.Register(r)
```

- [ ] **Step 5: 编译**

Run: `cd backend && go build ./...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add backend/api/model/data/operationlog/ backend/api/handler/operationlog/ backend/api/router/operationlog/ backend/api/router/register.go
git commit -m "feat(operationlog): add query API model, handler and route"
```

---

## Task 13: 全局 wiring（init + 中间件注册 + env）

**Files:**
- Modify: `backend/application/application.go`（`initComplexServices` 末尾）
- Modify: `backend/main.go`（中间件链）

- [ ] **Step 1: 在 application.go 初始化 operation log service**

在 `initComplexServices` 末尾、`return nil` 之前追加（参考 `spaceapp.InitDiagnoseService` 的写法）：

```go
operationlog.Init(
	ctx,
	infra.DB,
	infra.IDGenSVC,
	basicServices.userSVC.DomainSVC,
	oplogsvc.Config{
		BufferSize:    envInt("OPERATION_LOG_BUFFER_SIZE", 4096),
		RetentionDays: envInt("OPERATION_LOG_RETENTION_DAYS", 90),
	},
)
```

import 加：
```go
"github.com/ynet-dev/ynet-studio/backend/application/operationlog"
oplogsvc "github.com/ynet-dev/ynet-studio/backend/domain/operationlog/service"
```

`envInt` helper：若项目已有读 env 的工具则复用（grep `os.Getenv` + strconv 的现成 helper）；否则在 application.go 加：

```go
func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
```

> `OPERATION_LOG_ENABLED` 开关可选：若为 "false" 则跳过 `operationlog.Init`，中间件因 `Collect` 的 nil 守卫自动 no-op。

- [ ] **Step 2: 在 main.go 注册中间件**

Modify `backend/main.go`，在 `s.Use(middleware.I18nMW())` 之后追加：

```go
s.Use(middleware.OperationLogMW()) // after SessionAuthMW: needs uid in ctx
```

- [ ] **Step 3: 编译**

Run: `cd backend && go build ./...`
Expected: PASS

- [ ] **Step 4: 跑全量测试**

Run: `cd backend && go test ./domain/operationlog/... ./application/operationlog/... ./api/middleware/operationlog/... ./api/middleware/ -run 'Operation|Collect|Match|Extract|BuildEvent|List' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/application/application.go backend/main.go
git commit -m "feat(operationlog): wire service init and register middleware"
```

---

## Task 14: 前端 — 操作日志页与入口

**Files:**
- 定位并修改 workspace 子菜单：`frontend/packages/foundation/space-ui-adapter/src/components/workspace-sub-menu/index.tsx`
- 新建页面组件（与 `frontend/apps/coze-studio/src/pages/space-members.tsx` 同级或对应 workspace 包）
- API 调用：`frontend/packages/arch/api-schema/src/idl/operation_log/`（或在页面内直接 fetch `/api/operation_log/list`）

> 本任务前端结构需先在仓库内定位现有"空间成员"页与菜单的真实组织方式（已知存在 `space-members.tsx` 与 `workspace-sub-menu`）。以下为实现指引，具体 import/路由按定位结果调整。

- [ ] **Step 1: 定位现有空间页与菜单**

Run:
```bash
cd frontend && grep -rn "space-members\|空间成员\|SpaceMember" apps/coze-studio/src/pages packages/foundation/space-ui-adapter --include="*.tsx" | head
```
Expected: 找到菜单项注册处与成员页路由，记录其新增菜单项/路由的方式。

- [ ] **Step 2: 加菜单入口「操作日志」（仅 Owner/Admin 可见）**

在 workspace 子菜单按现有项的写法追加一项，指向新页面路由（如 `/space/:space_id/operation-log`）。可见性条件复用菜单里已有的当前用户空间角色判断（与"邀请成员"等管理项相同的 gate）。

- [ ] **Step 3: 新建操作日志页面组件**

包含：筛选栏（操作人输入、资源类型下拉、动作下拉、时间范围、关键词）+ 分页表格（时间 / 操作人 / 模块 / 资源 / 描述 / 状态）。调用 `POST /api/operation_log/list`，请求体含 `space_id` 与筛选项；用现有 UI 组件库（与 space-members 页一致的 Table/Pagination/Select）。

请求示例（按项目现有 api client 封装方式调整）：
```ts
const resp = await fetch('/api/operation_log/list', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ space_id: spaceId, page, page_size: pageSize, ...filters }),
});
const data = await resp.json(); // { logs, total }
```

- [ ] **Step 4: 构建验证**

Run（按项目 Rush 命令）：
```bash
cd frontend && rush build -t @coze-studio/app  # 或对应包；具体 target 以定位结果为准
```
Expected: 构建通过

- [ ] **Step 5: Commit**

```bash
git add frontend/
git commit -m "feat(operationlog): add space operation-log page and menu entry"
```

---

## Task 15: 端到端手动验证

**Files:** 无（验证步骤）

- [ ] **Step 1: 起后端（对接 220/226 远程中间件）**

按项目方式启动后端（参考 `docker/.env.debug` 对接远程 MySQL/Redis；不在本地起 middleware）。确认 `operation_log` 表已由自愈建出：
```bash
# 在能连 DB 的环境
mysql ... -e "SHOW TABLES LIKE 'operation_log';"
```
Expected: 返回 operation_log

- [ ] **Step 2: 触发写操作**

登录前端，在某空间创建/更新/删除一个工作流（命中白名单的接口）。

- [ ] **Step 3: 校验落库**

```bash
mysql ... -e "SELECT space_id,operator_id,module,action,description,status,created_at FROM operation_log ORDER BY created_at DESC LIMIT 5;"
```
Expected: 出现刚才的操作记录，字段正确（space_id/operator_id/action 等）。

- [ ] **Step 4: 校验查询接口与权限**

- 用 Owner/Admin 账号打开「操作日志」页：能看到记录，筛选/分页正常。
- 用普通成员账号调用 `/api/operation_log/list`：返回权限错误（`ErrOperationLogPermissionCode`）。

- [ ] **Step 5: 校验保留期清理（可选快速验证）**

临时设 `OPERATION_LOG_RETENTION_DAYS` 较小值并触发 `cleanupOnce`（或插入一条 created_at 很旧的记录后等清理周期），确认过期记录被删除。

- [ ] **Step 6: 最终提交（如有收尾修改）**

```bash
git add -A && git commit -m "test(operationlog): manual e2e verification fixes"
```

---

## Self-Review 检查结果

- **Spec 覆盖**：采集中间件(Task 9-11,13) / 路由白名单+语义映射(Task 9) / 异步落库(Task 7) / MySQL表(Task 1,3) / 保留期清理env可配(Task 7,13) / 查询API(Task 12) / 权限仅Owner-Admin(Task 8) / operator_name查询回查(Task 8) / 前端页面(Task 14) / env配置(Task 13) / 测试(Task 7-11,15) — 全部有对应任务。
- **占位符**：实现中有两处明确标注"现场确认"——(a) Task 8 `resolveOperatorNames` 的 user 批量查询真实方法名；(b) Task 11 `getLogID` 的 ctx key 与 errno 注册写法。均给出了定位命令与参照文件，非空泛 TODO。
- **类型一致性**：`entity.Event = entity.OperationLog`，`Collect(*entity.Event)`、`List(*entity.ListFilter)`、`OperationLogApplicationSVC`、`Match`/`ExtractSpaceID`/`ExtractBySpec`/`ParseInt64Public`、`buildEvent` 在各任务间签名一致。
