import { els } from "./elements.js";
import { state } from "./state.js";
import { fetchPlaza, fetchMyPublishedApps, togglePublic, deletePublishedApp, publishApp } from "../../shared/api.js";
import { getSession } from "../../shared/session.js";

const PHONE_SHELL_PRESETS = {
  mobile: { width: 360, height: 740 },
  tablet: { width: 720, height: 960 }
};

function getPhoneShellPreset(mode = "mobile") {
  return PHONE_SHELL_PRESETS[mode] || PHONE_SHELL_PRESETS.mobile;
}

function normalizePhoneScreenshot(sourceCanvas, mode = "mobile") {
  if (!sourceCanvas || !sourceCanvas.width || !sourceCanvas.height) return "";

  const preset = getPhoneShellPreset(mode);
  const targetRatio = preset.width / preset.height;
  const sourceWidth = sourceCanvas.width;
  const sourceHeight = sourceCanvas.height;
  const sourceRatio = sourceWidth / sourceHeight;
  let cropWidth = sourceWidth;
  let cropHeight = sourceHeight;
  let offsetX = 0;
  let offsetY = 0;

  if (sourceRatio > targetRatio) {
    cropWidth = Math.round(sourceHeight * targetRatio);
    offsetX = Math.max(0, Math.floor((sourceWidth - cropWidth) / 2));
  } else if (sourceRatio < targetRatio) {
    cropHeight = Math.round(sourceWidth / targetRatio);
  }

  const outputCanvas = document.createElement("canvas");
  outputCanvas.width = cropWidth;
  outputCanvas.height = cropHeight;

  const ctx = outputCanvas.getContext("2d");
  if (!ctx) return sourceCanvas.toDataURL("image/png");

  ctx.drawImage(
    sourceCanvas,
    offsetX,
    offsetY,
    cropWidth,
    cropHeight,
    0,
    0,
    outputCanvas.width,
    outputCanvas.height
  );

  return outputCanvas.toDataURL("image/png");
}

export function switchStage(stageName) {
  if (els.chatStage) els.chatStage.hidden = (stageName !== "chat");
  if (els.plazaStage) els.plazaStage.hidden = (stageName !== "plaza");
  if (els.settingsStage) els.settingsStage.hidden = (stageName !== "settings");
  
  const header = document.querySelector(".stage-header");
  if (header) {
    if (stageName === "chat") {
      header.classList.add("header-chat-mode");
    } else {
      header.classList.remove("header-chat-mode");
    }
  }

  if (els.headerLeft) {
    const shouldShowHeader = (stageName === "plaza" || stageName === "settings");
    els.headerLeft.style.display = shouldShowHeader ? "flex" : "none";
    if (stageName === "plaza") {
      if (els.plazaTitle) els.plazaTitle.textContent = "应用广场";
      if (els.plazaSubtitle) els.plazaSubtitle.textContent = "发现、复制并重混由社区创建的闪应用";
    } else if (stageName === "settings") {
      if (els.plazaTitle) els.plazaTitle.textContent = "用户设置";
      if (els.plazaSubtitle) els.plazaSubtitle.textContent = "管理你的基本信息与安全配置";
    }
  }
  
  if (els.plazaSearchContainer) {
    els.plazaSearchContainer.style.display = (stageName === "plaza") ? "flex" : "none";
    if (stageName !== "plaza" && els.plazaSearchInput) {
        els.plazaSearchInput.value = "";
    }
  }

  if (els.newApp) els.newApp.classList.remove("is-active");
  if (els.navMyApps) els.navMyApps.classList.remove("is-active");
  if (els.navAppSquare) els.navAppSquare.classList.remove("is-active");
}

export function formatDate(dateStr) {
  if (!dateStr) return "";
  const date = new Date(dateStr);
  const now = new Date();
  const diff = (now - date) / 1000;

  if (diff < 60) return "刚刚";
  if (diff < 3600) return Math.floor(diff / 60) + "分钟前";
  if (diff < 84600) return Math.floor(diff / 3600) + "小时前";
  if (diff < 86400 * 30) return Math.floor(diff / 86400) + "天前";

  return date.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' });
}

export function setChatEmpty(hidden) {
  if (els.chatEmpty) els.chatEmpty.hidden = hidden;
  if (els.chatWatermark) {
    els.chatWatermark.hidden = !hidden;
  }
}

export function scrollChatToBottom() {
  if (!els.chatScroller) return;
  requestAnimationFrame(() => {
    els.chatScroller.scrollTop = els.chatScroller.scrollHeight;
  });
}

export function clearChatFeed() {
  if (els.chatFeed) els.chatFeed.innerHTML = "";
  if (els.chatScroller) els.chatScroller.scrollTop = 0;
  setChatEmpty(false);
}

