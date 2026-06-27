# AIProduct Phase 2 交付与接续说明

日期: 2026-06-21
仓库: `/Users/luzhipeng/projects/ynet/coze-studio`

## 目的

本阶段目标是把超级智能体从“调试型能力集合”推进到“公司级可运营产品体系”:

- 模型、MCP、标准技能、智能体应用统一抽象成 AIProduct。
- 支持空间 + 全局两级资产可见性。
- 支持发布、审核、安装、卸载、升级、审计。
- 标准技能不再只是 prompt 或 `skills.md`,而是以 `SKILL.md` 标准文件夹/ZIP 包作为产品来源。
- 会话可以绑定运行时配置,记录模型/MCP/技能产品选择和 resolved snapshot。
- App Server manifest/openapi 能让前端和外部客户端发现这些能力。

## 已实现

### 数据库

新增 SQL:

- `/Users/luzhipeng/projects/ynet/coze-studio/docs/ynet-database-sql/104-ai-product-foundation.sql`

包含表:

- `ai_product`
- `ai_product_version`
- `ai_product_installation`
- `ai_product_audit_log`
- `super_agent_session_runtime_config`

### AIProduct 领域层

新增目录:

- `/Users/luzhipeng/projects/ynet/coze-studio/backend/domain/aiproduct`

已覆盖:

- 产品类型: `model`, `mcp_server`, `standard_skill`, `agent_app`
- 可见性: `private`, `space`, `global`
- 状态: `draft`, `reviewing`, `published`, `deprecated`, `archived`
- 安装状态: `active`, `disabled`, `uninstalled`
- 产品同步、商城列表、可见性检查、安装、卸载、升级、审计
- 会话 runtime config 的 upsert/get/delete 存储接口

### 标准技能同步为产品

核心文件:

- `/Users/luzhipeng/projects/ynet/coze-studio/backend/application/skill/skill_application.go`
- `/Users/luzhipeng/projects/ynet/coze-studio/backend/application/aiproduct/product_application.go`
- `/Users/luzhipeng/projects/ynet/coze-studio/backend/application/application.go`

已接入同步触发:

- 创建技能
- 更新技能
- upsert/delete 技能 assets
- 发布技能
- 审核技能
- 安装 marketplace 技能

映射规则:

- 全局待审核技能 -> AIProduct `reviewing`
- 全局审核拒绝 -> AIProduct `archived`
- 空间/全局审核通过且有 published version -> AIProduct `published`
- 私有或未发布 -> AIProduct `draft`
- `source_ref_type=skill`, `source_ref_id=skill_id`

### 产品 API 与 App Server 契约

新增/修改:

- `/Users/luzhipeng/projects/ynet/coze-studio/backend/api/handler/coze/super_agent_product_service.go`
- `/Users/luzhipeng/projects/ynet/coze-studio/backend/api/handler/coze/super_agent_run_service.go`
- `/Users/luzhipeng/projects/ynet/coze-studio/backend/api/router/coze/api.go`
- `/Users/luzhipeng/projects/ynet/coze-studio/backend/api/handler/coze/superagentapp/run_request.go`

新增路由:

- `POST /api/super-agent/products/list`
- `POST /api/super-agent/products/get`
- `POST /api/super-agent/products/install`
- `POST /api/super-agent/products/upgrade`
- `POST /api/super-agent/products/uninstall`
- `POST /api/super-agent/marketplace/products/list`
- `POST /api/super-agent/marketplace/products/get`
- `POST /api/super-agent/marketplace/products/install`
- `POST /api/super-agent/runtime-config/get`
- `POST /api/super-agent/runtime-config/update`
- `POST /api/super-agent/runtime-config/delete`

Manifest/OpenAPI 已新增:

- capability: `products`
- capability: `runtime_config`
- `products` contract
- `runtime_config` contract
- 对应 request schema、entry routes、OpenAPI operations

### 会话 Runtime Config 最小闭环

已实现:

- conversation 维度保存运行时配置。
- handler 校验会话属主。
- update 时生成 `resolved_snapshot`。
- resolver 会检查绑定产品:
  - 当前空间/用户可见
  - 类型匹配
  - 已安装且状态为 `active`
