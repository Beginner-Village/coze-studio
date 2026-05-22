# Spec B Phase 1 实施计划：召回质量急救 5 项

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把锚点 bad case `query="aaa" → AI 内容高分` 拦住。5 项后端急救改动，让 `aaa` 类无意义 query 在 retrieval 入口或最后过滤掉。

**Architecture:** 改 5 处后端 Go 代码（无 DB / 无前端改动）。3 个修在 `backend/domain/knowledge/service/retrieve.go`，1 个修在 `backend/infra/impl/document/searchstore/elasticsearch/elasticsearch_searchstore.go`，1 个修在 `backend/application/singleagent/create.go`。

**Tech Stack:** Go + Hertz + Eino + Elasticsearch + Milvus/OceanBase

**Spec**: [Spec B](../specs/2026-05-22-retrieval-quality-optimization.md)
**Research**: [Phase 0 调研报告](../research/2026-05-22-retrieval-bad-cases.md)

---

## File Structure（提前锁死）

修改后端文件：
- `backend/domain/knowledge/service/retrieve.go` — 加 isJunkQuery 守卫 (Task 1) + MinScore 兜底 (Task 2) + bad case log (Task 5)
- `backend/infra/impl/document/searchstore/elasticsearch/elasticsearch_searchstore.go` — ES top1 归一化 bug 修复 (Task 3)
- `backend/application/singleagent/create.go` — agent 默认 MinScore 从 0.01 改 0.3 (Task 2)
- 可能新建 `backend/domain/knowledge/service/query_filter.go` — `isJunkQuery` 独立文件便于测试

新增后端测试：
- `backend/domain/knowledge/service/query_filter_test.go` — isJunkQuery 单测
- `backend/infra/impl/document/searchstore/elasticsearch/elasticsearch_searchstore_test.go` — ES 归一化 bug 修复测试（追加或新建）

不改：
- 前端任何文件
- DB 任何表结构

---

## Phase 1 急救 (5 tasks)

### Task B1: Query 预过滤 (`isJunkQuery` + retrieve.go 入口守卫)

**Files:**
- Create: `backend/domain/knowledge/service/query_filter.go`
- Test: `backend/domain/knowledge/service/query_filter_test.go`
- Modify: `backend/domain/knowledge/service/retrieve.go` (around line 60, `Retrieve` 入口加守卫)

**Steps:**

- [ ] **Step 1: 写失败测试**

```go
// query_filter_test.go
package service

import (
	"testing"
)

func TestIsJunkQuery(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  bool
	}{
		{"empty", "", true},
		{"single char ascii", "a", true},
		{"single char chinese", "好", true},
		{"two char ascii", "ab", true},      // 长度 < MinQueryLen=3
		{"two char chinese", "你好", true},   // 长度 < MinQueryLen=3
		{"repeat ascii", "aaa", true},
		{"repeat ascii longer", "aaaa", true},
		{"repeat chinese", "啊啊啊", true},
		{"all punct", "!!!", true},
		{"all punct mixed", "?!@", true},
		{"all whitespace", "   ", true},
		{"valid short", "abc", false},        // 3 字符且非重复 → 通过
		{"valid chinese", "知识库", false},
		{"valid mixed", "API key", false},
		{"valid english phrase", "vector embedding", false},
		{"valid leading whitespace", "  hello", false}, // trim 后非 junk
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isJunkQuery(tt.query)
			if got != tt.want {
				t.Errorf("isJunkQuery(%q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: 跑测试 verify FAIL**

```bash
cd /Users/luzhipeng/projects/ynet/coze-studio/backend && go test ./domain/knowledge/service/ -run TestIsJunkQuery -v
```

Expected: FAIL with `undefined: isJunkQuery`

- [ ] **Step 3: 写实现**

```go
// query_filter.go
package service

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// MinQueryLen is the minimum number of meaningful runes (after trim) a query must have.
// Queries shorter than this are considered junk (too short to convey intent).
const MinQueryLen = 3

