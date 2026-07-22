import { useCallback, useEffect, useState } from 'react';
import { Check, RefreshCw, ShieldCheck, X } from 'lucide-react';
import { apiRequest } from '../../api';
import { Button, InlineNotice, Section, Status } from '../../ui';

const statusNames = {
  pending: '待审核', verifying: '验证中', verified: '待批准', approving: '导入中',
  approved: '已通过', rejected: '已驳回', active: '正常', disabled: '停用',
};

const channelTypeNames = {
  1: 'OpenAI 兼容', 14: 'Anthropic', 20: 'OpenRouter', 24: 'Gemini',
  25: 'Moonshot', 27: 'Perplexity', 40: 'SiliconFlow', 42: 'Mistral', 43: 'DeepSeek', 48: 'xAI',
};

export default function AdminChannels({ notify }) {
  const [reviews, setReviews] = useState([]);
  const [channels, setChannels] = useState([]);
  const [error, setError] = useState('');
  const [busyID, setBusyID] = useState('');

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

  return (
    <div className="admin-section-stack">
      {error && <InlineNotice tone="warning">{error}</InlineNotice>}
      <Section
        title="渠道审核"
        description="验证商户提交的上游凭据和模型，批准后导入 New API"
        actions={<Button icon={RefreshCw} onClick={load}>刷新</Button>}
      >
        <div className="table-wrap"><table className="data-table"><thead><tr><th>审核</th><th>商户账号</th><th>渠道</th><th>类型</th><th>基础地址</th><th>拟售模型</th><th>状态</th><th>操作</th></tr></thead><tbody>
          {reviews.map((review) => <tr key={review.id}><td><code>R{review.id}</code><small>{review.created_at ? new Date(review.created_at).toLocaleString('zh-CN') : '-'}</small></td><td>UID {review.user_id}</td><td><strong>{review.channel_name}</strong><small>{review.credential_configured ? '凭据已加密保存' : '凭据正文已清除'}</small></td><td>{channelTypeNames[review.channel_type] || `类型 ${review.channel_type}`}</td><td><code>{review.base_url}</code></td><td>{review.requested_models?.join('、') || '-'}</td><td><Status>{statusNames[review.status] || review.status}</Status><small>{review.verification_error || review.review_reason || ''}</small></td><td><div className="row-actions"><Button disabled={Boolean(busyID) || !['pending', 'verified'].includes(review.status)} icon={ShieldCheck} onClick={() => operate(review, 'verify')} variant="ghost">验证</Button><Button disabled={Boolean(busyID) || review.status !== 'verified'} icon={Check} onClick={() => operate(review, 'approve')} variant="ghost">通过</Button><Button disabled={Boolean(busyID) || !['pending', 'verified'].includes(review.status)} icon={X} onClick={() => operate(review, 'reject')} variant="danger">驳回</Button></div></td></tr>)}
          {reviews.length === 0 && <tr><td className="table-message" colSpan="8">暂无渠道审核记录</td></tr>}
        </tbody></table></div>
      </Section>

      <Section title="已导入渠道" description="展示平台归属和 New API 渠道映射，不回显上游密钥">
        <div className="table-wrap"><table className="data-table"><thead><tr><th>渠道</th><th>商户</th><th>New API 编号</th><th>类型</th><th>基础地址</th><th>模型</th><th>密钥指纹</th><th>状态</th></tr></thead><tbody>
          {channels.map((channel) => <tr key={channel.id}><td><strong>{channel.channel_name}</strong><small>C{channel.id} · R{channel.review_id}</small></td><td>M{channel.merchant_id}</td><td><code>New API {channel.newapi_channel_id}</code></td><td>{channelTypeNames[channel.channel_type] || `类型 ${channel.channel_type}`}</td><td><code>{channel.base_url}</code></td><td>{channel.models?.join('、') || '-'}</td><td><code>{channel.credential_fingerprint || '-'}</code></td><td><Status>{statusNames[channel.status] || channel.status}</Status></td></tr>)}
          {channels.length === 0 && <tr><td className="table-message" colSpan="8">暂无已导入渠道</td></tr>}
        </tbody></table></div>
      </Section>

      <InlineNotice>渠道探活、冷却和恢复记录将在路由阶段接入真实任务与状态。</InlineNotice>
    </div>
  );
}
