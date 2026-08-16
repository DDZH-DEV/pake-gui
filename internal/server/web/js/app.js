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
  switchTab("pack");
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

// —— Cloud (GitHub macOS) ——
const ghStatus = document.getElementById("gh-status");
const ghLoginLabel = document.getElementById("gh-login-label");
const ghDeviceHint = document.getElementById("gh-device-hint");
const ghDeviceCode = document.getElementById("gh-device-code");
const btnGhLogin = document.getElementById("btn-gh-login");
const btnGhLogout = document.getElementById("btn-gh-logout");
let cloudPollTimer = null;
let oauthPollTimer = null;

function applyGitHubSettingsView(s) {
  document.getElementById("gh-owner").value = s.owner || "DDZH-DEV";
  document.getElementById("gh-repo").value = s.repo || "pake-gui";
  document.getElementById("gh-workflow").value = s.workflow || "build-macos.yml";
  document.getElementById("gh-ref").value = s.ref || "";
  document.getElementById("gh-client-id").value = s.clientId || "";
  document.getElementById("gh-token").value = "";
  document.getElementById("gh-token").placeholder = s.tokenMasked
    ? `已保存 ${s.tokenMasked}（留空保留）`
    : "可选备用 PAT";
  if (s.configured && s.login) {
    ghLoginLabel.textContent = "已登录 @" + s.login;
    btnGhLogout.hidden = false;
    btnGhLogin.textContent = "重新授权";
    ghStatus.textContent = "已授权";
  } else if (s.configured) {
    ghLoginLabel.textContent = "已配置 Token（未显示用户名）";
    btnGhLogout.hidden = false;
    btnGhLogin.textContent = "使用 GitHub 授权";
    ghStatus.textContent = "已配置";
  } else {
    ghLoginLabel.textContent = "未登录";
    btnGhLogout.hidden = true;
    btnGhLogin.textContent = "使用 GitHub 授权";
    ghStatus.textContent = "未配置";
  }
}

async function loadGitHubSettings() {
  try {
    const res = await api("/api/cloud/github/settings");
    const data = await res.json();
    applyGitHubSettingsView(data.settings || {});
  } catch (err) {
    ghStatus.textContent = "加载失败：" + err.message;
  }
}

