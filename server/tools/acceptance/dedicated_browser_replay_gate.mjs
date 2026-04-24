#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { setTimeout as delay } from 'node:timers/promises';

const DEFAULT_BROWSER_URL = 'http://127.0.0.1:9333';
const DEFAULT_PREVIEW_URL = 'http://localhost:7456/';
const DEFAULT_AUTHORITY_URL = 'http://127.0.0.1:8787';
const DEFAULT_OUTPUT_DIR = '/Users/mawei/MyWork/SlgGame/workspace/acceptance/dedicated-browser-replay-gate';
const DEFAULT_TIMEOUT_MS = 90000;
const DEFAULT_POLL_MS = 500;

function parseArgs(argv) {
    const options = {
        browserUrl: DEFAULT_BROWSER_URL,
        previewUrl: DEFAULT_PREVIEW_URL,
        authorityUrl: DEFAULT_AUTHORITY_URL,
        outputDir: DEFAULT_OUTPUT_DIR,
        timeoutMs: DEFAULT_TIMEOUT_MS,
        pollMs: DEFAULT_POLL_MS,
        openTabIfMissing: true,
    };

    for (let index = 2; index < argv.length; index += 1) {
        const arg = argv[index];
        const next = argv[index + 1];
        if (
            (arg === '--browser-url' ||
                arg === '--preview-url' ||
                arg === '--authority-url' ||
                arg === '--output-dir' ||
                arg === '--timeout-ms' ||
                arg === '--poll-ms') &&
            !next
        ) {
            throw new Error(`Missing value for ${arg}`);
        }
        switch (arg) {
            case '--browser-url':
                options.browserUrl = next;
                index += 1;
                break;
            case '--preview-url':
                options.previewUrl = next;
                index += 1;
                break;
            case '--authority-url':
                options.authorityUrl = next.replace(/\/$/, '');
                index += 1;
                break;
            case '--output-dir':
                options.outputDir = next;
                index += 1;
                break;
            case '--timeout-ms':
                options.timeoutMs = Number(next);
                index += 1;
                break;
            case '--poll-ms':
                options.pollMs = Number(next);
                index += 1;
                break;
            case '--reuse-only':
                options.openTabIfMissing = false;
                break;
            case '--help':
            case '-h':
                printHelp();
                process.exit(0);
                break;
            default:
                throw new Error(`Unknown argument: ${arg}`);
        }
    }

    if (!Number.isFinite(options.timeoutMs) || options.timeoutMs <= 0) {
        throw new Error(`Invalid --timeout-ms: ${options.timeoutMs}`);
    }
    if (!Number.isFinite(options.pollMs) || options.pollMs <= 0) {
        throw new Error(`Invalid --poll-ms: ${options.pollMs}`);
    }

    return options;
}

function printHelp() {
    console.log(`Usage:
  node server/tools/acceptance/dedicated_browser_replay_gate.mjs [options]

Options:
  --browser-url <url>   Dedicated Chrome remote-debugging host. Default: ${DEFAULT_BROWSER_URL}
  --preview-url <url>   Preview page that exposes __MIS_RUNTIME_DEBUG__. Default: ${DEFAULT_PREVIEW_URL}
  --authority-url <url> Authority HTTP endpoint for compressed sect-day replay. Default: ${DEFAULT_AUTHORITY_URL}
  --output-dir <dir>    Directory for bounded replay evidence JSON. Default: ${DEFAULT_OUTPUT_DIR}
  --timeout-ms <ms>     Per-wait timeout. Default: ${DEFAULT_TIMEOUT_MS}
  --poll-ms <ms>        Poll interval for authority snapshot refresh. Default: ${DEFAULT_POLL_MS}
  --reuse-only          Fail instead of opening a preview tab when ${DEFAULT_PREVIEW_URL} is missing.
`);
}

function nowIso() {
    return new Date().toISOString();
}

function slugTimestamp() {
    return nowIso().replace(/[:.]/g, '-');
}

function ensureDirectory(dirPath) {
    fs.mkdirSync(dirPath, { recursive: true });
}

function summarizeSectAuthority(snapshot) {
    const sect = snapshot.sectAuthority ?? {};
    const diary = sect.diary ?? [];
    const eventFeedback = sect.eventFeedback ?? [];
    const recentPolicyEvents = sect.recentPolicyEvents ?? eventFeedback.filter((entry) => entry.category === 'policy');
    return {
        connected: !!sect.connected,
        sectId: sect.sectId ?? null,
        sceneVersion: sect.sceneVersion ?? 0,
        rosterCount: sect.rosterCount ?? 0,
        roster: (sect.roster ?? []).slice(0, 6).map((disciple) => ({
            id: disciple.id,
            name: disciple.name,
            identity: disciple.identity,
            identityText: disciple.identityText,
            realmStage: disciple.realmStage,
            assignmentKind: disciple.assignmentKind,
            workTargetText: disciple.workTargetText,
            contributionText: disciple.contributionText,
            hpText: disciple.hpText,
            relationshipText: disciple.relationshipText ?? '',
            emotionText: disciple.emotionText ?? '',
            recentExperienceText: disciple.recentExperienceText ?? '',
            memoryText: disciple.memoryText ?? '',
            moodReasonText: disciple.moodReasonText ?? '',
            promotion: disciple.promotion
                ? {
                      targetRank: disciple.promotion.targetRank,
                      targetRankText: disciple.promotion.targetRankText,
                      assessmentPassed: disciple.promotion.assessmentPassed,
                      readinessText: disciple.promotion.readinessText,
                      blockerText: disciple.promotion.blockerText,
                      impactText: disciple.promotion.impactText,
                      recentFeedbackText: disciple.promotion.recentFeedbackText,
                  }
                : null,
        })),
        artifactCount: sect.artifactCount ?? 0,
        artifacts: (sect.artifacts ?? []).slice(0, 8).map(summarizeClientArtifact),
        artifactCraftOptions: (sect.artifactCraftOptions ?? []).slice(0, 6).map(summarizeClientArtifactCraftOption),
        candidateCount: sect.candidateCount ?? 0,
        candidates: (sect.candidates ?? []).slice(0, 6).map((candidate) => ({
            id: candidate.id,
            name: candidate.name,
            identity: candidate.identity,
            realmStage: candidate.realmStage,
            source: candidate.source,
        })),
        taskCount: sect.taskCount ?? 0,
        tasks: (sect.tasks ?? []).slice(0, 16).map((task) => ({
            id: task.id,
            kind: task.kind,
            taskType: task.taskType,
            typeLabel: task.typeLabel,
            gradeText: task.gradeText,
            title: task.title,
            status: task.status,
            priority: task.priority,
            risk: task.risk,
            riskText: task.riskText,
            injuryRiskText: task.injuryRiskText,
            successRate: task.successRate,
            successRateText: task.successRateText,
            evaluationText: task.evaluationText,
            maxAssignees: task.maxAssignees,
            progressText: task.progressText,
            recommendedDiscipleIds: task.recommendedDiscipleIds ?? [],
            recommendedDiscipleNames: task.recommendedDiscipleNames ?? [],
            assignedDiscipleNames: task.assignedDiscipleNames ?? [],
            teamRequirementText: task.teamRequirementText,
            dispatchCostText: task.dispatchCostText,
            rewardText: task.rewardText,
            possibleCostText: task.possibleCostText,
            reputationRewardText: task.reputationRewardText,
            relationRewardText: task.relationRewardText,
            crisisClueText: task.crisisClueText,
            canDispatch: task.canDispatch,
            dispatchBlockedReason: task.dispatchBlockedReason,
        })),
        productionCount: sect.productionCount ?? 0,
        productions: (sect.productions ?? []).slice(0, 8).map((production) => ({
            id: production.id,
            kind: production.kind,
            status: production.status,
            priority: production.priority,
            progressText: production.progressText,
            cycleText: production.cycleText,
            targetCycleText: production.targetCycleText,
            bottleneckText: production.bottleneckText,
        })),
        buildingSummary: sect.buildingSummary
            ? {
                  level: sect.buildingSummary.level,
                  expansion: sect.buildingSummary.expansion,
                  buildingLimit: sect.buildingSummary.buildingLimit,
                  damagedCount: sect.buildingSummary.damagedCount,
                  lowEfficiencyCount: sect.buildingSummary.lowEfficiencyCount,
                  maintenancePressureText: sect.buildingSummary.maintenancePressureText,
              }
            : null,
        buildingCatalog: (sect.buildingCatalog ?? []).slice(0, 8).map((entry) => ({
            definitionKey: entry.definitionKey,
            label: entry.label,
            currentCount: entry.currentCount,
            maxCount: entry.maxCount,
            unlocked: entry.unlocked,
            canBuild: entry.canBuild,
            blockerText: entry.blockerText,
            maintenanceText: entry.maintenanceText,
            instances: (entry.instances ?? []).map((instance) => ({
                id: instance.id,
                level: instance.level,
                phase: instance.phase,
                efficiency: instance.efficiency,
                durability: instance.durability,
                maintenanceDebt: instance.maintenanceDebt,
                damagedReason: instance.damagedReason,
                })),
        })),
        overview: sect.overview
            ? {
                  staffingText: sect.overview.staffingText,
                  maintenanceText: sect.overview.maintenanceText,
                  materialText: sect.overview.materialText,
                  riskText: sect.overview.riskText,
                  satisfactionText: sect.overview.satisfactionText,
                  institutionEfficiencyText: sect.overview.institutionEfficiencyText,
                  adviceText: sect.overview.adviceText,
              }
            : null,
        institutionCount: sect.institutionCount ?? 0,
        institutions: (sect.institutions ?? []).slice(0, 8).map((institution) => ({
            id: institution.id,
            label: institution.label,
            kind: institution.kind,
            level: institution.level,
            enabled: institution.enabled,
            managerText: institution.managerText,
            managerEffectText: institution.managerEffectText,
            capacityText: institution.capacityText,
            efficiencyText: institution.efficiencyText,
            bottleneckText: institution.bottleneckText,
            effectSummaryText: institution.effectSummaryText,
            recentResultText: institution.recentResultText,
            operationHintText: institution.operationHintText,
            gatePolicyText: institution.gatePolicyText,
            exchangeText: institution.exchangeText,
            caveSlotText: institution.caveSlotText,
        })),
        cultivation: sect.cultivation
            ? {
                  discipleId: sect.cultivation.discipleId,
                  discipleName: sect.cultivation.discipleName,
                  realmStage: sect.cultivation.realmStage,
                  assignmentKind: sect.cultivation.assignmentKind,
                  cultivationProgressText: sect.cultivation.cultivationProgressText,
                  breakthroughText: sect.cultivation.breakthroughText,
                  omenText: sect.cultivation.omenText,
                  pressureText: sect.cultivation.pressureText,
              }
            : null,
        omenCount: sect.omenCount ?? 0,
        eventChoiceCount: sect.eventChoiceCount ?? 0,
        eventChoices: (sect.eventChoices ?? []).slice(0, 6).map(summarizeActiveSectEvent),
        resolvedEventCount: sect.resolvedEventCount ?? 0,
        resolvedEvents: (sect.resolvedEvents ?? []).slice(-8).map(summarizeResolvedSectEvent),
        diaryCount: sect.diaryCount ?? 0,
        diary: diary.slice(0, 12).map((entry) => ({
            id: entry.id,
            eventType: entry.eventType,
            summary: entry.summary,
            version: entry.version,
            timingText: entry.timingText ?? '',
            storyText: entry.storyText ?? '',
        })),
        eventFeedbackCount: sect.eventFeedbackCount ?? 0,
        eventFeedback: eventFeedback.slice(0, 12).map((entry) => ({
            id: entry.id,
            category: entry.category,
            eventType: entry.eventType,
            summary: entry.summary,
            version: entry.version,
            timingText: entry.timingText ?? '',
            storyText: entry.storyText ?? '',
        })),
        eventFeedbackCategories: [...new Set(eventFeedback.map((entry) => entry.category))].sort(),
        policies: (sect.policies ?? []).map(summarizeClientPolicy),
        goalCount: sect.goalCount ?? 0,
        goals: (sect.goals ?? []).slice(0, 8).map(summarizeClientGoal),
        resolvedGoalCount: sect.resolvedGoalCount ?? 0,
        resolvedGoals: (sect.resolvedGoals ?? []).slice(0, 8).map(summarizeClientResolvedGoal),
        crisisCount: sect.crisisCount ?? 0,
        crises: (sect.crises ?? []).slice(0, 8).map(summarizeClientCrisis),
        treatmentCount: sect.treatmentCount ?? 0,
        treatments: (sect.treatments ?? []).slice(0, 8).map(summarizeClientTreatment),
        order: sect.order ? summarizeClientOrder(sect.order) : null,
        monthlyAssessment: sect.monthlyAssessment ? summarizeClientMonthlyAssessment(sect.monthlyAssessment) : null,
        recentPolicyEvents: recentPolicyEvents.slice(0, 6).map((entry) => ({
            id: entry.id,
            category: entry.category,
            eventType: entry.eventType,
            summary: entry.summary,
            version: entry.version,
        })),
        recentPromotionEvents: (sect.recentPromotionEvents ?? eventFeedback.filter((entry) => entry.category === 'promotion'))
            .slice(0, 6)
            .map((entry) => ({
                id: entry.id,
                category: entry.category,
                eventType: entry.eventType,
                summary: entry.summary,
                version: entry.version,
            })),
        lastError: sect.lastError ?? null,
        lastPatchFromVersion: sect.lastPatchFromVersion ?? null,
        lastPatchToVersion: sect.lastPatchToVersion ?? null,
    };
}

function summarizeClientPolicy(policy) {
    return {
        category: policy.category,
        value: policy.value,
        label: policy.label,
        impactSummary: policy.impactSummary ?? [],
        optionCount: (policy.options ?? []).length,
        options: (policy.options ?? []).map((option) => ({
            value: option.value,
            label: option.label,
            impactSummary: option.impactSummary ?? [],
        })),
    };
}

function summarizeClientArtifact(artifact) {
    return {
        id: artifact.id,
        type: artifact.type,
        typeLabel: artifact.typeLabel,
        slot: artifact.slot,
        slotLabel: artifact.slotLabel,
        quality: artifact.quality,
        durability: artifact.durability,
        maxDurability: artifact.maxDurability,
        durabilityText: artifact.durabilityText,
        boundDiscipleId: artifact.boundDiscipleId ?? null,
        boundDiscipleName: artifact.boundDiscipleName ?? '',
        statsText: artifact.statsText ?? '',
        repairNeedText: artifact.repairNeedText ?? '',
        canEquip: !!artifact.canEquip,
        canUnequip: !!artifact.canUnequip,
        canRepair: !!artifact.canRepair,
    };
}

function summarizeClientArtifactCraftOption(option) {
    return {
        type: option.type,
        typeLabel: option.typeLabel,
        slotLabel: option.slotLabel,
        materialText: option.materialText,
    };
}

function summarizeClientGoal(goal) {
    return {
        id: goal.id,
        kind: goal.kind,
        title: goal.title,
        status: goal.status,
        progressText: goal.progressText ?? '',
        rewardText: goal.rewardText ?? '',
        outcomeText: goal.outcomeText ?? '',
        focusDiscipleText: goal.focusDiscipleText ?? '',
        tagsText: goal.tagsText ?? '',
        recommendationText: goal.recommendationText ?? '',
    };
}

function summarizeClientResolvedGoal(goal) {
    return {
        id: goal.id,
        outcome: goal.outcome,
        summary: goal.summary,
        resolvedAtVersion: goal.resolvedAtVersion ?? null,
    };
}

function summarizeClientCrisis(crisis) {
    return {
        id: crisis.id,
        title: crisis.title,
        status: crisis.status,
        stage: crisis.stage,
        stageText: crisis.stageText ?? '',
        severity: crisis.severity ?? 0,
        sourceText: crisis.sourceText ?? '',
        detail: crisis.detail ?? '',
        expiresText: crisis.expiresText ?? '',
        previewText: crisis.previewText ?? '',
        optionSummaryText: crisis.optionSummaryText ?? '',
        tagsText: crisis.tagsText ?? '',
        recommendationText: crisis.recommendationText ?? '',
    };
}

function summarizeClientTreatment(entry) {
    return {
        discipleId: entry.discipleId,
        discipleName: entry.discipleName,
        queueText: entry.queueText ?? '',
        injuryText: entry.injuryText ?? '',
        hpText: entry.hpText ?? '',
        pressureText: entry.pressureText ?? '',
        recoveryText: entry.recoveryText ?? '',
        herbCostText: entry.herbCostText ?? '',
        institutionText: entry.institutionText ?? '',
        recommendationText: entry.recommendationText ?? '',
    };
}

function summarizeClientOrder(order) {
    return {
        safetyText: order.safetyText ?? '',
        disciplineText: order.disciplineText ?? '',
        strifeRiskText: order.strifeRiskText ?? '',
        summaryText: order.summaryText ?? '',
        recommendationText: order.recommendationText ?? '',
    };
}

function summarizeClientMonthlyAssessment(assessment) {
    return {
        latestMonthText: assessment.latestMonthText ?? '',
        latestChampionText: assessment.latestChampionText ?? '',
        latestRewardText: assessment.latestRewardText ?? '',
        latestSummaryText: assessment.latestSummaryText ?? '',
        historyText: assessment.historyText ?? '',
        recommendationText: assessment.recommendationText ?? '',
    };
}

function summarizeRawPolicyState(policyState) {
    const categories = policyState?.presentation?.categories ?? {};
    return {
        taskPolicy: policyState?.task_policy ?? null,
        resourcePolicy: policyState?.resource_policy ?? null,
        recruitmentPolicy: policyState?.recruitment_policy ?? null,
        cultivationPolicy: policyState?.cultivation_policy ?? null,
        categories: Object.fromEntries(
            Object.entries(categories).map(([category, entry]) => [
                category,
                {
                    currentValue: entry.current_value,
                    currentLabel: entry.current_label,
                    impactSummary: entry.impact_summary ?? [],
                    optionCount: (entry.options ?? []).length,
                },
            ]),
        ),
    };
}

function summarizeRawBuilding(building) {
    return {
        id: building.building_id,
        definitionKey: building.definition_key,
        level: building.level,
        phase: building.phase,
        origin: building.origin ?? null,
        hp: building.hp,
        maxHp: building.max_hp,
        efficiency: building.efficiency,
        durability: building.durability,
        maintenanceDebt: building.maintenance_debt,
        damagedReason: building.damaged_reason ?? '',
    };
}

function summarizeRawArtifact(artifact) {
    return {
        id: artifact.item_id,
        type: artifact.type,
        quality: artifact.quality,
        durability: artifact.durability,
        maxDurability: artifact.max_durability,
        boundDiscipleId: artifact.bound_disciple_id ?? null,
        stats: artifact.stats ?? {},
    };
}

function summarizeRawFormation(formation) {
    return {
        formationId: formation.formation_id,
        kind: formation.kind,
        buildingId: formation.building_id,
        artifactItemId: formation.artifact_item_id,
        level: formation.level,
        stability: formation.stability,
        maintenanceDebt: formation.maintenance_debt,
        active: !!formation.active,
        effectSummary: formation.effect_summary ?? [],
    };
}

function summarizeRawGoal(goal) {
    return {
        id: goal.goal_id,
        kind: goal.kind,
        title: goal.title,
        status: goal.status,
        currentProgress: goal.current_progress ?? 0,
        targetProgress: goal.target_progress ?? 0,
        progressText: goal.progress_text ?? '',
        rewardSummary: goal.reward_summary ?? [],
        outcomeSummary: goal.outcome_summary ?? '',
        tags: goal.tags ?? [],
        resolvedAtVersion: goal.resolved_at_version ?? 0,
    };
}

function summarizeRawResolvedGoal(goal) {
    return {
        id: goal.goal_id,
        kind: goal.kind,
        outcome: goal.outcome,
        summary: goal.summary,
        resolvedAtVersion: goal.resolved_at_version ?? 0,
    };
}

function summarizeRawBuildingCatalogEntry(entry) {
    return {
        definitionKey: entry.definition_key,
        label: entry.label,
        maxLevel: entry.max_level,
        unlockSectLevel: entry.unlock_sect_level,
        requiredMainHallLevel: entry.required_main_hall_level ?? 0,
        maxCount: entry.max_count,
        currentCount: entry.current_count,
        buildCost: entry.build_cost ?? {},
        upgradeCostByLevel: entry.upgrade_cost_by_level ?? {},
        maintenanceByLevel: entry.maintenance_by_level ?? {},
        unlocked: !!entry.unlocked,
        canBuild: !!entry.can_build,
        blockers: entry.blockers ?? [],
        existingBuildings: (entry.existing_buildings ?? []).map((building) => ({
            id: building.building_id,
            level: building.level,
            phase: building.phase,
            efficiency: building.efficiency,
            durability: building.durability,
            maintenanceDebt: building.maintenance_debt,
            damagedReason: building.damaged_reason ?? '',
        })),
    };
}

function summarizeRawInstitution(institution) {
    return {
        id: institution.institution_id,
        kind: institution.kind,
        level: institution.level,
        enabled: !!institution.enabled,
        managerDiscipleId: institution.manager_disciple_id ?? null,
        managerEffect: institution.manager_effect
            ? {
                  managerScore: institution.manager_effect.manager_score ?? 0,
                  efficiencyBonus: institution.manager_effect.efficiency_bonus ?? 0,
              }
            : null,
        capacity: institution.capacity ?? 0,
        comfort: institution.comfort ?? 0,
        healingPower: institution.healing_power ?? 0,
        cultivationSupport: institution.cultivation_support ?? 0,
        taskCapacityBonus: institution.task_capacity_bonus ?? 0,
        exchangePressure: institution.exchange_pressure ?? 0,
        efficiency: institution.efficiency ?? 0,
        effectSummary: institution.effect_summary ?? [],
        gatePolicy: institution.gate_policy
            ? {
                  openToVisitors: !!institution.gate_policy.open_to_visitors,
                  allowWanderingCultivators: !!institution.gate_policy.allow_wandering_cultivators,
                  guardDiscipleCount: (institution.gate_policy.guard_disciple_ids ?? []).length,
              }
            : null,
        publicExchange: !!institution.public_exchange_enabled,
        caveSlots: (institution.cave_slots ?? []).map((slot) => ({
            slotId: slot.slot_id,
            occupiedBy: slot.occupied_by ?? null,
            reservedUntilDay: slot.reserved_until_day ?? 0,
            environmentBonus: slot.environment_bonus ?? 0,
        })),
    };
}

