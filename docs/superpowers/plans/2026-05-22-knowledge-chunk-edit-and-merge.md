# Spec A 实施计划：切片链路 Verify + 合并 + 状态徽章

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 端到端 verify + 修复知识库切片编辑后的 re-embedding 链路；新增切片合并能力；加状态徽章 + 5s 轮询提示。

**Architecture:** 后端基本不动（仅 Phase 1 修 broken 时按需补丁）。前端在 `knowledge-modal-base` 新增 4 个可复用单元（`SliceStatusBadge` / `useSliceStatusPolling` / `useMergeSlices` / `MergeSliceConfirmModal`），然后 text/table/image 三个 workspace 接入。

**Tech Stack:** TypeScript + React + ahooks + Vitest（前端）；Go + Hertz + Thrift + MQ + Embedding + OceanBase/Milvus（后端，仅 Phase 1 用到）

**Spec**: [Spec A v2](../specs/2026-05-22-knowledge-chunk-edit-design.md)

---

## File Structure（决策提前锁死）

新增前端文件：
```
frontend/packages/data/knowledge/knowledge-modal-base/src/
├── slice-status-badge/
│   ├── index.tsx                          # 组件本体
│   └── __tests__/index.test.tsx           # 单测
├── use-slice-status-polling/
│   ├── index.ts                           # hook
│   └── __tests__/index.test.ts            # 单测
├── use-merge-slices/
│   ├── index.ts                           # hook
│   └── __tests__/index.test.ts            # 单测
└── merge-slice-confirm-modal/
    ├── index.tsx                          # Modal
    └── __tests__/index.test.tsx           # 单测
```

修改前端文件：
- `knowledge-modal-base/src/index.tsx` — re-export 新单元
- `knowledge-ide-base/src/features/text-knowledge-workspace/components/level-content.tsx` — 多选 + 合并按钮 + 徽章
- `knowledge-ide-base/src/features/table-knowledge-workspace/components/table-data-view.tsx` — 多选 + 合并按钮 + 徽章
- `knowledge-ide-base/src/features/image-knowledge-workspace/index.tsx` — 仅徽章（不加合并）

后端：仅 Phase 1 临时 verify 日志（不 commit）+ broken fix（按 verify 结果）

---

## Phase 1: E2E Verify Re-embedding 链路

### Task 1.1: dev 环境跑通 + text-workspace verify

**Files:**
- Modify (temporary, do NOT commit): `backend/domain/knowledge/service/event_handle.go:518` `indexSlice` 函数

**Steps:**

- [ ] **Step 1: 临时加 verify 日志（不 commit）**

```go
// 在 event_handle.go 的 indexSlice 函数（line 518 附近）开头加:
logs.CtxInfof(ctx, "[VERIFY-A1] indexSlice received: slice_id=%d doc_id=%d", event.Slice.ID, event.Slice.DocumentID)

// 在 embedding 调用前后加（具体行号取决于 indexSlice 内部，自己定位）:
logs.CtxInfof(ctx, "[VERIFY-A1] embedding start: content_len=%d", len(content))
// ... embedding 调用 ...
logs.CtxInfof(ctx, "[VERIFY-A1] embedding done: vec_dim=%d err=%v", len(vec), err)

// 在向量库 upsert 前后加:
logs.CtxInfof(ctx, "[VERIFY-A1] upsert start: slice_id=%d", slice.ID)
// ... upsert ...
logs.CtxInfof(ctx, "[VERIFY-A1] upsert done: err=%v", err)

// 在最末尾 status=Done 前加:
logs.CtxInfof(ctx, "[VERIFY-A1] indexSlice finish: slice_id=%d", slice.ID)
```

- [ ] **Step 2: 启动 dev 环境**

```bash
# 后端
cd backend && make run  # 或现行 dev 命令
# 前端（另开终端）
cd frontend && rush install && cd apps/coze-studio && pnpm dev
```

- [ ] **Step 3: 准备测试数据**

操作：
1. 打开浏览器 dev 地址（默认 `http://localhost:8888` 或现行）
2. 创建/选一个 text 类型知识库
3. 上传一个 txt 文件等切片完成，确认有 ≥ 3 个 chunk

- [ ] **Step 4: 触发编辑**

操作：
1. 在切片列表里选第一个 chunk，hover → 点编辑按钮
2. 在 `LevelTextKnowledgeEditor` 弹出的编辑器里改一段内容（如开头加 `[VERIFY-A1-TEST]`）
3. 点保存

- [ ] **Step 5: 观察后端日志**

期望全部出现：
```
[VERIFY-A1] indexSlice received: slice_id=X doc_id=Y
[VERIFY-A1] embedding start: content_len=N
[VERIFY-A1] embedding done: vec_dim=D err=<nil>
[VERIFY-A1] upsert start: slice_id=X
[VERIFY-A1] upsert done: err=<nil>
[VERIFY-A1] indexSlice finish: slice_id=X
```

若任何一条没出现或带 `err=...`：**这是 broken 点**，记入下一个 task。

- [ ] **Step 6: 观察 DB**

```sql
SELECT slice_id, status, content, updated_at FROM knowledge_document_slice WHERE slice_id = X;
```

