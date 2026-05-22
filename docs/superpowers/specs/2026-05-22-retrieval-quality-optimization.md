# Spec B：知识库召回效果优化

- **Date**: 2026-05-22
- **Author**: luzhipeng (with Claude)
- **Status**: Draft（调研驱动型，具体方案待 Phase 0 落地）
- **Branch**: `feat/wip`（或独立分支）
- **Related**: [Spec A - 切片链路](./2026-05-22-knowledge-chunk-edit-design.md)

## 1. 背景

客户反馈："知识库召回效果很差，命中和向量的处理感觉有问题"。

**Bad case 锚点**：
- query = `aaa` → 召回关于 `AIxxxxx` 的内容，**且得分还很高**

这种典型假阳性需要在本 spec 内 100% 解决（acceptance gate）。

## 2. 范围

✅ 包含：
- Phase 0：调研 + bad case 收集 + root cause 分析
- Phase 1：快速止血（query 预过滤 + score threshold）
- Phase 2：中等调优（hybrid 融合权重、rerank topk）
- Phase 3：深度优化（embedding 模型、chunk size、query rewrite 等，按需）

❌ 不包含：
- 切片编辑/合并 → [Spec A](./2026-05-22-knowledge-chunk-edit-design.md)
- 新模型上线（如换 embedding model）：本 spec 仅做 verify 与建议，不强制换

## 3. 现状摸底（已 verify）

### 现有 retrieval 链路

[`backend/domain/knowledge/service/retrieve.go::Retrieve`](backend/domain/knowledge/service/retrieve.go)：

```
Query → queryRewriteNode → 并行：
                              ├─ vectorRetrieveNode (向量召回)
                              ├─ esRetrieveNode (BM25 全文)
                              └─ nl2sqlRetrieveNode (NL2SQL 表)
                          → reRankNode (合并 + rerank)
                          → packResults
```

**已知**：
- `SearchType` 三态：0=semantic / 1=fulltext / 2=hybrid
- 支持配置 `TopK` / `MinScore`
- 已有 rerank node
- **未 verify**：当前部署的 embedding 模型、rerank 模型、BM25 参数

### 未 verify 项（Phase 0 必查）
- 当前 query 是否有任何**预过滤**（长度、纯重复字符）→ 大概率没有
- 当前**最低分阈值**是否真的生效（MinScore 是否被设为 0）
- BM25 doc-length 归一化参数（k1, b）
- 向量 + ES 融合权重
- 当前 embedding 模型与配置

## 4. 设计（Phase 化）

### Phase 0：调研（必做，不调优代码）

**1. Bad case 收集**（≥ 20 条，至少覆盖 5 类）：
- 极短 query（1–3 字符）
- 纯重复字符（`aaa`、`xxx`）
- 拼写错（如客户域内词的错别字）
- 同义但召回不到（应命中没命中）
- 应不命中但命中了高分

来源：客户提供 + 现场日志挖取（看现有 retrieval log 拿 query）

**2. 跑分析脚本**：每条 bad case 跑一遍 retrieval，dump 中间结果：
- 向量召回 top10 + score
- ES 召回 top10 + score  
- 融合后 top10
- rerank 后 top10
- query rewrite 后的 query 是什么

**3. Root cause 分析**：对每条 bad case 归类是哪个环节出问题（embedding / BM25 / 融合 / rerank / query rewrite）

**4. 输出**：
- 调研报告 `docs/superpowers/research/2026-MM-DD-retrieval-bad-cases.md`
- 优先级排序的调优 backlog（哪些 Phase 1 处理、哪些 Phase 2、哪些 Phase 3）

**工作量**：1–2 天

### Phase 1：快速止血（不引入新模型/不动配置）

不需要 Phase 0 全部完成就能动手的"普世修复"：

