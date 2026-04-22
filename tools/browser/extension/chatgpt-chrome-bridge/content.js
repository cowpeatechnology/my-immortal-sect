const DEFAULT_SERVER_BASE = "http://127.0.0.1:8765";
const DEFAULT_CLIENT_ID = "chrome-extension-chatgpt-bridge";
const POLL_INTERVAL_MS = 2000;
const DOWNLOAD_KEYWORDS = ["download", "下载"];
const SAVE_KEYWORDS = ["save", "保存"];
const CLOSE_KEYWORDS = ["close", "关闭"];
const STOP_KEYWORDS = ["stop", "停止", "停止流式传输"];
const IMAGE_PROGRESS_KEYWORDS = [
  "正在创建图片",
  "最后微调一下",
  "生成图片中",
  "创建图片中",
  "creating image",
  "generating image",
  "final touches",
  "polishing"
];
const IMAGE_CARD_KEYWORDS = [
  "图片已创建",
  "已生成图片",
  "image created",
  "generated image"
];
const IMAGE_CHOICE_PROMPT_KEYWORDS = [
  "你更喜欢哪张图片",
  "which image do you prefer"
];
const IMAGE_CHOICE_SKIP_KEYWORDS = ["跳过", "skip"];
const IMAGE_CHOICE_PREFERENCE_KEYWORDS = [
  "图片 1 更佳",
  "图片 2 更佳",
  "image 1 is better",
  "image 2 is better"
];
const IMAGE_ERROR_PATTERNS = [
  /生成图片时出现错误/i,
  /无法生成图片/i,
  /无法创建图片/i,
  /we experienced an error when generating images/i,
  /there was an error generating (the )?image/i,
  /failed to generate (the )?image/i,
  /could(?: not|n't) generate (the )?image/i
];
const MENU_KEYWORDS = ["more", "更多", "options", "操作", "菜单", "actions"];
const ACTION_SCOPE_SELECTORS = [
  "[data-message-author-role='assistant']",
  "article",
  "[role='article']",
  "[data-testid*='conversation-turn']",
  "[data-testid*='message']",
  "main li",
  "main section"
];
const TRACEABLE_NETWORK_PATTERNS = [
  {
    label: "conversation",
    pattern: /\/backend-api\/f\/conversation(?:\?|$)/i
  },
  {
    label: "conversation_prepare",
    pattern: /\/backend-api\/f\/conversation\/prepare(?:\?|$)/i
  },
  {
    label: "async_status",
    pattern: /\/backend-api\/conversation\/[^/]+\/async-status(?:\?|$)/i
  },
  {
    label: "file_download",
    pattern: /\/backend-api\/files\/download\/[^/?#]+/i
  },
  {
    label: "estuary_content",
    pattern: /\/backend-api\/estuary\/content(?:\?|$)/i
  }
];

let pollTimer = null;
let pollInFlight = false;
let serverBase = DEFAULT_SERVER_BASE;
let clientId = DEFAULT_CLIENT_ID;
let workerMode = false;
let activeCommandId = null;
let bridgeNoiseSuppressionDepth = 0;
let networkTraceInstalled = false;
let networkEventCounter = 0;
let lastBusySignature = "";
let lastBusySignatureAt = 0;
let activeNetworkSignals = null;

function delay(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function runtimeMessage(message) {
  return new Promise((resolve, reject) => {
    try {
      chrome.runtime.sendMessage(message, (response) => {
        const lastError = chrome.runtime?.lastError;
        if (lastError) {
          reject(new Error(lastError.message));
          return;
        }
        if (!response) {
          resolve({});
          return;
        }
        if (response.ok === false) {
          reject(new Error(response.error || "Runtime message failed."));
          return;
        }
        resolve(response);
      });
    } catch (error) {
      reject(error);
    }
  });
}

function normalizeText(value) {
  return String(value || "").trim().toLowerCase();
}

function includesKeyword(text, keywords) {
  const haystack = normalizeText(text);
  return keywords.some((keyword) => haystack.includes(normalizeText(keyword)));
}

function elementSignalText(element) {
  return [
    element?.innerText,
    element?.textContent,
    element?.getAttribute?.("aria-label"),
    element?.getAttribute?.("title"),
    element?.getAttribute?.("data-testid"),
    element?.getAttribute?.("aria-description")
  ]
    .map((value) => String(value || ""))
    .join(" ")
    .trim();
}

function resolveHref(href) {
  if (!href) return "";
  try {
    return new URL(href, location.href).href;
  } catch {
    return "";
  }
}

function hrefLooksLikeFinalDownload(href) {
  const value = resolveHref(href);
  if (!/^https?:/i.test(value)) {
    return false;
  }
  if (/estuary\/content/i.test(value)) {
    return false;
  }
  return /download|attachment|backend-api\/files|oaiusercontent/i.test(value);
}

function dedupeElements(elements) {
  const seen = new Set();
  const result = [];
  for (const element of elements) {
    if (!element || seen.has(element)) {
      continue;
    }
    seen.add(element);
    result.push(element);
  }
  return result;
}

function nativeValueSetter(element) {
  if (element instanceof HTMLTextAreaElement) {
    return Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, "value")?.set || null;
  }
  if (element instanceof HTMLInputElement) {
    return Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")?.set || null;
  }
  return null;
}

function isVisible(element) {
  if (!element) return false;
  const rect = element.getBoundingClientRect();
  const style = window.getComputedStyle(element);
  return rect.width > 0 && rect.height > 0 && style.visibility !== "hidden" && style.display !== "none";
}

function findComposer() {
  const selectors = [
    "#prompt-textarea",
    "textarea[data-testid]",
    "textarea[placeholder]",
    "div#prompt-textarea",
    "div[contenteditable='true'][data-lexical-editor='true']",
    "main div[contenteditable='true']"
  ];

  for (const selector of selectors) {
    const candidates = Array.from(document.querySelectorAll(selector));
    const match = candidates.find((element) => isVisible(element));
    if (match) return match;
  }
  return null;
}

function findLoginButton() {
  const candidates = Array.from(document.querySelectorAll("button, a"));
  return candidates.find((element) => {
    const text = (element.innerText || element.textContent || "").trim();
    return isVisible(element) && (text === "登录" || text === "Log in");
  }) || null;
}

function findStopButton() {
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

  const visibleButtons = Array.from(document.querySelectorAll("button, [role='button']"));
  return visibleButtons.find((button) => (
    isVisible(button) && includesKeyword(elementSignalText(button), STOP_KEYWORDS)
  )) || null;
}

function findVisibleGenerationProgressNode() {
  const candidates = Array.from(document.querySelectorAll("main *"));
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
}

function findMainBusyNode() {
  const busyNodes = Array.from(document.querySelectorAll("[aria-busy='true']"));
  return busyNodes.find((element) => isVisible(element) && Boolean(element.closest("main"))) || null;
}

function generationSignals() {
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
}

function pageStatus() {
  const composer = findComposer();
  const loginButton = findLoginButton();
  const signals = generationSignals();
  const candidateChoice = pageCandidateChoiceState();
  return {
    url: location.href,
    title: document.title,
    workerMode,
    composerFound: Boolean(composer),
    loginRequired: Boolean(loginButton),
    loginButtonText: loginButton ? (loginButton.innerText || loginButton.textContent || "").trim() : null,
    imageCount: collectLargeImages().length,
    candidateChoiceVisible: candidateChoice.candidateChoiceVisible,
    ...signals
  };
}

async function waitForStablePageBaseline(timeoutMs = 15000) {
  const deadline = Date.now() + timeoutMs;
  let previousSignature = "";
  let stableHits = 0;

  while (Date.now() < deadline) {
    const status = pageStatus();
    const ready = (
      document.readyState === "complete" &&
      status.composerFound &&
      !status.loginRequired &&
      !status.busyGenerating &&
      !status.candidateChoiceVisible
    );
    const signature = JSON.stringify({
      url: status.url,
      title: status.title,
      composerFound: status.composerFound,
      loginRequired: status.loginRequired,
      imageCount: status.imageCount,
      busyGenerating: status.busyGenerating,
      progressText: status.progressText,
      candidateChoiceVisible: status.candidateChoiceVisible
    });

    if (ready && signature === previousSignature) {
      stableHits += 1;
    } else {
      previousSignature = signature;
      stableHits = ready ? 1 : 0;
    }

    if (ready && stableHits >= 2) {
      return status;
    }

    await delay(1000);
  }

  return pageStatus();
}

async function withBridgeNoiseSuppressed(callback) {
  bridgeNoiseSuppressionDepth += 1;
  try {
    return await callback();
  } finally {
    bridgeNoiseSuppressionDepth = Math.max(0, bridgeNoiseSuppressionDepth - 1);
  }
}

function traceableNetworkMatch(url) {
  const value = resolveHref(url) || String(url || "");
  if (!value) {
    return null;
  }
  if (value.startsWith(serverBase)) {
    return null;
  }
  for (const candidate of TRACEABLE_NETWORK_PATTERNS) {
    if (candidate.pattern.test(value)) {
      return {
        label: candidate.label,
        url: value
      };
    }
  }
  return null;
}

function clipText(value, maxLength = 240) {
  const text = String(value || "").replace(/\s+/g, " ").trim();
  if (text.length <= maxLength) {
    return text;
  }
  return `${text.slice(0, Math.max(0, maxLength - 1))}…`;
}

function extractFileId(url) {
  const match = String(url || "").match(/\/(file_[^/?#]+)/i);
  return match?.[1] || null;
}

function resetActiveNetworkSignals(commandId) {
  activeNetworkSignals = {
    commandId,
    startedAtMs: Date.now(),
    conversationEvents: [],
    conversationPrepareEvents: [],
    asyncStatusEvents: [],
    fileDownloadEvents: [],
    estuaryContentEvents: [],
    lastConversationErrorText: null,
    completionAnnounced: false
  };
  return activeNetworkSignals;
}

function currentActiveNetworkSignals(commandId = activeCommandId) {
  if (!activeNetworkSignals || !commandId || activeNetworkSignals.commandId !== commandId) {
    return null;
  }
  return activeNetworkSignals;
}

function appendUniqueSignalEvent(events, event, dedupeKey) {
  if (dedupeKey && events.some((item) => item.dedupeKey === dedupeKey)) {
    return false;
  }
  events.push({
    ...event,
    dedupeKey: dedupeKey || null
  });
  return true;
}

function recordNetworkSignal(label, url, details = {}, commandId = activeCommandId) {
  const state = currentActiveNetworkSignals(commandId);
  if (!state) {
    return null;
  }
  const observedAtMs = Date.now();
  const fileId = extractFileId(url);
  const event = {
    observedAtMs,
    url,
    fileId,
    details
  };
  if (label === "conversation") {
    state.conversationEvents.push(event);
    if (details?.responseSummary?.errorText) {
      state.lastConversationErrorText = String(details.responseSummary.errorText);
    }
  } else if (label === "conversation_prepare") {
    state.conversationPrepareEvents.push(event);
  } else if (label === "async_status") {
    state.asyncStatusEvents.push(event);
  } else if (label === "file_download") {
    appendUniqueSignalEvent(state.fileDownloadEvents, event, fileId || url);
  } else if (label === "estuary_content") {
    appendUniqueSignalEvent(state.estuaryContentEvents, event, fileId || url);
  }
  return state;
}

function networkSignalSummary(commandId = activeCommandId) {
  const state = currentActiveNetworkSignals(commandId);
  if (!state) {
    return {
      conversationCount: 0,
      conversationPrepareCount: 0,
      asyncStatusCount: 0,
      fileDownloadCount: 0,
      estuaryContentCount: 0,
      matchedFileIdCount: 0,
      completionReady: false,
      lastFileDownloadAtMs: null,
      lastEstuaryContentAtMs: null,
      lastErrorText: null
    };
  }

  const downloadIds = new Set(state.fileDownloadEvents.map((item) => item.fileId).filter(Boolean));
  const estuaryIds = new Set(state.estuaryContentEvents.map((item) => item.fileId).filter(Boolean));
  const matchedIds = Array.from(downloadIds).filter((fileId) => estuaryIds.has(fileId));
  const completionReady = matchedIds.length > 0 || (
    state.fileDownloadEvents.length > 0 && state.estuaryContentEvents.length > 0
  );
  return {
    conversationCount: state.conversationEvents.length,
    conversationPrepareCount: state.conversationPrepareEvents.length,
    asyncStatusCount: state.asyncStatusEvents.length,
    fileDownloadCount: state.fileDownloadEvents.length,
    estuaryContentCount: state.estuaryContentEvents.length,
    matchedFileIdCount: matchedIds.length,
    matchedFileIds: matchedIds,
    completionReady,
    lastFileDownloadAtMs: state.fileDownloadEvents.at(-1)?.observedAtMs || null,
    lastEstuaryContentAtMs: state.estuaryContentEvents.at(-1)?.observedAtMs || null,
    lastErrorText: state.lastConversationErrorText || null
  };
}

function maybeAnnounceNetworkCompletionSignal(commandId = activeCommandId) {
  const state = currentActiveNetworkSignals(commandId);
  if (!state || state.completionAnnounced) {
    return;
  }
  const summary = networkSignalSummary(commandId);
  if (!summary.completionReady) {
    return;
  }
  state.completionAnnounced = true;
  void safePostProgress(commandId, "network_completion_signal", {
    fileDownloadCount: summary.fileDownloadCount,
    estuaryContentCount: summary.estuaryContentCount,
    matchedFileIdCount: summary.matchedFileIdCount
  });
}

function extractConversationSummary(text) {
  const source = String(text || "");
  const asyncStatuses = Array.from(source.matchAll(/"conversation_async_status"\s*:\s*(\d+)/g))
    .map((match) => Number(match[1]))
    .filter((value) => Number.isFinite(value));
  const taskIds = Array.from(source.matchAll(/"image_gen_task_id"\s*:\s*"([^"]+)"/g))
    .map((match) => match[1]);
  const toolNames = Array.from(source.matchAll(/"tool_name"\s*:\s*"([^"]+)"/g))
    .map((match) => match[1]);
  const errors = [
    /We experienced an error when generating images\./i,
    /生成图片时出现错误/i,
    /Something went wrong/i
  ]
    .map((pattern) => source.match(pattern)?.[0] || null)
    .filter(Boolean);
  const uiCardShimmer = /"ui_card_shimmer"\s*:\s*true/i.test(source)
    ? true
    : /"ui_card_shimmer"\s*:\s*false/i.test(source)
      ? false
      : null;
  const replaceStreamStatus = /"replace_stream_status"\s*:\s*true/i.test(source)
    ? true
    : /"replace_stream_status"\s*:\s*false/i.test(source)
      ? false
      : null;
  const streamingAsyncStatus = /"streaming_async_status"\s*:\s*true/i.test(source)
    ? true
    : /"streaming_async_status"\s*:\s*false/i.test(source)
      ? false
      : null;
  return {
    conversationAsyncStatuses: asyncStatuses,
    imageGenTaskCount: taskIds.length,
    imageGenTaskTail: taskIds.at(-1)?.slice(-8) || null,
    toolName: toolNames.at(-1) || null,
    uiCardShimmer,
    replaceStreamStatus,
    streamingAsyncStatus,
    errorText: errors.at(-1) || null
  };
}

async function postLogEvent(kind, details = {}, commandId = activeCommandId) {
  if (!commandId) {
    return;
  }
  await withBridgeNoiseSuppressed(() => fetch(`${serverBase}/api/log`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      kind,
      command_id: commandId,
      client_id: clientId,
      event_timestamp: new Date().toISOString(),
      sequence: ++networkEventCounter,
      details
    })
  }));
}

async function safePostLogEvent(kind, details = {}, commandId = activeCommandId) {
  try {
    await postLogEvent(kind, details, commandId);
  } catch {
    // Logging should never break generation.
  }
}

function watchBusySignals(commandId = activeCommandId) {
  if (!commandId) {
    return;
  }
  const signals = generationSignals();
  const signature = JSON.stringify({
    busyGenerating: signals.busyGenerating,
    stopButtonVisible: signals.stopButtonVisible,
    busyNodeVisible: signals.busyNodeVisible,
    progressText: signals.progressText || null
  });
  if (signature === lastBusySignature) {
    return;
  }
  lastBusySignature = signature;
  lastBusySignatureAt = Date.now();
  void safePostLogEvent("ui_signal", {
    source: "generationSignals",
    ...signals,
    observedAtMs: lastBusySignatureAt
  }, commandId);
}

function installNetworkTrace() {
  if (networkTraceInstalled || typeof window.fetch !== "function") {
    return;
  }
  networkTraceInstalled = true;

  const originalFetch = window.fetch.bind(window);
  window.fetch = async (...args) => {
    const requestInfo = args[0];
    const init = args[1] || {};
    const method = String(
      init?.method ||
      (typeof Request !== "undefined" && requestInfo instanceof Request ? requestInfo.method : "GET") ||
      "GET"
    ).toUpperCase();
    const url = typeof requestInfo === "string"
      ? requestInfo
      : requestInfo?.url || "";
    const match = traceableNetworkMatch(url);
    const suppressed = bridgeNoiseSuppressionDepth > 0;
    const startMs = Date.now();
    let response;
    try {
      response = await originalFetch(...args);
    } catch (error) {
      if (!suppressed && match && activeCommandId) {
        void safePostLogEvent("network", {
          channel: "fetch",
          phase: "error",
          label: match.label,
          method,
          url: match.url,
          durationMs: Date.now() - startMs,
          error: error instanceof Error ? error.message : String(error)
        });
      }
      throw error;
    }

    if (!suppressed && match && activeCommandId) {
      const summary = {
        channel: "fetch",
        phase: "response",
        label: match.label,
        method,
        url: match.url,
        status: response.status,
        ok: response.ok,
        durationMs: Date.now() - startMs,
        redirected: response.redirected,
        contentType: response.headers.get("content-type") || null
      };

      if (match.label === "conversation" || match.label === "async_status") {
        try {
          const bodyText = await response.clone().text();
          if (match.label === "conversation") {
            summary.responseSummary = extractConversationSummary(bodyText);
          } else {
            let parsed = null;
            try {
              parsed = bodyText ? JSON.parse(bodyText) : null;
            } catch {
              parsed = { raw: clipText(bodyText, 200) };
            }
            summary.responseSummary = parsed;
          }
        } catch (error) {
          summary.responseSummary = {
            readError: error instanceof Error ? error.message : String(error)
          };
        }
      }

      recordNetworkSignal(match.label, match.url, summary);
      maybeAnnounceNetworkCompletionSignal();
      void safePostLogEvent("network", summary);
    }

    return response;
  };

  if (typeof PerformanceObserver === "function") {
    try {
      const observer = new PerformanceObserver((list) => {
        if (!activeCommandId || bridgeNoiseSuppressionDepth > 0) {
          return;
        }
        for (const entry of list.getEntries()) {
          const match = traceableNetworkMatch(entry.name);
          if (!match) {
            continue;
          }
          const details = {
            channel: "resource",
            phase: "observed",
            label: match.label,
            url: match.url,
            initiatorType: entry.initiatorType || null,
            startTimeMs: Math.round(entry.startTime || 0),
            durationMs: Math.round(entry.duration || 0),
            transferSize: typeof entry.transferSize === "number" ? entry.transferSize : null,
            encodedBodySize: typeof entry.encodedBodySize === "number" ? entry.encodedBodySize : null
          };
          recordNetworkSignal(match.label, match.url, details);
          maybeAnnounceNetworkCompletionSignal();
          void safePostLogEvent("network", {
            ...details
          });
        }
      });
      observer.observe({ entryTypes: ["resource"] });
    } catch {
      // Resource timing is optional.
    }
  }
}

async function fetchJson(url, init = {}) {
  const response = await withBridgeNoiseSuppressed(() => fetch(url, init));
  const text = await response.text();
  let data = {};
  try {
    data = text ? JSON.parse(text) : {};
  } catch {
    data = { raw: text };
  }
  if (!response.ok) {
    throw new Error(data.error || data.detail || `HTTP ${response.status}`);
  }
  return data;
}

function composerText(element) {
  if (element instanceof HTMLTextAreaElement || element instanceof HTMLInputElement) {
    return element.value || "";
  }
  if (element?.isContentEditable) {
    return element.innerText || element.textContent || "";
  }
  return "";
}

function dispatchInput(element, inputType, data = null) {
  element.dispatchEvent(new InputEvent("input", {
    bubbles: true,
    composed: true,
    inputType,
    data
  }));
}

function replaceContentEditableText(element, prompt) {
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
    inserted = document.execCommand("insertText", false, prompt);
  } catch {
    inserted = false;
  }

  if (!inserted) {
    element.textContent = "";
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

  dispatchInput(element, "insertText", prompt);
}

function setComposerValue(element, prompt) {
  element.focus();

  if (element instanceof HTMLTextAreaElement || element instanceof HTMLInputElement) {
    const setter = nativeValueSetter(element);
    if (setter) {
      setter.call(element, prompt);
    } else {
      element.value = prompt;
    }
    element.dispatchEvent(new Event("input", { bubbles: true }));
    element.dispatchEvent(new Event("change", { bubbles: true }));
    return;
  }

  if (element.isContentEditable) {
    replaceContentEditableText(element, prompt);
    return;
  }

  throw new Error("Unsupported composer element.");
}

function findSendButton() {
  const sendButtonSelectors = [
    "button[data-testid='send-button']",
    "button[aria-label*='发送']",
    "button[aria-label*='Send']"
  ];
  for (const selector of sendButtonSelectors) {
    const button = document.querySelector(selector);
    if (button && isVisible(button)) {
      return button;
    }
  }
  return null;
}

async function submitPrompt(element) {
  const deadline = Date.now() + 5000;

  while (Date.now() < deadline) {
    const button = findSendButton();
    if (button && !button.disabled) {
      button.click();
      return;
    }
    await delay(100);
  }

  for (const type of ["keydown", "keypress", "keyup"]) {
    element.dispatchEvent(new KeyboardEvent(type, {
      key: "Enter",
      code: "Enter",
      which: 13,
      keyCode: 13,
      bubbles: true,
      composed: true
    }));
  }
}

function collectLargeImageEntries() {
  const seen = new Set();
  return Array.from(document.querySelectorAll("img"))
    .map((img) => {
      const rect = img.getBoundingClientRect();
      return {
        element: img,
        src: img.currentSrc || img.src || "",
        width: img.naturalWidth || img.width || 0,
        height: img.naturalHeight || img.height || 0,
        visibleWidth: Math.round(rect.width || 0),
        visibleHeight: Math.round(rect.height || 0),
        alt: img.alt || ""
      };
    })
    .filter((img) => {
      if (!img.src || seen.has(img.src)) return false;
      if (img.src.startsWith("data:image/svg")) return false;
      if ((img.width < 256 && img.visibleWidth < 256) || (img.height < 256 && img.visibleHeight < 256)) return false;
      seen.add(img.src);
      return true;
    });
}

function imageSrcLooksGenerated(src) {
  return /backend-api\/estuary\/content|oaiusercontent|blob:/i.test(String(src || ""));
}

function stripImageEntry(entry) {
  const { element, ...rest } = entry;
  return rest;
}

function collectLargeImages() {
  return collectLargeImageEntries().map(stripImageEntry);
}

function scrollElementIntoView(element, block = "center") {
  if (!element?.scrollIntoView) {
    return;
  }
  try {
    element.scrollIntoView({
      behavior: "instant",
      block,
      inline: "nearest"
    });
  } catch {
    element.scrollIntoView();
  }
}

function scrollConversationToBottom() {
  const composer = findComposer();
  const main = document.querySelector("main");
  const scrollers = dedupeElements([
    composer,
    main,
    main?.parentElement,
    document.scrollingElement,
    document.documentElement,
    document.body
  ]).filter(Boolean);

  for (const element of scrollers) {
    try {
      if (typeof element.scrollTo === "function") {
        element.scrollTo({
          top: element.scrollHeight || Number.MAX_SAFE_INTEGER,
          behavior: "instant"
        });
      } else if ("scrollTop" in element) {
        element.scrollTop = element.scrollHeight || Number.MAX_SAFE_INTEGER;
      }
    } catch {
      // Ignore scroll container differences across ChatGPT layouts.
    }
  }

  scrollElementIntoView(composer, "end");
}

async function ensureBottomContext() {
  scrollConversationToBottom();
  await delay(200);
  scrollConversationToBottom();
}

function conversationSections() {
  return Array.from(document.querySelectorAll("main section"));
}

function isUserConversationSection(section) {
  return Boolean(section?.querySelector?.("[data-message-author-role='user']"));
}

function userConversationSections() {
  return conversationSections().filter(isUserConversationSection);
}

function sectionSignalText(section) {
  return String(section?.innerText || section?.textContent || "").replace(/\s+/g, " ").trim();
}

function isCandidateGeneratedImageLabel(text) {
  const value = normalizeText(text).replace(/\s+/g, " ");
  return /已生成图片\s*[12]\b|generated image\s*[12]\b|image\s*[12]\b/i.test(value);
}

function sectionContainsPrompt(section, prompt) {
  const promptText = normalizeText(prompt).replace(/\s+/g, " ").slice(0, 120);
  if (!promptText) {
    return false;
  }
  return normalizeText(sectionSignalText(section)).includes(promptText.slice(0, 80));
}

async function waitForSubmittedUserTurn(commandId, previousUserCount, prompt, timeoutMs = 10000) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const users = userConversationSections();
    const candidate = users.length > previousUserCount ? users.at(-1) : null;
    if (candidate && sectionContainsPrompt(candidate, prompt)) {
      scrollElementIntoView(candidate, "center");
      await safePostProgress(commandId, "user_turn_detected", {
        userTurnCount: users.length
      });
      return candidate;
    }
    await delay(250);
  }
  throw new Error("Submitted prompt did not appear in the ChatGPT thread.");
}