期望：
- `status` 经历 0 (Init) → 1 (Done)
- `content` 已是新值
- `updated_at` 已刷新

- [ ] **Step 7: 观察向量库**

OceanBase: `SELECT id, dim_count FROM knowledge_document_slice_vector WHERE slice_id = X;`（具体表名按现行 schema 调整）

期望：对应 slice_id 的向量数据时间戳是最新的（具体看 schema 是否有 ts 列）。或者拿 retrieval 反向验证：调 retrieval API 用新 content 的关键词查询，看是否能命中。

- [ ] **Step 8: 整理 verify 结果**

新建 `docs/superpowers/research/2026-05-22-verify-text-reindex.md`，写：
- 哪些日志看到 / 没看到
- DB 状态转换是否正确
- 向量库是否真的更新
- retrieval 是否命中新内容

- [ ] **Step 9: 暂不 revert 日志**（等 1.2、1.3 都用完再统一 revert）

### Task 1.2: table-workspace verify

**Files:** 不动文件（沿用 1.1 的临时日志）

**Steps:**

- [ ] **Step 1: 准备 table 数据**

操作：
1. 创建 table 类型知识库
2. 上传 CSV / 手工输入几行表格
3. 确认切片完成

- [ ] **Step 2: 触发编辑**

操作：
1. 选某行 → 编辑（弹 `useTableSegmentModal`）
2. 改某列值
3. 保存

- [ ] **Step 3: 观察 + 整理（同 Task 1.1 Steps 5-8）**

写到 `docs/superpowers/research/2026-05-22-verify-table-reindex.md`

### Task 1.3: image-workspace verify

**Files:** 不动文件（沿用 1.1 的临时日志）

**Steps:**

- [ ] **Step 1: 准备 image 数据**

操作：
1. 创建 image 类型知识库
2. 上传 2-3 张图片，等 OCR/caption 完成

- [ ] **Step 2: 触发编辑**

操作：
1. hover 某张图 → 编辑按钮
2. 在 `PhotoDetailModal` 改 caption
3. 保存

- [ ] **Step 3: 观察 + 整理（同 Task 1.1 Steps 5-8）**

写到 `docs/superpowers/research/2026-05-22-verify-image-reindex.md`。**特别注意**：image application 层 `UpdatePhotoCaption` 走的是 `listResp.Slices[0]`，如果图片切了多片，只第一片会重新 embedding（这是已知 limitation，写到 research 文档）。

### Task 1.4: revert 临时日志 + 出 broken report

**Files:**
- Modify: `backend/domain/knowledge/service/event_handle.go` (revert 临时日志)
- Create: `docs/superpowers/research/2026-05-22-reindex-chain-broken-report.md`

**Steps:**

- [ ] **Step 1: revert 临时日志**

```bash
git checkout -- backend/domain/knowledge/service/event_handle.go
git diff backend/domain/knowledge/service/event_handle.go
```

Expected: no diff (确认已回到原状)

- [ ] **Step 2: 汇总 broken report**

汇总 Task 1.1/1.2/1.3 的发现，写到 `docs/superpowers/research/2026-05-22-reindex-chain-broken-report.md`：
- 哪个 workspace 的哪一环节 broken
- 每个 broken 的 root cause（如能确定）
- 修复建议（patch 在哪几行）

- [ ] **Step 3: commit research 文档**

```bash
git add docs/superpowers/research/
git commit -m "docs(superpowers): re-embedding chain e2e verify report"
```

- [ ] **Step 4: 决策点**

根据 broken report 决定：
- **若全部 OK** → 跳过 Phase 1.5，直接进 Phase 2
- **若有 broken** → 进 Phase 1.5（每个 broken 一个独立 fix task；具体 fix 内容因 broken 不同，写到 broken report 后再细化 plan task）

### Task 1.5: 修 broken 链路（视 1.4 结论决定）

**Files:** 视具体 broken 决定，写到 broken report 后回头补这个 task 的细节

> **注**：这是动态 task，无法提前定写。Phase 1.4 决策后若需要，由人工或后续 plan 增量补充。

---

## Phase 2: 切片合并（Merge Slice）

### Task 2.1: 写 `useMergeSlices` hook + tests

**Files:**
- Create: `frontend/packages/data/knowledge/knowledge-modal-base/src/use-merge-slices/index.ts`
- Test: `frontend/packages/data/knowledge/knowledge-modal-base/src/use-merge-slices/__tests__/index.test.ts`

**Steps:**

- [ ] **Step 1: 写失败测试**

