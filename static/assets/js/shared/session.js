const TOKEN_KEY = "flashapp_token";
const USER_KEY = "flashapp_user";

export function readJSON(key) {
  try {
    const raw = localStorage.getItem(key);
    return raw ? JSON.parse(raw) : null;
  } catch {
    return null;
  }
}

export function getSession() {
  return {
    token: localStorage.getItem(TOKEN_KEY) || "",
    user: readJSON(USER_KEY)
  };
}

export function hasSession() {
  return Boolean(getSession().token);
}

export function saveSession(token, user) {
  if (token) {
    localStorage.setItem(TOKEN_KEY, token);
  } else {
    localStorage.removeItem(TOKEN_KEY);
  }

  if (user) {
    localStorage.setItem(USER_KEY, JSON.stringify(user));
  } else {
    localStorage.removeItem(USER_KEY);
  }
}

export function clearSession() {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(USER_KEY);
}

/**
 * Returns the session if it exists, otherwise returns null.
 * No longer redirects automatically.
 */
export function requireSession() {
  const session = getSession();
  if (!session.token) {
    return null;
  }
  return session;
}

export function goToHome() {
  window.location.replace("/");
}
