import { useCallback, useEffect, useMemo, useState } from 'react';
import { Check, Power, RefreshCw, Search, ShieldCheck, Trash2, X } from 'lucide-react';
import { apiRequest } from '../../api';
import { Button, InlineNotice, Section, Select, Status, TextInput } from '../../ui';

const reviewStatuses = new Set(['pending', 'verifying', 'verified', 'approving']);
const healthNames = {
  pending: '待调用', healthy: '健康', degraded: '降级', cooling: '冷却', recovering: '恢复中', disabled: '已停用',
};

function displayStatus(item) {
  if (item.kind === 'review') return item.status === 'rejected' ? '不通过' : reviewStatuses.has(item.status) ? '审核中' : '已审核';
  return item.status === 'active' ? '已审核' : '已下架';
}

function lifecycleRows(reviews, channels) {
  const importedReviews = new Set(channels.map((channel) => Number(channel.review_id || 0)).filter(Boolean));
  const pendingRows = reviews
    .filter((review) => !importedReviews.has(Number(review.id)) && review.status !== 'approved')
    .map((review) => ({ ...review, kind: 'review' }));
  return [...channels.map((channel) => ({ ...channel, kind: 'channel' })), ...pendingRows]
    .sort((left, right) => new Date(right.created_at || 0) - new Date(left.created_at || 0));
}

