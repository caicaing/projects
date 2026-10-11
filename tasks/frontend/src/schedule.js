const names = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'];
const units = { s: '秒', m: '分钟', h: '小时', d: '天' };

export function scheduleModel(schedule) {
  if (!schedule) return { kind: 'daily', time: '09:00',
    timezone: 'Asia/Shanghai', weekdays: [1], day: 1, every: 5, unit: 'm' };
  const model = { time: '09:00', minute: 0, weekdays: [1], day: 1,
    every: 5, unit: 'm', ...schedule };
  if (schedule.kind === 'cron') return detectCalendar(model);
  if (schedule.kind === 'interval' && !schedule.every) {
    const duration = durationSeconds(schedule.expression);
    for (const [unit, seconds] of [['d', 86400], ['h', 3600],
      ['m', 60], ['s', 1]]) {
      if (duration && duration % seconds === 0) {
        return { ...model, every: duration / seconds, unit };
      }
    }
    return { ...model, kind: 'legacy-interval' };
  }
  return model;
}

function durationSeconds(raw) {
  let total = 0, matched = '';
  for (const piece of String(raw).matchAll(/(\d+(?:\.\d+)?)(h|m|s)/g)) {
    total += Number(piece[1]) * ({ h: 3600, m: 60, s: 1 })[piece[2]];
    matched += piece[0];
  }
  return matched === raw ? total : null;
}

function detectCalendar(model) {
  const parts = model.expression.trim().split(/\s+/);
  if (parts.length !== 5) return model;
  const [minute, hour, day, month, week] = parts;
  if (!/^\d+$/.test(minute) || month !== '*') return model;
  if (hour === '*' && day === '*' && week === '*') {
    return { ...model, kind: 'hourly', minute: Number(minute) };
  }
  if (!/^\d+$/.test(hour)) return model;
  const time = `${hour.padStart(2, '0')}:${minute.padStart(2, '0')}`;
  if (day === '*' && week === '*') return { ...model, kind: 'daily', time };
  if (day === '*' && /^[0-7](,[0-7])*$/.test(week)) {
    return { ...model, kind: 'weekly', time,
      weekdays: [...new Set(week.split(',').map((d) => Number(d) % 7))] };
  }
  if (/^\d+$/.test(day) && week === '*') {
    return { ...model, kind: 'monthly', time, day: Number(day) };
  }
  return model;
}

export function scheduleSummary(schedule) {
  const s = scheduleModel(schedule);
  if (s.kind === 'daily') return `每天 ${s.time}`;
  if (s.kind === 'weekly') {
    return `${s.weekdays.map((d) => names[d]).join('、')} ${s.time}`;
  }
  if (s.kind === 'monthly') return `每月 ${s.day} 日 ${s.time}`;
  if (s.kind === 'hourly') return `每小时第 ${s.minute} 分钟`;
  if (s.kind === 'interval') return `每 ${s.every} ${units[s.unit]}`;
  if (s.kind === 'legacy-interval') return `每隔 ${s.expression}`;
  return `自定义计划：${s.expression}`;
}
