import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { isAlias, parseDocument, visit } from 'yaml';

export function configPath(args = process.argv.slice(2)) {
  if (!args.length) {
    return fileURLToPath(new URL('./conf.yaml', import.meta.url));
  }
  if (args.length !== 2 || args[0] !== '--config' || !args[1]) {
    throw new Error('用法：node dev-server.mjs [--config 配置文件]');
  }
  return args[1];
}

const fields = {
  backend: ['url', 'token'],
  web: ['host', 'port', 'origin', 'username', 'password'],
};

export function readConfig(path) {
  const source = readFileSync(path, 'utf8');
  if (Buffer.byteLength(source) > 1 << 20) {
    throw new Error('conf.yaml 不能超过 1 MiB');
  }
  const document = parseDocument(source);
  if (document.errors.length || document.warnings.length) {
    throw new Error('conf.yaml 格式错误');
  }
  visit(document, (_, node) => {
    if (isAlias(node) || node?.anchor) {
      throw new Error('配置不支持 YAML 锚点或别名');
    }
  });
  const config = document.toJS({ maxAliasCount: 0 });
  if (!config || typeof config !== 'object' || Array.isArray(config)) {
    throw new Error('conf.yaml 必须是配置对象');
  }
  return config;
}

export function loadConfig(path = configPath()) {
  const config = readConfig(path);
  for (const key of Object.keys(config)) {
    if (!Object.hasOwn(fields, key)) throw new Error(`未知配置段：${key}`);
  }
  for (const [section, allowed] of Object.entries(fields)) {
    const value = config[section];
    if (!value || typeof value !== 'object' || Array.isArray(value)) {
      throw new Error(`配置缺少 ${section} 对象`);
    }
    for (const key of Object.keys(value)) {
      if (!allowed.includes(key)) throw new Error(`未知字段：${section}.${key}`);
      if (!['string', 'number'].includes(typeof value[key])) {
        throw new Error(`${section}.${key} 必须是字符串或数字`);
      }
    }
  }
  return config;
}
