import { $, store, api, toast, loadSession, logout, taskPayload }
  from './common.js';
import { render, runDetail } from './views.js';
import { editTask, editChannel, saveEditor } from './forms.js';

let fetching = false;
let detailID = '';

function paint() {
  $('#main').innerHTML = render();
  $('#task-count').textContent = store.data.tasks.length;
  document.querySelectorAll('nav a').forEach((a) => {
    const active = a.dataset.page === store.page;
    a.classList.toggle('active', active);
    if (active) a.setAttribute('aria-current', 'page');
    else a.removeAttribute('aria-current');
  });
  if ($('#detail').open) {
    const run = store.data.runs.find((r) => r.id === detailID);
    if (run) $('#detail-body').innerHTML = runDetail(run);
  }
}

async function reload(forceCatalog = false) {
  if (fetching) return;
  fetching = true;
  try {
    if (!store.catalog.tasks.length || forceCatalog) {
      store.catalog = await api('catalog');
    }
    store.data = await api('state');
    store.connected = true;
    $('#connection-status').innerHTML = '调度引擎在线<small>Go 调度引擎</small>';
    $('#last-refresh').textContent = '最近同步 ' +
      new Date().toLocaleTimeString('zh-CN', { hour12: false });
    paint();
  } catch (err) {
    $('#connection-status').innerHTML = '连接已断开<small>请检查服务连接</small>';
    if (store.connected) toast(err.message, true);
    store.connected = false;
    if (!store.data.tasks.length) {
      $('#main').textContent = '连接失败：' + err.message;
    }
    throw err;
  } finally { fetching = false; }
}
store.reload = reload;

function navigate() {
  const page = location.hash.slice(1);
  const pages = ['overview', 'tasks', 'runs', 'channels', 'plugins'];
  store.page = pages.includes(page)
    ? page : 'overview';
  store.runPage = 0;
  paint();
}

async function action(button) {
  const name = button.dataset.action, id = button.dataset.id;
  if (name === 'new-task') return editTask();
  if (name === 'edit-task') return editTask(id);
  if (name === 'new-channel') return editChannel();
  if (name === 'edit-channel') return editChannel(id);
  if (name === 'detail') {
    detailID = id;
    const r = store.data.runs.find((item) => item.id === id);
    $('#detail-body').innerHTML = runDetail(r);
    return $('#detail').showModal();
  }
  if (name === 'prev-page' || name === 'next-page') {
    store.runPage += name === 'next-page' ? 1 : -1;
    return paint();
  }
  button.disabled = true;
  try {
    if (name === 'run-task') {
      await api(`tasks/${id}/run`, 'POST'); toast('任务已开始执行');
    }
    if (name === 'toggle-task') {
      const t = store.data.tasks.find((item) => item.id === id);
      await api(`tasks/${id}`, 'PUT',
        { ...taskPayload(t), enabled: !t.enabled });
      toast(t.enabled ? '调度已暂停' : '调度已启用');
    }
    if (name === 'delete-task' || name === 'delete-channel') {
      const kind = name === 'delete-task' ? 'tasks' : 'channels';
      const item = store.data[kind].find((entry) => entry.id === id);
      if (!confirm(`确认删除「${item.name}」？此操作无法撤销。`)) return;
      await api(`${kind}/${id}`, 'DELETE'); toast('已删除');
    }
    if (name === 'cancel-run') {
      await api(`runs/${id}/cancel`, 'POST'); toast('已请求取消执行');
    }
    if (name === 'test-channel') {
      await api(`channels/${id}/test`, 'POST'); toast('测试通知已发送');
    }
    await reload();
  } catch (err) { toast(err.message, true); }
  finally { button.disabled = false; }
}

document.addEventListener('click', (event) => {
  const close = event.target.closest('[data-close]');
  if (close) $(`#${close.dataset.close}`).close();
  const button = event.target.closest('[data-action]');
  if (button) action(button).catch((err) => toast(err.message, true));
});

document.addEventListener('change', (event) => {
  if (event.target.id === 'run-status-filter') {
    store.runStatus = event.target.value; store.runPage = 0; paint();
  }
  if (event.target.id === 'run-task-filter') {
    store.runTask = event.target.value; store.runPage = 0; paint();
  }
});

$('#search').addEventListener('input', (event) => {
  store.search = event.target.value;
  if (store.page !== 'tasks') location.hash = 'tasks';
  else paint();
});
$('#editor-form').addEventListener('submit', saveEditor);
$('#auth-button').addEventListener('click', () => {
  logout().catch((err) => toast(err.message, true));
});

document.addEventListener('keydown', (event) => {
  if (event.key === '/' && !event.ctrlKey && !event.metaKey &&
    !['INPUT', 'TEXTAREA', 'SELECT'].includes(event.target.tagName) &&
    !document.querySelector('dialog[open]')) {
    event.preventDefault(); $('#search').focus();
  }
});
window.addEventListener('hashchange', navigate);
navigate();
async function start() {
  if (!await loadSession()) return;
  await reload().catch(() => {});
  setInterval(() => {
    if (!document.hidden) reload().catch(() => {});
  }, 3000);
}
start().catch((err) => toast(err.message, true));
