import { $, esc, api, time } from './common.js';
import { scheduleModel, scheduleSummary } from './schedule.js';

let model;
let previewVersion = 0;

export function scheduleFields(schedule) {
  model = scheduleModel(schedule);
  const types = [['daily', '每天'], ['weekly', '每周'],
    ['monthly', '每月'], ['hourly', '每小时'], ['interval', '固定间隔'],
    ['cron', '高级自定义计划']];
  if (model.kind === 'legacy-interval') {
    types.push(['legacy-interval', '保留原有间隔']);
  }
  const choices = types.map(([value, label]) => `<option value="${value}"
    ${model.kind === value ? 'selected' : ''}>${label}</option>`).join('');
  return `<div class="form-grid"><label for="schedule-kind">执行周期
    <select name="schedule_kind" id="schedule-kind">${choices}</select>
    </label><label for="schedule-timezone">执行时区
    <input name="timezone" id="schedule-timezone"
      value="${esc(model.timezone || 'Asia/Shanghai')}" required></label>
    <div id="schedule-controls" class="full"></div>
    <div class="full"><p id="schedule-summary" class="field-description"></p>
      <button type="button" class="text-button" id="preview-schedule">
        预览接下来 5 次执行 ↗</button>
      <div class="preview-times" id="preview-times"></div></div></div>`;
}

export function bindSchedule() {
  renderControls();
  $('#schedule-kind').addEventListener('change', () => {
    model = { ...model, ...readValues(), kind: $('#schedule-kind').value };
    renderControls();
  });
  $('#schedule-controls').addEventListener('input', showSummary);
  $('#schedule-controls').addEventListener('change', showSummary);
  $('#schedule-timezone').addEventListener('input', showSummary);
  $('#preview-schedule').addEventListener('click', preview);
}

function renderControls() {
  const kind = model.kind;
  let html = '';
  if (['daily', 'weekly', 'monthly'].includes(kind)) {
    html += `<label for="schedule-time">执行时间
      <input type="time" id="schedule-time" name="schedule_time"
      value="${esc(model.time)}" step="60" required></label>`;
  }
  if (kind === 'weekly') html += weekdayFields();
  if (kind === 'monthly') {
    html += `<label for="schedule-day">每月执行日
      <select name="schedule_day" id="schedule-day">
      ${numberOptions(1, 31, model.day, '日')}</select></label>
      <p class="field-hint">选择 29–31 日时，无此日期的月份会跳过。</p>`;
  }
  if (kind === 'hourly') {
    html += `<label for="schedule-minute">每小时执行分钟
      <select name="schedule_minute" id="schedule-minute">
      ${numberOptions(0, 59, model.minute, '分钟')}</select></label>`;
  }
  if (kind === 'interval') html += intervalFields();
  if (kind === 'cron' || kind === 'legacy-interval') {
    const expression = kind === 'cron' &&
      model.expression?.trim().split(/\s+/).length !== 5
      ? '0 9 * * *' : model.expression;
    html += `<label for="schedule-expression">${kind === 'cron'
      ? '高级 Cron 表达式' : '原有间隔'}
      <input id="schedule-expression" name="expression" required
        value="${esc(expression || '0 9 * * *')}"></label>
      <p class="field-hint">常规任务可切换为时间选择，无需编写表达式。</p>`;
  }
  $('#schedule-controls').innerHTML = `<div class="form-grid">${html}</div>`;
  $('#schedule-timezone').disabled =
    kind === 'interval' || kind === 'legacy-interval';
  showSummary();
}

function numberOptions(min, max, selected, suffix) {
  return Array.from({ length: max - min + 1 }, (_, i) => {
    const n = min + i;
    return `<option value="${n}" ${Number(selected) === n
      ? 'selected' : ''}>${n} ${suffix}</option>`;
  }).join('');
}

function weekdayFields() {
  const days = [[1, '周一'], [2, '周二'], [3, '周三'], [4, '周四'],
    [5, '周五'], [6, '周六'], [0, '周日']];
  const checkboxes = days.map(([n, label]) => `<label class="weekday-option">
    <input type="checkbox" name="weekday" value="${n}"
    ${model.weekdays.includes(n) ? 'checked' : ''}>${label}</label>`).join('');
  return `<div class="full"><label>执行星期</label>
    <div class="weekday-picker">${checkboxes}</div></div>`;
}

function intervalFields() {
  const choices = [['s', '秒'], ['m', '分钟'], ['h', '小时'], ['d', '天']]
    .map(([value, label]) => `<option value="${value}"
      ${model.unit === value ? 'selected' : ''}>${label}</option>`).join('');
  return `<label for="schedule-every">每隔
    <input type="number" id="schedule-every" name="schedule_every"
      min="1" step="1" value="${model.every}" required></label>
    <label for="schedule-unit">间隔单位
    <select id="schedule-unit" name="schedule_unit">${choices}</select>
    </label><p class="field-hint full">最短 1 秒，最长 365 天。</p>`;
}

function readValues() {
  const form = $('#editor-form');
  const get = (name) => form.elements[name]?.value;
  const values = {};
  if (get('schedule_time') !== undefined) values.time = get('schedule_time');
  if (get('schedule_day') !== undefined) {
    values.day = Number(get('schedule_day'));
  }
  if (get('schedule_minute') !== undefined) {
    values.minute = Number(get('schedule_minute'));
  }
  if (get('schedule_every') !== undefined) {
    values.every = Number(get('schedule_every'));
    values.unit = get('schedule_unit');
  }
  if (get('expression') !== undefined) values.expression = get('expression');
  if (model.kind === 'weekly') {
    values.weekdays = [...form.querySelectorAll('[name="weekday"]:checked')]
      .map((input) => Number(input.value));
  }
  return values;
}

export function getSchedule() {
  const kind = $('#schedule-kind').value;
  const values = readValues();
  const s = { kind, timezone: $('#schedule-timezone').value };
  if (['daily', 'weekly', 'monthly'].includes(kind)) s.time = values.time;
  if (kind === 'weekly') {
    if (!values.weekdays.length) throw new Error('请至少选择一个执行星期');
    s.weekdays = values.weekdays;
  }
  if (kind === 'monthly') s.day = values.day;
  if (kind === 'hourly') s.minute = values.minute;
  if (kind === 'interval') {
    s.every = values.every; s.unit = values.unit;
  }
  if (kind === 'cron' || kind === 'legacy-interval') {
    s.kind = kind === 'legacy-interval' ? 'interval' : kind;
    s.expression = values.expression;
  }
  return s;
}

function showSummary() {
  previewVersion++;
  $('#preview-times').textContent = '';
  try {
    $('#schedule-summary').textContent = scheduleSummary(getSchedule());
  } catch (err) { $('#schedule-summary').textContent = err.message; }
}

async function preview() {
  const version = ++previewVersion;
  $('#preview-times').textContent = '计算中…';
  try {
    const value = await api('schedules/preview', 'POST', getSchedule());
    if (version !== previewVersion || !$('#preview-times')) return;
    $('#preview-times').textContent = value.times
      .map((t) => time(t)).join(' · ');
  } catch (err) {
    if (version === previewVersion && $('#preview-times')) {
      $('#preview-times').textContent = err.message;
    }
  }
}
