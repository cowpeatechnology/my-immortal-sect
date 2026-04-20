# Engineering Standards

These are the default standards for a 2D Cocos Creator project bootstrapped by Coordex.

## Default Standards

- Prefer small, reviewable changes over broad rewrites.
- Change only the files required for the scoped task.
- Validate with the real project commands whenever they are documented.
- For engine, platform, framework, editor, build, or runtime questions, read official docs first and freeze the external contract before coding or accepting a change.
- Bound research to what is required to freeze the external contract for the scoped task. After that, move into the smallest runnable implementation immediately.
- Do not let a scoped implementation task stay in commentary-only research after the implementation path is clear. If writing cannot start, report a blocker.
- Prefer existing documented engine, framework, platform, editor, and runtime capabilities before custom workaround code.
- Prefer configuration or framework-supported paths over runtime-code workarounds when the actual stack already provides the required control surface.
- If tooling cannot reliably operate an existing editor control, ask the human to perform that editor step before approving a file-level or code-level fallback.
- Prefer the simplest validation path that can answer the question, including the current preview, existing debug surface, bounded log reads, and straightforward observation fixes.
- If code fallback is still required for an editor-, platform-, or configuration-owned problem, explain in the handoff why the official path, editor path, and human-assist path were insufficient.
- Local browser preview is not the real mini-game container. If a result depends on platform-only runtime APIs, record that as environment-limited instead of overstating what the browser proved.
- After scene, component, or script edits in Cocos, save the current scene and wait briefly before judging the preview result. If external file edits still look stale, use the documented reimport or refresh path before assuming the implementation failed.
- Keep generator scripts, pipeline specs, preview-only compositions, and other tool artifacts out of the runtime Cocos `assets/` tree unless the shipped game actually loads them.
- Browser validation is a hard constraint: reuse the dedicated Chrome instance at `http://127.0.0.1:9333` with remote-debugging-port `9333` and user-data-dir `/tmp/chrome-mcp-dedicated-9333`.
- Reuse already-open dedicated-browser tabs whenever they already show the needed preview or page. Do not multiply tabs without need.
- Do not launch default Chrome, temporary Chrome profiles, or auto-connect fallback browsers during validation.
- If a command cannot be run, say so explicitly in the handoff.
- Do not silently widen task scope just because a neighboring issue is visible.
- When repeated context becomes necessary, write it back into checked-in docs.

## Default Research And Solution Order

Use this order when a task depends on engine, platform, framework, editor, build, runtime, or editor-extension behavior:

1. Read project docs and the assigned scope first
   - Confirm the active milestone, target stack, and task boundary from project docs before widening into engine references.

2. Prefer the local official Cocos 3.8 mirrors before generic web search
   - Manual mirror: `~/MyWork/我的知识库/Projects/我的宗门 Wiki/raw/sources/cocos-creator-3.8-manual-zh/`
   - API mirror: `~/MyWork/我的知识库/Projects/我的宗门 Wiki/raw/sources/cocos-creator-3.8-api-zh/`
   - Project-level distilled notes: `~/MyWork/我的知识库/Projects/我的宗门 Wiki/wiki/concepts/`
   - Project-level source notes: `~/MyWork/我的知识库/Projects/我的宗门 Wiki/wiki/sources/`
   - Treat the manual and API mirrors as the primary local authority. Treat `wiki/concepts` and `wiki/sources` as secondary notes that help locate or summarize official material, not as replacements for the official contract.

3. Freeze the official contract and preferred practice before designing
   - Identify the version-correct official behavior first.
   - Prefer documented official workflows, recommended usage patterns, and built-in best practices before inventing a custom design.

4. Check the editor and project configuration path before code
   - If the engine exposes the control in editor panels, project settings, scene properties, build settings, import settings, or extension APIs, treat that as the primary path.

5. Check built-in components, engine systems, and framework capabilities before custom code
   - Prefer existing components, runtime systems, asset workflows, and engine-supported composition over writing a new abstraction from scratch.

6. Ask for human assist before bypassing an editor-owned control
   - If the correct editor control exists but current tooling or MCP cannot read or modify it reliably, ask the human to perform that step before approving a code fallback.

7. Use community or network examples only after official sources leave a real gap
   - Community examples may help with implementation shape or edge cases, but they must not override the official contract already frozen from project docs and official references.

8. Write a custom implementation only as the last fallback
   - If the solution still requires custom code, explain in the handoff why the project docs, official docs, official practice, editor/config path, built-in components, and community examples were insufficient.
