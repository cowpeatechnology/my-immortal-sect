function render(value) {
  document.getElementById("output").textContent =
    typeof value === "string" ? value : JSON.stringify(value, null, 2);
}

function isChatGPTTab(tab) {
  return typeof tab?.url === "string" && tab.url.startsWith("https://chatgpt.com/");
}

async function activeChatGPTTab() {
  const tabs = await chrome.tabs.query({ active: true, currentWindow: true });
  return tabs.find((tab) => isChatGPTTab(tab)) || null;
}

async function resolveBindTarget() {
  const activeTab = await activeChatGPTTab();
  if (activeTab?.id) {
    return activeTab;
  }

  const tabs = await chrome.tabs.query({ url: "https://chatgpt.com/*" });
  if (tabs.length === 1 && tabs[0]?.id) {
    return tabs[0];
  }
  if (tabs.length > 1) {
    throw new Error("检测到多个 chatgpt.com 标签页，请先切到要绑定的那个页面再点击绑定。");
  }
  throw new Error("当前没有找到可绑定的 chatgpt.com 标签页。");
}

async function workerState() {
  const response = await chrome.runtime.sendMessage({ type: "bridge:getWorkerTab" });
  return response?.workerTab || null;
}

async function sendToWorker(message) {
  const workerTab = await workerState();
  if (!workerTab?.tabId) {
    throw new Error("还没有绑定 ChatGPT 工作页。");
  }
  return chrome.tabs.sendMessage(workerTab.tabId, message);
}

async function refreshWorkerInfo() {
  const workerTab = await workerState();
  const target = document.getElementById("workerInfo");
  if (!workerTab?.tabId) {
    target.textContent = "工作页：未绑定";
    return null;
  }

  const title = workerTab.title || workerTab.url || "";
  target.textContent = `工作页 #${workerTab.tabId} · ${title}`;
  return workerTab;
}

async function init() {
  const serverInput = document.getElementById("serverBase");
  const stored = await chrome.storage.local.get(["serverBase"]);
  if (stored.serverBase) {
    serverInput.value = stored.serverBase;
  }
  await refreshWorkerInfo();

  document.getElementById("save").addEventListener("click", async () => {
    const serverBase = serverInput.value.trim() || "http://127.0.0.1:8765";
    await chrome.storage.local.set({ serverBase });
    try {
      let result = { ok: true, serverBase, workerTabId: null };
      const workerTab = await workerState();
      if (workerTab?.tabId) {
        const response = await chrome.tabs.sendMessage(workerTab.tabId, { type: "bridge:setServerBase", serverBase });
        result = { ...response, workerTabId: workerTab.tabId };
      }
      render(result);
    } catch (error) {
      render(error instanceof Error ? error.message : String(error));
    }
  });

  document.getElementById("bind").addEventListener("click", async () => {
    try {
      const tab = await resolveBindTarget();
      const result = await chrome.runtime.sendMessage({ type: "bridge:setWorkerTab", tabId: tab.id });
      await refreshWorkerInfo();
      render(result);
    } catch (error) {
      render(error instanceof Error ? error.message : String(error));
    }
  });

  document.getElementById("unbind").addEventListener("click", async () => {
    try {
      const result = await chrome.runtime.sendMessage({ type: "bridge:clearWorkerTab", reason: "popup_unbind" });
      await refreshWorkerInfo();
      render(result);
    } catch (error) {
      render(error instanceof Error ? error.message : String(error));
    }
  });

  document.getElementById("status").addEventListener("click", async () => {
    try {
      const result = await sendToWorker({ type: "bridge:getStatus" });
      render(result);
    } catch (error) {
      render(error instanceof Error ? error.message : String(error));
    }
  });

  document.getElementById("poll").addEventListener("click", async () => {
    try {
      const result = await sendToWorker({ type: "bridge:pollNow" });
      render(result);
    } catch (error) {
      render(error instanceof Error ? error.message : String(error));
    }
  });

  render("ready");
}

init().catch((error) => {
  render(error instanceof Error ? error.message : String(error));
});