function recordValues(record) {
    return Object.values(record ?? {});
}

function summarizeEventOption(option) {
    return {
        id: option.id ?? option.option_id,
        label: option.label,
        description: option.description,
        requirementsText: option.requirementsText,
        resultPreviewText: option.resultPreviewText ?? option.previewText ?? option.result_preview?.summary,
    };
}

function summarizeActiveSectEvent(event) {
    return {
        id: event.id ?? event.event_id,
        kind: event.kind,
        title: event.title,
        status: event.status,
        severity: event.severity,
        expiresAtDay: event.expiresAtDay ?? event.expires_at_day,
        expiresText: event.expiresText,
        optionCount: (event.options ?? []).length,
        options: (event.options ?? []).slice(0, 4).map(summarizeEventOption),
        resultPreviewText: event.resultPreviewText ?? event.previewText ?? event.result_preview?.summary,
    };
}

function summarizeResolvedSectEvent(event) {
    return {
        id: event.id ?? event.event_id,
        kind: event.kind,
        outcome: event.outcome,
        summary: event.summary,
        resolvedAtVersion: event.resolvedAtVersion ?? event.resolved_at_version,
    };
}

function summarizeRawSectSnapshot(snapshot) {
    const state = snapshot?.state ?? {};
    const disciples = recordValues(state.disciples).sort((left, right) =>
        String(left.disciple_id ?? '').localeCompare(String(right.disciple_id ?? '')),
    );
    const artifacts = recordValues(state.inventory?.artifacts).sort((left, right) =>
        String(left.item_id ?? '').localeCompare(String(right.item_id ?? '')),
    );
    const formations = recordValues(state.formations).sort((left, right) =>
        String(left.formation_id ?? '').localeCompare(String(right.formation_id ?? '')),
    );
    const candidates = recordValues(state.admissions?.candidates).sort((left, right) =>
        String(left.candidate_id ?? '').localeCompare(String(right.candidate_id ?? '')),
    );
    const tasks = recordValues(state.tasks).sort((left, right) =>
        String(left.task_id ?? '').localeCompare(String(right.task_id ?? '')),
    );
    const taskDispatch = snapshot?.taskDispatch ?? {};
    const productions = recordValues(state.productions).sort((left, right) =>
        String(left.production_id ?? '').localeCompare(String(right.production_id ?? '')),
    );
    const institutions = recordValues(state.institutions?.by_id).sort((left, right) =>
        String(left.institution_id ?? '').localeCompare(String(right.institution_id ?? '')),
    );
    const buildings = recordValues(state.buildings).sort((left, right) =>
        String(left.building_id ?? '').localeCompare(String(right.building_id ?? '')),
    );
    const buildingCatalog = [...(snapshot?.buildingCatalog ?? [])].sort((left, right) =>
        String(left.definition_key ?? '').localeCompare(String(right.definition_key ?? '')),
    );
    const activeEvents = recordValues(state.events?.active_events).sort((left, right) =>
        String(left.event_id ?? '').localeCompare(String(right.event_id ?? '')),
    );
    const eventChoices = activeEvents.filter((event) => (event.options ?? []).length > 0);
    const crises = activeEvents.filter((event) => String(event.chain_stage ?? '').length > 0);
    const resolvedEvents = [...(state.events?.resolved_events ?? [])].sort((left, right) =>
        String(left.event_id ?? '').localeCompare(String(right.event_id ?? '')),
    );
    const goals = recordValues(state.goals?.by_id).sort((left, right) =>
        String(left.goal_id ?? '').localeCompare(String(right.goal_id ?? '')),
    );
    const resolvedGoals = [...(state.goals?.resolved ?? [])].sort((left, right) =>
        String(left.goal_id ?? '').localeCompare(String(right.goal_id ?? '')),
    );
    const feedback = snapshot?.eventSummaries ?? [];
    const recentPolicyEvents = feedback.filter((entry) => entry.category === 'policy');
    const recentPromotionEvents = feedback.filter((entry) => entry.category === 'promotion');
    const primaryDisciple = disciples.find((disciple) => disciple.disciple_id === 'disciple-1') ?? disciples[0] ?? null;
    const treatments = disciples
        .filter((disciple) => Number(disciple.injury_level ?? 0) > 0 || Number(disciple.hp ?? 0) < Number(disciple.max_hp ?? 0))
        .map((disciple) => ({
            discipleId: disciple.disciple_id,
            discipleName: disciple.name,
            injuryLevel: disciple.injury_level ?? 0,
            hp: disciple.hp ?? 0,
            maxHp: disciple.max_hp ?? 0,
            pressure: disciple.pressure ?? 0,
        }));
    return {
        connected: !!snapshot,
        sectId: snapshot?.sectId ?? null,
        userId: snapshot?.userId ?? null,
        sessionId: snapshot?.sessionId ?? null,
        sceneVersion: snapshot?.sceneVersion ?? 0,
        calendarDay: state.time?.calendar_day ?? 0,
        meta: {
            level: state.meta?.level ?? 0,
            expansion: state.meta?.expansion ?? 0,
            buildingLimit: state.meta?.building_limit ?? 0,
        },
        resources: state.resources?.stock ?? {},
        buildingCount: buildings.length,
        buildings: buildings.map(summarizeRawBuilding),
        buildingCatalog: buildingCatalog.map(summarizeRawBuildingCatalogEntry),
        rosterCount: disciples.length,
        roster: disciples.slice(0, 6).map((disciple) => ({
            id: disciple.disciple_id,
            name: disciple.name,
            identity: disciple.identity,
            realmStage: disciple.realm?.stage,
            assignmentKind: disciple.assignment_kind,
            workTargetText:
                disciple.work_target?.description ??
                disciple.work_target?.task_id ??
                disciple.work_target?.building_id ??
                '无',
            contributionBalance: state.contribution?.accounts?.[disciple.disciple_id]?.balance ?? 0,
            hp: disciple.hp,
            maxHp: disciple.max_hp,
            satisfaction: disciple.satisfaction,
            loyalty: disciple.loyalty,
            relationshipTags: disciple.relationship_tags ?? [],
            emotionTags: disciple.emotion_tags ?? [],
            recentExperienceSummary: disciple.recent_experience_summary ?? [],
            memorySummaries: (disciple.memories ?? []).slice(0, 4).map((entry) => entry.summary ?? ''),
            equipment: disciple.equipment ?? {},
            assessment: disciple.assessment
                ? {
                      targetRank: disciple.assessment.target_rank,
                      passed: disciple.assessment.passed,
                      score: disciple.assessment.score,
                      reason: disciple.assessment.reason,
                      resolvedAtVersion: disciple.assessment.resolved_at_version,
                  }
                : null,
        })),
        artifactCount: artifacts.length,
        artifacts: artifacts.slice(0, 12).map(summarizeRawArtifact),
        formationCount: formations.length,
        formations: formations.slice(0, 8).map(summarizeRawFormation),
        candidateCount: candidates.length,
        candidates: candidates.slice(0, 6).map((candidate) => ({
            id: candidate.candidate_id,
            name: candidate.name,
            identity: candidate.identity,
            realmStage: candidate.realm?.stage,
            source: candidate.source,
        })),
        taskCount: tasks.length,
        tasks: tasks.slice(0, 16).map((task) => ({
            id: task.task_id,
            kind: task.kind,
            taskType: task.type,
            grade: task.grade,
            title: task.title,
            status: task.status,
            priority: task.priority,
            completedProgressDays: task.completed_progress_days,
            requiredProgressDays: task.required_progress_days,
            assignedDiscipleIds: task.assigned_disciple_ids ?? [],
            risk: task.risk,
            maxAssignees: task.max_assignees,
            dispatchCost: task.dispatch_cost ?? {},
            contributionReward: task.contribution_reward,
            rewardResources: task.reward_resources ?? {},
            reputationReward: task.reputation_reward ?? 0,
            relationReward: task.relation_reward ?? {},
            crisisClue: task.crisis_clue ?? '',
            successRate: task.success_rate ?? null,
            evaluation: task.evaluation ?? null,
            recommendedDispatch: taskDispatch[task.task_id]
                ? {
                      taskId: taskDispatch[task.task_id].task_id,
                      recommendedDiscipleIds: taskDispatch[task.task_id].recommended_disciple_ids ?? [],
                      recommendedSuccessRate: taskDispatch[task.task_id].recommended_success_rate ?? null,
                      blockedReason: taskDispatch[task.task_id].blocked_reason ?? '',
                  }
                : null,
        })),
        productionCount: productions.length,
        productions: productions.slice(0, 8).map((production) => ({
            id: production.production_id,
            kind: production.kind,
            status: production.status,
            priority: production.priority,
            progressDays: production.progress_days,
            requiredProgressDays: production.required_progress_days,
            completedCycles: production.completed_cycles,
            targetCycles: production.target_cycles,
            blockedReason: production.blocked_reason,
        })),
        institutionCount: institutions.length,
        institutions: institutions.map(summarizeRawInstitution),
        policies: summarizeRawPolicyState(state.policies),
        goalCount: goals.length,
        goals: goals.slice(0, 8).map(summarizeRawGoal),
        resolvedGoalCount: resolvedGoals.length,
        resolvedGoals: resolvedGoals.slice(0, 8).map(summarizeRawResolvedGoal),
        cultivation: primaryDisciple
            ? {
                  discipleId: primaryDisciple.disciple_id,
                  discipleName: primaryDisciple.name,
                  realmStage: primaryDisciple.realm?.stage,
                  assignmentKind: primaryDisciple.assignment_kind,
                  cultivationPoints: primaryDisciple.realm?.cultivation_points,
                  readyForBreakthrough: primaryDisciple.realm?.ready_for_breakthrough,
                  breakthroughSuccessRate: primaryDisciple.cultivation_decision?.breakthrough_success_rate,
                  omenStatus: primaryDisciple.cultivation_decision?.omen_status,
                  pressure: primaryDisciple.pressure,
                  injuryLevel: primaryDisciple.injury_level,
              }
            : null,
        monthly: {
            lastSettledMonth: state.monthly?.last_settled_month ?? 0,
            lastSettlement: state.monthly?.last_settlement ?? null,
        },
        omenCount: activeEvents.length,
        eventTension: state.events?.tension ?? 0,
        crisisCount: crises.length,
        crises: crises.slice(0, 8).map(summarizeActiveSectEvent),
        eventChoiceCount: eventChoices.length,
        eventChoices: eventChoices.slice(0, 6).map(summarizeActiveSectEvent),
        resolvedEventCount: resolvedEvents.length,
        resolvedEvents: resolvedEvents.slice(-8).map(summarizeResolvedSectEvent),
        treatmentCount: treatments.length,
        treatments,
        order: {
            safety: state.order?.safety ?? 0,
            discipline: state.order?.discipline ?? 0,
            internalStrifeRisk: state.order?.internal_strife_risk ?? 0,
            summary: state.order?.summary ?? [],
            lastUpdatedVersion: state.order?.last_updated_version ?? 0,
        },
        monthlyAssessment: {
            lastMonthIndex: state.monthly_assessment?.last_month_index ?? 0,
            latest: state.monthly_assessment?.latest ?? null,
            historyCount: (state.monthly_assessment?.history ?? []).length,
        },
        diaryCount: snapshot?.diary?.length ?? 0,
        diary: (snapshot?.diary ?? []).slice(0, 12).map((entry) => ({
            id: entry.event_id,
            eventType: entry.event_type,
            summary: entry.summary,
            version: entry.version,
            relatedDay: entry.related_day,
            relatedTick: entry.related_tick,
            replaySource: entry.replay_source,
        })),
        eventFeedbackCount: feedback.length,
        eventFeedback: feedback.slice(0, 12).map((entry) => ({
            id: entry.event_id,
            category: entry.category,
            eventType: entry.event_type,
            summary: entry.summary,
            version: entry.version,
        })),
        eventFeedbackCategories: [...new Set(feedback.map((entry) => entry.category))].sort(),
        recentPolicyEvents: recentPolicyEvents.slice(0, 6).map((entry) => ({
            id: entry.event_id,
            category: entry.category,
            eventType: entry.event_type,
            summary: entry.summary,
            version: entry.version,
        })),
        recentPromotionEvents: recentPromotionEvents.slice(0, 6).map((entry) => ({
            id: entry.event_id,
            category: entry.category,
            eventType: entry.event_type,
            summary: entry.summary,
            version: entry.version,
        })),
    };
}

function hasReadableText(value, blockedFragments = []) {
    const text = typeof value === 'string' ? value.trim() : '';
    if (!text) {
        return false;
    }
    return !blockedFragments.some((fragment) => text.includes(fragment));
}

function countReadableRosterStoryEntries(roster) {
    return (roster ?? []).filter((disciple) =>
        hasReadableText(disciple.recentExperienceText, ['暂无', 'authority 未声明']) ||
        hasReadableText(disciple.memoryText, ['未记录', 'authority 未声明']) ||
        hasReadableText(disciple.moodReasonText, ['authority 未提供']) ||
        hasReadableText(disciple.relationshipText, ['关系平稳']) ||
        hasReadableText(disciple.emotionText, ['情绪平稳']),
    ).length;
}

function summarizeSnapshot(snapshot) {
    const guardTowerId = snapshot.session.guardTowerId;
    const ruinBuildingId = snapshot.session.ruinBuildingId;
    return {
        authority: {
            mode: snapshot.authority.mode,
            connected: snapshot.authority.connected,
            sessionId: snapshot.authority.sessionId,
            gameTick: snapshot.authority.gameTick,
            renderSource: snapshot.authority.renderSource,
            lastError: snapshot.authority.lastError,
            lastEvent: snapshot.authority.lastEvent,
        },
        session: {
            phase: snapshot.session.phase,
            outcome: snapshot.session.outcome,
            objective: snapshot.session.objective,
            elapsedSeconds: snapshot.session.elapsedSeconds,
            raidCountdownSeconds: snapshot.session.raidCountdownSeconds,
            defendRemainingSeconds: snapshot.session.defendRemainingSeconds,
            guardTowerId,
            ruinBuildingId,
            firstRaidTriggered: snapshot.session.firstRaidTriggered,
            firstRaidResolved: snapshot.session.firstRaidResolved,
            recoverReason: snapshot.session.recoverReason,
            damagedBuildingCount: snapshot.session.damagedBuildingCount,
            regeneratingNodeCount: snapshot.session.regeneratingNodeCount,
            riskIntensity: snapshot.session.riskIntensity,
            riskMitigation: snapshot.session.riskMitigation,
            threatCurve: snapshot.session.threatCurve,
            defenseRating: snapshot.session.defenseRating,
            guardDiscipleCount: snapshot.session.guardDiscipleCount,
            omenStatus: snapshot.session.omenStatus,
            omenText: snapshot.session.omenText,
            defenseSummary: snapshot.session.defenseSummary,
            damageSummary: snapshot.session.damageSummary,
            repairSuggestion: snapshot.session.repairSuggestion,
            sourceSummary: (snapshot.session.sourceSummary ?? []).map((source) => ({
                source: source.source,
                label: source.label,
                delta: source.delta,
            })),
        },
        stockpile: snapshot.stockpile,
        disciple: {
            tile: snapshot.disciple.tile,
            visualState: snapshot.disciple.visualState,
            carrying: snapshot.disciple.carrying,
            task: snapshot.disciple.task,
        },
        hostiles: snapshot.hostiles
            .filter((hostile) => hostile.active)
            .map((hostile) => ({
                id: hostile.id,
                tile: hostile.tile,
                renderPosition: hostile.renderPosition ?? null,
                interpolationProgress: hostile.interpolationProgress ?? null,
                interpolationTargetTile: hostile.interpolationTargetTile ?? null,
                hp: hostile.hp,
                maxHp: hostile.maxHp,
                targetBuildingId: hostile.targetBuildingId,
                visualState: hostile.visualState,
            })),
        damageFloaters: (snapshot.damageFloaters ?? []).map((floater) => ({
            id: floater.id,
            targetKind: floater.targetKind,
            targetId: floater.targetId,
            targetName: floater.targetName,
            amount: floater.amount,
            remainingSeconds: floater.remainingSeconds,
            position: floater.position,
        })),
        buildings: snapshot.buildings
            .map((building) => ({
                id: building.id,
                type: building.type,
                state: building.state,
                level: building.level,
                origin: building.origin,
                hp: building.hp,
                maxHp: building.maxHp,
                markedForDemolition: building.markedForDemolition,
                pendingAction: building.pendingAction,
                pendingLevel: building.pendingLevel,
                supplied: building.supplied,
            }))
            .sort((left, right) => left.id.localeCompare(right.id)),
        sectAuthority: summarizeSectAuthority(snapshot),
    };
}

function normalizeRestoreComparable(summary) {
    return {
        session: {
            phase: summary.session.phase,
            outcome: summary.session.outcome,
            objective: summary.session.objective,
            guardTowerId: summary.session.guardTowerId,
            ruinBuildingId: summary.session.ruinBuildingId,
            firstRaidTriggered: summary.session.firstRaidTriggered,
            firstRaidResolved: summary.session.firstRaidResolved,
            recoverReason: summary.session.recoverReason,
            damagedBuildingCount: summary.session.damagedBuildingCount,
            regeneratingNodeCount: summary.session.regeneratingNodeCount,
            riskIntensity: summary.session.riskIntensity,
            riskMitigation: summary.session.riskMitigation,
            threatCurve: summary.session.threatCurve,
            defenseRating: summary.session.defenseRating,
            guardDiscipleCount: summary.session.guardDiscipleCount,
            omenStatus: summary.session.omenStatus,
            omenText: summary.session.omenText,
            defenseSummary: summary.session.defenseSummary,
            damageSummary: summary.session.damageSummary,
            repairSuggestion: summary.session.repairSuggestion,
            sourceSummary: summary.session.sourceSummary,
        },
        stockpile: summary.stockpile,
        disciple: {
            tile: summary.disciple.tile,
            carrying: summary.disciple.carrying,
            task: summary.disciple.task,
        },
        buildings: summary.buildings,
    };
}

function snapshotsMatchForRestore(before, after) {
    return JSON.stringify(normalizeRestoreComparable(before)) === JSON.stringify(normalizeRestoreComparable(after));
}

function requireCondition(condition, message, details = undefined) {
    if (!condition) {
        const error = new Error(message);
        if (details !== undefined) {
            error.details = details;
        }
        throw error;
    }
}

function parseTileString(value) {
    const match = String(value ?? '').trim().match(/^(\d+)\s*,\s*(\d+)$/);
    if (!match) {
        return null;
    }
    return {
        col: Number(match[1]),
        row: Number(match[2]),
    };
}

class CdpConnection {
    constructor(wsUrl) {
        this.wsUrl = wsUrl;
        this.ws = null;
        this.nextId = 1;
        this.pending = new Map();
    }

    async connect() {
        await new Promise((resolve, reject) => {
            const ws = new WebSocket(this.wsUrl);
            this.ws = ws;
            ws.addEventListener('open', () => resolve());
            ws.addEventListener('error', (event) => reject(event.error || new Error('CDP websocket connect failed.')));
            ws.addEventListener('message', (event) => this.#handleMessage(event.data));
            ws.addEventListener('close', () => {
                for (const pending of this.pending.values()) {
                    pending.reject(new Error('CDP websocket closed.'));
                }
                this.pending.clear();
            });
        });
    }

    async close() {
        if (!this.ws) {
            return;
        }
        await new Promise((resolve) => {
            this.ws.addEventListener('close', () => resolve(), { once: true });
            this.ws.close();
        });
    }

    async send(method, params = {}) {
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
            throw new Error(`CDP websocket is not open for ${method}`);
        }

        const id = this.nextId += 1;
        const payload = { id, method, params };
        const promise = new Promise((resolve, reject) => {
            this.pending.set(id, { resolve, reject });
        });
        this.ws.send(JSON.stringify(payload));
        return promise;
    }

    async evaluate(expression) {
        const result = await this.send('Runtime.evaluate', {
            expression,
            awaitPromise: true,
            returnByValue: true,
        });
        if (result.exceptionDetails) {
            const description =
                result.exceptionDetails.exception?.description ||
                result.exceptionDetails.text ||
                'Runtime.evaluate failed.';
            throw new Error(description);
        }
        return result.result?.value;
    }

    #handleMessage(raw) {
        const message = JSON.parse(typeof raw === 'string' ? raw : raw.toString('utf8'));
        if (!Object.prototype.hasOwnProperty.call(message, 'id')) {
            return;
        }
        const pending = this.pending.get(message.id);
        if (!pending) {
            return;
        }
        this.pending.delete(message.id);
        if (message.error) {
            pending.reject(new Error(message.error.message || `CDP error for request ${message.id}`));
            return;
        }
        pending.resolve(message.result);
    }
}

async function fetchJson(url, init = undefined) {
    const response = await fetch(url, init);
    if (!response.ok) {
        throw new Error(`HTTP ${response.status} for ${url}`);
    }
    return response.json();
}

async function postJson(url, payload) {
    return fetchJson(url, {
        method: 'POST',
        headers: {
            'content-type': 'application/json',
        },
        body: JSON.stringify(payload),
    });
}

async function submitSectCommandHttp(options, context, command) {
    return postJson(`${options.authorityUrl}/v1/authority/sect/command`, {
        userId: context.userId,
        sectId: context.sectId,
        sessionId: context.sessionId,
        command,
    });
}

async function findOrOpenPreviewTarget(options) {
    const listUrl = `${options.browserUrl}/json/list`;
    const targets = await fetchJson(listUrl);
    const existing = targets
        .filter((target) => target.type === 'page' && String(target.url || '').startsWith(options.previewUrl))
        .at(-1);
    if (existing) {
        return existing;
    }
    requireCondition(
        options.openTabIfMissing,
        `No dedicated browser page matched ${options.previewUrl}; open the preview tab first or omit --reuse-only.`,
    );
    return fetchJson(`${options.browserUrl}/json/new?${encodeURIComponent(options.previewUrl)}`);
}

function runtimeCallExpression(method, args = []) {
    const serializedArgs = args.map((value) => JSON.stringify(value)).join(', ');
    return `(async () => {
  const runtime = globalThis.__MIS_RUNTIME_DEBUG__;
  if (!runtime) {
    throw new Error('__MIS_RUNTIME_DEBUG__ is missing on the dedicated preview page.');
  }
  const fn = runtime[${JSON.stringify(method)}];
  if (typeof fn !== 'function') {
    throw new Error(${JSON.stringify(`__MIS_RUNTIME_DEBUG__.${method} is missing.`)});
  }
  return await fn(${serializedArgs});
})()`;
}

