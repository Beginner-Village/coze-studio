# 策略（Strategy）渐进式披露 —— 前端实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 coze-studio 前端把"策略"做成资源库一等公民：列表入口、整页策略编辑器（场景树 + 四类能力项）、单智能体绑定弹窗，全程贴 `@coze-arch/coze-design` 现有风格。

**Architecture:** 复用资源库 entity-config 可插拔机制（仿 `use-database-config`）渲染策略列表；新增整页路由 `/strategy` 承载编辑器；绑定走 `bot-skill` store 的新 `strategies` slice（仿 `workflows`）。前端调用后端时用从 strategy IDL 生成的 `StrategyApi`。

**Tech Stack:** React + TS、Rush monorepo、RSBuild、`@coze-arch/coze-design`（基于 Semi）、CSS Modules（`*.module.less`）、`@coze-arch/i18n`、zustand（bot-detail store）。

## Global Constraints

- 仓库根：`/Users/luzhipeng/projects/ynet/coze-studio`；前端根：`frontend/`。
- 组件库统一 `@coze-arch/coze-design`（Button/Modal/Input/Table/Tree/Tag/Toast/Menu/Typography/Space）；图标 `@coze-arch/coze-design/icons`（`IconCoz*`）。
- i18n 一律 `I18n.t('key')`（`@coze-arch/i18n`）；新键加在 `frontend/packages/arch/resources/studio-i18n-resource/src/locales/{zh-CN,en}.json`。
- 样式用 CSS Modules `*.module.less` + `--coz-*` token（`--coz-bg-max`/`--coz-fg-plus`/`--coz-stroke-primary`）；卡片圆角 14px、按钮/输入 4px；主按钮 `type="primary" theme="solid"`、次按钮 `theme="borderless"`。
- `ResType` 枚举：`frontend/packages/arch/idl/src/auto-generated/plugin_develop/namespaces/resource_resource_common.ts`，import 路径 `@coze-arch/idl/plugin_develop`，新增 `Strategy = 10`。
- 能力项类型常量（前端）：`'workflow' | 'plugin' | 'knowledge' | 'prompt'`，与后端一致。
- 验证：构建/类型检查需 `rush install`（本机若未装依赖，先 `rush install` 再 `rush build -t <pkg>`，或包内 `tsc --noEmit`）。本机不能构建时，至少做 ESLint/tsc 单包检查并在 220/226 联调验证（见用户环境约定）。
- 提交规范：Conventional Commits（`feat(strategy): ...`）。未经用户明确要求不 push。

## Verification 适配说明

UI 任务以 typecheck（`tsc --noEmit` / `rush build -t <pkg>`）+ 渲染冒烟 + 关键交互手动验证为主；有 store/纯函数处（transform）写 vitest 单测。本机依赖不全时按用户约定改 `docker/.env.debug` 对接 220/226 联调，不本地起 middleware。

---

## File Structure（决策锁定）

```
frontend/
├─ packages/arch/idl/src/auto-generated/plugin_develop/namespaces/resource_resource_common.ts  # [改] ResType.Strategy=10
├─ packages/arch/idl/.../strategy/*                                  # [生成] StrategyApi（由后端 strategy.thrift 生成）
├─ packages/arch/resources/studio-i18n-resource/src/locales/
│  ├─ zh-CN.json / en.json                                           # [改] 策略相关键
├─ packages/studio/workspace/entry-base/src/pages/library/
│  ├─ hooks/use-entity-configs/use-strategy-config.tsx               # [新] 策略 entity config（镜像 use-database-config）
│  └─ hooks/use-entity-configs/index.ts                              # [改] 导出 useStrategyConfig
├─ packages/studio/workspace/entry-adapter/src/pages/library/index.tsx  # [改] 注册 useStrategyConfig
├─ apps/coze-studio/src/routes.tsx                                   # [改] /strategy 路由 + lazy import
├─ packages/agent-ide/strategy/                                      # [新] 策略编辑器页 + BotStrategyModal（新包）
│  └─ src/{pages/strategy-editor, components/bot-strategy-modal, components/capability-add, components/scenario-list}
└─ packages/studio/stores/bot-detail/src/
   ├─ store/bot-skill/store.ts                                       # [改] strategies slice
   ├─ store/bot-skill/transform.ts                                   # [改] transformDto2Vo/Vo2Dto.strategy
   └─ types/skill.ts                                                 # [改] StrategyItemType
```