```typescript
// use-merge-slices/__tests__/index.test.ts
import { renderHook, act } from '@testing-library/react';
import { vi, describe, it, expect, beforeEach } from 'vitest';
import { KnowledgeApi } from '@coze-arch/bot-api';
import { useMergeSlices } from '../index';

vi.mock('@coze-arch/bot-api', () => ({
  KnowledgeApi: {
    UpdateSlice: vi.fn(),
    DeleteSlice: vi.fn(),
  },
}));

describe('useMergeSlices', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('merges slices into the smallest-sequence target via UpdateSlice + DeleteSlice', async () => {
    (KnowledgeApi.UpdateSlice as any).mockResolvedValue({});
    (KnowledgeApi.DeleteSlice as any).mockResolvedValue({});

    const { result } = renderHook(() => useMergeSlices());
    const slices = [
      { slice_id: '101', sequence: 2, content: 'B' },
      { slice_id: '100', sequence: 1, content: 'A' },
      { slice_id: '102', sequence: 3, content: 'C' },
    ];

    await act(async () => {
      const res = await result.current.merge(slices);
      expect(res.ok).toBe(true);
      expect(res.target_slice_id).toBe('100');
    });

    expect(KnowledgeApi.UpdateSlice).toHaveBeenCalledWith({
      slice_id: '100',
      raw_text: 'A\n\nB\n\nC',
    });
    expect(KnowledgeApi.DeleteSlice).toHaveBeenCalledWith({
      slice_ids: ['101', '102'],
    });
  });

  it('returns error and skips DeleteSlice when UpdateSlice fails', async () => {
    (KnowledgeApi.UpdateSlice as any).mockRejectedValue(new Error('update failed'));

    const { result } = renderHook(() => useMergeSlices());
    const slices = [
      { slice_id: '100', sequence: 1, content: 'A' },
      { slice_id: '101', sequence: 2, content: 'B' },
    ];

    await act(async () => {
      const res = await result.current.merge(slices);
      expect(res.ok).toBe(false);
      expect(res.stage).toBe('update');
    });
    expect(KnowledgeApi.DeleteSlice).not.toHaveBeenCalled();
  });

  it('retries DeleteSlice once on failure, returns partial-ok on persistent failure', async () => {
    (KnowledgeApi.UpdateSlice as any).mockResolvedValue({});
    (KnowledgeApi.DeleteSlice as any)
      .mockRejectedValueOnce(new Error('first fail'))
      .mockRejectedValueOnce(new Error('second fail'));

    const { result } = renderHook(() => useMergeSlices());
    const slices = [
      { slice_id: '100', sequence: 1, content: 'A' },
      { slice_id: '101', sequence: 2, content: 'B' },
    ];

    await act(async () => {
      const res = await result.current.merge(slices);
      expect(res.ok).toBe(false);
      expect(res.stage).toBe('delete');
      expect(res.partial).toBe(true);
    });
    expect(KnowledgeApi.DeleteSlice).toHaveBeenCalledTimes(2);
  });

  it('rejects empty or single-item input', async () => {
    const { result } = renderHook(() => useMergeSlices());
    await act(async () => {
      const res = await result.current.merge([]);
      expect(res.ok).toBe(false);
      expect(res.stage).toBe('validate');
    });
    await act(async () => {
      const res = await result.current.merge([{ slice_id: '1', sequence: 1, content: 'A' }]);
      expect(res.ok).toBe(false);
      expect(res.stage).toBe('validate');
    });
  });
});
```

- [ ] **Step 2: 跑测试 verify FAIL**

```bash
cd frontend/packages/data/knowledge/knowledge-modal-base && pnpm test src/use-merge-slices
```

Expected: FAIL with `Cannot find module '../index'`

- [ ] **Step 3: 写最小实现**

```typescript
// use-merge-slices/index.ts
import { KnowledgeApi } from '@coze-arch/bot-api';

export interface MergeSliceInput {
  slice_id: string;
  sequence: number;
  content: string;
}

export type MergeResult =
  | { ok: true; target_slice_id: string }
  | { ok: false; stage: 'validate' | 'update' | 'delete'; partial?: boolean; error?: Error };

export const useMergeSlices = () => {
  const merge = async (slices: MergeSliceInput[]): Promise<MergeResult> => {
    if (slices.length < 2) {
      return { ok: false, stage: 'validate' };
    }

    const sorted = [...slices].sort((a, b) => a.sequence - b.sequence);
    const target = sorted[0];
    const others = sorted.slice(1);
    const mergedContent = sorted.map(s => s.content).join('\n\n');

    try {
      await KnowledgeApi.UpdateSlice({
        slice_id: target.slice_id,
        raw_text: mergedContent,
      });
    } catch (error) {
      return { ok: false, stage: 'update', error: error as Error };
    }

    const otherIds = others.map(s => s.slice_id);
    try {
      await KnowledgeApi.DeleteSlice({ slice_ids: otherIds });
    } catch (firstErr) {
      try {
        await KnowledgeApi.DeleteSlice({ slice_ids: otherIds });
      } catch (secondErr) {
        return { ok: false, stage: 'delete', partial: true, error: secondErr as Error };
      }
    }

    return { ok: true, target_slice_id: target.slice_id };
  };

  return { merge };
};
```

- [ ] **Step 4: 跑测试 verify PASS**

```bash
pnpm test src/use-merge-slices
```

