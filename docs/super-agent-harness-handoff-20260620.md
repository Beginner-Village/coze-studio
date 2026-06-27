# Super Agent Harness / App Server / Skill Store Handoff

Date: 2026-06-20

Workspace: `/Users/luzhipeng/projects/ynet/coze-studio`

Primary test environment: `http://10.10.10.226:8896`

Test space: `7652614054615187456`

Test agent: `7652617174313336832`

## 1. Purpose

The goal is to turn the current super-agent into an internal Codex/Hermes-like online agent platform:

- A durable App Server surface for external clients, comparable to Codex app-server usage.
- A real harness layer, not only prompt text: sessions, runs, trace, plan, sandbox files, command execution, artifacts, approvals, context compaction, and resume handoff.
- Standard folder skills, not prompt-only skills: `SKILL.md` plus `scripts/`, `references/`, `templates/`, and `assets/`.
- Space-level and global company-level skill marketplace, with publish/install/export/import/validation/asset management.
- UI surfaces that can support real work: session list, workspace, preview/artifacts, trace/tool call details, skill store, and skill package management.

This work is intentionally scoped to `agent_type=super` and `/api/super-agent/*`; ordinary agents and workflows should not change behavior unless a compatibility layer explicitly calls shared code.

## 2. Current Deployment Status

Verified on 2026-06-20:

- Host reachable: `10.10.10.226`
- Container: `coze-super`, running
- Disk: root filesystem `72G`, used `54G`, available `15G`, `79%`
- Current backend in container:
  - `/app/openynet`
  - SHA-256: `07a9615d4b83126d88aaf1dc56fef3db9234c1f6723c898160765fcb85a5ab10`
  - size: `173.4M`
- Current backend rollback:
  - `/app/openynet.bak-latest`
  - SHA-256: `0f92680916b04fac61946869d66c066b734557a938b6337d1e88a6706adda45a`

Do not accumulate many backups on 226. Keep one backend rollback and one static rollback unless there is a strong reason.

Local-only delta after the last 226 deployment:

- `POST /api/super-agent/harness/resume` has been implemented locally in this workspace, with route and manifest/OpenAPI contract tests passing.
- This local delta has not yet been packaged, uploaded, or Playwright-verified on `10.10.10.226:8896`.
- Treat the next deployment as backend-only unless the next session also touches frontend/static assets.
- A Linux amd64 backend binary was built locally at `/tmp/openynet-super-agent-next`.
  - SHA-256: `dd5b704b62c2c8292fb624d21b877f7cc1a53364fcf7c8fa98f4d445eac6f075`

## 3. What Has Been Implemented

### App Server Foundation

Implemented and exposed through manifest/OpenAPI:

- `GET /api/super-agent/manifest`
- `GET /api/super-agent/openapi.json`
- Runs: create, reply, stream, cancel, get, list
- Sessions: create, list, get, rename, delete
- Messages: list
- Trace replay: `POST /api/super-agent/traces/get`
- Sandbox/workspace APIs: exec, list, read, upload, download, delete, stat, grep, glob, edit, patch, move, mkdir
- Artifacts: list, download, delete, move
- Approvals: list, resolve, decision snapshots
- Harness: state, plan, tool outputs, cleanup, context clear, snapshot
- Harness resume: `POST /api/super-agent/harness/resume` is implemented locally but not deployed to 226 yet
- Skills and marketplace: create, get, update, delete, publish, list, marketplace list/get/install, package validate/import/export, assets list/get/upsert/delete

Bearer and session-auth paths were hardened so App Server clients can call draft super-agent runs without requiring the agent to be published through the old API connector path.

### Harness State and Trace

Implemented:

- Harness state includes plan, tool outputs, runtime skills, and context policy.
- `update_plan` can write App Server-visible plan state.
- Trace projection produces tool start/completed/failed events and plan/context synthetic events.
- Long tool outputs are offloaded to files and linked back from trace metadata:
  - `tool_output_path`
  - `tool_output_bytes`
  - `tool_output_offloaded=true`
- Snapshot can include messages, runs, workspace, harness state, context, tool outputs, artifacts, trace, approvals, approval decisions, and resume.
- Snapshot deduplicates synthetic trace events.
- `harness.snapshot` now includes a `resume` object by default.

Latest resume handoff fields:

