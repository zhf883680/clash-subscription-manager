const API_BASE = "/api";

const form = document.getElementById("subscribe-form");
const requestHeadersInput = document.getElementById("request-headers-input");
const submitButton = document.getElementById("submit-btn");

const loading = document.getElementById("loading");
const subscriptionsContainer = document.getElementById("subscriptions");
const emptyState = document.getElementById("empty-state");

const toast = document.getElementById("toast");
const toastMessage = document.getElementById("toast-message");

const editModal = document.getElementById("edit-modal");
const editForm = document.getElementById("edit-form");
const editRequestHeadersInput = document.getElementById("edit-request-headers-input");
const editSubmitButton = document.getElementById("edit-submit-btn");
const closeEditModalButton = document.getElementById("close-edit-modal");
const cancelEditModalButton = document.getElementById("cancel-edit-modal");

let currentSubscriptions = [];

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
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(payload),
    });

    const result = await response.json();
    if (!response.ok || !result.success) {
      throw new Error(result.error || result.message || "添加失败");
    }

    form.reset();
    requestHeadersInput.value = "";
    showToast("订阅已保存");
    await loadSubscriptions();
  } catch (error) {
    showToast(error.message || "添加失败", "error");
  } finally {
    setSubmitting(submitButton, false, "添加订阅");
  }
});

editForm.addEventListener("submit", async (event) => {
  event.preventDefault();

  const payload = buildSubscriptionPayload(editForm);
  if (!payload) {
    return;
  }

  const subscriptionID = editForm.elements.id.value;
  setSubmitting(editSubmitButton, true, "正在更新...");
  try {
    const response = await fetch(`${API_BASE}/subscribe/${encodeURIComponent(subscriptionID)}/refresh`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(payload),
    });

    const result = await response.json();
    if (!response.ok || !result.success) {
      throw new Error(result.error || result.message || "更新失败");
    }

    closeEditModal();
    showToast("订阅已更新");
    await loadSubscriptions();
  } catch (error) {
    showToast(error.message || "更新失败", "error");
  } finally {
    setSubmitting(editSubmitButton, false, "保存并更新");
  }
});

closeEditModalButton.addEventListener("click", closeEditModal);
cancelEditModalButton.addEventListener("click", closeEditModal);
editModal.addEventListener("click", (event) => {
  if (event.target === editModal) {
    closeEditModal();
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
  } catch (error) {
    subscriptionsContainer.innerHTML = "";
    emptyState.hidden = false;
    emptyState.innerHTML = `<p>${escapeHtml(error.message || "加载失败")}</p>`;
    showToast(error.message || "加载失败", "error");
  } finally {
    loading.hidden = true;
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
          </div>
          <div class="subscription-actions">
            <button class="btn btn-secondary" data-action="copy-download-url" data-id="${escapeHtml(subscription.id)}">复制下载地址</button>
            <button class="btn btn-secondary" data-action="edit-refresh" data-id="${escapeHtml(subscription.id)}">更新</button>
            <button class="btn btn-danger" data-action="delete" data-id="${escapeHtml(subscription.id)}">删除</button>
          </div>
        </article>
      `;
    })
    .join("");
}

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
    await copyDownloadURL(id);
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
      const response = await fetch(`${API_BASE}/subscribe/${encodeURIComponent(id)}`, {
        method: "DELETE",
      });
      const result = await response.json();
      if (!response.ok || !result.success) {
        throw new Error(result.error || result.message || "删除失败");
      }
      showToast("订阅已删除");
      await loadSubscriptions();
    } catch (error) {
      showToast(error.message || "删除失败", "error");
    }
  }
});

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
  editRequestHeadersInput.value = stringifyHeaders(subscription.request_headers || {});

  editModal.classList.remove("hidden");
}

function closeEditModal() {
  editModal.classList.add("hidden");
  editForm.reset();
  editRequestHeadersInput.value = "";
}

function buildSubscriptionPayload(targetForm) {
  const formData = new FormData(targetForm);
  const name = String(formData.get("name") || "").trim();
  const url = String(formData.get("url") || "").trim();
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

  return {
    name,
    url,
    type,
    request_headers: requestHeaders,
  };
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

async function copyDownloadURL(id) {
  const path = `/download/${encodeURIComponent(id)}`;
  const absoluteURL = new URL(path, window.location.origin).toString();

  if (!navigator.clipboard || typeof navigator.clipboard.writeText !== "function") {
    showToast(`当前浏览器不支持自动复制，请手动复制：${absoluteURL}`, "error");
    return;
  }

  try {
    await navigator.clipboard.writeText(absoluteURL);
    showToast("下载地址已复制到剪贴板");
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

loadSubscriptions();