export default function AdminChannels({ notify }) {
  const [reviews, setReviews] = useState([]);
  const [channels, setChannels] = useState([]);
  const [error, setError] = useState('');
  const [busyID, setBusyID] = useState('');
  const [query, setQuery] = useState('');
  const [merchantFilter, setMerchantFilter] = useState('all');

  const load = useCallback(async () => {
    setError('');
    try {
      const [reviewData, channelData] = await Promise.all([
        apiRequest('/api/admin/channel-reviews?p=1&page_size=100'),
        apiRequest('/api/admin/channels?p=1&page_size=100'),
      ]);
      setReviews(reviewData?.items || []);
      setChannels(channelData?.items || []);
    } catch (requestError) {
      setError(requestError.message || '读取渠道数据失败');
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const operate = async (review, action) => {
    let body;
    if (action === 'reject') {
      const reason = window.prompt('请输入驳回原因');
      if (!reason) return;
      body = { reason };
    }
    setBusyID(`review-${review.id}-${action}`);
    try {
      await apiRequest(`/api/admin/channel-reviews/${review.id}/${action}`, { method: 'POST', body });
      notify(action === 'verify' ? '渠道验证完成' : action === 'approve' ? '渠道已审核通过' : '渠道审核不通过');
      await load();
    } catch (requestError) {
      notify(requestError.message || '渠道审核操作失败');
    } finally {
      setBusyID('');
    }
  };

  const updateChannel = async (channel, action) => {
    setBusyID(`channel-${channel.id}-${action}`);
    try {
      if (action === 'delete') await apiRequest(`/api/admin/channels/${channel.id}`, { method: 'DELETE' });
      else await apiRequest(`/api/admin/channels/${channel.id}/status`, { method: 'POST', body: { enabled: channel.status !== 'active' } });
      notify(action === 'delete' || channel.status === 'active' ? '渠道已下架' : '渠道已启用');
      await load();
    } catch (requestError) {
      notify(requestError.message || '渠道状态操作失败');
    } finally {
      setBusyID('');
    }
  };

  const recover = async (channel) => {
    setBusyID(`channel-${channel.id}-recover`);
    try {
      await apiRequest(`/api/admin/channels/${channel.id}/health/reset`, { method: 'POST' });
      notify('渠道已进入恢复检测');
      await load();
    } catch (requestError) {
      notify(requestError.message || '恢复渠道失败');
    } finally {
      setBusyID('');
    }
  };

  const testModel = async (review, binding) => {
    setBusyID(`model-${binding.id}`);
    try {
      await apiRequest(`/api/admin/channel-reviews/${review.id}/model-tests`, { method: 'POST', body: { binding_id: binding.id } });
      notify('模型 hi 检查完成');
      await load();
    } catch (requestError) {
      notify(requestError.message || '模型检查失败');
    } finally {
      setBusyID('');
    }
  };

  const rows = useMemo(() => lifecycleRows(reviews, channels), [reviews, channels]);
  const merchants = useMemo(() => {
    const values = new Map();
    rows.forEach((item) => values.set(String(item.user_id), item.merchant_name || `用户 ${item.user_id}`));
    return [...values.entries()].filter(([id]) => id !== 'undefined').sort((left, right) => Number(left[0]) - Number(right[0]));
  }, [rows]);
  const visibleRows = rows.filter((item) => {
    const text = `${item.id}${item.user_id}${item.merchant_name || ''}${item.channel_name}${item.model || ''}${item.base_url || ''}`.toLowerCase();
    return text.includes(query.trim().toLowerCase()) && (merchantFilter === 'all' || String(item.user_id) === merchantFilter);
  });

  return (
    <div>
      {error && <InlineNotice tone="warning">{error}</InlineNotice>}
      <Section title="渠道管理" description="审核记录和已审核渠道按同一生命周期展示" actions={<Button icon={RefreshCw} onClick={load}>刷新</Button>}>
        <div className="filter-toolbar">
          <label className="search-box"><Search size={17} /><TextInput aria-label="搜索渠道或商户" placeholder="搜索 ID、渠道、模型或商户" value={query} onChange={(event) => setQuery(event.target.value)} /></label>
          <Select aria-label="筛选商户" value={merchantFilter} onChange={(event) => setMerchantFilter(event.target.value)}><option value="all">全部商户</option>{merchants.map(([id, name]) => <option key={id} value={id}>{name} · 用户 ID {id}</option>)}</Select>
        </div>
        <div className="table-wrap"><table className="data-table"><thead><tr><th>ID</th><th>商户</th><th>渠道</th><th>模型</th><th>基础地址</th><th>状态</th><th>健康</th><th>操作</th></tr></thead><tbody>
          {visibleRows.map((item) => {
            const canRecover = item.kind === 'channel' && (item.status === 'disabled' || ['disabled', 'cooling'].includes(item.health_state));
            return <tr key={`${item.kind}-${item.id}`}>
              <td><code>{item.id}</code><small>{item.created_at ? new Date(item.created_at).toLocaleString('zh-CN') : '-'}</small></td>
              <td><strong>{item.merchant_name || `用户 ${item.user_id}`}</strong><small>用户 ID {item.user_id}</small></td>
              <td><strong>{item.channel_name}</strong><small>{item.adapter_type || '-'}</small></td>
              <td>{item.kind === 'review' && item.models?.length ? item.models.map((binding) => <div key={binding.id} className="model-review-row"><span>{binding.upstream_model_name}</span>{binding.match_status === 'matched' && <Button disabled={Boolean(busyID)} variant="ghost" onClick={() => testModel(item, binding)}>hi 检查</Button>}</div>) : (item.matched_models?.length ? item.matched_models.map((model) => <span className="tag" key={model}>{model}</span>) : item.model || '-')}</td>
              <td><code>{item.base_url}</code></td>
              <td><Status>{displayStatus(item)}</Status><small>{item.verification_error || item.review_reason || ''}</small></td>
              <td>{item.kind === 'channel' ? <Status>{healthNames[item.health_state] || item.health_state || '待调用'}</Status> : '-'}</td>
              <td>{item.kind === 'review' ? <div className="row-actions"><Button disabled={Boolean(busyID) || !['pending', 'verified'].includes(item.status)} icon={ShieldCheck} onClick={() => operate(item, 'verify')} variant="ghost">验证</Button><Button disabled={Boolean(busyID) || item.status !== 'verified'} icon={Check} onClick={() => operate(item, 'approve')} variant="ghost">通过</Button><Button disabled={Boolean(busyID) || !['pending', 'verified'].includes(item.status)} icon={X} onClick={() => operate(item, 'reject')} variant="danger">驳回</Button></div> : <div className="row-actions"><Button disabled={Boolean(busyID) || item.status === 'archived'} icon={Power} onClick={() => updateChannel(item, 'status')} variant="ghost">{item.status === 'active' ? '下架' : '启用'}</Button><Button disabled={Boolean(busyID)} icon={Trash2} onClick={() => updateChannel(item, 'delete')} variant="danger">删除</Button>{canRecover && <Button disabled={Boolean(busyID)} icon={RefreshCw} variant="ghost" onClick={() => recover(item)}>恢复渠道</Button>}</div>}</td>
            </tr>;
          })}
          {visibleRows.length === 0 && <tr><td className="table-message" colSpan="8">暂无匹配渠道</td></tr>}
        </tbody></table></div>
      </Section>
    </div>
  );
}
