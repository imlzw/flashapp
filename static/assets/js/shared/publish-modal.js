import { publishApp } from "./api.js";

const PUBLISH_DEVICE_PRESETS = {
    mobile: { width: 360, height: 740 },
    tablet: { width: 720, height: 960 }
};

export function createPublishModal({ onPublishSuccess }) {
    const overlay = document.createElement('div');
    overlay.className = 'publish-overlay';
    overlay.innerHTML = `
        <div class="publish-modal">
            <div class="publish-header">
                <h2 class="publish-title">发布应用</h2>
                <button class="publish-close" id="publishClose" title="关闭">
                    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
                </button>
            </div>
            <div class="publish-body">
                <div class="publish-preview-section">
                    <span class="publish-preview-label">封面预览</span>
                    <div class="publish-preview-container">
                        <img id="publishPreviewImg" class="publish-preview-img" src="/assets/img/logo.svg" alt="预览图">
                    </div>
                </div>
                <div class="publish-info-section">
                    <div class="publish-field">
                        <label class="publish-label" for="publishTitleInput">应用标题</label>
                        <input type="text" id="publishTitleInput" class="publish-input" placeholder="给你的应用起个名字">
                    </div>
                    <div class="publish-field">
                        <label class="publish-label" for="publishPromptInput">应用描述 / Prompt</label>
                        <textarea id="publishPromptInput" class="publish-input publish-textarea" placeholder="描述这个应用的功能或使用的 Prompt"></textarea>
                    </div>
                </div>
            </div>
            <div class="publish-footer">
                <button class="publish-btn publish-btn-cancel" id="publishCancel">取消</button>
                <button class="publish-btn publish-btn-confirm" id="publishConfirm">确认发布</button>
            </div>
        </div>
    `;

    document.body.appendChild(overlay);

    const els = {
        overlay,
        close: overlay.querySelector('#publishClose'),
        cancel: overlay.querySelector('#publishCancel'),
        confirm: overlay.querySelector('#publishConfirm'),
        title: overlay.querySelector('#publishTitleInput'),
        prompt: overlay.querySelector('#publishPromptInput'),
        previewContainer: overlay.querySelector('.publish-preview-container'),
        previewImg: overlay.querySelector('#publishPreviewImg')
    };

    let currentAppId = null;
    let currentScreenshot = "";

    const close = () => {
        overlay.classList.remove('is-active');
    };

    const applyPreviewMode = (deviceMode = "mobile") => {
        const preset = PUBLISH_DEVICE_PRESETS[deviceMode] || PUBLISH_DEVICE_PRESETS.mobile;
        if (!els.previewContainer) return;
        els.previewContainer.style.setProperty('--publish-preview-width', String(preset.width));
        els.previewContainer.style.setProperty('--publish-preview-height', String(preset.height));
    };

    const show = ({ appId, title, prompt, screenshot, deviceMode = "mobile" }) => {
        currentAppId = appId;
        currentScreenshot = screenshot;
        els.title.value = title || "";
        els.prompt.value = prompt || "";
        applyPreviewMode(deviceMode);
        els.previewImg.src = screenshot || "/assets/img/logo.svg";
        overlay.classList.add('is-active');
        els.confirm.disabled = false;
        els.confirm.textContent = "确认发布";
    };

    const handlePublish = async () => {
        if (!currentAppId) return;
        
        const title = els.title.value.trim();
        const prompt = els.prompt.value.trim();

        try {
            els.confirm.disabled = true;
            els.confirm.textContent = "发布中...";
            
            await publishApp(currentAppId, currentScreenshot, title, prompt);
            
            close();
            if (onPublishSuccess) onPublishSuccess();
        } catch (error) {
            alert("发布失败: " + error.message);
        } finally {
            els.confirm.disabled = false;
            els.confirm.textContent = "确认发布";
        }
    };

    els.close.addEventListener('click', close);
    els.cancel.addEventListener('click', close);
    els.confirm.addEventListener('click', handlePublish);
    overlay.addEventListener('click', (e) => {
        if (e.target === overlay) close();
    });

    return { show, close };
}
