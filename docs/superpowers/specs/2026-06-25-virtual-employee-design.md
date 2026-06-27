# 虚拟员工(Virtual Employee / agent_app)设计文档

- 日期: 2026-06-25
- 状态: 设计待评审
- 来源: 与用户头脑风暴产出

## 1. 背景与目标

把一个在平台上调好、绑定了技能、测试通过的「超级智能体」一键发布成可被他人复用的「虚拟员工」(AIProduct 产品类型 `agent_app`)。

目标行为:
- 发布时把虚拟员工的**身份(技能集/能力/模型/提示词)** 与 **运行环境(技能文件 + 已装依赖)** 一起固化成可复用模板。
- 别人从市场「招聘」后,平台为每个使用者**按需开一个独立沙箱实例**运行该虚拟员工。
- 虚拟员工的**技能在使用者侧只读、不可改**;运行**产物归使用者**,按使用者/空间隔离。
- 实例**空闲自动回收、再用再起**。

非目标(本期不做): 对外 API/嵌入式调用、用户自带外部存储、跨节点镜像分发、资源计费配额、知识库/工作流/数据库/插件的整体冻结(先做以技能为核心的虚拟员工)。

## 2. 复用的现有地基(约八成复用)

| 能力 | 现状 | 关键位置 |
|---|---|---|
| AIProduct 产品/版本/可见性/安装/卸载/升级/审计/权限矩阵 | 已用于 standard_skill,可直接套 agent_app | `backend/domain/aiproduct/*`、`backend/application/aiproduct/*` |
| 沙箱 manager(EnsureSandbox/Exec/Checkpoint/Restore) | 完整 | `backend/pkg/agentsandbox/manager.go` |
| 沙箱回收 Reaper(空闲 pause 10min / kill 1h + checkpoint 到 MinIO) | 完整 | `backend/pkg/agentsandbox/reaper.go` |
| 沙箱 key 寻址(connector,agent,user)、Redis 注册表多节点 | 完整 | `backend/pkg/agentsandbox/key.go`、`registry.go` |
| 技能注入沙箱(SyncSkill,按内容 hash 去重) | 完整 | `backend/pkg/agentsandbox/manager.go` `SyncSkill` |
| 超级体身份(模型/提示词/技能集/能力开关/MCP) | 完整 | `single_agent_draft`、`SuperAgentToolConfig` |
| 能力门控(按开关剔除工具) | 完整 | `agentflow/node_super_agent_tool_gate.go` |
| session_runtime_config + runtime resolver | 完整 | `super_agent_session_runtime_config` 表 |
| 对象存储(MinIO/TOS) | 完整,单 bucket `openynet` | `backend/infra/.../storage` |

## 3. 已确定的核心设计决策(与用户确认)

1. **模板形态**:不打 Docker 镜像;用沙箱 **checkpoint**(把 `/workspace` 打 tar 存 MinIO)当「镜像」。**发布时主动构建**模板快照(不是等首次使用)。
2. **产物隔离**:仍用平台 MinIO,按 `space_id/user_id` 的 object key 前缀做逻辑隔离。
3. **技能只读**:`/skills` 发布时固化进模板,使用者侧绝对只读(只读挂载 + 工具层拦截双保险);`/workspace`、`/outputs` 使用者可写,产物归己。
4. **使用场景**:平台内多用户,从市场招聘后在平台内对话使用;身份=平台登录用户;每使用者一个独立沙箱实例。

## 4. 整体架构与端到端流程

```
① 发布方(超级体 owner)点「发布为虚拟员工」
        │
        ▼
② 发布流程(后端,异步构建)
   a. 身份快照:model/prompt/技能版本集/能力开关/MCP → agent_app.feature.agent_snapshot
   b. 构建模板:起临时沙箱 → 注入技能(SyncSkill,pinned 版本) → 装依赖 →(smoke)
                → checkpoint ⇒ MinIO: templates/agent_app/{product}/{version}.tgz
   c. 登记产品:ai_product(type=agent_app) + ai_product_version 〔复用 AIProduct〕
   d. (发全局)走审核 〔复用 skill review〕
        │
        ▼
③ 市场:别人看到「虚拟员工」卡片 → 招聘(install)
        │  招聘时物化一个【只读影子 agent】(从 product 快照填充,标记 source_product_id)
        ▼
④ 使用者用影子 agent 开会话(平台内对话)
        │
        ▼
⑤ 实例化(每使用者一个沙箱;影子 agent 各自 agentID → 现有 key 天然隔离)
   冷启动:restore 模板快照(依赖已在) + /skills 只读挂载
        │
        ▼
⑥ 运行:用冻结身份干活;/skills 只读,/workspace、/outputs 可写
        产物 offload ⇒ MinIO: artifacts/s{space}/u{user}/p{product}/c{conv}/...
        │
        ▼
⑦ 回收:复用 Reaper(空闲 pause/kill + workspace checkpoint);再用再 restore
```