> 新包 `@coze-arch/...agent-ide/strategy`（或并入现有 workspace-base/strategy-adapter）需在对应 `package.json`/rush.json 注册；具体放置遵循仓库现有 agent-ide 子包约定。

---

## Task 1: ResType.Strategy + i18n 键

**Files:**
- Modify: `frontend/packages/arch/idl/src/auto-generated/plugin_develop/namespaces/resource_resource_common.ts:116-126`
- Modify: `frontend/packages/arch/resources/studio-i18n-resource/src/locales/zh-CN.json`
- Modify: `frontend/packages/arch/resources/studio-i18n-resource/src/locales/en.json`

**Interfaces:**
- Produces: `ResType.Strategy = 10`；i18n 键集合。

- [ ] **Step 1: 加枚举值**

`resource_resource_common.ts` 现有：
```typescript
export enum ResType {
  Plugin = 1,
  // ...
  Voice = 9,
}
```
追加：
```typescript
  Voice = 9,
  Strategy = 10,
}
```

- [ ] **Step 2: 加 i18n 键（zh-CN.json）**

在既有 `library_resource_type_*` 附近追加：
```json
  "library_resource_type_strategy": "策略",
  "navigation_workspace_library_strategy": "策略",
  "strategy_create": "新建策略",
  "strategy_scenario_add": "添加场景",
  "strategy_scenario_name": "场景名称",
  "strategy_capability_add": "添加能力项",
  "strategy_capability_type_workflow": "工作流",
  "strategy_capability_type_plugin": "插件",
  "strategy_capability_type_knowledge": "知识库",
  "strategy_capability_type_prompt": "提示词",
  "strategy_model_facing_desc": "面向模型的描述",
  "strategy_prompt_content": "提示词内容",
  "strategy_publish": "发布",
  "strategy_bind_hint": "提供 3 个渐进披露工具：列场景 / 列能力 / 调用"
```

- [ ] **Step 3: 加 i18n 键（en.json，同 key 英文值）**

```json
  "library_resource_type_strategy": "Strategy",
  "navigation_workspace_library_strategy": "Strategy",
  "strategy_create": "New Strategy",
  "strategy_scenario_add": "Add Scenario",
  "strategy_scenario_name": "Scenario name",
  "strategy_capability_add": "Add Capability",
  "strategy_capability_type_workflow": "Workflow",
  "strategy_capability_type_plugin": "Plugin",
  "strategy_capability_type_knowledge": "Knowledge",
  "strategy_capability_type_prompt": "Prompt",
  "strategy_model_facing_desc": "Model-facing description",
  "strategy_prompt_content": "Prompt content",
  "strategy_publish": "Publish",
  "strategy_bind_hint": "Provides 3 progressive-disclosure tools"
```

- [ ] **Step 4: 验证 + 提交**

Run: `cd frontend && npx tsc --noEmit -p packages/arch/idl/tsconfig.json`（或单包构建）
Expected: 无类型错误。
```bash
git add frontend/packages/arch/idl/src/auto-generated/plugin_develop/namespaces/resource_resource_common.ts frontend/packages/arch/resources/studio-i18n-resource/src/locales/zh-CN.json frontend/packages/arch/resources/studio-i18n-resource/src/locales/en.json
git commit -m "feat(strategy): add Strategy ResType and i18n keys (frontend)"
```

---

## Task 2: 生成 StrategyApi 客户端

**Files:**
- Generate: `frontend/packages/arch/idl/.../strategy/*`（前端 IDL→TS 客户端）

**Interfaces:**
- Produces: `StrategyApi.{ListStrategy,GetStrategy,AddStrategy,UpdateStrategy,DeleteStrategy,PublishStrategy, AddScenario, ..., AddCapability, ...}`，import 路径仿 `MemoryApi`（用于 `use-database-config` 的 `MemoryApi.DeleteDatabase`）。

- [ ] **Step 1: 由后端 strategy.thrift 生成前端客户端**

