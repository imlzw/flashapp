import { els } from "./elements.js";
import { state } from "./state.js";
import { fetchPlaza, fetchMyPublishedApps, togglePublic, deletePublishedApp, forkApp } from "../../shared/api.js";
import { switchStage, formatDate, showGlobalPreview, clearChatFeed } from "./ui.js";
import { selectApp, refreshApps, performLogout } from "./apps.js";

let currentApps = [];
let isCurrentPlaza = true;

if (els.plazaSearchInput) {
  els.plazaSearchInput.addEventListener("input", renderFilteredCards);
}

export function renderFilteredCards() {
  const keyword = (els.plazaSearchInput ? els.plazaSearchInput.value.trim().toLowerCase() : "");
  
  if (!keyword) {
    renderCards(currentApps, isCurrentPlaza);
    return;
  }
  
  const filtered = currentApps.filter(app => {
    const title = (app.title || "未命名应用").toLowerCase();
    const prompt = (app.prompt || "").toLowerCase();
    return title.includes(keyword) || prompt.includes(keyword);
  });
  
  renderCards(filtered, isCurrentPlaza);
}

export async function renderPlaza() {
  switchStage("plaza");
  if (els.navAppSquare) els.navAppSquare.classList.add("is-active");
  if (els.plazaTitle) els.plazaTitle.textContent = "应用广场";
  if (els.plazaSubtitle) els.plazaSubtitle.textContent = "发现、复制并重混由社区创建的闪应用";
  if (els.plazaGrid) els.plazaGrid.innerHTML = "加载中...";
  
  try {
    const data = await fetchPlaza();
    currentApps = data.apps || [];
    isCurrentPlaza = true;
    renderFilteredCards();
  } catch (err) {
    if (els.plazaGrid) els.plazaGrid.innerHTML = "加载失败: " + err.message;
  }
}

export async function renderMyApps() {
  if (!state.user) {
    if (state.authModal) state.authModal.show();
    return;
  }

  switchStage("plaza");
  if (els.navMyApps) els.navMyApps.classList.add("is-active");
  if (els.plazaTitle) els.plazaTitle.textContent = "我的闪应用";
  if (els.plazaSubtitle) els.plazaSubtitle.textContent = "我已发布的应用";
  if (els.plazaGrid) els.plazaGrid.innerHTML = "加载中...";
  
  try {
    const data = await fetchMyPublishedApps();
    currentApps = data.apps || [];
    isCurrentPlaza = false;
    renderFilteredCards();
  } catch (err) {
    if (err.status === 401) {
        performLogout();
        if (state.authModal) state.authModal.show();
        return;
    }
    if (els.plazaGrid) els.plazaGrid.innerHTML = "加载失败: " + err.message;
  }
}

