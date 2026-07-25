import { useCallback, useEffect, useMemo, useState } from 'react';
import { Activity, Coins, Gauge, RefreshCw } from 'lucide-react';
import { apiRequest, formatQuota, formatUsageQuota } from '../api';
import { Button, EmptyNotice, InlineNotice, Pagination, Section } from '../ui';

const numberFormatter = new Intl.NumberFormat('zh-CN');

function logTime(value) {
  if (!value) return '-';
  const date = typeof value === 'number' ? new Date(value * 1000) : new Date(value);
  if (Number.isNaN(date.getTime())) return '-';
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false,
  }).format(date);
}

function firstTokenTime(value) {
  const milliseconds = Number(value || 0);
  if (!milliseconds) return '-';
  return `${(milliseconds / 1000).toFixed(2)}s`;
}

function dayLabel(value) {
  const date = typeof value === 'number' ? new Date(value * 1000) : new Date(value);
  if (Number.isNaN(date.getTime())) return '-';
  return `${String(date.getMonth() + 1).padStart(2, '0')}/${String(date.getDate()).padStart(2, '0')}`;
}

function dayKey(date) {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`;
}

function normalizeTrend(items) {
  const points = new Map();
  for (const item of Array.isArray(items) ? items : []) {
    const timestamp = typeof item.created_at === 'number' ? item.created_at * 1000 : Date.parse(item.created_at || '');
    if (!Number.isFinite(timestamp)) continue;
    const date = new Date(timestamp);
    const key = dayKey(date);
    const current = points.get(key) || { created_at: Math.floor(new Date(date.getFullYear(), date.getMonth(), date.getDate()).getTime() / 1000), quota: 0, count: 0 };
    current.quota += Number(item.quota || 0);
    current.count += Number(item.count || 0);
    points.set(key, current);
  }
  if (points.size === 0) return [];
  const latest = new Date(Math.max(...[...points.values()].map((item) => item.created_at * 1000)));
  latest.setHours(0, 0, 0, 0);
  return Array.from({ length: 7 }, (_, index) => {
    const date = new Date(latest);
    date.setDate(latest.getDate() - 6 + index);
    return points.get(dayKey(date)) || {
      created_at: Math.floor(date.getTime() / 1000), quota: 0, count: 0,
    };
  });
}

function tokenDetail(log) {
  return <span className="usage-token-detail"><span>输入 {numberFormatter.format(Number(log.prompt_tokens || 0))}</span><span>输出 {numberFormatter.format(Number(log.completion_tokens || 0))}</span></span>;
}

export function useUsage(endpoint) {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      setData(await apiRequest(endpoint));
    } catch (requestError) {
      setError(requestError.message || '读取调用数据失败');
    } finally {
      setLoading(false);
    }
  }, [endpoint]);

  useEffect(() => { load(); }, [load]);
  return { data, loading, error, load };
}

export function UsageTrend({ data, loading, error, title = '用量趋势', actions }) {
  const points = useMemo(() => normalizeTrend(data?.trend), [data]);
  const maximum = Math.max(...points.map((item) => Math.abs(item.quota)), 1);
  const stats = data?.stats || {};

  return (
    <Section title={title} description="最近 7 天调用额度" actions={actions}>
      {error && <InlineNotice tone="warning">{error}</InlineNotice>}
      {!error && !loading && points.length === 0 && <EmptyNotice title="暂无用量趋势" description="产生模型调用后将在这里显示。" />}
      {loading && <div className="usage-trend-loading" role="status">正在读取用量趋势</div>}
      {points.length > 0 && <>
        <div className="usage-summary-row">
          <span><Coins size={15} />消费额度<strong>{formatQuota(stats.quota)}</strong></span>
          <span><Activity size={15} />当前 RPM<strong>{numberFormatter.format(Number(stats.rpm || 0))}</strong></span>
          <span><Gauge size={15} />当前 TPM<strong>{numberFormatter.format(Number(stats.tpm || 0))}</strong></span>
        </div>
        <div className="usage-chart" aria-label="最近 7 天用量趋势">
          {points.map((point) => {
            const height = point.quota > 0 ? Math.max(8, Math.round((Math.abs(point.quota) / maximum) * 100)) : 0;
            return <div className="usage-bar-item" key={String(point.created_at)} title={`${dayLabel(point.created_at)} ${formatQuota(point.quota)}`}>
              <strong>{point.quota > 0 ? formatQuota(point.quota) : '—'}</strong>
              <span className="usage-bar-track"><i style={{ height: `${height}%` }} /></span>
              <small>{dayLabel(point.created_at)}</small>
            </div>;
          })}
        </div>
      </>}
    </Section>
  );
}

export function UsageLogTable({ data, loading, error, view = 'user', title = '调用记录', filters, onRefresh, page, pageSize, total, onPageChange, onPageSizeChange }) {
  const logs = data?.logs?.items || [];
  const pageInfo = data?.logs || {};
  const isAdmin = view === 'admin';
  const isMerchant = view === 'merchant';
  const columnCount = isAdmin ? 10 : isMerchant ? 8 : 9;
  return (
    <Section
      title={title}
      description="仅展示成功消费记录，明细保留全部历史"
      actions={onRefresh && <Button icon={RefreshCw} onClick={onRefresh}>刷新</Button>}
    >
      {filters}
      {error && <InlineNotice tone="warning">{error}</InlineNotice>}
      <div className="table-wrap"><table className="data-table usage-log-table"><thead><tr>
        <th className="usage-col-id">ID</th><th className="usage-col-time">时间</th><th className="usage-col-ttft">首 Token 时间</th>{isAdmin && <th className="usage-col-user">用户 ID</th>}<th className="usage-col-model">模型</th>{!isMerchant && <th className="usage-col-merchant">商户</th>}{(isAdmin || isMerchant) && <th className="usage-col-channel">渠道</th>}{!isAdmin && !isMerchant && <th className="usage-col-key">密钥</th>}<th className="usage-col-token">Token</th><th className="usage-col-cache">缓存 Token</th><th className="usage-col-amount">{isMerchant ? '自动入账' : '消费金额'}</th>
      </tr></thead><tbody>
        {logs.map((log) => <tr key={log.id || log.request_id}>
          <td className="usage-col-id"><code>{log.id || '-'}</code></td>
          <td className="usage-col-time">{logTime(log.created_at)}</td>
          <td className="usage-col-ttft">{firstTokenTime(log.first_token_ms)}</td>
          {isAdmin && <td className="usage-col-user"><code>{log.user_id || '-'}</code></td>}
          <td className="usage-col-model"><strong>{log.model_name || '平台路由模型'}</strong></td>
          {!isMerchant && <td className="usage-col-merchant"><strong>{log.merchant_name || '-'}</strong><small>用户 ID {log.merchant_user_id || '-'}</small></td>}
          {(isAdmin || isMerchant) && <td className="usage-col-channel"><strong>{log.channel_name || '-'}</strong><small>渠道 ID {log.channel_id || '-'}</small></td>}
          {!isAdmin && !isMerchant && <td className="usage-col-key">{log.token_id ? `密钥 ID ${log.token_id}` : '-'}</td>}
          <td className="usage-col-token">{tokenDetail(log)}</td>
          <td className="usage-col-cache">{numberFormatter.format(Number(log.cache_tokens || 0))}</td>
          <td className="usage-col-amount">{formatUsageQuota(isMerchant ? log.merchant_income_quota : log.quota)}</td>
        </tr>)}
        {!loading && logs.length === 0 && <tr><td className="table-message" colSpan={columnCount}>{error || '暂无调用记录'}</td></tr>}
        {loading && <tr><td className="table-message" colSpan={columnCount}>正在读取调用记录</td></tr>}
      </tbody></table></div>
      <Pagination
        page={page ?? pageInfo.page ?? 1}
        pageSize={pageSize ?? pageInfo.page_size ?? 20}
        total={total ?? pageInfo.total ?? logs.length}
        onPageChange={onPageChange}
        onPageSizeChange={onPageSizeChange}
        disabled={loading}
      />
    </Section>
  );
}
