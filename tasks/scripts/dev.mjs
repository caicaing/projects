import { spawn } from 'node:child_process';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { webConfig } from '../frontend/auth.mjs';
import { loadConfig, readConfig } from '../frontend/config.mjs';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
// Validate credentials before starting any scheduled tasks.
const paths = {
  '--backend-config': resolve(root, 'backend/conf.yaml'),
  '--frontend-config': resolve(root, 'frontend/conf.yaml'),
};
const args = process.argv.slice(2);
const checkOnly = args[0] === '--check';
if (checkOnly) args.shift();
const seen = new Set();
for (let i = 0; i < args.length; i += 2) {
  const name = args[i];
  if (!Object.hasOwn(paths, name) || !args[i + 1] || seen.has(name)) {
    throw new Error('用法：node scripts/dev.mjs [--check] '
      + '[--backend-config 文件] [--frontend-config 文件]');
  }
  seen.add(name);
  paths[name] = resolve(args[i + 1]);
}
const backendPath = paths['--backend-config'];
const frontendPath = paths['--frontend-config'];
const settings = loadConfig(frontendPath);
const api = webConfig(settings).backend;
const backend = readConfig(backendPath).backend;
if (api.protocol !== 'http:' || api.hostname !== '127.0.0.1') {
  throw new Error('联合启动仅支持本机 HTTP 后端，远程 API 请独立启动前端');
}
if (backend?.addr !== `127.0.0.1:${api.port || 80}`) {
  throw new Error('联合启动要求后端 backend.addr '
    + '与前端 backend.url 的地址一致');
}
if (backend.token !== settings.backend.token) {
  throw new Error('前后端配置中的 backend.token 必须一致');
}
if (checkOnly) process.exit(0);
const children = [
  spawn(resolve(root, 'backend/bin/taskflow'),
    ['-config', backendPath], { cwd: root, stdio: 'inherit' }),
  spawn(process.execPath, ['dev-server.mjs', '--config', frontendPath],
    { cwd: resolve(root, 'frontend'), stdio: 'inherit' }),
];
let stopping = false;
let exitCode = 0;

function stop(code = 0) {
  if (stopping) return;
  stopping = true;
  exitCode = code;
  children.forEach((child) => child.kill('SIGTERM'));
  const timer = setTimeout(() => {
    children.forEach((child) => child.kill('SIGKILL'));
  }, 20000);
  timer.unref();
}

for (const child of children) {
  child.on('error', (err) => { console.error(err.message); stop(1); });
  child.on('exit', (code) => stop(code || 0));
}
process.on('SIGINT', () => stop());
process.on('SIGTERM', () => stop());
process.on('exit', () => { process.exitCode = exitCode; });
