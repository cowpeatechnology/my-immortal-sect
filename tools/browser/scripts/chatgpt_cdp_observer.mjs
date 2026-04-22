#!/usr/bin/env node

import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import process from 'node:process';
import { setTimeout as delay } from 'node:timers/promises';

const DEFAULT_USER_DATA_DIR = path.join(
  os.homedir(),
  'Library',
  'Application Support',
  'Google',
  'Chrome'
);
const DEFAULT_OUTPUT_DIR = path.join(
  '/Users/mawei/MyWork/SlgGame/workspace/output/browser',
  'chatgpt-observer'
);
const DEFAULT_URL_SUBSTRING = 'chatgpt.com';
const DEFAULT_SAMPLE_INTERVAL_MS = 500;
const BINDING_NAME = '__codexObserveEmit';

const NETWORK_LABELS = [
  { label: 'conversation', pattern: /\/backend-api\/f\/conversation(?:\?|$)/i },
  { label: 'conversation_prepare', pattern: /\/backend-api\/f\/conversation\/prepare(?:\?|$)/i },
  { label: 'async_status', pattern: /\/backend-api\/conversation\/[^/]+\/async-status(?:\?|$)/i },
  { label: 'file_download', pattern: /\/backend-api\/files\/download\/[^/?#]+/i },
  { label: 'estuary_content', pattern: /\/backend-api\/estuary\/content(?:\?|$)/i }
];

function parseArgs(argv) {
  const options = {
    userDataDir: DEFAULT_USER_DATA_DIR,
    outputDir: DEFAULT_OUTPUT_DIR,
    urlSubstring: DEFAULT_URL_SUBSTRING,
    sampleIntervalMs: DEFAULT_SAMPLE_INTERVAL_MS,
    durationSeconds: 0
  };

  for (let index = 2; index < argv.length; index += 1) {
    const arg = argv[index];
    const next = argv[index + 1];
    if (arg === '--user-data-dir' && next) {
      options.userDataDir = next;
      index += 1;
      continue;
    }
    if (arg === '--output-dir' && next) {
      options.outputDir = next;
      index += 1;
      continue;
    }
    if (arg === '--url-substring' && next) {
      options.urlSubstring = next;
      index += 1;
      continue;
    }
    if (arg === '--sample-interval-ms' && next) {
      options.sampleIntervalMs = Number(next);
      index += 1;
      continue;
    }
    if (arg === '--duration-seconds' && next) {
      options.durationSeconds = Number(next);
      index += 1;
      continue;
    }
    if (arg === '--help' || arg === '-h') {
      printHelp();
      process.exit(0);
    }
    throw new Error(`Unknown argument: ${arg}`);
  }

  if (!Number.isFinite(options.sampleIntervalMs) || options.sampleIntervalMs < 100) {
    throw new Error(`Invalid --sample-interval-ms: ${options.sampleIntervalMs}`);
  }
  if (!Number.isFinite(options.durationSeconds) || options.durationSeconds < 0) {
    throw new Error(`Invalid --duration-seconds: ${options.durationSeconds}`);
  }

  return options;
}

function printHelp() {
  console.log(`Usage:
  node tools/browser/scripts/chatgpt_cdp_observer.mjs [options]

Options:
  --user-data-dir <path>      Chrome user data dir. Default: ${DEFAULT_USER_DATA_DIR}
  --output-dir <path>         Directory for jsonl logs. Default: ${DEFAULT_OUTPUT_DIR}
  --url-substring <text>      Target page URL filter. Default: ${DEFAULT_URL_SUBSTRING}
  --sample-interval-ms <ms>   In-page UI sampling interval. Default: ${DEFAULT_SAMPLE_INTERVAL_MS}
  --duration-seconds <sec>    Auto-stop after N seconds. Default: 0 (run until Ctrl+C)
`);
}

function nowIso() {
  return new Date().toISOString();
}

function slugTimestamp() {
  return nowIso().replace(/[:.]/g, '-');
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
    path: lines[1],
    wsUrl: `ws://127.0.0.1:${port}${lines[1]}`,
    portFile
  };
}

function clipText(value, maxLength = 600) {
  const text = String(value || '').replace(/\s+/g, ' ').trim();
  if (text.length <= maxLength) {
    return text;
  }
  return `${text.slice(0, Math.max(0, maxLength - 1))}…`;
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

function summarizeTextBody(text) {
  const source = String(text || '');
  const summary = {
    snippet: clipText(source, 1600),
    bodyLength: source.length,
    conversationAsyncStatuses: Array.from(
      source.matchAll(/"conversation_async_status"\s*:\s*(\d+)/g)
    )
      .map((match) => Number(match[1]))
      .filter((value) => Number.isFinite(value)),
    imageGenTaskIds: Array.from(
      source.matchAll(/"image_gen_task_id"\s*:\s*"([^"]+)"/g)
    ).map((match) => match[1]),
    toolNames: Array.from(
      source.matchAll(/"tool_name"\s*:\s*"([^"]+)"/g)
    ).map((match) => match[1]),
    uiCardShimmer: /"ui_card_shimmer"\s*:\s*true/i.test(source)
      ? true
      : /"ui_card_shimmer"\s*:\s*false/i.test(source)
        ? false
        : null,
    replaceStreamStatus: /"replace_stream_status"\s*:\s*true/i.test(source)
      ? true
      : /"replace_stream_status"\s*:\s*false/i.test(source)
        ? false
        : null,
    streamingAsyncStatus: /"streaming_async_status"\s*:\s*true/i.test(source)
      ? true
      : /"streaming_async_status"\s*:\s*false/i.test(source)
        ? false
        : null,
    errorText: null
  };
  const errorPatterns = [
    /生成图片时出现错误/i,
    /无法生成图片/i,
    /无法创建图片/i,
    /we experienced an error when generating images/i,
    /there was an error generating (the )?image/i,
    /failed to generate (the )?image/i
  ];
  for (const pattern of errorPatterns) {
    const match = source.match(pattern);
    if (match?.[0]) {
      summary.errorText = match[0];
      break;
    }
  }
  return summary;
}

