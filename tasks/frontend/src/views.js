import { store, esc, time, day, badge, heading, empty, plugin }
  from './common.js';
import { scheduleSummary } from './schedule.js';

export function render() {
  const views = { overview, tasks, runs, channels, plugins };
  return (views[store.page] || overview)();
}

function overview() {
  const today = day(Date.now());
  const todayRuns = store.data.runs.filter((r) => day(r.started_at) === today);
  const completed = todayRuns.filter((r) =>
    ['succeeded', 'failed', 'canceled', 'interrupted'].includes(r.status));
  const success = completed.filter((r) => r.status === 'succeeded').length;
  const enabled = store.data.tasks.filter((t) => t.enabled).length;
  const active = store.data.runs.filter((r) => r.status === 'running').length;
  const percent = completed.length
    ? Math.round(success / completed.length * 100) : null;
  return heading('WORKSPACE OVERVIEW', '调度概览',
    '任务自动运行，进度尽在掌握。', 'new-task', '新建任务') +
    `<div class="stats">
      ${stat('任务总数', store.data.tasks.length, '⊞',
        `<strong>${enabled} 个</strong>任务已启用`)}
      ${stat('今日执行', todayRuns.length, '◷', '基于保留记录 · 上海时间')}
      ${stat('今日成功率', percent ?? '—', '↗',
        `${completed.length} 次已完成执行`, percent === null ? '' : '%')}
      ${stat('正在执行', active, '⇄', '同一任务自动避免重复执行')}
    </div>
    <div class="section-grid">
      <section class="panel">
        <div class="panel-head"><div><h2>执行趋势</h2>
          <p>最近 7 天 · 基于保留的执行记录</p></div>
          <span class="tag">近 7 天</span></div>
        ${chart()}
      </section>
      <section class="panel">
        <div class="panel-head"><h2>开始构建你的工作流</h2>
          <span class="tag">QUICK START</span></div>
        <div class="quick-start">
          ${step('01', '配置一个任务', '选择任务插件，填写参数与执行计划。')}
          ${step('02', '连接通知渠道', '执行失败时，通过 Webhook 及时获知。')}
          ${step('03', '查看每一次执行', '手动试跑，在运行记录中检查输出。')}
        </div>
      </section>
    </div>
    <section class="panel task-panel">
      <div class="panel-head"><div><h2>定时任务</h2>
        <p>已配置 ${store.data.tasks.length} 个任务</p></div>
        <a href="#tasks" class="subtle-link">查看全部 ↗</a></div>
      ${taskTable(filteredTasks().slice(0, 5))}
      <div class="panel-foot"><span>调度精度 1 秒 · 重叠执行自动跳过</span>
        <span>单实例调度</span></div>
    </section>
    <section class="panel task-panel">
      <div class="panel-head"><h2>最近执行</h2>
        <a href="#runs" class="subtle-link">全部记录 ↗</a></div>
      ${runTable(store.data.runs.slice(0, 5))}
    </section>`;
}

function stat(label, value, icon, note, unit = '') {
  return `<div class="stat-card"><div class="stat-label">${label}
    <span class="stat-icon">${icon}</span></div>
    <div class="stat-number">${value}<small>${unit}</small></div>
    <div class="stat-note">${note}</div></div>`;
}

function step(number, title, description) {
  return `<div class="step"><span class="step-number">${number}</span>
    <div><h3>${title}</h3><p>${description}</p></div></div>`;
}

function chart() {
  const buckets = Array.from({ length: 7 }, (_, i) => {
    const date = new Date(Date.now() - (6 - i) * 86400000);
    const key = day(date);
    const items = store.data.runs.filter((r) => day(r.started_at) === key);
    return { label: date.toLocaleDateString('zh-CN', {
      month: '2-digit', day: '2-digit', timeZone: 'Asia/Shanghai',
    }), ok: items.filter((r) => r.status === 'succeeded').length,
    fail: items.filter((r) =>
      ['failed', 'interrupted', 'canceled'].includes(r.status)).length };
  });
  const max = Math.max(4, ...buckets.map((b) => b.ok + b.fail));
  const grid = [0, 1, 2].map((n) => {
    const y = 138 - n * 55;
    return `<line class="grid" x1="30" x2="490" y1="${y}" y2="${y}"/>
      <text x="9" y="${y + 3}">${Math.round(max * n / 2)}</text>`;
  }).join('');
  const bars = buckets.map((b, i) => {
    const x = 51 + i * 64;
    const ok = b.ok / max * 110, fail = b.fail / max * 110;
    return `<rect x="${x}" y="${138 - ok}" width="24" height="${ok}"
        rx="3" fill="#96b993"><title>成功 ${b.ok} 次</title></rect>
      <rect x="${x}" y="${138 - ok - fail}" width="24"
        height="${fail}" rx="3" fill="#d9a48c">
        <title>失败或中断 ${b.fail} 次</title></rect>
      <text x="${x + 12}" y="163" text-anchor="middle">${b.label}</text>`;
  }).join('');
  return `<div class="chart"><svg viewBox="0 0 510 175" role="img"
    aria-label="最近七天成功和失败次数柱状图">${grid}${bars}</svg>
    <div class="chart-legend"><span><i></i>执行成功</span>
    <span><i class="failed"></i>失败 / 中断 / 取消</span></div></div>`;
}

