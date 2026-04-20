# Cocos Creator Workflow

Use this document when an assigned task depends on Cocos Creator editor behavior, engine settings, or editor-driven asset and scene changes.

## Project MCP Server Facts

Current project-local Cocos MCP extension facts:

- extension root: `client/my-immortal-sect/extensions/cocos-mcp-server/`
- extension entry: `client/my-immortal-sect/extensions/cocos-mcp-server/source/main.ts`
- compiled entry: `client/my-immortal-sect/extensions/cocos-mcp-server/dist/main.js`
- runtime settings file: `client/my-immortal-sect/settings/mcp-server.json`
- default HTTP port: `9527`
- health endpoint: `http://127.0.0.1:9527/health`
- MCP endpoint: `http://127.0.0.1:9527/mcp`

Important behavior:

- This server is **not** a standalone background daemon.
- It is started by the Cocos Creator extension host inside the editor.
- When the editor is closed, the MCP server should be treated as unavailable.

## Default Assumptions

- Confirm the real Cocos Creator version and target platform from project docs before making editor-specific decisions.
- Prefer built-in editor workflows, built-in components, and project settings over custom runtime workarounds when the engine already provides the required control path.
- Prefer documented editor workflows over runtime-code workarounds when the engine already exposes the required control path.
- If a task depends on editor state you cannot verify, escalate to the human or supervisor instead of guessing.

## Reference Order

Use this reference order before you touch scenes, components, project settings, extensions, or runtime code:

1. Read project docs and scope first
   - Confirm the assigned task, the active Cocos version, and the target platform before looking up engine behavior.

2. Read the local Cocos 3.8 official mirrors first
   - Manual mirror: `~/MyWork/我的知识库/Projects/我的宗门 Wiki/raw/sources/cocos-creator-3.8-manual-zh/`
   - API mirror: `~/MyWork/我的知识库/Projects/我的宗门 Wiki/raw/sources/cocos-creator-3.8-api-zh/`
   - Distilled project notes: `~/MyWork/我的知识库/Projects/我的宗门 Wiki/wiki/concepts/`
   - Source-note index: `~/MyWork/我的知识库/Projects/我的宗门 Wiki/wiki/sources/`
   - Use the manual and API mirrors as the primary local authority. Use `wiki/concepts` and `wiki/sources` as helper notes for context, indexing, and source tracing.

3. Freeze the official contract and the official recommended path
   - Confirm both the version-correct engine behavior and the official preferred workflow before deciding how to implement.

## Configuration-First Order

1. Read official docs first
   - Freeze the external contract from the Cocos manual and the target platform docs before proposing changes that depend on orientation, safe area, build output, or editor-owned properties.

2. Look for official best practice before inventing a workaround
   - Prefer the engine's documented usage pattern, recommended workflow, and built-in composition model before designing a custom abstraction.

3. Treat editor and config as the primary control surface
   - Design resolution, build orientation, camera visibility, canvas-camera binding, node layer, and component properties are editor/config problems first, not runtime-code problems first.

4. Check built-in components and engine systems before custom code
   - If a built-in component, asset workflow, runtime system, or editor extension API can solve the problem, prefer that path over writing new engine-facing glue.

5. Ask for human assist before a code fallback
   - If the correct editor control exists but the current MCP surface cannot read or modify it reliably, ask the human to perform that step before dropping to file edits or workaround code.

6. Use community examples only after official sources leave a real gap
   - Community or network examples may help with edge cases or implementation shape, but they must not override the official contract already frozen from the project docs and official references.

7. Prefer the simplest validation path
   - Reuse the existing preview tab, built-in debug surfaces, bounded logs, and simple observation adjustments before inventing new runtime adapters only to inspect layout.

8. Record why a fallback was needed
   - If file edits or runtime code still become necessary, explain why the project docs, official docs, official practice, editor path, built-in components, human-assist path, and community examples were not enough.

## Local Preview Constraints

- Use the dedicated browser workflow for any browser preview or debug surface related to this project.
- Reuse existing dedicated-browser tabs whenever the needed preview is already open.
- For the current project stage, confirm the feature can compile and run through the documented `Cocos Creator` project path first. Do not escalate into WeChat/Douyin developer tools unless the assigned task explicitly depends on platform-only behavior or the human explicitly requests platform-container validation.
- A local browser preview is not the real mini-game container. If the result depends on platform-only APIs or runtime behavior, mark the evidence as environment-limited instead of pretending the browser fully proved it.
- After scene, component, or script changes, save the current scene and wait briefly before judging whether the preview has refreshed.
- If MCP becomes unavailable or clearly unstable, stop guessing and ask the human to restart it.
- If external file edits are not reflected in preview, prefer the documented reimport, refresh, or scene-reload path before assuming the gameplay logic is wrong or opening duplicate preview tabs.

