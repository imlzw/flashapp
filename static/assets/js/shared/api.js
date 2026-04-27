import { getSession } from "./session.js";

export function buildJSONHeaders(extra = {}) {
  return { "Content-Type": "application/json", ...extra };
}

export function authHeaders(extra = {}) {
  const headers = buildJSONHeaders(extra);
  const session = getSession();
  if (session.token) {
    headers.Authorization = "Bearer " + session.token;
  }
  return headers;
}

export async function fetchJSON(path, options = {}) {
  const response = await fetch(path, options);
  const contentType = response.headers.get("Content-Type") || "";
  const payload = contentType.includes("application/json")
    ? await response.json()
    : await response.text();

  if (!response.ok) {
    const message = typeof payload === "string" ? payload : payload.error || "请求失败";
    const error = new Error(message);
    error.status = response.status;
    throw error;
  }

  return payload;
}

export async function fetchHealth() {
  return fetchJSON("/api/health");
}

export async function submitAuth(payload) {
  return fetchJSON("/api/auth", {
    method: "POST",
    headers: buildJSONHeaders(),
    body: JSON.stringify(payload)
  });
}

export async function fetchApps() {
  return fetchJSON("/api/apps", {
    headers: authHeaders()
  });
}

export async function deleteApp(appId) {
  return fetchJSON("/api/delete", {
    method: "POST",
    headers: authHeaders(),
    body: JSON.stringify({ app_id: appId })
  });
}

export async function publishApp(appId, screenshot, title, prompt) {
  return fetchJSON("/api/publish", {
    method: "POST",
    headers: authHeaders(),
    body: JSON.stringify({ 
      app_id: appId, 
      screenshot: screenshot,
      title: title,
      prompt: prompt
    })
  });
}

export async function togglePublic(appId) {
  return fetchJSON("/api/apps/toggle_public", {
    method: "POST",
    headers: authHeaders(),
    body: JSON.stringify({ app_id: appId })
  });
}

export async function deletePublishedApp(appId) {
  return fetchJSON("/api/apps/delete_published", {
    method: "POST",
    headers: authHeaders(),
    body: JSON.stringify({ app_id: appId })
  });
}

export async function fetchPlaza() {
  return fetchJSON("/api/plaza", {
    headers: buildJSONHeaders()
  });
}

export async function fetchMyPublishedApps() {
  return fetchJSON("/api/my_published_apps", {
    headers: authHeaders()
  });
}

export async function forkApp(pubId) {
  return fetchJSON("/api/fork", {
    method: "POST",
    headers: authHeaders(),
    body: JSON.stringify({ pub_id: pubId })
  });
}

export async function changePassword(oldPassword, newPassword) {
  return fetchJSON("/api/user/password", {
    method: "POST",
    headers: authHeaders(),
    body: JSON.stringify({ old_password: oldPassword, new_password: newPassword })
  });
}

export async function changeNickname(newNickname) {
  return fetchJSON("/api/user/nickname", {
    method: "POST",
    headers: authHeaders(),
    body: JSON.stringify({ new_nickname: newNickname })
  });
}

export async function changeUsername(newUsername) {
  return fetchJSON("/api/user/username", {
    method: "POST",
    headers: authHeaders(),
    body: JSON.stringify({ new_username: newUsername })
  });
}
