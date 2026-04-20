# My Immortal Sect Codex Operating Manual

This file is the primary Codex-facing project charter for `My Immortal Sect` (`我的宗门`).

Detailed engineering execution rules live in [docs/process/engineering-standards.md](/Users/mawei/MyWork/SlgGame/docs/process/engineering-standards.md). This file defines repository-wide authority, stage gates, and the default multi-agent operating model.

## 1. Project identity

- Project: `My Immortal Sect` (`我的宗门`)
- Local repo codename: `SlgGame`
- Genre: xianxia sect-management sim inspired by RimWorld-style emergent narrative
- Platform: WeChat / Douyin mini-game
- Presentation: portrait 2.5D, sect map and big-head disciple presentation
- Core fantasy: the sect is home; disciples are emotional anchors; rare disciples are long-tail karma contracts

## 2. Current stage

The repository is in transition from planning into real M0 implementation.

Current top-level contents:
- `docs/`: authority docs and plans
- `hifi-prototype/`: early HTML/JS exploration artifact, preserve by default
- `tools/`: art-generation and workflow tooling, preserve by default

Current code worktrees:
- `client/my-immortal-sect/`: active Cocos Creator 3.8.8 client project
- `server/`: reserved Go server runtime skeleton
- `shared/`: reserved shared config / contract skeleton

Current Cocos runtime facts:
- project path: `client/my-immortal-sect/`
- project-local MCP settings: `client/my-immortal-sect/settings/mcp-server.json`
- MCP HTTP server port: `9527`
- MCP extension source-of-truth: `client/my-immortal-sect/extensions/cocos-mcp-server/`

Important clarification:
- `hifi-prototype/` may be used for visual exploration or interaction sketching, but it does **not** satisfy a gameplay milestone by itself.
- When the user asks for a map slice that is "really playable", that means a real stack implementation using the documented project stack, not an HTML-only prototype.
- Real implementation for this project means: `Cocos Creator + TypeScript + Tiled/TiledMap` on the client and `Go + Hollywood + PostgreSQL/Redis` on the server side as applicable to the slice.
- Repository initialization and scaffolding are now allowed when they are tied to an approved milestone or explicit supervisor work order.
- Git topology rule: `~/MyWork/SlgGame/.git` is the only canonical project repository. Any nested `.git` directory under the project root is a configuration error unless the human explicitly restores that exception.
- For Cocos MCP work, edit `client/my-immortal-sect/extensions/cocos-mcp-server/` directly as part of the root project repository, not as a separate maintained Git working tree.

## 3. Authority order

Resolve conflicts in this order:

1. Direct user instruction
2. This [AGENTS.md](/Users/mawei/MyWork/SlgGame/AGENTS.md)
3. [docs/vision/design-decisions.md](/Users/mawei/MyWork/SlgGame/docs/vision/design-decisions.md)
4. [docs/process/engineering-standards.md](/Users/mawei/MyWork/SlgGame/docs/process/engineering-standards.md)
5. Relevant ADRs under `docs/decisions/`
6. [docs/README.md](/Users/mawei/MyWork/SlgGame/docs/README.md) and current plans under `docs/plans/`
7. `docs/legacy/` for reference only

If a required authority doc is missing, do not invent it. Draft the missing doc or escalate to the user.

## 4. Non-negotiable game and architecture rules

- Do not split karma and divine descent into separate engines. They share one Storylet engine.
- Do not move emotional focus from internal sect disciples to external world spectacle.
- Do not silently trigger major karma events. Fate must be foreshadowed.
- Do not replace the big-head-first presentation strategy with full-body animation-heavy solutions.
- Do not move economy, battle settlement, karma triggers, or authoritative state decisions to the client.
- Do not modify Hollywood upstream source for business logic.
- Do not model disciples, buildings, or active storylets as standalone server actors in V1. They belong inside the simulation big-State handled by one `SimulationActor`.
- Do not use wall-clock time or package-level randomness inside simulation logic.
- Do not create a second event system, second save protocol, or second content pipeline because it feels faster in the moment.

## 5. Protected areas

Protected by default:
- `hifi-prototype/`
- `tools/`

Rules:
- Do not modify either protected area unless the user explicitly asks for prototype or tools work.
- If a task touches a protected area and gameplay docs at the same time, split ownership and keep tool changes isolated.
- `docs/` is authoritative, not disposable scratch space.

This workspace is currently a Git repository. Use non-destructive git workflows when needed, and do not rewrite or discard history unless the human explicitly asks.

## 6. Coordex Visible-Role Operating Model

This repository now uses Coordex's visible-role workflow as the authoritative project collaboration model.

