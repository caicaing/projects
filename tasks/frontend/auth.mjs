import { randomBytes, scrypt, scryptSync, timingSafeEqual } from 'node:crypto';
import { promisify } from 'node:util';
import { loadConfig } from './config.mjs';

const derive = promisify(scrypt);
const hour = 60 * 60 * 1000;
const lifetime = 8 * hour;
const idleTime = 30 * 60 * 1000;

export function webConfig(settings = loadConfig()) {
  const web = settings.web;
  const port = Number(web.port);
  if (!Number.isInteger(port) || port < 1 || port > 65535) {
    throw new Error('web.port 必须是有效端口');
  }
  const origin = new URL(web.origin);
  if (!['http:', 'https:'].includes(origin.protocol)
    || origin.username || origin.password || origin.pathname !== '/'
    || origin.search || origin.hash) {
    throw new Error('web.origin 必须是完整的站点源地址');
  }
  const local = ['localhost', '127.0.0.1', '[::1]'];
  if (origin.protocol === 'http:' && !local.includes(origin.hostname)) {
    throw new Error('公网 web.origin 必须使用 HTTPS');
  }
  const username = web.username;
  const password = web.password;
  const token = settings.backend.token;
  if (typeof username !== 'string' || !username || username.length > 128
    || typeof password !== 'string' || password.length < 12
    || password.length > 1024 || typeof token !== 'string'
    || token.length < 16 || token.length > 4096) {
    throw new Error('请配置 web.username、web.password（12–1024 字符）'
      + '和 backend.token（16–4096 字符）');
  }
  const backend = new URL(settings.backend.url);
  if (!['http:', 'https:'].includes(backend.protocol)
    || backend.username || backend.password || backend.pathname !== '/'
    || backend.search || backend.hash) {
    throw new Error('backend.url 必须是 HTTP 或 HTTPS 源地址');
  }
  if (typeof web.host !== 'string' || !web.host) {
    throw new Error('web.host 不能为空');
  }
  return { port, origin, username, password, token, backend, host: web.host };
}

export function json(res, status, value) {
  res.writeHead(status, { 'Content-Type': 'application/json; charset=utf-8' });
  res.end(JSON.stringify(value));
}

export function readBody(req, limit) {
  return new Promise((resolve, reject) => {
    let size = 0;
    let failed = false;
    let chunks = [];
    req.on('data', (chunk) => {
      if (failed) return;
      size += chunk.length;
      if (size > limit) {
        failed = true;
        chunks = [];
        reject(Object.assign(new Error('请求内容过大'), { status: 413 }));
      } else chunks.push(chunk);
    });
    req.on('end', () => resolve(Buffer.concat(chunks)));
    req.on('error', reject);
    req.on('aborted', () => reject(new Error('请求已中断')));
  });
}

