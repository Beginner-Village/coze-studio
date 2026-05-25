# 空间级 ES 重新同步功能 设计

- **Date**: 2026-05-25
- **Author**: luzhipeng (with Claude)
- **Status**: Draft，已与 user 口头确认设计方向
- **Scope**: 单 spec 单实施周期（1-2 天）

## 1. 背景

POC R2 上线后，多次出现 **ES 跟 MySQL 不同步** 引发的客户事故：

1. 从 POC dump 数据到现场，**ES 写入时 JS 大整数精度截断**，所有 ID 错了几百块 → 前端列表空 / 500
2. 客户业务方在 MySQL 直接改数据（绕过 Studio API），ES 没有跟着更新 → 前端看到过期数据
3. ES bulk 写入中途网络抖，部分 doc 丢失，无回滚机制
4. 没有任何"重建 ES 索引"的运维入口，每次都靠手撸 shell 脚本

需要一个**平台内置的"重新同步 ES"按钮**，让运营人员一键修复，避免每次都求开发。

## 2. 范围（Scope）

✅ **包含**：
- 按 **当前 space** 全量同步 4 类 ES 索引
- 后端新 API：`POST /api/space/resync_es`
- 前端入口：空间设置页（`space-management`）"数据维护" 区块 + 二次确认 modal
- 阻塞式：列表索引（project_draft / coze_resource / kb_entries）当场重写完返回；KB chunk 走异步 IndexSliceEvent（复用现有链路 + SliceStatusBadge 显示进度）
- 权限：仅 space owner 或 admin

❌ **不包含**（拆到后续）：
- 系统级（全 space）reindex
- 单资源 / 单 KB 颗粒度同步
- 后台任务系统 / 进度条
- ES → MySQL 反向 sync（数据对账）

## 3. 现状摸底

### 后端
- `backend/domain/search/service/`：现有 ES 写入入口
  - [`handler_project.go`](backend/domain/search/service/handler_project.go) — 智能体/project ES 写入
  - [`handler_resource.go`](backend/domain/search/service/handler_resource.go) — 资源 ES 写入
  - [`eventbus.go`](backend/domain/search/service/eventbus.go) — event-driven 入口
  - [`search.go`](backend/domain/search/service/search.go) — search service 接口
  - [`service.go`](backend/domain/search/service/service.go) — service 实现
- 触发点：业务侧（agent / resource / KB）变化时 publish event → search service 写 ES
- **没有现成的 Reindex/Resync API**（grep 0 结果）

### 知识库 chunk
- `knowledge_document_slice.status` = `Init` (0) / `Processing` / `Done` (1) / `Failed` / `Deactive`
- 编辑切片 → `UpdateSlice` 改 status=Init + 发 `IndexSliceEvent` → `indexSlice` consumer 处理
- 已有完整的"重新生成向量 + 写 ES" 链路，**可复用**

### 前端
- `frontend/apps/coze-studio/src/pages/space-management/index.tsx` — 空间列表+管理页主入口
- 已有 `CreateSpaceModal.tsx` / `ImportModal.tsx` / `MemberModal.tsx` 可参考 modal 样式
- Spec A 已实现 `SliceStatusBadge` + `useSliceStatusPolling` → re-embedding 进度可视化

## 4. 设计

### 4.1 整体数据流

