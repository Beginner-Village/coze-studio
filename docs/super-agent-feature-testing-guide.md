# 超级体新增功能 —— 测试指南(2026-06-21)

本次会话交付并部署到 `http://10.10.10.226:8896` 的功能,以及逐一的测试方法。

## 0. 环境与固定值

| 项 | 值 |
|---|---|
| 测试环境 | `http://10.10.10.226:8896` |
| 测试 space | `7652614054615187456` |
| 测试 super-agent(bot) | `7652617174313336832` |
| 测试 user_id | `7652614054610993152` |
| 当前部署二进制 | `de49b3c7…`(回滚 `f52d0578…`) |
| DB | `ssh dev@10.10.10.226` → `docker exec coze-mysql mysql -uroot -proot openynet` |

**如何调 API(带登录态)**:浏览器登录后打开
`http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange`,
按 F12 打开 Console,把下面每个 `fetch` 片段粘进去回车即可(自动带会话 cookie)。
Console 里先定义一个助手:
```js
const B='http://10.10.10.226:8896/api/super-agent';
const P=async(p,b)=>(await fetch(B+p,{method:'POST',headers:{'Content-Type':'application/json'},credentials:'include',body:JSON.stringify(b)})).json();
const SPACE='7652614054615187456',BOT='7652617174313336832',USER='7652614054610993152';
```

---

## 1. 本地单元测试(一条命令验证大部分逻辑)

```bash
cd /Users/luzhipeng/projects/ynet/coze-studio/backend

# super-agent 全域 + 新增 usermemory 包 + 路由/handler
SESSION_HMAC_SECRET=test-secret go test \
  ./domain/agent/singleagent/... \
  ./api/router/coze ./api/handler/coze/superagenttrace \
  ./application/singleagent ./application/skill -count=1
```
关键测试名(可单独 `-run`):
- `TestUserMemoryDAO_*` —— per-user DB 记忆 DAO(upsert/隔离/召回)
- `TestMemoryTools` —— memory_save/recall 工具走 DB、per-user 隔离、无用户优雅降级
- `TestRunPostRunReviewNoOpsForNonSuperAgent` —— **闭环复盘对普通 agent no-op(隔离铁律)**
- `TestSuperAgentReviewToolsetIsWhitelistedToMemoryAndSkill` —— 复盘 agent 只有 memory+skill,无 run_bash/web
- `TestLLMCompactionSummary` / `TestBuildContextCompactionSummaryPrefersLLMWhenAvailable` —— 压缩 LLM 升级 + 回退
- `TestPreHandlerReqDoesNotCompactNormalAgentHistory` —— **压缩对普通 agent 不触发(隔离铁律)**
- `TestValidateStandardSkillFilesRejectsForbiddenExtensions` —— 技能包禁可执行扩展名
- `TestSuperAgentHarnessResumeRouteReturnsLightweightHandoff` —— resume 端点

---

## 2. resume 续接端点

浏览器 Console:
```js
// 建会话→续接 handoff→缺 conversation_id 应被拒
const s = await P('/sessions/create',{space_id:SPACE,bot_id:BOT,title:'resume test'});
const conv = s.data.session.conversation_id;
console.log('resume:', await P('/harness/resume',{conversation_id:conv,bot_id:BOT,message_limit:5}));
console.log('reject(无conv):', await P('/harness/resume',{bot_id:BOT}));  // 期望 code=400
await P('/sessions/delete',{conversation_id:conv});
```
**期望**:第一条返回扁平 resume 对象(`version:'v1'`、`components:[messages,harness,context,tool_outputs,artifacts]`、`prompt` 非空);第二条 `code:400 conversation_id is required`。

---

## 3. per-user DB 记忆 + 闭环学习复盘(核心「grows with you」)

这是最重要的一条。流程:会话1 陈述事实 → 异步复盘 fork 自动把结构化记忆写进 DB → 查 DB 验证。

浏览器 Console:
```js
const s1 = await P('/sessions/create',{space_id:SPACE,bot_id:BOT,title:'mem test1'});
const c1 = s1.data.session.conversation_id;
await P('/runs/reply',{conversation_id:c1,bot_id:BOT,space_id:SPACE,user_id:USER,
  additional_messages:[{role:'user',content:'记住:我叫 Alex,主力语言 Go,生产部署在 224。',content_type:'text'}]});
console.log('已发,等 ~10-30s 让异步复盘写库');
```
等约 10–30 秒后,**查数据库**(新开终端):
```bash
ssh dev@10.10.10.226 'docker exec coze-mysql mysql --default-character-set=utf8mb4 -uroot -proot openynet \
  -e "SELECT id,kind,mem_key,content,agent_id FROM super_agent_user_memory WHERE user_id=7652614054610993152;"'
```
**期望**:出现结构化记忆行,例如
`kind=1(profile) user_name 用户的名字是 Alex` /
`kind=2(preference) programming_language …Go` /
`kind=3(fact) production_server …224`,且 `agent_id=7652617174313336832`。
> 说明:**run 自身不写这张表,是 run 结束后的「复盘 fork」自动写的**——这就是闭环学习。
> 注意复盘是 LLM 判断的,偶尔会判定"无需保存"而不写;换一句更明确的"请记住…"再试。

