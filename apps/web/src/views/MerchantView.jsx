import { useCallback, useEffect, useState } from 'react';
import { BadgeDollarSign, CloudCog, HeartPulse, Plus, ServerCog, Store, Waypoints } from 'lucide-react';
import { apiRequest } from '../api';
import { Button, InlineNotice, Metric, PageHeader, Section, Status } from '../ui';

const statusNames = {
  pending: '待审核', verifying: '验证中', verified: '待批准', approving: '导入中',
  approved: '已通过', rejected: '已驳回', active: '正常', inactive: '已退役',
  syncing: '同步中', sync_failed: '同步失败', disabled: '已停用',
};

const healthNames = {
  pending: '待探测', healthy: '健康', degraded: '降级', cooling: '冷却', recovering: '恢复中', disabled: '已停用',
};

const channelTypeNames = {
  1: 'OpenAI 兼容', 14: 'Anthropic', 20: 'OpenRouter', 24: 'Gemini',
  25: 'Moonshot', 27: 'Perplexity', 40: 'SiliconFlow', 42: 'Mistral', 43: 'DeepSeek', 48: 'xAI',
};

function formatMultiplier(value) {
  return `${(Number(value || 0) / 10_000).toFixed(4)}x`;
}

function formatAvailability(value) {
  return `${(Number(value || 0) / 10_000).toFixed(2)}%`;
}

function formatLatency(value) {
  const milliseconds = Number(value || 0);
  if (!milliseconds) return '-';
  return milliseconds >= 1000 ? `${(milliseconds / 1000).toFixed(2)}s` : `${milliseconds}ms`;
}

