# 超级智能体基础能力收敛交付记录

日期: 2026-06-21
仓库: `/Users/luzhipeng/projects/ynet/coze-studio`

## 目标

本阶段目标是先稳定超级智能体基础盘,不要继续扩张 AIProduct 产品层:

- 超级智能体 App Server 路由与 handler 可编译、可测、可被前端调用。
- 标准技能包能力可用: `SKILL.md` 文件夹技能、ZIP 导入/校验/导出、assets 管理、发布审核、空间/全局商城安装。
- harness/workspace/artifact/approval/trace/session 相关接口可被真实页面加载。
- 前端 schema、技能商城、空间技能、super mode 会话与 trace 基础视图不破坏主流程。
- 226 测试环境只读验证通过,上传/替换前必须先看磁盘空间,不能继续堆大备份。

## 本轮已修复

### 后端测试与编译阻塞

- 修复 `application/singleagent` 测试 fake 未实现当前 `SkillService` 接口的问题。
  - 文件: `/Users/luzhipeng/projects/ynet/coze-studio/backend/application/singleagent/sandbox_skill_test.go`
  - 补齐: review-aware `PublishSkill`, `ReviewSkill`, `ListPendingReviews`

- 修复 `api/handler/coze` 测试 import cycle。
  - 文件:
    - `/Users/luzhipeng/projects/ynet/coze-studio/backend/api/handler/coze/conversation_service_test.go`
    - `/Users/luzhipeng/projects/ynet/coze-studio/backend/api/handler/coze/message_service_test.go`
  - 做法: 这两个只调用导出 handler 的测试改为 `package coze_test`,显式使用 `coze.HandlerName`。

- 修复 `workflow_service_test.go` 漏 import `searchmock`。
  - 文件: `/Users/luzhipeng/projects/ynet/coze-studio/backend/api/handler/coze/workflow_service_test.go`

- 本地没有 MySQL 时,coze handler 集成测试不再继续跑到 panic。
  - 本地环境: skip。
  - CI 环境: 继续 fail,避免掩盖真实环境问题。

- 本地没有 `ynet-sandbox:rich` Docker 镜像时,agentsandbox 集成测试不再尝试从 Docker Hub 拉不存在的 public image。
  - 文件:
    - `/Users/luzhipeng/projects/ynet/coze-studio/backend/pkg/agentsandbox/integration_test.go`
    - `/Users/luzhipeng/projects/ynet/coze-studio/backend/pkg/agentsandbox/skill_injection_test.go`
  - 现在要求 Docker daemon 可用且本地存在 `ynet-sandbox:rich`,否则 skip。

## 当前已经实现的能力

### App Server / 超级智能体

核心路由集中在:

- `/Users/luzhipeng/projects/ynet/coze-studio/backend/api/router/coze/api.go`

已覆盖:

- `/api/super-agent/manifest`
- `/api/super-agent/openapi.json`
- runs: create/get/list/reply/stream/cancel
- sessions: create/get/list/rename/delete
- messages: list
- config: get/update
- approvals: list/resolve
- harness: state/plan/tool-outputs/cleanup/context/clear/resume/snapshot
- sandbox: exec
- traces: get
- artifacts: list/download/delete/move
- workspace: list/read/upload/write/download/delete/move/mkdir/stat/grep/glob/edit/patch
- skills: create/validate-package/import/import-runtime/export/get/update/delete/publish/list/assets
- marketplace: list/get/install

### 标准技能包与商城

核心实现:

- `/Users/luzhipeng/projects/ynet/coze-studio/backend/application/skill/skill_application.go`
- `/Users/luzhipeng/projects/ynet/coze-studio/backend/api/handler/skill/skill_service.go`
- `/Users/luzhipeng/projects/ynet/coze-studio/frontend/packages/arch/api-schema/src/idl/skill/skill.ts`
- `/Users/luzhipeng/projects/ynet/coze-studio/frontend/apps/coze-studio/src/pages/skill-marketplace/index.tsx`
- `/Users/luzhipeng/projects/ynet/coze-studio/frontend/apps/coze-studio/src/pages/space-skill/`

已覆盖:

- 标准 `SKILL.md` 技能文件夹。
- ZIP 上传导入。
- ZIP 校验但不入库。
- ZIP 导出。
- `assets/` 下图片/文件资产 upsert/list/get/delete。
- 全局/空间发布。
- 全局发布待审核、审核通过后进入全局商城。
- 从全局/空间商城安装到当前空间。

### Harness / Workspace / Trace

核心实现:

- `/Users/luzhipeng/projects/ynet/coze-studio/backend/application/singleagent/sandbox_workspace.go`
- `/Users/luzhipeng/projects/ynet/coze-studio/backend/application/singleagent/sandbox_artifact.go`
- `/Users/luzhipeng/projects/ynet/coze-studio/backend/application/singleagent/sandbox_skill.go`
- `/Users/luzhipeng/projects/ynet/coze-studio/backend/application/singleagent/super_agent_tool_config.go`
- `/Users/luzhipeng/projects/ynet/coze-studio/frontend/packages/agent-ide/entry/src/modes/super-mode/`

