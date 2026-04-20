# Cocos Creator Workflow

Use this document when an assigned task depends on Cocos Creator editor behavior, engine settings, or editor-driven asset and scene changes.

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

## Asset Boundary Rule

- Keep generator scripts, pipeline specs, preview sheets, and other tool artifacts outside the runtime Cocos `assets/` tree unless the shipped game actually loads them.
- Only final integration-ready runtime assets should live in engine-owned asset paths.

## Handoff Expectations

- Name the scene, prefab, asset, or configuration files touched.
- State whether the change was editor/config, runtime code, or asset-pipeline work.
- State which part was changed in editor-facing terms, not only raw code terms.
- Report the validation path and any remaining editor-only follow-up.
