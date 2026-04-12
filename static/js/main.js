// State
let subscriptions = [];
let templates = [];
let currentTemplateId = null;

// ── Utilities ────────────────────────────────────────────────────────────────

function api(path, options = {}) {
  const headers = { "Content-Type": "application/json" };
  const token = localStorage.getItem("token");
  if (token) headers["Authorization"] = `Bearer ${token}`;

  return fetch(`/api${path}`, { ...options, headers: { ...headers, ...options.headers } })
    .then(async (res) => {
      const json = await res.json().catch(() => ({}));
      if (!res.ok) throw new Error(json.error || `HTTP ${res.status}`);
      return json;
    });
}

function showToast(message, isError = false) {
  const toast = document.getElementById("toast");
  const msg = document.getElementById("toast-message");
  msg.textContent = message;
  toast.classList.toggle("error", isError);
  toast.classList.remove("hidden");
  clearTimeout(toast._timer);
  toast._timer = setTimeout(() => toast.classList.add("hidden"), 3500);
}

function copyToClipboard(text) {
  navigator.clipboard.writeText(text).then(
    () => showToast("已复制到剪贴板"),
    () => showToast("复制失败，请手动复制", true)
  );
}

function formatBytes(bytes) {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`;
}

function relativeTime(dateStr) {
  if (!dateStr) return "—";
  const diff = Date.now() - new Date(dateStr).getTime();
  const m = Math.floor(diff / 60000);
  if (m < 1) return "刚刚";
  if (m < 60) return `${m} 分钟前`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h} 小时前`;
  return `${Math.floor(h / 24)} 天前`;
}

// ── Tab Switching ─────────────────────────────────────────────────────────────

function initTabs() {
  const tabBtns = document.querySelectorAll(".workspace-tab");
  tabBtns.forEach((btn) => {
    btn.addEventListener("click", () => {
      tabBtns.forEach((b) => {
        b.classList.remove("is-active");
        b.setAttribute("aria-selected", "false");
      });
      btn.classList.add("is-active");
      btn.setAttribute("aria-selected", "true");

      const target = btn.dataset.tabTarget;
      document.querySelectorAll(".workspace-panel").forEach((p) => {
        p.classList.remove("is-active");
        p.hidden = true;
      });
      const panel = document.getElementById(target);
      if (panel) {
        panel.classList.add("is-active");
        panel.hidden = false;
      }
    });
  });
}

// ── Quick Action Routing ──────────────────────────────────────────────────────

function initQuickActions() {
  document.getElementById("hero-subscribe-btn")?.addEventListener("click", () => {
    switchTab("subscriptions-panel");
    document.getElementById("subscribe-form")?.querySelector("input[name=name]")?.focus();
  });

  document.getElementById("hero-template-btn")?.addEventListener("click", () => {
    switchTab("templates-panel");
    newTemplate();
  });

  document.getElementById("default-template-link-btn")?.addEventListener("click", () => {
    copyToClipboard(`${location.origin}/default-template`);
  });

  document.getElementById("default-expanded-template-link-btn")?.addEventListener("click", () => {
    copyToClipboard(`${location.origin}/default-template/proxies`);
  });
}

function switchTab(tabId) {
  const btn = document.querySelector(`[data-tab-target="${tabId}"]`);
  if (btn) btn.click();
}

// ── Dashboard Metrics ─────────────────────────────────────────────────────────

function updateDashboardMetrics() {
  document.getElementById("dashboard-subscription-count").textContent = subscriptions.length;

  const templateCount = document.getElementById("dashboard-template-count");
  if (templateCount) templateCount.textContent = templates.length;
}

// ── Advanced Settings Toggle ────────────────────────────────────────────────

function initAdvancedToggle() {
  document.querySelectorAll("[data-toggle-advanced]").forEach((btn) => {
    btn.addEventListener("click", () => {
      const targetId = btn.dataset.toggleAdvanced;
      const panel = document.getElementById(targetId);
      if (!panel) return;
      const isHidden = panel.hidden;
      panel.hidden = !isHidden;
      btn.setAttribute("aria-expanded", String(isHidden));
    });
  });
}

// ── Request Headers Helpers ─────────────────────────────────────────────────

function parseHeadersText(text) {
  const headers = {};
  text.split("\n").forEach((line) => {
    const idx = line.indexOf(":");
    if (idx > 0) {
      const key = line.slice(0, idx).trim();
      const val = line.slice(idx + 1).trim();
      if (key && val) headers[key] = val;
    }
  });
  return headers;
}