清理:
```bash
ssh dev@10.10.10.226 'docker exec coze-mysql mysql -uroot -proot openynet -e \
  "DELETE FROM super_agent_user_memory WHERE user_id=7652614054610993152;"'
```

---

## 4. 跨会话召回(grows with you 的可见效果)

接上一步(DB 里已有 Alex/Go/224),浏览器 Console **新建一个会话**问:
```js
const s2 = await P('/sessions/create',{space_id:SPACE,bot_id:BOT,title:'mem test2'});
const c2 = s2.data.session.conversation_id;
const r = await P('/runs/reply',{conversation_id:c2,bot_id:BOT,space_id:SPACE,user_id:USER,
  additional_messages:[{role:'user',content:'你还记得我吗?我叫什么?用什么语言?部署在哪?',content_type:'text'}]});
console.log((r.messages||[]).filter(m=>m.type==='answer').map(m=>m.content).join('\n'));
await P('/sessions/delete',{conversation_id:c2});
```
**期望**:答出 Alex / Go / 224。
> ⚠️ 已知 caveat:该测试 agent 还绑了**预先存在的 `getKeywordMemory`**(共享变量记忆),目前跨会话召回可能走它而非新的 `memory_recall`。两套记忆并存;新 DB 记忆由复盘 fork 可靠写入(见第 3 步直接查库),但让 agent 召回时优先用新系统的"统一"改造留待有监督会话(避免影响普通 agent)。

---

## 5. 上下文压缩 LLM 升级(超级体、长会话才触发)

只在**超级体**且历史超过 `AGENT_CONTEXT_COMPACT_MAX_BYTES`(~160KB)时触发,日常对话难手动触发。建议靠**单测**验证逻辑(第 1 节 `TestLLMCompactionSummary` 等)。若要线上观察:让超级体进行一段很长(几十轮、大量长文本)的会话,压缩触发后会在沙箱写
`/workspace/.agent/sessions/<conv>/context-summary.json`,可经 `POST /harness/state` 或 `/workspace/read` 查看其 `summary`(LLM 生成的结构化 checkpoint)。

---

## 6. 技能包禁可执行扩展名(技能市场安全)

浏览器 Console(用一段内置 base64 的坏包/好包):
```js
// 坏包:含 scripts/evil.exe  好包:含 scripts/run.py
const BAD='UEsDBBQAAAAAAFdG1VznewYELAAAACwAAAATAAAAZGVtby1za2lsbC9TS0lMTC5tZC0tLQpuYW1lOiBkZW1vLXNraWxsCmRlc2NyaXB0aW9uOiBkCi0tLQpib2R5UEsDBBQAAAAAAFdG1VzKbq21CQAAAAkAAAAbAAAAZGVtby1za2lsbC9zY3JpcHRzL2V2aWwuZXhlTVogYmluYXJ5UEsBAhQDFAAAAAAAV0bVXOd7BgQsAAAALAAAABMAAAAAAAAAAAAAAIABAAAAAGRlbW8tc2tpbGwvU0tJTEwubWRQSwECFAMUAAAAAABXRtVcym6ttQkAAAAJAAAAGwAAAAAAAAAAAAAAgAFdAAAAZGVtby1za2lsbC9zY3JpcHRzL2V2aWwuZXhlUEsFBgAAAAACAAIAigAAAJ8AAAAAAA==';
const r = await P('/skills/validate-package',{content:'data:application/zip;base64,'+BAD,filename:'demo.zip'});
console.log(r.data.validation);  // 期望 valid:false, error 含 "forbidden executable/binary file type ... scripts/evil.exe"
```
**期望**:`valid:false`,`error` 提示 forbidden executable/binary file type。把内容换成正常包(只有 .md/.py/图片)则 `valid:true`。

---

## 7. 隔离铁律验证:超级体功能不影响普通单智能体

- **代码层(最快)**:第 1 节里的两个隔离单测必须绿:
  `TestRunPostRunReviewNoOpsForNonSuperAgent`(复盘对普通 agent no-op)、
  `TestPreHandlerReqDoesNotCompactNormalAgentHistory`(压缩对普通 agent 不触发)。
- **线上层**:用一个**普通**(非 super)agent 跑一次正常对话,确认:① 行为/回答正常;② 它**没有** memory_save/memory_recall/skill_manage 这些超级体扩展工具;③ 不会触发复盘(DB `super_agent_user_memory` 不会因普通 agent 增行)。

---

## 一句话回归脚本(本地)
```bash
cd /Users/luzhipeng/projects/ynet/coze-studio/backend && \
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/... ./application/skill ./api/router/coze -count=1 && \
echo ALL_GREEN
```
