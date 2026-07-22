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
  return new Intl.NumberFormat('en-US', {
    style: 'currency', currency: 'USD', minimumFractionDigits: 2, maximumFractionDigits: 2,
  }).format(quota / 500_000);
}
