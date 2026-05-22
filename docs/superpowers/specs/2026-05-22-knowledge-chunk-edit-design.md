# 知识库切片编辑 + 自动重新 Embedding 设计（A 最安全版）

- **Date**: 2026-05-22
- **Author**: luzhipeng (with Claude)
- **Status**: Draft，待 review
- **Branch**: `feat/wip`（待重命名）

## 1. 背景

客户反馈：
- 知识库切片之后内容不能编辑
- 需要"修改完后自动更新切片向量"，下次召回看到改动后的内容
- 客户原话："实时改动、实时更新，下次命中就能看到改动后的能力"

补充：客户视角下"所有切片都是文本，都应该能改"——不区分文档类型。

## 2. 范围（Scope）

**本 spec 仅覆盖 A scope（最安全方式）。**

✅ 包含：
- text-workspace / image-workspace 加切片编辑入口（table-workspace 已有）
- 编辑后自动 re-embedding（复用已有链路）
- 切片状态徽章（重新索引中 / 失败 / 完成）
- 编辑失败重试

❌ 不包含（拆到后续 spec）：
- Version history / 回滚（C scope）
- Diff view（C scope）
- 批量编辑（C scope）
- 编辑 Drawer + SSE 推送（B scope）
- 召回质量优化（rerank / chunk size / query rewrite 调参，独立 spec）

## 3. 现状摸底（已 verify）

### 后端 ✅ 完整，本次不动

| 层 | 文件 | 函数 |
|---|---|---|
| handler | [backend/api/handler/coze/knowledge_service.go:410](backend/api/handler/coze/knowledge_service.go:410) | `UpdateSlice` |
| application | [backend/application/knowledge/knowledge.go:563](backend/application/knowledge/knowledge.go:563) | `KnowledgeApplicationService.UpdateSlice` |
| domain | [backend/domain/knowledge/service/knowledge.go:785](backend/domain/knowledge/service/knowledge.go:785) | `knowledgeSVC.UpdateSlice` |
| consumer | [backend/domain/knowledge/service/event_handle.go:518](backend/domain/knowledge/service/event_handle.go:518) | `indexSlice` |

链路：
```
UpdateSlice 内部：
  DB: slice.status = SliceStatusInit
  DB: slice.content = new content
  MQ: publish IndexSliceEvent

indexSlice consumer：
  embedding 调用
  向量库 upsert
  DB: slice.status = SliceStatusDone
```

### Slice status 枚举 ✅ 已有

[backend/domain/knowledge/internal/dal/model/progress.go:23](backend/domain/knowledge/internal/dal/model/progress.go:23):
- `SliceStatusInit` (0) — 待向量化
- `SliceStatusProcessing` (?) — 处理中
- `SliceStatusDone` (1) — 已完成
- `SliceStatusFailed` (?) — 失败（带 reason）
- `SliceStatusDeactive` (9) — 停用

### 前端

| Workspace | 切片编辑入口 |
|---|---|
| `table-knowledge-workspace` | ✅ 已有 `useTableSegmentModal`（[table-data-view.tsx:62-63](frontend/packages/data/knowledge/knowledge-ide-base/src/features/table-knowledge-workspace/components/table-data-view.tsx)）|
| `text-knowledge-workspace` | ❌ 没有 |
| `image-knowledge-workspace` | ❌ 没有 |

可复用组件：[knowledge-modal-base/table-segment-modal/hooks.tsx:112](frontend/packages/data/knowledge/knowledge-modal-base/src/table-segment-modal/hooks.tsx:112) `useTableSegmentModal`。

## 4. 设计

### 4.1 整体数据流

```
[前端 Modal 编辑] → POST /api/knowledge/slice/update (handler 已有)
                  → application.UpdateSlice (已有)
                  → domain.UpdateSlice (已有)
                      ├ DB: slice.status = Init
                      ├ DB: slice.content = new
                      └ MQ: publish IndexSliceEvent
                  → indexSlice consumer (已有)
                      ├ embedding 调用
                      ├ 向量库 upsert
                      └ DB: slice.status = Done

[前端 5s 轮询 /api/knowledge/slice/list?ids=...]
   → 看到 status=Done → 停转圈
   → 看到 status=Failed → 显示重试按钮
```

### 4.2 后端改动：**预期 0 行**

预期完全不动后端代码。**唯一可能的例外**：`/api/knowledge/slice/list` 若不支持按 `slice_ids[]` 批量查询，需要补一个 endpoint 或扩展现有 endpoint 的入参（见 §7 第 4 项 verify）。若真要补，控制在 < 20 行后端代码。

### 4.3 Retrieval 不加 status filter（关键安全决策）

**当前**：[backend/domain/knowledge/service/retrieve.go:188](backend/domain/knowledge/service/retrieve.go:188) 只过滤 `DocumentStatusEnable`，没过滤 slice-level status。

**决策**：**这一波不加**。理由：
1. 主表 `slice.content` 编辑后立刻更新，召回命中后拉到的就是**新内容**（仅向量分数是老的，可能排序不准）
2. 加 filter 的 risk 反而更大：可能有历史"僵尸切片"（embedding 历史失败、status 卡在 Init/Failed 但客户没意识到的）被一刀切，召回结果集突变
3. 0.5–3 秒的"分数老但内容新"窗口客户可接受
4. 未来若发现"老向量命中导致排序差"是真问题，单独评估

### 4.4 DB 改动：**0 行**

