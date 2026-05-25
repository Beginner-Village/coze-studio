# Task 13 verification: ES resync UI sub-route

Wires `DataMaintenanceSection` into the canonical `/space/:space_id/<sub>`
pattern so users can reach it from the SpaceLayout sub-menu, alongside
EMBEDDING / RERANK / EXPORT_IMPORT / MEMBERS.

## What changed

- New page: `frontend/apps/coze-studio/src/pages/space-data-maintenance.tsx`
  - Wraps `DataMaintenanceSection` with a `Layout` shell + i18n title
  - Pulls `space_id` from `useParams<{ space_id: string }>()`
- New enum value: `SpaceSubModuleEnum.DATA_MAINTENANCE = 'data-maintenance'`
  in `frontend/packages/foundation/space-ui-adapter/src/const.ts`
- New sub-menu item in
  `frontend/packages/foundation/space-ui-adapter/src/components/workspace-sub-menu/index.tsx`
  (placed between EXPORT_IMPORT and OBSERVABILITY in the "Manage" section,
  icon `IconCozAnalytics`, label "数据维护",
  `dataTestId='navigation_workspace_data_maintenance'`)
- Route registration in `frontend/apps/coze-studio/src/routes.tsx`
  (the active router file; `routes/index.tsx` is dead code with `routes.tsx`
  shadowing it) — path `data-maintenance`, lazy import, loader sets
  `subMenuKey: SpaceSubModuleEnum.DATA_MAINTENANCE`
- Mirror change in `routes/index.tsx` for future parity in case the wiring
  flips

## Build-time gotcha discovered & fixed

- `src/routes/index.tsx` is **NOT used**. `src/routes.tsx` (the flat file
  with the same name) shadows the directory import per Node/ESM resolution
  rules. App `app.tsx` does `import { router } from './routes'` which
  resolves to `routes.tsx`. All edits to `routes/index.tsx` were dead until
  this task moved the edit to `routes.tsx`.
- Flat `pages/space-data-maintenance.tsx` matches existing pattern
  (`space-export-import.tsx`, `space-model-config.tsx` etc.)
- Task 11's `space-management/index.tsx` modifications (the select-then-render
  pattern) are also dead code: `space-management` is wired only via the dead
  `routes/index.tsx`, not the live `routes.tsx`. Kept as-is for future
  reactivation.

## Verified on dev 220 (http://10.10.10.220:9888)

- [PASS] `npx tsc --noEmit` — 0 errors
- [PASS] `npx vitest run DataMaintenanceSection.test.tsx` — 4/4 pass
- [PASS] `rush build --to @coze-studio/app` — 18.5s success
- [PASS] Dist deployed to `ynet-web` nginx
  - `space_data_maintenance_missing_id` literal present in
    `async/2156.544cb9c9.js`
  - `"data-maintenance"` literal present 2× in `index~0` (enum + route path)
- [PASS] Browser navigate to `/space/7639215472289775616/data-maintenance`
  - SpaceLayout renders with full sidebar
  - "数据维护" sub-menu item visible between "导出/导入" and "可观测性"
  - Page renders heading "数据维护" + DataMaintenanceSection
  - "重新同步" button visible with red color
- [PASS] Click "重新同步" → confirm modal appears
  - Title: "确认重新同步 ES 索引？"
  - Body: "将清空本空间所有 ES 索引并从 MySQL 重写..."
  - Buttons: 取消 / 确认
- [PASS] Click "确认" → `POST /api/space/resync_es` returns
  `{"code":0,"msg":"success","counts":{"project_draft":4,"coze_resource":0,"kb_entries":4,"slice_reindex_jobs":439}}`
- Screenshot: `task13-data-maintenance-page.png`

## Known limitations

- Sub-menu visibility is unconditional (no owner-only filter at UI layer).
  Permission enforcement is server-side via `space_app.go` owner check.
  Matches EMBEDDING / RERANK / EXPORT_IMPORT pattern.