## Code Edit Default

For this project, the default code-edit loop is:

1. edit the file directly
2. save the file
3. reload or re-check the existing dedicated-browser preview at `127.0.0.1:9333`
4. judge the result from the preview/runtime evidence first

This is the normal path for:

- edits to existing TypeScript files
- edits to existing runtime assets already known to Cocos
- runtime logic verification where the preview is already loading the changed module

Do **not** treat Cocos-side refresh or reimport as the default follow-up after every code edit.

## Refresh/Reimport Exception

`project_refresh_assets` or `project_reimport_asset` is allowed only as an exception when there is concrete evidence that the preview is not loading the changed file graph yet.

Typical allowed cases:

- a brand-new script file or directory was added under `assets/`
- a new `.meta` file was added
- the browser preview was reloaded, but the new script or asset is still absent from the Cocos preview dependency graph
- the preview import map or runtime evidence shows the new file was not picked up by asset-db yet

Required discipline:

- describe the action as `refresh/reimport`, not as a normal compile/build step
- use the smallest scope possible, such as one asset URL or one folder
- do it only after the direct save + preview check has already failed to surface the new file
- do not normalize this fallback into the everyday workflow

In short:

- existing-file edits: save and verify in `9333`
- new-file ingestion problems: one bounded `refresh/reimport` is acceptable

## MCP Server Start Order

Use this order when the Cocos MCP server is required.

### 1. Preferred startup path: project settings auto-start

Source of truth:

- `client/my-immortal-sect/settings/mcp-server.json`

Current expected shape:

```json
{
  "port": 9527,
  "autoStart": true,
  "enableDebugLog": false,
  "debugLog": false,
  "allowedOrigins": ["*"],
  "maxConnections": 10
}
```

Expected behavior:

- when Cocos Creator opens the `client/my-immortal-sect/` project
- the extension `load()` hook reads `mcp-server.json`
- if `autoStart` is `true`, it calls the extension-local `mcpServer.start()`

### 2. Manual startup path inside the editor

If auto-start does not work:

- open the Cocos Creator project `client/my-immortal-sect/`
- open the extension panel from the editor menu:
  - `Extensions -> Cocos MCP Server`
- use the panel's start-server action

Code reference:

- `source/main.ts` registers `startServer()`
- the panel calls `Editor.Message.request('cocos-mcp-server', 'start-server')`

### 3. Health verification

After startup, verify in this order:

1. `http://127.0.0.1:9527/health`
   - expected response shape:
   - `{"status":"ok","tools":<count>}`
2. `http://127.0.0.1:9527/mcp`
   - expected to exist as the MCP HTTP endpoint
3. optional project config check:
   - `client/my-immortal-sect/settings/mcp-server.json`

### 4. If source code changed

If the extension source under `extensions/cocos-mcp-server/source/` changed:

```bash
cd client/my-immortal-sect/extensions/cocos-mcp-server
npm install
npm run build
```

Then reload or reopen the project in Cocos Creator before assuming the runtime server behavior changed.

## Known Limitation: Agent Start Boundary

If `9527` is already up, the agent can safely detect and use it.

If `9527` is **not** up:

- the agent can detect that the server is down
- but the agent must not assume it can always start the service from outside the editor

Reason:

- the start entry lives inside the Cocos extension host
- once the MCP HTTP service is down, the external HTTP control surface is also gone
- unless there is a separate verified editor-control path available, restarting the server is a human/editor action first

Practical rule:

- if the editor is open and the service is unexpectedly down, ask the human to restart the extension or reopen the project unless a verified editor automation path is already available in the current task
- do not pretend that shell access alone can reliably start the Cocos MCP server

## Known Fallback For Panel Save Problems

If the extension panel does not reliably persist `autoStart`:

- edit `client/my-immortal-sect/settings/mcp-server.json` directly
- set `autoStart` to `true`
- restart Cocos Creator and reopen the project

This is the accepted fallback for this project until the panel-side save issue is investigated and fixed.

## Asset Boundary Rule

- Keep generator scripts, pipeline specs, preview sheets, and other tool artifacts outside the runtime Cocos `assets/` tree unless the shipped game actually loads them.
- Only final integration-ready runtime assets should live in engine-owned asset paths.

## Handoff Expectations

- Name the scene, prefab, asset, or configuration files touched.
- State whether the change was editor/config, runtime code, or asset-pipeline work.
- State which part was changed in editor-facing terms, not only raw code terms.
- Report the validation path and any remaining editor-only follow-up.