function stringifyHeaders(headers) {
  return Object.entries(headers || {})
    .map(([k, v]) => `${k}: ${v}`)
    .join("\n");
}

// ── Subscription List ─────────────────────────────────────────────────────────

function renderSubscriptionList() {
  const listEl = document.getElementById("subscriptions");
  const loadingEl = document.getElementById("loading");
  const emptyEl = document.getElementById("empty-state");

  if (!listEl) return;

  loadingEl.hidden = true;

  if (subscriptions.length === 0) {
    listEl.innerHTML = "";
    emptyEl.hidden = false;
    return;
  }

  emptyEl.hidden = true;
  listEl.innerHTML = subscriptions
    .map(
      (sub) => `
    <article class="subscription-card" data-id="${sub.id}">
      <div class="subscription-head">
        <h3 class="subscription-name">${escapeHtml(sub.name)}</h3>
        <span class="subscription-type">${escapeHtml(sub.type || "unknown")}</span>
      </div>
      <div class="subscription-meta">
        <div class="subscription-meta-item">
          <strong>地址</strong>
          <span>${escapeHtml(sub.url)}</span>
        </div>
        ${sub.filter ? `<div class="subscription-meta-item"><strong>筛选</strong><span>${escapeHtml(sub.filter)}</span></div>` : ""}
        ${sub.file_size ? `<div class="subscription-meta-item"><strong>大小</strong><span>${formatBytes(sub.file_size)}</span></div>` : ""}
        ${sub.node_count ? `<div class="subscription-meta-item"><strong>节点</strong><span>${sub.node_count}</span></div>` : ""}
        <div class="subscription-meta-item">
          <strong>状态</strong>
          <span>${sub.status || "active"}</span>
        </div>
        ${sub.last_check ? `<div class="subscription-meta-item"><strong>检查</strong><span>${relativeTime(sub.last_check)}</span></div>` : ""}
        ${Object.keys(sub.request_headers || {}).length > 0 ? `<div class="subscription-meta-item"><strong>请求头</strong><span>${Object.keys(sub.request_headers).length} 条</span></div>` : ""}
      </div>
      <div class="subscription-actions">
        <button type="button" class="btn btn-secondary btn-compact copy-download-btn" data-id="${sub.id}">复制下载地址</button>
        <button type="button" class="btn btn-secondary btn-compact edit-subscription-btn" data-id="${sub.id}">编辑</button>
        <button type="button" class="btn btn-secondary btn-compact refresh-subscription-btn" data-id="${sub.id}">刷新</button>
        <button type="button" class="btn btn-danger btn-compact delete-subscription-btn" data-id="${sub.id}">删除</button>
      </div>
    </article>
  `
    )
    .join("");

  // Update subscription-last-updated
  const lastUpdatedEl = document.getElementById("subscription-last-updated");
  if (lastUpdatedEl) {
    const latest = subscriptions.reduce((max, s) => {
      if (!s.last_check) return max;
      const t = new Date(s.last_check).getTime();
      return t > max ? t : max;
    }, 0);
    lastUpdatedEl.textContent = latest ? relativeTime(new Date(latest).toISOString()) : "等待数据载入";
  }

  // Update header usage
  const headerUsageEl = document.getElementById("subscription-header-usage");
  if (headerUsageEl) {
    const count = subscriptions.filter((s) => s.request_headers && Object.keys(s.request_headers).length > 0).length;
    headerUsageEl.textContent = `${count} 条订阅已配置`;
  }

  // Bind events
  listEl.querySelectorAll(".copy-download-btn").forEach((btn) => {
    btn.addEventListener("click", () => copyToClipboard(`${location.origin}/download/${btn.dataset.id}`));
  });
  listEl.querySelectorAll(".edit-subscription-btn").forEach((btn) => {
    btn.addEventListener("click", () => openEditModal(btn.dataset.id));
  });
  listEl.querySelectorAll(".refresh-subscription-btn").forEach((btn) => {
    btn.addEventListener("click", () => refreshSubscription(btn.dataset.id));
  });
  listEl.querySelectorAll(".delete-subscription-btn").forEach((btn) => {
    btn.addEventListener("click", () => deleteSubscription(btn.dataset.id));
  });
}

function escapeHtml(str) {
  const div = document.createElement("div");
  div.textContent = str;
  return div.innerHTML;
}

// ── Subscription Form ─────────────────────────────────────────────────────────

