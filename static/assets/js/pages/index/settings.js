import { els } from "./elements.js";
import { state } from "./state.js";
import { switchStage, updateUserUI } from "./ui.js";
import { changePassword, changeNickname, changeUsername } from "../../shared/api.js";
import { saveSession, getSession } from "../../shared/session.js";

export function initSettings() {
  if (els.settingsBtn) {
    els.settingsBtn.addEventListener("click", () => {
      if (!state.user) {
        state.authModal.show();
        return;
      }
      switchStage("settings");
      
      if (els.settingsNickname) {
        els.settingsNickname.value = state.user.nickname || state.user.username;
      }
      if (els.settingsUsername) {
        els.settingsUsername.value = state.user.username;
      }
      
      if (els.profileForm) els.profileForm.reset();
      if (els.profileMessage) {
        els.profileMessage.textContent = "";
        els.profileMessage.className = "settings-message";
      }

      if (els.passwordForm) els.passwordForm.reset();
      if (els.passwordMessage) {
        els.passwordMessage.textContent = "";
        els.passwordMessage.className = "settings-message";
      }
    });
  }

  if (els.profileForm) {
    els.profileForm.addEventListener("submit", async (e) => {
      e.preventDefault();
      
      const newNickname = els.settingsNickname.value.trim();
      const newUsername = els.settingsUsername.value.trim();

      if (newNickname.length < 1) {
        showProfileMessage("昵称不能为空", "error");
        return;
      }
      if (newUsername.length < 3) {
        showProfileMessage("登录账号至少3个字符", "error");
        return;
      }

      if (els.saveProfileBtn) els.saveProfileBtn.disabled = true;
      showProfileMessage("正在保存...", "");

      try {
        let changed = false;
        // Handle nickname update
        if (newNickname !== (state.user.nickname || state.user.username)) {
            await changeNickname(newNickname);
            changed = true;
        }

        // Handle username update
        if (newUsername !== state.user.username) {
            await changeUsername(newUsername);
            changed = true;
        }
        
        if (changed) {
            showProfileMessage("保存成功", "success");
            
            // Update session state
            const session = getSession();
            if (session && session.user) {
                session.user.nickname = newNickname;
                session.user.username = newUsername;
                saveSession(session.token, session.user);
                state.user.nickname = newNickname;
                state.user.username = newUsername;
                updateUserUI();
            }
        } else {
            showProfileMessage("未做任何修改", "");
        }
      } catch (err) {
        showProfileMessage(err.message, "error");
      } finally {
        if (els.saveProfileBtn) els.saveProfileBtn.disabled = false;
      }
    });
  }

  if (els.passwordForm) {
    els.passwordForm.addEventListener("submit", async (e) => {
      e.preventDefault();
      
      const oldPassword = els.oldPassword.value;
      const newPassword = els.newPassword.value;
      
      if (newPassword.length < 6) {
        showPasswordMessage("新密码不能少于6位", "error");
        return;
      }

      if (els.savePasswordBtn) els.savePasswordBtn.disabled = true;
      showPasswordMessage("正在修改...", "");

      try {
        await changePassword(oldPassword, newPassword);
        showPasswordMessage("密码修改成功", "success");
        els.passwordForm.reset();
      } catch (err) {
        showPasswordMessage(err.message, "error");
      } finally {
        if (els.savePasswordBtn) els.savePasswordBtn.disabled = false;
      }
    });
  }
}

function showProfileMessage(text, type) {
  if (!els.profileMessage) return;
  els.profileMessage.textContent = text;
  els.profileMessage.className = "settings-message" + (type ? " " + type : "");
}

function showPasswordMessage(text, type) {
  if (!els.passwordMessage) return;
  els.passwordMessage.textContent = text;
  els.passwordMessage.className = "settings-message" + (type ? " " + type : "");
}
