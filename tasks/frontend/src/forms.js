import { $, store, esc, api, plugin, toast } from './common.js';

import { scheduleFields, bindSchedule, getSchedule }
  from './schedule-form.js';

let editing = { kind: 'task', id: '' };

export function editTask(id = '') {
  const t = store.data.tasks.find((item) => item.id === id);
  editing = { kind: 'task', id };
  $('#editor-title').textContent = t ? '编辑任务' : '新建任务';
  $('#editor-eyebrow').textContent = 'TASK CONFIGURATION';
  const type = t?.type || plugin('log')?.type || store.catalog.tasks[0]?.type;
  if (!type) return toast('暂无任务插件，请先在后端注册任务。', true);
  const config = t?.config || plugin(type)?.example || {};
  $('#editor-fields').innerHTML = taskFields(t, type, config);
  $('#editor-error').textContent = '';
  bindType(false);
  bindSchedule();
  $('#editor').showModal();
}

function taskFields(t, type, config) {
  const channels = store.data.channels.map((c) =>
    `<label class="checkbox-label"><input type="checkbox"
      name="notifier" value="${c.id}"
      ${t?.notifier_ids?.includes(c.id) ? 'checked' : ''}>
      ${esc(c.name)}${c.enabled ? '' : '（已暂停）'}</label>`).join('');
  return `<div class="form-grid">
    ${input('name', '任务名称', t?.name || '', true)}
    <label>任务类型<select id="plugin-type" name="type">
      ${typeOptions(store.catalog.tasks, type)}</select></label>
    <label class="full">描述<input name="description"
      value="${esc(t?.description || '')}" placeholder="这个任务负责什么？">
    </label>
    ${configField(config)}
  </div><div class="form-section-title">执行计划</div>
  ${scheduleFields(t?.schedule)}
  <div class="form-grid">
    ${input('timeout', '总超时时间', t?.timeout || '30s', true)}
    <label>失败重试次数<input type="number" name="retries" min="0"
      max="10" required value="${t?.retries || 0}"></label>
    ${input('retry_delay', '重试间隔', t?.retry_delay || '2s', true)}
    <label class="checkbox-label"><input type="checkbox" name="enabled"
      ${t?.enabled !== false ? 'checked' : ''}>启用定时调度</label>
  </div><div class="field-hint">总超时包含全部尝试与重试等待。
    暂停调度后仍可手动运行。</div>
  <div class="form-section-title">执行通知</div>
  <label>通知策略<select name="notify_on">
    ${options([['failure', '仅执行失败'], ['always', '每次执行完成'],
      ['never', '不发送通知']], t?.notify_on || 'failure')}</select></label>
  <div class="channel-options">${channels ||
    '<p class="muted">暂无通知渠道，请在「通知渠道」页面添加。</p>'}</div>`;
}

function input(name, label, value, required = false) {
  return `<label>${label}<input name="${name}" value="${esc(value)}"
    ${required ? 'required' : ''}></label>`;
}

function options(list, selected) {
  return list.map(([value, label]) => `<option value="${value}"
    ${selected === value ? 'selected' : ''}>${label}</option>`).join('');
}

function typeOptions(list, selected) {
  return list.map((p) => `<option value="${esc(p.type)}"
    ${selected === p.type ? 'selected' : ''}>${esc(p.name)}</option>`).join('');
}

function configField(config) {
  return `<div class="full"><div class="field-header">
    <label for="config">插件参数 · JSON</label><button type="button"
    class="text-button" id="reset-config">填入示例</button></div>
    <p id="plugin-description" class="field-description"></p>
    <textarea class="json-editor" id="config" name="config" required
      spellcheck="false">${esc(JSON.stringify(config, null, 2))}</textarea>
    <div class="field-hint">插件的参数由对应 Go 实现校验。</div></div>`;
}

function bindType(channel) {
  const describe = () => {
    const p = plugin($('#plugin-type').value, channel);
    $('#plugin-description').textContent = p?.description || '';
  };
  const reset = () => {
    const p = plugin($('#plugin-type').value, channel);
    $('#config').value = JSON.stringify(p?.example || {}, null, 2);
    describe();
  };
  $('#plugin-type').addEventListener('change', reset);
  $('#reset-config').addEventListener('click', reset);
  describe();
}

export function editChannel(id = '') {
  const c = store.data.channels.find((item) => item.id === id);
  editing = { kind: 'channel', id };
  $('#editor-title').textContent = c ? '编辑通知渠道' : '添加通知渠道';
  $('#editor-eyebrow').textContent = 'NOTIFICATION CONFIGURATION';
  const type = c?.type || plugin('webhook', true)?.type
    || store.catalog.notifiers[0]?.type;
  if (!type) return toast('暂无通知插件，请先在后端注册通知方式。', true);
  const config = c?.config || plugin(type, true)?.example || {};
  $('#editor-fields').innerHTML = `<div class="form-grid">
    ${input('name', '渠道名称', c?.name || '', true)}
    <label>通知类型<select id="plugin-type" name="type">
      ${typeOptions(store.catalog.notifiers, type)}</select></label>
    ${configField(config)}
    <label class="checkbox-label full"><input type="checkbox" name="enabled"
      ${c?.enabled !== false ? 'checked' : ''}>启用通知投递</label></div>`;
  $('#editor-error').textContent = '';
  bindType(true);
  $('#editor').showModal();
}

export async function saveEditor(event) {
  event.preventDefault();
  const form = $('#editor-form');
  const get = (name) => form.elements[name].value;
  $('#save-button').disabled = true;
  $('#editor-error').textContent = '';
  try {
    const config = JSON.parse(get('config'));
    if (!config || Array.isArray(config) || typeof config !== 'object') {
      throw new Error('插件参数必须是 JSON 对象');
    }
    const payload = { name: get('name'), type: get('type'), config,
      enabled: form.elements.enabled.checked };
    if (editing.kind === 'task') {
      Object.assign(payload, {
        description: get('description'), schedule: getSchedule(),
        timeout: get('timeout'), retries: Number(get('retries')),
        retry_delay: get('retry_delay'), notify_on: get('notify_on'),
        notifier_ids: [...form.querySelectorAll('[name="notifier"]:checked')]
          .map((input) => input.value),
      });
    }
    const collection = editing.kind === 'task' ? 'tasks' : 'channels';
    const path = collection + (editing.id ? `/${editing.id}` : '');
    await api(path, editing.id ? 'PUT' : 'POST', payload);
    $('#editor').close();
    toast('配置已保存');
    await store.reload();
  } catch (err) { $('#editor-error').textContent = err.message; }
  finally { $('#save-button').disabled = false; }
}
