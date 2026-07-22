import { useCallback, useEffect, useState } from 'react';
import { CloudCog, HeartPulse, Plus, ServerCog, Store, Waypoints } from 'lucide-react';
import { apiRequest } from '../api';
import { Button, EmptyNotice, InlineNotice, Metric, PageHeader, Section, Status } from '../ui';

const statusNames = {
  pending: '待审核', verifying: '验证中', verified: '待批准', approving: '导入中',
  approved: '已通过', rejected: '已驳回', active: '正常', disabled: '停用',
};

const channelTypeNames = {
  1: 'OpenAI 兼容', 14: 'Anthropic', 20: 'OpenRouter', 24: 'Gemini',
  25: 'Moonshot', 27: 'Perplexity', 40: 'SiliconFlow', 42: 'Mistral', 43: 'DeepSeek', 48: 'xAI',
};

export default function MerchantView({ account, openModal, notify }) {
  const [channels, setChannels] = useState([]);
  const [reviews, setReviews] = useState([]);
  const [merchant, setMerchant] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const loadChannels = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const data = await apiRequest('/api/merchant/channels');
      setChannels(data?.items || []);
      setReviews(data?.reviews || []);
      setMerchant(data?.merchant || null);
    } catch (requestError) {
      setError(requestError.message || '读取商户渠道失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { loadChannels(); }, [loadChannels]);
  const activeCount = channels.filter((channel) => channel.status === 'active').length;
  const pendingCount = reviews.filter((review) => ['pending', 'verifying', 'verified', 'approving'].includes(review.status)).length;
  const canSubmit = Boolean(account?.is_merchant && merchant?.merchant_id);

  return (
    <div className="view-shell">
      <PageHeader
        eyebrow={`商户中心${merchant?.merchant_id ? ` · M${merchant.merchant_id}` : ''}`}
        title="商户经营概览"
        description="查看渠道审核版本、New API 导入编号和当前运行状态。"
        actions={canSubmit && <Button icon={Plus} variant="primary" onClick={() => openModal({ type: 'channel', onSaved: loadChannels })}>提交新渠道</Button>}
      />

      {error && <InlineNotice tone="warning">{error}</InlineNotice>}
      {!canSubmit && Number(account?.role) >= 10 && <InlineNotice>管理员账号当前未绑定商户资料，可在商户审核页管理全平台申请。</InlineNotice>}

      <div className="metrics-grid">
        <Metric label="已导入渠道" value={String(channels.length)} hint={`正常 ${activeCount} 个`} icon={Waypoints} />
        <Metric label="审核处理中" value={String(pendingCount)} hint="密钥提交后不再回显" icon={CloudCog} tone="amber" />
        <Metric label="New API 映射" value={String(channels.filter((channel) => channel.newapi_channel_id).length)} hint="渠道编号已留存" icon={ServerCog} tone="blue" />
        <Metric label="商户状态" value={merchant?.merchant_status === 'active' ? '正常' : '--'} hint={account?.is_merchant ? '同账号商户身份' : '未绑定商户'} icon={Store} tone="green" />
      </div>

      <Section title="渠道审核版本" description="新渠道、密钥变更和基础地址变更均保留独立审核记录">
        <div className="table-wrap">
          <table className="data-table">
            <thead><tr><th>审核编号</th><th>渠道</th><th>类型</th><th>基础地址</th><th>拟售模型</th><th>凭据</th><th>状态</th><th>结果</th></tr></thead>
            <tbody>
              {reviews.map((review) => <tr key={review.id}><td><code>R{review.id}</code></td><td><strong>{review.channel_name}</strong></td><td>{channelTypeNames[review.channel_type] || `类型 ${review.channel_type}`}</td><td><code>{review.base_url}</code></td><td>{review.requested_models?.join('、') || '-'}</td><td>{review.credential_configured ? '已加密保存' : '已清除正文'}</td><td><Status>{statusNames[review.status] || review.status}</Status></td><td>{review.verification_error || review.review_reason || '-'}</td></tr>)}
              {!loading && reviews.length === 0 && <tr><td className="table-message" colSpan="8">暂无渠道审核记录</td></tr>}
              {loading && <tr><td className="table-message" colSpan="8">正在读取渠道审核记录</td></tr>}
            </tbody>
          </table>
        </div>
      </Section>

      <Section title="已导入渠道" description="审核通过后由平台创建 New API 渠道并保存归属关系">
        <div className="table-wrap">
          <table className="data-table">
            <thead><tr><th>渠道</th><th>New API 编号</th><th>类型</th><th>基础地址</th><th>模型</th><th>密钥指纹</th><th>状态</th></tr></thead>
            <tbody>
              {channels.map((channel) => <tr key={channel.id}><td><strong>{channel.channel_name}</strong><small>C{channel.id} · R{channel.review_id}</small></td><td><code>{channel.newapi_channel_id}</code></td><td>{channelTypeNames[channel.channel_type] || `类型 ${channel.channel_type}`}</td><td><code>{channel.base_url}</code></td><td>{channel.models?.join('、') || '-'}</td><td><code>{channel.credential_fingerprint || '-'}</code></td><td><Status>{statusNames[channel.status] || channel.status}</Status></td></tr>)}
              {!loading && channels.length === 0 && <tr><td className="table-message" colSpan="7">暂无已导入渠道</td></tr>}
            </tbody>
          </table>
        </div>
      </Section>

      <Section title="模型报价" description="标准模型和官方价格版本将在下一阶段接入">
        <EmptyNotice title="尚未发布模型报价" description="渠道通过审核后，再按官方价格版本创建报价。" />
      </Section>

      <Section title="渠道健康" description="探活和冷却将在路由阶段统一接入">
        <EmptyNotice title="暂无健康事件" description="模型报价生效后开始记录探活和真实调用结果。" />
      </Section>
    </div>
  );
}