- `version`
- `conversation_id`
- `agent_id`
- `summary`, `summary_path`, `summary_exists`
- `plan_path`, `plan_exists`, `plan_status`, `plan_item_count`
- `recent_message_count`, `message_ids`
- `tool_output_root`, `tool_output_count`, `tool_output_paths`
- `artifact_root`, `artifact_count`, `artifact_paths`
- `trace_event_count`, `approval_count`, `components`
- `prompt`

Local-only explicit resume endpoint:

- Route: `POST /api/super-agent/harness/resume`
- Required contract: `conversation_id`
- Optional contract: `agent_id`, `bot_id`, `space_id`, `connector_id`, `message_limit`, `artifact_limit`, include flags, `run_id`, `trace_limit`
- Response: the `resume` object directly, not the full snapshot payload
- Test currently passing:
  - `TestSuperAgentHarnessResumeRouteReturnsLightweightHandoff`
  - `TestSuperAgentHarnessResumeRouteRejectsMissingConversationID`
  - `TestSuperAgentManifestRouteReturnsAppServerContract`
  - `TestSuperAgentOpenAPIRouteReturnsMachineReadableSchema`

### Sessions and Isolation

Implemented:

- Durable session create/list/get/rename/delete APIs.
- Session-scoped context summaries:
  - `/workspace/.agent/sessions/<conversation_id>/context-summary.json`
- Session-scoped tool outputs:
  - `/workspace/.agent/tooloutputs/sessions/<conversation_id>/...`
- Session-scoped plan:
  - `/workspace/.agent/sessions/<conversation_id>/plan.json`
- Strict isolation: if `conversation_id` is present, harness does not fall back to global context summary.
- Session delete cleans session-scoped context, plan, and tool-output state.

### Context Compaction

There is an initial automatic context compaction implementation:

- Trigger: history bytes exceed `AGENT_CONTEXT_COMPACT_MAX_BYTES`
- Default max bytes: `160KB`
- Recent tail kept: `AGENT_CONTEXT_COMPACT_RECENT_MESSAGES`, default `16`
- Summary version: `v1`
- Trigger label: `history_bytes_exceeded`
- Summary path:
  - session-scoped: `/workspace/.agent/sessions/<conversation_id>/context-summary.json`
  - legacy/global: `/workspace/.agent/context-summary.json`
- App Server visibility:
  - manifest `harness.context_policy`
  - harness state `context.policy`
  - trace event `context.compacted`
  - snapshot context block

Important limitation: summarization is still a basic/rule-based compacted summary, not yet a high-quality LLM summarizer with explicit evaluation.

### Standard Skills and Marketplace

Implemented:

- Standard package roots:
  - `SKILL.md`
  - `scripts/`
  - `references/`
  - `templates/`
  - `assets/`
- Prompt-only skill creation is rejected for standard skill flows.
- Invalid package paths are rejected.
- Skill file trees are preserved through create/update/publish/install.
- Global publish scope is supported (`scope = 3`).
- Marketplace list/get/install flow works for global and space-visible skills.
- ZIP validation and ZIP import are implemented.
- ZIP export is implemented, including published snapshot export.
- Binary assets under `assets/` can be stored as data URLs.
- Asset APIs exist for list/get/upsert/delete.
- Skill marketplace UI exists at `/explore/project/latest`.
- Generated hero asset exists at:
  - `/Users/luzhipeng/projects/ynet/coze-studio/frontend/apps/coze-studio/src/assets/skill-marketplace-hero.webp`

### Frontend Work Already Touched

Relevant UI areas exist but are not all final:

- Super-mode workspace and trace UI:
  - `/Users/luzhipeng/projects/ynet/coze-studio/frontend/packages/agent-ide/entry/src/modes/super-mode`
- Codex-style trace panel:
  - `/Users/luzhipeng/projects/ynet/coze-studio/frontend/packages/agent-ide/entry/src/modes/super-mode/codex-trace`
- Session sidebar:
  - `/Users/luzhipeng/projects/ynet/coze-studio/frontend/packages/agent-ide/entry/src/modes/super-mode/super-session-sidebar.tsx`
- Skill center:
  - `/Users/luzhipeng/projects/ynet/coze-studio/frontend/apps/coze-studio/src/pages/space-skill`
- Global skill marketplace:
  - `/Users/luzhipeng/projects/ynet/coze-studio/frontend/apps/coze-studio/src/pages/skill-marketplace`

The chat/arrange UI has been under heavy iteration and user feedback was negative for several layout changes. Be careful before redeploying static assets. The latest backend deployments did not replace static resources.

## 4. What Still Needs Work

### Harness / Runtime

