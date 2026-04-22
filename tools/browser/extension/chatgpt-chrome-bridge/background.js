const CAPTURE_KEY = "downloadCaptureState";
const WORKER_TAB_KEY = "workerTabState";
const ARM_TTL_MS = 120000;

function storageGet(keys) {
  return new Promise((resolve) => chrome.storage.local.get(keys, resolve));
}

function storageSet(items) {
  return new Promise((resolve) => chrome.storage.local.set(items, resolve));
}

function storageRemove(keys) {
  return new Promise((resolve) => chrome.storage.local.remove(keys, resolve));
}

function downloadsSearch(query) {
  return new Promise((resolve) => chrome.downloads.search(query, resolve));
}

function tabsQuery(queryInfo) {
  return new Promise((resolve) => chrome.tabs.query(queryInfo, resolve));
}

function tabsGet(tabId) {
  return new Promise((resolve, reject) => {
    chrome.tabs.get(tabId, (tab) => {
      const lastError = chrome.runtime?.lastError;
      if (lastError) {
        reject(new Error(lastError.message));
        return;
      }
      resolve(tab);
    });
  });
}

function tabsSendMessage(tabId, message) {
  return new Promise((resolve, reject) => {
    chrome.tabs.sendMessage(tabId, message, (response) => {
      const lastError = chrome.runtime?.lastError;
      if (lastError) {
        reject(new Error(lastError.message));
        return;
      }
      resolve(response || {});
    });
  });
}

function isChatGPTTab(tab) {
  return Boolean(tab?.id) && typeof tab?.url === "string" && tab.url.startsWith("https://chatgpt.com/");
}

function normalizeWorkerTab(tab, previous = null) {
  return {
    tabId: Number(tab.id),
    windowId: typeof tab.windowId === "number" ? tab.windowId : (previous?.windowId ?? null),
    url: String(tab.url || ""),
    title: String(tab.title || ""),
    boundAt: previous?.boundAt || Date.now(),
    updatedAt: Date.now()
  };
}

async function getWorkerTabState() {
  const data = await storageGet([WORKER_TAB_KEY]);
  return data?.[WORKER_TAB_KEY] || null;
}

async function setWorkerTabState(state) {
  await storageSet({ [WORKER_TAB_KEY]: state });
  return state;
}

async function clearWorkerTabState() {
  await storageRemove([WORKER_TAB_KEY]);
}

async function notifyWorkerBinding(tabId, active) {
  try {
    await tabsSendMessage(tabId, { type: "bridge:workerBindingChanged", active });
    return true;
  } catch {
    return false;
  }
}

async function validateWorkerTabState() {
  const state = await getWorkerTabState();
  if (!state?.tabId) {
    const discovered = await discoverSingleChatGPTTab();
    if (discovered) {
      await setWorkerTabState(discovered);
    }
    return discovered;
  }

  try {
    const tab = await tabsGet(Number(state.tabId));
    if (!isChatGPTTab(tab)) {
      await clearWorkerTabState();
      return null;
    }
    const normalized = normalizeWorkerTab(tab, state);
    await setWorkerTabState(normalized);
    return normalized;
  } catch {
    await clearWorkerTabState();
    const discovered = await discoverSingleChatGPTTab();
    if (discovered) {
      await setWorkerTabState(discovered);
    }
    return discovered;
  }
}

async function discoverSingleChatGPTTab() {
  const tabs = await tabsQuery({ url: "https://chatgpt.com/*" });
  const candidates = tabs.filter(isChatGPTTab);
  if (candidates.length !== 1) {
    return null;
  }
  return normalizeWorkerTab(candidates[0]);
}

async function bindWorkerTab(tabId) {
  const nextTab = await tabsGet(Number(tabId));
  if (!isChatGPTTab(nextTab)) {
    throw new Error("选中的标签页不是 chatgpt.com。");
  }

  const previous = await validateWorkerTabState();
  if (previous?.tabId && previous.tabId !== nextTab.id) {
    await notifyWorkerBinding(previous.tabId, false);
  }

  const nextState = normalizeWorkerTab(nextTab);
  await setWorkerTabState(nextState);
  await notifyWorkerBinding(nextState.tabId, true);
  return nextState;
}

async function unbindWorkerTab(reason = "manual") {
  const previous = (await validateWorkerTabState()) || (await getWorkerTabState());
  await clearWorkerTabState();
  if (previous?.tabId) {
    await notifyWorkerBinding(previous.tabId, false);
  }
  return {
    cleared: Boolean(previous),
    reason,
    previous: previous || null
  };
}

async function getCaptureState() {
  const data = await storageGet([CAPTURE_KEY]);
  return data?.[CAPTURE_KEY] || null;
}

async function setCaptureState(state) {
  await storageSet({ [CAPTURE_KEY]: state });
  return state;
}

async function clearCaptureState() {
  await storageRemove([CAPTURE_KEY]);
}

function captureIsActive(state) {
  return Boolean(state?.commandId) && (Date.now() - Number(state?.armedAt || 0)) <= ARM_TTL_MS;
}

