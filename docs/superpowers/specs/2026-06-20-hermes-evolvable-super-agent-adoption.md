# Hermes 式「可进化」超级智能体 —— 借鉴分析与落地路线

Date: 2026-06-20

参考项目(已 clone 到本地 `/Users/luzhipeng/projects/ynet/hermes-ref/`):
- `NousResearch/hermes-agent`(主仓,Python,MIT)——"The self-improving AI agent / grows with you"
- `NousResearch/hermes-agent-self-evolution`(进化仓)——DSPy + GEPA 优化技能/提示词/代码

目标:把我们的平台超级体(`agent_type=super`,Go)从"能跑 Claude Code 式工具循环"升级为"**会自我进化**"的 harness,站在 Hermes 的肩膀上,而不是自己瞎试。

---

## 1. Hermes 的「可进化」到底是什么(基于源码)

Hermes 的自我改进是一个**三层栈**,核心数据是它自己的 **memory(我是谁的用户模型)** 和 **skills(这类任务怎么做)**:

### 第 1 层:每轮结束的「后台复盘 fork」(背后引擎)
- 每轮对话结束后,`run_agent.py` 调 `spawn_background_review_thread()`(`agent/background_review.py:700`)**fork 出一个隔离的 AIAgent**,带着本轮对话快照重放,**只给它 memory + skill_manage 两类工具**(白名单 `background_review.py:596`)。
- 复盘 agent 据此自主:把用户事实/偏好写进 memory;把"这类任务的做法/踩坑/用户纠正"写成或修补 skill;返回一行动作摘要("Memory: +3 facts · Skill 'foo' patched")。
- 触发技能创建的信号(`background_review.py:45-148` 的 `_SKILL_REVIEW_PROMPT`):用户纠正了风格/流程、出现了非平凡技巧、或已加载的 skill 被发现是错的 → **立刻 patch**。
- 硬约束:技能必须是**类级别**(可跨会话复用),拒绝 "fix-今天这个" 这种一次性命名。

### 第 2 层:周期性 nudge + Curator 策展
- **Nudge**(`agent/turn_context.py:203`):按轮次计数,每 ~5 轮提醒存 memory、每 ~3-5 次工具调用提醒更新/创建 skill;计数跨会话持久化,resume 不重复触发。
- **Curator**(`agent/curator.py`,1917 行):闲置触发(默认 7 天、空闲 >2h),把重叠的窄技能合并成"伞型"类级技能、把一次性细节下沉到 `references/templates/scripts/`、归档陈旧技能;有 **snapshot + 回滚**(`curator_backup.py`)保命,所有破坏性动作可恢复。
- **跨会话召回**:`hermes_state.py` 的 SQLite + **FTS5** 索引每条消息;`session_search` 工具(`tools/session_search_tool.py`)零 LLM 成本返回 top-N 历史会话 + ±5 消息上下文窗 + 首尾书签。

### 第 3 层:DSPy + GEPA 进化式优化(独立仓 hermes-agent-self-evolution)
- 把 skill/prompt/tool 描述包成 DSPy 模块 → 构造评测集(合成 + 从 SessionDB 挖真实用例 + 黄金集)→ **GEPA**(读执行轨迹理解"为什么失败"、做定向小突变,3-5 个样本即可,ICLR 2026)→ 在 holdout 上用 LLM-as-judge 评分 → 过约束门(全测试通过、字数上限、prompt 缓存兼容、benchmark 不回退)→ **以 PR + 人工审批**落地,绝不直接 commit。
- Agent 可自调用:检测低成功率技能 → 跑 `evolve_skill` → 评估 → 提改进版给人审批。