function appProbeExpression(probe, args = []) {
    const serializedArgs = args.map((value) => JSON.stringify(value)).join(', ');
    const callArgs = serializedArgs ? `app, cc, ${serializedArgs}` : 'app, cc';
    return `(async () => {
  const cc = globalThis.cc;
  const scene = cc?.director?.getScene?.();
  let app = null;
  const visit = (node) => {
    if (!node || app) {
      return;
    }
    const components = typeof node.getComponents === 'function' ? node.getComponents('cc.Component') : (node.components || []);
    for (const component of components || []) {
      if (
        typeof component.syncAuthorityRenderViewModels === 'function' &&
        typeof component.syncAuthorityHostilesFromSnapshot === 'function'
      ) {
        app = component;
        return;
      }
    }
    for (const child of node.children || []) {
      visit(child);
    }
  };
  visit(scene);
  if (!app) {
    throw new Error('SectMapBootstrap component was not found on the dedicated preview page.');
  }
  return await (${probe.toString()})(${callArgs});
})()`;
}

async function waitForRuntimeSnapshot(connection, options) {
    const startedAt = Date.now();
    let lastError = null;
    while (Date.now() - startedAt < options.timeoutMs) {
        try {
            const snapshot = await connection.evaluate(runtimeCallExpression('getSnapshot'));
            if (snapshot?.mapReady) {
                return snapshot;
            }
        } catch (error) {
            lastError = error;
        }
        await delay(options.pollMs);
    }
    throw new Error(`Timed out waiting for __MIS_RUNTIME_DEBUG__.getSnapshot(); last error: ${lastError instanceof Error ? lastError.message : 'none'}`);
}

async function reloadPreviewPage(connection) {
    try {
        await connection.send('Page.reload', { ignoreCache: true });
    } catch {
        await connection.evaluate(`(() => {
  globalThis.location.reload();
  return 'reloading';
})()`);
    }
    await delay(2500);
}

async function joinSectAuthority(connection) {
    return connection.evaluate(runtimeCallExpression('joinSectAuthority'));
}

function sectCommand(cmdId, type, baseVersion, payload) {
    return {
        cmdId,
        type,
        baseVersion,
        payload,
    };
}

async function executeSectAuthorityCommand(connection, command) {
    return connection.evaluate(
        runtimeCallExpression('executeSectAuthorityCommand', [
            command,
            {
                commandKey: `gate:${command.cmdId}`,
            },
        ]),
    );
}

async function advanceSectDays(options, snapshot, days) {
    const sect = snapshot.sectAuthority ?? {};
    requireCondition(sect.connected && sect.sectId, 'Sect authority snapshot is not connected before compressed day advance.', sect);
    return postJson(`${options.authorityUrl}/v1/authority/sect/debug/advance-days`, {
        userId: snapshot.authority?.playerId ?? 'preview-player',
        sectId: sect.sectId,
        sessionId: snapshot.authority?.sessionId ?? 'preview-player/preview-local',
        days,
    });
}

async function resetSectAuthority(options, snapshot, overrides = {}) {
    const userId = snapshot.authority?.playerId ?? 'preview-player';
    return postJson(`${options.authorityUrl}/v1/authority/sect/debug/reset`, {
        userId,
        sectId: `sect-${userId}`,
        sessionId: snapshot.authority?.sessionId ?? 'preview-player/preview-local',
        ...overrides,
    });
}

async function fetchLatestAuthoritySnapshot(connection) {
    return connection.evaluate(runtimeCallExpression('fetchAuthoritySnapshot'));
}

async function waitForCheckpoint(label, connection, options, predicate) {
    const startedAt = Date.now();
    let lastSnapshot = null;
    while (Date.now() - startedAt < options.timeoutMs) {
        const snapshot = await fetchLatestAuthoritySnapshot(connection);
        lastSnapshot = snapshot;
        if (predicate(snapshot)) {
            return snapshot;
        }
        await delay(options.pollMs);
    }
    throw new Error(`Timed out waiting for checkpoint ${label}. Last phase: ${lastSnapshot?.session?.phase ?? 'unknown'}`);
}

function findBuilding(snapshot, buildingId) {
    return snapshot.buildings.find((building) => building.id === buildingId) ?? null;
}

function findGuardTower(snapshot) {
    const guardTowerId = snapshot.session.guardTowerId;
    if (guardTowerId) {
        return findBuilding(snapshot, guardTowerId);
    }
    return snapshot.buildings.find((building) => building.type === 'guard_tower') ?? null;
}

function findResourceNode(snapshot, tile) {
    const tileText = `${tile.col},${tile.row}`;
    return snapshot.resourceNodes.find((resource) => resource.tile === tileText) ?? null;
}

function pickDesignationProbeResource(snapshot) {
    return snapshot.resourceNodes.find((resource) => resource.designated) ?? snapshot.resourceNodes[0] ?? null;
}

async function runPlayerFacingControlsProbe(connection) {
    return connection.evaluate(
        appProbeExpression((app, cc) => {
            const nodeEntries = [
                ['newGame', app.sessionActionRoot?.getChildByName('NewGameButton')],
                ['taskHall', app.sessionActionRoot?.getChildByName('TaskHallButton')],
                ['statusDetail', app.sessionActionRoot?.getChildByName('StatusDetailButton')],
            ];

            const inspectNode = ([key, node]) => {
                if (!node) {
                    return {
                        key,
                        exists: false,
                        activeInHierarchy: false,
                        selfHit: false,
                        blockedByPanel: false,
                        overlappedByStatusHud: false,
                        center: null,
                    };
                }
                const center = node.getComponent(cc.UITransform)?.convertToWorldSpaceAR(cc.v3(0, 0, 0)) ?? node.worldPosition;
                return {
                    key,
                    exists: true,
                    activeInHierarchy: node.activeInHierarchy,
                    selfHit: typeof app.isPointInsideNode === 'function' ? app.isPointInsideNode(node, center) : true,
                    blockedByPanel: typeof app.isLocationBlockedByScreenUI === 'function' ? app.isLocationBlockedByScreenUI(center) : false,
                    overlappedByStatusHud:
                        !!app.statusLabel?.node &&
                        typeof app.isPointInsideNode === 'function' &&
                        app.isPointInsideNode(app.statusLabel.node, center),
                    center: {
                        x: Number(center.x.toFixed(2)),
                        y: Number(center.y.toFixed(2)),
                    },
                };
            };

            const buttons = nodeEntries.map(inspectNode);
            const taskHallNode = nodeEntries.find(([key]) => key === 'taskHall')?.[1] ?? null;
            const beforeSectPanelActive = !!app.sectPanelRoot?.active;
            taskHallNode?.emit(cc.Node.EventType.MOUSE_UP);
            const afterSectPanelActive = !!app.sectPanelRoot?.active;
            if (afterSectPanelActive !== beforeSectPanelActive) {
                app.closeSectPanel?.();
            }
            app.refreshSectPanel?.();

            return {
                buttons,
                sectPanelOpened: !beforeSectPanelActive && afterSectPanelActive,
                sectPanelClosedAfterProbe: !app.sectPanelRoot?.active,
            };
        }),
    );
}

async function runSectAffairsInstitutionPanelProbe(connection) {
    return connection.evaluate(
        appProbeExpression((app, cc) => {
            const taskHallNode = app.sessionActionRoot?.getChildByName('TaskHallButton') ?? null;
            const wasOpen = !!app.sectPanelRoot?.active;
            if (!wasOpen) {
                taskHallNode?.emit(cc.Node.EventType.MOUSE_UP);
            }
            const opened = !!app.sectPanelRoot?.active;
            app.setSectPanelTab?.('overview');
            app.refreshSectPanel?.();
            const overviewText = app.sectPanelLabel?.string ?? '';
            app.setSectPanelTab?.('institutions');
            app.refreshSectPanel?.();
            const institutionText = app.sectPanelLabel?.string ?? '';
            const institutionSection = institutionText.includes('\n机构\n')
                ? institutionText.split('\n机构\n')[1]?.split('\n生产\n')[0] ?? ''
                : '';
            const institutionEntryCount = (institutionSection.match(/(^|\n)- /g) ?? []).length;
            if (!wasOpen && opened) {
                app.closeSectPanel?.();
            }
            return {
                sectPanelOpened: opened,
                sectPanelRestoredClosed: !wasOpen ? !app.sectPanelRoot?.active : true,
                overviewVisible: overviewText.includes('经营总览'),
                institutionSectionVisible: institutionText.includes('\n机构\n'),
                institutionEntryCount,
                textSample: `${overviewText}\n${institutionText}`.slice(0, 800),
            };
        }),
    );
}

async function runSectStoryFeedbackPanelProbe(connection) {
    return connection.evaluate(
        appProbeExpression((app, cc) => {
            const taskHallNode = app.sessionActionRoot?.getChildByName('TaskHallButton') ?? null;
            const wasOpen = !!app.sectPanelRoot?.active;
            if (!wasOpen) {
                taskHallNode?.emit(cc.Node.EventType.MOUSE_UP);
            }
            const opened = !!app.sectPanelRoot?.active;
            app.setSectPanelTab?.('disciples');
            app.refreshSectPanel?.();
            const discipleText = app.sectPanelLabel?.string ?? '';
            app.setSectPanelTab?.('events');
            app.refreshSectPanel?.();
            const eventText = app.sectPanelLabel?.string ?? '';
            const sectionBetween = (startToken, endToken) => {
                const sourceText = startToken.includes('故事反馈') ? discipleText : eventText;
                if (!sourceText.includes(startToken)) {
                    return '';
                }
                const afterStart = sourceText.split(startToken)[1] ?? '';
                return endToken ? (afterStart.split(endToken)[0] ?? '') : afterStart;
            };
            const rosterStorySection = sectionBetween('\n故事反馈\n', '\n近期晋升\n');
            const diarySection = sectionBetween('\n弟子日记\n', '\n宗门事件\n');
            const eventFeedbackSection = sectionBetween('\n宗门事件\n', '');
            const countEntries = (sectionText) => (sectionText.match(/(^|\n)- /g) ?? []).length;
            if (!wasOpen && opened) {
                app.closeSectPanel?.();
            }
            return {
                sectPanelOpened: opened,
                sectPanelRestoredClosed: !wasOpen ? !app.sectPanelRoot?.active : true,
                rosterVisible: discipleText.includes('\n名册 '),
                storySectionVisible: discipleText.includes('\n故事反馈\n'),
                diarySectionVisible: eventText.includes('\n弟子日记\n'),
                eventFeedbackSectionVisible: eventText.includes('\n宗门事件\n'),
                rosterStoryEntryCount: countEntries(rosterStorySection),
                diaryEntryCount: countEntries(diarySection),
                eventFeedbackEntryCount: countEntries(eventFeedbackSection),
                textSample: `${discipleText}\n${eventText}`.slice(0, 1200),
            };
        }),
    );
}

async function runSectExpansionRiskPanelProbe(connection) {
    return connection.evaluate(
        appProbeExpression((app, cc) => {
            const taskHallNode = app.sessionActionRoot?.getChildByName('TaskHallButton') ?? null;
            const wasOpen = !!app.sectPanelRoot?.active;
            if (!wasOpen) {
                taskHallNode?.emit(cc.Node.EventType.MOUSE_UP);
            }
            const opened = !!app.sectPanelRoot?.active;
            app.setSectPanelTab?.('overview');
            app.refreshSectPanel?.();
            const overviewText = app.sectPanelLabel?.string ?? '';
            app.setSectPanelTab?.('buildings');
            app.refreshSectPanel?.();
            const buildingText = app.sectPanelLabel?.string ?? '';
            app.setSectPanelTab?.('institutions');
            app.refreshSectPanel?.();
            const institutionText = app.sectPanelLabel?.string ?? '';
            app.setSectPanelTab?.('events');
            app.refreshSectPanel?.();
            const eventText = app.sectPanelLabel?.string ?? '';
            const text = [overviewText, buildingText, institutionText, eventText].join('\n');
            const normalizedText = text.replace(/\r/g, '');
            const sectionBetween = (startLabel, endLabel) => {
                const startToken = `${startLabel}\n`;
                if (!normalizedText.includes(startToken)) {
                    return '';
                }
                const afterStart = normalizedText.split(startToken)[1] ?? '';
                if (!endLabel) {
                    return afterStart;
                }
                const endToken = `\n${endLabel}\n`;
                return endToken ? (afterStart.split(endToken)[0] ?? '') : afterStart;
            };
            const countEntries = (sectionText) => (sectionText.match(/(^|\n)- /g) ?? []).length;
            const goalsSection = sectionBetween('宗门目标', '宗门等级');
            const crisisSection = sectionBetween('危机链', '药庐 / 秩序');
            const treatmentSection = sectionBetween('治疗队列', '说明：治疗恢复只由 authority 日结推进，不做本地回血/减压。');
            const monthlySection = sectionBetween('月度小比', '待决事件');
            if (!wasOpen && opened) {
                app.closeSectPanel?.();
            }
            return {
                sectPanelOpened: opened,
                sectPanelRestoredClosed: !wasOpen ? !app.sectPanelRoot?.active : true,
                goalsSectionVisible: normalizedText.includes('宗门目标\n'),
                crisisSectionVisible: normalizedText.includes('危机链\n'),
                treatmentSectionVisible: normalizedText.includes('药庐 / 秩序\n') && normalizedText.includes('治疗队列\n'),
                monthlyAssessmentSectionVisible: normalizedText.includes('月度小比\n'),
                goalsEntryCount: countEntries(goalsSection),
                crisisEntryCount: countEntries(crisisSection),
                treatmentEntryCount: countEntries(treatmentSection),
                monthlyHasAuthorityResult: !monthlySection.includes('authority 尚无月度小比结果'),
                textSample: normalizedText.slice(0, 1600),
            };
        }),
    );
}

async function runBlockedGatherFallbackProbe(connection, tile) {
    return connection.evaluate(
        appProbeExpression((app, _cc, tileArg) => {
            const key = app.getTileKey(tileArg.col, tileArg.row);
            const resource = app.resourceNodes.get(key);
            const before = resource?.designated ?? null;
            app.enterAuthorityBlockedState?.(
                'gate_gather_disconnected_probe',
                'authority.gather_disconnected_probe',
                'gate probe: authority disconnected for gather fallback check',
            );
            app.setMode?.('gather', 'gate gather fallback probe');
            app.toggleGatherDesignation?.(tileArg);
            const after = resource?.designated ?? null;
            return {
                tile: `${tileArg.col},${tileArg.row}`,
                before,
                after,
                unchanged: before === after,
                authorityConnected: app.authorityConnected,
                authorityRenderSource: app.authorityRenderSource,
                lastError: app.authorityLastError,
            };
        }, [tile]),
    );
}

async function runResourceDesignationRoundTrip(connection, options, tile) {
    const localIntent = await connection.evaluate(
        appProbeExpression((app, _cc, tileArg) => {
            const key = app.getTileKey(tileArg.col, tileArg.row);
            const resource = app.resourceNodes.get(key);
            const before = resource?.designated ?? null;
            app.setMode?.('gather', 'gate gather round-trip probe');
            app.toggleGatherDesignation?.(tileArg);
            return {
                tile: `${tileArg.col},${tileArg.row}`,
                before,
                requested: before === null ? null : !before,
                modeAfterIntent: app.inputMode,
                lastMessage: app.lastMessage,
            };
        }, [tile]),
    );

    requireCondition(localIntent.before !== null, 'Gather round-trip probe could not find a resource node.', localIntent);
    const targetDesignation = !localIntent.before;
    const afterIntentSnapshot = await waitForCheckpoint(
        'resource_designation_round_trip',
        connection,
        options,
        (snapshot) => {
            const resource = findResourceNode(snapshot, tile);
            return !!resource && resource.designated === targetDesignation && snapshot.authority.lastError === null;
        },
    );
    const afterIntentResource = findResourceNode(afterIntentSnapshot, tile);

    await connection.evaluate(
        runtimeCallExpression('executeAuthorityCommand', [
            {
                name: 'set_resource_designation',
                payload: {
                    resourceTile: tile,
                    designated: localIntent.before,
                },
            },
            {
                commandKey: `gate:restore_resource_designation:${tile.col},${tile.row}:${localIntent.before ? 'on' : 'off'}`,
            },
        ]),
    );
    const restoredSnapshot = await waitForCheckpoint(
        'resource_designation_restore',
        connection,
        options,
        (snapshot) => {
            const resource = findResourceNode(snapshot, tile);
            return !!resource && resource.designated === localIntent.before && snapshot.authority.lastError === null;
        },
    );
    const restoredResource = findResourceNode(restoredSnapshot, tile);

    return {
        tile: localIntent.tile,
        before: localIntent.before,
        requested: targetDesignation,
        afterAuthorityRoundTrip: afterIntentResource?.designated ?? null,
        restored: restoredResource?.designated ?? null,
        lastEventAfterIntent: afterIntentSnapshot.authority.lastEvent,
        lastErrorAfterIntent: afterIntentSnapshot.authority.lastError,
        evidenceSource: 'client gather intent handler -> authority command -> snapshot resourceNodes',
    };
}

async function getHostilePresentationProbe(connection) {
    return connection.evaluate(
        appProbeExpression((app) => {
            const hostile = app.hostileNpc;
            if (!hostile?.active) {
                return null;
            }
            const target = app.getTileCenter(hostile.tile);
            const duration = hostile.authorityInterpolationDurationSeconds;
            const progress =
                duration > 0 ? Math.min(1, hostile.authorityInterpolationElapsedSeconds / duration) : 1;
            return {
                id: hostile.id,
                tile: `${hostile.tile.col},${hostile.tile.row}`,
                renderPosition: {
                    x: Number(hostile.worldPosition.x.toFixed(2)),
                    y: Number(hostile.worldPosition.y.toFixed(2)),
                },
                targetCenter: {
                    x: Number(target.x.toFixed(2)),
                    y: Number(target.y.toFixed(2)),
                },
                interpolationProgress: Number(progress.toFixed(3)),
                interpolationDurationSeconds: Number(duration.toFixed(3)),
                pathLength: hostile.path.length,
                visualState: hostile.visualState,
                hp: Math.ceil(hostile.currentHp),
            };
        }),
    );
}

function pointsDiffer(left, right) {
    if (!left || !right) {
        return false;
    }
    return Math.abs(left.x - right.x) > 0.01 || Math.abs(left.y - right.y) > 0.01;
}

async function waitForCombatPresentationEvidence(connection, options) {
    const startedAt = Date.now();
    let previousHostileTile = null;
    let hostileInterpolation = null;
    let damageFloater = null;
    let lastSummary = null;

    while (Date.now() - startedAt < options.timeoutMs) {
        const snapshot = await fetchLatestAuthoritySnapshot(connection);
        lastSummary = summarizeSnapshot(snapshot);

        const activeHostile = snapshot.hostiles.find((hostile) => hostile.active) ?? null;
        if (activeHostile) {
            const currentTile = activeHostile.tile;
            if (!hostileInterpolation && previousHostileTile && previousHostileTile !== currentTile) {
                const start = await getHostilePresentationProbe(connection);
                await delay(Math.min(300, Math.max(100, options.pollMs)));
                const afterDelay = await getHostilePresentationProbe(connection);
                if (
                    start &&
                    afterDelay &&
                    start.id === afterDelay.id &&
                    start.tile === afterDelay.tile &&
                    start.pathLength === 0 &&
                    pointsDiffer(start.renderPosition, start.targetCenter) &&
                    (pointsDiffer(start.renderPosition, afterDelay.renderPosition) ||
                        afterDelay.interpolationProgress > start.interpolationProgress)
                ) {
                    hostileInterpolation = {
                        previousAuthorityTile: previousHostileTile,
                        currentAuthorityTile: currentTile,
                        start,
                        afterDelay,
                    };
                }
            }
            previousHostileTile = currentTile;
        }

        const runtimeSnapshot = await connection.evaluate(runtimeCallExpression('getSnapshot'));
        const floaters = runtimeSnapshot.damageFloaters ?? [];
        if (!damageFloater && floaters.length > 0) {
            damageFloater = {
                capturedAtPhase: runtimeSnapshot.session.phase,
                entries: floaters.map((floater) => ({
                    targetKind: floater.targetKind,
                    targetId: floater.targetId,
                    targetName: floater.targetName,
                    amount: floater.amount,
                    remainingSeconds: floater.remainingSeconds,
                })),
            };
        }

        if (hostileInterpolation && damageFloater) {
            return {
                hostileInterpolation,
                damageFloater,
            };
        }

        await delay(Math.min(options.pollMs, 300));
    }

    throw new Error(
        `Timed out waiting for combat presentation evidence. Last phase: ${lastSummary?.session?.phase ?? 'unknown'}`,
    );
}

function firstIdleDisciple(sect, excludeIds = []) {
    return (sect.roster ?? []).find(
        (disciple) => disciple.assignmentKind === 'idle' && !excludeIds.includes(disciple.id),
    );
}

function findTaskByTitlePrefix(sect, prefix) {
    return (sect.tasks ?? []).find((task) => String(task.title ?? '').startsWith(prefix)) ?? null;
}

