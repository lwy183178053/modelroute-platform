import { useCallback, useEffect, useMemo, useState } from 'react';
import { Activity, Coins, Gauge, RefreshCw } from 'lucide-react';
import { apiRequest, formatQuota } from '../api';
import { Button, EmptyNotice, InlineNotice, Section, Status } from '../ui';

const numberFormatter = new Intl.NumberFormat('zh-CN');

function logTime(value) {
  if (!value) return '-';
  const date = typeof value === 'number' ? new Date(value * 1000) : new Date(value);
  if (Number.isNaN(date.getTime())) return '-';
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false,
  }).format(date);
}

function dayLabel(value) {
  const date = typeof value === 'number' ? new Date(value * 1000) : new Date(value);
  if (Number.isNaN(date.getTime())) return '-';
  return `${String(date.getMonth() + 1).padStart(2, '0')}/${String(date.getDate()).padStart(2, '0')}`;
}

function parsedOther(log) {
  if (log?.other && typeof log.other === 'object') return log.other;
  if (typeof log?.other !== 'string' || !log.other.trim()) return {};
  try {
    return JSON.parse(log.other);
  } catch {
    return {};
  }
}

function cacheTokens(log) {
  return Number(log?.cache_tokens ?? parsedOther(log).cache_tokens ?? 0);
}

function requestType(type) {
  if (Number(type) === 6) return '已退款';
  if (Number(type) === 2) return '已结算';
  return `类型 ${type}`;
}

function normalizeTrend(items) {
  const points = new Map();
  for (const item of Array.isArray(items) ? items : []) {
    const key = Number(item.created_at) || String(item.created_at || '');
    const current = points.get(key) || { created_at: item.created_at, quota: 0, count: 0 };
    current.quota += Number(item.quota || 0);
    current.count += Number(item.count || 0);
    points.set(key, current);
  }
  return [...points.values()].sort((left, right) => Number(left.created_at) - Number(right.created_at));
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
      setError(requestError.message || '读取用量数据失败');
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
    <Section title={title} description="最近 7 天最终结算额度" actions={actions}>
      {error && <InlineNotice tone="warning">{error}</InlineNotice>}
      {!error && !loading && points.length === 0 && <EmptyNotice title="暂无结算趋势" description="产生模型调用后将在这里显示。" />}
      {loading && <div className="table-message">正在读取用量趋势</div>}
      {points.length > 0 && <>
        <div className="usage-summary-row">
          <span><Coins size={15} />结算额度<strong>{formatQuota(stats.quota)}</strong></span>
          <span><Activity size={15} />当前 RPM<strong>{numberFormatter.format(Number(stats.rpm || 0))}</strong></span>
          <span><Gauge size={15} />当前 TPM<strong>{numberFormatter.format(Number(stats.tpm || 0))}</strong></span>
        </div>
        <div className="usage-chart" aria-label="最近 7 天用量趋势">
          {points.map((point) => {
            const height = Math.max(8, Math.round((Math.abs(point.quota) / maximum) * 100));
            return <div className="usage-bar-item" key={String(point.created_at)} title={`${dayLabel(point.created_at)} ${formatQuota(point.quota)}`}>
              <strong>{formatQuota(point.quota)}</strong>
              <span className="usage-bar-track"><i style={{ height: `${height}%` }} /></span>
              <small>{dayLabel(point.created_at)}</small>
            </div>;
          })}
        </div>
      </>}
    </Section>
  );
}

export function UsageLogTable({ data, loading, error, admin = false, title = '最近调用', onRefresh }) {
  const logs = data?.logs?.items || [];
  return (
    <Section
      title={title}
      description="金额、退款与缓存命中均来自最终消费日志"
      actions={onRefresh && <Button icon={RefreshCw} onClick={onRefresh}>刷新</Button>}
    >
      {error && <InlineNotice tone="warning">{error}</InlineNotice>}
      <div className="table-wrap"><table className="data-table usage-log-table"><thead><tr>
        <th>ID</th><th>时间</th>{admin && <th>用户</th>}<th>模型</th><th>密钥</th><th>Token</th><th>缓存</th><th>结算额度</th><th>请求编号</th><th>状态</th>
      </tr></thead><tbody>
        {logs.map((log) => <tr key={log.id || log.request_id}>
          <td><code>{log.id || '-'}</code></td>
          <td>{logTime(log.created_at)}</td>
          {admin && <td><strong>{log.username || `用户 ID ${log.user_id || '-'}`}</strong></td>}
          <td><strong>{log.model_name || '平台路由模型'}</strong></td>
          <td>{log.token_name || (log.token_id ? `密钥 ${log.token_id}` : '-')}</td>
          <td><strong>{numberFormatter.format(Number(log.prompt_tokens || 0) + Number(log.completion_tokens || 0))}</strong><small>输入 {numberFormatter.format(Number(log.prompt_tokens || 0))} · 输出 {numberFormatter.format(Number(log.completion_tokens || 0))}</small></td>
          <td>{cacheTokens(log) > 0 ? `缓存 ${numberFormatter.format(cacheTokens(log))} Token` : '-'}</td>
          <td>{formatQuota(Number(log.type) === 6 ? -Math.abs(Number(log.quota || 0)) : log.quota)}</td>
          <td><code>{log.request_id || '-'}</code></td>
          <td><Status>{requestType(log.type)}</Status></td>
        </tr>)}
        {!loading && logs.length === 0 && <tr><td className="table-message" colSpan={admin ? 10 : 9}>{error || '暂无调用记录'}</td></tr>}
        {loading && <tr><td className="table-message" colSpan={admin ? 10 : 9}>正在读取调用日志</td></tr>}
      </tbody></table></div>
    </Section>
  );
}