| 修复 | 文件大致位置 | 代码量 |
|---|---|---|
| **query 长度/质量预过滤**：长度 < 3 字符、纯重复字符、纯标点 → 直接返回 empty results | `retrieve.go::Retrieve` 入口 | < 30 行 |
| **score threshold 强制下限**：rerank 输出 < `MinScore`（如 0.3）→ filter，配置可调 | `reRankNode` 输出后 | < 20 行 |
| **空结果优雅返回**：query 被预过滤掉时返回 `empty + log`，前端可显示"请输入更具体的关键词"提示 | 同上 | < 10 行 |

**Acceptance**：bad case `aaa` 不再返回结果（被预过滤拦截）。

**工作量**：1–2 天

### Phase 2：中等调优（基于 Phase 0 结论）

- 调 hybrid 召回的 vector / BM25 融合权重（如改 RRF k 值、改加权融合系数）
- 调 rerank topk（从 100 调到 30 等）
- 调 BM25 参数（k1=1.2 / b=0.75 → 实验值）
- 看是否需要 query rewrite 增强（短 query 加扩展词）

**工作量**：2–3 天 + 多轮回归

### Phase 3：深度优化（按需，依赖 Phase 0 调研结论）

| 方向 | 触发条件 | 工作量 |
|---|---|---|
| 换 embedding 模型 | Phase 0 发现 embedding 对短/中文/领域词不友好 | 3–5 天（含历史数据重新向量化）|
| chunk size / overlap 调整 | Phase 0 发现 chunk 太大/太小导致命中粒度问题 | 2–3 天（含重切现有数据）|
| 加 metadata filter（knowledge_id / document_type）| Phase 0 发现跨知识库串扰 | 1–2 天 |
| 加 query rewrite（用 LLM 扩展同义）| Phase 0 发现召回率低 | 2–3 天 |

## 5. 评估指标 & Acceptance Gate

| 指标 | 目标 |
|---|---|
| **准确率**：bad case 集合不再返回错误结果 | 100% bad case 被 Phase 1 + Phase 2 解决 |
| **召回率**：good case 仍能命中（不能 regression）| good case 命中率 ≥ 调优前 |
| **排序**：Top1 hit rate / NDCG@5 | NDCG@5 提升 ≥ 5%（按 Phase 0 评估集）|
| **性能**：P99 latency | 不增加 > 20% |

**Acceptance gate**：bad case `aaa` 返回 empty（必须，最低门槛）。

## 6. 测试

- **Golden set**: 30–50 条 query + 期望命中 chunk id 列表
- **自动化评估脚本** `scripts/eval-retrieval.go`，每次调优后跑一遍，对比指标
- **回归**：现有单元测试 + 集成测试全部通过

## 7. 风险

| Risk | 缓解 |
|---|---|
| Bad case 不够代表性 → 调优 overfit | 让客户在 Phase 0 提供 ≥ 20 条真实 case；自己再从日志挖 10 条 |
| Phase 3 换 embedding 影响历史向量 | 切换时双写过渡（保留旧向量库 N 天），灰度 retrieval 路径 |
| query 预过滤太激进，误伤合理短 query（如缩写、品名）| 配置化阈值，可调；前端给客户输入提示 |
| Phase 0 结论暴露的根本问题超出本 spec 能力 | 拆 issue 出来单独评估，不强行塞本 spec |

## 8. 工作量
| Phase | 工作量 |
|---|---|
| Phase 0 调研 | 1–2 天 |
| Phase 1 止血 | 1–2 天 |
| Phase 2 调优 | 2–3 天 |
| Phase 3 深度（按需）| 0–5 天 |
| **合计** | **5–10 天** |

## 9. 优先级 & 交付节奏

- **Sprint 1**: Phase 0 + Phase 1（保证 bad case `aaa` 被解决）→ 客户立刻能用
- **Sprint 2**: Phase 2（基于 Phase 0 结论调优）
- **Sprint 3**（按需）: Phase 3 深度优化

## 10. 写 plan 时需 verify
1. 当前 embedding 模型名称 + endpoint 配置（在 `infra/contract/embedding` 找）
2. 当前 rerank 模型 + topk 设置
3. `MinScore` 配置入口在哪、当前值
4. retrieval 日志是否记录 query + top results（用于挖 bad case）
5. 是否有现成的评估脚本/golden set 可复用