async function runSectMultidayManagementReplay(connection, options, seedSnapshot, runId) {
    const userId = seedSnapshot.authority?.playerId ?? 'preview-player';
    const sessionId = seedSnapshot.authority?.sessionId ?? 'preview-player/preview-local';
    let response = await resetSectAuthority(options, seedSnapshot);
    let snapshot = response.snapshot;
    let sect = summarizeRawSectSnapshot(snapshot);
    const context = {
        userId,
        sectId: sect.sectId,
        sessionId,
    };
    const commandPrefix = `gate-multiday-${runId}`;
    let commandIndex = 0;
    const nextCommandId = (slug) => `${commandPrefix}-${String(commandIndex += 1).padStart(2, '0')}-${slug}`;
    const evidence = {
        start: sect,
        commands: [],
        dayAdvances: [],
        feedbackCategories: [],
    };
    const feedbackCategories = new Set(evidence.start.eventFeedbackCategories ?? []);

    requireCondition(sect.connected && !!sect.sectId, 'Sect authority did not reset before multiday replay.', evidence.start);

    const execute = async (label, type, payload) => {
        const beforeVersion = sect.sceneVersion;
        response = await submitSectCommandHttp(
            options,
            context,
            sectCommand(nextCommandId(label), type, beforeVersion, payload),
        );
        snapshot = response.snapshot;
        sect = summarizeRawSectSnapshot(snapshot);
        for (const category of sect.eventFeedbackCategories ?? []) {
            feedbackCategories.add(category);
        }
        const commandEvidence = {
            label,
            type,
            beforeVersion,
            afterVersion: sect.sceneVersion,
            resultStatus: response.result?.status ?? null,
            error: response.result?.error ?? null,
            sect,
        };
        evidence.commands.push(commandEvidence);
        requireCondition(response.result?.status === 'COMMAND_RESULT_STATUS_ACCEPTED', `Sect command ${label} was rejected.`, commandEvidence);
        requireCondition(sect.sceneVersion > beforeVersion, `Sect command ${label} did not advance scene version.`, commandEvidence);
        return snapshot;
    };

    await execute('recruit', 'COMMAND_TYPE_START_RECRUITMENT', {
        candidateCount: 2,
        investmentSpiritStone: 20,
        durationDays: 5,
    });
    const candidates = [...(sect.candidates ?? [])].sort((left, right) => left.id.localeCompare(right.id));
    requireCondition(candidates.length >= 2, 'Recruitment did not expose at least two authority candidates.', sect);
    const acceptedCandidate = candidates[0];
    const rejectedCandidate = candidates[1];

    await execute('accept-candidate', 'COMMAND_TYPE_ACCEPT_CANDIDATE', {
        candidateId: acceptedCandidate.id,
    });
    const acceptedDisciple = (sect.roster ?? []).find((disciple) => disciple.name === acceptedCandidate.name);
    requireCondition(!!acceptedDisciple, 'Accepted candidate did not enter authority roster.', sect);

    await execute('reject-candidate', 'COMMAND_TYPE_REJECT_CANDIDATE', {
        candidateId: rejectedCandidate.id,
    });

    const taskTitlePrefix = `验收巡山 ${runId}`;
    await execute('publish-task', 'COMMAND_TYPE_PUBLISH_TASK', {
        kind: 'acceptance_patrol',
        title: taskTitlePrefix,
        description: 'dedicated browser gate compressed multiday task',
        priority: 80,
        requiredProgressDays: 1,
        risk: 5,
        maxAssignees: 1,
        minIdentity: 'outer_disciple',
        minRealm: 'mortal',
        requiredAptitude: {
            physique: 5,
        },
        contributionReward: 11,
        rewardResources: {
            spirit_stone: 5,
            herb: 2,
        },
    });
    const task = findTaskByTitlePrefix(sect, taskTitlePrefix);
    requireCondition(!!task, 'Published multiday gate task was not visible in authority task list.', sect);
    const taskDisciple = firstIdleDisciple(sect, ['disciple-1']) ?? firstIdleDisciple(sect);
    requireCondition(!!taskDisciple, 'No idle authority disciple was available for multiday task dispatch.', sect);
    await execute('assign-task', 'COMMAND_TYPE_ASSIGN_DISCIPLE_TASK', {
        taskId: task.id,
        discipleId: taskDisciple.id,
        discipleIds: [taskDisciple.id],
    });

    await execute('start-production', 'COMMAND_TYPE_START_PRODUCTION', {
        recipeId: 'formation_refine_mvp',
        priority: 75,
        targetCycles: 1,
    });

    await execute('start-cultivation', 'COMMAND_TYPE_START_CULTIVATION', {
        discipleId: 'disciple-1',
    });

    const advanceAndRefresh = async (label, days) => {
        response = await advanceSectDays(options, { authority: { playerId: userId, sessionId }, sectAuthority: { connected: true, sectId: context.sectId } }, days);
        snapshot = response.snapshot;
        sect = summarizeRawSectSnapshot(snapshot);
        for (const category of sect.eventFeedbackCategories ?? []) {
            feedbackCategories.add(category);
        }
        const advanceEvidence = {
            label,
            days,
            fromVersion: response.fromVersion ?? null,
            toVersion: response.toVersion ?? null,
            sect,
        };
        evidence.dayAdvances.push(advanceEvidence);
        requireCondition(response.daysAdvanced === days, `Sect day advance ${label} did not advance expected days.`, advanceEvidence);
        return snapshot;
    };

    await advanceAndRefresh('first-three-days', 3);
    requireCondition(
        sect.cultivation?.omenStatus === 'foreshadowed' || (sect.omenCount ?? 0) > 0,
        'Three-day advance did not produce authority cultivation omen visibility.',
        sect,
    );
    await execute('attempt-breakthrough', 'COMMAND_TYPE_ATTEMPT_BREAKTHROUGH', {
        discipleId: 'disciple-1',
    });
    await advanceAndRefresh('remaining-twenty-seven-days', 27);

    evidence.final = sect;
    evidence.feedbackCategories = [...feedbackCategories].sort();
    const completedTask = (sect.tasks ?? []).find((entry) => entry.id === task.id && entry.status === 'completed');
    const completedProduction = (sect.productions ?? []).find((production) => (production.completedCycles ?? 0) >= 1);
    const monthlyFeedback = evidence.feedbackCategories.includes('monthly');
    const eventFeedbackEnough = ['task_result', 'resource_change', 'production', 'cultivation', 'breakthrough', 'monthly', 'omen'].every(
        (category) => evidence.feedbackCategories.includes(category),
    );

    requireCondition((sect.rosterCount ?? 0) >= 2, 'Multiday replay did not retain recruited roster.', evidence.final);
    requireCondition(!!completedTask, 'Multiday replay did not complete authority task dispatch.', evidence.final);
    requireCondition(!!completedProduction, 'Multiday replay did not complete authority production cycle.', evidence.final);
    requireCondition(
        sect.cultivation?.realmStage === 'qi_entry' || (sect.cultivation?.injuryLevel ?? 0) > 0,
        'Multiday replay did not expose authority cultivation/breakthrough result.',
        evidence.final,
    );
    requireCondition(monthlyFeedback, 'Multiday replay did not expose monthly settlement feedback.', evidence);
    requireCondition(eventFeedbackEnough, 'Multiday replay did not expose all required event feedback categories.', evidence);

    const rejoin = await postJson(`${options.authorityUrl}/v1/authority/sect/join`, {
        userId: context.userId,
        sectId: context.sectId,
        sessionId: `${context.sessionId}:rejoin-proof`,
    });
    evidence.rejoinAfterCompressedAdvance = summarizeRawSectSnapshot(rejoin.snapshot);
    requireCondition(
        evidence.rejoinAfterCompressedAdvance.sceneVersion === evidence.final.sceneVersion &&
            evidence.rejoinAfterCompressedAdvance.calendarDay === evidence.final.calendarDay &&
            evidence.rejoinAfterCompressedAdvance.rosterCount === evidence.final.rosterCount &&
            evidence.rejoinAfterCompressedAdvance.monthly.lastSettledMonth === evidence.final.monthly.lastSettledMonth,
        'Multiday replay rejoin did not preserve saved authority state.',
        {
            final: evidence.final,
            rejoin: evidence.rejoinAfterCompressedAdvance,
        },
    );

    const browserSnapshot = await joinSectAuthority(connection);
    const browserSect = summarizeSectAuthority(browserSnapshot);
    const storyPanelProbe = await runSectStoryFeedbackPanelProbe(connection);
    const readableStoryFeedbackTypes = [
        ...(countReadableRosterStoryEntries(browserSect.roster) > 0 ? ['roster_story'] : []),
        ...((browserSect.diary ?? []).some((entry) => hasReadableText(entry.storyText) || hasReadableText(entry.summary))
            ? ['diary_story']
            : []),
        ...((browserSect.eventFeedbackCategories ?? []).length > 0 ? browserSect.eventFeedbackCategories : []),
    ].filter((value, index, values) => values.indexOf(value) === index);
    evidence.storyFeedback = {
        browserSect,
        storyPanelProbe,
        readableStoryFeedbackTypes,
        rosterStoryCount: countReadableRosterStoryEntries(browserSect.roster),
    };
    requireCondition(
        storyPanelProbe.rosterVisible && storyPanelProbe.storySectionVisible && storyPanelProbe.diarySectionVisible && storyPanelProbe.eventFeedbackSectionVisible,
        'Multiday replay could not expose roster/story/diary feedback sections in the sect panel.',
        evidence.storyFeedback,
    );
    requireCondition(
        readableStoryFeedbackTypes.length >= 3,
        'Multiday replay did not expose at least three readable story feedback types in bounded browser evidence.',
        evidence.storyFeedback,
    );

    return evidence;
}

function firstAuthorityChoiceEvent(sect) {
    return (sect.eventChoices ?? []).find((event) => (event.options ?? []).length > 0) ?? null;
}

async function runSectEventChoiceReplay(connection, options, seedSnapshot, runId) {
    const userId = seedSnapshot.authority?.playerId ?? 'preview-player';
    const sessionId = seedSnapshot.authority?.sessionId ?? 'preview-player/preview-local';
    let response = await resetSectAuthority(options, seedSnapshot);
    let sect = summarizeRawSectSnapshot(response.snapshot);
    requireCondition(sect.connected && !!sect.sectId, 'Sect authority did not reset before event choice replay.', sect);

    response = await advanceSectDays(
        options,
        {
            authority: { playerId: userId, sessionId },
            sectAuthority: { connected: true, sectId: sect.sectId },
        },
        2,
    );
    const before = summarizeRawSectSnapshot(response.snapshot);
    const rawChoiceEvent = firstAuthorityChoiceEvent(before);
    requireCondition(!!rawChoiceEvent, 'Two-day authority advance did not expose an active event choice.', before);

    const browserBeforeSnapshot = await joinSectAuthority(connection);
    const browserBefore = summarizeSectAuthority(browserBeforeSnapshot);
    const panelProbe = await runPlayerFacingControlsProbe(connection);
    requireCondition(panelProbe.sectPanelOpened, 'Event choice replay could not open the sect panel before choosing.', panelProbe);

    const selectedOption =
        (rawChoiceEvent.options ?? []).find((option) => option.id === 'send_aid') ??
        rawChoiceEvent.options?.[0] ??
        null;
    requireCondition(!!selectedOption?.id, 'Event choice replay did not find a selectable authority option.', rawChoiceEvent);

    const commandResponse = await submitSectCommandHttp(
        options,
        {
            userId,
            sectId: before.sectId,
            sessionId,
        },
        sectCommand(`gate-event-choice-${runId}`, 'COMMAND_TYPE_CHOOSE_EVENT_OPTION', before.sceneVersion, {
            eventId: rawChoiceEvent.id,
            optionId: selectedOption.id,
        }),
    );
    const after = summarizeRawSectSnapshot(commandResponse.snapshot);
    const eventStillActive = (after.eventChoices ?? []).some((event) => event.id === rawChoiceEvent.id);
    const resolvedChoice = (after.resolvedEvents ?? []).find(
        (event) => event.id === rawChoiceEvent.id && event.outcome === `option:${selectedOption.id}`,
    );
    const feedbackMentionsResolution = (after.eventFeedback ?? []).some(
        (entry) =>
            entry.category === 'omen' &&
            (String(entry.eventType ?? '').includes('resolved') || String(entry.summary ?? '').includes('天机收束')),
    );

    requireCondition(commandResponse.result?.status === 'COMMAND_RESULT_STATUS_ACCEPTED', 'Event choice authority command was not accepted.', {
        result: commandResponse.result,
        before: browserBefore,
        after,
    });
    requireCondition(!eventStillActive && !!resolvedChoice, 'Event choice did not resolve the selected authority event.', {
        choiceEvent: rawChoiceEvent,
        selectedOption,
        after,
    });
    requireCondition(feedbackMentionsResolution, 'Event choice did not produce event_log-derived feedback in the browser snapshot.', after);
    await joinSectAuthority(connection);

    return {
        reset: sect,
        before,
        browserBefore,
        panelProbe,
        selectedOption,
        commandResult: {
            status: commandResponse.result?.status ?? null,
            sceneVersion: commandResponse.result?.sceneVersion ?? null,
            patchFromVersion: commandResponse.result?.patch?.fromVersion ?? null,
            patchToVersion: commandResponse.result?.patch?.toVersion ?? null,
        },
        after,
        evidenceSource: 'dedicated browser sect panel opened -> authority HTTP ChooseEventOption command -> snapshot/patch/event_log feedback',
    };
}

function findRawTask(sect, taskId) {
    return (sect.tasks ?? []).find((task) => task.id === taskId) ?? null;
}

function findRawProduction(sect, productionId) {
    return (sect.productions ?? []).find((production) => production.id === productionId) ?? null;
}

function findClientArtifact(sect, artifactType) {
    return (sect.artifacts ?? []).find((artifact) => artifact.type === artifactType) ?? null;
}

function rawTaskMetric(task) {
    if (!task) {
        return null;
    }
    if (typeof task.successRate === 'number' && task.successRate > 0) {
        return task.successRate;
    }
    const recommended = task.recommendedDispatch?.recommendedSuccessRate;
    return typeof recommended === 'number' && recommended > 0 ? recommended : null;
}

function selectArtifactTaskCandidate(tasks, acceptedTypes) {
    return (
        (tasks ?? []).find(
            (task) =>
                acceptedTypes.includes(task.taskType) &&
                task.recommendedDispatch?.recommendedDiscipleIds?.includes('disciple-1') &&
                rawTaskMetric(task) !== null,
        ) ??
        (tasks ?? []).find((task) => acceptedTypes.includes(task.taskType) && rawTaskMetric(task) !== null) ??
        null
    );
}

function averageRawCandidateSpiritRoot(snapshot) {
    const candidates = Object.values(snapshot?.state?.admissions?.candidates ?? {});
    if (candidates.length === 0) {
        return 0;
    }
    const total = candidates.reduce((sum, candidate) => sum + (candidate.aptitude?.spirit_root ?? 0), 0);
    return Math.round((total / candidates.length) * 100) / 100;
}

async function runSectPolicyImpactReplay(connection, options, seedSnapshot, runId) {
    const userId = seedSnapshot.authority?.playerId ?? 'preview-player';
    const sessionId = seedSnapshot.authority?.sessionId ?? 'preview-player/preview-local';
    let response = await resetSectAuthority(options, seedSnapshot);
    let raw = summarizeRawSectSnapshot(response.snapshot);
    requireCondition(raw.connected && !!raw.sectId, 'Sect authority did not reset before policy replay.', raw);

    let browserSnapshot = await joinSectAuthority(connection);
    let browserSect = summarizeSectAuthority(browserSnapshot);
    const context = {
        userId,
        sectId: raw.sectId,
        sessionId,
    };
    const evidence = {
        startRaw: raw,
        startBrowser: browserSect,
        commands: [],
    };
    const commandPrefix = `gate-policy-${runId}`;
    let commandIndex = 0;
    const nextCommandId = (slug) => `${commandPrefix}-${String(commandIndex += 1).padStart(2, '0')}-${slug}`;

    const executeBrowserCommand = async (label, type, payload) => {
        const before = browserSect;
        browserSnapshot = await executeSectAuthorityCommand(
            connection,
            sectCommand(nextCommandId(label), type, before.sceneVersion, payload),
        );
        browserSect = summarizeSectAuthority(browserSnapshot);
        const commandEvidence = {
            label,
            type,
            beforeVersion: before.sceneVersion,
            afterVersion: browserSect.sceneVersion,
            lastError: browserSect.lastError,
            policies: browserSect.policies,
        };
        evidence.commands.push(commandEvidence);
        requireCondition(browserSect.lastError === null, `Policy replay browser command ${label} left sect lastError non-null.`, commandEvidence);
        requireCondition(browserSect.sceneVersion > before.sceneVersion, `Policy replay browser command ${label} did not advance version.`, commandEvidence);
        return browserSnapshot;
    };

    await executeBrowserCommand('task-production', 'COMMAND_TYPE_SET_POLICY', {
        policyCategory: 'task',
        policyValue: 'production',
        policy_category: 'task',
        policy_value: 'production',
    });
    await executeBrowserCommand('resource-war-preparation', 'COMMAND_TYPE_SET_POLICY', {
        policyCategory: 'resource',
        policyValue: 'war_preparation',
        policy_category: 'resource',
        policy_value: 'war_preparation',
    });
    await executeBrowserCommand('recruitment-selective', 'COMMAND_TYPE_SET_POLICY', {
        policyCategory: 'recruitment',
        policyValue: 'selective',
        policy_category: 'recruitment',
        policy_value: 'selective',
    });
    await executeBrowserCommand('cultivation-closed', 'COMMAND_TYPE_SET_POLICY', {
        policyCategory: 'cultivation',
        policyValue: 'closed_cultivation',
        policy_category: 'cultivation',
        policy_value: 'closed_cultivation',
    });
    evidence.afterPolicySwitchBrowser = browserSect;
    const afterPolicySwitchResponse = await postJson(`${options.authorityUrl}/v1/authority/sect/join`, context);
    const afterPolicySwitchRaw = summarizeRawSectSnapshot(afterPolicySwitchResponse.snapshot);
    evidence.afterPolicySwitchRaw = afterPolicySwitchRaw;

    requireCondition(
        afterPolicySwitchRaw.policies.taskPolicy === 'production' &&
            afterPolicySwitchRaw.policies.resourcePolicy === 'war_preparation' &&
            afterPolicySwitchRaw.policies.recruitmentPolicy === 'selective' &&
            afterPolicySwitchRaw.policies.cultivationPolicy === 'closed_cultivation',
        'Browser policy commands did not produce expected authority policy state.',
        {
            browser: browserSect,
            authority: afterPolicySwitchRaw.policies,
        },
    );

    await executeBrowserCommand('start-recruitment', 'COMMAND_TYPE_START_RECRUITMENT', {
        candidateCount: 0,
        investmentSpiritStone: 0,
        durationDays: 7,
    });
    await executeBrowserCommand('start-cultivation', 'COMMAND_TYPE_START_CULTIVATION', {
        discipleId: 'disciple-1',
    });
    evidence.afterBrowserIntents = browserSect;

    response = await advanceSectDays(
        options,
        {
            authority: { playerId: userId, sessionId },
            sectAuthority: { connected: true, sectId: context.sectId },
        },
        30,
    );
    raw = summarizeRawSectSnapshot(response.snapshot);
    browserSnapshot = await joinSectAuthority(connection);
    browserSect = summarizeSectAuthority(browserSnapshot);
    const formationTask = findRawTask(raw, 'pool-3');
    const formationProduction = findRawProduction(raw, 'prod-4-formation-refine');
    evidence.afterThirtyDaysRaw = raw;
    evidence.afterThirtyDaysBrowser = browserSect;
    evidence.metrics = {
        taskPolicy: raw.policies.taskPolicy,
        resourcePolicy: raw.policies.resourcePolicy,
        recruitmentPolicy: raw.policies.recruitmentPolicy,
        cultivationPolicy: raw.policies.cultivationPolicy,
        formationTaskPriority: formationTask?.priority ?? null,
        formationProductionPriority: formationProduction?.priority ?? null,
        candidateCount: raw.candidateCount,
        averageCandidateSpiritRoot: averageRawCandidateSpiritRoot(response.snapshot),
        cultivationPoints: raw.cultivation?.cultivationPoints ?? null,
        monthlySatisfactionDelta: raw.monthly.lastSettlement?.satisfaction_delta_total ?? null,
        policyEventCount: afterPolicySwitchRaw.recentPolicyEvents.length,
        postAdvancePolicyEventCount: raw.recentPolicyEvents.length,
        browserLastError: browserSect.lastError,
    };

    requireCondition(response.daysAdvanced === 30, 'Policy replay did not advance thirty compressed days.', evidence.metrics);
    requireCondition(evidence.metrics.formationTaskPriority > 52, 'Policy replay did not show production task priority effect.', evidence.metrics);
    requireCondition(evidence.metrics.formationProductionPriority > 54, 'Policy replay did not show war preparation production priority effect.', evidence.metrics);
    requireCondition(evidence.metrics.candidateCount === 2, 'Policy replay did not show selective recruitment count effect.', evidence.metrics);
    requireCondition(evidence.metrics.averageCandidateSpiritRoot >= 10, 'Policy replay did not show selective recruitment quality effect.', evidence.metrics);
    requireCondition(evidence.metrics.cultivationPoints > 0, 'Policy replay did not show cultivation progression under policy.', evidence.metrics);
    requireCondition(evidence.metrics.policyEventCount >= 4, 'Policy replay did not record policy events in bounded event feedback.', evidence.metrics);
    requireCondition(browserSect.lastError === null, 'Policy replay browser snapshot ended with sect lastError.', browserSect);
    requireCondition(browserSnapshot.authority?.lastError === null, 'Policy replay browser snapshot ended with authority lastError.', browserSnapshot.authority);

    return evidence;
}

