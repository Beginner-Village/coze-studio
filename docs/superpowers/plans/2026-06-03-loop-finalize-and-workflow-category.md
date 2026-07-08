# Loop 融合收尾 + 工作流分类 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把已合并的上游 Loop(#519–#537)收尾验证并确认与猎鹰的对接正常;在 coze-studio 工作流列表加「展示型分类」(文件夹卡片 + 钻入二级)。

**Architecture:** Phase 1 在 coze-loop 隔离分支 `ynet-upstream-merge-2026-06`(worktree `coze-loop-merge`)上验证融合(编译已过)+ Studio 对接,再并回主分支。Phase 2 在 coze-studio 的共用列表页 `BaseLibraryPage` 接入已存在的 `useFolderManagement` hook + 后端 folder API,仅对 workflow tab 开启文件夹视图。

**Tech Stack:** Go 1.24(coze-loop 后端) · React + Zustand + @coze-arch/coze-design(coze-studio 前端) · rush monorepo · 后端 folder 领域(Folder + ResourceFolderMapping, ResourceType=2)

参考:
- 融合方案:`coze-loop/docs/upstream-merge-plan-20260603.md`
- 分类 spec:`coze-studio/docs/superpowers/specs/2026-06-03-workflow-category-design.md`

---

## Phase 1 — Loop 融合收尾 + 猎鹰对接(coze-loop)

> 工作区:`/Users/luzhipeng/projects/ynet/coze-loop-merge`(worktree,分支 `ynet-upstream-merge-2026-06`,merge 已提交 `8af71111`,`go build ./...` 已通过)。

### Task 1.1: observability 单测(确认我们的 metrics 定制 + 上游 trace 改动共存)

**Files:**
- Verify: `coze-loop-merge/backend/modules/observability/...`

- [ ] **Step 1: 跑 observability 单测**

Run:
```bash
cd /Users/luzhipeng/projects/ynet/coze-loop-merge/backend
go test ./modules/observability/... 2>&1 | tail -30
```
Expected: 全 `ok`。若 `infra/metrics` 相关 FAIL → 是上游 #469/#492 与我们的 RED/business-metrics 合并点问题,打开冲突文件人工核对(保留我们的 `trace_consume` psm/tenant tags + SpanStats,接纳上游新增 metric),改完重跑。