function responseSectionsForUserTurn(userSection) {
  const sections = [];
  let current = userSection?.nextElementSibling || null;
  while (current) {
    if (current.tagName === "SECTION") {
      if (isUserConversationSection(current)) {
        break;
      }
      sections.push(current);
    }
    current = current.nextElementSibling;
  }
  return sections;
}

function collectGeneratedImagesInSections(sections, beforeSources = new Set()) {
  const images = [];
  for (const section of sections || []) {
    images.push(...normalizeGeneratedImages(section.querySelectorAll("img"), beforeSources));
  }
  return normalizeGeneratedImages(images.map((entry) => entry.element), null);
}

function collectResponseMediaButtons(sections, beforeSources = new Set()) {
  const results = [];
  const seen = new Set();
  for (const section of sections || []) {
    for (const control of Array.from(section.querySelectorAll("button, [role='button'], a[href]"))) {
      if (!control || seen.has(control) || !isVisible(control)) {
        continue;
      }
      const images = normalizeGeneratedImages(control.querySelectorAll("img"), beforeSources);
      const signalText = elementSignalText(control);
      const looksLikeMediaButton = images.length > 0 || includesKeyword(signalText, IMAGE_CARD_KEYWORDS);
      if (!looksLikeMediaButton) {
        continue;
      }
      seen.add(control);
      results.push({
        element: control,
        signalText,
        images,
        candidateVariant: isCandidateGeneratedImageLabel(signalText)
      });
    }
  }
  return results;
}

