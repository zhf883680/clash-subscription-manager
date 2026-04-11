const API_BASE = "/api";
const DEFAULT_TEMPLATE_CONTENT = `
proxy-providers: {}
proxy-groups:
  - { name: 🚩 PROXY, type: select, proxies: [♻️ 自动选择, 🚀 手动切换] }
  - { name: 🤖 人工智能, type: fallback, proxies: [✨ AISelect] }

  #  代理
  - { name: 🚀 手动切换, type: select ,include-all: true }
  - {
      name: ♻️ 自动选择,
      type: fallback,
      interval: 300,
      proxies: [ 🇸🇬 Singapore, 🇯🇵 Japan ,🇺🇸 USA, 🚀 手动切换],
    }
  # 基础节点
  - {
      name: ✨ AISelect,
      type: select,
      interval: 300,
      proxies: [🇺🇸 USA,🇸🇬 Singapore, 🇯🇵 Japan],
    }

  - {
      name: 🇺🇸 USA,
      type: fallback,
      interval: 300,
      timeout: 1000,
      tolerance: 100,
      lazy:false,
      include-all: true,
      exclude-type: direct,
      filter: "美国|United States",
    }
  - {
      name: 🇯🇵 Japan,
      type: url-test,
      interval: 300,
      timeout: 1000,
      tolerance: 100,
      url: "https://www.gstatic.com/generate_204",
      lazy:false,
      include-all: true,
      exclude-type: direct,
      filter: "日本|Japan",
    }
  - {
      name: 🇸🇬 Singapore,
      type: url-test,
      interval: 300,
      timeout: 1000,
      tolerance: 100,
      url: "https://www.gstatic.com/generate_204",
      lazy:false,
      include-all: true,
      exclude-type: direct,
      filter: "新加坡|Singapore",
    }
  - {
      name: 🇭🇰 HongKong,
      type: url-test,
      interval: 300,
      timeout: 1000,
      tolerance: 100,
      url: "https://www.gstatic.com/generate_204",
      lazy:false,
      include-all: true,
      exclude-type: direct,
      filter: "香港|Hong Kong",
    }

rule-providers:
  AWAvenue-Ads:
    type: http
    behavior: domain
    format: yaml
    # path可为空(仅限clash.meta 1.15.0以上版本)
    path: ./ruleset/AWAvenue-Ads.yaml
    url: "https://raw.githubusercontent.com/TG-Twilight/AWAvenue-Ads-Rule/main/Filters/AWAvenue-Ads-Rule-Clash.yaml"
    interval: 600

rules:

  - SRC-IP-CIDR,192.168.59.0/24,🚩 PROXY
  - SRC-PORT,7890,🚩 PROXY
  - DST-PORT,22,DIRECT
  - DOMAIN-SUFFIX,apifox.it.com,REJECT
  - AND,(AND,(DST-PORT,443),(NETWORK,UDP)),(NOT,((GEOSITE,cn))),REJECT
  - AND,(AND,(DST-PORT,443),(NETWORK,UDP)),(NOT,((GEOIP,CN))),REJECT
  - RULE-SET,AWAvenue-Ads,REJECT
  - GEOIP,lan,DIRECT,no-resolve

  - GEOSITE,category-ads-all,REJECT
  - GEOSITE,private,DIRECT
  - GEOSITE,icloud,DIRECT
  - GEOSITE,apple,DIRECT
  - GEOSITE,anthropic,🤖 人工智能

  - GEOSITE,category-ai-chat-!cn,🤖 人工智能

  - GEOSITE,youtube,🚩 PROXY
  - GEOSITE,google,🤖 人工智能
  - GEOSITE,github,🚩 PROXY
  - GEOSITE,onedrive,DIRECT
  - GEOSITE,microsoft,DIRECT

  - GEOSITE,CN,DIRECT
  - GEOSITE,steam@cn,DIRECT
  - GEOSITE,category-games@cn,DIRECT
  - GEOSITE,geolocation-!cn,🚩 PROXY
  - GEOSITE,telegram,🇭🇰 HongKong
  - GEOIP,google,🚩 PROXY
  - GEOIP,telegram,🇭🇰 HongKong
  - GEOIP,CN,DIRECT
  - MATCH,🚩 PROXY
`;

const form = document.getElementById("subscribe-form");
const requestHeadersInput = document.getElementById("request-headers-input");
const submitButton = document.getElementById("submit-btn");

