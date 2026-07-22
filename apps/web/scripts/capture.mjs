import { mkdir, stat } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright-core';
import { startLocalPreview, stopLocalPreview } from './local-preview.mjs';

const executablePath = process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH || 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe';
const baseUrl = process.env.UI_BASE_URL || 'http://127.0.0.1:4173';
const previewServer = await startLocalPreview();
const outputDir = fileURLToPath(new URL('../artifacts/', import.meta.url));
await mkdir(outputDir, { recursive: true });

const browser = await chromium.launch({ executablePath, headless: true });

async function openNavigation(page, label) {
  const target = page.getByRole('button', { name: label, exact: true });
  if (!(await target.isVisible())) await page.getByRole('button', { name: '打开菜单' }).click();
  await target.click();
  await page.waitForTimeout(120);
}

async function verifyLayout(page, name) {
  const dimensions = await page.evaluate(() => ({
    clientWidth: document.documentElement.clientWidth,
    scrollWidth: document.documentElement.scrollWidth,
    textLength: document.body.innerText.trim().length,
  }));
  if (dimensions.scrollWidth > dimensions.clientWidth + 2) throw new Error(`${name} 根页面横向溢出 ${dimensions.scrollWidth - dimensions.clientWidth}px`);
  if (dimensions.textLength < 200) throw new Error(`${name} 页面内容疑似空白`);
}

async function captureSet(viewport, suffix) {
  const page = await browser.newPage({ viewport, deviceScaleFactor: 1 });
  const runtimeErrors = [];
  page.on('pageerror', (error) => runtimeErrors.push(error.message));
  await page.route('**/api/**', (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const fulfill = (data, status = 200) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(data) });
    if (url.pathname === '/api/me') return fulfill({ success: true, data: { id: 1, username: 'modelroute', display_name: '平台管理员', role: 100, status: 1, group: 'default', quota: 128642000, used_quota: 4268000, request_count: 2846, is_merchant: true, merchant_id: 31, merchant_status: 'active' } });
    if (url.pathname === '/api/keys' && request.method() === 'GET') return fulfill({ success: true, data: { page: 1, page_size: 20, total: 1, items: [{ id: 9, name: '生产主密钥', key: 'abcd**********wxyz', status: 1, expired_time: -1, unlimited_quota: true }] } });
    if (url.pathname === '/api/merchant/channels') return fulfill({ success: true, data: { merchant: { merchant_id: 31, user_id: 1, merchant_status: 'active', is_merchant: true }, items: [{ id: 51, merchant_id: 31, review_id: 21, newapi_channel_id: 41, channel_name: 'OpenAI 主线路', channel_type: 1, base_url: 'https://api.merchant.example', models: ['gpt-4o-mini'], credential_fingerprint: 'a1b2c3...d4e5f6', status: 'active', created_at: '2026-07-22T14:00:00Z' }], reviews: [{ id: 21, user_id: 1, application_id: 11, channel_name: 'OpenAI 主线路', channel_type: 1, base_url: 'https://api.merchant.example', requested_models: ['gpt-4o-mini'], verified_models: ['gpt-4o-mini'], status: 'approved', credential_configured: false, created_at: '2026-07-22T14:00:00Z' }] } });
    if (url.pathname === '/api/admin/channel-reviews') return fulfill({ success: true, data: { page: 1, page_size: 100, total: 1, items: [{ id: 22, user_id: 8, channel_name: 'Anthropic 备用线路', channel_type: 14, base_url: 'https://api.anthropic.example', requested_models: ['claude-sonnet-4-5'], status: 'pending', credential_configured: true, created_at: '2026-07-22T15:00:00Z' }] } });
    if (url.pathname === '/api/admin/channels') return fulfill({ success: true, data: { page: 1, page_size: 100, total: 1, items: [{ id: 51, merchant_id: 31, review_id: 21, newapi_channel_id: 41, channel_name: 'OpenAI 主线路', channel_type: 1, base_url: 'https://api.merchant.example', models: ['gpt-4o-mini'], credential_fingerprint: 'a1b2c3...d4e5f6', status: 'active', created_at: '2026-07-22T14:00:00Z' }] } });
    return fulfill({ success: false, error: { code: 'not_mocked', message: url.pathname } }, 404);
  });
  await page.goto(baseUrl, { waitUntil: 'networkidle' });

  const captures = [
    ['模型市场', 'market'],
    ['控制台', 'console'],
    ['商户中心', 'merchant'],
    ['管理后台', 'admin'],
  ];

  for (const [label, filename] of captures) {
    if (filename !== 'market') await openNavigation(page, label);
    if (filename === 'admin') await page.getByRole('button', { name: '支付与充值' }).click();
    await verifyLayout(page, `${filename}-${suffix}`);
    const path = `${outputDir}${filename}-${suffix}.png`;
    await page.screenshot({ path, fullPage: true });
    if ((await stat(path)).size < 20_000) throw new Error(`${filename}-${suffix} 截图文件过小`);
  }

  await page.getByRole('button', { name: '编辑配置' }).click();
  let path = `${outputDir}payment-config-${suffix}.png`;
  await page.screenshot({ path, fullPage: false });
  if ((await stat(path)).size < 20_000) throw new Error(`payment-config-${suffix} 截图文件过小`);
  await page.getByRole('button', { name: '关闭', exact: true }).click();

  await page.getByRole('button', { name: '渠道与探活' }).click();
  await page.getByText('Anthropic 备用线路').waitFor();
  await verifyLayout(page, `admin-channels-${suffix}`);
  path = `${outputDir}admin-channels-${suffix}.png`;
  await page.screenshot({ path, fullPage: true });
  if ((await stat(path)).size < 20_000) throw new Error(`admin-channels-${suffix} 截图文件过小`);

  await openNavigation(page, '控制台');
  await page.getByRole('button', { name: '编辑路由策略' }).first().click();
  path = `${outputDir}route-modal-${suffix}.png`;
  await page.screenshot({ path, fullPage: false });
  if ((await stat(path)).size < 20_000) throw new Error(`route-modal-${suffix} 截图文件过小`);
  await page.getByRole('button', { name: '关闭', exact: true }).click();

  if (runtimeErrors.length) throw new Error(`${suffix} 存在运行时错误：${runtimeErrors.join('; ')}`);
  await page.close();
}

try {
  await captureSet({ width: 1440, height: 1000 }, 'desktop');
  await captureSet({ width: 390, height: 844 }, 'mobile');
  console.log('截图完成：8 个业务视图、2 个渠道管理视图、4 个关键弹窗，根页面无横向溢出。');
} finally {
  await browser.close();
  await stopLocalPreview(previewServer);
}