function initSubscriptionForm() {
  const form = document.getElementById("subscribe-form");
  if (!form) return;

  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    const btn = document.getElementById("submit-btn");
    const orig = btn.textContent;
    btn.disabled = true;
    btn.textContent = "添加中...";

    try {
      const fd = new FormData(form);
      const headersText = fd.get("request_headers_text") || "";
      const payload = {
        name: fd.get("name"),
        url: fd.get("url"),
        type: fd.get("type") || "clash",
        filter: fd.get("filter") || "",
        request_headers: parseHeadersText(headersText),
      };

      const res = await api("/subscribe", { method: "POST", body: JSON.stringify(payload) });
      if (res.success) {
        showToast("订阅添加成功");
        form.reset();
        document.getElementById("subscription-advanced").hidden = true;
        await loadSubscriptions();
        updateDashboardMetrics();
      } else {
        showToast(res.error || "添加失败", true);
      }
    } catch (err) {
      showToast(err.message, true);
    } finally {
      btn.disabled = false;
      btn.textContent = orig;
    }
  });
}

// ── Subscription CRUD ─────────────────────────────────────────────────────────

async function loadSubscriptions() {
  try {
    const res = await api("/subscriptions");
    subscriptions = res.data || [];
    renderSubscriptionList();
  } catch (err) {
    showToast("加载订阅失败: " + err.message, true);
  }
}

async function refreshSubscription(id) {
  const sub = subscriptions.find((s) => s.id === id);
  if (!sub) return;
  try {
    const res = await api(`/subscribe/${id}/refresh`, {
      method: "POST",
      body: JSON.stringify({
        name: sub.name,
        url: sub.url,
        type: sub.type,
        filter: sub.filter,
        request_headers: sub.request_headers || {},
      }),
    });
    if (res.success) {
      showToast("刷新成功");
      await loadSubscriptions();
      updateDashboardMetrics();
    } else {
      showToast(res.error || "刷新失败", true);
    }
  } catch (err) {
    showToast(err.message, true);
  }
}

async function deleteSubscription(id) {
  if (!confirm("确定要删除这个订阅吗？")) return;
  try {
    const res = await api(`/subscribe/${id}`, { method: "DELETE" });
    if (res.success) {
      showToast("已删除");
      await loadSubscriptions();
      updateDashboardMetrics();
    } else {
      showToast(res.error || "删除失败", true);
    }
  } catch (err) {
    showToast(err.message, true);
  }
}

// ── Edit Modal ────────────────────────────────────────────────────────────────

function openEditModal(id) {
  const sub = subscriptions.find((s) => s.id === id);
  if (!sub) return;

  const modal = document.getElementById("edit-modal");
  const form = document.getElementById("edit-form");
  form.reset();

  form.querySelector('[name="id"]').value = sub.id;
  form.querySelector('[name="name"]').value = sub.name;
  form.querySelector('[name="url"]').value = sub.url;
  form.querySelector('[name="type"]').value = sub.type || "clash";
  form.querySelector('[name="filter"]').value = sub.filter || "";
  document.getElementById("edit-request-headers-input").value = stringifyHeaders(sub.request_headers);

  modal.classList.remove("hidden");

  // Bind modal buttons
  document.getElementById("close-edit-modal").onclick = () => modal.classList.add("hidden");
  document.getElementById("cancel-edit-modal").onclick = () => modal.classList.add("hidden");

  document.getElementById("edit-save-btn").onclick = () => saveEditOnly(id);
  document.getElementById("edit-submit-btn").onclick = () => saveEditAndRefresh(id);

  form.onsubmit = (e) => {
    e.preventDefault();
    saveEditAndRefresh(id);
  };
}

async function saveEditOnly(id) {
  const form = document.getElementById("edit-form");
  const fd = new FormData(form);
  try {
    const res = await api(`/subscribe/${id}`, {
      method: "PUT",
      body: JSON.stringify({
        name: fd.get("name"),
        url: fd.get("url"),
        type: fd.get("type"),
        filter: fd.get("filter"),
        request_headers: parseHeadersText(fd.get("request_headers_text")),
      }),
    });
    if (res.success) {
      showToast("保存成功");
      document.getElementById("edit-modal").classList.add("hidden");
      await loadSubscriptions();
      updateDashboardMetrics();
    } else {
      showToast(res.error || "保存失败", true);
    }
  } catch (err) {
    showToast(err.message, true);
  }
}

