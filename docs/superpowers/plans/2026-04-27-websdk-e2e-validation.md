# WebSDK E2E Validation & Bug Fixes — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 WebSDK iframe 嵌入链路真正跑通，修两个 postMessage / iframeAppHost 安全 bug，留下可重复的本地验证 demo。

**Architecture:** 修复 SDK loader 传 `parent_origin` URL 参数 → agent-chat.tsx 严格校验 `event.origin` 并仅向白名单父 origin 发消息 → env-adapter 通过构建期 `STUDIO_HOST` 注入 host → docker build 验证 SDK 静态文件可访问 → 创建独立 origin 的本地 demo 跑通完整嵌入流程。

**Tech Stack:** TypeScript, React, Hertz (Go), Vanilla JS（SDK loader），Rsbuild

**Spec:** [docs/superpowers/specs/2026-04-27-websdk-e2e-validation-design.md](docs/superpowers/specs/2026-04-27-websdk-e2e-validation-design.md)

---

## Task 1: 修复 SDK Loader 传 parent_origin

**Files:**
- Modify: `backend/static/static/sdk/ynet-web-sdk.js`

- [ ] **Step 1: 阅读 SDK loader 当前代码（line 99-110）**

Run: `cat backend/static/static/sdk/ynet-web-sdk.js`
确认 `iframeUrl` 构建逻辑。

- [ ] **Step 2: 在 iframeUrl 中追加 parent_origin**

打开 `backend/static/static/sdk/ynet-web-sdk.js`，定位到第 101-110 行：

```javascript
var iframeUrl =
  baseUrl +
  '/agent-chat?bot_id=' +
  encodeURIComponent(botId) +
  '&token=' +
  encodeURIComponent(token) +
  '&mode=websdk';

if (config.workflowId) {
  iframeUrl += '&workflow_id=' + encodeURIComponent(config.workflowId);
}
```

改为：

```javascript
var iframeUrl =
  baseUrl +
  '/agent-chat?bot_id=' +
  encodeURIComponent(botId) +
  '&token=' +
  encodeURIComponent(token) +
  '&mode=websdk' +
  '&parent_origin=' + encodeURIComponent(window.location.origin);

if (config.workflowId) {
  iframeUrl += '&workflow_id=' + encodeURIComponent(config.workflowId);
}
```

- [ ] **Step 3: 验证 SDK 仍是合法 JS**

Run: `node -c backend/static/static/sdk/ynet-web-sdk.js`
Expected: 无报错（语法检查通过）

- [ ] **Step 4: 提交**

```bash
git add backend/static/static/sdk/ynet-web-sdk.js
git commit -m "fix(sdk): pass parent_origin to iframe for postMessage validation"
```

---

## Task 2: 修复 agent-chat.tsx postMessage 安全

**Files:**
- Modify: `frontend/apps/coze-studio/src/pages/agent-chat.tsx`

- [ ] **Step 1: 阅读现有代码**

Run: `cat frontend/apps/coze-studio/src/pages/agent-chat.tsx`

- [ ] **Step 2: 修改：origin 白名单 + 安全的 postMessage**

替换 useEffect 的 message handler 区段（line 41-72 附近）：

```tsx
const expectedParentOrigin = searchParams.get('parent_origin') || '';
const parentOrigin = useRef<string>(expectedParentOrigin);

useEffect(() => {
  const handleMessage = (event: MessageEvent) => {
    if (expectedParentOrigin && event.origin !== expectedParentOrigin) {
      console.warn(
        '[agent-chat] reject message from untrusted origin:',
        event.origin,
        'expected:',
        expectedParentOrigin,
      );
      return;
    }
    const { data } = event;
    if (!data || !data.type || data.source === MESSAGE_SOURCE_IFRAME) return;

    parentOrigin.current = event.origin;

    switch (data.type) {
      case 'INIT':
        if (data.payload?.token) setToken(data.payload.token);
        break;
      case 'UPDATE_TOKEN':
        if (data.payload?.token) setToken(data.payload.token);
        break;
    }
  };

  window.addEventListener('message', handleMessage);

  // 仅在已知 parent_origin 时通知 ready，避免向未知方暴露 iframe
  if (window.parent !== window && expectedParentOrigin) {
    window.parent.postMessage(
      { source: MESSAGE_SOURCE_IFRAME, type: 'READY', payload: {} },
      expectedParentOrigin,
    );
  }

  return () => window.removeEventListener('message', handleMessage);
}, [expectedParentOrigin]);
```

并修改 `auth.refreshToken`（line 95-107 附近），只在 parent_origin 已知时发：