### 额外发现:还有两个我们用得上的 harness 能力
- **脚本内 RPC 调工具**(`tools/code_execution_tool.py`):Python 脚本通过 UDS/文件 RPC 调 `web_search/read_file/...`,**只有 stdout 回灌模型**,把多步工具链压成"零上下文成本"的一轮。
- **高质量上下文压缩**(`agent/context_compressor.py`):50% 上下文阈值触发;保护头(system+前3)+按 **token 预算**保护尾(~20K);中段交给**廉价辅助模型**用**结构化模板**(Historical Task / Completed / Active State / Pending …)总结;**迭代式更新**旧摘要而非重写;**时间锚定**(把"给John发邮件"改写成"已于 2026-06-20 发出")避免 resume 重复执行;密钥前后双重脱敏;孤儿 tool_call 修复;`[CONTEXT COMPACTION — REFERENCE ONLY]` 前后哨兵防止弱模型把摘要当新指令。

---

## 2. 我们 vs Hermes(差距对照)

| 能力 | 我们超级体(Go)现状 | Hermes |
|---|---|---|
| 工具循环 | ✅ ReAct + sandbox 工具,实测 11s 4 工具跑通 | ✅ |
| 沙箱 | ✅ 自有 agentsandbox | ✅ 6 后端(local/docker/ssh/modal/daytona/singularity) |
| 计划 | ⚠️ 有 update_plan 工具但模型不主动用 | ✅ |
| 技能(按需创建) | ✅ skill_manage 可造标准文件夹技能 | ✅ |
| **任务后自主造技能** | ❌ 无 | ✅ 后台复盘 fork |
| **技能使用中自我改进/patch** | ❌ 无 | ✅ 复盘检测+patch |
| **周期性 memory/skill nudge** | ❌ 无 | ✅ turn 计数触发 |
| **Curator 技能库策展+回滚** | ❌ 无 | ✅ |
| **跨会话 FTS 检索** | ❌ 无(只有会话内 + resume) | ✅ FTS5 session_search |
| **用户建模** | ⚠️ 有 memory_recall/save,无持续用户画像 | ✅ Honcho dialectic |
| **DSPy/GEPA 进化优化** | ❌ 无 | ✅ 独立仓 |
| 上下文压缩 | ⚠️ 规则式 v1(160KB 阈值、留 16 条、非 LLM) | ✅ 结构化 LLM + 迭代 + 时间锚定 |
| 续接 resume | ✅ 刚交付 harness/resume | ✅ session 链 + branch |
| 脚本内 RPC 调工具 | ❌ 无 | ✅ 零上下文成本多步 |

我们其实**底子很好**:已经有 skill_manage、memory 工具、session 隔离、沙箱、trace、resume、技能市场。Hermes 缺口里**最值钱的「闭环学习」我们 80% 的原料都有**,缺的是把它们串成"每轮自动复盘 → 写 memory/skill"的回路。

---

## 3. 落地路线(按 ROI 排序,全部围绕"可进化")

**阶段 A —— 闭环学习核心(最高 ROI,我们原料齐全)**
1. **后台复盘 fork**:每轮(或每个 super-agent run)结束,fork 一个隔离子代理,只给 memory + skill_manage 工具,带对话快照复盘,产出 memory/skill 写入 + 一行动作摘要进 trace。对应 Hermes `background_review.py`。我们已有 deep_task 的子代理 spawn 机制可复用。
2. **周期性 nudge**:session 状态里存 turn 计数 + last-nudge,跨阈值时往系统提示注入"存 memory / 沉淀 skill"提示。对应 `turn_context.py`。
3. **任务后自主造技能 + patch**:复盘 fork 调 skill_manage(create/patch),约束"类级别命名"。我们 skill_manage 已支持 create/edit/write_file/diff,只差 patch 语义 + 自主触发。
4. **plan-first**:让多步任务先 update_plan(本会话已实测缺失,已起草 prompt 改动,待并入)。

**阶段 B —— 召回与策展**
5. **FTS 跨会话检索**:给会话存储加全文索引 + `session_search` 工具(我们用 OB/MySQL,可用其全文或向量检索替代 FTS5)。
6. **Curator + snapshot/回滚**:周期性合并/归档技能,带快照保命。

