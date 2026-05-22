# Spec A：切片链路 Verify + 合并 + 状态徽章

- **Date**: 2026-05-22
- **Author**: luzhipeng (with Claude)
- **Status**: Draft v2（v1 假设错被推翻，本版基于 verify 重写）
- **Branch**: `feat/wip`
- **Related**: [Spec B - 召回效果优化](./2026-05-22-retrieval-quality-optimization.md)

## 1. 背景与 v1 推翻

v1 假设"text/image-workspace 没编辑入口"。verify 后发现：

| Workspace | 入口 | 调用链 |
|---|---|---|
| text | `LevelTextKnowledgeEditor` → `useUpdateRemoteChunk` → `KnowledgeApi.UpdateSlice` | UpdateSlice → domain.UpdateSlice → DB+MQ → indexSlice consumer → embedding → 向量库 |
| table | `useTableSegmentModal` → `KnowledgeApi.UpdateSlice` | 同上 |
| image | `usePhotoDetailModal` → `KnowledgeApi.UpdatePhotoCaption` → application.UpdatePhotoCaption 内部转调 `DomainSVC.UpdateSlice`（[knowledge.go:1015](backend/application/knowledge/knowledge.go:1015)）| 同上 |

**三种类型理论上都已工作**，客户痛点"更新分片可能没做好"需要 e2e verify 找 broken 环节。

客户新增需求：**切片合并**（合并相邻多个 chunk）。

## 2. 范围

✅ 包含（P0 → P1）：
- Phase 1: e2e verify re-embedding 链路 + 修任何 broken 环节
- Phase 2: 切片合并（merge 相邻多个 chunk）
- Phase 3: 状态徽章 + 5s 轮询

❌ 不包含：
- 召回效果优化 → 见 [Spec B](./2026-05-22-retrieval-quality-optimization.md)

## 3. 设计

### 3.1 Phase 1: Re-embedding 链路 e2e Verify

**测试矩阵**（每个 workspace 都要跑）：

| Workspace | 编辑动作 | 期望可观测到的事件链 |
|---|---|---|
| text | 改某 chunk content | ① DB slice.status=Init ② DB slice.content=new ③ MQ event 投递 ④ indexSlice 消费 ⑤ embedding 调用成功 ⑥ 向量库 upsert 成功 ⑦ DB slice.status=Done ⑧ retrieval 拿到新内容 |
| table | 改某 row 列值 | 同上 |
| image | 改 caption | 同上（注意 application 层 `listResp.Slices[0]` 只更新第一片，image 通常每张图 1 片，OK）|

**验证手段**：
- dev 环境跑三种 e2e
- `indexSlice` 函数开头/embedding 调用前后/向量库 upsert 前后**临时加日志**（commit 前清掉）
- 检查 MQ consumer 是否注册 + 启动（grep `indexSlice` 注册点）
- 检查 embedding 配置（model key、endpoint）能不能正常调通
- retrieval 测试：编辑后等 status=Done，再调 retrieval API 看返回内容

**修复**：
- 工作量未知（0 ~ 50 行后端代码），取决于 broken 程度
- 每个 broken 一个 commit，message 注明 root cause

### 3.2 Phase 2: 切片合并

#### 设计选择：**前端两步**（推荐）

| 方案 | 优 | 缺 | 选择 |
|---|---|---|---|
| A. 后端新增 `MergeSlices(ids[])` atomic API | 原子性强、易测试 | 新增 ~50 行后端 + 新 IDL | ❌ |
| B. **前端两步：UpdateSlice(target, 拼接 content) + DeleteSlice(others)** | 复用现有 API、后端 0 改动 | 非原子（delete 失败 → 残留多余 chunk）| ✅ **推荐** |

理由：符合"少改后端"原则；半成功 risk 小（残留切片可读，比丢内容安全）。

#### UI 行为

1. 三种 workspace 的切片行尾加 checkbox（image 默认不显示合并入口，因为每张图 1 片）
2. 多选后顶部出现"合并"按钮
3. 按 `sequence` 连续性判断是否可合并；不连续则按钮 disabled + tooltip "只能合并相邻切片"
4. 点击 → 弹合并预览 Modal（显示拼接后内容预览）
5. 确定 → loading → step1 `UpdateSlice(min_seq_id, content=按 sequence 拼接)` → step2 `DeleteSlice(其他 ids)` → 列表刷新