function collectCandidateChoiceControls(sections) {
  const controls = [];
  const seen = new Set();
  for (const section of sections || []) {
    for (const control of Array.from(section.querySelectorAll("button, [role='button'], a[href]"))) {
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
}

function extractResponseErrorText(sections) {
  for (const section of sections || []) {
    const text = sectionSignalText(section);
    if (!text) {
      continue;
    }
    for (const pattern of IMAGE_ERROR_PATTERNS) {
      const match = text.match(pattern);
      if (match?.[0]) {
        return match[0];
      }
    }
  }
  return null;
}

function pageCandidateChoiceState() {
  const main = document.querySelector("main") || document.body;
  const scopes = main ? [main] : [];
  const candidateChoiceControls = collectCandidateChoiceControls(scopes);
  const skipButton = candidateChoiceControls.find((entry) => entry.isSkip)?.element || null;
  const preferenceButtons = candidateChoiceControls.filter((entry) => entry.isPreference);
  const choicePromptVisible = Array.from((main || document).querySelectorAll(
    "section, article, [role='article'], [data-message-author-role='assistant']"
  ))
    .some((element) => isVisible(element) && includesKeyword(sectionSignalText(element), IMAGE_CHOICE_PROMPT_KEYWORDS));
  const candidateChoiceVisible = (
    choicePromptVisible ||
    Boolean(skipButton) ||
    preferenceButtons.length > 0
  );
  return {
    candidateChoiceVisible,
    choicePromptVisible,
    skipButton,
    preferenceButtons
  };
}

function responseTurnState(userSection, beforeSources = new Set()) {
  const responseSections = responseSectionsForUserTurn(userSection);
  const images = collectGeneratedImagesInSections(responseSections, beforeSources);
  const mediaButtons = collectResponseMediaButtons(responseSections, beforeSources);
  const finalMediaButtons = mediaButtons.filter((entry) => !entry.candidateVariant);
  const candidateChoiceControls = collectCandidateChoiceControls(responseSections);
  const skipButton = candidateChoiceControls.find((entry) => entry.isSkip)?.element || null;
  const preferenceButtons = candidateChoiceControls.filter((entry) => entry.isPreference);
  const choicePromptVisible = responseSections.some((section) => (
    includesKeyword(sectionSignalText(section), IMAGE_CHOICE_PROMPT_KEYWORDS)
  ));
  const candidateChoiceVisible = (
    choicePromptVisible ||
    Boolean(skipButton) ||
    preferenceButtons.length > 0
  );
  const progressVisible = responseSections.some((section) => (
    includesKeyword(sectionSignalText(section), IMAGE_PROGRESS_KEYWORDS)
  ));
  const errorText = extractResponseErrorText(responseSections);
  const signature = JSON.stringify({
    sectionCount: responseSections.length,
    imageSources: images.map((image) => image.src),
    mediaButtons: mediaButtons.map((entry) => entry.signalText.slice(0, 160)),
    finalMediaButtons: finalMediaButtons.map((entry) => entry.signalText.slice(0, 160)),
    candidateChoiceVisible,
    progressVisible,
    errorText
  });
  return {
    userSection,
    responseSections,
    images,
    mediaButtons,
    finalMediaButtons,
    candidateChoiceVisible,
    choicePromptVisible,
    skipButton,
    preferenceButtons,
    progressVisible,
    errorText,
    signature
  };
}

function pickPreferredResponseMediaButton(state) {
  if (!state?.finalMediaButtons?.length) {
    return null;
  }
  const withImages = state.finalMediaButtons.filter((entry) => entry.images.length > 0);
  return withImages.at(-1) || state.finalMediaButtons.at(-1) || null;
}

function collectNewImageEntries(beforeSources) {
  return collectLargeImageEntries().filter((item) => !beforeSources.has(item.src));
}

function normalizeGeneratedImages(elements, beforeSources) {
  const seenSources = new Set();
  return Array.from(elements)
    .map((img) => {
      const rect = img.getBoundingClientRect();
      return {
        element: img,
        src: img.currentSrc || img.src || "",
        width: img.naturalWidth || img.width || 0,
        height: img.naturalHeight || img.height || 0,
        visibleWidth: Math.round(rect.width || 0),
        visibleHeight: Math.round(rect.height || 0),
        alt: img.alt || ""
      };
    })
    .filter((img) => {
      if (!img.src || seenSources.has(img.src)) return false;
      if (!imageSrcLooksGenerated(img.src)) return false;
      if ((img.width < 256 && img.visibleWidth < 256) || (img.height < 256 && img.visibleHeight < 256)) return false;
      if (beforeSources && beforeSources.has(img.src)) return false;
      seenSources.add(img.src);
      return true;
    });
}

function findClickableGeneratedImageTarget(element) {
  let current = element || null;
  let cursorPointerFallback = null;
  let depth = 0;

  while (current && depth < 8) {
    if (current.matches?.("button, [role='button'], a[href]")) {
      return current;
    }
    if (!cursorPointerFallback && current.matches?.(".cursor-pointer")) {
      cursorPointerFallback = current;
    }
    current = current.parentElement;
    depth += 1;
  }

  return cursorPointerFallback;
}

function collectGeneratedImageTargets(beforeSources) {
  const seenTargets = new Set();
  return collectLargeImageEntries()
    .filter((entry) => imageSrcLooksGenerated(entry.src) && !beforeSources.has(entry.src))
    .map((entry) => {
      const target = findClickableGeneratedImageTarget(entry.element);
      if (!target || !isVisible(target) || seenTargets.has(target)) {
        return null;
      }
      const images = normalizeGeneratedImages(target.querySelectorAll("img"), beforeSources);
      if (!images.length) {
        return null;
      }
      seenTargets.add(target);
      return {
        target,
        images,
        newImages: images,
        signalText: elementSignalText(target)
      };
    })
    .filter(Boolean);
}

function pickPreferredGeneratedImageTarget(beforeSources) {
  const entries = collectGeneratedImageTargets(beforeSources);
  const singleImageEntries = entries.filter((entry) => entry.images.length === 1);
  return singleImageEntries.at(-1) || entries.at(-1) || null;
}

async function imageToPayload(src) {
  const blob = await fetchImageBlob(src);
  return blobToPayload(src, blob);
}

async function fetchImageBlob(src) {
  if (src.startsWith("data:")) {
    const response = await withBridgeNoiseSuppressed(() => fetch(src));
    return response.blob();
  }

  const response = await withBridgeNoiseSuppressed(() => fetch(src, { credentials: "include" }));
  return response.blob();
}

async function blobToBase64(blob) {
  const arrayBuffer = await blob.arrayBuffer();
  const bytes = new Uint8Array(arrayBuffer);
  const chunkSize = 0x8000;
  let binary = "";
  for (let index = 0; index < bytes.length; index += chunkSize) {
    binary += String.fromCharCode(...bytes.subarray(index, index + chunkSize));
  }
  return btoa(binary);
}

async function blobToPayload(src, blob) {
  return {
    src,
    mimeType: blob.type || "image/png",
    sizeBytes: blob.size || 0,
    base64: await blobToBase64(blob)
  };
}

async function fetchUrlAsPayload(url) {
  if (!/^https?:/i.test(url)) {
    return imageToPayload(url);
  }
  const response = await runtimeMessage({
    type: "bridge:fetchUrlAsPayload",
    url
  });
  return response.payload;
}

async function armDownloadCapture(commandId) {
  await runtimeMessage({
    type: "bridge:armDownloadCapture",
    commandId
  });
}

async function getDownloadCapture() {
  const response = await runtimeMessage({
    type: "bridge:getDownloadCapture"
  });
  return response.capture || null;
}

async function clearDownloadCapture() {
  try {
    await runtimeMessage({
      type: "bridge:clearDownloadCapture"
    });
  } catch {
    // Ignore cleanup issues.
  }
}

async function postProgress(commandId, stage, details = {}) {
  await withBridgeNoiseSuppressed(() => fetch(`${serverBase}/api/progress`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      command_id: commandId,
      client_id: clientId,
      stage,
      details,
      timestamp: new Date().toISOString()
    })
  }));
}

