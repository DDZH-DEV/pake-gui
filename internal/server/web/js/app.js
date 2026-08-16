const form = document.getElementById("build-form");
const logEl = document.getElementById("log");
const logStatus = document.getElementById("log-status");
const toolList = document.getElementById("tool-list");
const envSummary = document.getElementById("env-summary");
const historyList = document.getElementById("history-list");
const cmdPreview = document.getElementById("cmd-preview");
const outDirInput = document.getElementById("out-dir");
const btnBuild = document.getElementById("btn-build");
const btnCancel = document.getElementById("btn-cancel");

let building = false;
let abortController = null;

function resolveToken() {
  const q = new URLSearchParams(location.search).get("token");
  if (q) {
    window.__PAKE_TOKEN__ = q;
    return q;
  }
  return window.__PAKE_TOKEN__ || "";
}

const TOKEN = resolveToken();

function apiHeaders(extra = {}) {
  const h = { ...extra };
  if (TOKEN) h["X-Pake-Token"] = TOKEN;
  return h;
}

async function api(path, options = {}) {
  const opts = { ...options };
  opts.headers = apiHeaders(opts.headers || {});
  const res = await fetch(path, opts);
  return res;
}

function appendLog(line) {
  logEl.textContent += (logEl.textContent ? "\n" : "") + line;
  logEl.scrollTop = logEl.scrollHeight;
}

function setBuilding(on) {
  building = on;
  btnBuild.disabled = on;
  btnCancel.hidden = !on;
  logStatus.textContent = on ? "打包进行中…" : "等待开始";
}

function boolFields() {
  return [
    "hideWindowDecorations",
    "hideTitleBar",
    "fullscreen",
    "maximize",
    "alwaysOnTop",
    "darkMode",
    "showSystemTray",
    "startToTray",
    "multiInstance",
    "multiWindow",
    "enableFind",
    "enableDragDrop",
    "incognito",
    "wasm",
    "forceInternalNavigation",
    "ignoreCertificateErrors",
    "useLocalFile",
    "debug",
    "keepBinary",
    "iterativeBuild",
  ];
}

function looksAbsolutePath(p) {
  if (!p) return false;
  return /^[a-zA-Z]:[\\/]/.test(p) || p.startsWith("\\\\") || p.startsWith("/");
}