// isJunkQuery returns true when a query is too short, all-same-rune, all-punctuation,
// or otherwise unsuitable for retrieval. Such queries are filtered at the Retrieve
// entry to prevent retrieval-system noise (e.g., embedding OOV-fallback collisions,
// BM25 top1-normalization artifacts) from producing spurious high-score hits.
func isJunkQuery(query string) bool {
	q := strings.TrimSpace(query)
	if q == "" {
		return true
	}
	if utf8.RuneCountInString(q) < MinQueryLen {
		return true
	}
	// All-same-rune?
	first, _ := utf8.DecodeRuneInString(q)
	allSame := true
	allPunctOrSpace := true
	for _, r := range q {
		if r != first {
			allSame = false
		}
		if !unicode.IsPunct(r) && !unicode.IsSpace(r) && !unicode.IsSymbol(r) {
			allPunctOrSpace = false
		}
		if !allSame && !allPunctOrSpace {
			break
		}
	}
	return allSame || allPunctOrSpace
}
```

- [ ] **Step 4: 跑测试 verify PASS**

```bash
cd /Users/luzhipeng/projects/ynet/coze-studio/backend && go test ./domain/knowledge/service/ -run TestIsJunkQuery -v
```

Expected: All 16 cases PASS

- [ ] **Step 5: 把守卫接入 Retrieve 入口**

修改 `backend/domain/knowledge/service/retrieve.go` 在 `Retrieve` 函数（around line 56-70）加：

```go
func (k *knowledgeSVC) Retrieve(ctx context.Context, request *RetrieveRequest) (*RetrieveResponse, error) {
	if request == nil || request.Query == "" {
		return nil, errorx.New(errno.ErrKnowledgeInvalidParamCode, errorx.KV("msg", "query is empty"))
	}
	if isJunkQuery(request.Query) {
		logs.CtxInfof(ctx, "[retrieve] junk query filtered: %q", request.Query)
		return &RetrieveResponse{}, nil
	}
	// ... 原代码续 ...
}
```

(具体行号取决于当前 `retrieve.go` 现状，自己定位 — 在原有的 empty-query 检查之后立刻加 junk filter）

- [ ] **Step 6: 跑整体 go build + vet**

```bash
cd /Users/luzhipeng/projects/ynet/coze-studio/backend && go build ./... && go vet ./...
```

Expected: no errors.

- [ ] **Step 7: Commit**

```bash
git add backend/domain/knowledge/service/query_filter.go \
        backend/domain/knowledge/service/query_filter_test.go \
        backend/domain/knowledge/service/retrieve.go
git commit -m "feat(retrieval): add isJunkQuery guard at Retrieve entry"
```

---

### Task B2: MinScore 兜底 + agent 默认值调整

**Files:**
- Modify: `backend/domain/knowledge/service/retrieve.go` (around line 579, MinScore 过滤前加兜底)
- Modify: `backend/application/singleagent/create.go` (line 109 + 143, 默认值 0.01 → 0.3)

**Steps:**

- [ ] **Step 1: 写失败测试（新增 retrieve_minscore_test.go）**

```go
// retrieve_minscore_test.go
package service

import (
	"testing"
)

