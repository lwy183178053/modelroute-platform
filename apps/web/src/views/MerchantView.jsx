import { useCallback, useEffect, useState } from 'react';
import {
  BadgeDollarSign, CloudCog, HeartPulse, Landmark, LayoutDashboard,
  Plus, ReceiptText, ServerCog, WalletCards, Waypoints, RefreshCw, Trash2, Power,
} from 'lucide-react';
import { apiRequest, formatImagePrice, formatOfferPrice, formatQuota } from '../api';
import { Button, InlineNotice, Metric, PageHeader, Section, Status } from '../ui';

const statusNames = {
  pending: '待审核', verifying: '验证中', verified: '待批准', approving: '导入中',
  approved: '已通过', rejected: '已驳回', active: '正常', archived: '已归档',
  syncing: '同步中', sync_failed: '同步失败', disabled: '已停用', inactive: '已下架',
};

const withdrawalStatusNames = { pending: '待审核', approved: '待打款', paid: '已打款', rejected: '已驳回' };
const ledgerNames = {
  earning: '调用结算', refund: '退款冲正', withdrawal_freeze: '提现冻结',
  withdrawal_release: '驳回释放', withdrawal_paid: '确认打款',
};
const healthNames = {
  pending: '待调用', healthy: '健康', degraded: '降级', cooling: '冷却', recovering: '恢复中', disabled: '已停用',
};
const protocolNames = { chat_completions: '聊天补全', responses: 'Responses', image_generation: '图片生成' };
const merchantSections = [
  { id: 'overview', label: '经营概览', icon: LayoutDashboard },
  { id: 'channels', label: '渠道管理', icon: Waypoints },
  { id: 'offers', label: '模型报价', icon: BadgeDollarSign },
  { id: 'health', label: '健康状态', icon: HeartPulse },
  { id: 'ledger', label: '结算账本', icon: ReceiptText },
  { id: 'withdrawals', label: '提现记录', icon: Landmark },
];

function formatAvailability(value) {
  return `${(Number(value || 0) / 10_000).toFixed(2)}%`;
}

function formatLatency(value) {
  const milliseconds = Number(value || 0);
  if (!milliseconds) return '-';
  return milliseconds >= 1000 ? `${(milliseconds / 1000).toFixed(2)}s` : `${milliseconds}ms`;
}