function normalizeDownloadItem(item) {
  return {
    downloadId: item.id,
    url: item.url || "",
    finalUrl: item.finalUrl || "",
    filename: item.filename || "",
    mime: item.mime || "",
    state: item.state || "",
    totalBytes: item.totalBytes || 0,
    fileSize: item.fileSize || 0,
    danger: item.danger || "",
    startTime: item.startTime || "",
    endTime: item.endTime || "",
    canResume: Boolean(item.canResume)
  };
}

function applyDownloadDelta(download, delta) {
  const next = { ...download };
  for (const [key, value] of Object.entries(delta || {})) {
    if (!value || typeof value !== "object" || !("current" in value)) {
      continue;
    }
    next[key] = value.current;
  }
  return next;
}

async function getDownloadItem(downloadId) {
  if (!downloadId) {
    return null;
  }
  const matches = await downloadsSearch({ id: downloadId });
  return matches?.[0] || null;
}

async function armDownloadCapture(commandId) {
  return setCaptureState({
    commandId,
    armedAt: Date.now(),
    status: "armed",
    download: null
  });
}

async function readBlobAsBase64(blob) {
  const arrayBuffer = await blob.arrayBuffer();
  const bytes = new Uint8Array(arrayBuffer);
  const chunkSize = 0x8000;
  let binary = "";
  for (let index = 0; index < bytes.length; index += chunkSize) {
    binary += String.fromCharCode(...bytes.subarray(index, index + chunkSize));
  }
  return btoa(binary);
}

async function fetchUrlAsPayload(url) {
  const response = await fetch(url, {
    credentials: "include",
    cache: "no-store"
  });
  const blob = await response.blob();
  return {
    src: url,
    mimeType: blob.type || "image/png",
    sizeBytes: blob.size || 0,
    base64: await readBlobAsBase64(blob)
  };
}

async function handleMessage(message, sender) {
  if (message?.type === "bridge:getWorkerTab") {
    return { workerTab: await validateWorkerTabState() };
  }

  if (message?.type === "bridge:setWorkerTab") {
    return { workerTab: await bindWorkerTab(message.tabId) };
  }

  if (message?.type === "bridge:clearWorkerTab") {
    return await unbindWorkerTab(String(message.reason || "manual"));
  }

  if (message?.type === "bridge:getWorkerBinding") {
    const workerTab = await validateWorkerTabState();
    return {
      workerTab,
      isWorkerTab: Boolean(workerTab?.tabId && sender?.tab?.id === workerTab.tabId)
    };
  }

  if (message?.type === "bridge:armDownloadCapture") {
    return { capture: await armDownloadCapture(String(message.commandId || "")) };
  }

  if (message?.type === "bridge:getDownloadCapture") {
    return { capture: await getCaptureState() };
  }

  if (message?.type === "bridge:clearDownloadCapture") {
    await clearCaptureState();
    return { cleared: true };
  }

  if (message?.type === "bridge:fetchUrlAsPayload") {
    const url = String(message.url || "");
    if (!url) {
      throw new Error("Missing url.");
    }
    return { payload: await fetchUrlAsPayload(url) };
  }

  throw new Error(`Unsupported background message type: ${String(message?.type || "")}`);
}

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  handleMessage(message, sender)
    .then((payload) => sendResponse({ ok: true, ...payload }))
    .catch((error) => {
      sendResponse({
        ok: false,
        error: error instanceof Error ? error.message : String(error)
      });
    });
  return true;
});

chrome.runtime.onInstalled.addListener(() => {
  void clearCaptureState();
});

chrome.tabs.onRemoved.addListener((tabId) => {
  void (async () => {
    const workerTab = await getWorkerTabState();
    if (workerTab?.tabId === tabId) {
      await clearWorkerTabState();
    }
  })();
});

chrome.tabs.onUpdated.addListener((tabId, changeInfo, tab) => {
  void (async () => {
    const workerTab = await getWorkerTabState();
    if (workerTab?.tabId !== tabId) {
      return;
    }
    if (typeof changeInfo.url === "string" && !changeInfo.url.startsWith("https://chatgpt.com/")) {
      await unbindWorkerTab("navigated_away");
      return;
    }
    if (isChatGPTTab(tab)) {
      await setWorkerTabState(normalizeWorkerTab(tab, workerTab));
    }
  })();
});

chrome.downloads.onCreated.addListener((item) => {
  void (async () => {
    const state = await getCaptureState();
    if (!captureIsActive(state)) {
      return;
    }
    if (state?.download?.downloadId) {
      return;
    }
    await setCaptureState({
      ...state,
      status: "created",
      download: normalizeDownloadItem(item)
    });
  })();
});

chrome.downloads.onChanged.addListener((delta) => {
  void (async () => {
    const state = await getCaptureState();
    if (!captureIsActive(state)) {
      return;
    }
    if (!state?.download?.downloadId || state.download.downloadId !== delta.id) {
      return;
    }

    let updatedDownload = applyDownloadDelta(state.download, delta);
    const refreshedDownload = await getDownloadItem(updatedDownload.downloadId);
    if (refreshedDownload) {
      updatedDownload = normalizeDownloadItem(refreshedDownload);
    }
    const updatedState = {
      ...state,
      status: updatedDownload.state === "complete"
        ? "complete"
        : updatedDownload.state === "interrupted"
          ? "interrupted"
          : "changed",
      download: updatedDownload
    };
    await setCaptureState(updatedState);
  })();
});