Expected: All 4 tests PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/packages/data/knowledge/knowledge-modal-base/src/use-merge-slices/
git commit -m "feat(knowledge): add useMergeSlices hook with two-step merge + retry"
```

### Task 2.2: 写 `MergeSliceConfirmModal` 组件 + tests

**Files:**
- Create: `frontend/packages/data/knowledge/knowledge-modal-base/src/merge-slice-confirm-modal/index.tsx`
- Test: `frontend/packages/data/knowledge/knowledge-modal-base/src/merge-slice-confirm-modal/__tests__/index.test.tsx`

**Steps:**

- [ ] **Step 1: 写失败测试**

```tsx
// merge-slice-confirm-modal/__tests__/index.test.tsx
import { render, screen, fireEvent } from '@testing-library/react';
import { vi, describe, it, expect } from 'vitest';
import { MergeSliceConfirmModal } from '../index';

describe('MergeSliceConfirmModal', () => {
  const slices = [
    { slice_id: '1', sequence: 1, content: 'First chunk' },
    { slice_id: '2', sequence: 2, content: 'Second chunk' },
  ];

  it('renders preview with concatenated content', () => {
    render(
      <MergeSliceConfirmModal
        visible
        slices={slices}
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />,
    );
    expect(screen.getByText(/First chunk/)).toBeInTheDocument();
    expect(screen.getByText(/Second chunk/)).toBeInTheDocument();
  });

  it('calls onConfirm when confirm clicked', () => {
    const onConfirm = vi.fn();
    render(
      <MergeSliceConfirmModal
        visible
        slices={slices}
        onConfirm={onConfirm}
        onCancel={vi.fn()}
      />,
    );
    fireEvent.click(screen.getByText('合并'));
    expect(onConfirm).toHaveBeenCalledTimes(1);
  });

  it('calls onCancel when cancel clicked', () => {
    const onCancel = vi.fn();
    render(
      <MergeSliceConfirmModal
        visible
        slices={slices}
        onConfirm={vi.fn()}
        onCancel={onCancel}
      />,
    );
    fireEvent.click(screen.getByText('取消'));
    expect(onCancel).toHaveBeenCalledTimes(1);
  });

  it('shows loading state and disables confirm when loading', () => {
    render(
      <MergeSliceConfirmModal
        visible
        slices={slices}
        loading
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />,
    );
    const btn = screen.getByText('合并').closest('button');
    expect(btn).toBeDisabled();
  });
});
```

- [ ] **Step 2: 跑测试 verify FAIL**

```bash
pnpm test src/merge-slice-confirm-modal
```

Expected: FAIL with module not found

- [ ] **Step 3: 写最小实现**

```tsx
// merge-slice-confirm-modal/index.tsx
import React from 'react';
import { Modal, Button } from '@coze-arch/coze-design';
import type { MergeSliceInput } from '../use-merge-slices';