#### 错误处理

| 场景 | 行为 |
|---|---|
| step1 UpdateSlice 失败 | toast "合并失败"，不做 step2，DB 完全没变 |
| step1 成功 step2 DeleteSlice 失败 | 重试 1 次 DeleteSlice；再失败 → toast "合并部分完成，多余切片需手动删除"，**不 rollback** UpdateSlice |

### 3.3 Phase 3: 状态徽章 + 轮询

- 切片行尾加 `SliceStatusBadge` 组件，三态：`Init/Processing → 转圈`，`Failed → 红色叹号 + 重试按钮`，`Done → 不显示`
- 新增 `useSliceStatusPolling(docId, watchedIds)` hook，每 5s 调 `ListSlice(document_id)`，前端 filter `watchedIds` 的 status，全部 Done 则停 timer
- 编辑或合并后启动轮询
- 轮询超时 120s → 停轮询，徽章变灰"长时间未完成"

## 4. 后端 / DB 改动

| 项 | 改动量 |
|---|---|
| 后端代码（Phase 1 修复） | **未知**：取决于 verify 结果。若链路完全正常 → 0 行；若发现 broken → 视 root cause 而定（如 consumer 没注册、embedding 配置错、向量库 upsert 失败等，每种 fix 量不同）。**这是 spec 内唯一可能动后端的位置**。 |
| 后端代码（Phase 2 合并） | **0**（前端两步方案） |
| 后端代码（Phase 3 徽章） | **0** |
| DB | **0** |

## 5. 前端改动清单

| # | 改动 | 大致位置 |
|---|---|---|
| 1 | 三个 workspace 切片行加 checkbox 多选 + 顶部"合并"按钮 | `features/{text,table,image}-knowledge-workspace/` 的列表组件 |
| 2 | 合并预览 Modal `MergeSliceConfirmModal` | `knowledge-modal-base/` |
| 3 | 合并 hook `useMergeSlices`（含 retry + error handling） | `knowledge-modal-base/` |
| 4 | 状态徽章 `SliceStatusBadge` | `knowledge-modal-base/` |
| 5 | 轮询 hook `useSliceStatusPolling` | `knowledge-modal-base/` |

## 6. 测试

| 类型 | 内容 |
|---|---|
| Phase 1 手动 e2e | text/table/image 三种各跑：编辑 → 看 DB + MQ + 向量库 + retrieval |
| 单测 | `useSliceStatusPolling`（5s 间隔 / Set 维护 / 自动停 / cleanup）|
| 单测 | `useMergeSlices`（成功路径 / step1 失败 / step2 失败 + retry）|
| 单测 | `SliceStatusBadge` 三态渲染 |
| 集成测 | 合并 2 个 chunk → ListSlice 确认只剩 1 个 + content 是拼接结果 |
| 回归 | table-workspace 原有编辑/删除不受影响 |

## 7. 风险

| Risk | 缓解 |
|---|---|
| Phase 1 verify 发现重大 broken（如 indexSlice consumer 没注册）| 工作量爆炸；先做 Phase 1 stop，跟用户重新评估 Phase 2/3 优先级 |
| ListSlice 不支持按 `slice_ids[]` 过滤 | 轮询拉整个 doc（< 1000 切片，几百 KB 可接受） |
| 合并后 sequence 跳号 | target 切片继承最小 sequence，其他切片 DELETE 后空出 |
| 客户在 `Processing` 时再点编辑/合并 | 按钮 disabled + tooltip "正在重新索引，请稍候" |

## 8. 工作量
- Phase 1 verify + 可能修复：1–2 天
- Phase 2 合并：2–3 天
- Phase 3 徽章+轮询：1–2 天
- **合计：4–7 天**

## 9. 写 plan 时需 verify
1. `ListSlice` 响应字段是否包含 `status` + `content`（必须才能前端 filter）
2. 三个 workspace 列表行的具体组件文件路径
3. `text-knowledge-editor` 是否支持外部传入"行尾插槽"用于挂徽章和 checkbox（否则要 wrap）
4. image-workspace `usePhotoDetailModal` 内部是否 expose "保存完成" callback 给轮询触发