Current operating assumptions:
- durable role chats live under `Agents/<role>/`
- the confirmed durable role set is `supervisor`, `engineer`, and `art_asset_producer`
- root chats remain temporary project-root conversations and are not part of the durable role roster
- the human remains the final authority over scope, priority, and product direction
- inside the visible-role system, `supervisor` acts as the project product owner and coordination lead
- older specialist `.codex/agents` templates have been moved to `.codex/agents/legacy/` and disabled; they are historical reference, not active project roles

Rationale:
- responsibilities stay explicit and inspectable
- durable context stays attached to visible role threads instead of hidden child-thread chains
- the operator can read planning, execution, blockers, and handoffs without reverse-engineering opaque subagent state

Runtime safety defaults:
- Do not read full observer / network / HAR / capture logs into Codex context by default. Use bounded sampling first: keyword search, timestamp windows, `tail`, `head`, or narrow offset-based reads.
- For runtime evidence, prefer `file path + timestamp/line window + event summary` over pasting raw dumps into the thread.
- Any role expected to run longer than 1 minute, or expected to watch an external runtime, must have an explicit heartbeat cadence in the work order. Default cadence: `60s`.
- Default silence handling: after `120s` without a useful heartbeat, the supervisor should inspect status, interrupt for a checkpoint, or close the run if the task is clearly wedged or no longer needed.

## 7. Confirmed Role Set

Default durable roles for this repository:

- `supervisor`: owns current goal, milestone planning, product decisions, task routing, scope boundaries, and final acceptance
- `engineer`: owns technical architecture, implementation, integration, debugging, and technical validation across the real project stack
- `art_asset_producer`: owns visual direction breakdown, asset planning, generation workflow usage, output naming, and delivery packaging

Confirmed responsibility merge:
- there is no separate standalone `product_manager` role in the current workflow
- there is no separate standalone `technical_architect`, `client_engineer`, `server_engineer`, or `qa_verifier` role in the default Coordex setup for this repository
- architecture ownership is folded into `engineer`
- acceptance ownership is folded into `supervisor`, with the human keeping final approval

Execution default for the current durable roles:
- model: `gpt-5.4`
- reasoning effort: `xhigh`

## 8. Coordination And Work Orders

Before non-trivial work begins, the supervisor should define:

- objective
- owner
- explicit write scope
- explicit no-touch scope
- authority docs to read first
- official external baseline when the task depends on engine / platform / editor / build rules
- expected deliverable
- preferred execution path, especially whether the task is editor/config-first or code-first
- human-assist checkpoint when an editor control exists but MCP coverage may be insufficient
- runtime contract for long-running work, including heartbeat cadence, silence timeout, and evidence source
- validation required before handoff
- records to update, including `docs/project/delivery-ledger.md` and the relevant `docs/features/F-xxx-<slug>.md` file when applicable

Default coordination rules:
- Each active subfunction has exactly one owner role.
- New task start, owner assignment, milestone changes, scope changes, and final acceptance belong to the human or `supervisor`.
- Once a subfunction is already active, direct peer coordination between visible roles is allowed only inside that confirmed scope.
- Direct peer coordination may ask questions, report blockers, exchange integration details, or hand off intermediate results.
- Direct peer coordination must not silently widen scope, self-assign a new feature, or self-accept final completion.
- Use `docs/process/structured-agent-communication-protocol.md` for role-to-role and role-to-supervisor coordination messages that need low drift.
- Mirror only high-value coordination events into `docs/project/thread-conversation-ledger.md` when durable visibility or auditability helps.

If a task only belongs to one domain, do not invent fake collaboration overhead. Prefer one owner role plus supervisor acceptance.

## 9. Handoff Contract

Every worker handoff must include:

- what changed
- files touched
- tests or checks run
- official docs or external contract used when the task depends on outside platform / engine behavior
- if code or raw file edits were used for a configuration-class task, why editor/config and human-assist paths were insufficient
- evidence references when runtime artifacts exist, using bounded paths / timestamps / line windows instead of full raw logs
- unresolved risks
- assumptions made
- recommended next role or human action
- startup or boot commands if runtime work exists
- validation entry URL / screen / command if the work can be exercised directly
- exact validation flow the supervisor or human should follow

The supervisor should not present work to the user as final if those items are missing.

## 10. Documentation discipline

When editing docs:
- one document, one topic
- preserve authority hierarchy
- do not rewrite history inside `legacy/` to match new decisions
- include status, dependencies, and unresolved questions when appropriate
- prefer ADRs for architecture decisions and plan docs for sequencing

For iterative development work:
- update [docs/project/delivery-ledger.md](/Users/mawei/MyWork/SlgGame/docs/project/delivery-ledger.md) to reflect current loop status
- keep one feature loop doc under `docs/features/` per substantial feature or initiative
- use templates under `docs/templates/` for supervisor work orders, worker handoffs, feature loop records, and thread messages