async function runSectArtifactImpactReplay(connection, options, seedSnapshot, runId) {
    const userId = seedSnapshot.authority?.playerId ?? 'preview-player';
    const sessionId = seedSnapshot.authority?.sessionId ?? 'preview-player/preview-local';
    let response = await resetSectAuthority(options, seedSnapshot);
    let raw = summarizeRawSectSnapshot(response.snapshot);
    requireCondition(raw.connected && !!raw.sectId, 'Sect authority did not reset before artifact replay.', raw);

    const context = {
        userId,
        sectId: raw.sectId,
        sessionId,
    };
    let browserSnapshot = await joinSectAuthority(connection);
    let browserSect = summarizeSectAuthority(browserSnapshot);
    const evidence = {
        beforeRaw: raw,
        beforeBrowser: browserSect,
        commands: [],
    };

    const combatTaskBefore = selectArtifactTaskCandidate(raw.tasks, ['combat', 'external', 'explore']);
    const productionTaskBefore = selectArtifactTaskCandidate(raw.tasks, ['production']);
    requireCondition(
        !!combatTaskBefore || !!productionTaskBefore,
        'Artifact replay did not expose any authority task metric candidate before equip.',
        raw.tasks,
    );

    const commandPrefix = `gate-artifact-${runId}`;
    let commandIndex = 0;
    const nextCommandId = (slug) => `${commandPrefix}-${String(commandIndex += 1).padStart(2, '0')}-${slug}`;
    const executeBrowserCommand = async (label, type, payload) => {
        const before = browserSect;
        browserSnapshot = await executeSectAuthorityCommand(connection, sectCommand(nextCommandId(label), type, before.sceneVersion, payload));
        browserSect = summarizeSectAuthority(browserSnapshot);
        const commandEvidence = {
            label,
            type,
            beforeVersion: before.sceneVersion,
            afterVersion: browserSect.sceneVersion,
            artifactCount: browserSect.artifactCount,
            lastError: browserSect.lastError,
        };
        evidence.commands.push(commandEvidence);
        requireCondition(browserSect.lastError === null, `Artifact replay browser command ${label} left sect lastError non-null.`, commandEvidence);
        requireCondition(
            browserSnapshot.authority?.lastError === null,
            `Artifact replay browser command ${label} left authority lastError non-null.`,
            {
                authority: browserSnapshot.authority,
                sect: browserSect,
            },
        );
        requireCondition(browserSect.sceneVersion > before.sceneVersion, `Artifact replay browser command ${label} did not advance version.`, commandEvidence);
        return browserSnapshot;
    };

    await executeBrowserCommand('craft-sword', 'COMMAND_TYPE_CRAFT_ARTIFACT', {
        artifactType: 'sword',
        quality: 1,
    });
    await executeBrowserCommand('craft-farm-tool', 'COMMAND_TYPE_CRAFT_ARTIFACT', {
        artifactType: 'farm_tool',
        quality: 1,
    });

    const craftedSword = findClientArtifact(browserSect, 'sword');
    const craftedTool = findClientArtifact(browserSect, 'farm_tool');
    requireCondition(!!craftedSword && !!craftedTool, 'Artifact replay did not expose crafted sword/tool in browser snapshot.', browserSect.artifacts);

    await executeBrowserCommand('equip-sword', 'COMMAND_TYPE_EQUIP_ARTIFACT', {
        itemId: craftedSword.id,
        discipleId: 'disciple-1',
    });
    await executeBrowserCommand('equip-farm-tool', 'COMMAND_TYPE_EQUIP_ARTIFACT', {
        itemId: craftedTool.id,
        discipleId: 'disciple-1',
    });

    response = await postJson(`${options.authorityUrl}/v1/authority/sect/join`, context);
    raw = summarizeRawSectSnapshot(response.snapshot);
    browserSnapshot = await joinSectAuthority(connection);
    browserSect = summarizeSectAuthority(browserSnapshot);
    evidence.afterRaw = raw;
    evidence.afterBrowser = browserSect;

    const swordAfter = findClientArtifact(browserSect, 'sword');
    const toolAfter = findClientArtifact(browserSect, 'farm_tool');
    const rosterDisciple = rawDiscipleByID(raw, 'disciple-1');
    const combatTaskAfter = combatTaskBefore ? findRawTask(raw, combatTaskBefore.id) : null;
    const productionTaskAfter = productionTaskBefore ? findRawTask(raw, productionTaskBefore.id) : null;
    const combatRateBefore = rawTaskMetric(combatTaskBefore);
    const combatRateAfter = rawTaskMetric(combatTaskAfter);
    const productionRateBefore = rawTaskMetric(productionTaskBefore);
    const productionRateAfter = rawTaskMetric(productionTaskAfter);

    evidence.metrics = {
        beforeArtifactCount: evidence.beforeRaw.artifactCount,
        afterArtifactCount: raw.artifactCount,
        swordBoundDiscipleId: swordAfter?.boundDiscipleId ?? null,
        toolBoundDiscipleId: toolAfter?.boundDiscipleId ?? null,
        rosterEquipment: rosterDisciple?.equipment ?? {},
        combatTaskId: combatTaskBefore?.id ?? null,
        combatRateBefore,
        combatRateAfter,
        productionTaskId: productionTaskBefore?.id ?? null,
        productionRateBefore,
        productionRateAfter,
        browserLastError: browserSect.lastError,
        authorityLastError: browserSnapshot.authority?.lastError ?? null,
    };

    requireCondition(raw.artifactCount >= evidence.beforeRaw.artifactCount+2, 'Artifact replay did not increase authority artifact inventory after crafting.', evidence.metrics);
    requireCondition(swordAfter?.boundDiscipleId === 'disciple-1', 'Artifact replay did not bind crafted sword to disciple-1.', {
        swordAfter,
        browserArtifacts: browserSect.artifacts,
    });
    requireCondition(toolAfter?.boundDiscipleId === 'disciple-1', 'Artifact replay did not bind crafted tool to disciple-1.', {
        toolAfter,
        browserArtifacts: browserSect.artifacts,
    });
    requireCondition(
        rosterDisciple?.equipment?.weapon === swordAfter?.id && rosterDisciple?.equipment?.tool === toolAfter?.id,
        'Artifact replay raw roster did not preserve equipped sword/tool slots.',
        {
            rosterDisciple,
            swordAfter,
            toolAfter,
        },
    );

    const metricDeltaObserved =
        (combatRateBefore !== null && combatRateAfter !== null && combatRateAfter > combatRateBefore) ||
        (productionRateBefore !== null && productionRateAfter !== null && productionRateAfter > productionRateBefore);
    requireCondition(metricDeltaObserved, 'Artifact replay did not produce a bounded authority task/production metric increase.', evidence.metrics);
    requireCondition(browserSect.lastError === null, 'Artifact replay browser snapshot ended with sect lastError.', browserSect);
    requireCondition(browserSnapshot.authority?.lastError === null, 'Artifact replay browser snapshot ended with authority lastError.', browserSnapshot.authority);

    return evidence;
}

function rawDiscipleByID(sect, discipleId) {
    return (sect.roster ?? []).find((disciple) => disciple.id === discipleId) ?? null;
}

function rawDiscipleByName(sect, discipleName) {
    return (sect.roster ?? []).find((disciple) => disciple.name === discipleName) ?? null;
}

function rawBuildingByID(sect, buildingId) {
    return (sect.buildings ?? []).find((building) => building.id === buildingId) ?? null;
}

function rawBuildingByDefinition(sect, definitionKey) {
    return (sect.buildings ?? []).find((building) => building.definitionKey === definitionKey) ?? null;
}

function rawFormationByBuilding(sect, buildingId) {
    return (sect.formations ?? []).find((formation) => formation.buildingId === buildingId) ?? null;
}

function sectExpansionContinuityComparable(raw) {
    return {
        sceneVersion: raw.sceneVersion,
        calendarDay: raw.calendarDay,
        rosterCount: raw.rosterCount,
        buildingCount: raw.buildingCount,
        formationCount: raw.formationCount ?? 0,
        artifactCount: raw.artifactCount,
        goalCount: raw.goalCount ?? 0,
        resolvedGoalCount: raw.resolvedGoalCount ?? 0,
        crisisCount: raw.crisisCount ?? 0,
        treatmentCount: raw.treatmentCount ?? 0,
        monthlyLastMonthIndex: raw.monthlyAssessment?.lastMonthIndex ?? 0,
        order: raw.order,
    };
}

function buildingMaintenanceMetrics(raw, buildingId) {
    const building = rawBuildingByID(raw, buildingId);
    return {
        buildingId,
        calendarDay: raw.calendarDay,
        resources: raw.resources,
        building,
        catalogEntry: (raw.buildingCatalog ?? []).find((entry) => entry.definitionKey === building?.definitionKey) ?? null,
        damagedCount: (raw.buildings ?? []).filter((entry) => entry.phase === 'damaged').length,
        lowEfficiencyCount: (raw.buildings ?? []).filter((entry) => (entry.efficiency ?? 100) < 100).length,
    };
}

async function runSectPromotionReplay(connection, options, seedSnapshot, runId) {
    const userId = seedSnapshot.authority?.playerId ?? 'preview-player';
    const sessionId = seedSnapshot.authority?.sessionId ?? 'preview-player/preview-local';
    let response = await resetSectAuthority(options, seedSnapshot);
    let raw = summarizeRawSectSnapshot(response.snapshot);
    requireCondition(raw.connected && !!raw.sectId, 'Sect authority did not reset before promotion replay.', raw);

    const context = {
        userId,
        sectId: raw.sectId,
        sessionId,
    };
    const commandPrefix = `gate-promotion-${runId}`;
    let commandIndex = 0;
    const nextCommandId = (slug) => `${commandPrefix}-${String(commandIndex += 1).padStart(2, '0')}-${slug}`;
    const evidence = {
        startRaw: raw,
        setupCommands: [],
    };

    const executeHttpCommand = async (label, type, payload) => {
        const beforeVersion = raw.sceneVersion;
        response = await submitSectCommandHttp(
            options,
            context,
            sectCommand(nextCommandId(label), type, beforeVersion, payload),
        );
        raw = summarizeRawSectSnapshot(response.snapshot);
        const commandEvidence = {
            label,
            type,
            beforeVersion,
            afterVersion: raw.sceneVersion,
            resultStatus: response.result?.status ?? null,
            error: response.result?.error ?? null,
            raw,
        };
        evidence.setupCommands.push(commandEvidence);
        requireCondition(response.result?.status === 'COMMAND_RESULT_STATUS_ACCEPTED', `Promotion setup command ${label} was rejected.`, commandEvidence);
        requireCondition(raw.sceneVersion > beforeVersion, `Promotion setup command ${label} did not advance scene version.`, commandEvidence);
        return response.snapshot;
    };

    const advanceRawDays = async (label, days) => {
        response = await postJson(`${options.authorityUrl}/v1/authority/sect/debug/advance-days`, {
            userId,
            sectId: context.sectId,
            sessionId,
            days,
        });
        raw = summarizeRawSectSnapshot(response.snapshot);
        const advanceEvidence = {
            label,
            days,
            daysAdvanced: response.daysAdvanced ?? null,
            fromVersion: response.fromVersion ?? null,
            toVersion: response.toVersion ?? null,
            raw,
        };
        evidence.setupCommands.push(advanceEvidence);
        requireCondition(response.daysAdvanced === days, `Promotion setup day advance ${label} did not advance expected days.`, advanceEvidence);
        return response.snapshot;
    };

    await executeHttpCommand('build-main-hall', 'COMMAND_TYPE_BUILD_BUILDING', {
        definitionKey: 'main_hall',
        origin: {
            col: 4,
            row: 6,
        },
    });
    await executeHttpCommand('upgrade-main-hall', 'COMMAND_TYPE_UPGRADE_BUILDING', {
        buildingId: 'building-1',
    });
    await executeHttpCommand('publish-contribution-task', 'COMMAND_TYPE_PUBLISH_TASK', {
        kind: 'promotion_trial',
        title: `晋升考核筹备 ${runId}`,
        description: 'dedicated browser gate promotion setup task',
        priority: 90,
        requiredProgressDays: 1,
        risk: 5,
        maxAssignees: 1,
        minIdentity: 'outer_disciple',
        minRealm: 'mortal',
        requiredAptitude: {
            physique: 5,
        },
        contributionReward: 30,
        rewardResources: {
            spirit_stone: 5,
        },
    });
    await executeHttpCommand('assign-starter-task', 'COMMAND_TYPE_ASSIGN_DISCIPLE_TASK', {
        taskId: 'task-1',
        discipleId: 'disciple-1',
        discipleIds: ['disciple-1'],
    });
    await advanceRawDays('complete-contribution-task', 1);
    await executeHttpCommand('start-cultivation', 'COMMAND_TYPE_START_CULTIVATION', {
        discipleId: 'disciple-1',
    });
    await advanceRawDays('prepare-breakthrough', 3);
    await executeHttpCommand('attempt-breakthrough', 'COMMAND_TYPE_ATTEMPT_BREAKTHROUGH', {
        discipleId: 'disciple-1',
    });

    const beforeAssessmentRaw = raw;
    const beforeAssessmentDisciple = rawDiscipleByID(beforeAssessmentRaw, 'disciple-1');
    requireCondition(
        beforeAssessmentDisciple?.realmStage === 'qi_entry' && beforeAssessmentDisciple?.contributionBalance >= 10,
        'Promotion setup did not produce an assessment-ready starter disciple.',
        beforeAssessmentRaw,
    );

    let browserSnapshot = await joinSectAuthority(connection);
    let browserSect = summarizeSectAuthority(browserSnapshot);
    evidence.beforeAssessmentRaw = beforeAssessmentRaw;
    evidence.beforeAssessmentBrowser = browserSect;

    browserSnapshot = await executeSectAuthorityCommand(
        connection,
        sectCommand(nextCommandId('start-assessment-browser'), 'COMMAND_TYPE_START_ASSESSMENT', browserSect.sceneVersion, {
            discipleId: 'disciple-1',
            targetRank: 'inner_disciple',
        }),
    );
    browserSect = summarizeSectAuthority(browserSnapshot);
    const afterAssessmentJoin = await postJson(`${options.authorityUrl}/v1/authority/sect/join`, context);
    const afterAssessmentRaw = summarizeRawSectSnapshot(afterAssessmentJoin.snapshot);
    const assessedDisciple = rawDiscipleByID(afterAssessmentRaw, 'disciple-1');
    evidence.afterAssessmentBrowser = browserSect;
    evidence.afterAssessmentRaw = afterAssessmentRaw;

    requireCondition(browserSect.lastError === null, 'Browser assessment command left sect lastError non-null.', browserSect);
    requireCondition(
        assessedDisciple?.assessment?.passed === true && assessedDisciple.assessment.targetRank === 'inner_disciple',
        'Authority assessment did not mark starter as passed for inner disciple.',
        afterAssessmentRaw,
    );

    browserSnapshot = await executeSectAuthorityCommand(
        connection,
        sectCommand(nextCommandId('promote-disciple-browser'), 'COMMAND_TYPE_PROMOTE_DISCIPLE', browserSect.sceneVersion, {
            discipleId: 'disciple-1',
            targetRank: 'inner_disciple',
        }),
    );
    browserSect = summarizeSectAuthority(browserSnapshot);
    const afterPromotionJoin = await postJson(`${options.authorityUrl}/v1/authority/sect/join`, context);
    const afterPromotionRaw = summarizeRawSectSnapshot(afterPromotionJoin.snapshot);
    const promotedDisciple = rawDiscipleByID(afterPromotionRaw, 'disciple-1');
    evidence.afterPromotionBrowser = browserSect;
    evidence.afterPromotionRaw = afterPromotionRaw;
    evidence.metrics = {
        browserLastError: browserSect.lastError,
        identity: promotedDisciple?.identity ?? null,
        contributionBalance: promotedDisciple?.contributionBalance ?? null,
        assessmentPassed: promotedDisciple?.assessment?.passed ?? null,
        promotionFeedbackCount: afterPromotionRaw.recentPromotionEvents.length,
        feedbackCategories: afterPromotionRaw.eventFeedbackCategories,
    };

    requireCondition(browserSect.lastError === null, 'Browser promotion command left sect lastError non-null.', browserSect);
    requireCondition(promotedDisciple?.identity === 'inner_disciple', 'Authority promotion did not update disciple identity.', {
        promotedDisciple,
        afterPromotionRaw,
    });
    requireCondition(
        afterPromotionRaw.eventFeedbackCategories.includes('promotion') && afterPromotionRaw.recentPromotionEvents.length > 0,
        'Promotion replay did not expose event_log-derived promotion feedback.',
        afterPromotionRaw,
    );

    return evidence;
}

async function runSectBuildingMaintenanceReplay(connection, options, seedSnapshot, runId) {
    const userId = seedSnapshot.authority?.playerId ?? 'preview-player';
    const sessionId = seedSnapshot.authority?.sessionId ?? 'preview-player/preview-local';
    let response = await resetSectAuthority(options, seedSnapshot, {
        resources: {
            spirit_stone: 92,
        },
    });
    let raw = summarizeRawSectSnapshot(response.snapshot);
    requireCondition(raw.connected && !!raw.sectId, 'Sect authority did not reset before building maintenance replay.', raw);

    const context = {
        userId,
        sectId: raw.sectId,
        sessionId,
    };
    const commandPrefix = `gate-building-maintenance-${runId}`;
    let commandIndex = 0;
    const nextCommandId = (slug) => `${commandPrefix}-${String(commandIndex += 1).padStart(2, '0')}-${slug}`;
    const evidence = {
        startRaw: raw,
        setupCommands: [],
    };

    const executeHttpCommand = async (label, type, payload) => {
        const beforeVersion = raw.sceneVersion;
        response = await submitSectCommandHttp(
            options,
            context,
            sectCommand(nextCommandId(label), type, beforeVersion, payload),
        );
        raw = summarizeRawSectSnapshot(response.snapshot);
        const commandEvidence = {
            label,
            type,
            beforeVersion,
            afterVersion: raw.sceneVersion,
            resultStatus: response.result?.status ?? null,
            error: response.result?.error ?? null,
            raw,
        };
        evidence.setupCommands.push(commandEvidence);
        requireCondition(response.result?.status === 'COMMAND_RESULT_STATUS_ACCEPTED', `Building maintenance command ${label} was rejected.`, commandEvidence);
        requireCondition(raw.sceneVersion > beforeVersion, `Building maintenance command ${label} did not advance scene version.`, commandEvidence);
        return response.snapshot;
    };

    const advanceRawDays = async (label, days) => {
        response = await postJson(`${options.authorityUrl}/v1/authority/sect/debug/advance-days`, {
            userId,
            sectId: context.sectId,
            sessionId,
            days,
        });
        raw = summarizeRawSectSnapshot(response.snapshot);
        const advanceEvidence = {
            label,
            days,
            daysAdvanced: response.daysAdvanced ?? null,
            fromVersion: response.fromVersion ?? null,
            toVersion: response.toVersion ?? null,
            raw,
        };
        evidence.setupCommands.push(advanceEvidence);
        requireCondition(response.daysAdvanced === days, `Building maintenance day advance ${label} did not advance expected days.`, advanceEvidence);
        return response.snapshot;
    };

    await executeHttpCommand('build-main-hall', 'COMMAND_TYPE_BUILD_BUILDING', {
        definitionKey: 'main_hall',
        origin: { col: 4, row: 6 },
    });
    await executeHttpCommand('upgrade-main-hall', 'COMMAND_TYPE_UPGRADE_BUILDING', {
        buildingId: 'building-1',
    });
    await executeHttpCommand('build-warehouse', 'COMMAND_TYPE_BUILD_BUILDING', {
        definitionKey: 'warehouse',
        origin: { col: 5, row: 6 },
    });
    const warehouseID = rawBuildingByDefinition(raw, 'warehouse')?.id ?? null;
    requireCondition(!!warehouseID, 'Building maintenance replay did not create a warehouse building.', raw);
    evidence.afterBuildRaw = raw;

    let damagedRaw = raw;
    for (const days of [30, 30, 30]) {
        await advanceRawDays(`maintenance-pressure-${days}d`, days);
        damagedRaw = raw;
        if (rawBuildingByID(raw, warehouseID)?.phase === 'damaged') {
            break;
        }
    }
    const damagedWarehouse = rawBuildingByID(damagedRaw, warehouseID);
    evidence.damagedRaw = damagedRaw;
    evidence.damagedMetrics = buildingMaintenanceMetrics(damagedRaw, warehouseID);
    requireCondition(
        damagedWarehouse?.phase === 'damaged' &&
            damagedWarehouse.maintenanceDebt > 0 &&
            damagedWarehouse.efficiency < 100 &&
            damagedWarehouse.durability < 100,
        'Building maintenance replay did not expose authority-owned damage/efficiency pressure.',
        evidence.damagedMetrics,
    );

    await executeHttpCommand('publish-repair-funding-task', 'COMMAND_TYPE_PUBLISH_TASK', {
        kind: 'maintenance_repair_funding',
        title: `建筑修复资金 ${runId}`,
        description: 'dedicated browser gate repair funding task',
        priority: 90,
        requiredProgressDays: 1,
        risk: 0,
        maxAssignees: 1,
        minIdentity: 'outer_disciple',
        minRealm: 'mortal',
        requiredAptitude: {
            physique: 1,
        },
        contributionReward: 0,
        rewardResources: {
            spirit_stone: 40,
        },
    });
    await executeHttpCommand('assign-repair-funding-task', 'COMMAND_TYPE_ASSIGN_DISCIPLE_TASK', {
        taskId: 'task-1',
        discipleId: 'disciple-1',
        discipleIds: ['disciple-1'],
    });
    await advanceRawDays('complete-repair-funding-task', 1);
    evidence.beforeRepairRaw = raw;

    await executeHttpCommand('repair-warehouse', 'COMMAND_TYPE_REPAIR_BUILDING', {
        buildingId: warehouseID,
    });
    const repairedRaw = raw;
    const repairedWarehouse = rawBuildingByID(repairedRaw, warehouseID);
    evidence.repairedRaw = repairedRaw;
    evidence.repairedMetrics = buildingMaintenanceMetrics(repairedRaw, warehouseID);
    requireCondition(
        repairedWarehouse?.phase === 'active' &&
            repairedWarehouse.hp === repairedWarehouse.maxHp &&
            repairedWarehouse.maintenanceDebt === 0 &&
            repairedWarehouse.efficiency === 100 &&
            repairedWarehouse.durability === 100,
        'Building maintenance replay repair command did not restore authority building state.',
        evidence.repairedMetrics,
    );

    const restoredJoin = await postJson(`${options.authorityUrl}/v1/authority/sect/join`, context);
    const restoredRaw = summarizeRawSectSnapshot(restoredJoin.snapshot);
    evidence.restoredRaw = restoredRaw;
    requireCondition(
        JSON.stringify(buildingMaintenanceMetrics(restoredRaw, warehouseID).building) ===
            JSON.stringify(buildingMaintenanceMetrics(repairedRaw, warehouseID).building),
        'Building maintenance replay rejoin did not restore latest repaired building state.',
        {
            repaired: buildingMaintenanceMetrics(repairedRaw, warehouseID),
            restored: buildingMaintenanceMetrics(restoredRaw, warehouseID),
        },
    );

    await advanceRawDays('post-restore-offline-continuity', 30);
    const offlineRaw = raw;
    const offlineWarehouse = rawBuildingByID(offlineRaw, warehouseID);
    let browserSnapshot = await joinSectAuthority(connection);
    let browserSect = summarizeSectAuthority(browserSnapshot);
    evidence.offlineRaw = offlineRaw;
    evidence.browserAfterOffline = browserSect;
    evidence.metrics = {
        warehouseID,
        damagedBeforeRepair: damagedWarehouse?.phase === 'damaged',
        repairRestoredActive:
            repairedWarehouse?.phase === 'active' &&
            repairedWarehouse.maintenanceDebt === 0 &&
            repairedWarehouse.efficiency === 100 &&
            repairedWarehouse.durability === 100,
        restoredMatchesRepair:
            JSON.stringify(buildingMaintenanceMetrics(restoredRaw, warehouseID).building) ===
            JSON.stringify(buildingMaintenanceMetrics(repairedRaw, warehouseID).building),
        offlineSceneVersionAdvanced: offlineRaw.sceneVersion > restoredRaw.sceneVersion,
        offlineBuildingStillAuthoritative: !!offlineWarehouse && offlineWarehouse.id === warehouseID,
        browserLastError: browserSect.lastError,
        authorityLastError: browserSnapshot.authority?.lastError ?? null,
        browserBuildingSummary: browserSect.buildingSummary,
    };

    requireCondition(evidence.metrics.offlineSceneVersionAdvanced, 'Building maintenance offline advance did not move authority version.', evidence.metrics);
    requireCondition(evidence.metrics.offlineBuildingStillAuthoritative, 'Building maintenance offline advance lost warehouse building state.', {
        offlineRaw,
        warehouseID,
    });
    requireCondition(browserSect.lastError === null, 'Building maintenance browser snapshot ended with sect lastError.', browserSect);
    requireCondition(browserSnapshot.authority?.lastError === null, 'Building maintenance browser snapshot ended with authority lastError.', browserSnapshot.authority);

    return evidence;
}