async function safePostProgress(commandId, stage, details = {}) {
  try {
    await postProgress(commandId, stage, details);
  } catch {
    // Progress reporting should not break image generation.
  }
}

function dispatchHover(element) {
  if (!element) return;
  const rect = element.getBoundingClientRect();
  const clientX = rect.left + Math.max(1, Math.floor(rect.width / 2));
  const clientY = rect.top + Math.max(1, Math.floor(rect.height / 2));
  for (const type of ["pointerenter", "mouseenter", "pointerover", "mouseover", "mousemove"]) {
    element.dispatchEvent(new MouseEvent(type, {
      bubbles: true,
      composed: true,
      clientX,
      clientY
    }));
  }
}

function clickElement(element) {
  if (!element) {
    throw new Error("Missing element to click.");
  }
  dispatchHover(element);
  element.dispatchEvent(new MouseEvent("mousedown", { bubbles: true, composed: true }));
  element.dispatchEvent(new MouseEvent("mouseup", { bubbles: true, composed: true }));
  element.click();
}

function buildActionScopes(imageEntry) {
  if (!imageEntry?.element) {
    return [];
  }

  const scopes = [];
  for (const selector of ACTION_SCOPE_SELECTORS) {
    const match = imageEntry.element.closest(selector);
    if (match) {
      scopes.push(match);
    }
  }

  let current = imageEntry.element.parentElement;
  let depth = 0;
  while (current && depth < 8) {
    scopes.push(current);
    current = current.parentElement;
    depth += 1;
  }

  return dedupeElements(scopes);
}

