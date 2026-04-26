# WebSDK 端到端验证与已知 Bug 修复 — 设计文档

**Date:** 2026-04-27
**Status:** Approved (awaiting plan & implementation)
**Topic:** websdk-e2e-validation

---

## 背景

最近未提交的代码改动构建了一条完整的 WebSDK iframe 嵌入链路：

| 文件 | 状态 | 改动 |
|------|------|------|
| [`backend/Dockerfile`](backend/Dockerfile:62) | M | 新增 `COPY backend/static/static/sdk` |
| [`backend/static/static/sdk/ynet-web-sdk.js`](backend/static/static/sdk/ynet-web-sdk.js) | 已存在 | 7.7KB SDK loader（4-15 创建） |
| [`frontend/packages/studio/open-platform/open-env-adapter/src/chat/index.ts`](frontend/packages/studio/open-platform/open-env-adapter/src/chat/index.ts) | M | SDK URL 改为 `location.origin` |
| [`frontend/packages/studio/workspace/project-publish/src/web-sdk-guide/index.tsx`](frontend/packages/studio/workspace/project-publish/src/web-sdk-guide/index.tsx) | M | 用户文档示例代码 |
| [`frontend/apps/coze-studio/src/routes.tsx`](frontend/apps/coze-studio/src/routes.tsx:175) | M | 新增 `/agent-chat` 路由 |
| [`frontend/apps/coze-studio/src/pages/agent-chat.tsx`](frontend/apps/coze-studio/src/pages/agent-chat.tsx) | ?? | 146 行新文件，iframe 内嵌页 |

**问题**：
1. **从未启动服务跑过任何一次** — 链路是否真的通不知道
2. **Dockerfile 改动未做 docker build 验证**
3. **代码审计已发现至少两个 bug**：
   - `parentOrigin = '*'` — postMessage 来源校验缺失（[agent-chat.tsx:41](frontend/apps/coze-studio/src/pages/agent-chat.tsx:41)）
   - `iframeAppHost = location.origin` 在嵌入到第三方页面时拿到的是第三方 origin，不是 studio 自己的（[chat/index.ts:21-24](frontend/packages/studio/open-platform/open-env-adapter/src/chat/index.ts:21)）

---

## 目标

1. **真的把 WebSDK 嵌入链路跑通一次**（端到端、跨域、token 注入、token 刷新）
2. **修两个已发现的 bug**
3. **留下可重复的本地测试方案**（demo HTML + README），以后改动 SDK 能快速回归

---

## 范围与约束

### 必做
- Docker build 验证（产物里 SDK 文件可访问）
- 单端 `/agent-chat?bot_id=...` 渲染验证
- 跨域 iframe 嵌入 demo（独立 origin）
- postMessage 双向通信（INIT 注入 token、UPDATE_TOKEN 刷新）
- 修 `parentOrigin = '*'`、`iframeAppHost = location.origin`

### 不做（Out of Scope）
- 不改 `BuilderChat` 组件实现
- 不引入自动化 E2E 框架（Playwright/Cypress）— 嵌入测试手动可控成本最低
- 不改后端 chat API

### SDK Loader 修改

`backend/static/static/sdk/ynet-web-sdk.js` 是手写 JS 文件（非压缩、可读），本设计需要修改它来传 `parent_origin` 参数到 iframe。修改点：[第 101-107 行](backend/static/static/sdk/ynet-web-sdk.js:101) 构建 `iframeUrl` 时追加 `&parent_origin=` + `encodeURIComponent(window.location.origin)`。

---

## 架构

### 完整链路