## 5. 数据模型

### 5.1 agent_app 产品(复用 `ai_product`, `type='agent_app'`)

`ai_product.feature`(身份快照 JSON):

```json
{
  "agent_snapshot": {
    "model": { "model_id": "1", "params": { "max_tokens": 8192 } },
    "prompt": "<系统提示>",
    "capabilities": { "sandbox": true, "web_search": true, "run_bash": true, "skill_manage": false },
    "mcp_servers": [ { "name": "company-docs", "type": "streamable_http", "endpoint_ref": "space_secret:..." } ],
    "skill_set": [
      { "skill_id": "765...", "skill_version": "4", "package_hash": "sha256:...", "name": "pdf-tools" }
    ]
  },
  "template": {
    "object_key": "templates/agent_app/{product}/{version}.tgz",
    "content_hash": "sha256:...",
    "build_status": "ready",
    "base_image": "ynet-sandbox:rich",
    "built_at": 1719000000000
  }
}
```

- MVP 身份快照只冻结 model + prompt + capabilities + mcp + skill_set;知识库/工作流/数据库/插件暂不纳入(后续按需扩展 `agent_snapshot`)。
- 模板快照与 `build_status` 也写进对应的 `ai_product_version.feature_snapshot`,实现按版本固化。

### 5.2 招聘 = 物化只读影子 agent

招聘(install agent_app product)时:
- 复用 `ai_product_installation` 记录招聘关系(target_space/user、product、pinned version)。
- 在使用者空间物化一个**只读影子 `single_agent_draft`**:`agent_type='super'`,从 `agent_snapshot` 填充(model/prompt/capabilities/mcp/skill_set);**新增列 `source_product_id`、`source_product_version`** 标记来源与只读。
- 使用者用这个影子 agent 开会话。复用现有 agent flow(基于 `agent_id` 运行),最小改动。

`single_agent_draft` 新增列: `source_product_id bigint NOT NULL DEFAULT 0`、`source_product_version varchar(64) NOT NULL DEFAULT ''`。`source_product_id != 0` 即「只读虚拟员工实例」。

### 5.3 沙箱 key 与实例

现状 key = `(connectorID, agentID, userID)`。每个使用者招聘各自的影子 agent → 各自独立 `agentID` → 现有 key 算法**天然给「每使用者一个独立沙箱实例」,无需改 key 维度**。

冷启动改造:当 agent 是虚拟员工实例(`source_product_id != 0`)时,首次 restore 来源 = product 模板快照(`templates/agent_app/{product}/{version}.tgz`),而非空白;并对 `/skills` 做只读挂载。

### 5.4 对象存储 key 规范

- 模板快照: `templates/agent_app/{productID}/{version}.tgz`
- 实例 workspace checkpoint(使用者运行数据): `workspaces/vemployee/s{space}/u{user}/{agentID}.tgz`
- 产物(`/outputs` offload): `artifacts/s{space}/u{user}/p{product}/c{conversation}/{filename}`

## 6. 发布流程

入口: 超级体编排页新增「发布为虚拟员工」按钮 → 弹窗(名称/描述/图标/可见性/版本号)。

后端 `PublishAgentApp`(异步,因构建可能慢):
1. 权限校验(creator 或 space manager)。
2. 身份快照:读 `single_agent_draft` → 冻结 `agent_snapshot`,pin 每个技能当前**已发布版本**。
3. 登记产品:写 `ai_product(type=agent_app, status=building)` + `ai_product_version`。
4. 异步构建模板 job:
   - 起临时构建沙箱(独立 build key,基础镜像 `ynet-sandbox:rich`)。
   - 注入技能:对 `skill_set` 每个 pinned 版本 `SyncSkill` 到 `/skills/<name>/`。
   - 装依赖:见 6.1。
   - (可选)smoke:对每个技能跑 manifest 里的 smoke 命令,失败标 warning。
   - `checkpoint` → `templates/agent_app/{product}/{version}.tgz`,记 `content_hash`。
   - 更新 `build_status=ready`(失败=`failed` + 原因);销毁构建沙箱。
5. 构建成功 → `status=published`(私有/空间)或 `reviewing`(全局,复用 skill review)。

前端轮询构建状态,展示 building / ready / failed。

### 6.1 依赖声明方式

技能依赖来源(MVP 同时支持 A+B):
- A. 技能包内 `requirements.txt` / `package.json` → 构建时自动 `pip install -r` / `npm i`。
- B. `SKILL.md` frontmatter 增加 `dependencies`(pip/npm 列表) → 构建时解析安装。
- apt 级系统依赖:MVP 仅允许 frontmatter 声明的**有限 apt 白名单**,避免任意提权。

## 7. 招聘 / 实例化 / 运行

