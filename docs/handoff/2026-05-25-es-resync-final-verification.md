# ES Resync 功能 — 220 dev 完整 e2e 验证报告

**Date**: 2026-05-25
**Branch**: feat/es-resync
**Tester user**: 351220960@qq.com (user_id 7639215472277192704)
**Test space**: `7639215472289775616` (Personal Space)

**Commits** (按顺序):
- `00ec5ffb4` Task 1 ResyncES Request/Response types
- `00025b782` Task 2 searchStore.DeleteByQuery infra
- `1869c4942` Task 3 ListBySpaceID for agent/resource/knowledge
- `0404806f2` Task 4 search domain ResyncSpace + test
- `4f8b5889c` Task 5 knowledge ResyncSpaceSlices + DeleteIndex
- `d5b61acd4` Task 6 SpaceApplicationService.ResyncES + permission
- `48a155bf3` Task 7 HTTP handler `/api/space/resync_es` + route
- `08ea665d6` Task 8 backend integration test handoff
- `b54964739` Task 9 SpaceApi.resyncES SDK
- `7dc6261a3` Task 10 DataMaintenanceSection component
- `36d2366a4` Task 11 wire-in to `space-management/index.tsx`

## TL;DR

**DONE_WITH_CONCERNS** — 端到端 happy path (浏览器 → nginx → backend → ES + RMQ)
完整跑通,HTTP 200 + counts 完全符合预期,ES 删除 + resync 修复 cycle 全 work。
但 Task 11 接入位置 (`pages/space-management/index.tsx`) 在 `routes.tsx` 没有挂载,
属于孤立页面 —— 真实用户在 UI 上看不到 "数据维护" 入口,需要在跟进任务里接到
左侧 SpaceLayout sub-menu (跟 `embedding-config` / `rerank-config` 同模式)。
后端代码 + API + SDK 完全 OK,前端组件本身也 OK,只是入口 navigation 缺失。

## Step 1: 后端 binary 确认

Task 8 的 hot-swap binary 还在 (May 25 13:44, 164M),没被覆盖。

```
ssh dev@10.10.10.220 'docker exec ynet-server ls -lh /app/openynet*'
-rwxr-xr-x 164.0M May 25 13:44 /app/openynet
-rwxr-xr-x 164.0M May 25 13:44 /app/openynet.bak-task8-1779716658
```

POST /api/space/resync_es 路由已经注册 (smoke test return HTTP 200 即使无 cookie):
```
curl http://localhost:9888/api/space/resync_es POST → HTTP 200 (会进 handler,只是无 session)
```

不需要重 build / hot-swap,直接复用。

## Step 2: 前端 dist 重 build + 部署

### Build (rush --to @coze-studio/app)

```
cd /Users/luzhipeng/projects/ynet/coze-studio
rush build --to @coze-studio/app

==[ SUCCESS WITH WARNINGS: 1 operation ]==
--[ WARNING: @coze-studio/app ]-----------[ 20.34 seconds ]--
(只是 browserslist + baseline-browser-mapping 过期 warning,非 build 错误)
rush build (21.09 seconds)
```

Build 成功,21 秒 (incremental,大部分包 cached)。dist:
```
frontend/apps/coze-studio/dist/ (306M, 1202 files)
  static/js/index~1.fdd79cea.js    May 25 22:14 (new hash, 含 Task 10/11 代码)
  static/js/index~2.e494e1e4.js    May 25 22:14
```

验证 dist 含新代码:
```
grep -rl "数据维护\|DataMaintenanceSection\|/api/space/resync_es" frontend/apps/coze-studio/dist
→ static/js/index~1.fdd79cea.js (含 "/api/space/resync_es" string)
→ static/js/index~2.e494e1e4.js (含 "数据维护" string + component)
```

### 部署到 220 ynet-web 容器

```
scp -r frontend/apps/coze-studio/dist dev@10.10.10.220:/tmp/coze-studio-dist-task12
# (306M, ~3 分钟)

ssh dev@10.10.10.220 '
  docker cp /tmp/coze-studio-dist-task12/. ynet-web:/usr/share/nginx/html/ &&
  docker exec ynet-web ls -lh /usr/share/nginx/html/
'
total 24K
-rw-r--r--    1 root root  497 Apr 16  2024 50x.html
-rw-r--r--    1 1000 1000  193 May 25 14:14 config.js
-rw-r--r--    1 1000 1000 3.0K May 25 14:14 favicon.png
-rw-r--r--    1 1000 1000 1.3K May 25 14:14 index.html
drwxr-xr-x    1 1000 1000 4.0K May 25 14:22 static
```