按仓库前端 IDL 生成流程（IDL2TS / 既有脚本，参考 `MemoryApi` 的生成来源），从后端 `idl/data/strategy/strategy_svc.thrift` 生成前端 API。确认生成产物导出 `StrategyApi` 与请求/响应类型。

- [ ] **Step 2: 验证 + 提交**

Run: 单包 `tsc --noEmit`。
```bash
git add frontend/packages/arch/idl
git commit -m "feat(strategy): generate StrategyApi frontend client"
```

---

## Task 3: 策略 entity config + 列表入口

**Files:**
- Create: `frontend/packages/studio/workspace/entry-base/src/pages/library/hooks/use-entity-configs/use-strategy-config.tsx`
- Modify: `.../use-entity-configs/index.ts`（导出）
- Modify: `frontend/packages/studio/workspace/entry-adapter/src/pages/library/index.tsx`

**Interfaces:**
- Consumes: `ResType.Strategy`（T1）、`StrategyApi`（T2）、`UseEntityConfigHook`
- Produces: `useStrategyConfig`，并注册进 `BaseLibraryPage` 的 `entityConfigs`

- [ ] **Step 1: 写 use-strategy-config（镜像 use-database-config.tsx）**

```tsx
import { useNavigate } from 'react-router-dom';
import { useRequest } from 'ahooks';
import { I18n } from '@coze-arch/i18n';
import { Toast, Menu } from '@coze-arch/coze-design';
import { IconCozWorkflow } from '@coze-arch/coze-design/icons'; // 替换为策略专属图标
import { ResType } from '@coze-arch/idl/plugin_develop';
import { StrategyApi } from '@coze-arch/idl/strategy';
import { TableAction } from '...'; // 复用 use-database-config 同来源
import { ActionKey } from '...';
import type { ResourceInfo } from '...';
import type { UseEntityConfigHook } from '../../types';

export const useStrategyConfig: UseEntityConfigHook = ({ spaceId, reloadList, getCommonActions }) => {
  const navigate = useNavigate();

  const { run: createStrategy } = useRequest(
    (name: string, description: string) => StrategyApi.AddStrategy({ space_id: spaceId, name, description }),
    { manual: true, onSuccess: res => { navigate(`/space/${spaceId}/strategy/${res.id}`); reloadList(); } },
  );
  const { run: deleteStrategy } = useRequest(
    (id: string) => StrategyApi.DeleteStrategy({ id }),
    { manual: true, onSuccess: () => { reloadList(); Toast.success(I18n.t('Delete_success')); } },
  );

  // 简单"新建策略"弹窗（名称+描述），完成后回调 createStrategy
  const { modal: createModal, open: openCreate } = useCreateStrategyModal({ onConfirm: createStrategy });

  return {
    modals: <>{createModal}</>,
    config: {
      typeFilter: { label: I18n.t('library_resource_type_strategy'), value: ResType.Strategy },
      onCreate: openCreate,
      renderCreateMenu: () => (
        <Menu.Item icon={<IconCozWorkflow />} onClick={openCreate}>{I18n.t('strategy_create')}</Menu.Item>
      ),
      target: [ResType.Strategy],
      onItemClick: (item: ResourceInfo) => navigate(`/space/${spaceId}/strategy/${item.res_id}`),
      renderActions: (item: ResourceInfo) => {
        const deleteDisabled = !item.actions?.find(a => a.key === ActionKey.Delete)?.enable;
        return (
          <TableAction
            deleteProps={{ disabled: deleteDisabled, deleteDesc: I18n.t('library_delete_desc'), handler: () => deleteStrategy(item.res_id || '') }}
            actionList={getCommonActions?.(item)}
          />
        );
      },
    },
  };
};
```
`useCreateStrategyModal` 用 `Modal`+`Input`（名称/描述）实现，仿 database 创建弹窗。

- [ ] **Step 2: 导出 + 注册**

`use-entity-configs/index.ts` 增加 `export { useStrategyConfig } from './use-strategy-config';`
`entry-adapter/.../library/index.tsx`（仿 `useDatabaseConfig`）：import → `const { config: strategyConfig, modals: strategyModals } = useStrategyConfig(configCommonParams);` → `entityConfigs` 数组加 `strategyConfig` → JSX 加 `{strategyModals}`。