export function createMessage(kind) {
  const article = document.createElement("article");
  article.className = "message message-" + kind;
  return article;
}

export function createNote(text, tone = "") {
  const note = document.createElement("div");
  note.className = "message-note";
  note.textContent = text;
  note.dataset.tone = tone;
  return note;
}

export function setAssistantNote(entry, text, tone = "") {
  let prefix = "";
  if (tone === "success") {
    prefix = "✨ ";
  } else if (tone === "danger") {
    prefix = "⚠️ ";
  } else if (tone === "warning") {
    prefix = `<span class="icon-spin">✦</span> `;
  } else if (!tone) {
    prefix = "✦ ";
  }

  entry.note.innerHTML = prefix + text;
  entry.note.dataset.tone = tone;
}

export function setBusy(busy) {
  state.isStreaming = busy;
  if (els.sendBtn) {
    els.sendBtn.disabled = busy;
    els.sendBtn.innerHTML = busy 
      ? '<span class="icon-spin">✦</span>' 
      : '<i class="icon-send"></i>';
  }
  if (els.newApp) els.newApp.disabled = busy;
}

export function updateUserUI() {
    const session = getSession();
    state.user = (session.token && session.user) ? session.user : null;

    if (els.userPanel) {
        els.userPanel.hidden = true;
        els.userPanel.style.display = "none";
    }
    if (els.userPanelUnlogged) {
        els.userPanelUnlogged.hidden = true;
        els.userPanelUnlogged.style.display = "none";
    }

    if (state.user) {
        if (els.userPanel) {
            els.userPanel.hidden = false;
            els.userPanel.style.display = "flex";
        }
        const displayName = state.user.nickname || state.user.username || "用户";
        if (els.accountName) els.accountName.textContent = displayName;
        
        const avatar = els.userPanel.querySelector('.user-avatar');
        if (avatar) {
            avatar.textContent = Array.from(displayName)[0].toUpperCase();
        }
    } else {
        if (els.userPanelUnlogged) {
            els.userPanelUnlogged.hidden = false;
            els.userPanelUnlogged.style.display = "flex";
        }
    }
}