```
[空间设置页 "数据维护" 卡片]
     │
     │ 点击 "重新同步 ES 索引"
     ▼
[确认 modal: "将清空并重建本空间所有 ES 索引..."]
     │ 用户确认
     ▼
POST /api/space/resync_es { space_id }
     │
     ▼
[backend: SpaceApplicationService.ResyncES]
     │
     ├─► 1. delete_by_query { term: { space_id: X } } 在 4 个 ES 索引
     │     - project_draft / coze_resource / kb_entries / openynet_<kb_id...>
     │
     ├─► 2. 扫 MySQL 重写列表索引（同步阻塞）
     │     - single_agent_draft + project → project_draft
     │     - app_resource → coze_resource
     │     - knowledge → kb_entries
     │
     ├─► 3. 找本 space 所有 KB → 对每个 KB 的所有 slice：
     │     - UPDATE knowledge_document_slice SET status=Init WHERE knowledge_id IN (...)
     │     - publish IndexSliceEvent for each（异步走 indexSlice consumer）
     │
     └─► 返回 { counts: { project_draft, coze_resource, kb_entries, slice_reindex_jobs } }
     
[前端]
     │ 收到 200 响应
     ▼
toast 显示各 index 重建条数
     │
     │ 用户可立刻进知识库列表/智能体列表，列表数据 immediately 可见 (列表 ES 已重写)
     │ 知识库内切片状态: 通过现有 SliceStatusBadge 显示"重新索引中" → 1-5 min 后变 Done
```

### 4.2 后端实现

#### 4.2.1 新增 handler

**`backend/api/handler/coze/space_resync_es.go`** （新建）

```go
// ResyncSpaceES .
// @router /api/space/resync_es [POST]
func ResyncSpaceES(ctx context.Context, c *app.RequestContext) {
    var req space.ResyncESRequest
    if err := c.BindAndValidate(&req); err != nil {
        c.String(consts.StatusBadRequest, err.Error())
        return
    }
    resp, err := application.SpaceSVC.ResyncES(ctx, &req)
    if err != nil {
        internalServerErrorResponse(ctx, c, err)
        return
    }
    c.JSON(consts.StatusOK, resp)
}
```

#### 4.2.2 IDL/types

`backend/api/model/data/space/resync.go`（新建）：
```go
type ResyncESRequest struct {
    SpaceID int64 `thrift:"space_id,1,required" json:"space_id"`
}

type ResyncESResponse struct {
    Code   int64               `json:"code"`
    Msg    string              `json:"msg"`
    Counts *ResyncESCounts     `json:"counts"`
}

type ResyncESCounts struct {
    ProjectDraft      int `json:"project_draft"`
    CozeResource      int `json:"coze_resource"`
    KbEntries         int `json:"kb_entries"`
    SliceReindexJobs  int `json:"slice_reindex_jobs"`
}
```

#### 4.2.3 application service

**`backend/application/space/resync_es.go`**（新建）

```go
func (s *SpaceApplicationService) ResyncES(ctx context.Context, req *space.ResyncESRequest) (*space.ResyncESResponse, error) {
    // 1. 权限校验:当前 user 是 space owner 或 admin
    if err := s.assertSpacePermission(ctx, req.SpaceID); err != nil {
        return nil, err
    }

    // 2. 复用 search domain 的能力扫描 MySQL 重写 ES
    counts, err := s.searchSvc.ResyncSpace(ctx, req.SpaceID)
    if err != nil {
        return nil, errorx.New(errno.ErrSpaceResyncESCode, errorx.KV("msg", err.Error()))
    }

    // 3. 触发本 space 所有 KB chunk 重新 embedding
    sliceJobs, err := s.knowledgeSvc.ResyncSpaceSlices(ctx, req.SpaceID)
    if err != nil {
        logs.CtxWarnf(ctx, "slice resync 部分失败,err: %v", err)
        // 不阻断:列表 ES 已经成功了
    }
    counts.SliceReindexJobs = sliceJobs

    return &space.ResyncESResponse{Code: 0, Counts: counts}, nil
}
```

#### 4.2.4 search domain 新增方法

**`backend/domain/search/service/resync.go`**（新建）