部署 OK,容器内 nginx html 已是最新 (May 25 14:14)。验证容器内含 "数据维护":
```
docker exec ynet-web grep -l 数据维护 /usr/share/nginx/html/static/js/*.js
→ /usr/share/nginx/html/static/js/index~2.e494e1e4.js
```

## Step 3: 浏览器 e2e (playwright)

### 尝试访问 `/space-management` 路由 → 失败 (路由未注册)

```
http://10.10.10.220:9888/space-management → 渲染 GlobalError 页面
  heading: "无法查看智能体"
  paragraph: "请检查你的网址或加入对应工作空间后重试"
```

排查发现:
- `frontend/apps/coze-studio/src/pages/space-management.tsx` 只 re-export `./space-management/index`
- `frontend/apps/coze-studio/src/routes.tsx` **完全没有** `path: 'space-management'` 注册项
- 全 codebase grep `pages/space-management` 没有任何 lazy import 引用
- 老 web bundle 也没有 `/space-management` 路由

**结论**: `SpaceManagementPage`(Task 11 接入的页面) 在 React Router 系统是孤立组件,
不能通过 URL 直达。真实 ynet 系统所有"空间管理"功能 (export-import, embedding-config,
rerank-config, members 等) 都挂在 `/space/:space_id/<sub-route>` 模式,sub-menu 在
`frontend/packages/foundation/space-ui-adapter/src/const.ts` 的 `SpaceSubModuleEnum` 注册。

### 用 playwright 在已登录态浏览器 fetch 验证 e2e (代替 UI 点击)

由于 UI 入口缺失,UI 点击路径不可达。但 SDK + 后端是 OK 的,所以用浏览器
**已登录 session** 直接 `fetch('/api/space/resync_es', POST)` 走完整 nginx → backend
链路,验证从 cookie 到 ES 的端到端:

```js
// 在 http://10.10.10.220:9888/space/.../develop 已登录页面执行
const res = await fetch('/api/space/resync_es', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  credentials: 'include',
  body: JSON.stringify({ space_id: '7639215472289775616' })
});
// → { status: 200, body: '{"code":0,"msg":"success","counts":{...}}' }
```

**结果**: HTTP 200,无 console error。SDK 调用的同样 endpoint + cookie 链路完全 work,
说明 Task 9 SDK + 后端集成 + nginx 代理 + session middleware 全 OK。

## Step 4: ES 不一致制造 + resync 修复 cycle

### Baseline
```
project_draft   docs=4   deleted=0
kb_entries      docs=233 deleted=0
coze_resource   docs=0   deleted=0
```

### 制造不一致 (delete 2 docs)
```
DELETE http://localhost:9200/project_draft/_doc/7643321617388404736
DELETE http://localhost:9200/kb_entries/_doc/7639991695316090880

project_draft   docs=3   deleted=2  ← -1
kb_entries      docs=232 deleted=2  ← -1
coze_resource   docs=0
```

### 调用 resync
```
curl -X POST http://localhost:8891/api/space/resync_es \
  -H "Cookie: i18next=zh-CN; session_key=<owner>" \
  -d '{"space_id":"7639215472289775616"}'

HTTP=200 time=1.746s
{"code":0,"msg":"success","counts":{
  "project_draft":4,
  "coze_resource":0,
  "kb_entries":4,
  "slice_reindex_jobs":439
}}
```

### Verify ES recovered
```
project_draft   docs=4  deleted=4  ← 恢复 (3→4, 删 tombstone +1)
kb_entries      docs=233 deleted=0 ← 恢复 (232→233, merge 已清 tombstone)
coze_resource   docs=0            (符合 Task 8 注释 - 该 space app_draft 表为空)
```

**完美回复** —— 删除的 doc 通过 ListBySpaceID + bulk upsert 完整 rewrite,
slice_reindex_jobs=439 个事件入 RMQ (`openynet_knowledge` topic)。

### Slice consumer 验证

```
docker logs --tail 300 ynet-server | grep indexSlice
→ event_handle.go:566/567 (*knowledgeSVC).indexSlice 大量 firing
```

Consumer 在消费 slice events。下游 Milvus 错误 (Task 8 文档的 `105000002 非可重试`)
是 **220 dev 环境问题** (vector store collection schema 异常),跟 Task 1-7 代码无关。

## 总览矩阵

