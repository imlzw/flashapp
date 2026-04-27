import { els } from "./elements.js";
import { state } from "./state.js";
import { authHeaders } from "../../shared/api.js";
import { 
  appendUserMessage, 
  createAssistantEntry, 
  setAssistantNote, 
  showPhonePreview, 
  setBusy, 
  scrollChatToBottom,
  clearChatFeed
} from "./ui.js";
import { refreshApps, performLogout, selectApp } from "./apps.js";

export function buildRequest(prompt) {
  if (state.currentAppId) {
    return {
      endpoint: "/api/update",
      payload: { app_id: state.currentAppId, title: "", prompt },
      successText: "应用更新成功"
    };
  }
  return {
    endpoint: "/api/create",
    payload: { title: "", prompt },
    successText: "应用创建成功"
  };
}

export async function sendPrompt() {
  if (!state.user) return state.authModal && state.authModal.show();
  const prompt = els.prompt.value.trim();
  if (!prompt) return;

  const request = buildRequest(prompt);
  appendUserMessage(prompt);
  els.prompt.value = "";

  const assistant = createAssistantEntry("内容生成中", '<span class="loading-dots"><span></span><span></span><span></span></span> 正在准备生成环境');

  try {
    setBusy(true);
    const response = await fetch(request.endpoint, {
      method: "POST",
      headers: authHeaders(),
      body: JSON.stringify(request.payload)
    });

    if (!response.ok) {
      if (response.status === 401) {
        performLogout();
        if (state.authModal) state.authModal.show();
        return;
      }
      const contentType = response.headers.get("Content-Type") || "";
      const errorPayload = contentType.includes("application/json") ? await response.json() : await response.text();
      throw new Error(typeof errorPayload === "string" ? errorPayload : errorPayload.error || "生成失败");
    }

    const appId = response.headers.get("X-App-ID");
    const previewURL = response.headers.get("X-Preview-URL");
    if (appId) state.currentAppId = appId;
    if (previewURL) state.currentPreviewURL = previewURL;

    const reader = response.body.getReader();
    const decoder = new TextDecoder("utf-8");
    let html = "";

    setAssistantNote(assistant, "正在实时编写代码...", "warning");
    assistant.content.innerHTML = '<pre class="streaming-code"><code></code></pre>';
    const codeContainer = assistant.content.querySelector("code");

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      const chunk = decoder.decode(value, { stream: true });
      html += chunk;
      if (codeContainer) codeContainer.textContent = html;
      scrollChatToBottom();
    }
    html += decoder.decode();
    
    if (html.trim()) {
      assistant.content.innerHTML = '<p class="message-text">应用生成完成，请查看下方预览。</p>';
      showPhonePreview(assistant, { html });
    }

    setTimeout(() => refreshApps(), 2500);

    const current = state.apps.find((item) => item.id === state.currentAppId);
    if (current) {
      selectApp(current);
      setAssistantNote(assistant, request.successText, "success");
      if (current.preview_url) showPhonePreview(assistant, { src: current.preview_url + "?ts=" + Date.now() });
    } else {
      setAssistantNote(assistant, request.successText, "success");
    }
  } catch (error) {
    assistant.content.textContent = error.message;
    if (assistant.previewWrapper) assistant.previewWrapper.hidden = true;
    setAssistantNote(assistant, "生成失败", "danger");
  } finally {
    setBusy(false);
    scrollChatToBottom();
  }
}