function ensureDirectory(dirPath) {
  fs.mkdirSync(dirPath, { recursive: true });
}

function createLogWriter(outputDir) {
  ensureDirectory(outputDir);
  const sessionId = `obs-${slugTimestamp()}`;
  const logPath = path.join(outputDir, `${sessionId}.jsonl`);
  const stream = fs.createWriteStream(logPath, { flags: 'a' });
  return {
    sessionId,
    logPath,
    write(record) {
      stream.write(`${JSON.stringify(record, null, 0)}\n`);
    },
    close() {
      return new Promise((resolve) => {
        stream.end(resolve);
      });
    }
  };
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

function buildObserverScript(bindingName, sampleIntervalMs) {
  return `
(() => {
  const BINDING = ${JSON.stringify(bindingName)};
  const SAMPLE_INTERVAL_MS = ${JSON.stringify(sampleIntervalMs)};
  if (globalThis.__codexObserverInstalled) {
    return { installed: true, reused: true };
  }
  globalThis.__codexObserverInstalled = true;

  const KEYWORDS = [
    '正在创建图片', '最后微调一下', '生成图片中', '创建图片中',
    'creating image', 'generating image', 'final touches', 'polishing',
    '你更喜欢哪张图片', 'which image do you prefer',
    '跳过', 'skip', 'download', '下载', 'save', '保存',
    'stop', '停止', 'menu', '更多'
  ];

  const emit = (event, data = {}) => {
    try {
      if (typeof globalThis[BINDING] !== 'function') {
        return;
      }
      globalThis[BINDING](JSON.stringify({
        ts: new Date().toISOString(),
        source: 'page',
        event,
        ...data
      }));
    } catch {}
  };

  const clip = (value, max = 300) => {
    const text = String(value || '').replace(/\\s+/g, ' ').trim();
    if (text.length <= max) {
      return text;
    }
    return text.slice(0, Math.max(0, max - 1)) + '…';
  };

  const signalText = (element) => {
    return [
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
  };

  const isVisible = (element) => {
    if (!element || !(element instanceof Element)) return false;
    const rect = element.getBoundingClientRect();
    const style = getComputedStyle(element);
    return rect.width > 0 && rect.height > 0 && style.display !== 'none' && style.visibility !== 'hidden';
  };

  const pathOf = (element) => {
    if (!element || !(element instanceof Element)) {
      return null;
    }
    const parts = [];
    let node = element;
    let depth = 0;
    while (node && depth < 5) {
      let part = node.tagName.toLowerCase();
      if (node.id) part += '#' + node.id;
      const cls = Array.from(node.classList || []).slice(0, 2);
      if (cls.length) part += '.' + cls.join('.');
      parts.unshift(part);
      node = node.parentElement;
      depth += 1;
    }
    return parts.join(' > ');
  };

  const looksInteresting = (text) => {
    const value = String(text || '').toLowerCase();
    return KEYWORDS.some((keyword) => value.includes(String(keyword).toLowerCase()));
  };

  const collectUiState = () => {
    const main = document.querySelector('main');
    const header = document.querySelector('header, [role="banner"]');
    const buttons = Array.from(document.querySelectorAll('button, [role="button"], a'));
    const visibleButtons = buttons.filter((element) => isVisible(element));
    const progressNodes = Array.from((main || document).querySelectorAll('*'))
      .filter((element) => isVisible(element))
      .map((element) => signalText(element))
      .filter((text) => text && text.length <= 120 && looksInteresting(text));

    const candidatePrompt = progressNodes.find((text) => /你更喜欢哪张图片|which image do you prefer/i.test(text)) || null;
    const stopVisible = visibleButtons.some((element) => /stop|停止/i.test(signalText(element)));
    const saveVisible = visibleButtons.some((element) => /save|保存/i.test(signalText(element)));
    const downloadVisible = visibleButtons.some((element) => /download|下载/i.test(signalText(element)));
    const menuVisible = visibleButtons.some((element) => /more|更多|menu|菜单|actions|操作/i.test(signalText(element)));
    const busyCount = document.querySelectorAll('[aria-busy="true"]').length;
    const headerAnimations = typeof header?.getAnimations === 'function' ? header.getAnimations().length : null;
    const documentAnimations = typeof document.getAnimations === 'function' ? document.getAnimations().length : null;
    const imageCount = document.querySelectorAll('img').length;
    return {
      url: location.href,
      title: document.title,
      stopVisible,
      saveVisible,
      downloadVisible,
      menuVisible,
      busyCount,
      candidatePrompt,
      progressTexts: progressNodes.slice(0, 8),
      headerAnimations,
      documentAnimations,
      imageCount
    };
  };

  let lastSignature = '';
  const emitUiSample = (reason) => {
    const state = collectUiState();
    const signature = JSON.stringify(state);
    if (signature === lastSignature && reason !== 'observer_ready') {
      return;
    }
    lastSignature = signature;
    emit('ui_sample', { reason, state });
  };

  const observer = new MutationObserver((mutations) => {
    let sawInteresting = false;
    for (const mutation of mutations) {
      const target = mutation.target instanceof Element ? mutation.target : mutation.target?.parentElement;
      const targetText = clip(signalText(target), 240);
      const attributeName = mutation.attributeName || null;
      const added = Array.from(mutation.addedNodes || [])
        .filter((node) => node instanceof Element || node instanceof Text)
        .map((node) => clip(node.textContent || '', 200))
        .filter(Boolean);
      const removed = Array.from(mutation.removedNodes || [])
        .filter((node) => node instanceof Element || node instanceof Text)
        .map((node) => clip(node.textContent || '', 200))
        .filter(Boolean);
      const interesting = looksInteresting(targetText) ||
        added.some(looksInteresting) ||
        removed.some(looksInteresting) ||
        attributeName === 'aria-busy' ||
        attributeName === 'class' ||
        attributeName === 'disabled' ||
        attributeName === 'hidden';
      if (!interesting) {
        continue;
      }
      sawInteresting = true;
      emit('dom_mutation', {
        mutationType: mutation.type,
        attributeName,
        targetPath: pathOf(target),
        targetText,
        added,
        removed,
        classList: target ? Array.from(target.classList || []).slice(0, 8) : [],
        attrs: target ? {
          ariaBusy: target.getAttribute?.('aria-busy') || null,
          ariaLabel: clip(target.getAttribute?.('aria-label') || '', 120),
          dataTestId: target.getAttribute?.('data-testid') || null,
          role: target.getAttribute?.('role') || null
        } : {}
      });
    }
    if (sawInteresting) {
      emitUiSample('mutation');
    }
  });

  observer.observe(document.documentElement, {
    subtree: true,
    childList: true,
    characterData: true,
    attributes: true,
    attributeFilter: ['class', 'style', 'hidden', 'disabled', 'aria-busy', 'aria-label', 'data-testid', 'title']
  });

  const animationEvent = (event) => {
    const target = event.target instanceof Element ? event.target : null;
    emit('animation_event', {
      type: event.type,
      animationName: event.animationName || null,
      elapsedTime: Number(event.elapsedTime || 0),
      targetPath: pathOf(target),
      targetText: clip(signalText(target), 200)
    });
    emitUiSample('animation_event');
  };

  document.addEventListener('animationstart', animationEvent, true);
  document.addEventListener('animationend', animationEvent, true);
  document.addEventListener('transitionstart', animationEvent, true);
  document.addEventListener('transitionend', animationEvent, true);

  globalThis.__codexObserverInterval = globalThis.setInterval(() => {
    emitUiSample('interval');
  }, SAMPLE_INTERVAL_MS);

  emit('observer_ready', {
    href: location.href,
    title: document.title
  });
  emitUiSample('observer_ready');
  return { installed: true, reused: false };
})();
`;
}

async function main() {
  const options = parseArgs(process.argv);
  const devTools = readDevToolsActivePort(options.userDataDir);
  const log = createLogWriter(options.outputDir);
  const connection = new CdpConnection(devTools.wsUrl);
  const requestState = new Map();
  let activeSessionId = null;
  let shuttingDown = false;

  const writeRecord = (record) => {
    log.write({
      ts: nowIso(),
      session_id: log.sessionId,
      ...record
    });
  };

  writeRecord({
    source: 'observer',
    event: 'startup',
    user_data_dir: options.userDataDir,
    devtools_port_file: devTools.portFile,
    ws_url: devTools.wsUrl,
    output_path: log.logPath
  });

  connection.onEvent(async (message) => {
    try {
      const { method, params, sessionId } = message;
      if (!method) {
        return;
      }

      if (method === 'Runtime.bindingCalled' && sessionId === activeSessionId) {
        const payload = JSON.parse(params.payload || '{}');
        writeRecord(payload);
        return;
      }

      if (sessionId !== activeSessionId) {
        return;
      }

      if (method === 'Page.frameNavigated') {
        writeRecord({
          source: 'cdp_page',
          event: 'frame_navigated',
          frame: {
            id: params.frame?.id || null,
            url: params.frame?.url || null,
            mimeType: params.frame?.mimeType || null
          }
        });
        return;
      }

      if (method === 'Page.lifecycleEvent') {
        writeRecord({
          source: 'cdp_page',
          event: 'lifecycle',
          name: params.name || null,
          frameId: params.frameId || null,
          loaderId: params.loaderId || null
        });
        return;
      }

      if (method === 'Network.requestWillBeSent') {
        const label = networkLabel(params.request?.url);
        if (!label) {
          return;
        }
        const item = {
          label,
          url: params.request.url,
          method: params.request.method,
          requestId: params.requestId,
          resourceType: params.type || null,
          fileId: extractFileId(params.request.url),
          startedAt: nowIso()
        };
        requestState.set(params.requestId, item);
        writeRecord({
          source: 'cdp_network',
          event: 'request',
          ...item,
          initiator: params.initiator?.type || null
        });
        return;
      }

      if (method === 'Network.responseReceived') {
        const item = requestState.get(params.requestId);
        const label = item?.label || networkLabel(params.response?.url);
        if (!label) {
          return;
        }
        const merged = {
          ...(item || {}),
          label,
          url: params.response.url,
          requestId: params.requestId,
          status: params.response.status,
          statusText: params.response.statusText,
          mimeType: params.response.mimeType,
          protocol: params.response.protocol,
          resourceType: params.type || item?.resourceType || null,
          fileId: extractFileId(params.response.url) || item?.fileId || null
        };
        requestState.set(params.requestId, merged);
        writeRecord({
          source: 'cdp_network',
          event: 'response',
          ...merged
        });
        return;
      }

      if (method === 'Network.loadingFailed') {
        const item = requestState.get(params.requestId);
        if (!item) {
          return;
        }
        writeRecord({
          source: 'cdp_network',
          event: 'loading_failed',
          requestId: params.requestId,
          label: item.label,
          url: item.url,
          errorText: params.errorText || null,
          canceled: Boolean(params.canceled)
        });
        return;
      }

      if (method === 'Network.loadingFinished') {
        const item = requestState.get(params.requestId);
        if (!item) {
          return;
        }
        writeRecord({
          source: 'cdp_network',
          event: 'loading_finished',
          requestId: params.requestId,
          label: item.label,
          url: item.url,
          encodedDataLength: params.encodedDataLength || 0,
          fileId: item.fileId || null
        });
        if (!['conversation', 'conversation_prepare', 'async_status'].includes(item.label)) {
          return;
        }
        try {
          const bodyResult = await connection.send(
            'Network.getResponseBody',
            { requestId: params.requestId },
            activeSessionId
          );
          const bodyText = bodyResult.base64Encoded
            ? Buffer.from(bodyResult.body || '', 'base64').toString('utf8')
            : String(bodyResult.body || '');
          writeRecord({
            source: 'network_body',
            event: 'body_summary',
            requestId: params.requestId,
            label: item.label,
            url: item.url,
            fileId: item.fileId || null,
            summary: summarizeTextBody(bodyText)
          });
        } catch (error) {
          writeRecord({
            source: 'network_body',
            event: 'body_summary_error',
            requestId: params.requestId,
            label: item.label,
            url: item.url,
            error: error instanceof Error ? error.message : String(error)
          });
        }
      }
    } catch (error) {
      writeRecord({
        source: 'observer',
        event: 'listener_error',
        error: error instanceof Error ? error.message : String(error)
      });
    }
  });

  const shutdown = async (reason) => {
    if (shuttingDown) {
      return;
    }
    shuttingDown = true;
    writeRecord({
      source: 'observer',
      event: 'shutdown',
      reason
    });
    try {
      if (activeSessionId) {
        await connection.send('Target.detachFromTarget', { sessionId: activeSessionId });
      }
    } catch {}
    try {
      await connection.close();
    } catch {}
    await log.close();
    console.log(`Observer stopped. Log written to: ${log.logPath}`);
  };

  process.on('SIGINT', async () => {
    await shutdown('sigint');
    process.exit(0);
  });
  process.on('SIGTERM', async () => {
    await shutdown('sigterm');
    process.exit(0);
  });

  await connection.connect();

  const targets = await connection.send('Target.getTargets');
  const candidates = (targets.targetInfos || []).filter((target) => (
    target.type === 'page' &&
    String(target.url || '').includes(options.urlSubstring)
  ));

  if (!candidates.length) {
    throw new Error(`No page target matched url substring: ${options.urlSubstring}`);
  }

  const target = candidates.at(-1);
  writeRecord({
    source: 'observer',
    event: 'target_selected',
    target: {
      targetId: target.targetId,
      title: target.title,
      url: target.url
    },
    candidate_count: candidates.length
  });

  const attachResult = await connection.send('Target.attachToTarget', {
    targetId: target.targetId,
    flatten: true
  });
  activeSessionId = attachResult.sessionId;

  await connection.send('Page.enable', {}, activeSessionId);
  await connection.send('Page.setLifecycleEventsEnabled', { enabled: true }, activeSessionId);
  await connection.send('Runtime.enable', {}, activeSessionId);
  await connection.send('Network.enable', {
    maxTotalBufferSize: 100 * 1024 * 1024,
    maxResourceBufferSize: 10 * 1024 * 1024
  }, activeSessionId);
  await connection.send('Runtime.addBinding', {
    name: BINDING_NAME
  }, activeSessionId);

  const observerSource = buildObserverScript(BINDING_NAME, options.sampleIntervalMs);
  await connection.send('Page.addScriptToEvaluateOnNewDocument', {
    source: observerSource
  }, activeSessionId);
  await connection.send('Runtime.evaluate', {
    expression: observerSource,
    awaitPromise: true,
    returnByValue: true
  }, activeSessionId);

  console.log(`Observer ready.`);
  console.log(`Log file: ${log.logPath}`);
  console.log(`Target: ${target.title} <${target.url}>`);
  console.log(`Now generate an image manually in ChatGPT. Press Ctrl+C when you want to stop observing.`);

  if (options.durationSeconds > 0) {
    await delay(options.durationSeconds * 1000);
    await shutdown('duration_elapsed');
    return;
  }

  // Keep process alive until signal.
  await new Promise(() => {});
}

main().catch(async (error) => {
  console.error(`ERROR: ${error instanceof Error ? error.message : String(error)}`);
  process.exit(1);
});