```
┌─────────────────────────────────────────────────────────┐
│  第三方页面 (e.g., http://localhost:9000/demo.html)     │
│  ┌────────────────────────────────────────────────────┐ │
│  │ <script src="http://studio/static/sdk/ynet.js"/>   │ │
│  │ new CozeWebSDK.WebChatClient({                     │ │
│  │   config: { bot_id, workflow_id },                 │ │
│  │   auth:   { type, token, onRefreshToken },         │ │
│  │   el: '#chat-container',                           │ │
│  │ })                                                  │ │
│  └────────────┬───────────────────────────────────────┘ │
│               │ 创建 iframe                              │
│               │ src=http://studio/agent-chat?bot_id=X   │
└───────────────┼────────────────────────────────────────-┘
                │
                ▼
┌─────────────────────────────────────────────────────────┐
│  iframe (http://studio/agent-chat)                      │
│  ┌────────────────────────────────────────────────────┐ │
│  │ AgentChatPage:                                     │ │
│  │   读 URL params (bot_id/workflow_id/title)         │ │
│  │   监听 postMessage(INIT, UPDATE_TOKEN)             │ │
│  │   渲染 <BuilderChat config={...} />                │ │
│  └────────────┬───────────────────────────────────────┘ │
└───────────────┼─────────────────────────────────────────┘
                │ HTTP/WS API
                ▼
        Studio Backend (chat / token / ...)
```

### postMessage 协议（已存在，本设计明确化）

**父→子 iframe**：
```json
{ "type": "INIT", "payload": { "token": "pat_xxx" } }
{ "type": "UPDATE_TOKEN", "payload": { "token": "pat_yyy" } }
```

**子→父 iframe**（已有，[agent-chat.tsx:65,98](frontend/apps/coze-studio/src/pages/agent-chat.tsx)）：
```json
{ "source": "ynet-sdk-iframe", "type": "READY", "payload": {} }
{ "source": "ynet-sdk-iframe", "type": "TOKEN_EXPIRED", "payload": {} }
```

**注意**：当前 `READY` 消息以 `'*'` 作为 target origin（[line 67](frontend/apps/coze-studio/src/pages/agent-chat.tsx:67)），这会向任意嵌入者暴露 iframe 状态。本设计要求改为只在已知 `parent_origin` 后才发 `READY`，或用初始空字符串等待父端先发 `INIT` 后再回复。

---

## Bug 修复

### Bug 1: `parentOrigin = '*'` — postMessage 安全漏洞

**位置**：[`agent-chat.tsx:41`](frontend/apps/coze-studio/src/pages/agent-chat.tsx:41)

**当前代码**：
```typescript
const parentOrigin = useRef<string>('*');

useEffect(() => {
  const handleMessage = (event: MessageEvent) => {
    const { data } = event;
    if (!data || !data.type || data.source === MESSAGE_SOURCE_IFRAME) return;
    if (event.origin) parentOrigin.current = event.origin;
    // ... 直接用 data.payload.token
  };
}, []);
```

**问题**：
- 没有 `event.origin` 白名单校验，任何来源都能注入 token
- 攻击者可以用一个嵌入了我们 iframe 的恶意页面，注入伪造 token

**修复**：
1. URL params 增加 `parent_origin`（由父 SDK 传入），作为 allowlist
2. `handleMessage` 进入时先校验 `event.origin === expectedParentOrigin`
3. 不匹配时记录 warning 并直接 return
4. 子→父 `READY` 消息改用 `parent_origin` 作为 target（不再 `'*'`，[line 67](frontend/apps/coze-studio/src/pages/agent-chat.tsx:67)）
5. 子→父 `TOKEN_EXPIRED` 消息已用 `parentOrigin.current`，但需在 `parent_origin` 缺失时不发，避免泄露
6. 父 SDK 端调用 `postMessage(data, parentOrigin)` 也明确目标 origin

```typescript
const expectedParentOrigin = searchParams.get('parent_origin') || '';

const handleMessage = (event: MessageEvent) => {
  if (expectedParentOrigin && event.origin !== expectedParentOrigin) {
    console.warn('[agent-chat] reject message from untrusted origin:', event.origin);
    return;
  }
  // ... 后续处理
};

// 仅在已知 parent_origin 时发 READY
if (window.parent !== window && expectedParentOrigin) {
  window.parent.postMessage(
    { source: MESSAGE_SOURCE_IFRAME, type: 'READY', payload: {} },
    expectedParentOrigin,
  );
}
```

### Bug 2: `iframeAppHost = location.origin` 语义错误

**位置**：[`open-env-adapter/chat/index.ts:21-24`](frontend/packages/studio/open-platform/open-env-adapter/src/chat/index.ts:21)