function revealImageActions(imageEntry, scopes) {
  dispatchHover(imageEntry?.element);
  for (const scope of scopes.slice(0, 4)) {
    dispatchHover(scope);
  }
}

function findDirectDownloadLink(scopes) {
  for (const scope of scopes) {
    const anchors = Array.from(scope.querySelectorAll("a[href]"));
    for (const anchor of anchors) {
      const href = resolveHref(anchor.href);
      if (!href) {
        continue;
      }
      if (anchor.hasAttribute("download")) {
        return anchor;
      }
      if (includesKeyword(elementSignalText(anchor), DOWNLOAD_KEYWORDS)) {
        return anchor;
      }
      if (hrefLooksLikeFinalDownload(href)) {
        return anchor;
      }
    }
  }
  return null;
}

function findVisibleControlByKeywords(scopes, keywords) {
  for (const scope of scopes) {
    const controls = Array.from(scope.querySelectorAll("button, a[href], [role='button'], [role='menuitem']"));
    for (const control of controls) {
      if (!isVisible(control)) {
        continue;
      }
      if (includesKeyword(elementSignalText(control), keywords)) {
        return control;
      }
    }
  }
  return null;
}

function findVisibleDialog() {
  const candidates = Array.from(document.querySelectorAll("[role='dialog'], dialog, [aria-modal='true']"));
  return candidates.reverse().find((element) => isVisible(element)) || null;
}

async function waitForVisibleDialog(timeoutMs) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const dialog = findVisibleDialog();
    if (dialog) {
      return dialog;
    }
    await delay(150);
  }
  return null;
}

async function waitForVisibleControlByKeywords(scopes, keywords, timeoutMs) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const control = findVisibleControlByKeywords(scopes, keywords);
    if (control) {
      return control;
    }
    await delay(150);
  }
  return null;
}

async function closeVisibleDialog() {
  const dialog = findVisibleDialog();
  if (!dialog) {
    return;
  }

  const closeButton = findVisibleControlByKeywords([dialog], CLOSE_KEYWORDS);
  if (closeButton) {
    clickElement(closeButton);
    await delay(250);
    return;
  }

  const escapeEvent = new KeyboardEvent("keydown", {
    key: "Escape",
    code: "Escape",
    keyCode: 27,
    which: 27,
    bubbles: true,
    composed: true
  });
  document.dispatchEvent(escapeEvent);
  await delay(250);
}

async function closeAllVisibleDialogs(maxAttempts = 8) {
  let closedCount = 0;
  for (let attempt = 0; attempt < maxAttempts; attempt += 1) {
    const dialog = findVisibleDialog();
    if (!dialog) {
      break;
    }
    await closeVisibleDialog();
    await delay(300);
    closedCount += 1;
  }
  return closedCount;
}

function findDialogPrimaryImage(dialog) {
  if (!dialog) {
    return null;
  }
  const candidates = collectLargeImageEntries()
    .filter((entry) => dialog.contains(entry.element) && imageSrcLooksGenerated(entry.src));
  return candidates
    .sort((left, right) => (right.visibleWidth * right.visibleHeight) - (left.visibleWidth * left.visibleHeight))[0] || null;
}

function findLikelyMenuTrigger(scopes) {
  for (const scope of scopes) {
    const controls = Array.from(scope.querySelectorAll("button, [role='button'], a[href]"));
    for (const control of controls) {
      if (!isVisible(control)) {
        continue;
      }
      const signals = elementSignalText(control);
      if (control.getAttribute("aria-haspopup") === "menu") {
        return control;
      }
      if (includesKeyword(signals, MENU_KEYWORDS)) {
        return control;
      }
    }
  }
  return null;
}

async function waitForVisibleDownloadMenuItem(timeoutMs, excludeElements = []) {
  const excluded = new Set(excludeElements.filter(Boolean));
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const candidates = Array.from(document.querySelectorAll("button, a[href], [role='menuitem'], [role='button']"));
    const match = candidates.find((element) => {
      if (excluded.has(element) || !isVisible(element)) {
        return false;
      }
      return includesKeyword(elementSignalText(element), DOWNLOAD_KEYWORDS);
    });
    if (match) {
      return match;
    }
    await delay(150);
  }
  return null;
}

async function waitForCapturedDownload(commandId, timeoutMs) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const capture = await getDownloadCapture();
    if (capture?.commandId !== commandId || !capture?.download) {
      await delay(500);
      continue;
    }
    if (capture.download.state === "interrupted") {
      throw new Error("ChatGPT download was interrupted before completion.");
    }
    if (
      capture.download.state === "complete" &&
      capture.download.filename
    ) {
      return capture.download;
    }
    await delay(500);
  }
  throw new Error("Timed out waiting for ChatGPT's final download file.");
}

async function clickAndCaptureDownload(commandId, control, timeoutMs, skipSecondClick = false) {
  await clearDownloadCapture();
  await armDownloadCapture(commandId);
  clickElement(control);

  if (!skipSecondClick) {
    const menuItem = await waitForVisibleDownloadMenuItem(3000, [control]);
    if (menuItem) {
      clickElement(menuItem);
    }
  }

  return waitForCapturedDownload(commandId, timeoutMs);
}

