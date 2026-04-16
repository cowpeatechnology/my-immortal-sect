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

Code worktrees such as `client/`, `server/`, and `shared/` do not exist yet.

Important clarification:
- `hifi-prototype/` may be used for visual exploration or interaction sketching, but it does **not** satisfy a gameplay milestone by itself.
- When the user asks for a map slice that is "really playable", that means a real stack implementation using the documented project stack, not an HTML-only prototype.
- Real implementation for this project means: `Cocos Creator + TypeScript + Tiled/TiledMap` on the client and `Go + Hollywood + PostgreSQL/Redis` on the server side as applicable to the slice.
- Repository initialization and scaffolding are now allowed when they are tied to an approved milestone or explicit supervisor work order.

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

This workspace is not guaranteed to be a Git repository. Do not assume git-based workflows, commits, or resets are available.

## 6. Main-thread operating model

The main Codex thread acts as the project supervisor, not as a single all-purpose implementer.

Default behavior:
- Intake the request.
- Classify whether it is product/design, architecture, client, server, tools/pipeline, or QA.
- If the task spans multiple domains, break it into owned work orders before implementation.
- Delegate domain work to specialized subagents when available.
- Keep final integration, conflict resolution, and user-facing synthesis in the main thread.

The main thread is responsible for:
- enforcing authority docs
- sequencing work
- preventing scope bleed
- requesting evidence from workers
- deciding when work is ready to integrate

Important distinction:
- Persistent in this project means persistent role definitions and persistent process rules.
- It does not mean keeping many long-lived worker threads alive forever.
- Outside the main supervisor thread, worker threads should usually be spawned for a scoped work order and then closed after handoff or acceptance.

Rationale:
- reduces context rot
- keeps write scope explicit
- avoids mixing tools work, product design, and implementation state in one agent thread

## 7. Required specialist roles

Default long-lived role templates for this repository:

- `gameplay_designer`: gameplay loops, disciple systems, karma flow, content-facing design docs, and product-facing game design
- `technical_architect`: architecture, ADRs, module boundaries, implementation plans
- `client_engineer`: Cocos Creator, TypeScript, tilemap rendering, UI/event-bus/client sync
- `server_engineer`: Go, Hollywood, simulation, save/sync, protobuf, persistence
- `art_asset_producer`: art requirement breakdown, prompt shaping, asset batch generation workflows, naming/output discipline, and visual acceptance preparation
- `qa_verifier`: acceptance criteria, test plans, regression review, risk review

Project-specific decision:
- Do not create a separate standalone `product_manager` agent for now.
- The user is the real product owner and project director.
- `gameplay_designer` covers ongoing game-system and feature-definition work.
- `technical_architect` covers execution structure and engineering tradeoffs.
- A dedicated PM agent at this stage would mostly add manager-of-manager overhead.

Temporary role templates may also be used for bounded work:
- `feature_worker`: short-lived implementation agent for a well-scoped coding task
- `tools_engineer`: short-lived tools/pipeline agent for support workflows, scripts, and one-off automation

Execution default for all subagents in this repository:
- model: `gpt-5.4`
- reasoning effort: `xhigh`

If a task only belongs to one domain, do not create fake collaboration overhead. Assign one owner and one verifier at most.

## 8. Work order format

Before specialized work begins on a cross-functional task, the supervisor should define:

- objective
- owner
- explicit write scope
- explicit no-touch scope
- authority docs to read first
- expected deliverable
- validation required before handoff
- loop record to update (`docs/project/delivery-ledger.md` and the relevant `docs/features/F-xxx-<slug>.md` file for non-trivial work)

Example:
- Objective: draft the first Storylet DSL spec
- Owner: `gameplay_designer`
- Write scope: `docs/design/systems/`, `docs/design/content/`
- No-touch scope: `tools/`, `hifi-prototype/`, root architecture docs
- Authority docs: `AGENTS.md`, `docs/vision/design-decisions.md`, `docs/README.md`
- Deliverable: one spec doc plus unresolved questions
- Validation: `technical_architect` reviews engine coupling and migration impact

Default collaboration topology:
- All worker communication is hub-and-spoke through the supervisor.
- Workers do not silently coordinate among themselves or assume another worker has seen their context.
- If one worker needs another domain, the request goes back to the supervisor for rerouting.

## 9. Handoff contract

Every worker handoff must include:

- what changed
- files touched
- tests or checks run
- unresolved risks
- assumptions made
- recommended next owner
- startup or boot commands if runtime work exists
- QA entry URL / screen / command if the work can be exercised directly
- exact validation flow the supervisor or QA should follow

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
- use templates under `docs/templates/` for supervisor work orders, worker handoffs, and feature loop records

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
