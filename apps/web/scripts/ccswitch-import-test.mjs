import assert from 'node:assert/strict';
import { buildCCSwitchImportURL, ccSwitchUsageScript } from '../src/ccswitch.js';

const url = buildCCSwitchImportURL({
  baseUrl: 'https://aggregate.lwylink.xyz/',
  apiKey: 'sk-test-secret',
  model: 'gpt-5.5',
  provider: 'OpenAI',
  keyName: '开发密钥',
});
const parsed = new URL(url);
assert.equal(parsed.protocol, 'ccswitch:');
assert.equal(parsed.host, 'v1');
assert.equal(parsed.pathname, '/import');
assert.equal(parsed.searchParams.get('resource'), 'provider');
assert.equal(parsed.searchParams.get('app'), 'codex');
assert.equal(parsed.searchParams.get('endpoint'), 'https://aggregate.lwylink.xyz/v1');
assert.equal(parsed.searchParams.get('homepage'), 'https://aggregate.lwylink.xyz');
assert.equal(parsed.searchParams.get('apiKey'), 'sk-test-secret');
assert.equal(parsed.searchParams.get('model'), 'gpt-5.5');
assert.equal(parsed.searchParams.get('usageEnabled'), 'true');
assert.equal(parsed.searchParams.get('usageAutoInterval'), '30');
assert.equal(Buffer.from(parsed.searchParams.get('usageScript'), 'base64').toString('utf8'), ccSwitchUsageScript);
assert.match(ccSwitchUsageScript, /\{\{baseUrl\}\}\/usage/);
assert.doesNotMatch(ccSwitchUsageScript, /\/v1\/v1\/usage/);
assert.match(ccSwitchUsageScript, /Bearer \{\{apiKey\}\}/);
assert.match(ccSwitchUsageScript, /"CNY"/);

console.log('CC Switch 导入链接测试通过');
