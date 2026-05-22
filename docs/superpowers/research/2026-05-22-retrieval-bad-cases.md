# 知识库召回 Phase 0 调研报告

- **Date**: 2026-05-22
- **Spec**: [Spec B - retrieval-quality-optimization](../specs/2026-05-22-retrieval-quality-optimization.md)
- **Status**: Phase 0 完成（仅调研、未改代码）

## 1. 调研背景

**锚点 bad case**：`query = "aaa"` → 召回到关于 AI 的内容且**得分很高**（典型假阳性）。
**目标**：Phase 0 摸清 retrieval pipeline 与默认参数，给出根因假设 + 调优 backlog，Phase 1 必须把 `aaa` 这类 query 拦住。

## 2. Retrieval Pipeline 现状

### 2.1 流程图

```
RetrieveRequest                                          retrieve.go:56
  └─> newRetrieveContext (load enabled docs/knowledge)   retrieve.go:110
  └─> Chain.Invoke
        ├─ queryRewriteNode (LLM 改写, 默认关)            retrieve.go:198
        └─ Parallel:
            ├─ vectorRetrieveNode (跳过当 SearchType=1)    retrieve.go:221
            │     └─ retrieveChannels                      retrieve.go:285
            │           └─ embedder.EmbedStrings(query)
            │           └─ store.Retrieve  (Milvus/OB/ES)
            ├─ esRetrieveNode  (跳过当 SearchType=0)       retrieve.go:253
            │     └─ es.NewMatchQuery(content, query)      es_searchstore.go:108
            ├─ nl2SqlRetrieveNode (仅 Table+Enable)        retrieve.go:347
            └─ passRequestContext                          retrieve.go:495
        └─ reRankNode                                      retrieve.go:499
              · selectedReranker = k.reranker (RRF默认) 或 space 模型 rerank
              · Rerank(Query, Data=[[vec],[es],...], TopN=Strategy.TopK)
              · filter item.Score < MinScore               retrieve.go:579
        └─ packResults (DB MGet, hit count++)              retrieve.go:590
```

### 2.2 关键参数（默认值，含源码引用）

| 参数 | 默认值 | 出处 |
|---|---|---|
| `Strategy.SearchType` | `0 = Semantic` | `api/model/crossdomain/knowledge/knowledge.go:139-145` |
| `Strategy.TopK` | `1`（agent 默认） / 注释 1-10 default 3 | `application/singleagent/create.go:108`、`knowledge.go:120` |
| `Strategy.MinScore` | **`0.01`**（agent 默认） / 注释 default 0.5 | `application/singleagent/create.go:109,143` |
| `EnableRerank` | `true`（agent 模板默认） | `application/singleagent/create.go:113` |
| `EnableQueryRewrite` | `true`（agent 模板默认） | `application/singleagent/create.go:114` |
| Milvus default topK | `4` | `searchstore/milvus/consts.go:21` |
| OB default topK | `4` | `searchstore/oceanbase/consts.go:21` |
| ES default topK | `10` | `searchstore/elasticsearch/consts.go:20` |
| RRF k | `60`（传 0 走默认） | `rerank/rrf/rrf.go:30` |
| ES BM25 k1/b | 未在代码层显式设置，走 ES 默认 `k1=1.2 / b=0.75` | `elasticsearch_searchstore.go:108` 仅 `NewMatchQuery` |
| Hybrid 融合 | RRF（多通道 list 直接进 RRF） | `retrieve.go:535-550` + `rrf.go:39` |
| Vector→ES 融合权重 | 无显式 weight，纯 rank 倒数 | `rrf.go:49` |

### 2.3 当前部署的 embedding / rerank

- **Embedding type**: `EMBEDDING_TYPE="ark"` (`backend/.env:88`)，`ARK_EMBEDDING_DIMS=2048` (`.env:101`)。`ARK_EMBEDDING_MODEL/AK/BASE_URL` 为空 → 实际线上靠 DB 里 SpaceEmbedding 配置覆盖（`SpaceEmbeddingProvider`, `space_embedding_provider.go:60`）；fallback HTTP embedding 走 `HTTP_EMBEDDING_ADDR="http://127.0.0.1:6543"`（自研服务）。
- **Vector store**: `VECTOR_STORE_TYPE="milvus"`，`MILVUS_ADDR=10.10.10.224:19530`（生产实际走 OB，按 [project_architecture] OB 已替代 Milvus）。
- **Rerank**: 默认 `RRFReranker(k=60)` (`init.go:184`)；若 space 配置了模型 rerank（如 jina openai-compatible，`rerank/openai/openai.go`），优先使用（`retrieve.go:561`）。**无环境变量显式开关**。
- **Score 归一化**：OB `1 - cosine_distance/2` → 落在 [0,1]（`ob_searchstore.go:284`）；Milvus 对 IP/COSINE 走 `(score+1)/2`（`milvus_searchstore.go:507`）；ES `score/firstScore` → top1 永远归一化为 **1.0**（`elasticsearch_searchstore.go:262`，**严重问题，见 §3**）。