**当前代码**：
```typescript
export const iframeAppHost =
  typeof location !== 'undefined' ? location.origin : '';

export const cozeOfficialHost =
  typeof location !== 'undefined' ? location.origin : '';
```

**问题**：
- 这两个变量在 SDK 打包后会嵌入 SDK 文件，被加载到第三方页面运行
- 第三方页面运行时 `location.origin` 是**第三方的 origin**，不是 studio 的
- iframe 应该指向 studio，但拿到的是第三方域名 → iframe 加载失败

**修复方案**：
1. 在打包时通过环境变量注入正确的 host：
   ```typescript
   declare const STUDIO_HOST: string; // 由 build 注入

   export const iframeAppHost = STUDIO_HOST || (typeof location !== 'undefined' ? location.origin : '');
   ```
2. `rsbuild.config.ts` 加 `define: { STUDIO_HOST: JSON.stringify(process.env.STUDIO_HOST) }`
3. 部署时通过环境变量 `STUDIO_HOST=https://studio.example.com` 配置
4. 本地开发回退到 `location.origin`（fallback 路径保留）

---

## 测试方案（4 步）

### Step 1: Docker Build 验证

**目标**：确认 Dockerfile 改动生效，SDK 文件能被静态服务器返回。

```bash
cd /Users/luzhipeng/projects/ynet/coze-studio
docker build -t ynet-studio-test:websdk -f backend/Dockerfile .
docker run --rm -d --name ynet-test -p 18080:8080 ynet-studio-test:websdk
sleep 5
curl -I http://localhost:18080/static/sdk/ynet-web-sdk.js
# 期望 HTTP 200, Content-Type: application/javascript
docker stop ynet-test
```

**断言**：
- ✅ docker build 成功（如果失败，说明 COPY 路径错或 SDK 文件不存在）
- ✅ SDK 文件 HTTP 200 响应

### Step 2: 单端访问 `/agent-chat`

**目标**：iframe 内嵌页能渲染。

**前置**：起本地后端 + 前端
```bash
# 后端（dev mode）
cd backend && go run cmd/server/main.go &

# 前端
cd frontend && rush start
```

**操作**：
1. 浏览器访问 `http://localhost:8080/agent-chat?bot_id=<test_bot>&token=<pat>&title=Test`
2. 看到聊天界面渲染
3. 发一条消息 "hello"
4. 收到回复

**断言**：
- ✅ 页面渲染无 console error
- ✅ 能发消息且收到回复
- ✅ Network tab 显示正确的 API 请求

### Step 3: 跨域 iframe 嵌入

**目标**：模拟真实第三方嵌入场景，验证完整链路。

**新建测试 demo**：

`e2e/websdk/demo.html`：
```html
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>YNet WebSDK Demo</title>
</head>
<body>
  <h1>WebSDK Embed Test</h1>
  <p>Origin: <code id="origin"></code></p>
  <div id="chat-container" style="width: 400px; height: 600px; border: 1px solid #ccc;"></div>

  <script>document.getElementById('origin').textContent = location.origin;</script>
  <script src="http://localhost:8080/static/sdk/ynet-web-sdk.js"></script>
  <script>
    const TEST_BOT_ID = '<填写测试 bot id>';
    const TEST_TOKEN = '<填写测试 PAT>';

    new CozeWebSDK.WebChatClient({
      config: { bot_id: TEST_BOT_ID },
      auth: {
        type: 'token',
        token: TEST_TOKEN,
        onRefreshToken: function () {
          console.log('[demo] onRefreshToken called');
          return TEST_TOKEN;
        }
      },
      el: '#chat-container',
      onReady: function () {
        console.log('[demo] chat widget ready');
      }
    });
  </script>
</body>
</html>
```