- Package, upload, and Playwright MCP verify the local `POST /api/super-agent/harness/resume` implementation on 226.
- Replace basic context compaction with higher-quality summarization and tests for summary quality.
- Add retention/quota policies for:
  - `/workspace/.agent/sessions/*`
  - `/workspace/.agent/tooloutputs/*`
  - `/outputs/*`
  - approvals and trace artifacts
- Add complete lifecycle cleanup beyond session deletion where needed.
- Make approval and policy boundaries clearer for dangerous commands, network calls, publishing, and agent-created skills.
- Align App Server contract with any future MCP server shape if this should be callable as an MCP server.

### Skills

- Review standard skill schema/frontmatter and make it stable for company use.
- Add stronger skill package validation:
  - max file count/size
  - executable script policy
  - binary asset limits
  - forbidden extensions
  - category/tag normalization
- Add review workflow for global publication.
- Add official/company curated state, owner/team metadata, version pinning UX, and rollback/install history.
- Ensure agent-managed skill edits are visible, diffable, and auditable.

### Frontend

- The user explicitly disliked several recent chat layout attempts. Do not continue from those visual choices blindly.
- Right chat composer must sit at the bottom while leaving chat history above it.
- After send, an assistant placeholder should appear immediately with a typing animation before streamed content arrives.
- Tool calls need a readable collapsed summary and expandable details:
  - tool name
  - arguments
  - return/result
  - offloaded output path when applicable
- The workspace/preview layout needs a stable, usable split; file preview must remain readable.
- Skill marketplace should follow the provided design files more closely.

User-provided design references:

- `/Users/luzhipeng/Downloads/技能商店 重设计.html`
- `/Users/luzhipeng/Downloads/技能商店 重设计 单文件.html`

### Testing and Release Hygiene

- The worktree is very dirty and contains many untracked files. Do not revert unrelated changes.
- Full `go test ./...` has known unrelated failures/import cycles in this repo. Use targeted packages and record the known full-suite limitation.
- Always record 226 deployment hashes and Playwright MCP evidence after replacing backend or static assets.
- Always check 226 disk before and after upload.

## 5. Key Code Paths

Backend:

- `/Users/luzhipeng/projects/ynet/coze-studio/backend/api/router/coze/api.go`
- `/Users/luzhipeng/projects/ynet/coze-studio/backend/api/router/coze/super_agent_run_route_test.go`
- `/Users/luzhipeng/projects/ynet/coze-studio/backend/api/handler/coze/super_agent_run_service.go`
- `/Users/luzhipeng/projects/ynet/coze-studio/backend/api/handler/coze/super_agent_harness_service.go`
- `/Users/luzhipeng/projects/ynet/coze-studio/backend/api/handler/coze/super_agent_session_service.go`
- `/Users/luzhipeng/projects/ynet/coze-studio/backend/api/handler/coze/super_agent_trace_service.go`
- `/Users/luzhipeng/projects/ynet/coze-studio/backend/api/handler/coze/super_agent_workspace_service.go`
- `/Users/luzhipeng/projects/ynet/coze-studio/backend/api/handler/coze/super_agent_artifact_service.go`
- `/Users/luzhipeng/projects/ynet/coze-studio/backend/api/handler/coze/super_agent_approval_service.go`
- `/Users/luzhipeng/projects/ynet/coze-studio/backend/api/handler/coze/superagenttrace/projection.go`
- `/Users/luzhipeng/projects/ynet/coze-studio/backend/application/singleagent/sandbox_workspace.go`
- `/Users/luzhipeng/projects/ynet/coze-studio/backend/application/singleagent/sandbox_skill.go`
- `/Users/luzhipeng/projects/ynet/coze-studio/backend/application/singleagent/sandbox_artifact.go`
- `/Users/luzhipeng/projects/ynet/coze-studio/backend/application/skill/skill_application.go`
- `/Users/luzhipeng/projects/ynet/coze-studio/backend/domain/agent/singleagent/internal/agentflow/super_agent.go`
- `/Users/luzhipeng/projects/ynet/coze-studio/backend/domain/agent/singleagent/internal/agentflow/agent_flow_runner.go`
- `/Users/luzhipeng/projects/ynet/coze-studio/backend/domain/agent/singleagent/internal/agentflow/node_tool_sandbox.go`
- `/Users/luzhipeng/projects/ynet/coze-studio/backend/domain/agent/singleagent/internal/agentflow/node_tool_skillmanage.go`

Frontend:

- `/Users/luzhipeng/projects/ynet/coze-studio/frontend/packages/agent-ide/entry/src/modes/super-mode/index.tsx`
- `/Users/luzhipeng/projects/ynet/coze-studio/frontend/packages/agent-ide/entry/src/modes/super-mode/super-chat-area.tsx`
- `/Users/luzhipeng/projects/ynet/coze-studio/frontend/packages/agent-ide/entry/src/modes/super-mode/sandbox-workspace.tsx`
- `/Users/luzhipeng/projects/ynet/coze-studio/frontend/packages/agent-ide/entry/src/modes/super-mode/codex-trace/codex-trace-panel.tsx`
- `/Users/luzhipeng/projects/ynet/coze-studio/frontend/packages/agent-ide/entry/src/modes/super-mode/codex-trace/trace-bridge.tsx`
- `/Users/luzhipeng/projects/ynet/coze-studio/frontend/apps/coze-studio/src/pages/skill-marketplace/index.tsx`
- `/Users/luzhipeng/projects/ynet/coze-studio/frontend/apps/coze-studio/src/pages/space-skill/index.tsx`
- `/Users/luzhipeng/projects/ynet/coze-studio/frontend/apps/coze-studio/src/pages/space-skill/hooks/use-skill-management.ts`

Database migration docs:

- `/Users/luzhipeng/projects/ynet/coze-studio/docs/ynet-database-sql/99-skill-version.sql`
- `/Users/luzhipeng/projects/ynet/coze-studio/docs/ynet-database-sql/100-skill-publish-marketplace.sql`

## 6. Historical Docs and Evidence

Primary record:

- `/Users/luzhipeng/projects/ynet/coze-studio/docs/super-agent-8896-playwright-test-record.md`

Design and plans:

- `/Users/luzhipeng/projects/ynet/coze-studio/docs/superpowers/specs/2026-06-17-super-agent-harness-design.md`
- `/Users/luzhipeng/projects/ynet/coze-studio/docs/superpowers/specs/2026-06-19-super-agent-codex-app-server-skill-marketplace-design.md`
- `/Users/luzhipeng/projects/ynet/coze-studio/docs/superpowers/plans/2026-06-19-super-agent-codex-app-server-foundation.md`
- `/Users/luzhipeng/projects/ynet/coze-studio/docs/superpowers/plans/2026-06-19-standard-skill-zip-validation.md`
- `/Users/luzhipeng/projects/ynet/coze-studio/docs/superpowers/plans/2026-06-19-standard-skill-zip-export.md`

Selected runtime evidence files:

- `/Users/luzhipeng/projects/ynet/coze-studio/super-agent-playwright-snapshot-resume-handoff-20260620.json`
- `/Users/luzhipeng/projects/ynet/coze-studio/super-agent-playwright-tooloutput-trace-link-20260620.json`
- `/Users/luzhipeng/projects/ynet/coze-studio/super-agent-playwright-plan-mtime-trace-20260620.json`
- `/Users/luzhipeng/projects/ynet/coze-studio/super-agent-playwright-session-plan-state-20260620.json`
- `/Users/luzhipeng/projects/ynet/coze-studio/super-agent-playwright-session-context-summary-final-20260620.json`
- `/Users/luzhipeng/projects/ynet/coze-studio/super-agent-playwright-session-context-strict-final-20260620.json`
- `/Users/luzhipeng/projects/ynet/coze-studio/super-agent-playwright-session-tooloutputs-state-pass-20260620.json`
- `/Users/luzhipeng/projects/ynet/coze-studio/super-agent-playwright-session-tooloutput-cleanup-pass-20260620.json`
- `/Users/luzhipeng/projects/ynet/coze-studio/super-agent-playwright-harness-context-clear-final-20260620.json`
- `/Users/luzhipeng/projects/ynet/coze-studio/super-agent-playwright-harness-context-policy-20260620.json`
- `/Users/luzhipeng/projects/ynet/coze-studio/super-agent-playwright-snapshot-trace-dedupe-20260620.json`

Selected skill and marketplace evidence:

- `/Users/luzhipeng/projects/ynet/coze-studio/super-agent-playwright-mcp-8896-skill-marketplace-e2e.json`
- `/Users/luzhipeng/projects/ynet/coze-studio/super-agent-playwright-mcp-8896-standard-skill-enforcement.json`
- `/Users/luzhipeng/projects/ynet/coze-studio/super-agent-playwright-mcp-8896-skill-zip-after-fix-final.json`
- `/Users/luzhipeng/projects/ynet/coze-studio/super-agent-playwright-mcp-8896-skill-zip-reject-after-fix-final.json`
- `/Users/luzhipeng/projects/ynet/coze-studio/super-agent-playwright-mcp-8896-skill-export-final.json`
- `/Users/luzhipeng/projects/ynet/coze-studio/super-agent-playwright-mcp-8896-skill-assets-final.json`
- `/Users/luzhipeng/projects/ynet/coze-studio/super-agent-playwright-mcp-8896-skill-asset-delete-final.json`
- `/Users/luzhipeng/projects/ynet/coze-studio/super-agent-playwright-mcp-8896-skill-assets-list-get-final.json`
- `/Users/luzhipeng/projects/ynet/coze-studio/super-agent-playwright-mcp-8896-skill-marketplace-padding-final.json`
- `/Users/luzhipeng/projects/ynet/coze-studio/super-agent-playwright-mcp-8896-skill-marketplace-padding-1280-final.json`

## 7. Local Test Commands

Use targeted tests first:

```bash
cd /Users/luzhipeng/projects/ynet/coze-studio/backend

SESSION_HMAC_SECRET=test-secret go test ./api/router/coze ./api/handler/coze/superagenttrace ./application/singleagent ./domain/agent/singleagent/internal/agentflow -count=1

SESSION_HMAC_SECRET=test-secret go test ./application/skill ./api/router/skill ./api/handler/skill ./domain/skill/... -count=1

SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgentHarnessResumeRouteReturnsLightweightHandoff|TestSuperAgentHarnessResumeRouteRejectsMissingConversationID|TestSuperAgentManifestRouteReturnsAppServerContract|TestSuperAgentOpenAPIRouteReturnsMachineReadableSchema' -count=1
```

Frontend checks:

```bash
cd /Users/luzhipeng/projects/ynet/coze-studio

pnpm --dir frontend/packages/arch/bot-api exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/packages/agent-ide/entry exec tsc --noEmit -p tsconfig.json
pnpm --dir frontend/apps/coze-studio exec tsc --noEmit -p tsconfig.json

pnpm --dir frontend/packages/agent-ide/entry test -- --run src/modes/super-mode/__tests__/super-chat-area.test.tsx src/modes/super-mode/codex-trace/__tests__/codex-trace-panel.test.tsx
pnpm --dir frontend/apps/coze-studio test -- --run src/pages/space-skill/__tests__/index.test.tsx src/pages/skill-marketplace/__tests__/index.test.tsx
```

Linux backend build:

```bash
cd /Users/luzhipeng/projects/ynet/coze-studio/backend

GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-s -w' -o /tmp/openynet-super-agent-next main.go
shasum -a 256 /tmp/openynet-super-agent-next
```

Frontend production build:

```bash
cd /Users/luzhipeng/projects/ynet/coze-studio

pnpm --dir frontend/apps/coze-studio build
tar -C frontend/apps/coze-studio/dist -czf /tmp/coze-studio-static-next.tar.gz .
shasum -a 256 /tmp/coze-studio-static-next.tar.gz
```

Before claiming success:

```bash
cd /Users/luzhipeng/projects/ynet/coze-studio

git diff --check -- <files-you-touched>
```

## 8. 226 Deploy Procedure

Always check disk first:

```bash
ssh dev@10.10.10.226 'df -h /; docker ps --format "{{.Names}} {{.Status}}" | grep coze-super || true'
```

Backend deployment:

```bash
scp /tmp/openynet-super-agent-next dev@10.10.10.226:/home/dev/openynet-super-agent-next

ssh dev@10.10.10.226 '
  set -e
  docker exec coze-super sh -lc "cp /app/openynet /app/openynet.bak-latest"
  docker cp /home/dev/openynet-super-agent-next coze-super:/app/openynet
  docker exec coze-super sh -lc "chmod +x /app/openynet && sha256sum /app/openynet /app/openynet.bak-latest"
  docker restart coze-super
  rm -f /home/dev/openynet-super-agent-next
  df -h /
'
```

Static deployment:

```bash
scp /tmp/coze-studio-static-next.tar.gz dev@10.10.10.226:/home/dev/coze-studio-static-next.tar.gz

ssh dev@10.10.10.226 '
  set -e
  docker exec coze-super sh -lc "rm -rf /app/resources/static.bak-latest && mv /app/resources/static /app/resources/static.bak-latest && mkdir -p /app/resources/static"
  docker cp /home/dev/coze-studio-static-next.tar.gz coze-super:/tmp/coze-studio-static-next.tar.gz
  docker exec coze-super sh -lc "tar -xzf /tmp/coze-studio-static-next.tar.gz -C /app/resources/static && rm -f /tmp/coze-studio-static-next.tar.gz"
  docker restart coze-super
  rm -f /home/dev/coze-studio-static-next.tar.gz
  df -h /
'
```

