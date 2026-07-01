# Studio Sandbox/BashTool 226 Acceptance

Target: `http://10.10.10.226:8896`

This acceptance proves the super-agent App Server surface can:

- create a durable super-agent session
- run a sandbox Bash command
- create a workspace directory
- write and read a file
- apply a Codex-style patch
- produce, list, and download an artifact
- reject unsafe path access outside `/workspace`, `/uploads`, `/outputs`, and `/skills`
- record audited super-agent write operations into `openynet.operation_log`

## Preconditions

- 226 must pass resource gates: `/` has at least 20G available and is below 80% usage.
- Studio backend must be deployed on `coze-super`.
- No active runtime env may reference `10.10.10.220`.
- Use a temporary PAT owned by the target super-agent creator. Do not print the PAT value in logs.

Default 226 test agent:

- `SUPER_AGENT_ID=7652617174313336832`
- `SUPER_AGENT_SPACE_ID=7652614054615187456`
- owner/user: `7652614054610993152`

## Run

```bash
cd /Users/luzhipeng/projects/ynet/coze-studio/.claude/worktrees/delivery-gap-closure-20260701/e2e

BASE_URL=http://10.10.10.226:8896 \
SUPER_AGENT_API_KEY="$SUPER_AGENT_API_KEY" \
SUPER_AGENT_ID=7652617174313336832 \
SUPER_AGENT_SPACE_ID=7652614054615187456 \
pnpm exec playwright test tests/super-agent-sandbox-bash.spec.ts \
  --project=super-agent-api --reporter=html,list
```

## Operation Log Proof

After the Playwright run, query 226 MySQL:

```bash
sshpass -p root1234 ssh -o StrictHostKeyChecking=no dev@10.10.10.226 \
  'docker exec coze-mysql mysql -uroot -proot -N -e "
SELECT id, operator_id, module, action, resource_id, resource_name, path, status, created_at
FROM openynet.operation_log
WHERE module = \"super_agent\"
ORDER BY created_at DESC
LIMIT 20;"'
```

Expected:

- `operator_id` equals the PAT owner user id.
- `module=super_agent`.
- actions include `run_bash`, `write_file`, `patch_file`, and `download_artifact`.
- paths include `/api/super-agent/sandbox/exec`, `/api/super-agent/workspace/write`, `/api/super-agent/workspace/patch`, and `/api/super-agent/artifacts/download`.

## 2026-07-01 Result

Evidence directory:

`/Users/luzhipeng/projects/ynet/coze-studio/.claude/worktrees/delivery-gap-closure-20260701/reports/acceptance/20260701-studio-sandbox-226-final`

Command:

```bash
BASE_URL=http://10.10.10.226:8896 \
SUPER_AGENT_API_KEY="$(cat "$REPORT/.super-agent-api-token")" \
SUPER_AGENT_ID=7652617174313336832 \
SUPER_AGENT_SPACE_ID=7652614054615187456 \
./node_modules/.bin/playwright test tests/super-agent-sandbox-bash.spec.ts \
  --project=super-agent-api --reporter=html,list
```

Result: `1 passed (40.8s)`.

Positive routes verified with HTTP status below 500 and business `code=0`:

- `/api/super-agent/sessions/create`
- `/api/super-agent/sessions/get`
- `/api/super-agent/sessions/list`
- `/api/super-agent/sessions/rename`
- `/api/super-agent/sessions/delete`
- `/api/super-agent/sandbox/exec`
- `/api/super-agent/workspace/mkdir`
- `/api/super-agent/workspace/list`
- `/api/super-agent/workspace/write`
- `/api/super-agent/workspace/read`
- `/api/super-agent/workspace/patch`
- `/api/super-agent/workspace/edit`
- `/api/super-agent/workspace/download`
- `/api/super-agent/workspace/stat`
- `/api/super-agent/workspace/grep`
- `/api/super-agent/workspace/glob`
- `/api/super-agent/workspace/upload`
- `/api/super-agent/workspace/move`
- `/api/super-agent/workspace/delete`
- `/api/super-agent/artifacts/list`
- `/api/super-agent/artifacts/download`
- `/api/super-agent/artifacts/move`
- `/api/super-agent/artifacts/delete`
- `/api/super-agent/harness/state`
- `/api/super-agent/harness/plan`
- `/api/super-agent/harness/tool-outputs`
- `/api/super-agent/harness/snapshot`
- `/api/super-agent/harness/resume`
- `/api/super-agent/harness/context/clear`
- `/api/super-agent/harness/cleanup`

Negative security check:

- `/api/super-agent/workspace/read` with `path=/etc/passwd` returned a controlled business error, `path must be under /workspace, /uploads, /outputs, /skills`. This is expected and proves the unsafe path guard; it is not counted as a positive-route failure.

226 gates at acceptance time:

- Root disk: `72G total / 49G used / 20G available / 72%`, byte-level available space `21121396736`.
- Memory: `15GiB total`, `9.1GiB available`.
- Load average: `1.67 / 1.84 / 1.73`.
- Running container env scan: no `10.10.10.220` hit.

226 runtime proof:

- `coze-super` logs showed all positive super-agent route calls returned HTTP 200 during the passing run.
- `openynet.conversation` contained session `7657235064329076736` for agent `7652617174313336832` and creator `7652614054610993152` from the passing run.
- `openynet.operation_log` contained audited `super_agent` rows for operator `7652614054610993152`, including `run_bash`, `create_dir`, `write_file`, `patch_file`, `delete_file`, and `download_artifact`.

## Cleanup

Delete the temporary PAT row after acceptance:

```bash
sshpass -p root1234 ssh -o StrictHostKeyChecking=no dev@10.10.10.226 \
  'docker exec coze-mysql mysql -uroot -proot -e "DELETE FROM openynet.api_key WHERE name = \"codex-super-agent-e2e-20260701\";"'
```