- [ ] **Step 2: 跑 evaluation 单测(含 #520 agent eval 全量)**

Run:
```bash
go test ./modules/evaluation/... 2>&1 | tail -30
```
Expected: 全 `ok`。FAIL 时按报错定位(多为 mock/wire 缺失,确认 eval_target 恢复完整)。

### Task 1.2: 关键路径 vet + 启动自检

- [ ] **Step 1: vet 全后端**

Run:
```bash
cd /Users/luzhipeng/projects/ynet/coze-loop-merge/backend
go vet ./modules/evaluation/... ./modules/observability/... 2>&1 | tail -20
```
Expected: 无输出(通过)。

### Task 1.3: 记录 experiment schema 变更到部署侧

**Files:**
- Modify: `coze-loop/docs/upstream-merge-plan-20260603.md`(追加部署 checklist)

- [ ] **Step 1: 提取上游 experiment 表结构变更**

Run:
```bash
cd /Users/luzhipeng/projects/ynet/coze-loop-upstream-latest
git show upstream/main:release/deployment/docker-compose/bootstrap/mysql-init/patch-sql/experiment_alter.sql
```
Expected: 打印出 `ALTER TABLE experiment ...` 加列语句。

- [ ] **Step 2: 把这些 ALTER 写进部署 runbook**

把上一步的 ALTER 语句记到 `upstream-merge-plan-20260603.md` 的「DB Migration」节,标注「220/POC 部署前手动执行」。

- [ ] **Step 3: Commit**

```bash
cd /Users/luzhipeng/projects/ynet/coze-loop
git add docs/upstream-merge-plan-20260603.md
git commit -m "docs(loop): record experiment_alter schema change for deploy"
```

### Task 1.4: 本地 docker compose 冒烟(验证 #520 + 工作流批测真能跑)

> 若本地无法起全栈(依赖 MySQL/ClickHouse/RocketMQ),跳到 Task 1.5 用 220/POC 验证,并在此 task 标注 skipped。

- [ ] **Step 1: 起 Loop 后端 + 依赖**

Run(参考 coze-loop README 的 docker compose):
```bash
cd /Users/luzhipeng/projects/ynet/coze-loop-merge
docker compose -f docker-compose.yml up -d 2>&1 | tail -10
```
Expected: 容器起来。

- [ ] **Step 2: 建实验冒烟**

通过 Loop 前端或 API:建 dataset → 选 YNETWorkflow(target_type=4) 或 CustomAgent(=10) target → 选 evaluator → SubmitExperiment → 看 run 状态到 Success。
Expected: 实验跑批产出聚合结果,无 panic。

### Task 1.5: 确认猎鹰对接(Studio session/trace)融合后未回退

**Files:**
- Verify: `coze-loop-merge/backend/...`(我们的 `feat(integration): bridge Studio session` 改动)

- [ ] **Step 1: 确认 Studio 集成改动仍在**

Run:
```bash
cd /Users/luzhipeng/projects/ynet/coze-loop-merge
git log --oneline | grep -i "bridge Studio session"
grep -rn "Studio" backend/modules/observability/domain/trace/service/ingestion.go | head
```
Expected: 能看到 bridge commit;ingestion 里 Studio 相关逻辑未被上游 #469 重构覆盖掉。若被覆盖 → 人工把我们的 server-to-server trace ingest 逻辑重新接到上游新版 ingestion.go。

- [ ] **Step 2: 确认 /metrics 端口定制仍在(:8889 + sidecar :8890)**

Run:
```bash
grep -rn "8889\|8890\|LISTEN_ADDR" backend/ | grep -v "_test.go" | head
```
Expected: 端口定制仍在。

### Task 1.6: 并回主分支(等用户确认后执行)

- [ ] **Step 1: 主分支合并 worktree 分支**

Run(⚠️ 仅在用户确认后):
```bash
cd /Users/luzhipeng/projects/ynet/coze-loop
git checkout ynet-main
git merge --no-ff ynet-upstream-merge-2026-06 -m "merge: upstream #519-#537 + eval_target restore + agent eval"
```
Expected: fast-forward 或干净 merge(同源,无新冲突)。

- [ ] **Step 2: 清理 worktree**

```bash
git worktree remove ../coze-loop-merge
```

---

## Phase 2 — 工作流分类(coze-studio)

> 工作目录:`/Users/luzhipeng/projects/ynet/coze-studio`。前端在 `frontend/`(rush + pnpm)。
> 改动集中在 `frontend/packages/studio/workspace/entry-base/src/pages/library/`。

### Task 2.0: 坐实后端 folder 数据契约(过滤 + 归属)

**Files:**
- Read: `coze-studio/backend/api/handler/coze/plugin_develop_service.go`(GetFolderList / MoveResourcesToFolder)
- Read: `coze-studio/backend/domain/folder/`(ResourceFolderMapping)
- Read: `coze-studio/frontend/packages/*/api-schema`(LibraryResourceList 请求/响应类型)

- [ ] **Step 1: 确认 LibraryResourceList 能否按 folder 过滤、resource 是否带 folder 归属**

Run:
```bash
cd /Users/luzhipeng/projects/ynet/coze-studio
grep -rn "folder_id\|FolderID\|folder" backend/api/model/*/*.go 2>/dev/null | grep -i "resource\|library" | head
grep -rn "LibraryResourceList" backend/api/handler/coze/*.go | head
```
Expected: 判定两种情况之一:
- (A) `LibraryResourceListRequest` 已有 `folder_id` 过滤字段 + 响应 resource 带 `folder_id` → 后端不用改,前端直接用。
- (B) 没有 → 进 Step 2 补后端。

- [ ] **Step 2(仅当 B): 后端给 LibraryResourceList 加 folder_id 过滤 + 响应带 folder_id**

在 `LibraryResourceListRequest` 加可选 `folder_id`;在 list 实现里 join `ResourceFolderMapping`(resource_type=2) 过滤;响应 resource 带回 `folder_id`(无归属为空)。改完:
```bash
cd backend && go build ./... 2>&1 | tail -5
```
Expected: build 通过。

- [ ] **Step 3: 前端 api-schema 同步类型(若 Step 2 改了 IDL)**

按 coze-studio 的 IDL→TS 生成流程重新生成 `@coze-studio/api-schema`(或手补类型),确认 `plugin_api.get_folder_list` 返回 `{code, data: FolderInfo[]}` 与 hook 一致。

### Task 2.1: BaseLibraryPage 接入 folder 状态(仅 workflow tab)

**Files:**
- Modify: `frontend/packages/studio/workspace/entry-base/src/pages/library/index.tsx`(BaseLibraryPage,~行95 之后)

- [ ] **Step 1: 读现状确认接入点**

Run:
```bash
sed -n '79,140p' frontend/packages/studio/workspace/entry-base/src/pages/library/index.tsx
```
确认 `spaceId` / `sourceType` / `listResp` 变量名与下面骨架一致(不一致则按实际改名)。

- [ ] **Step 2: 加 folder 状态 + 仅 workflow 启用**

在 BaseLibraryPage 组件体内(listResp 定义后)加:
```tsx
import { ResType } from '...'; // 复用文件已 import 的 ResType
import { useFolderManagement } from './hooks/use-folder-management';

const folderEnabled = Number(sourceType) === ResType.Workflow;
const { folders, createFolder, moveResourcesToFolder, refreshFolders } =
  useFolderManagement({ spaceId, onSuccess: () => listResp.reload() });
const [currentFolderId, setCurrentFolderId] = useState<string | null>(null);
```

- [ ] **Step 3: typecheck**

Run:
```bash
cd frontend && npx vue-tsc --noEmit -p packages/studio/workspace/entry-base 2>&1 | tail -10 || npm run typecheck 2>&1 | tail -10
```
Expected: 无新增类型错误。(coze-studio 用 tsc;若该包无独立 typecheck 脚本,用 rush/turbo 跑该包构建)

### Task 2.2: 一级渲染文件夹卡片 + 未分类工作流

**Files:**
- Create: `frontend/packages/studio/workspace/entry-base/src/pages/library/components/folder-card.tsx`
- Modify: `frontend/.../library/index.tsx`(grid 渲染区 ~行372-404)

- [ ] **Step 1: 新建 FolderCard 组件**

`folder-card.tsx`:
```tsx
import { type FC } from 'react';

export interface FolderCardProps {
  name: string;
  count: number;
  onClick: () => void;
}

export const FolderCard: FC<FolderCardProps> = ({ name, count, onClick }) => (
  <div
    data-testid="workspace.library.folder-card"
    className="grid-item p-[12px] cursor-pointer flex flex-col justify-between"
    onClick={onClick}
  >
    <div className="text-[24px]">📁</div>
    <div className="font-[500] truncate">{name}</div>
    <div className="text-[12px] text-[#999]">{count}</div>
  </div>
);
```

- [ ] **Step 2: 一级 grid 顶部插入文件夹卡片**

在 `index.tsx` 的 `<GridList>` 内,`listResp.data?.list.map(...)` 之前,当 `folderEnabled && !currentFolderId` 时渲染文件夹卡片:
```tsx
{folderEnabled && !currentFolderId &&
  folders.map(folder => (
    <GridItem key={`folder-${folder.id}`}>
      <FolderCard
        name={folder.name}
        count={folderCounts[folder.id] ?? 0}
        onClick={() => setCurrentFolderId(folder.id)}
      />
    </GridItem>
  ))}
```
其中 `folderCounts` 由 Step 3 计算。

- [ ] **Step 3: 计算各 folder 工作流数 + 一级只显示未分类**

若 Task 2.0 选了 (A)/(B) 让 list 带 `folder_id`:在 `listResp` 数据上派生:
```tsx
const folderCounts = useMemo(() => {
  const m: Record<string, number> = {};
  (listResp.data?.list ?? []).forEach(r => {
    if (r.folder_id) m[r.folder_id] = (m[r.folder_id] ?? 0) + 1;
  });
  return m;
}, [listResp.data]);

const visibleList = useMemo(() => {
  const all = listResp.data?.list ?? [];
  if (!folderEnabled) return all;
  if (currentFolderId) return all.filter(r => r.folder_id === currentFolderId);
  return all.filter(r => !r.folder_id); // 一级只看未分类
}, [listResp.data, folderEnabled, currentFolderId]);
```
把 grid 的 `listResp.data?.list.map` 改为 `visibleList.map`。

> 注:若工作流量大、list 是分页 infinite-scroll,folderCounts 仅统计已加载页;数量可改为后端在 `get_folder_list` 返回 count(Task 2.0 Step 2 一并补)。优先用后端 count,避免分页统计不准。

- [ ] **Step 4: typecheck** (同 Task 2.1 Step 3)

### Task 2.3: 钻入二级 + 面包屑返回

**Files:**
- Modify: `frontend/.../library/index.tsx`(grid 区上方)

- [ ] **Step 1: 二级时显示面包屑**

在 GridList 之前:
```tsx
{folderEnabled && currentFolderId && (
  <div className="px-[24px] mb-[12px] text-[14px]">
    <span className="cursor-pointer text-[#4d53e8]"
      onClick={() => setCurrentFolderId(null)}>← 全部</span>
    <span className="mx-[6px]">/</span>
    <span className="font-[500]">
      {folders.find(f => f.id === currentFolderId)?.name}
    </span>
  </div>
)}
```
(二级的 grid 内容已由 Task 2.2 Step 3 的 `visibleList` 过滤为该 folder 工作流。)

- [ ] **Step 2: 切 tab 时重置层级**

在 `sourceType` 变化的 effect(或 reloadDeps)里加 `setCurrentFolderId(null)`,避免切到别的资源 tab 仍停在某 folder。

- [ ] **Step 3: typecheck**

### Task 2.4: 头部「＋ 添加分类」按钮 + 命名弹窗

**Files:**
- Modify: `frontend/.../library/components/library-header.tsx`(~行192-342 按钮区)
- Modify: `frontend/.../library/index.tsx`(传入 onCreateFolder)

- [ ] **Step 1: header 仅 workflow tab 显示按钮**

在 `library-header.tsx` 的按钮区(import/create 旁),`sourceType === ResType.Workflow` 时加:
```tsx
<Button
  theme="borderless"
  type="secondary"
  data-testid="workspace.library.header.add-folder"
  onClick={onAddFolder}
>
  ＋ 添加分类
</Button>
```
`onAddFolder` 经 props 从 BaseLibraryPage 传入。

- [ ] **Step 2: BaseLibraryPage 实现命名弹窗 → createFolder**

用 coze-design 的 Modal/Input(参考 header 已 import 的组件)。提交前做同名校验:
```tsx
const onAddFolder = () => {
  // 打开 Modal,输入 name;确认时:
  if (folders.some(f => f.name === name.trim())) {
    Toast.error('分类已存在'); return;
  }
  await createFolder(name.trim());
};
```

- [ ] **Step 3: typecheck**

### Task 2.5: 工作流卡片 ⋮ 加「移入/移出分类」

**Files:**
- Modify: `frontend/packages/studio/workspace/entry-base/src/pages/library/hooks/use-entity-configs/use-workflow-config.tsx`(给 `getCommonActions` 注入 folder action)
- Modify: `frontend/.../library/index.tsx`(把 move 能力经 entityConfigs 透传)

- [ ] **Step 1: 通过 getCommonActions 注入「移入/移出分类」**

`renderActions` 经 `getCommonActions(record)` 扩展(Explore 确认的扩展点)。注入两项:
```tsx
// 在 BaseLibraryPage 组装 getCommonActions 时
const getCommonActions = (record) => folderEnabled ? [
  {
    actionKey: 'moveToFolder',
    actionText: '移入分类',
    handler: () => openMovePopover(record), // 选 folder → moveResourcesToFolder(folderId, [record.res_id], 2)
  },
  ...(record.folder_id ? [{
    actionKey: 'removeFromFolder',
    actionText: '移出分类',
    handler: () => moveResourcesToFolder('0', [record.res_id], 2),
  }] : []),
] : [];
```
> 「移出」用 `folder_id='0'`——Task 2.0 Step 1 必须确认后端把 `folder_id=0` 当作「移出/无分类」;若后端用别的约定(如删映射的专用接口),按实际改。

- [ ] **Step 2: 移动后刷新**

`moveResourcesToFolder` 的 `onSuccess` 已配 `listResp.reload()`(Task 2.1);确认移动后列表与 folderCounts 刷新。

- [ ] **Step 3: typecheck**

### Task 2.6: 错误处理(folder list 失败降级扁平)

**Files:**
- Modify: `frontend/.../library/index.tsx`

- [ ] **Step 1: get_folder_list 失败时不阻塞主列表**

`useFolderManagement` 已 catch 错误并 `console.error`(hook 行70-72)。在 BaseLibraryPage 加:当 `folders` 为空且加载失败标志时,`folderEnabled` 退化为不渲染文件夹卡片(列表照常扁平展示)。给 hook 加一个 `error` 返回值或用 `folders.length===0 && !loading` 作为降级信号,渲染时:
```tsx
const showFolders = folderEnabled && folders.length > 0;
```
把 Task 2.2/2.3 的 `folderEnabled &&` 判断改为 `showFolders &&`(钻入/面包屑同理),保证 folder 接口挂了也不白屏。

- [ ] **Step 2: typecheck**

### Task 2.7: 整包构建 + 冒烟

- [ ] **Step 1: 构建 entry-base 包**

Run:
```bash
cd /Users/luzhipeng/projects/ynet/coze-studio/frontend
npx vue-tsc --noEmit -p packages/studio/workspace/entry-base 2>&1 | tail -15
```
Expected: 0 类型错误。

- [ ] **Step 2: 起前端 dev 手动冒烟**

按 coze-studio 前端启动方式起 dev,进 library → 工作流 tab,验证 spec §7 测试点:
- ＋ 添加分类 → 出现文件夹卡片(数量 0)
- 工作流 ⋮ → 移入分类 → 从未分类消失、文件夹数量 +1、钻入可见
- ⋮ → 移出分类 → 回到最外层
- 点文件夹钻入二级、面包屑「← 全部」返回
- folder 接口模拟失败时列表仍扁平展示不崩

- [ ] **Step 3: Commit**

```bash
cd /Users/luzhipeng/projects/ynet/coze-studio
git add frontend/packages/studio/workspace/entry-base/src/pages/library
git commit -m "feat(library): workflow folder category (folder cards + drill-in)"
```

---

## Self-Review 结论(plan 作者自检)

- **Spec 覆盖**:添加分类→Task 2.4;文件夹卡片+未分类→Task 2.2;钻入二级+面包屑→Task 2.3;移入/移出→Task 2.5;降级→Task 2.6;数量统计→Task 2.2 Step 3。✅ 全覆盖。
- **已知前置依赖**:Task 2.2/2.5 依赖 Task 2.0 坐实的数据契约(list 带 `folder_id` + `folder_id=0` 移出约定)。Task 2.0 是其余前端 task 的硬前置,已置顶。
- **YAGNI**:不做嵌套/拖拽/多分类/批量测试,与 spec 一致。
- **风险点**:coze-studio 前端 typecheck/启动命令需按项目实际(rush/turbo)对齐 —— Task 2.1 Step 3 已留校准空间。