**阶段 C —— 真·进化**
7. **上下文压缩升级**:规则式 → 结构化 LLM 摘要(辅助模型)+ 迭代更新 + 时间锚定 + 双重脱敏 + 哨兵。直接替换我们 v1 压缩。
8. **DSPy/GEPA 技能优化**:离线进化技能/提示词,LLM-as-judge + 约束门 + 人工审批。最大工程量,放最后。

**横向增强**
9. **脚本内 RPC 调工具**:把多步工具链压成零上下文成本一轮(沙箱里已能跑 python)。

---

## 4. 建议的第一刀

**阶段 A 的 #1+#2+#3 合起来就是 Hermes 的「闭环学习」最小可用版**,且我们原料齐全、纯后端、可 TDD、与对方足迹不冲突:

> 每个 super-agent run 结束 → fork 一个"复盘"子代理(白名单 memory_recall/memory_save/skill_manage)→ 它读本轮轨迹,沉淀用户事实到 memory、把可复用做法/用户纠正写成或 patch 类级 skill → 动作摘要进 trace/harness 状态。再加 turn 计数 nudge 兜底。

这一刀落地后,我们的超级体就从"每次从零开始"变成"**用得越多越懂你、技能越攒越厚**"——这正是用户要的 evolvable。

验证:真实端到端——同一个用户连续两个会话,第二个会话能用到第一个会话沉淀的 skill/memory(对齐"真实测试到位再说完成")。

---

## 5. 闭环学习 MVP —— 代码落地计划(已摸准集成点)

集成点(均已在代码中确认):
- **触发钩子**:`backend/api/handler/coze/super_agent_run_service.go` 的 `superAgentRun` 非流式分支,在 `OpenapiAgentRunNoStream` 成功返回后异步触发(goroutine)。只作用于 `/api/super-agent/*` + `agent_type=super`,不碰普通 agent/conversation。
- **复盘 fork = 受限 react agent**:复用 `react.NewAgent`,只挂白名单工具 `{memory_save, memory_recall, skill_manage}`(均已存在:`node_tool_memory.go`、`node_tool_skillmanage.go`),系统提示=复盘提示词,输入=本轮轨迹(从 message 服务取最近消息)。跑完即把 memory(沙箱内追加)、skill(沙箱 `/skills/<name>/` create/edit)写回**同一 sandbox key**。
- **写路径零重构**:memory 是 `crosssandbox.Manager` 上按 key 的字符串列表追加;skill_manage 已支持 create/edit(=patch)/write_file。复盘 agent 直接调这些工具即可,无需抽函数。
- **chatModel** 复用主 run 的同款(`BuildAgent` 里构造 chatModel 的同一路径)。

MVP 增量(按可测试性):
1. **复盘提示词** `SuperAgentReviewPrompt`(`system_prompt.go`)——指示:读本轮轨迹 → 提炼用户事实/偏好 `memory_save`;把可复用做法/用户纠正写成或 patch **类级** skill(拒绝一次性命名);只动 memory/skill,不答题。单测断言其内容(对齐 `SuperAgentExtraPrompt` 测法)。✅ 第一刀
2. **受限复盘 agent 构造器** `buildSuperAgentReviewAgent(ctx, conf, chatModel)`——只挂白名单工具;单测断言工具集只含 memory/skill,绝无 run_bash/网络等。
3. **异步触发 + 守卫**:run 完成后 goroutine 跑复盘;守卫=仅超级体、轨迹过短跳过、超时、单会话并发去重、把"review: +N memory · skill X 创建/patch"写进 trace/harness 状态。
4. **周期 nudge 兜底**:harness/session 状态存 turn 计数,跨阈值往系统提示注入"沉淀 memory/skill"提示。

验收(真实双会话):会话1 完成后复盘自动 `memory_save` + 造一个类级 skill;会话2(同 agent)`memory_recall` 能读到、`/skills` 里能看到该 skill。Playwright MCP 在 226 实测留证。

风险/守卫:复盘是额外 LLM 调用→必须异步非阻塞、有超时和频率上限;skill 命名必须类级(否则技能库被一次性垃圾淹没,正是 Hermes Curator 要解决的问题,本 MVP 先靠提示词约束 + 后续 Curator 兜底)。