```tsx
const auth = useMemo(
  () => ({
    type: 'external' as const,
    token,
    refreshToken: () => {
      if (window.parent !== window && expectedParentOrigin) {
        window.parent.postMessage(
          {
            source: MESSAGE_SOURCE_IFRAME,
            type: 'TOKEN_EXPIRED',
            payload: {},
          },
          expectedParentOrigin,
        );
      }
      return Promise.resolve('');
    },
  }),
  [token, expectedParentOrigin],
);
```

- [ ] **Step 3: typecheck**

Run: `cd /Users/luzhipeng/projects/ynet/coze-studio/frontend && rush check && rush build --to-version-policy frontend-apps`
Expected: 无类型错误（如果 rush 命令不一致，按项目 README 改为对应 typecheck 命令）

- [ ] **Step 4: 提交**

```bash
git add frontend/apps/coze-studio/src/pages/agent-chat.tsx
git commit -m "fix(websdk): validate postMessage origin against parent_origin allowlist"
```

---

## Task 3: 修复 env-adapter 注入 STUDIO_HOST

**Files:**
- Modify: `frontend/packages/studio/open-platform/open-env-adapter/src/chat/index.ts`
- Modify: `frontend/apps/coze-studio/rsbuild.config.ts` 或对应构建配置

- [ ] **Step 1: 修改 chat/index.ts**

打开文件，定位 line 21-24：
```typescript
export const iframeAppHost =
  typeof location !== 'undefined' ? location.origin : '';

export const cozeOfficialHost =
  typeof location !== 'undefined' ? location.origin : '';
```

改为：
```typescript
declare const STUDIO_HOST: string | undefined;

const studioHost: string = (() => {
  // build 期注入优先；回退到 location.origin（dev 模式）
  if (typeof STUDIO_HOST !== 'undefined' && STUDIO_HOST) return STUDIO_HOST;
  if (typeof location !== 'undefined') return location.origin;
  return '';
})();

export const iframeAppHost = studioHost;
export const cozeOfficialHost = studioHost;
```

- [ ] **Step 2: 在 rsbuild config 注入**

定位 `frontend/apps/coze-studio/rsbuild.config.ts`：

```bash
grep -n "DefinePlugin\|define\|globals" frontend/apps/coze-studio/rsbuild.config.ts
```

如果有 `tools.rspack.plugins` 或类似 `source.define`，加：
```typescript
source: {
  define: {
    STUDIO_HOST: JSON.stringify(process.env.STUDIO_HOST || ''),
    // ... 其他 define
  },
},
```

如果没有 source.define，参考已有 `process.env.IS_OVERSEA` 等 define 的写法。

- [ ] **Step 3: typecheck**

Run: 同 Task 2 的 typecheck 命令
Expected: 通过

- [ ] **Step 4: 提交**

```bash
git add frontend/packages/studio/open-platform/open-env-adapter/src/chat/index.ts frontend/apps/coze-studio/rsbuild.config.ts
git commit -m "fix(websdk): inject STUDIO_HOST at build time for iframe host resolution"
```

---

## Task 4: Docker Build 验证

**Files:** 无（仅验证）

- [ ] **Step 1: Docker build**

```bash
cd /Users/luzhipeng/projects/ynet/coze-studio
docker build -t ynet-studio-test:websdk -f backend/Dockerfile .
```
Expected: 成功，无 COPY 错误

如果失败：
- 报错 "no such file or directory" → 检查 `backend/static/static/sdk/ynet-web-sdk.js` 存在
- 报错平台不兼容（Mac M 系列） → 加 `--platform linux/amd64`

- [ ] **Step 2: 启动容器**

```bash
docker run --rm -d --name ynet-test -p 18080:8080 ynet-studio-test:websdk
sleep 5
```

- [ ] **Step 3: 验证 SDK 文件可访问**

```bash
curl -i http://localhost:18080/static/sdk/ynet-web-sdk.js | head -10
```
Expected:
- HTTP/1.1 200
- Content-Type: application/javascript 或 text/javascript

- [ ] **Step 4: 验证 SDK 内容里有 parent_origin**

```bash
curl -s http://localhost:18080/static/sdk/ynet-web-sdk.js | grep -c "parent_origin"
```
Expected: ≥ 1（确认 Task 1 的修改生效）

- [ ] **Step 5: 停止容器**

```bash
docker stop ynet-test
```

- [ ] **Step 6: 截图保存**

将 curl 的成功输出截图，保存到 `e2e/websdk/screenshots/01-docker-build.png`（PR 附件用）。

无需提交（仅验证）。