export default function MerchantView({ account, openModal }) {
  const [channels, setChannels] = useState([]);
  const [reviews, setReviews] = useState([]);
  const [merchant, setMerchant] = useState(null);
  const [models, setModels] = useState([]);
  const [offers, setOffers] = useState([]);
  const [health, setHealth] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [channelData, modelData, offerData, healthData] = await Promise.all([
        apiRequest('/api/merchant/channels'),
        apiRequest('/api/models'),
        apiRequest('/api/merchant/offers'),
        apiRequest('/api/merchant/health'),
      ]);
      setChannels(channelData?.items || []);
      setReviews(channelData?.reviews || []);
      setMerchant(channelData?.merchant || null);
      setModels(modelData || []);
      setOffers(offerData || []);
      setHealth(healthData || []);
    } catch (requestError) {
      setError(requestError.message || '读取商户经营数据失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);
  const activeCount = channels.filter((channel) => channel.status === 'active').length;
  const pendingCount = reviews.filter((review) => ['pending', 'verifying', 'verified', 'approving'].includes(review.status)).length;
  const activeOffers = offers.filter((offer) => offer.offer_status === 'active').length;
  const healthyOffers = health.filter((offer) => offer.health_state === 'healthy').length;
  const canSubmit = Boolean(account?.is_merchant && merchant?.merchant_id);

  return (
    <div className="view-shell">
      <PageHeader
        eyebrow={`商户中心${merchant?.merchant_id ? ` · M${merchant.merchant_id}` : ''}`}
        title="商户经营概览"
        description="管理渠道、报价版本以及当前探活和真实请求健康状态。"
        actions={canSubmit && <><Button icon={Plus} onClick={() => openModal({ type: 'channel', onSaved: load })}>提交新渠道</Button><Button disabled={!channels.length || !models.length} icon={BadgeDollarSign} variant="primary" onClick={() => openModal({ type: 'quote', channels: channels.filter((channel) => channel.status === 'active'), models: models.filter((model) => model.status === 'active'), onSaved: load })}>创建报价</Button></>}
      />

      {error && <InlineNotice tone="warning">{error}</InlineNotice>}
      {!canSubmit && Number(account?.role) >= 10 && <InlineNotice>管理员账号当前未绑定商户资料，可在商户审核页管理全平台申请。</InlineNotice>}

      <div className="metrics-grid">
        <Metric label="已导入渠道" value={String(channels.length)} hint={`正常 ${activeCount} 个`} icon={Waypoints} />
        <Metric label="审核处理中" value={String(pendingCount)} hint="密钥提交后不再回显" icon={CloudCog} tone="amber" />
        <Metric label="在售报价" value={String(activeOffers)} hint={`${offers.length} 个报价版本`} icon={ServerCog} tone="blue" />
        <Metric label="健康报价" value={String(healthyOffers)} hint={merchant?.merchant_status === 'active' ? '商户身份正常' : '商户身份待处理'} icon={HeartPulse} tone="green" />
      </div>

      <Section title="渠道审核版本" description="新渠道、密钥变更和基础地址变更均保留独立审核记录">
        <div className="table-wrap"><table className="data-table"><thead><tr><th>审核编号</th><th>渠道</th><th>类型</th><th>基础地址</th><th>拟售模型</th><th>凭据</th><th>状态</th><th>结果</th></tr></thead><tbody>
          {reviews.map((review) => <tr key={review.id}><td><code>R{review.id}</code></td><td><strong>{review.channel_name}</strong></td><td>{channelTypeNames[review.channel_type] || `类型 ${review.channel_type}`}</td><td><code>{review.base_url}</code></td><td>{review.requested_models?.join('、') || '-'}</td><td>{review.credential_configured ? '已加密保存' : '已清除正文'}</td><td><Status>{statusNames[review.status] || review.status}</Status></td><td>{review.verification_error || review.review_reason || '-'}</td></tr>)}
          {!loading && reviews.length === 0 && <tr><td className="table-message" colSpan="8">暂无渠道审核记录</td></tr>}
          {loading && <tr><td className="table-message" colSpan="8">正在读取渠道审核记录</td></tr>}
        </tbody></table></div>
      </Section>

      <Section title="已导入渠道" description="审核通过后由平台创建 New API 渠道并保存归属关系">
        <div className="table-wrap"><table className="data-table"><thead><tr><th>渠道</th><th>New API 编号</th><th>类型</th><th>基础地址</th><th>模型</th><th>密钥指纹</th><th>状态</th></tr></thead><tbody>
          {channels.map((channel) => <tr key={channel.id}><td><strong>{channel.channel_name}</strong><small>C{channel.id} · R{channel.review_id}</small></td><td><code>{channel.newapi_channel_id}</code></td><td>{channelTypeNames[channel.channel_type] || `类型 ${channel.channel_type}`}</td><td><code>{channel.base_url}</code></td><td>{channel.models?.join('、') || '-'}</td><td><code>{channel.credential_fingerprint || '-'}</code></td><td><Status>{statusNames[channel.status] || channel.status}</Status></td></tr>)}
          {!loading && channels.length === 0 && <tr><td className="table-message" colSpan="7">暂无已导入渠道</td></tr>}
        </tbody></table></div>
      </Section>

      <Section title="模型报价" description="调价生成新版本，New API 同步成功后才进入用户路由">
        <div className="table-wrap"><table className="data-table"><thead><tr><th>报价</th><th>模型</th><th>渠道</th><th>倍率</th><th>版本状态</th><th>健康</th><th>同步结果</th></tr></thead><tbody>
          {offers.map((offer) => <tr key={offer.offer_id}><td><code>O{offer.offer_id}</code></td><td><strong>{offer.model}</strong><small>{offer.protocol}</small></td><td>{offer.channel_name}<small>C{offer.merchant_channel_id}</small></td><td><span className="price-multiple">{formatMultiplier(offer.multiplier_bps)}</span></td><td><Status>{offer.offer_status === 'active' ? '在售' : statusNames[offer.offer_status] || offer.offer_status}</Status></td><td><Status>{healthNames[offer.health_state] || offer.health_state}</Status></td><td>{offer.sync_error || '-'}</td></tr>)}
          {!loading && offers.length === 0 && <tr><td className="table-message" colSpan="7">暂无报价版本</td></tr>}
        </tbody></table></div>
      </Section>

      <Section title="渠道健康" description="主动探活和真实请求只记录状态、耗时、渠道与请求编号">
        <div className="table-wrap"><table className="data-table"><thead><tr><th>模型</th><th>渠道</th><th>状态</th><th>可用率</th><th>P95 首字</th></tr></thead><tbody>
          {health.map((item) => <tr key={`${item.offer_id}-${item.merchant_channel_id}`}><td><strong>{item.model}</strong></td><td>{item.channel_name}</td><td><Status>{healthNames[item.health_state] || item.health_state}</Status></td><td>{formatAvailability(item.availability_ppm)}</td><td>{formatLatency(item.ttft_p95_ms)}</td></tr>)}
          {!loading && health.length === 0 && <tr><td className="table-message" colSpan="5">暂无健康记录</td></tr>}
        </tbody></table></div>
      </Section>
    </div>
  );
}