已覆盖:

- workspace 文件树、读写、上传、下载、移动、删除、mkdir、stat。
- grep/glob/edit/patch。
- sandbox exec。
- `/skills` 运行时技能目录展示。
- `/outputs` artifact 管理。
- harness plan、tool outputs、context summary、resume/snapshot。
- trace projection 与前端 trace 面板。

## 验证结果

### 后端 Go 测试

通过:

```bash
cd /Users/luzhipeng/projects/ynet/coze-studio/backend
SESSION_HMAC_SECRET=test-secret-for-super-agent go test -gcflags="all=-N -l" ./api/handler/coze
SESSION_HMAC_SECRET=test-secret-for-super-agent go test ./api/router/coze ./api/handler/skill ./application/skill ./application/singleagent ./domain/agent/singleagent/internal/agentflow ./domain/skill/...
SESSION_HMAC_SECRET=test-secret-for-super-agent go test ./api/middleware ./application/conversation ./domain/conversation/agentrun/service ./pkg/agentsandbox/...
```

说明:

- `api/handler/coze` 使用 Mockey,需要 `-gcflags="all=-N -l"`。
- `SESSION_HMAC_SECRET` 是当前测试必需环境变量。
- 本地缺 MySQL 时,workflow/conversation handler 集成测试 skip。
- 本地缺 `ynet-sandbox:rich` 时,agentsandbox Docker 集成测试 skip。

### 前端 Vitest

通过:

```bash
cd /Users/luzhipeng/projects/ynet/coze-studio/frontend/apps/coze-studio
rushx test -- --run src/pages/skill-marketplace/__tests__/index.test.tsx src/pages/skill-marketplace/__tests__/review-queue.test.tsx src/pages/space-skill/__tests__/standard-skill-files.test.ts src/pages/space-skill/__tests__/index.test.tsx

cd /Users/luzhipeng/projects/ynet/coze-studio/frontend/packages/arch/api-schema
rushx test -- --run __tests__/skill-super-agent.test.ts

cd /Users/luzhipeng/projects/ynet/coze-studio/frontend/packages/arch/bot-api
rushx test -- --run __tests__/developer-api-super-agent-runs.test.ts __tests__/developer-api-super-agent-workspace.test.ts __tests__/developer-api-super-agent-artifacts.test.ts __tests__/developer-api-super-agent-skills.test.ts __tests__/bot-open-api-super-agent-manifest.test.ts

cd /Users/luzhipeng/projects/ynet/coze-studio/frontend/packages/agent-ide/entry
rushx test -- --run src/modes/super-mode/__tests__/sandbox-workspace-utils.test.ts src/modes/super-mode/__tests__/super-session-sidebar.test.tsx src/modes/super-mode/codex-trace/__tests__/trace-bridge.test.tsx src/modes/super-mode/__tests__/super-hero.test.tsx src/modes/super-mode/__tests__/super-capabilities-section.test.tsx
```

结果:

- app 内技能商城/空间技能相关: 6 files, 24 tests passed。
- api-schema: 1 file, 1 test passed。
- bot-api: 7 files, 24 tests passed。
- agent-ide super-mode: 10 files, 45 tests passed。

### 前端 Typecheck

通过:

```bash
pnpm --dir /Users/luzhipeng/projects/ynet/coze-studio/frontend/packages/arch/api-schema exec tsc --noEmit -p tsconfig.json
pnpm --dir /Users/luzhipeng/projects/ynet/coze-studio/frontend/packages/arch/bot-api exec tsc --noEmit -p tsconfig.json
pnpm --dir /Users/luzhipeng/projects/ynet/coze-studio/frontend/packages/agent-ide/entry exec tsc --noEmit -p tsconfig.json
pnpm --dir /Users/luzhipeng/projects/ynet/coze-studio/frontend/apps/coze-studio exec tsc --noEmit -p tsconfig.json
```

### 226 Playwright MCP 验证

测试环境:

- URL: `http://10.10.10.226:8896`
- SSH: `dev@10.10.10.226`
- 本轮只读检查,没有上传或替换服务。

磁盘:

- `/`: 72G total, 61G used, 7.8G available, 89% used。
- 后续上传必须先 `df -h`,不要连续保留多个大备份。

技能商店:

- 页面: `http://10.10.10.226:8896/explore/project/latest`
- 结果: 页面可访问,显示「技能商店」侧边栏、英雄区、统计卡片、搜索框、3 个标准技能包卡片、详情/安装入口。
- 截图: `/Users/luzhipeng/projects/ynet/coze-studio/super-agent-skill-store-226-20260621.png`