- [ ] **Step 3: 验证 + 提交**

Run: 单包 typecheck；本机可跑则起 dev 看库页出现"策略"筛选与"新建策略"。
```bash
git add frontend/packages/studio/workspace/entry-base/src/pages/library/hooks/use-entity-configs frontend/packages/studio/workspace/entry-adapter/src/pages/library/index.tsx
git commit -m "feat(strategy): add strategy entity config and library entry"
```

---

## Task 4: /strategy 路由 + 编辑器页骨架

**Files:**
- Modify: `frontend/apps/coze-studio/src/routes.tsx`（lazy import + route）
- Create: `frontend/packages/agent-ide/strategy/src/pages/strategy-editor/index.tsx`
- Create: `.../strategy-editor/index.module.less`

**Interfaces:**
- Consumes: `StrategyApi.GetStrategy`
- Produces: `StrategyPage` 导出；路由 `/space/:space_id/strategy/:strategy_id`

- [ ] **Step 1: 加路由（仿 work_flow）**

`routes.tsx` 顶部 lazy：
```tsx
const StrategyPage = lazy(() =>
  import('@coze-agent-ide/strategy').then(res => ({ default: res.StrategyPage })),
);
```
在 library/work_flow 同级 children 加：
```tsx
{ path: 'strategy/:strategy_id', Component: StrategyPage, loader: () => ({ hasSider: false, requireAuth: true }) },
```

- [ ] **Step 2: 编辑器骨架（Header + 左场景列 + 右能力区，三栏）**

`strategy-editor/index.tsx` 用 `useParams` 取 `strategy_id`，`useRequest(StrategyApi.GetStrategy)` 拉详情；布局：顶部 Header（策略名/描述/`[发布]`），左侧场景列表占位，右侧能力项区占位。全部 `@coze-arch/coze-design` + `.module.less`（圆角/间距/token 按 Global Constraints）。空状态、loading（`Spin`）处理齐。

- [ ] **Step 3: 验证 + 提交**

Run: 单包 typecheck；可跑则点击库页策略卡进入 `/strategy/:id` 看到骨架渲染。
```bash
git add frontend/apps/coze-studio/src/routes.tsx frontend/packages/agent-ide/strategy
git commit -m "feat(strategy): add /strategy route and editor page skeleton"
```

---

## Task 5: 场景 CRUD 交互

**Files:**
- Create: `frontend/packages/agent-ide/strategy/src/components/scenario-list/index.tsx`
- Modify: `strategy-editor/index.tsx`（接入）

**Interfaces:**
- Consumes: `StrategyApi.{AddScenario,UpdateScenario,DeleteScenario}`
- Produces: 左侧场景列表组件（选中态 + 增/改名/删/拖拽排序）

- [ ] **Step 1: 场景列表组件**

`Tree` 或自定义列表渲染场景；底部"[+ 添加场景]"（`Modal`+`Input` 取名→`AddScenario`）；每项 hover 出"重命名/删除"（`Menu`）；选中项高亮（`--coz-*`）；选中后通知父组件 `onSelect(scenarioId)`。拖拽排序调 `UpdateScenario(sort_order)`（v1 可用上下移按钮替代拖拽以降复杂度，排序写回）。

- [ ] **Step 2: 接入编辑器，维护 selectedScenarioId**

`strategy-editor` 持 `selectedScenarioId` state，传给 scenario-list 与右侧能力区。增删后 `refetch` 详情。

- [ ] **Step 3: 验证 + 提交**

Run: typecheck；可跑则验证增/改名/删/选中。
```bash
git add frontend/packages/agent-ide/strategy/src/components/scenario-list frontend/packages/agent-ide/strategy/src/pages/strategy-editor
git commit -m "feat(strategy): scenario list CRUD in editor"
```

---

## Task 6: 能力项添加（四类型）

**Files:**
- Create: `frontend/packages/agent-ide/strategy/src/components/capability-add/index.tsx`
- Create: `.../capability-add/{workflow-picker,plugin-picker,knowledge-picker,prompt-editor}.tsx`
- Modify: `strategy-editor/index.tsx`（右侧能力区接入）