function collectOptions() {
  const fd = new FormData(form);
  const get = (k) => (fd.get(k) || "").toString().trim();
  const num = (k) => {
    const v = Number(get(k));
    return Number.isFinite(v) && v > 0 ? v : 0;
  };

  const injectRaw = get("inject");
  const inject = injectRaw
    ? injectRaw.split(",").map((s) => s.trim()).filter(Boolean)
    : [];

  const outDir = get("outDir");
  let allowExternalOutDir = false;
  if (outDir && looksAbsolutePath(outDir)) {
    const builds = window.__PAKE_BUILDS__ || "";
    const normalized = outDir.replace(/\//g, "\\").toLowerCase();
    const buildsNorm = String(builds).replace(/\//g, "\\").toLowerCase();
    if (!buildsNorm || !normalized.startsWith(buildsNorm)) {
      allowExternalOutDir = confirm(
        "输出目录不在默认 builds 目录内。\n\n" +
          outDir +
          "\n\n确认允许写入该路径吗？"
      );
      if (!allowExternalOutDir) {
        throw new Error("已取消：不允许写入 builds 以外的目录");
      }
    }
  }

  const opts = {
    url: get("url"),
    name: get("name"),
    icon: get("icon"),
    width: num("width"),
    height: num("height"),
    zoom: num("zoom"),
    appVersion: get("appVersion"),
    title: get("title"),
    identifier: get("identifier"),
    userAgent: get("userAgent"),
    activationShortcut: get("activationShortcut"),
    targets: get("targets"),
    safeDomain: get("safeDomain"),
    proxyUrl: get("proxyUrl"),
    outDir,
    allowExternalOutDir,
    inject,
  };

  for (const key of boolFields()) {
    opts[key] = form.elements[key]?.checked === true;
  }

  return opts;
}

function fillForm(opts) {
  if (!opts) return;
  for (const [key, value] of Object.entries(opts)) {
    const el = form.elements[key];
    if (!el) continue;
    if (el.type === "checkbox") {
      el.checked = Boolean(value);
    } else if (key === "inject" && Array.isArray(value)) {
      el.value = value.join(", ");
    } else if (value != null && value !== "") {
      el.value = value;
    }
  }
}

async function refreshEnv() {
  envSummary.textContent = "检测中…";
  try {
    const res = await api("/api/env");
    const data = await res.json();
    toolList.innerHTML = "";
    for (const t of data.tools || []) {
      const li = document.createElement("li");
      const left = document.createElement("div");
      left.innerHTML = `<strong>${t.name}</strong><div class="meta">${t.version || t.detail || ""}</div>`;
      const badge = document.createElement("span");
      badge.className = `badge ${t.ok ? "ok" : "bad"}`;
      badge.textContent = t.ok ? "就绪" : "缺失";
      li.append(left, badge);
      toolList.appendChild(li);
    }
    envSummary.textContent = data.ready
      ? `环境可用 · ${data.os}/${data.arch}`
      : `环境不完整 · ${data.os}/${data.arch}（可先安装 pake-cli / Rust）`;
    if (data.builds && !outDirInput.placeholder) {
      outDirInput.placeholder = data.builds;
    }
  } catch (err) {
    envSummary.textContent = "环境检测失败：" + err.message;
  }
}

async function refreshHistory() {
  try {
    const res = await api("/api/history");
    const items = await res.json();
    historyList.innerHTML = "";
    if (!items?.length) {
      historyList.innerHTML = `<li><span class="meta">暂无记录</span></li>`;
      return;
    }
    for (const item of items.slice(0, 8)) {
      const li = document.createElement("li");
      const left = document.createElement("div");
      const ok = item.result?.ok;
      left.innerHTML = `<button type="button" data-id="${item.id}">${item.options?.name || "未命名"}</button>
        <div class="meta">${item.options?.url || ""}</div>`;
      const badge = document.createElement("span");
      badge.className = `badge ${ok ? "ok" : "bad"}`;
      badge.textContent = ok ? "成功" : "失败";
      li.append(left, badge);
      li.querySelector("button").addEventListener("click", () => fillForm(item.options));
      historyList.appendChild(li);
    }
  } catch {
    historyList.innerHTML = `<li><span class="meta">无法加载历史</span></li>`;
  }
}

async function previewCmd() {
  try {
    const opts = collectOptions();
    const res = await api("/api/preview-cmd", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(opts),
    });
    const data = await res.json();
    if (!data.ok) {
      cmdPreview.textContent = data.error || "无法生成命令";
      return;
    }
    cmdPreview.textContent = data.command;
    if (!outDirInput.value) outDirInput.placeholder = data.outDir;
  } catch (err) {
    cmdPreview.textContent = err.message;
  }
}

async function startBuild(e) {
  e.preventDefault();
  if (building) return;

  let opts;
  try {
    opts = collectOptions();
  } catch (err) {
    appendLog(err.message);
    return;
  }
  if (!opts.url || !opts.name) {
    appendLog("请填写网址和应用名称");
    return;
  }

  setBuilding(true);
  appendLog("—— 开始打包 ——");
  abortController = new AbortController();

  try {
    const res = await api("/api/build", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(opts),
      signal: abortController.signal,
    });

    if (!res.ok || !res.body) {
      appendLog("请求失败: " + res.status + " " + (await res.text()));
      setBuilding(false);
      return;
    }

    const reader = res.body.getReader();
    const decoder = new TextDecoder();
    let buffer = "";

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });
      const parts = buffer.split("\n\n");
      buffer = parts.pop() || "";
      for (const chunk of parts) {
        handleSSE(chunk);
      }
    }
  } catch (err) {
    if (err.name !== "AbortError") {
      appendLog("错误: " + err.message);
    }
  } finally {
    setBuilding(false);
    refreshHistory();
  }
}

function handleSSE(chunk) {
  const lines = chunk.split("\n");
  let event = "message";
  let data = "";
  for (const line of lines) {
    if (line.startsWith("event:")) event = line.slice(6).trim();
    if (line.startsWith("data:")) data += line.slice(5).trim();
  }
  if (!data) return;
  let payload;
  try {
    payload = JSON.parse(data);
  } catch {
    appendLog(data);
    return;
  }
  if (event === "log") {
    appendLog(payload.line || "");
  } else if (event === "done") {
    if (payload.ok) {
      appendLog("✓ 打包完成: " + (payload.outDir || ""));
      logStatus.textContent = "打包成功";
    } else {
      appendLog("✗ 打包失败: " + (payload.message || ""));
      logStatus.textContent = "打包失败";
    }
  }
}

