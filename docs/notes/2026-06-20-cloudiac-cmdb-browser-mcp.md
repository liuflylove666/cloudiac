# CloudIaC CMDB continuation and browser MCP recovery note

- Date: 2026-06-20 Asia/Singapore.
- Project: CloudIaC CMDB PRD work. Actual usable checkout path during this run was `/Volumes/scrt-sfx-923-2829osx_x64/workspace/cloudiac`; the previous `/Volumes/scrt-sfx-9.2.3-2829.osx_x64/...` mount disappeared/reappeared under the shorter name.
- Source memory: `/Users/flyer7766/.codex/memories/extensions/ad_hoc/notes/2026-06-20-cloudiac-cmdb-browser-mcp.md`.
- User constraints to preserve: do not run host install/compile tests; validate backend/frontend with Docker Compose build/up; use in-app browser for UI regression when available; preserve dirty worktree and do not revert unrelated changes.
- CMDB work added/continued: application-level upstream/downstream dependency model and API/UI. New backend concept `CmdbApplicationRelation` stores manual application relations with source `manual_application`; update API is `PUT /api/v1/cmdb/applications/relations`; frontend service method is `updateApplicationRelations`.
- UI behavior: CMDB `应用依赖` table has a `维护依赖` action, moved next to the application column for first-screen visibility. Opening it should direct the app detail drawer to the `依赖维护` tab. Asset list search was changed from the old PageSearch condition dropdown to a single keyword `InputSearch` plus DSL search, with `keywordSearch` styling.
- Docker validation completed: `docker compose build iac-portal iac-web` passed. Because Docker Desktop bind mount failed on the remounted `/Volumes/scrt-sfx-923-2829osx_x64` path, a temporary deploy root `/private/tmp/cloudiac-cmdb-compose-20260619-2348` copied from `.deploy/cloudiac` was used for successful `docker compose up -d iac-portal iac-web`; `iac-portal` was healthy and `iac-web` was up. The running web chunk contained `manageRelations`, `updateApplicationRelations`, and `keywordSearch`.
- Browser MCP issue: `browser@openai-bundled` plugin cache existed at version `26.616.32156`, but `node_repl/js` was not exposed. Root cause found in `/Users/flyer7766/.codex/config.toml`: `[features] js_repl = false`. Browser plugin depends on `node_repl/js`. The config was changed to `js_repl = true` and the new checkout path was added as a trusted project. However, after one Codex restart, Codex rewrote `js_repl` back to false once, so if browser tools are still missing, re-check that line and set it true again before starting/reloading a new thread.
- Relevant config expectations: `[plugins."browser@openai-bundled"] enabled = true`; `[mcp_servers.node_repl] command = "/Applications/Codex.app/Contents/Resources/cua_node/bin/node_repl"`; env includes `BROWSER_USE_AVAILABLE_BACKENDS = "chrome,iab"`; browser-client SHA matched `NODE_REPL_TRUSTED_BROWSER_CLIENT_SHA256S`.
- If MCP browser still fails after `js_repl = true`, the current thread probably did not hot-load tools. Start a fresh/reloaded Codex thread after confirming the config remains true, then use `tool_search` for `node_repl js` and initialize Browser with `/Users/flyer7766/.codex/plugins/cache/openai-bundled/browser/26.616.32156/scripts/browser-client.mjs`.

## 2026-06-20 follow-up validation

- `js_repl = true` was still present in `/Users/flyer7766/.codex/config.toml`, and the in-app browser connection worked.
- Local services were reachable: `http://127.0.0.1/` returned CloudIaC UI, and `http://127.0.0.1:9030/api/v1/check` returned success for build `docker-compose`, version `v1.3.5`.
- Browser UI regression passed:
  - Opened CloudIaC UI and entered organization `CMDB验证组织`.
  - Opened `/org/org-d8qk6fsd6t1s73fu2kr0/m-other-resource`.
  - Confirmed `资产 CMDB` page loaded, `资产列表` data displayed, and `应用依赖` tab was available.
  - Confirmed the `应用依赖` table displayed `维护依赖` next to each application, including `Payment Service`.
  - Clicked `Payment Service -> 维护依赖`; the drawer opened to `应用依赖详情` and defaulted to the `依赖维护` tab.
  - Confirmed the dependency maintenance form displayed upstream/downstream multi-select controls and a `保存` button.
- Browser save path passed:
  - Entered temporary manual relations through the UI and clicked `保存`.
  - Database showed three `manual_application` rows:
    - `Billing API -> Payment Service`
    - `Billing Core -> Payment Service`
    - `Payment Service -> Risk Engine`
  - Cleaned up via `PUT /api/v1/cmdb/applications/relations` with empty upstream/downstream arrays for `Payment Service`.
  - Final DB check: `iac_cmdb_application_relation` count returned `0`.
- Final runtime state after validation:
  - `iac-portal`: healthy.
  - `mysql`: healthy.
  - `consul`: healthy.
  - `ct-runner`: healthy.
  - `iac-web`: up.

## Verification commands used

```sh
curl -sS http://127.0.0.1:9030/api/v1/check
docker ps --format '{{.Names}}\t{{.Status}}' | rg 'iac-portal|iac-web|mysql|consul|ct-runner'
docker exec mysql mysql -ucloudiac -pmysqlpass cloudiac -N -e 'select count(*) from iac_cmdb_application_relation;'
```

