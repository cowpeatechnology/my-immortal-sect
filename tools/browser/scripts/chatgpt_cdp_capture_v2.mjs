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
  'chatgpt-capture-v2'
);
const DEFAULT_URL_SUBSTRING = 'chatgpt.com/c/';

const NETWORK_LABELS = [
  { label: 'conversation', pattern: /\/backend-api\/f\/conversation(?:\?|$)/i },
  { label: 'async_status', pattern: /\/backend-api\/conversation\/[^/]+\/async-status(?:\?|$)/i },
  { label: 'file_download', pattern: /\/backend-api\/files\/download\/[^/?#]+/i },
  { label: 'estuary_content', pattern: /\/backend-api\/estuary\/content(?:\?|$)/i }
];

function parseArgs(argv) {
  const options = {
    userDataDir: DEFAULT_USER_DATA_DIR,
    outputDir: DEFAULT_OUTPUT_DIR,
    urlSubstring: DEFAULT_URL_SUBSTRING,
    basename: '',
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
    if (arg === '--basename' && next) {
      options.basename = next;
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

  if (!Number.isFinite(options.durationSeconds) || options.durationSeconds < 0) {
    throw new Error(`Invalid --duration-seconds: ${options.durationSeconds}`);
  }

  return options;
}

function printHelp() {
  console.log(`Usage:
  node tools/browser/scripts/chatgpt_cdp_capture_v2.mjs [options]

Options:
  --user-data-dir <path>      Chrome user data dir. Default: ${DEFAULT_USER_DATA_DIR}
  --output-dir <path>         Directory for captured images. Default: ${DEFAULT_OUTPUT_DIR}
  --url-substring <text>      Target page URL filter. Default: ${DEFAULT_URL_SUBSTRING}
  --basename <name>           Output filename prefix. Default: final-capture
  --duration-seconds <sec>    Auto-stop after N seconds. Default: 0 (wait until save or Ctrl+C)
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

function safeStem(value) {
  const stem = String(value || '').trim().replace(/[^a-zA-Z0-9._-]+/g, '-');
  return stem || 'final-capture';
}

function outputPathFor(options, candidate) {
  const requestedStem = safeStem(options.basename || 'final-capture');
  const originalName = String(candidate.fileName || '').split('/').pop() || '';
  const originalExt = path.extname(originalName) || extFromMimeType(candidate.mimeType);
  const baseName = options.basename
    ? `${requestedStem}${originalExt}`
    : `${requestedStem}-${slugTimestamp()}${originalExt}`;
  return path.join(options.outputDir, baseName);
}

class JsonlLog {
  constructor(outputDir) {
    ensureDirectory(outputDir);
    this.sessionId = `capture-v2-${slugTimestamp()}`;
    this.logPath = path.join(outputDir, `${this.sessionId}.jsonl`);
    this.stream = fs.createWriteStream(this.logPath, { flags: 'a' });
  }

  write(record) {
    this.stream.write(`${JSON.stringify({ ts: nowIso(), ...record })}\n`);
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

function createRun(runIndex, requestId, conversationId) {
  return {
    runIndex,
    requestId,
    conversationId,
    startedAt: nowIso(),
    asyncEvents: [],
    fileDownloads: [],
    estuaryContents: [],
    finalSaved: false,
    waitingForFinalAfterAsync: false
  };
}

function latestValidatedFinal(run, requireLoadedBeforeAsync = false) {
  const finalDownloadsByFileId = new Map();
  for (const item of run.fileDownloads) {
    if (!item.fileId || item.isPartial || item.status !== 'success') {
      continue;
    }
    finalDownloadsByFileId.set(item.fileId, item);
  }

  let best = null;
  for (const item of run.estuaryContents) {
    if (!item.fileId || item.status !== 200 || !item.bodyBuffer) {
      continue;
    }
    const matchedDownload = finalDownloadsByFileId.get(item.fileId);
    if (!matchedDownload) {
      continue;
    }
    if (requireLoadedBeforeAsync && run.asyncEvents.length > 0) {
      const lastAsync = run.asyncEvents.at(-1);
      if ((item.sequence || 0) > (lastAsync.sequence || 0)) {
        continue;
      }
    }
    if (!best || (item.sequence || 0) > (best.sequence || 0)) {
      best = {
        ...item,
        fileName: matchedDownload.fileName,
        downloadUrl: matchedDownload.downloadUrl
      };
    }
  }
  return best;
}

async function main() {
  const options = parseArgs(process.argv);
  ensureDirectory(options.outputDir);

  const log = new JsonlLog(options.outputDir);
  const devTools = readDevToolsActivePort(options.userDataDir);
  const connection = new CdpConnection(devTools.wsUrl);
  const requestState = new Map();

  let activeSessionId = null;
  let activeRun = null;
  let runIndex = 0;
  let eventSequence = 0;
  let shuttingDown = false;
  let finishMain = null;
  const finished = new Promise((resolve) => {
    finishMain = resolve;
  });

  const writeRecord = (record) => {
    log.write({
      session_id: log.sessionId,
      ...record
    });
  };

  const finalizeIfReady = async (reason) => {
    if (!activeRun || activeRun.finalSaved) {
      return false;
    }

    const candidate = latestValidatedFinal(activeRun, false);
    if (!candidate) {
      writeRecord({
        source: 'capture_v2',
        event: 'finalize_pending',
        reason,
        runIndex: activeRun.runIndex,
        asyncCount: activeRun.asyncEvents.length,
        fileDownloadCount: activeRun.fileDownloads.length,
        estuaryCount: activeRun.estuaryContents.length
      });
      return false;
    }

    const outputPath = outputPathFor(options, candidate);
    fs.writeFileSync(outputPath, candidate.bodyBuffer);
    activeRun.finalSaved = true;
    activeRun.outputPath = outputPath;

    writeRecord({
      source: 'capture_v2',
      event: 'final_saved',
      reason,
      runIndex: activeRun.runIndex,
      outputPath,
      fileId: candidate.fileId,
      fileName: candidate.fileName,
      mimeType: candidate.mimeType,
      sizeBytes: candidate.bodyBuffer.length
    });

    console.log(`Saved final image: ${outputPath}`);
    console.log(`Matched file: ${candidate.fileName}`);
    await shutdown('saved_final_image');
    return true;
  };

  const shutdown = async (reason) => {
    if (shuttingDown) {
      return;
    }
    shuttingDown = true;

    writeRecord({
      source: 'capture_v2',
      event: 'shutdown',
      reason,
      activeRun: activeRun ? {
        runIndex: activeRun.runIndex,
        finalSaved: activeRun.finalSaved,
        asyncCount: activeRun.asyncEvents.length,
        fileDownloadCount: activeRun.fileDownloads.length,
        estuaryCount: activeRun.estuaryContents.length,
        outputPath: activeRun.outputPath || null
      } : null
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
    if (finishMain) {
      finishMain();
    }
  };

  process.on('SIGINT', async () => {
    await shutdown('sigint');
    process.exit(0);
  });
  process.on('SIGTERM', async () => {
    await shutdown('sigterm');
    process.exit(0);
  });

  writeRecord({
    source: 'capture_v2',
    event: 'startup',
    userDataDir: options.userDataDir,
    outputDir: options.outputDir,
    urlSubstring: options.urlSubstring,
    basename: options.basename || null,
    wsUrl: devTools.wsUrl
  });

  connection.onEvent(async (message) => {
    try {
      const { method, params, sessionId } = message;
      if (!method || sessionId !== activeSessionId) {
        return;
      }

      if (method === 'Network.requestWillBeSent') {
        const label = networkLabel(params.request?.url);
        if (!label) {
          return;
        }
        const item = {
          sequence: ++eventSequence,
          label,
          url: params.request.url,
          method: params.request.method,
          requestId: params.requestId,
          fileId: extractFileId(params.request.url),
          startedAt: nowIso()
        };

        if (label === 'conversation') {
          runIndex += 1;
          activeRun = createRun(
            runIndex,
            params.requestId,
            params.request.url.match(/conversation\/([^/?#]+)/i)?.[1] || null
          );
          console.log(`Detected generation run #${activeRun.runIndex}. Waiting for final image...`);
          writeRecord({
            source: 'capture_v2',
            event: 'run_started',
            runIndex: activeRun.runIndex,
            requestId: params.requestId,
            url: params.request.url
          });
        }

        if (activeRun) {
          item.runIndex = activeRun.runIndex;
        }
        requestState.set(params.requestId, item);

        writeRecord({
          source: 'network',
          event: 'request',
          ...item
        });
        return;
      }

      if (method === 'Network.responseReceived') {
        const item = requestState.get(params.requestId);
        if (!item) {
          return;
        }
        item.sequence = item.sequence || ++eventSequence;
        item.status = params.response.status;
        item.statusText = params.response.statusText;
        item.mimeType = params.response.mimeType;
        requestState.set(params.requestId, item);

        writeRecord({
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

      if (method === 'Network.loadingFinished') {
        const item = requestState.get(params.requestId);
        if (!item || !activeRun || item.runIndex !== activeRun.runIndex) {
          return;
        }

        item.loadingFinishedAt = nowIso();
        item.sequence = ++eventSequence;
        requestState.set(params.requestId, item);

        if (item.label === 'file_download') {
          const bodyResult = await connection.send(
            'Network.getResponseBody',
            { requestId: params.requestId },
            activeSessionId
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
          activeRun.fileDownloads.push(record);
          writeRecord({
            source: 'capture_v2',
            event: 'file_download_ready',
            runIndex: activeRun.runIndex,
            ...record
          });
          return;
        }

        if (item.label === 'estuary_content') {
          const bodyResult = await connection.send(
            'Network.getResponseBody',
            { requestId: params.requestId },
            activeSessionId
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
          activeRun.estuaryContents.push(record);
          writeRecord({
            source: 'capture_v2',
            event: 'estuary_content_ready',
            runIndex: activeRun.runIndex,
            requestId: record.requestId,
            fileId: record.fileId,
            mimeType: record.mimeType,
            encodedDataLength: record.encodedDataLength,
            status: record.status
          });
          if (activeRun.waitingForFinalAfterAsync) {
            await finalizeIfReady('estuary_after_async');
          }
          return;
        }

        if (item.label === 'async_status') {
          const bodyResult = await connection.send(
            'Network.getResponseBody',
            { requestId: params.requestId },
            activeSessionId
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
          activeRun.asyncEvents.push(record);
          writeRecord({
            source: 'capture_v2',
            event: 'async_status_seen',
            runIndex: activeRun.runIndex,
            requestId: params.requestId,
            body: parsed
          });

          if (parsed.status === 'OK') {
            activeRun.waitingForFinalAfterAsync = true;
            const saved = await finalizeIfReady('async_status_ok');
            if (!saved) {
              console.log('Observed async-status OK, waiting for a validated final image...');
            }
          }
          return;
        }
      }
    } catch (error) {
      writeRecord({
        source: 'capture_v2',
        event: 'listener_error',
        error: error instanceof Error ? error.message : String(error)
      });
    }
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
  const attachResult = await connection.send('Target.attachToTarget', {
    targetId: target.targetId,
    flatten: true
  });
  activeSessionId = attachResult.sessionId;

  await connection.send('Network.enable', {
    maxTotalBufferSize: 100 * 1024 * 1024,
    maxResourceBufferSize: 20 * 1024 * 1024
  }, activeSessionId);
  await connection.send('Page.enable', {}, activeSessionId);

  writeRecord({
    source: 'capture_v2',
    event: 'target_selected',
    targetId: target.targetId,
    title: target.title,
    url: target.url
  });

  console.log('V2 capture watcher ready.');
  console.log(`Target: ${target.title} <${target.url}>`);
  console.log(`Output dir: ${options.outputDir}`);
  console.log('Now generate an image manually in ChatGPT. The watcher will save the validated final image and exit.');

  if (options.durationSeconds > 0) {
    await delay(options.durationSeconds * 1000);
    await shutdown('duration_elapsed');
    return;
  }

  await finished;
}

main()
  .then(() => {
    process.exit(0);
  })
  .catch(async (error) => {
    console.error(`ERROR: ${error instanceof Error ? error.message : String(error)}`);
    process.exit(1);
  });
