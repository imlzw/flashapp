import { initTheme, toggleTheme } from "../shared/theme.js";
import { createAuthModal } from "../shared/auth-modal.js";
import { createPublishModal } from "../shared/publish-modal.js";
import { els } from "./index/elements.js";
import { state } from "./index/state.js";
import { 
  switchStage, 
  updateUserUI, 
  clearChatFeed,
  showSelectedAppMessage
} from "./index/ui.js";
import { 
  loadHealth, 
  refreshApps, 
  performLogout, 
  selectApp,
  resetCurrentSelection 
} from "./index/apps.js";
import { renderPlaza, renderMyApps } from "./index/plaza.js";
import { sendPrompt } from "./index/chat.js";
import { initSettings } from "./index/settings.js";

function bindEvents() {
  if (els.logoutBtn) els.logoutBtn.addEventListener("click", performLogout);

  const toggleSidebar = () => document.body.classList.toggle("sidebar-open");
  const closeSidebar = () => document.body.classList.remove("sidebar-open");

  if (els.sidebarToggle) els.sidebarToggle.addEventListener("click", toggleSidebar);
  if (els.sidebarCloseBtn) els.sidebarCloseBtn.addEventListener("click", closeSidebar);
  if (els.sidebarOverlay) els.sidebarOverlay.addEventListener("click", closeSidebar);

  if (els.newApp) {
    els.newApp.addEventListener("click", () => {
        switchStage("chat");
        if (els.newApp) els.newApp.classList.add("is-active");
        resetCurrentSelection();
        clearChatFeed();
        if (els.prompt) els.prompt.value = "";
        closeSidebar();
    });
  }
  
  if (els.navMyApps) els.navMyApps.addEventListener("click", () => {
      renderMyApps();
      closeSidebar();
  });
  if (els.navAppSquare) els.navAppSquare.addEventListener("click", () => {
      renderPlaza();
      closeSidebar();
  });

  initSettings();

  if (els.sendBtn) els.sendBtn.addEventListener("click", sendPrompt);
  if (els.prompt) {
    els.prompt.addEventListener("keydown", (event) => {
        if ((event.ctrlKey || event.metaKey) && event.key === "Enter") {
        sendPrompt();
        }
    });
  }

  const triggerAuth = () => state.authModal.show();
  if (els.sidebarLoginBtn) els.sidebarLoginBtn.addEventListener("click", triggerAuth);
  
  if (els.sidebarThemeBtn) els.sidebarThemeBtn.addEventListener("click", toggleTheme);

  // Suggestion chips
  document.querySelectorAll('.suggestion-chip').forEach(chip => {
      chip.addEventListener('click', () => {
          if (els.prompt) {
            els.prompt.value = chip.textContent.replace('✦ ', '');
            sendPrompt();
          }
      });
  });

  // Global Preview Modal close logic
  if (els.closePreviewModal) {
    els.closePreviewModal.addEventListener("click", () => {
      if (els.previewModal) els.previewModal.hidden = true;
      document.body.classList.remove("modal-open");
      if (els.previewModalBody) els.previewModalBody.innerHTML = "";
    });
  }

  if (els.previewModal) {
    const backdrop = els.previewModal.querySelector(".modal-backdrop");
    if (backdrop) {
      backdrop.addEventListener("click", () => {
        els.previewModal.hidden = true;
        document.body.classList.remove("modal-open");
        if (els.previewModalBody) els.previewModalBody.innerHTML = "";
      });
    }
  }

  // Handle custom events for cross-module communication
  window.addEventListener('flashapp:refresh-apps', refreshApps);
  window.addEventListener('flashapp:switch-stage', (e) => switchStage(e.detail));
  window.addEventListener('flashapp:show-app-msg', (e) => showSelectedAppMessage(e.detail));
}

async function bootstrap() {
  try {
    initTheme();
    
    state.authModal = createAuthModal({
        onLoginSuccess: () => {
            updateUserUI();
            loadHealth();
            refreshApps();
        }
    });

    state.publishModal = createPublishModal({
        onPublishSuccess: () => {
            alert("发布成功！");
            refreshApps();
        }
    });

    updateUserUI();
    bindEvents();
    clearChatFeed();
    
    if (state.user) {
        await loadHealth();
        await refreshApps();
    }

    // Initialize stage
    switchStage("chat");
  } catch (err) {
      console.error("Bootstrap error:", err);
  }
}

bootstrap();