const loading = document.getElementById("loading");
const subscriptionsContainer = document.getElementById("subscriptions");
const emptyState = document.getElementById("empty-state");

const templateForm = document.getElementById("template-form");
const templateContentInput = document.getElementById("template-content-input");
const templateSubmitButton = document.getElementById("template-submit-btn");
const templateSetDefaultButton = document.getElementById("template-set-default-btn");
const copyTemplateURLButton = document.getElementById("copy-template-url-btn");
const copyExpandedTemplateURLButton = document.getElementById("copy-expanded-template-url-btn");
const deleteTemplateButton = document.getElementById("delete-template-btn");
const newTemplateButton = document.getElementById("new-template-btn");
const templatesContainer = document.getElementById("templates");
const templateEmptyState = document.getElementById("template-empty-state");
const templateSubscriptionOptions = document.getElementById("template-subscription-options");
const selectAllTemplateSubscriptionsButton = document.getElementById("select-all-template-subscriptions-btn");
const clearTemplateSubscriptionsButton = document.getElementById("clear-template-subscriptions-btn");

const toast = document.getElementById("toast");
const toastMessage = document.getElementById("toast-message");

const editModal = document.getElementById("edit-modal");
const editForm = document.getElementById("edit-form");
const editRequestHeadersInput = document.getElementById("edit-request-headers-input");
const editSubmitButton = document.getElementById("edit-submit-btn");
const editSaveButton = document.getElementById("edit-save-btn");
const closeEditModalButton = document.getElementById("close-edit-modal");
const cancelEditModalButton = document.getElementById("cancel-edit-modal");

let currentSubscriptions = [];
let currentTemplates = [];

form.addEventListener("submit", async (event) => {
  event.preventDefault();

  const payload = buildSubscriptionPayload(form);
  if (!payload) {
    return;
  }

  setSubmitting(submitButton, true, "正在添加...");
  try {
    const response = await fetch(`${API_BASE}/subscribe`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });

    const result = await response.json();
    if (!response.ok || !result.success) {
      throw new Error(result.error || result.message || "添加失败");
    }

    form.reset();
    requestHeadersInput.value = "";
    showToast("订阅已保存");
    await Promise.all([loadSubscriptions(), loadTemplates()]);
  } catch (error) {
    showToast(error.message || "添加失败", "error");
  } finally {
    setSubmitting(submitButton, false, "添加订阅");
  }
});

editForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  await submitEditForm({ refresh: true });
});

editSaveButton.addEventListener("click", async () => {
  await submitEditForm({ refresh: false });
});

