# Dedicated Browser Acceptance Replay Gate

Use this gate when `engineer` or `supervisor` needs bounded acceptance evidence from the real Cocos preview without relying on oversized logs or ad-hoc console interpretation.

## Scope

This gate reuses the dedicated Chrome instance at `http://127.0.0.1:9333` and the preview page at `http://localhost:7456/`.

It proves one repeatable authority-owned path:

- default player-facing controls are not covered by the permanent HUD, and the `宗务` control opens the sect panel
- resource gather designation is submitted as an authority intent and reflected back through snapshot state
- disconnected gather designation fails closed without client-local fallback mutation
- hostile movement is presented through snapshot-target interpolation rather than hard tile jumps
- combat HP drops produce bounded damage-floater evidence
- defend/recover/restore checkpoints keep bounded risk intensity, mitigation, omen, defense summary, damage summary, and repair suggestion fields so raid feedback remains authority-owned
- the sect authority loop runs a compressed multiday management replay covering recruitment, task dispatch, production, cultivation, monthly settlement, and event feedback
- the same multiday sect replay opens the roster/story/diary surfaces and records bounded disciple-memory, relationship, and story-feedback evidence with at least three readable feedback types
- the event choice surface opens through the sect panel, submits one `ChooseEventOption` intent, and verifies the resulting resolved event / feedback came from authority snapshot state
- the policy surface accepts authority `SetPolicy` intents and records multiday policy-driven metric changes in bounded evidence
- the artifact surface crafts and equips at least one authority artifact, then records bounded task or production metric deltas plus equipped-slot evidence
- the promotion surface path submits `StartAssessment` and `PromoteDisciple` intents through the dedicated browser command bridge, then verifies the authority roster identity and promotion feedback
- the building maintenance path builds/upgrades authority buildings, proves maintenance pressure can damage efficiency/durability, repairs through `RepairBuilding`, rejoins latest state, and advances a post-restore continuity window
- the institution management path opens the sect affairs overview, verifies at least two authority-fed institution detail entries, and records manager/capacity/medicine/cave/treasury effect summaries
- the external task path dispatches at least one authority external task through the browser command bridge, then records success rate, risk, result evaluation, feedback, and last-error state
- the expansion-risk midgame replay upgrades sect core buildings, configures institutions, dispatches external/combat work, equips artifacts, attaches formations, handles governance/crisis choices, surfaces treatment/order/monthly feedback, and proves rejoin/offline continuity with bounded JSON
- `reset` returns the preview to `clear_ruin`
- the mainline progresses through `place_guard_tower -> upgrade_guard_tower -> defend -> recover`
- `restore_latest` preserves bounded recover-state shape
- the same restored session continues to `second_cycle_ready`

## Preconditions

- dedicated Chrome is already reachable at `127.0.0.1:9333`
- the preview page can be opened at `http://localhost:7456/`
- authority server is running at `127.0.0.1:8787`
- the preview page exposes `globalThis.__MIS_RUNTIME_DEBUG__`
- the authority server exposes the bounded acceptance-only sect debug endpoints used by this gate: `/v1/authority/sect/debug/reset` and `/v1/authority/sect/debug/advance-days`
- `/v1/authority/sect/debug/reset` may receive bounded initial resource overrides for deterministic shortage setup; this is acceptance setup only and must not become a gameplay command

Suggested authority startup:

```bash
cd /Users/mawei/MyWork/SlgGame/server
go run ./cmd/gameserver
```

## Gate Command

```bash
cd /Users/mawei/MyWork/SlgGame
node server/tools/acceptance/dedicated_browser_replay_gate.mjs
```

If the preview tab is not already open in dedicated Chrome, the script opens `http://localhost:7456/` inside the same dedicated browser. Use `--reuse-only` when the current role must prove reuse of the already-open tab.

Use `--authority-url <url>` only when the authority server is not on the default `http://127.0.0.1:8787`.

## Output

The gate writes bounded JSON evidence to:

- `workspace/acceptance/dedicated-browser-replay-gate/latest.json`
- `workspace/acceptance/dedicated-browser-replay-gate/<timestamp>.json`

The JSON captures only checkpoint summaries and assertions, not full runtime logs.

## Required Assertions

