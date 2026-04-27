/**
 * Theme Engine for FlashApp
 * Handles light/dark mode switching and persistence.
 */

const STORAGE_KEY = "flashapp_theme";

export function getTheme() {
  const saved = localStorage.getItem(STORAGE_KEY);
  if (saved) return saved;
  
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

export function applyTheme(theme) {
  document.documentElement.setAttribute("data-theme", theme);
  localStorage.setItem(STORAGE_KEY, theme);
  
  // Custom event for components that need to react to theme changes
  window.dispatchEvent(new CustomEvent("themechange", { detail: { theme } }));
}

export function toggleTheme() {
  const current = getTheme();
  const next = current === "dark" ? "light" : "dark";
  applyTheme(next);
  return next;
}

function injectThemeFloater() {
  if (document.getElementById("globalThemeFloater")) return;

  const btn = document.createElement("button");
  btn.id = "globalThemeFloater";
  btn.className = "theme-floater";
  btn.title = "切换主题";
  btn.type = "button";
  btn.innerHTML = `<svg class="sun-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="5"></circle><line x1="12" y1="1" x2="12" y2="3"></line><line x1="12" y1="21" x2="12" y2="23"></line><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"></line><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"></line><line x1="1" y1="12" x2="3" y2="12"></line><line x1="21" y1="12" x2="23" y2="12"></line><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"></line><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"></line></svg>`;

  btn.addEventListener("click", () => {
    toggleTheme();
  });

  document.body.appendChild(btn);
}

// Initial application
export function initTheme() {
  const theme = getTheme();
  applyTheme(theme);
  
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", injectThemeFloater);
  } else {
    injectThemeFloater();
  }
}