export interface MergeSliceConfirmModalProps {
  visible: boolean;
  slices: MergeSliceInput[];
  loading?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export const MergeSliceConfirmModal: React.FC<MergeSliceConfirmModalProps> = ({
  visible,
  slices,
  loading = false,
  onConfirm,
  onCancel,
}) => {
  if (!visible) return null;

  const sorted = [...slices].sort((a, b) => a.sequence - b.sequence);
  const preview = sorted.map(s => s.content).join('\n\n');

  return (
    <Modal
      visible={visible}
      title={`合并 ${slices.length} 个切片`}
      onCancel={onCancel}
      footer={
        <>
          <Button onClick={onCancel} disabled={loading}>取消</Button>
          <Button type="primary" onClick={onConfirm} loading={loading} disabled={loading}>
            合并
          </Button>
        </>
      }
    >
      <div style={{ maxHeight: 400, overflow: 'auto', whiteSpace: 'pre-wrap', padding: 12, border: '1px solid #eee', borderRadius: 4 }}>
        {preview}
      </div>
    </Modal>
  );
};
```

- [ ] **Step 4: 跑测试 verify PASS**

```bash
pnpm test src/merge-slice-confirm-modal
```

Expected: All 4 tests PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/packages/data/knowledge/knowledge-modal-base/src/merge-slice-confirm-modal/
git commit -m "feat(knowledge): add MergeSliceConfirmModal with content preview"
```

### Task 2.3: 把合并功能接入 text-workspace

**Files:**
- Modify: `frontend/packages/data/knowledge/knowledge-ide-base/src/features/text-knowledge-workspace/components/level-content.tsx`
- Modify: `frontend/packages/data/knowledge/knowledge-modal-base/src/index.tsx` (re-export 新单元)

**Steps:**

- [ ] **Step 1: 先 export 新单元**

修改 `knowledge-modal-base/src/index.tsx`，在已有 export 后追加：
```typescript
export { useMergeSlices } from './use-merge-slices';
export { MergeSliceConfirmModal } from './merge-slice-confirm-modal';
export type { MergeSliceInput, MergeResult } from './use-merge-slices';
```

- [ ] **Step 2: 在 `level-content.tsx` 增加多选 + 合并入口**

修改 [level-content.tsx](frontend/packages/data/knowledge/knowledge-ide-base/src/features/text-knowledge-workspace/components/level-content.tsx)：

```tsx
import { useState } from 'react';
import { Toast } from '@coze-arch/coze-design';
import {
  useMergeSlices,
  MergeSliceConfirmModal,
} from '@coze-data/knowledge-modal-base';

// 在 LevelContent 组件内加：
const [selectedIds, setSelectedIds] = useState<string[]>([]);
const [mergeModalOpen, setMergeModalOpen] = useState(false);
const { merge } = useMergeSlices();
const [merging, setMerging] = useState(false);

const selectedSlices = renderLevelSegmentsData
  .filter(c => selectedIds.includes(String(c.slice_id)))
  .map(c => ({
    slice_id: String(c.slice_id),
    sequence: c.sequence,
    content: c.content ?? '',
  }));

// 判断 sequence 是否连续：
const isContiguous = selectedSlices.length >= 2 && (() => {
  const sorted = [...selectedSlices].sort((a, b) => a.sequence - b.sequence);
  for (let i = 1; i < sorted.length; i++) {
    if (sorted[i].sequence !== sorted[i - 1].sequence + 1) return false;
  }
  return true;
})();

const handleMerge = async () => {
  setMerging(true);
  const res = await merge(selectedSlices);
  setMerging(false);
  if (res.ok) {
    Toast.success('合并成功，正在重新索引...');
    setMergeModalOpen(false);
    setSelectedIds([]);
    // 注：Task 3.3 Step 2 实施时回头在这里 setEditedIds(prev => [...prev, res.target_slice_id]) 触发 polling
  } else if (res.stage === 'update') {
    Toast.error('合并失败');
  } else if (res.stage === 'delete' && res.partial) {
    Toast.warning('合并部分完成，多余切片需手动删除');
    setMergeModalOpen(false);
    setSelectedIds([]);
  }
};

// 在 JSX 顶部加 toolbar:
{selectedIds.length >= 2 && (
  <div style={{ padding: 8 }}>
    <Button
      onClick={() => setMergeModalOpen(true)}
      disabled={!isContiguous}
      title={!isContiguous ? '只能合并相邻切片' : undefined}
    >
      合并 {selectedIds.length} 个
    </Button>
  </div>
)}

// 在 chunks 旁加 checkbox（具体怎么传给 LevelTextKnowledgeEditor 看其 props 是否支持，
// 若不支持需要 wrap 或外置 checkbox）:
// 见 Step 3

<MergeSliceConfirmModal
  visible={mergeModalOpen}
  slices={selectedSlices}
  loading={merging}
  onConfirm={handleMerge}
  onCancel={() => setMergeModalOpen(false)}
/>
```

- [ ] **Step 3: 决策 checkbox 怎么挂**

`LevelTextKnowledgeEditor` 是黑盒组件。两种方式：
- (A) 看 `text-knowledge-editor` 是否支持 `renderChunkExtra` 或类似插槽 props → 若支持，传 checkbox
- (B) 不支持则在 `level-content.tsx` 外面 wrap 一层，每个 chunk 渲染时在外面套 div + checkbox

先 grep `text-knowledge-editor/scenes/level/main.tsx` 看 props 定义：

```bash
grep -A 20 "interface.*Props\|type.*Props" \
  frontend/packages/data/knowledge/common/components/src/text-knowledge-editor/scenes/level/main.tsx
```

根据结果选 A 或 B。**若需 wrap (B) 则改动较大，可考虑改 `text-knowledge-editor` 加 slot props（fork 风险小，因 monorepo 内部包）**。

- [ ] **Step 4: 手动 verify**

启动 dev 环境：
1. 选 text 知识库
2. 多选 2-3 个相邻 chunk
3. 顶部出现"合并"按钮 → 点击 → 弹 Modal → 确认
4. 列表刷新只剩 1 个 chunk，content 是拼接结果

- [ ] **Step 5: Commit**

```bash
git add frontend/packages/data/knowledge/
git commit -m "feat(knowledge): wire merge slice into text-workspace"
```

### Task 2.4: 把合并功能接入 table-workspace

**Files:**
- Modify: `frontend/packages/data/knowledge/knowledge-ide-base/src/features/table-knowledge-workspace/components/table-data-view.tsx`

**Steps:**

- [ ] **Step 1: 看现有 table 选择能力**

```bash
grep -n "rowSelection\|select\|checkbox" \
  frontend/packages/data/knowledge/knowledge-ide-base/src/features/table-knowledge-workspace/components/table-data-view.tsx
```

table 类型用的是 `TableView`，大概率已支持 `rowSelection`，复用即可。

- [ ] **Step 2: 加多选 + 合并按钮（参考 2.3 Step 2 模板，替换数据源为 table 的 slice 列表）**

代码模式同 2.3，只是 `selectedSlices` 从 table rows 抽取。

- [ ] **Step 3: 手动 verify**

table 知识库 → 多选连续 row → 合并 → 刷新只剩 1 行

- [ ] **Step 4: Commit**

```bash
git add frontend/packages/data/knowledge/knowledge-ide-base/src/features/table-knowledge-workspace/
git commit -m "feat(knowledge): wire merge slice into table-workspace"
```

> **注**：image-workspace **不接入合并**（每张图通常 1 片，无意义）。

---

## Phase 3: 状态徽章 + 轮询

### Task 3.1: 写 `SliceStatusBadge` 组件 + tests

**Files:**
- Create: `frontend/packages/data/knowledge/knowledge-modal-base/src/slice-status-badge/index.tsx`
- Test: `frontend/packages/data/knowledge/knowledge-modal-base/src/slice-status-badge/__tests__/index.test.tsx`

**Steps:**

- [ ] **Step 1: 写失败测试**

```tsx
// slice-status-badge/__tests__/index.test.tsx
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { SliceStatusBadge } from '../index';

describe('SliceStatusBadge', () => {
  it('renders nothing when status is Done', () => {
    const { container } = render(<SliceStatusBadge status="Done" />);
    expect(container.firstChild).toBeNull();
  });

  it('renders spinner when status is Init or Processing', () => {
    const { rerender } = render(<SliceStatusBadge status="Init" />);
    expect(screen.getByText(/重新索引中/)).toBeInTheDocument();
    rerender(<SliceStatusBadge status="Processing" />);
    expect(screen.getByText(/重新索引中/)).toBeInTheDocument();
  });

  it('renders retry button + reason when status is Failed', () => {
    const onRetry = vi.fn();
    render(<SliceStatusBadge status="Failed" reason="embedding timeout" onRetry={onRetry} />);
    expect(screen.getByText(/embedding timeout/)).toBeInTheDocument();
    fireEvent.click(screen.getByText('重试'));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it('renders timeout state when status is Timeout', () => {
    render(<SliceStatusBadge status="Timeout" />);
    expect(screen.getByText(/长时间未完成/)).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: 跑测试 verify FAIL**

```bash
pnpm test src/slice-status-badge
```

Expected: FAIL module not found

- [ ] **Step 3: 写实现**

```tsx
// slice-status-badge/index.tsx
import React from 'react';
import { Spin, Button, Tooltip } from '@coze-arch/coze-design';

export type SliceBadgeStatus = 'Init' | 'Processing' | 'Done' | 'Failed' | 'Timeout';

export interface SliceStatusBadgeProps {
  status: SliceBadgeStatus;
  reason?: string;
  onRetry?: () => void;
}

export const SliceStatusBadge: React.FC<SliceStatusBadgeProps> = ({ status, reason, onRetry }) => {
  if (status === 'Done') return null;

  if (status === 'Init' || status === 'Processing') {
    return (
      <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4, color: '#666' }}>
        <Spin size="small" />
        <span style={{ fontSize: 12 }}>重新索引中</span>
      </span>
    );
  }

  if (status === 'Failed') {
    return (
      <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4, color: '#d44' }}>
        <Tooltip content={reason ?? '未知错误'}>
          <span style={{ fontSize: 12 }}>索引失败</span>
        </Tooltip>
        {onRetry && <Button size="small" onClick={onRetry}>重试</Button>}
      </span>
    );
  }

  if (status === 'Timeout') {
    return <span style={{ fontSize: 12, color: '#999' }}>长时间未完成</span>;
  }

  return null;
};
```

- [ ] **Step 4: 跑测试 verify PASS**

```bash
pnpm test src/slice-status-badge
```

Expected: All 4 tests PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/packages/data/knowledge/knowledge-modal-base/src/slice-status-badge/
git commit -m "feat(knowledge): add SliceStatusBadge with 4 states + retry"
```