- 支持绑定:
  - `model_product_id`
  - `mcp_product_ids`
  - `skill_product_ids`
  - `tool_policy`
  - `context_policy`

暂未做:

- 尚未把 resolved snapshot 接入 agentflow 的实际模型选择/MCP 工具装配/技能注入主链路。
- 尚未提供前端 runtime config 管理 UI。

## 还差什么

优先级建议:

1. 把 `super_agent_session_runtime_config.resolved_snapshot` 接到 agentflow builder。
2. MCP 产品化: MCP server 的注册、发布、安装、版本、权限、运行时装配。
3. 模型产品化: 模型供应商/凭据/额度/空间安装/会话选择。
4. 前端产品商城: `/explore/project/latest` 只保留技能商店后,再接产品 API 而不是旧 skill marketplace API。
5. 空间资产管理: 已安装产品列表、升级提示、卸载、版本回滚。
6. 权限与审计 UI: 展示谁发布、谁审核、谁安装、谁升级。
7. 部署到 226 后用 Playwright MCP 验证真实页面和 API。

## 测试记录

通过:

```bash
cd /Users/luzhipeng/projects/ynet/coze-studio/backend
SESSION_HMAC_SECRET=test-secret-for-super-agent go test ./api/handler/coze ./api/router/coze ./domain/aiproduct/... ./application/aiproduct ./application/skill ./api/handler/coze/superagentapp
```

结果:

- `api/handler/coze` passed
- `api/router/coze` passed
- `domain/aiproduct/...` passed
- `application/aiproduct` passed
- `application/skill` passed
- `api/handler/coze/superagentapp` passed

全量后端测试:

```bash
cd /Users/luzhipeng/projects/ynet/coze-studio/backend
SESSION_HMAC_SECRET=test-secret-for-super-agent go test ./...
```

结果: 失败,但失败点集中在已有不相关问题:

- `domain/memory/variables/internal/dal`: fmt `%s` 与 int64 参数不匹配
- `application/base/appinfra`: 测试引用未定义符号
- `application/space/import`: 测试调用签名未跟随 `GenerateMapping` 新参数
- `application/space/sync`: sqlite `ON CONFLICT` 约束缺失
- `infra/impl/modelmgr/database`: 测试未使用 import 和未定义 `entity`
- `infra/impl/rdb`: 本地 MySQL root 访问被拒绝
- 部分 workflow internal 测试有既有 nil panic/build failed

## 打包/上传到 226 注意事项

这轮只做本地实现与测试,尚未上传 226。

上传前必须先检查磁盘:

```bash
ssh dev@10.10.10.226 'df -h /; du -sh ~/coze-studio* 2>/dev/null || true'
```

建议:

- 不要连续保留大体积备份。
- 只上传 backend 二进制/必要静态产物。
- SQL 先在测试库执行 `docs/ynet-database-sql/104-ai-product-foundation.sql`。
- 部署后先测 API,再开 Playwright 验证页面。

部署后建议验证:

```bash
curl -sS http://10.10.10.226:8896/api/super-agent/manifest | jq '.data.capabilities,.data.products,.data.runtime_config'
curl -sS http://10.10.10.226:8896/api/super-agent/openapi.json | jq '.paths | keys[] | select(contains("products") or contains("runtime-config"))'
```

## 下一阶段目标话术

可以直接把下面这段作为新目标:

> 继续推进 AIProduct Phase 2: 在现有 AIProduct 数据库、领域层、标准技能同步、产品 API、runtime_config 最小闭环基础上,把 resolved snapshot 接入超级智能体 agentflow 运行时。要求模型产品、MCP 产品、标准技能产品都能按空间+全局安装结果解析; 会话 runtime config 能实际影响模型选择、MCP 工具装配、标准技能注入; 前端只保留技能商店并接产品 API 展示/安装/升级/卸载; 补权限、审计、版本升级策略; 在本地跑 focused tests,再检查 10.10.10.226 磁盘空间后部署并用 Playwright MCP 验证。