**Interfaces:**
- Consumes: `StrategyApi.AddCapability`、现有 `WorkflowModalBase`（`@coze-workflow/components`）、插件选择器、知识库选择器
- Produces: "[+ 添加能力项 ▾]" 分类型下拉 + 四个选择/编辑流

- [ ] **Step 1: 类型下拉**

`Menu` 列出 工作流/插件/知识库/提示词（i18n `strategy_capability_type_*` + 各类型 `IconCoz*`），点选打开对应 picker。

- [ ] **Step 2: 四类 picker**

- workflow：复用 `WorkflowModalBase`（过滤本空间已发布工作流）→ 选中得 `workflow_id`(+version)→ `AddCapability({scenario_id,type:'workflow',ref_id,ref_version})`。
- plugin：复用现有插件工具选择器 → 得 `plugin_id`+`tool_id` → `AddCapability({type:'plugin',ref_id:tool_id,ref_sub_id:plugin_id})`。
- knowledge：复用 KB 选择器 → 得 `knowledge_id` + 检索参数(top_k) → `AddCapability({type:'knowledge',ref_id,retrieve_config:JSON})`。
- prompt：`Modal`+富文本/`TextArea` 编辑 → `AddCapability({type:'prompt',prompt_content})`。

每个流都可填 `alias_name`/`alias_description`（面向模型描述，i18n `strategy_model_facing_desc`）。

- [ ] **Step 3: 验证 + 提交**

Run: typecheck；可跑则四类各加一条，确认列表出现并带类型标签。
```bash
git add frontend/packages/agent-ide/strategy/src/components/capability-add frontend/packages/agent-ide/strategy/src/pages/strategy-editor
git commit -m "feat(strategy): add capability pickers for workflow/plugin/knowledge/prompt"
```

---

## Task 7: 能力项列表渲染 + 行编辑 + 发布

**Files:**
- Create: `frontend/packages/agent-ide/strategy/src/components/capability-list/index.tsx`
- Modify: `strategy-editor/index.tsx`（Header 发布按钮）

**Interfaces:**
- Consumes: `StrategyApi.{UpdateCapability,DeleteCapability,PublishStrategy}`

- [ ] **Step 1: 能力项列表（Table）**

列：`[类型 Tag] 名称 | 面向模型描述 | 操作`。类型 Tag 用 `Tag` 配色区分四类。行内可编辑 alias_name/alias_description（`UpdateCapability`）；操作含删除、上下移排序（`UpdateCapability(sort_order)`）。

- [ ] **Step 2: 发布**

Header `[发布]` 按钮 → `PublishStrategy(id)` → `Toast.success`。已发布显示版本/状态标签。

- [ ] **Step 3: 验证 + 提交**

Run: typecheck；可跑则验证编辑描述、删除、排序、发布。
```bash
git add frontend/packages/agent-ide/strategy/src/components/capability-list frontend/packages/agent-ide/strategy/src/pages/strategy-editor
git commit -m "feat(strategy): capability list editing and strategy publish"
```

---

## Task 8: bot-skill store strategies slice

**Files:**
- Modify: `frontend/packages/studio/stores/bot-detail/src/types/skill.ts`（`StrategyItemType`）
- Modify: `frontend/packages/studio/stores/bot-detail/src/store/bot-skill/store.ts`
- Modify: `frontend/packages/studio/stores/bot-detail/src/store/bot-skill/transform.ts`
- Test: `frontend/packages/studio/stores/bot-detail/src/store/bot-skill/transform.test.ts`

**Interfaces:**
- Produces: `BotSkillStore.strategies: StrategyItemType[]`、`updateSkillStrategies`、`transformDto2Vo.strategy`、`transformVo2Dto.strategy`

- [ ] **Step 1: 类型**

`types/skill.ts` 加（仿 `WorkFlowItemType`）：
```typescript
export interface StrategyItemType {
  strategy_id: string;
  name: string;
  desc: string;
  icon_url?: string;
  version?: string;
}
```

- [ ] **Step 2: store slice（仿 workflows 的 72/113/178/221/305 行）**

