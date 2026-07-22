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
