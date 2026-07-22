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
    if (url.pathname === '/api/models' || url.pathname === '/api/admin/models') return fulfill({ success: true, data: [{ id: 101, public_name: 'gpt-4o-mini', provider: 'OpenAI', protocol: 'chat_completions', status: 'active', price: { id: 501, version: 2, billing_mode: 'token', model_ratio_micros: 600000, completion_ratio_micros: 4000000, cache_ratio_micros: 500000, create_cache_ratio_micros: 0, image_ratio_micros: 0, audio_ratio_micros: 0, audio_completion_ratio_micros: 0 } }] });
    if (url.pathname === '/api/offers') return fulfill({ success: true, data: [
      { offer_id: 201, merchant_id: 31, merchant_name: '北辰云智', merchant_channel_id: 51, channel_name: 'OpenAI 主线路', model: 'gpt-4o-mini', protocol: 'chat_completions', multiplier_bps: 7200, health_state: 'healthy', availability_ppm: 999800, ttft_p95_ms: 1200 },
      { offer_id: 202, merchant_id: 32, merchant_name: '矩阵算力', merchant_channel_id: 52, channel_name: 'OpenAI 备用线路', model: 'gpt-4o-mini', protocol: 'chat_completions', multiplier_bps: 5800, health_state: 'degraded', availability_ppm: 987500, ttft_p95_ms: 1650 },
    ] });
    if (url.pathname === '/api/keys' && request.method() === 'GET') return fulfill({ success: true, data: { page: 1, page_size: 20, total: 1, items: [{ id: 9, name: '生产主密钥', key: 'abcd**********wxyz', status: 1, expired_time: -1, unlimited_quota: true }] } });
    if (url.pathname === '/api/routing-policies/9') return fulfill({ success: true, data: { id: 71, newapi_token_id: 9, mode: 'smart', priority: 'stability', minimum_multiplier_bps: 4000, maximum_multiplier_bps: 8000, merchant_ids: [] } });
    if (url.pathname === '/api/usage') return fulfill({ success: true, data: {
      stats: { quota: 1250000, rpm: 2, tpm: 1800 },
      trend: [{ created_at: 1784131200, quota: 250000, count: 2 }, { created_at: 1784217600, quota: 1000000, count: 5 }],
      logs: { page: 1, page_size: 20, total: 1, items: [{ id: 901, created_at: 1784692800, type: 2, model_name: 'gpt-4o-mini', quota: 500000, prompt_tokens: 2400, completion_tokens: 680, token_name: '生产主密钥', request_id: 'req-user-1001', other: '{"cache_tokens":1200}' }] },
    } });
    if (url.pathname === '/api/merchant/channels') return fulfill({ success: true, data: { merchant: { merchant_id: 31, user_id: 1, merchant_status: 'active', is_merchant: true }, items: [{ id: 51, merchant_id: 31, review_id: 21, newapi_channel_id: 41, channel_name: 'OpenAI 主线路', channel_type: 1, base_url: 'https://api.merchant.example', models: ['gpt-4o-mini'], credential_fingerprint: 'a1b2c3...d4e5f6', status: 'active', created_at: '2026-07-22T14:00:00Z' }], reviews: [{ id: 21, user_id: 1, application_id: 11, channel_name: 'OpenAI 主线路', channel_type: 1, base_url: 'https://api.merchant.example', requested_models: ['gpt-4o-mini'], verified_models: ['gpt-4o-mini'], status: 'approved', credential_configured: false, created_at: '2026-07-22T14:00:00Z' }] } });
    if (url.pathname === '/api/merchant/offers' || url.pathname === '/api/merchant/health') return fulfill({ success: true, data: [{ offer_id: 201, merchant_channel_id: 51, channel_name: 'OpenAI 主线路', model: 'gpt-4o-mini', protocol: 'chat_completions', multiplier_bps: 7200, offer_status: 'active', health_state: 'healthy', availability_ppm: 999800, ttft_p95_ms: 1200 }] });
    if (url.pathname === '/api/merchant/balance') return fulfill({ success: true, data: { merchant_id: 31, available_quota: 9000000, frozen_quota: 2500000, lifetime_earned_quota: 14000000, lifetime_refunded_quota: 500000, lifetime_paid_quota: 2000000 } });
    if (url.pathname === '/api/merchant/ledger') return fulfill({ success: true, data: { page: 1, page_size: 50, total: 2, items: [
      { id: 81, merchant_id: 31, entry_type: 'earning', available_delta: 900000, frozen_delta: 0, available_after: 9000000, frozen_after: 2500000, description: '模型调用结算', newapi_log_id: 1001, source_type: 2, model_name: 'gpt-4o-mini', gross_quota: 1000000, created_at: '2026-07-22T15:00:00Z' },
      { id: 80, merchant_id: 31, entry_type: 'withdrawal_freeze', available_delta: -2500000, frozen_delta: 2500000, available_after: 8100000, frozen_after: 2500000, description: '提交人工提现并冻结余额', withdrawal_no: 'MW202607220001', created_at: '2026-07-22T14:00:00Z' },
    ] } });
    if (url.pathname === '/api/merchant/withdrawals') return fulfill({ success: true, data: { page: 1, page_size: 50, total: 1, items: [{ id: 8, withdrawal_no: 'MW202607220001', merchant_id: 31, amount_quota: 2500000, available_before_quota: 10600000, recipient_type: 'alipay', recipient_account_masked: '138****2621', status: 'pending', created_at: '2026-07-22T14:00:00Z' }] } });
    if (url.pathname === '/api/admin/channel-reviews') return fulfill({ success: true, data: { page: 1, page_size: 100, total: 1, items: [{ id: 22, user_id: 8, channel_name: 'Anthropic 备用线路', channel_type: 14, base_url: 'https://api.anthropic.example', requested_models: ['claude-sonnet-4-5'], status: 'pending', credential_configured: true, created_at: '2026-07-22T15:00:00Z' }] } });
    if (url.pathname === '/api/admin/channels') return fulfill({ success: true, data: { page: 1, page_size: 100, total: 1, items: [{ id: 51, merchant_id: 31, review_id: 21, newapi_channel_id: 41, channel_name: 'OpenAI 主线路', channel_type: 1, base_url: 'https://api.merchant.example', models: ['gpt-4o-mini'], credential_fingerprint: 'a1b2c3...d4e5f6', status: 'active', created_at: '2026-07-22T14:00:00Z' }] } });
    if (url.pathname === '/api/admin/channels/51/runtime') return fulfill({ success: true, data: { newapi_channel_id: 41, status: 1, balance_usd_micros: 12500000, balance_updated_time: 1784649600, response_time_ms: 420, test_time: 1784649500 } });
    if (url.pathname === '/api/admin/overview') return fulfill({ success: true, data: {
      generated_at: '2026-07-22T16:00:00Z', sync_delay_seconds: 16,
      summary: { today_gross_quota: 9000000, today_merchant_quota: 8100000, today_platform_quota: 900000, today_refund_quota: 500000, today_request_count: 128, today_successful_requests: 126, pending_merchant_applications: 1, pending_channel_reviews: 1, pending_withdrawals: 2, open_reconciliation: 1, total_channels: 2, active_channels: 1, cooling_channels: 1, disabled_channels: 1, payment_enabled: true, last_synced_through: 1784719984, last_sync_success_at: '2026-07-22T15:59:44Z' },
      services: { database: { status: 'healthy', latency_ms: 3 }, redis: { status: 'healthy', latency_ms: 2 }, new_api: { status: 'healthy', latency_ms: 24 } },
    } });
    if (url.pathname === '/api/admin/usage') return fulfill({ success: true, data: {
      stats: { quota: 4750000, rpm: 7, tpm: 9200 },
      trend: [{ created_at: 1784692800, quota: 1750000, count: 18 }],
      logs: { page: 1, page_size: 20, total: 1, items: [{ id: 9901, user_id: 7, username: 'alice', created_at: 1784692800, type: 2, model_name: 'gpt-4o-mini', quota: 700000, prompt_tokens: 3100, completion_tokens: 920, token_name: '生产密钥', request_id: 'req-admin-9001', channel: 41, other: '{"cache_tokens":800}' }] },
    } });
    if (url.pathname === '/api/admin/platform-settings') return fulfill({ success: true, data: { smtp: { server: 'smtp.example.com', port: 465, account: 'notice@example.com', from: 'ModelRoute <notice@example.com>', ssl_enabled: true, starttls_enabled: false, insecure_skip_verify: false, force_auth_login: false, password_write_only: true }, registration: { email_verification_enabled: true, register_enabled: true, password_login_enabled: true, password_register_enabled: true } } });
    if (url.pathname === '/api/admin/groups') return fulfill({ success: true, data: [{ name: 'default', ratio_bps: 10000 }, { name: 'vip', ratio_bps: 8000 }] });
    if (url.pathname === '/api/admin/redemptions') return fulfill({ success: true, data: { page: 1, page_size: 20, total: 1, items: [{ id: 6, name: '七月活动码', key: 'MR-JULY-2026', status: 1, quota: 500000, count: 2, expired_time: 1785168000 }] } });
    if (url.pathname === '/api/admin/system-tasks') return fulfill({ success: true, data: [{ task_id: 'systask_1', type: 'log_cleanup', status: 'succeeded', created_at: 1784649600, updated_at: 1784649660 }] });
    if (url.pathname === '/api/admin/payment-settings') return fulfill({ success: true, data: { id: 11, version: 3, gateway_url: 'https://zpayz.cn', merchant_pid_masked: '202***********878', key_configured: true, payment_methods: ['alipay', 'wxpay'], minimum_usd_cents: 1000, maximum_usd_cents: 2000000, exchange_rate_cny_cents_per_usd: 720, enabled: true, notify_url: 'https://aggregate.lwylink.xyz/api/payments/easypay/notify', return_url: 'https://aggregate.lwylink.xyz/payments/return', created_at: '2026-07-22T09:00:00Z' } });
    if (url.pathname === '/api/admin/payment-orders') return fulfill({ success: true, data: { page: 1, page_size: 50, total: 2, items: [
      { id: 21, order_no: 'MR202607220001', user_id: 7, payment_type: 'alipay', usd_cents: 10000, cny_cents: 72000, payment_status: 'paid', credit_status: 'succeeded', third_party_trade_no: 'EP202607220009', paid_at: '2026-07-22T09:05:00Z' },
      { id: 22, order_no: 'MR202607220002', user_id: 8, payment_type: 'wxpay', usd_cents: 5000, cny_cents: 36000, payment_status: 'paid', credit_status: 'unknown', third_party_trade_no: 'EP202607220010', paid_at: '2026-07-22T09:15:00Z' },
    ] } });
    if (url.pathname === '/api/admin/withdrawals') return fulfill({ success: true, data: { page: 1, page_size: 50, total: 2, items: [
      { id: 8, withdrawal_no: 'MW202607220001', merchant_id: 31, merchant_name: '北辰云智', amount_quota: 2500000, available_before_quota: 10600000, recipient_type: 'alipay', recipient_account: '13800132621', recipient_account_masked: '138****2621', status: 'pending', created_at: '2026-07-22T14:00:00Z' },
      { id: 9, withdrawal_no: 'MW202607220002', merchant_id: 32, merchant_name: '矩阵算力', amount_quota: 5000000, available_before_quota: 8200000, recipient_type: 'bank', recipient_account: '6217000012341098', recipient_account_masked: '6217****1098', status: 'approved', created_at: '2026-07-22T13:00:00Z' },
    ] } });
    if (url.pathname === '/api/admin/settlement-reconciliation') return fulfill({ success: true, data: { page: 1, page_size: 50, total: 1, items: [{ id: 12, source_id: 14, newapi_log_id: 1004, newapi_user_id: 18, source_type: 2, model_name: 'mkt-ov-missing', newapi_channel_id: 99, gross_quota: 50, reason_code: 'offer_mapping_missing', status: 'open', created_at: '2026-07-22T15:30:00Z' }] } });
    if (url.pathname === '/api/payment-orders') return fulfill({ success: true, data: { page: 1, page_size: 10, total: 1, items: [{ id: 21, order_no: 'MR202607220001', usd_cents: 10000, cny_cents: 72000, payment_status: 'paid', credit_status: 'succeeded', created_at: '2026-07-22T09:00:00Z' }] } });
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
    await verifyLayout(page, `${filename}-${suffix}`);
    const path = `${outputDir}${filename}-${suffix}.png`;
    await page.screenshot({ path, fullPage: true });
    if ((await stat(path)).size < 20_000) throw new Error(`${filename}-${suffix} 截图文件过小`);
  }

  await page.getByRole('button', { name: '调用日志' }).click();
  await page.getByText('req-admin-9001').waitFor();
  await verifyLayout(page, `admin-usage-${suffix}`);
  let path = `${outputDir}admin-usage-${suffix}.png`;
  await page.screenshot({ path, fullPage: true });
  if ((await stat(path)).size < 20_000) throw new Error(`admin-usage-${suffix} 截图文件过小`);

  for (const [label, ready, filename] of [
    ['平台设置', () => page.getByLabel('SMTP 服务器'), 'admin-settings'],
    ['分组倍率', () => page.getByLabel('vip 分组倍率'), 'admin-groups'],
    ['兑换码管理', () => page.getByText('MR-JULY-2026', { exact: true }), 'admin-redemptions'],
    ['系统任务', () => page.getByText('systask_1', { exact: true }), 'admin-tasks'],
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

  await page.getByRole('button', { name: '模型与定价' }).click();
  await page.getByText('价格版本 v2').waitFor();
  await verifyLayout(page, `admin-models-${suffix}`);
  path = `${outputDir}admin-models-${suffix}.png`;
  await page.screenshot({ path, fullPage: true });
  if ((await stat(path)).size < 20_000) throw new Error(`admin-models-${suffix} 截图文件过小`);

  await page.getByRole('button', { name: '提现审核' }).click();
  await page.getByText('MW202607220001').waitFor();
  await verifyLayout(page, `admin-withdrawals-${suffix}`);
  path = `${outputDir}admin-withdrawals-${suffix}.png`;
  await page.screenshot({ path, fullPage: true });
  if ((await stat(path)).size < 20_000) throw new Error(`admin-withdrawals-${suffix} 截图文件过小`);

  await openNavigation(page, '商户中心');
  await page.getByRole('button', { name: '申请提现' }).click();
  path = `${outputDir}withdrawal-modal-${suffix}.png`;
  await page.screenshot({ path, fullPage: false });
  if ((await stat(path)).size < 20_000) throw new Error(`withdrawal-modal-${suffix} 截图文件过小`);
  await page.getByRole('button', { name: '关闭', exact: true }).click();

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
  console.log('截图完成：桌面与移动端业务视图、新增运营页面和关键弹窗均已生成，根页面无横向溢出。');
} finally {
  await browser.close();
  await stopLocalPreview(previewServer);
}