async function maybeSkipCandidateChoice(commandId, state, tracker) {
  if (!state?.candidateChoiceVisible || !state?.skipButton) {
    return false;
  }

  const now = Date.now();
  if (tracker.lastSkipAt && (now - tracker.lastSkipAt) < 4000) {
    return false;
  }

  tracker.detected = true;
  tracker.attempts = Number(tracker.attempts || 0) + 1;
  tracker.lastSkipAt = now;
  await safePostProgress(commandId, "candidate_choice_skipped", {
    attempts: tracker.attempts,
    preferenceButtonCount: state.preferenceButtons?.length || 0
  });
  clickElement(state.skipButton);
  await delay(1500);
  return true;
}

async function resolveLingeringCandidateChoice(commandId, timeoutMs = 10000) {
  const deadline = Date.now() + timeoutMs;
  const tracker = {
    detected: false,
    attempts: 0,
    lastSkipAt: 0
  };

  while (Date.now() < deadline) {
    const state = pageCandidateChoiceState();
    if (!state.candidateChoiceVisible) {
      return tracker.detected;
    }
    if (!tracker.detected) {
      tracker.detected = true;
      await safePostProgress(commandId, "preflight_candidate_choice_detected", {
        preferenceButtonCount: state.preferenceButtons?.length || 0
      });
    }
    const skipped = await maybeSkipCandidateChoice(commandId, state, tracker);
    if (!skipped) {
      await delay(1000);
    }
  }

  throw new Error("Lingering candidate image choice could not be cleared before submission.");
}

async function captureDialogSaveDownload(commandId, beforeSources, settleMs, responseState = null) {
  const responseButton = pickPreferredResponseMediaButton(responseState);
  const targetEntry = !responseButton ? pickPreferredGeneratedImageTarget(beforeSources) : null;
  const triggerElement = responseButton?.element || targetEntry?.target || null;
  if (!triggerElement) {
    throw new Error("Could not locate a clickable target for the latest generated image.");
  }

  scrollElementIntoView(triggerElement, "center");
  await delay(200);
  await safePostProgress(commandId, "opening_image_dialog", {
    responseSectionCount: responseState?.responseSections?.length || 0,
    mediaButtonCount: responseState?.mediaButtons?.length || 0,
    buttonImageCount: responseButton?.images?.length || targetEntry?.images?.length || 0,
    newImageCount: targetEntry?.newImages?.length || responseState?.images?.length || 0
  });
  clickElement(triggerElement);

  const dialog = await waitForVisibleDialog(5000);
  if (!dialog) {
    throw new Error("Could not open ChatGPT's image dialog.");
  }

  const saveButton = await waitForVisibleControlByKeywords([dialog], SAVE_KEYWORDS, 5000);
  if (!saveButton) {
    throw new Error("Could not find the Save action in ChatGPT's image dialog.");
  }

  const dialogImage = (
    findDialogPrimaryImage(dialog) ||
    responseButton?.images?.at(-1) ||
    responseState?.images?.at(-1) ||
    targetEntry?.newImages?.at(-1) ||
    targetEntry?.images?.at(-1) ||
    null
  );
  try {
    const captured = await clickAndCaptureDownload(commandId, saveButton, Math.max(settleMs, 20000), true);
    const finalUrl = resolveHref(captured.finalUrl || captured.url || dialogImage?.src || "");
    return {
      method: "modal-save-download",
      url: finalUrl || (dialogImage?.src || ""),
      downloads: [{
        path: captured.filename || "",
        sourceUrl: finalUrl || (dialogImage?.src || ""),
        mimeType: captured.mime || "image/png",
        sizeBytes: captured.fileSize || captured.totalBytes || 0,
        width: dialogImage?.width || 0,
        height: dialogImage?.height || 0,
        alt: dialogImage?.alt || ""
      }]
    };
  } catch (error) {
    if (!dialogImage?.src) {
      throw error;
    }
    await safePostProgress(commandId, "dialog_image_fetch_fallback", {
      method: "dialog-image-fetch",
      src: dialogImage.src,
      error: error instanceof Error ? error.message : String(error)
    });
    const payload = await imageToPayload(dialogImage.src);
    return {
      method: "dialog-image-fetch",
      url: dialogImage.src,
      images: [{
        ...payload,
        width: dialogImage.width || 0,
        height: dialogImage.height || 0,
        alt: dialogImage.alt || ""
      }]
    };
  }
}

async function captureLegacyDownloadPayload(commandId, beforeSources, settleMs, responseState = null) {
  const targetEntry = responseState?.images?.at(-1) || collectNewImageEntries(beforeSources).at(-1) || null;
  const scopes = responseState?.responseSections?.length
    ? responseState.responseSections
    : (targetEntry ? buildActionScopes(targetEntry) : []);
  if (!scopes.length) {
    throw new Error("Could not locate the latest generated image card.");
  }

  if (targetEntry?.element) {
    revealImageActions(targetEntry, scopes);
  } else {
    for (const scope of scopes.slice(-2)) {
      scrollElementIntoView(scope, "center");
      dispatchHover(scope);
    }
  }

  const directLink = findDirectDownloadLink(scopes);
  if (directLink) {
    const href = resolveHref(directLink.href);
    await safePostProgress(commandId, "download_link_found", {
      method: "direct-link",
      href
    });
    const payload = await fetchUrlAsPayload(href);
    return {
      method: "direct-link",
      url: href,
      images: [{
        ...payload,
        width: targetEntry?.width || 0,
        height: targetEntry?.height || 0,
        alt: targetEntry?.alt || ""
      }]
    };
  }

  const directControl = findVisibleControlByKeywords(scopes, DOWNLOAD_KEYWORDS);
  if (directControl) {
    await safePostProgress(commandId, "download_control_found", {
      method: "visible-download-control"
    });
    const captured = await clickAndCaptureDownload(commandId, directControl, Math.max(settleMs, 12000), true);
    const finalUrl = resolveHref(captured.finalUrl || captured.url);
    const payload = await fetchUrlAsPayload(finalUrl);
    return {
      method: "download-control",
      url: finalUrl,
      images: [{
        ...payload,
        width: targetEntry?.width || 0,
        height: targetEntry?.height || 0,
        alt: targetEntry?.alt || ""
      }]
    };
  }

  const menuTrigger = findLikelyMenuTrigger(scopes);
  if (menuTrigger) {
    await safePostProgress(commandId, "download_menu_trigger_found", {
      method: "menu-trigger"
    });
    const captured = await clickAndCaptureDownload(commandId, menuTrigger, Math.max(settleMs, 12000), false);
    const finalUrl = resolveHref(captured.finalUrl || captured.url);
    const payload = await fetchUrlAsPayload(finalUrl);
    return {
      method: "captured-download",
      url: finalUrl,
      images: [{
        ...payload,
        width: targetEntry?.width || 0,
        height: targetEntry?.height || 0,
        alt: targetEntry?.alt || ""
      }]
    };
  }

  const responseImage = (
    responseState?.images?.at(-1) ||
    targetEntry?.newImages?.at(-1) ||
    targetEntry?.images?.at(-1) ||
    null
  );
  if (responseImage?.src) {
    await safePostProgress(commandId, "response_image_fetch_fallback", {
      method: "response-image-fetch",
      src: responseImage.src
    });
    const payload = await imageToPayload(responseImage.src);
    return {
      method: "response-image-fetch",
      url: responseImage.src,
      images: [{
        ...payload,
        width: responseImage.width || 0,
        height: responseImage.height || 0,
        alt: responseImage.alt || ""
      }]
    };
  }

  throw new Error("Could not find a final download action on the latest generated image.");
}