- `assertions.reset_clear_ruin = true`
- `assertions.primary_controls_unobstructed = true`
- `assertions.sect_panel_opens = true`
- `assertions.resource_designation_round_trip = true`
- `assertions.resource_designation_fail_closed_when_blocked = true`
- `assertions.sect_multiday_authority_connected = true`
- `assertions.sect_multiday_recruitment_roster = true`
- `assertions.sect_multiday_task_result = true`
- `assertions.sect_multiday_production = true`
- `assertions.sect_multiday_cultivation = true`
- `assertions.sect_multiday_monthly = true`
- `assertions.sect_multiday_event_feedback = true`
- `assertions.sect_multiday_story_feedback_surface = true`
- `assertions.sect_multiday_story_feedback_types = true`
- `assertions.sect_multiday_restore_offline_continuity_gate = true`
- `assertions.sect_event_choice_panel_opens = true`
- `assertions.sect_event_choice_authority_result = true`
- `assertions.sect_event_choice_feedback = true`
- `assertions.sect_policy_browser_switch = true`
- `assertions.sect_policy_multiday_metrics = true`
- `assertions.sect_policy_last_error_null = true`
- `assertions.sect_artifact_browser_command = true`
- `assertions.sect_artifact_metric_delta = true`
- `assertions.sect_artifact_last_error_null = true`
- `assertions.sect_promotion_assessment_browser_command = true`
- `assertions.sect_promotion_authority_result = true`
- `assertions.sect_promotion_feedback = true`
- `assertions.sect_promotion_last_error_null = true`
- `assertions.sect_building_maintenance_pressure = true`
- `assertions.sect_building_damage_repair = true`
- `assertions.sect_building_restore_latest = true`
- `assertions.sect_building_offline_continuity = true`
- `assertions.sect_building_last_error_null = true`
- `assertions.sect_institution_overview_panel_opens = true`
- `assertions.sect_institution_two_panels = true`
- `assertions.sect_institution_authority_effects = true`
- `assertions.sect_institution_last_error_null = true`
- `assertions.sect_external_dispatch_browser_command = true`
- `assertions.sect_external_dispatch_result = true`
- `assertions.sect_external_dispatch_feedback = true`
- `assertions.sect_external_dispatch_last_error_null = true`
- `assertions.sect_expansion_risk_buildings = true`
- `assertions.sect_expansion_risk_institutions = true`
- `assertions.sect_expansion_risk_artifacts_and_formations = true`
- `assertions.sect_expansion_risk_goals_and_monthly = true`
- `assertions.sect_expansion_risk_crisis_and_treatment = true`
- `assertions.sect_expansion_risk_panel_surface = true`
- `assertions.sect_expansion_risk_restore_offline = true`
- `assertions.sect_expansion_risk_last_error_null = true`
- `assertions.place_guard_tower = true`
- `assertions.upgrade_guard_tower = true`
- `assertions.guard_tower_lv2_active = true`
- `assertions.defend_started = true`
- `assertions.hostile_interpolation_not_hard_jump = true`
- `assertions.damage_floater_captured = true`
- `assertions.recover_started = true`
- `assertions.authority_risk_defense_feedback_visible = true`
- `assertions.restore_matches_recover_state = true`
- `assertions.second_cycle_ready = true`
- `assertions.final_last_error_null = true`

## Bounded Checkpoints

Each checkpoint records only:

- authority connection, session id, game tick, last error, last event
- session phase, objective, guard/ruin ids, damaged/regenerating counts
- session risk intensity, mitigation, threat curve, defense rating, guard count, omen text/status, defense summary, damage summary, repair suggestion, and bounded risk source summary
- stockpile
- disciple task/carrying/render-relevant state
- active hostile summaries
- active damage-floater summaries
- building summaries
- artifact inventory, equipped-slot, and task/production metric summaries
- player-facing control / resource designation / combat presentation summaries
- sect authority roster, candidates, tasks, production, cultivation, active/resolved event choices, tension, diary, and event feedback summaries
- disciple story feedback summaries limited to authority-derived recent experience, emotion/relationship tags, mood reasons, diary rows, and event-feedback categories
- promotion evidence records only roster assessment/readiness summaries, browser command results, authority identity changes, and promotion feedback summaries
- building maintenance evidence records only authority building catalog/instances, maintenance pressure, damage/repair metrics, resources, scene versions, and last-error state
- institution management evidence records only sect affairs overview text flags, bounded institution summaries, manager assignment/effect metrics, and last-error state
- external dispatch evidence records only task type, risk, success rate, recommendation, result status/evaluation, reward summary, feedback categories, scene versions, and last-error state
- expansion-risk evidence records only bounded sect state needed for acceptance: core building levels/phases, institution count, artifact/formation summaries, resolved goals, crisis/treatment/order/monthly counters, sect-panel section visibility, rejoin/offline continuity comparables, and last-error state
- expansion-risk panel evidence only requires reopening the sect panel and exposing the bounded management surface (`经营总览` / `宗门等级` / `政策`); goals/crisis/treatment/monthly continuity is validated from bounded runtime/browser summaries instead of relying on one long panel string

Current checkpoint set:

- `reset_clear_ruin`
- `place_guard_tower`
- `upgrade_guard_tower`
- `guard_tower_lv2_active`
- `defend_started`
- `recover_started`
- `restore_latest`
- `second_cycle_ready`
- `sect_multiday_start`
- `sect_multiday_final`
- `sect_multiday_story_feedback`
- `sect_event_choice_before`
- `sect_event_choice_after`
- `sect_policy_start`
- `sect_policy_after_switch`
- `sect_policy_after_thirty_days`
- `sect_artifact_before`
- `sect_artifact_after_equip`
- `sect_promotion_before_assessment`
- `sect_promotion_after_assessment`
- `sect_promotion_after_promote`
- `sect_building_maintenance_after_build`
- `sect_building_maintenance_damaged`
- `sect_building_maintenance_repaired`
- `sect_building_maintenance_restored`
- `sect_building_maintenance_offline`
- `sect_institution_management_after_commands`
- `sect_institution_management_panel`
- `sect_external_dispatch_before`
- `sect_external_dispatch_after_dispatch`
- `sect_external_dispatch_after_result`
- `sect_expansion_risk_start`
- `sect_expansion_risk_final`
- `sect_expansion_risk_rejoin`
- `sect_expansion_risk_offline`
- `sect_expansion_risk_browser`
- `sect_expansion_risk_panel`

## Acceptance Use

### Engineer submit

Reference:

- the script file
- the generated `latest.json`
- the command used to run the gate

### Supervisor accept

Run the same command independently and compare the resulting bounded assertions plus checkpoint summaries.

If the gate cannot attach to `127.0.0.1:9333`, cannot find `__MIS_RUNTIME_DEBUG__`, or returns a non-null authority error at the final checkpoint, acceptance must fail closed.
