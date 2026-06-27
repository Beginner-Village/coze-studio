# Standard Skill ZIP Export Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let users and App Server clients download any permitted standard skill as a reusable ZIP package.

**Architecture:** Backend owns package generation so exported packages match server validation and permission rules. The App Server endpoint returns a JSON data URL for UI download and external integrations; owned skills export the current draft, marketplace-visible skills export the published snapshot.

**Tech Stack:** Go/Hertz backend, existing skill application service, React skill center, existing API schema and DeveloperApi wrappers, Vitest and Go tests.

---

### Task 1: Backend ZIP Export Service

**Files:**
- Modify: `backend/application/skill/skill_application.go`
- Test: `backend/application/skill/skill_application_test.go`

- [ ] **Step 1: Write failing tests**

Add tests that create a standard skill with `SKILL.md`, `scripts/run.sh`, and `assets/logo.png`, call `ExportSkillPackage`, decode the returned ZIP, and assert the ZIP contains `skill-name/SKILL.md`, `skill-name/scripts/run.sh`, and decoded binary `skill-name/assets/logo.png`. Add a marketplace test that publishes a skill, updates the draft, then exports from another space and asserts the exported ZIP uses the published snapshot.

- [ ] **Step 2: Verify RED**

Run:

```bash
env SESSION_HMAC_SECRET=test-secret go test ./application/skill -run 'TestSkillApplicationServiceExportsStandardSkillZipPackage|TestSkillApplicationServiceExportsPublishedMarketplaceSnapshot' -count=1
```

Expected: fails because `ExportSkillPackage` is undefined.

- [ ] **Step 3: Implement minimal service**

Add `SkillPackageExport` with `Filename`, `Content`, `Size`, and `FilePaths`. Resolve the skill by permission, build a deterministic ZIP with a sanitized root folder, decode base64 data URLs back to bytes, and return `data:application/zip;base64,...`.

- [ ] **Step 4: Verify GREEN**

Run the same Go command and expect both tests to pass.

### Task 2: App Server Route, Manifest, and SDK

**Files:**
- Modify: `backend/api/handler/skill/skill_service.go`
- Modify: `backend/api/router/coze/api.go`
- Modify: `backend/api/handler/coze/super_agent_run_service.go`
- Test: `backend/api/router/coze/super_agent_run_route_test.go`
- Modify: `frontend/packages/arch/api-schema/src/idl/skill/skill.ts`
- Test: `frontend/packages/arch/api-schema/__tests__/skill-super-agent.test.ts`
- Modify: `frontend/packages/arch/idl/src/auto-generated/developer_api/index.ts`
- Modify: `frontend/packages/arch/idl/src/auto-generated/developer_api/namespaces/developer_api.ts`
- Test: `frontend/packages/arch/bot-api/__tests__/developer-api-super-agent-skills.test.ts`

- [ ] **Step 1: Write failing contract tests**

Assert manifest exposes `POST /api/super-agent/skills/export`, required fields `space_id` and `skill_id`, `SuperAgentExportSkillPackage` schema metadata, and DeveloperApi POST body mapping.

- [ ] **Step 2: Verify RED**

Run targeted Go/Vitest contract tests and expect missing-route/missing-function failures.

- [ ] **Step 3: Implement route and SDK wrappers**

Add handler `ExportSkillPackage`, route `_skills.POST("/export", ...)`, manifest route/schema, api-schema interfaces/API, and DeveloperApi method/interfaces.

- [ ] **Step 4: Verify GREEN**

Run the targeted Go/Vitest contract tests and expect them to pass.

### Task 3: Skill Center Download Action

**Files:**
- Modify: `frontend/apps/coze-studio/src/pages/space-skill/hooks/use-skill-management.ts`
- Modify: `frontend/apps/coze-studio/src/pages/space-skill/index.tsx`
- Test: `frontend/apps/coze-studio/src/pages/space-skill/__tests__/index.test.tsx`

- [ ] **Step 1: Write failing UI test**

Assert marketplace cards show `下载 ZIP` together with `安装`, proving users can export reusable standard packages directly from the skill center.

- [ ] **Step 2: Verify RED**

Run:

```bash
pnpm --dir frontend/apps/coze-studio test -- --run src/pages/space-skill/__tests__/index.test.tsx
```

Expected: fails because `下载 ZIP` is absent.

- [ ] **Step 3: Implement UI**

Add `exportSkillPackage(skillId)` to the hook, a small browser download helper in the page, and card actions for own and marketplace skills.

- [ ] **Step 4: Verify GREEN**

Run the same Vitest command and expect the skill center tests to pass.

### Task 4: Build, Deploy, and Playwright Verify

**Files:**
- Modify: `docs/super-agent-8896-playwright-test-record.md`

- [ ] **Step 1: Run full local verification**

Run Go tests, frontend tests, typecheck, and production build.

- [ ] **Step 2: Deploy to 8896**

Build Linux backend, package frontend static assets, SHA-check uploads, back up `/app/openynet` and `/app/resources/static`, replace, and restart `coze-super`.

- [ ] **Step 3: Playwright MCP verify**

On `http://10.10.10.226:8896/space/7652614054615187456/skills`, create/import a temporary skill, call `POST /api/super-agent/skills/export`, decode the returned ZIP shape, delete the temporary skill, and save console/network/result files.

- [ ] **Step 4: Record evidence**

Append hashes, backup paths, commands, and Playwright files to `docs/super-agent-8896-playwright-test-record.md`.