```go
// ResyncSpace drops all ES docs for given space + replays writes from MySQL.
func (s *searchSvc) ResyncSpace(ctx context.Context, spaceID int64) (*ResyncESCounts, error) {
    counts := &ResyncESCounts{}

    // (a) delete_by_query 4 个列表索引按 space_id
    for _, idx := range []string{"project_draft", "coze_resource", "kb_entries"} {
        if err := s.searchStore.DeleteByQuery(ctx, idx, map[string]any{
            "term": map[string]any{"space_id": spaceID},
        }); err != nil {
            return nil, fmt.Errorf("delete_by_query %s: %w", idx, err)
        }
    }

    // (b) 扫 MySQL 重写 project_draft
    drafts, err := s.agentRepo.ListBySpaceID(ctx, spaceID)
    if err != nil { return nil, err }
    for _, d := range drafts {
        if err := s.indexProjectDraft(ctx, d); err != nil { continue }
        counts.ProjectDraft++
    }

    // (c) 扫 MySQL 重写 coze_resource
    resources, err := s.resourceRepo.ListBySpaceID(ctx, spaceID)
    if err != nil { return nil, err }
    for _, r := range resources {
        if err := s.indexResource(ctx, r); err != nil { continue }
        counts.CozeResource++
    }

    // (d) 扫 MySQL 重写 kb_entries
    kbs, err := s.knowledgeRepo.ListBySpaceID(ctx, spaceID)
    if err != nil { return nil, err }
    for _, kb := range kbs {
        if err := s.indexKbEntry(ctx, kb); err != nil { continue }
        counts.KbEntries++
    }

    return counts, nil
}
```

#### 4.2.5 knowledge domain 新增方法

**`backend/domain/knowledge/service/resync_slices.go`**（新建）

```go
// ResyncSpaceSlices marks all slices in space's KBs as Init + publishes IndexSliceEvent for each.
// Returns count of slices queued for re-embedding.
func (k *knowledgeSVC) ResyncSpaceSlices(ctx context.Context, spaceID int64) (int, error) {
    kbs, err := k.knowledgeRepo.ListBySpaceID(ctx, spaceID)
    if err != nil { return 0, err }

    total := 0
    for _, kb := range kbs {
        slices, err := k.sliceRepo.ListByKnowledgeID(ctx, kb.ID)
        if err != nil { continue }

        ids := make([]int64, 0, len(slices))
        for _, s := range slices { ids = append(ids, s.ID) }
        if err := k.sliceRepo.BatchSetStatus(ctx, ids, int32(SliceStatusInit), ""); err != nil { continue }

        for _, s := range slices {
            event := events.NewIndexSliceEvent(s, kb.Document)
            body, _ := sonic.Marshal(event)
            k.producer.Send(ctx, body, eventbus.WithShardingKey(strconv.FormatInt(s.DocumentID, 10)))
            total++
        }
    }
    return total, nil
}
```

### 4.3 前端实现

#### 4.3.1 空间设置页新增 "数据维护" section

**`frontend/apps/coze-studio/src/pages/space-management/components/DataMaintenanceSection.tsx`**（新建）

```tsx
export const DataMaintenanceSection = ({ spaceId }: { spaceId: string }) => {
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [loading, setLoading] = useState(false);

  const handleResync = async () => {
    setLoading(true);
    try {
      const resp = await SpaceApi.ResyncES({ space_id: spaceId });
      Toast.success(
        `同步完成:智能体 ${resp.counts.project_draft} / 资源 ${resp.counts.coze_resource} / 知识库 ${resp.counts.kb_entries} / 切片重新索引中 ${resp.counts.slice_reindex_jobs}`
      );
    } catch (e) {
      Toast.error(`同步失败: ${e.message}`);
    } finally {
      setLoading(false);
      setConfirmOpen(false);
    }
  };

  return (
    <Section title="数据维护">
      <Card>
        <div className="title">重新同步 ES 索引</div>
        <div className="desc">
          清空本空间所有 ES 索引，从 MySQL 重新写入。适用于:列表数据展示不全 / 检索结果跟实际不符 / 从备份导入数据后。
        </div>
        <Button danger onClick={() => setConfirmOpen(true)}>重新同步</Button>
      </Card>
      <ConfirmModal
        open={confirmOpen}
        title="确认重新同步 ES 索引?"
        content="将清空本空间所有 ES 索引并从 MySQL 重写。期间 1-5 分钟内列表查询可能为空,知识库检索结果可能不完整。仅 admin / space owner 可执行。"
        onConfirm={handleResync}
        onCancel={() => setConfirmOpen(false)}
        loading={loading}
      />
    </Section>
  );
};
```

