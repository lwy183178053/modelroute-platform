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
page.setDefaultTimeout(10000);

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
      return json({ success: true, data: { id: 1, username: 'modelroute', display_name: '平台管理员', role: 100, status: 1, group: 'default', quota: 128642000, used_quota: 4268000, is_merchant: true, merchant_id: 31, merchant_status: 'active' } });
    }
    if (url.pathname === '/api/models') {
      return json({ success: true, data: [{ id: 101, public_name: 'gpt-4o-mini', provider: 'OpenAI', protocol: 'chat_completions', status: 'active', price: { id: 501, version: 2, billing_mode: 'token', model_ratio_micros: 600000, completion_ratio_micros: 4000000, cache_ratio_micros: 500000, create_cache_ratio_micros: 0, image_ratio_micros: 0, audio_ratio_micros: 0, audio_completion_ratio_micros: 0 } }] });
    }
    if (url.pathname === '/api/offers') {
      return json({ success: true, data: [
        { offer_id: 201, merchant_id: 31, merchant_name: '北辰云智', merchant_channel_id: 51, channel_name: 'OpenAI 主线路', model: 'gpt-4o-mini', protocol: 'chat_completions', multiplier_bps: 7200, health_state: 'healthy', availability_ppm: 999800, ttft_p95_ms: 1200 },
        { offer_id: 202, merchant_id: 32, merchant_name: '矩阵算力', merchant_channel_id: 52, channel_name: 'OpenAI 备用线路', model: 'gpt-4o-mini', protocol: 'chat_completions', multiplier_bps: 5800, health_state: 'degraded', availability_ppm: 987500, ttft_p95_ms: 1650 },
      ] });
    }
    if (url.pathname === '/api/keys' && request.method() === 'GET') {
      return json({ success: true, data: { page: 1, page_size: 20, total: 1, items: [{ id: 9, name: '真实主密钥', key: 'abcd**********wxyz', status: 1, created_time: 1784690000, accessed_time: 1784692000, expired_time: -1, unlimited_quota: true }] } });
    }
    if (url.pathname === '/api/routing-policies/9' && request.method() === 'GET') {
      return json({ success: true, data: { id: 71, newapi_token_id: 9, mode: 'smart', priority: 'stability', minimum_multiplier_bps: 4000, maximum_multiplier_bps: 8000, merchant_ids: [] } });
    }
    if (url.pathname === '/api/routing-policies/9' && request.method() === 'PUT') {
      state.routeSaveCalls += 1;
      state.routeSaveBody = request.postDataJSON();
      return json({ success: true, data: { id: 71, newapi_token_id: 9, ...state.routeSaveBody } });
    }
    if (url.pathname === '/api/usage' && request.method() === 'GET') {
      return json({ success: true, data: {
        start_timestamp: 1784131200,
        end_timestamp: 1784736000,
        stats: { quota: 1250000, rpm: 2, tpm: 1800 },
        trend: [
          { created_at: 1784131200, model_name: 'gpt-4o-mini', quota: 250000, count: 2 },
          { created_at: 1784217600, model_name: 'gpt-4o-mini', quota: 1000000, count: 5 },
        ],
        logs: { page: 1, page_size: 20, total: 1, items: [
          { id: 901, created_at: 1784692800, type: 2, model_name: 'gpt-4o-mini', quota: 500000, prompt_tokens: 2400, completion_tokens: 680, token_name: '真实主密钥', request_id: 'req-user-1001', other: '{"cache_tokens":1200}' },
        ] },
      } });
    }
    if (url.pathname === '/api/redemptions/redeem' && request.method() === 'POST') {
      state.userRedemptionCalls += 1;
      state.userRedemptionBody = request.postDataJSON();
      return json({ success: true, data: 500000 });
    }
    if (url.pathname === '/api/merchant/offers' && request.method() === 'GET') {
      return json({ success: true, data: [{ offer_id: 201, merchant_channel_id: 51, channel_name: 'OpenAI 主线路', model: 'gpt-4o-mini', protocol: 'chat_completions', multiplier_bps: 7200, offer_status: 'active', health_state: 'healthy', availability_ppm: 999800, ttft_p95_ms: 1200 }] });
    }
    if (url.pathname === '/api/merchant/health' && request.method() === 'GET') {
      return json({ success: true, data: [{ offer_id: 201, merchant_channel_id: 51, channel_name: 'OpenAI 主线路', model: 'gpt-4o-mini', protocol: 'chat_completions', multiplier_bps: 7200, offer_status: 'active', health_state: 'healthy', availability_ppm: 999800, ttft_p95_ms: 1200 }] });
    }
    if (url.pathname === '/api/merchant/balance' && request.method() === 'GET') {
      return json({ success: true, data: { merchant_id: 31, available_quota: 9000000, frozen_quota: 2500000, lifetime_earned_quota: 14000000, lifetime_refunded_quota: 500000, lifetime_paid_quota: 2000000 } });
    }
    if (url.pathname === '/api/merchant/ledger' && request.method() === 'GET') {
      return json({ success: true, data: { page: 1, page_size: 50, total: 2, items: [
        { id: 81, merchant_id: 31, entry_type: 'earning', available_delta: 900000, frozen_delta: 0, available_after: 9000000, frozen_after: 2500000, description: '模型调用结算', newapi_log_id: 1001, source_type: 2, model_name: 'gpt-4o-mini', gross_quota: 1000000, created_at: '2026-07-22T15:00:00Z' },
        { id: 80, merchant_id: 31, entry_type: 'withdrawal_freeze', available_delta: -2500000, frozen_delta: 2500000, available_after: 8100000, frozen_after: 2500000, description: '提交人工提现并冻结余额', withdrawal_no: 'MW202607220001', created_at: '2026-07-22T14:00:00Z' },
      ] } });
    }
    if (url.pathname === '/api/merchant/withdrawals' && request.method() === 'GET') {
      return json({ success: true, data: { page: 1, page_size: 50, total: 1, items: [{ id: 8, withdrawal_no: 'MW202607220001', merchant_id: 31, amount_quota: 2500000, available_before_quota: 10600000, recipient_type: 'alipay', recipient_account_masked: '138****2621', status: 'pending', created_at: '2026-07-22T14:00:00Z' }] } });
    }
    if (url.pathname === '/api/merchant/withdrawals' && request.method() === 'POST') {
      state.withdrawalCreateCalls += 1;
      state.withdrawalCreateBody = request.postDataJSON();
      return json({ success: true, data: { id: 10, withdrawal_no: 'MW202607220010', merchant_id: 31, amount_quota: state.withdrawalCreateBody.amount_quota, recipient_type: state.withdrawalCreateBody.recipient_type, recipient_account_masked: '138****2621', status: 'pending', created_at: '2026-07-22T16:00:00Z' } }, 201);
    }
    if (url.pathname === '/api/merchant/offers' && request.method() === 'POST') {
      state.offerCreateCalls += 1;
      state.offerCreateBody = request.postDataJSON();
      return json({ success: true, data: { id: 203, status: 'active', ...state.offerCreateBody } }, 201);
    }
    if (url.pathname === '/api/admin/models' && request.method() === 'GET') {
      return json({ success: true, data: [{ id: 101, public_name: 'gpt-4o-mini', provider: 'OpenAI', protocol: 'chat_completions', status: 'active', price: { id: 501, version: 2, billing_mode: 'token', model_ratio_micros: 600000, completion_ratio_micros: 4000000, cache_ratio_micros: 500000, create_cache_ratio_micros: 0, image_ratio_micros: 0, audio_ratio_micros: 0, audio_completion_ratio_micros: 0 } }] });
    }
    if (url.pathname === '/api/admin/models' && request.method() === 'POST') {
      state.modelCreateCalls += 1;
      state.modelCreateBody = request.postDataJSON();
      return json({ success: true, data: { id: 102, status: 'active', ...state.modelCreateBody } }, 201);
    }
    if (url.pathname === '/api/admin/models/101/prices' && request.method() === 'POST') {
      state.pricePublishCalls += 1;
      state.pricePublishBody = request.postDataJSON();
      return json({ success: true, data: { id: 502, version: 3, ...state.pricePublishBody } }, 201);
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
    if (url.pathname === '/api/admin/overview' && request.method() === 'GET') {
      return json({ success: true, data: {
        generated_at: '2026-07-22T16:00:00Z', sync_delay_seconds: 16,
        summary: {
          today_gross_quota: 9000000, today_merchant_quota: 8100000, today_platform_quota: 900000,
          today_refund_quota: 500000, today_request_count: 128, today_successful_requests: 126,
          pending_merchant_applications: 1, pending_channel_reviews: 1, pending_withdrawals: 2,
          open_reconciliation: 1, total_channels: 2, active_channels: 1, cooling_channels: 1,
          disabled_channels: 1, payment_enabled: true, last_synced_through: 1784719984,
          last_sync_success_at: '2026-07-22T15:59:44Z', last_sync_error: '',
        },
        services: {
          database: { status: 'healthy', latency_ms: 3 },
          redis: { status: 'healthy', latency_ms: 2 },
          new_api: { status: 'healthy', latency_ms: 24 },
        },
      } });
    }
    if (url.pathname === '/api/admin/usage' && request.method() === 'GET') {
      return json({ success: true, data: {
        start_timestamp: 1784131200, end_timestamp: 1784736000,
        stats: { quota: 4750000, rpm: 7, tpm: 9200 },
        trend: [{ created_at: 1784692800, model_name: 'gpt-4o-mini', quota: 1750000, count: 18 }],
        logs: { page: 1, page_size: 20, total: 1, items: [
          { id: 9901, user_id: 7, username: 'alice', created_at: 1784692800, type: 2, model_name: 'gpt-4o-mini', quota: 700000, prompt_tokens: 3100, completion_tokens: 920, token_name: '生产密钥', request_id: 'req-admin-9001', channel: 41, other: '{"cache_tokens":800}' },
        ] },
      } });
    }
    if (url.pathname === '/api/admin/platform-settings' && request.method() === 'GET') {
      return json({ success: true, data: {
        smtp: { server: 'smtp.example.com', port: 465, account: 'notice@example.com', from: 'ModelRoute <notice@example.com>', ssl_enabled: true, starttls_enabled: false, insecure_skip_verify: false, force_auth_login: false, password_write_only: true },
        registration: { email_verification_enabled: true, register_enabled: true, password_login_enabled: true, password_register_enabled: true },
      } });
    }
    if (url.pathname === '/api/admin/platform-settings' && request.method() === 'PUT') {
      state.platformSettingCalls += 1;
      state.platformSettingBody = request.postDataJSON();
      return json({ success: true, data: { ...state.platformSettingBody, smtp: { ...state.platformSettingBody.smtp, password: undefined, password_write_only: true } } });
    }
    if (url.pathname === '/api/admin/groups' && request.method() === 'GET') {
      return json({ success: true, data: [{ name: 'default', ratio_bps: 10000 }, { name: 'vip', ratio_bps: 8000 }] });
    }
    if (url.pathname === '/api/admin/groups' && request.method() === 'PUT') {
      state.groupSettingCalls += 1;
      state.groupSettingBody = request.postDataJSON();
      return json({ success: true, data: state.groupSettingBody.groups });
    }
    if (url.pathname === '/api/admin/redemptions' && request.method() === 'GET') {
      return json({ success: true, data: { page: 1, page_size: 20, total: 1, items: [
        { id: 6, name: '七月活动码', key: 'MR-JULY-2026', status: 1, quota: 500000, count: 2, expired_time: 1785168000, created_time: 1784649600 },
      ] } });
    }
    if (url.pathname === '/api/admin/redemptions' && request.method() === 'POST') {
      state.redemptionCreateCalls += 1;
      state.redemptionCreateBody = request.postDataJSON();
      return json({ success: true, data: ['MR-NEW-A', 'MR-NEW-B'] }, 201);
    }
    if (url.pathname === '/api/admin/redemptions/6' && request.method() === 'PUT') {
      state.redemptionUpdateCalls += 1;
      state.redemptionUpdateBody = request.postDataJSON();
      return json({ success: true, data: { id: 6, status: 1, ...state.redemptionUpdateBody } });
    }
    if (url.pathname === '/api/admin/redemptions/6/status' && request.method() === 'POST') {
      state.redemptionStatusCalls += 1;
      state.redemptionStatusBody = request.postDataJSON();
      return json({ success: true, data: { id: 6, status: state.redemptionStatusBody.enabled ? 1 : 2 } });
    }
    if (url.pathname === '/api/admin/redemptions/6' && request.method() === 'DELETE') {
      state.redemptionDeleteCalls += 1;
      return json({ success: true, data: null });
    }
    if (url.pathname === '/api/admin/redemptions/invalid' && request.method() === 'DELETE') {
      state.redemptionCleanupCalls += 1;
      return json({ success: true, data: 1 });
    }
    if (url.pathname === '/api/admin/system-tasks' && request.method() === 'GET') {
      return json({ success: true, data: [
        { task_id: 'systask_1', type: 'log_cleanup', status: 'succeeded', created_at: 1784649600, updated_at: 1784649660 },
      ] });
    }
    if (url.pathname === '/api/admin/system-tasks/systask_1' && request.method() === 'GET') {
      return json({ success: true, data: { task_id: 'systask_1', type: 'log_cleanup', status: 'succeeded', progress: 100, result: { deleted: 286 }, created_at: 1784649600, updated_at: 1784649660 } });
    }
    if (url.pathname === '/api/admin/system-tasks/log-cleanup' && request.method() === 'POST') {
      state.logCleanupCalls += 1;
      state.logCleanupBody = request.postDataJSON();
      return json({ success: true, data: { task_id: 'systask_2', type: 'log_cleanup', status: 'pending' } }, 202);
    }
    if (url.pathname === '/api/merchant/channels') {
      return json({ success: true, data: { merchant: { merchant_id: 31, user_id: 1, merchant_status: 'active', is_merchant: true }, items: [{ id: 51, merchant_id: 31, review_id: 21, newapi_channel_id: 41, channel_name: 'OpenAI 主线路', channel_type: 1, base_url: 'https://api.merchant.example', models: ['gpt-4o-mini'], credential_fingerprint: 'a1b2c3...d4e5f6', status: 'active', created_at: '2026-07-22T14:00:00Z' }] } });
    }
    if (url.pathname === '/api/admin/merchant-applications') {
      return json({ success: true, data: { page: 1, page_size: 20, total: 1, items: [{ id: 11, user_id: 7, display_name: '东海模型服务', website_url: 'https://merchant.example', business_description: '企业模型 API 服务', status: 'pending', created_at: '2026-07-22T14:00:00Z', initial_channel: { id: 21, user_id: 7, application_id: 11, channel_name: '主线路', channel_type: 1, base_url: 'https://api.merchant.example', requested_models: ['gpt-4o-mini'], status: 'pending', credential_configured: true, created_at: '2026-07-22T14:00:00Z' } }] } });
    }
    if (url.pathname === '/api/admin/merchants') {
      return json({ success: true, data: { page: 1, page_size: 20, total: 1, items: [{ id: 31, user_id: 8, display_name: '北辰云智', website_url: 'https://beichen.example', status: 'active', approved_at: '2026-07-21T14:00:00Z' }] } });
    }
    if (url.pathname === '/api/admin/channel-reviews' && request.method() === 'GET') {
      return json({ success: true, data: { page: 1, page_size: 100, total: 1, items: [{ id: 22, user_id: 8, channel_name: 'Anthropic 备用线路', channel_type: 14, base_url: 'https://api.anthropic.example', requested_models: ['claude-sonnet-4-5'], status: 'pending', credential_configured: true, created_at: '2026-07-22T15:00:00Z' }] } });
    }
    if (url.pathname === '/api/admin/channels' && request.method() === 'GET') {
      return json({ success: true, data: { page: 1, page_size: 100, total: 2, items: [
        { id: 51, merchant_id: 31, review_id: 21, newapi_channel_id: 41, channel_name: 'OpenAI 主线路', channel_type: 1, base_url: 'https://api.merchant.example', models: ['gpt-4o-mini'], credential_fingerprint: 'a1b2c3...d4e5f6', status: 'active', created_at: '2026-07-22T14:00:00Z' },
        { id: 52, merchant_id: 32, review_id: 23, newapi_channel_id: 42, channel_name: 'OpenAI 备用线路', channel_type: 1, base_url: 'https://api.backup.example', models: ['gpt-4o-mini'], credential_fingerprint: 'd7e8f9...a0b1c2', status: 'disabled', created_at: '2026-07-22T14:10:00Z' },
      ] } });
    }
    if (url.pathname === '/api/admin/channels/52/health/reset' && request.method() === 'POST') {
      state.healthResetCalls += 1;
      return json({ success: true, data: { channel_id: 52, health_state: 'pending' } });
    }
    if (/^\/api\/admin\/channels\/\d+\/runtime$/.test(url.pathname) && request.method() === 'GET') {
      const channelID = Number(url.pathname.split('/')[4]);
      return json({ success: true, data: { newapi_channel_id: channelID === 51 ? 41 : 42, status: channelID === 51 ? 1 : 2, balance_usd_micros: channelID === 51 ? 12500000 : 3200000, balance_updated_time: 1784649600, response_time_ms: channelID === 51 ? 420 : 910, test_time: 1784649500 } });
    }
    if (/^\/api\/admin\/channels\/\d+\/test$/.test(url.pathname) && request.method() === 'POST') {
      state.channelTestCalls += 1;
      return json({ success: true, data: { status: 'succeeded', result: { time: 420 } } });
    }
    if (/^\/api\/admin\/channels\/\d+\/balance\/refresh$/.test(url.pathname) && request.method() === 'POST') {
      state.channelBalanceCalls += 1;
      return json({ success: true, data: { status: 'succeeded', result: { balance: 12.5 } } });
    }
    if (url.pathname === '/api/admin/payment-settings' && request.method() === 'GET') {
      return json({ success: true, data: { id: 11, version: 3, gateway_url: 'https://zpayz.cn', merchant_pid_masked: '202***********878', key_configured: true, payment_methods: ['alipay', 'wxpay'], minimum_usd_cents: 1000, maximum_usd_cents: 2000000, exchange_rate_cny_cents_per_usd: 720, enabled: true, notify_url: 'https://aggregate.lwylink.xyz/api/payments/easypay/notify', return_url: 'https://aggregate.lwylink.xyz/payments/return', created_at: '2026-07-22T09:00:00Z' } });
    }
    if (url.pathname === '/api/admin/payment-settings' && request.method() === 'PUT') {
      state.paymentSettingCalls += 1;
      state.paymentSettingBody = request.postDataJSON();
      return json({ success: true, data: { id: 12, version: 4, gateway_url: state.paymentSettingBody.gateway_url, merchant_pid_masked: '202***********878', key_configured: true, payment_methods: state.paymentSettingBody.payment_methods, minimum_usd_cents: state.paymentSettingBody.minimum_usd_cents, maximum_usd_cents: state.paymentSettingBody.maximum_usd_cents, exchange_rate_cny_cents_per_usd: state.paymentSettingBody.exchange_rate_cny_cents_per_usd, enabled: state.paymentSettingBody.enabled, notify_url: 'https://aggregate.lwylink.xyz/api/payments/easypay/notify', return_url: 'https://aggregate.lwylink.xyz/payments/return', created_at: '2026-07-22T10:00:00Z' } }, 201);
    }
    if (url.pathname === '/api/admin/payment-orders' && request.method() === 'GET') {
      return json({ success: true, data: { page: 1, page_size: 50, total: 2, items: [
        { id: 21, order_no: 'MR202607220001', user_id: 7, payment_type: 'alipay', usd_cents: 10000, cny_cents: 72000, exchange_rate_cny_cents_per_usd: 720, newapi_quota: 50000000, payment_status: 'paid', credit_status: 'succeeded', third_party_trade_no: 'EP202607220009', paid_at: '2026-07-22T09:05:00Z', credited_at: '2026-07-22T09:05:10Z', created_at: '2026-07-22T09:00:00Z' },
        { id: 22, order_no: 'MR202607220002', user_id: 8, payment_type: 'wxpay', usd_cents: 5000, cny_cents: 36000, exchange_rate_cny_cents_per_usd: 720, newapi_quota: 25000000, payment_status: 'paid', credit_status: 'failed', third_party_trade_no: 'EP202607220010', paid_at: '2026-07-22T09:15:00Z', created_at: '2026-07-22T09:10:00Z' },
      ] } });
    }
    if (url.pathname === '/api/admin/payment-orders/22/credit/retry' && request.method() === 'POST') {
      state.paymentRetryCalls += 1;
      return json({ success: true, data: null });
    }
    if (url.pathname === '/api/admin/withdrawals' && request.method() === 'GET') {
      return json({ success: true, data: { page: 1, page_size: 50, total: 2, items: [
        { id: 8, withdrawal_no: 'MW202607220001', merchant_id: 31, merchant_name: '北辰云智', amount_quota: 2500000, available_before_quota: 10600000, recipient_type: 'alipay', recipient_account_masked: '138****2621', recipient_account: '13800132621', status: 'pending', created_at: '2026-07-22T14:00:00Z' },
        { id: 9, withdrawal_no: 'MW202607220002', merchant_id: 32, merchant_name: '矩阵算力', amount_quota: 5000000, available_before_quota: 8200000, recipient_type: 'bank', recipient_account_masked: '6217****1098', recipient_account: '6217000012341098', status: 'approved', review_reason: '资料核对通过', created_at: '2026-07-22T13:00:00Z' },
      ] } });
    }
    if (/^\/api\/admin\/withdrawals\/\d+\/(approve|reject|paid)$/.test(url.pathname) && request.method() === 'POST') {
      state.withdrawalActionCalls += 1;
      state.withdrawalActionPath = url.pathname;
      state.withdrawalActionBody = request.postDataJSON();
      return json({ success: true, data: { id: Number(url.pathname.split('/')[4]), status: url.pathname.endsWith('/paid') ? 'paid' : url.pathname.endsWith('/reject') ? 'rejected' : 'approved' } });
    }
    if (url.pathname === '/api/admin/settlement-reconciliation' && request.method() === 'GET') {
      return json({ success: true, data: { page: 1, page_size: 50, total: 1, items: [{ id: 12, source_id: 14, newapi_log_id: 1004, newapi_user_id: 18, source_type: 2, model_name: 'mkt-ov-missing', newapi_channel_id: 99, gross_quota: 50, reason_code: 'offer_mapping_missing', details: { newapi_channel_id: 99 }, status: 'open', created_at: '2026-07-22T15:30:00Z' }] } });
    }
    if (url.pathname === '/api/admin/settlement-reconciliation/12/ignore' && request.method() === 'POST') {
      state.reconciliationIgnoreCalls += 1;
      state.reconciliationIgnoreBody = request.postDataJSON();
      return json({ success: true, data: { id: 12, status: 'ignored' } });
    }
    if (url.pathname === '/api/admin/channel-reviews/22/verify' && request.method() === 'POST') {
      state.channelVerifyCalls += 1;
      return json({ success: true, data: { review_id: 22, models: ['claude-sonnet-4-5'] } });
    }
    if (url.pathname === '/api/admin/merchant-applications/11/verify' && request.method() === 'POST') {
      state.merchantVerifyCalls += 1;
      return json({ success: true, data: { review_id: 21, models: ['gpt-4o-mini'] } });
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
  const adminMockState = {
    revealCalls: 0, merchantVerifyCalls: 0, channelVerifyCalls: 0,
    routeSaveCalls: 0, routeSaveBody: null, offerCreateCalls: 0, offerCreateBody: null,
    modelCreateCalls: 0, modelCreateBody: null, pricePublishCalls: 0, pricePublishBody: null,
    healthResetCalls: 0, paymentSettingCalls: 0, paymentSettingBody: null, paymentRetryCalls: 0,
    withdrawalCreateCalls: 0, withdrawalCreateBody: null, withdrawalActionCalls: 0,
    withdrawalActionPath: '', withdrawalActionBody: null, reconciliationIgnoreCalls: 0,
    reconciliationIgnoreBody: null, userRedemptionCalls: 0, userRedemptionBody: null,
    platformSettingCalls: 0, platformSettingBody: null, groupSettingCalls: 0, groupSettingBody: null,
    redemptionCreateCalls: 0, redemptionCreateBody: null, redemptionUpdateCalls: 0, redemptionUpdateBody: null,
    redemptionStatusCalls: 0, redemptionStatusBody: null, redemptionDeleteCalls: 0, redemptionCleanupCalls: 0,
    logCleanupCalls: 0, logCleanupBody: null, channelTestCalls: 0, channelBalanceCalls: 0,
  };
  await mockAdminAPI(page, adminMockState);
  await page.goto(baseUrl, { waitUntil: 'networkidle' });
  await page.getByRole('heading', { name: '模型服务市场' }).waitFor();
  await page.getByText('平台管理员').waitFor();
  await page.getByText('gpt-4o-mini').first().waitFor();
  await page.getByText('北辰云智').first().waitFor();
  assert(await page.getByText('GPT-5.2').count() === 0, '模型市场仍展示演示报价');

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
  await page.getByText('req-user-1001').waitFor();
  await page.getByText('缓存 1,200 Token').waitFor();
  assert(await page.locator('[aria-label="最近 7 天用量趋势"]').count() === 1, '控制台未渲染真实用量趋势');
  assert(!(await page.locator('body').innerText()).includes('mkt-ov-'), '控制台泄露了内部报价别名');
  await page.getByLabel('兑换码', { exact: true }).fill('MR-USER-CODE');
  await page.getByRole('button', { name: '立即兑换' }).click();
  await page.getByText('兑换码充值成功').waitFor();
  assert(adminMockState.userRedemptionCalls === 1 && adminMockState.userRedemptionBody.key === 'MR-USER-CODE', '用户兑换码没有调用真实接口');
  await page.getByText('真实主密钥').first().waitFor();
  assert(await page.getByRole('checkbox', { name: '启用真实主密钥' }).isChecked(), '有效密钥的启用开关未选中');
  await page.getByRole('button', { name: '复制' }).first().click();
  await page.getByText('密钥已复制').waitFor();
  assert(adminMockState.revealCalls === 1, '复制掩码密钥时未读取完整密钥');
  await page.getByRole('button', { name: '编辑路由策略' }).first().click();
  await page.getByRole('dialog', { name: '编辑路由策略' }).waitFor();
  assert(await page.getByLabel('最低成交倍率').inputValue() === '0.4', '路由弹窗未读取当前最低倍率');
  await page.getByLabel('固定商户').check();
  assert(await page.getByText('固定商户范围').isVisible(), '固定路由商户选择未显示');
  await page.getByRole('checkbox', { name: /北辰云智/ }).check();
  await page.getByRole('button', { name: '保存策略' }).click();
  assert(adminMockState.routeSaveCalls === 1, '保存路由策略没有调用真实接口');
  assert(adminMockState.routeSaveBody.mode === 'fixed' && adminMockState.routeSaveBody.merchant_ids.includes(31), '固定路由参数不正确');

  await page.getByRole('button', { name: '商户中心', exact: true }).click();
  await page.getByText('OpenAI 主线路').last().waitFor();
  await page.getByText('41', { exact: true }).waitFor();
  const importedChannels = page.getByRole('heading', { name: '已导入渠道' }).locator('xpath=ancestor::section');
  await importedChannels.getByText('正常', { exact: true }).waitFor();
  await page.getByText('0.7200x').waitFor();
  await page.getByText('$18.00', { exact: true }).waitFor();
  await page.getByRole('button', { name: '申请提现' }).click();
  const withdrawalDialog = page.getByRole('dialog', { name: '申请人工提现' });
  await withdrawalDialog.getByLabel('提现金额（美元）').fill('5');
  await withdrawalDialog.getByLabel('收款方式').selectOption('alipay');
  await withdrawalDialog.getByLabel('收款账户').fill('13800132621');
  await withdrawalDialog.getByRole('button', { name: '提交提现申请' }).click();
  assert(adminMockState.withdrawalCreateCalls === 1, '商户提现没有调用真实接口');
  assert(adminMockState.withdrawalCreateBody.amount_quota === 2500000, '提现美元金额没有转换为 New API 整数额度');
  await page.getByRole('button', { name: '创建报价' }).click();
  const quoteDialog = page.getByRole('dialog', { name: '创建模型报价' });
  await quoteDialog.getByLabel('标准模型').selectOption('101');
  await quoteDialog.getByLabel('商户渠道').selectOption('51');
  await quoteDialog.getByLabel('成交倍率').fill('0.65');
  await quoteDialog.getByRole('button', { name: '提交报价' }).click();
  assert(adminMockState.offerCreateCalls === 1, '创建报价没有调用真实接口');
  assert(adminMockState.offerCreateBody.multiplier_bps === 6500, '报价倍率没有转换为整数基点');

  await page.getByRole('button', { name: '管理后台', exact: true }).click();
  await page.getByText('$18.00', { exact: true }).first().waitFor();
  await page.getByText('126 / 128').waitFor();
  assert(await page.getByRole('button', { name: /商户审核/ }).getByText('1', { exact: true }).count() === 1, '管理员侧栏未显示真实商户待办数量');
  await page.getByRole('button', { name: '调用日志' }).click();
  await page.getByText('req-admin-9001').waitFor();
  await page.getByText('alice', { exact: true }).waitFor();
  await page.getByRole('button', { name: '模型与定价' }).click();
  await page.getByText('价格版本 v2').waitFor();
  await page.getByRole('button', { name: '新增标准模型' }).click();
  const modelDialog = page.getByRole('dialog', { name: '新增标准模型' });
  await modelDialog.getByLabel('模型名称').fill('text-embedding-3-small');
  await modelDialog.getByLabel('模型厂商').fill('OpenAI');
  await modelDialog.getByLabel('协议').selectOption('embeddings');
  await modelDialog.getByRole('button', { name: '创建模型' }).click();
  assert(adminMockState.modelCreateCalls === 1, '新增标准模型没有调用真实接口');
  await page.getByRole('button', { name: '发布价格' }).first().click();
  const priceDialog = page.getByRole('dialog', { name: '发布官方价格' });
  await priceDialog.getByRole('button', { name: '发布价格' }).click();
  assert(adminMockState.pricePublishCalls === 1, '发布官方价格没有调用真实接口');
  await page.getByRole('button', { name: '用户管理' }).click();
  await page.getByText('alice@example.com').waitFor();
  await page.getByRole('button', { name: '审计日志' }).click();
  await page.getByText('user.disable').waitFor();
  await page.getByRole('button', { name: '商户审核' }).click();
  await page.getByText('东海模型服务').waitFor();
  await page.getByRole('button', { name: '验证' }).click();
  await page.getByText('渠道验证通过').waitFor();
  assert(adminMockState.merchantVerifyCalls === 1, '管理员验证没有调用真实商户审核接口');
  await page.getByRole('button', { name: '渠道与探活' }).click();
  await page.getByText('Anthropic 备用线路').waitFor();
  await page.getByText('New API 41').waitFor();
  await page.getByText('已停用').waitFor();
  const importedAdminChannels = page.getByRole('heading', { name: '已导入渠道' }).locator('xpath=ancestor::section');
  await importedAdminChannels.getByText('$12.50', { exact: true }).waitFor();
  await importedAdminChannels.getByRole('button', { name: '测试渠道' }).first().click();
  await page.getByText('渠道测试完成').waitFor();
  await importedAdminChannels.getByRole('button', { name: '刷新余额' }).first().click();
  await page.getByText('渠道余额已刷新').waitFor();
  assert(adminMockState.channelTestCalls === 1 && adminMockState.channelBalanceCalls === 1, '渠道测试或余额刷新没有调用真实接口');
  await page.getByRole('button', { name: '恢复渠道' }).click();
  await page.getByText('渠道已进入恢复探测').waitFor();
  assert(adminMockState.healthResetCalls === 1, '恢复渠道没有调用真实健康接口');
  await page.getByRole('button', { name: '验证' }).click();
  await page.getByText('渠道验证通过').waitFor();
  assert(adminMockState.channelVerifyCalls === 1, '管理员渠道页验证没有调用真实渠道审核接口');
  await page.getByRole('button', { name: '平台设置' }).click();
  await page.getByRole('heading', { name: '邮件与注册' }).waitFor();
  assert(await page.getByLabel('SMTP 服务器').inputValue() === 'smtp.example.com', '平台设置未加载 SMTP 配置');
  await page.getByRole('button', { name: '保存邮件与注册' }).click();
  await page.getByText('邮件与注册设置已保存').waitFor();
  assert(adminMockState.platformSettingCalls === 1 && adminMockState.platformSettingBody.smtp.password === '', '邮件注册设置没有按只写密码规则提交');
  await page.getByRole('button', { name: '分组倍率' }).click();
  await page.getByRole('heading', { name: '分组倍率' }).waitFor();
  await page.getByLabel('vip 分组倍率').fill('0.75');
  await page.getByRole('button', { name: '保存分组倍率' }).click();
  await page.getByText('分组倍率已保存').waitFor();
  assert(adminMockState.groupSettingCalls === 1 && adminMockState.groupSettingBody.groups.find((group) => group.name === 'vip')?.ratio_bps === 7500, '分组倍率没有转换为整数基点');
  await page.getByRole('button', { name: '兑换码管理' }).click();
  await page.getByText('MR-JULY-2026').waitFor();
  await page.getByRole('button', { name: '新建兑换码' }).click();
  const redemptionDialog = page.getByRole('dialog', { name: '新建兑换码' });
  await redemptionDialog.getByLabel('名称').fill('批量活动码');
  await redemptionDialog.getByLabel('单码额度（美元）').fill('2');
  await redemptionDialog.getByLabel('生成数量').fill('2');
  await redemptionDialog.getByRole('button', { name: '创建兑换码' }).click();
  await page.getByText('兑换码已创建').waitFor();
  assert(adminMockState.redemptionCreateCalls === 1 && adminMockState.redemptionCreateBody.quota === 1000000 && adminMockState.redemptionCreateBody.count === 2, '兑换码额度或数量没有按整数规则提交');
  await page.getByRole('button', { name: '停用', exact: true }).first().click();
  await page.getByText('兑换码已停用').waitFor();
  assert(adminMockState.redemptionStatusCalls === 1 && adminMockState.redemptionStatusBody.enabled === false, '兑换码启停没有调用真实接口');
  await page.getByRole('button', { name: '编辑', exact: true }).first().click();
  const editRedemptionDialog = page.getByRole('dialog', { name: '编辑兑换码' });
  await editRedemptionDialog.getByLabel('名称').fill('更新活动码');
  await editRedemptionDialog.getByLabel('单码额度（美元）').fill('1.4');
  await editRedemptionDialog.getByRole('button', { name: '保存兑换码' }).click();
  await page.getByText('兑换码已更新').waitFor();
  assert(adminMockState.redemptionUpdateCalls === 1 && adminMockState.redemptionUpdateBody.quota === 700000 && !('count' in adminMockState.redemptionUpdateBody), '编辑兑换码参数不符合整数额度规则');
  page.once('dialog', (dialog) => dialog.accept());
  await page.getByRole('button', { name: '删除', exact: true }).first().click();
  await page.getByText('兑换码已删除').waitFor();
  assert(adminMockState.redemptionDeleteCalls === 1, '删除兑换码没有调用真实接口');
  page.once('dialog', (dialog) => dialog.accept());
  await page.getByRole('button', { name: '清理失效' }).click();
  await page.getByText('已清理 1 条失效兑换码').waitFor();
  assert(adminMockState.redemptionCleanupCalls === 1, '清理失效兑换码没有调用真实接口');
  await page.getByRole('button', { name: '系统任务' }).click();
  await page.getByText('systask_1').waitFor();
  await page.getByRole('button', { name: '查看详情' }).click();
  const taskDetailDialog = page.getByRole('dialog', { name: '系统任务详情' });
  await taskDetailDialog.getByText('已删除日志 286 条').waitFor();
  await taskDetailDialog.getByRole('button', { name: '关闭' }).last().click();
  await page.getByRole('button', { name: '创建日志清理任务' }).click();
  const cleanupDialog = page.getByRole('dialog', { name: '创建日志清理任务' });
  await cleanupDialog.getByLabel('清理截止时间').fill('2026-07-20T00:00');
  await cleanupDialog.getByRole('button', { name: '创建任务' }).click();
  await page.getByText('日志清理任务已创建').waitFor();
  assert(adminMockState.logCleanupCalls === 1 && Number.isInteger(adminMockState.logCleanupBody.before_timestamp), '日志清理任务没有提交整数时间戳');
  await page.getByRole('button', { name: '支付与充值' }).click();
  await page.getByRole('heading', { name: '易支付设置' }).waitFor();
  assert(
    await page.getByText('https://zpayz.cn').isVisible(),
    '易支付网关未显示',
  );
  const bodyText = await page.locator('body').innerText();
  assert(!/KEY[^\n]{0,24}[A-Za-z0-9]{24,}/.test(bodyText), '页面疑似显示了完整支付密钥');
  await page.getByText('MR202607220002').waitFor();
  await page.getByRole('button', { name: '编辑配置' }).click();
  const paymentDialog = page.getByRole('dialog', { name: '编辑易支付配置' });
  await paymentDialog.getByLabel('人民币兑美元汇率').fill('7.25');
  await paymentDialog.getByRole('button', { name: '保存支付配置' }).click();
  assert(adminMockState.paymentSettingCalls === 1, '支付配置没有调用真实接口');
  assert(adminMockState.paymentSettingBody.exchange_rate_cny_cents_per_usd === 725, '充值汇率没有转换为整数人民币分');
  await page.getByRole('button', { name: '重试入账' }).click();
  await page.getByText('入账任务已重新排队').waitFor();
  assert(adminMockState.paymentRetryCalls === 1, '失败入账没有调用重试接口');
  await page.getByRole('button', { name: '提现审核' }).click();
  await page.getByText('MW202607220001').waitFor();
  await page.getByRole('button', { name: '通过' }).first().click();
  const approveDialog = page.getByRole('dialog', { name: '通过提现申请' });
  await approveDialog.getByLabel('审核原因').fill('账本与收款信息核对通过');
  await approveDialog.getByRole('button', { name: '确认通过' }).click();
  assert(adminMockState.withdrawalActionCalls === 1 && adminMockState.withdrawalActionPath.endsWith('/8/approve'), '提现审核通过没有调用真实接口');
  await page.getByRole('button', { name: '确认打款' }).click();
  const paidDialog = page.getByRole('dialog', { name: '确认线下打款' });
  await paidDialog.getByLabel('打款流水号').fill('BANK-009');
  await paidDialog.getByLabel('凭证说明').fill('银行回单 009');
  await paidDialog.getByRole('button', { name: '确认已打款' }).click();
  assert(adminMockState.withdrawalActionCalls === 2 && adminMockState.withdrawalActionPath.endsWith('/9/paid'), '确认线下打款没有调用真实接口');
  await page.getByRole('button', { name: '忽略异常' }).click();
  const ignoreDialog = page.getByRole('dialog', { name: '忽略对账异常' });
  await ignoreDialog.getByLabel('处置原因').fill('确认不是平台路由产生的日志');
  await ignoreDialog.getByRole('button', { name: '确认忽略' }).click();
  assert(adminMockState.reconciliationIgnoreCalls === 1, '忽略对账异常没有调用真实接口');

  const anonymousPage = await browser.newPage({ viewport: { width: 390, height: 844 } });
  let anonymousLoggedIn = false;
  let merchantApplication = null;
  let merchantApplicationCalls = 0;
  let paymentOrderCalls = 0;
  await anonymousPage.route('**/api/**', (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const fulfill = (data, status = 200) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(data) });
    if (url.pathname === '/api/me' && anonymousLoggedIn) return fulfill({ success: true, data: { id: 7, username: 'alice', display_name: 'Alice', role: 1, status: 1, group: 'default', quota: 500000, used_quota: 120000, is_merchant: false, merchant_status: 'none' } });
    if (url.pathname === '/api/me') return fulfill({ success: false, error: { code: 'authentication_required', message: '请先登录' } }, 401);
    if (url.pathname === '/api/models') return fulfill({ success: true, data: [{ id: 101, public_name: 'gpt-4o-mini', provider: 'OpenAI', protocol: 'chat_completions', status: 'active', price: { version: 2, billing_mode: 'token', model_ratio_micros: 600000, completion_ratio_micros: 4000000 } }] });
    if (url.pathname === '/api/offers') return fulfill({ success: true, data: [{ offer_id: 201, merchant_id: 31, merchant_name: '北辰云智', merchant_channel_id: 51, channel_name: 'OpenAI 主线路', model: 'gpt-4o-mini', protocol: 'chat_completions', multiplier_bps: 7200, health_state: 'healthy', availability_ppm: 999800, ttft_p95_ms: 1200 }] });
    if (url.pathname === '/api/auth/login' && request.method() === 'POST') {
      anonymousLoggedIn = true;
      return fulfill({ success: true, data: { id: 7, username: 'alice', display_name: 'Alice', role: 1, status: 1, group: 'default' } });
    }
    if (url.pathname === '/api/keys') return fulfill({ success: true, data: { page: 1, page_size: 20, total: 0, items: [] } });
    if (url.pathname === '/api/payment-settings') return fulfill({ success: true, data: { version: 3, payment_methods: ['alipay', 'wxpay'], minimum_usd_cents: 1000, maximum_usd_cents: 2000000, exchange_rate_cny_cents_per_usd: 720, enabled: true } });
    if (url.pathname === '/api/payment-orders' && request.method() === 'GET') return fulfill({ success: true, data: { page: 1, page_size: 20, total: 0, items: [] } });
    if (url.pathname === '/api/payment-orders' && request.method() === 'POST') {
      paymentOrderCalls += 1;
      return fulfill({ success: true, data: { order: { id: 31, order_no: 'MR-USER-1', usd_cents: 10000, cny_cents: 72000, payment_status: 'pending', credit_status: 'pending' }, payment_url: `${baseUrl}/payment-target` } }, 201);
    }
    if (url.pathname === '/api/merchant/application' && request.method() === 'GET') return fulfill({ success: true, data: merchantApplication });
    if (url.pathname === '/api/merchant/application' && request.method() === 'POST') {
      merchantApplicationCalls += 1;
      merchantApplication = { id: 11, user_id: 7, display_name: '北辰云智', website_url: 'https://merchant.example', business_description: '企业模型 API 服务', status: 'pending', initial_channel: { id: 21, channel_name: '主线路', channel_type: 1, base_url: 'https://api.merchant.example', requested_models: ['gpt-4o-mini'], status: 'pending', credential_configured: true } };
      return fulfill({ success: true, data: merchantApplication }, 201);
    }
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
  await anonymousPage.getByRole('button', { name: '申请成为商户' }).click();
  const merchantDialog = anonymousPage.getByRole('dialog', { name: '申请成为商户' });
  await merchantDialog.getByLabel('商户名称').fill('北辰云智');
  await merchantDialog.getByLabel('网站地址').fill('https://merchant.example');
  await merchantDialog.getByLabel('上游基础地址').fill('https://api.merchant.example/v1');
  await merchantDialog.getByLabel('上游 API 密钥').fill('sk-upstream-secret');
  await merchantDialog.getByLabel('业务说明').fill('企业模型 API 服务');
  await merchantDialog.getByRole('button', { name: '提交申请' }).click();
  await anonymousPage.getByText('商户申请已提交').waitFor();
  assert(merchantApplicationCalls === 1, '商户申请没有调用平台接口');
  await anonymousPage.getByText('等待管理员审核').waitFor();
  await anonymousPage.getByRole('button', { name: '立即充值' }).click();
  const rechargeDialog = anonymousPage.getByRole('dialog', { name: '账户充值' });
  await rechargeDialog.getByLabel('充值额度（美元）').fill('100');
  await rechargeDialog.getByText('¥720.00').waitFor();
  await rechargeDialog.getByRole('button', { name: '前往支付' }).click();
  await anonymousPage.waitForURL('**/payment-target');
  assert(paymentOrderCalls === 1, '充值订单没有调用平台接口');
  await anonymousPage.close();

  const returnPage = await browser.newPage({ viewport: { width: 390, height: 844 } });
  const returnMethods = [];
  await returnPage.route('**/api/**', (route) => {
    const request = route.request();
    const url = new URL(request.url());
    returnMethods.push(`${request.method()} ${url.pathname}`);
    const fulfill = (data, status = 200) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(data) });
    if (url.pathname === '/api/me') return fulfill({ success: true, data: { id: 7, username: 'alice', display_name: 'Alice', role: 1, status: 1, quota: 500000 } });
    if (url.pathname === '/api/models') return fulfill({ success: true, data: [] });
    if (url.pathname === '/api/offers') return fulfill({ success: true, data: [] });
    if (url.pathname === '/api/payment-orders/MR-RETURN') return fulfill({ success: true, data: { id: 31, order_no: 'MR-RETURN', user_id: 7, usd_cents: 10000, cny_cents: 72000, payment_status: 'paid', credit_status: 'pending', paid_at: '2026-07-22T10:00:00Z' } });
    return fulfill({ success: false, error: { code: 'not_mocked', message: url.pathname } }, 404);
  });
  await returnPage.goto(`${baseUrl}/payments/return?out_trade_no=MR-RETURN`, { waitUntil: 'networkidle' });
  await returnPage.getByRole('heading', { name: '支付结果' }).waitFor();
  await returnPage.getByText('已支付', { exact: true }).waitFor();
  await returnPage.getByText('等待入账', { exact: true }).waitFor();
  assert(returnMethods.every((item) => item.startsWith('GET ')), `支付返回页触发了写操作：${returnMethods.join(', ')}`);
  await returnPage.close();

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

  console.log('UI 冒烟测试通过：角色导航、真实账号数据、路由策略、报价、支付、结算和人工提现交互正常。');
} finally {
  await browser.close();
  await stopLocalPreview(previewServer);
}