After deploy:

```bash
curl -sS http://10.10.10.226:8896/api/super-agent/manifest | jq '.code, .data.protocol_version, .data.harness.snapshot_fields'
curl -I http://10.10.10.226:8896/
```

## 9. Playwright MCP Verification Checklist

Minimum backend smoke:

- Open `/space/7652614054615187456/bot/7652617174313336832/arrange`.
- Confirm not redirected to `/sign`.
- Confirm `GET /api/super-agent/manifest` returns `code=0`.
- Confirm manifest includes the routes or schemas changed in the slice.
- Create a temporary session through `/api/super-agent/sessions/create`.
- Exercise the new endpoint using browser-context `fetch`.
- Clean temporary sandbox files and delete the temporary session.
- Save the result JSON under the workspace root with a descriptive name.
- Append the result to `/Users/luzhipeng/projects/ynet/coze-studio/docs/super-agent-8896-playwright-test-record.md`.

Recommended harness probes:

- `POST /api/super-agent/harness/state`
- `POST /api/super-agent/harness/resume`
- `POST /api/super-agent/harness/snapshot`
- `POST /api/super-agent/traces/get`
- `POST /api/super-agent/harness/context/clear`
- `POST /api/super-agent/harness/tool-outputs`
- `POST /api/super-agent/harness/cleanup`

Recommended skill probes:

- `POST /api/super-agent/skills/validate-package`
- `POST /api/super-agent/skills/import-package`
- `POST /api/super-agent/skills/export`
- `GET /api/super-agent/marketplace/list?scope=3&page=1&page_size=200`
- `GET /api/super-agent/marketplace/get`
- `POST /api/super-agent/marketplace/install`
- `GET /api/super-agent/skills/assets/list`
- `GET /api/super-agent/skills/assets/get`
- `POST /api/super-agent/skills/assets/upsert`
- `POST /api/super-agent/skills/assets/delete`

## 10. Suggested Next Slice

Recommended next backend slice:

Package, upload, and verify the local `POST /api/super-agent/harness/resume` implementation.

Reason:

- The route is now implemented locally and covered by route/manifest tests, but 226 still runs the previous backend hash.
- It should be verified through browser-context Playwright MCP fetch before another feature slice builds on it.

Expected behavior to verify on 226:

- `GET /api/super-agent/manifest` exposes `routes["harness.resume"]`.
- `external_api.entry_routes["harness.resume"]` is `POST /api/super-agent/harness/resume`.
- `GET /api/super-agent/openapi.json` exposes operation `harness.resume`.
- Calling `POST /api/super-agent/harness/resume` with a real `conversation_id` returns the lightweight resume object directly.
- Response must not include full snapshot blocks such as top-level `messages`, `harness`, `context`, or `tool_outputs`.

Suggested local test before build:

```bash
cd /Users/luzhipeng/projects/ynet/coze-studio/backend

SESSION_HMAC_SECRET=test-secret go test ./api/router/coze -run 'TestSuperAgentHarnessResumeRouteReturnsLightweightHandoff|TestSuperAgentHarnessResumeRouteRejectsMissingConversationID|TestSuperAgentManifestRouteReturnsAppServerContract|TestSuperAgentOpenAPIRouteReturnsMachineReadableSchema' -count=1
```

## 11. Handoff Prompt For The Next Agent

Use this when handing off to another Codex/Claude Code session:

```text
Read /Users/luzhipeng/projects/ynet/coze-studio/docs/super-agent-harness-handoff-20260620.md first.
Then read /Users/luzhipeng/projects/ynet/coze-studio/docs/super-agent-8896-playwright-test-record.md only for the specific area you touch.

Continue the super-agent Codex/Hermes-like App Server work. Do not revert unrelated dirty worktree changes. The local workspace already has POST /api/super-agent/harness/resume implemented and its route/manifest tests passing, but it has not yet been deployed to 10.10.10.226:8896. Next: run the local targeted Go tests plus Linux build, deploy backend carefully keeping only one rollback, verify manifest/openapi/resume with Playwright MCP, and record evidence and hashes back into docs/super-agent-8896-playwright-test-record.md.
```