function formatDate(value) {
  if (!value) return '-';
  return new Intl.DateTimeFormat('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }).format(new Date(value));
}

function formatQuotaDelta(value) {
  const amount = Number(value || 0);
  if (!amount) return '-';
  return `${amount > 0 ? '+' : '-'}${formatQuota(Math.abs(amount))}`;
}

function formatMultiplier(value) {
  const bps = Number(value || 0);
  return bps === 0 ? '免费' : `${(bps / 10_000).toFixed(2)}x`;
}

function formatOfferValue(offer) {
  return offer.protocol === 'image_generation' ? formatImagePrice(offer.image_prices) : `${formatOfferPrice(offer.price_tiers)} / 百万 Token`;
}

export default function MerchantView({ account, openModal }) {
  const [section, setSection] = useState('overview');
  const [channels, setChannels] = useState([]);
  const [reviews, setReviews] = useState([]);
  const [merchant, setMerchant] = useState(null);
  const [models, setModels] = useState([]);
  const [offers, setOffers] = useState([]);
  const [health, setHealth] = useState([]);
  const [balance, setBalance] = useState(null);
  const [ledger, setLedger] = useState([]);
  const [withdrawals, setWithdrawals] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [busyChannel, setBusyChannel] = useState(0);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const financeRequests = account?.merchant_id
        ? [apiRequest('/api/merchant/balance'), apiRequest('/api/merchant/ledger?page_size=50'), apiRequest('/api/merchant/withdrawals?page_size=50')]
        : [Promise.resolve(null), Promise.resolve(null), Promise.resolve(null)];
      const [channelData, modelData, offerData, healthData, balanceData, ledgerData, withdrawalData] = await Promise.all([
        apiRequest('/api/merchant/channels'), apiRequest('/api/models'), apiRequest('/api/merchant/offers'),
        apiRequest('/api/merchant/health'), ...financeRequests,
      ]);
      setChannels(channelData?.items || []);
      setReviews(channelData?.reviews || []);
      setMerchant(channelData?.merchant || null);
      setModels(modelData || []);
      setOffers(offerData || []);
      setHealth(healthData || []);
      setBalance(balanceData || null);
      setLedger(ledgerData?.items || []);
      setWithdrawals(withdrawalData?.items || []);
    } catch (requestError) {
      setError(requestError.message || '读取商户经营数据失败');
    } finally {
      setLoading(false);
    }
  }, [account?.merchant_id]);

  useEffect(() => { load(); }, [load]);
  const syncChannel = async (channel) => {
    setBusyChannel(channel.id);
    try { await apiRequest(`/api/merchant/channels/${channel.id}/models/sync`, { method: 'POST' }); await load(); }
    catch (requestError) { setError(requestError.message || '同步渠道模型失败'); }
    finally { setBusyChannel(0); }
  };
  const toggleChannel = async (channel) => {
    setBusyChannel(channel.id);
    try { await apiRequest(`/api/merchant/channels/${channel.id}/status`, { method: 'POST', body: { enabled: channel.status !== 'active' } }); await load(); }
    catch (requestError) { setError(requestError.message || '修改渠道状态失败'); }
    finally { setBusyChannel(0); }
  };
  const archiveChannel = async (channel) => {
    if (!window.confirm(`确认下架渠道“${channel.channel_name}”吗？`)) return;
    setBusyChannel(channel.id);
    try { await apiRequest(`/api/merchant/channels/${channel.id}`, { method: 'DELETE' }); await load(); }
    catch (requestError) { setError(requestError.message || '下架渠道失败'); }
    finally { setBusyChannel(0); }
  };
  const activeCount = channels.filter((channel) => channel.status === 'active').length;
  const pendingCount = reviews.filter((review) => ['pending', 'verifying', 'verified', 'approving'].includes(review.status)).length;
  const activeOffers = offers.filter((offer) => offer.offer_status === 'active').length;
  const healthyChannels = health.flatMap((offer) => offer.channels || []).filter((channel) => channel.health_state === 'healthy').length;
  const canSubmit = Boolean(account?.is_merchant && merchant?.merchant_id);

  return (
    <div className="view-shell merchant-view">
      <PageHeader eyebrow={`商户中心${merchant?.merchant_id ? ` · M${merchant.merchant_id}` : ''}`} title="商户经营概览" description="管理渠道、模型报价、运行健康、结算账本和人工提现。" actions={merchant && <Status>{merchant.merchant_status === 'active' ? '正常' : statusNames[merchant.merchant_status] || merchant.merchant_status}</Status>} />
      {error && <InlineNotice tone="warning">{error}</InlineNotice>}
      {!error && !loading && !canSubmit && <InlineNotice tone="warning">当前商户资料尚未就绪</InlineNotice>}

      <div className="admin-layout merchant-layout">
        <aside className="admin-sidebar" aria-label="商户功能">
          {merchantSections.map(({ id, label, icon: Icon }) => <button aria-label={label} className={section === id ? 'active' : ''} key={id} onClick={() => setSection(id)} type="button"><Icon size={17} /><span>{label}</span></button>)}
        </aside>

        <div className="admin-content">
          {section === 'overview' && <>
            <div className="metrics-grid">
              <Metric label="已导入渠道" value={String(channels.length)} hint={`正常 ${activeCount} 个`} icon={Waypoints} />
              <Metric label="审核处理中" value={String(pendingCount)} hint="密钥提交后不再回显" icon={CloudCog} tone="amber" />
              <Metric label="在售模型" value={String(activeOffers)} hint={`${offers.length} 个报价版本`} icon={ServerCog} tone="blue" />
              <Metric label="健康渠道" value={String(healthyChannels)} hint={merchant?.merchant_status === 'active' ? '商户身份正常' : '商户身份待处理'} icon={HeartPulse} tone="green" />
            </div>
            <div className="metrics-grid">
              <Metric label="可提现余额" value={formatQuota(balance?.available_quota)} hint="提交后立即冻结" icon={WalletCards} />
              <Metric label="冻结余额" value={formatQuota(balance?.frozen_quota)} hint="等待审核或线下打款" icon={Landmark} tone="amber" />
              <Metric label="累计结算" value={formatQuota(balance?.lifetime_earned_quota)} hint={`退款冲正 ${formatQuota(balance?.lifetime_refunded_quota)}`} icon={BadgeDollarSign} tone="green" />
              <Metric label="累计提现" value={formatQuota(balance?.lifetime_paid_quota)} hint="只统计已确认打款" icon={Landmark} tone="blue" />
            </div>
          </>}

          {section === 'channels' && <>
            <Section title="渠道审核版本" actions={canSubmit && <Button icon={Plus} variant="primary" onClick={() => openModal({ type: 'channel', models: models.filter((model) => model.status === 'active'), onSaved: load })}>提交新渠道</Button>}>
              <div className="table-wrap"><table className="data-table"><thead><tr><th>审核编号</th><th>渠道</th><th>模型</th><th>检测适配器</th><th>基础地址</th><th>凭据</th><th>状态</th><th>结果</th></tr></thead><tbody>
                {reviews.map((review) => <tr key={review.id}><td><code>R{review.id}</code></td><td><strong>{review.channel_name}</strong></td><td>{review.model || '-'}</td><td>{review.adapter_type || '-'}</td><td><code>{review.base_url}</code></td><td>{review.credential_configured ? '已加密保存' : '已清除正文'}</td><td><Status>{statusNames[review.status] || review.status}</Status></td><td>{review.verification_error || review.review_reason || '-'}</td></tr>)}
                {!loading && reviews.length === 0 && <tr><td className="table-message" colSpan="8">暂无渠道审核记录</td></tr>}
                {loading && <tr><td className="table-message" colSpan="8">正在读取渠道审核记录</td></tr>}
              </tbody></table></div>
            </Section>
            <Section title="已导入渠道">
              <div className="table-wrap"><table className="data-table"><thead><tr><th>渠道</th><th>New API 编号</th><th>已同步模型</th><th>检测适配器</th><th>基础地址</th><th>密钥指纹</th><th>状态</th><th></th></tr></thead><tbody>
                {channels.map((channel) => <tr key={channel.id}><td><strong>{channel.channel_name}</strong><small>C{channel.id} · R{channel.review_id}</small></td><td><code>{channel.newapi_channel_id}</code></td><td>{channel.models?.length ? channel.models.map((model) => <span className="tag" key={model}>{model}</span>) : (channel.model || '-')}</td><td>{channel.adapter_type || '-'}</td><td><code>{channel.base_url}</code></td><td><code>{channel.credential_fingerprint || '-'}</code></td><td><Status>{statusNames[channel.status] || channel.status}</Status></td><td><div className="row-actions"><Button icon={RefreshCw} disabled={busyChannel === channel.id} variant="ghost" onClick={() => syncChannel(channel)}>同步模型</Button><Button icon={Power} disabled={busyChannel === channel.id || channel.status === 'archived'} variant="ghost" onClick={() => toggleChannel(channel)}>{channel.status === 'active' ? '下架' : '启用'}</Button><Button icon={Trash2} disabled={busyChannel === channel.id} variant="danger" onClick={() => archiveChannel(channel)}>删除</Button></div></td></tr>)}
                {!loading && channels.length === 0 && <tr><td className="table-message" colSpan="8">暂无已导入渠道</td></tr>}
              </tbody></table></div>
            </Section>
          </>}

          {section === 'offers' && <Section title="模型报价" actions={canSubmit && <Button disabled={!channels.length || !models.length} icon={BadgeDollarSign} variant="primary" onClick={() => openModal({ type: 'quote', models: models.filter((model) => model.status === 'active'), onSaved: load })}>设置报价</Button>}>
            <div className="table-wrap"><table className="data-table"><thead><tr><th>报价</th><th>模型</th><th>倍率</th><th>当前价格</th><th>关联渠道</th><th>版本状态</th><th>健康</th><th></th></tr></thead><tbody>
              {offers.map((offer) => <tr key={offer.offer_id}><td><code>O{offer.offer_id}</code></td><td><strong>{offer.model}</strong><small>{protocolNames[offer.protocol] || offer.protocol}</small></td><td><span className="price-multiple">{formatMultiplier(offer.multiplier_bps)}</span></td><td><strong>{formatOfferValue(offer)}</strong>{offer.price_tiers?.length > 1 && <small>{offer.price_tiers.length} 个上下文阶梯</small>}</td><td><strong>{offer.channels?.length || 0} 个</strong><small>{offer.channels?.map((channel) => channel.channel_name).join('、') || '-'}</small></td><td><Status>{offer.offer_status === 'active' ? '在售' : statusNames[offer.offer_status] || offer.offer_status}</Status></td><td><Status>{healthNames[offer.health_state] || offer.health_state}</Status></td><td><Button variant="ghost" onClick={() => openModal({ type: 'quote', models, offer, onSaved: load })}>调整倍率</Button></td></tr>)}
              {!loading && offers.length === 0 && <tr><td className="table-message" colSpan="8">暂无报价版本</td></tr>}
            </tbody></table></div>
          </Section>}

          {section === 'health' && <Section title="渠道健康" description="健康渠道只由真实调用更新，冷却到期后执行一次恢复检测。">
            <div className="table-wrap"><table className="data-table"><thead><tr><th>模型</th><th>渠道</th><th>状态</th><th>可用率</th><th>P95 首字</th></tr></thead><tbody>
              {health.flatMap((offer) => (offer.channels || []).map((channel) => <tr key={`${offer.offer_id}-${channel.merchant_channel_id}`}><td><strong>{offer.model}</strong></td><td>{channel.channel_name}</td><td><Status>{healthNames[channel.health_state] || channel.health_state}</Status></td><td>{formatAvailability(channel.availability_ppm)}</td><td>{formatLatency(channel.ttft_p95_ms)}</td></tr>))}
              {!loading && health.flatMap((offer) => offer.channels || []).length === 0 && <tr><td className="table-message" colSpan="5">暂无健康记录</td></tr>}
            </tbody></table></div>
          </Section>}

          {section === 'ledger' && <Section title="结算账本">
            <div className="table-wrap"><table className="data-table"><thead><tr><th>类型</th><th>关联对象</th><th>可用变动</th><th>冻结变动</th><th>变动后余额</th><th>时间</th></tr></thead><tbody>
              {ledger.map((entry) => <tr key={entry.id}><td><Status>{ledgerNames[entry.entry_type] || entry.entry_type}</Status></td><td><strong>{entry.model_name || entry.withdrawal_no || '-'}</strong><small>{entry.newapi_log_id ? `New API 日志 ${entry.newapi_log_id}` : entry.description}</small></td><td>{formatQuotaDelta(entry.available_delta)}</td><td>{formatQuotaDelta(entry.frozen_delta)}</td><td>{formatQuota(entry.available_after)}<small>冻结 {formatQuota(entry.frozen_after)}</small></td><td>{formatDate(entry.created_at)}</td></tr>)}
              {!loading && ledger.length === 0 && <tr><td className="table-message" colSpan="6">暂无结算账本记录</td></tr>}
            </tbody></table></div>
          </Section>}

          {section === 'withdrawals' && <Section title="提现记录" actions={canSubmit && <Button disabled={!balance?.available_quota} icon={Landmark} variant="primary" onClick={() => openModal({ type: 'withdraw', balance, onSaved: load })}>申请提现</Button>}>
            <div className="table-wrap"><table className="data-table"><thead><tr><th>申请编号</th><th>金额</th><th>收款方式</th><th>收款账户</th><th>状态</th><th>申请时间</th></tr></thead><tbody>
              {withdrawals.map((item) => <tr key={item.id}><td><code>{item.withdrawal_no}</code></td><td><strong>{formatQuota(item.amount_quota)}</strong></td><td>{item.recipient_type === 'alipay' ? '支付宝' : item.recipient_type === 'bank' ? '银行卡' : '其他'}</td><td><code>{item.recipient_account_masked}</code></td><td><Status>{withdrawalStatusNames[item.status] || item.status}</Status></td><td>{formatDate(item.created_at)}</td></tr>)}
              {!loading && withdrawals.length === 0 && <tr><td className="table-message" colSpan="6">暂无提现申请</td></tr>}
            </tbody></table></div>
          </Section>}
        </div>
      </div>
    </div>
  );
}