templateForm.addEventListener("submit", async (event) => {
  event.preventDefault();

  const payload = buildTemplatePayload();
  if (!payload) {
    return;
  }

  const currentID = templateForm.elements.id.value;
  const isEditing = Boolean(currentID);
  setSubmitting(templateSubmitButton, true, isEditing ? "正在保存..." : "正在创建...");
  try {
    const response = await fetch(
      isEditing ? `${API_BASE}/templates/${encodeURIComponent(currentID)}` : `${API_BASE}/templates`,
      {
        method: isEditing ? "PUT" : "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      },
    );

    const result = await response.json();
    if (!response.ok || !result.success) {
      throw new Error(result.error || result.message || "保存失败");
    }

    showToast(isEditing ? "模板已更新" : "模板已创建");
    await loadTemplates(result.data?.id || currentID);
  } catch (error) {
    showToast(error.message || "保存失败", "error");
  } finally {
    setSubmitting(templateSubmitButton, false, "保存模板");
  }
});

templateSetDefaultButton.addEventListener("click", async () => {
  const currentID = templateForm.elements.id.value;
  if (!currentID) {
    showToast("请先选择一个模板", "error");
    return;
  }

  try {
    const response = await fetch(`${API_BASE}/templates/${encodeURIComponent(currentID)}/default`, {
      method: "POST",
    });
    const result = await response.json();
    if (!response.ok || !result.success) {
      throw new Error(result.error || result.message || "设置默认失败");
    }
    showToast("默认模板已更新");
    await loadTemplates(currentID);
  } catch (error) {
    showToast(error.message || "设置默认失败", "error");
  }
});

deleteTemplateButton.addEventListener("click", async () => {
  const currentID = templateForm.elements.id.value;
  if (!currentID) {
    resetTemplateEditor();
    return;
  }
  if (!window.confirm("确定删除这个模板吗？")) {
    return;
  }

  try {
    const response = await fetch(`${API_BASE}/templates/${encodeURIComponent(currentID)}`, {
      method: "DELETE",
    });
    const result = await response.json();
    if (!response.ok || !result.success) {
      throw new Error(result.error || result.message || "删除失败");
    }
    showToast("模板已删除");
    await loadTemplates();
  } catch (error) {
    showToast(error.message || "删除失败", "error");
  }
});

copyTemplateURLButton.addEventListener("click", async () => {
  const currentID = templateForm.elements.id.value;
  if (!currentID) {
    showToast("请先选择一个模板", "error");
    return;
  }
  await copyAbsoluteURL(`${API_BASE}/templates/${encodeURIComponent(currentID)}/render`, "模板下载地址已复制到剪贴板");
});

copyExpandedTemplateURLButton.addEventListener("click", async () => {
  const currentID = templateForm.elements.id.value;
  if (!currentID) {
    showToast("请先选择一个模板", "error");
    return;
  }
  await copyAbsoluteURL(`${API_BASE}/templates/${encodeURIComponent(currentID)}/render-proxies`, "全节点模板地址已复制到剪贴板");
});

newTemplateButton.addEventListener("click", () => {
  resetTemplateEditor();
});

selectAllTemplateSubscriptionsButton.addEventListener("click", () => {
  renderTemplateSubscriptionOptions(getAllSubscriptionIDs());
});

clearTemplateSubscriptionsButton.addEventListener("click", () => {
  renderTemplateSubscriptionOptions([]);
});

closeEditModalButton.addEventListener("click", closeEditModal);
cancelEditModalButton.addEventListener("click", closeEditModal);
editModal.addEventListener("click", (event) => {
  if (event.target === editModal) {
    closeEditModal();
  }
});

subscriptionsContainer.addEventListener("click", async (event) => {
  const target = event.target.closest("button[data-action]");
  if (!target) {
    return;
  }

  const { action, id } = target.dataset;
  if (!id) {
    return;
  }

  if (action === "copy-download-url") {
    await copyAbsoluteURL(`/download/${encodeURIComponent(id)}`, "下载地址已复制到剪贴板");
    return;
  }

  if (action === "edit-refresh") {
    openEditModal(id);
    return;
  }

  if (action === "delete") {
    if (!window.confirm("确定删除这个订阅吗？")) {
      return;
    }

    try {
      const response = await fetch(`${API_BASE}/subscribe/${encodeURIComponent(id)}`, { method: "DELETE" });
      const result = await response.json();
      if (!response.ok || !result.success) {
        throw new Error(result.error || result.message || "删除失败");
      }
      showToast("订阅已删除");
      await Promise.all([loadSubscriptions(), loadTemplates()]);
    } catch (error) {
      showToast(error.message || "删除失败", "error");
    }
  }
});

templatesContainer.addEventListener("click", async (event) => {
  const target = event.target.closest("button[data-action]");
  if (!target) {
    return;
  }

  const { action, id } = target.dataset;
  if (!id) {
    return;
  }

  if (action === "edit-template") {
    openTemplateEditor(id);
    return;
  }

  if (action === "copy-template-url") {
    await copyAbsoluteURL(`${API_BASE}/templates/${encodeURIComponent(id)}/render`, "模板下载地址已复制到剪贴板");
  }
});

async function loadSubscriptions() {
  loading.hidden = false;
  emptyState.hidden = true;

  try {
    const response = await fetch(`${API_BASE}/subscriptions`);
    const result = await response.json();

    if (!response.ok || !result.success) {
      throw new Error(result.error || result.message || "加载失败");
    }

    currentSubscriptions = result.data || [];
    renderSubscriptions(currentSubscriptions);
    syncTemplateSubscriptionOptions();
  } catch (error) {
    subscriptionsContainer.innerHTML = "";
    emptyState.hidden = false;
    emptyState.innerHTML = `<p>${escapeHtml(error.message || "加载失败")}</p>`;
    showToast(error.message || "加载失败", "error");
  } finally {
    loading.hidden = true;
  }
}

async function loadTemplates(preferredTemplateID) {
  try {
    const response = await fetch(`${API_BASE}/templates`);
    const result = await response.json();
    if (!response.ok || !result.success) {
      throw new Error(result.error || result.message || "加载模板失败");
    }

    currentTemplates = result.data || [];
    renderTemplates(currentTemplates);

    if (!currentTemplates.length) {
      resetTemplateEditor();
      return;
    }

    const activeTemplate =
      currentTemplates.find((item) => item.id === preferredTemplateID) ||
      currentTemplates.find((item) => item.id === templateForm.elements.id.value) ||
      currentTemplates.find((item) => item.is_default) ||
      currentTemplates[0];
    if (activeTemplate) {
      fillTemplateEditor(activeTemplate);
    }
  } catch (error) {
    currentTemplates = [];
    renderTemplates([]);
    resetTemplateEditor();
    showToast(error.message || "加载模板失败", "error");
  }
}

function renderSubscriptions(subscriptions) {
  if (!subscriptions.length) {
    subscriptionsContainer.innerHTML = "";
    emptyState.hidden = false;
    emptyState.innerHTML = "<p>还没有订阅，先添加一个。</p>";
    return;
  }

  emptyState.hidden = true;
  subscriptionsContainer.innerHTML = subscriptions
    .map((subscription) => {
      const headerCount = Object.keys(subscription.request_headers || {}).length;

      return `
        <article class="subscription-card">
          <div class="subscription-head">
            <h3 class="subscription-name">${escapeHtml(subscription.name)}</h3>
            <span class="subscription-type">${escapeHtml(subscription.type || "unknown")}</span>
          </div>
          <div class="subscription-meta">
            <div>${escapeHtml(subscription.url)}</div>
            <div><strong>创建时间：</strong>${formatDate(subscription.created_at)}</div>
            <div><strong>最近更新：</strong>${formatDate(subscription.updated_at)}</div>
            <div><strong>文件大小：</strong>${formatFileSize(subscription.file_size || 0)}</div>
            <div><strong>请求头：</strong>${headerCount} 项</div>
            <div><strong>Filter：</strong>${escapeHtml(subscription.filter || "未设置")}</div>
          </div>
          <div class="subscription-actions">
            <button class="btn btn-secondary" data-action="copy-download-url" data-id="${escapeHtml(subscription.id)}">复制下载地址</button>
            <button class="btn btn-secondary" data-action="edit-refresh" data-id="${escapeHtml(subscription.id)}">编辑</button>
            <button class="btn btn-danger" data-action="delete" data-id="${escapeHtml(subscription.id)}">删除</button>
          </div>
        </article>
      `;
    })
    .join("");
}

function renderTemplates(templates) {
  if (!templates.length) {
    templatesContainer.innerHTML = "";
    templateEmptyState.hidden = false;
    return;
  }

  templateEmptyState.hidden = true;
  templatesContainer.innerHTML = templates
    .map((template) => `
      <article class="template-card ${template.id === templateForm.elements.id.value ? "active" : ""}">
        <div class="template-card-head">
          <h3 class="template-name">${escapeHtml(template.name)}</h3>
          ${template.is_default ? '<span class="template-badge">默认</span>' : ""}
        </div>
        <div class="template-actions">
          <button type="button" class="btn btn-secondary" data-action="edit-template" data-id="${escapeHtml(template.id)}">编辑</button>
          <button type="button" class="btn btn-secondary" data-action="copy-template-url" data-id="${escapeHtml(template.id)}">复制地址</button>
        </div>
      </article>
    `)
    .join("");
}

function openEditModal(id) {
  const subscription = currentSubscriptions.find((item) => item.id === id);
  if (!subscription) {
    showToast("未找到订阅记录", "error");
    return;
  }

  editForm.elements.id.value = subscription.id;
  editForm.elements.name.value = subscription.name || "";
  editForm.elements.url.value = subscription.url || "";
  editForm.elements.type.value = subscription.type || "clash";
  editForm.elements.filter.value = subscription.filter || "";
  editRequestHeadersInput.value = stringifyHeaders(subscription.request_headers || {});
  editModal.classList.remove("hidden");
}

function closeEditModal() {
  editModal.classList.add("hidden");
  editForm.reset();
  editRequestHeadersInput.value = "";
}

function openTemplateEditor(id) {
  const template = currentTemplates.find((item) => item.id === id);
  if (!template) {
    showToast("未找到模板", "error");
    return;
  }
  fillTemplateEditor(template);
  renderTemplates(currentTemplates);
}

function fillTemplateEditor(template) {
  templateForm.elements.id.value = template.id || "";
  templateForm.elements.name.value = template.name || "";
  templateContentInput.value = template.content || DEFAULT_TEMPLATE_CONTENT;
  renderTemplateSubscriptionOptions(resolveTemplateSelectedSubscriptionIDs(template));
  renderTemplates(currentTemplates);
}

function resetTemplateEditor() {
  templateForm.reset();
  templateForm.elements.id.value = "";
  templateContentInput.value = DEFAULT_TEMPLATE_CONTENT;
  renderTemplateSubscriptionOptions(getAllSubscriptionIDs());
  renderTemplates(currentTemplates);
}

function buildSubscriptionPayload(targetForm) {
  const formData = new FormData(targetForm);
  const name = String(formData.get("name") || "").trim();
  const url = String(formData.get("url") || "").trim();
  const filter = String(formData.get("filter") || "").trim();
  const type = String(formData.get("type") || "").trim();
  const requestHeadersText = String(formData.get("request_headers_text") || "");

  if (!name || !url) {
    showToast("请填写订阅名称和地址", "error");
    return null;
  }

  let requestHeaders;
  try {
    requestHeaders = parseHeaders(requestHeadersText);
  } catch (error) {
    showToast(error.message, "error");
    return null;
  }

  return { name, url, filter, type, request_headers: requestHeaders };
}

function buildTemplatePayload() {
  const formData = new FormData(templateForm);
  const name = String(formData.get("name") || "").trim();
  const content = String(formData.get("content") || "").trim();
  const selectedSubscriptionIDs = getCheckedTemplateSubscriptionIDs();
  const allSubscriptionIDs = getAllSubscriptionIDs();

  if (!name) {
    showToast("请填写模板名称", "error");
    return null;
  }
  if (!content) {
    showToast("请填写模板内容", "error");
    return null;
  }

  const useAllSubscriptions =
    allSubscriptionIDs.length === 0 || selectedSubscriptionIDs.length === allSubscriptionIDs.length;

  return {
    name,
    content,
    selected_subscription_ids: useAllSubscriptions ? [] : selectedSubscriptionIDs,
    use_all_subscriptions: useAllSubscriptions,
  };
}

function getAllSubscriptionIDs() {
  return currentSubscriptions.map((subscription) => subscription.id).filter(Boolean);
}

function getCheckedTemplateSubscriptionIDs() {
  return Array.from(
    templateSubscriptionOptions.querySelectorAll('input[name="selected_subscription_ids"]:checked'),
  ).map((input) => input.value);
}

function resolveTemplateSelectedSubscriptionIDs(template) {
  const allSubscriptionIDs = getAllSubscriptionIDs();
  if (!template || template.use_all_subscriptions !== false) {
    return allSubscriptionIDs;
  }

  const existingIDs = new Set(allSubscriptionIDs);
  return (template.selected_subscription_ids || []).filter((id) => existingIDs.has(id));
}

function renderTemplateSubscriptionOptions(selectedSubscriptionIDs) {
  if (!currentSubscriptions.length) {
    templateSubscriptionOptions.innerHTML = '<p class="template-subscription-empty">当前还没有订阅，后续添加后会默认全选。</p>';
    selectAllTemplateSubscriptionsButton.disabled = true;
    clearTemplateSubscriptionsButton.disabled = true;
    return;
  }

  selectAllTemplateSubscriptionsButton.disabled = false;
  clearTemplateSubscriptionsButton.disabled = false;

  const selectedSet = new Set(selectedSubscriptionIDs);
  templateSubscriptionOptions.innerHTML = currentSubscriptions
    .map((subscription) => {
      const isChecked = selectedSet.has(subscription.id) ? "checked" : "";
      return `
        <label class="template-subscription-option">
          <input type="checkbox" name="selected_subscription_ids" value="${escapeHtml(subscription.id)}" ${isChecked}>
          <span class="template-subscription-copy">
            <strong>${escapeHtml(subscription.name || "未命名订阅")}</strong>
            <span>${escapeHtml(subscription.url || "")}</span>
          </span>
        </label>
      `;
    })
    .join("");
}

function syncTemplateSubscriptionOptions() {
  const activeTemplate = currentTemplates.find((item) => item.id === templateForm.elements.id.value);
  if (activeTemplate) {
    renderTemplateSubscriptionOptions(resolveTemplateSelectedSubscriptionIDs(activeTemplate));
    return;
  }

  const hasRenderedOptions = Boolean(
    templateSubscriptionOptions.querySelector('input[name="selected_subscription_ids"]'),
  );
  if (hasRenderedOptions) {
    renderTemplateSubscriptionOptions(getCheckedTemplateSubscriptionIDs());
    return;
  }

  renderTemplateSubscriptionOptions(getAllSubscriptionIDs());
}

async function submitEditForm({ refresh }) {
  const payload = buildSubscriptionPayload(editForm);
  if (!payload) {
    return;
  }

  const subscriptionID = editForm.elements.id.value;
  const loadingText = refresh ? "正在更新..." : "正在保存...";

  setSubmitting(editSaveButton, true, refresh ? "仅保存修改" : loadingText);
  setSubmitting(editSubmitButton, true, refresh ? loadingText : "保存并更新");
  try {
    const response = await fetch(
      refresh
        ? `${API_BASE}/subscribe/${encodeURIComponent(subscriptionID)}/refresh`
        : `${API_BASE}/subscribe/${encodeURIComponent(subscriptionID)}`,
      {
        method: refresh ? "POST" : "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      },
    );

    const result = await response.json();
    if (!response.ok || !result.success) {
      throw new Error(result.error || result.message || (refresh ? "更新失败" : "保存失败"));
    }

    closeEditModal();
    showToast(refresh ? "订阅已更新" : "订阅修改已保存");
    await Promise.all([loadSubscriptions(), loadTemplates()]);
  } catch (error) {
    showToast(error.message || (refresh ? "更新失败" : "保存失败"), "error");
  } finally {
    setSubmitting(editSaveButton, false, "仅保存修改");
    setSubmitting(editSubmitButton, false, "保存并更新");
  }
}

function parseHeaders(text) {
  const headers = {};
  const lines = text.split(/\r?\n/);

  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed) {
      continue;
    }

    const separatorIndex = trimmed.indexOf(":");
    if (separatorIndex <= 0) {
      throw new Error(`请求头格式错误：${trimmed}`);
    }

    const key = trimmed.slice(0, separatorIndex).trim();
    const value = trimmed.slice(separatorIndex + 1).trim();
    if (!key || !value) {
      throw new Error(`请求头格式错误：${trimmed}`);
    }

    headers[key] = value;
  }

  return headers;
}