---

## Task 5: 创建本地测试 Demo

**Files:**
- Create: `e2e/websdk/demo.html`
- Create: `e2e/websdk/attack.html`
- Create: `e2e/websdk/README.md`
- Create: `e2e/websdk/.gitignore`

- [ ] **Step 1: 创建 demo.html**

写入 `e2e/websdk/demo.html`:
```html
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>YNet WebSDK Embed Demo</title>
</head>
<body>
  <h1>WebSDK Embed Test</h1>
  <p>Page origin: <code id="page-origin"></code></p>
  <p>Studio iframe origin: <code id="iframe-origin"></code></p>
  <div id="chat-container" style="width: 400px; height: 600px; border: 1px solid #ccc;"></div>

  <script>
    document.getElementById('page-origin').textContent = location.origin;
    document.getElementById('iframe-origin').textContent = 'http://localhost:8080';
  </script>
  <script src="http://localhost:8080/static/sdk/ynet-web-sdk.js"></script>
  <script>
    // 替换为真实测试 bot 和 PAT
    const TEST_BOT_ID = window.TEST_BOT_ID || '<填写测试 bot id>';
    const TEST_TOKEN  = window.TEST_TOKEN  || '<填写测试 PAT>';

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

- [ ] **Step 2: 创建 attack.html（验证 origin 校验拒绝恶意来源）**

写入 `e2e/websdk/attack.html`:
```html
<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><title>Attack Origin</title></head>
<body>
  <h1>Hostile Origin Test</h1>
  <p>This page tries to embed agent-chat without proper parent_origin.</p>
  <iframe id="frame" src="http://localhost:8080/agent-chat?bot_id=ATTACK&token=fake&mode=websdk" style="width:400px;height:600px"></iframe>
  <button onclick="injectFakeToken()">Inject Fake Token</button>
  <pre id="log"></pre>
  <script>
    const log = (msg) => {
      document.getElementById('log').textContent += msg + '\n';
    };
    function injectFakeToken() {
      const frame = document.getElementById('frame');
      frame.contentWindow.postMessage({ type: 'INIT', payload: { token: 'evil_token_xxx' } }, '*');
      log('[attack] sent fake INIT message');
    }
    window.addEventListener('message', (e) => {
      log('[attack] received from ' + e.origin + ': ' + JSON.stringify(e.data));
    });
  </script>
</body>
</html>
```

- [ ] **Step 3: 创建 README.md**

写入 `e2e/websdk/README.md`:
```markdown
# WebSDK 本地嵌入测试

## 前置条件

1. 后端运行在 `http://localhost:8080`
2. SDK 文件 `http://localhost:8080/static/sdk/ynet-web-sdk.js` 可访问
3. 准备一个测试 bot ID 和对应 PAT (Personal Access Token)

## Step 1: 配置 demo

```bash
cd e2e/websdk
cp demo.html demo.html.local
sed -i.bak 's|<填写测试 bot id>|YOUR_BOT_ID|' demo.html.local
sed -i.bak 's|<填写测试 PAT>|pat_xxx|' demo.html.local
```

(demo.html.local 已 gitignore，不会入版本库)

## Step 2: 起独立 origin 服务

```bash
python3 -m http.server 9000
```

## Step 3: 浏览器验证

打开 `http://localhost:9000/demo.html.local`：

- [ ] iframe 加载成功（DevTools → Elements 看到 `<iframe>`）
- [ ] iframe src 是 `http://localhost:8080/agent-chat?bot_id=...&parent_origin=http%3A%2F%2Flocalhost%3A9000`
- [ ] DevTools Console 看到 `[demo] chat widget ready`
- [ ] 父页面 origin = `http://localhost:9000`，iframe origin = `http://localhost:8080`
- [ ] 发送消息成功收到回复
- [ ] Network tab 看到 cross-origin postMessage 流量

## Step 4: 验证 origin 校验（恶意来源被拒绝）

```bash
# 另一个 terminal，起恶意 origin
python3 -m http.server 9001  # cwd 也是 e2e/websdk
```

访问 `http://localhost:9001/attack.html`：
- [ ] Console 看到 "reject message from untrusted origin: http://localhost:9001"
- [ ] iframe 没有进入对话状态（attack.html 的 token 注入被拒）

## Step 5: Token 刷新流程

回到 demo.html.local 的页面：
- [ ] 修改 demo 触发 token 失效（手动改 TEST_TOKEN）
- [ ] Console 看到 `[demo] onRefreshToken called`
- [ ] 父页面收到子 iframe 发来的 `TOKEN_EXPIRED` 消息

