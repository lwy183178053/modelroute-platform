import assert from 'node:assert/strict';
import { configureQuotaPerUnit, formatQuota, getQuotaPerUnit } from '../src/api.js';

configureQuotaPerUnit(750_000);
assert.equal(getQuotaPerUnit(), 750_000);
assert.equal(formatQuota(750_000), '¥1.00');
assert.equal(formatQuota(37_500_000), '¥50.00');
assert.throws(() => configureQuotaPerUnit(500_001), /额度单位/);

console.log('前端动态额度单位测试通过');
