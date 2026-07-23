export class APIError extends Error {
  constructor(message, status, code) {
    super(message);
    this.name = 'APIError';
    this.status = status;
    this.code = code;
  }
}

export async function apiRequest(path, { method = 'GET', body, signal } = {}) {
  const headers = { Accept: 'application/json' };
  if (body !== undefined) headers['Content-Type'] = 'application/json';

  const response = await fetch(path, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
    credentials: 'same-origin',
    signal,
  });

  let payload;
  try {
    payload = await response.json();
  } catch {
    throw new APIError('服务返回了无效响应', response.status, 'invalid_response');
  }

  if (!response.ok || payload?.success === false) {
    throw new APIError(
      payload?.error?.message || '请求处理失败',
      response.status,
      payload?.error?.code || 'request_failed',
    );
  }
  return payload?.data;
}

export function formatQuota(value) {
  const quota = Number(value || 0);
  return new Intl.NumberFormat('zh-CN', {
    style: 'currency', currency: 'CNY', minimumFractionDigits: 2, maximumFractionDigits: 2,
  }).format(quota / 500_000);
}

export function formatPriceMicros(value) {
  if (value == null) return '-';
  return new Intl.NumberFormat('zh-CN', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 6,
  }).format(Number(value) / 1_000_000);
}

export function formatOfferPrice(tiers) {
  const tier = tiers?.[0];
  if (!tier) return '-';
  const input = formatPriceMicros(tier.input_price_micros);
  const output = tier.output_price_micros == null ? '' : ` / ${formatPriceMicros(tier.output_price_micros)} 输出`;
  return `${input} 输入${output}`;
}

export function formatImagePrice(items) {
  if (!items?.length) return '-';
  const minimum = Math.min(...items.map((item) => Number(item.price_micros || 0)));
  return minimum === 0 ? '免费' : `${formatPriceMicros(minimum)} / 张起`;
}