async function saveGitHubSettings() {
  const body = {
    owner: document.getElementById("gh-owner").value.trim() || "DDZH-DEV",
    repo: document.getElementById("gh-repo").value.trim() || "pake-gui",
    token: document.getElementById("gh-token").value.trim(),
    clientId: document.getElementById("gh-client-id").value.trim(),
    workflow: document.getElementById("gh-workflow").value.trim() || "build-macos.yml",
    ref: document.getElementById("gh-ref").value.trim(),
  };
  const res = await api("/api/cloud/github/settings", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  const data = await res.json();
  if (!data.ok) {
    ghStatus.textContent = data.error || "保存失败";
    appendLog("GitHub 设置失败：" + (data.error || ""));
    return;
  }
  ghStatus.textContent = "已保存";
  appendLog("GitHub 设置已保存");
  document.getElementById("gh-token").value = "";
  applyGitHubSettingsView(data.settings || {});
}

async function testGitHubSettings() {
  ghStatus.textContent = "测试中…";
  const res = await api("/api/cloud/github/test", { method: "POST" });
  const data = await res.json();
  if (!data.ok) {
    ghStatus.textContent = data.error || "连接失败";
    appendLog("GitHub 测试失败：" + (data.error || ""));
    return;
  }
  ghStatus.textContent = `连接成功 · 默认分支 ${data.defaultBranch || "?"}`;
  appendLog("GitHub 连接成功：" + (data.owner || "") + "/" + (data.repo || ""));
}

function stopOAuthPoll() {
  if (oauthPollTimer) {
    clearInterval(oauthPollTimer);
    oauthPollTimer = null;
  }
}

async function pollOAuthStatus() {
  const res = await api("/api/cloud/github/oauth/status");
  const data = await res.json();
  const session = data.session || {};
  if (session.pending && session.userCode) {
    ghDeviceCode.hidden = false;
    ghDeviceCode.textContent = session.userCode;
    ghDeviceHint.textContent = "请在浏览器中确认设备码，等待授权完成…";
  }
  if (session.done) {
    stopOAuthPoll();
    ghDeviceCode.hidden = true;
    if (session.ok) {
      ghDeviceHint.textContent = "授权成功";
      appendLog("GitHub 授权成功" + (session.login ? "：@" + session.login : ""));
      applyGitHubSettingsView(data.settings || {});
    } else {
      ghDeviceHint.textContent = session.error || "授权失败";
      appendLog("GitHub 授权失败：" + (session.error || ""));
      loadGitHubSettings();
    }
  }
}

async function startGitHubOAuth() {
  stopOAuthPoll();
  const body = {
    clientId: document.getElementById("gh-client-id").value.trim(),
    owner: document.getElementById("gh-owner").value.trim() || "DDZH-DEV",
    repo: document.getElementById("gh-repo").value.trim() || "pake-gui",
  };
  // Persist client id / repo first
  await api("/api/cloud/github/settings", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      ...body,
      workflow: document.getElementById("gh-workflow").value.trim() || "build-macos.yml",
      ref: document.getElementById("gh-ref").value.trim(),
      token: "",
      clientId: body.clientId,
    }),
  });

  ghStatus.textContent = "正在发起授权…";
  ghDeviceHint.textContent = "正在打开浏览器…";
  const res = await api("/api/cloud/github/oauth/start", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  const data = await res.json();
  if (!data.ok) {
    ghStatus.textContent = data.error || "无法启动授权";
    ghDeviceHint.textContent = data.error || "请检查 Client ID，并确认 OAuth App 已启用 Device Flow";
    appendLog("GitHub 授权启动失败：" + (data.error || ""));
    return;
  }
  const device = data.device || {};
  ghDeviceCode.hidden = false;
  ghDeviceCode.textContent = device.userCode || "";
  ghDeviceHint.textContent = "浏览器已打开；若未打开，请访问 " + (device.verificationUri || "https://github.com/login/device");
  ghStatus.textContent = "等待授权确认…";
  appendLog("GitHub 设备码：" + (device.userCode || "") + " → " + (device.verificationUri || ""));
  oauthPollTimer = setInterval(() => {
    pollOAuthStatus().catch(() => {});
  }, 2000);
}

async function logoutGitHub() {
  stopOAuthPoll();
  const res = await api("/api/cloud/github/oauth/logout", { method: "POST" });
  const data = await res.json();
  if (!data.ok) {
    appendLog("退出失败：" + (data.error || ""));
    return;
  }
  ghDeviceCode.hidden = true;
  ghDeviceHint.textContent = "已退出登录";
  appendLog("已退出 GitHub 登录");
  applyGitHubSettingsView(data.settings || {});
}

function stateBadge(state) {
  if (state === "success") return { cls: "ok", text: "成功" };
  if (state === "failed") return { cls: "bad", text: "失败" };
  if (state === "canceled") return { cls: "bad", text: "取消" };
  if (state === "running") return { cls: "ok", text: "进行中" };
  if (state === "queued") return { cls: "ok", text: "排队" };
  return { cls: "bad", text: state || "?" };
}

