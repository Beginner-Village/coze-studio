# 工作流分类(展示型) 设计文档

> 日期:2026-06-03　项目:coze-studio(猎鹰 Studio)
> 范围:仅「工作流分类」展示功能。批量测试、上游融合收尾不在本 spec(后者见 `coze-loop/docs/upstream-merge-plan-20260603.md`,在实施 plan 中统一编排)。

---

## 1. 目标与背景

工作流数量变多后,列表难以浏览。提供**展示型分类**:把工作流归到不同「文件夹」里,点文件夹钻入查看。纯展示分组,**不改变工作流本身的运行、权限、归属语义**。

## 2. 交互(已与用户确认)

文件夹卡片 + 钻入二级,类似文件管理器:

- **一级视图**:列表区先显示**文件夹卡片**(文件夹名 + 内含工作流数量),其下显示**未分类工作流**卡片。
- **添加分类**:列表右上角「＋ 添加分类」→ 弹窗输入分类名称 → 新建一个文件夹卡片。
- **钻入二级**:点文件夹卡片 → 进入该分类,显示其下工作流 + 面包屑「← 全部」返回一级。
- **移入/移出**:工作流卡片 `⋮` 菜单 →「移入分类」(选目标文件夹)/「移出分类」(回到最外层未分类)。

## 3. 复用现有能力(不重造)

### 后端(已完整,基本不动)
- 文件夹领域:`backend/domain/folder/`(`Folder` 表、`ResourceFolderMapping`,`ResourceType=2` 即 workflow)
- API:`backend/api/handler/coze/plugin_develop_service.go`
  - `CreateFolder` / `GetFolderList` / `MoveResourcesToFolder`
- 数据库表已存在,无需 migration。

### 前端(已有基础设施)
- Hook:`packages/studio/workspace/entry-base/src/pages/library/hooks/use-folder-management.tsx`
  - 已封装 `folders` / `createFolder(name, description?)` / `moveResourcesToFolder(folderId, resourceIds, resourceType)` / `refreshFolders`,内部调 `plugin_api`。
- 工作流列表落在 coze library 页:`packages/studio/workspace/entry-base/src/pages/library/`(列表渲染组件 + `hooks/use-entity-configs/use-workflow-config.tsx`)。

### 缺口
工作流列表页**尚未把 `useFolderManagement` 接成「文件夹卡片 + 钻入二级」的展示 UI**。本功能即补这一层 UI 接入。

## 4. 组件与数据流

### 数据流
1. 进入工作流列表 → 并行拉:`GetFolderList(spaceId)` + 工作流列表(现有接口)。
2. 一级渲染:文件夹卡片(来自 folders,每个统计其下工作流数)+ 未分类工作流(无 folder 映射的)。
3. 点文件夹 → 组件内状态 `currentFolderId` 置为该 folder → 二级渲染该 folder 下工作流(按映射过滤)。
4. 「← 全部」→ `currentFolderId = null` 回一级。
5. 移入/移出 → `moveResourcesToFolder(folderId | 0, [workflowId], 2)` → 成功后 `refreshFolders` + 刷新列表。

### 组件单元(单一职责)
- **WorkflowFolderView**(新增):工作流列表的「文件夹层」容器。持有 `currentFolderId` 状态;一级渲染文件夹卡片 + 未分类工作流,二级渲染面包屑 + 该分类工作流。复用现有工作流卡片渲染。
- **FolderCard**(新增):单个文件夹卡片(名称 + 数量 + 点击钻入)。
- **AddCategoryButton + 弹窗**(新增):输入名称 → `createFolder`。
- **工作流卡片 `⋮` 菜单扩展**:加「移入分类 / 移出分类」项 → `moveResourcesToFolder`。
- 复用 `useFolderManagement`(已存在)做所有 folder 读写。

## 5. 范围边界(YAGNI)

**做**:单层文件夹、一个工作流归一个分类、添加分类、移入/移出、钻入/返回、未分类显示在最外层、数量统计。

**不做**(以后需要再加):
- 文件夹嵌套(只一层)
- 拖拽归类(先用 `⋮` 菜单移动)
- 工作流多分类
- 批量测试(独立功能,本 spec 不含)
- 文件夹重命名/删除(首版可不做;若简单可作为 `⋮` 附带,留待 plan 评估)

## 6. 错误处理

- `GetFolderList` 失败:列表降级为「全部工作流扁平展示」(不阻塞主流程),提示「分类加载失败」。
- `createFolder` 同名:后端若不约束,前端提交前做同名校验,提示「分类已存在」。
- `moveResourcesToFolder` 失败:toast 报错,列表不变(不做乐观更新或失败回滚)。

## 7. 测试

- 创建分类 → 一级出现新文件夹卡片(数量 0)。
- 移入:把工作流移入分类 → 该工作流从未分类消失,文件夹数量 +1,钻入可见。
- 移出:从分类移出 → 回到最外层未分类。
- 钻入/返回:点文件夹进二级、面包屑返回一级。
- `GetFolderList` 失败时降级为扁平列表不报错崩溃。

## 8. 关键文件清单

| 用途 | 路径 |
|---|---|
| 文件夹管理 hook(复用) | `frontend/packages/studio/workspace/entry-base/src/pages/library/hooks/use-folder-management.tsx` |
| 工作流列表页(改造落点) | `frontend/packages/studio/workspace/entry-base/src/pages/library/`(列表渲染组件,plan 首步精确定位) |
| workflow 配置 hook | `.../library/hooks/use-entity-configs/use-workflow-config.tsx` |
| 后端 folder API | `backend/api/handler/coze/plugin_develop_service.go`(CreateFolder/GetFolderList/MoveResourcesToFolder) |
| 后端 folder 领域 | `backend/domain/folder/`(Folder + ResourceFolderMapping, ResourceType=2) |

> **Plan 首步**:精确定位 library 工作流列表的渲染组件(确认 workflow tab 的列表 JSX 所在文件),作为接入 `WorkflowFolderView` 的位置。