- 默认：`getDefaultBotSkillStore()` 加 `strategies: []`
- 类型：`BotSkillStore` 加 `strategies: StrategyItemType[]`
- action 声明：`BotSkillAction` 加 `updateSkillStrategies: (s: StrategyItemType[]) => void`
- action 实现：`updateSkillStrategies: strategies => set(s => ({ ...s, strategies }))`
- init：`strategies: transformDto2Vo.strategy(botInfo?.strategy_info_list, optionData?.strategy_detail_map)`

- [ ] **Step 3: 写失败测试（transform 往返）**

`transform.test.ts`：
```typescript
it('strategy dto<->vo roundtrip', () => {
  const vo = transformDto2Vo.strategy([{ strategy_id: '1' }], { 1: { id: '1', name: 'S', description: 'd' } });
  expect(vo[0].name).toBe('S');
  const dto = transformVo2Dto.strategy(vo);
  expect(dto[0].strategy_id).toBe('1');
});
```
Run: `cd frontend && npx vitest run packages/studio/stores/bot-detail/src/store/bot-skill/transform.test.ts`
Expected: FAIL。

- [ ] **Step 4: 实现 transform（仿 transform.ts 的 workflow）**

`transformDto2Vo.strategy(list, detailMap)` 映射 `{strategy_id,name,desc,icon_url,version}`；`transformVo2Dto.strategy(strategies)` → `[{strategy_id}]`。

- [ ] **Step 5: 测试通过 + 提交**

Run: 同上 vitest。Expected: PASS。
```bash
git add frontend/packages/studio/stores/bot-detail/src/store/bot-skill frontend/packages/studio/stores/bot-detail/src/types/skill.ts
git commit -m "feat(strategy): add strategies slice to bot-skill store"
```

---

## Task 9: 智能体绑定弹窗 BotStrategyModal

**Files:**
- Create: `frontend/packages/agent-ide/strategy/src/components/bot-strategy-modal/index.tsx`
- Modify: 智能体技能区入口（agent-ide 技能面板，仿 workflow/knowledge 入口）接入

**Interfaces:**
- Consumes: `StrategyApi.ListStrategy`、`useBotSkillStore`（`strategies`/`updateSkillStrategies`，T8）

- [ ] **Step 1: 绑定弹窗（仿 external-knowledge-modal / BotWorkflowModal）**

`Modal` 列出本空间已发布策略（`ListStrategy`），多选/单选加入；确认时 `useBotSkillStore.getState().updateSkillStrategies(next)`。每行显示策略名+描述+"提供 3 个工具"提示（`strategy_bind_hint`）。

- [ ] **Step 2: 技能区入口 + 已绑展示**

在智能体技能面板（workflow/plugin 入口同处）加"策略"入口按钮打开弹窗；已绑策略以卡片渲染（名称 + `strategy_bind_hint` 副标题 + 移除按钮，移除调 `updateSkillStrategies`）。

- [ ] **Step 3: 验证 + 提交**

Run: typecheck；可跑则在智能体编辑页绑/解策略，保存后重载确认 `strategies` 持久化（依赖后端 Task 7 字段）。
```bash
git add frontend/packages/agent-ide/strategy/src/components/bot-strategy-modal
git commit -m "feat(strategy): bind strategy to single agent via skill panel"
```

---

## Self-Review（落计划时执行）

- **Spec 覆盖**：§5.1 列表入口→T1/T3；StrategyApi→T2；§5.3 编辑器→T4/T5/T6/T7；§5.4 绑定→T8/T9；§5.5 i18n→T1。全覆盖。
- **类型一致**：`StrategyItemType`/`ResType.Strategy`/能力项类型常量在 Global Constraints 固定；T3/T6/T9 复用。
- **占位扫描**：图标 `IconCozWorkflow` 为占位，实现时换策略专属 `IconCoz*`（在 `@coze-arch/coze-design/icons` 选语义相近项）——已显式标注，非隐藏 TODO。
- **依赖外部**：T9 持久化依赖后端 Task 7（SingleAgent.strategies）与 Task 5（StrategyApi）。后端未就绪时 T9 可先用 mock 数据完成 UI，再联调。

## 依赖顺序

（后端 Task 5 IDL 先行）→ 前端 T1 → T2 →（T3 列表；T4→T5→T6→T7 编辑器链）→ T8 → T9。
T4–T7 是编辑器主线，T8/T9 是绑定主线，二者可并行。