async function refreshCloudJobs() {
  const lists = [
    document.getElementById("cloud-job-list"),
    document.getElementById("cloud-job-list-2"),
  ].filter(Boolean);
  if (!lists.length) return;
  try {
    const res = await api("/api/cloud/jobs");
    const data = await res.json();
    const jobs = data.jobs || [];
    let needPoll = false;

    const renderInto = (el) => {
      el.innerHTML = "";
      if (!jobs.length) {
        el.innerHTML = `<li><span class="meta">暂无云端任务</span></li>`;
        return;
      }
      for (const job of jobs.slice(0, 12)) {
        const st = job.status?.state || "";
        if (st === "running" || st === "queued") needPoll = true;
        const li = document.createElement("li");
        const left = document.createElement("div");
        const name = job.request?.name || job.id;
        const plat = job.request?.platform || "macos";
        left.innerHTML = `<button type="button">${name}</button>
          <div class="meta">${plat} · ${job.id}<br/>${job.status?.message || ""}</div>`;
        const badge = document.createElement("span");
        const b = stateBadge(st);
        badge.className = `badge ${b.cls}`;
        badge.textContent = b.text;
        li.append(left, badge);
        left.querySelector("button").addEventListener("click", async () => {
          await api(`/api/cloud/jobs/${encodeURIComponent(job.id)}?action=open`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ action: "open" }),
          });
        });
        el.appendChild(li);
      }
    };

    for (const el of lists) renderInto(el);

    if (needPoll && !cloudPollTimer) {
      cloudPollTimer = setInterval(refreshCloudJobs, 8000);
    }
    if (!needPoll && cloudPollTimer) {
      clearInterval(cloudPollTimer);
      cloudPollTimer = null;
    }
  } catch {
    for (const el of lists) {
      el.innerHTML = `<li><span class="meta">无法加载云端任务</span></li>`;
    }
  }
}

async function submitCloudMacOS() {
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
  const platform = document.getElementById("cloud-platform")?.value || "macos";
  if (platform === "android") {
    appendLog("Android 云端打包尚未实现（T03 预留）");
    return;
  }

  const body = {
    platform: "macos",
    url: opts.url,
    name: opts.name,
    icon: opts.icon,
    width: opts.width,
    height: opts.height,
    appVersion: opts.appVersion,
    identifier: opts.identifier,
    hideTitleBar: opts.hideTitleBar,
    multiArch: document.getElementById("cloud-multi-arch")?.checked === true,
    newWindow: document.getElementById("cloud-new-window")?.checked === true,
    targets: document.getElementById("cloud-targets")?.value || "dmg",
  };

  appendLog("—— 提交 macOS 云端任务 ——");
  const res = await api("/api/cloud/jobs", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  const data = await res.json();
  if (!data.ok) {
    appendLog("提交失败：" + (data.error || res.status));
    switchTab("cloud");
    return;
  }
  appendLog("已创建任务：" + data.job?.id + "（后台轮询 GitHub Actions，产物在 builds/macos）");
  refreshCloudJobs();
  switchTab("cloud");
}

function switchTab(name) {
  const buttons = document.querySelectorAll(".tab-btn");
  const panels = {
    pack: document.getElementById("tab-pack"),
    cloud: document.getElementById("tab-cloud"),
    jobs: document.getElementById("tab-jobs"),
    env: document.getElementById("tab-env"),
  };
  for (const btn of buttons) {
    const on = btn.dataset.tab === name;
    btn.classList.toggle("active", on);
    btn.setAttribute("aria-selected", on ? "true" : "false");
  }
  for (const [key, panel] of Object.entries(panels)) {
    if (!panel) continue;
    const on = key === name;
    panel.classList.toggle("active", on);
    panel.hidden = !on;
  }
  if (name === "jobs" || name === "cloud") refreshCloudJobs();
  if (name === "env") refreshEnv();
  if (name === "jobs") refreshHistory();
}

document.querySelectorAll(".tab-btn").forEach((btn) => {
  btn.addEventListener("click", () => switchTab(btn.dataset.tab));
});

document.getElementById("btn-gh-save")?.addEventListener("click", saveGitHubSettings);
document.getElementById("btn-gh-test")?.addEventListener("click", testGitHubSettings);
document.getElementById("btn-gh-login")?.addEventListener("click", startGitHubOAuth);
document.getElementById("btn-gh-logout")?.addEventListener("click", logoutGitHub);
document.getElementById("btn-cloud-macos")?.addEventListener("click", submitCloudMacOS);
document.getElementById("btn-refresh-cloud")?.addEventListener("click", refreshCloudJobs);
document.getElementById("btn-refresh-cloud-2")?.addEventListener("click", refreshCloudJobs);

refreshEnv();
refreshHistory();
loadGitHubSettings();
refreshCloudJobs();