## 11. Engineering discipline

- Use plan-first behavior for cross-module or architecture-affecting changes.
- Prefer minimal, explicit edits over speculative scaffolding.
- Keep client and server responsibilities sharply separated.
- Keep tools and pipelines decoupled from production runtime assumptions.
- Treat configuration IDs as immutable once they become part of the live content model.
- Every accepted development loop should leave a durable trail: updated docs, validation notes, and at least one traceable commit once git is active in the local repository.

## 12. Practical defaults

- If the user asks for design, start with docs and plans before code.
- If the user asks for implementation, confirm whether they want exploratory prototype work or the real documented stack.
- If the user asks for testing, evaluate against documented M0 acceptance criteria instead of ad-hoc taste.
- If the user asks for art-pipeline work, isolate it from gameplay/system design work unless the request explicitly joins them.
- If the task depends on external platform, engine, editor, or build behavior, check official docs first and freeze the contract before implementation.
- If the task is configuration-class work, prefer editor/config paths first, then human assist if MCP is insufficient, and only then consider code fallback.
- For the current client-local phase, treat `Cocos Creator` compile/export capability plus the documented preview path as the primary validation surface. Do not schedule WeChat/Douyin developer-tool or real mini-game container testing by default; defer that work until the human explicitly asks to switch into platform-container validation.

## Coordex Workflow

<!-- COORDEX:PROJECT-WORKFLOW:START -->
This block is maintained by Coordex from the `2d-cocos-creator-game-development` template. Keep project-specific identity facts elsewhere in the root file; keep repeatable coordination rules here or in the referenced durable docs.

- Template: `2D Cocos Creator Game Development`
- Durable role threads live under `Agents/<role>/`.
- The supervisor owns planning, routing, scope boundaries, and final acceptance.
- The supervisor plans and routes by default; it does not absorb worker-owned implementation unless the human explicitly assigns that ownership.
- Each active subfunction must have exactly one implementation owner role.
- The human-readable current plan lives in `.coordex/current-plan.md`.
- `.coordex/project-board.json` is the machine-consumed Coordex board state. Keep it schema-accurate if it must be repaired, but do not treat it as a free-form planning note.
- Role-local durable context belongs in `docs/project/role-state/<role>.md` instead of long chat history.
- When the same missing fact causes repeated mistakes, write it back into this file or another checked-in project doc.

### Default Roles

- `supervisor`: product owner, milestone planner, dispatcher, and final accepter.
- `engineer`: technical architecture, implementation, integration, debugging, and technical validation.
- `art_asset_producer`: visual direction, asset planning, and asset production for assigned scope.

### Required Durable Docs

- `docs/project/project-method.md`
- `docs/project/delivery-ledger.md`
- `docs/project/thread-conversation-ledger.md`
- `docs/project/decision-log.md`
- `docs/process/dedicated-browser-workflow.md`
- `docs/process/cocos-mcp-workflow.md`
- `docs/process/engineering-standards.md`
- `docs/process/development-loop.md`
- `docs/process/structured-agent-communication-protocol.md`

### Template-Specific Expectations

- This template assumes a 2D Cocos Creator game workflow unless project-specific docs later narrow the stack further.
- Browser validation is only valid when attached to the dedicated Chrome instance at `http://127.0.0.1:9333`.
- Keep changing execution state in plan, ledger, and role-state files instead of bloating the root `AGENTS.md`.
<!-- COORDEX:PROJECT-WORKFLOW:END -->

## Coordex Agent Roles

<!-- COORDEX:AGENT-ROSTER:START -->
This block is maintained by Coordex and keeps active role agents aligned across the local role directories under `Agents/`, the Codex threads started from those directories, and this project-level roster.

Agent threads should start in `Agents/<role>/` so Codex loads instructions from the project root down to the role directory: this `AGENTS.md`, then `Agents/AGENTS.md`, then `Agents/<role>/AGENTS.md`.

Root chats created from Coordex remain project-root conversations and are intentionally excluded from this role roster.

| Role | Directory | Thread | Responsibility |
| --- | --- | --- | --- |
| `art_asset_producer` | `Agents/art_asset_producer/` | `art_asset_producer` | Visual direction, asset planning, and SVG or image production for assigned milestones. |
| `engineer` | `Agents/engineer/` | `engineer` | Technical architecture, implementation, integration, debugging, and technical validation. |
| `supervisor` | `Agents/supervisor/` | `supervisor` | Product owner and project coordinator. Owns milestone planning, routing, and final acceptance. |

<!-- COORDEX:AGENT-ROSTER:END -->
