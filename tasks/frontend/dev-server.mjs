import http from 'node:http';
import { readFile } from 'node:fs/promises';
import { dirname, resolve, extname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { webConfig, createAuth, json } from './auth.mjs';
import { proxy } from './proxy.mjs';

const root = dirname(fileURLToPath(import.meta.url));
const config = webConfig();
const auth = createAuth(config);
const mime = { '.html': 'text/html', '.js': 'text/javascript',
  '.css': 'text/css', '.svg': 'image/svg+xml' };
const policy = "default-src 'self'; script-src 'self'; style-src 'self'; "
  + "img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; "
  + "base-uri 'none'; form-action 'self'; object-src 'none'";
const publicFiles = new Set(['/login.html', '/login.js', '/style.css',
  '/forms.css', '/favicon.svg']);

function securityHeaders(res) {
  res.setHeader('Cache-Control', 'no-store');
  res.setHeader('Content-Security-Policy', policy);
  res.setHeader('X-Content-Type-Options', 'nosniff');
  res.setHeader('X-Frame-Options', 'DENY');
  res.setHeader('Referrer-Policy', 'no-referrer');
}

async function staticFile(req, res, pathname) {
  if (!['GET', 'HEAD'].includes(req.method)) {
    res.writeHead(405, { Allow: 'GET, HEAD' });
    return res.end();
  }
  const source = pathname.startsWith('/src/');
  const base = resolve(root, source ? 'src' : 'public');
  const relative = source ? pathname.slice(4) : pathname;
  const filename = relative === '/' ? '/index.html' : relative;
  const path = resolve(base, '.' + filename);
  if (!path.startsWith(base + '/') || !mime[extname(path)]) {
    res.writeHead(404); return res.end('Not found');
  }
  try {
    const body = await readFile(path);
    res.writeHead(200, {
      'Content-Type': mime[extname(path)] + '; charset=utf-8',
      'Content-Length': body.length,
    });
    res.end(req.method === 'HEAD' ? undefined : body);
  } catch { res.writeHead(404); res.end('Not found'); }
}

const server = http.createServer(async (req, res) => {
  securityHeaders(res);
  try {
    const url = new URL(req.url, 'http://localhost');
    if (!auth.sameOrigin(req)) {
      return json(res, 403, { error: '站点地址不匹配或禁止跨站请求' });
    }
    if (url.pathname.startsWith('/auth/')) {
      return await auth.route(req, res, url.pathname);
    }
    if (url.pathname.startsWith('/api/') || url.pathname === '/healthz') {
      const mutation = !['GET', 'HEAD'].includes(req.method);
      if (!auth.requireSession(req, res, mutation)) return;
      return await proxy(req, res, url, config);
    }
    const pathname = decodeURIComponent(url.pathname);
    if (!publicFiles.has(pathname) && !auth.session(req)) {
      if (['/', '/index.html'].includes(pathname)) {
        res.writeHead(303, { Location: '/login.html' });
        return res.end();
      }
      return json(res, 401, { error: '请先登录管理台' });
    }
    await staticFile(req, res, pathname);
  } catch (err) {
    if (!res.headersSent && !res.destroyed) {
      json(res, err.status || 400, { error: err.status
        ? err.message : '请求参数错误' });
    }
  }
});
server.requestTimeout = 35000;
server.headersTimeout = 10000;
server.maxHeadersCount = 100;
server.on('error', (err) => { console.error(err.message); process.exit(1); });
server.listen(config.port, config.host, () => {
  console.log(`管理台：${config.origin.origin} → ${config.backend.origin}`);
});
for (const signal of ['SIGINT', 'SIGTERM']) {
  process.on(signal, () => {
    server.close(() => process.exit(0));
    server.closeIdleConnections?.();
    setTimeout(() => server.closeAllConnections(), 5000).unref();
  });
}