function filteredTasks() {
  const query = store.search.toLowerCase();
  return store.data.tasks.filter((t) =>
    `${t.name} ${t.description} ${t.type}`.toLowerCase().includes(query));
}

function tasks() {
  const list = filteredTasks();
  return heading('TASK MANAGEMENT', '定时任务',
    '管理执行计划，按需试跑或暂停。', 'new-task', '新建任务') +
    `<section class="panel"><div class="panel-head"><h2>任务列表</h2>
      <span class="tag">${list.length} 个任务</span></div>
      ${taskTable(list)}
      <div class="panel-foot"><span>时间显示为 Asia/Shanghai</span>
        <span>任务执行期间，修改配置将在下次执行时生效</span></div>
    </section>`;
}

function taskTable(list) {
  if (!list.length) return empty(store.search ? '没有匹配的任务' : '准备就绪',
    store.search ? '试试其他关键词。' : '创建第一个任务，让调度引擎开始工作。',
    store.search ? '' : 'new-task');
  const rows = list.map((t) => {
    const r = store.data.runs.find((item) => item.task_id === t.id);
    return `<tr><td><div class="task-cell"><span class="task-symbol">
      ${t.type === 'http' ? '↗' : '≡'}</span><div>
      <div class="task-name">${esc(t.name)}</div>
      <div class="task-desc">${esc(t.description || plugin(t.type)?.name
        || t.type)}</div></div></div></td>
      <td><div class="mono">${esc(scheduleSummary(t.schedule))}</div>
        <div class="task-desc">${t.schedule.kind !== 'interval'
          ? esc(t.schedule.timezone) : '固定间隔'}</div></td>
      <td>${r ? badge(r.status) : '<span class="time-text">待执行</span>'}</td>
      <td class="time-text">${time(t.next_run, true)}</td>
      <td><button class="toggle ${t.enabled ? 'on' : ''}"
        role="switch" aria-checked="${t.enabled}"
        aria-label="${t.enabled ? '暂停' : '启用'} ${esc(t.name)}"
        data-action="toggle-task" data-id="${t.id}"></button></td>
      <td><div class="row-actions">
        <button class="text-button" data-action="run-task"
          data-id="${t.id}">运行</button>
        <button class="text-button" data-action="edit-task"
          data-id="${t.id}">编辑</button>
        <button class="text-button danger" data-action="delete-task"
          data-id="${t.id}">删除</button></div></td></tr>`;
  }).join('');
  return `<div class="table-wrap"><table><thead><tr>
    <th>任务名称</th><th>执行计划</th><th>最近状态</th>
    <th>下次执行</th><th>启用</th><th>操作</th>
    </tr></thead><tbody>${rows}</tbody></table></div>`;
}

function runs() {
  const list = store.data.runs.filter((r) =>
    (!store.runStatus || r.status === store.runStatus) &&
    (!store.runTask || r.task_id === store.runTask));
  const pages = Math.max(1, Math.ceil(list.length / 15));
  store.runPage = Math.min(store.runPage, pages - 1);
  const states = ['running', 'succeeded', 'failed', 'canceled',
    'interrupted', 'skipped'];
  const statusOptions = states.map((s) => `<option value="${s}"
    ${store.runStatus === s ? 'selected' : ''}>${statusName(s)}</option>`);
  const taskOptions = store.data.tasks.map((t) => `<option value="${t.id}"
    ${store.runTask === t.id ? 'selected' : ''}>${esc(t.name)}</option>`);
  return heading('EXECUTION HISTORY', '运行记录',
    '检查执行结果、输出和通知投递情况。', '', '') +
    `<section class="panel"><div class="panel-head"><h2>执行历史</h2>
      <div class="filters"><select id="run-task-filter" aria-label="任务筛选">
        <option value="">全部任务</option>${taskOptions.join('')}</select>
        <select id="run-status-filter" aria-label="状态筛选">
        <option value="">全部状态</option>${statusOptions.join('')}</select>
      </div></div>
      ${runTable(list.slice(store.runPage * 15, store.runPage * 15 + 15))}
      <div class="panel-foot"><span>共 ${list.length} 条保留记录 · 上海时间</span>
        <div class="row-actions"><button class="text-button"
          data-action="prev-page" ${store.runPage === 0 ? 'disabled' : ''}>
          ← 上一页</button><span>${store.runPage + 1} / ${pages}</span>
          <button class="text-button" data-action="next-page"
          ${store.runPage >= pages - 1 ? 'disabled' : ''}>下一页 →</button>
        </div></div></section>`;
}

function statusName(s) {
  return ({ running: '执行中', succeeded: '成功', failed: '失败',
    canceled: '已取消', interrupted: '已中断', skipped: '已跳过' })[s];
}

