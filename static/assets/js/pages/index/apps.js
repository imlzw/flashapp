import { els } from "./elements.js";
import { state } from "./state.js";
import { fetchApps, fetchHealth, deleteApp } from "../../shared/api.js";
import { clearSession } from "../../shared/session.js";
import { updateUserUI, clearChatFeed, createAssistantEntry, setAssistantNote, showSelectedAppMessage } from "./ui.js";

export function resetCurrentSelection() {
  state.currentAppId = "";
  state.currentPreviewURL = "";
  renderApps();
}

export function selectApp(app) {
  state.currentAppId = app ? app.id : "";
  state.currentPreviewURL = app ? app.preview_url || "" : "";
  renderApps();
}

export function renderApps() {
  if (!els.appList) return;
  els.appList.innerHTML = "";
  if (els.appCount) els.appCount.textContent = String(state.apps.length);
  
  if (!state.user) {
      if (els.appEmpty) {
        els.appEmpty.textContent = "登录后查看历史应用";
        els.appEmpty.style.display = "block";
      }
      return;
  }

  if (els.appEmpty) {
    els.appEmpty.style.display = state.apps.length > 0 ? "none" : "block";
    els.appEmpty.textContent = "暂无应用";
  }

  for (const app of state.apps) {
    const item = document.createElement("button");
    item.type = "button";
    item.className = "app-item";
    item.title = app.title || "未命名应用";

    if (app.id === state.currentAppId) {
      item.classList.add("active");
    }

    const icon = document.createElement("div");
    icon.className = "app-item-icon";
    icon.innerHTML = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"></path><line x1="9" y1="10" x2="15" y2="10"></line></svg>`;

    const title = document.createElement("span");
    title.className = "app-item-title";
    title.textContent = app.title || "未命名应用";

    const deleteBtn = document.createElement("button");
    deleteBtn.className = "app-item-delete";
    deleteBtn.innerHTML = `<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>`;
    deleteBtn.title = "删除应用";
    deleteBtn.addEventListener("click", async (e) => {
      e.stopPropagation();
      if (!confirm(`确定要删除应用 "${app.title || "未命名应用"}" 吗？`)) return;
      try {
        await deleteApp(app.id);
        if (state.currentAppId === app.id) {
          resetCurrentSelection();
          clearChatFeed();
        }
        await refreshApps();
      } catch (err) {
        alert("删除失败: " + err.message);
      }
    });

    item.appendChild(icon);
    item.appendChild(title);
    item.appendChild(deleteBtn);
    item.addEventListener("click", () => {
      // switchStage will be imported in main or handled via event
      window.dispatchEvent(new CustomEvent('flashapp:switch-stage', { detail: 'chat' }));
      selectApp(app);
      showSelectedAppMessage(app);
      document.body.classList.remove("sidebar-open");
    });
    els.appList.appendChild(item);
  }
}

export async function loadHealth() {
  try {
    const payload = await fetchHealth();
    if (els.healthMode) els.healthMode.textContent = payload.mock_agent ? "Mock" : "Live";
    if (els.healthModel) els.healthModel.textContent = payload.llm_model || "unknown";
  } catch (error) {
    if (els.healthMode) els.healthMode.textContent = "失败";
    if (els.healthModel) els.healthModel.textContent = error.message;
  }
}

export async function refreshApps() {
  if (!state.user) return;
  try {
    const payload = await fetchApps();
    state.apps = payload.apps || [];
    renderApps();

    if (!state.currentAppId) return;

    const current = state.apps.find((item) => item.id === state.currentAppId);
    if (!current) {
      resetCurrentSelection();
      clearChatFeed();
      return;
    }

    state.currentPreviewURL = current.preview_url || "";
    renderApps();
  } catch (error) {
    if (error.status === 401) {
      performLogout();
      return;
    }
    const entry = createAssistantEntry("读取失败", error.message);
    setAssistantNote(entry, "读取失败", "danger");
  }
}

export function performLogout() {
    clearSession();
    state.user = null;
    state.apps = [];
    state.currentAppId = "";
    updateUserUI();
    renderApps();
    clearChatFeed();
}