超级智能体 arrange:

- 页面: `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange`
- 结果:
  - 会话列表存在,显示当前会话与最近会话。
  - 沙箱工作区存在。
  - 预览与调试区域存在。
  - 右下输入框存在,placeholder 为「发送消息...」。
  - 「人设 · 技能 · MCP」配置按钮存在。
  - 配置弹框可打开,包含模型、人设、能力权限、技能与 MCP、取消/完成。
- 截图: `/Users/luzhipeng/projects/ynet/coze-studio/super-agent-arrange-226-20260621.png`

网络:

- `/api/super-agent/config/get` => 200
- `/api/super-agent/sessions/list` => 200
- `/api/super-agent/workspace/list` => 200
- `/api/super-agent/harness/state` => 200
- `/api/super-agent/traces/get` => 200

## 如何谨慎打包和上传

本轮没有上传。下一次要部署时建议按这个顺序:

1. 先检查 226 空间:

```bash
ssh dev@10.10.10.226 'df -h /; docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"'
```

2. 本地先跑门禁:

```bash
cd /Users/luzhipeng/projects/ynet/coze-studio/backend
SESSION_HMAC_SECRET=test-secret-for-super-agent go test -gcflags="all=-N -l" ./api/handler/coze
SESSION_HMAC_SECRET=test-secret-for-super-agent go test ./api/router/coze ./api/handler/skill ./application/skill ./application/singleagent ./domain/agent/singleagent/internal/agentflow ./domain/skill/... ./api/middleware ./application/conversation ./domain/conversation/agentrun/service ./pkg/agentsandbox/...

pnpm --dir /Users/luzhipeng/projects/ynet/coze-studio/frontend/apps/coze-studio exec tsc --noEmit -p tsconfig.json
```

3. 前端静态资源上传时不要继续堆备份:

- 进入容器前先看 `/app/resources` 占用。
- 只保留一个最近可回滚点。
- 替换前保留 `/app/resources/static/config.js`。
- 替换后先 `curl -I http://10.10.10.226:8896/explore/project/latest`。
- 再用 Playwright MCP 打开技能商店和 arrange 页面。

4. 后端服务替换时也只保留一个回滚点:

- 先确认当前进程/容器路径。
- 备份当前二进制到一个带时间戳的文件。
- 上传新二进制后立即 smoke:
  - `/api/super-agent/manifest`
  - `/api/super-agent/openapi.json`
  - `/api/super-agent/sessions/list`
  - `/api/super-agent/harness/state`

## 还差什么

- 还没有真正的 AIProduct / 产品层抽象。
  - 当前只是 super-agent routes + skill marketplace + harness foundation。
  - session create 还没有模型产品、MCP 产品、技能产品、运行时资产配置。

- 标准技能商城已经能用,但还需要产品级治理:
  - 公司级分类、审核流状态页、安装量/评分/作者/版本升级。
  - 安装后版本锁定与升级提示。
  - 全局技能和空间技能的权限/审计。

- 远程环境未执行本轮代码上传。
  - 226 当前页面验证的是已部署版本。
  - 本轮本地修复主要是测试稳定性和交付收敛。

- 本地没有跑全量前端 monorepo 测试。
  - 已跑和 super-agent/skill 直接相关的 scoped tests/typecheck。

## 二阶段 AIProduct 参考计划

下一阶段再做 AIProduct,不要混进当前基础盘修复:

1. 定义产品层:
   - Model Product
   - MCP Product
   - Standard Skill Product
   - Agent/App Server Product

2. 定义安装关系:
   - 全局产品 -> 空间安装。
   - 空间产品 -> bot/session 绑定。
   - 版本锁定、升级、卸载、回滚。

3. 定义 session runtime config:
   - `model_product_id`
   - `mcp_product_ids`
   - `skill_product_ids`
   - `runtime_assets`
   - `tool_policy`
   - `context_policy`

4. 定义治理:
   - 发布审核。
   - 权限。
   - 审计日志。
   - 调用统计。
   - 风险等级。

5. 参考文档:
   - `/Users/luzhipeng/projects/ynet/coze-studio/docs/himarket-reference-comparison-plan-20260621.md`
   - `/Users/luzhipeng/projects/ynet/coze-studio/docs/super-agent-8896-playwright-test-record.md`
   - `/Users/luzhipeng/projects/ynet/coze-studio/docs/super-agent-feature-testing-guide.md`
   - `/Users/luzhipeng/projects/ynet/coze-studio/docs/super-agent-harness-handoff-20260620.md`
   - `/Users/luzhipeng/projects/ynet/coze-studio/docs/superpowers/specs/2026-06-20-hermes-evolvable-super-agent-adoption.md`
   - `/Users/luzhipeng/projects/ynet/coze-studio/docs/superpowers/specs/2026-06-20-super-agent-memory-redesign.md`