async function runSectInstitutionManagementReplay(connection, options, seedSnapshot, runId) {
    const userId = seedSnapshot.authority?.playerId ?? 'preview-player';
    const sessionId = seedSnapshot.authority?.sessionId ?? 'preview-player/preview-local';
    let response = await resetSectAuthority(options, seedSnapshot);
    let raw = summarizeRawSectSnapshot(response.snapshot);
    requireCondition(raw.connected && !!raw.sectId, 'Sect authority did not reset before institution management replay.', raw);

    const context = {
        userId,
        sectId: raw.sectId,
        sessionId,
    };
    let browserSnapshot = await joinSectAuthority(connection);
    let browserSect = summarizeSectAuthority(browserSnapshot);
    const commandPrefix = `gate-institution-${runId}`;
    let commandIndex = 0;
    const nextCommandId = (slug) => `${commandPrefix}-${String(commandIndex += 1).padStart(2, '0')}-${slug}`;
    const evidence = {
        startRaw: raw,
        startBrowser: browserSect,
        commands: [],
    };

    const executeBrowserCommand = async (label, type, payload) => {
        const before = browserSect;
        browserSnapshot = await executeSectAuthorityCommand(
            connection,
            sectCommand(nextCommandId(label), type, before.sceneVersion, payload),
        );
        browserSect = summarizeSectAuthority(browserSnapshot);
        const commandEvidence = {
            label,
            type,
            beforeVersion: before.sceneVersion,
            afterVersion: browserSect.sceneVersion,
            lastError: browserSect.lastError,
        };
        evidence.commands.push(commandEvidence);
        requireCondition(browserSect.lastError === null, `Institution replay browser command ${label} left sect lastError non-null.`, commandEvidence);
        requireCondition(browserSect.sceneVersion > before.sceneVersion, `Institution replay browser command ${label} did not advance version.`, commandEvidence);
        return browserSnapshot;
    };

    for (const institutionId of ['task_hall', 'medicine_hut', 'treasury', 'dormitory', 'cave']) {
        await executeBrowserCommand(`assign-${institutionId}`, 'COMMAND_TYPE_ASSIGN_INSTITUTION_MANAGER', {
            institutionId,
            discipleId: 'disciple-1',
        });
    }
    await executeBrowserCommand('set-exchange-rule', 'COMMAND_TYPE_SET_EXCHANGE_RULE', {
        exchangeItemId: 'treasury-spirit-grain',
        contributionCost: 5,
        monthlyLimit: 10,
        enabled: true,
    });
    await executeBrowserCommand('reserve-cave', 'COMMAND_TYPE_RESERVE_CAVE', {
        discipleId: 'disciple-1',
        durationDays: 2,
    });

    response = await postJson(`${options.authorityUrl}/v1/authority/sect/join`, context);
    raw = summarizeRawSectSnapshot(response.snapshot);
    browserSnapshot = await joinSectAuthority(connection);
    browserSect = summarizeSectAuthority(browserSnapshot);
    const panelProbe = await runSectAffairsInstitutionPanelProbe(connection);
    const taskHall = raw.institutions.find((institution) => institution.id === 'task_hall') ?? null;
    const medicine = raw.institutions.find((institution) => institution.id === 'medicine_hut') ?? null;
    const treasury = raw.institutions.find((institution) => institution.id === 'treasury') ?? null;
    const cave = raw.institutions.find((institution) => institution.id === 'cave') ?? null;
    const managedInstitutionCount = raw.institutions.filter((institution) => !!institution.managerDiscipleId).length;
    const clientInstitutionDetails = (browserSect.institutions ?? []).filter(
        (institution) =>
            institution.managerText &&
            institution.capacityText &&
            institution.efficiencyText &&
            institution.effectSummaryText &&
            institution.operationHintText,
    );

    evidence.afterRaw = raw;
    evidence.afterBrowser = browserSect;
    evidence.panelProbe = panelProbe;
    evidence.metrics = {
        managedInstitutionCount,
        browserLastError: browserSect.lastError,
        authorityLastError: browserSnapshot.authority?.lastError ?? null,
        overviewPresent: !!browserSect.overview && panelProbe.overviewVisible,
        browserInstitutionDetailCount: clientInstitutionDetails.length,
        panelInstitutionEntryCount: panelProbe.institutionEntryCount,
        taskHallTaskCapacityBonus: taskHall?.taskCapacityBonus ?? null,
        medicineHealingPower: medicine?.healingPower ?? null,
        treasuryExchangePressure: treasury?.exchangePressure ?? null,
        caveSlotCount: cave?.caveSlots?.length ?? 0,
        caveOccupied: (cave?.caveSlots ?? []).some((slot) => slot.occupiedBy === 'disciple-1'),
    };

    requireCondition(evidence.metrics.overviewPresent, 'Institution replay did not expose sect affairs overview in browser panel.', evidence);
    requireCondition(
        evidence.metrics.browserInstitutionDetailCount >= 2 && evidence.metrics.panelInstitutionEntryCount >= 2,
        'Institution replay did not expose at least two institution detail entries.',
        evidence.metrics,
    );
    requireCondition(evidence.metrics.managedInstitutionCount >= 5, 'Institution replay did not preserve manager assignments in authority state.', evidence.metrics);
    requireCondition(
        evidence.metrics.taskHallTaskCapacityBonus > 0 &&
            evidence.metrics.medicineHealingPower > 0 &&
            evidence.metrics.caveSlotCount > 0 &&
            evidence.metrics.caveOccupied,
        'Institution replay did not expose task hall, medicine hut, and cave authority effects.',
        evidence.metrics,
    );
    requireCondition(browserSect.lastError === null, 'Institution replay browser snapshot ended with sect lastError.', browserSect);
    requireCondition(browserSnapshot.authority?.lastError === null, 'Institution replay browser snapshot ended with authority lastError.', browserSnapshot.authority);

    return evidence;
}

async function runSectExternalRiskReplay(connection, options, seedSnapshot, runId) {
    const userId = seedSnapshot.authority?.playerId ?? 'preview-player';
    const sessionId = seedSnapshot.authority?.sessionId ?? 'preview-player/preview-local';
    let response = await resetSectAuthority(options, seedSnapshot);
    let raw = summarizeRawSectSnapshot(response.snapshot);
    requireCondition(raw.connected && !!raw.sectId, 'Sect authority did not reset before external risk replay.', raw);

    let browserSnapshot = await joinSectAuthority(connection);
    let browserSect = summarizeSectAuthority(browserSnapshot);
    const context = {
        userId,
        sectId: raw.sectId,
        sessionId,
    };
    const evidence = {
        beforeRaw: raw,
        beforeBrowser: browserSect,
        selectedTask: null,
        afterDispatchRaw: null,
        afterDispatchBrowser: null,
        afterResultRaw: null,
        afterResultBrowser: null,
        metrics: {},
    };

    const selectedRawTask =
        (raw.tasks ?? []).find(
            (task) =>
                task.taskType === 'external' &&
                task.status === 'published' &&
                (task.recommendedDispatch?.recommendedDiscipleIds ?? []).length > 0,
        ) ?? null;
    requireCondition(!!selectedRawTask, 'External risk replay did not find a dispatchable raw authority external task.', raw);
    const selectedBrowserTask = (browserSect.tasks ?? []).find((task) => task.id === selectedRawTask.id) ?? null;
    const discipleIds = [...(selectedRawTask.recommendedDispatch?.recommendedDiscipleIds ?? [])].slice(0, selectedRawTask.maxAssignees || 1);
    requireCondition(discipleIds.length > 0, 'External risk replay selected task had no recommended authority disciples.', selectedBrowserTask);
    evidence.selectedTask = {
        browser: selectedBrowserTask,
        raw: selectedRawTask,
        discipleIds,
    };

    browserSnapshot = await executeSectAuthorityCommand(
        connection,
        sectCommand(`gate-external-dispatch-${runId}`, 'COMMAND_TYPE_ASSIGN_DISCIPLE_TASK', browserSect.sceneVersion, {
            taskId: selectedRawTask.id,
            discipleId: discipleIds[0],
            discipleIds,
        }),
    );
    browserSect = summarizeSectAuthority(browserSnapshot);
    response = await postJson(`${options.authorityUrl}/v1/authority/sect/join`, context);
    raw = summarizeRawSectSnapshot(response.snapshot);
    const afterDispatchRawTask = findRawTask(raw, selectedRawTask.id);
    const afterDispatchBrowserTask = (browserSect.tasks ?? []).find((task) => task.id === selectedRawTask.id) ?? null;
    evidence.afterDispatchRaw = raw;
    evidence.afterDispatchBrowser = browserSect;
    requireCondition(browserSect.lastError === null, 'External browser dispatch left sect lastError non-null.', browserSect);
    requireCondition(browserSnapshot.authority?.lastError === null, 'External browser dispatch left authority lastError non-null.', browserSnapshot.authority);
    requireCondition(afterDispatchRawTask?.status === 'accepted', 'External dispatch was not accepted by authority.', {
        afterDispatchRawTask,
        raw,
    });
    requireCondition((afterDispatchRawTask?.successRate ?? 0) > 0, 'External dispatch did not expose authority success rate.', afterDispatchRawTask);

    const daysToSettle = Math.max(1, afterDispatchRawTask.requiredProgressDays ?? selectedRawTask.requiredProgressDays ?? 1);
    response = await advanceSectDays(options, browserSnapshot, daysToSettle);
    raw = summarizeRawSectSnapshot(response.snapshot);
    browserSnapshot = await joinSectAuthority(connection);
    browserSect = summarizeSectAuthority(browserSnapshot);
    const resultRawTask = findRawTask(raw, selectedRawTask.id);
    const resultBrowserTask = (browserSect.tasks ?? []).find((task) => task.id === selectedRawTask.id) ?? null;
    const feedbackCategories = new Set(raw.eventFeedbackCategories ?? []);

    evidence.afterResultRaw = raw;
    evidence.afterResultBrowser = browserSect;
    evidence.metrics = {
        browserLastError: browserSect.lastError,
        authorityLastError: browserSnapshot.authority?.lastError ?? null,
        taskId: selectedRawTask.id,
        taskType: resultRawTask?.taskType ?? selectedRawTask.taskType,
        resultStatus: resultRawTask?.status ?? null,
        successRate: resultRawTask?.successRate ?? afterDispatchRawTask?.successRate ?? null,
        risk: resultRawTask?.risk ?? selectedRawTask.risk ?? null,
        evaluation: resultRawTask?.evaluation ?? null,
        rewardResources: resultRawTask?.rewardResources ?? {},
        reputationReward: resultRawTask?.reputationReward ?? 0,
        relationReward: resultRawTask?.relationReward ?? {},
        browserSuccessRateText: resultBrowserTask?.successRateText ?? afterDispatchBrowserTask?.successRateText ?? '',
        browserRiskText: resultBrowserTask?.riskText ?? afterDispatchBrowserTask?.riskText ?? '',
        browserEvaluationText: resultBrowserTask?.evaluationText ?? '',
        feedbackCategories: [...feedbackCategories].sort(),
        taskFeedbackCount: (raw.eventFeedback ?? []).filter((entry) => entry.category === 'task_result').length,
    };

    requireCondition(
        resultRawTask?.status === 'completed' || resultRawTask?.status === 'failed',
        'External task did not settle to an authority result after compressed day advance.',
        evidence.metrics,
    );
    requireCondition((evidence.metrics.successRate ?? 0) > 0, 'External result did not preserve success rate evidence.', evidence.metrics);
    requireCondition((evidence.metrics.risk ?? -1) >= 0, 'External result did not preserve risk evidence.', evidence.metrics);
    requireCondition(!!evidence.metrics.evaluation, 'External result did not preserve evaluation evidence.', evidence.metrics);
    requireCondition(feedbackCategories.has('task_result'), 'External result did not emit bounded task_result feedback.', {
        feedback: raw.eventFeedback,
        categories: evidence.metrics.feedbackCategories,
    });
    requireCondition(browserSect.lastError === null, 'External replay browser snapshot ended with sect lastError.', browserSect);
    requireCondition(browserSnapshot.authority?.lastError === null, 'External replay browser snapshot ended with authority lastError.', browserSnapshot.authority);

    return evidence;
}