function runTable(list) {
  if (!list.length) return empty('还没有执行记录',
    '任务运行后，结果和输出会出现在这里。');
  const rows = list.map((r) => `<tr><td>
    <button class="text-button" data-action="detail" data-id="${r.id}">
      ${esc(r.task_name)}</button><div class="task-desc mono">
      ${r.id.slice(0, 12)}</div></td>
    <td>${badge(r.status)}</td>
    <td class="time-text">${time(r.started_at)}</td>
    <td class="mono">${r.finished_at ? `${r.duration_ms} ms` : '—'}</td>
    <td class="time-text">${r.trigger === 'manual' ? '手动' : '定时'}</td>
    <td class="time-text">${r.attempts}</td>
    <td><div class="row-actions"><button class="text-button"
      data-action="detail" data-id="${r.id}">详情</button>
      ${r.status === 'running' ? `<button class="text-button danger"
        data-action="cancel-run" data-id="${r.id}">取消</button>` : ''}
    </div></td></tr>`).join('');
  return `<div class="table-wrap"><table><thead><tr><th>任务 / 执行 ID</th>
    <th>执行状态</th><th>开始时间</th><th>耗时</th><th>触发方式</th>
    <th>尝试次数</th><th>操作</th></tr></thead><tbody>${rows}</tbody>
    </table></div>`;
}

function channels() {
  const cards = store.data.channels.map((c) => `<article
    class="panel channel-card"><div class="card-top">
    <span class="task-symbol">${c.type === 'webhook' ? '↗' : '≡'}</span>
    <div><h3>${esc(c.name)}</h3><div class="card-type">
      ${esc(plugin(c.type, true)?.name || c.type)}</div></div>
    ${badge(c.enabled ? 'enabled' : 'disabled')}</div>
    <div class="card-info">${esc(plugin(c.type, true)?.description || '')}
      ${c.config.url ? `<code>${esc(c.config.url)}</code>` : ''}</div>
    <div class="card-actions"><button class="text-button"
      data-action="test-channel" data-id="${c.id}">测试通知</button>
      <button class="text-button" data-action="edit-channel"
      data-id="${c.id}">编辑</button><button class="text-button danger"
      data-action="delete-channel" data-id="${c.id}">删除</button>
    </div></article>`).join('');
  return heading('NOTIFICATION CHANNELS', '通知渠道',
    '将执行结果送达你的工作流，支持多个渠道。', 'new-channel', '添加渠道') +
    `<div class="info-strip">在任务配置中选择通知策略和渠道。
      暂停的渠道会跳过投递，投递失败会保留在执行详情中。</div>` +
    (cards ? `<div class="cards">${cards}</div>` :
      `<section class="panel">${empty('让重要结果及时送达',
        '添加 Webhook 或控制台通知，然后在任务中关联。',
        'new-channel', '添加通知渠道')}</section>`);
}

function plugins() {
  return heading('PLUGIN REGISTRY', '插件目录',
    '注册一次，在管理台直接配置使用。', '', '') +
    `<div class="info-strip">新增 Go 插件，实现统一接口并注册到对应工厂，
      在 backend/cmd/taskflow/plugins.go 注册后重新编译。
      插件描述和示例参数会自动显示在管理台。</div>` +
    pluginSection('任务插件', store.catalog.tasks) +
    pluginSection('通知插件', store.catalog.notifiers);
}

function pluginSection(title, list) {
  const cards = list.map((p) => `<article class="panel plugin-card">
    <div class="card-top"><span class="task-symbol">⊞</span>
      <div><h3>${esc(p.name)}</h3><div class="card-type mono">
        ${esc(p.type)}</div></div><span class="tag">已注册</span></div>
    <p class="card-info">${esc(p.description)}</p>
    <div class="field-hint">配置示例</div>
    <pre>${esc(JSON.stringify(p.example, null, 2))}</pre></article>`).join('');
  return `<section class="plugin-section"><h2>${title}</h2>
    <div class="cards">${cards}</div></section>`;
}

export function runDetail(r) {
  const deliveries = r.deliveries.map((d) => `<div class="delivery">
    <span>${esc(d.channel_name)}</span>${badge(d.status)}</div>
    ${d.error ? `<p class="delivery-error">${esc(d.error)}</p>`
      : ''}`).join('');
  return `<dl class="detail-grid"><div><dt>任务</dt>
    <dd>${esc(r.task_name)}</dd></div><div><dt>状态</dt>
    <dd>${badge(r.status)}</dd></div><div><dt>开始时间</dt>
    <dd>${time(r.started_at)}</dd></div><div><dt>结束时间</dt>
    <dd>${time(r.finished_at)}</dd></div><div><dt>执行 ID</dt>
    <dd class="mono">${r.id}</dd></div><div><dt>尝试次数 / 耗时</dt>
    <dd>${r.attempts} 次 / ${r.duration_ms} ms</dd></div></dl>
    <div class="output-label">执行输出</div>
    <pre class="output">${esc(r.output || '暂无输出')}</pre>
    ${r.error ? `<div class="output-label">错误信息</div>
      <pre class="output error">${esc(r.error)}</pre>` : ''}
    <div class="output-label">通知投递</div>
    ${deliveries || '<p class="muted">暂无投递记录</p>'}`;
}
