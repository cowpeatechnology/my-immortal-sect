#!/usr/bin/env node

import fs from 'node:fs';
import http from 'node:http';
import os from 'node:os';
import path from 'node:path';
import process from 'node:process';

const DEFAULT_USER_DATA_DIR = path.join(
  os.homedir(),
  'Library',
  'Application Support',
  'Google',
  'Chrome'
);
const DEFAULT_URL_SUBSTRING = 'chatgpt.com/c/';
const DEFAULT_HOST = '127.0.0.1';
const DEFAULT_PORT = 8776;
const DEFAULT_SETTLE_MS = 3000;
const DEFAULT_GRACE_MS = 30000;
const DEFAULT_MAX_DEBUG_EVENTS = 400;
const DEFAULT_DEBUG_EVENTS_LIMIT = 20;
const DEFAULT_STATE_DIR = path.join(process.cwd(), 'workspace', 'output', 'chatgpt-capture-v3-server');
const MAX_LOG_STRING_LENGTH = 4096;
const MAX_LOG_ARRAY_ITEMS = 32;
const MAX_LOG_OBJECT_KEYS = 64;
const MAX_LOG_DEPTH = 6;
const IMAGE_CHOICE_PROMPT_KEYWORDS = [
  '你更喜欢哪张图片',
  'which image do you prefer'
];
const IMAGE_CHOICE_SKIP_KEYWORDS = ['跳过', 'skip'];
const IMAGE_CHOICE_PREFERENCE_KEYWORDS = [
  '图片 1 更佳',
  '图片 2 更佳',
  'image 1 is better',
  'image 2 is better'
];
const STOP_KEYWORDS = ['stop', '停止', '停止流式传输'];
const IMAGE_PROGRESS_KEYWORDS = [
  '正在创建图片',
  '最后微调一下',
  '生成图片中',
  '创建图片中',
  'creating image',
  'generating image',
  'final touches',
  'polishing'
];