document.getElementById("btn-preview").addEventListener("click", previewCmd);
document.getElementById("btn-clear-log").addEventListener("click", () => {
  logEl.textContent = "";
});
document.getElementById("btn-refresh-env").addEventListener("click", refreshEnv);
document.getElementById("btn-open-out").addEventListener("click", async () => {
  const path = outDirInput.value.trim();
  const res = await api("/api/open-output", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ path }),
  });
  const data = await res.json();
  if (!data.ok) appendLog(data.error || "无法打开目录");
});
document.getElementById("btn-open-log").addEventListener("click", async () => {
  await api("/api/open-log", { method: "POST" });
});
document.getElementById("btn-install-pake").addEventListener("click", async () => {
  appendLog("正在安装 pake-cli …");
  const res = await api("/api/install-pake", { method: "POST" });
  const data = await res.json();
  appendLog(data.output || data.error || "");
  refreshEnv();
});
btnCancel.addEventListener("click", async () => {
  await api("/api/cancel", { method: "POST" });
  abortController?.abort();
  appendLog("已请求取消（将结束打包进程树）");
});

const iconPathInput = document.getElementById("icon-path");
const iconFileInput = document.getElementById("icon-file");
const iconPreviewImg = document.getElementById("icon-preview-img");
const iconPreviewPlaceholder = document.getElementById("icon-preview-placeholder");
const iconHint = document.getElementById("icon-hint");

function setIconPreview(src) {
  if (src) {
    iconPreviewImg.src = src;
    iconPreviewImg.hidden = false;
    iconPreviewPlaceholder.hidden = true;
  } else {
    iconPreviewImg.removeAttribute("src");
    iconPreviewImg.hidden = true;
    iconPreviewPlaceholder.hidden = false;
  }
}

function clearIcon() {
  iconPathInput.value = "";
  iconFileInput.value = "";
  setIconPreview("");
  iconHint.textContent = "支持 png / ico / icns / jpg / webp，最大 8MB";
}

document.getElementById("btn-clear-icon").addEventListener("click", clearIcon);

iconFileInput.addEventListener("change", async () => {
  const file = iconFileInput.files?.[0];
  if (!file) return;
  iconHint.textContent = "正在上传 " + file.name + " …";
  const fd = new FormData();
  fd.append("icon", file);
  try {
    const res = await api("/api/upload-icon", { method: "POST", body: fd });
    const data = await res.json();
    if (!data.ok) {
      iconHint.textContent = data.error || "上传失败";
      iconFileInput.value = "";
      return;
    }
    iconPathInput.value = data.path;
    setIconPreview(data.preview);
    iconHint.textContent = "已选择：" + (data.filename || file.name);
  } catch (err) {
    iconHint.textContent = "上传失败：" + err.message;
    iconFileInput.value = "";
  }
});

iconPathInput.addEventListener("change", () => {
  const v = iconPathInput.value.trim();
  if (/^https?:\/\//i.test(v)) {
    setIconPreview(v);
    iconHint.textContent = "使用网络图标地址";
  } else if (!v) {
    setIconPreview("");
  }
});

// drag & drop onto preview
const iconPreview = document.getElementById("icon-preview");
["dragenter", "dragover"].forEach((ev) => {
  iconPreview.addEventListener(ev, (e) => {
    e.preventDefault();
    iconPreview.classList.add("dragover");
  });
});
["dragleave", "drop"].forEach((ev) => {
  iconPreview.addEventListener(ev, (e) => {
    e.preventDefault();
    iconPreview.classList.remove("dragover");
  });
});
iconPreview.addEventListener("drop", (e) => {
  const file = e.dataTransfer?.files?.[0];
  if (!file) return;
  const dt = new DataTransfer();
  dt.items.add(file);
  iconFileInput.files = dt.files;
  iconFileInput.dispatchEvent(new Event("change"));
});

form.addEventListener("submit", startBuild);

refreshEnv();
refreshHistory();