### Task 3.2: 写 `useSliceStatusPolling` hook + tests

**Files:**
- Create: `frontend/packages/data/knowledge/knowledge-modal-base/src/use-slice-status-polling/index.ts`
- Test: `frontend/packages/data/knowledge/knowledge-modal-base/src/use-slice-status-polling/__tests__/index.test.ts`

**Steps:**

- [ ] **Step 1: 写失败测试**

```typescript
// use-slice-status-polling/__tests__/index.test.ts
import { renderHook, act, waitFor } from '@testing-library/react';
import { vi, describe, it, expect, beforeEach, afterEach } from 'vitest';
import { KnowledgeApi } from '@coze-arch/bot-api';
import { useSliceStatusPolling } from '../index';

vi.mock('@coze-arch/bot-api', () => ({
  KnowledgeApi: { ListSlice: vi.fn() },
}));

describe('useSliceStatusPolling', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.clearAllMocks();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it('polls ListSlice every 5s and reports status', async () => {
    (KnowledgeApi.ListSlice as any).mockResolvedValue({
      slices: [
        { slice_id: '1', status: 0 }, // Init
        { slice_id: '2', status: 1 }, // Done
      ],
    });

    const { result } = renderHook(() =>
      useSliceStatusPolling({ documentId: 'D1', watchedIds: ['1', '2'] }),
    );

    await act(async () => { await vi.advanceTimersByTimeAsync(0); });
    expect(KnowledgeApi.ListSlice).toHaveBeenCalledTimes(1);
    expect(result.current.statusMap['1']).toBe('Init');
    expect(result.current.statusMap['2']).toBe('Done');

    await act(async () => { await vi.advanceTimersByTimeAsync(5000); });
    expect(KnowledgeApi.ListSlice).toHaveBeenCalledTimes(2);
  });

  it('stops polling when all watched ids are Done', async () => {
    (KnowledgeApi.ListSlice as any)
      .mockResolvedValueOnce({ slices: [{ slice_id: '1', status: 0 }] })
      .mockResolvedValueOnce({ slices: [{ slice_id: '1', status: 1 }] });

    const { result } = renderHook(() =>
      useSliceStatusPolling({ documentId: 'D1', watchedIds: ['1'] }),
    );

    await act(async () => { await vi.advanceTimersByTimeAsync(0); });
    await act(async () => { await vi.advanceTimersByTimeAsync(5000); });
    await act(async () => { await vi.advanceTimersByTimeAsync(5000); });

    expect(KnowledgeApi.ListSlice).toHaveBeenCalledTimes(2);
  });

  it('marks watched ids as Timeout after 120s', async () => {
    (KnowledgeApi.ListSlice as any).mockResolvedValue({
      slices: [{ slice_id: '1', status: 0 }],
    });
    const { result } = renderHook(() =>
      useSliceStatusPolling({ documentId: 'D1', watchedIds: ['1'] }),
    );

    await act(async () => { await vi.advanceTimersByTimeAsync(125000); });
    expect(result.current.statusMap['1']).toBe('Timeout');
  });

  it('cleans up timer on unmount', async () => {
    (KnowledgeApi.ListSlice as any).mockResolvedValue({ slices: [{ slice_id: '1', status: 0 }] });
    const { unmount } = renderHook(() =>
      useSliceStatusPolling({ documentId: 'D1', watchedIds: ['1'] }),
    );
    await act(async () => { await vi.advanceTimersByTimeAsync(0); });
    unmount();
    await act(async () => { await vi.advanceTimersByTimeAsync(20000); });
    expect(KnowledgeApi.ListSlice).toHaveBeenCalledTimes(1);
  });
});
```

