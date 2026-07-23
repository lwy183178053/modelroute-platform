import { useCallback, useEffect, useState } from 'react';
import { Check, RefreshCw, ShieldCheck, X, Power, Trash2, Search } from 'lucide-react';
import { apiRequest } from '../../api';
import { Button, InlineNotice, Section, Status, TextInput, Select } from '../../ui';

const statusNames = {
  pending: '待审核', verifying: '验证中', verified: '待批准', approving: '导入中',
  approved: '已通过', rejected: '已驳回', active: '正常', disabled: '已停用',
};

const healthNames = {
  pending: '待探测', healthy: '健康', degraded: '降级', cooling: '冷却', recovering: '恢复中', disabled: '已停用',
};

export default function AdminChannels({ notify }) {
  const [reviews, setReviews] = useState([]);
  const [channels, setChannels] = useState([]);
  const [offers, setOffers] = useState([]);
  const [runtimes, setRuntimes] = useState({});
  const [error, setError] = useState('');
  const [busyID, setBusyID] = useState('');
  const [query, setQuery] = useState('');
  const [merchantFilter, setMerchantFilter] = useState('all');

  const load = useCallback(async () => {
    setError('');
    try {
      const [reviewData, channelData, offerData] = await Promise.all([
        apiRequest('/api/admin/channel-reviews?p=1&page_size=100'),
        apiRequest('/api/admin/channels?p=1&page_size=100'),
        apiRequest('/api/offers'),
      ]);
      setReviews(reviewData?.items || []);
      const channelItems = channelData?.items || [];
      setChannels(channelItems);
      setOffers(offerData || []);
      const runtimeResults = await Promise.allSettled(channelItems.map((channel) => apiRequest(`/api/admin/channels/${channel.id}/runtime`)));
      setRuntimes(Object.fromEntries(channelItems.flatMap((channel, index) => runtimeResults[index].status === 'fulfilled' ? [[channel.id, runtimeResults[index].value]] : [])));
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
    setBusyID(`${review.id}-${action}`);
    try {
      await apiRequest(`/api/admin/channel-reviews/${review.id}/${action}`, { method: 'POST', body });
      notify(action === 'verify' ? '渠道验证通过' : action === 'approve' ? '渠道已导入' : '渠道申请已驳回');
      await load();
    } catch (requestError) {
      notify(requestError.message || '渠道审核操作失败');
    } finally {
      setBusyID('');
    }
  };

  const recover = async (channel) => {
    setBusyID(`${channel.id}-recover`);
    try {
      await apiRequest(`/api/admin/channels/${channel.id}/health/reset`, { method: 'POST' });
      notify('渠道已进入恢复探测');
      await load();
    } catch (requestError) {
      notify(requestError.message || '恢复渠道失败');
    } finally {
      setBusyID('');
    }
  };

  const updateChannel = async (channel, action) => {
    setBusyID(`${channel.id}-${action}`);
    try {
      if (action === 'delete') await apiRequest(`/api/admin/channels/${channel.id}`, { method: 'DELETE' });
      else await apiRequest(`/api/admin/channels/${channel.id}/status`, { method: 'POST', body: { enabled: channel.status !== 'active' } });
      notify(action === 'delete' ? '渠道已下架' : channel.status === 'active' ? '渠道已下架' : '渠道已启用');
      await load();
    } catch (requestError) {
      notify(requestError.message || '渠道状态操作失败');
    } finally {
      setBusyID('');
    }
  };

  const testModel = async (review, binding) => {
    setBusyID(`${review.id}-${binding.id}`);
    try { await apiRequest(`/api/admin/channel-reviews/${review.id}/model-tests`, { method: 'POST', body: { binding_id: binding.id } }); notify('模型 hi 检查完成'); await load(); }
    catch (requestError) { notify(requestError.message || '模型检查失败'); }
    finally { setBusyID(''); }
  };

  const visibleReviews = reviews.filter((review) => `${review.channel_name}${review.model || ''}${review.user_id}`.toLowerCase().includes(query.trim().toLowerCase()));
  const visibleChannels = channels.filter((channel) => {
    const matchesQuery = `${channel.channel_name}${channel.model || ''}${channel.merchant_id}${channel.id}`.toLowerCase().includes(query.trim().toLowerCase());
    return matchesQuery && (merchantFilter === 'all' || String(channel.merchant_id) === merchantFilter);
  });

  return (
    <div className="admin-section-stack">
      {error && <InlineNotice tone="warning">{error}</InlineNotice>}
      <div className="filter-toolbar"><label className="search-box"><Search size={17} /><TextInput aria-label="搜索渠道或商户" placeholder="搜索渠道、模型或商户编号" value={query} onChange={(event) => setQuery(event.target.value)} /></label><Select aria-label="筛选商户" value={merchantFilter} onChange={(event) => setMerchantFilter(event.target.value)}><option value="all">全部商户</option>{[...new Set(channels.map((channel) => channel.merchant_id))].map((id) => <option key={id} value={id}>商户 M{id}</option>)}</Select></div>
      <Section
        title="渠道审核"
        description="验证商户提交的上游凭据和模型，批准后导入 New API"
        actions={<Button icon={RefreshCw} onClick={load}>刷新</Button>}
      >
        <div className="table-wrap"><table className="data-table"><thead><tr><th>审核</th><th>商户账号</th><th>渠道</th><th>模型</th><th>检测适配器</th><th>基础地址</th><th>状态</th><th>操作</th></tr></thead><tbody>
          {visibleReviews.map((review) => <tr key={review.id}><td><code>R{review.id}</code><small>{review.created_at ? new Date(review.created_at).toLocaleString('zh-CN') : '-'}</small></td><td>UID {review.user_id}</td><td><strong>{review.channel_name}</strong><small>{review.credential_configured ? '凭据已加密保存' : '凭据正文已清除'}</small></td><td>{review.models?.length ? review.models.map((binding) => <div key={binding.id} className="model-review-row"><span>{binding.upstream_model_name}</span>{binding.match_status === 'matched' && <Button disabled={Boolean(busyID)} variant="ghost" onClick={() => testModel(review, binding)}>hi 检查</Button>}</div>) : (review.model || '-')}</td><td>{review.adapter_type || '-'}</td><td><code>{review.base_url}</code></td><td><Status>{statusNames[review.status] || review.status}</Status><small>{review.verification_error || review.review_reason || ''}</small></td><td><div className="row-actions"><Button disabled={Boolean(busyID) || !['pending', 'verified'].includes(review.status)} icon={ShieldCheck} onClick={() => operate(review, 'verify')} variant="ghost">验证</Button><Button disabled={Boolean(busyID) || review.status !== 'verified'} icon={Check} onClick={() => operate(review, 'approve')} variant="ghost">通过</Button><Button disabled={Boolean(busyID) || !['pending', 'verified'].includes(review.status)} icon={X} onClick={() => operate(review, 'reject')} variant="danger">驳回</Button></div></td></tr>)}
          {reviews.length === 0 && <tr><td className="table-message" colSpan="8">暂无渠道审核记录</td></tr>}
        </tbody></table></div>
      </Section>

      <Section title="已导入渠道" description="展示平台归属、New API 映射和当前报价健康，不回显上游密钥">
        <div className="table-wrap"><table className="data-table"><thead><tr><th>渠道</th><th>商户</th><th>New API 编号</th><th>模型</th><th>检测适配器</th><th>密钥指纹</th><th>状态</th><th>渠道健康</th><th>运行数据</th><th>操作</th></tr></thead><tbody>
          {visibleChannels.map((channel) => {
            const channelOffer = offers.find((offer) => (offer.channels || []).some((item) => item.merchant_channel_id === channel.id));
            const channelHealth = channel.health_state || channelOffer?.channels?.find((item) => item.merchant_channel_id === channel.id)?.health_state;
            const canRecover = channel.status === 'disabled' || channelHealth === 'disabled' || channelHealth === 'cooling';
            const runtime = runtimes[channel.id];
            return <tr key={channel.id}><td><strong>{channel.channel_name}</strong><small>C{channel.id} · R{channel.review_id}</small></td><td>M{channel.merchant_id}</td><td><code>New API {channel.newapi_channel_id}</code></td><td>{channel.model || '-'}</td><td>{channel.adapter_type || '-'}</td><td><code>{channel.credential_fingerprint || '-'}</code></td><td><Status>{statusNames[channel.status] || channel.status}</Status></td><td>{channelHealth ? <Status>{healthNames[channelHealth] || channelHealth}</Status> : '-'}</td><td>{runtime ? <><strong>¥{(Number(runtime.balance_usd_micros || 0) / 1_000_000).toFixed(2)}</strong><small>{runtime.response_time_ms || 0} ms · 状态 {runtime.status}</small></> : <small>运行数据未就绪</small>}</td><td><div className="row-actions"><Button disabled={Boolean(busyID)} icon={Power} onClick={() => updateChannel(channel, 'status')} variant="ghost">{channel.status === 'active' ? '下架' : '启用'}</Button><Button disabled={Boolean(busyID)} icon={Trash2} onClick={() => updateChannel(channel, 'delete')} variant="danger">删除</Button>{canRecover && <Button disabled={Boolean(busyID)} icon={RefreshCw} variant="ghost" onClick={() => recover(channel)}>恢复渠道</Button>}</div></td></tr>;
          })}
          {channels.length === 0 && <tr><td className="table-message" colSpan="10">暂无已导入渠道</td></tr>}
        </tbody></table></div>
      </Section>
    </div>
  );
}