`e2e/websdk/README.md`（操作指南）：
```markdown
# WebSDK 本地嵌入测试

## 前置
1. 后端启动在 :8080，前端构建产物已 serve
2. SDK 文件 http://localhost:8080/static/sdk/ynet-web-sdk.js 可访问
3. 准备一个测试 bot 和对应 PAT

## 操作

### 起独立 origin 服务
```bash
cd e2e/websdk
sed -i 's/<填写测试 bot id>/your_bot_id/' demo.html
sed -i 's/<填写测试 PAT>/pat_xxx/' demo.html
python3 -m http.server 9000
```

### 浏览器访问
打开 http://localhost:9000/demo.html

### 验证清单
- [ ] iframe 加载成功（DevTools → Elements 看到 iframe）
- [ ] iframe src 是 http://localhost:8080/agent-chat?...
- [ ] DevTools → Console 看到 "[demo] chat widget ready"
- [ ] 父页面 origin = http://localhost:9000，iframe origin = http://localhost:8080
- [ ] 点击 Network 看到 cross-origin postMessage 流量
- [ ] 发送消息成功
- [ ] 修改 demo 触发 token 刷新（手动模拟），日志看到 onRefreshToken
```

**断言**：见 README 验证清单

### Step 4: postMessage 安全验证

**目标**：验证 Bug 1 修复后的行为。

**操作**：
1. 起一个**没有正确 parent_origin** 的恶意页面（`http://localhost:9001`）
2. 嵌入 SDK 用 `postMessage` 发 `INIT` 消息
3. 期望 console 看到 "reject message from untrusted origin: http://localhost:9001"

**额外**：`http://localhost:9000` 的合法 demo 仍能正常通信。

---

## 文件结构

### 新建
```
e2e/websdk/
├── README.md          # 测试操作指南
├── demo.html          # 跨域嵌入测试 demo
└── attack.html        # 恶意 origin 模拟（验证 Bug 1 修复）
```

### 修改
```
frontend/apps/coze-studio/src/pages/agent-chat.tsx               # 修 Bug 1（postMessage 安全）
frontend/packages/studio/open-platform/open-env-adapter/src/chat/index.ts  # 修 Bug 2（host 注入）
frontend/apps/coze-studio/rsbuild.config.ts (or similar)         # 注入 STUDIO_HOST
backend/static/static/sdk/ynet-web-sdk.js                        # 追加 parent_origin URL 参数
```

### 不改但需验证
```
backend/Dockerfile
frontend/apps/coze-studio/src/routes.tsx
frontend/packages/studio/workspace/project-publish/src/web-sdk-guide/index.tsx
```

---

## 测试矩阵

| 测试场景 | 预期结果 |
|----------|----------|
| docker build | 成功 |
| `/static/sdk/ynet-web-sdk.js` HTTP GET | 200 + 正确 MIME |
| `/agent-chat?bot_id=...&token=...` 直接访问 | 渲染、能发消息 |
| 跨域 demo (port 9000) 嵌入 | iframe 加载、postMessage INIT 注入 token、对话成功 |
| 恶意 origin (port 9001) 嵌入 | postMessage 被拒绝（console warning） |
| 触发 onRefreshToken | 父子 iframe 完成 UPDATE_TOKEN 流程，新 token 生效 |

---

## 风险与缓解

| 风险 | 缓解 |
|------|------|
| 没有 testbot 和 PAT 无法验证 | 在 dev 环境创建测试 bot，PAT 写在 README 但不入 git（ignore demo.html.local）|
| docker build 在 Mac M 系列芯片可能架构不匹配 | `docker build --platform linux/amd64`，README 注明 |
| BuilderChat 组件依赖很多 store/context，agent-chat.tsx 可能初始化失败 | 测试中观察 console error，修复 missing provider |
| 修了 `iframeAppHost` 后既有部署可能受影响 | 修改保留 fallback 到 `location.origin`，不影响默认部署 |

---

## 验收标准

1. ✅ `docker build` 成功，SDK 文件 HTTP 200
2. ✅ 单端 `/agent-chat?bot_id=...` 能聊天
3. ✅ 跨域 demo 嵌入聊天能跑通完整流程
4. ✅ 恶意 origin 被拒绝，正常 origin 不受影响
5. ✅ token 刷新流程验证通过
6. ✅ `e2e/websdk/` 目录留下 README + demo.html，可重复跑
7. ✅ 全部 typecheck 通过
8. ✅ 截图/录屏作为 PR 附件