async function runSectExpansionRiskReplay(connection, options, seedSnapshot, runId) {
    const userId = seedSnapshot.authority?.playerId ?? 'preview-player';
    const sessionId = seedSnapshot.authority?.sessionId ?? 'preview-player/preview-local';
    let response = await resetSectAuthority(options, seedSnapshot, {
        resources: {
            spirit_stone: 320,
            ore: 140,
            herb: 36,
            formation_mat: 24,
            beast_mat: 24,
            spirit_grain: 80,
        },
    });
    let raw = summarizeRawSectSnapshot(response.snapshot);
    requireCondition(raw.connected && !!raw.sectId, 'Sect authority did not reset before expansion-risk replay.', raw);

    const context = {
        userId,
        sectId: raw.sectId,
        sessionId,
    };
    const evidence = {
        startRaw: raw,
        commands: [],
        dayAdvances: [],
        afterSetupRaw: null,
        finalRaw: null,
        rejoinRaw: null,
        offlineRaw: null,
        finalBrowser: null,
        panelProbe: null,
        metrics: {},
    };
    const commandPrefix = `gate-expansion-risk-${runId}`;
    let commandIndex = 0;
    const nextCommandId = (slug) => `${commandPrefix}-${String(commandIndex += 1).padStart(2, '0')}-${slug}`;

    const recordPeaks = (peaks, sect) => {
        peaks.maxGoalCount = Math.max(peaks.maxGoalCount, sect.goalCount ?? 0);
        peaks.maxResolvedGoalCount = Math.max(peaks.maxResolvedGoalCount, sect.resolvedGoalCount ?? 0);
        peaks.maxCrisisCount = Math.max(peaks.maxCrisisCount, sect.crisisCount ?? 0);
        peaks.maxTreatmentCount = Math.max(peaks.maxTreatmentCount, sect.treatmentCount ?? 0);
        peaks.maxMonthlyMonthIndex = Math.max(peaks.maxMonthlyMonthIndex, sect.monthlyAssessment?.lastMonthIndex ?? 0);
        peaks.maxSceneVersion = Math.max(peaks.maxSceneVersion, sect.sceneVersion ?? 0);
    };

    const peaks = {
        maxGoalCount: raw.goalCount ?? 0,
        maxResolvedGoalCount: raw.resolvedGoalCount ?? 0,
        maxCrisisCount: raw.crisisCount ?? 0,
        maxTreatmentCount: raw.treatmentCount ?? 0,
        maxMonthlyMonthIndex: raw.monthlyAssessment?.lastMonthIndex ?? 0,
        maxSceneVersion: raw.sceneVersion ?? 0,
    };

    const execute = async (label, type, payload) => {
        const beforeVersion = raw.sceneVersion;
        response = await submitSectCommandHttp(
            options,
            context,
            sectCommand(nextCommandId(label), type, beforeVersion, payload),
        );
        raw = summarizeRawSectSnapshot(response.snapshot);
        recordPeaks(peaks, raw);
        const commandEvidence = {
            label,
            type,
            beforeVersion,
            afterVersion: raw.sceneVersion,
            resultStatus: response.result?.status ?? null,
            error: response.result?.error ?? null,
            raw,
        };
        evidence.commands.push(commandEvidence);
        requireCondition(response.result?.status === 'COMMAND_RESULT_STATUS_ACCEPTED', `Expansion-risk command ${label} was rejected.`, commandEvidence);
        requireCondition(raw.sceneVersion > beforeVersion, `Expansion-risk command ${label} did not advance scene version.`, commandEvidence);
        return response.snapshot;
    };

    const advance = async (label, days) => {
        response = await postJson(`${options.authorityUrl}/v1/authority/sect/debug/advance-days`, {
            userId,
            sectId: context.sectId,
            sessionId,
            days,
        });
        raw = summarizeRawSectSnapshot(response.snapshot);
        recordPeaks(peaks, raw);
        const advanceEvidence = {
            label,
            days,
            daysAdvanced: response.daysAdvanced ?? null,
            fromVersion: response.fromVersion ?? null,
            toVersion: response.toVersion ?? null,
            raw,
        };
        evidence.dayAdvances.push(advanceEvidence);
        requireCondition(response.daysAdvanced === days, `Expansion-risk day advance ${label} did not advance expected days.`, advanceEvidence);
        return response.snapshot;
    };

    for (const policy of [
        ['task', 'combat'],
        ['resource', 'war_preparation'],
        ['recruitment', 'affiliated'],
        ['cultivation', 'closed_cultivation'],
    ]) {
        await execute(`set-policy-${policy[0]}`, 'COMMAND_TYPE_SET_POLICY', {
            policyCategory: policy[0],
            policyValue: policy[1],
        });
    }

    await execute('build-main-hall', 'COMMAND_TYPE_BUILD_BUILDING', {
        definitionKey: 'main_hall',
        origin: { col: 4, row: 6 },
    });
    await execute('upgrade-main-hall', 'COMMAND_TYPE_UPGRADE_BUILDING', {
        buildingId: 'building-1',
    });
    await execute('build-gate', 'COMMAND_TYPE_BUILD_BUILDING', {
        definitionKey: 'gate',
        origin: { col: 7, row: 6 },
    });
    const gateID = rawBuildingByDefinition(raw, 'gate')?.id ?? null;
    requireCondition(!!gateID, 'Expansion-risk replay did not create a gate building.', raw);
    await execute('upgrade-gate', 'COMMAND_TYPE_UPGRADE_BUILDING', {
        buildingId: gateID,
    });
    await execute('build-warehouse', 'COMMAND_TYPE_BUILD_BUILDING', {
        definitionKey: 'warehouse',
        origin: { col: 5, row: 6 },
    });
    const warehouseID = rawBuildingByDefinition(raw, 'warehouse')?.id ?? null;
    requireCondition(!!warehouseID, 'Expansion-risk replay did not create a warehouse building.', raw);

    await execute('start-recruitment', 'COMMAND_TYPE_START_RECRUITMENT', {
        candidateCount: 2,
        investmentSpiritStone: 12,
        durationDays: 5,
    });
    const candidates = [...(raw.candidates ?? [])].sort((left, right) => String(left.id ?? '').localeCompare(String(right.id ?? '')));
    requireCondition(candidates.length >= 2, 'Expansion-risk recruitment did not expose two candidates.', raw);
    await execute('accept-candidate', 'COMMAND_TYPE_ACCEPT_CANDIDATE', { candidateId: candidates[0].id });
    await execute('reject-candidate', 'COMMAND_TYPE_REJECT_CANDIDATE', { candidateId: candidates[1].id });
    const acceptedDisciple = rawDiscipleByName(raw, candidates[0].name);
    requireCondition(!!acceptedDisciple, 'Expansion-risk accepted candidate did not enter roster.', raw);

    await execute('assign-task-manager', 'COMMAND_TYPE_ASSIGN_INSTITUTION_MANAGER', {
        institutionId: 'task_hall',
        discipleId: 'disciple-1',
    });
    await execute('assign-medicine-manager', 'COMMAND_TYPE_ASSIGN_INSTITUTION_MANAGER', {
        institutionId: 'medicine_hut',
        discipleId: acceptedDisciple.id,
    });
    await execute('craft-sword', 'COMMAND_TYPE_CRAFT_ARTIFACT', { artifactType: 'sword', quality: 1 });
    await execute('craft-farm-tool', 'COMMAND_TYPE_CRAFT_ARTIFACT', { artifactType: 'farm_tool', quality: 1 });
    await execute('craft-formation-disk', 'COMMAND_TYPE_CRAFT_ARTIFACT', { artifactType: 'formation_disk', quality: 1 });
    const sword = (raw.artifacts ?? []).find((artifact) => artifact.type === 'sword') ?? null;
    const tool = (raw.artifacts ?? []).find((artifact) => artifact.type === 'farm_tool') ?? null;
    const formationDisk = (raw.artifacts ?? []).find((artifact) => artifact.type === 'formation_disk') ?? null;
    requireCondition(!!sword && !!tool && !!formationDisk, 'Expansion-risk replay did not expose crafted sword/tool/formation disk.', raw.artifacts);
    await execute('equip-sword', 'COMMAND_TYPE_EQUIP_ARTIFACT', {
        itemId: sword.id,
        discipleId: 'disciple-1',
    });
    await execute('equip-farm-tool', 'COMMAND_TYPE_EQUIP_ARTIFACT', {
        itemId: tool.id,
        discipleId: 'disciple-1',
    });
    await execute('start-production', 'COMMAND_TYPE_START_PRODUCTION', {
        recipeId: 'formation_refine_mvp',
        priority: 78,
        targetCycles: 1,
    });

    const merchantTaskTitle = `扩张验收外务 ${runId}`;
    await execute('publish-merchant-task', 'COMMAND_TYPE_PUBLISH_TASK', {
        kind: 'merchant_commission',
        title: merchantTaskTitle,
        description: 'dedicated browser gate compressed expansion external task',
        priority: 88,
        requiredProgressDays: 1,
        risk: 18,
        maxAssignees: 1,
        minIdentity: 'outer_disciple',
        minRealm: 'mortal',
        requiredAptitude: {
            physique: 3,
        },
        contributionReward: 30,
        rewardResources: {
            spirit_stone: 8,
            herb: 2,
        },
    });
    const merchantTask = findTaskByTitlePrefix(raw, merchantTaskTitle);
    requireCondition(!!merchantTask, 'Expansion-risk replay could not resolve published merchant task.', raw.tasks);
    await execute('assign-merchant-task', 'COMMAND_TYPE_ASSIGN_DISCIPLE_TASK', {
        taskId: merchantTask.id,
        discipleId: acceptedDisciple.id,
        discipleIds: [acceptedDisciple.id],
    });

    let governanceChoiceHandled = false;
    let crisisChoiceHandled = false;
    let breakthroughAttempted = false;
    let promotionIssued = false;
    let combatTaskPublished = false;
    let starterCombatTaskID = null;
    let repairFundingPublished = false;
    let warehouseRepaired = false;
    let formationMaintained = false;
    let defenseConfigured = false;
    let cultivationConfigured = false;

    for (let day = 1; day <= 60; day += 1) {
        await advance(`expansion-risk-day-${day}`, 1);

        if (!governanceChoiceHandled) {
            const governanceEvent = (raw.eventChoices ?? []).find((event) => event.kind === 'sect_governance_choice') ?? null;
            if (governanceEvent) {
                await execute('choose-governance', 'COMMAND_TYPE_CHOOSE_EVENT_OPTION', {
                    eventId: governanceEvent.id,
                    optionId: 'send_aid',
                });
                governanceChoiceHandled = true;
            }
        }

        const merchantTaskState = findRawTask(raw, merchantTask.id);
        if (!combatTaskPublished && governanceChoiceHandled && merchantTaskState?.status === 'completed') {
            const acceptedCombatTitle = `扩张验收战斗甲 ${runId}`;
            await execute('publish-combat-task-accepted', 'COMMAND_TYPE_PUBLISH_TASK', {
                kind: 'perimeter_watch',
                type: 'combat',
                title: acceptedCombatTitle,
                description: 'dedicated browser gate compressed crisis combat task for accepted disciple',
                priority: 92,
                requiredProgressDays: 10,
                risk: 88,
                maxAssignees: 1,
                minIdentity: 'outer_disciple',
                minRealm: 'mortal',
                requiredAptitude: {
                    physique: 2,
                },
                contributionReward: 12,
                rewardResources: {
                    beast_mat: 4,
                },
            });
            const acceptedCombatTask = findTaskByTitlePrefix(raw, acceptedCombatTitle);
            requireCondition(!!acceptedCombatTask, 'Expansion-risk replay could not resolve accepted-disciple combat task.', raw.tasks);
            await execute('assign-combat-task-accepted', 'COMMAND_TYPE_ASSIGN_DISCIPLE_TASK', {
                taskId: acceptedCombatTask.id,
                discipleId: acceptedDisciple.id,
                discipleIds: [acceptedDisciple.id],
            });

            const starterCombatTitle = `扩张验收战斗乙 ${runId}`;
            await execute('publish-combat-task-starter', 'COMMAND_TYPE_PUBLISH_TASK', {
                kind: 'perimeter_watch',
                type: 'combat',
                title: starterCombatTitle,
                description: 'dedicated browser gate compressed crisis combat task for starter disciple',
                priority: 90,
                requiredProgressDays: 8,
                risk: 82,
                maxAssignees: 1,
                minIdentity: 'outer_disciple',
                minRealm: 'mortal',
                requiredAptitude: {
                    physique: 1,
                },
                contributionReward: 10,
                rewardResources: {
                    beast_mat: 3,
                },
            });
            const starterCombatTask = findTaskByTitlePrefix(raw, starterCombatTitle);
            requireCondition(!!starterCombatTask, 'Expansion-risk replay could not resolve starter combat task.', raw.tasks);
            await execute('assign-combat-task-starter', 'COMMAND_TYPE_ASSIGN_DISCIPLE_TASK', {
                taskId: starterCombatTask.id,
                discipleId: 'disciple-1',
                discipleIds: ['disciple-1'],
            });
            starterCombatTaskID = starterCombatTask.id;
            combatTaskPublished = true;
        }

        if (!crisisChoiceHandled) {
            const crisisEvent = (raw.eventChoices ?? []).find((event) => event.kind === 'sect_crisis_choice') ?? null;
            if (crisisEvent) {
                await execute('choose-crisis', 'COMMAND_TYPE_CHOOSE_EVENT_OPTION', {
                    eventId: crisisEvent.id,
                    optionId: 'fortify_perimeter',
                });
                crisisChoiceHandled = true;
            }
        }

        if (!defenseConfigured && (crisisChoiceHandled || day >= 20)) {
            await execute('set-gate-policy', 'COMMAND_TYPE_SET_GATE_POLICY', {
                enforcementStrictness: 2,
                guardDiscipleIds: ['disciple-1'],
            });
            await execute('attach-defense-formation', 'COMMAND_TYPE_ATTACH_FORMATION_TO_BUILDING', {
                buildingId: gateID,
                artifactItemId: formationDisk.id,
                formationKind: 'defense',
            });
            const attachedFormation = rawFormationByBuilding(raw, gateID);
            requireCondition(!!attachedFormation && attachedFormation.active, 'Expansion-risk replay did not attach an active defense formation.', {
                gateID,
                formations: raw.formations,
            });
            defenseConfigured = true;
        }

        const starterCombatTaskState = starterCombatTaskID ? findRawTask(raw, starterCombatTaskID) : null;
        if (
            !cultivationConfigured &&
            (crisisChoiceHandled || day >= 20) &&
            (!starterCombatTaskState || starterCombatTaskState.status !== 'accepted')
        ) {
            await execute('reserve-cave', 'COMMAND_TYPE_RESERVE_CAVE', {
                discipleId: 'disciple-1',
                durationDays: 2,
            });
            await execute('start-cultivation', 'COMMAND_TYPE_START_CULTIVATION', {
                discipleId: 'disciple-1',
            });
            cultivationConfigured = true;
        }

        if (!breakthroughAttempted && raw.cultivation?.readyForBreakthrough && raw.cultivation?.omenStatus === 'foreshadowed') {
            await execute('attempt-breakthrough', 'COMMAND_TYPE_ATTEMPT_BREAKTHROUGH', {
                discipleId: 'disciple-1',
            });
            breakthroughAttempted = true;
        }

        const starter = rawDiscipleByID(raw, 'disciple-1');
        if (
            breakthroughAttempted &&
            !promotionIssued &&
            starter &&
            starter.identity === 'outer_disciple' &&
            starter.realmStage === 'qi_entry' &&
            (starter.contributionBalance ?? 0) >= 20 &&
            (starter.loyalty ?? 0) >= 60 &&
            (starter.satisfaction ?? 0) >= 55 &&
            (starter.hp ?? 0) > 0
        ) {
            await execute('start-assessment', 'COMMAND_TYPE_START_ASSESSMENT', {
                discipleId: 'disciple-1',
                targetRank: 'inner_disciple',
            });
            const assessedStarter = rawDiscipleByID(raw, 'disciple-1');
            if (assessedStarter?.assessment?.passed) {
                await execute('promote-disciple', 'COMMAND_TYPE_PROMOTE_DISCIPLE', {
                    discipleId: 'disciple-1',
                    targetRank: 'inner_disciple',
                });
                promotionIssued = true;
            }
        }

        const warehouse = rawBuildingByID(raw, warehouseID);
        if (!repairFundingPublished && day >= 30 && warehouse?.phase === 'damaged') {
            const fundingTaskTitle = `扩张修缮资金 ${runId}`;
            await execute('publish-repair-funding-task', 'COMMAND_TYPE_PUBLISH_TASK', {
                kind: 'maintenance_repair_funding',
                title: fundingTaskTitle,
                description: 'dedicated browser gate expansion repair funding task',
                priority: 95,
                requiredProgressDays: 1,
                risk: 0,
                maxAssignees: 1,
                minIdentity: 'outer_disciple',
                minRealm: 'mortal',
                requiredAptitude: {
                    physique: 1,
                },
                contributionReward: 0,
                rewardResources: {
                    spirit_stone: 40,
                },
            });
            const fundingTask = findTaskByTitlePrefix(raw, fundingTaskTitle);
            requireCondition(!!fundingTask, 'Expansion-risk replay could not resolve repair funding task.', raw.tasks);
            await execute('assign-repair-funding-task', 'COMMAND_TYPE_ASSIGN_DISCIPLE_TASK', {
                taskId: fundingTask.id,
                discipleId: acceptedDisciple.id,
                discipleIds: [acceptedDisciple.id],
            });
            repairFundingPublished = true;
        }

        if (!warehouseRepaired && warehouse?.phase === 'damaged' && Number(raw.resources?.spirit_stone ?? 0) >= Math.max(1, Number(warehouse.maintenanceDebt ?? 0))) {
            await execute('repair-warehouse', 'COMMAND_TYPE_REPAIR_BUILDING', {
                buildingId: warehouseID,
            });
            warehouseRepaired = true;
        }

        const formation = rawFormationByBuilding(raw, gateID);
        if (
            !formationMaintained &&
            formation &&
            (!formation.active || Number(formation.maintenanceDebt ?? 0) > 0) &&
            Number(raw.resources?.spirit_stone ?? 0) >= 6
        ) {
            await execute('maintain-formation', 'COMMAND_TYPE_MAINTAIN_FORMATION', {
                buildingId: gateID,
            });
            formationMaintained = true;
        }
    }

    evidence.afterSetupRaw = sectExpansionContinuityComparable(raw);
    evidence.finalRaw = raw;
    const rejoin = await postJson(`${options.authorityUrl}/v1/authority/sect/join`, {
        userId: context.userId,
        sectId: context.sectId,
        sessionId: `${context.sessionId}:expansion-risk-rejoin`,
    });
    const rejoinRaw = summarizeRawSectSnapshot(rejoin.snapshot);
    evidence.rejoinRaw = rejoinRaw;
    requireCondition(
        JSON.stringify(sectExpansionContinuityComparable(rejoinRaw)) === JSON.stringify(sectExpansionContinuityComparable(raw)),
        'Expansion-risk replay rejoin did not preserve bounded authority state.',
        {
            final: sectExpansionContinuityComparable(raw),
            rejoin: sectExpansionContinuityComparable(rejoinRaw),
        },
    );

    await advance('expansion-risk-offline-window', 30);
    evidence.offlineRaw = raw;

    const browserSnapshot = await joinSectAuthority(connection);
    const browserSect = summarizeSectAuthority(browserSnapshot);
    const panelProbe = await runSectExpansionRiskPanelProbe(connection);
    evidence.finalBrowser = browserSect;
    evidence.panelProbe = panelProbe;

    const mainHall = rawBuildingByDefinition(raw, 'main_hall');
    const gate = rawBuildingByDefinition(raw, 'gate');
    const warehouse = rawBuildingByDefinition(raw, 'warehouse');
    const resolvedEventKinds = new Set((raw.resolvedEvents ?? []).map((event) => event.kind));
    const resolvedEventOutcomes = (raw.resolvedEvents ?? []).map((event) => event.outcome);
    evidence.metrics = {
        governanceChoiceHandled,
        crisisChoiceHandled,
        breakthroughAttempted,
        promotionIssued,
        repairFundingPublished,
        warehouseRepaired,
        formationMaintained,
        mainHallLevel: mainHall?.level ?? 0,
        gateLevel: gate?.level ?? 0,
        warehousePhase: warehouse?.phase ?? null,
        formationCount: raw.formationCount ?? 0,
        formationActive: rawFormationByBuilding(raw, gateID)?.active ?? false,
        resolvedGoalCount: raw.resolvedGoalCount ?? 0,
        resolvedGoalOutcomes: (raw.resolvedGoals ?? []).map((goal) => goal.outcome),
        maxGoalCount: peaks.maxGoalCount,
        maxResolvedGoalCount: peaks.maxResolvedGoalCount,
        maxCrisisCount: peaks.maxCrisisCount,
        maxTreatmentCount: peaks.maxTreatmentCount,
        maxMonthlyMonthIndex: peaks.maxMonthlyMonthIndex,
        order: raw.order,
        monthlyAssessment: raw.monthlyAssessment,
        resolvedEventKinds: [...resolvedEventKinds].sort(),
        resolvedEventOutcomes,
        browserLastError: browserSect.lastError,
        authorityLastError: browserSnapshot.authority?.lastError ?? null,
    };

    requireCondition((mainHall?.level ?? 0) >= 2 && (gate?.level ?? 0) >= 2, 'Expansion-risk replay did not upgrade sect core buildings.', evidence.metrics);
    requireCondition((raw.institutionCount ?? 0) >= 5, 'Expansion-risk replay did not retain bounded institution state.', raw.institutions);
    requireCondition((raw.artifactCount ?? 0) >= 3, 'Expansion-risk replay did not retain authority artifact inventory.', raw.artifacts);
    requireCondition((raw.formationCount ?? 0) >= 1, 'Expansion-risk replay did not retain attached formation state.', raw.formations);
    requireCondition((raw.resolvedGoalCount ?? 0) >= 2 || peaks.maxResolvedGoalCount >= 2, 'Expansion-risk replay did not resolve enough authority goals.', {
        goals: raw.goals,
        resolvedGoals: raw.resolvedGoals,
        peaks,
    });
    requireCondition(peaks.maxMonthlyMonthIndex >= 1, 'Expansion-risk replay did not expose monthly assessment progression.', evidence.metrics);
    requireCondition(governanceChoiceHandled, 'Expansion-risk replay did not handle governance event choice.', evidence.metrics);
    requireCondition(crisisChoiceHandled || resolvedEventKinds.has('sect_crisis_choice'), 'Expansion-risk replay did not handle crisis-choice authority path.', evidence.metrics);
    requireCondition(peaks.maxTreatmentCount > 0, 'Expansion-risk replay did not expose any authority treatment queue evidence.', evidence.metrics);
    requireCondition((raw.order?.safety ?? 0) > 0 && (raw.order?.discipline ?? 0) > 0, 'Expansion-risk replay did not preserve bounded order state.', evidence.metrics);
    requireCondition(
        panelProbe.sectPanelOpened &&
            panelProbe.sectPanelRestoredClosed &&
            panelProbe.textSample.includes('经营总览') &&
            panelProbe.textSample.includes('宗门等级') &&
            panelProbe.textSample.includes('政策'),
        'Expansion-risk replay could not reopen the sect panel and expose the bounded management surface.',
        panelProbe,
    );
    requireCondition(
        browserSect.connected &&
            browserSect.sectId === raw.sectId &&
            (browserSect.rosterCount ?? 0) >= 2 &&
            (browserSect.institutionCount ?? 0) >= 5 &&
            browserSect.lastError === null,
        'Expansion-risk replay browser summary did not retain bounded visible sect-management surface.',
        browserSect,
    );
    requireCondition(browserSect.lastError === null, 'Expansion-risk replay browser sect snapshot ended with lastError.', browserSect);
    requireCondition(browserSnapshot.authority?.lastError === null, 'Expansion-risk replay authority snapshot ended with lastError.', browserSnapshot.authority);
    requireCondition(
        raw.sceneVersion > rejoinRaw.sceneVersion &&
            (raw.buildingCount ?? 0) === (rejoinRaw.buildingCount ?? 0) &&
            (raw.formationCount ?? 0) === (rejoinRaw.formationCount ?? 0) &&
            (raw.rosterCount ?? 0) === (rejoinRaw.rosterCount ?? 0),
        'Expansion-risk replay offline continuity window did not preserve bounded authority structures.',
        {
            rejoin: sectExpansionContinuityComparable(rejoinRaw),
            offline: sectExpansionContinuityComparable(raw),
        },
    );

    return evidence;
}