| Check | 状态 | 说明 |
|---|---|---|
| 后端 binary 已部署 | PASS | Task 8 hot-swap binary 还在 |
| `/api/space/resync_es` 路由注册 | PASS | smoke test 200 |
| 前端 rush build | PASS | 21s,SUCCESS WITH WARNINGS (无 error) |
| 前端 dist 含 "数据维护" + 新 component | PASS | hash 是 `fdd79cea.js`/`e494e1e4.js` |
| 部署到 220 ynet-web 容器 | PASS | docker cp,nginx html 已更新 |
| 浏览器登录态正常 | PASS | 351220960@qq.com session 有效 |
| 浏览器直 fetch resync_es | PASS | HTTP 200,counts 正确 |
| ES 删除 + resync 修复 | PASS | project_draft 3→4, kb_entries 232→233 |
| RMQ slice events 入队 | PASS | slice_reindex_jobs=439 |
| Slice consumer firing | PASS | indexSlice log 大量 |
| **UI "数据维护" 入口可见** | **FAIL** | Task 11 接入的页面在 routes.tsx 未挂载,UI 不可达 |
| Milvus 下游 reindex | FAIL (env) | 220 dev vector store schema 问题,与代码无关 |

## 已知 issue / 跟代码无关的环境问题

1. **220 Milvus vector store 不健康** (Task 8 已记录): `code=105000002 非可重试`
   对该 space 4 个 KB 的 vector channel 都报错。文本通道 (`kb_entries` ES index)
   work,只是向量索引补不上。跟 Task 1-7 代码无关。

2. **`coze_resource` 始终为 0**: Task 8 文档第 99-122 行已分析:
   该 space `app_draft` 表 0 行,resync 语义只从 apps 反写 coze_resource,
   插件/工作流/知识/提示词的 coze_resource 文档不会被重新 emit。这是设计约定,
   不是 bug。如果要让 plugin/workflow/prompt/knowledge 也回填到 coze_resource,
   需要单独 follow-up task。

## 主要 concern (跟 Task 1-11 代码相关)

**Task 11 接入位置错误,导致真实用户在 UI 看不到 "数据维护" 入口**:

`frontend/apps/coze-studio/src/pages/space-management/index.tsx` (Task 11 改的)
不在 `routes.tsx` 路由表里,也不在 SpaceLayout 的 sub-menu (`SpaceSubModuleEnum`) 里。
导航上去不到这个页面。

**两条可选 follow-up 方案**:

### A. 改成 sub-route 模式 (推荐,跟其他管理功能一致)
1. `SpaceSubModuleEnum` 加 `DATA_MAINTENANCE = 'data-maintenance'`
2. `routes.tsx` 在 `path: ':space_id'` children 里加:
   ```tsx
   {
     path: 'data-maintenance',
     Component: DataMaintenancePage,  // 新建,引用 DataMaintenanceSection
     loader: () => ({ subMenuKey: SpaceSubModuleEnum.DATA_MAINTENANCE }),
   }
   ```
3. SpaceLayout sub-menu 配置加新项 (跟 EMBEDDING/RERANK 平行)
4. `DataMaintenanceSection` 直接用 `useParams().space_id`,不再需要 selector

### B. 保留 Task 11 现有 `SpaceManagementPage`,补 routes 注册
适用于产品想要"列出所有空间 + 任选一个做维护"的管理员视图。

需要管理员判断: 产品是想要 **per-space sub-route** (推荐 A) 还是 **跨空间的管理员页**
(B)。Task 12 scope 是 "不改代码,只做 build + deploy + verify",所以 wiring fix
留给 Task 13。

## 结论

**部分通过 (DONE_WITH_CONCERNS)**:
- 后端代码 (Tasks 1-8) 完整 e2e 跑通: HTTP API + ES delete + bulk rewrite + RMQ
  slice dispatch 全 work
- 前端 SDK + Component (Tasks 9-10) 编译产物正确,部署到容器,bundle 含 component
- 前端 wiring (Task 11) 实现是孤立页面,UI navigation 不可达,需要 follow-up 调整接入点

代码正确性 + API 契约 + 后端 e2e 链路 — 100% PASS。
真实用户从 UI 点按钮的路径 — 0% (因为没入口),但已经验证 fetch /api/space/resync_es
从浏览器 (已登录) 走整套链路完整 work,SDK 调用相同 endpoint。

唯一阻塞用户体验 e2e 的是 Task 11 接入位置问题,需要 follow-up task 修。