- [ ] **Step 2: 跑测试 verify FAIL**

```bash
pnpm test src/use-slice-status-polling
```

Expected: FAIL module not found

- [ ] **Step 3: 写实现**

```typescript
// use-slice-status-polling/index.ts
import { useEffect, useRef, useState } from 'react';
import { KnowledgeApi } from '@coze-arch/bot-api';
import type { SliceBadgeStatus } from '../slice-status-badge';

const POLL_INTERVAL_MS = 5000;
const TIMEOUT_MS = 120000;

export interface UseSliceStatusPollingParams {
  documentId: string;
  watchedIds: string[];
}

const mapStatus = (status: number): SliceBadgeStatus => {
  switch (status) {
    case 0: return 'Init';
    case 1: return 'Done';
    case 9: return 'Init';
    default: return status === 2 ? 'Processing' : status > 1 ? 'Failed' : 'Init';
  }
};

export const useSliceStatusPolling = ({ documentId, watchedIds }: UseSliceStatusPollingParams) => {
  const [statusMap, setStatusMap] = useState<Record<string, SliceBadgeStatus>>({});
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const startTsRef = useRef<number>(0);
  const watchedRef = useRef<Set<string>>(new Set());

  useEffect(() => {
    if (watchedIds.length === 0) return;
    watchedRef.current = new Set(watchedIds);
    startTsRef.current = Date.now();

    const poll = async () => {
      try {
        const resp = await KnowledgeApi.ListSlice({ document_id: documentId });
        const slices = (resp as any)?.slices ?? [];
        const updates: Record<string, SliceBadgeStatus> = {};
        let allDone = true;
        for (const s of slices) {
          const id = String(s.slice_id);
          if (!watchedRef.current.has(id)) continue;
          const mapped = mapStatus(s.status);
          updates[id] = mapped;
          if (mapped !== 'Done') allDone = false;
        }
        setStatusMap(prev => ({ ...prev, ...updates }));

        if (allDone && Object.keys(updates).length === watchedRef.current.size) {
          if (timerRef.current) {
            clearInterval(timerRef.current);
            timerRef.current = null;
          }
          return;
        }

        if (Date.now() - startTsRef.current > TIMEOUT_MS) {
          const timeoutUpdates: Record<string, SliceBadgeStatus> = {};
          for (const id of watchedRef.current) {
            if (statusMap[id] !== 'Done') timeoutUpdates[id] = 'Timeout';
          }
          setStatusMap(prev => ({ ...prev, ...timeoutUpdates }));
          if (timerRef.current) {
            clearInterval(timerRef.current);
            timerRef.current = null;
          }
        }
      } catch (err) {
        // 轮询错误不中断 (下一 tick 重试)
      }
    };

    poll();
    timerRef.current = setInterval(poll, POLL_INTERVAL_MS);

    return () => {
      if (timerRef.current) {
        clearInterval(timerRef.current);
        timerRef.current = null;
      }
    };
  }, [documentId, watchedIds.join(',')]);

  return { statusMap };
};
```

- [ ] **Step 4: 跑测试 verify PASS**

```bash
pnpm test src/use-slice-status-polling
```