async function main() {
    const options = parseArgs(process.argv);
    ensureDirectory(options.outputDir);

    const target = await findOrOpenPreviewTarget(options);
    requireCondition(target.webSocketDebuggerUrl, `Target ${target.url} does not expose a page websocket.`);

    const connection = new CdpConnection(target.webSocketDebuggerUrl);
    const runId = `dedicated-browser-replay-gate-${slugTimestamp()}`;
    const evidence = {
        runId,
        ts: nowIso(),
        browserUrl: options.browserUrl,
        previewUrl: options.previewUrl,
        target: {
            id: target.id ?? target.targetId ?? null,
            title: target.title ?? null,
            url: target.url ?? null,
        },
        checkpoints: {},
        playerFacing: {},
        assertions: {},
    };

    try {
        await connection.connect();
        await reloadPreviewPage(connection);
        let initialRuntimeSnapshot = await waitForRuntimeSnapshot(connection, options);
        if (!initialRuntimeSnapshot.authority?.connected || initialRuntimeSnapshot.authority?.lastError !== null) {
            initialRuntimeSnapshot = await connection.evaluate(runtimeCallExpression('resetAuthoritySession'));
            requireCondition(
                initialRuntimeSnapshot.authority?.connected && initialRuntimeSnapshot.authority?.lastError === null,
                'Dedicated browser gate could not establish a clean authority session before replay.',
                summarizeSnapshot(initialRuntimeSnapshot),
            );
        }

        const sectMultiday = await runSectMultidayManagementReplay(connection, options, initialRuntimeSnapshot, runId);
        evidence.playerFacing.sectMultiday = sectMultiday;
        evidence.checkpoints.sect_multiday_start = sectMultiday.start;
        evidence.checkpoints.sect_multiday_final = sectMultiday.final;
        evidence.assertions.sect_multiday_authority_connected = sectMultiday.start.connected && sectMultiday.final.connected;
        evidence.assertions.sect_multiday_recruitment_roster = sectMultiday.final.rosterCount >= 2;
        evidence.assertions.sect_multiday_task_result = sectMultiday.feedbackCategories.includes('task_result');
        evidence.assertions.sect_multiday_production = sectMultiday.feedbackCategories.includes('production');
        evidence.assertions.sect_multiday_cultivation = sectMultiday.feedbackCategories.includes('cultivation');
        evidence.assertions.sect_multiday_monthly = sectMultiday.feedbackCategories.includes('monthly');
        evidence.assertions.sect_multiday_event_feedback = ['task_result', 'resource_change', 'production', 'cultivation', 'breakthrough', 'monthly', 'omen'].every(
            (category) => sectMultiday.feedbackCategories.includes(category),
        );
        evidence.checkpoints.sect_multiday_story_feedback = sectMultiday.storyFeedback;
        evidence.assertions.sect_multiday_story_feedback_surface =
            sectMultiday.storyFeedback.storyPanelProbe.rosterVisible &&
            sectMultiday.storyFeedback.storyPanelProbe.storySectionVisible &&
            sectMultiday.storyFeedback.storyPanelProbe.diarySectionVisible &&
            sectMultiday.storyFeedback.storyPanelProbe.eventFeedbackSectionVisible;
        evidence.assertions.sect_multiday_story_feedback_types = sectMultiday.storyFeedback.readableStoryFeedbackTypes.length >= 3;
        evidence.assertions.sect_multiday_restore_offline_continuity_gate = true;

        const sectEventChoice = await runSectEventChoiceReplay(connection, options, initialRuntimeSnapshot, runId);
        evidence.playerFacing.sectEventChoice = sectEventChoice;
        evidence.checkpoints.sect_event_choice_before = sectEventChoice.before;
        evidence.checkpoints.sect_event_choice_after = sectEventChoice.after;
        evidence.assertions.sect_event_choice_panel_opens = sectEventChoice.panelProbe.sectPanelOpened;
        evidence.assertions.sect_event_choice_authority_result = !!sectEventChoice.after.resolvedEvents.find(
            (event) => event.outcome === `option:${sectEventChoice.selectedOption.id}`,
        );
        evidence.assertions.sect_event_choice_feedback = sectEventChoice.after.eventFeedbackCategories.includes('omen');

        const sectPolicyImpact = await runSectPolicyImpactReplay(connection, options, initialRuntimeSnapshot, runId);
        evidence.playerFacing.sectPolicyImpact = sectPolicyImpact;
        evidence.checkpoints.sect_policy_start = sectPolicyImpact.startRaw;
        evidence.checkpoints.sect_policy_after_switch = sectPolicyImpact.afterPolicySwitchRaw;
        evidence.checkpoints.sect_policy_after_thirty_days = sectPolicyImpact.afterThirtyDaysRaw;
        evidence.assertions.sect_policy_browser_switch =
            sectPolicyImpact.afterPolicySwitchRaw.policies.taskPolicy === 'production' &&
            sectPolicyImpact.afterPolicySwitchRaw.policies.resourcePolicy === 'war_preparation' &&
            sectPolicyImpact.afterPolicySwitchRaw.policies.recruitmentPolicy === 'selective' &&
            sectPolicyImpact.afterPolicySwitchRaw.policies.cultivationPolicy === 'closed_cultivation';
        evidence.assertions.sect_policy_multiday_metrics =
            sectPolicyImpact.metrics.formationTaskPriority > 52 &&
            sectPolicyImpact.metrics.formationProductionPriority > 54 &&
            sectPolicyImpact.metrics.candidateCount === 2 &&
            sectPolicyImpact.metrics.averageCandidateSpiritRoot >= 10 &&
            sectPolicyImpact.metrics.cultivationPoints > 0 &&
            sectPolicyImpact.metrics.policyEventCount >= 4;
        evidence.assertions.sect_policy_last_error_null = sectPolicyImpact.metrics.browserLastError === null;

        const sectArtifactImpact = await runSectArtifactImpactReplay(connection, options, initialRuntimeSnapshot, runId);
        evidence.playerFacing.sectArtifactImpact = sectArtifactImpact;
        evidence.checkpoints.sect_artifact_before = sectArtifactImpact.beforeBrowser;
        evidence.checkpoints.sect_artifact_after_equip = sectArtifactImpact.afterBrowser;
        evidence.assertions.sect_artifact_browser_command =
            sectArtifactImpact.metrics.afterArtifactCount >= sectArtifactImpact.metrics.beforeArtifactCount + 2 &&
            sectArtifactImpact.metrics.swordBoundDiscipleId === 'disciple-1' &&
            sectArtifactImpact.metrics.toolBoundDiscipleId === 'disciple-1';
        evidence.assertions.sect_artifact_metric_delta =
            (sectArtifactImpact.metrics.combatRateBefore !== null &&
                sectArtifactImpact.metrics.combatRateAfter !== null &&
                sectArtifactImpact.metrics.combatRateAfter > sectArtifactImpact.metrics.combatRateBefore) ||
            (sectArtifactImpact.metrics.productionRateBefore !== null &&
                sectArtifactImpact.metrics.productionRateAfter !== null &&
                sectArtifactImpact.metrics.productionRateAfter > sectArtifactImpact.metrics.productionRateBefore);
        evidence.assertions.sect_artifact_last_error_null =
            sectArtifactImpact.metrics.browserLastError === null && sectArtifactImpact.metrics.authorityLastError === null;

        const sectPromotion = await runSectPromotionReplay(connection, options, initialRuntimeSnapshot, runId);
        evidence.playerFacing.sectPromotion = sectPromotion;
        evidence.checkpoints.sect_promotion_before_assessment = sectPromotion.beforeAssessmentRaw;
        evidence.checkpoints.sect_promotion_after_assessment = sectPromotion.afterAssessmentRaw;
        evidence.checkpoints.sect_promotion_after_promote = sectPromotion.afterPromotionRaw;
        evidence.assertions.sect_promotion_assessment_browser_command =
            sectPromotion.metrics.assessmentPassed === true ||
            rawDiscipleByID(sectPromotion.afterAssessmentRaw, 'disciple-1')?.assessment?.passed === true;
        evidence.assertions.sect_promotion_authority_result = sectPromotion.metrics.identity === 'inner_disciple';
        evidence.assertions.sect_promotion_feedback = sectPromotion.metrics.promotionFeedbackCount > 0;
        evidence.assertions.sect_promotion_last_error_null = sectPromotion.metrics.browserLastError === null;

        const sectBuildingMaintenance = await runSectBuildingMaintenanceReplay(connection, options, initialRuntimeSnapshot, runId);
        evidence.playerFacing.sectBuildingMaintenance = sectBuildingMaintenance;
        evidence.checkpoints.sect_building_maintenance_after_build = sectBuildingMaintenance.afterBuildRaw;
        evidence.checkpoints.sect_building_maintenance_damaged = sectBuildingMaintenance.damagedRaw;
        evidence.checkpoints.sect_building_maintenance_repaired = sectBuildingMaintenance.repairedRaw;
        evidence.checkpoints.sect_building_maintenance_restored = sectBuildingMaintenance.restoredRaw;
        evidence.checkpoints.sect_building_maintenance_offline = sectBuildingMaintenance.offlineRaw;
        evidence.assertions.sect_building_maintenance_pressure = sectBuildingMaintenance.metrics.damagedBeforeRepair;
        evidence.assertions.sect_building_damage_repair = sectBuildingMaintenance.metrics.repairRestoredActive;
        evidence.assertions.sect_building_restore_latest = sectBuildingMaintenance.metrics.restoredMatchesRepair;
        evidence.assertions.sect_building_offline_continuity = sectBuildingMaintenance.metrics.offlineSceneVersionAdvanced;
        evidence.assertions.sect_building_last_error_null =
            sectBuildingMaintenance.metrics.browserLastError === null && sectBuildingMaintenance.metrics.authorityLastError === null;

        const sectInstitutionManagement = await runSectInstitutionManagementReplay(connection, options, initialRuntimeSnapshot, runId);
        evidence.playerFacing.sectInstitutionManagement = sectInstitutionManagement;
        evidence.checkpoints.sect_institution_management_after_commands = sectInstitutionManagement.afterRaw;
        evidence.checkpoints.sect_institution_management_panel = sectInstitutionManagement.afterBrowser;
        evidence.assertions.sect_institution_overview_panel_opens =
            sectInstitutionManagement.panelProbe.sectPanelOpened && sectInstitutionManagement.metrics.overviewPresent;
        evidence.assertions.sect_institution_two_panels =
            sectInstitutionManagement.metrics.browserInstitutionDetailCount >= 2 &&
            sectInstitutionManagement.metrics.panelInstitutionEntryCount >= 2;
        evidence.assertions.sect_institution_authority_effects =
            sectInstitutionManagement.metrics.managedInstitutionCount >= 5 &&
            sectInstitutionManagement.metrics.taskHallTaskCapacityBonus > 0 &&
            sectInstitutionManagement.metrics.medicineHealingPower > 0 &&
            sectInstitutionManagement.metrics.caveOccupied;
        evidence.assertions.sect_institution_last_error_null =
            sectInstitutionManagement.metrics.browserLastError === null && sectInstitutionManagement.metrics.authorityLastError === null;

        const sectExternalRisk = await runSectExternalRiskReplay(connection, options, initialRuntimeSnapshot, runId);
        evidence.playerFacing.sectExternalRisk = sectExternalRisk;
        evidence.checkpoints.sect_external_dispatch_before = sectExternalRisk.beforeBrowser;
        evidence.checkpoints.sect_external_dispatch_after_dispatch = sectExternalRisk.afterDispatchBrowser;
        evidence.checkpoints.sect_external_dispatch_after_result = sectExternalRisk.afterResultBrowser;
        evidence.assertions.sect_external_dispatch_browser_command =
            sectExternalRisk.afterDispatchRaw.tasks.find((task) => task.id === sectExternalRisk.metrics.taskId)?.status === 'accepted';
        evidence.assertions.sect_external_dispatch_result =
            (sectExternalRisk.metrics.resultStatus === 'completed' || sectExternalRisk.metrics.resultStatus === 'failed') &&
            (sectExternalRisk.metrics.successRate ?? 0) > 0 &&
            (sectExternalRisk.metrics.risk ?? -1) >= 0 &&
            !!sectExternalRisk.metrics.evaluation;
        evidence.assertions.sect_external_dispatch_feedback = sectExternalRisk.metrics.feedbackCategories.includes('task_result');
        evidence.assertions.sect_external_dispatch_last_error_null =
            sectExternalRisk.metrics.browserLastError === null && sectExternalRisk.metrics.authorityLastError === null;

        const sectExpansionRisk = await runSectExpansionRiskReplay(connection, options, initialRuntimeSnapshot, runId);
        evidence.playerFacing.sectExpansionRisk = sectExpansionRisk;
        evidence.checkpoints.sect_expansion_risk_start = sectExpansionRisk.startRaw;
        evidence.checkpoints.sect_expansion_risk_final = sectExpansionRisk.finalRaw;
        evidence.checkpoints.sect_expansion_risk_rejoin = sectExpansionRisk.rejoinRaw;
        evidence.checkpoints.sect_expansion_risk_offline = sectExpansionRisk.offlineRaw;
        evidence.checkpoints.sect_expansion_risk_browser = sectExpansionRisk.finalBrowser;
        evidence.checkpoints.sect_expansion_risk_panel = sectExpansionRisk.panelProbe;
        evidence.assertions.sect_expansion_risk_buildings = sectExpansionRisk.metrics.mainHallLevel >= 2 && sectExpansionRisk.metrics.gateLevel >= 2;
        evidence.assertions.sect_expansion_risk_institutions = (sectExpansionRisk.finalRaw.institutionCount ?? 0) >= 5;
        evidence.assertions.sect_expansion_risk_artifacts_and_formations =
            (sectExpansionRisk.finalRaw.artifactCount ?? 0) >= 3 && (sectExpansionRisk.finalRaw.formationCount ?? 0) >= 1;
        evidence.assertions.sect_expansion_risk_goals_and_monthly =
            ((sectExpansionRisk.finalRaw.resolvedGoalCount ?? 0) >= 2 || sectExpansionRisk.metrics.maxResolvedGoalCount >= 2) &&
            sectExpansionRisk.metrics.maxMonthlyMonthIndex >= 1;
        evidence.assertions.sect_expansion_risk_crisis_and_treatment =
            (sectExpansionRisk.metrics.crisisChoiceHandled ||
                (sectExpansionRisk.metrics.resolvedEventKinds ?? []).includes('sect_crisis_choice')) &&
            sectExpansionRisk.metrics.maxTreatmentCount > 0;
        evidence.assertions.sect_expansion_risk_panel_surface =
            sectExpansionRisk.panelProbe.sectPanelOpened &&
            sectExpansionRisk.panelProbe.sectPanelRestoredClosed &&
            sectExpansionRisk.panelProbe.textSample.includes('经营总览') &&
            sectExpansionRisk.panelProbe.textSample.includes('宗门等级') &&
            sectExpansionRisk.panelProbe.textSample.includes('政策');
        evidence.assertions.sect_expansion_risk_restore_offline =
            JSON.stringify(sectExpansionContinuityComparable(sectExpansionRisk.finalRaw)) ===
                JSON.stringify(sectExpansionContinuityComparable(sectExpansionRisk.rejoinRaw)) &&
            (sectExpansionRisk.offlineRaw.sceneVersion ?? 0) > (sectExpansionRisk.rejoinRaw.sceneVersion ?? 0);
        evidence.assertions.sect_expansion_risk_last_error_null =
            sectExpansionRisk.metrics.browserLastError === null && sectExpansionRisk.metrics.authorityLastError === null;

        let resetSnapshot = await connection.evaluate(runtimeCallExpression('resetAuthoritySession'));
        let resetSummary = summarizeSnapshot(resetSnapshot);
        const controlsProbe = await runPlayerFacingControlsProbe(connection);
        evidence.playerFacing.controls = controlsProbe;
        const buttonsUnobstructed = controlsProbe.buttons.every(
            (button) =>
                button.exists &&
                button.activeInHierarchy &&
                button.selfHit &&
                !button.blockedByPanel &&
                !button.overlappedByStatusHud,
        );
        requireCondition(buttonsUnobstructed, 'One or more primary controls are not clickable or are overlapped.', controlsProbe);
        requireCondition(controlsProbe.sectPanelOpened, 'Task hall / sect panel button did not open the sect panel.', controlsProbe);
        evidence.assertions.primary_controls_unobstructed = true;
        evidence.assertions.sect_panel_opens = true;

        const resourceForProbe = pickDesignationProbeResource(resetSnapshot);
        requireCondition(!!resourceForProbe, 'Reset snapshot did not expose a resource node for designation probing.', resetSummary);
        const resourceTile = parseTileString(resourceForProbe.tile);
        requireCondition(!!resourceTile, 'Selected resource tile for designation probing was unparsable.', resourceForProbe);
        const blockedGatherProbe = await runBlockedGatherFallbackProbe(connection, resourceTile);
        evidence.playerFacing.resourceDesignationFailClosed = blockedGatherProbe;
        requireCondition(
            blockedGatherProbe.unchanged && blockedGatherProbe.authorityConnected === false,
            'Disconnected gather designation probe changed local resource state or did not enter blocked mode.',
            blockedGatherProbe,
        );
        evidence.assertions.resource_designation_fail_closed_when_blocked = true;

        resetSnapshot = await connection.evaluate(runtimeCallExpression('resetAuthoritySession'));
        resetSummary = summarizeSnapshot(resetSnapshot);
        evidence.checkpoints.reset_clear_ruin = resetSummary;
        requireCondition(resetSnapshot.authority.connected, 'Reset replay did not connect authority.', resetSummary);
        requireCondition(resetSnapshot.authority.lastError === null, 'Reset replay left authority.lastError non-null.', resetSummary);
        requireCondition(resetSnapshot.session.phase === 'clear_ruin', 'Reset replay did not return to clear_ruin.', resetSummary);
        evidence.assertions.reset_clear_ruin = true;

        const designationRoundTrip = await runResourceDesignationRoundTrip(connection, options, resourceTile);
        evidence.playerFacing.resourceDesignationRoundTrip = designationRoundTrip;
        requireCondition(
            designationRoundTrip.afterAuthorityRoundTrip === designationRoundTrip.requested &&
                designationRoundTrip.restored === designationRoundTrip.before &&
                designationRoundTrip.lastErrorAfterIntent === null,
            'Resource designation did not round-trip through authority snapshot.',
            designationRoundTrip,
        );
        evidence.assertions.resource_designation_round_trip = true;

        const ruinBuilding = findBuilding(resetSnapshot, resetSnapshot.session.ruinBuildingId);
        requireCondition(!!ruinBuilding, 'Reset snapshot did not expose ruinBuildingId in building list.', resetSummary);
        const ruinOrigin = parseTileString(ruinBuilding.origin);
        requireCondition(!!ruinOrigin, 'Reset snapshot exposed an unparsable ruin origin.', ruinBuilding);

        await connection.evaluate(
            runtimeCallExpression('executeAuthorityCommand', [
                {
                    name: 'toggle_demolition',
                    payload: {
                        buildingId: ruinBuilding.id,
                    },
                },
                {
                    commandKey: `gate:toggle_demolition:${ruinBuilding.id}`,
                },
            ]),
        );

        const placeGuardTowerSnapshot = await waitForCheckpoint(
            'place_guard_tower',
            connection,
            options,
            (snapshot) => snapshot.session.phase === 'place_guard_tower',
        );
        evidence.checkpoints.place_guard_tower = summarizeSnapshot(placeGuardTowerSnapshot);
        evidence.assertions.place_guard_tower = true;

        await connection.evaluate(
            runtimeCallExpression('executeAuthorityCommand', [
                {
                    name: 'place_building',
                    payload: {
                        buildingType: 'guard_tower',
                        origin: ruinOrigin,
                    },
                },
                {
                    commandKey: `gate:place_guard_tower:${ruinOrigin.col},${ruinOrigin.row}`,
                },
            ]),
        );

        const upgradePhaseSnapshot = await waitForCheckpoint(
            'upgrade_guard_tower',
            connection,
            options,
            (snapshot) => snapshot.session.phase === 'upgrade_guard_tower',
        );
        evidence.checkpoints.upgrade_guard_tower = summarizeSnapshot(upgradePhaseSnapshot);
        evidence.assertions.upgrade_guard_tower = true;

        const lv1GuardTower = await waitForCheckpoint(
            'guard_tower_lv1_active',
            connection,
            options,
            (snapshot) => {
                const guardTower = findGuardTower(snapshot);
                return !!guardTower && guardTower.state === 'active' && guardTower.level === 1;
            },
        );

        const guardTower = findGuardTower(lv1GuardTower);
        requireCondition(!!guardTower, 'Could not resolve guard tower after placement.', summarizeSnapshot(lv1GuardTower));

        await connection.evaluate(
            runtimeCallExpression('executeAuthorityCommand', [
                {
                    name: 'request_upgrade',
                    payload: {
                        buildingId: guardTower.id,
                    },
                },
                {
                    commandKey: `gate:request_upgrade:${guardTower.id}`,
                },
            ]),
        );

        const lv2Snapshot = await waitForCheckpoint(
            'guard_tower_lv2_active',
            connection,
            options,
            (snapshot) => {
                const upgraded = findGuardTower(snapshot);
                return !!upgraded && upgraded.state === 'active' && upgraded.level >= 2;
            },
        );
        evidence.checkpoints.guard_tower_lv2_active = summarizeSnapshot(lv2Snapshot);
        evidence.assertions.guard_tower_lv2_active = true;

        const defendSnapshot = await waitForCheckpoint(
            'defend_started',
            connection,
            options,
            (snapshot) => snapshot.session.phase === 'defend',
        );
        const defendSummary = summarizeSnapshot(defendSnapshot);
        evidence.checkpoints.defend_started = defendSummary;
        requireCondition(
            defendSummary.session.riskIntensity >= 0 &&
                defendSummary.session.threatCurve >= 1 &&
                defendSummary.session.defenseRating >= 0 &&
                defendSummary.session.guardDiscipleCount >= 0 &&
                defendSummary.session.omenStatus !== '' &&
                defendSummary.session.defenseSummary !== '' &&
                defendSummary.session.sourceSummary.length > 0,
            'Defend checkpoint did not expose bounded risk/defense summary fields.',
            defendSummary,
        );
        evidence.assertions.defend_started = true;

        const combatPresentationEvidence = await waitForCombatPresentationEvidence(connection, options);
        evidence.playerFacing.combatPresentation = combatPresentationEvidence;
        evidence.assertions.hostile_interpolation_not_hard_jump = true;
        evidence.assertions.damage_floater_captured = true;

        const recoverSnapshot = await waitForCheckpoint(
            'recover_started',
            connection,
            options,
            (snapshot) => snapshot.session.phase === 'recover',
        );
        const recoverSummary = summarizeSnapshot(recoverSnapshot);
        evidence.checkpoints.recover_started = recoverSummary;
        const hasBoundedRecoverContext =
            recoverSummary.session.damagedBuildingCount > 0 ||
            recoverSummary.session.regeneratingNodeCount > 0 ||
            recoverSummary.session.recoverReason !== 'none';
        requireCondition(
            recoverSummary.session.damageSummary !== '' &&
                recoverSummary.session.repairSuggestion !== '' &&
                hasBoundedRecoverContext,
            'Recover checkpoint did not expose bounded damage/recovery feedback.',
            recoverSummary,
        );
        evidence.assertions.recover_started = true;

        const restoredSnapshot = await connection.evaluate(runtimeCallExpression('restoreAuthoritySession'));
        const restoredSummary = summarizeSnapshot(restoredSnapshot);
        evidence.checkpoints.restore_latest = restoredSummary;
        requireCondition(
            snapshotsMatchForRestore(recoverSummary, restoredSummary),
            'restore_latest did not preserve bounded recover state.',
            {
                before: normalizeRestoreComparable(recoverSummary),
                after: normalizeRestoreComparable(restoredSummary),
            },
        );
        requireCondition(restoredSnapshot.authority.lastError === null, 'restore_latest returned authority.lastError.', restoredSummary);
        evidence.assertions.restore_matches_recover_state = true;

        const secondCycleReadySnapshot = await waitForCheckpoint(
            'second_cycle_ready',
            connection,
            options,
            (snapshot) =>
                snapshot.session.phase === 'second_cycle_ready' &&
                snapshot.session.damagedBuildingCount === 0 &&
                snapshot.authority.lastError === null,
        );
        const secondCycleSummary = summarizeSnapshot(secondCycleReadySnapshot);
        evidence.checkpoints.second_cycle_ready = secondCycleSummary;
        evidence.assertions.authority_risk_defense_feedback_visible = true;
        evidence.assertions.second_cycle_ready = true;
        evidence.assertions.final_last_error_null = secondCycleReadySnapshot.authority.lastError === null;

        const latestPath = path.join(options.outputDir, 'latest.json');
        const timestampedPath = path.join(options.outputDir, `${runId}.json`);
        const payload = `${JSON.stringify(evidence, null, 2)}\n`;
        fs.writeFileSync(timestampedPath, payload, 'utf8');
        fs.writeFileSync(latestPath, payload, 'utf8');

        console.log(`Dedicated browser replay gate passed.`);
        console.log(`Evidence: ${timestampedPath}`);
        console.log(
            JSON.stringify(
                {
                    runId,
                    previewUrl: options.previewUrl,
                    finalPhase: secondCycleSummary.session.phase,
                    finalLastError: secondCycleSummary.authority.lastError,
                    restoreMatchesRecoverState: evidence.assertions.restore_matches_recover_state,
                    primaryControlsUnobstructed: evidence.assertions.primary_controls_unobstructed,
                    resourceDesignationRoundTrip: evidence.assertions.resource_designation_round_trip,
                    hostileInterpolationNotHardJump: evidence.assertions.hostile_interpolation_not_hard_jump,
                    damageFloaterCaptured: evidence.assertions.damage_floater_captured,
                    sectMultidayRoster: evidence.assertions.sect_multiday_recruitment_roster,
                    sectMultidayMonthly: evidence.assertions.sect_multiday_monthly,
                    sectMultidayEventFeedback: evidence.assertions.sect_multiday_event_feedback,
                    sectMultidayStoryFeedbackSurface: evidence.assertions.sect_multiday_story_feedback_surface,
                    sectMultidayStoryFeedbackTypes: evidence.assertions.sect_multiday_story_feedback_types,
                    sectEventChoicePanelOpens: evidence.assertions.sect_event_choice_panel_opens,
                    sectEventChoiceAuthorityResult: evidence.assertions.sect_event_choice_authority_result,
                    sectEventChoiceFeedback: evidence.assertions.sect_event_choice_feedback,
                    sectPolicyBrowserSwitch: evidence.assertions.sect_policy_browser_switch,
                    sectPolicyMultidayMetrics: evidence.assertions.sect_policy_multiday_metrics,
                    sectPolicyLastErrorNull: evidence.assertions.sect_policy_last_error_null,
                    sectArtifactBrowserCommand: evidence.assertions.sect_artifact_browser_command,
                    sectArtifactMetricDelta: evidence.assertions.sect_artifact_metric_delta,
                    sectArtifactLastErrorNull: evidence.assertions.sect_artifact_last_error_null,
                    sectPromotionAssessmentBrowserCommand: evidence.assertions.sect_promotion_assessment_browser_command,
                    sectPromotionAuthorityResult: evidence.assertions.sect_promotion_authority_result,
                    sectPromotionFeedback: evidence.assertions.sect_promotion_feedback,
                    sectPromotionLastErrorNull: evidence.assertions.sect_promotion_last_error_null,
                    sectBuildingMaintenancePressure: evidence.assertions.sect_building_maintenance_pressure,
                    sectBuildingDamageRepair: evidence.assertions.sect_building_damage_repair,
                    sectBuildingRestoreLatest: evidence.assertions.sect_building_restore_latest,
                    sectBuildingOfflineContinuity: evidence.assertions.sect_building_offline_continuity,
                    sectBuildingLastErrorNull: evidence.assertions.sect_building_last_error_null,
                    sectInstitutionOverviewPanelOpens: evidence.assertions.sect_institution_overview_panel_opens,
                    sectInstitutionTwoPanels: evidence.assertions.sect_institution_two_panels,
                    sectInstitutionAuthorityEffects: evidence.assertions.sect_institution_authority_effects,
                    sectInstitutionLastErrorNull: evidence.assertions.sect_institution_last_error_null,
                    sectExternalDispatchBrowserCommand: evidence.assertions.sect_external_dispatch_browser_command,
                    sectExternalDispatchResult: evidence.assertions.sect_external_dispatch_result,
                    sectExternalDispatchFeedback: evidence.assertions.sect_external_dispatch_feedback,
                    sectExternalDispatchLastErrorNull: evidence.assertions.sect_external_dispatch_last_error_null,
                    sectExpansionRiskBuildings: evidence.assertions.sect_expansion_risk_buildings,
                    sectExpansionRiskInstitutions: evidence.assertions.sect_expansion_risk_institutions,
                    sectExpansionRiskArtifactsAndFormations: evidence.assertions.sect_expansion_risk_artifacts_and_formations,
                    sectExpansionRiskGoalsAndMonthly: evidence.assertions.sect_expansion_risk_goals_and_monthly,
                    sectExpansionRiskCrisisAndTreatment: evidence.assertions.sect_expansion_risk_crisis_and_treatment,
                    sectExpansionRiskPanelSurface: evidence.assertions.sect_expansion_risk_panel_surface,
                    sectExpansionRiskRestoreOffline: evidence.assertions.sect_expansion_risk_restore_offline,
                    sectExpansionRiskLastErrorNull: evidence.assertions.sect_expansion_risk_last_error_null,
                    authorityRiskDefenseFeedbackVisible: evidence.assertions.authority_risk_defense_feedback_visible,
                },
                null,
                2,
            ),
        );
    } finally {
        await connection.close().catch(() => undefined);
    }
}

main().catch((error) => {
    console.error(`ERROR: ${error instanceof Error ? error.message : String(error)}`);
    if (error && typeof error === 'object' && 'details' in error && error.details !== undefined) {
        console.error(JSON.stringify(error.details, null, 2));
    }
    process.exit(1);
});