export function renderCards(apps, isPlaza) {
  if (!els.plazaGrid) return;
  els.plazaGrid.innerHTML = "";
  if (apps.length === 0) {
    els.plazaGrid.innerHTML = "<div class='empty'>暂无应用</div>";
    return;
  }

  const iconPlay = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="5 3 19 12 5 21 5 3"></polygon></svg>`;
  const iconRemix = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2L2 7l10 5 10-5-10-5z"></path><path d="M2 17l10 5 10-5"></path><path d="M2 12l10 5 10-5"></path></svg>`;
  const iconEdit = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z"></path><path d="m15 5 4 4"></path></svg>`;
  const iconPublic = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><line x1="2" y1="12" x2="22" y2="12"></line><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"></path></svg>`;
  const iconPrivate = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect><path d="M7 11V7a5 5 0 0 1 10 0v4"></path></svg>`;
  const iconDelete = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>`;

    for (const app of apps) {
      const card = document.createElement("div");
      card.className = "plaza-card";
      const isPublic = app.is_public;
      const publicIcon = isPublic ? iconPublic : iconPrivate;

      card.innerHTML = `
        <div class="plaza-card-visual-wrapper">
          <div class="phone-shell-scaled-container">
            <div class="phone-shell phone-shell-scaled">
              <span class="phone-btn-side phone-btn-power"></span>
              <span class="phone-btn-side phone-btn-volume"></span>
              <div class="phone-screen">
                <img class="plaza-card-cover" src="${app.screenshot_url || '/assets/img/logo.svg'}" alt="Preview">
              </div>
            </div>
          </div>
        </div>
        <div class="plaza-card-content">
          <div class="plaza-card-title-row">
            <h3 class="plaza-card-title">${app.title || "未命名应用"}</h3>
            <span class="plaza-card-version">v${app.version || 1}</span>
          </div>
          <p class="plaza-card-prompt">${app.prompt || ""}</p>
          <div class="plaza-card-meta">
            <span class="plaza-card-author">作者: ${app.author_name || "匿名"}</span>
            <span class="plaza-card-date">${formatDate(app.created_at)}</span>
          </div>
          <div class="plaza-card-actions">
             ${isPlaza ? `
               <button class="plaza-btn preview-btn" title="运行应用">${iconPlay}</button>
               <button class="plaza-btn fork-btn" title="重混应用">${iconRemix}</button>
             ` : `
               <button class="plaza-btn preview-btn" title="运行应用">${iconPlay}</button>
               <button class="plaza-btn edit-btn" title="继续编辑">${iconEdit}</button>
               <button class="plaza-btn toggle-public-btn ${isPublic ? 'active' : ''}" title="${isPublic ? '设为私密' : '设为公开'}">${publicIcon}</button>
               <button class="plaza-btn delete-card-btn" title="下架并删除发布记录">${iconDelete}</button>
             `}
          </div>
        </div>
      `;
      const previewBtn = card.querySelector(".preview-btn");
      const forkBtn = card.querySelector(".fork-btn");
      const editBtn = card.querySelector(".edit-btn");
      const togglePublicBtn = card.querySelector(".toggle-public-btn");
      const deleteCardBtn = card.querySelector(".delete-card-btn");
      const visualWrapper = card.querySelector(".plaza-card-visual-wrapper");

      if (visualWrapper) visualWrapper.onclick = () => showGlobalPreview(app);

      previewBtn.onclick = () => showGlobalPreview(app);

      if (editBtn) {
        editBtn.onclick = async () => {
          const originalId = app.original_app_id || app.id;
          const targetApp = state.apps.find(a => a.id === originalId);
          if (targetApp) {
            switchStage("chat");
            selectApp(targetApp);
            // showSelectedAppMessage will be imported or handled via event
            window.dispatchEvent(new CustomEvent('flashapp:show-app-msg', { detail: targetApp }));
          } else {
            try {
              editBtn.disabled = true;
              const originalBtnContent = editBtn.innerHTML;
              editBtn.innerHTML = `<span class="icon-spin">✦</span>`;
              const res = await forkApp(app.id);
              await refreshApps();
              const newApp = state.apps.find(a => a.id === res.id);
              switchStage("chat");
              selectApp(newApp || { id: res.id, preview_url: res.preview_url });
              window.dispatchEvent(new CustomEvent('flashapp:show-app-msg', { detail: newApp || { id: res.id, preview_url: res.preview_url } }));
              editBtn.innerHTML = originalBtnContent;
            } catch (err) { alert("无法恢复该应用的编辑会话: " + err.message); } finally { editBtn.disabled = false; }
          }
        };
      }

      if (forkBtn) {
        forkBtn.onclick = async () => {
          try {
            if (!state.user) return state.authModal && state.authModal.show();
            forkBtn.disabled = true;
            forkBtn.innerHTML = "正在复制...";
            const res = await forkApp(app.id);
            await refreshApps();
            selectApp({ id: res.id, preview_url: res.preview_url });
            switchStage("chat");
          } catch (err) { alert("复制失败: " + err.message); forkBtn.disabled = false; forkBtn.innerHTML = `${iconRemix} 重混应用`; }
        };
      }

      if (togglePublicBtn) {
        togglePublicBtn.onclick = async () => {
          try {
            const isMakingPublic = !app.is_public;
            if (isMakingPublic) {
              if (!confirm("确定要将此应用设为公开吗？公开后，其他用户将可以在应用广场查看并重混该应用。")) {
                return;
              }
            }

            togglePublicBtn.disabled = true;
            const res = await togglePublic(app.original_app_id || app.id);
            
            if (res.is_public) {
              alert("应用已设为公开");
            } else {
              alert("应用已设为私密");
            }
            
            await renderMyApps();
          } catch (err) { alert("操作失败: " + err.message); } finally { togglePublicBtn.disabled = false; }
        };
      }

      if (deleteCardBtn) {
        deleteCardBtn.onclick = async () => {
          if (!confirm("确定要删除此发布记录吗？(这不会删除您的原始会话)")) return;
          try {
            deleteCardBtn.disabled = true;
            await deletePublishedApp(app.original_app_id || app.id);
            await renderMyApps();
          } catch (err) { alert("删除失败: " + err.message); } finally { deleteCardBtn.disabled = false; }
        };
      }
      els.plazaGrid.appendChild(card);
    }
}