## 截图要求

PR 附件应包含：
- 01-docker-build.png（curl SDK 文件 200 响应）
- 02-demo-loaded.png（demo 页面 + iframe + chat widget ready）
- 03-message-rejected.png（attack.html 触发拒绝日志）
- 04-token-refresh.png（TOKEN_EXPIRED 消息 console 日志）
```

- [ ] **Step 4: 创建 .gitignore**

写入 `e2e/websdk/.gitignore`:
```
demo.html.local
demo.html.local.bak
screenshots/
```

- [ ] **Step 5: 提交**

```bash
git add e2e/websdk/
git commit -m "docs(websdk): add local embed test demo and README"
```

---

## Task 6: 端到端手动验证（必做）

**Files:** 无（验证 + 截图）

- [ ] **Step 1: 启动后端**

```bash
cd /Users/luzhipeng/projects/ynet/coze-studio
# 按项目 dev 启动方式
docker compose -f docker/docker-compose.dev.yml up -d  # 或对应启动方式
```

确认 `http://localhost:8080` 可访问。

- [ ] **Step 2: 启动前端 dev server**

```bash
cd frontend
rush install --to coze-studio
rush start  # 或对应命令
```

如果前端要构建后服务，build 后用 nginx 或后端 static 服务。

- [ ] **Step 3: 单端访问 /agent-chat**

浏览器打开 `http://localhost:8080/agent-chat?bot_id=YOUR_BOT&token=YOUR_PAT&mode=websdk`：

- [ ] 页面渲染聊天框
- [ ] DevTools Console 无 error
- [ ] 能发消息收到回复
- [ ] 截图保存到 `e2e/websdk/screenshots/02-direct-access.png`

- [ ] **Step 4: 跨域 demo 嵌入**

在另一终端：
```bash
cd e2e/websdk
# 创建 demo.html.local（按 README 操作）
python3 -m http.server 9000
```

浏览器打开 `http://localhost:9000/demo.html.local`：

- [ ] iframe 加载（看 DevTools Network）
- [ ] iframe URL 含 `parent_origin=http%3A%2F%2Flocalhost%3A9000`
- [ ] Console 看到 `[demo] chat widget ready`
- [ ] 能发消息收到回复
- [ ] 截图保存到 `e2e/websdk/screenshots/03-cross-origin.png`

- [ ] **Step 5: attack.html 验证拒绝**

```bash
# 9001 用 attack.html
python3 -m http.server 9001
```

浏览器访问 `http://localhost:9001/attack.html`，点 "Inject Fake Token" 按钮：

- [ ] Console 输出 `[agent-chat] reject message from untrusted origin: http://localhost:9001`
- [ ] iframe 内容未变成 evil_token_xxx 状态
- [ ] 截图保存到 `e2e/websdk/screenshots/04-attack-rejected.png`

- [ ] **Step 6: 整理截图**

```bash
ls -la e2e/websdk/screenshots/
```

(screenshots/ 目录已 gitignore，仅作 PR 附件)

如本地缺少 testbot 或 PAT，本步骤需先在 dev 环境创建测试 bot。如果完全无法跑通，记录错误现象到 PR 描述并求助。

---

## Task 7: 自检报告 + 验收

- [ ] **Step 1: typecheck + 编译验证**

```bash
cd /Users/luzhipeng/projects/ynet/coze-studio
go build ./backend/...
cd frontend && rush check
```
Expected: 全部通过

- [ ] **Step 2: 检查 git 状态**

```bash
git status
git log --oneline -10
```
Expected: 工作区干净，能看到本次所有 commits

- [ ] **Step 3: 总结清单写入 PR 描述**

PR 描述应包含：
- ✅ Docker build 成功 + SDK 200 响应（截图 01）
- ✅ 单端 /agent-chat 渲染成功（截图 02）
- ✅ 跨域嵌入完整流程（截图 03）
- ✅ 恶意 origin 被拒绝（截图 04）
- ✅ Token 刷新流程已验证

---

## 验收清单

- [ ] `docker build` 成功，SDK 文件 HTTP 200，URL 含 `parent_origin`
- [ ] 单端访问 `/agent-chat` 能聊天
- [ ] 跨域 demo 嵌入完整流程通跑
- [ ] 恶意 origin 被 console.warn 拒绝且不影响 token
- [ ] Token 刷新（onRefreshToken 触发，TOKEN_EXPIRED 投递）
- [ ] `e2e/websdk/` 目录留下 README + demo + attack，可重复使用
- [ ] typecheck 通过
- [ ] 截图齐全，挂入 PR 描述