## 3. Bad case 根因假设

针对 `aaa → AI 内容、高分`：

1. **Query 预处理侧 — 0 防护**：`retrieve.go:60` 只校验 `len(Query)==0`，无任何长度/重复字符/纯标点过滤；rewriteNode 在无 ChatHistory 或 `EnableQueryRewrite=false` 时直接透传（`retrieve.go:199-206`）。`aaa` 原封不动进下游。
2. **Embedding 侧**：`aaa` 这种 OOV 且无语义的 3 字符 token，ark/HTTP embedder 大概率退化到训练分布均值附近（"unknown 向量"）；与"AIxxxxx" 共享高频英文/字母 cluster，cosine 相似度天然不低（典型 0.6+）。`SpaceEmbedder` 无任何 query 长度过滤（`space_embedding_provider.go:150`）。
3. **ES BM25 侧**：`NewMatchQuery(content, "aaa")` 走 ES 默认 standard analyzer，对中文按字切。`aaa` 若被分成单 token，且某文档恰含 "aaa"/"aab" 等，BM25 在短 query 上分数极易冲高；又 ES retrieve 把 top1 强归一化为 **score/firstScore = 1.0**（`elasticsearch_searchstore.go:262`），**任何 ES 召回的 top1 score 都是 1.0**，后续 RRF 看到的输入根本不是相对相关度。
4. **融合侧**：RRF 只看 rank 位置不看 score（`rrf.go:49 score = 1/(rank+60)`），所以 vector top1 + ES top1 都拿到 `1/61 ≈ 0.0164`。两路都"召回到"就赢，不区分相关性。最终 reranked score 都是 0.0164 量级，`MinScore=0.01` 阈值毫无作用。
5. **Rerank 侧**：默认 RRF 不看语义（见上）；若启用模型 rerank（jina 类），`openai/openai.go:67` 会把全部召回 doc.Content 喂给模型 rerank，模型对 `aaa` 与 AI 长文档的 relevance 行为 unknown，可能仍给 0.3+；且**模型输出未做下限过滤**，直接进 `MinScore`。
6. **结果处理侧**：`retrieve.go:579` 仅 `< MinScore` 过滤，**`MinScore` agent 默认 0.01**（`create.go:109,143`），等于没拦住任何东西。注释写 "default 0.5" 但代码实际 0.01，注释与现实不一致。

**结论**：5 个环节叠加，`aaa` 高分召回是**必然结果**，不是偶发 bug。

## 4. 10 个 Synthetic Bad Case

| # | 类 | Query | 期望 | 测试方法 |
|---|---|---|---|---|
| 1 | A 极短 | `a` | empty | 直接 POST 知识库 retrieval，验证返回 length=0 |
| 2 | A 极短 | `好` | empty 或仅命中精确含"好"且 score>0.5 的 chunk | 同上 |
| 3 | B 重复 | `aaa` | empty（锚点 case） | 同上，验证 score 全部 <0.3 或返回空 |
| 4 | B 重复 | `!!!` | empty | 同上 |
| 5 | C 应不命中 | `量子纠缠`（KB 中无相关内容） | empty 或 score<0.4 | 与未启用知识库的响应对比 |
| 6 | C 应不命中 | `xyz123abc`（随机字符串） | empty | 同上 |
| 7 | D 拼写错 | `知识苦`（应是"知识库"） | 命中"知识库"相关 chunk | 与正确 query 结果对比，比对 chunk_id 集合 |
| 8 | D 拼写错 | `vetcor`（应是 vector） | 命中 vector 相关 chunk | 同上 |
| 9 | E 同义未召 | `搜索增强`（KB 用词是 RAG/检索增强生成） | 命中 RAG 相关 chunk | 验证 top1 是预期 chunk_id |
| 10 | E 同义未召 | `LLM 上下文窗口` | 命中"context length"/"上下文长度"chunk | 同上 |