#### 4.3.2 接入 space-management 主页面

`pages/space-management/index.tsx` 顶部加 `<DataMaintenanceSection spaceId={currentSpaceId} />`，权限不足时隐藏整个 section。

### 4.4 错误处理

| 场景 | 行为 |
|---|---|
| 用户无权限（非 owner / 非 admin）| 403 + toast "无权限执行此操作" |
| delete_by_query 部分失败 | 整体回滚不可能（ES 不支持 transaction），记 error log，返回部分计数 + warn message |
| MySQL 扫描失败 | 返回 500，列表索引可能处于部分重建状态。用户可重试 |
| IndexSliceEvent publish 失败 | 列表 ES 已成功，slice 重建 skip。返回 warn 让用户单独再点一次（幂等）|
| 知识库切片 re-embedding 中途失败 | 走现有 `SliceStatusBadge.Failed` 流程，每个 chunk 单独重试 |

### 4.5 并发与一致性

- **不引入分布式锁**：一个 space 同时被两人点"重新同步"概率极低；ES delete_by_query + write 是幂等的，重复执行没害
- **不引入 task 队列**：列表索引 5-30 秒完成，HTTP 阻塞够用；超 30 秒由前端 timeout 报错让用户重试
- **slice re-embedding 复用现有 MQ**：天然异步、可重试、负载均衡

### 4.6 测试

| 类型 | 内容 |
|---|---|
| **后端单测** | `searchSvc.ResyncSpace` mock searchStore 返回 + assert delete_by_query 3 次 + index 调用次数 |
| **后端单测** | `knowledgeSvc.ResyncSpaceSlices` mock sliceRepo + producer，assert status 改 Init + event 发送数量 |
| **后端集成测试** | 在 220 dev 跑一遍 ResyncES，verify ES 数据跟 MySQL 一致 |
| **前端单测** | `DataMaintenanceSection` 渲染 + 点按钮触发 confirm + confirm 触发 API + loading 状态 |
| **手动 E2E** | 在 POC space 制造 ES/MySQL 不一致（手动 curl 删几条 ES doc），点重新同步，验证恢复 |

## 5. 风险清单

| Risk | 缓解 |
|---|---|
| delete_by_query 慢（KB 多的 space）| `?refresh=false&wait_for_completion=true` + 控制 batch size |
| `space_id` 字段没在某些索引里（hash 截断历史 bug）| 跑前先 verify mappings 含 space_id 字段；缺则跳过该 index 并 warn |
| KB chunk 多（万级）一次性 publish 大量 IndexSliceEvent → MQ 压力 | rate limit publish（每 100 个 sleep 100ms）|
| 列表索引部分重建状态（MySQL scan 中途失败）| 报 500 提示用户重试。下次成功会完整覆盖 |
| 跨 space 数据泄露（delete_by_query 漏掉条件）| 单元测试断言 query 必须含 `term.space_id` |

## 6. 工作量

| 项 | 时间 |
|---|---|
| 后端 API + service + repo 方法 | 0.5-1 天 |
| 前端 section + modal + i18n | 0.5 天 |
| 联调 + 220 dev 验证 + 手动 e2e | 0.5 天 |
| **合计** | **1.5-2 天** |

## 7. 实施阶段（写 plan 时）需 verify 的开放问题

1. `agentRepo.ListBySpaceID` / `resourceRepo.ListBySpaceID` / `knowledgeRepo.ListBySpaceID` 等 repo 方法**是否已存在**；不存在需要新增
2. `searchStore.DeleteByQuery` 方法**是否已存在**（infra 层）；不存在需要新增
3. POC ES 是否所有索引都有 `space_id` 字段（之前 dump 看 mappings 时 `audit_logs` 等可能没有）；缺字段则该 index 跳过
4. 前端 `space-management/index.tsx` 现有结构能否平滑插入新 section
5. `SpaceApi.ResyncES` 前端 SDK 需要新生成（thrift IDL → ts client）