func TestEffectiveMinScore(t *testing.T) {
	tests := []struct {
		name     string
		strategy float64
		want     float64
	}{
		{"low strategy below floor", 0.01, 0.3},  // 兜底
		{"medium strategy below floor", 0.2, 0.3}, // 兜底
		{"strategy at floor", 0.3, 0.3},
		{"strategy above floor", 0.5, 0.5},
		{"strategy very high", 0.9, 0.9},
		{"strategy zero", 0.0, 0.3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := effectiveMinScore(tt.strategy)
			if got != tt.want {
				t.Errorf("effectiveMinScore(%v) = %v, want %v", tt.strategy, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: 跑测试 verify FAIL**

```bash
cd backend && go test ./domain/knowledge/service/ -run TestEffectiveMinScore -v
```

Expected: FAIL undefined.

- [ ] **Step 3: 写实现 + 接入**

在 `retrieve.go` 增加：

```go
// MinScoreFloor is the hard-coded lower bound for MinScore in retrieval.
// Callers may pass a higher value via Strategy.MinScore; lower values are
// raised to this floor to prevent low-confidence hits from being returned.
const MinScoreFloor = 0.3

func effectiveMinScore(strategy float64) float64 {
	if strategy < MinScoreFloor {
		return MinScoreFloor
	}
	return strategy
}
```

在 `retrieve.go:579`（reRankNode 过滤处）把 `Strategy.MinScore` 替换为 `effectiveMinScore(ptr.From(Strategy.MinScore))`（具体调用形式按现有代码）。

- [ ] **Step 4: 跑测试 verify PASS**

```bash
go test ./domain/knowledge/service/ -run TestEffectiveMinScore -v
```

Expected: 6 cases PASS.

- [ ] **Step 5: 改 agent template 默认值**

`backend/application/singleagent/create.go`:
- Line 109: `MinScore: ptr.Of(0.01)` → `MinScore: ptr.Of(0.3)`
- Line 143: 同上

修改前 grep 一下确认具体位置：
```bash
grep -n "0.01\|MinScore" backend/application/singleagent/create.go
```

- [ ] **Step 6: build + vet**

```bash
cd backend && go build ./... && go vet ./...
```

- [ ] **Step 7: Commit**

```bash
git add backend/domain/knowledge/service/retrieve.go \
        backend/domain/knowledge/service/retrieve_minscore_test.go \
        backend/application/singleagent/create.go
git commit -m "feat(retrieval): MinScore floor at 0.3 + agent template default raised"
```

---

### Task B3: ES top1 归一化 bug 修复

**Files:**
- Modify: `backend/infra/impl/document/searchstore/elasticsearch/elasticsearch_searchstore.go` (around line 262)
- Test: same dir, add or extend test file

**Steps:**

- [ ] **Step 1: Read 现有代码定位 bug**

```bash
sed -n '250,280p' backend/infra/impl/document/searchstore/elasticsearch/elasticsearch_searchstore.go
```

应当看到类似：
```go
for i, hit := range searchResp.Hits.Hits {
    if i == 0 {
        firstScore = hit.Score
    }
    score := hit.Score / firstScore  // ← BUG: top1 永远 = 1.0
    docs = append(docs, doc.WithScore(score))
}
```

- [ ] **Step 2: 决策怎么修**

两种修法：
- **(a) 保留 raw BM25 score**：`docs = append(docs, doc.WithScore(hit.Score))`。优点：最简单；缺点：BM25 score 量纲不固定（可能 5、可能 50），上层 MinScore 兜底（0.3）压根接不上。需要在文档说清楚。
- **(b) sigmoid 归一化**：`normalizedScore := 1 / (1 + math.Exp(-(hit.Score - 5))) // sigmoid centered around BM25 score 5`，把 BM25 映射到 (0,1)。优点：score 量纲稳定；缺点：5 这个 center 是猜的，可能要调。

**选 (a)** — 简单优先。结合 Task B2 的 MinScore floor 0.3 失效在 ES 路径上是可以接受的（因为 ES 路径走 RRF 进 rerank，rerank 输出 score 是 [0,1] 量纲，MinScore floor 仍然在 rerank 后生效，ES raw score 只用于 rerank 内部 rank 排序，不直接影响 final filter）。

如果 reviewer 觉得 (b) 更好，可在 fix iteration 中再换。

- [ ] **Step 3: 写测试**

```go
// elasticsearch_searchstore_test.go (新增或追加)
package elasticsearch

import (
	"testing"
)

func TestExtractDocsScorePreservesRawBM25(t *testing.T) {
	// 模拟 search hits 入参（如果现有代码把 score 抽取做成内部函数，直接调它；
	// 否则在 PR 中 refactor 一下把 score-mapping 提到独立函数 extractDocScore）
	hits := []hitLite{
		{ID: "1", Score: 12.5},
		{ID: "2", Score: 6.0},
		{ID: "3", Score: 3.0},
	}
	docs := extractDocsFromHits(hits)
	if len(docs) != 3 {
		t.Fatalf("expected 3 docs, got %d", len(docs))
	}
	if docs[0].Score() != 12.5 {
		t.Errorf("doc[0] score: want 12.5, got %v", docs[0].Score())
	}
	if docs[1].Score() != 6.0 {
		t.Errorf("doc[1] score: want 6.0, got %v", docs[1].Score())
	}
	if docs[2].Score() != 3.0 {
		t.Errorf("doc[2] score: want 3.0, got %v", docs[2].Score())
	}
}
```

注：如果当前代码把 hits → docs 的转换内嵌在大函数里没法测，**先 refactor 提取 `extractDocsFromHits` 函数**（这是 Task B3 实施前置）。`hitLite` 是个最小本地 struct，避免依赖完整 ES client 类型。

- [ ] **Step 4: 跑测试 verify FAIL**

```bash
cd backend && go test ./infra/impl/document/searchstore/elasticsearch/ -run TestExtractDocsScorePreservesRawBM25 -v
```

Expected: FAIL (函数未提取出来 OR top1 仍然归一化)

- [ ] **Step 5: 写实现**

提取 `extractDocsFromHits(hits []hitLite) []*schema.Document` 函数，内部用 raw `hit.Score`（不除 firstScore）。原代码调用点改为 `docs := extractDocsFromHits(...)`。

- [ ] **Step 6: 跑测试 verify PASS**

```bash
go test ./infra/impl/document/searchstore/elasticsearch/ -run TestExtractDocsScorePreservesRawBM25 -v
```

Expected: PASS

- [ ] **Step 7: 跑 ES 包全部测试 + build**

```bash
go test ./infra/impl/document/searchstore/elasticsearch/ -v
go build ./...
```

- [ ] **Step 8: Commit**

```bash
git add backend/infra/impl/document/searchstore/elasticsearch/
git commit -m "fix(retrieval): preserve raw BM25 score in ES retrieve (top1 was always 1.0)"
```

---

### Task B4: Empty-result UX 改进

**Files:**
- Modify: `backend/application/singleagent/single_agent.go` (around line 620 — 找到 retrieve 调用后处理 empty response 的代码)
- Modify: `backend/domain/workflow/internal/nodes/knowledge/knowledge_retrieve.go` (around line 80)

**Steps:**

- [ ] **Step 1: 定位现有处理**

```bash
grep -n "RetrieveResponse\|len.*RetrieveSlice\|len.*\.Hits" backend/application/singleagent/single_agent.go
grep -n "RetrieveResponse\|len.*Retrieve" backend/domain/workflow/internal/nodes/knowledge/knowledge_retrieve.go
```

读上下文 ±20 行。

- [ ] **Step 2: 加 empty-result 友好日志 + 保留 query**

在 retrieve 返回 empty 时打 info 日志（含 query + knowledge_id + 调用方）：

```go
// single_agent.go
if len(retrieveResp.GetRetrieveSlices()) == 0 {
    logs.CtxInfof(ctx, "[retrieve] empty result: query=%q knowledge_ids=%v", query, knowledgeIDs)
    // 维持原有 empty 处理，但额外把 query 透传给前端（如已透传则跳过）
}
```

`knowledge_retrieve.go` 同理。

- [ ] **Step 3: build + vet**

```bash
cd backend && go build ./... && go vet ./...
```

- [ ] **Step 4: Commit**

```bash
git add backend/application/singleagent/single_agent.go \
        backend/domain/workflow/internal/nodes/knowledge/knowledge_retrieve.go
git commit -m "feat(retrieval): log empty results with query for diagnostics"
```

---

### Task B5: Bad case 日志

**Files:**
- Modify: `backend/domain/knowledge/service/retrieve.go` (around line 586, packResults 之前 / reRankNode 之后)

**Steps:**

- [ ] **Step 1: 找到位置**

```bash
grep -n "packResults\|reRankNode" backend/domain/knowledge/service/retrieve.go
```

- [ ] **Step 2: 加 dump 日志**

在 rerank 完之后 / packResults 之前，dump query + top3 from each channel + top3 final score：

```go
logs.CtxInfof(ctx, "[retrieve-dump] query=%q vector_top3=%v es_top3=%v final_top3=%v",
    request.Query,
    topN(vectorResults, 3),    // helper: 取前 3 的 score+slice_id 字符串化
    topN(esResults, 3),
    topN(finalResults, 3),
)
```

加 helper 函数 `topN(results []*schema.Document, n int) string`，返回类似 `[{1234567890:0.91},{...},{...}]` 的字符串。

放到 retrieve.go 文件底部即可（或新 helper 文件 `retrieve_log.go`）。

- [ ] **Step 3: build + vet**

```bash
cd backend && go build ./... && go vet ./...
```

- [ ] **Step 4: Commit**

```bash
git add backend/domain/knowledge/service/retrieve.go \
        # if separate helper file:
        backend/domain/knowledge/service/retrieve_log.go
git commit -m "feat(retrieval): dump query + per-channel top3 for bad case mining"
```

---

## Integration verify（5 个 task 全部 land 后做一次）

### Manual e2e（必须 user 跑，subagent 跑不了）

- [ ] 启动 dev 后端 + 前端
- [ ] 在 dev 的某个有知识库的 agent 上，发以下 query，看返回是否符合预期：

| Query | 期望 |
|---|---|
| `aaa` | 返回空（被 isJunkQuery 拦） |
| `好` | 返回空（< MinQueryLen） |
| `!!!` | 返回空（all punct） |
| `量子纠缠`（KB 无相关） | 返回空或 score < 0.3 |
| `知识库`（KB 有此词） | 命中相关 chunk，score ≥ 0.3 |

如果以上 5 个验证全部通过 → Phase 1 ACCEPTANCE GATE 达成。

后端日志看：
- isJunkQuery 守卫触发：`[retrieve] junk query filtered: "aaa"`
- bad case dump 日志（每次 retrieve）：`[retrieve-dump] query="..." vector_top3=[...] es_top3=[...] final_top3=[...]`

---

## 风险

| Risk | 缓解 |
|---|---|
| ES bug fix (Task B3) 改 raw BM25 score 后，rerank/RRF 行为变化 | 因为 RRF 只看 rank 不看 score，所以 raw score 改不影响 RRF 输出顺序；只影响 model-rerank 路径（jina 等），但模型 rerank 输出自己的 [0,1] score 覆盖。OK。 |
| MinScore floor 0.3 可能误伤合理 query | 0.3 是基于 OB cosine 距离归一化后的保守值。如发现误伤，配置可降到 0.2 / 0.15。已加 logs 便于观察。 |
| Empty-result UX 改动 (Task B4) 影响 agent 历史行为 | 仅追加日志，不改返回 shape。无 caller 风险。 |
| 单测覆盖率不够 | Task B1/B2 有完整单测，B3 通过提取函数实现可测，B4/B5 是 logging 不需单测（手动 e2e 验证） |

## 工作量

| Task | 估时 |
|---|---|
| B1 isJunkQuery + 接入 | 0.5 天 |
| B2 MinScore floor + 默认调整 | 0.5 天 |
| B3 ES bug fix（含 refactor 提取函数） | 0.5-1 天 |
| B4 Empty-result UX | 0.5 天 |
| B5 Bad case log | 0.5 天 |
| Integration manual e2e | 0.5 天 |
| **合计** | **2.5-3 天** |
