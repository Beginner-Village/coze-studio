# Studio 226 Acceptance Notes

## Base URLs

- Studio UI/API: `http://10.10.10.226:8896`
- Loop through Studio proxy: `http://10.10.10.226:8896/loop`
- Direct Loop: `http://10.10.10.226:8082`

Do not use `10.10.10.220` in Playwright, Postman, compose files, or scripts.

## Current Verified State

- Studio runs in standalone container `coze-super`.
- Latest accepted binary hash for the Card AOP route-audit fix:
  `cbfd1da0cb640c2762de9e8addac74eb55054f5befd11f797954317ef7d0c7eb`.
- `/api/super-agent/manifest` returns HTTP 200.
- `/aop-web/IDC10001.do` and `/aop-web/IDC10033.do` return AOP success envelopes with `header.errorCode="0"` when no real card backend is configured.
- Playwright MCP route audit on 2026-07-01 covered Workspace, Store, Templates, and 21 workspace routes with safe button clicks: `networkErrorCount=0`, `consoleErrorCount=0`.

## Playwright

Use the 226 base URL:

```bash
cd coze-studio/e2e
BASE_URL=http://10.10.10.226:8896 pnpm test
```

For manual browser acceptance, use a dedicated 226 test account and verify:

- route navigation
- Loop evaluation tabs
- operation log page
- super-agent Sandbox/BashTool flow
- Card page has no `/aop-web` 502

## Card Backend

`YNET_CARD_BACKEND_URL` controls real Card backend proxy mode. If unset, Studio uses the local AOP compatibility handler for 226 acceptance. This prevents the test environment from depending on an unavailable external `agent.finmall.com` service.