- 招聘:install agent_app product → 物化只读影子 agent(5.2)。
- 开会话:使用者用影子 agent 开会话(现有 super-mode 会话机制)。
- 实例沙箱冷启动:`source_product_id != 0` → restore 模板快照 + `/skills` 只读挂载。
- 运行:agent flow 用快照身份(model/prompt/capabilities)运行;capabilities gate 用**冻结值**(使用者不能放权);`read_skill`/技能 = 模板里固化版本(只读)。
- 产物:`/outputs` → checkpoint/运行后 offload 到 `artifacts/s{space}/u{user}/...`。

## 8. 权限与隔离

- **技能只读(双保险)**: ① 沙箱 restore 后把 `/skills` 以 `:ro` 重挂(或 `chmod 0555`); ② 工具层 `write_file`/`edit_file`/`run_bash`(写重定向)对 `/skills` 前缀一律拒写。
- **身份只读**: `source_product_id != 0` 的影子 agent,编排页/`super-agent/config` 写接口拒绝;capabilities 不可被使用者放大。
- **产物隔离**: object key 含 `s{space}/u{user}`;list/download 校验 caller == owner。
- **凭证**: 模型/MCP 凭证沿用 `*_ref` 引用,不进快照明文,不入日志(与已落地的 data-URL 日志脱敏一致)。

## 9. 回收与生命周期

- 实例沙箱复用 Reaper:空闲 pause(10min)/kill(1h);kill 前 checkpoint 实例 workspace(使用者运行数据)到 `workspaces/vemployee/...`;再用再 restore(**优先实例 checkpoint,无则模板**,保留使用者运行进度)。
- 模板升级:product 发新版 → 新模板快照;使用者实例继续用招聘 pinned 版本(稳定);升级走 AIProduct upgrade(使用者主动),升级后下次冷启动用新模板。
- 卸载:install 状态置 `uninstalled`,影子 agent 停用,实例沙箱回收;产物保留(归使用者)。
- 模板 GC:`deprecated`/`archived` 且无活跃实例的旧模板快照定期清理。

## 10. 需评审确认的关键决策点

1. **招聘=物化只读影子 agent**(vs 运行时临时 config)。推荐影子 agent(复用现有按 agent_id 寻址的 agent flow/sandbox/resolver 最充分)。
2. **依赖声明** = 技能包 `requirements.txt`/`package.json` + `SKILL.md` frontmatter `dependencies`。
3. **实例冷启动 restore 优先级** = 实例自身 checkpoint > 模板(保留使用者运行进度)。
4. **可见性 MVP 先做 space**(空间内招聘),global + review 放 P1。

## 11. MVP 分期

**P0(最小闭环,平台内空间级)**
- agent_app 发布(身份快照 + 异步模板构建 + 登记产品),可见性 space。
- 市场招聘(install → 物化只读影子 agent)。
- 实例化:冷启动 restore 模板 + `/skills` 只读挂载 + 双保险拦截。
- 平台内对话运行;产物按 `s{space}/u{user}` 前缀。
- 复用 Reaper 回收;影子 agent 只读 UI。

**P1**
- 全局市场 + review;依赖声明规范化 + smoke;升级闭环;产物 offload 索引与下载/导出。

**P2**
- 对外 API/嵌入;用户自带外部存储;资源配额与监控;模板 GC 自动化。

## 12. 风险与缓解

- 模板构建慢/失败 → 异步 job + 状态机 + 重试 + 失败原因回显 + 构建超时上限。
- 模板快照大(含依赖) → 对象存储成本;按 `content_hash` 去重 + 旧版 GC。
- `/skills` 只读绕过 → 只读挂载 + 工具拦截双保险 + 审计。
- 影子 agent 泄漏写权限 → `source_product_id` 门控所有写路径 + 负向测试覆盖。
- 不破坏存量沙箱 → 影子 agent 各自 agentID,现有 `(agent,user)` 沙箱不受影响。
- 凭证泄漏 → 只存 `*_ref`,不进快照/日志明文。

## 13. 测试策略

- 单测:身份快照冻结、模板构建状态机、影子 agent 物化、capabilities gate 冻结、`/skills` 写拦截、产物 key 前缀、resolver 拒绝越权。
- 集成(226 Playwright):发布→构建→招聘→实例 restore→对话→产物隔离→回收 全链路。
- 回归:现有 skill 产品 + 普通/超级体会话 + 沙箱 checkpoint/restore 不破坏。

## 14. 验收标准

- 一个超级体能发布成 `agent_app` 产品,异步构建出模板快照(`build_status=ready`)。
- 另一个使用者能从空间市场招聘,物化出只读影子 agent。
- 使用者开会话即得到一个从模板 restore 的独立沙箱实例,冷启动不再装依赖。
- 使用者在实例里对 `/skills` 的任何写操作被拒(只读挂载 + 工具拦截);`/workspace`、`/outputs` 可写。
- 产物落在 `artifacts/s{space}/u{user}/...`,跨使用者不可见。
- 实例空闲被 Reaper 回收并 checkpoint,再次使用秒级恢复。
- 现有 skill 产品流程与普通/超级体会话回归通过。
