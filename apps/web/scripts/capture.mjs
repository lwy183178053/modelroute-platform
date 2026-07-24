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

async function openActionMenu(page, index = 0) {
  await page.getByRole('button', { name: '操作', exact: true }).nth(index).click();
  const menu = page.getByRole('menu').last();
  await menu.waitFor();
  return menu;
}

async function clickRowAction(page, label, index = 0) {
  const menu = await openActionMenu(page, index);
  await menu.getByRole('menuitem', { name: label, exact: true }).click();
}

async function verifyLayout(page, name, minimumTextLength = 200) {
  const dimensions = await page.evaluate(() => ({
    clientWidth: document.documentElement.clientWidth,
    scrollWidth: document.documentElement.scrollWidth,
    textLength: document.body.innerText.trim().length,
  }));
  if (dimensions.scrollWidth > dimensions.clientWidth + 2) throw new Error(`${name} 根页面横向溢出 ${dimensions.scrollWidth - dimensions.clientWidth}px`);
  if (dimensions.textLength < minimumTextLength) throw new Error(`${name} 页面内容疑似空白`);
}

async function captureSet(viewport, suffix) {
  const page = await browser.newPage({ viewport, deviceScaleFactor: 1 });
  const runtimeErrors = [];
  page.on('pageerror', (error) => runtimeErrors.push(error.message));
  await page.route('**/api/**', (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const fulfill = (data, status = 200) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(data) });
    if (url.pathname === '/api/version') return fulfill({ success: true, data: { version: 'capture', quota_per_unit: 750000 } });
    if (url.pathname === '/api/me') return fulfill({ success: true, data: { id: 1, username: 'modelroute', display_name: '平台管理员', role: 100, status: 1, quota: 128642000, used_quota: 4268000, request_count: 2846, is_merchant: true, merchant_id: 31, merchant_status: 'active' } });
    if (url.pathname === '/api/announcements') return fulfill({ success: true, data: { unread_count: 1, items: [{ id: 9, title: '商户提现安排', body: '本周五统一处理人工提现，请核对收款信息。', audiences: ['admin'], pinned: true, status: 'published', is_read: false, publish_at: '2026-07-22T08:00:00Z' }] } });
    if (url.pathname === '/api/admin/announcements') return fulfill({ success: true, data: [{ id: 9, title: '商户提现安排', body: '本周五统一处理人工提现，请核对收款信息。', audiences: ['admin'], pinned: true, status: 'published', publish_at: '2026-07-22T08:00:00Z', created_at: '2026-07-22T07:00:00Z' }] });
    if (url.pathname === '/api/models' || url.pathname === '/api/admin/models') return fulfill({ success: true, data: [
      { id: 101, public_name: 'gpt-4o-mini', provider: 'OpenAI', status: 'active', price: { id: 501, version: 2, price_kind: 'text', tiers: [{ tier_order: 1, minimum_input_tokens: 0, maximum_input_tokens: null, input_price_micros: 150000, output_price_micros: 600000, cache_read_price_micros: 75000, cache_write_price_micros: 200000 }] } },
      { id: 102, public_name: 'gpt-image-1', provider: 'OpenAI', status: 'active', price: { id: 502, version: 1, price_kind: 'image', image_prices: [{ quality: 'low', size: '1024x1024', price_micros: 11000 }, { quality: 'high', size: '1024x1024', price_micros: 167000 }] } },
    ] });
    if (url.pathname === '/api/offers') return fulfill({ success: true, data: [
      { offer_id: 201, merchant_id: 31, merchant_name: '北辰云智', model: 'gpt-4o-mini', multiplier_bps: 12000, call_count: 32400, token_usage: 2600000000, last_called_at: new Date(Date.now() - 10 * 60_000).toISOString(), price_tiers: [{ tier_order: 1, minimum_input_tokens: 0, maximum_input_tokens: null, input_price_micros: 180000, output_price_micros: 720000, cache_read_price_micros: 90000, cache_write_price_micros: 240000 }], channel_count: 2, health_state: 'healthy', availability_ppm: 999800, ttft_p95_ms: 1200, cache_hit_ppm: 720000, cache_sample_input_tokens: 2000000, cache_weight_ready: true, channels: [{ merchant_channel_id: 51, channel_name: 'OpenAI 主线路', health_state: 'healthy', availability_ppm: 999800, ttft_p95_ms: 1200, cache_hit_ppm: 680000, cache_sample_input_tokens: 1200000, cache_weight_ready: true }, { merchant_channel_id: 52, channel_name: 'OpenAI 备用线路', health_state: 'degraded', availability_ppm: 987500, ttft_p95_ms: 1650, cache_hit_ppm: 0, cache_sample_input_tokens: 500000, cache_weight_ready: false }] },
      { offer_id: 202, merchant_id: 32, merchant_name: '矩阵算力', model: 'gpt-image-1', multiplier_bps: 0, call_count: 1200, token_usage: 0, last_called_at: new Date(Date.now() - 2 * 60 * 60_000).toISOString(), image_prices: [{ quality: 'low', size: '1024x1024', price_micros: 0 }], channel_count: 1, health_state: 'healthy', availability_ppm: 998000, ttft_p95_ms: 2100, cache_hit_ppm: 0, cache_sample_input_tokens: 0, cache_weight_ready: false, channels: [{ merchant_channel_id: 53, channel_name: '图片线路', health_state: 'healthy', availability_ppm: 998000, ttft_p95_ms: 2100 }] },
    ] });
    if (url.pathname === '/api/keys' && request.method() === 'GET') return fulfill({ success: true, data: { page: 1, page_size: 20, total: 1, items: [{ id: 9, name: '生产主密钥', key: 'abcd**********wxyz', status: 1, expired_time: -1, unlimited_quota: true, routing_policy_id: 71, routing_policy_name: '缓存优先主路由' }] } });
    if (url.pathname === '/api/routing-policies') return fulfill({ success: true, data: [{ id: 71, user_id: 1, name: '缓存优先主路由', priority: 'cache', minimum_multiplier_bps: 0, maximum_multiplier_bps: 50000, minimum_cache_hit_ppm: 600000, bound_key_count: 1, updated_at: '2026-07-23T14:00:00Z' }] });
    if (url.pathname === '/api/usage') return fulfill({ success: true, data: {
      stats: { quota: 1250000, rpm: 2, tpm: 1800 },
      trend: [{ created_at: 1784131200, quota: 250000, count: 2 }, { created_at: 1784217600, quota: 1000000, count: 5 }],
      filters: { models: ['gpt-4o-mini'], merchants: [{ user_id: 8, name: '北辰云智' }] },
      logs: { page: 1, page_size: 20, total: 1, items: [{ id: 901, user_id: 7, created_at: 1784692800, model_name: 'gpt-4o-mini', quota: 500000, prompt_tokens: 2400, completion_tokens: 680, cache_tokens: 1200, token_id: 9, merchant_user_id: 8, merchant_name: '北辰云智', channel_id: 51, channel_name: 'OpenAI 主线路', request_id: 'req-user-1001' }] },
    } });
    if (url.pathname === '/api/merchant/channels') return fulfill({ success: true, data: { merchant: { user_id: 1, merchant_name: 'ModelRoute 官方', merchant_status: 'active', is_merchant: true }, items: [{ id: 51, user_id: 1, merchant_name: 'ModelRoute 官方', review_id: 21, channel_name: 'OpenAI 测试渠道', matched_models: ['gpt-4o-mini'], adapter_type: 'openai_compatible', base_url: 'https://api.lwylink.xyz', credential_fingerprint: 'a1b2c3...d4e5f6', status: 'active', health_state: 'healthy', created_at: '2026-07-22T14:00:00Z' }], reviews: [{ id: 21, user_id: 1, channel_name: 'OpenAI 测试渠道', adapter_type: 'openai_compatible', base_url: 'https://api.lwylink.xyz', status: 'approved', credential_configured: true, created_at: '2026-07-22T14:00:00Z' }] } });
    if (url.pathname === '/api/merchant/profile') return fulfill({ success: true, data: { id: 31, user_id: 1, display_name: 'ModelRoute 官方', website_url: 'https://aggregate.lwylink.xyz', business_description: '提供稳定的多厂商模型服务。', status: 'active', is_platform_merchant: true } });
    if (url.pathname === '/api/merchant/offers') return fulfill({ success: true, data: [{ offer_id: 201, merchant_id: 31, merchant_name: '北辰云智', model: 'gpt-4o-mini', multiplier_bps: 12000, call_count: 32400, token_usage: 2600000000, last_called_at: new Date(Date.now() - 10 * 60_000).toISOString(), price_tiers: [{ tier_order: 1, minimum_input_tokens: 0, maximum_input_tokens: null, input_price_micros: 180000, output_price_micros: 720000, cache_read_price_micros: 90000, cache_write_price_micros: 240000 }], channel_count: 1, offer_status: 'active', health_state: 'healthy', availability_ppm: 999800, ttft_p95_ms: 1200, cache_hit_ppm: 720000, cache_sample_input_tokens: 2000000, cache_weight_ready: true, channels: [{ merchant_channel_id: 51, channel_name: 'OpenAI 主线路', health_state: 'healthy', availability_ppm: 999800, ttft_p95_ms: 1200, cache_hit_ppm: 680000, cache_sample_input_tokens: 1200000, cache_weight_ready: true }] }] });
    if (url.pathname === '/api/merchant/balance') return fulfill({ success: true, data: { merchant_id: 31, available_quota: 90000000, frozen_quota: 25000000, lifetime_earned_quota: 140000000, lifetime_refunded_quota: 5000000, lifetime_paid_quota: 20000000 } });
    if (url.pathname === '/api/merchant/usage') return fulfill({ success: true, data: { filters: { models: ['gpt-4o-mini'] }, logs: { page: 1, page_size: 20, total: 1, items: [{ id: 902, created_at: 1784692800, model_name: 'gpt-4o-mini', prompt_tokens: 3100, completion_tokens: 920, cache_tokens: 800, channel_id: 51, channel_name: 'OpenAI 主线路', merchant_income_quota: 630000, request_id: 'req-merchant-9001' }] } } });
    if (url.pathname === '/api/merchant/withdrawals') return fulfill({ success: true, data: { page: 1, page_size: 20, total: 1, items: [{ id: 8, withdrawal_no: 'MW202607220001', merchant_id: 31, amount_quota: 50000000, withdrawal_tax_bps: 750, tax_quota: 3750000, net_payout_quota: 46250000, available_before_quota: 90000000, recipient_type: 'alipay', recipient_account_masked: '138****2621', status: 'pending', created_at: '2026-07-22T14:00:00Z' }] } });
     if (url.pathname === '/api/admin/channel-reviews') return fulfill({ success: true, data: { page: 1, page_size: 100, total: 1, items: [{ id: 22, user_id: 8, channel_name: 'Anthropic 备用线路', canonical_model_id: 107, model: 'claude-sonnet-5', provider: 'Anthropic', adapter_type: 'anthropic', base_url: 'https://api.anthropic.example', status: 'verified', credential_configured: true, models: [{ id: 221, upstream_model_name: 'claude-sonnet-5', canonical_model_id: 107, match_status: 'matched', test_status: 'passed', test_latency_ms: 286, test_error: '' }], created_at: '2026-07-22T15:00:00Z' }] } });
    if (url.pathname === '/api/admin/channels') return fulfill({ success: true, data: { page: 1, page_size: 100, total: 1, items: [{ id: 51, user_id: 1, merchant_name: 'ModelRoute 官方', review_id: 21, channel_name: 'OpenAI 主线路', matched_models: ['gpt-4o-mini'], adapter_type: 'openai_compatible', base_url: 'https://api.merchant.example', credential_fingerprint: 'a1b2c3...d4e5f6', status: 'active', health_state: 'healthy', created_at: '2026-07-22T14:00:00Z' }] } });
    if (url.pathname === '/api/admin/overview') return fulfill({ success: true, data: {
      generated_at: '2026-07-22T16:00:00Z', sync_delay_seconds: 16,
      summary: { today_gross_quota: 9000000, today_merchant_quota: 8100000, today_platform_quota: 900000, today_request_count: 128, today_successful_requests: 126, pending_merchant_applications: 1, pending_channel_reviews: 1, pending_withdrawals: 2, total_channels: 2, active_channels: 1, cooling_channels: 1, disabled_channels: 1, payment_enabled: true, last_synced_through: 1784719984, last_sync_success_at: '2026-07-22T15:59:44Z' },
      services: { database: { status: 'healthy', latency_ms: 3 }, redis: { status: 'healthy', latency_ms: 2 }, new_api: { status: 'healthy', latency_ms: 24 } },
    } });
    if (url.pathname === '/api/admin/channel-logs') return fulfill({ success: true, data: {
      stats: { quota: 4750000, rpm: 7, tpm: 9200 },
      trend: [{ created_at: 1784692800, quota: 1750000, count: 18 }],
      filters: { models: ['gpt-4o-mini'], merchants: [{ user_id: 8, name: '北辰云智' }] },
      logs: { page: 1, page_size: 20, total: 1, items: [{ id: 9901, user_id: 7, created_at: 1784692800, model_name: 'gpt-4o-mini', quota: 700000, prompt_tokens: 3100, completion_tokens: 920, cache_tokens: 800, merchant_user_id: 8, merchant_name: '北辰云智', channel_id: 51, channel_name: 'OpenAI 主线路', request_id: 'req-admin-9001' }] },
    } });
    if (url.pathname === '/api/admin/platform-settings') return fulfill({ success: true, data: { smtp: { server: 'smtp.example.com', port: 465, account: 'notice@example.com', from: 'ModelRoute <notice@example.com>', ssl_enabled: true, starttls_enabled: false, insecure_skip_verify: false, force_auth_login: false, password_write_only: true }, registration: { email_verification_enabled: true, register_enabled: true, password_login_enabled: true, password_register_enabled: true } } });
    if (url.pathname === '/api/admin/redemptions') return fulfill({ success: true, data: { page: 1, page_size: 20, total: 1, items: [{ id: 6, name: '七月活动码', key: 'MR-JULY-2026', status: 1, quota: 500000, count: 2, expired_time: 1785168000 }] } });
    if (url.pathname === '/api/admin/payment-settings') return fulfill({ success: true, data: { id: 11, version: 3, gateway_url: 'https://zpayz.cn', merchant_pid_masked: '202***********878', key_configured: true, payment_methods: ['alipay', 'wxpay'], minimum_cny_cents: 100, maximum_cny_cents: 2000000, withdrawal_tax_bps: 750, enabled: true, notify_url: 'https://aggregate.lwylink.xyz/api/payments/easypay/notify', return_url: 'https://aggregate.lwylink.xyz/payments/return', created_at: '2026-07-22T09:00:00Z' } });
    if (url.pathname === '/api/admin/payment-orders') return fulfill({ success: true, data: { page: 1, page_size: 50, total: 2, items: [
      { id: 21, order_no: 'MR202607220001', user_id: 7, payment_type: 'alipay', amount_cny_cents: 10000, newapi_quota: 50000000, payment_status: 'paid', credit_status: 'succeeded', third_party_trade_no: 'EP202607220009', paid_at: '2026-07-22T09:05:00Z' },
      { id: 22, order_no: 'MR202607220002', user_id: 8, payment_type: 'wxpay', amount_cny_cents: 5000, newapi_quota: 25000000, payment_status: 'paid', credit_status: 'unknown', third_party_trade_no: 'EP202607220010', paid_at: '2026-07-22T09:15:00Z' },
    ] } });
    if (url.pathname === '/api/admin/withdrawals') return fulfill({ success: true, data: { page: 1, page_size: 50, total: 2, items: [
      { id: 8, withdrawal_no: 'MW202607220001', merchant_id: 31, merchant_name: '北辰云智', amount_quota: 50000000, tax_quota: 1500000, net_payout_quota: 48500000, available_before_quota: 90000000, recipient_type: 'alipay', recipient_account: '13800132621', recipient_account_masked: '138****2621', status: 'pending', created_at: '2026-07-22T14:00:00Z' },
      { id: 9, withdrawal_no: 'MW202607220002', merchant_id: 32, merchant_name: '矩阵算力', amount_quota: 25000000, tax_quota: 750000, net_payout_quota: 24250000, available_before_quota: 82000000, recipient_type: 'alipay', recipient_account: '13900130021', recipient_account_masked: '139****0021', status: 'approved', created_at: '2026-07-22T13:00:00Z' },
    ] } });
    if (url.pathname === '/api/payment-settings') return fulfill({ success: true, data: { version: 3, payment_methods: ['alipay', 'wxpay'], minimum_cny_cents: 100, maximum_cny_cents: 2000000, withdrawal_tax_bps: 750, enabled: true } });
    if (url.pathname === '/api/payment-orders') return fulfill({ success: true, data: { page: 1, page_size: 10, total: 1, items: [{ id: 21, order_no: 'MR202607220001', amount_cny_cents: 10000, newapi_quota: 50000000, payment_status: 'paid', credit_status: 'succeeded', created_at: '2026-07-22T09:00:00Z' }] } });
    return fulfill({ success: false, error: { code: 'not_mocked', message: url.pathname } }, 404);
  });
  await page.goto(baseUrl, { waitUntil: 'networkidle' });
  for (const label of ['调用次数', 'Token 用量', '最后调用']) await page.getByRole('columnheader', { name: label, exact: true }).waitFor();
  await page.getByText('32.4K', { exact: true }).waitFor();
  await page.getByText('2.6B', { exact: true }).waitFor();
  await page.getByText('10分钟前', { exact: true }).waitFor();

  const captures = [
    ['模型市场', 'market'],
    ['控制台', 'console'],
    ['商户中心', 'merchant'],
    ['管理后台', 'admin'],
  ];

  for (const [label, filename] of captures) {
    if (filename !== 'market') await openNavigation(page, label);
    await verifyLayout(page, `${filename}-${suffix}`);
    const path = `${outputDir}${filename}-${suffix}.png`;
    await page.screenshot({ path, fullPage: true });
    if ((await stat(path)).size < 20_000) throw new Error(`${filename}-${suffix} 截图文件过小`);
  }

  await page.getByRole('button', { name: '公告', exact: true }).click();
  await page.getByRole('heading', { name: '公告中心' }).waitFor();
  await verifyLayout(page, `announcement-panel-${suffix}`);
  let path = `${outputDir}announcement-panel-${suffix}.png`;
  await page.screenshot({ path, fullPage: false });
  if ((await stat(path)).size < 20_000) throw new Error(`announcement-panel-${suffix} 截图文件过小`);
  await page.getByRole('button', { name: '关闭公告' }).click();

  await page.getByRole('button', { name: '公告管理' }).click();
  await page.getByText('商户提现安排').waitFor();
  await verifyLayout(page, `admin-announcements-${suffix}`);
  path = `${outputDir}admin-announcements-${suffix}.png`;
  await page.screenshot({ path, fullPage: true });
  if ((await stat(path)).size < 20_000) throw new Error(`admin-announcements-${suffix} 截图文件过小`);

  await page.getByRole('button', { name: '渠道调用日志' }).click();
  await page.getByText('OpenAI 主线路', { exact: true }).waitFor();
  await verifyLayout(page, `admin-usage-${suffix}`);
  path = `${outputDir}admin-usage-${suffix}.png`;
  await page.screenshot({ path, fullPage: true });
  if ((await stat(path)).size < 20_000) throw new Error(`admin-usage-${suffix} 截图文件过小`);

  for (const [label, ready, filename] of [
    ['平台设置', () => page.getByLabel('SMTP 服务器'), 'admin-settings'],
    ['兑换码管理', () => page.getByText('MR-JULY-2026', { exact: true }), 'admin-redemptions'],
  ]) {
    await page.getByRole('button', { name: label }).click();
    await ready().first().waitFor();
    await verifyLayout(page, `${filename}-${suffix}`);
    path = `${outputDir}${filename}-${suffix}.png`;
    await page.screenshot({ path, fullPage: true });
    if ((await stat(path)).size < 20_000) throw new Error(`${filename}-${suffix} 截图文件过小`);
  }

  await page.getByRole('button', { name: '支付与充值' }).click();
  await page.getByRole('button', { name: '编辑配置' }).click();
  path = `${outputDir}payment-config-${suffix}.png`;
  await page.screenshot({ path, fullPage: false });
  if ((await stat(path)).size < 20_000) throw new Error(`payment-config-${suffix} 截图文件过小`);
  await page.getByRole('button', { name: '关闭', exact: true }).click();

  await page.getByRole('button', { name: '渠道与探活' }).click();
  await page.getByText('Anthropic 备用线路').waitFor();
  await verifyLayout(page, `admin-channels-${suffix}`);
  path = `${outputDir}admin-channels-${suffix}.png`;
  await page.screenshot({ path, fullPage: true });
  if ((await stat(path)).size < 20_000) throw new Error(`admin-channels-${suffix} 截图文件过小`);
  await page.getByRole('row').filter({ hasText: 'Anthropic 备用线路' }).getByRole('button', { name: '查看模型（1）', exact: true }).click();
  await page.getByRole('dialog', { name: '渠道模型' }).waitFor();
  await verifyLayout(page, `admin-channel-models-${suffix}`);
  path = `${outputDir}admin-channel-models-${suffix}.png`;
  await page.screenshot({ path, fullPage: false });
  if ((await stat(path)).size < 20_000) throw new Error(`admin-channel-models-${suffix} 截图文件过小`);
  await page.getByRole('dialog', { name: '渠道模型' }).getByRole('button', { name: '关闭', exact: true }).click();

  await page.getByRole('button', { name: '模型与定价' }).click();
  await page.getByText('v2', { exact: true }).waitFor();
  await verifyLayout(page, `admin-models-${suffix}`);
  path = `${outputDir}admin-models-${suffix}.png`;
  await page.screenshot({ path, fullPage: true });
  if ((await stat(path)).size < 20_000) throw new Error(`admin-models-${suffix} 截图文件过小`);
   await clickRowAction(page, '发布价格');
  await page.getByLabel('第 1 档输入价格').waitFor();
  await verifyLayout(page, `admin-model-price-${suffix}`);
  path = `${outputDir}admin-model-price-${suffix}.png`;
  await page.screenshot({ path, fullPage: false });
  if ((await stat(path)).size < 20_000) throw new Error(`admin-model-price-${suffix} 截图文件过小`);
  await page.getByRole('button', { name: '关闭', exact: true }).click();
   await clickRowAction(page, '发布价格', 1);
  await page.getByRole('heading', { name: '图片单张价格矩阵' }).waitFor();
  await verifyLayout(page, `admin-image-price-${suffix}`);
  path = `${outputDir}admin-image-price-${suffix}.png`;
  await page.screenshot({ path, fullPage: false });
  if ((await stat(path)).size < 20_000) throw new Error(`admin-image-price-${suffix} 截图文件过小`);
  await page.getByRole('button', { name: '关闭', exact: true }).click();

  await page.getByRole('button', { name: '提现审核' }).click();
  await page.getByText('MW202607220001').waitFor();
  await verifyLayout(page, `admin-withdrawals-${suffix}`);
  path = `${outputDir}admin-withdrawals-${suffix}.png`;
  await page.screenshot({ path, fullPage: true });
  if ((await stat(path)).size < 20_000) throw new Error(`admin-withdrawals-${suffix} 截图文件过小`);

  await openNavigation(page, '商户中心');
  const merchantSidebar = page.getByRole('complementary', { name: '商户功能' });
  await merchantSidebar.getByRole('button', { name: '渠道管理', exact: true }).click();
  await page.getByRole('row').filter({ hasText: 'OpenAI 测试渠道' }).getByRole('button', { name: '查看模型（1）', exact: true }).click();
  await page.getByRole('dialog', { name: '渠道模型' }).waitFor();
  await verifyLayout(page, `merchant-channel-models-${suffix}`);
  path = `${outputDir}merchant-channel-models-${suffix}.png`;
  await page.screenshot({ path, fullPage: false });
  if ((await stat(path)).size < 20_000) throw new Error(`merchant-channel-models-${suffix} 截图文件过小`);
  await page.getByRole('dialog', { name: '渠道模型' }).getByRole('button', { name: '关闭', exact: true }).click();
  await merchantSidebar.getByRole('button', { name: '上架模型', exact: true }).click();
  for (const label of ['调用次数', 'Token 用量', '最后调用']) await page.getByRole('columnheader', { name: label, exact: true }).waitFor();
  await page.getByText('32.4K', { exact: true }).waitFor();
  await page.getByText('2.6B', { exact: true }).waitFor();
  await page.getByText('10分钟前', { exact: true }).waitFor();
  await verifyLayout(page, `merchant-offers-${suffix}`);
  path = `${outputDir}merchant-offers-${suffix}.png`;
  await page.screenshot({ path, fullPage: true });
  if ((await stat(path)).size < 20_000) throw new Error(`merchant-offers-${suffix} 截图文件过小`);
  await clickRowAction(page, '查看渠道状态');
  await page.getByRole('columnheader', { name: '可用率', exact: true }).waitFor();
  await verifyLayout(page, `merchant-channel-health-${suffix}`);
  path = `${outputDir}merchant-channel-health-${suffix}.png`;
  await page.screenshot({ path, fullPage: true });
  if ((await stat(path)).size < 20_000) throw new Error(`merchant-channel-health-${suffix} 截图文件过小`);
  await merchantSidebar.getByRole('button', { name: '调用记录', exact: true }).click();
  await page.getByText('OpenAI 主线路', { exact: true }).waitFor();
  await verifyLayout(page, `merchant-usage-${suffix}`);
  path = `${outputDir}merchant-usage-${suffix}.png`;
  await page.screenshot({ path, fullPage: true });
  if ((await stat(path)).size < 20_000) throw new Error(`merchant-usage-${suffix} 截图文件过小`);
  await merchantSidebar.getByRole('button', { name: '提现记录', exact: true }).click();
  await page.getByRole('button', { name: '申请提现' }).click();
  await page.getByLabel('提现金额（人民币）').fill('100');
  await page.getByLabel('支付宝账户').fill('13800132621');
  path = `${outputDir}withdrawal-modal-${suffix}.png`;
  await page.screenshot({ path, fullPage: false });
  if ((await stat(path)).size < 20_000) throw new Error(`withdrawal-modal-${suffix} 截图文件过小`);
  await page.getByRole('button', { name: '关闭', exact: true }).click();

  await openNavigation(page, '控制台');
  const consoleSidebar = page.getByRole('complementary', { name: '控制台功能' });
  await consoleSidebar.getByRole('button', { name: '调用记录', exact: true }).click();
  await page.getByText('北辰云智', { exact: true }).waitFor();
  await verifyLayout(page, `console-usage-${suffix}`);
  path = `${outputDir}console-usage-${suffix}.png`;
  await page.screenshot({ path, fullPage: true });
  if ((await stat(path)).size < 20_000) throw new Error(`console-usage-${suffix} 截图文件过小`);
  await consoleSidebar.getByRole('button', { name: '账户设置', exact: true }).click();
  await page.getByText('用户 ID 1 · 管理员').waitFor();
  await verifyLayout(page, `console-settings-${suffix}`, 120);
  path = `${outputDir}console-settings-${suffix}.png`;
  await page.screenshot({ path, fullPage: true });
  if ((await stat(path)).size < 20_000) throw new Error(`console-settings-${suffix} 截图文件过小`);
  await consoleSidebar.getByRole('button', { name: '路由管理', exact: true }).click();
  await page.getByText('缓存优先主路由').waitFor();
  await clickRowAction(page, '编辑');
  await page.getByRole('dialog', { name: '编辑路由策略' }).waitFor();
  path = `${outputDir}route-modal-${suffix}.png`;
  await page.screenshot({ path, fullPage: false });
  if ((await stat(path)).size < 20_000) throw new Error(`route-modal-${suffix} 截图文件过小`);
  await page.getByRole('button', { name: '关闭', exact: true }).click();
  await consoleSidebar.getByRole('button', { name: 'API 密钥', exact: true }).click();
  await page.getByText('生产主密钥').waitFor();
  path = `${outputDir}console-keys-${suffix}.png`;
  await page.screenshot({ path, fullPage: true });
  if ((await stat(path)).size < 20_000) throw new Error(`console-keys-${suffix} 截图文件过小`);

  if (runtimeErrors.length) throw new Error(`${suffix} 存在运行时错误：${runtimeErrors.join('; ')}`);
  await page.close();
}

try {
  await captureSet({ width: 1440, height: 1000 }, 'desktop');
  await captureSet({ width: 390, height: 844 }, 'mobile');
  console.log('截图完成：桌面与移动端三套侧栏、上架模型、统一调用记录、文本阶梯、图片矩阵和人民币支付均已生成，根页面无横向溢出。');
} finally {
  await browser.close();
  await stopLocalPreview(previewServer);
}
