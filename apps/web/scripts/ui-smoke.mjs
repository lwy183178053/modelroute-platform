import { chromium } from 'playwright-core';
import { startLocalPreview, stopLocalPreview } from './local-preview.mjs';

const executablePath =
  process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH ||
  'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe';
const baseUrl = process.env.UI_BASE_URL || 'http://127.0.0.1:4173';
const previewServer = await startLocalPreview();

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

const browser = await chromium.launch({ executablePath, headless: true });
const page = await browser.newPage({ viewport: { width: 1440, height: 1000 } });

async function mockAdminAPI(targetPage, state) {
  await targetPage.route('**/api/**', async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const json = (data, status = 200) => route.fulfill({
      status,
      contentType: 'application/json',
      body: JSON.stringify(data),
    });
    if (url.pathname === '/api/me') {
      return json({ success: true, data: { id: 1, username: 'modelroute', display_name: '平台管理员', role: 100, status: 1, group: 'default', quota: 128642000, used_quota: 4268000 } });
    }
    if (url.pathname === '/api/keys' && request.method() === 'GET') {
      return json({ success: true, data: { page: 1, page_size: 20, total: 1, items: [{ id: 9, name: '真实主密钥', key: 'abcd**********wxyz', status: 1, created_time: 1784690000, accessed_time: 1784692000, expired_time: -1, unlimited_quota: true }] } });
    }
    if (url.pathname === '/api/admin/users') {
      return json({ success: true, data: { page: 1, page_size: 20, total: 2, items: [
        { id: 1, username: 'modelroute', display_name: '平台管理员', email: '', role: 100, status: 1, quota: 128642000, used_quota: 4268000, request_count: 2846 },
        { id: 7, username: 'alice', display_name: 'Alice', email: 'alice@example.com', role: 1, status: 1, quota: 500000, used_quota: 120000, request_count: 18 },
      ] } });
    }
    if (url.pathname === '/api/admin/audit-logs') {
      return json({ success: true, data: { page: 1, page_size: 20, total: 1, items: [{ id: 3, actor_user_id: 1, actor_role: '100', action: 'user.disable', object_type: 'user', object_id: '7', result: 'success', source_ip: '127.0.0.1', created_at: '2026-07-22T12:00:00Z' }] } });
    }
    if (url.pathname === '/api/keys/9/reveal') {
      state.revealCalls += 1;
      return json({ success: true, data: { key: 'sk-full-secret-key' } });
    }
    if (url.pathname.startsWith('/api/admin/users/') || url.pathname.startsWith('/api/keys')) {
      return json({ success: true, data: null });
    }
    return json({ success: false, error: { code: 'not_mocked', message: url.pathname } }, 404);
  });
}

