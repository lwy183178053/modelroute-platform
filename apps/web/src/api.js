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

let quotaPerUnit = 0;

export function configureQuotaPerUnit(value) {
  const unit = Number(value);
  if (!Number.isSafeInteger(unit) || unit <= 0 || unit % 100 !== 0) {
    throw new Error('核心额度单位配置无效');
  }
  quotaPerUnit = unit;
}

export function getQuotaPerUnit() {
  if (!quotaPerUnit) throw new Error('核心额度单位尚未加载');
  return quotaPerUnit;
}

export function formatQuota(value) {
  const quota = Number(value || 0);
  return new Intl.NumberFormat('zh-CN', {
    style: 'currency', currency: 'CNY', minimumFractionDigits: 2, maximumFractionDigits: 2,
  }).format(quota / getQuotaPerUnit());
}

export function formatPriceMicros(value) {
  if (value == null) return '-';
  return new Intl.NumberFormat('zh-CN', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 6,
  }).format(Number(value) / 1_000_000);
}

export function formatImagePrice(items) {
  if (!items?.length) return '-';
  const minimum = Math.min(...items.map((item) => Number(item.price_micros || 0)));
  return minimum === 0 ? '免费' : `${formatPriceMicros(minimum)} / 张起`;
}

export function formatCompactCount(value) {
  const count = Math.max(0, Number(value || 0));
  const units = [[1_000_000_000, 'B'], [1_000_000, 'M'], [1_000, 'K']];
  for (const [threshold, suffix] of units) {
    if (count >= threshold) {
      const compact = count / threshold;
      return `${compact >= 100 ? Math.round(compact) : compact.toFixed(1).replace(/\.0$/, '')}${suffix}`;
    }
  }
  return String(Math.round(count));
}

export function formatLastCalled(value) {
  if (!value) return '暂无调用';
  const timestamp = new Date(value).getTime();
  if (Number.isNaN(timestamp)) return '暂无调用';
  const minutes = Math.max(0, Math.floor((Date.now() - timestamp) / 60_000));
  if (minutes < 1) return '刚刚';
  if (minutes < 60) return `${minutes}分钟前`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}小时前`;
  return `${Math.floor(hours / 24)}天前`;
}