async function waitForResponseCaptureReady(commandId, beforeSources, responseState = null, timeoutMs = 20000) {
  if (!responseState?.userSection) {
    return responseState;
  }

  const deadline = Date.now() + Math.max(timeoutMs, 0);
  let bestState = responseState;
  const choiceTracker = {
    detected: false,
    attempts: 0,
    lastSkipAt: 0
  };
  while (Date.now() < deadline) {
    const currentState = responseTurnState(responseState.userSection, beforeSources);
    if (currentState.responseSections.length > 0) {
      bestState = currentState;
    }

    if (currentState.candidateChoiceVisible) {
      if (!choiceTracker.detected) {
        choiceTracker.detected = true;
        await safePostProgress(commandId, "candidate_choice_detected", {
          preferenceButtonCount: currentState.preferenceButtons?.length || 0
        });
      }
      const skipped = await maybeSkipCandidateChoice(commandId, currentState, choiceTracker);
      if (skipped) {
        continue;
      }
      await delay(1500);
      continue;
    }

    const captureReady = (
      (currentState.images.length > 0 || currentState.finalMediaButtons.length > 0) &&
      !currentState.progressVisible &&
      !generationSignals().busyGenerating
    );
    if (captureReady) {
      await safePostProgress(commandId, "capture_ready", {
        imageCount: currentState.images.length,
        mediaButtonCount: currentState.finalMediaButtons.length,
        responseSectionCount: currentState.responseSections.length
      });
      return currentState;
    }

    await delay(1500);
  }

  return bestState;
}

async function captureFinalAsset(commandId, beforeSources, settleMs, responseState = null) {
  responseState = await waitForResponseCaptureReady(
    commandId,
    beforeSources,
    responseState,
    Math.max(settleMs, 20000)
  );
  const modalDeadline = Date.now() + Math.max(settleMs, 20000);
  let modalAttempts = 0;
  let lastModalError = null;

  while (Date.now() < modalDeadline) {
    modalAttempts += 1;
    try {
      return await captureDialogSaveDownload(commandId, beforeSources, settleMs, responseState);
    } catch (error) {
      lastModalError = error;
      await safePostProgress(commandId, "modal_save_retry", {
        attempt: modalAttempts,
        error: error instanceof Error ? error.message : String(error),
        busyGenerating: generationSignals().busyGenerating
      });
      await closeVisibleDialog();
      await delay(1500);
    }
  }

  await safePostProgress(commandId, "modal_save_fallback", {
    attempts: modalAttempts,
    error: lastModalError instanceof Error ? lastModalError.message : String(lastModalError || "Unknown modal save error")
  });
  return captureLegacyDownloadPayload(commandId, beforeSources, settleMs, responseState);
}

async function waitForGenerationCompletion(commandId, userTurnSection, beforeSources, timeoutMs) {
  const deadline = Date.now() + timeoutMs;
  let previousSignature = "";
  let stableHits = 0;
  let pollCount = 0;
  let sawBusy = false;
  let sawResponseSections = false;
  let sawMedia = false;
  let announcedResponse = false;
  let announcedMedia = false;
  const choiceTracker = {
    detected: false,
    attempts: 0,
    lastSkipAt: 0
  };

  while (Date.now() < deadline) {
    pollCount += 1;
    await ensureBottomContext();
    const state = responseTurnState(userTurnSection, beforeSources);
    const signals = generationSignals();
    const networkSignals = networkSignalSummary(commandId);
    watchBusySignals(commandId);
    if (signals.busyGenerating) {
      sawBusy = true;
    }

    if (state.responseSections.length > 0) {
      sawResponseSections = true;
      if (!announcedResponse) {
        announcedResponse = true;
        await safePostProgress(commandId, "assistant_reply_detected", {
          responseSectionCount: state.responseSections.length
        });
      }
    }

    if (state.candidateChoiceVisible) {
      if (!choiceTracker.detected) {
        choiceTracker.detected = true;
        await safePostProgress(commandId, "candidate_choice_detected", {
          preferenceButtonCount: state.preferenceButtons?.length || 0
        });
      }
      const skipped = await maybeSkipCandidateChoice(commandId, state, choiceTracker);
      if (skipped) {
        previousSignature = "";
        stableHits = 0;
        continue;
      }
      await delay(1500);
      continue;
    }

    if (state.errorText && !signals.busyGenerating) {
      await safePostProgress(commandId, "generation_error_detected", {
        message: state.errorText,
        responseSectionCount: state.responseSections.length,
        pollCount
      });
      throw new Error(state.errorText);
    }

    const mediaReady = state.images.length > 0 || state.finalMediaButtons.length > 0;
    if (!mediaReady && networkSignals.lastErrorText && !signals.busyGenerating && state.responseSections.length > 0) {
      await safePostProgress(commandId, "generation_error_detected", {
        message: networkSignals.lastErrorText,
        source: "conversation_signal",
        responseSectionCount: state.responseSections.length,
        pollCount
      });
      throw new Error(networkSignals.lastErrorText);
    }

    if (mediaReady) {
      sawMedia = true;
      if (state.signature && state.signature === previousSignature) {
        stableHits += 1;
      } else {
        previousSignature = state.signature;
        stableHits = 1;
      }

      if (!announcedMedia) {
        announcedMedia = true;
        await safePostProgress(commandId, "media_detected", {
          imageCount: state.images.length,
          mediaButtonCount: state.finalMediaButtons.length,
          responseSectionCount: state.responseSections.length,
          progressVisible: state.progressVisible
        });
      }

      const finalResponseReady = (
        state.finalMediaButtons.length > 0 ||
        (state.images.length === 1 && !state.candidateChoiceVisible)
      );

      if (
        networkSignals.completionReady &&
        state.responseSections.length > 0 &&
        finalResponseReady &&
        !signals.busyGenerating &&
        !state.progressVisible
      ) {
        const completedAt = new Date().toISOString();
        await safePostProgress(commandId, "completed", {
          imageCount: state.images.length,
          mediaButtonCount: state.finalMediaButtons.length,
          progressVisible: state.progressVisible,
          completedAt,
          pollCount,
          completionSource: "network_signal",
          fileDownloadCount: networkSignals.fileDownloadCount,
          estuaryContentCount: networkSignals.estuaryContentCount,
          matchedFileIdCount: networkSignals.matchedFileIdCount
        });
        return {
          images: state.images,
          response: state,
          generation: {
            completed: true,
            completed_at: completedAt,
            final_image_count: state.images.length,
            busy_observed: sawBusy,
            response_sections_observed: sawResponseSections,
            media_observed: sawMedia,
            stable_polls: stableHits,
            poll_count: pollCount,
            completion_source: "network_signal"
          }
        };
      }

      if (stableHits >= 2 && !signals.busyGenerating && !state.progressVisible && finalResponseReady) {
        const completedAt = new Date().toISOString();
        await safePostProgress(commandId, "completed", {
          imageCount: state.images.length,
          mediaButtonCount: state.finalMediaButtons.length,
          progressVisible: state.progressVisible,
          completedAt,
          pollCount,
          completionSource: "dom_stable"
        });
        return {
          images: state.images,
          response: state,
          generation: {
            completed: true,
            completed_at: completedAt,
            final_image_count: state.images.length,
            busy_observed: sawBusy,
            response_sections_observed: sawResponseSections,
            media_observed: sawMedia,
            stable_polls: stableHits,
            poll_count: pollCount,
            completion_source: "dom_stable"
          }
        };
      }
    }

    await delay(1500);
  }

  throw new Error("Timed out waiting for generated images.");
}

async function settleResponseState(commandId, userTurnSection, beforeSources, initialState, settleMs) {
  if (settleMs <= 0) {
    return initialState;
  }

  const deadline = Date.now() + settleMs;
  let bestState = initialState;
  let previousSignature = initialState?.signature || "";
  let stableHits = 1;
  const choiceTracker = {
    detected: false,
    attempts: 0,
    lastSkipAt: 0
  };

  while (Date.now() < deadline) {
    await delay(2000);
    await ensureBottomContext();
    const currentState = responseTurnState(userTurnSection, beforeSources);
    watchBusySignals(commandId);
    if (currentState.responseSections.length > 0) {
      bestState = currentState;
      if (currentState.signature && currentState.signature === previousSignature) {
        stableHits += 1;
      } else {
        previousSignature = currentState.signature;
        stableHits = 1;
      }
    }

    if (currentState.candidateChoiceVisible) {
      if (!choiceTracker.detected) {
        choiceTracker.detected = true;
        await safePostProgress(commandId, "candidate_choice_detected", {
          preferenceButtonCount: currentState.preferenceButtons?.length || 0
        });
      }
      const skipped = await maybeSkipCandidateChoice(commandId, currentState, choiceTracker);
      if (skipped) {
        previousSignature = "";
        stableHits = 0;
      }
      continue;
    }

    if (stableHits >= 2 && !generationSignals().busyGenerating && !bestState.progressVisible) {
      await safePostProgress(commandId, "assets_settled", {
        imageCount: bestState?.images?.length || 0,
        mediaButtonCount: bestState?.finalMediaButtons?.length || 0,
        responseSectionCount: bestState?.responseSections?.length || 0,
        progressVisible: bestState?.progressVisible || false
      });
      return bestState;
    }
  }

  return bestState;
}