try {
  const adminMockState = { revealCalls: 0 };
  await mockAdminAPI(page, adminMockState);
  await page.goto(baseUrl, { waitUntil: 'networkidle' });
  await page.getByRole('heading', { name: '模型服务市场' }).waitFor();
  await page.getByText('平台管理员').waitFor();

  const views = [
    ['控制台', '账户概览'],
    ['商户中心', '商户经营概览'],
    ['管理后台', '运营管理'],
    ['模型市场', '模型服务市场'],
  ];

  for (const [navigation, heading] of views) {
    await page.getByRole('button', { name: navigation, exact: true }).click();
    await page.getByRole('heading', { name: heading }).waitFor();
  }

  await page.getByRole('button', { name: '控制台', exact: true }).click();
  await page.getByText('真实主密钥').waitFor();
  assert(await page.getByRole('checkbox', { name: '启用真实主密钥' }).isChecked(), '有效密钥的启用开关未选中');
  await page.getByRole('button', { name: '复制' }).first().click();
  await page.getByText('密钥已复制').waitFor();
  assert(adminMockState.revealCalls === 1, '复制掩码密钥时未读取完整密钥');
  await page.getByRole('button', { name: '编辑路由策略' }).first().click();
  await page.getByRole('dialog', { name: '编辑路由策略' }).waitFor();
  await page.getByLabel('固定商户').check();
  assert(await page.getByText('固定商户范围').isVisible(), '固定路由商户选择未显示');
  await page.getByRole('button', { name: '保存策略' }).click();

  await page.getByRole('button', { name: '商户中心', exact: true }).click();
  await page.getByRole('button', { name: '调整报价' }).first().click();
  await page.getByRole('dialog', { name: '调整模型报价' }).waitFor();
  await page.getByRole('button', { name: '提交调价' }).click();

  await page.getByRole('button', { name: '管理后台', exact: true }).click();
  await page.getByRole('button', { name: '用户管理' }).click();
  await page.getByText('alice@example.com').waitFor();
  await page.getByRole('button', { name: '审计日志' }).click();
  await page.getByText('user.disable').waitFor();
  await page.getByRole('button', { name: '支付与充值' }).click();
  await page.getByRole('heading', { name: '易支付设置' }).waitFor();
  assert(
    await page.getByText('https://zpayz.cn/').isVisible(),
    '易支付网关未显示',
  );
  const bodyText = await page.locator('body').innerText();
  assert(!/KEY[^\n]{0,24}[A-Za-z0-9]{24,}/.test(bodyText), '页面疑似显示了完整支付密钥');
  await page.getByRole('button', { name: '测试连接' }).click();
  await page.getByText('网关连接正常').waitFor();

  const anonymousPage = await browser.newPage({ viewport: { width: 390, height: 844 } });
  let anonymousLoggedIn = false;
  await anonymousPage.route('**/api/**', (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const fulfill = (data, status = 200) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(data) });
    if (url.pathname === '/api/me' && anonymousLoggedIn) return fulfill({ success: true, data: { id: 7, username: 'alice', display_name: 'Alice', role: 1, status: 1, group: 'default', quota: 500000, used_quota: 120000 } });
    if (url.pathname === '/api/me') return fulfill({ success: false, error: { code: 'authentication_required', message: '请先登录' } }, 401);
    if (url.pathname === '/api/auth/login' && request.method() === 'POST') {
      anonymousLoggedIn = true;
      return fulfill({ success: true, data: { id: 7, username: 'alice', display_name: 'Alice', role: 1, status: 1, group: 'default' } });
    }
    if (url.pathname === '/api/keys') return fulfill({ success: true, data: { page: 1, page_size: 20, total: 0, items: [] } });
    return fulfill({ success: false, error: { code: 'not_mocked', message: url.pathname } }, 404);
  });
  await anonymousPage.goto(baseUrl, { waitUntil: 'networkidle' });
  await anonymousPage.getByRole('button', { name: '登录' }).waitFor();
  assert(await anonymousPage.getByRole('button', { name: '管理后台', exact: true }).count() === 0, '未登录状态显示了管理后台入口');
  assert(await anonymousPage.getByRole('button', { name: '商户中心', exact: true }).count() === 0, '未登录状态显示了商户中心入口');
  await anonymousPage.getByRole('button', { name: '登录' }).click();
  const loginDialog = anonymousPage.getByRole('dialog', { name: '账号登录' });
  await loginDialog.waitFor();
  await loginDialog.getByLabel('账号', { exact: true }).fill('alice');
  await loginDialog.getByLabel('密码', { exact: true }).fill('secret123');
  await loginDialog.getByRole('button', { name: '登录' }).click();
  await anonymousPage.getByRole('heading', { name: '账户概览' }).waitFor();
  await anonymousPage.getByText('UID 7 · 普通账号').waitFor();
  await anonymousPage.getByRole('main').getByText('$1.00').waitFor();
  assert(await anonymousPage.getByRole('button', { name: '管理后台', exact: true }).count() === 0, '普通用户显示了管理后台入口');
  assert(await anonymousPage.getByRole('button', { name: '商户中心', exact: true }).count() === 0, '普通用户显示了商户中心入口');
  await anonymousPage.close();

  const resetPage = await browser.newPage({ viewport: { width: 390, height: 844 } });
  await resetPage.route('**/api/**', (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const fulfill = (data, status = 200) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(data) });
    if (url.pathname === '/api/me') return fulfill({ success: false, error: { code: 'authentication_required', message: '请先登录' } }, 401);
    if (url.pathname === '/api/auth/password-reset/confirm' && request.method() === 'POST') return fulfill({ success: true, data: 'TempPass1234' });
    return fulfill({ success: false, error: { code: 'not_mocked', message: url.pathname } }, 404);
  });
  await resetPage.goto(`${baseUrl}/?email=alice%40example.com&token=reset-token`, { waitUntil: 'networkidle' });
  const resetDialog = resetPage.getByRole('dialog', { name: '确认密码重置' });
  await resetDialog.getByRole('button', { name: '确认重置' }).click();
  await resetDialog.getByText('TempPass1234').waitFor();
  await resetPage.close();

  console.log('UI 冒烟测试通过：角色导航、真实账号数据、路由策略、报价和支付设置交互正常。');
} finally {
  await browser.close();
  await stopLocalPreview(previewServer);
}