async function saveEditAndRefresh(id) {
  const form = document.getElementById("edit-form");
  const fd = new FormData(form);
  const sub = subscriptions.find((s) => s.id === id);
  try {
    const res = await api(`/subscribe/${id}/refresh`, {
      method: "POST",
      body: JSON.stringify({
        name: fd.get("name"),
        url: fd.get("url"),
        type: fd.get("type"),
        filter: fd.get("filter"),
        request_headers: parseHeadersText(fd.get("request_headers_text")),
      }),
    });
    if (res.success) {
      showToast("保存并刷新成功");
      document.getElementById("edit-modal").classList.add("hidden");
      await loadSubscriptions();
      updateDashboardMetrics();
    } else {
      showToast(res.error || "操作失败", true);
    }
  } catch (err) {
    showToast(err.message, true);
  }
}

// ── Template List ─────────────────────────────────────────────────────────────

function renderTemplateList() {
  const listEl = document.getElementById("templates");
  const emptyEl = document.getElementById("template-empty-state");
  if (!listEl) return;

  if (templates.length === 0) {
    listEl.innerHTML = "";
    emptyEl.hidden = false;
    return;
  }

  emptyEl.hidden = true;
  listEl.innerHTML = templates
    .map(
      (t) => `
    <article class="template-card ${t.id === currentTemplateId ? "active" : ""}" data-id="${t.id}">
      <div class="template-card-head">
        <h3 class="template-name">${escapeHtml(t.name)}</h3>
      </div>
      <p class="template-card-meta">
        ${t.use_all_subscriptions ? "使用全部订阅" : `${(t.selected_subscription_ids || []).length} 个订阅`}
        · ${relativeTime(t.updated_at)}
      </p>
      <div class="template-actions">
        <button type="button" class="btn btn-secondary btn-compact select-template-btn" data-id="${t.id}">选择</button>
      </div>
    </article>
  `
    )
    .join("");

  const lastUpdated = document.getElementById("template-last-updated");
  if (lastUpdated) {
    const latest = templates.reduce((max, t) => {
      if (!t.updated_at) return max;
      const d = new Date(t.updated_at).getTime();
      return d > max ? d : max;
    }, 0);
    lastUpdated.textContent = latest ? relativeTime(new Date(latest).toISOString()) : "等待数据载入";
  }

  // Bind events
  listEl.querySelectorAll(".select-template-btn").forEach((btn) => {
    btn.addEventListener("click", () => selectTemplate(btn.dataset.id));
  });
}

async function selectTemplate(id) {
  try {
    const res = await api(`/templates/${id}`);
    if (res.success) {
      currentTemplateId = id;
      populateTemplateForm(res.data);
      renderTemplateList();
    }
  } catch (err) {
    showToast("加载模板失败: " + err.message, true);
  }
}

function populateTemplateForm(t) {
  const form = document.getElementById("template-form");
  if (!form) return;
  form.reset();
  form.querySelector('[name="id"]').value = t.id;
  form.querySelector('[name="name"]').value = t.name;
  form.querySelector('[name="content"]').value = t.content;
  document.getElementById("template-current-name").textContent = t.name;
  document.getElementById("template-editor-status").textContent = "编辑模式";
  document.getElementById("template-current-subscription-count").textContent = t.use_all_subscriptions
    ? "将使用全部订阅"
    : `${(t.selected_subscription_ids || []).length} 个订阅`;

  // Populate subscription options
  renderTemplateSubscriptionOptions(t);

  // Show delete button only for existing templates
  const deleteBtn = document.getElementById("delete-template-btn");
  if (deleteBtn) deleteBtn.style.display = t.id ? "" : "none";
}

function renderTemplateSubscriptionOptions(template) {
  const container = document.getElementById("template-subscription-options");
  if (!container) return;

  if (subscriptions.length === 0) {
    container.innerHTML = '<p class="template-subscription-empty">暂无订阅可供选择</p>';
    return;
  }

  const useAll = template?.use_all_subscriptions === true;
  const selected = new Set(template?.selected_subscription_ids || []);

  container.innerHTML = subscriptions
    .map(
      (sub) => `
    <label class="template-subscription-option">
      <input type="checkbox" name="template-subs" value="${sub.id}" ${useAll || selected.has(sub.id) ? "checked" : ""}>
      <div class="template-subscription-copy">
        <strong>${escapeHtml(sub.name)}</strong>
        <span>${escapeHtml(sub.url)}</span>
      </div>
    </label>
  `
    )
    .join("");
}

// ── Template Form ────────────────────────────────────────────────────────────

