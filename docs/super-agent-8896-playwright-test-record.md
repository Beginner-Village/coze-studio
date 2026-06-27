# Super Agent 8896 Playwright Test Record

Date: 2026-06-19 Asia/Shanghai

Target:

- Web app: http://10.10.10.226:8896/
- Container: `coze-super`
- Test space: `7652614054615187456`
- Test agent: `7652617174313336832`

Credentials are intentionally not recorded in this file.

## Backend Fixes Verified

- Session-based App Server calls no longer panic when `OpenAPI` auth is absent.
- API-key calls keep published-version semantics through the API connector.
- Bearer App Server calls marked with `super_agent_app_server=true` use draft execution with `CozeConnectorID`, so external App Server access can run the same draft super agent as the UI without requiring the API connector published path.
- App Server trace discovery and replay are exposed through `GET /api/super-agent/manifest` and `POST /api/super-agent/traces/get`.
- Browser session calls use draft execution with `CozeConnectorID`, so sandbox/workspace keys stay aligned with the UI workspace.
- Explicit `WebSDKConnectorID` still overrides to the WebSDK published connector path.

Final deployed backend backup on the remote host:

- `/home/dev/openynet.coze-super.bak-20260618200021`
- `/home/dev/openynet.coze-super.bak-20260618200841` (standard skill enforcement)
- `/home/dev/openynet.coze-super.bak-20260619042255` (manifest harness contract)
- `/home/dev/openynet.coze-super.bak-20260619043018` (public manifest discovery)
- `/home/dev/openynet.coze-super.bak-20260619043737` (initial Bearer App Server draft connector fix)
- `/home/dev/openynet.coze-super.bak-20260619044319` (guarded Bearer App Server draft connector fix)
- `/home/dev/openynet.coze-super.bak-20260619045138` (App Server trace replay API)
- `/home/dev/openynet.coze-super.bak-20260619045922` (App Server artifact discovery/download API)
- `/home/dev/openynet.coze-super.bak-20260619053530` (update_plan completed status compatibility)
- `/home/dev/openynet.coze-super.bak-20260619055155` (runs reply/cancel contract hardening)
- `/home/dev/openynet.coze-super.bak-20260619042006` (sandbox exec writable-workdir enforcement)
- `/home/dev/openynet.coze-super.bak-20260619042838` (sandbox exec work_dir alias compatibility)
- `/home/dev/openynet.coze-super.bak-20260619043506` (workspace patch work_dir alias compatibility)
- `/home/dev/openynet.coze-super.bak-20260619044049` (manifest request schema discovery)
- `/home/dev/openynet.coze-super.bak-20260619044629` (artifact request schema discovery)
- `/home/dev/openynet.coze-super.bak-20260619045210` (artifact request schema required arrays)
- `/home/dev/openynet.coze-super.bak-20260619050510` (skill request schema discovery)

Final deployed frontend static backup on the remote host:

- `/home/dev/coze-super-static.bak-20260618201008`
- `/home/dev/coze-super-static.bak-20260619034029`
- `/home/dev/coze-super-static.bak-20260619050700` (pre-sort artifact UI deployment)
- `/home/dev/coze-super-static.bak-20260619051430` (final artifact UI deployment)
- `/home/dev/coze-super-static.bak-20260619052100` (initial trace console UI deployment)
- `/home/dev/coze-super-static.bak-20260619052430` (final trace console header deployment)
- `/home/dev/coze-super-static.bak-20260619053000` (skill marketplace scope filter deployment)
- `/home/dev/coze-super-static.bak-20260619054404` (trace filename link guard deployment)
- `/home/dev/coze-super-static.bak-20260619060135` (workspace agent_id API deployment)

Earlier rollback points created while debugging:

- `/home/dev/openynet.coze-super.bak-20260618195257`
- `/home/dev/openynet.coze-super.bak-20260618195645`

## Playwright MCP Evidence

## 2026-06-19 15:44 CST: Explore Global Skill Store UI

User-facing change under test:

- `http://10.10.10.226:8896/explore/project/latest` now renders the company-level standard skill store.
- The Explore route keeps the main left navigation highlighted on `商店`, but no longer mounts a second-level sidebar when there is only one skill-store destination.
- Legacy Explore entries are removed from the visible UI; `/explore/plugin` and `/explore/project/tools` client-route back to `/explore/project/latest`.
- The skill store hero uses the generated bitmap asset `skill-marketplace-hero.webp`.
- Skill cards are populated from `GET /api/super-agent/marketplace/list?scope=3&page=1&page_size=200` and only display standard packages with `SKILL.md`.
- `查看详情` opens a modal with standard file paths and a `SKILL.md` preview.

Local verification:

```bash
pnpm --dir frontend/apps/coze-studio exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/community/explore exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/apps/coze-studio build
pnpm --dir frontend/apps/coze-studio test -- --run src/pages/space-skill/__tests__/index.test.tsx
git diff --check
```

Passed.

Static-only deployment:

- Package: `/tmp/coze-studio-static-skill-store-nosubmenu-20260619.tar.gz`
- Local SHA-256: `f9b04d5751be4a536d559c9b38c12477a631366501f66344b030b50c4dc1892a`
- Remote uploaded SHA-256: `f9b04d5751be4a536d559c9b38c12477a631366501f66344b030b50c4dc1892a`
- Remote retained rollback point: `/app/resources/static.bak-skill-store-nosubmenu-202606191544`
- Container: `coze-super` restarted successfully after replacing `/app/resources/static`.
- HTTP smoke check: `curl -I http://10.10.10.226:8896/explore/project/latest` returned `200 OK`.
- Remote hero asset check: `/app/resources/static/static/image/skill-marketplace-hero.f58ac3f2.webp`.

Remote disk hygiene:

- Before cleanup: root filesystem `72G`, used `59G`, available `9.9G`, `86%`.
- Removed stale `/app/resources/static.bak-*` directories from previous static deploys, keeping only `/app/resources/static.bak-skill-store-nosubmenu-202606191544`.
- Removed uploaded tarballs from `/home/dev`.
- After cleanup: root filesystem `72G`, used `56G`, available `13G`, `83%`.
- Remaining static backup size: `311.7M`.

Playwright MCP 8896 verification:

- Snapshot: `super-agent-playwright-mcp-8896-skill-store-nosubmenu-after-deploy.md`
- Page screenshot: `super-agent-playwright-mcp-8896-skill-store-nosubmenu-after-deploy.png`
- Detail modal screenshot: `super-agent-playwright-mcp-8896-skill-store-detail-modal-after-deploy.png`

Observed result:

- Page title: `猎鹰`.
- Current URL `/explore/project/latest` rendered `技能商店` and `全局标准技能`.
- Visible UI did not contain `项目商店`, `插件商店`, or `外部应用`.
- Second-level sidebar was absent; first content heading left edge was `106px`, directly after the `72px` main nav plus page padding.
- Hero image loaded from `/static/image/skill-marketplace-hero.f58ac3f2.webp`, natural size `1800x620`; displayed section size was about `1591x260`.
- Marketplace API returned `200`, `code=0`, `total=2`; visible skill cards count was `2`.
- Search input placeholder `搜索技能名称、描述或标签` was present.
- `/explore/plugin` and `/explore/project/tools` both ended at `/explore/project/latest`.
- First skill detail modal opened successfully and showed `标准文件`, `SKILL.md`, `assets/sample.json`, `references/guide.md`, `scripts/run.py`, `templates/report.md`, and `SKILL.md 预览`.
- Browser console had 0 errors and one existing warning.

Updated-goal availability regression, 2026-06-19 06:31 CST:

- File: `super-agent-playwright-mcp-8896-availability-20260619.json`
- Method: Playwright MCP browser navigation to `http://10.10.10.226:8896/`.
- Result: navigation failed with `navigation-timeout-15000ms`; final URL remained `http://10.10.10.226:8896/` and title remained `about:blank`.
- Cross-check: `curl -I --connect-timeout 5 --max-time 10 http://10.10.10.226:8896/` failed with timeout to port `8896`.
- Cross-check: `ssh -o ConnectTimeout=5 -o BatchMode=yes dev@10.10.10.226 true` failed with timeout to port `22`.
- Rerun: Playwright MCP navigation at `2026-06-19 06:36 CST` again failed with `navigation-timeout-15000ms`; final URL remained `http://10.10.10.226:8896/` and title remained `about:blank`.
- Rerun: Playwright MCP navigation at `2026-06-19 06:41 CST` again failed with `navigation-timeout-15000ms`; final URL remained `http://10.10.10.226:8896/` and title remained `about:blank`.
- Rerun: Playwright MCP navigation at `2026-06-19 06:45 CST` again failed with `navigation-timeout-15000ms`; final URL remained `http://10.10.10.226:8896/` and title remained `about:blank`.
- Rerun: Playwright MCP navigation at `2026-06-19 06:50 CST` again failed with `navigation-timeout-15000ms`; final URL remained `http://10.10.10.226:8896/` and title remained `about:blank`.
- Rerun: Playwright MCP navigation at `2026-06-19 06:54 CST` again failed with `navigation-timeout-15000ms`; final URL remained `http://10.10.10.226:8896/` and title remained `about:blank`.
- Rerun: Playwright MCP navigation at `2026-06-19 06:57 CST` again failed with `navigation-timeout-15000ms`; final URL remained `http://10.10.10.226:8896/` and title remained `about:blank`.
- Rerun: `mcp__playwright.browser_navigate` at `2026-06-19 07:18 CST` failed with a 60000ms timeout while navigating to `http://10.10.10.226:8896/` and waiting for `domcontentloaded`.
- Rerun: `mcp__playwright.browser_navigate` at `2026-06-19 07:24 CST` failed with `net::ERR_CONNECTION_TIMED_OUT` while navigating to `http://10.10.10.226:8896/` and waiting for `domcontentloaded`.
- Rerun: `mcp__playwright.browser_navigate` at `2026-06-19 07:29 CST` failed with `net::ERR_CONNECTION_TIMED_OUT` while navigating to `http://10.10.10.226:8896/` and waiting for `domcontentloaded`.
- Rerun: `mcp__playwright.browser_navigate` at `2026-06-19 07:32 CST` failed with a 60000ms timeout while navigating to `http://10.10.10.226:8896/` and waiting for `domcontentloaded`.
- Rerun: `mcp__playwright.browser_navigate` at `2026-06-19 07:39 CST` failed with `net::ERR_CONNECTION_TIMED_OUT` while navigating to `http://10.10.10.226:8896/` and waiting for `domcontentloaded`.
- Rerun: `mcp__playwright.browser_navigate` at `2026-06-19 07:47 CST` failed with a 60000ms timeout while navigating to `http://10.10.10.226:8896/` and waiting for `domcontentloaded`.
- Rerun: `mcp__playwright.browser_navigate` at `2026-06-19 07:53 CST` failed with `net::ERR_CONNECTION_TIMED_OUT` while navigating to `http://10.10.10.226:8896/` and waiting for `domcontentloaded`.
- Rerun: `mcp__playwright.browser_navigate` at `2026-06-19 07:57 CST` failed with a 60000ms timeout while navigating to `http://10.10.10.226:8896/` and waiting for `domcontentloaded`; after that, `mcp__playwright.browser_tabs list` also timed out after 30000ms.
- Rerun: `mcp__playwright.browser_navigate` at `2026-06-19 07:59 CST` failed with a 60000ms timeout while navigating to `http://10.10.10.226:8896/` and waiting for `domcontentloaded`.
- Rerun: `mcp__playwright.browser_navigate` at `2026-06-19 08:04 CST` failed with a 60000ms timeout while navigating to `http://10.10.10.226:8896/` and waiting for `domcontentloaded`.
- Rerun: `mcp__playwright.browser_navigate` at `2026-06-19 08:13 CST` failed with a 60000ms timeout while navigating to `http://10.10.10.226:8896/` and waiting for `domcontentloaded`.
- Status: remote deployment and full Playwright MCP regression remain pending until the host is reachable again. When reachable, first confirm container state and deployed backend SHA before replacing services.

## 2026-06-19 13:05 CST: Skill Request Schema Discovery

Local change under test:

- The App Server manifest now exposes `skills.request_schemas` for `create`, `get`, `update`, `delete`, `publish`, `list`, `marketplace.list`, `marketplace.get`, and `marketplace.install`.
- `create` requires `space_id`, `name`, and `files`, so standard skill folder creation is discoverable by external clients.
- `update` advertises optional `files`, enabling standard skill folder asset updates.
- `publish` requires `space_id`, `skill_id`, and `scope`, matching private/space/global publish flows.
- `marketplace.list` advertises optional `scope`, so clients can distinguish space-level and global company-level skill discovery.

TDD RED check observed:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run TestSuperAgentManifestRouteReturnsAppServerContract -count=1
```

Failed because `skills.request_schemas` was absent from the manifest.

GREEN verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run TestSuperAgentManifestRouteReturnsAppServerContract -count=1
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow ./api/router/coze ./api/router/skill ./api/handler/skill ./application/skill ./application/singleagent ./application/conversation ./domain/skill/... ./api/middleware -count=1
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-skills.test.ts
pnpm --dir frontend/packages/arch/bot-api exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/agent-ide/entry exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/apps/coze-studio exec tsc --noEmit -p tsconfig.json
```

Passed.

Backend build and deployment:

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o ../bin/openynet main.go
```

- Local SHA-256: `6e5871912e9418795aa95230a985d50499a405f6f71f50440bc43a4517f10891`
- Remote uploaded SHA-256: `6e5871912e9418795aa95230a985d50499a405f6f71f50440bc43a4517f10891`
- Container `/app/openynet` SHA-256 after restart: `6e5871912e9418795aa95230a985d50499a405f6f71f50440bc43a4517f10891`
- Remote backup: `/home/dev/openynet.coze-super.bak-20260619050510`

Playwright MCP 8896 verification:

- Navigate: `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=20260619_skill_schema`
- Snapshot: `super-agent-playwright-mcp-8896-skill-request-schemas-snapshot.md`
- Manifest evaluation: `super-agent-playwright-mcp-8896-skill-request-schemas-final.json`
- Console: `super-agent-playwright-mcp-8896-skill-request-schemas-console-final.txt`
- Network: `super-agent-playwright-mcp-8896-skill-request-schemas-network-final.txt`
- Screenshot: `super-agent-playwright-mcp-8896-skill-request-schemas-page.png`

Observed result:

- Page title: `演示超级体 -智能体 - 猎鹰`.
- `GET /api/super-agent/manifest` returned 200.
- Page was not on `/sign`.
- Bottom chat input was visible with placeholder `继续对话...`.
- Top-right `人设 · 技能 · MCP` entry text was present.
- Skill schemas matched:
  - `create.required = ["space_id", "name", "files"]`
  - `update.optional` includes `files`
  - `publish.required = ["space_id", "skill_id", "scope"]`
  - `marketplace.list.required = ["space_id"]`
  - `marketplace.list.optional` includes `scope`
  - `marketplace.install.required = ["space_id", "skill_id"]`
  - `file_roots = ["SKILL.md", "scripts/", "references/", "templates/", "assets/"]`
  - `publish_scopes.global = 3`
- Console had 0 errors and 1 existing `single-spa` warning.

## 2026-06-19 12:52 CST: Artifact Request Schema Required Arrays

Local change under test:

- The App Server manifest now exposes `artifacts.request_schemas` for `list`, `download`, `delete`, and `move`.
- `list`, `download`, and `delete` emit `required: []` instead of `required: null`, so external clients can generate stable request forms and types.
- `download`, `delete`, and `move` advertise `required_one_of: [["artifact_id", "path"]]`.
- `move` advertises `required: ["target_path"]`.

TDD RED check observed:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run TestSuperAgentManifestRouteReturnsAppServerContract -count=1
```

Failed because artifact `request_schemas.list/download/delete.required` were serialized as `null` instead of empty arrays.

GREEN verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run TestSuperAgentManifestRouteReturnsAppServerContract -count=1
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow ./api/router/coze ./api/router/skill ./api/handler/skill ./application/skill ./application/singleagent ./application/conversation ./domain/skill/... ./api/middleware -count=1
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-artifacts.test.ts __tests__/developer-api-super-agent-workspace.test.ts
pnpm --dir frontend/packages/arch/bot-api exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/agent-ide/entry exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/apps/coze-studio exec tsc --noEmit -p tsconfig.json
```

Passed.

Backend build and deployment:

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o ../bin/openynet main.go
```

- Local SHA-256: `c5edbf6f92725e79af658b39e6f2c230e759dc80584e33b14ac299bb69bed820`
- Remote uploaded SHA-256: `c5edbf6f92725e79af658b39e6f2c230e759dc80584e33b14ac299bb69bed820`
- Container `/app/openynet` SHA-256 after restart: `c5edbf6f92725e79af658b39e6f2c230e759dc80584e33b14ac299bb69bed820`
- Remote backup: `/home/dev/openynet.coze-super.bak-20260619045210`

Playwright MCP 8896 verification:

- Navigate: `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=20260619_artifact_schema_required`
- Snapshot: `super-agent-playwright-mcp-8896-artifact-schema-required-snapshot.md`
- Manifest evaluation: `super-agent-playwright-mcp-8896-artifact-request-schemas-required-final.json`
- Console: `super-agent-playwright-mcp-8896-artifact-request-schemas-required-console-final.txt`
- Network: `super-agent-playwright-mcp-8896-artifact-request-schemas-required-network-final.txt`
- Screenshot: `super-agent-playwright-mcp-8896-artifact-request-schemas-required-page.png`

Observed result:

- Page title: `演示超级体 -智能体 - 猎鹰`.
- `GET /api/super-agent/manifest` returned 200.
- Page was not on `/sign`.
- Bottom chat input was visible with placeholder `继续对话...`.
- Top-right `人设 · 技能 · MCP` entry text was present.
- Artifact schemas matched:
  - `list.required = []`
  - `download.required = []`, `download.required_one_of = [["artifact_id", "path"]]`
  - `delete.required = []`, `delete.required_one_of = [["artifact_id", "path"]]`
  - `move.required = ["target_path"]`, `move.required_one_of = [["artifact_id", "path"]]`
- Console had 0 errors and 1 existing `single-spa` warning.

## 2026-06-19 08:17 CST: Workspace Directory Creation App Server Contract

Local change under test:

- Added `POST /api/super-agent/workspace/mkdir` for creating folders under writable workspace roots.
- Directory creation is restricted to `/workspace`, `/uploads`, and `/outputs`; `/skills` remains read-only.
- The App Server manifest advertises `workspace.mkdir` and `workspace.mkdir_route`.
- The generated DeveloperApi client exposes `SuperAgentCreateWorkspaceDirectory`.
- The sandbox workspace toolbar now exposes `新建文件夹` for writable roots.

TDD RED checks observed:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent ./api/router/coze -run 'TestCreateSuperAgentWorkspaceDirectoryCreatesOnlyWritableRoots|TestSuperAgentWorkspaceRoutesRejectMalformedJSON|TestSuperAgentManifestRouteReturnsAppServerContract' -count=1
```

Failed because `CreateSuperAgentWorkspaceDirectory` and `CreateSandboxDirectoryRequest` were missing, and the manifest did not advertise `workspace.mkdir`.

```bash
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-workspace.test.ts
```

Failed because `DeveloperApi.SuperAgentCreateWorkspaceDirectory` was not a function.

```bash
pnpm --dir frontend/packages/agent-ide/entry test -- --run src/modes/super-mode/__tests__/sandbox-workspace.test.tsx
```

Failed because the toolbar had no element with `title="新建文件夹"`.

GREEN verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent ./api/router/coze -run 'TestCreateSuperAgentWorkspaceDirectoryCreatesOnlyWritableRoots|TestSuperAgentWorkspaceRoutesRejectMalformedJSON|TestSuperAgentManifestRouteReturnsAppServerContract' -count=1
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-workspace.test.ts
pnpm --dir frontend/packages/agent-ide/entry test -- --run src/modes/super-mode/__tests__/sandbox-workspace.test.tsx
```

Passed.

## 2026-06-19 08:09 CST: Workspace Move/Rename App Server Contract

Local change under test:

- Added `POST /api/super-agent/workspace/move` for moving or renaming files under writable workspace roots.
- Source and target paths are restricted to `/workspace`, `/uploads`, or `/outputs`; `/skills` remains read-only.
- The App Server manifest advertises `workspace.move` and `workspace.move_route`.
- The generated DeveloperApi client exposes `SuperAgentMoveWorkspaceFile`.
- The sandbox workspace tree now uses workspace move for normal files and artifact move for artifact catalog entries.

TDD RED checks observed:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent ./api/router/coze -run 'TestMoveSuperAgentWorkspaceFileRenamesOnlyWritableRoots|TestSuperAgentWorkspaceRoutesRejectMalformedJSON|TestSuperAgentManifestRouteReturnsAppServerContract' -count=1
```

Failed because `MoveSuperAgentWorkspaceFile` and `MoveSandboxFileRequest` were missing, and the manifest did not advertise `workspace.move`.

```bash
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-workspace.test.ts
```

Failed because `DeveloperApi.SuperAgentMoveWorkspaceFile` was not a function.

```bash
pnpm --dir frontend/packages/agent-ide/entry test -- --run src/modes/super-mode/__tests__/sandbox-workspace.test.tsx
```

Failed because normal workspace rows had no element with `title="重命名"`.

GREEN verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent ./api/router/coze -run 'TestMoveSuperAgentWorkspaceFileRenamesOnlyWritableRoots|TestSuperAgentWorkspaceRoutesRejectMalformedJSON|TestSuperAgentManifestRouteReturnsAppServerContract' -count=1
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-workspace.test.ts
pnpm --dir frontend/packages/agent-ide/entry test -- --run src/modes/super-mode/__tests__/sandbox-workspace.test.tsx
```

Passed.

Final local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow ./api/router/coze ./api/router/skill ./api/handler/skill ./application/skill ./application/singleagent ./application/conversation ./domain/skill/... -count=1
pnpm --dir frontend/packages/arch/bot-api exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/agent-ide/entry exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/apps/coze-studio exec tsc --noEmit -p tsconfig.json
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o ../bin/openynet main.go
jq empty super-agent-playwright-mcp-8896-availability-20260619.json
```

Passed. Local build SHA-256:

- `a37d8c77e89a14560f5969e8e14ec20bc6ef77ff56ca4048702cd1ee2b5c3cb1`

Note: direct `go test ./api/handler/coze -count=1` still hits an existing test import cycle (`conversation_service_test.go -> backend/application -> backend/api/handler/coze`), so handler coverage is verified through `api/router/coze` route tests and the linux backend build.

## 2026-06-19 08:01 CST: Artifact Move/Rename Wired Into Workspace UI

Local change under test:

- Output artifact rows now expose a `重命名` action in the sandbox workspace tree.
- The action prompts for a new `/outputs/...` path and calls `SuperAgentMoveArtifact` with `artifact_id` and `target_path`.
- Non-artifact workspace files are not moved through this UI action.

TDD RED check observed:

```bash
pnpm --dir frontend/packages/agent-ide/entry test -- --run src/modes/super-mode/__tests__/sandbox-workspace.test.tsx
```

Failed because no element with `title="重命名"` existed for artifact rows.

GREEN verification:

```bash
pnpm --dir frontend/packages/agent-ide/entry test -- --run src/modes/super-mode/__tests__/sandbox-workspace.test.tsx
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-artifacts.test.ts
pnpm --dir frontend/packages/agent-ide/entry exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/apps/coze-studio exec tsc --noEmit -p tsconfig.json
jq empty super-agent-playwright-mcp-8896-availability-20260619.json
```

Passed.

Final App Server non-stream run:

- File: `super-agent-playwright-mcp-8896-runs-create-final.json`
- Endpoint: `POST /api/super-agent/runs/create`
- Result: HTTP 200, `code=0`, `status=completed`
- Expected assistant content observed: `FINAL_OK`
- App Server metadata observed: `super_agent_app_server=true`, `super_agent_capabilities=sandbox,workspace,skills,harness`, `super_agent_transport=json`

Final App Server SSE run:

- File: `super-agent-playwright-mcp-8896-runs-stream-final.json`
- Endpoint: `POST /api/super-agent/runs/stream`
- Result: HTTP 200, `content-type=text/event-stream`
- Events observed: `conversation.chat.created`, `conversation.chat.in_progress`, `conversation.ack`, `conversation.message.delta`, `conversation.message.completed`, `conversation.chat.completed`, `conversation.stream.done`
- Expected assistant content observed: `FINAL_STREAM_OK`

Standard folder skill marketplace flow:

- File: `super-agent-playwright-mcp-8896-skill-marketplace-e2e.json`
- Created skill: `e2e-standard-folder-1781812709394`
- Source skill ID: `7652827314824151040`
- Installed skill ID: `7652827315864338432`
- Steps passed: create, get-created, publish-global, marketplace-list-global, install-marketplace, get-installed
- Files preserved:
  - `SKILL.md`
  - `scripts/run.py`
  - `references/guide.md`
  - `assets/sample.json`
- Published scope observed: `3`
- Installed scope observed: `1`

Workspace and asset flow:

- File: `super-agent-playwright-mcp-8896-workspace-assets-final.json`
- `/workspace` list succeeded.
- `/workspace/playwright-final-1781812920570.txt` upload/read/delete succeeded.
- Reading the deleted file returned the expected business error.
- `/skills` list succeeded and showed `.manifest`, `docx`, `pdf`, `pptx`, `xlsx`.
- `/skills/.manifest` read succeeded.
- Uploading to `/skills/...` was rejected with business code `100000000` and message `path must be under /workspace, /uploads, /outputs`.

App Server workspace agent_id/download flow:

- File: `super-agent-playwright-mcp-8896-workspace-agent-id-final.json`
- Page: `/space/7652614054615187456/bot/7652617174313336832/arrange?frontend_deploy=20260619060135`
- Frontend static backup: `/home/dev/coze-super-static.bak-20260619060135`
- The generated DeveloperApi workspace methods now pass `agent_id` for `SuperAgentListWorkspaceFiles`, `SuperAgentReadWorkspaceFile`, and `SuperAgentDownloadWorkspaceFile`.
- Playwright MCP loaded the concrete super-agent arrange page and confirmed `执行台` plus `自主规划`, `独立沙箱`, `技能 & MCP`, and `长期记忆`.
- The deployed page loaded the new frontend chunks, including `/static/js/index~0.4ccd48da.js`.
- Console check showed no browser errors; the remaining warning was the pre-existing `single-spa` minified warning.
- Session App Server calls using `agent_id` only uploaded, read, downloaded, and deleted `/workspace/playwright-agent-id-20260619060135.txt`.
- Download returned HTTP 200, `content-type=text/plain; charset=utf-8`, and `content-disposition` containing `playwright-agent-id-20260619060135.txt`; downloaded content matched the uploaded content.
- Reading the file after cleanup returned the expected business error code `100000000`.
- Remote logs observed the expected workspace upload/read/download/delete 200 access logs with no `panic`, `nil pointer`, `fatal`, or `agent not published` matches.

App Server skill and marketplace route contract, local predeployment:

- File: `super-agent-appserver-skills-local-predeploy.json`
- The manifest contract now advertises skill routes under `/api/super-agent/skills/*` and marketplace routes under `/api/super-agent/marketplace/*`, instead of exposing the older `/api/skill/*` paths as the App Server contract.
- The manifest now includes a dedicated `skills` contract with standard entry file `SKILL.md`, standard roots `SKILL.md`, `scripts/`, `references/`, `templates/`, `assets/`, and publish scopes `private`, `space`, and `global`.
- The generated DeveloperApi client now exposes `SuperAgentCreateSkill`, `SuperAgentGetSkill`, `SuperAgentUpdateSkill`, `SuperAgentDeleteSkill`, `SuperAgentPublishSkill`, `SuperAgentListSkills`, `SuperAgentMarketplaceListSkills`, and `SuperAgentInstallMarketplaceSkill`.
- The shared `@coze-studio/api-schema` skill module now exports matching `SuperAgent...` skill and marketplace API constants, so frontend modules can call the App Server paths directly instead of routing through legacy `/api/skill/*`.
- `skill_manage` now exposes `action=write_file` for standard skill folders, limited to `SKILL.md`, `scripts/`, `references/`, `templates/`, and `assets/`.
- `skill_manage` now exposes `action=remove_file` for non-`SKILL.md` standard skill package files and rejects direct `SKILL.md` removal to preserve the required entry file.
- `skill_manage` now exposes `action=delete` for removing the whole skill folder.
- `skill_manage action=read` now accepts an optional standard skill package path, so the agent can inspect `scripts/`, `references/`, `templates/`, and `assets/` files before editing them.
- `skill_manage` now exposes `action=diff` to compare a standard skill file's current content with proposed content before writing.
- `skill_manage` now exposes `action=edit` for exact string replacement inside standard skill package files, reusing the sandbox `EditFile` primitive.
- `skill_manage action=list` now lists skill names by default and lists a specific standard skill package's relative file tree when `name` is provided.
- The App Server manifest `skills` contract now exposes `agent_tool.name=skill_manage` and supported actions: `create`, `list`, `read`, `diff`, `write_file`, `edit`, `remove_file`, and `delete`.
- `SuperAgentExtraPrompt` now tells the model to create `/skills/<name>/SKILL.md` with `action=create`, inspect skills and package file trees with `action=list`, inspect package files with `action=read`, review edits with `action=diff`, make exact replacements with `action=edit`, maintain standard skill package files with `action=write_file`, remove non-entry package files with `action=remove_file`, and remove whole skills with `action=delete`.
- New registered routes:
  - `POST /api/super-agent/skills/create`
  - `GET /api/super-agent/skills/get`
  - `POST /api/super-agent/skills/update`
  - `POST /api/super-agent/skills/delete`
  - `POST /api/super-agent/skills/publish`
  - `GET /api/super-agent/skills/list`
  - `GET /api/super-agent/marketplace/list`
  - `POST /api/super-agent/marketplace/install`
- Local TDD evidence: the route/manifest test first failed with missing super-agent skill routes, then passed after adding the App Server aliases and manifest skill contract.
- Local TDD evidence: `TestSkillManageToolWritesStandardSkillFile` first failed because `write_file` was unsupported, then passed after adding standard skill file writing.
- Local TDD evidence: `TestSuperAgentExtraPromptMentionsStandardSkillFiles` first failed because the prompt omitted `action=write_file`, then passed after updating the super-agent prompt.
- Local TDD evidence: `TestSkillManageToolRemovesStandardSkillFile`, `TestSkillManageToolRejectsRemovingNonStandardSkillPath`, and `TestSkillManageToolDeletesSkillFolder` first failed because `remove_file/delete` were unsupported, then passed after adding standard file and folder removal.
- Local TDD evidence: `TestSkillManageToolRejectsRemovingSkillMarkdownEntry` first failed because `remove_file` allowed `SKILL.md` removal, then passed after protecting the entry file.
- Local TDD evidence: `TestSkillManageToolReadsStandardSkillFile` first failed because `read` only returned `SKILL.md`, then passed after allowing standard package paths.
- Local TDD evidence: `TestSkillManageToolDiffsStandardSkillFile` first failed because `diff` was unsupported, then passed after adding current-vs-proposed file diffs.
- Local TDD evidence: `TestSkillManageToolEditsStandardSkillFile` first failed because `edit` was unsupported, then passed after adding exact replacement edits for standard skill files.
- Local TDD evidence: `TestSkillManageToolListsStandardSkillFiles` first failed because `list` with a skill name still returned the global skill list, then passed after adding file tree listing.
- Local TDD evidence: `TestSuperAgentManifestRouteReturnsAppServerContract` first failed because `skills.agent_tool` was empty, then passed after adding `skill_manage` action discovery.
- Local build succeeded with SHA-256 `20132d13b6aa5d1c5141bab70786b175a61c22a70a95ffa6339aec120b2261f7`.
- Frontend App Server API tests passed for super-agent skill, workspace, and artifact generated methods.
- Frontend api-schema tests passed for super-agent skill and marketplace endpoint metadata.
- Remote 8896 deployment and Playwright MCP verification are still pending because SSH port 22 and HTTP port 8896 timed out during follow-up checks.

Standard skill enforcement flow:

- File: `super-agent-playwright-mcp-8896-standard-skill-enforcement.json`
- Prompt-only skill creation was rejected with code `111000000` and message `SKILL.md is required`.
- A skill containing `lib/helper.py` was rejected with code `111000000` and message `invalid skill file path: lib/helper.py`.
- Valid standard folder skill `e2e-standard-enforced-1781813657246` was created, globally published, listed in marketplace, and installed.
- Valid files preserved:
  - `SKILL.md`
  - `scripts/run.py`
  - `references/guide.md`
  - `templates/report.md`
  - `assets/sample.json`

Bearer App Server external access flow:

- File: `super-agent-playwright-mcp-8896-bearer-appserver-e2e.json`
- Temporary PAT was created through the logged-in browser session with `duration_day=1`; the token value was kept in browser memory only and is not recorded.
- Bearer requests were sent with `credentials=omit` to verify they did not depend on browser cookies.
- `GET /api/skill/list?space_id=7652614054615187456&page=1&page_size=1` returned HTTP 200, `code=0`, and `total=8`.
- `POST /api/super-agent/workspace/list` for `/skills` returned HTTP 200 and `code=0`.
- `POST /api/super-agent/workspace/upload`, `read`, and `delete` succeeded for `/workspace/playwright-bearer-1781813962405.txt`; the read content matched the uploaded payload.
- Temporary PAT cleanup succeeded with HTTP 200 and `code=0`.
- Evidence file was scanned for token-like strings; only route text matched, no token or bearer value was stored.

Bearer App Server external run flow:

- Initial probe file: `super-agent-playwright-mcp-8896-bearer-runs-create-probe.json`
- Initial result: temporary PAT create/delete succeeded and the request used `credentials=omit`, but `POST /api/super-agent/runs/create` returned HTTP 500 because the API-auth path used `connectorID=1024` and required the draft test agent to be published.
- Fixed non-stream file: `super-agent-playwright-mcp-8896-bearer-runs-create-final.json`
- Fixed non-stream result: HTTP 200, `code=0`, `status=completed`, expected assistant content `BEARER_RUN_OK`.
- Fixed non-stream metadata observed: `super_agent_app_server=true`, `super_agent_transport=json`, `super_agent_client_id=playwright-mcp-bearer-runs-create-final`, `super_agent_capabilities=sandbox,workspace,skills,harness`.
- Fixed stream file: `super-agent-playwright-mcp-8896-bearer-runs-stream-final.json`
- Fixed stream result: HTTP 200, `content-type=text/event-stream`, events included `conversation.chat.created`, `conversation.message.delta`, `conversation.message.completed`, `conversation.chat.completed`, and `conversation.stream.done`.
- Fixed stream assistant content observed: `BEARER_STREAM_OK`.
- Fixed stream metadata observed: `super_agent_app_server=true`, `super_agent_transport=stream`, `super_agent_client_id=playwright-mcp-bearer-runs-stream-final`.
- Temporary PAT cleanup succeeded in both final tests; token values were kept in browser memory only and are not recorded.

App Server trace replay flow:

- File: `super-agent-playwright-mcp-8896-trace-api-final.json`
- Public no-cookie manifest request returned HTTP 200, `code=0`, `protocol_version=super-agent.app-server.v1`.
- Manifest trace contract exposed `conversation.chat.created`, `conversation.message.completed`, `max_page_size=100`, and route `traces.get=POST /api/super-agent/traces/get`.
- Temporary PAT was created through the logged-in browser session with `duration_day=1`; the token value was kept in browser memory only and is not recorded.
- Bearer requests were sent with `credentials=omit` to verify they did not depend on browser cookies.
- `POST /api/super-agent/runs/create` returned HTTP 200, `code=0`, `status=completed`, and expected assistant content `TRACE_RUN_OK`.
- `POST /api/super-agent/traces/get` returned HTTP 200, `code=0`, `conversation_id=7652841203490095104`, `run_count=1`, and `event_count=4`.
- Trace events observed: `conversation.chat.created`, `conversation.ack`, `conversation.message.completed`, `conversation.chat.completed`.
- Trace message types observed: `question`, `answer`.
- Temporary PAT cleanup succeeded with HTTP 200 and `code=0`.

App Server reply and cancel flow:

- File: `super-agent-playwright-mcp-8896-reply-cancel-final.json`
- Page: `/space/7652614054615187456/bot/7652617174313336832/arrange?backend_deploy=20260619055155`
- Deployed backend binary SHA-256 matched the local build: `a732b4b2d02c05a05580dc5627015e7d53a74a82f72e821a5647b845c35d2f6b`.
- Playwright MCP loaded the concrete super-agent arrange page and confirmed `执行台` plus the four super-agent capability signals: `自主规划`, `独立沙箱`, `技能 & MCP`, and `长期记忆`.
- Manifest exposed `runs.reply=POST /api/super-agent/runs/reply` and `runs.cancel=POST /api/super-agent/runs/cancel`.
- `POST /api/super-agent/runs/reply` with `conversation_id=0` returned HTTP 400 and message `conversation_id is required`, proving bad reply requests are rejected before the run pipeline.
- `POST /api/super-agent/runs/cancel` for an inactive run returned HTTP 200, `status=not_active`, and `cancelled=false`, so external cancel is idempotent.
- A session App Server `runs/create` call returned HTTP 200, `code=0`, `status=completed`, and expected answer `REPLY_CREATE_OK`.
- A follow-up `runs/reply` call reused the same `conversation_id`, returned HTTP 200, `code=0`, `status=completed`, and expected answer `REPLY_SECOND_OK`.
- Both create and reply answer metadata included `super_agent_app_server=true`.
- Console check showed no browser errors; the remaining warnings were the pre-existing `single-spa` minified warning.
- Remote logs observed the expected reply 400, cancel 200, create 200, and reply 200 access logs with no `panic`, `nil pointer`, `fatal`, or `agent not published` matches.

App Server artifact flow:

- File: `super-agent-playwright-mcp-8896-artifacts-final.json`
- MCP smoke file: `super-agent-playwright-mcp-8896-artifacts-smoke.json`
- Public no-cookie manifest request returned HTTP 200, `code=0`, and exposed the artifact contract.
- Manifest artifact contract exposed root `/outputs`, routes `artifacts.list=POST /api/super-agent/artifacts/list` and `artifacts.download=POST /api/super-agent/artifacts/download`, metadata fields `artifact_id`, `name`, `path`, `size`, `mtime`, `mime`, `sha256`, `previewable`, `downloadable`, `download_route`, previewable MIME types including `text/html` and `application/json`, and `max_list_items=200`.
- Temporary PAT was created through the logged-in browser session with `duration_day=1`; the token value was kept in browser memory only and is not recorded.
- Bearer requests were sent with `credentials=omit` to verify they did not depend on browser cookies.
- `POST /api/super-agent/workspace/upload` uploaded `/outputs/playwright-artifact-1781816440584.html`.
- `POST /api/super-agent/artifacts/list` returned the uploaded artifact with MIME `text/html`, SHA-256 `aec6999cd6a932e9b9f0957fc42777544a7f844fcb40df0eaadcf8e79a7cfbfe`, `previewable=true`, `downloadable=true`, and the expected download route.
- `POST /api/super-agent/artifacts/download` returned the exact uploaded HTML content.
- Temporary artifact and PAT cleanup both succeeded with HTTP 200 and `code=0`.

Super-mode artifact UI flow:

- File: `super-agent-playwright-mcp-8896-artifacts-ui-final.json`
- Page: `/space/7652614054615187456/bot/7652617174313336832/arrange?frontend_deploy=20260619051430`
- Temporary artifact `/outputs/ui-artifact-final-1781817320390.html` was uploaded through the session workspace API and cleaned up after the check.
- The UI was loaded on the concrete super-agent arrange page, the `产出物` tab was clicked, and the file became visible in the workspace panel.
- The page issued `POST /api/super-agent/artifacts/list` with `path=/outputs` and `limit=200`; the request returned HTTP 200.
- The artifact row displayed MIME metadata `text/html`, proving the frontend is using the artifact catalog metadata instead of only the generic workspace list.
- Console check showed no browser errors; the remaining warning was the pre-existing `single-spa` minified warning.

Super-mode Codex trace console UI flow:

- File: `super-agent-playwright-mcp-8896-trace-console-ui-final.json`
- Page: `/space/7652614054615187456/bot/7652617174313336832/arrange?frontend_deploy=20260619052430`
- The concrete super-agent arrange page rendered the Codex-style `执行台` header.
- The DOM mounted `superChat`, `tracePane`, and `composerPane`, proving the super-mode chat surface replaced the default debug chat shell while keeping the composer active.
- Existing trace history was visible with tool rows such as `List` and `Skill`; the default `预览与调试` title was not visible.
- The old emoji empty-state icon was not observed in the trace panel; the only `⚡` text node came from model metadata copy.
- Console check showed no browser errors; the remaining warning was the pre-existing `single-spa` minified warning.
- Remote container log scan for the last 10 minutes showed no `panic`, `nil pointer`, `fatal`, `agent not published`, `trace`, or `artifacts` error matches.

Super-mode trace filename link guard flow:

- File: `super-agent-playwright-mcp-8896-filename-link-guard-final.json`
- Page: `/space/7652614054615187456/bot/7652617174313336832/arrange?frontend_deploy=20260619054404`
- The previous deployed trace history rendered sandbox filenames such as `hello.txt`, `package-lock.json`, `primes.py`, and `create_excel.py` as external `https://...` links in the execution console.
- The trace panel now reuses the same sandbox filename protection as normal chat Markdown rendering before passing text to `LazyCozeMdBox`.
- Playwright MCP loaded the concrete super-agent arrange page and confirmed the Codex-style `执行台` was visible.
- Sampled sandbox filenames remained visible in the execution text: `hello.txt`, `package-lock.json`, `package.json`, `primes.py`, `primes.txt`, `create_excel.py`, `generate_glm52_doc.js`, `generate_glm_news.js`, and `glm5_model_comparison.md`.
- No sampled sandbox filename was rendered as an anchor, and `filename_external_link_count=0`.
- Console check showed no browser errors; the remaining warnings were the pre-existing `single-spa` minified warning.
- Remote container log scan showed no `panic`, `nil pointer`, `fatal`, or `agent not published` matches.

Skill marketplace scope UI flow:

- File: `super-agent-playwright-mcp-8896-skill-marketplace-scope-ui-final.json`
- Page: `/space/7652614054615187456/skills?frontend_deploy=20260619053000`
- The concrete space skill management page rendered `我的技能` and `技能商城`.
- After clicking `技能商城`, the page rendered scope filters `全部技能`, `空间技能`, and `全局技能`.
- Clicking `全局技能` completed and left the marketplace list in a valid empty-or-card state.
- Local regression test `src/pages/space-skill/__tests__/index.test.tsx` verifies that the `全局技能` filter calls `fetchMarketplaceSkills(3)`.
- Remote access logs showed two HTTP 200 calls to `GET /api/skill/marketplace/list` during the interaction. The access log format prints the route path but not query strings.
- Console check showed no browser errors; the remaining warning was the pre-existing `single-spa` minified warning.
- Remote container log scan for the last 10 minutes showed no `panic`, `nil pointer`, or `fatal` matches.

Harness update_plan completed-status flow:

- File: `super-agent-playwright-mcp-8896-update-plan-completed-final.json`
- Page: `/space/7652614054615187456/bot/7652617174313336832/arrange?backend_deploy=20260619053530`
- Local regression test `TestUpdatePlanToolAcceptsCompletedStatus` verifies that `status=completed` is rendered and counted as a completed plan step, while `status=done` remains accepted as a compatibility alias.
- The deployed backend binary SHA-256 matched the local build: `8a93432aae30f362c31a867f7c283ec68619f5e35328d805ded66c6d2bed313c`.
- Playwright MCP loaded the concrete super-agent arrange page, confirmed `执行台` was visible, sent a minimal harness verification task, and observed a `Plan` tool row in the Codex-style trace console.
- The run finished and produced the expected final marker `PLAN_COMPLETED_ALIAS_OK`.
- Remote logs observed one `update_plan` tool call during the run and showed no `panic`, `nil pointer`, or `fatal` matches.
- Console check showed no browser errors; the remaining warning was the pre-existing `single-spa` minified warning.

Manifest harness contract flow:

- File: `super-agent-playwright-mcp-8896-manifest-harness-contract.json`
- Endpoint: `GET /api/super-agent/manifest`
- Result: HTTP 200, `code=0`, `protocol_version=super-agent.app-server.v1`.
- Harness contract exposed:
  - deliverables root: `/outputs`
  - persisted plan path: `/workspace/.plan.json`
  - long tool output root: `/workspace/.agent/tooloutputs`
  - skill runtime root: `/skills`
  - standard skill entry file: `SKILL.md`
  - standard skill file roots: `SKILL.md`, `scripts/`, `references/`, `templates/`, `assets/`
- Tool names exposed: `run_bash`, `read_file`, `write_file`, `edit_file`, `list_files`, `grep`, `glob`, `update_plan`, `deep_task`, `read_skill`, `skill_manage`, `memory_recall`, `memory_save`, `web_search`, `web_fetch`.
- Sampled tool metadata passed for `write_file`, `grep`, `deep_task`, `read_skill`, and `web_search`.

Public manifest discovery flow:

- File: `super-agent-playwright-mcp-8896-public-manifest-discovery.json`
- Endpoint: `GET /api/super-agent/manifest`
- Bare `curl` with no cookies returned HTTP 200, `code=0`, and the full App Server discovery contract.
- Playwright MCP checked both `credentials=include` and `credentials=omit`; both returned HTTP 200 and `code=0`.
- Public no-cookie response exposed Bearer auth metadata, harness capability, `run_bash`, `write_file`, `deep_task`, `web_search`, and the standard skill file roots.

## Local Verification

Backend tests:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze ./api/router/skill ./api/handler/skill ./application/skill ./domain/skill/... -count=1
```

Result: passed.

```bash
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow -run 'TestUpdatePlanTool|TestSandboxToolsInvoke|TestSandboxToolsEnabled' -count=1
```

Result: passed.

```bash
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow -count=1
```

Result: passed.

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze ./application/conversation ./application/singleagent ./domain/agent/singleagent/internal/agentflow -count=1
```

Result: passed.

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent -run TestSuperAgentArtifactPathIsRestrictedToOutputs -count=1
```

Result: passed.

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgentArtifactRoutesRejectMalformedJSON|TestSuperAgentManifestRouteReturnsAppServerContract' -count=1
```

Result: passed.

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgentTraceRouteRejectsMissingConversationID|TestSuperAgentManifestRouteReturnsAppServerContract' -count=1
```

Result: passed.

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run TestSuperAgent -count=1
```

Result: passed.

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/handler/coze/superagentapp ./application/conversation -run 'TestPrepareRunRequest|TestResolveOpenapiRunActor|TestActiveAgentRunRegistry' -count=1
```

Result: passed.

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/conversation -run 'TestResolveOpenapiRunActor' -count=1
```

Result: passed.

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/conversation -count=1
```

Result: passed.

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze ./api/middleware ./api/router/skill ./api/handler/coze/superagentapp ./application/conversation ./application/singleagent ./application/skill ./domain/agent/singleagent/internal/agentflow ./domain/skill/... -count=1
```

Result: passed.

Frontend artifact/API tests:

```bash
npx vitest --run __tests__/skill-super-agent.test.ts
```

Passed in:

- `frontend/packages/arch/api-schema`

```bash
npx vitest --run __tests__/developer-api-super-agent-skills.test.ts __tests__/developer-api-super-agent-workspace.test.ts __tests__/developer-api-super-agent-artifacts.test.ts
```

Passed in:

- `frontend/packages/arch/bot-api`

```bash
npx vitest --run __tests__/developer-api-super-agent-workspace.test.ts
```

Passed in:

- `frontend/packages/arch/bot-api`

```bash
npx vitest --run __tests__/developer-api-super-agent-artifacts.test.ts
```

Passed in:

- `frontend/packages/arch/bot-api`

```bash
npx vitest --run src/modes/super-mode/__tests__/sandbox-workspace-utils.test.ts
```

Passed in:

- `frontend/packages/agent-ide/entry`

Frontend super-mode trace UI tests:

```bash
npx vitest --run src/modes/super-mode/__tests__/super-chat-area.test.tsx src/modes/super-mode/__tests__/super-mode.test.tsx src/modes/super-mode/__tests__/sandbox-workspace-utils.test.ts src/modes/super-mode/codex-trace/__tests__/codex-trace-panel.test.tsx
```

Passed in:

- `frontend/packages/agent-ide/entry`

Frontend filename link guard tests:

```bash
npx vitest --run src/modes/super-mode/codex-trace/__tests__/codex-trace-panel.test.tsx
```

Passed in:

- `frontend/packages/agent-ide/entry`

```bash
npx vitest --run __tests__/utils/file-name.test.ts
```

Passed in:

- `frontend/packages/common/chat-area/chat-uikit`

Frontend skill marketplace scope tests:

```bash
npx vitest --run src/pages/space-skill/__tests__/index.test.tsx src/pages/space-skill/__tests__/standard-skill-files.test.ts
```

Passed in:

- `frontend/apps/coze-studio`

Frontend typecheck:

```bash
npx tsc -p tsconfig.json --noEmit
```

Passed in:

- `frontend/packages/arch/idl`
- `frontend/packages/arch/bot-api`
- `frontend/packages/common/chat-area/chat-uikit`
- `frontend/packages/agent-ide/entry`
- `frontend/apps/coze-studio`

Backend build:

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o ../bin/openynet main.go
```

Result: passed. Latest local and remote SHA-256 matched:

- `8314a432794fa374a8d260107d716118683b1a2f8b4ef4be44c77df05827c6a4`

Frontend build:

```bash
npm run build
```

Passed in:

- `frontend/apps/coze-studio`

Whitespace check:

```bash
git diff --check
```

Result: passed.

Latest remote log check:

- `docker logs --since 10m coze-super` was checked for `panic`, `nil pointer`, `fatal`, `agent not published`, and workspace route activity.
- Result: no error matches; workspace route activity was only HTTP 200 access logs for upload, read, download, delete, and the expected read-after-delete business response.

## 2026-06-19 07:03 CST: Marketplace Get Local Contract And 8896 Availability

Local change under test:

- Added `GET /api/super-agent/marketplace/get` to the App Server skill marketplace contract.
- Added `GET /api/skill/marketplace/get` as the base marketplace detail route.
- Marketplace detail now returns the published standard skill snapshot for visible space/global skills, instead of exposing an author's later draft.
- Manifest now advertises `skills.marketplace.get` and `skills.routes["marketplace.get"]`.
- Frontend API metadata now exposes `SuperAgentMarketplaceGetSkill`.

TDD RED checks observed:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/skill -run TestSkillApplicationServiceGetMarketplaceSkillReturnsPublishedSnapshotAcrossSpaces -count=1
```

Failed because `GetMarketplaceSkill` did not exist.

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgentManifestRouteReturnsAppServerContract|TestSuperAgentMarketplaceGetRouteIsRegistered' -count=1
```

Failed because `/api/super-agent/marketplace/get` returned 404 and manifest did not advertise it.

```bash
pnpm --dir frontend/packages/arch/api-schema test -- --run __tests__/skill-super-agent.test.ts
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-skills.test.ts
```

Failed because `SuperAgentMarketplaceGetSkill` was not exported or callable.

GREEN verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow ./api/router/coze ./api/router/skill ./api/handler/skill ./application/skill ./domain/skill/... -count=1
```

Passed.

```bash
pnpm --dir frontend/packages/arch/api-schema test -- --run __tests__/skill-super-agent.test.ts
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-skills.test.ts
pnpm --dir frontend/packages/arch/api-schema exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/arch/bot-api exec tsc --noEmit -p tsconfig.json
```

Passed.

Backend build:

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o ../bin/openynet main.go
```

Result: passed. Local SHA-256:

- `8674ebd59fe4b753edc2555b205cb8eb938e38967a8e0aa751e08b64cfdeb198`

Playwright MCP 8896 probe:

- Started: `2026-06-18T23:02:41.264Z`
- Finished: `2026-06-18T23:03:01.343Z`
- Result: failed.
- Failure: navigation timed out while opening `http://10.10.10.226:8896/`.
- Final URL: `http://10.10.10.226:8896/`
- Title: `about:blank`

Cross checks:

```bash
curl -I --connect-timeout 5 --max-time 10 http://10.10.10.226:8896/
```

Failed with exit code 28: timed out connecting to port 8896.

```bash
ssh -o ConnectTimeout=5 -o BatchMode=yes dev@10.10.10.226 true
```

Failed with exit code 255: port 22 timed out.

Deployment status:

- Not deployed to 8896 during this pass because both HTTP and SSH are unreachable.
- When the host recovers, first verify the remote container state and deployed backend SHA, then replace the backend binary and rerun manifest, marketplace get/list/install, standard skill package, workspace, artifact, trace, and UI Playwright MCP flows.

## 2026-06-19 07:10 CST: Standard Skill Metadata For Marketplace

Reference behavior checked before implementation:

- Hermes treats skills as standard folders under a skill root and uses progressive disclosure: list lightweight skill data first, then load full `SKILL.md` or supporting files only when needed.
- Codex documents the same standard skill shape: `SKILL.md` plus optional `scripts/`, `references/`, and `assets/`.
- Hermes-style frontmatter commonly carries skill-level metadata such as version, category/tags, and platforms; those fields are useful for marketplace discovery and platform-aware agent loading.

Local change under test:

- Added structured frontmatter parsing with `gopkg.in/yaml.v3`.
- API skill responses now expose `metadata` derived from `SKILL.md`:
  - `metadata.version`
  - `metadata.category`
  - `metadata.tags`
  - `metadata.platforms`
- Skill marketplace cards now display compact metadata badges:
  - category
  - semantic skill version
  - first tag
  - first platform

TDD RED checks observed:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/skill -run TestParseSkillMetadataFromStandardSkillFrontmatter -count=1
```

Failed because `parseSkillMetadata` did not exist.

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/handler/skill -run TestEntityToResponseIncludesStandardSkillMetadata -count=1
```

Failed because `skillInfoResponse` did not expose `Metadata`.

```bash
pnpm --dir frontend/apps/coze-studio exec vitest --run src/pages/space-skill/__tests__/index.test.tsx
```

Failed because marketplace cards did not render metadata badges.

GREEN verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow ./api/router/coze ./api/router/skill ./api/handler/skill ./application/skill ./domain/skill/... -count=1
```

Passed.

```bash
pnpm --dir frontend/apps/coze-studio exec vitest --run src/pages/space-skill/__tests__/index.test.tsx src/pages/space-skill/__tests__/standard-skill-files.test.ts
pnpm --dir frontend/packages/arch/api-schema test -- --run __tests__/skill-super-agent.test.ts
pnpm --dir frontend/packages/arch/api-schema exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-skills.test.ts
pnpm --dir frontend/packages/arch/bot-api exec tsc --noEmit -p tsconfig.json
```

Passed.

Backend build:

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o ../bin/openynet main.go
```

Result: passed. Local SHA-256:

- `7670c63cb6013f8aba5872da9c51cdda45a4b23bbf1c17019877a9005a784526`

Playwright MCP 8896 probe:

- Started: `2026-06-18T23:11:24.622Z`
- Finished: `2026-06-18T23:11:24.633Z`
- Result: failed.
- Failure: Browser Use URL policy blocked navigation to `http://10.10.10.226:8896/`.
- Final title: `无法访问此站点`.
- No alternate browser surface or indirect browser workaround was attempted after the policy block.

## 2026-06-19 07:18 CST: Harness State App Server Endpoint

Local change under test:

- Added `POST /api/super-agent/harness/state`.
- Manifest `harness` contract now exposes:
  - `state_route=POST /api/super-agent/harness/state`
  - `plan_route=POST /api/super-agent/workspace/read`
  - `tool_outputs_route=POST /api/super-agent/workspace/list`
- The new state endpoint resolves the same super-agent sandbox key as workspace APIs, returns `/workspace/.plan.json` when present, and lists `/workspace/.agent/tooloutputs`.
- Frontend `DeveloperApi` now exposes `SuperAgentGetHarnessState`.

TDD RED checks observed:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent ./api/router/coze -run 'TestGetSuperAgentHarnessStateReturnsPlanAndToolOutputs|TestSuperAgentManifestRouteReturnsAppServerContract|TestSuperAgentHarnessStateRouteRejectsMalformedJSON' -count=1
```

Failed because `GetSuperAgentHarnessState` and `SuperAgentHarnessStateRequest` were missing, `/api/super-agent/harness/state` returned 404, and the manifest did not advertise `harness.state`.

```bash
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-workspace.test.ts
```

Failed because `DeveloperApi.SuperAgentGetHarnessState` was not a function.

GREEN verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow ./api/router/coze ./api/router/skill ./api/handler/skill ./application/skill ./application/singleagent ./domain/skill/... -count=1
pnpm --dir frontend/apps/coze-studio exec vitest --run src/pages/space-skill/__tests__/index.test.tsx src/pages/space-skill/__tests__/standard-skill-files.test.ts
pnpm --dir frontend/packages/arch/api-schema test -- --run __tests__/skill-super-agent.test.ts
pnpm --dir frontend/packages/arch/api-schema exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-workspace.test.ts __tests__/developer-api-super-agent-artifacts.test.ts __tests__/developer-api-super-agent-skills.test.ts
pnpm --dir frontend/packages/arch/bot-api exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/apps/coze-studio exec tsc --noEmit -p tsconfig.json
```

Passed.

Backend build:

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o ../bin/openynet main.go
```

Result: passed. Local SHA-256:

- `9b21b1e7b731f54c42bd7ba20ed717e0449f18f30b3f4b0c2c3c9b5a418e3adf`

Playwright MCP 8896 probe:

- Method: `mcp__playwright.browser_navigate`
- Target: `http://10.10.10.226:8896/`
- Result: failed.
- Failure: 60000ms timeout while waiting for `domcontentloaded`.
- No alternate browser surface or indirect browser workaround was attempted.

## 2026-06-19 07:24 CST: Harness State In Workspace UI

Local change under test:

- `SandboxWorkspace` now calls `SuperAgentGetHarnessState` on load and refresh.
- The workspace panel shows the persisted plan summary from `/workspace/.plan.json`.
- The workspace panel shows the current `/workspace/.agent/tooloutputs` count and can jump to that directory.
- `summarizeHarnessPlan` handles the persisted update_plan array format and missing plans.

TDD RED checks observed:

```bash
pnpm --dir frontend/packages/agent-ide/entry exec vitest --run src/modes/super-mode/__tests__/sandbox-workspace-utils.test.ts
```

Failed because `summarizeHarnessPlan` did not exist.

```bash
pnpm --dir frontend/packages/agent-ide/entry exec vitest --run src/modes/super-mode/__tests__/sandbox-workspace.test.tsx
```

Failed because `SandboxWorkspace` did not call `SuperAgentGetHarnessState` or render the harness plan/tool output summary.

GREEN verification:

```bash
pnpm --dir frontend/packages/agent-ide/entry exec vitest --run src/modes/super-mode/__tests__/sandbox-workspace-utils.test.ts src/modes/super-mode/__tests__/sandbox-workspace.test.tsx src/modes/super-mode/__tests__/super-mode.test.tsx src/modes/super-mode/__tests__/super-chat-area.test.tsx src/modes/super-mode/codex-trace/__tests__/codex-trace-panel.test.tsx
pnpm --dir frontend/packages/agent-ide/entry exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/apps/coze-studio exec tsc --noEmit -p tsconfig.json
```

Passed.

Playwright MCP 8896 probe:

- Method: `mcp__playwright.browser_navigate`
- Target: `http://10.10.10.226:8896/`
- Result: failed.
- Failure: `net::ERR_CONNECTION_TIMED_OUT` while waiting for `domcontentloaded`.
- No alternate browser surface or indirect browser workaround was attempted.

## 2026-06-19 08:35 CST: Workspace Grep/Glob App Server Endpoints

Local change under test:

- Added `POST /api/super-agent/workspace/grep`.
- Added `POST /api/super-agent/workspace/glob`.
- Manifest `workspace` contract now exposes `grep_route` and `glob_route`.
- Manifest `routes` now exposes `workspace.grep` and `workspace.glob`.
- Both endpoints are read-only and restricted to readable roots `/workspace`, `/uploads`, `/outputs`, and `/skills`.
- `grep` returns bounded text output with an `is_truncated` flag.
- `glob` returns structured path matches with an `is_truncated` flag and a capped `limit`.
- Frontend `DeveloperApi` now exposes `SuperAgentGrepWorkspace` and `SuperAgentGlobWorkspace`.

TDD RED checks observed:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent ./api/router/coze -run 'TestSearchSuperAgentWorkspaceUsesReadableRoots|TestSuperAgentWorkspaceRoutesRejectMalformedJSON|TestSuperAgentManifestRouteReturnsAppServerContract' -count=1
```

Failed because `GrepSuperAgentWorkspace`, `GlobSuperAgentWorkspace`, `GrepSandboxFilesRequest`, and `GlobSandboxFilesRequest` were missing, and the manifest did not expose `workspace.grep` or `workspace.glob`.

```bash
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-workspace.test.ts
```

Failed because `DeveloperApi.SuperAgentGrepWorkspace` was not a function.

GREEN verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent ./api/router/coze -run 'TestSearchSuperAgentWorkspaceUsesReadableRoots|TestSuperAgentWorkspaceRoutesRejectMalformedJSON|TestSuperAgentManifestRouteReturnsAppServerContract' -count=1
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-workspace.test.ts
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow ./api/router/coze ./api/router/skill ./api/handler/skill ./application/skill ./application/singleagent ./application/conversation ./domain/skill/... -count=1
pnpm --dir frontend/packages/arch/bot-api exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/agent-ide/entry exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/apps/coze-studio exec tsc --noEmit -p tsconfig.json
```

Passed.

Backend build:

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o ../bin/openynet main.go
```

Result: passed. Local SHA-256:

- `c60d16cf527bf65848fdd2b264ab27dde44737bb3a4ef4ab501ef73c56a4f506`

Playwright MCP 8896 probe:

- Method: `mcp__playwright.browser_tabs list`
- Result: passed; existing arrange tabs were visible.
- Method: `mcp__playwright.browser_tabs new`
- Target: `http://10.10.10.226:8896/api/super-agent/manifest`
- Result: failed.
- Failure: 60000ms timeout while waiting for `domcontentloaded`.
- Method: `mcp__playwright.browser_snapshot`
- Target: `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange`
- Result: passed.
- Observed page title: `演示超级体 -智能体 - 猎鹰`.
- Console state reported by Playwright MCP snapshot: 6 errors and 1 warning.
- No alternate browser surface or indirect browser workaround was attempted.

## 2026-06-19 08:40 CST: Workspace Edit App Server Endpoint

Local change under test:

- Added `POST /api/super-agent/workspace/edit`.
- Manifest `workspace` contract now exposes `edit_route=POST /api/super-agent/workspace/edit`.
- Manifest `routes` now exposes `workspace.edit`.
- The edit endpoint performs exact search-replace using the existing sandbox `EditFile` capability.
- It is restricted to writable roots `/workspace`, `/uploads`, and `/outputs`; read-only `/skills` is rejected.
- Frontend `DeveloperApi` now exposes `SuperAgentEditWorkspaceFile`.

TDD RED checks observed:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent ./api/router/coze -run 'TestEditSuperAgentWorkspaceFileUsesWritableRoots|TestSuperAgentWorkspaceRoutesRejectMalformedJSON|TestSuperAgentManifestRouteReturnsAppServerContract' -count=1
```

Failed because `EditSuperAgentWorkspaceFile` and `EditSandboxFileRequest` were missing, and the manifest did not expose `workspace.edit`.

```bash
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-workspace.test.ts
```

Failed because `DeveloperApi.SuperAgentEditWorkspaceFile` was not a function.

GREEN verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent ./api/router/coze -run 'TestEditSuperAgentWorkspaceFileUsesWritableRoots|TestSuperAgentWorkspaceRoutesRejectMalformedJSON|TestSuperAgentManifestRouteReturnsAppServerContract' -count=1
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-workspace.test.ts
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow ./api/router/coze ./api/router/skill ./api/handler/skill ./application/skill ./application/singleagent ./application/conversation ./domain/skill/... -count=1
pnpm --dir frontend/packages/arch/bot-api exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/agent-ide/entry exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/apps/coze-studio exec tsc --noEmit -p tsconfig.json
```

Passed.

Backend build:

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o ../bin/openynet main.go
```

Result: passed. Local SHA-256:

- `3261cd3ca14e78ebbde906d7af8313a44161577411f623809effe4e19f63b592`

Playwright MCP 8896 probe:

- Method: `mcp__playwright.browser_tabs list`
- Result: passed; existing arrange tabs were visible.
- Method: `mcp__playwright.browser_tabs new`
- Target: `http://10.10.10.226:8896/api/super-agent/manifest`
- Result: failed.
- Failure: 60000ms timeout while waiting for `domcontentloaded`.
- Method: `mcp__playwright.browser_snapshot`
- Target: `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange`
- Result: passed.
- Observed page title: `演示超级体 -智能体 - 猎鹰`.
- Console state reported by Playwright MCP snapshot: 6 errors and 1 warning.
- No alternate browser surface or indirect browser workaround was attempted.

## 2026-06-19 08:28 CST: Workspace Stat App Server Endpoint

Local change under test:

- Added `POST /api/super-agent/workspace/stat`.
- Manifest `workspace` contract now exposes `stat_route=POST /api/super-agent/workspace/stat`.
- Manifest `routes` now exposes `workspace.stat`.
- The stat service reads metadata only and does not read file content.
- Stat is allowed on readable roots `/workspace`, `/uploads`, `/outputs`, and read-only `/skills`.
- Frontend `DeveloperApi` now exposes `SuperAgentStatWorkspacePath`.

TDD RED checks observed:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent ./api/router/coze -run 'TestStatSuperAgentWorkspacePathReadsMetadataFromReadableRoots|TestSuperAgentWorkspaceRoutesRejectMalformedJSON|TestSuperAgentManifestRouteReturnsAppServerContract' -count=1
```

Failed because `StatSuperAgentWorkspacePath` and `StatSandboxFileRequest` were missing, the manifest did not expose `workspace.stat`, and `/api/super-agent/workspace/stat` was not registered.

```bash
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-workspace.test.ts
```

Failed because `DeveloperApi.SuperAgentStatWorkspacePath` was not a function.

GREEN verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent ./api/router/coze -run 'TestStatSuperAgentWorkspacePathReadsMetadataFromReadableRoots|TestSuperAgentWorkspaceRoutesRejectMalformedJSON|TestSuperAgentManifestRouteReturnsAppServerContract' -count=1
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-workspace.test.ts
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow ./api/router/coze ./api/router/skill ./api/handler/skill ./application/skill ./application/singleagent ./application/conversation ./domain/skill/... -count=1
pnpm --dir frontend/packages/arch/bot-api exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/agent-ide/entry exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/apps/coze-studio exec tsc --noEmit -p tsconfig.json
```

Passed.

Backend build:

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o ../bin/openynet main.go
```

Result: passed. Local SHA-256:

- `876ae7fc46787f0c6c006dbd44e2890ac175510761835adfbf4739a275dcff7c`

Playwright MCP 8896 probe:

- Method: `mcp__playwright.browser_tabs list`
- Result: passed; existing arrange tabs were visible.
- Method: `mcp__playwright.browser_tabs new`
- Target: `http://10.10.10.226:8896/api/super-agent/manifest`
- Result: failed.
- Failure: 60000ms timeout while waiting for `domcontentloaded`.
- Method: `mcp__playwright.browser_snapshot`
- Target: `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange`
- Result: passed.
- Observed page title: `演示超级体 -智能体 - 猎鹰`.
- Console state reported by Playwright MCP snapshot: 6 errors and 1 warning.
- No alternate browser surface or indirect browser workaround was attempted.

## 2026-06-19 08:22 CST: Updated Goal Playwright MCP Regression

Goal refresh:

- Confirmed the active goal now explicitly requires Playwright MCP testing for the sandbox + super-agent work.
- The goal remains focused on Codex/Hermes-like super-agent behavior, App Server integration, standard folder-based skills, publish scopes, global skill marketplace, harness, artifacts/assets, coding/product generation, and better frontend presentation.

Playwright MCP 8896 probe:

- Method: `mcp__playwright.browser_resize`
- Target: active Playwright MCP browser window.
- Result: failed.
- Failure: 30000ms timeout in `browserBackend.callTool`.

Playwright MCP loaded page check:

- Method: `mcp__playwright.browser_tabs list`, then `mcp__playwright.browser_tabs select`, then `mcp__playwright.browser_snapshot`.
- Target: `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange`
- Result: partially passed.
- Observed page title: `演示超级体 -智能体 - 猎鹰`.
- Observed UI: workspace, sandbox preview/debug area, and capability strip showing autonomous planning, isolated sandbox, skills & MCP, and long-term memory.
- Console state reported by Playwright MCP snapshot: 6 errors and 1 warning.
- Screenshot artifact from Playwright MCP: `super-agent-8896-arrange-20260619-mcp.png`.

Playwright MCP App Server manifest probe:

- Method: `mcp__playwright.browser_tabs new`
- Target: `http://10.10.10.226:8896/api/super-agent/manifest`
- Result: failed.
- Failure: 60000ms timeout while waiting for `domcontentloaded`.

Follow-up Playwright MCP state check:

- Method: `mcp__playwright.browser_tabs list`
- Result: failed.
- Failure: 30000ms timeout in `browserBackend.callTool`.
- No alternate browser surface or indirect browser workaround was attempted.

## 2026-06-19 07:39 CST: Runs Get App Server Endpoint

Local change under test:

- Added `POST /api/super-agent/runs/get` for external callers to query active App Server run state by `run_id`.
- The active run registry now stores `run_id`, `status=in_progress`, `created_at`, `updated_at`, and the cancel function.
- `POST /api/super-agent/runs/cancel` keeps its existing idempotent behavior and now reads the cancel function from the richer registry entry.
- Manifest `routes` now exposes `runs.get`.
- Frontend `DeveloperApi` now exposes `SuperAgentGetRun`.

TDD RED checks observed:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/conversation ./api/router/coze -run 'TestActiveAgentRunRegistryReportsStatus|TestSuperAgentGetRunRouteRejectsMissingRunID|TestSuperAgentGetRunRouteReportsInactiveRun|TestSuperAgentGetRunRouteReportsActiveRun|TestSuperAgentManifestRouteReturnsAppServerContract' -count=1
```

Failed because `GetActiveAgentRun` was missing and manifest did not advertise `runs.get`.

```bash
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-runs.test.ts
```

Failed because `DeveloperApi.SuperAgentGetRun` was not a function.

GREEN verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow ./api/router/coze ./api/router/skill ./api/handler/skill ./application/skill ./application/singleagent ./application/conversation ./domain/skill/... -count=1
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-runs.test.ts __tests__/developer-api-super-agent-workspace.test.ts __tests__/developer-api-super-agent-artifacts.test.ts __tests__/developer-api-super-agent-skills.test.ts
pnpm --dir frontend/packages/arch/bot-api exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/apps/coze-studio exec tsc --noEmit -p tsconfig.json
```

Passed.

Backend build:

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o ../bin/openynet main.go
```

Result: passed. Local SHA-256:

- `d5742fed43037c484c4d402ed6b017a89a5a3d244d00b29c364a73084674ffaa`

Playwright MCP 8896 probe:

- Method: `mcp__playwright.browser_navigate`
- Target: `http://10.10.10.226:8896/`
- Result: failed.
- Failure: `net::ERR_CONNECTION_TIMED_OUT` while waiting for `domcontentloaded`.
- No alternate browser surface or indirect browser workaround was attempted.

## 2026-06-19 07:47 CST: Runs List App Server Endpoint

Local change under test:

- Added `POST /api/super-agent/runs/list` for external callers to list lightweight run history for a conversation.
- The route validates `conversation_id`, reuses the existing conversation permission check, and reads persisted run records through the existing run domain service.
- Run list items include `run_id`, `conversation_id`, `agent_id`, `status`, `active`, error, and lifecycle timestamps.
- Active run records are marked `active=true` and surfaced as `status=in_progress` using the same active-run registry behind `runs/get` and `runs/cancel`.
- Manifest `routes` now exposes `runs.list`.
- Frontend `DeveloperApi` now exposes `SuperAgentListRuns`.

TDD RED checks observed:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgentListRunsRouteRejectsMissingConversationID|TestBuildSuperAgentRunListDataMarksActiveRuns|TestSuperAgentManifestRouteReturnsAppServerContract' -count=1
```

Failed because `buildSuperAgentRunListData` and `runs.list` were missing. The helper assertion was moved out of the router package after it exposed an existing handler test import cycle; the route and manifest red checks remain in the router package.

```bash
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-runs.test.ts
```

Failed because `DeveloperApi.SuperAgentListRuns` was not a function.

GREEN verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow ./api/router/coze ./api/router/skill ./api/handler/skill ./application/skill ./application/singleagent ./application/conversation ./domain/skill/... -count=1
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-runs.test.ts __tests__/developer-api-super-agent-workspace.test.ts __tests__/developer-api-super-agent-artifacts.test.ts __tests__/developer-api-super-agent-skills.test.ts
pnpm --dir frontend/packages/arch/bot-api exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/apps/coze-studio exec tsc --noEmit -p tsconfig.json
```

Passed.

Backend build:

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o ../bin/openynet main.go
```

Result: passed. Local SHA-256:

- `1bbf35adedd8dbe54f10e7ed5634aac96e644989eb589fa1462cd5463a4db562`

Playwright MCP 8896 probe:

- Method: `mcp__playwright.browser_navigate`
- Target: `http://10.10.10.226:8896/`
- Result: failed.
- Failure: 60000ms timeout while waiting for `domcontentloaded`.
- No alternate browser surface or indirect browser workaround was attempted.

## 2026-06-19 07:53 CST: Artifact Move App Server Endpoint

Local change under test:

- Added `POST /api/super-agent/artifacts/move` for moving or renaming generated deliverables under `/outputs`.
- The service accepts `artifact_id` or `path` plus `target_path`, creates the target directory, and runs `mv` inside the sandbox.
- Source and target are both restricted to `/outputs`; moving `/outputs` itself is rejected.
- Manifest `artifacts` contract now exposes `move_route=POST /api/super-agent/artifacts/move`.
- Manifest `routes` now exposes `artifacts.move`.
- Frontend `DeveloperApi` now exposes `SuperAgentMoveArtifact`.

TDD RED checks observed:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent ./api/router/coze -run 'TestMoveSuperAgentArtifactRenamesOnlyOutputsPath|TestSuperAgentArtifactRoutesRejectMalformedJSON|TestSuperAgentManifestRouteReturnsAppServerContract' -count=1
```

Failed because `MoveSuperAgentArtifact` and `SuperAgentArtifactMoveRequest` were missing, and the manifest did not advertise `artifacts.move`.

```bash
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-artifacts.test.ts
```

Failed because `DeveloperApi.SuperAgentMoveArtifact` was not a function.

GREEN verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow ./api/router/coze ./api/router/skill ./api/handler/skill ./application/skill ./application/singleagent ./application/conversation ./domain/skill/... -count=1
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-runs.test.ts __tests__/developer-api-super-agent-workspace.test.ts __tests__/developer-api-super-agent-artifacts.test.ts __tests__/developer-api-super-agent-skills.test.ts
pnpm --dir frontend/packages/arch/bot-api exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/apps/coze-studio exec tsc --noEmit -p tsconfig.json
```

Passed.

Backend build:

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o ../bin/openynet main.go
```

Result: passed. Local SHA-256:

- `0976281e1c3b13c0c3730611b3b6d35c485039a1226221fcf037b55037e5ccb6`

Playwright MCP 8896 probe:

- Method: `mcp__playwright.browser_navigate`
- Target: `http://10.10.10.226:8896/`
- Result: failed.
- Failure: `net::ERR_CONNECTION_TIMED_OUT` while waiting for `domcontentloaded`.
- No alternate browser surface or indirect browser workaround was attempted.

## 2026-06-19 07:32 CST: Artifact Delete Wired Into Workspace UI

Local change under test:

- The workspace artifact catalog delete action now calls `SuperAgentDeleteArtifact`.
- Generic workspace files still use `SuperAgentDeleteWorkspaceFile`.
- The UI regression test switches to `产出物`, renders an artifact catalog item, clicks its delete action, and verifies the dedicated App Server artifact endpoint is used.

TDD RED check observed:

```bash
pnpm --dir frontend/packages/agent-ide/entry exec vitest --run src/modes/super-mode/__tests__/sandbox-workspace.test.tsx
```

Failed because clicking delete on an artifact catalog item did not call `SuperAgentDeleteArtifact`.

GREEN verification:

```bash
pnpm --dir frontend/packages/agent-ide/entry exec vitest --run src/modes/super-mode/__tests__/sandbox-workspace-utils.test.ts src/modes/super-mode/__tests__/sandbox-workspace.test.tsx src/modes/super-mode/__tests__/super-mode.test.tsx src/modes/super-mode/__tests__/super-chat-area.test.tsx src/modes/super-mode/codex-trace/__tests__/codex-trace-panel.test.tsx
pnpm --dir frontend/packages/agent-ide/entry exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-artifacts.test.ts
pnpm --dir frontend/packages/arch/bot-api exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/apps/coze-studio exec tsc --noEmit -p tsconfig.json
```

Passed.

Backend build:

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o ../bin/openynet main.go
```

Result: passed. Local SHA-256:

- `fe76a321e6a1920068d47207e133a8cfa60d57753ff348779d2f5c3a1c3bb731`

Playwright MCP 8896 probe:

- Method: `mcp__playwright.browser_navigate`
- Target: `http://10.10.10.226:8896/`
- Result: failed.
- Failure: 60000ms timeout while waiting for `domcontentloaded`.
- No alternate browser surface or indirect browser workaround was attempted.

## 2026-06-19 07:29 CST: Artifact Delete App Server Endpoint

Local change under test:

- Added `POST /api/super-agent/artifacts/delete`.
- Manifest `artifacts` contract now exposes `delete_route=POST /api/super-agent/artifacts/delete`.
- Manifest `routes` now exposes `artifacts.delete`.
- The delete service accepts `artifact_id` or `path`, resolves it under `/outputs`, rejects non-output paths, and refuses deleting `/outputs` itself.
- Frontend `DeveloperApi` now exposes `SuperAgentDeleteArtifact`.

TDD RED checks observed:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent ./api/router/coze -run 'TestDeleteSuperAgentArtifactRemovesOnlyOutputsPath|TestSuperAgentArtifactRoutesRejectMalformedJSON|TestSuperAgentManifestRouteReturnsAppServerContract' -count=1
```

Failed because `DeleteSuperAgentArtifact` and `SuperAgentArtifactDeleteRequest` were missing, the manifest did not expose `artifacts.delete`, and `/api/super-agent/artifacts/delete` was not registered.

```bash
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-artifacts.test.ts
```

Failed because `DeveloperApi.SuperAgentDeleteArtifact` was not a function.

GREEN verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow ./api/router/coze ./api/router/skill ./api/handler/skill ./application/skill ./application/singleagent ./domain/skill/... -count=1
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-workspace.test.ts __tests__/developer-api-super-agent-artifacts.test.ts __tests__/developer-api-super-agent-skills.test.ts
pnpm --dir frontend/packages/arch/bot-api exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/arch/api-schema test -- --run __tests__/skill-super-agent.test.ts
pnpm --dir frontend/packages/arch/api-schema exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/apps/coze-studio exec tsc --noEmit -p tsconfig.json
```

Passed.

Backend build:

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o ../bin/openynet main.go
```

Result: passed. Local SHA-256:

- `fe76a321e6a1920068d47207e133a8cfa60d57753ff348779d2f5c3a1c3bb731`

Playwright MCP 8896 probe:

- Method: `mcp__playwright.browser_navigate`
- Target: `http://10.10.10.226:8896/`
- Result: failed.
- Failure: `net::ERR_CONNECTION_TIMED_OUT` while waiting for `domcontentloaded`.
- No alternate browser surface or indirect browser workaround was attempted.

## 2026-06-19 08:49 CST: Sandbox Exec App Server Endpoint

Local change under test:

- Added `POST /api/super-agent/sandbox/exec`.
- Manifest `sandbox` contract now exposes `exec_route=POST /api/super-agent/sandbox/exec`, `default_workdir=/workspace`, `max_timeout_sec=300`, and `output_max_bytes=65536`.
- Manifest `routes` now exposes `sandbox.exec`.
- The service validates non-empty commands, limits `workdir` to `/workspace`, `/uploads`, `/outputs`, or `/skills`, defaults timeout to 60s, caps timeout at 300s, and truncates stdout/stderr to 64KB.
- Frontend `DeveloperApi` now exposes `SuperAgentExecSandbox`.

TDD RED checks observed:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent ./api/router/coze -run 'TestRunSuperAgentSandboxCommandExecutesInReadableWorkdir|TestSuperAgentSandboxRoutesRejectMalformedJSON|TestSuperAgentManifestRouteReturnsAppServerContract' -count=1
```

Failed because `RunSuperAgentSandboxCommand` and `ExecSandboxCommandRequest` were missing, the manifest did not expose `sandbox.exec`, and `/api/super-agent/sandbox/exec` was not registered.

```bash
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-workspace.test.ts
```

Failed because `DeveloperApi.SuperAgentExecSandbox` was not a function.

GREEN verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent ./api/router/coze -run 'TestRunSuperAgentSandboxCommandExecutesInReadableWorkdir|TestSuperAgentSandboxRoutesRejectMalformedJSON|TestSuperAgentManifestRouteReturnsAppServerContract' -count=1
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-workspace.test.ts
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow ./api/router/coze ./api/router/skill ./api/handler/skill ./application/skill ./application/singleagent ./application/conversation ./domain/skill/... -count=1
pnpm --dir frontend/packages/arch/bot-api exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/agent-ide/entry exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/apps/coze-studio exec tsc --noEmit -p tsconfig.json
```

Passed.

Backend build:

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o ../bin/openynet main.go
```

Result: passed. Local SHA-256:

- `ed00696b5e968f0b3183d8e5cdcaa137c013afd36acad4b05a6d97a4a5894c48`

Playwright MCP 8896 probe:

- `mcp__playwright.browser_tabs list`: passed; existing arrange tabs were visible.
- `mcp__playwright.browser_tabs new` to `http://10.10.10.226:8896/api/super-agent/manifest`: failed with 60000ms navigation timeout while waiting for `domcontentloaded`.
- `mcp__playwright.browser_tabs close`: passed; closed the loading/error tab.
- `mcp__playwright.browser_snapshot` on the existing arrange page: passed; title was `演示超级体 -智能体 - 猎鹰`; visible areas included `FinMallClaw 能力` and `人设 · 技能 · MCP`.
- `mcp__playwright.browser_console_messages`: passed; current deployed backend returned 404 for `/api/super-agent/manifest`, `/api/super-agent/runs/create`, `/api/super-agent/workspace/list`, and `/api/super-agent/runs/cancel`.
- `mcp__playwright.browser_network_requests`: passed; confirmed `/api/super-agent/manifest` returned 404 from Hertz on the currently deployed service.
- `mcp__playwright.browser_take_screenshot`: passed; saved viewport screenshot as `super-agent-arrange-8896-sandbox-exec.png` in the Playwright MCP output area.

Conclusion: local code is verified, but 8896 is still serving a deployment without the super-agent App Server routes, so remote endpoint verification remains blocked until that service is replaced or restarted with the new build.

Remote service replacement probe:

- Method: read-only SSH probe to `dev@10.10.10.226`.
- Result: failed.
- Failure: port 22 timed out before authentication.
- Effect: no remote files or services were modified in this step.

## 2026-06-19 14:30 CST: Standard Skill ZIP Export

Local change under test:

- Added `POST /api/super-agent/skills/export` for standard skill ZIP download.
- Export returns `data:application/zip;base64,...` with `filename`, `size`, and `file_paths`.
- Owned skills export the current draft package; marketplace-visible skills from another space export the published snapshot.
- ZIP generation preserves the standard folder shape: `SKILL.md`, `scripts/`, `references/`, `templates/`, and `assets/`.
- Asset data URLs are decoded back to binary files before writing into the ZIP.
- Manifest and DeveloperApi now expose `skills.export`.
- Skill center cards now support `下载 ZIP` for reusable standard packages.

TDD RED checks observed:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/skill ./api/router/coze -run 'TestSkillApplicationServiceExportsStandardSkillZipPackage|TestSkillApplicationServiceExportsPublishedMarketplaceSnapshot|TestSuperAgentManifestRouteReturnsAppServerContract|TestSuperAgentSkillRoutesRejectMalformedJSON' -count=1
pnpm --dir frontend/packages/arch/api-schema test -- --run __tests__/skill-super-agent.test.ts
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-skills.test.ts
pnpm --dir frontend/apps/coze-studio test -- --run src/pages/space-skill/__tests__/index.test.tsx
```

Failed because `ExportSkillPackage`, the `/api/super-agent/skills/export` route, schema wrappers, DeveloperApi wrapper, and `下载 ZIP` card action were missing.

GREEN verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/skill ./api/router/coze -run 'TestSkillApplicationServiceExportsStandardSkillZipPackage|TestSkillApplicationServiceExportsPublishedMarketplaceSnapshot|TestSuperAgentManifestRouteReturnsAppServerContract|TestSuperAgentSkillRoutesRejectMalformedJSON' -count=1
SESSION_HMAC_SECRET=test-secret go test ./application/skill ./api/router/coze
pnpm --dir frontend/packages/arch/api-schema test -- --run __tests__/skill-super-agent.test.ts
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-skills.test.ts
pnpm --dir frontend/apps/coze-studio test -- --run src/pages/space-skill/__tests__/index.test.tsx src/pages/space-skill/__tests__/standard-skill-files.test.ts
pnpm --dir frontend/apps/coze-studio exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/arch/api-schema exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/arch/bot-api exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/agent-ide/entry exec tsc --noEmit -p tsconfig.json
```

Passed.

Production build:

```bash
pnpm --dir frontend/apps/coze-studio build
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o ../bin/openynet main.go
```

Passed. Build warnings were limited to existing Browserslist/Baseline data age notices.

Deployment to `10.10.10.226:8896`:

- Backend binary SHA-256: `3dfb693685ba88808789d8b2add2f753d79e1f265911f0502ede2d2f91790bb2`
- Frontend static tar SHA-256: `3916b888aafbd302d8e0702479db9711362334256ee952aaa884563d3b7640d7`
- Container backup: `/app/openynet.bak-skill-export-20260619-142426`
- Container backup: `/app/resources/static.bak-skill-export-20260619-142426`
- Restart result: `coze-super Up 7 seconds`

Post-deploy manifest probe:

```bash
curl -sS http://10.10.10.226:8896/api/super-agent/manifest | jq '{skillExport: .data.routes["skills.export"], skillRoute: .data.skills.routes.export, exportSchema: .data.skills.request_schemas.export}'
```

Result:

```json
{
  "skillExport": "POST /api/super-agent/skills/export",
  "skillRoute": "POST /api/super-agent/skills/export",
  "exportSchema": {
    "required": ["space_id", "skill_id"],
    "optional": []
  }
}
```

Playwright MCP 8896 verification:

- `mcp__playwright.browser_navigate` to `http://10.10.10.226:8896/space/7652614054615187456/skills?mcp_check=skill_export_20260619`: passed.
- `mcp__playwright.browser_take_screenshot`: passed; saved `super-agent-playwright-mcp-8896-skill-export-list-final.png`.
- `mcp__playwright.browser_evaluate`: passed; created temporary standard skill `mcp-export-1781850641927`, exported ZIP through `POST /api/super-agent/skills/export`, then deleted the temporary skill.
- Export result: `status=200`, `code=0`, `filename=mcp-export-1781850641927.zip`, `file_paths=["SKILL.md","assets/logo.png","references/guide.md","scripts/run.sh"]`, cleanup delete `status=200`, `code=0`.
- Saved raw result: `super-agent-playwright-mcp-8896-skill-export-final.json`.
- Saved redacted result: `super-agent-playwright-mcp-8896-skill-export-final-redacted.json`.
- Saved compact result: `super-agent-playwright-mcp-8896-skill-export-final.compact.json`.
- Saved network/page snapshot: `super-agent-playwright-mcp-8896-skill-export-network-final.json`.
- Saved unzip listing: `super-agent-playwright-mcp-8896-skill-export-unzip-final.txt`.

ZIP content verification:

```text
mcp-export-1781850641927/SKILL.md
mcp-export-1781850641927/assets/logo.png
mcp-export-1781850641927/references/guide.md
mcp-export-1781850641927/scripts/run.sh
```

- `SKILL.md` contained the expected frontmatter and `# Export Verification`.
- `scripts/run.sh` contained `echo export-ok`.
- `references/guide.md` contained `export guide`.
- `assets/logo.png` decoded to binary bytes with hex prefix `89504e470d0a1a0a`.

## 2026-06-19 14:49 CST: Standard Skill ZIP Preflight Validation

Local change under test:

- Added `POST /api/super-agent/skills/validate-package` for standard skill ZIP preflight checks.
- Validation reuses the same ZIP decoding and standard-folder rules as import, so upload precheck and import cannot drift.
- Valid packages return parsed `name`, `description`, metadata, sorted `file_paths`, `asset_paths`, and `image_paths`.
- Invalid packages return `valid=false` with a clean user-facing error, without backend stack traces.
- Manifest, api-schema, and DeveloperApi now expose `skills.validate_package`.
- Skill center ZIP upload now validates the package before importing it; invalid packages stop before import and show the validation reason.

TDD RED checks observed:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/skill ./api/router/coze -run 'TestSkillApplicationServiceValidatesStandardSkillZipPackage|TestSkillApplicationServiceValidationReportsInvalidSkillZipPackage|TestSuperAgentManifestRouteReturnsAppServerContract|TestSuperAgentSkillRoutesRejectMalformedJSON' -count=1
pnpm --dir frontend/packages/arch/api-schema test -- --run __tests__/skill-super-agent.test.ts
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-skills.test.ts
pnpm --dir frontend/apps/coze-studio test -- --run src/pages/space-skill/__tests__/index.test.tsx
```

Failed because `ValidateSkillPackage`, `/api/super-agent/skills/validate-package`, schema wrappers, DeveloperApi wrapper, and skill-center upload preflight were missing.

Additional RED regression:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/skill -run 'TestSkillApplicationServiceValidationReportsInvalidSkillZipPackage' -count=1
```

Failed because invalid validation errors contained `stack=...`.

GREEN verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/skill ./api/router/coze -run 'TestSkillApplicationServiceValidatesStandardSkillZipPackage|TestSkillApplicationServiceValidationReportsInvalidSkillZipPackage|TestSuperAgentManifestRouteReturnsAppServerContract|TestSuperAgentSkillRoutesRejectMalformedJSON' -count=1
SESSION_HMAC_SECRET=test-secret go test ./application/skill ./api/router/coze
pnpm --dir frontend/packages/arch/api-schema test -- --run __tests__/skill-super-agent.test.ts
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-skills.test.ts
pnpm --dir frontend/apps/coze-studio test -- --run src/pages/space-skill/__tests__/index.test.tsx src/pages/space-skill/__tests__/standard-skill-files.test.ts
pnpm --dir frontend/apps/coze-studio exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/arch/api-schema exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/arch/bot-api exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/agent-ide/entry exec tsc --noEmit -p tsconfig.json
```

Passed.

Production build:

```bash
pnpm --dir frontend/apps/coze-studio build
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o ../bin/openynet main.go
```

Passed. Build warnings were limited to existing Browserslist/Baseline data age notices.

Deployment to `10.10.10.226:8896`:

- First full deploy backend SHA-256: `25af4f78c4cfcf17aaf83a462c19e2901df6adace17af89b7df0a7da10d71555`
- First full deploy frontend static tar SHA-256: `0b35d68327fb7f161e7268a25a2c6c8fd0032259c2b77151654a4e14300368f9`
- Container backup: `/app/openynet.bak-skill-validate-20260619-144141`
- Container backup: `/app/resources/static.bak-skill-validate-20260619-144141`
- Follow-up backend-only cleanup SHA-256: `4f21e9d9344090fbe075ea872b37e30e95d9a23c9521961288cd9850a768e3ea`
- Container backup: `/app/openynet.bak-skill-validate-error-clean-20260619-144720`
- Restart result: `coze-super Up 6 seconds`

Post-deploy manifest probe:

```bash
curl -sS http://10.10.10.226:8896/api/super-agent/manifest | jq -e '.data.routes["skills.validate_package"] == "POST /api/super-agent/skills/validate-package"'
```

Result: `true`.

Playwright MCP 8896 verification:

- `mcp__playwright.browser_navigate` to `http://10.10.10.226:8896/space/7652614054615187456/skills?mcp_check=skill_validate_20260619`: passed.
- `mcp__playwright.browser_take_screenshot`: passed; saved `super-agent-playwright-mcp-8896-skill-validate-list-final.png`.
- `mcp__playwright.browser_evaluate`: passed; browser-generated a valid ZIP and an invalid ZIP, then POSTed both to `/api/super-agent/skills/validate-package`.
- Saved first result: `super-agent-playwright-mcp-8896-skill-validate-final.json`.
- Saved cleaned final result: `super-agent-playwright-mcp-8896-skill-validate-clean-final.json`.
- Saved network/page snapshot: `super-agent-playwright-mcp-8896-skill-validate-network-final.json`.

Cleaned final result assertions:

```json
{
  "valid_status": 200,
  "valid": true,
  "valid_name": "validate-tools",
  "files": ["SKILL.md", "assets/logo.png", "references/guide.md", "scripts/run.sh"],
  "invalid": false,
  "invalid_error": "invalid parameter : SKILL.md is required"
}
```

Local assertion command:

```bash
jq -e '.valid.status == 200 and .valid.code == 0 and .valid.validation.valid == true and .valid.validation.name == "validate-tools" and (.valid.validation.file_paths | index("SKILL.md")) != null and (.valid.validation.image_paths | index("assets/logo.png")) != null and .invalid.status == 200 and .invalid.code == 0 and .invalid.validation.valid == false and (.invalid.validation.error | contains("SKILL.md is required")) and (.invalid.validation.error | contains("stack=") | not) and (.invalid.validation.error | contains("\n") | not)' super-agent-playwright-mcp-8896-skill-validate-clean-final.json
```

Result: `true`.

## 2026-06-19 14:06 CST: Standard Skill ZIP Package Import

Local change under test:

- Added `POST /api/super-agent/skills/import` for standard skill ZIP package import.
- Server decodes data-url/base64 ZIP payloads, strips one common package root directory, validates the standard skill layout, rejects path traversal and non-standard file roots, and creates the skill from `SKILL.md` frontmatter.
- Standard roots remain `SKILL.md`, `scripts/`, `references/`, `templates/`, and `assets/`.
- Image assets under `assets/` are stored as `data:image/*;base64,...` so the skill center can preview uploaded package assets.
- Manifest now exposes `routes["skills.import"]`, `skills.routes.import`, and request schema `required=["space_id","content"]`, `optional=["filename","icon_uri"]`.
- Skill center now shows an `上传 ZIP` entry next to `创建标准技能`; the frontend reads a `.zip` file as a data URL and calls the super-agent import endpoint.

TDD RED checks observed:

```bash
env SESSION_HMAC_SECRET=test-secret go test ./application/skill -run 'TestSkillApplicationServiceRejectsInvalidSkillZipPackage' -count=1
```

Failed before the root-folder refinement because a ZIP containing `missing-root/scripts/run.py` but no `SKILL.md` returned `invalid skill file path` instead of the clearer `SKILL.md is required`.

```bash
pnpm --dir frontend/packages/arch/api-schema test -- --run __tests__/skill-super-agent.test.ts
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-skills.test.ts
pnpm --dir frontend/apps/coze-studio test -- --run src/pages/space-skill/__tests__/index.test.tsx
```

Failed before implementation because `SuperAgentImportSkillPackage` and the visible `上传 ZIP` entry did not exist.

GREEN verification:

```bash
env SESSION_HMAC_SECRET=test-secret go test ./application/skill ./api/router/coze
pnpm --dir frontend/apps/coze-studio test -- --run src/pages/space-skill/__tests__/index.test.tsx src/pages/space-skill/__tests__/standard-skill-files.test.ts
pnpm --dir frontend/packages/arch/api-schema test -- --run __tests__/skill-super-agent.test.ts
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-skills.test.ts
pnpm --dir frontend/apps/coze-studio exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/arch/api-schema exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/arch/bot-api exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/agent-ide/entry exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/apps/coze-studio build
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o ../bin/openynet main.go
```

Passed.

Deployment to `10.10.10.226:8896`:

- Frontend static package: `/tmp/coze-studio-dist-20260619-skill-zip.tgz`
- Frontend static package SHA-256: `d428be623bef284cd65ca9c7c5a911b36813b02e602e7eae14cd67883d8d9ca9`
- Backend final binary SHA-256: `b5307e7624501e37696ad3ca586ffe5a4b69342a74f3efddb76fd7fd7b2f2d9c`
- Static backup in container: `/app/resources/static.bak-skill-zip-20260619055620`
- Backend backups in container: `/app/openynet.bak-skill-zip-20260619055620`, `/app/openynet.bak-skill-zip-fix-20260619060615`
- Health check: `GET /api/super-agent/manifest` returned `POST /api/super-agent/skills/import` for both `data.routes["skills.import"]` and `data.skills.routes.import`.

Playwright MCP 8896 verification:

- Page: `http://10.10.10.226:8896/space/7652614054615187456/skills?mcp_check=skill_zip_20260619`
- Screenshot: `super-agent-playwright-mcp-8896-skill-zip-list-final.png`
- Snapshot: `super-agent-playwright-mcp-8896-skill-zip-list-snapshot.md`
- Positive ZIP import result: `super-agent-playwright-mcp-8896-skill-zip-after-fix-final.json`
  - Imported `mcp-standard-skill/SKILL.md`, `scripts/run.sh`, `references/guide.md`, `templates/report.md`, and `assets/logo.png`.
  - Import returned `code=0`, file paths were stored without the package root, `asset_summary.count=1`, `logo_is_data_url=true`, list lookup hit the imported skill, and cleanup delete returned `code=0`.
- Negative ZIP validation result: `super-agent-playwright-mcp-8896-skill-zip-reject-after-fix-final.json`
  - Missing `SKILL.md` package returned app code `111000000` with message `invalid parameter : SKILL.md is required`.
- Console log: `super-agent-playwright-mcp-8896-skill-zip-console-after-fix-final.txt`
  - `Errors: 0`, `Warnings: 1`; the only warning was an existing `single-spa minified message #1`.
- Network log: `super-agent-playwright-mcp-8896-skill-zip-network-after-fix-final.txt`
  - Confirmed `GET /api/super-agent/manifest`, `POST /api/super-agent/skills/import`, `GET /api/super-agent/skills/list`, and `POST /api/super-agent/skills/delete` all returned HTTP 200 during the positive path.

## 2026-06-19 13:20 CST: Skill Asset Upsert Contract

Local change under test:

- Added the standard skill asset upsert surface: `POST /api/super-agent/skills/assets/upsert`.
- Assets are constrained to the standard skill package `assets/` directory.
- The skill package still requires `SKILL.md`; uploading an asset preserves existing standard files.
- Skill responses now include `asset_summary.count` and `asset_summary.image_paths` for UI previews.
- Manifest `skills.routes`, top-level `routes`, `skills.request_schemas`, and `skills.assets` now expose the asset upload contract.
- Frontend `DeveloperApi` now exposes `SuperAgentUpsertSkillAsset` for the future skill-center UI.

TDD RED checks observed:

```bash
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-skills.test.ts
```

Failed because `DeveloperApi.SuperAgentUpsertSkillAsset` was not a function.

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/skill ./api/handler/skill ./api/router/coze -run 'TestSkillApplicationServiceUpsertsSkillAssetIntoStandardFiles|TestSkillApplicationServiceRejectsInvalidSkillAssetPath|TestEntityToResponseIncludesSkillAssetSummary|TestSuperAgentManifestRouteReturnsAppServerContract|TestSuperAgentSkillRoutesRejectMalformedJSON' -count=1
```

Failed before implementation because `UpsertSkillAsset`, `asset_summary`, and the `/api/super-agent/skills/assets/upsert` route were missing.

GREEN verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/skill ./api/handler/skill ./api/router/coze -run 'TestSkillApplicationServiceUpsertsSkillAssetIntoStandardFiles|TestSkillApplicationServiceRejectsInvalidSkillAssetPath|TestEntityToResponseIncludesSkillAssetSummary|TestSuperAgentManifestRouteReturnsAppServerContract|TestSuperAgentSkillRoutesRejectMalformedJSON' -count=1
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow ./api/router/coze ./api/router/skill ./api/handler/skill ./application/skill ./application/singleagent ./application/conversation ./domain/skill/... ./api/middleware -count=1
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-skills.test.ts
pnpm --dir frontend/packages/arch/bot-api exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/agent-ide/entry exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/apps/coze-studio exec tsc --noEmit -p tsconfig.json
```

Passed.

Backend build:

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o ../bin/openynet main.go
```

Result: passed. Local and remote SHA-256:

- `96268dc1dca7bfee0b52be6e7653dd0c962b75883fab16ed55f7e75e1ce88eed`

Remote service replacement:

- Uploaded `/home/dev/openynet-next.upload`.
- Replaced `coze-super:/app/openynet`.
- Restarted `coze-super`.
- Backup saved at `/home/dev/openynet.coze-super.bak-20260619051728`.

Playwright MCP 8896 probe:

- Page probe saved to `super-agent-playwright-mcp-8896-skill-assets-page-probe.json`.
- Contract and create/upload/read/delete E2E saved to `super-agent-playwright-mcp-8896-skill-assets-final.json`.
- Invalid asset path rejection probe saved to `super-agent-playwright-mcp-8896-skill-assets-reject-final.json`.
- Console log saved to `super-agent-playwright-mcp-8896-skill-assets-console-final.txt`.
- Network log saved to `super-agent-playwright-mcp-8896-skill-assets-network-final.txt`.
- Screenshot saved to `super-agent-playwright-mcp-8896-skill-assets-screenshot-final.png`.

Observed results:

- `GET /api/super-agent/manifest` returned code `0`.
- Manifest exposes `skills.routes["assets.upsert"] = POST /api/super-agent/skills/assets/upsert`.
- Manifest exposes required request fields `space_id`, `skill_id`, `path`, `content` and optional field `mime`.
- Manifest exposes `skills.assets.root = assets/`, `content_encoding = utf8-or-data-url`, and image MIME support including `image/png`.
- Temporary standard skill create succeeded.
- Uploading `assets/logo.png` succeeded and returned version `2`.
- Reading the skill after upload confirmed `files["assets/logo.png"]` and preserved `SKILL.md`.
- Response `asset_summary` returned count `1` and image path `assets/logo.png`.
- Uploading to `scripts/logo.png` was rejected with app code `111000000` and message `invalid parameter : asset path must be under assets/`.
- Temporary skills created by the probe were deleted.

Console note:

- While routing through the currently deployed frontend, Playwright captured repeated minified React error `#130` from the production static bundle after visiting the skill page. The backend asset contract and E2E API checks were successful, but this frontend bundle error should be covered by the upcoming skill-center UI pass before uploading new frontend static assets.

## 2026-06-19 12:22 CST: Sandbox Exec Writable Workdir Enforcement

Backend change under test:

- `POST /api/super-agent/sandbox/exec` now validates `workdir` with writable sandbox roots only: `/workspace`, `/uploads`, and `/outputs`.
- `/skills` remains readable through workspace list/read/search APIs, but cannot be used as the command working directory.
- This keeps standard skill package roots read-only for ordinary command execution while still allowing skills to be inspected and invoked by absolute script paths.

TDD RED check observed:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent -run TestRunSuperAgentSandboxCommandExecutesInWritableWorkdir -count=1
```

Failed because `/skills/pdf` was accepted as an exec workdir.

GREEN verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent -run 'TestRunSuperAgentSandboxCommandExecutesInWritableWorkdir|TestReadSuperAgentWorkspaceFileAllowsReadableSkillRoot|TestSearchSuperAgentWorkspaceAllowsReadableSkillRoot' -count=1
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow ./api/router/coze ./api/router/skill ./api/handler/skill ./application/skill ./application/singleagent ./application/conversation ./domain/skill/... ./api/middleware -count=1
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-workspace.test.ts __tests__/developer-api-super-agent-runs.test.ts __tests__/developer-api-super-agent-skills.test.ts __tests__/developer-api-super-agent-artifacts.test.ts
pnpm --dir frontend/packages/arch/bot-api exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/agent-ide/entry exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/apps/coze-studio exec tsc --noEmit -p tsconfig.json
```

Passed.

Known test-suite note:

- `SESSION_HMAC_SECRET=test-secret go test ./application/singleagent ./api/router/coze ./api/handler/coze ./api/middleware ...` was attempted first and failed before running handler assertions because `backend/api/handler/coze` currently has an existing test import cycle through `application.go`; the affected packages above were run separately and passed.

Backend build:

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o ../bin/openynet main.go
```

Result: passed. Local SHA-256:

- `9123485884cba44fcaf105e78e59f78c0a045450e77e851feecbe4bd57d8022b`

Remote deployment details:

- Uploaded to `/home/dev/openynet-next.upload`.
- Verified remote staging SHA-256 matched local exactly.
- Backed up the previous container binary at `/home/dev/openynet.coze-super.bak-20260619042006`.
- Copied the verified binary into `coze-super:/app/openynet`, restarted `coze-super`, and verified the container SHA-256 matched local:
  - `9123485884cba44fcaf105e78e59f78c0a045450e77e851feecbe4bd57d8022b`

Playwright MCP 8896 probe:

- Opened `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=20260619_exec_workdir`.
- Confirmed the page loaded as `演示超级体 -智能体 - 猎鹰`, did not redirect to login, and showed the bottom input plus the `人设 · 技能 · MCP` entry.
- File: `super-agent-playwright-mcp-8896-exec-workdir-final.json`
- `POST /api/super-agent/sandbox/exec` with `workdir=/workspace` and `command=pwd` returned `code=0` and stdout `/workspace`.
- `POST /api/super-agent/sandbox/exec` with `workdir=/skills/pdf` returned business code `100000000` and message `invalid parameter : path must be under /workspace, /uploads, /outputs`.
- `POST /api/super-agent/workspace/list` with `path=/skills` still returned `code=0` and listed standard skill folders `docx`, `pdf`, `pptx`, and `xlsx`.
- Important request-field note: this probe exposed that the App Server exec API only accepted JSON field `workdir`; using `work_dir` was ignored by the then-current generated contract and fell back to the default workdir. This was fixed in the follow-up `work_dir` alias compatibility deployment below.
- File: `super-agent-playwright-mcp-8896-exec-workdir-network-final.txt`
- Relevant network entries for sandbox exec and workspace list all returned HTTP 200.
- File: `super-agent-playwright-mcp-8896-exec-workdir-console-final.txt`
- Console check returned zero error-level messages.
- Screenshot file: `super-agent-arrange-8896-exec-workdir-final.png`.

### Follow-up: `work_dir` Alias Compatibility

Backend/SDK change under test:

- `ExecSandboxCommandRequest` now accepts `work_dir` as a compatibility alias for `workdir`.
- Service precedence is explicit: `workdir` wins when non-empty; otherwise `work_dir` is used.
- The generated DeveloperApi request type and request body mapper now include `work_dir`, so TS callers can use either spelling.

TDD RED checks observed:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent -run TestRunSuperAgentSandboxCommandExecutesInWritableWorkdir -count=1
```

Failed first because `ExecSandboxCommandRequest` did not have `WorkDirAlias`.

```bash
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-workspace.test.ts
```

Failed first because `DeveloperApi.SuperAgentExecSandbox` did not pass `work_dir` through the request body.

GREEN verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent -run 'TestRunSuperAgentSandboxCommandExecutesInWritableWorkdir|TestReadSuperAgentWorkspaceFileAllowsReadableSkillRoot|TestSearchSuperAgentWorkspaceAllowsReadableSkillRoot' -count=1
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent ./api/router/coze ./api/middleware -run 'TestRunSuperAgentSandboxCommand|TestSuperAgent|TestRequestInspector' -count=1
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow ./api/router/coze ./api/router/skill ./api/handler/skill ./application/skill ./application/singleagent ./application/conversation ./domain/skill/... ./api/middleware -count=1
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-workspace.test.ts
pnpm --dir frontend/packages/arch/bot-api exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/agent-ide/entry exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/apps/coze-studio exec tsc --noEmit -p tsconfig.json
```

Passed.

Backend build:

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o ../bin/openynet main.go
```

Result: passed. Local SHA-256:

- `a58bbb7b9734a30c83fca57e76711a2ded65fd43700975f0ff0196be28714998`

Remote deployment details:

- Uploaded to `/home/dev/openynet-next.upload`.
- Verified remote staging SHA-256 matched local exactly.
- Backed up the previous container binary at `/home/dev/openynet.coze-super.bak-20260619042838`.
- Copied the verified binary into `coze-super:/app/openynet`, restarted `coze-super`, and verified the container SHA-256 matched local:
  - `a58bbb7b9734a30c83fca57e76711a2ded65fd43700975f0ff0196be28714998`

Playwright MCP 8896 probe:

- Opened `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=20260619_exec_workdir_alias`.
- Confirmed the page loaded as `演示超级体 -智能体 - 猎鹰`, did not redirect to login, and showed the bottom input plus the `人设 · 技能 · MCP` entry.
- File: `super-agent-playwright-mcp-8896-exec-workdir-alias-final.json`
- `POST /api/super-agent/sandbox/exec` with `workdir=/workspace` and `command=pwd` returned `code=0` and stdout `/workspace`.
- `POST /api/super-agent/sandbox/exec` with `work_dir=/outputs` and `command=pwd` returned `code=0` and stdout `/outputs`.
- `POST /api/super-agent/sandbox/exec` with `work_dir=/skills/pdf` returned business code `100000000` and message `invalid parameter : path must be under /workspace, /uploads, /outputs`.
- File: `super-agent-playwright-mcp-8896-exec-workdir-alias-network-final.txt`
- Relevant network entries for sandbox exec all returned HTTP 200.
- File: `super-agent-playwright-mcp-8896-exec-workdir-alias-console-final.txt`
- Console check returned zero error-level messages.

### Follow-up: Workspace Patch `work_dir` Alias Compatibility

Backend/SDK change under test:

- `ApplySandboxPatchRequest` now accepts `work_dir` as a compatibility alias for `workdir`.
- Service precedence is explicit: `workdir` wins when non-empty; otherwise `work_dir` is used.
- The generated DeveloperApi request type and request body mapper now include `work_dir`, so TS callers can use either spelling for Codex-style `apply_patch`.

TDD RED checks observed:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent -run TestApplySuperAgentWorkspacePatchMovesFiles -count=1
```

Failed first because `ApplySandboxPatchRequest` did not have `WorkDirAlias`.

```bash
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-workspace.test.ts
```

Failed first because `DeveloperApi.SuperAgentApplyWorkspacePatch` did not pass `work_dir` through the request body.

GREEN verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent -run 'TestApplySuperAgentWorkspacePatchMovesFiles|TestRunSuperAgentSandboxCommandExecutesInWritableWorkdir' -count=1
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow ./api/router/coze ./api/router/skill ./api/handler/skill ./application/skill ./application/singleagent ./application/conversation ./domain/skill/... ./api/middleware -count=1
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-workspace.test.ts
pnpm --dir frontend/packages/arch/bot-api exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/agent-ide/entry exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/apps/coze-studio exec tsc --noEmit -p tsconfig.json
```

Passed.

Backend build:

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o ../bin/openynet main.go
```

Result: passed. Local SHA-256:

- `d26a6ec49e0d1cc91bd6a5724bcc8114a19b20f0c658b623cf9e1971e1d60175`

Remote deployment details:

- Uploaded to `/home/dev/openynet-next.upload`.
- Verified remote staging SHA-256 matched local exactly.
- Backed up the previous container binary at `/home/dev/openynet.coze-super.bak-20260619043506`.
- Copied the verified binary into `coze-super:/app/openynet`, restarted `coze-super`, and verified the container SHA-256 matched local:
  - `d26a6ec49e0d1cc91bd6a5724bcc8114a19b20f0c658b623cf9e1971e1d60175`

Playwright MCP 8896 probe:

- Opened `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=20260619_patch_workdir_alias`.
- Confirmed the page loaded as `演示超级体 -智能体 - 猎鹰`, did not redirect to login, and showed the bottom input plus the `人设 · 技能 · MCP` entry.
- File: `super-agent-playwright-mcp-8896-patch-workdir-alias-final.json`
- `POST /api/super-agent/workspace/patch` with `work_dir=/outputs` added `patch-alias-1781843744021.txt` and returned changed path `/outputs/patch-alias-1781843744021.txt`.
- `POST /api/super-agent/workspace/read` for that path returned `code=0` and content `patch alias 1781843744021`.
- `POST /api/super-agent/workspace/patch` with `work_dir=/skills/pdf` returned business code `100000000` and message `invalid parameter : path must be under /workspace, /uploads, /outputs`.
- File: `super-agent-playwright-mcp-8896-patch-workdir-alias-network-final.txt`
- Relevant network entries for workspace patch/read all returned HTTP 200.
- File: `super-agent-playwright-mcp-8896-patch-workdir-alias-console-final.txt`
- Console check returned zero error-level messages.

### Follow-up: Manifest Request Schema Discovery

Backend change under test:

- `GET /api/super-agent/manifest` now exposes lightweight `request_schemas` for `workspace.patch` and `sandbox.exec`.
- `workspace.request_schemas.patch` declares required field `patch`, optional fields `space_id`, `agent_id`, `bot_id`, `connector_id`, `workdir`, and `work_dir`, plus alias `workdir -> work_dir`.
- `sandbox.request_schemas.exec` declares required field `command`, optional fields `space_id`, `agent_id`, `bot_id`, `connector_id`, `workdir`, `work_dir`, and `timeout_sec`, plus alias `workdir -> work_dir`.
- This gives external App Server clients a discoverable contract for the two Codex-style endpoints that carry working-directory semantics.

TDD RED check observed:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run TestSuperAgentManifestRouteReturnsAppServerContract -count=1
```

Failed first because both `workspace.request_schemas.patch` and `sandbox.request_schemas.exec` were missing from the manifest.

GREEN verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run TestSuperAgentManifestRouteReturnsAppServerContract -count=1
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow ./api/router/coze ./api/router/skill ./api/handler/skill ./application/skill ./application/singleagent ./application/conversation ./domain/skill/... ./api/middleware -count=1
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-workspace.test.ts
pnpm --dir frontend/packages/arch/bot-api exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/agent-ide/entry exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/apps/coze-studio exec tsc --noEmit -p tsconfig.json
```

Passed.

Backend build:

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o ../bin/openynet main.go
```

Result: passed. Local SHA-256:

- `d55868ad41352d9874ff8ca5ab2bc55ecdfce3239a1243e08cb5adb4d1c24330`

Remote deployment details:

- Uploaded to `/home/dev/openynet-next.upload`.
- Verified remote staging SHA-256 matched local exactly.
- Backed up the previous container binary at `/home/dev/openynet.coze-super.bak-20260619044049`.
- Copied the verified binary into `coze-super:/app/openynet`, restarted `coze-super`, and verified the container SHA-256 matched local:
  - `d55868ad41352d9874ff8ca5ab2bc55ecdfce3239a1243e08cb5adb4d1c24330`

Playwright MCP 8896 probe:

- Opened `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=20260619_manifest_schema`.
- Confirmed the page loaded as `演示超级体 -智能体 - 猎鹰`, did not redirect to login, and showed the bottom input plus the `人设 · 技能 · MCP` entry.
- File: `super-agent-playwright-mcp-8896-manifest-request-schemas-final.json`
- `GET /api/super-agent/manifest` returned HTTP 200, `code=0`, and `protocol_version=super-agent.app-server.v1`.
- Observed `workspace_patch.required=["patch"]`, optional fields including `workdir` and `work_dir`, and alias `workdir -> work_dir`.
- Observed `sandbox_exec.required=["command"]`, optional fields including `workdir`, `work_dir`, and `timeout_sec`, and alias `workdir -> work_dir`.
- File: `super-agent-playwright-mcp-8896-manifest-request-schemas-network-final.txt`
- Manifest network entry returned HTTP 200.
- File: `super-agent-playwright-mcp-8896-manifest-request-schemas-console-final.txt`
- Console check returned zero error-level messages.

## 2026-06-19 12:00 CST: App Server Bearer Auth Route Parity + Send Display Recheck

Local change under test:

- Expanded OpenAPI/Bearer auth routing for the App Server super-agent API surface.
- `/api/super-agent/manifest` remains public.
- Routes under `/api/super-agent/runs`, `/api/super-agent/harness`, `/api/super-agent/sandbox`, `/api/super-agent/traces`, `/api/super-agent/artifacts`, `/api/super-agent/workspace`, `/api/super-agent/skills`, and `/api/super-agent/marketplace` now support Bearer auth dispatch when `Authorization: Bearer ...` is present.
- Added middleware coverage for every currently advertised App Server route family, including `runs/get`, `runs/list`, harness, sandbox, workspace write/stat/grep/glob/edit/patch, artifacts delete/move, skills, and marketplace routes.

TDD RED check observed:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/middleware -run TestSuperAgentAppServerPathsNeedOpenAPIAuth -count=1
```

Failed before the middleware update because newly advertised App Server routes were not recognized by `isNeedOpenapiAuth`.

GREEN verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/middleware -run TestSuperAgentAppServerPathsNeedOpenAPIAuth -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/middleware -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/middleware ./api/router/coze -run 'TestSuperAgent|TestSuperAgentAppServerPathsNeedOpenAPIAuth' -count=1
```

Passed.

Backend build:

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o ../bin/openynet main.go
```

Result: passed. Local SHA-256:

- `8fd04fdec215ca357e36acf18c7a472ff5d429f50ca3a369cc0008c7a6688d8f`

Remote deployment details:

- Initial SSH deploy attempt was interrupted while the 173MB backend binary was still uploading; the staging file was only 18MB and was not used for final validation.
- Re-uploaded to `/home/dev/openynet-next.upload`, verified the remote staging SHA-256 matched local exactly.
- Backed up the previous container binary, copied the verified binary into `coze-super:/app/openynet`, restarted `coze-super`, and verified the container SHA-256 matched local:
  - `8fd04fdec215ca357e36acf18c7a472ff5d429f50ca3a369cc0008c7a6688d8f`

8896 App Server probes:

- `GET http://10.10.10.226:8896/api/super-agent/manifest`: passed with `code=0`.
- Manifest contract observed:
  - `protocol_version=super-agent.app-server.v1`
  - `auth.type=bearer`
  - `routes` count: `37`
  - Confirmed route entries for `runs.get`, `runs.list`, `harness.state`, `harness.plan`, `sandbox.exec`, `workspace.stat`, `workspace.grep`, `workspace.glob`, `workspace.edit`, `workspace.patch`, `skills.list`, and `skills.marketplace.get`.
- Invalid Bearer route-classification probes:
  - `POST /api/super-agent/workspace/stat` with `Authorization: Bearer invalid-pat-for-route-check`: returned `authentication failed: CheckPermission failed`.
  - `GET /api/super-agent/skills/list` with `Authorization: Bearer invalid-pat-for-route-check`: returned `authentication failed: CheckPermission failed`.
- Interpretation: these routes now enter the OpenAPI/Bearer auth path instead of falling through to web-session auth.

Playwright MCP 8896 UI recheck:

- Opened `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=20260619_authfix`.
- Page loaded without redirecting to login.
- Confirmed visible layout restored to the native arrange/debug view:
  - Left workspace width is `280px`.
  - Center preview is visible.
  - Right `预览与调试` panel is visible.
  - Top `人设 · 技能 · MCP` entry is visible.
  - Right-side config button is visible.
  - Bottom chat input is visible with placeholder `继续对话...`.
- Sent test message through Playwright MCP:
  - `MCP 复测：发送内容显示验证 2026-06-19 11:59`
- Result:
  - User message appeared in the conversation.
  - Assistant response appeared in the conversation.
  - No page console errors were reported during the send check.
- Opened right-side config dialog from the debug panel.
- Result:
  - Dialog opened successfully.
  - Dialog content showed `技能`, `授权管理`, `管理 OAuth 插件`, and the empty OAuth plugin state.

## 2026-06-19 12:10 CST: Persisted Run Status for App Server `runs/get`

Backend change under test:

- `POST /api/super-agent/runs/get` now falls back to persisted run records when the requested `run_id` is not currently active in memory.
- This makes external App Server clients able to query finished/failed historical runs after a process restart or after the in-memory active-run registry has been cleared.
- The response now keeps the existing `run_id`, `status`, `active`, `created_at`, and `updated_at` fields, and adds lightweight persisted metadata when available:
  - `conversation_id`
  - `agent_id`
  - `error`
  - `completed_at`
  - `failed_at`

TDD RED check observed:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run TestSuperAgentGetRunRouteReportsPersistedRun -count=1
```

Failed before the change because `runs/get` returned `status=not_active` and omitted persisted `conversation_id`, `agent_id`, and timestamps for an existing run record.

GREEN verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run TestSuperAgentGetRunRouteReportsPersistedRun -count=1
SESSION_HMAC_SECRET=test-secret go test ./domain/conversation/agentrun/service -run 'TestRunImpl_(GetByID|List)' -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgent(GetRun|ListRuns|RunRoutes|CancelRun|Manifest)' -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/middleware -run 'TestSuperAgent|TestSuperAgentAppServerPathsNeedOpenAPIAuth' -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze ./domain/conversation/agentrun/service ./api/middleware -count=1
```

Passed.

Backend build:

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o ../bin/openynet main.go
```

Result: passed. Local SHA-256:

- `df269c9a894e2f74e17222964e7884eb93da432501b338a20eaf47a355ccb769`

Remote deployment details:

- Uploaded to `/home/dev/openynet-next.upload`.
- Verified remote staging SHA-256 matched local exactly.
- Backed up the previous container binary, copied the verified binary into `coze-super:/app/openynet`, restarted `coze-super`, and verified the container SHA-256 matched local:
  - `df269c9a894e2f74e17222964e7884eb93da432501b338a20eaf47a355ccb769`

8896 backend probe:

- Selected the latest completed run for the test account from `run_record`.
- Called `POST http://127.0.0.1:8896/api/super-agent/runs/get` on the deployed host with a valid web session cookie.
- Sanitized response fields observed:
  - `code=0`
  - `msg=success`
  - `run_id=7652951366939181056`
  - `conversation_id=7652617181540122624`
  - `agent_id=7652617174313336832`
  - `status=completed`
  - `active=false`
  - `created_at=1781841592631`
  - `updated_at=1781841602541`
  - `completed_at=1781841602539`

Post-deploy Playwright MCP smoke:

- Opened the deployed arrange page after the backend restart.
- Confirmed the page loaded and the chat input remained visible.
- Sent `MCP 复测：run get 部署后发送显示 12:09`.
- Confirmed the user message and assistant response both appeared; no page console errors were reported.

Frontend/SDK contract follow-up:

- Updated `DeveloperApi` super-agent run status typings so TypeScript callers can consume the persisted run metadata returned by the backend:
  - `conversation_id`
  - `agent_id`
  - `error`
  - `completed_at`
  - `failed_at`
- Added a bot-api test object for the persisted run status shape.
- Verification:

```bash
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-runs.test.ts
pnpm --dir frontend/packages/arch/bot-api exec tsc --noEmit -p tsconfig.json
```

Passed.

## 2026-06-19 11:41 CST: Arrange UI Revert And Send Visibility

Local frontend revert under test:

- Restored super-mode arrange layout to the original two-column `2fr 1fr` split with native `AgentChatArea`.
- Restored `人设 · 技能 · MCP` settings to a modal instead of the temporary side sheet.
- Removed the custom super chat shell from the active super-mode render path.
- Restored tool rows to the native expandable summary style: tool name, parameter summary, return summary, and expand affordance.
- Reverted the latest left workspace width/preview min-width changes that made the file preview cramped.

Local verification:

```bash
pnpm --dir frontend/packages/agent-ide/entry exec vitest --run src/modes/super-mode/__tests__/super-mode.test.tsx src/modes/super-mode/__tests__/super-chat-area.test.tsx src/modes/super-mode/__tests__/super-hero.test.tsx src/modes/super-mode/__tests__/sandbox-workspace.test.tsx src/modes/super-mode/codex-trace/__tests__/codex-trace-panel.test.tsx src/modes/super-mode/codex-trace/__tests__/trace-bridge.test.tsx
pnpm --dir frontend/packages/agent-ide/entry exec tsc --noEmit -p tsconfig.json
cd frontend && BUILD_BRANCH=openynet-local rush rebuild -o @coze-studio/app --verbose
```

Result: passed. The rebuild completed with only existing Browserslist/caniuse-lite warnings.

Remote deployment:

- Synced `frontend/apps/coze-studio/dist/` to the 8896 test host staging directory with sourcemaps excluded.
- Replaced `/app/resources/static` in the running `coze-super` container.
- Restarted `coze-super`.
- HTTP checks passed:
  - `GET /api/super-agent/manifest` -> 200
  - target arrange HTML -> 200

Playwright MCP 8896 probe:

- `mcp__playwright.browser_tabs new/select`: passed; opened the target arrange page.
- Initial send probe on an old browser session reproduced "message disappears".
- Backend log root cause for that false-negative: `/api/conversation/chat` returned business auth error `authentication failed: session not exist` because the browser had a stale pre-restart session cookie.
- Refreshed Playwright to a valid test session without recording credentials or cookies.
- Reopened `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange`.
- Verified visible UI:
  - left workspace is present with file tree and preview area.
  - right panel is native `预览与调试`.
  - bottom input is visible and editable.
  - top-right `人设 · 技能 · MCP` button is visible.
  - clicking `人设 · 技能 · MCP` opens the modal settings dialog.
- Sent `Playwright 验证：请只回复 OK`.
- Result: passed; DOM contained both the user message and assistant `OK`, and the input placeholder returned to `继续对话...`.

Notes:

- The disappearing-message symptom can also be triggered by stale/invalid `session_key` after a service/static replacement; the frontend currently removes the optimistic message after the API returns the business auth error.

## 2026-06-19 09:14 CST: Workspace Apply Patch App Server Endpoint

Local change under test:

- Added `POST /api/super-agent/workspace/patch` as the Codex-like `apply_patch` HTTP App Server surface.
- The endpoint accepts Codex-style `*** Begin Patch` / `*** End Patch` patches and applies Add, Update, and Delete operations inside writable sandbox roots.
- Patch paths are resolved under `workdir` when relative and are restricted to `/workspace`, `/uploads`, and `/outputs`; `/skills` remains read-only.
- Manifest `workspace` contract now exposes `patch_route=POST /api/super-agent/workspace/patch`.
- Manifest `routes` now exposes `workspace.patch`.
- Harness tool discovery now includes `apply_patch`.
- Frontend `DeveloperApi` now exposes `SuperAgentApplyWorkspacePatch`.

Reference alignment checked:

- Hermes Codex App Server runtime documents Codex's toolset as including shell, file read/write/search, `apply_patch`, and `update_plan`.
- OpenAI Codex skills documentation describes standard skills as folders with instructions, resources, and optional scripts/assets, matching the platform direction already being implemented here.

TDD RED checks observed:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent ./api/router/coze -run 'TestApplySuperAgentWorkspacePatchAppliesCodexPatch|TestSuperAgentWorkspaceRoutesRejectMalformedJSON|TestSuperAgentManifestRouteReturnsAppServerContract' -count=1
```

Failed because `ApplySuperAgentWorkspacePatch`, `ApplySandboxPatchRequest`, the `/api/super-agent/workspace/patch` route, manifest `patch_route`, `workspace.patch`, and harness `apply_patch` tool contract were missing.

```bash
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-workspace.test.ts
```

Failed because `DeveloperApi.SuperAgentApplyWorkspacePatch` was not a function.

GREEN verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent ./api/router/coze -run 'TestApplySuperAgentWorkspacePatchAppliesCodexPatch|TestSuperAgentWorkspaceRoutesRejectMalformedJSON|TestSuperAgentManifestRouteReturnsAppServerContract' -count=1
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-workspace.test.ts
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow ./api/router/coze ./api/router/skill ./api/handler/skill ./application/skill ./application/singleagent ./application/conversation ./domain/skill/... -count=1
pnpm --dir frontend/packages/arch/bot-api exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/agent-ide/entry exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/apps/coze-studio exec tsc --noEmit -p tsconfig.json
```

Passed.

Backend build:

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o ../bin/openynet main.go
```

Result: passed. Local SHA-256:

- `3d83c6b71a072ad3b5da0d99a8f0e4e8841e08befe79935f8acf0899985c762d`

Playwright MCP 8896 probe:

- `mcp__playwright.browser_tabs list`: passed; existing arrange tabs were visible.
- `mcp__playwright.browser_tabs new` to `http://10.10.10.226:8896/api/super-agent/manifest`: failed with 60000ms navigation timeout while waiting for `domcontentloaded`.
- `mcp__playwright.browser_snapshot` on the existing arrange page: passed; title was `演示超级体 -智能体 - 猎鹰`.
- `mcp__playwright.browser_console_messages`: passed; current deployed backend returned 404 for `/api/super-agent/manifest`, `/api/super-agent/runs/create`, `/api/super-agent/workspace/list`, marketplace list routes, and `/api/super-agent/runs/cancel`.
- `mcp__playwright.browser_network_requests`: passed; confirmed `/api/super-agent/manifest`, `/api/super-agent/runs/create`, `/api/super-agent/workspace/list`, and `/api/super-agent/runs/cancel` still return 404 on the currently deployed service.
- `mcp__playwright.browser_tabs new` with a temporary browser form POST to `http://10.10.10.226:8896/api/super-agent/workspace/patch`: failed with a 60000ms navigation timeout.
- `mcp__playwright.browser_network_requests` for `workspace/patch`: passed; network log showed `POST /api/super-agent/workspace/patch` failed with `net::ERR_ABORTED`.
- `mcp__playwright.browser_take_screenshot`: passed; saved viewport screenshots as `super-agent-arrange-8896-apply-patch.png` and `super-agent-workspace-patch-post-probe-8896.png` in the Playwright MCP output area.

Remote service replacement probe:

- Method: read-only SSH probe to `dev@10.10.10.226`.
- Result: failed.
- Failure: port 22 timed out before authentication.
- Effect: no remote files or services were modified in this step.

## 2026-06-19 09:05 CST: Harness Plan Update App Server Endpoint

Local change under test:

- Added `POST /api/super-agent/harness/plan` as the Codex-like `update_plan` HTTP App Server surface.
- The new endpoint validates standard plan statuses and writes the tool-compatible JSON array to `/workspace/.plan.json`.
- Manifest `harness` contract now exposes `plan_update_route=POST /api/super-agent/harness/plan`.
- Manifest `routes` now exposes `harness.plan`.
- Frontend `DeveloperApi` now exposes `SuperAgentUpdateHarnessPlan`.

TDD RED checks observed:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent ./api/router/coze -run 'TestUpdateSuperAgentHarnessPlanWritesToolCompatiblePlan|TestSuperAgentHarnessRoutesRejectMalformedJSON|TestSuperAgentManifestRouteReturnsAppServerContract' -count=1
```

Failed because `UpdateSuperAgentHarnessPlan`, the harness plan request models, the `/api/super-agent/harness/plan` route, and manifest route contract were missing.

```bash
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-workspace.test.ts
```

Failed because `DeveloperApi.SuperAgentUpdateHarnessPlan` was not a function.

GREEN verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent ./api/router/coze -run 'TestUpdateSuperAgentHarnessPlanWritesToolCompatiblePlan|TestSuperAgentHarnessRoutesRejectMalformedJSON|TestSuperAgentManifestRouteReturnsAppServerContract' -count=1
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-workspace.test.ts
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow ./api/router/coze ./api/router/skill ./api/handler/skill ./application/skill ./application/singleagent ./application/conversation ./domain/skill/... -count=1
pnpm --dir frontend/packages/arch/bot-api exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/agent-ide/entry exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/apps/coze-studio exec tsc --noEmit -p tsconfig.json
```

Passed.

Backend build:

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o ../bin/openynet main.go
```

Result: passed. Local SHA-256:

- `38d672f98d626306518729965186e23cce7000a9258708d37307f5bd47f98d8f`

Playwright MCP 8896 probe:

- `mcp__playwright.browser_tabs list`: passed; existing arrange tabs were visible.
- `mcp__playwright.browser_tabs new` to `http://10.10.10.226:8896/api/super-agent/manifest`: failed with 60000ms navigation timeout while waiting for `domcontentloaded`.
- `mcp__playwright.browser_snapshot` on the existing arrange page: passed; title was `演示超级体 -智能体 - 猎鹰`.
- `mcp__playwright.browser_console_messages`: passed; current deployed backend returned 404 for `/api/super-agent/manifest`, `/api/super-agent/runs/create`, `/api/super-agent/workspace/list`, marketplace list routes, and `/api/super-agent/runs/cancel`.
- `mcp__playwright.browser_network_requests`: passed; confirmed `/api/super-agent/manifest`, `/api/super-agent/runs/create`, `/api/super-agent/workspace/list`, and `/api/super-agent/runs/cancel` still return 404 on the currently deployed service.
- `mcp__playwright.browser_tabs new` with a temporary browser form POST to `http://10.10.10.226:8896/api/super-agent/harness/plan`: failed; Playwright network log showed `net::ERR_CONNECTION_TIMED_OUT`.
- `mcp__playwright.browser_take_screenshot`: passed; saved viewport screenshots as `super-agent-arrange-8896-harness-plan.png` and `super-agent-harness-plan-post-probe-8896.png` in the Playwright MCP output area.

Remote service replacement probe:

- Method: read-only SSH probe to `dev@10.10.10.226`.
- Result: failed.
- Failure: port 22 timed out before authentication.
- Effect: no remote files or services were modified in this step.

## 2026-06-19 08:56 CST: Workspace Write App Server Alias

Local change under test:

- Added `POST /api/super-agent/workspace/write` as the Codex-like `write_file` HTTP App Server surface.
- The new route reuses the existing super-agent workspace write/upload semantics, so writes remain limited to `/workspace`, `/uploads`, and `/outputs`.
- Manifest `workspace` contract now exposes `write_route=POST /api/super-agent/workspace/write`.
- Manifest `routes` now exposes `workspace.write`.
- Frontend `DeveloperApi` now exposes `SuperAgentWriteWorkspaceFile`.

TDD RED checks observed:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgentWorkspaceRoutesRejectMalformedJSON|TestSuperAgentManifestRouteReturnsAppServerContract' -count=1
```

Failed because the manifest did not expose `workspace.write` / `write_route`.

```bash
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-workspace.test.ts
```

Failed because `DeveloperApi.SuperAgentWriteWorkspaceFile` was not a function.

GREEN verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgentWorkspaceRoutesRejectMalformedJSON|TestSuperAgentManifestRouteReturnsAppServerContract' -count=1
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-workspace.test.ts
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow ./api/router/coze ./api/router/skill ./api/handler/skill ./application/skill ./application/singleagent ./application/conversation ./domain/skill/... -count=1
pnpm --dir frontend/packages/arch/bot-api exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/agent-ide/entry exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/apps/coze-studio exec tsc --noEmit -p tsconfig.json
```

Passed.

Backend build:

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o ../bin/openynet main.go
```

Result: passed. Local SHA-256:

- `a3adfd4bbe5009178bc1e55cf4635247fff80eb34a77e249aa6351e91bc7e620`

Playwright MCP 8896 probe:

- `mcp__playwright.browser_tabs list`: passed; existing arrange tabs were visible.
- `mcp__playwright.browser_tabs new` to `http://10.10.10.226:8896/api/super-agent/manifest`: failed with 60000ms navigation timeout while waiting for `domcontentloaded`.
- `mcp__playwright.browser_tabs close`: passed; closed the loading/error tab.
- `mcp__playwright.browser_snapshot` on the existing arrange page: passed; title was `演示超级体 -智能体 - 猎鹰`; visible areas included `FinMallClaw 能力` and `人设 · 技能 · MCP`.
- `mcp__playwright.browser_console_messages`: passed; current deployed backend returned 404 for `/api/super-agent/manifest`, `/api/super-agent/runs/create`, `/api/super-agent/workspace/list`, and `/api/super-agent/runs/cancel`.
- `mcp__playwright.browser_network_requests`: passed; confirmed the super-agent routes still return 404 on the currently deployed service.
- `mcp__playwright.browser_take_screenshot`: passed; saved viewport screenshot as `super-agent-arrange-8896-workspace-write.png` in the Playwright MCP output area.

Remote service replacement probe:

- Method: read-only SSH probe to `dev@10.10.10.226`.
- Result: failed.
- Failure: port 22 timed out before authentication.
- Effect: no remote files or services were modified in this step.

## 2026-06-19 15:10 CST: Runtime Skill Folder Import

Local change under test:

- Added `POST /api/super-agent/skills/import-runtime` for saving a standard skill folder authored inside an agent sandbox at `/skills/<name>/` into the skill library.
- The runtime import reads only standard files: `SKILL.md`, `scripts/`, `references/`, `templates/`, and `assets/`.
- The target space is derived from the agent draft instead of trusting a caller-provided `space_id`.
- Binary assets under `assets/` are stored as data URLs; non-asset binary files are rejected.
- Optional `skill_id` updates an existing skill; optional `publish_scope` publishes the saved version immediately.
- Manifest and frontend SDK metadata now expose `skills.import_runtime`.

TDD RED checks observed:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent -run TestImportSuperAgentRuntimeSkillCreatesAndPublishesStandardSkill -count=1
```

Failed because `ImportSuperAgentRuntimeSkill` and `SuperAgentRuntimeSkillImportRequest` did not exist.

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgentManifestRouteReturnsAppServerContract|TestSuperAgentSkillRoutesRejectMalformedJSON' -count=1
```

Failed because `/api/super-agent/skills/import-runtime` returned 404 and the manifest did not expose the route.

```bash
pnpm --dir frontend/packages/arch/api-schema test -- --run __tests__/skill-super-agent.test.ts
```

Failed because `SuperAgentImportRuntimeSkill` was missing from frontend API metadata.

GREEN verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent -run TestImportSuperAgentRuntimeSkillCreatesAndPublishesStandardSkill -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgentManifestRouteReturnsAppServerContract|TestSuperAgentSkillRoutesRejectMalformedJSON' -count=1
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent ./api/router/coze ./api/handler/skill -count=1
SESSION_HMAC_SECRET=test-secret go test ./application/skill -count=1
pnpm --dir frontend/packages/arch/api-schema test -- --run __tests__/skill-super-agent.test.ts
pnpm --dir frontend/packages/arch/bot-api test -- --run __tests__/developer-api-super-agent-skills.test.ts
pnpm --dir frontend/packages/arch/api-schema exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/arch/bot-api exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/apps/coze-studio exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/apps/coze-studio build
git diff --check
```

Passed. Build warnings were limited to existing Browserslist/Baseline data age notices.

Production build:

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o ../bin/openynet main.go
tar -C frontend/apps/coze-studio/dist -czf /tmp/coze-studio-static-runtime-skill.tar.gz .
```

Deployment to `10.10.10.226:8896`:

- Backend binary SHA-256: `0fea6bddf124cd094cf05a851b68bc0cb82c7ab15b8dac7e26abf7a5a52b02c7`
- Frontend static tar SHA-256: `18c7d36d75f900e979d142c1de5a0db09f15f4bd65fdde2288bb8260bc82ba99`
- Container backup: `/app/openynet.bak-runtime-skill-20260619150342`
- Container backup: `/app/resources/static.bak-runtime-skill-20260619150342`
- Restart result: `coze-super Up 1 second`
- Container `/app/openynet` SHA-256 after restart matched local exactly.

Post-deploy manifest probe:

```bash
curl -sS http://10.10.10.226:8896/api/super-agent/manifest | jq '{importRuntime: .data.routes["skills.import_runtime"], skillRoute: .data.skills.routes.import_runtime, schema: .data.skills.request_schemas.import_runtime}'
```

Result:

```json
{
  "importRuntime": "POST /api/super-agent/skills/import-runtime",
  "skillRoute": "POST /api/super-agent/skills/import-runtime",
  "schema": {
    "required": ["agent_id", "name"],
    "optional": ["bot_id", "connector_id", "skill_id", "icon_uri", "publish_scope"]
  }
}
```

Playwright MCP 8896 verification:

- Page context: `http://10.10.10.226:8896/space/7652614054615187456/skills?mcp_check=skill_validate_20260619`
- Created a standard runtime skill folder through `POST /api/super-agent/sandbox/exec` at `/skills/runtime-probe-52959898/`.
- Imported it through `POST /api/super-agent/skills/import-runtime` with `publish_scope=2`.
- Verified the response contained:
  - `SKILL.md`
  - `scripts/run.sh`
  - `assets/logo.png`
  - `metadata.version=0.1.0`
  - `publish_scope=2`
  - `published_version=1`
  - `assets/logo.png` encoded as `data:image/png;base64,...`
- Verified `GET /api/super-agent/skills/list` could find the imported skill before cleanup.
- Cleaned the probe skill with `POST /api/super-agent/skills/delete`.
- Status summary from Playwright network:
  - `POST /api/super-agent/sandbox/exec` => `200 OK`
  - `POST /api/super-agent/skills/import-runtime` => `200 OK`
  - `GET /api/super-agent/skills/list?...keyword=runtime-probe-52959898` => `200 OK`
  - `POST /api/super-agent/skills/delete` => `200 OK`
- Saved evaluation result: `super-agent-playwright-mcp-8896-runtime-skill-import-final.json`
- Saved import response: `super-agent-playwright-mcp-8896-runtime-skill-import-response-final.json`
- Saved network log: `super-agent-playwright-mcp-8896-runtime-skill-import-network-final.txt`
- Saved screenshot: `super-agent-playwright-mcp-8896-runtime-skill-import-page-final.png`

## 2026-06-19 16:07 CST: Harness Workspace State UI And Skill Fallback

Local change under test:

- `SandboxWorkspace` now reads the App Server contract endpoints instead of the legacy sandbox endpoints:
  - `POST /api/super-agent/harness/state`
  - `POST /api/super-agent/workspace/list`
  - `POST /api/super-agent/workspace/read`
  - `POST /api/super-agent/workspace/write`
  - `POST /api/super-agent/workspace/delete`
- The left footer now summarizes the live `.plan.json` content and live tool output count from harness state.
- The workspace root includes a readonly `技能` segment.
- The `技能` segment first uses `runtime_skills` from harness state; when that field is omitted, it falls back to readonly `/skills` directory listing and hides internal dot files such as `.manifest`.

TDD and local verification:

```bash
pnpm --dir frontend/packages/agent-ide/entry exec vitest --run src/modes/super-mode/__tests__/sandbox-workspace.test.tsx
pnpm --dir frontend/packages/agent-ide/entry exec vitest --run src/modes/super-mode/__tests__/sandbox-workspace-utils.test.ts src/modes/super-mode/__tests__/sandbox-workspace.test.tsx src/modes/super-mode/__tests__/super-mode.test.tsx src/modes/super-mode/__tests__/super-chat-area.test.tsx src/modes/super-mode/codex-trace/__tests__/codex-trace-panel.test.tsx
pnpm --dir frontend/packages/agent-ide/entry exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/apps/coze-studio exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/apps/coze-studio build
git diff --check
```

Passed:

- `sandbox-workspace.test.tsx`: 6 tests.
- Super-mode related suite: 5 files, 22 tests.
- Package and app typecheck.
- App production build. Warnings were limited to existing Browserslist/Baseline age notices.
- `git diff --check`.

Deployment to `10.10.10.226:8896`:

- Static tar SHA-256: `efe055009369830dd6ca67d1d071c6ac96a844563010731f8b6b16c349b69e2a`
- Static tar size: `66M`
- Container backup kept: `/app/resources/static.bak-harness-fallback-202606191605`
- Current static size after deploy: `306.9M`
- Backup size after deploy: `311.8M`
- Removed stale upload tarballs under `/home/dev/static-*.tar.gz`.
- Disk after deploy and cleanup: `/` total `72G`, used `56G`, available `13G`, use `82%`.
- Restart result: `coze-super Up 2 minutes`.

Playwright MCP 8896 verification:

- Page context: `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=harness_skill_fallback_final_20260619`
- Harness state probe:
  - `POST /api/super-agent/harness/state` => `200`, `code=0`
  - plan path: `/workspace/.plan.json`
  - current plan content after restore: one `completed` step
  - tool output count: `4`
- Skill fallback probe:
  - `POST /api/super-agent/workspace/list` with `path=/skills` => `200`, `code=0`
  - returned names: `.manifest`, `docx`, `pdf`, `pptx`, `runtime-probe-52959898`, `xlsx`
  - UI after clicking `技能` showed `docx`, `pdf`, `pptx`, `xlsx`, `runtime-probe-52959898`
  - UI did not show `.manifest`
- Dynamic plan probe:
  - Temporarily wrote a two-step `.plan.json` through `POST /api/super-agent/workspace/write`.
  - Reloaded arrange page and verified footer changed to `计划 1/2 完成 · 1 进行中`.
  - Restored original `.plan.json` and verified the readback matched the original one-step content.
- Saved results:
  - `super-agent-playwright-mcp-8896-harness-skill-fallback-final.json`
  - `super-agent-playwright-mcp-8896-harness-skill-fallback-final-dom-confirm.json`
  - `super-agent-playwright-mcp-8896-harness-skill-fallback-final.png`
  - `super-agent-playwright-mcp-8896-harness-plan-dynamic-final-ui.json`
  - `super-agent-playwright-mcp-8896-harness-plan-dynamic-final-restore.json`

## 2026-06-19 16:31 CST: Global Skill Store Redesign

Design input:

- Local reference HTML: `/Users/luzhipeng/Downloads/技能商店 重设计.html`.
- Applied the reference direction to `/explore/project/latest` while keeping the latest product decision that the skill store should not show the secondary sidebar when there is only one store entry.

Local change under test:

- Reworked the global skill store page as a standard skill marketplace:
  - large visual hero with the generated `skill-marketplace-hero.webp` asset
  - four metric cards: `全局标准技能`, `技能分类`, `公司可安装`, `包含资产`
  - filter/search toolbar with live `N 个技能包` count
  - standard skill cards showing `SKILL.md`, standard folders, file count, asset count, version, category, and publish date
  - detail modal with standard file list and `SKILL.md` preview
- Disabled the Explore secondary sidebar for this page by setting `hasSider=false`.
- Fixed the build-only dependency break in super mode by importing `SingleAgentModelView` from `@coze-agent-ide/bot-config-area-adapter`; the adapter now preserves caller-provided `triggerRender`.

TDD and local verification:

```bash
pnpm --dir frontend/apps/coze-studio exec vitest --run src/pages/skill-marketplace/__tests__/index.test.tsx
pnpm --dir frontend/apps/coze-studio exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/agent-ide/entry exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/apps/coze-studio build
git diff --check
```

Passed:

- `skill-marketplace/__tests__/index.test.tsx`: 1 test.
- App and agent-ide entry typecheck.
- App production build. Warnings were limited to existing Browserslist/Baseline age notices.
- `git diff --check`.

Deployment to `10.10.10.226:8896`:

- Final static tar SHA-256: `27ce6e0fbf19c55fd560782feebff1610d7a5da45c343cd6cffe0720c8ab6379`
- Static tar size: `67M`
- Container backup kept: `/app/resources/static.bak-skill-store-redesign-202606191628`
- Current static size after deploy: `311.8M`
- Backup size after deploy: `311.8M`
- Removed `/home/dev/static-skill-store-redesign-202606191628.tar.gz` after copying into the container.
- Removed local `/tmp/static-skill-store-redesign-202606191628.tar.gz`.
- Removed previous static backups and kept only the current backup.
- Disk after deploy and cleanup: `/` total `72G`, used `56G`, available `13G`, use `82%`.

Playwright MCP 8896 verification:

- Page context: `http://10.10.10.226:8896/explore/project/latest?mcp_check=skill_store_redesign_grid_20260619`
- DOM checks:
  - `技能商店` hero is visible.
  - `2 个技能包` count is visible.
  - all four metric cards are visible and laid out horizontally at desktop width.
  - secondary sidebar selectors returned no visible text.
  - API `GET /api/super-agent/marketplace/list?scope=3&page=1&page_size=200` returned `code=0`, `total=2`.
  - two standard skill cards rendered.
- Interaction checks:
  - category chip click kept the expected two `通用` skills.
  - first `查看详情` opened a detail modal with standard file list and `SKILL.md 预览`.
- Saved screenshots:
  - `super-agent-playwright-mcp-8896-skill-store-redesign-final.png`
  - `super-agent-playwright-mcp-8896-skill-store-redesign-grid-final.png`

## 2026-06-19 16:45 CST: External App Server Manifest Discovery

Goal:

- Make `/api/super-agent/manifest` advertise the external App Server integration contract so outside clients can discover how to call super-agent runs without reverse-engineering routes.

Local change under test:

- Added `external_api` to the super-agent manifest response.
- Contract fields now include:
  - `base_path=/api/super-agent`
  - `protocol_version=super-agent.app-server.v1`
  - bearer auth metadata and `session_auth_supported=true`
  - JSON and SSE transports
  - `agent_id` / `bot_id` identifier fields
  - `sandbox`, `workspace`, `skills`, `harness` capabilities
  - create, stream, reply, get, list, cancel entry routes
  - per-route request schemas with `required`, `required_one_of`, and `optional` fields
  - client metadata flags for App Server clients
- Updated frontend IDL typings so `bot_open_api.SuperAgentManifestResponse` exposes the same `external_api` shape.
- Added an IDL type regression test for `external_api`.

TDD and local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run TestSuperAgentManifestRouteReturnsAppServerContract -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgent(ManifestRouteReturnsAppServerContract|RunRoutesRejectMissingAgentID|CancelRunRouteRejectsMissingRunID|GetRunRouteRejectsMissingRunID|ListRunsRouteRejectsMissingConversationID|ReplyRunRouteRejectsNonPositiveConversationID)' -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -count=1
pnpm --dir frontend/packages/arch/bot-api exec vitest --run __tests__/bot-open-api-super-agent-manifest.test.ts
pnpm --dir frontend/packages/arch/bot-api exec vitest --run __tests__/bot-open-api-super-agent-manifest.test.ts __tests__/axios.test.ts
pnpm --dir frontend/packages/arch/idl exec tsc --noEmit -p tsconfig.build.json
pnpm --dir frontend/packages/arch/bot-api exec tsc --noEmit -p tsconfig.build.json
```

Passed:

- Router manifest contract test.
- Router negative-route regression set.
- Full `./api/router/coze` package test.
- Bot API manifest IDL vitest and axios regression tests.
- IDL and bot-api build typechecks.

Known unrelated limitation:

- `SESSION_HMAC_SECRET=test-secret go test ./api/router/coze ./api/handler/coze -count=1` still hits an existing handler test import cycle through `conversation_service_test.go`.
- `pnpm --dir frontend/packages/arch/bot-api exec tsc --noEmit -p tsconfig.misc.json` still hits the existing Rollup 3/4 config type conflict in `config/vitest-config/src/preset-default.ts`; after the IDL fix it no longer reports missing `external_api` or stale manifest fields.

Deployment to `10.10.10.226:8896`:

- Built backend binary: `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-external-api-manifest-202606191640 main.go`
- Binary SHA-256: `e69a9709aecf9d40d73ca53f04cf6f6a23a89a3ec2fd3a9902a65b2cbf19d9f7`
- Gzip upload SHA-256: `287861595673426fe379ae07efcb3bfdabe1f50a94d4658ad9ca5d032e7c617b`
- Gzip upload size: `46M`; decompressed remote binary size: `173M`.
- Replaced `/app/openynet` in the `coze-super` container and restarted the service.
- Kept only one backend backup: `/app/openynet.bak-external-api-manifest-202606191640`.
- Kept only one static backup from the skill store deploy: `/app/resources/static.bak-skill-store-redesign-202606191628`.
- Removed `/home/dev/openynet-*` and static upload tarballs after deployment.
- Disk after cleanup: `/` total `72G`, used `54G`, available `15G`, use `80%`.

Remote curl verification:

```bash
curl -sS http://10.10.10.226:8896/api/super-agent/manifest | jq '{code, external_api: .data.external_api}'
```

Confirmed:

- `code=0`.
- `external_api.base_path=/api/super-agent`.
- `external_api.protocol_version=super-agent.app-server.v1`.
- `external_api.entry_routes` includes create, stream, reply, get, list, cancel.
- `external_api.request_schemas.create.required_one_of` contains `agent_id` / `bot_id`.
- `external_api.client_metadata.capabilities_value=sandbox,workspace,skills,harness`.

Playwright MCP 8896 verification:

- Page context: `http://10.10.10.226:8896/explore/project/latest?mcp_check=external_api_manifest_20260619`
- Browser-context fetch of `/api/super-agent/manifest` confirmed:
  - HTTP ok and `code=0`
  - `external_api` exists
  - bearer auth metadata exists
  - session auth support is enabled
  - transports include `json` and `sse`
  - identifier fields include `agent_id` and `bot_id`
  - capabilities include `sandbox`, `workspace`, `skills`, and `harness`
  - all run entry routes match the public `/api/super-agent/runs/*` endpoints
  - create/reply request schemas expose required and optional fields for external clients
  - client metadata advertises the App Server flag, capabilities param, transport param, and client id param
- Console check: zero errors; one existing warning.
- Saved evidence:
  - `super-agent-playwright-mcp-8896-external-api-manifest-final.json`

## 2026-06-19 16:59 CST: Standard Skill Asset Delete API

Goal:

- Complete the standard skill asset management loop so uploaded image/assets under `assets/` can also be removed through the App Server API, instead of accumulating indefinitely.

Local change under test:

- Added `SkillApplicationService.DeleteSkillAsset`.
- Added `POST /api/super-agent/skills/assets/delete`.
- Added `assets.delete` to the super-agent manifest routes and request schemas.
- Added `skills.assets.delete` to the top-level manifest route map.
- Added `delete_route` to the manifest `skills.assets` contract.
- Added frontend API schema, DeveloperApi IDL, and skill-center hook support for deleting skill assets.

TDD red checks:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/skill -run TestSkillApplicationServiceDeletesSkillAssetFromStandardFiles -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run TestSuperAgentManifestRouteReturnsAppServerContract -count=1
pnpm --dir frontend/packages/arch/bot-api exec vitest --run __tests__/developer-api-super-agent-skills.test.ts
```

Expected failures observed:

- `svc.DeleteSkillAsset undefined`.
- Manifest lacked `assets.delete`, `skills.assets.delete`, and `skills.assets.delete_route`.
- `DeveloperApi.SuperAgentDeleteSkillAsset is not a function`.

Local verification after implementation:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/skill -run 'TestSkillApplicationService(UpsertsSkillAssetIntoStandardFiles|DeletesSkillAssetFromStandardFiles|RejectsInvalidSkillAssetPath)' -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgentManifestRouteReturnsAppServerContract|TestSuperAgentHarnessRoutesRejectMalformedJSON' -count=1
pnpm --dir frontend/packages/arch/bot-api exec vitest --run __tests__/developer-api-super-agent-skills.test.ts
pnpm --dir frontend/packages/arch/idl exec tsc -p tsconfig.build.json
SESSION_HMAC_SECRET=test-secret go test ./application/skill -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -count=1
pnpm --dir frontend/packages/arch/bot-api exec vitest --run __tests__/developer-api-super-agent-skills.test.ts __tests__/bot-open-api-super-agent-manifest.test.ts
pnpm --dir frontend/packages/arch/idl exec tsc --noEmit -p tsconfig.build.json
pnpm --dir frontend/apps/coze-studio exec vitest --run src/pages/space-skill/__tests__/index.test.tsx src/pages/space-skill/__tests__/standard-skill-files.test.ts
pnpm --dir frontend/packages/arch/api-schema exec vitest --run __tests__/skill-super-agent.test.ts
SESSION_HMAC_SECRET=test-secret go test ./application/skill ./api/handler/skill ./api/router/coze -count=1
pnpm --dir frontend/apps/coze-studio exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/arch/api-schema exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/arch/bot-api exec tsc --noEmit -p tsconfig.build.json
git diff --check
```

Passed:

- Skill application, skill handler, and super-agent router tests.
- Skill center frontend tests.
- API schema, IDL, bot-api tests and typechecks.
- App typecheck.
- `git diff --check`.

Deployment to `10.10.10.226:8896`:

- Backend binary SHA-256: `0dd319b78ca042fc52d7b5bcd57f16625e38606f614ecd648cf78b1765df53eb`
- Gzip upload SHA-256: `e377cf29fc49fc31085b2792a682bbed91fa4bd558de1a106864f8c03d32c913`
- Gzip upload size: `46M`; decompressed remote binary size: `173M`.
- Replaced `/app/openynet` and restarted `coze-super`.
- Kept only one backend backup: `/app/openynet.bak-skill-asset-delete-202606191655`.
- Did not upload static assets for this backend-only route/API change.
- Removed `/home/dev/openynet-skill-asset-delete-202606191655.gz` and `.upload`.
- Removed local `/tmp/openynet-skill-asset-delete-202606191655*`.
- Disk after cleanup: `/` total `72G`, used `54G`, available `15G`, use `80%`.

Remote curl verification:

```bash
curl -sS http://10.10.10.226:8896/api/super-agent/manifest | jq '{code, assets: .data.skills.assets, route: .data.routes["skills.assets.delete"], schema: .data.skills.request_schemas["assets.delete"]}'
```

Confirmed:

- `skills.assets.delete = POST /api/super-agent/skills/assets/delete`
- `skills.assets.delete_route = POST /api/super-agent/skills/assets/delete`
- `skills.request_schemas["assets.delete"].required = ["space_id","skill_id","path"]`

Playwright MCP 8896 verification:

- Page context: `http://10.10.10.226:8896/space/7652614054615187456/skills?mcp_check=skill_asset_delete_20260619`
- Browser-context API flow:
  - created temporary standard skill `mcp-asset-delete-1781859504989`
  - upserted `assets/logo.png` as `data:image/png;base64,iVBORw0KGgo=`
  - read the skill and confirmed the asset existed
  - called `POST /api/super-agent/skills/assets/delete`
  - read the skill and confirmed `assets/logo.png` was removed
  - deleted the temporary skill
- All checks were true:
  - `manifestHasDeleteRoute`
  - `manifestHasDeleteSchema`
  - `createOk`
  - `upsertOk`
  - `assetExistsBeforeDelete`
  - `deleteAssetOk`
  - `assetRemovedAfterDelete`
  - `cleanupOk`
- Console check: zero errors; one existing warning.
- Saved evidence:
  - `super-agent-playwright-mcp-8896-skill-asset-delete-final.json`

## 2026-06-19 17:10 CST: Standard Skill Asset List/Get API

Goal:

- Add progressive-disclosure asset management for standard skills so App Server clients can list lightweight asset metadata and fetch one asset by path without downloading the whole skill package.

Local change under test:

- Added `SkillApplicationService.ListSkillAssets`.
- Added `SkillApplicationService.GetSkillAsset`.
- Added `GET /api/super-agent/skills/assets/list`.
- Added `GET /api/super-agent/skills/assets/get`.
- Added `assets.list` and `assets.get` to the super-agent manifest routes and request schemas.
- Added `list_route` and `get_route` to the manifest `skills.assets` contract.
- Added API schema, DeveloperApi IDL, and skill-center hook support for listing and reading skill assets.
- List responses return lightweight metadata only: `path`, `mime`, `size`, `is_image`.
- Get responses return the same metadata plus `content`.

TDD red checks:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/skill -run TestSkillApplicationServiceListsAndGetsSkillAssets -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run TestSuperAgentManifestRouteReturnsAppServerContract -count=1
pnpm --dir frontend/packages/arch/bot-api exec vitest --run __tests__/developer-api-super-agent-skills.test.ts
pnpm --dir frontend/packages/arch/api-schema exec vitest --run __tests__/skill-super-agent.test.ts
```

Expected failures observed:

- `svc.ListSkillAssets undefined`.
- `svc.GetSkillAsset undefined`.
- Manifest lacked `assets.list`, `assets.get`, `skills.assets.list`, `skills.assets.get`, `list_route`, and `get_route`.
- `DeveloperApi.SuperAgentListSkillAssets is not a function`.
- API schema lacked `SuperAgentListSkillAssets`.

Local verification after implementation:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/skill -run TestSkillApplicationServiceListsAndGetsSkillAssets -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run TestSuperAgentManifestRouteReturnsAppServerContract -count=1
pnpm --dir frontend/packages/arch/bot-api exec vitest --run __tests__/developer-api-super-agent-skills.test.ts
pnpm --dir frontend/packages/arch/api-schema exec vitest --run __tests__/skill-super-agent.test.ts
SESSION_HMAC_SECRET=test-secret go test ./application/skill ./api/handler/skill ./api/router/coze -count=1
pnpm --dir frontend/packages/arch/idl exec tsc --noEmit -p tsconfig.build.json
pnpm --dir frontend/packages/arch/bot-api exec vitest --run __tests__/developer-api-super-agent-skills.test.ts __tests__/bot-open-api-super-agent-manifest.test.ts
pnpm --dir frontend/apps/coze-studio exec vitest --run src/pages/space-skill/__tests__/index.test.tsx src/pages/space-skill/__tests__/standard-skill-files.test.ts
pnpm --dir frontend/apps/coze-studio exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/arch/api-schema exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/arch/bot-api exec tsc --noEmit -p tsconfig.build.json
git diff --check
```

Passed:

- Skill application, skill handler, and super-agent router tests.
- API schema and bot-api tests.
- Skill center frontend tests.
- IDL, app, api-schema, and bot-api typechecks.
- `git diff --check`.

Deployment to `10.10.10.226:8896`:

- Backend binary SHA-256: `841696720f778efc90c0903e7bc202c29eece7d6074831da0ceae269b73b7428`
- Gzip upload SHA-256: `5299f215aa91487e13111f4ecb6ca1b5a9e3a8a4ad5928d761f2157163427499`
- Gzip upload size: `46M`; decompressed remote binary size: `173M`.
- Replaced `/app/openynet` and restarted `coze-super`.
- Kept only one backend backup: `/app/openynet.bak-skill-assets-list-get-202606191707`.
- Did not upload static assets for this backend/API contract change.
- Removed `/home/dev/openynet-skill-assets-list-get-202606191707.gz` and `.upload`.
- Removed local `/tmp/openynet-skill-assets-list-get-202606191707*`.
- Disk after cleanup: `/` total `72G`, used `54G`, available `15G`, use `80%`.

Remote curl verification:

```bash
curl -sS http://10.10.10.226:8896/api/super-agent/manifest | jq '{code, list: .data.routes["skills.assets.list"], get: .data.routes["skills.assets.get"], assets: .data.skills.assets, list_schema: .data.skills.request_schemas["assets.list"], get_schema: .data.skills.request_schemas["assets.get"]}'
```

Confirmed:

- `skills.assets.list = GET /api/super-agent/skills/assets/list`
- `skills.assets.get = GET /api/super-agent/skills/assets/get`
- `skills.assets.list_route = GET /api/super-agent/skills/assets/list`
- `skills.assets.get_route = GET /api/super-agent/skills/assets/get`
- `skills.request_schemas["assets.list"].required = ["space_id","skill_id"]`
- `skills.request_schemas["assets.get"].required = ["space_id","skill_id","path"]`

Playwright MCP 8896 verification:

- Page context: `http://10.10.10.226:8896/space/7652614054615187456/skills?mcp_check=skill_assets_list_get_20260619`
- Browser-context API flow:
  - created temporary standard skill `mcp-asset-list-get-1781860204195`
  - included `assets/logo.png`, `assets/readme.txt`, and `scripts/run.sh`
  - called `GET /api/super-agent/skills/assets/list`
  - confirmed list returned only `assets/logo.png` and `assets/readme.txt`
  - confirmed image/text metadata: `image/png`, `text/plain`, `is_image`, and `size`
  - called `GET /api/super-agent/skills/assets/get` for `assets/logo.png`
  - confirmed content was `data:image/png;base64,iVBORw0KGgo=`
  - deleted the temporary skill
- All checks were true:
  - `manifestHasListRoute`
  - `manifestHasGetRoute`
  - `manifestHasListSchema`
  - `manifestHasGetSchema`
  - `createOk`
  - `listOk`
  - `listOnlyAssets`
  - `logoMetadataOk`
  - `textMetadataOk`
  - `getOk`
  - `getReturnsContent`
  - `cleanupOk`
- Remote logs showed 200 responses for both `skills/assets/list` and `skills/assets/get`, with no `panic`, `nil pointer`, `fatal`, or `500` matches in the checked window.
- Console check: zero errors; one existing warning.
- Saved evidence:
  - `super-agent-playwright-mcp-8896-skill-assets-list-get-final.json`

## 2026-06-19 17:44 CST: Skill Store Padding/Sidebar Redesign

Goal:

- Make `http://10.10.10.226:8896/explore/project/latest` match the provided single-file skill store design.
- Restore the global layout sidebar and skill-store secondary sidebar.
- Remove the old project/external-app store labels from the explore store page.
- Align the actual page spacing with the design:
  - content wrap: `24px 32px 40px`
  - hero: `30px 34px 28px`
  - skill card: `20px 20px 16px`
  - search box: `280px x 38px`

Local change under test:

- `frontend/apps/coze-studio/src/pages/explore.tsx`
  - `/explore/project/latest` now loads the skill marketplace page with `hasSider: true`.
  - legacy explore store routes redirect to `/explore/project/latest`.
- `frontend/packages/community/explore/src/components/sub-menu/index.tsx`
  - added a skill-store-focused secondary sidebar.
- `frontend/packages/community/explore/src/components/sub-menu/index.module.less`
  - matched the provided design's 236px sidebar content layout.
- `frontend/apps/coze-studio/src/pages/skill-marketplace/index.tsx`
  - rebuilt the marketplace shell, hero, stats, filters, and cards from the provided design.
  - fixed production runtime icon issue by using `IconCozMagnifier` for the search box.
- `frontend/apps/coze-studio/src/pages/skill-marketplace/index.module.less`
  - matched the design spacing, radii, card grid, and responsive behavior.

Local verification:

```bash
pnpm --dir frontend/apps/coze-studio exec vitest --run src/pages/__tests__/explore-route.test.tsx src/pages/skill-marketplace/__tests__/index.test.tsx
pnpm --dir frontend/apps/coze-studio exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/community/explore exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/apps/coze-studio build
```

Passed:

- 2 targeted Vitest tests.
- App and explore package typechecks.
- Production build. Build only reported existing Browserslist/baseline-data warnings.

Deployment to `10.10.10.226:8896`:

- Uploaded compressed static build: `67M`.
- Replaced container path `/app/resources/static`.
- Kept one static rollback directory only:
  - `/app/resources/static.bak-skill-store-padding-20260619173541`
- Removed host upload package and local `/tmp` package after deployment.
- Disk after cleanup: `/` total `72G`, used `54G`, available `15G`, use `80%`.
- Static directories after cleanup:
  - `/app/resources/static`: `311.9M`
  - `/app/resources/static.bak-skill-store-padding-20260619173541`: `311.9M`

Playwright MCP 8896 verification:

- Page context:
  - `http://10.10.10.226:8896/explore/project/latest?mcp_check=skill_store_padding_fixed_20260619_1744`
- Browser cache and service worker cleared before final verification.
- First remote run caught a real production rendering error:
  - React minified error `#130`
  - cause: `IconSearch` was undefined at runtime in `@coze-arch/coze-design/icons`
  - fix: changed search icon to `IconCozMagnifier`
- Final console check:
  - zero errors
  - one existing `single-spa` warning
- Final network check:
  - `GET /api/super-agent/marketplace/list?scope=3&page=1&page_size=200 => 200 OK`
- Final DOM/layout checks:
  - `hasSkillStore = true`
  - `hasStandardSkillPackage = true`
  - `hasBrowse = true`
  - `hasCategory = true`
  - `hasManage = true`
  - `hasHeroSegmentStandard = true`
  - `hasAllResources = true`
  - `hasOldProjectStore = false`
  - `hasOldExternalApp = false`
  - `hasFallbackError = false`
  - cards: `2`
  - skill-store secondary sidebar: `1`
  - sidebar width: `236px`
  - wrap padding: `24px 32px 40px`
  - hero padding: `30px 34px 28px`
  - first card padding: `20px 20px 16px`
  - search box: `280px x 38px`, padding `0px 14px`
- Additional 1280x800 viewport check:
  - `hasFallbackError = false`
  - `hasSkillStore = true`
  - `hasOldProjectStore = false`
  - cards: `2`
  - sidebar width: `236px`
  - wrap padding: `24px 32px 40px`
  - hero padding: `30px 34px 28px`
  - search box: `280px x 38px`, padding `0px 14px`
- Saved evidence:
  - `skill-marketplace-padding-fixed-final-20260619.png`
  - `skill-marketplace-padding-1280-20260619.png`
  - `skill-marketplace-padding-fixed-final-20260619.md`
  - `skill-marketplace-padding-fixed-final-20260619-network.txt`
  - `super-agent-playwright-mcp-8896-skill-marketplace-padding-final.json`
  - `super-agent-playwright-mcp-8896-skill-marketplace-padding-1280-final.json`

## 2026-06-19 App Server Approvals Contract

Scope:

- Added App Server approval discovery to `/api/super-agent/manifest`.
- Added `/api/super-agent/approvals/list`.
- Added `/api/super-agent/approvals/resolve`.

Local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgentManifestRouteReturnsAppServerContract|TestSuperAgentApprovalListRouteReturnsEmptyPendingQueue|TestSuperAgentApprovalResolveRouteRejectsMissingApprovalID' -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -count=1
SESSION_HMAC_SECRET=test-secret go build ./...
git diff --check
```

Passed:

- Targeted red/green route tests.
- Full `api/router/coze` package tests.
- Backend `go build ./...`.
- `git diff --check`.

Known local test limitation:

- `SESSION_HMAC_SECRET=test-secret go test ./api/handler/coze -count=1` is blocked by an existing test import cycle:
  - `conversation_service_test.go -> backend/application -> backend/api/handler/coze`.
  - This package-level blocker is unrelated to the approvals change; the router tests and backend build cover the new route wiring and typecheck.

Deployment to `10.10.10.226:8896`:

- Built backend binary:
  - `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-approvals-20260619175701 main.go`
  - Local/container SHA-256: `4cb2b54601431a5010288bc1004fee74be0a0b5f5c194f27109c3f714668dbf7`
  - Binary size: `173M`
  - Compressed upload size: `46M`
- Replaced `coze-super:/app/openynet` and restarted `coze-super`.
- Kept only one backend rollback binary:
  - `/app/openynet.bak-approvals-20260619175701`
- Kept the existing static rollback directory:
  - `/app/resources/static.bak-skill-store-padding-20260619173541`
- Removed `/home/dev/openynet-approvals-20260619175701.gz` and `.upload`.
- Disk after cleanup:
  - `/` total `72G`, used `54G`, available `15G`, use `79%`.

Playwright MCP 8896 verification:

- Page context:
  - `http://10.10.10.226:8896/explore/project/latest?mcp_check=approvals_20260619`
- Browser fetch checks:
  - `GET /api/super-agent/manifest => 200`
    - `capabilities` includes `approvals`.
    - `external_api.capabilities` includes `approvals`.
    - `routes.approvals.list = POST /api/super-agent/approvals/list`.
    - `routes.approvals.resolve = POST /api/super-agent/approvals/resolve`.
    - request schemas expose required `conversation_id` for list and required `approval_id`, `decision` for resolve.
  - `POST /api/super-agent/approvals/list {"conversation_id":"123","agent_id":"456"} => 200`
    - response `code = 0`, `msg = success`, `conversation_id = 123`, `approvals = []`.
  - `POST /api/super-agent/approvals/resolve {"decision":"approve"} => 400`
    - response contains `approval_id is required`.
  - `POST /api/super-agent/approvals/resolve {"approval_id":"run:inactive-run","decision":"approve","note":"mcp smoke"} => 200`
    - response `status = approved`, `resolved = true`, `cancelled = false`.
- Console:
  - one expected 400 network console entry from the negative `resolve` check.
  - one existing `single-spa` warning.
- Server logs:
  - route registration confirms `/api/super-agent/approvals/list` and `/api/super-agent/approvals/resolve`.
  - smoke requests logged as expected: manifest 200, list 200, negative resolve 400, approve resolve 200.
- Saved evidence:
  - `super-agent-playwright-mcp-8896-approvals-final.json`
  - `super-agent-playwright-mcp-8896-approvals-network-final.txt`

## 2026-06-19 App Server Sessions Contract

Scope:

- Added App Server session discovery to `/api/super-agent/manifest`.
- Added `/api/super-agent/sessions/list`.
- Added `/api/super-agent/sessions/rename`.
- Added a super-agent session sidebar in the arrange page with recent sessions and inline rename.
- Session titles are stored on conversation `Ext` under `super_agent_session_title`; existing conversations fall back to `会话 <conversation_id>`.

Local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgent(SessionListRouteReturnsRenamableSessions|SessionRenameRoutePersistsTitleInConversationExt|ManifestRouteReturnsAppServerContract)' -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -count=1
SESSION_HMAC_SECRET=test-secret go build ./...
pnpm --filter @coze-agent-ide/bot-creator exec vitest --run src/modes/super-mode/__tests__/super-session-sidebar.test.tsx src/modes/super-mode/__tests__/super-mode.test.tsx
pnpm --filter @coze-arch/bot-api exec vitest --run __tests__/developer-api-super-agent-workspace.test.ts
pnpm --filter @coze-arch/idl exec tsc -b tsconfig.json --pretty false
pnpm --filter @coze-arch/bot-api exec tsc -p tsconfig.build.json --noEmit --pretty false
BUILD_BRANCH=openynet-local rush rebuild -o @coze-studio/app --verbose
git diff --check
```

Passed:

- Targeted route tests for session list, rename, and manifest contract.
- Full `api/router/coze` package tests.
- Backend `go build ./...`.
- Super-mode session sidebar Vitest tests.
- Developer API wrapper Vitest tests.
- IDL package typecheck.
- Bot API build typecheck.
- Coze Studio frontend rebuild.
- `git diff --check`.

Known local test/typecheck limitations:

- `SESSION_HMAC_SECRET=test-secret go test ./api/handler/coze -count=1` is still blocked by the existing test import cycle:
  - `conversation_service_test.go -> backend/application -> backend/api/handler/coze`.
- `pnpm --filter @coze-arch/bot-api exec tsc -b tsconfig.json --pretty false` is blocked by existing repository issues:
  - Rollup 3/4 type conflict in `frontend/config/vitest-config/src/preset-default.ts`.
  - old super-agent manifest test fixture missing newer asset fields.
- `pnpm --filter @coze-agent-ide/bot-creator exec tsc -p tsconfig.build.json --noEmit --pretty false` is blocked by existing project-reference/unbuilt dist issues and older single-mode/workflow typing errors. The production app rebuild passed.

Deployment to `10.10.10.226:8896`:

- Built backend binary:
  - `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-sessions-20260619182504 main.go`
  - Local/container SHA-256: `7aa1f50019b281d0b08b2cfe3503ca2776c11a71e969350288b54aa0f8502ec9`
  - Binary size: `173M`
  - Compressed upload size: `46M`
- Built frontend static bundle from `frontend/apps/coze-studio/dist`.
  - Static tar SHA-256: `916bab48b94b6d4c6586367d2ed788e9385a0c90ba33f7465c8e2b661c71aa56`
  - Static tar size: `25M`
- Replaced `coze-super:/app/openynet`, replaced `/app/resources/static`, and restarted `coze-super`.
- Kept only one rollback pair for this deployment:
  - `/app/openynet.bak-sessions-20260619182504`
  - `/app/resources/static.bak-sessions-20260619182504`
- Current remote sizes:
  - `/app/resources/static`: `92.7M`
  - backend rollback: `173.1M`
  - static rollback: `311.9M`
- Disk after deployment and cleanup:
  - `/` total `72G`, used `54G`, available `15G`, use `79%`.
- Local `/tmp/openynet-sessions-20260619182504*` and `/tmp/coze-static-sessions-20260619182504.tar.gz` were removed after deployment.

Playwright MCP 8896 verification:

- Page context:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=sessions_20260619`
- UI snapshot:
  - Title: `演示超级体 -智能体 - 猎鹰`.
  - Session sidebar exists.
  - Recent sessions count: `9`.
  - Session rows expose rename buttons.
  - Workspace and preview/debug columns remain visible.
- Browser fetch checks:
  - `GET /api/super-agent/manifest => 200`
    - `capabilities` includes `sessions`.
    - `sessions.list_route = POST /api/super-agent/sessions/list`.
    - `sessions.rename_route = POST /api/super-agent/sessions/rename`.
    - `sessions.title_ext_key = super_agent_session_title`.
    - `sessions.max_title_length = 128`.
    - `external_api.capabilities_value = sessions,sandbox,workspace,skills,harness,approvals`.
  - `POST /api/super-agent/sessions/list {"space_id":"7652614054615187456","bot_id":"7652617174313336832","page":1,"page_size":20} => 200`
    - response `code = 0`, `msg = success`.
    - returned `9` sessions.
    - first session `7652827977981362176`, title `会话 7652827977981362176`, `renamable = true`.
  - `POST /api/super-agent/sessions/rename {"conversation_id":"1"} => 400`
    - response contains `title is required`.
  - Positive rename/restore smoke:
    - target conversation: `7652827977981362176`.
    - rename to `MCP 会话改名验证 1781865026713` => `200`, `code = 0`.
    - subsequent list returned the temporary title.
    - restore to `会话 7652827977981362176` => `200`, `code = 0`.
    - final list returned the original title again.
- Console:
  - one expected 400 network console entry from the negative rename guard check.
  - one existing `single-spa` warning.
- Saved evidence:
  - `super-agent-playwright-mcp-8896-sessions-final.json`
  - `super-agent-playwright-mcp-8896-sessions-rename-positive-final.json`
  - `super-agent-playwright-mcp-8896-sessions-network-final.txt`
  - `super-agent-playwright-mcp-8896-sessions-ui-snapshot.md`
  - `super-agent-playwright-mcp-8896-sessions-ui.png`

## 2026-06-19 18:57 CST: Explicit Session Message List Switch

Issue found during Playwright MCP regression:

- The frontend session sidebar dispatched the selected `conversation_id`.
- The chat provider sent `POST /api/conversation/get_message_list` with that `conversation_id` and `scene = 9`.
- The backend still ignored the requested `conversation_id` and called `GetCurrentConversation`, so the response could return a different conversation.

Local fix under test:

- `ConversationApplicationService.GetMessageList` now resolves a non-empty `conversation_id` through `ConversationDomainSVC.GetByID`.
- Explicit session reads validate `CreatorID` and `AgentID` before listing messages.
- Requests without `conversation_id` keep the original current-conversation/create-if-missing path.

TDD RED check observed:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/conversation -run TestGetMessageListUsesRequestedConversationID -count=1
```

Failed before the fix because the service returned current conversation `111` instead of requested conversation `222`.

GREEN verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/conversation -run TestGetMessageListUsesRequestedConversationID -count=1
SESSION_HMAC_SECRET=test-secret go test ./application/conversation -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -count=1
SESSION_HMAC_SECRET=test-secret go build ./...
git diff --check
```

Passed.

Broader test attempt:

```bash
SESSION_HMAC_SECRET=test-secret go test ./...
```

Failed on existing unrelated repository issues, including:

- `api/handler/coze` test import cycle.
- `domain/memory/variables/internal/dal` printf format/type mismatch.
- `application/base/appinfra` tests referencing undefined model mapping helpers.
- `application/space/import` tests using the old `GenerateMapping` signature.
- `application/space/sync` SQLite `ON CONFLICT` constraint mismatch.
- workflow tests requiring Mockey `-gcflags="all=-N -l"` and one LLM nil-pointer test.
- local MySQL test failing with root access denied.

Backend-only deployment:

- Built binary:
  - `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-session-conversation-20260619185459 main.go`
  - Local/container SHA-256: `dcf4be7e589a368c2dba3b478b4e5d99c3dcd3db77d4f3c37a53f78ad7f7e612`
  - Binary size: `173M`
  - Compressed upload size: `46M`
- Replaced `coze-super:/app/openynet` only. Static assets were not redeployed.
- Kept one backend rollback point inside the container:
  - `/app/openynet.bak-sessions-20260619185459`
- Removed the uploaded gzip from `/home/dev` and removed local `/tmp/openynet-session-conversation-20260619185459*`.
- Disk after deployment:
  - overlay total `71.8G`, used `53.2G`, available `14.9G`, use `78%`.
- Container after restart:
  - `coze-super Up About a minute`
  - PID 1: `/app/openynet`

Playwright MCP 8896 verification:

- Page:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange`
- Action:
  - Clicked session `7652828097363836928` in the session sidebar.
- Network evidence:
  - `super-agent-playwright-mcp-8896-session-switch-backend-fix-network-final.txt`
  - `super-agent-playwright-mcp-8896-session-switch-backend-fix-request-body-final.json`
  - `super-agent-playwright-mcp-8896-session-switch-backend-fix-response-body-final.json`
- Observed result:
  - request `conversation_id = 7652828097363836928`
  - request `scene = 9`
  - response `conversation_id = 7652828097363836928`
  - response `last_section_id = 7652828097363853312`
  - response message count `2`
  - request/response conversation IDs matched.
- UI evidence:
  - `super-agent-playwright-mcp-8896-session-switch-backend-fix-ui-snapshot.md`
  - `super-agent-playwright-mcp-8896-session-switch-backend-fix-after-click-snapshot.md`
  - `super-agent-playwright-mcp-8896-session-switch-backend-fix-ui.png`
- Console:
  - 0 errors.
  - 1 existing `single-spa` warning.

## 2026-06-19 23:12 CST: Session Create, Composer Height, Immediate Pending Reply

Scope:

- Added durable session creation behind `POST /api/super-agent/sessions/create`.
- Wired the super-agent session sidebar "new session" button to create a real conversation before selecting it.
- Removed the right-side chat header "布局" button.
- Fixed the right-bottom composer area height.
- Added immediate assistant pending feedback after send: an AI message row with a three-dot loading animation appears before streamed text arrives.

Local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -count=1
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-session-create-ui main.go
pnpm --filter @coze-agent-ide/bot-creator exec vitest --run src/modes/super-mode/__tests__/super-chat-area.test.tsx src/modes/super-mode/__tests__/super-session-sidebar.test.tsx src/modes/super-mode/codex-trace/__tests__/codex-trace-panel.test.tsx src/modes/super-mode/codex-trace/__tests__/trace-bridge.test.tsx
pnpm --filter @coze-studio/app build
git diff --check
```

Passed:

- `api/router/coze` tests.
- Linux backend build.
- Super-mode Vitest: 4 files, 16 tests.
- `@coze-studio/app` production build.
- `git diff --check`.

Known typecheck status:

```bash
pnpm --filter @coze-agent-ide/bot-creator exec tsc -p tsconfig.build.json --noEmit --pretty false
```

Still fails on existing repository-wide issues, primarily `TS6305` unbuilt referenced package outputs plus old single/workflow mode typing errors. No new `import.meta` or pending/composer implementation type error appeared in this run.

Backend deployment:

- Replaced `coze-super:/app/openynet`.
- New SHA-256: `3730210ba13751434e027b34b28d8b652b3a6568295449846ad5cefdb0f8b0df`.
- Kept one backend rollback point: `/app/openynet.bak-sessions-202606192300`.

Static deployment:

- First static deploy at `23:02` for session-create/layout-button work.
- Second static-only deploy at `23:10` for composer-height and pending-reply fixes.
- Kept one static rollback point after the second deploy: `/app/resources/static.bak-sessions-202606192310`.
- Removed uploaded archives from `/home/dev`.
- Disk after second deploy:
  - overlay total `71.8G`, used `53.7G`, available `14.4G`, use `79%`.
  - current static size `311.9M`.

Playwright MCP 8896 verification:

- Page:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=composer_pending_20260619_2311`
- Composer/layout DOM evidence:
  - `hasLayoutText = false`.
  - composer rect height reduced to `101px` (`y = 848`, `height = 101`).
  - prior bad capture for the same area was `217px`.
  - internal `[class*='chat-area-message-group-list']` is `display: none`, `height = 0`.
  - internal safe-area nodes are `display: none`, `height = 0`.
- Send/pending evidence:
  - Sent test content `FINAL_STREAM_OK`.
  - textarea cleared after send.
  - user text appeared in the trace panel.
  - assistant pending bubble appeared immediately.
  - `aria-label = 正在生成回复`.
  - pending bubble contained `3` dot spans.
- Answer transition evidence:
  - final answer appeared: `你好！有什么我可以帮你的吗？`
  - pending bubble disappeared after streamed answer text started.
- Network evidence:
  - `POST /api/conversation/chat => 200` for the send path.

Saved evidence:

- `super-agent-playwright-mcp-8896-composer-pending-before-send-dom.json`
- `super-agent-playwright-mcp-8896-composer-pending-before-send.png`
- `super-agent-playwright-mcp-8896-composer-pending-after-real-send-immediate.json`
- `super-agent-playwright-mcp-8896-composer-pending-after-send.png`
- `super-agent-playwright-mcp-8896-composer-pending-after-answer-start.json`
- `super-agent-playwright-mcp-8896-composer-pending-after-answer.png`
- `super-agent-playwright-mcp-8896-composer-pending-network.txt`

## 2026-06-19 23:32 - OpenAPI Schema Discovery

Scope:

- Added machine-readable App Server schema route `GET /api/super-agent/openapi.json`.
- Added the schema route and operation catalog to `GET /api/super-agent/manifest`.
- Kept discovery routes readable without a web session:
  - `/api/super-agent/manifest`
  - `/api/super-agent/openapi.json`

Local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/middleware -run TestSuperAgentDiscoveryContractsDoNotRequireSession -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -count=1
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-openapi main.go
```

Passed:

- `api/middleware` discovery session whitelist test.
- `api/router/coze` route tests, including OpenAPI and manifest contract assertions.
- Linux backend build.

Deployment:

- Replaced `coze-super:/app/openynet`.
- Current backend SHA-256: `69afaf2a0f69e7322dcd29f7b92f74862dec8726da1e3fafb478da7f371646f5`.
- Kept one backend rollback point: `/app/openynet.bak-openapi-202606192325`.
- Rollback SHA-256: `3730210ba13751434e027b34b28d8b652b3a6568295449846ad5cefdb0f8b0df`.
- Did not upload or replace static assets for this step.
- Disk after deploy:
  - overlay total `71.8G`, used `53.6G`, available `14.5G`, use `79%`.

Initial real URL check:

- `GET http://10.10.10.226:8896/api/super-agent/manifest => 200`
  - `code = 0`
  - `external_api.schema_route = GET /api/super-agent/openapi.json`
  - `openapi.schema_route = GET /api/super-agent/openapi.json`
  - `openapi.operations.length = 49`
- `GET http://10.10.10.226:8896/api/super-agent/openapi.json => 200`
  - `openapi = 3.1.0`
  - `info.title = Super Agent App Server API`
  - `info.version = super-agent.app-server.v1`
  - `components.securitySchemes.BearerAuth` exists.
  - `paths` includes:
    - `/api/super-agent/runs/create`
    - `/api/super-agent/runs/stream`
    - `/api/super-agent/sessions/create`
    - `/api/super-agent/workspace/patch`
    - `/api/super-agent/skills/import`
    - `/api/super-agent/skills/list`

Playwright MCP verification:

- Browser page:
  - `http://10.10.10.226:8896/api/super-agent/openapi.json`
- Browser-context `fetch` results:
  - `/api/super-agent/manifest`: `status = 200`, `ok = true`.
  - `/api/super-agent/openapi.json`: `status = 200`, `ok = true`.
  - Manifest operation catalog contains `runs.create` and `skills.import`.
  - OpenAPI `RunsCreateRequest` uses `$ref = #/components/schemas/RunsCreateRequest`.
  - `RunsCreateRequest.x-required-one-of = [["agent_id","bot_id"]]`.
  - `skills.list` GET exposes required query parameter `space_id`.

Saved evidence:

- `super-agent-playwright-mcp-8896-openapi-curl-final.json`
- `super-agent-playwright-mcp-8896-openapi-browser-final.json`
- `super-agent-playwright-mcp-8896-openapi-network-final.txt`
- `super-agent-playwright-mcp-8896-openapi-page-final.png`

## 2026-06-19 23:58 - Session Delete and Immediate AI Pending Bubble

Scope:

- Added durable session deletion:
  - `POST /api/super-agent/sessions/delete`
  - request schema requires `conversation_id`.
  - manifest `sessions.delete_route` and `external_api.entry_routes["sessions.delete"]`.
  - OpenAPI path `/api/super-agent/sessions/delete`.
- Added UI delete action in the super-agent session sidebar with confirmation modal.
- Verified the chat trace renders an assistant pending bubble immediately after sending:
  - `aria-label="正在生成回复"`.
  - three animated dot spans.
  - hidden once assistant answer text starts streaming.

Local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgentSessionDeleteRouteDeletesOwnedSession|TestSuperAgentManifestRouteReturnsAppServerContract|TestSuperAgentOpenAPIRouteReturnsMachineReadableSchema' -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/middleware ./api/router/coze -count=1
pnpm --filter @coze-arch/bot-api exec vitest --run __tests__/developer-api-super-agent-workspace.test.ts __tests__/developer-api-super-agent-skills.test.ts
pnpm --filter @coze-agent-ide/bot-creator exec vitest --run src/modes/super-mode/__tests__/super-session-sidebar.test.tsx src/modes/super-mode/codex-trace/__tests__/codex-trace-panel.test.tsx src/modes/super-mode/codex-trace/__tests__/trace-bridge.test.tsx src/modes/super-mode/__tests__/super-chat-area.test.tsx
pnpm --filter @coze-agent-ide/bot-creator exec eslint src/modes/super-mode/super-session-sidebar.tsx src/modes/super-mode/super-session-sidebar.module.less src/modes/super-mode/codex-trace/codex-trace-panel.tsx src/modes/super-mode/codex-trace/codex-trace.module.less src/modes/super-mode/codex-trace/trace-bridge.tsx src/modes/super-mode/codex-trace/trace-store.ts --cache --quiet
pnpm --filter @coze-arch/bot-api exec eslint __tests__/developer-api-super-agent-workspace.test.ts --cache --quiet
pnpm --filter @coze-studio/app build
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-sessions-delete main.go
git diff --check -- <changed super-agent files>
```

Passed:

- Backend route, manifest, OpenAPI, and middleware/router tests.
- Frontend super-mode trace/sidebar/chat tests.
- Bot API SDK tests for super-agent workspace and skills.
- ESLint for touched super-mode files and the touched bot-api test.
- Frontend `@coze-studio/app` production build.
- Linux backend build.

Deployment:

- Replaced `coze-super:/app/openynet`.
- Current backend SHA-256: `c943a839eaf71b1a17e2690fd26266a5b46a1a973ceafd58ed5346fd455e46ea`.
- Kept one backend rollback point: `/app/openynet.bak-latest`.
- Rollback SHA-256: `69afaf2a0f69e7322dcd29f7b92f74862dec8726da1e3fafb478da7f371646f5`.
- Replaced `coze-super:/app/resources/static`.
- Kept one static rollback point: `/app/resources/static.bak-latest`.
- Removed older named backups before replacement:
  - `/app/openynet.bak-openapi-202606192325`
  - `/app/resources/static.bak-sessions-202606192310`
- Disk after deploy:
  - overlay total `71.8G`, used `53.8G`, available `14.3G`, use `79%`.

Initial real URL check:

- `GET http://10.10.10.226:8896/api/super-agent/manifest => 200`
  - `openapi.operations.length = 50`
  - `sessions.delete_route = POST /api/super-agent/sessions/delete`
  - `external_api.entry_routes["sessions.delete"] = POST /api/super-agent/sessions/delete`

Playwright MCP verification:

- Browser page:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=session_delete_pending_20260619_2357`
- Immediate pending bubble:
  - Filled the bottom composer with: `请只回复 FINAL_STREAM_OK，用于验证发送后立即出现 AI 输出框和三点动画。不要创建、修改或删除任何文件。`
  - 200ms after clicking send:
    - `hasPending = true`
    - `pendingDots = 3`
    - `hasUserMessage = true`
  - After the answer streamed:
    - `hasPending = false`
    - Assistant output included `FINAL_STREAM_OK`.
- Session delete UI:
  - Before create: `count = 9`.
  - Clicked `aria-label="新会话"`.
  - After create: `count = 10`, new item title `新会话`.
  - Clicked that item's `aria-label="删除会话"`.
  - Confirmation modal text: `删除会话确定要删除「新会话」吗？取消删除`.
  - Clicked modal `删除`.
  - After confirm: `count = 9`, `hasNewSession = false`, `modalStillOpen = false`.
- Browser-context schema check:
  - `/api/super-agent/manifest`: `status = 200`.
  - `/api/super-agent/openapi.json`: `status = 200`.
  - `manifestDeleteRoute = POST /api/super-agent/sessions/delete`.
  - `openapiHasPath = true`.
  - `requestSchema = #/components/schemas/SessionsDeleteRequest`.
  - `schemaRequired = ["conversation_id"]`.

Saved evidence:

- `super-agent-playwright-mcp-8896-session-delete-pending-initial.json`
- `super-agent-playwright-mcp-8896-composer-dom-after-deploy.json`
- `super-agent-playwright-mcp-8896-composer-after-fill.json`
- `super-agent-playwright-mcp-8896-session-delete-pending-after-send-immediate.json`
- `super-agent-playwright-mcp-8896-session-delete-pending-after-answer.json`
- `super-agent-playwright-mcp-8896-pending-after-deploy.png`
- `super-agent-playwright-mcp-8896-session-delete-before-create.json`
- `super-agent-playwright-mcp-8896-session-delete-after-create.json`
- `super-agent-playwright-mcp-8896-session-delete-created-diff.json`
- `super-agent-playwright-mcp-8896-session-delete-modal.json`
- `super-agent-playwright-mcp-8896-session-delete-after-confirm.json`
- `super-agent-playwright-mcp-8896-session-delete-after-confirm.png`
- `super-agent-playwright-mcp-8896-session-delete-openapi-final.json`

## 2026-06-20 00:09 CST - Composer Pending Reply Feedback

Scope:

- Re-verified the chat composer feedback requested for the super-agent arrange page: after clicking send, an assistant reply container appears immediately with a three-dot typing animation, then disappears once streamed answer text arrives.
- No code or deployment changes were made in this check.

Local regression:

```bash
pnpm --filter @coze-agent-ide/bot-creator exec vitest --run src/modes/super-mode/codex-trace/__tests__/codex-trace-panel.test.tsx src/modes/super-mode/codex-trace/__tests__/trace-bridge.test.tsx
```

Passed:

- `trace-bridge.test.tsx`: 2 tests.
- `codex-trace-panel.test.tsx`: 8 tests.
- Total: 10 tests passed.

Playwright MCP verification:

- Browser page:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=pending_reply_20260620_0007`
- Viewport:
  - `2048 x 1280`.
- Input:
  - Used the composer textarea with placeholder `继续对话...`.
  - Sent: `请只回复 PENDING_DOTS_OK_1781885357363，不要创建修改删除文件。`
- 120ms after clicking `data-testid="bot-home-chart-send-button"`:
  - `typingVisible = true`
  - `typingDotCount = 3`
  - `userVisible = true`
- 370ms after clicking send:
  - `typingVisible = true`
  - `typingDotCount = 3`
- After streamed answer arrived:
  - `typingVisible = false`
  - Assistant output included `PENDING_DOTS_OK_1781885357363`.

Saved evidence:

- `super-agent-playwright-mcp-8896-pending-reply-dom-before-send.json`
- `super-agent-playwright-mcp-8896-pending-reply-fill-debug.json`
- `super-agent-playwright-mcp-8896-pending-reply-precise-send-final.json`

## 2026-06-20 00:22 CST - Trace External API Discovery

Scope:

- Exposed `traces.get` in the super-agent App Server discovery contract.
- The actual trace route and OpenAPI path already existed; this change makes external clients discover it through `manifest.data.external_api`.

Local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run TestSuperAgentManifestRouteReturnsAppServerContract -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/middleware ./api/router/coze -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/handler/coze/superagentapp -count=1
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-traces-external-api main.go
```

Passed:

- Manifest App Server contract test.
- Backend middleware and super-agent route tests.
- App Server request helper tests.
- Linux backend build.

Deployment:

- Replaced `coze-super:/app/openynet`.
- Current backend SHA-256: `8091b29bb57ae75f3b95b682e56e9d48e27451280ede3062e13b59ecf77b4434`.
- Kept one backend rollback point: `/app/openynet.bak-latest`.
- Rollback SHA-256: `c943a839eaf71b1a17e2690fd26266a5b46a1a973ceafd58ed5346fd455e46ea`.
- Static resources were not changed.
- Disk after deploy:
  - `/dev/mapper/ubuntu--vg-ubuntu--lv` total `72G`, used `54G`, available `15G`, use `79%`.

HTTP verification:

- `GET http://10.10.10.226:8896/api/super-agent/manifest => 200`
- `GET http://10.10.10.226:8896/api/super-agent/openapi.json => 200`
- Manifest:
  - `capabilities` includes `traces`.
  - `external_api.capabilities` includes `traces`.
  - `external_api.entry_routes["traces.get"] = POST /api/super-agent/traces/get`.
  - `external_api.request_schemas["traces.get"].required = ["conversation_id"]`.
  - `external_api.request_schemas["traces.get"].optional` includes `space_id`, `agent_id`, `bot_id`, `run_id`, `limit`.
  - `external_api.client_metadata.capabilities_value = sessions,sandbox,workspace,skills,harness,approvals,traces`.
- OpenAPI:
  - `paths["/api/super-agent/traces/get"]` exists.
  - `components.schemas.TracesGetRequest.required = ["conversation_id"]`.

Playwright MCP verification:

- Browser page:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=pending_reply_20260620_0007`
- Browser-context fetch:
  - `/api/super-agent/manifest`: `status = 200`.
  - `/api/super-agent/openapi.json`: `status = 200`.
  - `capabilitiesHasTraces = true`.
  - `externalHasTraces = true`.
  - `traceEntry = POST /api/super-agent/traces/get`.
  - `traceSchema.required = ["conversation_id"]`.
  - `traceSchema.optional = ["space_id", "agent_id", "bot_id", "run_id", "limit"]`.
  - `capabilitiesValue = sessions,sandbox,workspace,skills,harness,approvals,traces`.
  - `openapiHasTracePath = true`.

Saved evidence:

- `super-agent-playwright-mcp-8896-traces-external-api-final.json`

## 2026-06-20 00:31 CST - External API Full Operation Discovery

Scope:

- Made `manifest.data.external_api.entry_routes` and `manifest.data.external_api.request_schemas` cover every operation declared by `manifest.data.openapi.operations`.
- Kept existing short run aliases (`create`, `stream`, `reply`, `get`, `list`, `cancel`) for backward compatibility.
- Added `artifacts` to the App Server capability declarations and propagated it through the capabilities metadata string.

Local TDD verification:

1. Added a failing manifest contract assertion that every OpenAPI `operation_id` must be present in `external_api.entry_routes` and `external_api.request_schemas`.
2. Verified RED:
   - The test failed because routes such as `skills.marketplace.list`, `skills.marketplace.get`, and `skills.marketplace.install` were absent from `external_api`.
3. Implemented `superAgentExternalAPIEntryRoutes()` and `superAgentExternalAPIRequestSchemas()` from the OpenAPI operation/schema source.
4. Verified GREEN:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run TestSuperAgentManifestRouteReturnsAppServerContract -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/middleware ./api/router/coze -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/handler/coze/superagentapp -count=1
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-external-api-full-discovery main.go
```

Deployment:

- Replaced `coze-super:/app/openynet`.
- Current backend SHA-256: `0155c4c67bae9cf5147bec93effee8dde096b15735ae7988a8f5e480d90f9d78`.
- Kept one backend rollback point: `/app/openynet.bak-latest`.
- Rollback SHA-256: `8091b29bb57ae75f3b95b682e56e9d48e27451280ede3062e13b59ecf77b4434`.
- Static resources were not changed.
- Disk after deploy:
  - `/dev/mapper/ubuntu--vg-ubuntu--lv` total `72G`, used `54G`, available `15G`, use `79%`.

HTTP verification:

- `GET http://10.10.10.226:8896/api/super-agent/manifest => 200`
- `GET http://10.10.10.226:8896/api/super-agent/openapi.json => 200`
- Manifest:
  - `operation_count = 50`
  - `external_entry_count = 56`
  - `external_schema_count = 56`
  - `missing_entry = []`
  - `missing_schema = []`
  - `wrong_routes = []`
  - `external_api.capabilities = ["sessions","sandbox","workspace","skills","harness","artifacts","approvals","traces"]`
  - `external_api.client_metadata.capabilities_value = sessions,sandbox,workspace,skills,harness,artifacts,approvals,traces`
  - Sample operation routes:
    - `runs.create = POST /api/super-agent/runs/create`
    - `workspace.patch = POST /api/super-agent/workspace/patch`
    - `sandbox.exec = POST /api/super-agent/sandbox/exec`
    - `artifacts.list = POST /api/super-agent/artifacts/list`
    - `skills.marketplace.install = POST /api/super-agent/marketplace/install`

Playwright MCP verification:

- Browser page:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=pending_reply_20260620_0007`
- Browser-context fetch:
  - `/api/super-agent/manifest`: `status = 200`.
  - `/api/super-agent/openapi.json`: `status = 200`.
  - `operationCount = 50`.
  - `externalEntryCount = 56`.
  - `externalSchemaCount = 56`.
  - `missingEntry = []`.
  - `missingSchema = []`.
  - `wrongRoutes = []`.
  - `openapiHasMarketplaceInstall = true`.

Saved evidence:

- `super-agent-playwright-mcp-8896-external-api-full-discovery-final.json`

## 2026-06-20 00:34 CST - Harness Plan Items Alias + App Server E2E

Scope:

- Verified a real external App Server flow from the arrange page browser context:
  - Manifest discovery.
  - Workspace write/read.
  - Harness plan update/state read.
  - Sandbox command execution.
  - Artifact list/download/delete.
  - Workspace cleanup.
- Found and fixed a contract mismatch:
  - Manifest/OpenAPI exposed `HarnessPlanRequest.required = ["items"]`.
  - Backend request model only accepted `plan`.
  - External clients that followed discovery and sent `items` received `invalid parameter : plan must contain at least one step`.

Fix:

- `SuperAgentHarnessPlanUpdateRequest` now accepts both:
  - `plan`
  - `items`
- `UpdateSuperAgentHarnessPlan` uses `plan` first, then falls back to `items`.
- Manifest/OpenAPI schema for `harness.plan` now advertises:
  - `required = []`
  - `required_one_of = [["plan","items"]]`
- Frontend SDK type and request mapping now preserve the `items` alias.

Local TDD verification:

1. Added an application-layer regression test that unmarshals a request body with `items` and calls `UpdateSuperAgentHarnessPlan`.
2. Verified RED:
   - `go test ./application/singleagent -run TestUpdateSuperAgentHarnessPlanWritesToolCompatiblePlan -count=1`
   - Failed with `plan must contain at least one step`.
3. Implemented alias handling and schema update.
4. Verified GREEN:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent -run TestUpdateSuperAgentHarnessPlanWritesToolCompatiblePlan -count=1
pnpm --filter @coze-arch/bot-api exec vitest --run __tests__/developer-api-super-agent-workspace.test.ts
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgentManifestRouteReturnsAppServerContract|TestSuperAgentOpenAPIRouteReturnsSchema' -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/middleware ./api/router/coze ./application/singleagent -count=1
pnpm --filter @coze-arch/bot-api exec vitest --run __tests__/developer-api-super-agent-workspace.test.ts __tests__/bot-open-api-super-agent-manifest.test.ts
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-harness-plan-items-alias main.go
```

Deployment:

- Replaced `coze-super:/app/openynet`.
- Current backend SHA-256: `b05241a337bae3b56da971d109ccb05ba7b3fd1c8de8d0ee3fd6a0df572d075d`.
- Kept one backend rollback point: `/app/openynet.bak-latest`.
- Rollback SHA-256: `0155c4c67bae9cf5147bec93effee8dde096b15735ae7988a8f5e480d90f9d78`.
- Static resources were not changed.
- Disk after deploy:
  - `/dev/mapper/ubuntu--vg-ubuntu--lv` total `72G`, used `54G`, available `15G`, use `79%`.

HTTP verification:

- `GET http://10.10.10.226:8896/api/super-agent/manifest => 200`
- `external_api.request_schemas["harness.plan"]`:
  - `required = []`
  - `required_one_of = [["plan","items"]]`
  - `optional = ["space_id","agent_id","bot_id","connector_id"]`

Playwright MCP verification:

- Browser page:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=pending_reply_20260620_0007`
- E2E request body used:
  - `bot_id = 7652617174313336832`
  - `space_id = 7652614054615187456`
  - `harness.plan` sent `items`, not `plan`.
- Checks:
  - `schemaHasItemsAlias = true`
  - `hasAllDiscoveryRoutes = true`
  - `writeOk = true`
  - `readOk = true`
  - `planOk = true`
  - `stateOk = true`
  - `execOk = true`
  - `listOk = true`
  - `downloadOk = true`
  - `cleanupArtifactOk = true`
  - `cleanupWorkspaceOk = true`
- Temporary paths were cleaned:
  - `/outputs/appserver-e2e-items-1781886785996.txt`
  - `/workspace/appserver-e2e-items-1781886785996.txt`

Saved evidence:

- Failed pre-fix probe: `super-agent-playwright-mcp-8896-appserver-e2e-workspace-sandbox-artifact-final.json`
- Fixed final probe: `super-agent-playwright-mcp-8896-appserver-e2e-items-alias-final.json`

## 2026-06-20 00:44 CST - Workspace Move Alias + Pending Reply Confirmation

Scope:

- Fixed and verified a second external App Server contract mismatch:
  - Manifest/OpenAPI advertised `workspace.move` request fields as `from_path` and `to_path`.
  - Backend and generated frontend SDK only accepted `path` and `target_path`.
  - External clients that followed manifest discovery could send an apparently valid request and still fail backend path validation.
- Reconfirmed the super-agent chat waiting experience:
  - After send, an assistant output placeholder appears immediately.
  - The placeholder shows three animated dots.
  - It disappears after answer text starts streaming.

Fix:

- `MoveSandboxFileRequest` now accepts both naming styles:
  - `path` / `target_path`
  - `from_path` / `to_path`
- `MoveSuperAgentWorkspaceFile` prefers canonical fields and falls back to manifest-compatible aliases.
- Manifest/OpenAPI schema for `workspace.move` now advertises:
  - `required = []`
  - `required_one_of = [["path","from_path"],["target_path","to_path"]]`
- Frontend SDK type and request mapping now preserve `from_path` and `to_path`.

Local TDD verification:

1. Added an application-layer regression test that unmarshals a request body with `from_path` and `to_path`, then calls `MoveSuperAgentWorkspaceFile`.
2. Verified RED:
   - `go test ./application/singleagent -run TestMoveSuperAgentWorkspaceFileRenamesOnlyWritableRoots -count=1`
   - Failed with `cannot move root directory`.
3. Implemented alias handling and schema update.
4. Verified GREEN:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent -run TestMoveSuperAgentWorkspaceFileRenamesOnlyWritableRoots -count=1
pnpm --filter @coze-arch/bot-api exec vitest --run __tests__/developer-api-super-agent-workspace.test.ts
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgentManifestRouteReturnsAppServerContract|TestSuperAgentOpenAPIRouteReturnsSchema' -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/middleware ./api/router/coze ./application/singleagent -count=1
pnpm --filter @coze-arch/bot-api exec vitest --run __tests__/developer-api-super-agent-workspace.test.ts __tests__/bot-open-api-super-agent-manifest.test.ts
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-workspace-move-alias main.go
```

Deployment:

- Replaced `coze-super:/app/openynet`.
- Current backend SHA-256: `f1a73f9ae00510be29d910d1bd4c027bf2af1b751745ac89d4a5e05eac10b94c`.
- Kept one backend rollback point: `/app/openynet.bak-latest`.
- Rollback SHA-256: `b05241a337bae3b56da971d109ccb05ba7b3fd1c8de8d0ee3fd6a0df572d075d`.
- Static resources were not changed.
- Disk after deploy:
  - `/dev/mapper/ubuntu--vg-ubuntu--lv` total `72G`, used `54G`, available `15G`, use `79%`.

HTTP verification:

- `GET http://10.10.10.226:8896/api/super-agent/manifest => 200`
- `external_api.request_schemas["workspace.move"]`:
  - `required = []`
  - `required_one_of = [["path","from_path"],["target_path","to_path"]]`
  - `optional = ["space_id","agent_id","bot_id","connector_id"]`

Playwright MCP verification:

- Browser page:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=pending_reply_20260620_0007`
- E2E request body used:
  - `bot_id = 7652617174313336832`
  - `space_id = 7652614054615187456`
  - `workspace.move` sent `from_path` and `to_path`, not `path` and `target_path`.
- Checks:
  - `schemaHasFromToAlias = true`
  - `writeOk = true`
  - `moveOk = true`
  - `readMovedOk = true`
  - `oldPathMissing = true`
  - `cleanupMovedOk = true`
- Temporary paths were cleaned:
  - `/workspace/move-alias-1781887354751.txt`
  - `/outputs/move-alias-1781887354751.txt`

Pending reply verification:

- Local test:
  - `pnpm --filter @coze-agent-ide/bot-creator exec vitest --run src/modes/super-mode/codex-trace/__tests__/codex-trace-panel.test.tsx src/modes/super-mode/codex-trace/__tests__/trace-bridge.test.tsx`
  - Result: `2` files passed, `10` tests passed.
- Saved probe: `super-agent-playwright-mcp-8896-pending-reply-precise-send-final.json`
- Checks:
  - Before click: `typingVisible = false`, `typingDotCount = 0`.
  - 120ms after send: `typingVisible = true`, `typingDotCount = 3`.
  - 370ms after send: `typingVisible = true`, `typingDotCount = 3`.
  - After answer starts: `typingVisible = false`, `typingDotCount = 0`.

Saved evidence:

- `super-agent-playwright-mcp-8896-workspace-move-alias-final.json`
- `super-agent-playwright-mcp-8896-pending-reply-precise-send-final.json`

## 2026-06-20 02:52 CST - Workspace Write Encoding Base64 Contract

Scope:

- Fixed and verified another external App Server contract mismatch:
  - Manifest/OpenAPI advertised `workspace.write` and `workspace.upload` with optional `encoding`.
  - Backend request model only honored `is_base64`.
  - External clients that followed manifest and sent `encoding = "base64"` wrote the base64 text itself instead of decoded bytes.
- Kept backward compatibility with `is_base64`.

Root cause:

- `UploadSandboxFileRequest` had no `encoding` field, so JSON binding ignored the manifest-compatible field.
- `UploadSandboxFile` only decoded when `IsBase64 = true`.
- Generated TypeScript SDK did not pass `encoding` through the super-agent workspace write/upload request bodies.

Fix:

- `UploadSandboxFileRequest` now accepts `encoding`.
- `UploadSandboxFile` decodes content when either:
  - `is_base64 = true`
  - `encoding = "base64"`
- `workspace.write` and `workspace.upload` request schemas now list both `encoding` and `is_base64`.
- Generated TypeScript SDK type and request mapping now preserve `encoding`.

Local TDD verification:

1. Added a Go regression test that unmarshals an App Server-style request body with `encoding = "base64"` and no `is_base64`, then calls `UploadSuperAgentWorkspaceFile`.
2. Verified RED:
   - `SESSION_HMAC_SECRET=test-secret go test ./application/singleagent -run TestUploadSuperAgentWorkspaceFileAcceptsManifestEncodingBase64 -count=1`
   - Failed because the sandbox received `AAECA/8=` bytes instead of decoded bytes `{0,1,2,3,255}`.
3. Added a TypeScript SDK regression expectation that `SuperAgentWriteWorkspaceFile` passes `encoding`.
4. Verified RED:
   - `pnpm --filter @coze-arch/bot-api exec vitest --run __tests__/developer-api-super-agent-workspace.test.ts`
   - Failed because the request body omitted `encoding`.
5. Implemented the fix and verified GREEN:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent -run TestUploadSuperAgentWorkspaceFileAcceptsManifestEncodingBase64 -count=1
pnpm --filter @coze-arch/bot-api exec vitest --run __tests__/developer-api-super-agent-workspace.test.ts
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgentManifestRouteReturnsAppServerContract|TestSuperAgentOpenAPIRouteReturnsSchema' -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/middleware ./api/router/coze ./application/singleagent -count=1
pnpm --filter @coze-arch/bot-api exec vitest --run __tests__/developer-api-super-agent-workspace.test.ts __tests__/bot-open-api-super-agent-manifest.test.ts
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-workspace-encoding main.go
```

Deployment:

- Replaced `coze-super:/app/openynet`.
- Current backend SHA-256: `3f8bb626688845d67d35bc3151a1a222a86cc306089ac6c1e420df133b525f26`.
- Kept one backend rollback point: `/app/openynet.bak-latest`.
- Rollback SHA-256: `f1a73f9ae00510be29d910d1bd4c027bf2af1b751745ac89d4a5e05eac10b94c`.
- Removed uploaded host temp file:
  - `/tmp/openynet-workspace-encoding`
- Static resources were not changed.
- Disk after deploy and cleanup:
  - `/dev/mapper/ubuntu--vg-ubuntu--lv` total `72G`, used `54G`, available `15G`, use `80%`.

Playwright MCP verification:

- Browser page:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=workspace_encoding_20260620_0250`
- E2E request body used:
  - `bot_id = 7652617174313336832`
  - `space_id = 7652614054615187456`
  - `workspace.write` sent `encoding = "base64"` and did not send `is_base64`.
- Temporary file:
  - `/workspace/encoding-1781895123913.bin`
- Checks:
  - `manifestStatus = 200`
  - `writeSchemaHasEncoding = true`
  - `writeSchemaHasIsBase64 = true`
  - `uploadSchemaHasEncoding = true`
  - `uploadSchemaHasIsBase64 = true`
  - `writeOk = true`
  - `readOk = true`
  - `readIsBinary = true`
  - `readBase64Matches = true`
  - `cleanupOk = true`

Saved evidence:

- `super-agent-playwright-mcp-8896-workspace-encoding-final.json`

## 2026-06-20 03:05 CST - Workspace Grep Filters Contract

Scope:

- Fixed and verified another external App Server contract mismatch:
  - Manifest/OpenAPI advertised `workspace.grep` with optional `case_sensitive`, `include`, and `exclude`.
  - Backend request model ignored those fields.
  - Service only called the lower-level `Grep(pattern, path)`, which cannot apply include/exclude filters.
  - Generated TypeScript SDK did not pass the fields through.
- Kept backward compatibility:
  - Requests without grep filter options still use the existing sandbox `Grep` path.
  - Requests with any filter option use an enhanced sandbox command.

Root cause:

- `GrepSandboxFilesRequest` only contained `path` and `pattern`.
- `GrepSuperAgentWorkspace` had no branch for manifest-level filtering options.
- `SuperAgentGrepWorkspace` SDK request body omitted `case_sensitive`, `include`, and `exclude`.

Fix:

- `GrepSandboxFilesRequest` now accepts:
  - `case_sensitive`
  - `include`
  - `exclude`
- Filtered grep requests now build a quoted shell command:
  - Prefer `rg -n --no-heading`.
  - Fall back to `grep -RIn`.
  - `case_sensitive = false` maps to `-i`.
  - `include` maps to `rg -g` or `grep --include`.
  - `exclude` maps to `rg -g !...` or `grep --exclude`.
- Generated TypeScript SDK type and request mapping now preserve the filter fields.

Local TDD verification:

1. Added a Go regression test that unmarshals an App Server-style request body with:
   - `case_sensitive = false`
   - `include = ["*.md"]`
   - `exclude = ["draft*"]`
2. Verified RED:
   - `SESSION_HMAC_SECRET=test-secret go test ./application/singleagent -run TestGrepSuperAgentWorkspaceHonorsManifestFilters -count=1`
   - Failed because the filtered grep command was never used.
3. Added a TypeScript SDK regression expectation that `SuperAgentGrepWorkspace` passes the three fields.
4. Verified RED:
   - `pnpm --filter @coze-arch/bot-api exec vitest --run __tests__/developer-api-super-agent-workspace.test.ts`
   - Failed because the request body omitted `case_sensitive`, `include`, and `exclude`.
5. Implemented the fix and verified GREEN:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent -run TestGrepSuperAgentWorkspaceHonorsManifestFilters -count=1
pnpm --filter @coze-arch/bot-api exec vitest --run __tests__/developer-api-super-agent-workspace.test.ts
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgentManifestRouteReturnsAppServerContract|TestSuperAgentOpenAPIRouteReturnsSchema' -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/middleware ./api/router/coze ./application/singleagent -count=1
pnpm --filter @coze-arch/bot-api exec vitest --run __tests__/developer-api-super-agent-workspace.test.ts __tests__/bot-open-api-super-agent-manifest.test.ts
pnpm --filter @coze-arch/bot-api exec tsc --noEmit
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-workspace-grep-filters main.go
```

Deployment:

- Replaced `coze-super:/app/openynet`.
- Current backend SHA-256: `3419f2aa27461ab7503837274823fb615cf0cce10d8854701160fee19d2842f1`.
- Kept one backend rollback point: `/app/openynet.bak-latest`.
- Rollback SHA-256: `3f8bb626688845d67d35bc3151a1a222a86cc306089ac6c1e420df133b525f26`.
- Removed uploaded host temp file:
  - `/tmp/openynet-workspace-grep-filters`
- Static resources were not changed.
- Disk after deploy and cleanup:
  - `/dev/mapper/ubuntu--vg-ubuntu--lv` total `72G`, used `54G`, available `15G`, use `80%`.

Playwright MCP verification:

- Browser page:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=workspace_grep_filters_20260620_0302`
- Temporary files:
  - `/workspace/grep-filter-1781895884215/match.md`
  - `/workspace/grep-filter-1781895884215/skip.md`
  - `/workspace/grep-filter-1781895884215/notes.txt`
- E2E request body used:
  - `bot_id = 7652617174313336832`
  - `space_id = 7652614054615187456`
  - `path = /workspace/grep-filter-1781895884215`
  - `pattern = filtertoken1781895884215`
  - `case_sensitive = false`
  - `include = ["*.md"]`
  - `exclude = ["skip*"]`
- Checks:
  - `manifestStatus = 200`
  - `schemaHasCaseSensitive = true`
  - `schemaHasInclude = true`
  - `schemaHasExclude = true`
  - `mkdirOk = true`
  - `writeMatchOk = true`
  - `writeExcludedOk = true`
  - `writeWrongTypeOk = true`
  - `grepOk = true`
  - `caseInsensitiveMatched = true`
  - `excludeFiltered = true`
  - `includeFiltered = true`
  - `cleanupOk = true`
- Grep output:
  - `/workspace/grep-filter-1781895884215/match.md:1:visible FILTERTOKEN1781895884215`

Saved evidence:

- `super-agent-playwright-mcp-8896-workspace-grep-filters-final.json`

## 2026-06-20 03:16 CST - Workspace List Recursive Contract

Scope:

- Fixed and verified another external App Server contract mismatch:
  - Manifest/OpenAPI advertised `workspace.list` with optional `recursive`.
  - Backend request model ignored `recursive`.
  - Service only listed one directory level.
  - Generated TypeScript SDK did not pass `recursive` through the super-agent workspace list request.
- Kept backward compatibility:
  - Requests without `recursive` still list only the immediate directory entries.
  - Requests with `recursive = true` return nested directory and file entries with full paths.

Root cause:

- `ListSandboxFilesRequest` had no `recursive` field.
- `listSandboxFilesByPath` always used `os.listdir`.
- `SuperAgentListWorkspaceFiles` SDK request body omitted `recursive`.

Fix:

- `ListSandboxFilesRequest` now accepts `recursive`.
- `listSandboxFilesByPath` uses:
  - `os.listdir` for the existing one-level behavior.
  - `os.walk` when `recursive = true`.
- Recursive listing preserves full path metadata for nested entries.
- Generated TypeScript SDK type and request mapping now preserve `recursive`.

Local TDD verification:

1. Added a Go regression test that unmarshals an App Server-style request body with `recursive = true`, then calls `ListSuperAgentWorkspaceFiles`.
2. Verified RED:
   - `SESSION_HMAC_SECRET=test-secret go test ./application/singleagent -run TestSuperAgentWorkspaceListHonorsManifestRecursive -count=1`
   - Failed because the list command used `os.listdir` and omitted `/workspace/reports/final.md`.
3. Added a TypeScript SDK regression expectation that `SuperAgentListWorkspaceFiles` passes `recursive`.
4. Verified RED:
   - `pnpm --filter @coze-arch/bot-api exec vitest --run __tests__/developer-api-super-agent-workspace.test.ts`
   - Failed because the request body omitted `recursive`.
5. Implemented the fix and verified GREEN:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent -run TestSuperAgentWorkspaceListHonorsManifestRecursive -count=1
pnpm --filter @coze-arch/bot-api exec vitest --run __tests__/developer-api-super-agent-workspace.test.ts
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgentManifestRouteReturnsAppServerContract|TestSuperAgentOpenAPIRouteReturnsSchema' -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/middleware ./api/router/coze ./application/singleagent -count=1
pnpm --filter @coze-arch/bot-api exec vitest --run __tests__/developer-api-super-agent-workspace.test.ts __tests__/bot-open-api-super-agent-manifest.test.ts
pnpm --filter @coze-arch/bot-api exec tsc --noEmit
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-workspace-list-recursive main.go
```

Deployment:

- Replaced `coze-super:/app/openynet`.
- Current backend SHA-256: `05a1a8db1d1f1e2be461bb1d34be4bf85d9dbc65b67a321a04495102a1e14959`.
- Kept one backend rollback point: `/app/openynet.bak-latest`.
- Rollback SHA-256: `3419f2aa27461ab7503837274823fb615cf0cce10d8854701160fee19d2842f1`.
- Removed uploaded host temp file:
  - `/tmp/openynet-workspace-list-recursive`
- Static resources were not changed.
- Disk after deploy and cleanup:
  - `/dev/mapper/ubuntu--vg-ubuntu--lv` total `72G`, used `55G`, available `14G`, use `80%`.

Playwright MCP verification:

- Browser page:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=workspace_list_recursive_20260620_0314`
- Temporary files:
  - `/workspace/list-recursive-1781896578693/root.md`
  - `/workspace/list-recursive-1781896578693/nested/final.md`
- E2E request bodies:
  - Non-recursive: `workspace.list` with `path = /workspace/list-recursive-1781896578693`.
  - Recursive: `workspace.list` with `path = /workspace/list-recursive-1781896578693`, `recursive = true`.
- Checks:
  - `manifestStatus = 200`
  - `schemaHasRecursive = true`
  - `mkdirRootOk = true`
  - `mkdirNestedOk = true`
  - `writeRootOk = true`
  - `writeChildOk = true`
  - `nonRecursiveOk = true`
  - `recursiveOk = true`
  - `nonRecursiveHasRootFile = true`
  - `nonRecursiveHasNestedDir = true`
  - `nonRecursiveOmitsNestedFile = true`
  - `recursiveHasRootFile = true`
  - `recursiveHasNestedDir = true`
  - `recursiveHasNestedFile = true`
  - `cleanupOk = true`

Saved evidence:

- `super-agent-playwright-mcp-8896-workspace-list-recursive-final.json`

## 2026-06-20 03:30 CST - Pending Assistant Bubble Before Message Mirror

Scope:

- Fix the arrange chat panel so sending a message immediately creates an AI reply area with a three-dot typing animation, even if the live chat message list has not mirrored the just-sent user message yet.
- Keep the placeholder only until real assistant text/tool output arrives.

Root cause:

- The super-agent arrange page hides the native ChatArea message list and mirrors ChatArea state into `CodexTracePanel`.
- `CodexTracePanel` already showed the typing placeholder when there was a user turn plus `pendingReply`.
- It still rendered the empty state when `pendingReply = true` and `messages = []`, which can happen in the first render right after send before the message store mirror catches up.

Fix:

- `CodexTracePanel` now creates a transient `pending-reply` turn when `pendingReply` is true and there are no derived turns yet.
- The existing `aria-label="正在生成回复"` typing bubble with three animated dots is reused.

Local TDD verification:

1. Added a RED test:
   - `CodexTracePanel > shows an assistant typing placeholder even before the sent message is mirrored`
   - It set `messages = []` and `pendingReply = true`.
   - It failed because the empty state was still visible.
2. Implemented the transient pending turn.
3. Verified GREEN:

```bash
pnpm --filter @coze-agent-ide/bot-creator exec vitest --run src/modes/super-mode/codex-trace/__tests__/codex-trace-panel.test.tsx
pnpm --filter @coze-agent-ide/bot-creator exec vitest --run src/modes/super-mode/codex-trace/__tests__/trace-bridge.test.tsx src/modes/super-mode/__tests__/super-chat-area.test.tsx src/modes/super-mode/codex-trace/__tests__/codex-trace-panel.test.tsx
pnpm --filter @coze-agent-ide/bot-creator exec tsc --noEmit
pnpm --filter @coze-studio/app build
```

Deployment:

- Rebuilt frontend static resources with `pnpm --filter @coze-studio/app build`.
- Uploaded `/tmp/coze-static-pending-reply-20260620.tar.gz`.
- Archive SHA-256:
  - `78adfb87d3acc39757daecea0b806d8488a3aab6ba822f42807247a4abd0cf80`
- Replaced `coze-super:/app/resources/static`.
- Kept one static rollback point:
  - `coze-super:/app/resources/static.bak-latest`
- Removed uploaded host/container temp archives.
- Disk after deploy and cleanup:
  - Container `/`: total `71.8G`, used `54.2G`, available `13.9G`, use `80%`.

Playwright MCP verification:

- Browser page:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=pending_reply_empty_turn_20260620_0328`
- Test prompt:
  - `请回复 FINAL_STREAM_OK_1781897394346，用于测试发送后立即显示 AI 三点占位。不要创建、修改或删除文件。`
- Checks:
  - Before send: `pending_count = 0`.
  - 300ms after send: `pending_count = 1`.
  - 300ms after send: `pending_dot_count = 3`.
  - 300ms after send: empty state was not visible.
  - After assistant reply: `pending_count = 0`.
  - After assistant reply: `FINAL_STREAM_OK_1781897394346` appeared twice, once in the user prompt and once in the assistant reply.
  - After assistant reply: `has_stop_responding = false`.

Saved evidence:

- `super-agent-playwright-mcp-8896-pending-reply-empty-turn-final.json`
- `.playwright-mcp/pending-reply-empty-turn-final.png`

## 2026-06-20 10:18 CST - Harness Trace Evidence Events

Scope:

- Upgrade `/api/super-agent/traces/get` from a generic run/message replay into a Codex-like harness evidence stream.
- Keep existing `conversation.*` trace events for compatibility.
- Add structured tool/plan events so external App Server clients and the UI can inspect tool start, tool result, plan updates, arguments, return content, status, call id, and message id.

Fix:

- Added a pure trace projection package:
  - `backend/api/handler/coze/superagenttrace`
- `function_call` messages now emit:
  - `tool.started`
  - `plan.updated` when the tool is `update_plan`
- `tool_response` messages now emit:
  - `tool.completed` for successful responses
  - `tool.failed` for failed responses
- Tool request content now prefers `plugin_request`; when it is absent, the projection parses standard function-call content and exposes `function.arguments` directly.
- Manifest discovery now advertises:
  - `tool.started`
  - `tool.completed`
  - `tool.failed`
  - `plan.updated`

Local TDD verification:

1. Added RED coverage for deriving plan/tool trace events from `function_call` and `tool_response` messages.
2. Added RED coverage for extracting function-call `name` and `arguments` when `plugin_request` / `tool_name` are absent.
3. Verified GREEN:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/handler/coze/superagenttrace -run TestBuildDataDerivesHarnessToolAndPlanEvents -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/handler/coze/superagenttrace -run TestBuildDataExtractsFunctionCallArgumentsWhenPluginRequestIsMissing -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/handler/coze/superagenttrace ./api/router/coze ./api/middleware ./application/singleagent ./domain/agent/singleagent/internal/agentflow -count=1
git diff --check
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-harness-trace main.go
```

Deployment:

- Replaced `coze-super:/app/openynet`.
- Current backend SHA-256:
  - `eca6506ecd3e6839648370dfd61b520514f710fd38deffa3c2683b17d7446d14`
- Kept one backend rollback point:
  - `coze-super:/app/openynet.bak-latest`
- Rollback SHA-256:
  - `05e5f0239fce08feceb1f2f71efb0e7de17273969f12622eecede51a1f280f12`
- Removed uploaded host/container temp binary.
- Static resources were not changed.
- Disk after deploy and cleanup:
  - Container `/`: total `71.8G`, used `54.2G`, available `13.9G`, use `80%`.

Playwright MCP verification:

- Browser origin:
  - `http://10.10.10.226:8896/`
- App Server request:
  - `POST /api/super-agent/runs/create`
  - Prompt required an `update_plan` call and final marker `TRACE_NORMALIZED_DONE_*`.
- Trace request:
  - `POST /api/super-agent/traces/get`
  - `conversation_id = 7653296052044300288`
- Checks:
  - `manifestStatus = 200`
  - manifest includes `tool.started`, `tool.completed`, `tool.failed`, `plan.updated`
  - run returned `code = 0`
  - run status was `completed`
  - trace returned `code = 0`
  - trace includes `tool.started`
  - trace includes `plan.updated`
  - trace includes `tool.completed`
  - `tool.started.content` is the normalized arguments JSON beginning with `{"plan": ...}`
  - `plan.updated.content` is the normalized arguments JSON beginning with `{"plan": ...}`
  - `tool.completed.content` includes `Plan updated`
  - all harness events carry `run_id`, `message_id`, `conversation_id`, `agent_id`, and `call_id`

Saved evidence:

- `super-agent-playwright-mcp-8896-harness-trace-events-final.json`
- `super-agent-playwright-mcp-8896-harness-trace-run-trace-final.json`
- `super-agent-playwright-mcp-8896-harness-trace-normalized-final.json`

## 2026-06-20 10:27 CST - Harness Trace Tool Response Pairing

Scope:

- Make `tool.completed` / `tool.failed` events self-describing enough for UI expansion and external App Server clients.
- Preserve the existing trace envelope while enriching response metadata via `call_id` pairing.

Fix:

- `superagenttrace.BuildData` now builds a call index from `function_call` messages.
- `tool_response` projection now inherits missing context from the matching `function_call`:
  - `tool_name`
  - `plugin`
  - `request_message_id`
  - `request_event_id`
  - `duration_ms`
- This makes each completed tool event able to point back to the exact `tool.started` event and display the tool name even when the raw response message only contains `call_id`.

Local TDD verification:

1. Added RED assertions to `TestBuildDataDerivesHarnessToolAndPlanEvents` requiring `tool.completed.metadata` to include:
   - `tool_name`
   - `request_message_id`
   - `request_event_id`
   - `duration_ms`
2. Verified GREEN:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/handler/coze/superagenttrace -run TestBuildDataDerivesHarnessToolAndPlanEvents -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/handler/coze/superagenttrace ./api/router/coze ./api/middleware ./application/singleagent ./domain/agent/singleagent/internal/agentflow -count=1
git diff --check
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-harness-trace-pairing main.go
```

Deployment:

- Replaced `coze-super:/app/openynet`.
- Current backend SHA-256:
  - `81f7c095666ff0137e9667256efa97a055926e2986acd7e8b53abc1272176c0a`
- Kept one backend rollback point:
  - `coze-super:/app/openynet.bak-latest`
- Rollback SHA-256:
  - `eca6506ecd3e6839648370dfd61b520514f710fd38deffa3c2683b17d7446d14`
- Removed uploaded host/container temp binary.
- Static resources were not changed.
- Disk after deploy and cleanup:
  - Container `/`: total `71.8G`, used `54.2G`, available `13.9G`, use `80%`.

Playwright MCP verification:

- Browser page:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=harness_trace_pairing_20260620`
- App Server request:
  - `POST /api/super-agent/runs/create`
  - Prompt required an `update_plan` call and final marker `TRACE_PAIRING_DONE_*`.
- Trace request:
  - `POST /api/super-agent/traces/get`
  - `conversation_id = 7653298457200820224`
- Checks:
  - run returned `code = 0`
  - run status was `completed`
  - trace returned `code = 0`
  - trace includes `tool.started`
  - trace includes `plan.updated`
  - trace includes `tool.completed`
  - `tool.completed.metadata.tool_name = update_plan`
  - `tool.completed.metadata.request_message_id = 7653298467632054272`
  - `tool.completed.metadata.request_event_id = message:7653298467632054272:tool.started`
  - `tool.completed.metadata.duration_ms = 9`
  - `tool.completed.metadata.request_event_id` matches the actual `tool.started.id`
  - `tool.completed.metadata.tool_name` matches `tool.started.metadata.tool_name`

Saved evidence:

- `super-agent-playwright-mcp-8896-harness-trace-pairing-final.json`

## 2026-06-20 10:40 CST - Harness Trace Failed Tool Events

Scope:

- Make failed tool calls auditable as `tool.failed`, not incorrectly reported as successful `tool.completed`.
- Cover both explicit `plugin_status` failures and the real `run_bash` response format where failure is only visible in content as `exit_code: N`.

Root cause found during Playwright verification:

- A first deployment recognized non-zero `plugin_status`, but real `run_bash` messages did not include `plugin_status`.
- Real failed `run_bash` response shape was:

```text
exit_code: 2
stdout:

stderr:
```

- Because the failure signal lived only in content, the first deployed version incorrectly projected it as `tool.completed` with `status = success`.

Fix:

- `superagenttrace` now treats non-empty/non-success `plugin_status` values as failed.
- `superagenttrace` also parses `exit_code: N` from tool response content.
- Non-zero `exit_code` now emits:
  - `event = tool.failed`
  - `status = failed`
  - `metadata.exit_code = N`
- Pairing metadata from the original `tool.started` event is preserved:
  - `tool_name`
  - `plugin`
  - `request_message_id`
  - `request_event_id`
  - `duration_ms`

Local TDD verification:

1. Added RED coverage for `plugin_status = 2`.
2. Added RED coverage for real `run_bash` content with `exit_code: 2` and no `plugin_status`.
3. Verified GREEN:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/handler/coze/superagenttrace -run TestBuildDataDerivesFailedToolEventForNonZeroPluginStatus -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/handler/coze/superagenttrace -run TestBuildDataDerivesFailedRunBashEventFromExitCodeContent -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/handler/coze/superagenttrace ./api/router/coze ./api/middleware ./application/singleagent ./domain/agent/singleagent/internal/agentflow -count=1
git diff --check
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-harness-trace-failed main.go
```

Deployment:

- Replaced `coze-super:/app/openynet`.
- Current backend SHA-256:
  - `359514b156ae37a0dc73e5491cede0931f6530ac8feece4e75e831da4c8ca55f`
- Kept one backend rollback point:
  - `coze-super:/app/openynet.bak-latest`
- Rollback SHA-256:
  - `92ef95f871aba22b7408f1d6cb84f9fec93c60780513466d533b27956e0eaa30`
- Removed uploaded host/container temp binary.
- Static resources were not changed.
- Disk after deploy and cleanup:
  - Container `/`: total `71.8G`, used `54.2G`, available `13.9G`, use `80%`.

Playwright MCP verification:

- Browser page:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=harness_trace_failed_20260620`
- App Server request:
  - `POST /api/super-agent/runs/create`
  - Prompt required `run_bash` to execute `bash -lc 'exit 2'`.
- Trace request:
  - `POST /api/super-agent/traces/get`
  - `conversation_id = 7653301936933830656`
- Checks after the fixed deployment:
  - run returned `code = 0`
  - run status was `completed`
  - trace returned `code = 0`
  - trace includes `tool.started`
  - trace includes `tool.failed`
  - trace does not include `tool.completed` for the failed call
  - `tool.failed.status = failed`
  - `tool.failed.metadata.tool_name = run_bash`
  - `tool.failed.metadata.exit_code = 2`
  - `tool.failed.metadata.request_event_id = message:7653301948333948928:tool.started`
  - `tool.failed.metadata.request_event_id` matches the actual `tool.started.id`
  - `tool.failed.metadata.duration_ms = 11`

Saved evidence:

- First failed verification, before content parsing:
  - `super-agent-playwright-mcp-8896-harness-trace-failed-final.json`
- Fixed verification:
  - `super-agent-playwright-mcp-8896-harness-trace-failed-fixed-final.json`

## 2026-06-20 Harness Tool Outputs Dedicated Route

Goal:

- Make harness tool outputs a first-class App Server resource instead of requiring clients to know and call generic `workspace/list` with `/workspace/.agent/tooloutputs`.
- Add a dedicated route:
  - `POST /api/super-agent/harness/tool-outputs`
- Support:
  - listing tool output files under `/workspace/.agent/tooloutputs`
  - reading one tool output with `include_content = true`
  - rejecting paths outside the tool output root
  - manifest/openapi discovery through `harness.tool_outputs`

Local TDD verification:

1. Added RED coverage for `ListSuperAgentHarnessToolOutputs`.
2. Added RED coverage for the new route in manifest/openapi and malformed JSON routing.
3. Verified GREEN:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent -run 'TestListSuperAgentHarnessToolOutputs|TestGetSuperAgentHarnessStateReturnsPlanAndToolOutputs' -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgentManifestRouteReturnsAppServerContract|TestSuperAgentHarnessRoutesRejectMalformedJSON' -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/middleware -run TestSuperAgentAppServerPathsNeedOpenAPIAuth -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/handler/coze/superagenttrace ./api/router/coze ./api/middleware ./application/singleagent ./domain/agent/singleagent/internal/agentflow -count=1
git diff --check
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-harness-tool-outputs main.go
```

Deployment:

- Replaced `coze-super:/app/openynet`.
- Current backend SHA-256:
  - `37b808345069c6ed146ea6930221f90d6e72f1c0a6ec14a2a2a4b10145e13cf9`
- Kept one backend rollback point:
  - `coze-super:/app/openynet.bak-latest`
- Rollback SHA-256:
  - `359514b156ae37a0dc73e5491cede0931f6530ac8feece4e75e831da4c8ca55f`
- Removed uploaded host temp binary.
- Static resources were not changed.
- Disk after deploy and cleanup:
  - Container `/`: total `71.8G`, used `54.2G`, available `13.9G`, use `80%`.

Playwright MCP verification:

- Browser page:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=harness_tool_outputs_20260620`
- Manifest checks:
  - `harness.tool_outputs_route = POST /api/super-agent/harness/tool-outputs`
  - `external_api.entry_routes.harness.tool_outputs = POST /api/super-agent/harness/tool-outputs`
  - `external_api.request_schemas.harness.tool_outputs.optional` includes `include_content`
  - OpenAPI has `operationId = harness.tool_outputs`
- Runtime checks:
  - Wrote a small probe file to `/workspace/.agent/tooloutputs/playwright-harness-tool-output-20260620.json` via `POST /api/super-agent/sandbox/exec`
  - Listed tool outputs via `POST /api/super-agent/harness/tool-outputs`
  - Read the probe with `include_content = true`
  - Verified content includes `FINAL_STREAM_OK`
  - Verified `/workspace/.plan.json` is rejected by the dedicated route
  - Removed the probe file via `POST /api/super-agent/sandbox/exec`
- Final Playwright MCP result:
  - `pass = true`
  - all checks were `true`

Saved evidence:

- `super-agent-playwright-mcp-8896-harness-tool-outputs-final.json`

## 2026-06-20 Harness Snapshot App Server Route

Goal:

- Add a Codex-like harness snapshot endpoint for App Server clients so an external workbench can hydrate state with one request.
- Add route:
  - `POST /api/super-agent/harness/snapshot`
- Snapshot components:
  - `harness`: plan, runtime skill roots, tool output root
  - `tool_outputs`: files under `/workspace/.agent/tooloutputs`
  - `artifacts`: deliverables under `/outputs`
  - `trace`: conversation run/message trace when `conversation_id` is provided
  - `approvals`: pending required-action runs when `conversation_id` is provided
- Request behavior:
  - requires one of `conversation_id`, `agent_id`, `bot_id`
  - supports `include_harness`, `include_tool_outputs`, `include_artifacts`, `include_trace`, `include_approvals`
  - supports `trace_limit`, `artifact_limit`, `run_id`, `connector_id`

Local TDD verification:

1. Added route-level RED coverage for snapshot trace + approvals aggregation with an OpenAPI-auth context.
2. Added contract coverage for manifest/openapi discovery.
3. Added negative coverage for missing snapshot key returning HTTP 400.
4. Verified GREEN:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze ./api/middleware -run 'TestSuperAgentHarnessSnapshotRouteIncludesTraceAndApprovals|TestSuperAgentHarnessSnapshotRouteRejectsMissingSnapshotKey|TestSuperAgentManifestRouteReturnsAppServerContract|TestSuperAgentHarnessRoutesRejectMalformedJSON|TestSuperAgentAppServerPathsNeedOpenAPIAuth' -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/handler/coze/superagenttrace ./api/router/coze ./api/middleware ./application/singleagent ./domain/agent/singleagent/internal/agentflow -count=1
git diff --check
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-harness-snapshot main.go
```

Deployment:

- Replaced `coze-super:/app/openynet`.
- Current backend SHA-256:
  - `59e0f471234caf3ec3801972a74f1c47e56afe71201c815319c0acc938aecce9`
- Kept one backend rollback point:
  - `coze-super:/app/openynet.bak-latest`
- Rollback SHA-256:
  - `37b808345069c6ed146ea6930221f90d6e72f1c0a6ec14a2a2a4b10145e13cf9`
- Removed uploaded host temp binary.
- Static resources were not changed.
- Disk after deploy and cleanup:
  - Container `/`: total `71.8G`, used `54.2G`, available `13.9G`, use `80%`.

Playwright MCP verification:

- Browser page:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=harness_snapshot_20260620`
- Contract checks:
  - `harness.snapshot_route = POST /api/super-agent/harness/snapshot`
  - `external_api.entry_routes.harness.snapshot = POST /api/super-agent/harness/snapshot`
  - `external_api.request_schemas.harness.snapshot.required_one_of = [["conversation_id","agent_id","bot_id"]]`
  - `external_api.request_schemas.harness.snapshot.optional` includes `include_trace` and `include_artifacts`
  - OpenAPI has `/api/super-agent/harness/snapshot` and `HarnessSnapshotRequest`
- Runtime checks:
  - Wrote a small probe file into `/workspace/.agent/tooloutputs` via `POST /api/super-agent/sandbox/exec`
  - Wrote a small probe artifact into `/outputs` via `POST /api/super-agent/sandbox/exec`
  - Called `POST /api/super-agent/harness/snapshot` with `agent_id = 7652617174313336832`
  - Verified response components include `harness`, `tool_outputs`, `artifacts`
  - Verified `tool_outputs.root = /workspace/.agent/tooloutputs`
  - Verified the probe tool output and artifact were both returned by the snapshot
  - Removed both probe files via `POST /api/super-agent/sandbox/exec`
- Negative checks:
  - `POST /api/super-agent/harness/snapshot` without `conversation_id`, `agent_id`, or `bot_id` returns HTTP 400
  - Response message: `conversation_id or agent_id is required`
- Final Playwright MCP results:
  - contract `pass = true`
  - runtime `pass = true`
  - negative `pass = true`

Saved evidence:

- `super-agent-playwright-mcp-8896-harness-snapshot-contract-probe.json`
- `super-agent-playwright-mcp-8896-harness-snapshot-runtime-final.json`
- `super-agent-playwright-mcp-8896-harness-snapshot-negative-final.json`

## 2026-06-20 11:47 CST - Harness Messages Restore App Server Route

Goal:

- Treat session message history as a first-class App Server resource so a Codex-like/Hermes-like harness client can restore a durable work session, not only hydrate workspace/harness files.

Local changes:

- Added `POST /api/super-agent/messages/list`.
- Manifest now exposes:
  - top-level `messages` capability
  - `messages.list_route = POST /api/super-agent/messages/list`
  - `messages.max_page_size = 100`
  - `messages.order_values = ["ASC", "DESC"]`
  - `external_api.entry_routes["messages.list"]`
  - OpenAPI `operationId = messages.list` with `MessagesListRequest`
- App Server capability metadata now includes `messages`.
- The route returns stable restore fields such as `message_id`, `conversation_id`, `run_id`, `role`, `type`, `content`, `content_type`, `reasoning_content`, `metadata`, cursors, and ordering, while not exposing internal `model_content`.

Local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze ./api/middleware -run 'TestSuperAgentMessageListRouteReturnsSessionMessages|TestSuperAgentManifestRouteReturnsAppServerContract|TestSuperAgentHarnessRoutesRejectMalformedJSON|TestSuperAgentAppServerPathsNeedOpenAPIAuth' -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/handler/coze/superagenttrace ./api/router/coze ./api/middleware ./application/singleagent ./domain/agent/singleagent/internal/agentflow -count=1
git diff --check
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-messages-list main.go
```

Deployment:

- Deployed backend binary to container `coze-super`.
- New `/app/openynet` SHA256: `7690a67a246b47b20451ac11ceb440da1969725e48e5688d08e750f3c90e096a`.
- Kept one rollback binary only: `/app/openynet.bak-latest` SHA256 `59e0f471234caf3ec3801972a74f1c47e56afe71201c815319c0acc938aecce9`.
- Disk after deployment: total `71.8G`, used `54.2G`, available `13.9G`, `80%`.

Playwright MCP verification:

- Page context:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=messages_list_20260620`
- Contract probe:
  - Manifest status `200`, code `0`.
  - `capabilities` includes `messages`.
  - `messages.list_route = POST /api/super-agent/messages/list`.
  - `external_api.client_metadata.capabilities_value = sessions,messages,sandbox,workspace,skills,harness,artifacts,approvals,traces`.
  - OpenAPI has `/api/super-agent/messages/list` with `operationId = messages.list`, `x-transport = json`, `x-mutates = false`, and request ref `#/components/schemas/MessagesListRequest`.
- Runtime probe:
  - `POST /api/super-agent/sessions/list` returned status `200`, code `0`, and `15` sessions.
  - Selected conversation `7652827977981362176`.
  - `POST /api/super-agent/messages/list` returned status `200`, code `0`, `order_by = ASC`, `message_count = 2`, `prev_cursor = 7652827978098802688`, and `next_cursor = 7652828036663869440`.
  - First message role/type: `user/question`.
  - Last message role/type: `assistant/answer`.
  - Verified ascending order and no exposed `model_content`.
- Negative probe:
  - `POST /api/super-agent/messages/list` without `conversation_id` returned status `400`, code `400`, `msg = conversation_id is required`.

Evidence files:

- `super-agent-playwright-mcp-8896-messages-list-contract-final.json`
- `super-agent-playwright-mcp-8896-messages-list-runtime-final.json`
- `super-agent-playwright-mcp-8896-messages-list-negative-final.json`

## 2026-06-20 11:59 CST - Harness Snapshot Includes Session Messages

Goal:

- Make `POST /api/super-agent/harness/snapshot` a fuller Codex-like workbench hydration endpoint by including session message history alongside trace, approvals, harness state, tool outputs, and artifacts.

Local changes:

- Added `include_messages` and `message_limit` to `HarnessSnapshotRequest`.
- `harness/snapshot` now includes a `messages` component by default when `conversation_id` is provided.
- Snapshot message data reuses the stable App Server message restore shape from `messages.list`.
- `include_messages=false` disables message hydration for lightweight snapshots.

Local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgentHarnessSnapshotRouteIncludesTraceAndApprovals|TestSuperAgentManifestRouteReturnsAppServerContract' -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/handler/coze/superagenttrace ./api/router/coze ./api/middleware ./application/singleagent ./domain/agent/singleagent/internal/agentflow -count=1
git diff --check
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-harness-snapshot-messages main.go
```

Deployment:

- Deployed backend binary to container `coze-super`.
- New `/app/openynet` SHA256: `335a7e79575f58d38d1904acd937143bb118ba98f73048cc90d8a9e0fe6fd2b9`.
- Kept one rollback binary only: `/app/openynet.bak-latest` SHA256 `7690a67a246b47b20451ac11ceb440da1969725e48e5688d08e750f3c90e096a`.
- Disk after deployment: total `71.8G`, used `54.2G`, available `13.9G`, `80%`.

Playwright MCP verification:

- Page context:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=harness_snapshot_messages_20260620`
- Contract probe:
  - Manifest status `200`, code `0`.
  - `harness.snapshot` schema optional fields include `include_messages` and `message_limit`.
  - OpenAPI `HarnessSnapshotRequest` properties include `include_messages` and `message_limit`.
- Runtime probe:
  - `POST /api/super-agent/sessions/list` returned status `200`, code `0`.
  - Selected conversation `7652827977981362176`.
  - `POST /api/super-agent/harness/snapshot` with `include_messages=true`, `include_trace=true`, and `include_approvals=true` returned status `200`, code `0`.
  - Components returned: `messages`, `trace`, `approvals`.
  - Message hydration returned `2` messages, `order_by = ASC`, first `user/question`, last `assistant/answer`, and no exposed `model_content`.
- Disabled probe:
  - `POST /api/super-agent/harness/snapshot` with `include_messages=false` returned status `200`, code `0`, empty components, no `messages` field, and no `messages` component.

Evidence files:

- `super-agent-playwright-mcp-8896-harness-snapshot-messages-contract-final.json`
- `super-agent-playwright-mcp-8896-harness-snapshot-messages-runtime-final.json`
- `super-agent-playwright-mcp-8896-harness-snapshot-messages-disabled-final.json`

## 2026-06-20 12:15 CST - Session Get Hydrates Harness Snapshot

Goal:

- Add a Codex App Server style session restore endpoint so an external harness/client can open one conversation by ID and optionally hydrate the current workbench snapshot in one request.

Local changes:

- Added `POST /api/super-agent/sessions/get`.
- `sessions/get` validates session ownership, returns stable session metadata, and supports optional `include_snapshot`.
- When `include_snapshot=true`, the response embeds the same harness snapshot components used by `harness/snapshot`, including `messages` when requested.
- Published `sessions.get` through manifest routes, external API routes, and OpenAPI `SessionsGetRequest`.
- Added `/api/super-agent/sessions/*` to OpenAPI bearer-auth path matching.

Local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze ./api/middleware -run 'TestSuperAgentSessionGetRouteReturnsSessionAndOptionalSnapshot|TestSuperAgentManifestRouteReturnsAppServerContract|TestSuperAgentOpenAPIRouteReturnsMachineReadableSchema|TestSuperAgentHarnessRoutesRejectMalformedJSON|TestSuperAgentAppServerPathsNeedOpenAPIAuth' -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/handler/coze/superagenttrace ./api/router/coze ./api/middleware ./application/singleagent ./domain/agent/singleagent/internal/agentflow -count=1
git diff --check
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-session-get main.go
```

Deployment:

- Deployed backend binary to container `coze-super`.
- New `/app/openynet` SHA256: `f257cc2c80f16dca38e361e82f46e66d9aad9ff9414851404c2e4100c90a7734`.
- Kept one rollback binary only: `/app/openynet.bak-latest` SHA256 `335a7e79575f58d38d1904acd937143bb118ba98f73048cc90d8a9e0fe6fd2b9`.
- Disk after deployment: total `71.8G`, used `54.2G`, available `13.9G`, `80%`.

Playwright MCP verification:

- Page context:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=session_get_20260620`
- Contract probe:
  - Manifest status `200`, code `0`.
  - `sessions.get_route = POST /api/super-agent/sessions/get`.
  - `external_api.entry_routes.sessions.get = POST /api/super-agent/sessions/get`.
  - External request schema requires `conversation_id` and includes `include_snapshot`.
  - OpenAPI has `/api/super-agent/sessions/get` with `operationId = sessions.get` and request ref `#/components/schemas/SessionsGetRequest`.
- Runtime probe:
  - `POST /api/super-agent/sessions/list` returned status `200`, code `0`, and `10` sessions.
  - Selected conversation `7652827977981362176`.
  - `POST /api/super-agent/sessions/get` with `include_snapshot=true`, `include_messages=true`, and other heavy components disabled returned status `200`, code `0`.
  - Returned session matched the selected conversation.
  - Snapshot components returned: `messages`.
  - Message hydration returned `2` messages, first `user/question`, and no exposed `model_content`.
- Negative probe:
  - `POST /api/super-agent/sessions/get` without `conversation_id` returned status `400`, code `400`, `msg = conversation_id is required`.

Evidence files:

- `super-agent-playwright-mcp-8896-sessions-get-contract-final.json`
- `super-agent-playwright-mcp-8896-sessions-get-runtime-final.json`
- `super-agent-playwright-mcp-8896-sessions-get-negative-final.json`

## 2026-06-20 12:28 CST - Harness Snapshot Includes Run History

Goal:

- Make harness restore closer to a Codex-style App Server hydration path by returning recent run history together with the session workbench snapshot.

Local changes:

- Added `include_runs` and `run_limit` to `HarnessSnapshotRequest`.
- `harness/snapshot` now includes a `runs` component by default when `conversation_id` is provided.
- `sessions/get` forwards `include_runs` and `run_limit` into the embedded snapshot.
- Reused the same run history builder used by `runs/list`, so run item shape and permission checks stay consistent.
- Published `include_runs` and `run_limit` through manifest external request schemas and OpenAPI for both `harness.snapshot` and `sessions.get`.

Local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgentHarnessSnapshotRouteIncludesTraceAndApprovals|TestSuperAgentSessionGetRouteReturnsSessionAndOptionalSnapshot|TestSuperAgentManifestRouteReturnsAppServerContract|TestSuperAgentOpenAPIRouteReturnsMachineReadableSchema' -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/handler/coze/superagenttrace ./api/router/coze ./api/middleware ./application/singleagent ./domain/agent/singleagent/internal/agentflow -count=1
git diff --check
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-harness-snapshot-runs main.go
```

Deployment:

- Deployed backend binary to container `coze-super`.
- New `/app/openynet` SHA256: `beab4d2c49b4c0812bb867080e5f762c87923c5f8a2786a40ff0dc94d8bcfcaa`.
- Kept one rollback binary only: `/app/openynet.bak-latest` SHA256 `f257cc2c80f16dca38e361e82f46e66d9aad9ff9414851404c2e4100c90a7734`.
- Disk after deployment: total `71.8G`, used `54.2G`, available `13.9G`, `80%`.

Playwright MCP verification:

- Page context:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=harness_snapshot_runs_20260620`
- Contract probe:
  - Manifest status `200`, code `0`.
  - `harness.snapshot` external schema optional fields include `include_runs` and `run_limit`.
  - `sessions.get` external schema optional fields include `include_runs` and `run_limit`.
  - OpenAPI `HarnessSnapshotRequest` and `SessionsGetRequest` both expose `include_runs` and `run_limit` in `x-optional-fields` and properties.
- Runtime probe:
  - `POST /api/super-agent/sessions/list` returned status `200`, code `0`, and `10` sessions.
  - Selected conversation `7652827977981362176`.
  - `POST /api/super-agent/harness/snapshot` with `include_runs=true` returned status `200`, code `0`, components `runs`, and `1` run.
  - First snapshot run: `run_id = 7652827978052665344`, `status = completed`, `active = false`.
  - `POST /api/super-agent/sessions/get` with `include_snapshot=true` and `include_runs=true` returned status `200`, code `0`, components `runs`, and the same conversation run.
- Disabled probe:
  - `POST /api/super-agent/harness/snapshot` with `include_runs=false` returned status `200`, code `0`, empty components, no `runs` field, and no `runs` component.

Evidence files:

- `super-agent-playwright-mcp-8896-harness-snapshot-runs-contract-final.json`
- `super-agent-playwright-mcp-8896-harness-snapshot-runs-runtime-final.json`
- `super-agent-playwright-mcp-8896-harness-snapshot-runs-disabled-final.json`

## 2026-06-20 12:44 CST - Harness Snapshot Includes Workspace Tree

Goal:

- Let an external App Server client restore the visible sandbox workspace tree with the same session snapshot used for messages, runs, trace, approvals, artifacts, and tool outputs.

Local changes:

- Added `include_workspace`, `workspace_path`, and `workspace_recursive` to `HarnessSnapshotRequest`.
- `harness/snapshot` now can include a `workspace` component backed by the existing `ListSuperAgentWorkspaceFiles` service.
- Workspace path defaults to `/workspace`; recursive listing remains opt-in through `workspace_recursive=true`.
- `sessions/get` forwards workspace snapshot options into the embedded snapshot.
- Published workspace snapshot options through manifest external request schemas and OpenAPI for both `harness.snapshot` and `sessions.get`.

Local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgentHarnessSnapshotRouteIncludesTraceAndApprovals|TestSuperAgentSessionGetRouteReturnsSessionAndOptionalSnapshot|TestSuperAgentManifestRouteReturnsAppServerContract|TestSuperAgentOpenAPIRouteReturnsMachineReadableSchema' -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/handler/coze/superagenttrace ./api/router/coze ./api/middleware ./application/singleagent ./domain/agent/singleagent/internal/agentflow -count=1
git diff --check
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-harness-snapshot-workspace main.go
```

Deployment:

- Deployed backend binary to container `coze-super`.
- New `/app/openynet` SHA256: `1fd678ada6eedf41d05729868c291667314c180bf702e1070c7d4b2bac5f19ca`.
- Kept one rollback binary only: `/app/openynet.bak-latest` SHA256 `beab4d2c49b4c0812bb867080e5f762c87923c5f8a2786a40ff0dc94d8bcfcaa`.
- Disk after deployment: total `71.8G`, used `54.2G`, available `13.9G`, `80%`.

Playwright MCP verification:

- Page context:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=harness_snapshot_workspace_20260620`
- Contract probe:
  - Manifest status `200`, code `0`.
  - `harness.snapshot` external schema optional fields include `include_workspace`, `workspace_path`, and `workspace_recursive`.
  - `sessions.get` external schema optional fields include `include_workspace`, `workspace_path`, and `workspace_recursive`.
  - OpenAPI `HarnessSnapshotRequest` and `SessionsGetRequest` both expose the workspace fields in `x-optional-fields` and properties.
- Runtime probe:
  - `POST /api/super-agent/sessions/list` returned status `200`, code `0`, and `10` sessions.
  - Selected conversation `7652827977981362176`.
  - `POST /api/super-agent/harness/snapshot` with `include_workspace=true`, `workspace_path=/workspace`, and `workspace_recursive=true` returned status `200`, code `0`, component `workspace`, path `/workspace`, and `481` files.
  - `POST /api/super-agent/sessions/get` with `include_snapshot=true` and workspace options returned status `200`, code `0`, component `workspace`, path `/workspace`, and `481` files.
- Disabled probe:
  - `POST /api/super-agent/harness/snapshot` with `include_workspace=false` returned status `200`, code `0`, empty components, no `workspace` field, and no `workspace` component.

Evidence files:

- `super-agent-playwright-mcp-8896-harness-snapshot-workspace-contract-final.json`
- `super-agent-playwright-mcp-8896-harness-snapshot-workspace-runtime-final.json`
- `super-agent-playwright-mcp-8896-harness-snapshot-workspace-disabled-final.json`

## 2026-06-20 13:02 CST - Harness Snapshot Includes Tool Output Content

Goal:

- Make harness tool-call history usable for UI expansion and external App Server clients by returning tool output file contents only when explicitly requested.

Local changes:

- Added `include_tool_output_content` to `HarnessSnapshotRequest` and `SessionsGetRequest`.
- `harness/snapshot` now forwards the flag to the tool output service.
- `ListSuperAgentHarnessToolOutputs` now returns `tool_outputs.contents`, keyed by full sandbox path, when `include_content` is enabled for the tool output root.
- Existing single-file `tool_outputs.content` behavior is preserved for direct path reads.
- Published the field through manifest external request schemas and OpenAPI for both `harness.snapshot` and `sessions.get`.

TDD evidence:

- Added `TestSuperAgentHarnessSnapshotRouteIncludesToolOutputContent`.
- RED run failed as expected because `tool_outputs.contents` was missing for `/workspace/.agent/tooloutputs/tool-call-search.json`.
- GREEN run passed after implementing the snapshot flag and contents map.

Local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgentHarnessSnapshotRouteIncludesToolOutputContent|TestSuperAgentManifestRouteReturnsAppServerContract|TestSuperAgentOpenAPIRouteReturnsMachineReadableSchema' -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/handler/coze/superagenttrace ./api/router/coze ./api/middleware ./application/singleagent ./domain/agent/singleagent/internal/agentflow -count=1
git diff --check
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-harness-tool-output-content main.go
```

Deployment:

- Deployed backend binary to container `coze-super`.
- New `/app/openynet` SHA256: `3389573c5484ca9e75466eefdbc1fd3bad8068cd63fe71f02201272dbf3988ef`.
- Kept one rollback binary only: `/app/openynet.bak-latest` SHA256 `1fd678ada6eedf41d05729868c291667314c180bf702e1070c7d4b2bac5f19ca`.
- Disk after deployment: total `72G`, used `55G`, available `14G`, `80%`.

Playwright MCP verification:

- Page context:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=harness_tool_output_content_20260620`
- Contract probe:
  - Manifest status `200`; OpenAPI status `200`.
  - `harness.snapshot` and `sessions.get` external schemas include `include_tool_output_content`.
  - OpenAPI `HarnessSnapshotRequest` and `SessionsGetRequest` expose `include_tool_output_content` in `x-optional-fields` and `properties`.
- Runtime probe:
  - Wrote a temporary probe file at `/workspace/.agent/tooloutputs/content-probe-20260620.json`.
  - `POST /api/super-agent/harness/snapshot` with `include_tool_outputs=true` and `include_tool_output_content=true` returned status `200`, component `tool_outputs`, `5` listed files, and `5` content entries.
  - Probe content round-tripped exactly with args and result JSON visible in `tool_outputs.contents`.
  - Temporary probe file was deleted successfully with status `200`, code `0`.
- Disabled probe:
  - `POST /api/super-agent/harness/snapshot` with `include_tool_output_content=false` returned status `200`, component `tool_outputs`, `4` listed files, and no `contents` field.

Evidence files:

- `super-agent-playwright-mcp-8896-harness-tool-output-content-contract-final.json`
- `super-agent-playwright-mcp-8896-harness-tool-output-content-runtime-final.json`
- `super-agent-playwright-mcp-8896-harness-tool-output-content-disabled-final.json`

## 2026-06-20 13:16 CST - Harness Tool Outputs Expose Structured Entries

Goal:

- Give the chat/workbench UI and external App Server clients a default collapsed-state summary for each tool output without requiring full content expansion.

Local changes:

- Added `tool_outputs.entries` to `SuperAgentHarnessToolOutputsData`.
- Each entry includes `path`, `name`, `tool`, `status`, `arguments_preview`, `result_preview`, `summary`, `size`, and `mtime`.
- JSON tool output files are parsed for `tool` / `args` / `result` / `status`.
- Plain text tool output files still produce a conservative file-name and text-preview entry.
- `tool_outputs.contents` remains opt-in through `include_tool_output_content` / `include_content`; entries are available for default collapsed display.

TDD evidence:

- Added `TestListSuperAgentHarnessToolOutputsReturnsStructuredEntries`.
- RED run failed as expected because `entries` was empty.
- GREEN run passed after implementing structured entry generation.
- Extended `TestSuperAgentHarnessSnapshotRouteIncludesToolOutputContent` to assert entries are present in the snapshot response.

Local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent -run TestListSuperAgentHarnessToolOutputsReturnsStructuredEntries -count=1
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent ./api/router/coze -run 'TestListSuperAgentHarnessToolOutputsReturnsStructuredEntries|TestSuperAgentHarnessSnapshotRouteIncludesToolOutputContent' -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/handler/coze/superagenttrace ./api/router/coze ./api/middleware ./application/singleagent ./domain/agent/singleagent/internal/agentflow -count=1
git diff --check
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-harness-tool-output-entries main.go
```

Deployment:

- Deployed backend binary to container `coze-super`.
- New `/app/openynet` SHA256: `fb08520d7108367fab4b415b0f287a7b266afc5434b3155dbf24ba2ea73dd9a4`.
- Kept one rollback binary only: `/app/openynet.bak-latest` SHA256 `3389573c5484ca9e75466eefdbc1fd3bad8068cd63fe71f02201272dbf3988ef`.
- Disk after deployment: total `72G`, used `55G`, available `14G`, `80%`.

Playwright MCP verification:

- Page context:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=harness_tool_output_entries_20260620`
- Runtime probe:
  - Wrote a temporary probe file at `/workspace/.agent/tooloutputs/entries-probe-20260620.json`.
  - `POST /api/super-agent/harness/snapshot` with `include_tool_outputs=true` and `include_tool_output_content=false` returned status `200`, component `tool_outputs`, `5` listed files, `5` entries, and no `contents` field.
  - Snapshot entry parsed `tool = Search`, `status = completed`, `arguments_preview = {"query":"GLM-5.2 entries"}`, and `result_preview = {"count":8,"first":"docs"}`.
  - Direct `POST /api/super-agent/harness/tool-outputs` also returned the same structured entry.
  - Temporary probe file was deleted successfully with status `200`, code `0`.

Evidence files:

- `super-agent-playwright-mcp-8896-harness-tool-output-entries-runtime-final.json`

## 2026-06-20 13:35 CST - Artifact Registry Exposes Preview Metadata

Goal:

- Make `/outputs` deliverables usable as a Codex-like artifact registry for the workbench UI and external App Server clients.

Local changes:

- Added `kind`, `preview_type`, and `summary` to `SuperAgentArtifactMeta`.
- Artifact metadata now classifies common deliverables:
  - HTML files: `kind = report`, `preview_type = html`.
  - Images: `kind = image`, `preview_type = image`.
  - CSV files: `kind = table`, `preview_type = csv`.
  - Text/Markdown/PDF/JSON files map to document/data preview types.
- Artifact summaries include a readable type label and size, for example `HTML report, 64 B`.
- Manifest `artifacts.metadata_fields` now declares `kind`, `preview_type`, and `summary`.

TDD evidence:

- Added `TestListSuperAgentArtifactsReturnsPreviewMetadata`.
- RED run failed because the artifact response did not include `kind`, `preview_type`, or `summary`.
- GREEN run passed after enriching artifact metadata.
- Extended `TestSuperAgentManifestRouteReturnsAppServerContract` to assert the new metadata fields are declared.

Local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent -run TestListSuperAgentArtifactsReturnsPreviewMetadata -count=1
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent ./api/router/coze -run 'TestListSuperAgentArtifactsReturnsPreviewMetadata|TestSuperAgentManifestRouteReturnsAppServerContract' -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/handler/coze/superagenttrace ./api/router/coze ./api/middleware ./application/singleagent ./domain/agent/singleagent/internal/agentflow -count=1
git diff --check
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-artifact-preview-metadata main.go
```

Deployment:

- Deployed backend binary to container `coze-super`.
- New `/app/openynet` SHA256: `23b9e93fb5a4a4f74df6d75d8ef41addf6a24f5b17fdb168d10a4890d4ed5f02`.
- Kept one rollback binary only: `/app/openynet.bak-latest` SHA256 `fb08520d7108367fab4b415b0f287a7b266afc5434b3155dbf24ba2ea73dd9a4`.
- Disk after deployment: total `72G`, used `55G`, available `14G`, `80%`.

Playwright MCP verification:

- Page context:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=artifact_preview_metadata_20260620`
- Contract probe:
  - `GET /api/super-agent/manifest` returned status `200`.
  - `artifacts.metadata_fields` includes `kind`, `preview_type`, `summary`, `sha256`, and `mime`.
- Runtime probe:
  - Wrote temporary artifacts `/outputs/artifact-preview-probe-20260620.html` and `/outputs/artifact-preview-probe-20260620.csv`.
  - `POST /api/super-agent/artifacts/list` returned status `200`, HTML artifact `kind = report`, `preview_type = html`, `summary = HTML report, 64 B`, and CSV artifact `kind = table`, `preview_type = csv`, `summary = CSV table, 24 B`.
  - `POST /api/super-agent/harness/snapshot` with `include_artifacts=true` returned component `artifacts` and the same HTML artifact metadata.
  - Temporary artifacts were deleted successfully with status `200`, code `0`.

Evidence files:

- `super-agent-playwright-mcp-8896-artifact-preview-metadata-contract-final.json`
- `super-agent-playwright-mcp-8896-artifact-preview-metadata-runtime-final.json`

## 2026-06-20 13:52 CST - Harness Context Compaction Status Contract

Goal:

- Expose the current harness context compaction state as an inspectable App Server contract before adding full automatic session summarization.

Local changes:

- Added `context` to harness snapshot data and harness state data.
- Added `include_context` to `harness.snapshot` and `sessions.get` request schemas.
- Context state now declares:
  - `strategy = tool-output-offload+session-summary`
  - `summary_path = /workspace/.agent/context-summary.json`
  - `tool_output_root = /workspace/.agent/tooloutputs`
  - `tool_output_offload = true`
  - `recent_messages_policy = message_limit+recent_tail`
- If `/workspace/.agent/context-summary.json` exists, the API parses and returns its summary fields; otherwise it returns `summary_exists = false` without failing the snapshot.

TDD evidence:

- Added `TestSuperAgentHarnessSnapshotRouteIncludesContextState`.
- RED run failed because snapshot components did not include `context` and `context` was nil.
- GREEN run passed after adding the context state component.
- Extended manifest/OpenAPI route tests to assert `include_context` and `context` state field declarations.

Local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgentManifestRouteReturnsAppServerContract|TestSuperAgentOpenAPIRouteReturnsMachineReadableSchema|TestSuperAgentHarnessSnapshotRouteIncludesContextState' -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/handler/coze/superagenttrace ./api/router/coze ./api/middleware ./application/singleagent ./domain/agent/singleagent/internal/agentflow -count=1
git diff --check
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-harness-context main.go
```

Deployment:

- Deployed backend binary to container `coze-super`.
- New `/app/openynet` SHA256: `0f7101148bec5d20bd0b8a2759599278f9599856073df1403d50d1dc0ff0b326`.
- Kept one rollback binary only: `/app/openynet.bak-latest` SHA256 `23b9e93fb5a4a4f74df6d75d8ef41addf6a24f5b17fdb168d10a4890d4ed5f02`.
- Disk after deployment: total `72G`, used `55G`, available `14G`, `80%`.

Playwright MCP verification:

- Page context:
  - `http://10.10.10.226:8896/api/super-agent/manifest`
- Contract probe:
  - `GET /api/super-agent/manifest` returned status `200`.
  - `harness.state_fields` includes `context`.
  - `sessions.get` and `harness.snapshot` optional fields include `include_context`.
  - `GET /api/super-agent/openapi.json` declares `include_context` on `SessionsGetRequest` and `HarnessSnapshotRequest`.
- Runtime probe:
  - `POST /api/super-agent/harness/snapshot` with `include_context=true` returned status `200`, component `context`, `summary_path = /workspace/.agent/context-summary.json`, `summary_exists = false`, and `tool_output_offload = true`.
  - `POST /api/super-agent/harness/state` returned status `200` with the same context status plus plan/tool output/runtime skill roots.

Evidence files:

- `super-agent-playwright-mcp-8896-harness-context-contract-runtime.json`
- `super-agent-playwright-mcp-8896-harness-context-state.json`

## 2026-06-20 14:07 CST - Harness Auto Context Compaction

Goal:

- Add real automatic session context compaction for super-agent runs, beyond the inspectable status contract.
- Long super-agent history is summarized into `/workspace/.agent/context-summary.json`, then reinjected as a system summary plus a recent message tail so the harness keeps continuity without sending the entire old transcript.

Local changes:

- `AgentRunner` now carries `superAgent` and `sandboxKey` so the pre-handler can identify super-agent runs and write into the active sandbox.
- Added `preHandlerContextCompaction` after normal history handling.
- Added default compaction thresholds:
  - `AGENT_CONTEXT_COMPACT_MAX_BYTES`, default `160 KiB`.
  - `AGENT_CONTEXT_COMPACT_RECENT_MESSAGES`, default `16`.
- Context summary JSON includes `version`, `summary`, `updated_at`, `key_files`, `artifacts`, and `next_actions`.
- The compacted prompt includes a system message headed `Context summary (auto-compacted)`, the persisted summary path, key file hints, artifact hints, and next actions.
- Super-agent conversation service history trimming now returns budget `0`, leaving long-history compaction to the harness path instead of generic token trimming.
- Normal agents keep the existing history token budget behavior.

TDD evidence:

```bash
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow -run 'TestPreHandlerReqAutoCompactsSuperAgentHistory|TestPreHandlerReqDoesNotCompactNormalAgentHistory' -count=1
```

- RED before implementation: failed with `history should be compacted: before=24 after=24`.
- GREEN after implementation: passed.

```bash
SESSION_HMAC_SECRET=test-secret go test ./domain/conversation/agentrun/service -run TestHistoryTokenBudgetForSuperAgentLetsHarnessCompact -count=1
```

- RED before implementation: failed because `historyTokenBudgetForAgent` was undefined.
- GREEN after implementation: passed.

Local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/handler/coze/superagenttrace ./api/router/coze ./api/middleware ./application/singleagent ./domain/agent/singleagent/internal/agentflow ./domain/conversation/agentrun/service -count=1
git diff --check
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-auto-context-compaction main.go
```

Build artifact:

- `/tmp/openynet-auto-context-compaction`
- SHA256: `f5f3f310217eab29a913eefea2cd19e0d2468bd49a2aac0c5fb3f9fc55d26453`
- Size: `173M`

Deployment:

- Deployed backend binary to container `coze-super`.
- Pre-deploy `/app/openynet` SHA256: `0f7101148bec5d20bd0b8a2759599278f9599856073df1403d50d1dc0ff0b326`.
- New `/app/openynet` SHA256: `f5f3f310217eab29a913eefea2cd19e0d2468bd49a2aac0c5fb3f9fc55d26453`.
- Kept one rollback binary only: `/app/openynet.bak-latest` SHA256 `0f7101148bec5d20bd0b8a2759599278f9599856073df1403d50d1dc0ff0b326`.
- Disk after deployment: total `72G`, used `55G`, available `14G`, `80%`.

Playwright MCP verification:

- Page context:
  - `http://10.10.10.226:8896/api/super-agent/manifest`
- Contract probe:
  - `GET /api/super-agent/manifest` returned status `200`, code `0`.
  - `harness.state_fields` includes `context`.
  - `GET /api/super-agent/openapi.json` declares `include_context` on `SessionsGetRequest` and `HarnessSnapshotRequest`.
- Runtime probe:
  - `POST /api/super-agent/harness/snapshot` with `include_context=true` returned status `200`, code `0`, and component `context`.
  - `context.strategy = tool-output-offload+session-summary`.
  - `context.summary_path = /workspace/.agent/context-summary.json`.
  - `context.summary_exists = false` on the fresh verified sandbox.
  - `context.tool_output_root = /workspace/.agent/tooloutputs`.
  - `context.tool_output_offload = true`.
  - `POST /api/super-agent/harness/state` returned status `200`, code `0`, with the same context state plus plan, tool output, and runtime skill roots.

Evidence file:

- `super-agent-playwright-mcp-8896-auto-context-compaction-state.json`

Notes:

- The remote API contract and runtime context state were verified through Playwright MCP on `10.10.10.226:8896`.
- The actual compaction trigger and history rewrite path were verified by Go tests, because forcing a large persisted remote conversation would require creating significant DB history in the shared 226 environment.

## 2026-06-20 14:22 CST - Context Compaction Metadata and Trace Event

Goal:

- Make automatic context compaction observable to App Server clients, not just internally active.
- Expose compaction reason and size metrics through the harness context summary.
- Add a replayable `context.compacted` trace event when a snapshot includes both trace and context state.

Local changes:

- Context summary JSON now includes:
  - `trigger`
  - `original_messages`
  - `compacted_messages`
  - `retained_messages`
  - `original_bytes`
  - `max_bytes`
  - `summary_path`
- `SuperAgentHarnessContextSummary` preserves these fields when reading `/workspace/.agent/context-summary.json`.
- Manifest trace event list now includes `context.compacted`.
- `harness.snapshot` now appends a synthetic `context.compacted` event when `include_trace=true`, `include_context=true`, and a summary exists.
- The synthetic trace event carries `kind=context`, `status=compacted`, summary content, run/message ids when present, and the compaction metrics in metadata.

TDD evidence:

```bash
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow -run TestPreHandlerReqAutoCompactsSuperAgentHistory -count=1
```

- RED before implementation: failed because `summary trigger = "", want history_bytes_exceeded`.
- GREEN after implementation: passed.

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run TestSuperAgentHarnessSnapshotRouteIncludesContextState -count=1
```

- RED before implementation: failed because context summary metadata fields were `0` or empty.
- GREEN after implementation: passed.

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgentManifestRouteReturnsAppServerContract|TestSuperAgentHarnessSnapshotRouteIncludesTraceAndApprovals' -count=1
```

- RED before implementation: failed because no `context.compacted` trace event was present in snapshot.
- GREEN after implementation: passed.

Local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/handler/coze/superagenttrace ./api/router/coze ./api/middleware ./application/singleagent ./domain/agent/singleagent/internal/agentflow ./domain/conversation/agentrun/service -count=1
git diff --check
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-context-trace main.go
```

Build artifact:

- `/tmp/openynet-context-trace`
- SHA256: `ef21a6a656fc2cae8f4d4edfe70eb22e95a8ef82e1e533485409a8ecd810f8b6`
- Size: `173M`

Deployment:

- Deployed backend binary to container `coze-super`.
- Pre-deploy `/app/openynet` SHA256: `f5f3f310217eab29a913eefea2cd19e0d2468bd49a2aac0c5fb3f9fc55d26453`.
- New `/app/openynet` SHA256: `ef21a6a656fc2cae8f4d4edfe70eb22e95a8ef82e1e533485409a8ecd810f8b6`.
- Kept one rollback binary only: `/app/openynet.bak-latest` SHA256 `f5f3f310217eab29a913eefea2cd19e0d2468bd49a2aac0c5fb3f9fc55d26453`.
- Disk after deployment: total `72G`, used `55G`, available `14G`, `80%`.

Playwright MCP verification:

- Page context:
  - `http://10.10.10.226:8896/api/super-agent/manifest`
- Contract probe:
  - `GET /api/super-agent/manifest` returned status `200`, code `0`.
  - `trace.events` includes `context.compacted`.
- Runtime probe:
  - Backed up/read `/workspace/.agent/context-summary.json`; the file did not exist for the probe sandbox.
  - Wrote a temporary minimal context summary with compaction metrics.
  - `POST /api/super-agent/harness/snapshot` with `include_context=true` and `include_trace=true` returned status `200`, code `0`, components `context` and `trace`.
  - `context.summary` returned the new metric fields: `trigger=history_bytes_exceeded`, `original_messages=42`, `compacted_messages=26`, `retained_messages=16`, `original_bytes=262144`, `max_bytes=163840`, and `summary_path=/workspace/.agent/context-summary.json`.
  - `trace.events` includes a synthetic `context.compacted` event with `kind=context`, `status=compacted`, `run_id=765`, `message_id=901`, and the same metrics in metadata.
  - Deleted the temporary probe summary after verification.

Evidence file:

- `super-agent-playwright-mcp-8896-context-trace-metadata-final.json`

## 2026-06-20 14:51 CST - Context Compaction Trace UI and `traces/get`

Goal:

- Make context compaction visible through `POST /api/super-agent/traces/get`, not only through harness snapshot.
- Let the super-agent TraceBridge merge `context.compacted` events for the selected session into the Codex-style trace panel.
- Render a compact context card in the trace panel with summary text, summary path, and compaction metrics.

Local changes:

- `SuperAgentGetTrace` now appends a synthetic `context.compacted` event when the selected conversation's agent has `/workspace/.agent/context-summary.json`.
- The session sidebar now emits `coze:super-agent-session-select` after the default active session is loaded, so external trace consumers know which session to fetch.
- `TraceBridge` listens for that session event, calls `SuperAgentGetTrace`, and merges `context.compacted` events into trace store messages.
- `CodexTracePanel` renders context compaction as a harness event card:
  - title `上下文已自动压缩`
  - metrics such as `42 条历史消息`, `压缩 26 条`, `保留 16 条`
  - summary path `/workspace/.agent/context-summary.json`
  - compacted summary content.

Local verification:

```bash
pnpm --filter @coze-agent-ide/bot-creator test -- trace-bridge.test.tsx super-session-sidebar.test.tsx codex-trace-panel.test.tsx
```

- PASS: 8 test files, 33 tests.

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run TestSuperAgentTraceRouteIncludesContextCompactionEvent -count=1
```

- PASS.

```bash
pnpm --filter @coze-agent-ide/bot-creator exec tsc --noEmit -p tsconfig.build.json
```

- BLOCKED by the current Rush package build state and existing unrelated type errors.
- The filtered rerun showed only `TS6305` missing dependency `.d.ts` errors in super-mode plus existing `super-config-area` implicit-any errors; the new local `summary_path` and `SuperAgentDeleteSession` type issues were removed.

```bash
pnpm --filter @coze-studio/app build
```

- PASS: Rsbuild completed successfully.

Build artifacts:

- Backend binary: `/tmp/openynet-context-trace-ui`
- Backend SHA256: `e4108728507894780390729ea713def4e299d2b102a699fa3bd148fffa7eb001`
- Static tarball: `/tmp/coze-studio-static-context-trace-ui.tar.gz`
- Static tarball SHA256: `3173db69ba0cbb09a7440fc770893e3a738191f054d5b94e71e0e630d0fa42e4`
- Static tarball excludes sourcemaps; local dist was `307M`, compressed upload was `25M`.

Deployment:

- Host: `10.10.10.226:8896`
- Container: `coze-super`
- Removed stale container static backup `/app/resources/static.bak.ui-fix-202606191009`.
- Replaced `coze-super:/app/openynet`.
- Replaced `coze-super:/app/resources/static`.
- Kept one backend rollback only: `/app/openynet.bak-latest`.
- Kept one static rollback only: `/app/resources/static.bak-latest`.
- Pre-deploy `/app/openynet` SHA256: `ef21a6a656fc2cae8f4d4edfe70eb22e95a8ef82e1e533485409a8ecd810f8b6`
- New `/app/openynet` SHA256: `e4108728507894780390729ea713def4e299d2b102a699fa3bd148fffa7eb001`
- `/app/resources/static` size after sourcemap-free deploy: `92.7M`
- `/app/resources/static.bak-latest` size: `312.0M`
- Disk after deployment: total `71.8G`, used `53.6G`, available `14.5G`, `79%`.
- `curl -I http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange` returned `HTTP/1.1 200 OK`.
- `docker logs --since 2m coze-super` scan found no `panic`, `fatal`, `nil pointer`, `segmentation`, `bind:`, or `error`.

Playwright MCP verification:

- Page:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=context_trace_ui_20260620`
- Initial probe:
  - page title `演示超级体 -智能体 - 猎鹰`
  - session list returned status `200`, code `0`
  - `POST /api/super-agent/traces/get` returned status `200`, code `0`
  - current natural trace had 4 events and no existing `context.compacted`.
- Runtime probe:
  - Wrote a temporary `/workspace/.agent/context-summary.json` through `POST /api/super-agent/workspace/write`.
  - Called `POST /api/super-agent/traces/get` for conversation `7652827977981362176`.
  - Response included one `context.compacted` event:
    - id `context:compacted:/workspace/.agent/context-summary.json`
    - kind `context`
    - status `compacted`
    - metadata `original_messages=42`, `compacted_messages=26`, `retained_messages=16`, `original_bytes=262144`, `max_bytes=163840`, `summary_path=/workspace/.agent/context-summary.json`.
  - Dispatched `coze:super-agent-session-select` in the page.
  - UI rendered `上下文已自动压缩`, `42 条历史消息`, `压缩 26 条`, `保留 16 条`, and `/workspace/.agent/context-summary.json`.
  - Deleted the temporary summary through `POST /api/super-agent/workspace/delete`.
  - Cleanup check confirmed `/workspace/.agent/context-summary.json` no longer exists.

Evidence files:

- `super-agent-playwright-context-trace-api-probe-20260620.json`
- `super-agent-playwright-context-trace-ui-final-20260620.json`
- `super-agent-playwright-context-cleanup-check-20260620.json`
- `super-agent-context-trace-ui-final-20260620.png`

## 2026-06-20: App Server approval decision persistence

Goal:

- Align the harness/App Server approval flow with Codex-style auditability.
- `POST /api/super-agent/approvals/resolve` should persist a machine-readable decision record under the sandbox workspace so external clients can recover and inspect approval decisions after the HTTP response.

Backend changes verified:

- `approvals.resolve` response now includes:
  - `decision_path`
  - `persisted`
- Decision records are written as JSON under `/workspace/.agent/approvals`.
- `GET /api/super-agent/manifest` now exposes:
  - `approvals.decision_root=/workspace/.agent/approvals`
  - `approvals.decision_format=json`
  - `approvals.response_fields` including `decision_path` and `persisted`.

Local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run TestSuperAgentManifestRouteReturnsAppServerContract -count=1
```

- PASS.

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgentApproval|TestSuperAgentHarnessSnapshotRouteIncludesTraceAndApprovals|TestSuperAgentManifestRouteReturnsAppServerContract|TestSuperAgentOpenAPIRouteReturnsMachineReadableSchema|TestSuperAgentTraceRouteIncludesContextCompactionEvent' -count=1
```

- PASS.

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -count=1
```

- PASS.

Build and deployment:

- Backend binary: `/tmp/openynet-approval-decision`
- Backend SHA256: `0fe0726875afd1048fd6f4fd71364782a48e800dfec25ea6247ebc70f31a0cf1`
- Host: `10.10.10.226:8896`
- Container: `coze-super`
- Static resources were not replaced.
- Kept one backend rollback only: `/app/openynet.bak-latest`.
- Pre-deploy `/app/openynet` SHA256: `e4108728507894780390729ea713def4e299d2b102a699fa3bd148fffa7eb001`
- New `/app/openynet` SHA256: `0fe0726875afd1048fd6f4fd71364782a48e800dfec25ea6247ebc70f31a0cf1`
- Disk after deployment: total `72G`, used `54G`, available `15G`, `79%`.

Playwright MCP verification:

- Page:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=approval_decision_20260620`
- Manifest probe:
  - `approvals.decision_root` returned `/workspace/.agent/approvals`.
  - `approvals.decision_format` returned `json`.
  - `approvals.response_fields` included `decision_path` and `persisted`.
- Runtime probe:
  - Called `POST /api/super-agent/approvals/resolve` with `decision=approve`.
  - Response returned `persisted=true`.
  - Response returned `decision_path=/workspace/.agent/approvals/run-playwright-approval-decision-1781939362834.json`.
  - Read back the decision through `POST /api/super-agent/workspace/read`.
  - Parsed JSON contained `version=v1`, matching `approval_id`, `decision=approve`, `status=approved`, `source=app_server`.
  - Deleted the probe file through `POST /api/super-agent/workspace/delete`.
  - Cleanup read returned a file-not-found error, confirming the probe file was removed.
- Container log scan:
  - `POST /api/super-agent/approvals/resolve` returned HTTP 200.
  - The only warning matched the intentional cleanup read after deleting the probe file.

Evidence files:

- `super-agent-playwright-approval-decision-final-20260620.json`

## 2026-06-20: Harness snapshot approval decision recovery

Goal:

- Make `POST /api/super-agent/harness/snapshot` self-contained for App Server clients.
- Pending approvals were already exposed as `approvals`; this change adds resolved/rejected/cancelled approval decision records as `approval_decisions`, read from `/workspace/.agent/approvals/*.json`.

Backend changes verified:

- `harness/snapshot` includes `approval_decisions` when `include_approvals=true`.
- Each decision item includes the persisted decision record plus `decision_path`, `size`, and `mtime`.
- Decision records are filtered by `conversation_id` and optional `run_id`.
- `GET /api/super-agent/manifest` now exposes `harness.snapshot_fields`, including `approval_decisions`.

Local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgentManifestRouteReturnsAppServerContract|TestSuperAgentHarnessSnapshotRouteIncludesTraceAndApprovals' -count=1
```

- PASS.

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgentApproval|TestSuperAgentHarnessSnapshot|TestSuperAgentManifestRouteReturnsAppServerContract|TestSuperAgentOpenAPIRouteReturnsMachineReadableSchema|TestSuperAgentTraceRouteIncludesContextCompactionEvent' -count=1
```

- PASS.

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -count=1
```

- PASS.

Build and deployment:

- Backend binary: `/tmp/openynet-approval-snapshot`
- Backend SHA256: `9845b549121f6e260ab9b43065fc0b1d4783a4ffecb5c72270d5529be88e0e34`
- Host: `10.10.10.226:8896`
- Container: `coze-super`
- Static resources were not replaced.
- Kept one backend rollback only: `/app/openynet.bak-latest`.
- Pre-deploy `/app/openynet` SHA256: `0fe0726875afd1048fd6f4fd71364782a48e800dfec25ea6247ebc70f31a0cf1`
- New `/app/openynet` SHA256: `9845b549121f6e260ab9b43065fc0b1d4783a4ffecb5c72270d5529be88e0e34`
- Disk after deployment: total `72G`, used `54G`, available `15G`, `79%`.

Playwright MCP verification:

- Page:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=approval_snapshot_20260620`
- Manifest probe:
  - `harness.snapshot_fields` included `approval_decisions`.
  - `approvals.response_fields` included `decision_path` and `persisted`.
- Runtime probe:
  - Called `POST /api/super-agent/approvals/resolve` with `decision=approve`.
  - Response returned `persisted=true`.
  - Called `POST /api/super-agent/harness/snapshot` with only `include_approvals=true`.
  - Snapshot returned `components=["approvals","approval_decisions"]`.
  - Snapshot returned one matching `approval_decisions` item:
    - `approval_id=run:playwright-approval-snapshot-1781940620930`
    - `status=approved`
    - `source=app_server`
    - `decision_path=/workspace/.agent/approvals/run-playwright-approval-snapshot-1781940620930.json`
  - Read back the same file through `POST /api/super-agent/workspace/read`.
  - Deleted the probe file through `POST /api/super-agent/workspace/delete`.
  - Cleanup read returned a file-not-found error, confirming the probe file was removed.
- Container log scan:
  - `POST /api/super-agent/approvals/resolve` returned HTTP 200.
  - `POST /api/super-agent/harness/snapshot` returned HTTP 200.
  - The only warning matched the intentional cleanup read after deleting the probe file.

Evidence files:

- `super-agent-playwright-approval-snapshot-final-20260620.json`

## 2026-06-20: Structured harness tool output entries

Goal:

- Make tool calls directly inspectable by App Server clients and frontend panels.
- `POST /api/super-agent/harness/tool-outputs` entries should expose the tool call id, structured arguments, and structured result/error instead of requiring clients to parse raw JSON content themselves.

Backend changes verified:

- Tool output entries now include:
  - `tool_call_id`
  - `arguments`
  - `result`
  - `error`
- Existing fields remain available:
  - `tool`
  - `status`
  - `arguments_preview`
  - `result_preview`
  - `summary`
- `GET /api/super-agent/manifest` now exposes `harness.tool_output_entry_fields`.

Local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent -run TestListSuperAgentHarnessToolOutputsReturnsStructuredEntries -count=1
```

- PASS.

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run TestSuperAgentManifestRouteReturnsAppServerContract -count=1
```

- PASS.

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent -count=1
```

- PASS.

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgentManifestRouteReturnsAppServerContract|TestSuperAgentHarnessSnapshot|TestSuperAgentOpenAPIRouteReturnsMachineReadableSchema' -count=1
```

- PASS.

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -count=1
```

- PASS.

Build and deployment:

- Backend binary: `/tmp/openynet-tool-output-entries`
- Backend SHA256: `3b8e0da9b9fc88c3e3cbd0aa4bce5d517c176e7df76b13644cfebf389801ac69`
- Host: `10.10.10.226:8896`
- Container: `coze-super`
- Static resources were not replaced.
- Kept one backend rollback only: `/app/openynet.bak-latest`.
- Pre-deploy `/app/openynet` SHA256: `9845b549121f6e260ab9b43065fc0b1d4783a4ffecb5c72270d5529be88e0e34`
- New `/app/openynet` SHA256: `3b8e0da9b9fc88c3e3cbd0aa4bce5d517c176e7df76b13644cfebf389801ac69`
- Disk after deployment: total `72G`, used `54G`, available `15G`, `79%`.

Playwright MCP verification:

- Page:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=tool_output_entries_20260620`
- Manifest probe:
  - `harness.tool_output_entry_fields` included `tool_call_id`, `arguments`, and `result`.
- Runtime probe:
  - Wrote `/workspace/.agent/tooloutputs/call-playwright-tool-output-1781941178321.json`.
  - Called `POST /api/super-agent/harness/tool-outputs` with `include_content=true`.
  - Matching entry returned:
    - `tool_call_id=call-playwright-tool-output-1781941178321`
    - `tool=run_bash`
    - `status=completed`
    - `arguments.command=printf hello`
    - `arguments.timeout_sec=3`
    - `result.exit_code=0`
    - `result.stdout=hello`
  - Called `POST /api/super-agent/harness/tool-outputs` for the single path and got the same structured entry plus raw content.
  - Deleted the probe file through `POST /api/super-agent/workspace/delete`.
  - Cleanup read returned a file-not-found error, confirming the probe file was removed.
- Container log scan:
  - Both `POST /api/super-agent/harness/tool-outputs` calls returned HTTP 200.
  - The only warning matched the intentional cleanup read after deleting the probe file.

Evidence files:

- `super-agent-playwright-tool-output-entries-final-20260620.json`

## 2026-06-20 15:56 CST - Structured Long Tool Output Offload

Goal:

- Make natural long tool outputs from the super-agent harness parseable by App Server clients.
- Replace raw `.txt` offload files with structured JSON that includes the tool name, arguments, status, result content, byte size, truncated flag, and summary.
- Wire the structured offload helper into real tools instead of only supporting manually written probe files.

Local changes:

- `offloadOrTruncate` now writes `/workspace/.agent/tooloutputs/<hash>.json` for long outputs.
- Added `offloadToolResultOrTruncate` and `toolOutputOffloadMeta`.
- Structured offload JSON fields include:
  - `version`
  - `tool_call_id`
  - `tool`
  - `status`
  - `arguments`
  - `summary`
  - `result.content`
  - `result.bytes`
  - `result.truncated`
- Connected structured offload metadata to:
  - `run_bash`
  - `read_file`
  - `grep`
  - `glob`
  - `web_search`
  - `web_fetch`
- Short outputs still return inline after UTF-8 cleanup.
- If structured write fails, the model still receives the existing truncated inline fallback.

TDD evidence:

```bash
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow -run TestOffloadOrTruncate -count=1
```

- RED before implementation: failed with `structured tool output was not written to sandbox file`.
- GREEN after implementation: passed.

```bash
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow -run 'TestOffloadToolResultOrTruncateWritesToolArguments|TestOffloadOrTruncate' -count=1
```

- RED before implementation: failed because `offloadToolResultOrTruncate` and `toolOutputOffloadMeta` were undefined.
- GREEN after implementation: passed.

Local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow -count=1
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgentManifestRouteReturnsAppServerContract|TestSuperAgentHarnessSnapshotRouteIncludesToolOutputContent' -count=1
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow ./application/singleagent ./api/router/coze -count=1
git diff --check -- backend/domain/agent/singleagent/internal/agentflow/node_tool_policy.go backend/domain/agent/singleagent/internal/agentflow/node_tool_policy_test.go backend/domain/agent/singleagent/internal/agentflow/node_tool_sandbox.go backend/domain/agent/singleagent/internal/agentflow/node_tool_extensions.go
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-tool-offload-json main.go
```

Build and deployment:

- Backend binary: `/tmp/openynet-tool-offload-json`
- Backend SHA256: `3666b5a1ef3e77ef797c9d3aa5854d24321b5cc372a0615c8144808e73f8387c`
- Host: `10.10.10.226:8896`
- Container: `coze-super`
- Static resources were not replaced.
- Kept one backend rollback only: `/app/openynet.bak-latest`.
- Pre-deploy `/app/openynet` SHA256: `3b8e0da9b9fc88c3e3cbd0aa4bce5d517c176e7df76b13644cfebf389801ac69`
- New `/app/openynet` SHA256: `3666b5a1ef3e77ef797c9d3aa5854d24321b5cc372a0615c8144808e73f8387c`
- Rollback `/app/openynet.bak-latest` SHA256: `3b8e0da9b9fc88c3e3cbd0aa4bce5d517c176e7df76b13644cfebf389801ac69`
- Disk after deployment: total `72G`, used `54G`, available `15G`, `79%`.

Playwright MCP verification:

- Page:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=tool_offload_json_20260620`
- Manifest probe:
  - `GET /api/super-agent/manifest` returned HTTP `200`, `code=0`.
  - `harness.tool_output_entry_fields` includes `arguments` and `result`.
- Runtime probe:
  - Wrote `/workspace/.agent/tooloutputs/tool-offload-json-1781942159091.json` with the same JSON shape produced by long-output offload.
  - Called `POST /api/super-agent/harness/tool-outputs` with `include_content=true`.
  - Matching entry returned:
    - `tool=run_bash`
    - `status=completed`
    - `arguments.command=python3 - <<'PY'...`
    - `arguments.timeout_sec=3`
    - `result.bytes=4560`
    - `result.truncated=true`
    - `result_preview` includes `structured-offload`
  - Called `POST /api/super-agent/harness/tool-outputs` for the single path and received the same entry plus raw content.
  - Deleted the probe file through `POST /api/super-agent/sandbox/exec`.
  - Verified on the container that `/workspace/.agent/tooloutputs/tool-offload-json-1781942159091.json` no longer exists.
- Container log scan:
  - `POST /api/super-agent/sandbox/exec` returned HTTP `200` for write and cleanup.
  - Both `POST /api/super-agent/harness/tool-outputs` calls returned HTTP `200`.
  - No related error lines were present in the scanned tail.

Evidence files:

- `super-agent-playwright-tool-offload-json-final-20260620.json`

## 2026-06-20 16:11 CST - Harness Tool Output Cleanup Route

Goal:

- Add a narrow App Server cleanup surface for super-agent harness tool output files.
- Keep cleanup restricted to `/workspace/.agent/tooloutputs` so external clients can reclaim long tool-output storage without touching workspace, uploads, outputs, skills, or unrelated system files.
- Support dry-run before deletion for safer operational use on the shared `10.10.10.226` test environment.

Local changes:

- Added `POST /api/super-agent/harness/cleanup`.
- Added `CleanupSuperAgentHarnessToolOutputs` application service.
- Cleanup request fields:
  - `space_id`
  - `agent_id`
  - `bot_id`
  - `connector_id`
  - `paths`
  - `dry_run`
- Cleanup response fields:
  - `root`
  - `dry_run`
  - `matched`
  - `deleted`
  - `paths`
- Each requested path is normalized through `sanitizeSuperAgentHarnessToolOutputPath`.
- Empty path lists are rejected.
- The tool output root itself is rejected as a deletion target.
- Paths outside `/workspace/.agent/tooloutputs` are rejected.
- Manifest/OpenAPI now advertise `harness.cleanup` as a mutating harness operation.

TDD evidence:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent -run 'TestCleanupSuperAgentHarnessToolOutputsDryRunAndApply|TestCleanupSuperAgentHarnessToolOutputsRejectsPathOutsideRoot' -count=1
```

- RED before implementation: failed because `fakeHarnessCleanupSandboxManager`, `CleanupSuperAgentHarnessToolOutputs`, and `SuperAgentHarnessCleanupRequest` were undefined.
- GREEN after implementation: passed.

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgentManifestRouteReturnsAppServerContract|TestSuperAgentHarnessRoutesRejectMalformedJSON' -count=1
```

- RED before implementation: `/api/super-agent/harness/cleanup` returned HTTP `404` for malformed JSON.
- GREEN after implementation: passed.

Local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent ./api/router/coze -count=1
git diff --check -- backend/application/singleagent/sandbox_workspace.go backend/application/singleagent/sandbox_workspace_test.go backend/api/handler/coze/super_agent_harness_service.go backend/api/router/coze/api.go backend/api/handler/coze/super_agent_run_service.go backend/api/router/coze/super_agent_run_route_test.go
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-harness-cleanup main.go
```

Notes:

- Direct `go test ./api/handler/coze -count=1` still hits an existing test import cycle through `conversation_service_test.go -> application -> api/handler/coze`; `api/router/coze` compiles the handler path and passed, and the full backend build passed.

Build and deployment:

- Backend binary: `/tmp/openynet-harness-cleanup`
- Backend SHA256: `c58a089c82ae8515e82a436afb7421be6588ed5c369d805438105b5821cd6c79`
- Host: `10.10.10.226:8896`
- Container: `coze-super`
- Static resources were not replaced.
- Kept one backend rollback only: `/app/openynet.bak-latest`.
- Pre-deploy `/app/openynet` SHA256: `3666b5a1ef3e77ef797c9d3aa5854d24321b5cc372a0615c8144808e73f8387c`
- New `/app/openynet` SHA256: `c58a089c82ae8515e82a436afb7421be6588ed5c369d805438105b5821cd6c79`
- Rollback `/app/openynet.bak-latest` SHA256: `3666b5a1ef3e77ef797c9d3aa5854d24321b5cc372a0615c8144808e73f8387c`
- Disk after deployment: total `72G`, used `54G`, available `15G`, `79%`.

Playwright MCP verification:

- Page:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=harness_cleanup_20260620`
- Manifest/OpenAPI probe:
  - `GET /api/super-agent/manifest` returned HTTP `200`, `code=0`.
  - `harness.cleanup_route = POST /api/super-agent/harness/cleanup`.
  - `external_api.entry_routes.harness.cleanup = POST /api/super-agent/harness/cleanup`.
  - `external_api.request_schemas.harness.cleanup.required = ["paths"]`.
  - Cleanup schema optional fields include `dry_run`.
  - `GET /api/super-agent/openapi.json` includes `/api/super-agent/harness/cleanup`, `operationId=harness.cleanup`, and `x-mutates=true`.
- Runtime probe:
  - Wrote `/workspace/.agent/tooloutputs/cleanup-probe-1781943062961.json` via `POST /api/super-agent/sandbox/exec`.
  - `POST /api/super-agent/harness/tool-outputs` returned the probe entry with `tool=run_bash`.
  - `POST /api/super-agent/harness/cleanup` with `dry_run=true` returned `matched=1`, `deleted=0`.
  - A sandbox existence check after dry-run returned `exists`.
  - `POST /api/super-agent/harness/cleanup` without `dry_run` returned `matched=1`, `deleted=1`.
  - A sandbox existence check after cleanup returned `missing`.
  - Reading the single tool output path after cleanup returned business code `100000000` with file-not-found text.
  - Cleanup with path `/workspace/keep.txt` returned business code `100000000` and message `path must be under /workspace/.agent/tooloutputs`.
  - Container check confirmed `/workspace/.agent/tooloutputs/cleanup-probe-1781943062961.json` no longer exists.
- Container log scan:
  - `POST /api/super-agent/sandbox/exec` returned HTTP `200` for write and existence checks.
  - `POST /api/super-agent/harness/tool-outputs` returned HTTP `200`.
  - `POST /api/super-agent/harness/cleanup` returned HTTP `200` for dry-run, apply, and rejected outside-path business response.
  - No related error lines were present in the scanned tail.

Evidence files:

- `super-agent-playwright-harness-cleanup-final-20260620.json`

## 2026-06-20 16:38 CST - Harness Cleanup `keep_latest` Prefix Isolation

Goal:

- Make `harness.cleanup` safe for shared tool-output directories when using retention cleanup.
- Let App Server clients clean only their own run/session/probe group by passing `prefix` with `keep_latest`, instead of applying retention across every file under `/workspace/.agent/tooloutputs`.

Local changes:

- `SuperAgentHarnessCleanupRequest` now accepts `prefix`.
- `SuperAgentHarnessCleanupData` returns the normalized prefix when supplied.
- `keep_latest` candidate selection filters files by prefix before sorting by `mtime`.
- Prefix values may be relative to `/workspace/.agent/tooloutputs` or absolute under that root.
- Prefix values outside `/workspace/.agent/tooloutputs` are rejected.
- Manifest/OpenAPI request schema for `harness.cleanup` advertises optional `prefix`.

TDD evidence:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent -run TestCleanupSuperAgentHarnessToolOutputsKeepLatestWithPrefix -count=1
```

- RED before implementation: failed because `SuperAgentHarnessCleanupRequest` did not have field `Prefix`.
- GREEN after implementation: passed.

Local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent -run 'TestCleanupSuperAgentHarnessToolOutputsKeepLatestWithPrefix|TestCleanupSuperAgentHarnessToolOutputsKeepLatest|TestCleanupSuperAgentHarnessToolOutputsRejectsPathOutsideRoot' -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run TestSuperAgentManifestRouteReturnsAppServerContract -count=1
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent ./api/router/coze -count=1
git diff --check -- backend/application/singleagent/sandbox_workspace.go backend/application/singleagent/sandbox_workspace_test.go backend/api/handler/coze/super_agent_run_service.go backend/api/router/coze/super_agent_run_route_test.go
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-harness-cleanup-prefix main.go
```

Build and deployment:

- Backend binary: `/tmp/openynet-harness-cleanup-prefix`
- Backend SHA256: `354babd69ac8f6516b10f7cd6cdf03954f8adc29d207ebb8348716543b6ff304`
- Host: `10.10.10.226:8896`
- Container: `coze-super`
- Static resources were not replaced.
- Kept one backend rollback only: `/app/openynet.bak-latest`.
- Pre-deploy `/app/openynet` SHA256: `62b2b05fc6e7cd318d357d47d71491efcabc7e4025c3a59da377e7eeb2b9a222`
- New `/app/openynet` SHA256: `354babd69ac8f6516b10f7cd6cdf03954f8adc29d207ebb8348716543b6ff304`
- Rollback `/app/openynet.bak-latest` SHA256: `62b2b05fc6e7cd318d357d47d71491efcabc7e4025c3a59da377e7eeb2b9a222`
- Disk after deployment: total `72G`, used `54G`, available `15G`, `79%`.

Playwright MCP verification:

- Page:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=harness_cleanup_prefix_20260620`
- Manifest probe:
  - `GET /api/super-agent/manifest` returned HTTP `200`, `code=0`.
  - `external_api.entry_routes.harness.cleanup = POST /api/super-agent/harness/cleanup`.
  - `external_api.request_schemas.harness.cleanup.optional` includes `prefix`.
  - `required_one_of = [["paths"], ["keep_latest"]]`.
- Runtime probe:
  - Created four probe files via `POST /api/super-agent/sandbox/exec`:
    - prefixed old/mid/new JSON files under `/workspace/.agent/tooloutputs`
    - unrelated legacy file under the same root
  - `POST /api/super-agent/harness/cleanup` with `keep_latest=1`, the unique prefix, and `dry_run=true` returned:
    - `matched=2`
    - `deleted=0`
    - paths only for the prefixed old/mid files
    - no unrelated legacy path
    - no newest prefixed path
  - Applying the same cleanup without `dry_run` returned `matched=2`, `deleted=2`.
  - Existence check after apply confirmed:
    - old and mid were deleted
    - newest prefixed file remained
    - unrelated legacy file remained
  - Explicit cleanup of the remaining newest and legacy probe files returned `deleted=2`.
  - Final existence check confirmed all probe files were removed.

Evidence files:

- `super-agent-playwright-harness-cleanup-prefix-final-20260620.json`

## 2026-06-20 16:59 CST - Session-Scoped Harness Context Summary

Goal:

- Prevent super-agent automatic context compaction from leaking across multiple conversations that share the same user, agent, connector, and sandbox.
- Store new compacted summaries under a conversation-scoped path while keeping the old global summary path as a compatibility fallback.

Local changes:

- Added `conversation_id` to the crossdomain agent runtime request path.
- The agent run layer now passes `ConversationID` through:
  - conversation agent run service
  - internal agent runtime path
  - crossdomain agent contract
  - crossdomain single-agent implementation
  - single-agent service
  - agentflow `AgentRequest`
- Super-agent context compaction now writes:
  - with conversation id: `/workspace/.agent/sessions/<conversation_id>/context-summary.json`
  - without conversation id: `/workspace/.agent/context-summary.json`
- `formatContextSummaryForModel` now references the actual persisted summary path.
- `harness.state`, `harness.snapshot`, and `traces/get` read conversation-scoped summaries when `conversation_id` is available, and fall back to the legacy global summary path.
- Manifest/OpenAPI request schema for `harness.state` advertises optional `conversation_id`.

TDD evidence:

```bash
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow -run TestPreHandlerReqAutoCompactsSuperAgentHistoryByConversation -count=1
```

- RED before implementation: failed because `AgentRequest` had no `ConversationID` field.
- GREEN after implementation: passed and verified the summary was written to `/workspace/.agent/sessions/456/context-summary.json`, not the global path.

Local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow -run 'TestPreHandlerReqAutoCompactsSuperAgentHistoryByConversation|TestPreHandlerReqAutoCompactsSuperAgentHistory|TestPreHandlerReqDoesNotCompactNormalAgentHistory' -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgentHarnessSnapshotRouteUsesConversationScopedContextState|TestSuperAgentHarnessSnapshotRouteIncludesContextState|TestSuperAgentManifestRouteReturnsAppServerContract|TestSuperAgentTraceGetIncludesContextCompactedEvent' -count=1
SESSION_HMAC_SECRET=test-secret go test ./crossdomain/impl/singleagent ./domain/agent/singleagent/internal/agentflow ./domain/conversation/agentrun/service ./application/singleagent ./api/router/coze -count=1
git diff --check -- backend/api/model/crossdomain/singleagent/single_agent.go backend/crossdomain/contract/agent/single_agent.go backend/crossdomain/impl/singleagent/single_agent.go backend/domain/agent/singleagent/service/single_agent_impl.go backend/domain/conversation/agentrun/service/agent_run_impl.go backend/domain/conversation/agentrun/internal/singleagent_run.go backend/domain/agent/singleagent/internal/agentflow/agent_flow_runner.go backend/domain/agent/singleagent/internal/agentflow/agent_flow_runner_test.go backend/api/model/app/developer_api/sandbox_workspace.go backend/application/singleagent/sandbox_workspace.go backend/api/handler/coze/super_agent_harness_service.go backend/api/handler/coze/super_agent_trace_service.go backend/api/handler/coze/super_agent_run_service.go backend/api/router/coze/super_agent_run_route_test.go
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-session-context-summary main.go
```

Build and deployment:

- Backend binary: `/tmp/openynet-session-context-summary`
- Backend SHA256: `7762ef8234f2e32206d8a49cb3e882f1f2b05bc7032c1dca65f4c05746ece7de`
- Host: `10.10.10.226:8896`
- Container: `coze-super`
- Static resources were not replaced.
- Kept one backend rollback only: `/app/openynet.bak-latest`.
- Pre-deploy `/app/openynet` SHA256: `354babd69ac8f6516b10f7cd6cdf03954f8adc29d207ebb8348716543b6ff304`
- New `/app/openynet` SHA256: `7762ef8234f2e32206d8a49cb3e882f1f2b05bc7032c1dca65f4c05746ece7de`
- Rollback `/app/openynet.bak-latest` SHA256: `354babd69ac8f6516b10f7cd6cdf03954f8adc29d207ebb8348716543b6ff304`
- Disk after deployment: total `72G`, used `54G`, available `15G`, `79%`.

Playwright MCP verification:

- Page:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=session_context_summary_20260620`
- Manifest probe:
  - `GET /api/super-agent/manifest` returned HTTP `200`, `code=0`.
  - `external_api.request_schemas.harness.state.optional` includes `conversation_id`.
- Runtime probe:
  - Created temporary session `7653399285727232000` through `POST /api/super-agent/sessions/create`.
  - Wrote `/workspace/.agent/sessions/7653399285727232000/context-summary.json` through `POST /api/super-agent/sandbox/exec`.
  - `POST /api/super-agent/harness/snapshot` with the temporary `conversation_id` returned:
    - `context.summary_path = /workspace/.agent/sessions/7653399285727232000/context-summary.json`
    - `context.summary.summary_path = /workspace/.agent/sessions/7653399285727232000/context-summary.json`
    - `context.summary.run_id = run-1781945881450`
  - `POST /api/super-agent/harness/state` with the same `conversation_id` returned the same session-scoped context summary path.
  - `POST /api/super-agent/traces/get` returned a `context.compacted` trace event with:
    - `conversation_id = 7653399285727232000`
    - `run_id = run-1781945881450`
    - `metadata.summary_path = /workspace/.agent/sessions/7653399285727232000/context-summary.json`
  - Removed the temporary summary file through `POST /api/super-agent/sandbox/exec`.
  - Existence check confirmed the temporary summary file no longer exists.
  - Deleted the temporary session through `POST /api/super-agent/sessions/delete`.

Evidence files:

- `super-agent-playwright-session-context-summary-final-20260620.json`

## 2026-06-20 - Harness Session Context Strict Isolation

Goal:

- Tighten the previous session-scoped context summary behavior so a request with `conversation_id` never falls back to `/workspace/.agent/context-summary.json`.
- Keep the legacy global summary path only for requests without `conversation_id`.

Local changes:

- `superAgentHarnessContextState` now reads exactly one summary path:
  - with conversation id: `/workspace/.agent/sessions/<conversation_id>/context-summary.json`
  - without conversation id: `/workspace/.agent/context-summary.json`
- Added a regression test proving `harness.snapshot` with `conversation_id` returns `summary_exists=false` when the session summary is missing, even if a global summary exists.
- Updated trace/snapshot tests to use session-scoped summary fixtures when the request carries `conversation_id`.

TDD evidence:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgentHarnessSnapshotRouteDoesNotFallbackGlobalContextWhenConversationScopedMissing|TestSuperAgentHarnessSnapshotRouteUsesConversationScopedContextState|TestSuperAgentHarnessSnapshotRouteIncludesContextState' -count=1
```

- RED before implementation: failed because `harness.snapshot` fell back to `/workspace/.agent/context-summary.json`.
- GREEN after implementation: passed and returned `/workspace/.agent/sessions/123/context-summary.json` with `summary_exists=false`.

Local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -count=1
SESSION_HMAC_SECRET=test-secret go test ./crossdomain/impl/singleagent ./domain/agent/singleagent/internal/agentflow ./domain/conversation/agentrun/service ./application/singleagent ./api/router/coze -count=1
git diff --check -- backend/application/singleagent/sandbox_workspace.go backend/api/router/coze/super_agent_run_route_test.go
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-session-context-strict main.go
```

Build and deployment:

- Backend binary: `/tmp/openynet-session-context-strict`
- Backend SHA256: `c6ac576cce3d39c58e9bde25ad42b674e50ef146b971bef0ab3f5a94ba6b93a9`
- Host: `10.10.10.226:8896`
- Container: `coze-super`
- Static resources were not replaced.
- Kept one backend rollback only: `/app/openynet.bak-latest`.
- Pre-deploy `/app/openynet` SHA256: `7762ef8234f2e32206d8a49cb3e882f1f2b05bc7032c1dca65f4c05746ece7de`
- New `/app/openynet` SHA256: `c6ac576cce3d39c58e9bde25ad42b674e50ef146b971bef0ab3f5a94ba6b93a9`
- Rollback `/app/openynet.bak-latest` SHA256: `7762ef8234f2e32206d8a49cb3e882f1f2b05bc7032c1dca65f4c05746ece7de`
- Disk after deployment: total `72G`, used `54G`, available `15G`, `79%`.

Playwright MCP verification:

- Page:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=session_context_strict_20260620`
- Runtime probe:
  - Created temporary session `7653404188629532672` through `POST /api/super-agent/sessions/create`.
  - Backed up `/workspace/.agent/context-summary.json`; it did not exist before this probe.
  - Wrote a global probe summary to `/workspace/.agent/context-summary.json`.
  - `POST /api/super-agent/harness/snapshot` with the temporary `conversation_id` and no session summary returned:
    - `context.summary_path = /workspace/.agent/sessions/7653404188629532672/context-summary.json`
    - `context.summary_exists = false`
    - `context.summary = null`
  - Wrote `/workspace/.agent/sessions/7653404188629532672/context-summary.json`.
  - `POST /api/super-agent/harness/snapshot` with the same `conversation_id` returned the session-scoped summary and did not expose the global probe.
  - `POST /api/super-agent/traces/get` returned a `context.compacted` trace event with:
    - `metadata.summary_path = /workspace/.agent/sessions/7653404188629532672/context-summary.json`
    - `content = SESSION_PROBE_VISIBLE_STRICT_1781947023281`
  - Removed the temporary session summary.
  - Removed the temporary global probe summary.
  - Deleted the temporary session.

Evidence files:

- `super-agent-playwright-session-context-strict-final-20260620.json`

## 2026-06-20 - Harness Context Clear API

Goal:

- Add an App Server-visible harness API for clearing compressed context summaries.
- When `conversation_id` is provided, clear only `/workspace/.agent/sessions/<conversation_id>/context-summary.json`.
- Do not expose arbitrary path deletion through this endpoint, and do not delete the legacy global summary during a session-scoped clear.

Local changes:

- Added `POST /api/super-agent/harness/context/clear`.
- Added `SuperAgentHarnessContextClearRequest` / `SuperAgentHarnessContextClearResponse`.
- Added `SingleAgentApplicationService.ClearSuperAgentHarnessContext`.
- Added manifest/OpenAPI exposure:
  - `harness.context_clear`
  - `harness.context_clear_route`
  - `HarnessContextClearRequest`
- Added route tests proving:
  - session-scoped clear deletes `/workspace/.agent/sessions/123/context-summary.json`
  - global `/workspace/.agent/context-summary.json` is not deleted by session-scoped clear
  - manifest/OpenAPI exposes the new operation.

TDD evidence:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgentHarnessContextClearRouteDeletesConversationScopedSummaryOnly|TestSuperAgentManifestRouteIncludesHarnessContextClearOperation' -count=1
```

- RED before implementation:
  - `POST /api/super-agent/harness/context/clear` returned HTTP `404`.
  - Manifest had no `context_clear_route` / `harness.context_clear`.
- GREEN after implementation: both tests passed.

Local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgentHarnessContextClearRouteDeletesConversationScopedSummaryOnly|TestSuperAgentManifestRouteIncludesHarnessContextClearOperation' -count=1
SESSION_HMAC_SECRET=test-secret go test ./crossdomain/impl/singleagent ./domain/agent/singleagent/internal/agentflow ./domain/conversation/agentrun/service ./application/singleagent ./api/router/coze -count=1
git diff --check -- backend/api/model/app/developer_api/sandbox_workspace.go backend/application/singleagent/sandbox_workspace.go backend/api/handler/coze/super_agent_harness_service.go backend/api/handler/coze/super_agent_run_service.go backend/api/router/coze/api.go backend/api/router/coze/super_agent_run_route_test.go
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-harness-context-clear main.go
```

Build and deployment:

- Backend binary: `/tmp/openynet-harness-context-clear`
- Backend SHA256: `226f583f489796ca1538450b59be127c01ecf5732b3c4fb4d97d88ff84f56205`
- Host: `10.10.10.226:8896`
- Container: `coze-super`
- Static resources were not replaced.
- Kept one backend rollback only: `/app/openynet.bak-latest`.
- Pre-deploy `/app/openynet` SHA256: `c6ac576cce3d39c58e9bde25ad42b674e50ef146b971bef0ab3f5a94ba6b93a9`
- New `/app/openynet` SHA256: `226f583f489796ca1538450b59be127c01ecf5732b3c4fb4d97d88ff84f56205`
- Rollback `/app/openynet.bak-latest` SHA256: `c6ac576cce3d39c58e9bde25ad42b674e50ef146b971bef0ab3f5a94ba6b93a9`
- Disk after deployment: total `72G`, used `54G`, available `15G`, `79%`.

Playwright MCP verification:

- Page:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=harness_context_clear_20260620`
- Runtime probe:
  - `GET /api/super-agent/manifest` exposed:
    - `harness.context_clear_route = POST /api/super-agent/harness/context/clear`
    - `external_api.entry_routes["harness.context_clear"]`
    - `openapi.operations[].operation_id = harness.context_clear`
  - Created temporary session `7653408386238644224`.
  - Backed up `/workspace/.agent/context-summary.json`; it did not exist before this probe.
  - Wrote a global probe summary to `/workspace/.agent/context-summary.json`.
  - Wrote a session summary to `/workspace/.agent/sessions/7653408386238644224/context-summary.json`.
  - `POST /api/super-agent/harness/snapshot` confirmed the session summary existed before clear.
  - `POST /api/super-agent/harness/context/clear` returned:
    - `summary_path = /workspace/.agent/sessions/7653408386238644224/context-summary.json`
    - `summary_exists_before = true`
    - `cleared = true`
  - `POST /api/super-agent/harness/snapshot` confirmed the session summary no longer existed.
  - `POST /api/super-agent/workspace/read` confirmed the global probe summary survived the session-scoped clear.
  - Removed the temporary session summary.
  - Removed the temporary global probe summary.
  - Deleted the temporary session.

Evidence files:

- `super-agent-playwright-harness-context-clear-final-20260620.json`

## 2026-06-20 - Session Delete Cleans Harness Context Summary

Question addressed:

- The harness already has automatic context compaction for super agents:
  - runner trigger: history bytes exceed `AGENT_CONTEXT_COMPACT_MAX_BYTES` (default `160KB`)
  - recent tail: `AGENT_CONTEXT_COMPACT_RECENT_MESSAGES` (default `16`)
  - persisted summary:
    - session-scoped: `/workspace/.agent/sessions/<conversation_id>/context-summary.json`
    - legacy/global: `/workspace/.agent/context-summary.json`
  - injected model message: `Context summary (auto-compacted)`
  - visible trace event: `context.compacted`
- Gap fixed in this slice:
  - deleting a durable session must also clear that session-scoped harness summary
  - session deletion must not delete the legacy/global summary
- Remaining harness alignment items:
  - LLM-quality summarization instead of the current rule-based snippet summary
  - explicit compaction policy in manifest/runtime state
  - per-session lifecycle cleanup for summaries, tool outputs, artifacts, approvals, and plans
  - quota/retention policy for `/workspace/.agent/*`
  - UI visibility for context strategy and compaction events

Local changes:

- Updated `SuperAgentDeleteSession` to call harness context cleanup after successful conversation deletion.
- Added owner-scoped cleanup context for app-server calls that pass `user_id` but do not have browser/session auth in context.
- Kept cleanup best-effort: session deletion still succeeds if sandbox cleanup fails.
- Extended `TestSuperAgentSessionDeleteRouteDeletesOwnedSession` to assert:
  - `/workspace/.agent/sessions/123/context-summary.json` is removed
  - `/workspace/.agent/context-summary.json` is not removed

Local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run TestSuperAgentSessionDeleteRouteDeletesOwnedSession -count=1
SESSION_HMAC_SECRET=test-secret go test ./crossdomain/impl/singleagent ./domain/agent/singleagent/internal/agentflow ./domain/conversation/agentrun/service ./application/singleagent ./api/router/coze -count=1
git diff --check -- backend/api/handler/coze/super_agent_session_service.go backend/api/router/coze/super_agent_run_route_test.go
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-session-delete-context-cleanup main.go
```

Results:

- Target test passed.
- Related backend packages passed.
- Linux build passed.

Build and deployment:

- Backend binary: `/tmp/openynet-session-delete-context-cleanup`
- Backend SHA256: `bfbcf5abddc3be5b61b0851ca12f011920380643904f334bab5be761c221bece`
- Host: `10.10.10.226:8896`
- Container: `coze-super`
- Static resources were not replaced.
- Kept one backend rollback only: `/app/openynet.bak-latest`.
- Pre-deploy `/app/openynet` SHA256: `226f583f489796ca1538450b59be127c01ecf5732b3c4fb4d97d88ff84f56205`
- New `/app/openynet` SHA256: `bfbcf5abddc3be5b61b0851ca12f011920380643904f334bab5be761c221bece`
- Rollback `/app/openynet.bak-latest` SHA256: `226f583f489796ca1538450b59be127c01ecf5732b3c4fb4d97d88ff84f56205`
- Disk after deployment: total `72G`, used `54G`, available `15G`, `79%`.

Playwright MCP verification:

- Page:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=session_delete_context_cleanup_20260620`
- Runtime probe:
  - Created temporary session `7653411445048082432`.
  - Wrote session summary to `/workspace/.agent/sessions/7653411445048082432/context-summary.json`.
  - `POST /api/super-agent/workspace/read` confirmed the summary was readable before deletion.
  - `POST /api/super-agent/harness/state` returned:
    - `summary_exists = true`
    - `summary_path = /workspace/.agent/sessions/7653411445048082432/context-summary.json`
    - `strategy = tool-output-offload+session-summary`
  - `POST /api/super-agent/sessions/delete` deleted the temporary session.
  - A second `POST /api/super-agent/workspace/read` on the same summary path returned app error `100000000` with `No such file or directory`, proving the summary was removed.
  - Legacy/global `/workspace/.agent/context-summary.json` state was unchanged.

Evidence files:

- `super-agent-playwright-session-delete-context-cleanup-pass-20260620.json`

## 2026-06-20 - Session Delete Cleans Session Tool Outputs

Question addressed:

- Harness long tool outputs are now session-scoped for super-agent runs:
  - session path: `/workspace/.agent/tooloutputs/sessions/<conversation_id>/<digest>.json`
  - legacy/global path remains supported: `/workspace/.agent/tooloutputs/<digest>.json`
  - offload JSON includes `conversation_id` when a run has a conversation id
- Gap fixed in this slice:
  - deleting a durable session also removes `/workspace/.agent/tooloutputs/sessions/<conversation_id>`
  - deleting a session does not remove legacy/global tool outputs

Local changes:

- `AgentRunner.StreamExecute` now passes `ConversationID` into tool-output offload context.
- `offloadToolResultOrTruncate` writes new run-scoped tool outputs under the session directory and preserves the legacy fallback when no conversation id is available.
- `SuperAgentDeleteSession` now best-effort removes the session tool-output directory after conversation deletion.
- Extended route and agentflow tests to cover session-scoped offload and cleanup.

TDD evidence:

```bash
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow -run TestOffloadToolResultOrTruncateScopesOutputToConversation -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run TestSuperAgentSessionDeleteRouteDeletesOwnedSession -count=1
```

- RED before implementation:
  - `withToolOutputConversationID` was missing for offload scoping.
  - session deletion did not issue `rm -rf '/workspace/.agent/tooloutputs/sessions/123'`.
- GREEN after implementation: both target tests passed.

Local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow ./application/singleagent ./api/router/coze -count=1
git diff --check -- backend/domain/agent/singleagent/internal/agentflow/node_tool_policy.go backend/domain/agent/singleagent/internal/agentflow/node_tool_policy_test.go backend/domain/agent/singleagent/internal/agentflow/agent_flow_runner.go backend/api/handler/coze/super_agent_session_service.go backend/api/router/coze/super_agent_run_route_test.go
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-session-tooloutput-cleanup main.go
```

Results:

- Target tests passed.
- Related backend packages passed.
- Linux build passed.

Build and deployment:

- Backend binary: `/tmp/openynet-session-tooloutput-cleanup`
- Backend SHA256: `21d0c57cc79d2d163442108838ff15cdcf76106e4149a396f58f9dca9a2588ac`
- Host: `10.10.10.226:8896`
- Container: `coze-super`
- Static resources were not replaced.
- Kept one backend rollback only: `/app/openynet.bak-latest`.
- Pre-deploy `/app/openynet` SHA256: `bfbcf5abddc3be5b61b0851ca12f011920380643904f334bab5be761c221bece`
- New `/app/openynet` SHA256: `21d0c57cc79d2d163442108838ff15cdcf76106e4149a396f58f9dca9a2588ac`
- Rollback `/app/openynet.bak-latest` SHA256: `bfbcf5abddc3be5b61b0851ca12f011920380643904f334bab5be761c221bece`
- Disk after deployment: total `72G`, used `54G`, available `15G`, `79%`.

Playwright MCP verification:

- Page:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=session_tooloutput_cleanup_20260620`
- Runtime probe:
  - Created temporary session `7653414881147748352`.
  - Wrote session tool output to `/workspace/.agent/tooloutputs/sessions/7653414881147748352/probe-tool-output.json`.
  - Wrote legacy/global tool output to `/workspace/.agent/tooloutputs/legacy-probe-7653414881147748352.json`.
  - `POST /api/super-agent/harness/tool-outputs` read the session tool output with `tool=grep` and a compact summary.
  - `POST /api/super-agent/sessions/delete` deleted the temporary session.
  - A second workspace read for the session tool-output path returned app error `100000000` with `No such file or directory`, proving the session tool output was removed.
  - Legacy/global tool output remained readable after session deletion and was cleaned by the probe.

Evidence files:

- `super-agent-playwright-session-tooloutput-cleanup-pass-20260620.json`

## 2026-06-20 - Harness Context Policy Contract

Question addressed:

- The super-agent already auto-compacts context, but external App Server clients need a discoverable contract instead of inferring hidden code defaults.
- Added a shared context compaction policy surface to:
  - `GET /api/super-agent/manifest` under `data.harness.context_policy`
  - `POST /api/super-agent/harness/state` under `data.context.policy`

Policy fields:

- `strategy = tool-output-offload+session-summary`
- `summary_version = v1`
- `trigger = history_bytes_exceeded`
- `max_bytes = 163840`
- `recent_messages = 16`
- `summary_max_runes = 1600`
- `summary_path = /workspace/.agent/context-summary.json`
- `session_summary_path_template = /workspace/.agent/sessions/{conversation_id}/context-summary.json`
- `clear_route = POST /api/super-agent/harness/context/clear`

TDD evidence:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run TestSuperAgentManifestRouteIncludesHarnessContextClearOperation -count=1
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent -run TestGetSuperAgentHarnessStateReturnsPlanAndToolOutputs -count=1
```

- RED before implementation:
  - manifest `context_policy` fields were empty.
  - harness state `context.policy` fields were empty.
- GREEN after implementation: both target tests passed.

Local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./domain/agent/singleagent/internal/agentflow ./application/singleagent ./api/router/coze -count=1
git diff --check -- api/model/app/developer_api/sandbox_workspace.go application/singleagent/sandbox_workspace.go api/handler/coze/super_agent_run_service.go application/singleagent/sandbox_workspace_test.go api/router/coze/super_agent_run_route_test.go ../docs/super-agent-8896-playwright-test-record.md
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-harness-context-policy main.go
```

Results:

- Target tests passed.
- Related backend packages passed.
- Linux build passed.

Build and deployment:

- Backend binary: `/tmp/openynet-harness-context-policy`
- Backend SHA256: `0d2caf03e745b6d4446c446a2b999f95629fe98fa6aa553ac974c0540b970e37`
- Host: `10.10.10.226:8896`
- Container: `coze-super`
- Static resources were not replaced.
- Kept one backend rollback only: `/app/openynet.bak-latest`.
- Pre-deploy `/app/openynet` SHA256: `21d0c57cc79d2d163442108838ff15cdcf76106e4149a396f58f9dca9a2588ac`
- New `/app/openynet` SHA256: `0d2caf03e745b6d4446c446a2b999f95629fe98fa6aa553ac974c0540b970e37`
- Rollback `/app/openynet.bak-latest` SHA256: `21d0c57cc79d2d163442108838ff15cdcf76106e4149a396f58f9dca9a2588ac`
- Disk after deployment: total `72G`, used `54G`, available `15G`, `79%`.

Playwright MCP verification:

- Page:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=harness_context_policy_20260620`
- Runtime probe:
  - Browser-context `GET /api/super-agent/manifest` returned HTTP `200`, `code=0`.
  - Browser-context `POST /api/super-agent/harness/state` returned HTTP `200`, `code=0`.
  - Manifest `data.harness.context_policy` matched all expected policy fields.
  - State `data.context.policy` matched all expected policy fields.
  - Manifest still advertises `context` in `data.harness.state_fields`.

Evidence files:

- `super-agent-playwright-harness-context-policy-20260620.json`

## 2026-06-20 - Session-Scoped Tool Outputs in Harness State

Question addressed:

- After tool outputs moved under `/workspace/.agent/tooloutputs/sessions/<conversation_id>`, `harness/state`, `harness/tool-outputs`, and `harness/snapshot` must default to the current session directory when `conversation_id` is provided.
- Without this, Codex-style consoles and external App Server clients can miss the current session's long tool-output files.

Local changes:

- Added `path` to `SuperAgentHarnessToolOutputsState`.
- Added `conversation_id` to `SuperAgentHarnessToolOutputsRequest`.
- `POST /api/super-agent/harness/state` now returns:
  - `tool_outputs.root = /workspace/.agent/tooloutputs`
  - `tool_outputs.path = /workspace/.agent/tooloutputs/sessions/<conversation_id>` when a conversation id is present
- `POST /api/super-agent/harness/tool-outputs` now defaults to the same session path when `conversation_id` is present and `path` is omitted.
- `POST /api/super-agent/harness/snapshot` forwards `conversation_id` to the tool-output listing.
- Manifest request schema now advertises `conversation_id` for `harness.tool_outputs`.
- Legacy behavior remains available when no `conversation_id` is supplied: the default path stays `/workspace/.agent/tooloutputs`.

TDD evidence:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent -run TestGetSuperAgentHarnessStateDefaultsToolOutputsToConversationScope -count=1
```

- RED before implementation:
  - `tool_outputs.path` was empty in the state JSON.
  - state listed legacy root file `/workspace/.agent/tooloutputs/call-1.json` instead of `/workspace/.agent/tooloutputs/sessions/456/call-session.json`.
- GREEN after implementation: target test passed.

Local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent ./api/router/coze ./domain/agent/singleagent/internal/agentflow -count=1
git diff --check -- api/model/app/developer_api/sandbox_workspace.go application/singleagent/sandbox_workspace.go api/handler/coze/super_agent_harness_service.go api/handler/coze/super_agent_run_service.go application/singleagent/sandbox_workspace_test.go ../docs/super-agent-8896-playwright-test-record.md
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-session-tooloutputs-state main.go
```

Results:

- Target test passed.
- Related backend packages passed.
- Linux build passed.

Build and deployment:

- Backend binary: `/tmp/openynet-session-tooloutputs-state`
- Backend SHA256: `b915108a9841a44644d21d287e3e4841492a7b5a77374fc593347c92c4359011`
- Host: `10.10.10.226:8896`
- Container: `coze-super`
- Static resources were not replaced.
- Kept one backend rollback only: `/app/openynet.bak-latest`.
- Pre-deploy `/app/openynet` SHA256: `0d2caf03e745b6d4446c446a2b999f95629fe98fa6aa553ac974c0540b970e37`
- New `/app/openynet` SHA256: `b915108a9841a44644d21d287e3e4841492a7b5a77374fc593347c92c4359011`
- Rollback `/app/openynet.bak-latest` SHA256: `0d2caf03e745b6d4446c446a2b999f95629fe98fa6aa553ac974c0540b970e37`
- Disk after deployment: total `72G`, used `54G`, available `15G`, `79%`.

Playwright MCP verification:

- Page:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=session_tooloutputs_state_20260620`
- Runtime probe:
  - First probe file `super-agent-playwright-session-tooloutputs-state-20260620.json` failed because the probe script read `data.conversation_id`; the actual response nests the id under `data.session.conversation_id`.
  - The corrected probe removed the stale temporary session `7653420126699520000`.
  - Created temporary session `7653420366437548032`.
  - Wrote `/workspace/.agent/tooloutputs/sessions/7653420366437548032/probe-state.json`.
  - `POST /api/super-agent/harness/state` returned `tool_outputs.path = /workspace/.agent/tooloutputs/sessions/7653420366437548032` and listed the probe file.
  - `POST /api/super-agent/harness/tool-outputs` with `conversation_id` and no `path` returned the same session path and a structured entry with `tool=grep`.
  - `POST /api/super-agent/harness/snapshot` returned only `tool_outputs` and pointed at the same session path.
  - Manifest request schema for `harness.tool_outputs` includes `conversation_id`.
  - Deleting the temporary session removed the session tool-output directory; reading the probe file afterwards returned app error `100000000` with `No such file or directory`.

Evidence files:

- `super-agent-playwright-session-tooloutputs-state-pass-20260620.json`

## 2026-06-20 - Session-Scoped Harness Plan

Question addressed:

- Harness summary and tool outputs were already session-scoped, but plan state still used the global `/workspace/.plan.json`.
- Multiple durable sessions for the same super-agent share one sandbox, so a global plan can leak one session's task state into another session.

Local changes:

- `POST /api/super-agent/harness/plan` accepts `conversation_id`.
- With `conversation_id`, plan updates write to `/workspace/.agent/sessions/<conversation_id>/plan.json`.
- `POST /api/super-agent/harness/state` reads the session plan path when `conversation_id` is present.
- `POST /api/super-agent/harness/snapshot` inherits the same state behavior.
- `GET /api/super-agent/manifest` exposes:
  - `harness.session_plan_path_template = /workspace/.agent/sessions/{conversation_id}/plan.json`
  - `external_api.request_schemas["harness.plan"].optional` includes `conversation_id`
- Deleting a session now removes `/workspace/.agent/sessions/<conversation_id>/plan.json`.
- Compatibility: without `conversation_id`, the default remains `/workspace/.plan.json`.

TDD evidence:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent -run 'TestGetSuperAgentHarnessStateDefaultsPlanToConversationScope|TestUpdateSuperAgentHarnessPlanWritesConversationScopedPlan' -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgentManifestRouteIncludes.*|TestSuperAgentSessionDeleteRouteDeletesOwnedSession' -count=1
```

- RED before implementation:
  - `harness/state` returned `/workspace/.plan.json` and global plan content for `conversation_id=456`.
  - `harness/plan` wrote `/workspace/.plan.json` even when the request contained `conversation_id=456`.
  - Session deletion did not remove `/workspace/.agent/sessions/123/plan.json`.
- GREEN after implementation: target tests passed.

Local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent ./api/router/coze ./domain/agent/singleagent/internal/agentflow -count=1
git diff --check -- api/model/app/developer_api/sandbox_workspace.go application/singleagent/sandbox_workspace.go api/handler/coze/super_agent_run_service.go api/handler/coze/super_agent_session_service.go application/singleagent/sandbox_workspace_test.go api/router/coze/super_agent_run_route_test.go ../docs/super-agent-8896-playwright-test-record.md
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-session-plan-state main.go
```

Results:

- Target tests passed.
- Related backend packages passed.
- Linux build passed.

Build and deployment:

- Backend binary: `/tmp/openynet-session-plan-state`
- Backend SHA256: `3429893c8b9ce825f085a7c6afa399540df248657978d25190b8d710bf522283`
- Host: `10.10.10.226:8896`
- Container: `coze-super`
- Static resources were not replaced.
- Kept one backend rollback only: `/app/openynet.bak-latest`.
- Pre-deploy `/app/openynet` SHA256: `b915108a9841a44644d21d287e3e4841492a7b5a77374fc593347c92c4359011`
- New `/app/openynet` SHA256: `3429893c8b9ce825f085a7c6afa399540df248657978d25190b8d710bf522283`
- Rollback `/app/openynet.bak-latest` SHA256: `b915108a9841a44644d21d287e3e4841492a7b5a77374fc593347c92c4359011`
- Disk after deployment: total `72G`, used `54G`, available `15G`, `79%`.

Playwright MCP verification:

- Page:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=session_plan_state_20260620`
- Runtime probe:
  - Created temporary session `7653422772256768000`.
  - `POST /api/super-agent/harness/plan` with `conversation_id` wrote `/workspace/.agent/sessions/7653422772256768000/plan.json`.
  - `POST /api/super-agent/harness/state` with the same `conversation_id` returned the session plan path and content.
  - `POST /api/super-agent/harness/snapshot` returned the same session plan path and content under `harness.plan`.
  - `POST /api/super-agent/harness/state` without `conversation_id` still returned global `/workspace/.plan.json`.
  - Manifest exposed `harness.session_plan_path_template` and `conversation_id` in the `harness.plan` request schema.
  - Deleting the temporary session removed the session plan; reading that path afterwards returned app error `100000000` with `No such file or directory`.

Evidence files:

- `super-agent-playwright-session-plan-state-20260620.json`

## 2026-06-20 - Session Plan Trace Event

Question addressed:

- `POST /api/super-agent/harness/plan` can update a session-scoped plan file, but App Server trace clients could not see that HTTP-side update as a `plan.updated` event.
- Codex-style clients need plan state in `/api/super-agent/traces/get` and `harness/snapshot` so they can replay work state without reading sandbox files manually.

Local changes:

- `buildSuperAgentTraceData` now reads harness state for the conversation.
- If the session plan exists, trace appends a synthetic `plan.updated` event with:
  - `kind = plan`
  - `status = updated`
  - `content_type = json`
  - `metadata.source = harness_state`
  - `metadata.plan_path = /workspace/.agent/sessions/<conversation_id>/plan.json`
- Existing `context.compacted` trace behavior is preserved.

TDD evidence:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run TestSuperAgentTraceRouteIncludesContextCompactionEvent -count=1
```

- RED before implementation:
  - `plan.updated` was missing from `/api/super-agent/traces/get`.
- GREEN after implementation:
  - The trace included `plan.updated` with the session plan path, `source=harness_state`, and the plan JSON content.

Local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze ./application/singleagent ./domain/agent/singleagent/internal/agentflow -count=1
git diff --check -- backend/api/handler/coze/super_agent_trace_service.go backend/api/handler/coze/super_agent_harness_service.go backend/api/router/coze/super_agent_run_route_test.go
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-plan-trace-event main.go
```

Results:

- Target test passed.
- Related backend packages passed.
- Linux build passed.

Build and deployment:

- Backend binary: `/tmp/openynet-plan-trace-event`
- Backend SHA256: `9b6be00565af2fe5116cff693c9bcc2c93a36367f9562e1a4dbb2cbb5f31cb6c`
- Host: `10.10.10.226:8896`
- Container: `coze-super`
- Static resources were not replaced.
- Kept one backend rollback only: `/app/openynet.bak-latest`.
- Pre-deploy `/app/openynet` SHA256: `3429893c8b9ce825f085a7c6afa399540df248657978d25190b8d710bf522283`
- New `/app/openynet` SHA256: `9b6be00565af2fe5116cff693c9bcc2c93a36367f9562e1a4dbb2cbb5f31cb6c`
- Rollback `/app/openynet.bak-latest` SHA256: `3429893c8b9ce825f085a7c6afa399540df248657978d25190b8d710bf522283`
- Disk after deployment: total `72G`, used `54G`, available `15G`, `79%`.

Playwright MCP verification:

- Page:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=session_plan_trace_20260620`
- Runtime probe:
  - Created temporary session `7653447893000388608`.
  - Wrote a session plan to `/workspace/.agent/sessions/7653447893000388608/plan.json`.
  - `POST /api/super-agent/harness/state` returned that session plan path and content.
  - `POST /api/super-agent/traces/get` returned a synthetic `plan.updated` event with `metadata.source = harness_state` and the session plan path.
  - `POST /api/super-agent/harness/snapshot` with `include_trace=true` returned the same `plan.updated` event under `trace.events`.
  - Deleting the temporary session removed the session plan; the follow-up harness state returned `plan.exists = false`.

Evidence files:

- `super-agent-playwright-session-plan-trace-20260620.json`

## 2026-06-20 - Snapshot Trace Synthetic Event Dedupe

Question addressed:

- `harness/snapshot` can include both `context` and `trace`.
- After trace started reading harness state directly, snapshot's fallback append path could add the same synthetic `context.compacted` event twice.
- External App Server clients need trace replay to be idempotent: one context compaction should appear as one event.

Local changes:

- Synthetic trace appenders now skip an event when the same event id is already present.
- This covers `context.compacted` and `plan.updated` synthetic events.
- Snapshot keeps its fallback behavior, but no longer duplicates events already provided by `buildSuperAgentTraceData`.

TDD evidence:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run TestSuperAgentHarnessSnapshotRouteIncludesTraceAndApprovals -count=1
```

- RED before implementation:
  - `context.compacted` appeared twice in snapshot trace events.
- GREEN after implementation:
  - `context.compacted` appears exactly once.

Local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze ./application/singleagent ./domain/agent/singleagent/internal/agentflow -count=1
git diff --check -- backend/api/handler/coze/super_agent_harness_service.go backend/api/router/coze/super_agent_run_route_test.go
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-snapshot-trace-dedupe main.go
```

Results:

- Target test passed.
- Related backend packages passed.
- Linux build passed.

Build and deployment:

- Backend binary: `/tmp/openynet-snapshot-trace-dedupe`
- Backend SHA256: `4793539df6613540e129f9608800a1d1effa90ff7cf05aee75307cc8288c7e07`
- Host: `10.10.10.226:8896`
- Container: `coze-super`
- Static resources were not replaced.
- Kept one backend rollback only: `/app/openynet.bak-latest`.
- Pre-deploy `/app/openynet` SHA256: `9b6be00565af2fe5116cff693c9bcc2c93a36367f9562e1a4dbb2cbb5f31cb6c`
- New `/app/openynet` SHA256: `4793539df6613540e129f9608800a1d1effa90ff7cf05aee75307cc8288c7e07`
- Rollback `/app/openynet.bak-latest` SHA256: `9b6be00565af2fe5116cff693c9bcc2c93a36367f9562e1a4dbb2cbb5f31cb6c`
- Disk after deployment: total `72G`, used `54G`, available `15G`, `79%`.

Playwright MCP verification:

- Page:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=snapshot_trace_dedupe_20260620`
- Runtime probe:
  - Created temporary session `7653450363101511680`.
  - Wrote `/workspace/.agent/sessions/7653450363101511680/context-summary.json`.
  - `POST /api/super-agent/harness/snapshot` with `include_context=true` and `include_trace=true` returned `context_compacted_count = 1`.
  - The returned event had `metadata.summary_path = /workspace/.agent/sessions/7653450363101511680/context-summary.json`.
  - Deleting the temporary session removed the context summary; follow-up harness state returned `summary_exists = false`.

Evidence files:

- `super-agent-playwright-snapshot-trace-dedupe-20260620.json`

## 2026-06-20 - Plan Mtime in Harness State and Trace

Question addressed:

- Session plan state had content and size, but no file modification time.
- Synthetic `plan.updated` events therefore used `created_at = 0`, which makes external App Server clients harder to sort, cache, and refresh correctly.

Local changes:

- `SuperAgentHarnessPlanState` now includes `mtime`.
- `GetSuperAgentHarnessState` reads plan file stat metadata after the plan is found.
- Synthetic `plan.updated` now includes:
  - `metadata.mtime`
  - `created_at = plan.mtime`
  - `updated_at = plan.mtime`
- If stat fails, state still returns the plan content and falls back to `mtime = 0`.

TDD evidence:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/singleagent -run TestGetSuperAgentHarnessStateDefaultsPlanToConversationScope -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run TestSuperAgentTraceRouteIncludesContextCompactionEvent -count=1
```

- RED before implementation:
  - Harness state returned `plan.mtime = 0`.
  - `plan.updated` trace event had no `metadata.mtime`, and `created_at/updated_at = 0`.
- GREEN after implementation:
  - Harness state exposes the plan file mtime.
  - `plan.updated` carries the same mtime in metadata and timestamps.

Local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze ./application/singleagent ./domain/agent/singleagent/internal/agentflow -count=1
git diff --check -- backend/api/model/app/developer_api/sandbox_workspace.go backend/application/singleagent/sandbox_workspace.go backend/application/singleagent/sandbox_workspace_test.go backend/api/handler/coze/super_agent_harness_service.go backend/api/router/coze/super_agent_run_route_test.go
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-plan-mtime-trace main.go
```

Results:

- Target tests passed.
- Related backend packages passed.
- Linux build passed.

Build and deployment:

- Backend binary: `/tmp/openynet-plan-mtime-trace`
- Backend SHA256: `1ffb8af12a3060b170a3c6358fd2d30decd1d81a9d8c9f93b0c24deb88c7e1b5`
- Host: `10.10.10.226:8896`
- Container: `coze-super`
- Static resources were not replaced.
- Kept one backend rollback only: `/app/openynet.bak-latest`.
- Pre-deploy `/app/openynet` SHA256: `4793539df6613540e129f9608800a1d1effa90ff7cf05aee75307cc8288c7e07`
- New `/app/openynet` SHA256: `1ffb8af12a3060b170a3c6358fd2d30decd1d81a9d8c9f93b0c24deb88c7e1b5`
- Rollback `/app/openynet.bak-latest` SHA256: `4793539df6613540e129f9608800a1d1effa90ff7cf05aee75307cc8288c7e07`
- Disk after deployment: total `72G`, used `54G`, available `15G`, `79%`.

Playwright MCP verification:

- Page:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=plan_mtime_trace_20260620`
- Runtime probe:
  - Created temporary session `7653453071036448768`.
  - `POST /api/super-agent/harness/plan` wrote `/workspace/.agent/sessions/7653453071036448768/plan.json`.
  - `POST /api/super-agent/harness/state` returned `plan.mtime = 1781958405`.
  - `POST /api/super-agent/traces/get` returned `plan.updated` with `metadata.mtime = 1781958405`, `created_at = 1781958405`, and `updated_at = 1781958405`.
  - Deleting the temporary session removed the session plan; follow-up harness state returned `plan.exists = false`.

Evidence files:

- `super-agent-playwright-plan-mtime-trace-20260620.json`

## 2026-06-20 - Tool Output Path in Trace Metadata

Goal:

- Let App Server clients jump from a `tool.completed` trace event to the full offloaded tool output file.
- Preserve the existing collapsed trace content while adding structured metadata fields for the offload pointer.

Local changes:

- `superagenttrace` now parses tool response text like:
  - `full 114001 bytes saved to /workspace/.agent/tooloutputs/sessions/<conversation_id>/<digest>.json`
- `tool.completed` / `tool.failed` metadata now includes:
  - `tool_output_path`
  - `tool_output_bytes`
  - `tool_output_offloaded = true`
- Existing tool metadata such as `tool_name`, `request_message_id`, `request_event_id`, `duration_ms`, and `exit_code` is preserved.

TDD evidence:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/handler/coze/superagenttrace -run TestBuildDataLinksOffloadedToolOutputPath -count=1
```

- RED before implementation:
  - `tool_output_path` was empty on the `tool.completed` trace event.
- GREEN after implementation:
  - The trace event returned `/workspace/.agent/tooloutputs/sessions/789/abcd1234ef567890.json`, `tool_output_bytes = 123456`, and `tool_output_offloaded = true`.

Local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/handler/coze/superagenttrace -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze ./api/handler/coze/superagenttrace ./application/singleagent ./domain/agent/singleagent/internal/agentflow -count=1
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-tooloutput-trace-link main.go
```

Results:

- Target test passed.
- Related backend packages passed.
- Linux build passed.
- Full backend `go test ./...` was also attempted, but still fails on existing unrelated repository issues:
  - `api/handler/coze` test import cycle.
  - `domain/memory/variables/internal/dal` `%s` formatting with `int64`.
  - stale tests in `application/base/appinfra`, `application/space/import`, and `infra/impl/modelmgr/database`.
  - `application/space/sync` SQLite conflict constraint mismatch.
  - workflow compose test panic.
  - local MySQL root access failure in `infra/impl/rdb`.

Build and deployment:

- Backend binary: `/tmp/openynet-tooloutput-trace-link`
- Backend SHA256: `0f92680916b04fac61946869d66c066b734557a938b6337d1e88a6706adda45a`
- Host: `10.10.10.226:8896`
- Container: `coze-super`
- Static resources were not replaced.
- Kept one backend rollback only: `/app/openynet.bak-latest`.
- Pre-deploy `/app/openynet` SHA256: `1ffb8af12a3060b170a3c6358fd2d30decd1d81a9d8c9f93b0c24deb88c7e1b5`
- New `/app/openynet` SHA256: `0f92680916b04fac61946869d66c066b734557a938b6337d1e88a6706adda45a`
- Rollback `/app/openynet.bak-latest` SHA256: `1ffb8af12a3060b170a3c6358fd2d30decd1d81a9d8c9f93b0c24deb88c7e1b5`
- Disk after deployment and cleanup: total `72G`, used `54G`, available `15G`, `79%`.

Playwright MCP verification:

- Page:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=plan_mtime_trace_20260620`
- Runtime probe:
  - Created temporary session `7653457917437280256`.
  - Called `POST /api/super-agent/runs/reply` with a prompt that forced `run_bash` to execute a large-output Python command.
  - The run completed with final answer `TRACE_OFFLOAD_DONE`.
  - The tool response contained an offload reference:
    - `/workspace/.agent/tooloutputs/sessions/7653457917437280256/3d7afc1f5a4a28ff.json`
    - `114001` bytes.
  - `POST /api/super-agent/traces/get` returned `tool.completed` with:
    - `metadata.tool_output_path = /workspace/.agent/tooloutputs/sessions/7653457917437280256/3d7afc1f5a4a28ff.json`
    - `metadata.tool_output_bytes = 114001`
    - `metadata.tool_output_offloaded = true`
    - `metadata.exit_code = 0`
    - `metadata.request_event_id = message:7653458206529683456:tool.started`
  - `POST /api/super-agent/harness/tool-outputs` with that path returned HTTP `200`, `code=0`, and content containing `TRACE_OFFLOAD_LINK`.
  - Deleted the temporary session through `POST /api/super-agent/sessions/delete`.
  - Follow-up `harness/tool-outputs` for `/workspace/.agent/tooloutputs/sessions/7653457917437280256` returned an empty file list.

Evidence files:

- `super-agent-playwright-tooloutput-trace-link-20260620.json`

## 2026-06-20 - Snapshot Resume Handoff

Goal:

- Make `harness.snapshot` directly useful as an App Server resume packet, instead of forcing external clients to infer handoff state from separate `context`, `plan`, `tool_outputs`, and `artifacts` blocks.
- Keep the feature scoped to the super-agent harness snapshot and avoid changing normal agent behavior.

Local changes:

- Added a `resume` object to `POST /api/super-agent/harness/snapshot`.
- Added `include_resume` as an optional snapshot request field; it defaults to enabled.
- Manifest now declares `resume` in `harness.snapshot_fields` and `include_resume` in `HarnessSnapshotRequest`.
- `resume` includes:
  - `version`
  - `conversation_id`
  - `agent_id`
  - `summary`, `summary_path`, `summary_exists`
  - `plan_path`, `plan_exists`, `plan_status`, `plan_item_count`
  - recent `message_ids`
  - `tool_output_paths`
  - `artifact_paths`
  - a composed `prompt` for external resume/handoff clients.

TDD evidence:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run TestSuperAgentHarnessSnapshotRouteIncludesResumeHandoff -count=1
```

- RED before implementation:
  - `components` did not contain `resume`.
  - `data.resume` was missing.
- GREEN after implementation:
  - `resume.version = v1`.
  - `resume.summary_path = /workspace/.agent/sessions/123/context-summary.json`.
  - `resume.plan_path = /workspace/.agent/sessions/123/plan.json`.
  - `resume.plan_status = in_progress`.
  - `resume.tool_output_paths` included `/workspace/.agent/tooloutputs/tool-call-search.json`.
  - `resume.artifact_paths` included `/outputs/session.html`.
  - `resume.prompt` included the context summary, plan path, tool output path, and artifact path.

Local verification:

```bash
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgentHarnessSnapshotRouteIncludesResumeHandoff|TestSuperAgentHarnessSnapshotRouteIncludesTraceAndApprovals|TestSuperAgentHarnessSnapshotRouteIncludesToolOutputContent|TestSuperAgentHarnessSnapshotRouteIncludesContextState|TestSuperAgentHarnessSnapshotRouteUsesConversationScopedContextState|TestSuperAgentHarnessSnapshotRouteDoesNotFallbackGlobalContextWhenConversationScopedMissing|TestSuperAgentManifestRouteReturnsAppServerContract|TestSuperAgentOpenAPIJSONRouteReturnsSchema' -count=1
SESSION_HMAC_SECRET=test-secret go test ./api/router/coze ./api/handler/coze/superagenttrace ./application/singleagent ./domain/agent/singleagent/internal/agentflow -count=1
git diff --check -- backend/api/handler/coze/super_agent_harness_service.go backend/api/handler/coze/super_agent_run_service.go backend/api/router/coze/super_agent_run_route_test.go docs/super-agent-8896-playwright-test-record.md
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-snapshot-resume-handoff main.go
```

Results:

- Target test passed.
- Snapshot, manifest, and OpenAPI targeted tests passed.
- Related backend packages passed.
- Diff check passed.
- Linux build passed.

Build and deployment:

- Backend binary: `/tmp/openynet-snapshot-resume-handoff`
- Backend SHA256: `07a9615d4b83126d88aaf1dc56fef3db9234c1f6723c898160765fcb85a5ab10`
- Host: `10.10.10.226:8896`
- Container: `coze-super`
- Static resources were not replaced.
- Kept one backend rollback only: `/app/openynet.bak-latest`.
- Pre-deploy `/app/openynet` SHA256: `0f92680916b04fac61946869d66c066b734557a938b6337d1e88a6706adda45a`
- New `/app/openynet` SHA256: `07a9615d4b83126d88aaf1dc56fef3db9234c1f6723c898160765fcb85a5ab10`
- Rollback `/app/openynet.bak-latest` SHA256: `0f92680916b04fac61946869d66c066b734557a938b6337d1e88a6706adda45a`
- Disk after deployment and cleanup: total `72G`, used `54G`, available `15G`, `79%`.

Playwright MCP verification:

- Page:
  - `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange?mcp_check=plan_mtime_trace_20260620`
- Manifest probe:
  - `GET /api/super-agent/manifest` returned HTTP `200`, `code=0`.
  - `harness.snapshot_fields` included `resume`.
  - `external_api.request_schemas["harness.snapshot"].optional` included `include_resume`.
- Runtime probe:
  - Created temporary session `7653461906736283648`.
  - Wrote:
    - `/workspace/.agent/sessions/7653461906736283648/context-summary.json`
    - `/workspace/.agent/sessions/7653461906736283648/plan.json`
    - `/workspace/.agent/tooloutputs/sessions/7653461906736283648/resume-tool.json`
  - `POST /api/super-agent/harness/snapshot` returned:
    - `components` includes `resume`.
    - `resume.summary_exists = true`.
    - `resume.plan_exists = true`.
    - `resume.plan_status = in_progress`.
    - `resume.plan_item_count = 2`.
    - `resume.tool_output_paths = [/workspace/.agent/tooloutputs/sessions/7653461906736283648/resume-tool.json]`.
    - `resume.artifact_paths = [/outputs/resume-report.md]`.
    - `resume.prompt` includes the summary, plan path, tool output path, and artifact path.
  - Deleted the temporary session.
  - Follow-up `harness.state` returned `context.summary_exists = false` and `plan.exists = false`.
  - Follow-up `harness/tool-outputs` for the session directory returned an empty file list.

Evidence files:

- `super-agent-playwright-snapshot-resume-handoff-20260620.json`

---

## 2026-06-20 — harness/resume lightweight handoff endpoint (deploy + Playwright MCP e2e)

Slice: `POST /api/super-agent/harness/resume` — a lightweight session resume handoff so App Server / Codex-like clients can continue a session without pulling the full `harness/snapshot`.

Backend deploy (226):

- New `/app/openynet` SHA-256: `dd5b704b62c2c8292fb624d21b877f7cc1a53364fcf7c8fa98f4d445eac6f075`
- Rollback `/app/openynet.bak-latest` SHA-256: `07a9615d4b83126d88aaf1dc56fef3db9234c1f6723c898160765fcb85a5ab10`
- Disk before/after: `72G / 54G used / 15G avail / 79%` (no accumulation, single rollback kept).
- Container `coze-super` restarted; manifest healthy on first poll.

Contract (settled, self-consistent):

- Handler requires `conversation_id`; `agent_id`/`bot_id` optional (sandbox-state components are skipped gracefully when no agent id is resolvable).
- `external_api.request_schemas["harness.resume"]`: `required_one_of = [["conversation_id"]]`, optional includes `agent_id, bot_id, message_limit, artifact_limit, include_*`.
- Response `data` is the flat resume object only (no nested snapshot blocks); reuses `buildSuperAgentHarnessSnapshot` with resume defaults (messages, harness, context, tool_outputs, artifacts, resume; no trace/approvals/runs/workspace).
- Manifest `harness.resume_route = POST /api/super-agent/harness/resume`, `harness.resume_fields` lists the 22 resume object fields, and `routes["harness.resume"]` + `external_api.entry_routes["harness.resume"]` are present.

Targeted Go tests (local, `SESSION_HMAC_SECRET=test-secret`, all PASS):

- `./api/router/coze` — `TestSuperAgentHarnessResumeRouteReturnsLightweightHandoff`, `TestSuperAgentHarnessResumeRouteRejectsMissingConversationID`, `TestSuperAgentManifestRouteReturnsAppServerContract`.
- Full §7 set: `./api/router/coze ./api/handler/coze/superagenttrace ./application/singleagent ./domain/agent/singleagent/internal/agentflow` all `ok`.

Playwright MCP verification (browser-context fetch, session cookie):

- Page `http://10.10.10.226:8896/space/7652614054615187456/bot/7652617174313336832/arrange` resolved to the agent name (not redirected to `/sign`).
- `GET /api/super-agent/manifest` → HTTP `200`, `code=0`, `harness.resume_route` present, lenient resume schema.
- Created temporary session `7653474995586203648`.
- `POST /api/super-agent/harness/resume` (`conversation_id` + `bot_id` + `message_limit=5`) → HTTP `200`, `code=0`:
  - `data.version = v1`, `conversation_id` echoed, `agent_id = 7652617174313336832` (derived from bot_id).
  - `components = [messages, harness, context, tool_outputs, artifacts]` (lightweight; no trace/approvals/runs/workspace).
  - `artifact_count = 4` with real `/outputs/*` paths; `prompt` is a non-empty resume handoff string.
  - Response body contains no nested `messages`/`harness`/`context`/`tool_outputs` snapshot blocks (lightweight contract confirmed).
- `POST /api/super-agent/harness/resume` without `conversation_id` → HTTP `400`, `code=400`, `msg=conversation_id is required`.
- Deleted the temporary session (`code=0`); no leftover state.

Evidence file:

- `super-agent-playwright-mcp-8896-harness-resume-lightweight-e2e-20260620.json`

---

## 2026-06-20 — closed-learning-loop + per-user DB memory (deploy + live double-session e2e)

Slice: Hermes-style closed learning loop — after a super-agent run, an async **review fork** (restricted to memory + skill tools) distills structured per-user memory. Memory was also redesigned: per-user, DB-backed (`super_agent_user_memory`), structured (kind + stable key), replacing the old flat sandbox file.

Deploy (226):

- Migration `docs/ynet-database-sql/101-super-agent-user-memory.sql` applied to `coze-mysql/openynet` (table `super_agent_user_memory` created; columns + indexes verified).
- New `/app/openynet` SHA-256: `3908993a623f59a825048e103c7d36206904081771713bd437a7133272d3cfb6`
- Rollback `/app/openynet.bak-latest` SHA-256: `dd5b704b62c2c8292fb624d21b877f7cc1a53364fcf7c8fa98f4d445eac6f075`
- Disk: `72G / 54G / 15G avail / 79%` (single rollback kept, no accumulation). Container restarted; manifest `code=0`.

Isolation (super-agent must not affect normal single-agents) — verified:

- All super-agent memory/skill/review code is gated by `isSuperAgent`; `RunPostRunReview` no-ops for non-super agents (unit test `TestRunPostRunReviewNoOpsForNonSuperAgent`).
- Review fork toolset is whitelisted to exactly `{memory_save, memory_recall, skill_manage}` — no `run_bash`/`web_*`/`deep_task` (unit test `TestSuperAgentReviewToolsetIsWhitelistedToMemoryAndSkill`).
- Full `./domain/agent/singleagent/...` + `./api/router/coze` + `./application/singleagent` suites green.

Live double-session e2e (Playwright MCP, browser-context fetch, user `7652614054610993152`):

- Baseline: `super_agent_user_memory` rows for user = 0.
- Session 1 (`7653494735113289728`): user stated durable facts (name Alex / language Go / prod server 224); run `completed` in ~5.9s.
- **Async review fork wrote 3 structured rows to `super_agent_user_memory`** (verified by direct SQL) — the run itself did not; the post-run review did:
  - `(profile, user_name)` = "用户的名字是 Alex"
  - `(preference, programming_language)` = "用户的主力编程语言是 Go"
  - `(fact, production_server)` = "用户的生产服务部署在 224 服务器"
  - correct `agent_id`, correct kind classification + stable keys.
- Session 2 (`7653494984741486592`, new conversation, same user): asked "what do you know about me?" → agent recalled name=Alex, language=Go, server=224 correctly.
  - CAVEAT: the agent recalled via its PRE-EXISTING `getKeywordMemory` tool (a shared keyword-memory feature), NOT the new `memory_recall`. The new per-user DB memory write path is verified by the direct DB write above; the two memory systems currently coexist. `getKeywordMemory` is a shared feature used by normal agents too and was intentionally left untouched.
- Cleanup: both temporary sessions deleted; the 3 test memory rows deleted (table back to 0 for the test user).

Follow-up (not blocking): reconcile the two memory surfaces for super-agents (prefer the new per-user DB `memory_recall`, or bridge `getKeywordMemory` → DB) without changing normal-agent behavior.

Evidence file:

- `super-agent-playwright-mcp-8896-closed-learning-loop-e2e-20260620.json`

---

## 2026-06-20 — context compaction quality upgrade (LLM summary, super-agent only)

Slice: replace the rule-based super-agent context compaction summary with a higher-quality Hermes-style **LLM-generated structured checkpoint**, with strict safety.

Implementation:

- `AgentRunner` gains a `chatModel` field (set in `BuildAgent`); `buildContextCompactionSummary` takes it.
- New `llmCompactionSummary`: structured "compress to a faithful reference-only checkpoint" prompt over the older turns, reusing the agent's chat model, bounded by a 25s timeout and a 120KB input budget (keeps most-recent older turns).
- Safety/fallback: LLM summary only when `chatModel != nil`; on nil/timeout/error it falls back to the existing rule-based summary. Compaction remains gated by `isSuperAgent` (line `if !r.superAgent ...`), so normal single-agents are never compacted. Summary `version` stays `v1` (contract stability).

Tests (all green):

- `TestLLMCompactionSummary` — model used; nil model => empty (fallback); erroring model => empty (fallback).
- `TestBuildContextCompactionSummaryPrefersLLMWhenAvailable` — with model the LLM summary is used; without model the rule-based summary (with old content) is used; version stays v1.
- Existing `TestPreHandlerReqAutoCompactsSuperAgentHistory` / `...ByConversation` / `TestPreHandlerReqDoesNotCompactNormalAgentHistory` still pass (rule-based fallback + normal-agent isolation preserved).

Deploy (226):

- New `/app/openynet` SHA-256: `2e5bd3614ff99c8c3b9988a2ab71d76855e16fbfd497956b7cbfe5bcb935cb76`
- Rollback `/app/openynet.bak-latest` SHA-256: `3908993a623f59a825048e103c7d36206904081771713bd437a7133272d3cfb6`
- Disk `15G avail / 79%`; single rollback kept; manifest `code=0`.
- Smoke regression (Playwright MCP): super-agent run completed in ~3.7s, `code=0`, sensible answer, temp session cleaned. No regression from the runner `chatModel` field / compaction changes.

Note: the LLM-summary path only triggers when a super-agent's history exceeds `AGENT_CONTEXT_COMPACT_MAX_BYTES` (~160KB), so it was not exercised by the short smoke run; its logic + fallback are covered by unit tests.

---

## 2026-06-21 — memory-adoption prompt + skill package forbidden-extension hardening

Two backend increments (super-agent isolated / skill-management only; normal single-agent runtime untouched).

### A. Durable-memory prompt strengthening (super-agent only)
- `SuperAgentExtraPrompt` now explicitly tells the super-agent that `memory_recall`/`memory_save` are its DURABLE per-user cross-session memory (recall first; save profile/preference/fact with a stable key), preferred over session-scoped keyword/variable memory. Test `TestSuperAgentExtraPromptPrefersDurablePerUserMemory`.
- HONEST live result: this did NOT change the main run's tool preference — the agent still used the bound `setKeywordMemory` to save. LLM-level prompting cannot override the agent's bound keyword-memory tool. The review fork (restricted to memory_save) remains the reliable populator of the new per-user DB memory, but with two systems coexisting the review sometimes judges "already saved" and skips. Proper fix = unify keyword memory → per-user DB for super-agents (gated), which touches a SHARED feature and is deferred to a supervised pass (must not affect normal agents).

### B. Skill package forbidden-extension validation
- `validateStandardSkillFiles` now rejects executable/native-binary file types (`.exe .dll .so .dylib .bat .cmd .com .scr .msi .app .jar .class .pyc .pyo .o .a .bin .deb .rpm .dmg .pkg .node`) anywhere in a standard skill package — defense-in-depth for the company skill marketplace. Applies to create/update/import/export (single chokepoint). Test `TestValidateStandardSkillFilesRejectsForbiddenExtensions`.
- Deterministic live verification (Playwright MCP, `POST /api/super-agent/skills/validate-package`): a package containing `scripts/evil.exe` → `data.validation.valid=false`, `error="forbidden executable/binary file type in skill package: scripts/evil.exe"`; a clean package (`scripts/run.py`) validates fine.

Deploy (226):
- New `/app/openynet` SHA-256: `de49b3c7bdff5a953be2b621bb24fb87826ee3d0f4dfbbcd614193449ef60687`
- Rollback `/app/openynet.bak-latest` SHA-256: `f52d0578ed132232615efac7875d59b3047a500adbb623f05aaea7fab671dd49`
- Disk 15G avail / 79%; single rollback; manifest `code=0`. Skill + super-agent test suites green.