function stringifyHeaders(headers) {
  return Object.entries(headers)
    .map(([key, value]) => `${key}: ${value}`)
    .join("\n");
}

function setSubmitting(button, isSubmitting, loadingText) {
  button.disabled = isSubmitting;
  if (isSubmitting) {
    button.dataset.originalText = button.textContent;
    button.textContent = loadingText;
    return;
  }

  button.textContent = button.dataset.originalText || button.textContent;
}

async function copyAbsoluteURL(pathname, successMessage) {
  const absoluteURL = new URL(pathname, window.location.origin).toString();

  if (!navigator.clipboard || typeof navigator.clipboard.writeText !== "function") {
    showToast(`当前浏览器不支持自动复制，请手动复制：${absoluteURL}`, "error");
    return;
  }

  try {
    await navigator.clipboard.writeText(absoluteURL);
    showToast(successMessage);
  } catch (error) {
    showToast(`复制失败，请手动复制：${absoluteURL}`, "error");
  }
}

function showToast(message, type = "success") {
  toast.className = `toast ${type}`;
  toastMessage.textContent = message;
  toast.classList.remove("hidden");

  window.clearTimeout(showToast.timer);
  showToast.timer = window.setTimeout(() => {
    toast.className = "toast hidden";
  }, 2400);
}

function formatDate(value) {
  if (!value) {
    return "未知";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "未知";
  }

  return date.toLocaleString("zh-CN");
}

function formatFileSize(bytes) {
  if (!bytes) {
    return "0 B";
  }

  const units = ["B", "KB", "MB", "GB"];
  let size = bytes;
  let unitIndex = 0;
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex += 1;
  }
  return `${size.toFixed(size >= 10 || unitIndex === 0 ? 0 : 1)} ${units[unitIndex]}`;
}

function escapeHtml(value) {
  const node = document.createElement("div");
  node.textContent = value ?? "";
  return node.innerHTML;
}

Promise.all([loadSubscriptions(), loadTemplates()]).then(() => {
  if (!currentTemplates.length) {
    resetTemplateEditor();
  }
});
