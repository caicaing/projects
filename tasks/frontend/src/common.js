export const store = {
  data: { tasks: [], channels: [], runs: [] },
  catalog: { tasks: [], notifiers: [] },
  page: 'overview', search: '', runStatus: '', runTask: '', runPage: 0,
  connected: false, reload: async () => {},
};

export const $ = (selector) => document.querySelector(selector);
export const esc = (value) => String(value ?? '').replace(/[&<>"']/g,
  (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;',
    '"': '&quot;', "'": '&#39;' })[c]);

const labels = {
  succeeded: '成功', failed: '失败', running: '执行中',
  canceled: '已取消', interrupted: '已中断', skipped: '已跳过',
  enabled: '已启用', disabled: '已暂停', sent: '已送达',
};

export function badge(status) {
  return `<span class="badge ${esc(status)}">` +
    `${esc(labels[status] || status)}</span>`;
}

export function time(raw, short = false) {
  if (!raw) return '—';
  const options = { timeZone: 'Asia/Shanghai', hour12: false,
    month: '2-digit', day: '2-digit', hour: '2-digit',
    minute: '2-digit', second: '2-digit' };
  if (short) delete options.second;
  return new Date(raw).toLocaleString('zh-CN', options);
}

export function day(raw) {
  return new Date(raw).toLocaleDateString('en-CA',
    { timeZone: 'Asia/Shanghai' });
}

export function plugin(type, channel = false) {
  const list = channel ? store.catalog.notifiers : store.catalog.tasks;
  return list.find((p) => p.type === type);
}

let csrf = '';

export function openAuth() {
  store.data = { tasks: [], channels: [], runs: [] };
  store.catalog = { tasks: [], notifiers: [] };
  document.querySelectorAll('dialog[open]').forEach((d) => d.close());
  $('#main').textContent = '登录已失效，请重新登录';
  location.replace('/login.html');
}

export async function loadSession() {
  sessionStorage.removeItem('taskflow-token');
  const response = await fetch('/auth/session', { cache: 'no-store' });
  if (response.status === 401) { openAuth(); return false; }
  const value = await response.json();
  if (!response.ok) throw new Error(value.error || '无法读取登录状态');
  csrf = value.csrf;
  return true;
}

export async function logout() {
  const response = await fetch('/auth/logout', {
    method: 'POST', headers: { 'X-CSRF-Token': csrf },
  });
  if (!response.ok && response.status !== 401) {
    throw new Error('退出失败，请重试');
  }
  csrf = '';
  openAuth();
}

export async function api(path, method = 'GET', payload) {
  const headers = {};
  if (method !== 'GET') headers['X-CSRF-Token'] = csrf;
  if (payload !== undefined) headers['Content-Type'] = 'application/json';
  const response = await fetch(`/api/${path}`, {
    method, headers,
    body: payload === undefined ? undefined : JSON.stringify(payload),
  });
  let value;
  try { value = await response.json(); }
  catch { throw new Error(`服务返回异常：HTTP ${response.status}`); }
  if (!response.ok) {
    if (response.status === 401) openAuth();
    throw new Error(value.error || `HTTP ${response.status}`);
  }
  return value;
}

let toastTimer;
export function toast(message, error = false) {
  clearTimeout(toastTimer);
  $('#toast').textContent = message;
  $('#toast').className = error ? 'visible error' : 'visible';
  toastTimer = setTimeout(() => { $('#toast').className = ''; }, 4000);
}

export function empty(title, description, action = '', label = '新建任务') {
  return `<div class="empty"><div class="empty-symbol">◷</div>
    <h3>${esc(title)}</h3><p>${esc(description)}</p>
    ${action ? `<button class="button secondary" data-action="${action}">
      ${esc(label)}</button>` : ''}</div>`;
}

export function heading(eyebrow, title, subtitle, action, label) {
  return `<div class="page-heading"><div>
    <div class="eyebrow">${eyebrow}</div><h1>${title}</h1>
    <p class="muted">${subtitle}</p></div><div class="heading-actions">
    <button class="button secondary" data-action="refresh">↻ 刷新</button>
    ${action ? `<button class="button primary" data-action="${action}">
      <span class="plus">+</span>${label}</button>` : ''}
    </div></div>`;
}

export function taskPayload(t) {
  const keys = ['name', 'description', 'type', 'config', 'schedule',
    'enabled', 'timeout', 'retries', 'retry_delay', 'notify_on',
    'notifier_ids'];
  return Object.fromEntries(keys.map((key) => [key, t[key]]));
}
