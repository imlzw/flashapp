import { submitAuth } from "./api.js";
import { saveSession } from "./session.js";

export function createAuthModal({ onLoginSuccess }) {
    const overlay = document.createElement('div');
    overlay.className = 'auth-overlay';
    overlay.innerHTML = `
        <div class="auth-modal">
            <button class="auth-close" id="authClose" title="关闭">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
            </button>
            <div class="auth-modal-banner"></div>
            <div class="auth-modal-content">
                <div class="auth-brand">
                    <img src="/assets/img/logo.svg" alt="闪创 Logo" class="auth-brand-logo">
                    <span class="auth-brand-name">闪创</span>
                </div>
                <div class="auth-form">
                    <div class="auth-input-group">
                        <input type="text" id="authUsername" class="auth-input" placeholder="请输入用户名" autocomplete="username">
                    </div>
                    <div class="auth-input-group">
                        <input type="password" id="authPassword" class="auth-input" placeholder="请输入密码" autocomplete="current-password">
                    </div>
                    <button class="auth-submit-btn" id="authSubmit">登录</button>
                    <div class="auth-status" id="authStatus"></div>
                    <div class="auth-mode-switch">
                        <span id="authModeText">没有账号？</span>
                        <a class="auth-mode-link" id="authModeLink">立即注册</a>
                    </div>
                </div>
                <div class="auth-footer">
                    <input type="checkbox" class="auth-checkbox" id="authAgree">
                    <label for="authAgree">我已阅读并同意 闪创 的 <a href="#">服务协议</a> 和 <a href="#">隐私政策</a></label>
                </div>
            </div>
        </div>
    `;

    document.body.appendChild(overlay);

    const els = {
        overlay,
        close: overlay.querySelector('#authClose'),
        username: overlay.querySelector('#authUsername'),
        password: overlay.querySelector('#authPassword'),
        submit: overlay.querySelector('#authSubmit'),
        status: overlay.querySelector('#authStatus'),
        modeLink: overlay.querySelector('#authModeLink'),
        modeText: overlay.querySelector('#authModeText'),
        agree: overlay.querySelector('#authAgree')
    };

    let mode = 'login'; // login or register

    const setStatus = (msg, tone = '') => {
        els.status.textContent = msg;
        els.status.dataset.tone = tone;
    };

    const toggleMode = () => {
        mode = mode === 'login' ? 'register' : 'login';
        els.submit.textContent = mode === 'login' ? '登录' : '注册';
        els.modeText.textContent = mode === 'login' ? '没有账号？' : '已有账号？';
        els.modeLink.textContent = mode === 'login' ? '立即注册' : '立即登录';
        setStatus('');
    };

    const close = () => {
        overlay.classList.remove('is-active');
        setTimeout(() => {
            // reset form
            els.username.value = '';
            els.password.value = '';
            els.agree.checked = false;
            setStatus('');
        }, 300);
    };

    const show = () => {
        overlay.classList.add('is-active');
    };

    const handleAuth = async () => {
        const username = els.username.value.trim();
        const password = els.password.value.trim();

        if (!username || !password) {
            setStatus('请输入用户名和密码', 'warning');
            return;
        }

        if (!els.agree.checked) {
            setStatus('请阅读并同意服务协议', 'warning');
            return;
        }

        try {
            els.submit.disabled = true;
            setStatus('正在处理...', 'warning');
            
            const payload = await submitAuth({
                mode,
                username,
                password
            });

            saveSession(payload.token, payload.user);
            setStatus('认证成功！', 'success');
            
            setTimeout(() => {
                close();
                if (onLoginSuccess) onLoginSuccess(payload);
            }, 1000);

        } catch (error) {
            setStatus(error.message, 'danger');
        } finally {
            els.submit.disabled = false;
        }
    };

    els.close.addEventListener('click', close);
    els.modeLink.addEventListener('click', toggleMode);
    els.submit.addEventListener('click', handleAuth);
    overlay.addEventListener('click', (e) => {
        if (e.target === overlay) close();
    });
    
    // Allow enter key
    [els.username, els.password].forEach(el => {
        el.addEventListener('keydown', (e) => {
            if (e.key === 'Enter') handleAuth();
        });
    });

    return { show, close };
}
