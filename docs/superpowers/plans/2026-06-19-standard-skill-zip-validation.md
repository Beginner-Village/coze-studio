# Standard Skill ZIP Validation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let users and App Server clients validate standard skill ZIP packages before importing them.

**Architecture:** Backend reuses the same ZIP decoding and standard folder validation path as import, returning structured validation details for valid packages and a normal validation response for invalid packages. Frontend skill center calls validation before import so users see exact standard-package errors early.

**Tech Stack:** Go/Hertz backend, existing skill application service, React skill center, existing api-schema and DeveloperApi wrappers, Go tests and Vitest.

---

### Task 1: Backend Validation Service

**Files:**
- Modify: `backend/application/skill/skill_application.go`
- Test: `backend/application/skill/skill_application_test.go`

- [ ] **Step 1: Write failing tests**

Add a test that calls `ValidateSkillPackage` with a ZIP containing `SKILL.md`, `scripts/run.py`, `references/guide.md`, and `assets/logo.png`, then asserts the result is valid, includes parsed metadata, sorted file paths, file count, and asset image paths. Add a second test with a ZIP missing `SKILL.md` and assert the result is invalid with a useful error message.

- [ ] **Step 2: Verify RED**

Run:

```bash
SESSION_HMAC_SECRET=test-secret go test ./application/skill -run 'TestSkillApplicationServiceValidatesStandardSkillZipPackage|TestSkillApplicationServiceValidationReportsInvalidSkillZipPackage' -count=1
```

Expected: fails because `ValidateSkillPackage` is undefined.

- [ ] **Step 3: Implement minimal service**

Add `SkillPackageValidation` and `ValidateSkillPackage`. Reuse `decodeSkillPackageContent`, `extractStandardSkillPackage`, `parseSkillFrontmatter`, and asset path helpers so validation and import cannot drift.

- [ ] **Step 4: Verify GREEN**

Run the same Go command and expect both tests to pass.

### Task 2: App Server Contract and SDK

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

Assert manifest exposes `POST /api/super-agent/skills/validate-package` with required `content`, optional `filename`; assert api-schema and DeveloperApi expose `SuperAgentValidateSkillPackage`.

- [ ] **Step 2: Verify RED**

Run targeted Go/Vitest tests and expect missing route/function failures.

- [ ] **Step 3: Implement route and wrappers**

Add handler `ValidateSkillPackage`, route `_skills.POST("/validate-package", ...)`, manifest route/schema, api-schema interfaces/API, and DeveloperApi method/interfaces.

- [ ] **Step 4: Verify GREEN**

Run targeted Go/Vitest tests and expect pass.

### Task 3: Skill Center Upload Preflight

**Files:**
- Modify: `frontend/apps/coze-studio/src/pages/space-skill/hooks/use-skill-management.ts`
- Modify: `frontend/apps/coze-studio/src/pages/space-skill/index.tsx`
- Test: `frontend/apps/coze-studio/src/pages/space-skill/__tests__/index.test.tsx`

- [ ] **Step 1: Write failing UI test**

Assert choosing a `.zip` calls `validateSkillPackage` before `importSkillPackage`; if validation returns invalid, assert import is not called and the page shows the validation message.

- [ ] **Step 2: Verify RED**

Run:

```bash
pnpm --dir frontend/apps/coze-studio test -- --run src/pages/space-skill/__tests__/index.test.tsx
```

Expected: fails because upload does not validate before import.

- [ ] **Step 3: Implement preflight**

Add `validateSkillPackage` to the hook and call it in `handleZipInputChange` before import. If validation is invalid, show its message and stop.

- [ ] **Step 4: Verify GREEN**

Run the same Vitest command and expect pass.

### Task 4: Verify, Deploy, and Record

**Files:**
- Modify: `docs/super-agent-8896-playwright-test-record.md`

- [ ] **Step 1: Run local verification**

Run backend tests, frontend tests, typecheck, and production build.

- [ ] **Step 2: Deploy to 8896**

Build Linux backend, package frontend static assets, SHA-check uploads, back up current container files, replace, and restart `coze-super`.

- [ ] **Step 3: Playwright MCP verify**

Open the skill center on `10.10.10.226:8896`, call `POST /api/super-agent/skills/validate-package` from the browser context with both valid and invalid ZIP packages, and save result/network files.

- [ ] **Step 4: Record evidence**

Append hashes, backup paths, commands, and Playwright files to `docs/super-agent-8896-playwright-test-record.md`.