function initTemplateForm() {
  const form = document.getElementById("template-form");
  if (!form) return;

  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    const btn = document.getElementById("template-submit-btn");
    const orig = btn.textContent;
    btn.disabled = true;
    btn.textContent = "保存中...";

    try {
      const fd = new FormData(form);
      const useAll = fd.get("use_all") === "on";
      const selectedIds = useAll
        ? []
        : Array.from(form.querySelectorAll('input[name="template-subs"]:checked')).map((c) => c.value);

      const payload = {
        name: fd.get("name"),
        content: fd.get("content"),
        use_all_subscriptions: useAll,
        selected_subscription_ids: selectedIds,
      };

      const id = fd.get("id");
      const isNew = !id;
      const res = isNew
        ? await api("/templates", { method: "POST", body: JSON.stringify(payload) })
        : await api(`/templates/${id}`, { method: "PUT", body: JSON.stringify(payload) });

      if (res.success) {
        showToast(isNew ? "模板创建成功" : "模板更新成功");
        await loadTemplates();
        if (res.data?.id) {
          currentTemplateId = res.data.id;
          await selectTemplate(res.data.id);
        }
        updateDashboardMetrics();
      } else {
        showToast(res.error || "保存失败", true);
      }
    } catch (err) {
      showToast(err.message, true);
    } finally {
      btn.disabled = false;
      btn.textContent = orig;
    }
  });

  document.getElementById("copy-template-url-btn")?.addEventListener("click", () => {
    if (currentTemplateId) copyToClipboard(`${location.origin}/api/templates/${currentTemplateId}/render`);
    else showToast("请先选择一个模板", true);
  });

  document.getElementById("copy-expanded-template-url-btn")?.addEventListener("click", () => {
    if (currentTemplateId) copyToClipboard(`${location.origin}/api/templates/${currentTemplateId}/render-proxies`);
    else showToast("请先选择一个模板", true);
  });

  document.getElementById("delete-template-btn")?.addEventListener("click", async () => {
    if (!currentTemplateId) return;
    if (!confirm("确定要删除这个模板吗？")) return;
    try {
      const res = await api(`/templates/${currentTemplateId}`, { method: "DELETE" });
      if (res.success) {
        showToast("已删除");
        currentTemplateId = null;
        newTemplate();
        await loadTemplates();
        updateDashboardMetrics();
      } else {
        showToast(res.error || "删除失败", true);
      }
    } catch (err) {
      showToast(err.message, true);
    }
  });

  document.getElementById("new-template-btn")?.addEventListener("click", newTemplate);

  document.getElementById("select-all-template-subscriptions-btn")?.addEventListener("click", () => {
    document.querySelectorAll('#template-form input[name="template-subs"]').forEach((c) => (c.checked = true));
  });

  document.getElementById("clear-template-subscriptions-btn")?.addEventListener("click", () => {
    document.querySelectorAll('#template-form input[name="template-subs"]').forEach((c) => (c.checked = false));
  });
}

function newTemplate() {
  currentTemplateId = null;
  const form = document.getElementById("template-form");
  if (form) form.reset();
  document.getElementById("template-editor-status").textContent = "新建模式";
  document.getElementById("template-current-name").textContent = "未命名模板";
  document.getElementById("template-current-subscription-count").textContent = "将使用全部订阅";
  const deleteBtn = document.getElementById("delete-template-btn");
  if (deleteBtn) deleteBtn.style.display = "none";
  renderTemplateSubscriptionOptions({});
  renderTemplateList();
}

// ── Template CRUD ─────────────────────────────────────────────────────────────

async function loadTemplates() {
  try {
    const res = await api("/templates");
    templates = res.data || [];
    renderTemplateList();
  } catch (err) {
    showToast("加载模板失败: " + err.message, true);
  }
}

// ── Reload button ────────────────────────────────────────────────────────────

function initReloadButton() {
  document.getElementById("reload-subscriptions-btn")?.addEventListener("click", async () => {
    const btn = document.getElementById("reload-subscriptions-btn");
    const orig = btn.textContent;
    btn.disabled = true;
    btn.textContent = "刷新中...";
    try {
      await loadSubscriptions();
      await loadTemplates();
      updateDashboardMetrics();
      showToast("已刷新");
    } finally {
      btn.disabled = false;
      btn.textContent = orig;
    }
  });
}

// ── Init ─────────────────────────────────────────────────────────────────────

document.addEventListener("DOMContentLoaded", async () => {
  initTabs();
  initQuickActions();
  initAdvancedToggle();
  initSubscriptionForm();
  initTemplateForm();
  initReloadButton();

  await loadSubscriptions();
  await loadTemplates();
  updateDashboardMetrics();
  renderTemplateList();
});