### 4.5 前端改动清单

| # | 改动 | 文件/位置 |
|---|---|---|
| 1 | text-workspace 切片列表行尾加"编辑"按钮 | [features/text-knowledge-workspace/components/](frontend/packages/data/knowledge/knowledge-ide-base/src/features/text-knowledge-workspace/components/) 下的列表组件（impl plan 阶段定位） |
| 2 | image-workspace 切片列表行尾加"编辑"按钮 | [features/image-knowledge-workspace/](frontend/packages/data/knowledge/knowledge-ide-base/src/features/image-knowledge-workspace/) 下（impl plan 阶段 verify 结构）|
| 3 | 点击 → 打开 Modal，复用 `useTableSegmentModal` | impl plan 阶段 verify Modal 是否硬绑 table 字段；如硬绑则重构成通用 `chunk-content-modal` |
| 4 | 新增 `SliceStatusBadge` 小组件 | 建议放 [knowledge-modal-base/](frontend/packages/data/knowledge/knowledge-modal-base/) 或就近 |
| 5 | 新增 `useSliceStatusPolling` hook | 建议放 [knowledge-modal-base/](frontend/packages/data/knowledge/knowledge-modal-base/) |

### 4.6 UI 行为

1. 切片行尾 "编辑" 按钮 → 弹既有 Modal → 改 content → 点保存
2. Modal 关闭 → 该行徽章 = "重新索引中"（转圈）
3. 前端调既有 `/api/knowledge/slice/list` API **每 5 秒一次**，仅请求被编辑过的 slice_ids（批量）
4. 状态变 `Done` → 转圈消失（静默，不打 toast）
5. 状态变 `Failed` → 显示"重试"按钮 + hover tip 显示失败原因
   - 点击重试 → 用**原 content** 重新调 `UpdateSlice`
6. 轮询超时 120 秒 → 停止轮询，显示"长时间未完成，请刷新页面"

### 4.7 错误处理

| 场景 | 行为 |
|---|---|
| `UpdateSlice` 返回非 2xx | Modal 不关闭，toast "保存失败：xxx"，用户可重试或取消 |
| `UpdateSlice` 内部 MQ 推送失败 | 后端会回非 2xx（[knowledge.go:851](backend/domain/knowledge/service/knowledge.go:851)） → 同上 toast |
| 轮询期间 status 变 `Failed` | 行尾显示"重试"按钮 + 失败原因 hover |
| 轮询 120 秒还没 Done | 停止轮询，徽章变灰"长时间未完成" |
| 用户在 status=Init/Processing 时再次点编辑 | 按钮 disabled，hover tip "正在重新索引，请稍候" |
| 同一页面同时编辑多个切片 | 共用 1 个 timer，请求批量带 `slice_ids[]` |

### 4.8 并发与一致性

- **不引入乐观锁**。客户场景：一个客户运营人员维护知识库，并发编辑同一切片的概率极低，沿用 last-write-wins
- **轮询节流**：同一页面只起 1 个 timer；被编辑过的 slice_ids 维护成 Set；Set 空了就停 timer
- **页面切换**：Drawer/Modal 关掉、路由切走 → cleanup timer 避免内存泄漏

### 4.9 测试

| 类型 | 内容 |
|---|---|
| **手动 E2E（必做）** | text / table / image 三种 workspace 各跑一遍：上传文件 → 切片完成 → 编辑某 chunk → 保存 → 看徽章转圈 → 等 Done → 在该知识库 retrieval 查询命中是**新内容** |
| 前端单测 | `useSliceStatusPolling` hook（5s 间隔 / Set 维护 / 自动停 / cleanup）|
| 前端单测 | `SliceStatusBadge` 三种状态渲染 |
| 回归 | table-workspace 原有编辑功能不受影响（同一个 Modal）|

## 5. 风险清单

| Risk | 说明 | 缓解 |
|---|---|---|
| 复用 `useTableSegmentModal` 可能不支持 text/image | Modal 内部可能 hardcode table 字段（列名）| **impl plan 阶段先读 Modal 源码**；若硬绑 → 重构成通用 chunk-content-modal |
| 轮询请求量上升 | 每 5 秒 × N 个浏览器标签页 | 短期不大；若客户反馈卡顿，下一波升级到 SSE（B scope）|
| 老向量短窗排序不准 | 用户编辑完立刻搜索可能命中老向量分数但拉到新文本 | 接受；告知客户可等几秒再查 |
| image-workspace 现有 UI 结构未知 | 还没 verify 切片列表是否跟 text-workspace 同结构 | **impl plan 阶段先 verify** |

## 6. 工作量估算

| 项 | 时间 |
|---|---|
| 前端（复用 Modal + text/image 接入 + 徽章 + 轮询）| 2–3 天 |
| 后端 | 0 |
| DB | 0 |
| 联调测试 | 1 天 |
| **合计** | **3–4 天** |

## 7. 实施阶段（写 plan 时）需 verify 的开放问题

1. `useTableSegmentModal` 内部是否硬绑 table 字段（列名等）—— 决定是直接复用还是重构成通用 modal
2. image-workspace 切片列表的结构（行是否能挂"编辑"按钮）
3. text-workspace 切片列表的结构同上
4. `/api/knowledge/slice/list` API 是否支持按 `slice_ids` 批量查询（如不支持，需要后端补 endpoint—— 这会破坏"后端 0 改动"的承诺，要提前 verify）