export function createPhonePreview(options = {}) {
  const wrapper = document.createElement("div");
  wrapper.className = "inline-phone";
  wrapper.dataset.mode = "mobile";
  wrapper.hidden = true;

  const controls = document.createElement("div");
  controls.className = "preview-controls";
  if (options.hideToolbar) controls.style.display = "none";

  const selectors = document.createElement("div");
  selectors.className = "device-selectors";

  const createModeBtn = (mode, iconSvg) => {
    const btn = document.createElement("button");
    btn.type = "button";
    btn.className = "control-btn" + (mode === "mobile" ? " active" : "");
    btn.innerHTML = iconSvg;
    btn.title = mode.charAt(0).toUpperCase() + mode.slice(1);
    btn.onclick = (e) => {
      e.preventDefault();
      e.stopPropagation();
      wrapper.setAttribute("data-mode", mode);
      selectors.querySelectorAll(".control-btn").forEach(b => b.classList.remove("active"));
      btn.classList.add("active");
    };
    return btn;
  };

  const mobileBtn = createModeBtn("mobile", '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="5" y="2" width="14" height="20" rx="2" ry="2"></rect><line x1="12" y1="18" x2="12" y2="18"></line></svg>');
  const tabletBtn = createModeBtn("tablet", '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="4" y="2" width="16" height="20" rx="2" ry="2"></rect><line x1="12" y1="18" x2="12" y2="18"></line></svg>');
  selectors.append(mobileBtn, tabletBtn);

  const actions = document.createElement("div");
  actions.className = "preview-actions";
  actions.style.display = "flex";
  actions.style.gap = "8px";

  const fullScreenBtn = document.createElement("button");
  fullScreenBtn.className = "control-btn";
  fullScreenBtn.innerHTML = '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M15 3h6v6M9 21H3v-6M21 3l-7 7M3 21l7-7"></path></svg>';
  fullScreenBtn.title = "全屏预览";
  if (options.hideFullScreen) fullScreenBtn.style.display = "none";
  fullScreenBtn.onclick = () => {
    const isFullscreen = wrapper.classList.toggle("fullscreen");
    document.body.classList.toggle("preview-fullscreen-active", isFullscreen);
    fullScreenBtn.innerHTML = isFullscreen 
      ? '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 14h6v6M20 10h-6V4M14 10l7-7M10 14l-7 7"></path></svg>'
      : '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M15 3h6v6M9 21H3v-6M21 3l-7 7M3 21l7-7"></path></svg>';
    fullScreenBtn.title = isFullscreen ? "退出全屏" : "全屏预览";
  };

  const publishBtn = document.createElement("button");
  publishBtn.className = "control-btn";
  publishBtn.innerHTML = '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path><polyline points="17 8 12 3 7 8"></polyline><line x1="12" y1="3" x2="12" y2="15"></line></svg>';
  publishBtn.title = "发布应用";
  publishBtn.onclick = async () => {
    if (!state.currentAppId) return alert("没有可以发布的应用");
    try {
      publishBtn.disabled = true;
      let screenshot = "";
      try {
        const h2c = window.html2canvas;
        if (h2c && frame && frame.contentDocument && frame.contentDocument.body) {
          const deviceMode = wrapper.dataset.mode || "mobile";
          const canvas = await h2c(frame.contentDocument.body, { scale: 2, useCORS: true });
          screenshot = normalizePhoneScreenshot(canvas, deviceMode);
        } else if (!h2c) {
          console.warn("html2canvas library not loaded yet.");
        }
      } catch (screenshotErr) { console.warn("Screenshot capture failed:", screenshotErr); }

      const currentApp = state.apps.find(a => a.id === state.currentAppId);
      
      let initialTitle = currentApp ? currentApp.title : "";
      let initialPrompt = currentApp ? currentApp.prompt : "";

      if (currentApp && currentApp.is_published) {
        if (currentApp.published_title) initialTitle = currentApp.published_title;
        if (currentApp.published_prompt) initialPrompt = currentApp.published_prompt;
      }
      
      if (state.publishModal) {
        state.publishModal.show({
          appId: state.currentAppId,
          title: initialTitle,
          prompt: initialPrompt,
          screenshot: screenshot,
          deviceMode: wrapper.dataset.mode || "mobile"
        });
      } else {
        // Fallback if modal not initialized
        await publishApp(state.currentAppId, screenshot);
        alert("发布成功！");
        window.dispatchEvent(new CustomEvent('flashapp:refresh-apps'));
      }
    } catch (err) { alert("操作失败: " + err.message); } finally { publishBtn.disabled = false; }
  };

  const externalBtn = document.createElement("button");
  externalBtn.className = "control-btn";
  externalBtn.innerHTML = '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"></path><polyline points="15 3 21 3 21 9"></polyline><line x1="10" y1="14" x2="21" y2="3"></line></svg>';
  externalBtn.title = "新窗口打开";
  
  actions.append(fullScreenBtn, externalBtn);
  if (options.showPublish) actions.append(publishBtn);
  controls.append(selectors, actions);

  const shell = document.createElement("div");
  shell.className = "phone-shell";
  const powerBtn = document.createElement("span");
  powerBtn.className = "phone-btn-side phone-btn-power";
  const volumeBtn = document.createElement("span");
  volumeBtn.className = "phone-btn-side phone-btn-volume";
  const screen = document.createElement("div");
  screen.className = "phone-screen";

  const frame = document.createElement("iframe");
  frame.className = "inline-preview-frame";
  frame.title = "预览窗口";
  frame.setAttribute("sandbox", "allow-scripts allow-same-origin");

  const injectScrollbar = () => {
    try {
      const doc = frame.contentDocument || frame.contentWindow.document;
      if (!doc) return;
      const styleId = "flashapp-scrollbar-fix";
      if (doc.getElementById(styleId)) return;
      const style = doc.createElement("style");
      style.id = styleId;
      style.textContent = `::-webkit-scrollbar { display: none !important; width: 0 !important; height: 0 !important; } html { scrollbar-width: none !important; -ms-overflow-style: none !important; }`;
      doc.head.appendChild(style);
      const script = doc.createElement("script");
      script.id = "flashapp-drag-scroll";
      script.textContent = `(function(){let isDown=false,startY,scrollTop,isDragging=false;const getClientY=(e)=>e.clientY||(e.touches&&e.touches.length>0?e.touches[0].clientY:0);const start=(e)=>{isDown=true;isDragging=false;startY=getClientY(e);scrollTop=document.documentElement.scrollTop||document.body.scrollTop;document.body.style.cursor='grabbing';document.body.style.userSelect='none';};const end=()=>{isDown=false;document.body.style.cursor='';document.body.style.userSelect='';};const move=(e)=>{if(!isDown)return;const y=getClientY(e);const walk=(y-startY)*1.5;if(Math.abs(walk)>3){isDragging=true;e.preventDefault();}window.scrollTo(0,scrollTop-walk);};document.addEventListener('mousedown',start);document.addEventListener('touchstart',start,{passive:false});document.addEventListener('mouseleave',end);document.addEventListener('mouseup',end);document.addEventListener('touchend',end);document.addEventListener('mousemove',move,{passive:false});document.addEventListener('touchmove',move,{passive:false});document.addEventListener('click',(e)=>{if(isDragging){e.preventDefault();e.stopPropagation();}},true);})();`;
      doc.head.appendChild(script);
    } catch (e) {}
  };
  frame.onload = injectScrollbar;

  screen.appendChild(frame);
  shell.append(powerBtn, volumeBtn, screen);
  wrapper.append(controls, shell);

  let lastUrl = "", lastHtml = "";
  externalBtn.onclick = () => {
    if (lastUrl) window.open(lastUrl, "_blank");
    else if (lastHtml) {
      const win = window.open("", "_blank");
      win.document.write(lastHtml);
      win.document.close();
    }
  };

  const updateRefs = ({ html, src }) => {
    if (src) lastUrl = src;
    if (html) lastHtml = html;
  };

  return { wrapper, shell, screen, frame, updateRefs };
}