测试通过：用 `application/knowledge` 的 retrieve API（或写个一次性脚本调 `knowledgeSVC.Retrieve`），dump 中间 vector/es/rerank score。

## 5. 调优 Backlog（优先级）

| P | 调优项 | 难度 | 收益 | Phase |
|---|---|---|---|---|
| P0 | Query 预过滤（长度<3 / 纯重复 / 纯标点 → 直接 empty） | 低 | 必须 | 1 |
| P0 | `MinScore` 强制下限（hardcode 兜底 0.3，配置可降） | 低 | 必须 | 1 |
| P0 | ES top1 强归一化 1.0 这个 bug（`elasticsearch_searchstore.go:262`）改成绝对 BM25 score 或全局 sigmoid | 中 | 高 | 1/2 |
| P1 | 默认 agent template `MinScore` 从 0.01 → 0.3+（`create.go:109,143`） | 低 | 高 | 1 |
| P1 | RRF 改加权融合：vector 权重 > BM25（针对短 query 抗噪） | 中 | 高 | 2 |
| P1 | Rerank topk 显式控制（现在传 `Strategy.TopK`，召回阶段 4 条进 rerank 几乎没用） | 中 | 高 | 2 |
| P2 | BM25 调参 / 改 analyzer（ik_smart for 中文） | 中 | 中 | 2 |
| P2 | Query rewrite 增强：短 query 触发同义扩展、错字纠正 | 高 | 中 | 3 |
| P3 | Embedding 模型替换（bge-m3 / e5-large）+ 历史重灌 | 高 | 高 | 3 |
| P3 | Chunk size / overlap 复盘 | 中 | 中 | 3 |
| P3 | metadata filter：跨 knowledge_id / document_type 防串扰 | 低 | 中 | 3 |

## 6. Phase 1 急救清单

按 patch 简易度排，目标：`aaa` 返回 empty。

1. **Query 预过滤** — `retrieve.go:60` 处 `Retrieve` 入口加守卫
   ```
   if isJunkQuery(req.Query) { return empty + log }
   // isJunkQuery: utf8 长度 < MinQueryLen(=2) 或全部 rune 同字符 或全部 unicode.Punct
   ```
2. **MinScore 兜底** — `retrieve.go:579` 在使用 `Strategy.MinScore` 前 `max(strategy, 0.3)`；同步把 `application/singleagent/create.go:109,143` 默认值改 `ptr.Of(0.3)`。
3. **ES 归一化 bug** — `elasticsearch_searchstore.go:262` 改为 `doc.WithScore(score)`（保留 raw BM25）或除以一个固定常量（如 max_expect=20），避免 top1 永远 1.0 喂坏 RRF。
4. **Empty-result UX** — `Retrieve` 返回 `&RetrieveResponse{}` 时上层（agent/workflow caller）保留 query 文本，前端友好提示。位置见 `application/singleagent/single_agent.go:620` 及 workflow `nodes/knowledge/knowledge_retrieve.go:80`。
5. **Bad case log** — `retrieve.go:586` 处加 `logs.CtxInfof` dump query + 各通道 top3 + final score，便于后续挖更多 bad case。

## 7. 调研盲点（需要后续 verify）

- 生产实际 `Strategy.MinScore` 值（DB 里 agent.knowledge.min_score 字段）：默认是 0.01，但客户/老版 bot 可能改过，需 SQL 查一遍。
- `SpaceRerankProvider` 在生产是否被启用、用的是哪个模型 endpoint（`config` 字段在 `space_rerank` 表里）。需登录 224 查。
- ES 实际 analyzer 配置（生产 ES index mapping 是否 `ik_smart`/`standard`）。需 `GET /knowledge_*/_mapping`。
- HTTP embedding（`HTTP_EMBEDDING_ADDR=127.0.0.1:6543`）实际跑的是哪个模型、是否归一化、是否对短文本特殊处理。需看 `~/projects/...` 里这个 sidecar 项目。
- `query rewrite`（`messages2query`）在无 chat history 时直接 bypass（`retrieve.go:199`），单轮检索完全没走改写——是否符合预期？
- OB 向量库实际生产参数（IVF 类型、`distance_type` 是否真是 cosine_distance）。需查 OB ddl。
- 是否有现成 golden set / 评估脚本 — `scripts/` 下未发现 `eval-retrieval*`，需新建。
