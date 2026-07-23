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
const pageErrors = [];
page.on('pageerror', (error) => pageErrors.push(error.message));
page.setDefaultTimeout(10000);

async function mockAdminAPI(targetPage, state, account = null) {
  await targetPage.route('**/api/**', async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const json = (data, status = 200) => route.fulfill({
      status,
      contentType: 'application/json',
      body: JSON.stringify(data),
    });
    if (url.pathname === '/api/me') {
      return json({ success: true, data: account || { id: 1, username: 'modelroute', display_name: '平台管理员', role: 100, status: 1, group: 'default', quota: 128642000, used_quota: 4268000, account_type: 'admin', is_merchant: true, merchant_name: 'ModelRoute 官方', merchant_status: 'active' } });
    }
    if (url.pathname === '/api/announcements' && request.method() === 'GET') {
      return json({ success: true, data: { unread_count: state.announcementReadCalls ? 0 : 1, items: [{ id: 9, title: '商户结算安排', body: '本周五统一处理人工提现。', audiences: ['admin'], pinned: true, status: 'published', is_read: state.announcementReadCalls > 0, publish_at: '2026-07-22T08:00:00Z' }] } });
    }
    if (url.pathname === '/api/announcements/9/read' && request.method() === 'POST') {
      state.announcementReadCalls += 1;
      return json({ success: true, data: null });
    }
    if (url.pathname === '/api/admin/announcements' && request.method() === 'GET') {
      return json({ success: true, data: [{ id: 9, title: '商户结算安排', body: '本周五统一处理人工提现。', audiences: ['admin'], pinned: true, status: 'published', publish_at: '2026-07-22T08:00:00Z', created_at: '2026-07-22T07:00:00Z' }] });
    }
    if (url.pathname === '/api/admin/announcements' && request.method() === 'POST') {
      state.announcementCreateCalls += 1;
      state.announcementCreateBody = request.postDataJSON();
      return json({ success: true, data: { id: 10, status: 'draft', ...state.announcementCreateBody } }, 201);
    }
    if (url.pathname === '/api/models') {
      return json({ success: true, data: [{ id: 101, public_name: 'gpt-4o-mini', provider: 'OpenAI', protocol: 'chat_completions', status: 'active', adapter_hint: 'openai_compatible', price: { id: 501, version: 2, price_kind: 'text', tiers: [{ tier_order: 1, minimum_input_tokens: 0, maximum_input_tokens: null, input_price_micros: 150000, output_price_micros: 600000, cache_read_price_micros: 75000, cache_write_price_micros: 200000 }] } }] });
    }
    if (url.pathname === '/api/merchant/channel-defaults' && request.method() === 'GET') {
      return json({ success: true, data: { configured: true, channel_type: 'openai_compatible', base_url: 'https://api.lwylink.xyz' } });
    }
    if (url.pathname === '/api/offers') {
      return json({ success: true, data: [
        { offer_id: 201, merchant_id: 31, merchant_name: '北辰云智', model: 'gpt-4o-mini', protocol: 'chat_completions', multiplier_bps: 7200, price_tiers: [{ tier_order: 1, minimum_input_tokens: 0, maximum_input_tokens: null, input_price_micros: 108000, output_price_micros: 432000, cache_read_price_micros: 54000, cache_write_price_micros: 144000 }], channel_count: 1, health_state: 'healthy', availability_ppm: 999800, ttft_p95_ms: 1200 },
        { offer_id: 202, merchant_id: 32, merchant_name: '矩阵算力', model: 'gpt-4o-mini', protocol: 'chat_completions', multiplier_bps: 5800, price_tiers: [{ tier_order: 1, minimum_input_tokens: 0, maximum_input_tokens: null, input_price_micros: 87000, output_price_micros: 348000, cache_read_price_micros: 43500, cache_write_price_micros: 116000 }], channel_count: 1, health_state: 'degraded', availability_ppm: 987500, ttft_p95_ms: 1650 },
      ] });
    }
    if (url.pathname === '/api/keys' && request.method() === 'GET') {
      return json({ success: true, data: { page: 1, page_size: 20, total: 1, items: [{ id: 9, name: '真实主密钥', key: 'full-secret-key', status: 1, created_time: 1784690000, accessed_time: 1784692000, expired_time: -1, unlimited_quota: true }] } });
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
      return json({ success: true, data: [{ offer_id: 201, model: 'gpt-4o-mini', protocol: 'chat_completions', multiplier_bps: 7200, call_count: 32400, token_usage: 2600000000, last_called_at: new Date(Date.now() - 10 * 60_000).toISOString(), price_tiers: [{ tier_order: 1, minimum_input_tokens: 0, maximum_input_tokens: null, input_price_micros: 108000, output_price_micros: 432000, cache_read_price_micros: 54000, cache_write_price_micros: 144000 }], channels: [{ merchant_channel_id: 51, channel_name: 'OpenAI 主线路', health_state: 'healthy', availability_ppm: 999800, ttft_p95_ms: 1200 }], offer_status: 'active', health_state: 'healthy', availability_ppm: 999800, ttft_p95_ms: 1200 }] });
    }
    if (url.pathname === '/api/merchant/health' && request.method() === 'GET') {
      return json({ success: true, data: [{ offer_id: 201, model: 'gpt-4o-mini', protocol: 'chat_completions', multiplier_bps: 7200, call_count: 32400, token_usage: 2600000000, last_called_at: new Date(Date.now() - 10 * 60_000).toISOString(), price_tiers: [{ tier_order: 1, minimum_input_tokens: 0, maximum_input_tokens: null, input_price_micros: 108000, output_price_micros: 432000, cache_read_price_micros: 54000, cache_write_price_micros: 144000 }], channels: [{ merchant_channel_id: 51, channel_name: 'OpenAI 主线路', health_state: 'healthy', availability_ppm: 999800, ttft_p95_ms: 1200 }], offer_status: 'active', health_state: 'healthy', availability_ppm: 999800, ttft_p95_ms: 1200 }] });
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
    if (url.pathname === '/api/merchant/offers/101' && request.method() === 'PUT') {
      state.offerCreateCalls += 1;
      state.offerCreateBody = request.postDataJSON();
      return json({ success: true, data: { id: 203, status: 'active', ...state.offerCreateBody } }, 201);
    }
    if (url.pathname === '/api/merchant/offers/101' && request.method() === 'DELETE') {
      state.offerRetireCalls += 1;
      return json({ success: true, data: null });
    }
    if (url.pathname === '/api/admin/models' && request.method() === 'GET') {
      return json({ success: true, data: [{ id: 101, public_name: 'gpt-4o-mini', provider: 'OpenAI', protocol: 'chat_completions', status: 'active', adapter_hint: 'openai_compatible', price: { id: 501, version: 2, price_kind: 'text', tiers: [{ tier_order: 1, minimum_input_tokens: 0, maximum_input_tokens: null, input_price_micros: 150000, output_price_micros: 600000, cache_read_price_micros: 75000, cache_write_price_micros: 200000 }] } }] });
    }
    if (url.pathname === '/api/admin/models' && request.method() === 'POST') {
      state.modelCreateCalls += 1;
      state.modelCreateBody = request.postDataJSON();
      return json({ success: true, data: { id: 102, status: 'active', ...state.modelCreateBody } }, 201);
    }
    if (url.pathname === '/api/admin/models/detect' && request.method() === 'POST') {
      state.modelDetectCalls += 1;
      state.modelDetectBody = request.postDataJSON();
      return json({ success: true, data: { adapter_type: 'openai_compatible', capabilities: ['chat_completions'], latency_ms: 286 } });
    }
    if (url.pathname === '/api/admin/models/101/prices' && request.method() === 'POST') {
      state.pricePublishCalls += 1;
      state.pricePublishBody = request.postDataJSON();
      return json({ success: true, data: { id: 502, version: 3, ...state.pricePublishBody } }, 201);
    }
    if (url.pathname === '/api/admin/models/101' && request.method() === 'DELETE') {
      state.modelDeleteCalls += 1;
      return json({ success: true, data: { result: 'archived' } });
    }
    if (url.pathname === '/api/admin/users') {
      const allUsers = [
        { id: 1, username: 'modelroute', display_name: '平台管理员', email: '', role: 100, status: 1, quota: 128642000, used_quota: 4268000, request_count: 2846, account_type: 'admin' },
        { id: 7, username: 'alice', display_name: 'Alice', email: 'alice@example.com', role: 1, status: 1, quota: 500000, used_quota: 120000, request_count: 18, account_type: 'user' },
        { id: 8, username: 'merchant', display_name: '北辰云智', email: 'merchant@example.com', role: 1, status: 1, quota: 800000, used_quota: 320000, request_count: 35, account_type: 'merchant', merchant_status: 'active' },
      ];
      const type = url.searchParams.get('type');
      const items = type ? allUsers.filter((user) => user.account_type === type) : allUsers;
      return json({ success: true, data: { page: 1, page_size: 50, total: items.length, items } });
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
    if (url.pathname === '/api/admin/channel-logs' && request.method() === 'GET') {
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
    if (url.pathname === '/api/merchant/channel-tests' && request.method() === 'POST') {
      state.merchantChannelTestCalls += 1;
      state.merchantChannelTestBody = request.postDataJSON();
      return json({ success: true, data: { id: 91, canonical_model_id: 101, adapter_type: 'openai_compatible', capabilities: ['chat_completions'], latency_ms: 328, test_receipt: 'receipt-for-channel-test-123456', expires_at: '2026-07-22T16:15:00Z' } }, 201);
    }
    if (url.pathname === '/api/merchant/channel-models/sync' && request.method() === 'POST') {
      state.channelModelSyncCalls += 1;
      state.channelModelSyncBody = request.postDataJSON();
      return json({ success: true, data: { synced_at: '2026-07-23T08:00:00Z', items: [
        { upstream_model_name: 'gpt-4o-mini', canonical_model_id: 101, public_name: 'gpt-4o-mini', provider: 'OpenAI', match_status: 'matched' },
        { upstream_model_name: 'unlisted-chat-model', canonical_model_id: null, match_status: 'unmatched' },
      ] } });
    }
    if (url.pathname === '/api/merchant/channels' && request.method() === 'POST') {
      state.channelSubmitCalls += 1;
      state.channelSubmitBody = request.postDataJSON();
      return json({ success: true, data: { id: 25, status: 'pending' } }, 201);
    }
    if (url.pathname === '/api/merchant/channels' && request.method() === 'GET') {
      return json({ success: true, data: { merchant: { user_id: 1, merchant_name: 'ModelRoute 官方', merchant_status: 'active', is_merchant: true }, reviews: [
        { id: 21, user_id: 1, channel_name: 'OpenAI 主线路', base_url: 'https://api.merchant.example', status: 'approved', credential_configured: false, created_at: '2026-07-22T14:00:00Z' },
        { id: 24, user_id: 1, channel_name: 'Gemini 测试线路', base_url: 'https://api.gemini.example', status: 'rejected', credential_configured: false, review_reason: '模型响应异常', created_at: '2026-07-22T15:00:00Z' },
      ], items: [{ id: 51, user_id: 1, merchant_name: 'ModelRoute 官方', review_id: 21, canonical_model_id: 101, model: 'gpt-4o-mini', matched_models: ['gpt-4o-mini'], provider: 'OpenAI', protocol: 'chat_completions', adapter_type: 'openai_compatible', channel_name: 'OpenAI 主线路', base_url: 'https://api.merchant.example', credential_fingerprint: 'a1b2c3...d4e5f6', status: 'active', health_state: 'healthy', created_at: '2026-07-22T14:00:00Z' }] } });
    }
    if (url.pathname === '/api/admin/merchant-applications') {
      return json({ success: true, data: { page: 1, page_size: 20, total: 1, items: [{ id: 11, user_id: 7, display_name: '东海模型服务', website_url: 'https://merchant.example', business_description: '企业模型 API 服务', status: 'pending', created_at: '2026-07-22T14:00:00Z', initial_channel: { id: 21, user_id: 7, application_id: 11, canonical_model_id: 101, model: 'gpt-4o-mini', adapter_type: 'openai_compatible', channel_name: '主线路', base_url: 'https://api.merchant.example', status: 'pending', credential_configured: true, created_at: '2026-07-22T14:00:00Z' } }] } });
    }
    if (url.pathname === '/api/admin/channel-reviews' && request.method() === 'GET') {
      return json({ success: true, data: { page: 1, page_size: 100, total: 1, items: [{ id: 22, user_id: 8, canonical_model_id: 102, model: 'claude-sonnet-4-5', adapter_type: 'anthropic', channel_name: 'Anthropic 备用线路', base_url: 'https://api.anthropic.example', status: 'pending', credential_configured: true, models: [{ id: 221, upstream_model_name: 'claude-sonnet-4-5', match_status: 'matched', test_status: 'pending' }], created_at: '2026-07-22T15:00:00Z' }] } });
    }
    if (url.pathname === '/api/admin/channels' && request.method() === 'GET') {
      return json({ success: true, data: { page: 1, page_size: 100, total: 2, items: [
        { id: 51, user_id: 8, merchant_name: '北辰云智', review_id: 21, canonical_model_id: 101, model: 'gpt-4o-mini', matched_models: ['gpt-4o-mini'], adapter_type: 'openai_compatible', channel_name: 'OpenAI 主线路', base_url: 'https://api.merchant.example', credential_fingerprint: 'a1b2c3...d4e5f6', status: 'active', health_state: 'healthy', created_at: '2026-07-22T14:00:00Z' },
        { id: 52, user_id: 9, merchant_name: '矩阵算力', review_id: 23, canonical_model_id: 101, model: 'gpt-4o-mini', matched_models: ['gpt-4o-mini'], adapter_type: 'openai_compatible', channel_name: 'OpenAI 备用线路', base_url: 'https://api.backup.example', credential_fingerprint: 'd7e8f9...a0b1c2', status: 'disabled', health_state: 'cooling', created_at: '2026-07-22T14:10:00Z' },
      ] } });
    }
    if (url.pathname === '/api/admin/channels/52/health/reset' && request.method() === 'POST') {
      state.healthResetCalls += 1;
      return json({ success: true, data: { channel_id: 52, health_state: 'pending' } });
    }
    if (url.pathname === '/api/admin/payment-settings' && request.method() === 'GET') {
      return json({ success: true, data: { id: 11, version: 3, gateway_url: 'https://zpayz.cn', merchant_pid_masked: '202***********878', key_configured: true, payment_methods: ['alipay', 'wxpay'], minimum_cny_cents: 100, maximum_cny_cents: 2000000, enabled: true, notify_url: 'https://aggregate.lwylink.xyz/api/payments/easypay/notify', return_url: 'https://aggregate.lwylink.xyz/payments/return', created_at: '2026-07-22T09:00:00Z' } });
    }
    if (url.pathname === '/api/admin/payment-settings' && request.method() === 'PUT') {
      state.paymentSettingCalls += 1;
      state.paymentSettingBody = request.postDataJSON();
      return json({ success: true, data: { id: 12, version: 4, gateway_url: state.paymentSettingBody.gateway_url, merchant_pid_masked: '202***********878', key_configured: true, payment_methods: state.paymentSettingBody.payment_methods, minimum_cny_cents: state.paymentSettingBody.minimum_cny_cents, maximum_cny_cents: state.paymentSettingBody.maximum_cny_cents, enabled: state.paymentSettingBody.enabled, notify_url: 'https://aggregate.lwylink.xyz/api/payments/easypay/notify', return_url: 'https://aggregate.lwylink.xyz/payments/return', created_at: '2026-07-22T10:00:00Z' } }, 201);
    }
    if (url.pathname === '/api/admin/payment-orders' && request.method() === 'GET') {
      return json({ success: true, data: { page: 1, page_size: 50, total: 2, items: [
        { id: 21, order_no: 'MR202607220001', user_id: 7, payment_type: 'alipay', amount_cny_cents: 10000, newapi_quota: 50000000, payment_status: 'paid', credit_status: 'succeeded', third_party_trade_no: 'EP202607220009', paid_at: '2026-07-22T09:05:00Z', credited_at: '2026-07-22T09:05:10Z', created_at: '2026-07-22T09:00:00Z' },
        { id: 22, order_no: 'MR202607220002', user_id: 8, payment_type: 'wxpay', amount_cny_cents: 5000, newapi_quota: 25000000, payment_status: 'paid', credit_status: 'failed', third_party_trade_no: 'EP202607220010', paid_at: '2026-07-22T09:15:00Z', created_at: '2026-07-22T09:10:00Z' },
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
      return json({ success: true, data: { page: 1, page_size: 50, total: 1, items: [{ id: 12, source_id: 14, newapi_log_id: 1004, source_type: 2, model_name: 'mkt-ov-missing', channel_id: 51, gross_quota: 50, reason_code: 'offer_mapping_missing', status: 'open', created_at: '2026-07-22T15:30:00Z' }] } });
    }
    if (url.pathname === '/api/admin/settlement-reconciliation/12/ignore' && request.method() === 'POST') {
      state.reconciliationIgnoreCalls += 1;
      state.reconciliationIgnoreBody = request.postDataJSON();
      return json({ success: true, data: { id: 12, status: 'ignored' } });
    }
    if (url.pathname === '/api/admin/channel-reviews/22/verify' && request.method() === 'POST') {
      state.channelVerifyCalls += 1;
      return json({ success: true, data: { review_id: 22, adapter_type: 'anthropic', capabilities: ['chat_completions'], latency_ms: 315 } });
    }
    if (url.pathname === '/api/admin/merchant-applications/11/verify' && request.method() === 'POST') {
      state.merchantVerifyCalls += 1;
      return json({ success: true, data: { review_id: 21, adapter_type: 'openai_compatible', capabilities: ['chat_completions'], latency_ms: 280 } });
    }
    if (url.pathname.startsWith('/api/admin/users/') || url.pathname.startsWith('/api/keys')) {
      return json({ success: true, data: null });
    }
    return json({ success: false, error: { code: 'not_mocked', message: url.pathname } }, 404);
  });
}

try {
  const adminMockState = {
    merchantVerifyCalls: 0, channelVerifyCalls: 0,
    routeSaveCalls: 0, routeSaveBody: null, offerCreateCalls: 0, offerCreateBody: null, offerRetireCalls: 0,
    merchantChannelTestCalls: 0, merchantChannelTestBody: null,
    channelModelSyncCalls: 0, channelModelSyncBody: null, channelSubmitCalls: 0, channelSubmitBody: null,
    modelCreateCalls: 0, modelCreateBody: null, modelDetectCalls: 0, modelDetectBody: null,
    modelDeleteCalls: 0, pricePublishCalls: 0, pricePublishBody: null,
    healthResetCalls: 0, paymentSettingCalls: 0, paymentSettingBody: null, paymentRetryCalls: 0,
    withdrawalCreateCalls: 0, withdrawalCreateBody: null, withdrawalActionCalls: 0,
    withdrawalActionPath: '', withdrawalActionBody: null, reconciliationIgnoreCalls: 0,
    reconciliationIgnoreBody: null, userRedemptionCalls: 0, userRedemptionBody: null,
    platformSettingCalls: 0, platformSettingBody: null,
    redemptionCreateCalls: 0, redemptionCreateBody: null, redemptionUpdateCalls: 0, redemptionUpdateBody: null,
    redemptionStatusCalls: 0, redemptionStatusBody: null, redemptionDeleteCalls: 0, redemptionCleanupCalls: 0,
    announcementReadCalls: 0, announcementCreateCalls: 0, announcementCreateBody: null,
  };
  await mockAdminAPI(page, adminMockState);
  await page.goto(baseUrl, { waitUntil: 'networkidle' });
  await page.getByRole('heading', { name: '模型服务市场' }).waitFor();
  await page.getByText('平台管理员').waitFor();
  await page.getByText('gpt-4o-mini').first().waitFor();
  await page.getByText('北辰云智').first().waitFor();
  assert(await page.getByText('GPT-5.2').count() === 0, '模型市场仍展示演示报价');
  assert(await page.getByRole('button', { name: '接入文档', exact: true }).count() === 0, '模型市场仍显示无效接入文档按钮');
  assert(await page.getByRole('button', { name: '帮助中心' }).count() === 0, '顶部仍保留无行为帮助按钮');
  await page.getByRole('button', { name: '公告', exact: true }).click();
  await page.getByRole('heading', { name: '公告中心' }).waitFor();
  await page.getByRole('button', { name: /商户结算安排/ }).click();
  await page.getByText('本周五统一处理人工提现。').waitFor();
  assert(adminMockState.announcementReadCalls === 1, '打开公告详情没有标记已读');
  await page.getByRole('button', { name: '关闭公告' }).click();
  await page.getByRole('button', { name: /平台管理员/ }).click();
  await page.getByRole('heading', { name: '账户概览' }).waitFor();

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
  const consoleSidebar = page.getByRole('complementary', { name: '控制台功能' });
  for (const label of ['账户概览', 'API 密钥与路由', '调用记录', '充值中心', '账户设置']) {
    assert(await consoleSidebar.getByRole('button', { name: label, exact: true }).count() === 1, `用户控制台侧栏缺少${label}`);
  }
  assert(!(await page.locator('body').innerText()).includes('账户分组'), '用户控制台仍显示外围分组');
  await page.getByRole('main').getByText('¥257.28', { exact: true }).waitFor();
  assert(await page.locator('[aria-label="最近 7 天用量趋势"]').count() === 1, '账户概览未渲染真实用量趋势');
  await consoleSidebar.getByRole('button', { name: '调用记录', exact: true }).click();
  await page.getByText('req-user-1001').waitFor();
  await page.getByText('缓存 1,200 Token').waitFor();
  assert(await page.locator('[aria-label="最近 7 天用量趋势"]').count() === 0, '调用记录页仍重复渲染用量趋势');
  assert(!(await page.locator('body').innerText()).includes('mkt-ov-'), '控制台泄露了内部报价别名');
  await consoleSidebar.getByRole('button', { name: '充值中心', exact: true }).click();
  await page.getByLabel('兑换码', { exact: true }).fill('MR-USER-CODE');
  await page.getByRole('button', { name: '立即兑换' }).click();
  await page.getByText('兑换码充值成功').waitFor();
  assert(adminMockState.userRedemptionCalls === 1 && adminMockState.userRedemptionBody.key === 'MR-USER-CODE', '用户兑换码没有调用真实接口');
  await consoleSidebar.getByRole('button', { name: 'API 密钥与路由', exact: true }).click();
  await page.getByText('真实主密钥').first().waitFor();
  assert(await page.getByRole('checkbox', { name: '启用真实主密钥' }).isChecked(), '有效密钥的启用开关未选中');
  await page.getByRole('button', { name: '复制' }).first().click();
  await page.getByText('密钥已复制').waitFor();
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
  const merchantSidebar = page.getByRole('complementary', { name: '商户功能' });
  for (const label of ['经营概览', '渠道管理', '模型报价', '健康状态', '结算账本', '提现记录']) {
    assert(await merchantSidebar.getByRole('button', { name: label, exact: true }).count() === 1, `商户侧栏缺少${label}`);
  }
  assert(!(await page.locator('body').innerText()).includes('管理员账号当前未绑定商户资料'), '管理员仍显示专用商户缺失提示');
  await merchantSidebar.getByRole('button', { name: '渠道管理', exact: true }).click();
  await page.getByText('OpenAI 主线路').last().waitFor();
  await page.getByRole('heading', { name: '渠道管理' }).waitFor();
  await page.getByText('51', { exact: true }).waitFor();
  await page.getByText('已审核', { exact: true }).waitFor();
  await page.getByRole('button', { name: '提交新渠道' }).click();
  const channelDialog = page.getByRole('dialog', { name: '提交新渠道' });
  await channelDialog.getByLabel('渠道名称').fill('企业多模型线路');
  await channelDialog.getByLabel('上游基础地址').waitFor();
  assert(await channelDialog.getByLabel('上游基础地址').inputValue() === 'https://api.lwylink.xyz', '渠道弹窗没有读取本地默认上游地址');
  const channelPageURL = page.url();
  await channelDialog.getByRole('button', { name: '一键获取模型' }).click();
  await page.waitForTimeout(200);
  assert(adminMockState.channelModelSyncCalls === 1, '一键获取模型没有调用接口');
  const channelDialogVisible = await channelDialog.isVisible().catch(() => false);
  const channelDialogText = channelDialogVisible ? await channelDialog.innerText() : await page.locator('body').innerText();
  assert(channelDialogVisible && page.url() === channelPageURL, `同步上游模型后离开了渠道弹窗：${page.url()} ${pageErrors.join('; ')} ${channelDialogText}`);
  assert(channelDialogText.includes('已匹配 1 个模型') && channelDialogText.includes('平台未收录'), `同步结果没有显示在渠道弹窗：${channelDialogText}`);
  assert(adminMockState.channelModelSyncCalls === 1 && adminMockState.channelModelSyncBody.channel_type === 'openai_compatible', '同步模型没有提交密钥类型');
  await channelDialog.getByLabel('上游 API 密钥').fill('sk-local-test-key');
  await channelDialog.getByRole('button', { name: '提交渠道审核' }).click();
  await page.getByText('渠道已提交审核').waitFor();
  assert(adminMockState.channelSubmitCalls === 1 && adminMockState.channelSubmitBody.channel_name === '企业多模型线路', '渠道审核没有提交真实表单');
  await merchantSidebar.getByRole('button', { name: '模型报价', exact: true }).click();
  for (const label of ['调用次数', 'Token 用量', '最后调用']) {
    assert(await page.getByRole('columnheader', { name: label, exact: true }).count() === 1, `报价表缺少${label}列名`);
  }
  await page.getByText('0.72x').waitFor();
  await page.getByText('32.4K', { exact: true }).waitFor();
  await page.getByText('2.6B', { exact: true }).waitFor();
  await page.getByText('10分钟前', { exact: true }).waitFor();
  assert(await page.getByText('当前价格', { exact: true }).count() === 0, '模型报价仍显示当前价格');
  await merchantSidebar.getByRole('button', { name: '提现记录', exact: true }).click();
  await page.getByText('MW202607220001', { exact: true }).waitFor();
  await page.getByRole('button', { name: '申请提现' }).click();
  const withdrawalDialog = page.getByRole('dialog', { name: '申请人工提现' });
  await withdrawalDialog.getByLabel('提现金额（人民币）').fill('5');
  await withdrawalDialog.getByLabel('收款方式').selectOption('alipay');
  await withdrawalDialog.getByLabel('收款账户').fill('13800132621');
  await withdrawalDialog.getByRole('button', { name: '提交提现申请' }).click();
  assert(adminMockState.withdrawalCreateCalls === 1, '商户提现没有调用真实接口');
  assert(adminMockState.withdrawalCreateBody.amount_quota === 2500000, '提现人民币金额没有转换为平台整数额度');
  await merchantSidebar.getByRole('button', { name: '模型报价', exact: true }).click();
  await page.getByRole('button', { name: '设置报价' }).click();
  const quoteDialog = page.getByRole('dialog', { name: '设置模型报价' });
  assert(await quoteDialog.getByLabel('模型厂商').inputValue() === 'OpenAI', '模型报价没有按厂商分组选择');
  await quoteDialog.getByLabel('模型型号').selectOption('101');
  assert(await quoteDialog.getByLabel('商户渠道').count() === 0, '模型报价仍要求选择单个渠道');
  await quoteDialog.getByLabel('报价倍率').fill('0.65');
  await quoteDialog.getByRole('button', { name: '保存报价' }).click();
  assert(adminMockState.offerCreateCalls === 1, '创建报价没有调用真实接口');
  assert(adminMockState.offerCreateBody.multiplier_bps === 6500, '报价倍率没有转换为整数基点');
  page.once('dialog', (dialog) => dialog.accept());
  await page.getByRole('button', { name: '取消报价' }).first().click();
  assert(adminMockState.offerRetireCalls === 1, '取消报价没有调用真实接口');

  await page.getByRole('button', { name: '管理后台', exact: true }).click();
  await page.getByText('¥18.00', { exact: true }).first().waitFor();
  assert(await page.getByRole('button', { name: '审计日志', exact: true }).count() === 0, '管理后台仍显示操作记录入口');
  assert(await page.getByRole('button', { name: '系统任务', exact: true }).count() === 0, '管理后台仍显示系统任务入口');
  await page.getByText('126 / 128').waitFor();
  assert(await page.getByRole('button', { name: /商户审核/ }).getByText('1', { exact: true }).count() === 1, '管理员侧栏未显示真实商户待办数量');
  await page.getByRole('button', { name: '公告管理' }).click();
  await page.getByText('商户结算安排').waitFor();
  await page.getByRole('button', { name: '新建公告' }).click();
  const announcementDialog = page.getByRole('dialog', { name: '新建公告' });
  await announcementDialog.getByLabel('公告标题').fill('系统维护通知');
  await announcementDialog.getByLabel('公告正文').fill('今晚进行短时维护。');
  await announcementDialog.getByRole('checkbox', { name: '普通用户' }).check();
  await announcementDialog.getByRole('button', { name: '保存草稿' }).click();
  await page.getByText('公告草稿已保存').waitFor();
  assert(adminMockState.announcementCreateCalls === 1 && adminMockState.announcementCreateBody.audiences.includes('user'), '公告创建没有提交受众');
  await page.getByRole('button', { name: '渠道调用日志' }).click();
  await page.getByText('req-admin-9001').waitFor();
  await page.getByText('alice', { exact: true }).waitFor();
  assert(await page.getByText('渠道 41', { exact: true }).count() === 0, '渠道日志仍显示内部渠道编号');
  await page.getByRole('button', { name: '模型与定价' }).click();
  await page.getByText('v2', { exact: true }).waitFor();
  await page.getByRole('button', { name: '新增标准模型' }).click();
  const modelDialog = page.getByRole('dialog', { name: '新增标准模型' });
  await modelDialog.getByLabel('模型名称').fill('gpt-custom-chat');
  await modelDialog.getByLabel('模型厂商').selectOption('OpenAI');
  await modelDialog.getByLabel('协议').selectOption('chat_completions');
  assert(await modelDialog.getByLabel('生命周期').count() === 0, '模型表单仍显示生命周期');
  assert(await modelDialog.getByLabel('核价来源').count() === 0, '模型表单仍显示核价来源');
  await modelDialog.getByLabel('检测基础地址').fill('https://api.custom.example/v1');
  await modelDialog.getByLabel('检测 API 密钥').fill('sk-custom-secret');
  await modelDialog.getByRole('button', { name: '发送 hi 检测' }).click();
  await modelDialog.getByText('检测通过').waitFor();
  await modelDialog.getByLabel('第 1 档输入价格').fill('0.2');
  await modelDialog.getByLabel('第 1 档输出价格').fill('0.8');
  await modelDialog.getByRole('button', { name: '创建模型' }).click();
  assert(adminMockState.modelDetectCalls === 1, '新增自定义模型前没有执行临时 hi 检测');
  assert(adminMockState.modelCreateCalls === 1, '新增标准模型没有调用真实接口');
  assert(adminMockState.modelCreateBody.provider === 'OpenAI' && !('lifecycle' in adminMockState.modelCreateBody), '模型创建仍提交生命周期');
  assert(!('currency' in adminMockState.modelCreateBody.price) && !('usd_exchange_rate_micros' in adminMockState.modelCreateBody.price), '模型价格仍提交币种或汇率');
  assert(adminMockState.modelCreateBody.price.tiers.length === 1 && adminMockState.modelCreateBody.price.tiers[0].input_price_micros === 200000 && adminMockState.modelCreateBody.price.tiers[0].output_price_micros === 800000, '文本价格没有按整数微单位提交');
  await page.getByRole('button', { name: '发布价格' }).first().click();
  const priceDialog = page.getByRole('dialog', { name: '发布模型价格' });
  await priceDialog.getByRole('button', { name: '新增价格阶梯' }).click();
  await priceDialog.getByLabel('第 1 档上下文上限').fill('200000');
  await priceDialog.getByLabel('第 2 档输入价格').fill('0.3');
  await priceDialog.getByLabel('第 2 档输出价格').fill('1.2');
  assert(await priceDialog.getByRole('button', { name: '删除第 2 档' }).count() === 1, '价格阶梯缺少删除操作');
  await priceDialog.getByRole('button', { name: '发布价格' }).click();
  assert(adminMockState.pricePublishCalls === 1, '发布官方价格没有调用真实接口');
  assert(adminMockState.pricePublishBody.tiers.length === 2 && adminMockState.pricePublishBody.tiers[0].maximum_input_tokens === 200000 && adminMockState.pricePublishBody.tiers[1].minimum_input_tokens === 200001, '价格阶梯没有按连续区间提交');
  page.once('dialog', (dialog) => dialog.accept());
  await page.getByRole('button', { name: '删除', exact: true }).first().click();
  await page.getByText('标准模型已归档下架').waitFor();
  assert(adminMockState.modelDeleteCalls === 1, '模型删除或归档接口没有调用');
  await page.getByRole('button', { name: '用户管理' }).click();
  await page.getByText('alice@example.com').waitFor();
  await page.getByRole('button', { name: '商户审核' }).click();
  await page.getByText('东海模型服务').waitFor();
  await page.getByRole('button', { name: '验证' }).click();
  await page.getByText('渠道验证通过').waitFor();
  assert(adminMockState.merchantVerifyCalls === 1, '管理员验证没有调用真实商户审核接口');
  await page.getByRole('button', { name: '渠道与探活' }).click();
  await page.getByText('Anthropic 备用线路').waitFor();
  await page.getByText('51', { exact: true }).waitFor();
  await page.getByText('已下架').waitFor();
  assert(await page.getByRole('heading', { name: '已导入渠道' }).count() === 0, '渠道管理仍拆成独立的已导入渠道表');
  assert(!(await page.locator('body').innerText()).includes('¥12.50'), '渠道管理仍显示上游运行余额');
  assert(await page.getByRole('button', { name: '测试渠道' }).count() === 0, '渠道管理仍显示行内测试按钮');
  assert(await page.getByRole('button', { name: '刷新余额' }).count() === 0, '渠道管理仍显示刷新余额按钮');
  await page.getByRole('button', { name: '恢复渠道' }).click();
  await page.getByText('渠道已进入恢复检测').waitFor();
  assert(adminMockState.healthResetCalls === 1, '恢复渠道没有调用真实健康接口');
  await page.getByRole('button', { name: '验证' }).click();
  await page.getByText('渠道验证完成').waitFor();
  assert(adminMockState.channelVerifyCalls === 1, '管理员渠道页验证没有调用真实渠道审核接口');
  await page.getByRole('button', { name: '平台设置' }).click();
  await page.getByRole('heading', { name: '邮件与注册' }).waitFor();
  assert(await page.getByLabel('SMTP 服务器').inputValue() === 'smtp.example.com', '平台设置未加载 SMTP 配置');
  await page.getByRole('button', { name: '保存邮件与注册' }).click();
  await page.getByText('邮件与注册设置已保存').waitFor();
  assert(adminMockState.platformSettingCalls === 1 && adminMockState.platformSettingBody.smtp.password === '', '邮件注册设置没有按只写密码规则提交');
  assert(await page.getByRole('button', { name: '分组倍率', exact: true }).count() === 0, '管理员侧栏仍显示分组倍率');
  await page.getByRole('button', { name: '兑换码管理' }).click();
  await page.getByText('MR-JULY-2026').waitFor();
  await page.getByRole('button', { name: '新建兑换码' }).click();
  const redemptionDialog = page.getByRole('dialog', { name: '新建兑换码' });
  await redemptionDialog.getByLabel('名称').fill('批量活动码');
  await redemptionDialog.getByLabel('单码额度（人民币）').fill('2');
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
  await editRedemptionDialog.getByLabel('单码额度（人民币）').fill('1.4');
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
  await paymentDialog.getByLabel('最低充值金额（人民币）').fill('2');
  await paymentDialog.getByRole('button', { name: '保存支付配置' }).click();
  assert(adminMockState.paymentSettingCalls === 1, '支付配置没有调用真实接口');
  assert(adminMockState.paymentSettingBody.minimum_cny_cents === 200 && !('exchange_rate_cny_cents_per_usd' in adminMockState.paymentSettingBody), '人民币充值范围提交不正确');
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

  const merchantPage = await browser.newPage({ viewport: { width: 390, height: 844 } });
  await mockAdminAPI(merchantPage, { announcementReadCalls: 0 }, {
    id: 8, username: 'merchant', display_name: '普通商户', role: 1, status: 1, group: 'default',
    quota: 500000, used_quota: 120000, is_merchant: true, merchant_id: 31, merchant_status: 'active',
  });
  await merchantPage.goto(baseUrl, { waitUntil: 'networkidle' });
  assert(await merchantPage.getByRole('button', { name: '管理后台', exact: true }).count() === 0, '普通商户显示了管理后台入口');
  await merchantPage.getByRole('button', { name: '打开菜单' }).click();
  await merchantPage.getByRole('button', { name: '商户中心', exact: true }).click();
  const ordinaryMerchantSidebar = merchantPage.getByRole('complementary', { name: '商户功能' });
  for (const label of ['经营概览', '渠道管理', '模型报价', '健康状态', '结算账本', '提现记录']) {
    assert(await ordinaryMerchantSidebar.getByRole('button', { name: label, exact: true }).count() === 1, `普通商户侧栏缺少${label}`);
  }
  await ordinaryMerchantSidebar.getByRole('button', { name: '模型报价', exact: true }).click();
  await merchantPage.getByText('2.6B', { exact: true }).waitFor();
  await merchantPage.getByText('10分钟前', { exact: true }).waitFor();
  await merchantPage.close();

  const anonymousPage = await browser.newPage({ viewport: { width: 390, height: 844 } });
  let anonymousLoggedIn = false;
  let merchantApplication = null;
  let merchantApplicationCalls = 0;
  let paymentOrderCalls = 0;
  let paymentOrderBody = null;
  await anonymousPage.route('**/api/**', (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const fulfill = (data, status = 200) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(data) });
    if (url.pathname === '/api/me' && anonymousLoggedIn) return fulfill({ success: true, data: { id: 7, username: 'alice', display_name: 'Alice', role: 1, status: 1, group: 'default', quota: 500000, used_quota: 120000, is_merchant: false, merchant_status: 'none' } });
    if (url.pathname === '/api/me') return fulfill({ success: false, error: { code: 'authentication_required', message: '请先登录' } }, 401);
    if (url.pathname === '/api/models') return fulfill({ success: true, data: [{ id: 101, public_name: 'gpt-4o-mini', provider: 'OpenAI', protocol: 'chat_completions', status: 'active', price: { version: 2, price_kind: 'text', tiers: [{ minimum_input_tokens: 0, maximum_input_tokens: null, input_price_micros: 150000, output_price_micros: 600000 }] } }] });
    if (url.pathname === '/api/offers') return fulfill({ success: true, data: [{ offer_id: 201, merchant_id: 31, merchant_name: '北辰云智', model: 'gpt-4o-mini', protocol: 'chat_completions', multiplier_bps: 7200, price_tiers: [{ minimum_input_tokens: 0, maximum_input_tokens: null, input_price_micros: 108000, output_price_micros: 432000 }], channel_count: 1, health_state: 'healthy', availability_ppm: 999800, ttft_p95_ms: 1200 }] });
    if (url.pathname === '/api/auth/login' && request.method() === 'POST') {
      anonymousLoggedIn = true;
      return fulfill({ success: true, data: { id: 7, username: 'alice', display_name: 'Alice', role: 1, status: 1, group: 'default' } });
    }
    if (url.pathname === '/api/keys') return fulfill({ success: true, data: { page: 1, page_size: 20, total: 0, items: [] } });
    if (url.pathname === '/api/payment-settings') return fulfill({ success: true, data: { version: 3, payment_methods: ['alipay', 'wxpay'], minimum_cny_cents: 100, maximum_cny_cents: 2000000, enabled: true } });
    if (url.pathname === '/api/payment-orders' && request.method() === 'GET') return fulfill({ success: true, data: { page: 1, page_size: 20, total: 0, items: [] } });
    if (url.pathname === '/api/payment-orders' && request.method() === 'POST') {
      paymentOrderCalls += 1;
      paymentOrderBody = request.postDataJSON();
      return fulfill({ success: true, data: { order: { id: 31, order_no: 'MR-USER-1', amount_cny_cents: 10000, newapi_quota: 50000000, payment_status: 'pending', credit_status: 'pending' }, payment_url: `${baseUrl}/payment-target` } }, 201);
    }
    if (url.pathname === '/api/merchant/channel-tests' && request.method() === 'POST') return fulfill({ success: true, data: { id: 92, canonical_model_id: 101, adapter_type: 'openai_compatible', capabilities: ['chat_completions'], latency_ms: 305, test_receipt: 'receipt-for-merchant-apply-12345', expires_at: '2026-07-22T16:15:00Z' } }, 201);
    if (url.pathname === '/api/merchant/application' && request.method() === 'GET') return fulfill({ success: true, data: merchantApplication });
    if (url.pathname === '/api/merchant/application' && request.method() === 'POST') {
      merchantApplicationCalls += 1;
      merchantApplication = { id: 11, user_id: 7, display_name: '北辰云智', website_url: 'https://merchant.example', business_description: '企业模型 API 服务', status: 'pending', initial_channel: { id: 21, channel_name: '主线路', canonical_model_id: 101, model: 'gpt-4o-mini', adapter_type: 'openai_compatible', base_url: 'https://api.merchant.example', status: 'pending', credential_configured: true } };
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
  await anonymousPage.getByRole('main').getByText('¥1.00').waitFor();
  assert(await anonymousPage.getByRole('button', { name: '管理后台', exact: true }).count() === 0, '普通用户显示了管理后台入口');
  assert(await anonymousPage.getByRole('button', { name: '商户中心', exact: true }).count() === 0, '普通用户显示了商户中心入口');
  const anonymousConsoleSidebar = anonymousPage.getByRole('complementary', { name: '控制台功能' });
  await anonymousConsoleSidebar.getByRole('button', { name: '账户设置', exact: true }).click();
  await anonymousPage.getByText('用户 ID 7 · 普通用户').waitFor();
  await anonymousPage.getByRole('button', { name: '申请成为商户' }).click();
  const merchantDialog = anonymousPage.getByRole('dialog', { name: '申请成为商户' });
  await merchantDialog.getByLabel('商户名称').fill('北辰云智');
  await merchantDialog.getByLabel('网站地址').fill('https://merchant.example');
  await merchantDialog.getByLabel('上游基础地址').fill('https://api.merchant.example/v1');
  await merchantDialog.getByLabel('上游 API 密钥').fill('sk-upstream-secret');
  await merchantDialog.getByLabel('业务说明').fill('企业模型 API 服务');
  await merchantDialog.getByRole('button', { name: '发送 hi 检测' }).click();
  await merchantDialog.getByText('检测通过').waitFor();
  await merchantDialog.getByRole('button', { name: '提交申请' }).click();
  await anonymousPage.getByText('商户申请已提交').waitFor();
  assert(merchantApplicationCalls === 1, '商户申请没有调用平台接口');
  await anonymousPage.getByText('等待管理员审核').waitFor();
  await anonymousConsoleSidebar.getByRole('button', { name: '充值中心', exact: true }).click();
  await anonymousPage.getByRole('button', { name: '立即充值' }).click();
  const rechargeDialog = anonymousPage.getByRole('dialog', { name: '账户充值' });
  await rechargeDialog.getByLabel('充值金额（人民币）').fill('100');
  await rechargeDialog.getByText('¥100.00').first().waitFor();
  await rechargeDialog.getByRole('button', { name: '前往支付' }).click();
  await anonymousPage.waitForURL('**/payment-target');
  assert(paymentOrderCalls === 1 && paymentOrderBody.amount_cny_cents === 10000 && !('usd_cents' in paymentOrderBody), '人民币充值订单参数不正确');
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
    if (url.pathname === '/api/payment-orders/MR-RETURN') return fulfill({ success: true, data: { id: 31, order_no: 'MR-RETURN', user_id: 7, amount_cny_cents: 10000, newapi_quota: 50000000, payment_status: 'paid', credit_status: 'pending', paid_at: '2026-07-22T10:00:00Z' } });
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

console.log('UI 冒烟测试通过：三套侧栏、多模型渠道、零至五倍报价、密钥路由、人民币支付、结算和人工提现交互正常。');
} finally {
  await browser.close();
  await stopLocalPreview(previewServer);
}