Expected: All 4 tests PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/packages/data/knowledge/knowledge-modal-base/src/use-slice-status-polling/
git commit -m "feat(knowledge): add useSliceStatusPolling hook (5s interval, 120s timeout)"
```

### Task 3.3: 三个 workspace 集成徽章 + polling

**Files:**
- Modify: `knowledge-modal-base/src/index.tsx` (export new units)
- Modify: `knowledge-ide-base/src/features/text-knowledge-workspace/components/level-content.tsx`
- Modify: `knowledge-ide-base/src/features/table-knowledge-workspace/components/table-data-view.tsx`
- Modify: `knowledge-ide-base/src/features/image-knowledge-workspace/index.tsx`

**Steps:**

- [ ] **Step 1: Export 新单元**

修改 `knowledge-modal-base/src/index.tsx` 追加：

```typescript
export { SliceStatusBadge } from './slice-status-badge';
export { useSliceStatusPolling } from './use-slice-status-polling';
export type { SliceBadgeStatus, SliceStatusBadgeProps } from './slice-status-badge';
```

- [ ] **Step 2: text-workspace 集成**

在 `level-content.tsx`：

```tsx
import { SliceStatusBadge, useSliceStatusPolling } from '@coze-data/knowledge-modal-base';

// 内部加：
const [editedIds, setEditedIds] = useState<string[]>([]);
const { statusMap } = useSliceStatusPolling({
  documentId,
  watchedIds: editedIds,
});

// 在 useUpdateRemoteChunk 的 onUpdateChunk callback 里加：
const handleEdited = (chunk: Chunk) => {
  setEditedIds(prev => prev.includes(String(chunk.slice_id))
    ? prev
    : [...prev, String(chunk.slice_id)]);
};

// chunk 旁边渲染（具体位置看 LevelTextKnowledgeEditor 是否支持插槽；
// 不支持则在 chunk 外 wrap div）：
{statusMap[String(chunk.slice_id)] && (
  <SliceStatusBadge
    status={statusMap[String(chunk.slice_id)]}
    onRetry={() => {/* 调 KnowledgeApi.UpdateSlice 用原 content 重新发 */}}
  />
)}

// Task 2.3 里的 handleMerge 成功后也调 setEditedIds 加入合并目标 id：
if (res.ok) {
  setEditedIds(prev => [...prev, res.target_slice_id]);
  // ... 其余原逻辑 ...
}
```

- [ ] **Step 3: table-workspace 集成（同 Step 2 模式）**

在 `table-data-view.tsx` 把 polling + badge 接到现有 row 渲染。Badge 渲染到列尾或行操作区。

- [ ] **Step 4: image-workspace 集成**

`image-workspace/index.tsx` 在 Card 上叠 badge。`usePhotoDetailModal` 的 `onSubmit` callback 触发 `setEditedIds`：

```tsx
const [editedIds, setEditedIds] = useState<string[]>([]);
const { statusMap } = useSliceStatusPolling({
  documentId: '<根据 image 所属 doc>',
  watchedIds: editedIds,
});

const { node, open } = usePhotoDetailModal({
  // ... 原 props ...
  onSubmit: () => {
    resetUrlQueryParams();
    // image 没有直接的 slice_id 暴露，需要从 photoList 找到对应 slice_id（如果有），
    // 或者编辑后直接 reload 整个 photoList（最简单）
    reloadAsync();
  },
});

// Card 上叠：
{statusMap[String(slice_id)] && (
  <SliceStatusBadge status={statusMap[String(slice_id)]} />
)}
```

> **注**：image-workspace 的 PhotoInfo 是否携带 slice_id？写 plan 时未 verify，**此 step 实施前先 grep `PhotoInfo` 看字段，没有就 fallback 到"编辑后无 badge、靠 reloadAsync 刷新内容"。** 不能因为 image 阻塞主功能。

- [ ] **Step 5: 手动 verify 全部**

启动 dev：
1. text 知识库：编辑 chunk → 看 badge 转圈 → 5s 内变 Done → badge 消失
2. table 知识库：同上
3. image 知识库：编辑 caption → badge（若有）；reload 拿到新 caption
4. text 知识库：合并 2 chunk → badge 转圈在合并后的 chunk 上 → Done → 消失

- [ ] **Step 6: Commit**

```bash
git add frontend/packages/data/knowledge/
git commit -m "feat(knowledge): integrate SliceStatusBadge + polling in 3 workspaces"
```

---

## Self-Review

**Spec 覆盖：**
- ✅ Phase 1 e2e verify → Task 1.1-1.4
- ✅ Phase 1 fix broken → Task 1.5（动态）
- ✅ Phase 2 切片合并 → Task 2.1-2.4
- ✅ Phase 3 状态徽章 + 轮询 → Task 3.1-3.3
- ✅ 三种 workspace 都接入

**已知 open question 留到实施阶段处理**：
1. `text-knowledge-editor` 是否支持插槽 props（Task 2.3 Step 3 + Task 3.3 Step 2 都涉及）→ 实施时先 grep 决策
2. `PhotoInfo` 是否带 `slice_id`（Task 3.3 Step 4）→ 实施前 grep
3. table-workspace `TableView` 的 rowSelection 接口（Task 2.4 Step 1）→ 实施前 grep

这些是"实施前 5 分钟可以查清的小问题"，不是 plan 漏洞。