async function runGenerateImage(command) {
  const commandId = command?.id;
  const prompt = command?.payload?.prompt;
  const timeoutMs = Number(command?.payload?.timeout_ms || 600000);
  const postCompletionSettleMs = Number(command?.payload?.post_completion_settle_ms || 12000);
  if (!commandId || typeof commandId !== "string") {
    throw new Error("Command is missing its id.");
  }
  if (!prompt || typeof prompt !== "string") {
    throw new Error("Command payload missing prompt.");
  }

  activeCommandId = commandId;
  lastBusySignature = "";
  lastBusySignatureAt = 0;
  resetActiveNetworkSignals(commandId);
  await safePostLogEvent("command_window", {
    phase: "start",
    promptLength: prompt.length,
    timeoutMs,
    postCompletionSettleMs
  }, commandId);

  try {
    await safePostProgress(commandId, "preparing", {});
    watchBusySignals(commandId);
    const preflightDialogsClosed = await closeAllVisibleDialogs();
    if (preflightDialogsClosed > 0) {
      await safePostProgress(commandId, "dialogs_closed_preflight", {
        closedCount: preflightDialogsClosed
      });
    }
    await resolveLingeringCandidateChoice(commandId, 10000);
    const status = await waitForStablePageBaseline();
    if (status.loginRequired) {
      throw new Error("ChatGPT page is not logged in.");
    }

    await safePostProgress(commandId, "page_ready", {
      title: status.title,
      imageCount: status.imageCount
    });

    const composer = findComposer();
    if (!composer) {
      throw new Error("Could not find the ChatGPT composer.");
    }

    await ensureBottomContext();
    const beforeUserCount = userConversationSections().length;
    const beforeSources = new Set(collectLargeImages().map((item) => item.src));
    setComposerValue(composer, prompt);
    if (!composerText(composer).trim()) {
      throw new Error("Prompt was not inserted into the ChatGPT composer.");
    }
    await safePostProgress(commandId, "prompt_inserted", {
      promptLength: prompt.length
    });
    await delay(200);
    await submitPrompt(composer);
    await safePostProgress(commandId, "submitted", {
      existingUserTurnCount: beforeUserCount,
      existingImageCount: beforeSources.size
    });
    watchBusySignals(commandId);

    const userTurnSection = await waitForSubmittedUserTurn(commandId, beforeUserCount, prompt, 15000);
    const completion = await waitForGenerationCompletion(commandId, userTurnSection, beforeSources, timeoutMs);
    const settledResponse = await settleResponseState(
      commandId,
      userTurnSection,
      beforeSources,
      completion.response,
      postCompletionSettleMs
    );
    await safePostProgress(commandId, "resolving_final_download", {
      settleMs: postCompletionSettleMs
    });
    const finalCapture = await captureFinalAsset(
      commandId,
      beforeSources,
      postCompletionSettleMs,
      settledResponse
    );
    const cleanupDialogsClosed = await closeAllVisibleDialogs();
    await ensureBottomContext();
    await safePostProgress(commandId, "dialogs_closed_post_capture", {
      closedCount: cleanupDialogsClosed
    });
    await safePostProgress(commandId, "download_fetched", {
      method: finalCapture.method,
      imageCount: finalCapture.images?.length || finalCapture.downloads?.length || 0,
      url: finalCapture.url,
      path: finalCapture.downloads?.[0]?.path || null
    });
    await safePostLogEvent("command_window", {
      phase: "success",
      captureMethod: finalCapture.method,
      captureUrl: finalCapture.url,
      downloadedFile: finalCapture.downloads?.[0]?.path || null
    }, commandId);

    return {
      url: location.href,
      title: document.title,
      generation: {
        ...completion.generation,
        settle_ms: postCompletionSettleMs,
        capture_method: finalCapture.method,
        capture_url: finalCapture.url,
        downloaded_file: finalCapture.downloads?.[0]?.path || null
      },
      images: finalCapture.images || [],
      downloads: finalCapture.downloads || []
    };
  } catch (error) {
    await safePostLogEvent("command_window", {
      phase: "error",
      error: error instanceof Error ? error.message : String(error)
    }, commandId);
    throw error;
  } finally {
    activeNetworkSignals = null;
    activeCommandId = null;
  }
}

async function executeCommand(command) {
  if (!command || typeof command !== "object") {
    throw new Error("Invalid command.");
  }

  if (command.type === "status") {
    return pageStatus();
  }

  if (command.type === "generate_image") {
    return runGenerateImage(command);
  }

  throw new Error(`Unsupported command type: ${command.type}`);
}

async function postResult(commandId, status, payload, errorMessage = null) {
  await withBridgeNoiseSuppressed(() => fetch(`${serverBase}/api/result`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      command_id: commandId,
      client_id: clientId,
      status,
      payload,
      error: errorMessage
    })
  }));
}

function safeStorageSet(data) {
  try {
    chrome.storage.local.set(data, () => {
      void chrome.runtime?.lastError;
    });
  } catch {
    // Extension storage is optional for the bridge workflow.
  }
}

function loadStoredSettings(callback) {
  try {
    chrome.storage.local.get(["serverBase", "clientId"], (data) => {
      if (chrome.runtime?.lastError) {
        callback({});
        return;
      }
      callback(data || {});
    });
  } catch {
    callback({});
  }
}

async function pollOnce() {
  if (!workerMode || pollInFlight) return;
  pollInFlight = true;

  try {
    const response = await fetchJson(`${serverBase}/api/next?client_id=${encodeURIComponent(clientId)}`);
    const command = response.command;
    if (!command) return;

    try {
      const payload = await executeCommand(command);
      await postResult(command.id, "ok", payload, null);
      safeStorageSet({ lastResultAt: Date.now(), lastError: null });
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      await postResult(command.id, "error", null, message);
      safeStorageSet({ lastResultAt: Date.now(), lastError: message });
    }
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    safeStorageSet({ lastPollError: message, lastPollAt: Date.now() });
  } finally {
    pollInFlight = false;
  }
}

function stopPolling() {
  if (!pollTimer) return;
  window.clearInterval(pollTimer);
  pollTimer = null;
}

function startPolling() {
  if (!workerMode || pollTimer) return;
  pollTimer = window.setInterval(pollOnce, POLL_INTERVAL_MS);
  pollOnce();
}

function applyWorkerMode(active) {
  workerMode = Boolean(active);
  safeStorageSet({ workerMode, workerModeUpdatedAt: Date.now() });
  if (workerMode) {
    startPolling();
  } else {
    stopPolling();
  }
  return workerMode;
}

async function syncWorkerMode() {
  try {
    const response = await runtimeMessage({ type: "bridge:getWorkerBinding" });
    applyWorkerMode(Boolean(response?.isWorkerTab));
    return response;
  } catch (error) {
    applyWorkerMode(false);
    safeStorageSet({
      lastWorkerSyncError: error instanceof Error ? error.message : String(error),
      lastWorkerSyncAt: Date.now()
    });
    return null;
  }
}

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message?.type === "bridge:getStatus") {
    sendResponse({
      ok: true,
      status: pageStatus(),
      serverBase,
      clientId
    });
    return true;
  }

  if (message?.type === "bridge:setServerBase") {
    serverBase = String(message.serverBase || DEFAULT_SERVER_BASE);
    safeStorageSet({ serverBase });
    sendResponse({ ok: true, serverBase });
    return true;
  }

  if (message?.type === "bridge:pollNow") {
    if (!workerMode) {
      sendResponse({ ok: false, error: "当前页面不是已绑定的工作页。" });
      return true;
    }
    pollOnce().then(() => sendResponse({ ok: true })).catch((error) => {
      sendResponse({ ok: false, error: error instanceof Error ? error.message : String(error) });
    });
    return true;
  }

  if (message?.type === "bridge:workerBindingChanged") {
    sendResponse({ ok: true, workerMode: applyWorkerMode(Boolean(message.active)) });
    return true;
  }

  return false;
});

loadStoredSettings((data) => {
  if (typeof data.serverBase === "string" && data.serverBase) {
    serverBase = data.serverBase;
  }
  if (typeof data.clientId === "string" && data.clientId) {
    clientId = data.clientId;
  } else {
    safeStorageSet({ clientId });
  }
  installNetworkTrace();
  void syncWorkerMode();
});