export function createAuth(config) {
  const salt = randomBytes(32);
  const passwordHash = scryptSync(config.password, salt, 32);
  // Only the derived credential is retained after startup.
  config.password = undefined;
  const sessions = new Map();
  const attempts = new Map();
  const secure = config.origin.protocol === 'https:';
  const cookieName = secure ? '__Host-taskflow-session' : 'taskflow-session';
  let inflight = 0;

  function cleanup() {
    const now = Date.now();
    for (const [key, value] of sessions) {
      if (value.expires <= now || value.touched + idleTime <= now) {
        sessions.delete(key);
      }
    }
    for (const [key, value] of attempts) {
      if (value.until <= now) attempts.delete(key);
    }
  }
  const timer = setInterval(cleanup, 60000);
  timer.unref();

  function sameOrigin(req, mutation = false) {
    const site = req.headers['sec-fetch-site'];
    if (req.headers.host !== config.origin.host
      || (site && !['same-origin', 'none'].includes(site))) return false;
    const origin = req.headers.origin;
    return origin ? origin === config.origin.origin : !mutation;
  }

  function cookieID(req) {
    const cookies = (req.headers.cookie || '').split(';');
    const values = cookies.map((c) => c.trim())
      .filter((c) => c.startsWith(cookieName + '='));
    if (values.length !== 1) return '';
    return values[0].slice(cookieName.length + 1);
  }

  function session(req) {
    const id = cookieID(req);
    const value = sessions.get(id);
    const now = Date.now();
    if (!value) return null;
    if (value.expires <= now || value.touched + idleTime <= now) {
      sessions.delete(id);
      return null;
    }
    value.touched = now;
    return value;
  }

  function setCookie(res, id, maxAge) {
    const flags = secure ? '; Secure' : '';
    res.setHeader('Set-Cookie', `${cookieName}=${id}; Path=/; HttpOnly; `
      + `SameSite=Strict; Max-Age=${maxAge}${flags}`);
  }

  function consume(key, max, window) {
    const now = Date.now();
    let value = attempts.get(key);
    if (!value || value.until <= now) {
      if (attempts.size >= 10000) return false;
      value = { count: 0, until: now + window };
      attempts.set(key, value);
    }
    value.count++;
    return value.count <= max;
  }

  async function login(req, res) {
    if (!consume('global', 120, 60000)
      || !consume(`ip:${req.socket.remoteAddress}`, 10, 15 * 60000)
      || inflight >= 4) {
      res.setHeader('Retry-After', '900');
      return json(res, 429, { error: '登录过于频繁，请稍后重试' });
    }
    if (req.headers['content-type']?.split(';')[0] !== 'application/json') {
      return json(res, 415, { error: '请使用 application/json' });
    }
    inflight++;
    try {
      const input = JSON.parse((await readBody(req, 4096)).toString());
      if (!input || typeof input.username !== 'string'
        || typeof input.password !== 'string' || input.password.length > 1024
        || input.username.length > 128) {
        return json(res, 400, { error: '登录参数错误' });
      }
      const actual = await derive(input.password, salt, 32);
      if (!timingSafeEqual(actual, passwordHash)
        || input.username !== config.username) {
        return json(res, 401, { error: '账号或密码错误' });
      }
      cleanup();
      if (sessions.size >= 1000) {
        return json(res, 503, { error: '会话数量已满，请稍后重试' });
      }
      sessions.delete(cookieID(req));
      const id = randomBytes(32).toString('hex');
      const value = { csrf: randomBytes(32).toString('hex'),
        expires: Date.now() + lifetime, touched: Date.now() };
      sessions.set(id, value);
      setCookie(res, id, lifetime / 1000);
      return json(res, 200, { username: config.username, csrf: value.csrf });
    } finally { inflight--; }
  }

  function requireSession(req, res, mutation = false) {
    const value = session(req);
    if (!value) {
      json(res, 401, { error: '请先登录管理台' });
      return null;
    }
    if (mutation && (!sameOrigin(req, true)
      || req.headers['x-csrf-token'] !== value.csrf)) {
      json(res, 403, { error: '请求来源或 CSRF 校验失败' });
      return null;
    }
    return value;
  }

  async function route(req, res, path) {
    const methods = { '/auth/login': 'POST', '/auth/session': 'GET',
      '/auth/logout': 'POST' };
    if (!methods[path]) return json(res, 404, { error: '接口不存在' });
    if (req.method !== methods[path]) {
      res.setHeader('Allow', methods[path]);
      return json(res, 405, { error: '请求方法不支持' });
    }
    if (!sameOrigin(req, req.method !== 'GET')) {
      return json(res, 403, { error: '禁止跨站请求' });
    }
    if (path === '/auth/login') return login(req, res);
    const value = requireSession(req, res, req.method !== 'GET');
    if (!value) return;
    if (path === '/auth/session') {
      return json(res, 200, { username: config.username, csrf: value.csrf });
    }
    sessions.delete(cookieID(req));
    setCookie(res, '', 0);
    return json(res, 200, { ok: true });
  }
  return { sameOrigin, session, requireSession, route };
}