const NETWORK_LABELS = [
  { label: 'conversation', pattern: /\/backend-api\/f\/conversation(?:\?|$)/i },
  { label: 'async_status', pattern: /\/backend-api\/conversation\/[^/]+\/async-status(?:\?|$)/i },
  { label: 'file_download', pattern: /\/backend-api\/files\/download\/[^/?#]+/i },
  { label: 'estuary_content', pattern: /\/backend-api\/estuary\/content(?:\?|$)/i }
];

function printHelp() {
  console.log(`Usage:
  node tools/browser/scripts/chatgpt_cdp_observer_v3.mjs [options]

Options:
  --user-data-dir <path>   Chrome user data dir. Default: ${DEFAULT_USER_DATA_DIR}
  --state-dir <path>       Observer state dir. Default: ${DEFAULT_STATE_DIR}
  --host <host>            HTTP bind host. Default: ${DEFAULT_HOST}
  --port <port>            HTTP bind port. Default: ${DEFAULT_PORT}
  --url-substring <text>   Target page URL filter. Default: ${DEFAULT_URL_SUBSTRING}
  --settle-ms <ms>         Wait after async-status OK before resolving. Default: ${DEFAULT_SETTLE_MS}
  --grace-ms <ms>          Max extra wait after async-status OK. Default: ${DEFAULT_GRACE_MS}
  --max-debug-events <n>   In-memory debug event ring size. Default: ${DEFAULT_MAX_DEBUG_EVENTS}
`);
}

function parseArgs(argv) {
  const options = {
    userDataDir: DEFAULT_USER_DATA_DIR,
    stateDir: DEFAULT_STATE_DIR,
    host: DEFAULT_HOST,
    port: DEFAULT_PORT,
    urlSubstring: DEFAULT_URL_SUBSTRING,
    settleMs: DEFAULT_SETTLE_MS,
    graceMs: DEFAULT_GRACE_MS,
    maxDebugEvents: DEFAULT_MAX_DEBUG_EVENTS
  };

  for (let index = 2; index < argv.length; index += 1) {
    const arg = argv[index];
    const next = argv[index + 1];
    if ((arg === '--help') || (arg === '-h')) {
      printHelp();
      process.exit(0);
    }
    if (arg === '--user-data-dir' && next) {
      options.userDataDir = next;
      index += 1;
      continue;
    }
    if (arg === '--state-dir' && next) {
      options.stateDir = next;
      index += 1;
      continue;
    }
    if (arg === '--host' && next) {
      options.host = next;
      index += 1;
      continue;
    }
    if (arg === '--port' && next) {
      options.port = Number(next);
      index += 1;
      continue;
    }
    if (arg === '--url-substring' && next) {
      options.urlSubstring = next;
      index += 1;
      continue;
    }
    if (arg === '--settle-ms' && next) {
      options.settleMs = Number(next);
      index += 1;
      continue;
    }
    if (arg === '--grace-ms' && next) {
      options.graceMs = Number(next);
      index += 1;
      continue;
    }
    if (arg === '--max-debug-events' && next) {
      options.maxDebugEvents = Number(next);
      index += 1;
      continue;
    }
    throw new Error(`Unknown argument: ${arg}`);
  }

  if (!Number.isInteger(options.port) || options.port <= 0 || options.port > 65535) {
    throw new Error(`Invalid --port value: ${options.port}`);
  }
  if (!Number.isFinite(options.settleMs) || options.settleMs < 0) {
    throw new Error(`Invalid --settle-ms value: ${options.settleMs}`);
  }
  if (!Number.isFinite(options.graceMs) || options.graceMs < options.settleMs) {
    throw new Error(`Invalid --grace-ms value: ${options.graceMs}`);
  }
  if (!Number.isFinite(options.maxDebugEvents) || options.maxDebugEvents < 50) {
    throw new Error(`Invalid --max-debug-events value: ${options.maxDebugEvents}`);
  }

  return options;
}

function nowIso() {
  return new Date().toISOString();
}

function slugTimestamp() {
  return nowIso().replace(/[:.]/g, '-');
}

function safeStem(value, fallback = 'capture') {
  const stem = String(value || '')
    .trim()
    .replace(/[^a-zA-Z0-9._-]+/g, '-')
    .replace(/^-+|-+$/g, '');
  return stem || fallback;
}

function ensureDirectory(dirPath) {
  fs.mkdirSync(dirPath, { recursive: true });
}

function readDevToolsActivePort(userDataDir) {
  const portFile = path.join(userDataDir, 'DevToolsActivePort');
  const raw = fs.readFileSync(portFile, 'utf8');
  const lines = raw
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean);
  if (lines.length < 2) {
    throw new Error(`Invalid DevToolsActivePort contents in ${portFile}`);
  }
  const port = Number(lines[0]);
  if (!Number.isInteger(port) || port <= 0 || port > 65535) {
    throw new Error(`Invalid DevTools port in ${portFile}: ${lines[0]}`);
  }
  return {
    port,
    wsUrl: `ws://127.0.0.1:${port}${lines[1]}`,
    portFile
  };
}

function networkLabel(url) {
  const value = String(url || '');
  for (const candidate of NETWORK_LABELS) {
    if (candidate.pattern.test(value)) {
      return candidate.label;
    }
  }
  return null;
}

function extractFileId(value) {
  const match = String(value || '').match(/(?:\/|id=)(file_[^/?#&]+)/i);
  return match?.[1] || null;
}

function readJson(text, fallback = null) {
  try {
    return JSON.parse(String(text || ''));
  } catch {
    return fallback;
  }
}

function extFromMimeType(mimeType) {
  if (/png/i.test(String(mimeType || ''))) {
    return '.png';
  }
  if (/jpe?g/i.test(String(mimeType || ''))) {
    return '.jpg';
  }
  if (/webp/i.test(String(mimeType || ''))) {
    return '.webp';
  }
  return '.bin';
}

function sanitizeForLog(value, depth = 0) {
  if (value === null || value === undefined) {
    return value ?? null;
  }

  if (Buffer.isBuffer(value)) {
    return {
      type: 'Buffer',
      byteLength: value.length
    };
  }

  if (typeof value === 'string') {
    if (value.length <= MAX_LOG_STRING_LENGTH) {
      return value;
    }
    return `${value.slice(0, MAX_LOG_STRING_LENGTH)}… [truncated ${value.length - MAX_LOG_STRING_LENGTH} chars]`;
  }

  if (typeof value !== 'object') {
    return value;
  }

  if (depth >= MAX_LOG_DEPTH) {
    if (Array.isArray(value)) {
      return `[array:${value.length}]`;
    }
    return '[object]';
  }

  if (Array.isArray(value)) {
    const items = value
      .slice(0, MAX_LOG_ARRAY_ITEMS)
      .map((item) => sanitizeForLog(item, depth + 1));
    if (value.length > MAX_LOG_ARRAY_ITEMS) {
      items.push(`[+${value.length - MAX_LOG_ARRAY_ITEMS} more items]`);
    }
    return items;
  }

  const entries = Object.entries(value).filter(([, nested]) => nested !== undefined);
  const output = {};
  for (const [key, nested] of entries.slice(0, MAX_LOG_OBJECT_KEYS)) {
    output[key] = sanitizeForLog(nested, depth + 1);
  }
  if (entries.length > MAX_LOG_OBJECT_KEYS) {
    output.__truncatedKeys = entries.length - MAX_LOG_OBJECT_KEYS;
  }
  return output;
}

function summarizeDebugEvent(record) {
  const summary = {};
  const keys = [
    'ts',
    'source',
    'event',
    'runIndex',
    'requestId',
    'sequence',
    'label',
    'method',
    'status',
    'statusText',
    'mimeType',
    'fileId',
    'fileName',
    'sizeBytes',
    'encodedDataLength',
    'jobId',
    'assignedJobId',
    'state',
    'version',
    'queueDepth',
    'reason',
    'clickedSkip',
    'choiceCleared',
    'validatedFinalCount',
    'preferenceButtonCount',
    'asyncCount',
    'fileDownloadCount',
    'estuaryCount',
    'error',
    'url',
    'startedAt',
    'updatedAt',
    'ackedAt'
  ];
  for (const key of keys) {
    if (record[key] !== undefined) {
      summary[key] = record[key];
    }
  }
  if (record.body && typeof record.body === 'object') {
    summary.body = sanitizeForLog(record.body);
  }
  if (record.pageStatus && typeof record.pageStatus === 'object') {
    summary.pageStatus = sanitizeForLog({
      composerFound: record.pageStatus.composerFound,
      loginRequired: record.pageStatus.loginRequired,
      busyGenerating: record.pageStatus.busyGenerating,
      candidateChoiceVisible: record.pageStatus.candidateChoiceVisible,
      preferenceButtonCount: record.pageStatus.preferenceButtonCount,
      progressText: record.pageStatus.progressText,
      title: record.pageStatus.title,
      url: record.pageStatus.url
    });
  }
  if (record.target && typeof record.target === 'object') {
    summary.target = sanitizeForLog({
      targetId: record.target.targetId,
      title: record.target.title,
      url: record.target.url
    });
  }
  if (Array.isArray(record.initialCandidateFileIds) && record.initialCandidateFileIds.length) {
    summary.initialCandidateFileIds = sanitizeForLog(record.initialCandidateFileIds);
  }
  if (Array.isArray(record.chosenFileIds) && record.chosenFileIds.length) {
    summary.chosenFileIds = sanitizeForLog(record.chosenFileIds);
  }
  if (Array.isArray(record.preferredFileIds) && record.preferredFileIds.length) {
    summary.preferredFileIds = sanitizeForLog(record.preferredFileIds);
  }
  return summary;
}

function createRun(runIndex, requestId, conversationId) {
  return {
    runIndex,
    requestId,
    conversationId,
    startedAt: nowIso(),
    assignedJobId: null,
    asyncEvents: [],
    fileDownloads: [],
    estuaryContents: [],
    waitingForFinalAfterAsync: false,
    asyncOkAtMs: null,
    settleTimer: null,
    graceTimer: null,
    finalized: false
  };
}

function collectValidatedFinals(run) {
  const finalDownloadsByFileId = new Map();
  for (const item of run.fileDownloads) {
    if (!item.fileId || item.status !== 'success' || item.isPartial) {
      continue;
    }
    finalDownloadsByFileId.set(item.fileId, item);
  }

  const finals = [];
  for (const item of run.estuaryContents) {
    if (!item.fileId || item.status !== 200 || !item.bodyBuffer) {
      continue;
    }
    const matchedDownload = finalDownloadsByFileId.get(item.fileId);
    if (!matchedDownload) {
      continue;
    }
    finals.push({
      ...item,
      fileName: matchedDownload.fileName,
      downloadUrl: matchedDownload.downloadUrl
    });
  }

  finals.sort((left, right) => (left.sequence || 0) - (right.sequence || 0));
  return finals;
}

function latestValidatedFinal(run, preferredFileIds = []) {
  const finals = collectValidatedFinals(run);
  if (!finals.length) {
    return null;
  }

  const preferred = Array.isArray(preferredFileIds)
    ? preferredFileIds.filter((value) => typeof value === 'string' && value)
    : [];
  if (preferred.length) {
    const preferredSet = new Set(preferred);
    for (let index = finals.length - 1; index >= 0; index -= 1) {
      if (preferredSet.has(finals[index].fileId)) {
        return finals[index];
      }
    }
  }

  return finals.at(-1) || null;
}

function validatedFinalCount(run) {
  return collectValidatedFinals(run).length;
}

class JsonlLog {
  constructor(logDir) {
    ensureDirectory(logDir);
    this.sessionId = `observer-v3-${slugTimestamp()}`;
    this.path = path.join(logDir, `${this.sessionId}.jsonl`);
    this.stream = fs.createWriteStream(this.path, { flags: 'a' });
  }

  write(payload) {
    this.stream.write(`${JSON.stringify({ ts: nowIso(), ...payload })}\n`);
  }

  async close() {
    await new Promise((resolve) => this.stream.end(resolve));
  }
}

class CdpConnection {
  constructor(wsUrl) {
    this.wsUrl = wsUrl;
    this.ws = null;
    this.nextId = 1;
    this.pending = new Map();
    this.listeners = new Set();
  }

  async connect() {
    await new Promise((resolve, reject) => {
      const ws = new WebSocket(this.wsUrl);
      this.ws = ws;
      ws.addEventListener('open', () => resolve());
      ws.addEventListener('error', (event) => reject(event.error || new Error('WebSocket connection failed.')));
      ws.addEventListener('message', (event) => this.#handleMessage(event.data));
      ws.addEventListener('close', () => {
        for (const pending of this.pending.values()) {
          pending.reject(new Error('CDP websocket closed.'));
        }
        this.pending.clear();
      });
    });
  }

  onEvent(listener) {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  }

  async send(method, params = {}, sessionId = undefined) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      throw new Error('CDP websocket is not connected.');
    }
    const id = this.nextId += 1;
    const payload = { id, method, params };
    if (sessionId) {
      payload.sessionId = sessionId;
    }
    const response = await new Promise((resolve, reject) => {
      this.pending.set(id, { resolve, reject });
      this.ws.send(JSON.stringify(payload));
    });
    return response;
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

  #handleMessage(raw) {
    const payload = JSON.parse(String(raw));
    if (payload.id) {
      const pending = this.pending.get(payload.id);
      if (!pending) {
        return;
      }
      this.pending.delete(payload.id);
      if (payload.error) {
        pending.reject(new Error(payload.error.message || JSON.stringify(payload.error)));
      } else {
        pending.resolve(payload.result);
      }
      return;
    }
    for (const listener of this.listeners) {
      listener(payload);
    }
  }
}

class ObserverServer {
  constructor(options) {
    this.options = options;
    this.paths = {
      root: options.stateDir,
      cache: path.join(options.stateDir, 'cache'),
      logs: path.join(options.stateDir, 'logs')
    };
    ensureDirectory(this.paths.cache);
    ensureDirectory(this.paths.logs);

    this.log = new JsonlLog(this.paths.logs);
    this.serverSessionId = this.log.sessionId;
    this.debugEvents = [];
    this.jobs = new Map();
    this.waitingJobIds = [];
    this.jobWaiters = new Map();
    this.requestState = new Map();
    this.connection = null;
    this.httpServer = null;
    this.activeSessionId = null;
    this.activeTarget = null;
    this.activeRun = null;
    this.runIndex = 0;
    this.eventSequence = 0;
    this.shuttingDown = false;
  }

  snapshotRun(run = this.activeRun) {
    if (!run) {
      return null;
    }
    return {
      runIndex: run.runIndex,
      requestId: run.requestId,
      conversationId: run.conversationId,
      startedAt: run.startedAt,
      assignedJobId: run.assignedJobId,
      asyncCount: run.asyncEvents.length,
      fileDownloadCount: run.fileDownloads.length,
      estuaryCount: run.estuaryContents.length,
      waitingForFinalAfterAsync: run.waitingForFinalAfterAsync,
      asyncOkAtMs: run.asyncOkAtMs,
      finalized: run.finalized
    };
  }

  recordDebug(event) {
    this.debugEvents.push(event);
    if (this.debugEvents.length > this.options.maxDebugEvents) {
      this.debugEvents.shift();
    }
  }

  writeRecord(payload) {
    const record = {
      session_id: this.serverSessionId,
      ...payload
    };
    const safeRecord = sanitizeForLog(record);
    this.log.write(safeRecord);
    this.recordDebug(safeRecord);
  }

  listDebugEvents(limit = DEFAULT_DEBUG_EVENTS_LIMIT, eventName = '') {
    const safeLimit = Math.max(1, Math.min(200, Number(limit) || DEFAULT_DEBUG_EVENTS_LIMIT));
    const filter = String(eventName || '').trim();
    const events = filter
      ? this.debugEvents.filter((record) => record.event === filter)
      : this.debugEvents;
    return events.slice(-safeLimit).map((record) => summarizeDebugEvent(record));
  }

  snapshotJob(job) {
    return {
      id: job.id,
      label: job.label,
      state: job.state,
      version: job.version,
      createdAt: job.createdAt,
      updatedAt: job.updatedAt,
      assignedRunIndex: job.assignedRunIndex,
      ackedAt: job.ackedAt,
      error: job.error,
      result: job.result
        ? {
            fileName: job.result.fileName,
            mimeType: job.result.mimeType,
            sizeBytes: job.result.sizeBytes,
            cacheToken: job.result.cacheToken,
            readyAt: job.result.readyAt
          }
        : null
    };
  }

  notifyJobWaiters(jobId) {
    const waiters = this.jobWaiters.get(jobId);
    if (!waiters || waiters.length === 0) {
      return;
    }
    const job = this.jobs.get(jobId);
    if (!job) {
      return;
    }
    const remaining = [];
    for (const waiter of waiters) {
      if (job.version > waiter.sinceVersion || !['waiting', 'assigned'].includes(job.state)) {
        clearTimeout(waiter.timer);
        waiter.resolve({ timedOut: false, job: this.snapshotJob(job) });
      } else {
        remaining.push(waiter);
      }
    }
    if (remaining.length > 0) {
      this.jobWaiters.set(jobId, remaining);
    } else {
      this.jobWaiters.delete(jobId);
    }
  }

  updateJob(job, patch, eventName, extra = {}) {
    Object.assign(job, patch);
    job.updatedAt = nowIso();
    job.version += 1;
    this.writeRecord({
      source: 'observer_v3',
      event: eventName,
      jobId: job.id,
      state: job.state,
      version: job.version,
      ...extra
    });
    this.notifyJobWaiters(job.id);
    return job;
  }

  registerJob(payload) {
    const job = {
      id: `job-${slugTimestamp()}-${Math.random().toString(36).slice(2, 8)}`,
      label: safeStem(payload?.label || payload?.basename || 'capture-job', 'capture-job'),
      state: 'waiting',
      version: 1,
      createdAt: nowIso(),
      updatedAt: nowIso(),
      assignedRunIndex: null,
      ackedAt: null,
      error: null,
      result: null
    };
    this.jobs.set(job.id, job);
    this.waitingJobIds.push(job.id);
    this.writeRecord({
      source: 'observer_v3',
      event: 'job_registered',
      jobId: job.id,
      label: job.label,
      queueDepth: this.waitingJobIds.length
    });
    return this.snapshotJob(job);
  }

  async waitForJobUpdate(jobId, sinceVersion, timeoutMs) {
    const job = this.jobs.get(jobId);
    if (!job) {
      throw new Error(`Unknown job: ${jobId}`);
    }
    if (job.version > sinceVersion || !['waiting', 'assigned'].includes(job.state)) {
      return { timedOut: false, job: this.snapshotJob(job) };
    }

    return new Promise((resolve) => {
      const timer = setTimeout(() => {
        const waiters = this.jobWaiters.get(jobId) || [];
        this.jobWaiters.set(
          jobId,
          waiters.filter((entry) => entry !== waiter)
        );
        resolve({ timedOut: true, job: this.snapshotJob(job) });
      }, timeoutMs);

      const waiter = { sinceVersion, resolve, timer };
      const waiters = this.jobWaiters.get(jobId) || [];
      waiters.push(waiter);
      this.jobWaiters.set(jobId, waiters);
    });
  }

  jobById(jobId) {
    const job = this.jobs.get(jobId);
    return job ? this.snapshotJob(job) : null;
  }

  ackJob(jobId) {
    const job = this.jobs.get(jobId);
    if (!job) {
      throw new Error(`Unknown job: ${jobId}`);
    }
    if (job.result?.cachePath && fs.existsSync(job.result.cachePath)) {
      fs.unlinkSync(job.result.cachePath);
    }
    this.updateJob(job, {
      state: 'acked',
      ackedAt: nowIso()
    }, 'job_acked');
    return this.snapshotJob(job);
  }

  cancelJob(jobId) {
    const job = this.jobs.get(jobId);
    if (!job) {
      throw new Error(`Unknown job: ${jobId}`);
    }
    if (['acked', 'ready', 'failed', 'cancelled'].includes(job.state)) {
      return this.snapshotJob(job);
    }
    this.waitingJobIds = this.waitingJobIds.filter((candidate) => candidate !== jobId);
    this.updateJob(job, {
      state: 'cancelled'
    }, 'job_cancelled');
    return this.snapshotJob(job);
  }

  assignNextWaitingJob(run) {
    while (this.waitingJobIds.length > 0) {
      const jobId = this.waitingJobIds.shift();
      const job = this.jobs.get(jobId);
      if (!job || job.state !== 'waiting') {
        continue;
      }
      run.assignedJobId = job.id;
      this.updateJob(job, {
        state: 'assigned',
        assignedRunIndex: run.runIndex
      }, 'job_assigned', {
        runIndex: run.runIndex
      });
      return job;
    }
    return null;
  }

  clearRunTimers(run) {
    if (run?.settleTimer) {
      clearTimeout(run.settleTimer);
      run.settleTimer = null;
    }
    if (run?.graceTimer) {
      clearTimeout(run.graceTimer);
      run.graceTimer = null;
    }
  }

  async evaluatePageAction(action, payload = {}) {
    if (!this.connection || !this.activeSessionId) {
      throw new Error('CDP session is not ready.');
    }

    const expression = `(() => {
      const payload = ${JSON.stringify({ action, ...payload })};
      const IMAGE_CHOICE_PROMPT_KEYWORDS = ${JSON.stringify(IMAGE_CHOICE_PROMPT_KEYWORDS)};
      const IMAGE_CHOICE_SKIP_KEYWORDS = ${JSON.stringify(IMAGE_CHOICE_SKIP_KEYWORDS)};
      const IMAGE_CHOICE_PREFERENCE_KEYWORDS = ${JSON.stringify(IMAGE_CHOICE_PREFERENCE_KEYWORDS)};
      const STOP_KEYWORDS = ${JSON.stringify(STOP_KEYWORDS)};
      const IMAGE_PROGRESS_KEYWORDS = ${JSON.stringify(IMAGE_PROGRESS_KEYWORDS)};

      const delay = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
      const normalizeText = (value) => String(value || '').trim().toLowerCase();
      const includesKeyword = (text, keywords) => {
        const haystack = normalizeText(text);
        return keywords.some((keyword) => haystack.includes(normalizeText(keyword)));
      };
      const elementSignalText = (element) => [
        element?.innerText,
        element?.textContent,
        element?.getAttribute?.('aria-label'),
        element?.getAttribute?.('title'),
        element?.getAttribute?.('data-testid'),
        element?.getAttribute?.('aria-description')
      ]
        .map((value) => String(value || ''))
        .join(' ')
        .trim();
      const sectionSignalText = (element) => [
        element?.innerText,
        element?.textContent,
        element?.getAttribute?.('aria-label')
      ]
        .map((value) => String(value || ''))
        .join(' ')
        .trim();
      const isVisible = (element) => {
        if (!element) return false;
        const rect = element.getBoundingClientRect();
        const style = window.getComputedStyle(element);
        return rect.width > 0 && rect.height > 0 && style.visibility !== 'hidden' && style.display !== 'none';
      };
      const nativeValueSetter = (element) => {
        if (element instanceof HTMLTextAreaElement) {
          return Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, 'value')?.set || null;
        }
        if (element instanceof HTMLInputElement) {
          return Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')?.set || null;
        }
        return null;
      };
      const findComposer = () => {
        const selectors = [
          '#prompt-textarea',
          'textarea[data-testid]',
          'textarea[placeholder]',
          'div#prompt-textarea',
          'div[contenteditable="true"][data-lexical-editor="true"]',
          'main div[contenteditable="true"]'
        ];
        for (const selector of selectors) {
          const candidates = Array.from(document.querySelectorAll(selector));
          const match = candidates.find((element) => isVisible(element));
          if (match) return match;
        }
        return null;
      };
      const findLoginButton = () => {
        const candidates = Array.from(document.querySelectorAll('button, a'));
        return candidates.find((element) => {
          const text = (element.innerText || element.textContent || '').trim();
          return isVisible(element) && (text === '登录' || text === 'Log in');
        }) || null;
      };
      const findSendButton = () => {
        const selectors = [
          "button[data-testid='send-button']",
          "button[aria-label*='发送']",
          "button[aria-label*='Send']"
        ];
        for (const selector of selectors) {
          const button = document.querySelector(selector);
          if (button && isVisible(button)) {
            return button;
          }
        }
        return null;
      };
      const findStopButton = () => {
        const selectors = [
          "button[data-testid='stop-button']",
          "button[aria-label*='停止']",
          "button[aria-label*='Stop']"
        ];
        for (const selector of selectors) {
          const button = document.querySelector(selector);
          if (button && isVisible(button)) {
            return button;
          }
        }
        const visibleButtons = Array.from(document.querySelectorAll('button, [role="button"]'));
        return visibleButtons.find((button) => (
          isVisible(button) && includesKeyword(elementSignalText(button), STOP_KEYWORDS)
        )) || null;
      };
      const findVisibleGenerationProgressNode = () => {
        const candidates = Array.from(document.querySelectorAll('main *'));
        return candidates.find((element) => {
          if (!isVisible(element)) {
            return false;
          }
          const signalText = elementSignalText(element);
          if (!signalText || signalText.length > 120) {
            return false;
          }
          return includesKeyword(signalText, IMAGE_PROGRESS_KEYWORDS);
        }) || null;
      };
      const findMainBusyNode = () => {
        const busyNodes = Array.from(document.querySelectorAll('[aria-busy="true"]'));
        return busyNodes.find((element) => isVisible(element) && Boolean(element.closest('main'))) || null;
      };
      const collectCandidateChoiceControls = (sections) => {
        const controls = [];
        const seen = new Set();
        for (const section of sections || []) {
          for (const control of Array.from(section.querySelectorAll('button, [role="button"], a[href]'))) {
            if (!control || seen.has(control) || !isVisible(control)) {
              continue;
            }
            seen.add(control);
            const signalText = elementSignalText(control);
            if (
              includesKeyword(signalText, IMAGE_CHOICE_SKIP_KEYWORDS) ||
              includesKeyword(signalText, IMAGE_CHOICE_PREFERENCE_KEYWORDS)
            ) {
              controls.push({
                element: control,
                signalText,
                isSkip: includesKeyword(signalText, IMAGE_CHOICE_SKIP_KEYWORDS),
                isPreference: includesKeyword(signalText, IMAGE_CHOICE_PREFERENCE_KEYWORDS)
              });
            }
          }
        }
        return controls;
      };
      const extractFileIdFromValue = (value) => {
        const raw = String(value || '');
        const markerIndex = raw.indexOf('file_');
        if (markerIndex < 0) {
          return null;
        }
        const rest = raw.slice(markerIndex);
        const endIndex = rest.search(/[/?#&]/);
        if (endIndex < 0) {
          return rest || null;
        }
        return rest.slice(0, endIndex) || null;
      };
      const assistantTurnNodes = () => Array.from((document.querySelector('main') || document).querySelectorAll(
        'section, article, [role="article"], [data-message-author-role="assistant"]'
      ))
        .filter((element) => isVisible(element));
      const candidateChoiceScopes = () => assistantTurnNodes().filter((element) => {
        if (includesKeyword(sectionSignalText(element), IMAGE_CHOICE_PROMPT_KEYWORDS)) {
          return true;
        }
        const controls = collectCandidateChoiceControls([element]);
        return controls.some((entry) => entry.isSkip || entry.isPreference);
      });
      const latestAssistantScope = () => assistantTurnNodes().at(-1) || document.querySelector('main') || document.body;
      const collectVisibleImageFileIds = (scopes) => {
        const fileIds = [];
        const seen = new Set();
        const candidates = [];
        for (const scope of scopes || []) {
          for (const element of Array.from(scope.querySelectorAll('img[src], a[href], source[srcset]'))) {
            if (!element || !isVisible(element)) {
              continue;
            }
            candidates.push(element);
          }
        }
        for (const element of candidates) {
          const values = [
            element.getAttribute?.('src'),
            element.getAttribute?.('href'),
            element.getAttribute?.('srcset')
          ];
          for (const value of values) {
            const fileId = extractFileIdFromValue(value);
            if (fileId && !seen.has(fileId)) {
              seen.add(fileId);
              fileIds.push(fileId);
            }
          }
        }
        return fileIds;
      };
      const pageCandidateChoiceState = () => {
        const main = document.querySelector('main') || document.body;
        const scopes = main ? [main] : [];
        const controls = collectCandidateChoiceControls(scopes);
        const skipButton = controls.find((entry) => entry.isSkip)?.element || null;
        const preferenceButtons = controls.filter((entry) => entry.isPreference);
        const choicePromptVisible = assistantTurnNodes()
          .some((element) => includesKeyword(sectionSignalText(element), IMAGE_CHOICE_PROMPT_KEYWORDS));
        return {
          candidateChoiceVisible: choicePromptVisible || Boolean(skipButton) || preferenceButtons.length > 0,
          choicePromptVisible,
          skipButton,
          preferenceButtons
        };
      };
      const generationSignals = () => {
        const sendButton = findSendButton();
        const stopButton = findStopButton();
        const busyNode = findMainBusyNode();
        const progressNode = findVisibleGenerationProgressNode();
        const busyGenerating = Boolean(stopButton) || Boolean(busyNode) || Boolean(progressNode);
        return {
          busyGenerating,
          stopButtonVisible: Boolean(stopButton),
          sendButtonEnabled: Boolean(sendButton && !sendButton.disabled),
          busyNodeVisible: Boolean(busyNode),
          progressText: progressNode ? elementSignalText(progressNode) : null
        };
      };
      const pageStatus = () => {
        const composer = findComposer();
        const loginButton = findLoginButton();
        const candidateChoice = pageCandidateChoiceState();
        return {
          url: location.href,
          title: document.title,
          composerFound: Boolean(composer),
          loginRequired: Boolean(loginButton),
          loginButtonText: loginButton ? (loginButton.innerText || loginButton.textContent || '').trim() : null,
          candidateChoiceVisible: candidateChoice.candidateChoiceVisible,
          preferenceButtonCount: candidateChoice.preferenceButtons.length,
          ...generationSignals()
        };
      };
      const composerText = (element) => {
        if (element instanceof HTMLTextAreaElement || element instanceof HTMLInputElement) {
          return element.value || '';
        }
        if (element?.isContentEditable) {
          return element.innerText || element.textContent || '';
        }
        return '';
      };
      const dispatchInput = (element, inputType, data = null) => {
        element.dispatchEvent(new InputEvent('input', {
          bubbles: true,
          composed: true,
          inputType,
          data
        }));
      };
      const replaceContentEditableText = (element, prompt) => {
        element.focus();
        const selection = window.getSelection();
        if (selection) {
          selection.removeAllRanges();
          const range = document.createRange();
          range.selectNodeContents(element);
          selection.addRange(range);
        }
        let inserted = false;
        try {
          inserted = document.execCommand('insertText', false, prompt);
        } catch {
          inserted = false;
        }
        if (!inserted) {
          element.textContent = '';
          const textNode = document.createTextNode(prompt);
          element.append(textNode);
          if (selection) {
            selection.removeAllRanges();
            const range = document.createRange();
            range.selectNodeContents(element);
            range.collapse(false);
            selection.addRange(range);
          }
        }
        dispatchInput(element, 'insertText', prompt);
      };
      const setComposerValue = (element, prompt) => {
        element.focus();
        if (element instanceof HTMLTextAreaElement || element instanceof HTMLInputElement) {
          const setter = nativeValueSetter(element);
          if (setter) {
            setter.call(element, prompt);
          } else {
            element.value = prompt;
          }
          element.dispatchEvent(new Event('input', { bubbles: true }));
          element.dispatchEvent(new Event('change', { bubbles: true }));
          return;
        }
        if (element.isContentEditable) {
          replaceContentEditableText(element, prompt);
          return;
        }
        throw new Error('Unsupported composer element.');
      };
      const submitPrompt = async (element) => {
        const deadline = Date.now() + 5000;
        while (Date.now() < deadline) {
          const button = findSendButton();
          if (button && !button.disabled) {
            button.click();
            return;
          }
          await delay(100);
        }
        for (const type of ['keydown', 'keypress', 'keyup']) {
          element.dispatchEvent(new KeyboardEvent(type, {
            key: 'Enter',
            code: 'Enter',
            which: 13,
            keyCode: 13,
            bubbles: true,
            composed: true
          }));
        }
      };
      const scrollElementIntoView = (element, block = 'center') => {
        if (!element?.scrollIntoView) {
          return;
        }
        try {
          element.scrollIntoView({
            behavior: 'instant',
            block,
            inline: 'nearest'
          });
        } catch {
          element.scrollIntoView();
        }
      };
      const scrollConversationToBottom = () => {
        const composer = findComposer();
        const main = document.querySelector('main');
        const scrollers = [composer, main, main?.parentElement, document.scrollingElement, document.documentElement, document.body]
          .filter(Boolean);
        for (const element of scrollers) {
          try {
            if (typeof element.scrollTo === 'function') {
              element.scrollTo({
                top: element.scrollHeight || Number.MAX_SAFE_INTEGER,
                behavior: 'instant'
              });
            } else if ('scrollTop' in element) {
              element.scrollTop = element.scrollHeight || Number.MAX_SAFE_INTEGER;
            }
          } catch {}
        }
        scrollElementIntoView(composer, 'end');
      };
      const ensureBottomContext = async () => {
        scrollConversationToBottom();
        await delay(200);
        scrollConversationToBottom();
      };
      const maybeSkipCandidateChoice = async () => {
        const state = pageCandidateChoiceState();
        if (!state.candidateChoiceVisible || !state.skipButton) {
          return false;
        }
        state.skipButton.click();
        await delay(1500);
        return true;
      };
      const resolveLingeringCandidateChoice = async (timeoutMs = 10000) => {
        const deadline = Date.now() + timeoutMs;
        while (Date.now() < deadline) {
          const state = pageCandidateChoiceState();
          if (!state.candidateChoiceVisible) {
            return false;
          }
          const skipped = await maybeSkipCandidateChoice();
          if (!skipped) {
            await delay(1000);
          }
        }
        throw new Error('Lingering candidate image choice could not be cleared before submission.');
      };
      const run = async () => {
        if (payload.action === 'status') {
          return pageStatus();
        }
        if (payload.action === 'resolve_candidate_choice') {
          const timeoutMs = Math.max(1000, Number(payload.timeoutMs || 15000));
          const postClearSettleMs = Math.max(0, Number(payload.postClearSettleMs || 1500));
          const initialScopes = candidateChoiceScopes();
          const initialCandidateFileIds = collectVisibleImageFileIds(
            initialScopes.length ? initialScopes : [latestAssistantScope()]
          );
          const deadline = Date.now() + timeoutMs;
          let clickedSkip = false;

          while (Date.now() < deadline) {
            const state = pageCandidateChoiceState();
            if (!state.candidateChoiceVisible) {
              break;
            }
            if (state.skipButton) {
              scrollElementIntoView(state.skipButton, 'center');
              state.skipButton.click();
              clickedSkip = true;
              await delay(1200);
              continue;
            }
            await delay(500);
          }

          const afterAttemptState = pageCandidateChoiceState();
          const choiceCleared = !afterAttemptState.candidateChoiceVisible;
          if (choiceCleared && postClearSettleMs > 0) {
            await delay(postClearSettleMs);
          }

          const resolvedScopes = candidateChoiceScopes();
          const chosenFileIds = collectVisibleImageFileIds(
            resolvedScopes.length ? resolvedScopes : [latestAssistantScope()]
          );

          return {
            clickedSkip,
            choiceCleared,
            initialCandidateFileIds,
            chosenFileIds,
            pageStatus: pageStatus()
          };
        }
        if (payload.action === 'send_prompt') {
          const prompt = String(payload.prompt || '').trim();
          if (!prompt) {
            throw new Error('Prompt cannot be empty.');
          }
          await resolveLingeringCandidateChoice(10000);
          const before = pageStatus();
          if (before.loginRequired) {
            throw new Error('ChatGPT page is not logged in.');
          }
          if (before.busyGenerating) {
            throw new Error('ChatGPT page is still generating.');
          }
          const composer = findComposer();
          if (!composer) {
            throw new Error('Could not find the ChatGPT composer.');
          }
          await ensureBottomContext();
          setComposerValue(composer, prompt);
          if (!composerText(composer).trim()) {
            throw new Error('Prompt was not inserted into the ChatGPT composer.');
          }
          await delay(200);
          await submitPrompt(composer);
          return {
            submitted: true,
            promptLength: prompt.length,
            pageStatus: pageStatus()
          };
        }
        throw new Error(\`Unsupported page action: \${payload.action}\`);
      };
      return run();
    })()`;

    const result = await this.connection.send('Runtime.evaluate', {
      expression,
      awaitPromise: true,
      returnByValue: true
    }, this.activeSessionId);

    if (result?.exceptionDetails) {
      const description = result.exceptionDetails?.exception?.description
        || result.exceptionDetails?.text
        || 'Runtime.evaluate failed.';
      throw new Error(description);
    }

    return result?.result?.value ?? null;
  }

  failAssignedJob(run, code, message, extra = {}) {
    if (!run?.assignedJobId) {
      return;
    }
    const job = this.jobs.get(run.assignedJobId);
    if (!job || ['ready', 'failed', 'acked', 'cancelled'].includes(job.state)) {
      return;
    }
    this.updateJob(job, {
      state: 'failed',
      error: {
        code,
        message
      }
    }, 'job_failed', {
      runIndex: run.runIndex,
      ...extra
    });
  }

  async waitForPotentialCandidateChoice(run, timeoutMs = 2500, pollMs = 250) {
    const deadline = Date.now() + timeoutMs;
    let lastStatus = null;

    while (Date.now() < deadline) {
      try {
        lastStatus = await this.evaluatePageAction('status');
      } catch (error) {
        this.writeRecord({
          source: 'observer_v3',
          event: 'candidate_choice_status_error',
          runIndex: run?.runIndex || null,
          error: error instanceof Error ? error.message : String(error)
        });
        return null;
      }

      if (lastStatus?.candidateChoiceVisible) {
        return lastStatus;
      }

      await new Promise((resolve) => setTimeout(resolve, pollMs));
    }

    return lastStatus;
  }

  async resolveCandidateChoiceForRun(run, reason) {
    const initialStatus = await this.waitForPotentialCandidateChoice(run, 2500, 250);
    if (!initialStatus?.candidateChoiceVisible) {
      return {
        hadCandidateChoice: false,
        preferredFileIds: []
      };
    }

    this.writeRecord({
      source: 'observer_v3',
      event: 'candidate_choice_detected',
      runIndex: run.runIndex,
      reason,
      preferenceButtonCount: initialStatus.preferenceButtonCount || 0
    });

    let resolution = null;
    try {
      resolution = await this.evaluatePageAction('resolve_candidate_choice', {
        timeoutMs: 15000,
        postClearSettleMs: 1500
      });
    } catch (error) {
      this.failAssignedJob(
        run,
        'candidate_choice_resolution_failed',
        error instanceof Error ? error.message : String(error),
        { reason }
      );
      return {
        hadCandidateChoice: true,
        failed: true,
        preferredFileIds: []
      };
    }

    this.writeRecord({
      source: 'observer_v3',
      event: 'candidate_choice_resolved',
      runIndex: run.runIndex,
      reason,
      clickedSkip: Boolean(resolution?.clickedSkip),
      choiceCleared: Boolean(resolution?.choiceCleared),
      initialCandidateFileIds: Array.isArray(resolution?.initialCandidateFileIds) ? resolution.initialCandidateFileIds : [],
      chosenFileIds: Array.isArray(resolution?.chosenFileIds) ? resolution.chosenFileIds : [],
      pageStatus: resolution?.pageStatus || null
    });

    if (!resolution?.choiceCleared) {
      this.failAssignedJob(
        run,
        'candidate_choice_timeout',
        'Candidate image choice could not be cleared automatically.',
        { reason }
      );
      return {
        hadCandidateChoice: true,
        failed: true,
        preferredFileIds: []
      };
    }

    return {
      hadCandidateChoice: true,
      failed: false,
      preferredFileIds: Array.isArray(resolution?.chosenFileIds) ? resolution.chosenFileIds.filter(Boolean) : []
    };
  }

  async finalizeAssignedJob(run, reason) {
    if (!run || run.finalized) {
      return false;
    }

    let preferredFileIds = [];
    if (validatedFinalCount(run) > 1) {
      const candidateResolution = await this.resolveCandidateChoiceForRun(run, reason);
      if (candidateResolution?.failed) {
        return false;
      }
      preferredFileIds = candidateResolution?.preferredFileIds || [];
    }

    const candidate = latestValidatedFinal(run, preferredFileIds);
    if (!candidate) {
      this.writeRecord({
        source: 'observer_v3',
        event: 'finalize_pending',
        reason,
        runIndex: run.runIndex,
        assignedJobId: run.assignedJobId,
        asyncCount: run.asyncEvents.length,
        fileDownloadCount: run.fileDownloads.length,
        estuaryCount: run.estuaryContents.length,
        validatedFinalCount: validatedFinalCount(run)
      });
      return false;
    }

    run.finalized = true;
    this.clearRunTimers(run);

    if (!run.assignedJobId) {
      this.writeRecord({
        source: 'observer_v3',
        event: 'final_ready_without_job',
        reason,
        runIndex: run.runIndex,
        fileId: candidate.fileId,
        fileName: candidate.fileName
      });
      return true;
    }

    const job = this.jobs.get(run.assignedJobId);
    if (!job) {
      return true;
    }
    if (job.state === 'cancelled') {
      this.writeRecord({
        source: 'observer_v3',
        event: 'final_dropped_for_cancelled_job',
        runIndex: run.runIndex,
        jobId: job.id,
        reason
      });
      return true;
    }

    const ext = path.extname(String(candidate.fileName || '')) || extFromMimeType(candidate.mimeType);
    const cacheToken = `${job.id}${ext}`;
    const cachePath = path.join(this.paths.cache, cacheToken);
    fs.writeFileSync(cachePath, candidate.bodyBuffer);

    this.updateJob(job, {
      state: 'ready',
      result: {
        cacheToken,
        cachePath,
        fileName: String(candidate.fileName || '').split('/').pop() || `capture${ext}`,
        mimeType: candidate.mimeType,
        sizeBytes: candidate.bodyBuffer.length,
        readyAt: nowIso()
      }
    }, 'job_ready', {
      runIndex: run.runIndex,
      reason,
      fileId: candidate.fileId,
      sizeBytes: candidate.bodyBuffer.length,
      validatedFinalCount: validatedFinalCount(run),
      preferredFileIds
    });

    return true;
  }

  scheduleAsyncResolution(run) {
    this.clearRunTimers(run);
    run.settleTimer = setTimeout(() => {
      this.finalizeAssignedJob(run, 'async_ok_settle_elapsed').catch((error) => {
        this.writeRecord({
          source: 'observer_v3',
          event: 'settle_timer_error',
          runIndex: run.runIndex,
          error: error instanceof Error ? error.message : String(error)
        });
      });
    }, this.options.settleMs);

    run.graceTimer = setTimeout(() => {
      try {
        if (!run.finalized) {
          this.failAssignedJob(
            run,
            'async_ok_timeout',
            `No validated final image arrived within ${this.options.graceMs}ms after async-status OK.`,
            { asyncOkAtMs: run.asyncOkAtMs }
          );
        }
      } catch (error) {
        this.writeRecord({
          source: 'observer_v3',
          event: 'grace_timer_error',
          runIndex: run.runIndex,
          error: error instanceof Error ? error.message : String(error)
        });
      }
    }, this.options.graceMs);
  }

  async maybeFinalizeAfterArtifact(run, reason) {
    if (!run?.waitingForFinalAfterAsync || run.finalized || !run.asyncOkAtMs) {
      return;
    }
    const elapsed = Date.now() - run.asyncOkAtMs;
    if (elapsed >= this.options.settleMs) {
      await this.finalizeAssignedJob(run, reason);
    }
  }

  async handleCdpMessage(message) {
    const { method, params, sessionId } = message;
    if (!method || sessionId !== this.activeSessionId) {
      return;
    }

    if (method === 'Network.requestWillBeSent') {
      const label = networkLabel(params.request?.url);
      if (!label) {
        return;
      }
      const item = {
        sequence: ++this.eventSequence,
        label,
        url: params.request.url,
        method: params.request.method,
        requestId: params.requestId,
        fileId: extractFileId(params.request.url),
        startedAt: nowIso()
      };

      if (label === 'conversation') {
        if (this.activeRun && !this.activeRun.finalized && this.activeRun.assignedJobId) {
          this.failAssignedJob(
            this.activeRun,
            'superseded_by_new_run',
            'A new conversation run started before the previous assigned run completed.'
          );
          this.clearRunTimers(this.activeRun);
        }
        this.runIndex += 1;
        this.activeRun = createRun(
          this.runIndex,
          params.requestId,
          params.request.url.match(/conversation\/([^/?#]+)/i)?.[1] || null
        );
        this.assignNextWaitingJob(this.activeRun);
        this.writeRecord({
          source: 'observer_v3',
          event: 'run_started',
          runIndex: this.activeRun.runIndex,
          requestId: params.requestId,
          url: params.request.url,
          assignedJobId: this.activeRun.assignedJobId
        });
      }

      if (this.activeRun) {
        item.runIndex = this.activeRun.runIndex;
      }
      this.requestState.set(params.requestId, item);

      this.writeRecord({
        source: 'network',
        event: 'request',
        ...item
      });
      return;
    }

    if (method === 'Network.responseReceived') {
      const item = this.requestState.get(params.requestId);
      if (!item) {
        return;
      }
      item.status = params.response.status;
      item.statusText = params.response.statusText;
      item.mimeType = params.response.mimeType;
      this.requestState.set(params.requestId, item);

      this.writeRecord({
        source: 'network',
        event: 'response',
        requestId: params.requestId,
        runIndex: item.runIndex || null,
        label: item.label,
        status: item.status,
        mimeType: item.mimeType,
        fileId: item.fileId || null,
        url: item.url
      });
      return;
    }

    if (method !== 'Network.loadingFinished') {
      return;
    }

    const item = this.requestState.get(params.requestId);
    if (!item || !this.activeRun || item.runIndex !== this.activeRun.runIndex) {
      return;
    }

    item.sequence = ++this.eventSequence;
    item.loadingFinishedAt = nowIso();
    this.requestState.set(params.requestId, item);

    if (item.label === 'file_download') {
      const bodyResult = await this.connection.send(
        'Network.getResponseBody',
        { requestId: params.requestId },
        this.activeSessionId
      );
      const bodyText = bodyResult.base64Encoded
        ? Buffer.from(bodyResult.body || '', 'base64').toString('utf8')
        : String(bodyResult.body || '');
      const parsed = readJson(bodyText, {});
      const fileName = String(parsed.file_name || '');
      const record = {
        sequence: item.sequence,
        requestId: params.requestId,
        fileId: item.fileId || extractFileId(parsed.download_url),
        url: item.url,
        status: parsed.status || null,
        fileName,
        isPartial: /\.part\d+\./i.test(fileName),
        downloadUrl: parsed.download_url || null,
        fileSizeBytes: parsed.file_size_bytes || null
      };
      this.activeRun.fileDownloads.push(record);
      this.writeRecord({
        source: 'observer_v3',
        event: 'file_download_ready',
        runIndex: this.activeRun.runIndex,
        ...record
      });
      await this.maybeFinalizeAfterArtifact(this.activeRun, 'file_download_after_async');
      return;
    }

    if (item.label === 'estuary_content') {
      const bodyResult = await this.connection.send(
        'Network.getResponseBody',
        { requestId: params.requestId },
        this.activeSessionId
      );
      const bodyBuffer = bodyResult.base64Encoded
        ? Buffer.from(bodyResult.body || '', 'base64')
        : Buffer.from(String(bodyResult.body || ''), 'utf8');
      const record = {
        sequence: item.sequence,
        requestId: params.requestId,
        fileId: item.fileId,
        url: item.url,
        status: item.status || 200,
        mimeType: item.mimeType || 'application/octet-stream',
        encodedDataLength: params.encodedDataLength || 0,
        bodyBuffer
      };
      this.activeRun.estuaryContents.push(record);
      this.writeRecord({
        source: 'observer_v3',
        event: 'estuary_content_ready',
        runIndex: this.activeRun.runIndex,
        requestId: record.requestId,
        fileId: record.fileId,
        mimeType: record.mimeType,
        encodedDataLength: record.encodedDataLength,
        status: record.status
      });
      await this.maybeFinalizeAfterArtifact(this.activeRun, 'estuary_after_async');
      return;
    }

    if (item.label === 'async_status') {
      const bodyResult = await this.connection.send(
        'Network.getResponseBody',
        { requestId: params.requestId },
        this.activeSessionId
      );
      const bodyText = bodyResult.base64Encoded
        ? Buffer.from(bodyResult.body || '', 'base64').toString('utf8')
        : String(bodyResult.body || '');
      const parsed = readJson(bodyText, {});
      const record = {
        sequence: item.sequence,
        requestId: params.requestId,
        url: item.url,
        body: parsed
      };
      this.activeRun.asyncEvents.push(record);
      this.writeRecord({
        source: 'observer_v3',
        event: 'async_status_seen',
        runIndex: this.activeRun.runIndex,
        requestId: params.requestId,
        body: parsed
      });

      if (parsed.status === 'OK') {
        this.activeRun.waitingForFinalAfterAsync = true;
        this.activeRun.asyncOkAtMs = Date.now();
        this.scheduleAsyncResolution(this.activeRun);
      }
    }
  }

  async initCdp() {
    const devTools = readDevToolsActivePort(this.options.userDataDir);
    this.connection = new CdpConnection(devTools.wsUrl);
    await this.connection.connect();
    this.connection.onEvent((message) => {
      this.handleCdpMessage(message).catch((error) => {
        this.writeRecord({
          source: 'observer_v3',
          event: 'listener_error',
          error: error instanceof Error ? error.message : String(error)
        });
      });
    });

    const targets = await this.connection.send('Target.getTargets');
    const candidates = (targets.targetInfos || []).filter((target) => (
      target.type === 'page' &&
      String(target.url || '').includes(this.options.urlSubstring)
    ));
    if (!candidates.length) {
      throw new Error(`No page target matched url substring: ${this.options.urlSubstring}`);
    }

    const target = candidates.at(-1);
    const attachResult = await this.connection.send('Target.attachToTarget', {
      targetId: target.targetId,
      flatten: true
    });
    this.activeSessionId = attachResult.sessionId;
    this.activeTarget = {
      targetId: target.targetId,
      title: target.title,
      url: target.url
    };

    await this.connection.send('Network.enable', {
      maxTotalBufferSize: 100 * 1024 * 1024,
      maxResourceBufferSize: 20 * 1024 * 1024
    }, this.activeSessionId);
    await this.connection.send('Page.enable', {}, this.activeSessionId);
    await this.connection.send('Runtime.enable', {}, this.activeSessionId);

    this.writeRecord({
      source: 'observer_v3',
      event: 'cdp_ready',
      wsUrl: devTools.wsUrl,
      target: this.activeTarget
    });
  }

  readRequestBody(req) {
    return new Promise((resolve, reject) => {
      let raw = '';
      req.on('data', (chunk) => {
        raw += String(chunk);
        if (raw.length > 5 * 1024 * 1024) {
          reject(new Error('Request body too large.'));
          req.destroy();
        }
      });
      req.on('end', () => {
        if (!raw) {
          resolve({});
          return;
        }
        try {
          resolve(JSON.parse(raw));
        } catch (error) {
          reject(new Error(`Invalid JSON body: ${error instanceof Error ? error.message : String(error)}`));
        }
      });
      req.on('error', reject);
    });
  }

  sendJson(res, statusCode, payload) {
    const body = `${JSON.stringify(payload, null, 2)}\n`;
    res.writeHead(statusCode, {
      'Content-Type': 'application/json; charset=utf-8',
      'Content-Length': Buffer.byteLength(body),
      'Access-Control-Allow-Origin': '*',
      'Access-Control-Allow-Methods': 'GET,POST,OPTIONS',
      'Access-Control-Allow-Headers': 'Content-Type'
    });
    res.end(body);
  }

  sendBinary(res, statusCode, headers, bodyBuffer) {
    res.writeHead(statusCode, {
      'Access-Control-Allow-Origin': '*',
      ...headers
    });
    res.end(bodyBuffer);
  }

  async routeRequest(req, res) {
    const url = new URL(req.url || '/', `http://${this.options.host}:${this.options.port}`);
    const pathname = url.pathname;

    if (req.method === 'OPTIONS') {
      this.sendJson(res, 204, {});
      return;
    }

    if (req.method === 'GET' && pathname === '/healthz') {
      this.sendJson(res, 200, {
        ok: true,
        serverSessionId: this.serverSessionId,
        target: this.activeTarget,
        activeRun: this.snapshotRun(),
        waitingJobs: this.waitingJobIds.length,
        totalJobs: this.jobs.size
      });
      return;
    }

    if (req.method === 'GET' && pathname === '/debug/state') {
      this.sendJson(res, 200, {
        ok: true,
        serverSessionId: this.serverSessionId,
        target: this.activeTarget,
        activeRun: this.snapshotRun(),
        jobs: Array.from(this.jobs.values()).map((job) => this.snapshotJob(job)),
        debugEvents: this.debugEvents
      });
      return;
    }

    if (req.method === 'GET' && pathname === '/debug/events') {
      const limit = url.searchParams.get('limit') || DEFAULT_DEBUG_EVENTS_LIMIT;
      const eventName = url.searchParams.get('event') || '';
      const items = this.listDebugEvents(limit, eventName);
      this.sendJson(res, 200, {
        ok: true,
        serverSessionId: this.serverSessionId,
        filter: {
          event: eventName || null
        },
        count: items.length,
        items
      });
      return;
    }

    if (req.method === 'GET' && pathname === '/page/status') {
      const status = await this.evaluatePageAction('status');
      this.sendJson(res, 200, {
        ok: true,
        page: status
      });
      return;
    }

    if (req.method === 'POST' && pathname === '/jobs/register') {
      const payload = await this.readRequestBody(req);
      const job = this.registerJob(payload);
      this.sendJson(res, 201, { ok: true, job });
      return;
    }

    const waitMatch = pathname.match(/^\/jobs\/([^/]+)\/wait$/);
    if (req.method === 'GET' && waitMatch) {
      const jobId = decodeURIComponent(waitMatch[1]);
      const sinceVersion = Number(url.searchParams.get('since_version') || '0');
      const timeoutMs = Number(url.searchParams.get('timeout_ms') || '30000');
      if (!Number.isFinite(timeoutMs) || timeoutMs <= 0 || timeoutMs > 120000) {
        this.sendJson(res, 400, { ok: false, error: 'Invalid timeout_ms.' });
        return;
      }
      const result = await this.waitForJobUpdate(jobId, sinceVersion, timeoutMs);
      this.sendJson(res, 200, { ok: true, ...result });
      return;
    }

    const resultMatch = pathname.match(/^\/jobs\/([^/]+)\/result$/);
    if (req.method === 'GET' && resultMatch) {
      const jobId = decodeURIComponent(resultMatch[1]);
      const job = this.jobs.get(jobId);
      if (!job) {
        this.sendJson(res, 404, { ok: false, error: 'Job not found.' });
        return;
      }
      if (job.state !== 'ready' || !job.result?.cachePath || !fs.existsSync(job.result.cachePath)) {
        this.sendJson(res, 409, { ok: false, error: 'Result not ready.' });
        return;
      }
      const body = fs.readFileSync(job.result.cachePath);
      this.sendBinary(res, 200, {
        'Content-Type': job.result.mimeType || 'application/octet-stream',
        'Content-Length': body.length,
        'Content-Disposition': `attachment; filename="${job.result.fileName}"`,
        'X-Observer-Job-Id': job.id,
        'X-Observer-File-Name': encodeURIComponent(job.result.fileName),
        'X-Observer-Mime-Type': job.result.mimeType || 'application/octet-stream'
      }, body);
      return;
    }

    const ackMatch = pathname.match(/^\/jobs\/([^/]+)\/ack$/);
    if (req.method === 'POST' && ackMatch) {
      const jobId = decodeURIComponent(ackMatch[1]);
      const job = this.ackJob(jobId);
      this.sendJson(res, 200, { ok: true, job });
      return;
    }

    const cancelMatch = pathname.match(/^\/jobs\/([^/]+)\/cancel$/);
    if (req.method === 'POST' && cancelMatch) {
      const jobId = decodeURIComponent(cancelMatch[1]);
      const job = this.cancelJob(jobId);
      this.sendJson(res, 200, { ok: true, job });
      return;
    }

    if (req.method === 'POST' && pathname === '/actions/send-prompt') {
      const payload = await this.readRequestBody(req);
      const result = await this.evaluatePageAction('send_prompt', {
        prompt: String(payload?.prompt || '')
      });
      this.sendJson(res, 200, {
        ok: true,
        result
      });
      return;
    }

    if (req.method === 'POST' && pathname === '/actions/resolve-candidate-choice') {
      const payload = await this.readRequestBody(req);
      const result = await this.evaluatePageAction('resolve_candidate_choice', {
        timeoutMs: Number(payload?.timeoutMs || 15000),
        postClearSettleMs: Number(payload?.postClearSettleMs || 1500)
      });
      this.sendJson(res, 200, {
        ok: true,
        result
      });
      return;
    }

    const jobMatch = pathname.match(/^\/jobs\/([^/]+)$/);
    if (req.method === 'GET' && jobMatch) {
      const jobId = decodeURIComponent(jobMatch[1]);
      const job = this.jobById(jobId);
      if (!job) {
        this.sendJson(res, 404, { ok: false, error: 'Job not found.' });
        return;
      }
      this.sendJson(res, 200, { ok: true, job });
      return;
    }

    this.sendJson(res, 404, {
      ok: false,
      error: `Unknown route: ${req.method} ${pathname}`
    });
  }

  async startHttpServer() {
    this.httpServer = http.createServer((req, res) => {
      this.routeRequest(req, res).catch((error) => {
        this.writeRecord({
          source: 'observer_v3',
          event: 'http_error',
          error: error instanceof Error ? error.message : String(error),
          method: req.method,
          url: req.url
        });
        this.sendJson(res, 500, {
          ok: false,
          error: error instanceof Error ? error.message : String(error)
        });
      });
    });

    await new Promise((resolve, reject) => {
      this.httpServer.once('error', reject);
      this.httpServer.listen(this.options.port, this.options.host, () => resolve());
    });
  }

  async start() {
    this.writeRecord({
      source: 'observer_v3',
      event: 'startup',
      options: {
        host: this.options.host,
        port: this.options.port,
        stateDir: this.options.stateDir,
        userDataDir: this.options.userDataDir,
        urlSubstring: this.options.urlSubstring,
        settleMs: this.options.settleMs,
        graceMs: this.options.graceMs
      }
    });

    await this.initCdp();
    await this.startHttpServer();

    console.log('ChatGPT CDP observer V3 is ready.');
    console.log(`HTTP: http://${this.options.host}:${this.options.port}`);
    console.log(`Target: ${this.activeTarget.title} <${this.activeTarget.url}>`);
    console.log(`Logs: ${this.log.path}`);
    console.log(`Cache dir: ${this.paths.cache}`);
  }

  async shutdown(reason) {
    if (this.shuttingDown) {
      return;
    }
    this.shuttingDown = true;

    this.writeRecord({
      source: 'observer_v3',
      event: 'shutdown',
      reason,
      activeRun: this.snapshotRun()
    });

    if (this.activeRun) {
      this.clearRunTimers(this.activeRun);
    }

    for (const waiters of this.jobWaiters.values()) {
      for (const waiter of waiters) {
        clearTimeout(waiter.timer);
        waiter.resolve({
          timedOut: false,
          job: null,
          shuttingDown: true
        });
      }
    }
    this.jobWaiters.clear();

    if (this.httpServer) {
      await new Promise((resolve) => this.httpServer.close(() => resolve()));
    }
    if (this.connection && this.activeSessionId) {
      try {
        await this.connection.send('Target.detachFromTarget', { sessionId: this.activeSessionId });
      } catch {}
    }
    if (this.connection) {
      try {
        await this.connection.close();
      } catch {}
    }
    await this.log.close();
  }
}

async function main() {
  const options = parseArgs(process.argv);
  const server = new ObserverServer(options);

  process.on('SIGINT', async () => {
    await server.shutdown('sigint');
    process.exit(0);
  });
  process.on('SIGTERM', async () => {
    await server.shutdown('sigterm');
    process.exit(0);
  });

  await server.start();
}

main().catch((error) => {
  console.error(`ERROR: ${error instanceof Error ? error.message : String(error)}`);
  process.exit(1);
});