export function showPhonePreview(entry, { html = "", src = "" }) {
  if (!html && !src) return;
  if (entry.updateRefs) entry.updateRefs({ html, src });
  const wrapper = entry.previewWrapper || entry.wrapper;
  const frame = entry.previewFrame || entry.frame;
  if (wrapper) wrapper.hidden = false;
  if (src) {
    if (frame) { frame.removeAttribute("srcdoc"); frame.src = src; }
    return;
  }
  if (frame) frame.removeAttribute("src");
  
  const scrollbarStyle = `<style>::-webkit-scrollbar { display: none !important; width: 0 !important; height: 0 !important; } html { scrollbar-width: none !important; -ms-overflow-style: none !important; }</style><script>(function(){let isDown=false,startY,scrollTop,isDragging=false;const getClientY=(e)=>e.clientY||(e.touches&&e.touches.length>0?e.touches[0].clientY:0);const start=(e)=>{isDown=true;isDragging=false;startY=getClientY(e);scrollTop=document.documentElement.scrollTop||document.body.scrollTop;document.body.style.cursor='grabbing';document.body.style.userSelect='none';};const end=()=>{isDown=false;document.body.style.cursor='';document.body.style.userSelect='';};const move=(e)=>{if(!isDown)return;const y=getClientY(e);const walk=(y-startY)*1.5;if(Math.abs(walk)>3){isDragging=true;e.preventDefault();}window.scrollTo(0,scrollTop-walk);};document.addEventListener('mousedown',start);document.addEventListener('touchstart',start,{passive:false});document.addEventListener('mouseleave',end);document.addEventListener('mouseup',end);document.addEventListener('touchend',end);document.addEventListener('mousemove',move,{passive:false});document.addEventListener('touchmove',move,{passive:false});document.addEventListener('click',(e)=>{if(isDragging){e.preventDefault();e.stopPropagation();}},true);})();</script>`;
  const finalHtml = html.includes("</head>") ? html.replace("</head>", scrollbarStyle + "</head>") : scrollbarStyle + html;
  if (frame) frame.srcdoc = finalHtml;
}

export function showGlobalPreview(app) {
  if (!els.previewModal || !els.previewModalBody) return;
  els.previewModalBody.innerHTML = "";
  const preview = createPhonePreview({ hideFullScreen: true });
  els.previewModalBody.appendChild(preview.wrapper);
  els.previewModal.hidden = false;
  document.body.classList.add("modal-open");
  preview.wrapper.hidden = false;
  if (app.preview_url) showPhonePreview(preview, { src: app.preview_url + "?ts=" + Date.now() });
}

export function appendUserMessage(text) {
  setChatEmpty(true);
  const article = createMessage("user");
  const bubble = document.createElement("div");
  bubble.className = "message-bubble message-bubble-user";
  const content = document.createElement("p");
  content.className = "message-text";
  content.textContent = text;
  bubble.appendChild(content);
  article.appendChild(bubble);
  if (els.chatFeed) els.chatFeed.appendChild(article);
  scrollChatToBottom();
}

export function createAssistantEntry(noteText, bodyText, shouldScroll = true) {
  setChatEmpty(true);
  const article = createMessage("assistant");
  const note = createNote(noteText);
  const bubble = document.createElement("div");
  bubble.className = "message-bubble message-bubble-assistant";
  const content = document.createElement("p");
  content.className = "message-text";
  content.innerHTML = bodyText;
  const preview = createPhonePreview({ showPublish: true });
  bubble.appendChild(content);
  article.append(note, bubble, preview.wrapper);
  if (els.chatFeed) els.chatFeed.appendChild(article);
  if (shouldScroll) scrollChatToBottom();
  return { note, content, previewWrapper: preview.wrapper, previewFrame: preview.frame, updateRefs: preview.updateRefs };
}

export function showSelectedAppMessage(app) {
  clearChatFeed();
  if (!app) return;

  const entry = createAssistantEntry(
    "已切换到当前应用",
    "继续发送需求即可基于这个应用继续修改。",
    false
  );

  if (app.preview_url) {
    showPhonePreview(entry, { src: app.preview_url + "?ts=" + Date.now() });
  }
}
