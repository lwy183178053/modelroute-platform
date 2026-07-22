export const marketOffers = [
  {
    rank: 1,
    model: 'GPT-5.2',
    provider: 'OpenAI',
    merchant: '北辰云智',
    tag: '企业认证',
    availability: '99.98%',
    latency: '1.2s',
    calls: '128.4K',
    tokens: '8.2B',
    cache: '68.4%',
    multiplier: '0.72x',
    inputPrice: '¥1.26/M',
    outputPrice: '¥10.08/M',
    state: '健康',
  },
  {
    rank: 2,
    model: 'Claude Sonnet 4.5',
    provider: 'Anthropic',
    merchant: '青川 API',
    tag: '个人认证',
    availability: '99.95%',
    latency: '1.5s',
    calls: '96.7K',
    tokens: '6.9B',
    cache: '73.1%',
    multiplier: '0.68x',
    inputPrice: '¥2.04/M',
    outputPrice: '¥10.20/M',
    state: '健康',
  },
  {
    rank: 3,
    model: 'DeepSeek V3.2',
    provider: 'DeepSeek',
    merchant: '矩阵算力',
    tag: '企业认证',
    availability: '99.89%',
    latency: '0.8s',
    calls: '84.2K',
    tokens: '12.6B',
    cache: '58.9%',
    multiplier: '0.55x',
    inputPrice: '¥0.28/M',
    outputPrice: '¥0.84/M',
    state: '健康',
  },
  {
    rank: 4,
    model: 'Gemini 3 Pro',
    provider: 'Gemini',
    merchant: '星环中转',
    tag: '企业认证',
    availability: '98.76%',
    latency: '2.1s',
    calls: '63.1K',
    tokens: '4.7B',
    cache: '64.2%',
    multiplier: '0.61x',
    inputPrice: '¥1.22/M',
    outputPrice: '¥7.32/M',
    state: '降级',
  },
  {
    rank: 5,
    model: 'Qwen3-Max',
    provider: 'Qwen',
    merchant: '远峰模型',
    tag: '个人认证',
    availability: '99.62%',
    latency: '1.0s',
    calls: '51.8K',
    tokens: '5.1B',
    cache: '61.7%',
    multiplier: '0.48x',
    inputPrice: '¥0.58/M',
    outputPrice: '¥2.30/M',
    state: '健康',
  },
];

export const merchantChannels = [
  { name: 'OpenAI 主线路', provider: 'OpenAI', channel: '#1028', models: 8, availability: '99.98%', latency: '1.2s', checked: '38 秒前', state: '健康' },
  { name: 'Claude 企业线路', provider: 'Anthropic', channel: '#1041', models: 5, availability: '99.95%', latency: '1.5s', checked: '1 分钟前', state: '健康' },
  { name: 'Gemini 备用线路', provider: 'Gemini', channel: '#1063', models: 4, availability: '97.82%', latency: '2.8s', checked: '冷却至 10:26', state: '冷却' },
];

export const merchantQuotes = [
  { model: 'GPT-5.2', channel: 'OpenAI 主线路', official: '¥1.75 / ¥14.00', multiplier: '0.72x', final: '¥1.26 / ¥10.08', calls: '128.4K', revenue: '¥8,642.30', state: '在售' },
  { model: 'Claude Sonnet 4.5', channel: 'Claude 企业线路', official: '¥3.00 / ¥15.00', multiplier: '0.68x', final: '¥2.04 / ¥10.20', calls: '96.7K', revenue: '¥6,281.46', state: '在售' },
  { model: 'Gemini 3 Pro', channel: 'Gemini 备用线路', official: '¥2.00 / ¥12.00', multiplier: '0.61x', final: '¥1.22 / ¥7.32', calls: '63.1K', revenue: '¥2,106.88', state: '冷却' },
];

export const paymentOrders = [
  { id: 'CZ202607220184', user: 'u_10826', amount: '¥500.00', method: '支付宝', trade: '20260722••6819', paid: '09:36:18', state: '已完成', credit: '已入账' },
  { id: 'CZ202607220179', user: 'u_10593', amount: '¥200.00', method: '微信', trade: '20260722••4721', paid: '09:22:07', state: '已支付', credit: '入账异常' },
  { id: 'CZ202607220171', user: 'u_10288', amount: '¥1,000.00', method: '支付宝', trade: '20260722••0936', paid: '08:54:42', state: '已完成', credit: '已入账' },
  { id: 'CZ202607220168', user: 'u_10931', amount: '¥50.00', method: 'QQ 钱包', trade: '-', paid: '-', state: '待支付', credit: '未入账' },
];

export const recentCalls = [
  { id: 'req_a8923f', model: 'GPT-5.2', merchant: '北辰云智', tokens: '42,816', cache: '命中 68%', cost: '¥0.2846', duration: '1.34s', time: '刚刚', state: '成功' },
  { id: 'req_73b1dc', model: 'Claude Sonnet 4.5', merchant: '青川 API', tokens: '18,204', cache: '未命中', cost: '¥0.1972', duration: '2.08s', time: '2 分钟前', state: '成功' },
  { id: 'req_19a0fe', model: 'Gemini 3 Pro', merchant: '星环中转', tokens: '-', cache: '-', cost: '¥0.0000', duration: '3.00s', time: '6 分钟前', state: '已退款' },
];

export const merchantReviews = [
  { id: 'SQ-0382', applicant: '蓝海算力', website: 'api.bluecompute.cn', provider: 'OpenAI / Claude', submitted: '18 分钟前', state: '待验证' },
  { id: 'SQ-0381', applicant: '远山模型', website: 'model.farhill.ai', provider: 'Gemini', submitted: '42 分钟前', state: '待审核' },
  { id: 'SQ-0378', applicant: '云上接口', website: 'api.cloudtop.cn', provider: 'DeepSeek', submitted: '2 小时前', state: '验证失败' },
];

export const withdrawals = [
  { id: 'TX-20260722-016', merchant: '矩阵算力', amount: '¥8,000.00', available: '¥18,642.30', account: '支付宝 · 138****2621', submitted: '09:12', state: '待审核' },
  { id: 'TX-20260722-015', merchant: '青川 API', amount: '¥5,000.00', available: '¥12,306.42', account: '银行卡 · 6217 **** 1098', submitted: '08:34', state: '待打款' },
];
