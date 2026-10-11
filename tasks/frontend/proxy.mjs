import http from 'node:http';
import https from 'node:https';
import { json, readBody } from './auth.mjs';

export async function proxy(req, res, url, config) {
  const body = await readBody(req, 1 << 20);
  if (res.destroyed) return;
  const target = new URL(config.backend.origin);
  target.pathname = url.pathname;
  target.search = url.search;
  const transport = target.protocol === 'https:' ? https : http;
  // Never forward browser credentials or untrusted proxy headers upstream.
  const headers = {
    host: config.origin.host,
    authorization: `Bearer ${config.token}`,
    origin: config.origin.origin,
    'content-length': body.length,
    accept: 'application/json',
  };
  if (req.headers['content-type']) {
    headers['content-type'] = req.headers['content-type'];
  }
  const upstream = transport.request(target, {
    method: req.method, headers,
  }, (response) => {
    // Allow only API content; upstream cookies and redirects stay private.
    res.writeHead(response.statusCode, {
      'Content-Type': response.headers['content-type'] || 'application/json',
    });
    response.pipe(res);
    response.on('error', () => res.destroy());
  });
  upstream.setTimeout(30000, () => upstream.destroy());
  upstream.on('error', () => {
    if (res.headersSent) return res.destroy();
    json(res, 502, { error: '后端连接失败，请检查 API 服务' });
  });
  res.on('close', () => {
    if (!res.writableEnded) upstream.destroy();
  });
  upstream.end(body);
}
