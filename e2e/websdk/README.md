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

(`demo.html.local` 已 gitignore，不会入版本库)

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
# 另一个 terminal，起恶意 origin 服务
cd e2e/websdk
python3 -m http.server 9001
```

访问 `http://localhost:9001/attack.html`：
- [ ] Console 看到 `[agent-chat] reject message from untrusted origin: http://localhost:9001`
- [ ] iframe 没有进入对话状态（attack.html 的 token 注入被拒）

## Step 5: Token 刷新流程

回到 demo.html.local 的页面：
- [ ] 修改 demo 触发 token 失效（手动改 TEST_TOKEN）
- [ ] Console 看到 `[demo] onRefreshToken called`
- [ ] 父页面收到子 iframe 发来的 `TOKEN_EXPIRED` 消息

## 截图要求

PR 附件应包含：
- `01-docker-build.png` (curl SDK 文件 200 响应)
- `02-demo-loaded.png` (demo 页面 + iframe + chat widget ready)
- `03-message-rejected.png` (attack.html 触发拒绝日志)
- `04-token-refresh.png` (TOKEN_EXPIRED 消息 console 日志)
