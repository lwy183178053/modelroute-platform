import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  BadgeDollarSign, HeartPulse, Landmark, LayoutDashboard,
  Plus, ServerCog, Settings2, WalletCards, Waypoints, FileText,
} from 'lucide-react';
import { apiRequest, formatCompactCount, formatLastCalled, formatQuota, getQuotaPerUnit } from '../api';
import { ActionMenu, Button, Field, InlineNotice, Metric, PageHeader, Pagination, Section, Select, Status, TextInput } from '../ui';
import { UsageLogTable, useUsage } from './UsagePanels';
import ChannelModelsModal, { ChannelModelsButton } from '../ChannelModelsModal';

const statusNames = {
  active: '已上架', inactive: '已下架', deleted: '已删除',
};
const withdrawalStatusNames = { pending: '待审核', approved: '待打款', paid: '已打款', rejected: '已驳回' };
const healthNames = {
  pending: '待检查', healthy: '正常', cooling: '冷却', error: '错误',
};
const merchantSections = [
  { id: 'overview', label: '经营概览', icon: LayoutDashboard },
  { id: 'channels', label: '渠道管理', icon: Waypoints },
  { id: 'offers', label: '上架模型', icon: BadgeDollarSign },
  { id: 'usage', label: '调用记录', icon: FileText },
  { id: 'withdrawals', label: '提现记录', icon: Landmark },
  { id: 'settings', label: '商户设置', icon: Settings2 },
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

function formatMultiplier(value) {
  const bps = Number(value || 0);
  return bps === 0 ? '免费' : `${(bps / 10_000).toFixed(2)}x`;
}

function formatCacheHit(value, ready, isImage = false) {
  if (isImage) return '-';
  if (!ready) return '样本不足';
  return `${(Number(value || 0) / 10_000).toFixed(2)}%`;
}

function countModelStates(models = []) {
  return models.reduce((counts, model) => {
    const state = model.health_state || 'pending';
    counts[state] = (counts[state] || 0) + 1;
    return counts;
  }, { pending: 0, healthy: 0, cooling: 0, error: 0 });
}

export default function MerchantView({ account, openModal }) {
  const minimumWithdrawalQuota = 50 * getQuotaPerUnit();
  const [section, setSection] = useState('overview');
  const [channels, setChannels] = useState([]);
  const [merchant, setMerchant] = useState(null);
  const [models, setModels] = useState([]);
  const [offers, setOffers] = useState([]);
  const [balance, setBalance] = useState(null);
  const [withdrawals, setWithdrawals] = useState([]);
  const [withdrawalTotal, setWithdrawalTotal] = useState(0);
  const [withdrawalPage, setWithdrawalPage] = useState(1);
  const [withdrawalPageSize, setWithdrawalPageSize] = useState(20);
  const [usagePage, setUsagePage] = useState(1);
  const [usagePageSize, setUsagePageSize] = useState(20);
  const [usageModel, setUsageModel] = useState('all');
  const [expandedOffer, setExpandedOffer] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [busyChannel, setBusyChannel] = useState(0);
  const [businessDescription, setBusinessDescription] = useState('');
  const [profileSaving, setProfileSaving] = useState(false);
  const [profileMessage, setProfileMessage] = useState('');
  const [modelTarget, setModelTarget] = useState(null);

  const usageEndpoint = useMemo(() => {
    const query = new URLSearchParams({ p: String(usagePage), page_size: String(usagePageSize) });
    if (usageModel !== 'all') query.set('model_name', usageModel);
    return `/api/merchant/usage?${query}`;
  }, [usageModel, usagePage, usagePageSize]);
  const usage = useUsage(usageEndpoint);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const financeRequests = account?.is_merchant || Number(account?.role) >= 10
        ? [apiRequest('/api/merchant/balance'), apiRequest(`/api/merchant/withdrawals?p=${withdrawalPage}&page_size=${withdrawalPageSize}`)]
        : [Promise.resolve(null), Promise.resolve(null)];
      const [channelData, profileData, modelData, offerData, balanceData, withdrawalData] = await Promise.all([
        apiRequest('/api/merchant/channels'), apiRequest('/api/merchant/profile'), apiRequest('/api/models'), apiRequest('/api/merchant/offers'), ...financeRequests,
      ]);
      setChannels(channelData?.items || []);
      setMerchant(profileData || channelData?.merchant || null);
      setBusinessDescription(profileData?.business_description || '');
      setModels(modelData || []);
      setOffers(offerData || []);
      setBalance(balanceData || null);
      setWithdrawals(withdrawalData?.items || []);
      setWithdrawalTotal(Number(withdrawalData?.total || 0));
    } catch (requestError) {
      setError(requestError.message || '读取商户经营数据失败');
    } finally {
      setLoading(false);
    }
  }, [account?.is_merchant, account?.role, withdrawalPage, withdrawalPageSize]);

  useEffect(() => { load(); }, [load]);

  const syncChannel = async (channel) => {
    setBusyChannel(channel.id);
    try { await apiRequest(`/api/merchant/channels/${channel.id}/models/sync`, { method: 'POST' }); await load(); }
    catch (requestError) { setError(requestError.message || '同步渠道模型失败'); }
    finally { setBusyChannel(0); }
  };
  const checkChannelModel = async (channel, model) => {
    setBusyChannel(`model-${model.id}`);
    setError('');
    try {
      await apiRequest(`/api/merchant/channels/${channel.id}/models/${model.id}/check`, { method: 'POST' });
      await load();
    } catch (requestError) {
      await load();
      setError(requestError.message || '模型检查失败');
    } finally {
      setBusyChannel(0);
    }
  };
  const deleteChannel = async (channel) => {
    if (!window.confirm(`确认删除渠道“${channel.channel_name}”吗？`)) return;
    setBusyChannel(channel.id);
    try { await apiRequest(`/api/merchant/channels/${channel.id}`, { method: 'DELETE' }); await load(); }
    catch (requestError) { setError(requestError.message || '删除渠道失败'); }
    finally { setBusyChannel(0); }
  };
  const modelForOffer = (offer) => models.find((item) => item.public_name === offer.model);
  const changeOfferStatus = async (offer, enabled) => {
    const model = modelForOffer(offer);
    if (!model) return;
    try { await apiRequest(`/api/merchant/offers/${model.id}/status`, { method: 'POST', body: { enabled } }); await load(); }
    catch (requestError) { setError(requestError.message || `${enabled ? '上架' : '下架'}模型失败`); }
  };
  const deleteOffer = async (offer) => {
    const model = modelForOffer(offer);
    if (!model || !window.confirm(`确认删除“${offer.model}”吗？历史调用记录仍会保留。`)) return;
    try { await apiRequest(`/api/merchant/offers/${model.id}`, { method: 'DELETE' }); await load(); }
    catch (requestError) { setError(requestError.message || '删除上架模型失败'); }
  };
  const saveProfile = async () => {
    const description = businessDescription.trim();
    if (description.length < 10 || description.length > 2000) {
      setProfileMessage('商户介绍需为 10 至 2000 字');
      return;
    }
    setProfileSaving(true);
    setProfileMessage('');
    try {
      const profile = await apiRequest('/api/merchant/profile', { method: 'PUT', body: { business_description: description } });
      setMerchant(profile);
      setBusinessDescription(profile.business_description || '');
      setProfileMessage('商户介绍已保存');
    } catch (requestError) {
      setProfileMessage(requestError.message || '保存商户介绍失败');
    } finally {
      setProfileSaving(false);
    }
  };

  const channelModelCounts = countModelStates(channels.flatMap((channel) => channel.models || []));
  const activeOffers = offers.filter((offer) => offer.offer_status === 'active').length;
  const canSubmit = Boolean(account?.is_merchant || Number(account?.role) >= 10);
  const selectedModelRow = modelTarget ? channels.find((item) => item.id === modelTarget) : null;
  const merchantName = merchant?.display_name || merchant?.merchant_name || '';

  return (
    <div className="view-shell merchant-view">
      <PageHeader eyebrow={`商户中心 · 用户 ID ${account.id}`} title="商户经营概览" description="管理渠道、上架模型、调用收入和人工提现。" actions={merchantName && <span className="merchant-name-badge" title={merchantName}>{merchantName}</span>} />
      {error && <InlineNotice tone="warning">{error}</InlineNotice>}
      {!error && !loading && !canSubmit && <InlineNotice tone="warning">当前商户资料尚未就绪</InlineNotice>}

      <div className="admin-layout merchant-layout">
        <aside className="admin-sidebar" aria-label="商户功能">
          {merchantSections.map(({ id, label, icon: Icon }) => <button aria-label={label} className={section === id ? 'active' : ''} key={id} onClick={() => setSection(id)} type="button"><Icon size={17} /><span>{label}</span></button>)}
        </aside>

        <div className="admin-content">
          {section === 'overview' && <>
            <div className="metrics-grid">
              <Metric label="渠道总数" value={String(channels.length)} hint="渠道内模型独立维护状态" icon={Waypoints} />
              <Metric label="正常模型" value={String(channelModelCounts.healthy)} hint={`${channelModelCounts.pending} 个待检查`} icon={HeartPulse} tone="green" />
              <Metric label="冷却模型" value={String(channelModelCounts.cooling)} hint="每 3 分钟自动检查" icon={ServerCog} tone="amber" />
              <Metric label="错误模型" value={String(channelModelCounts.error)} hint={`${activeOffers} 个模型正在上架`} icon={ServerCog} tone="blue" />
            </div>
            <div className="metrics-grid">
              <Metric label="可提现余额" value={formatQuota(balance?.available_quota)} hint="成功调用后自动入账" icon={WalletCards} />
              <Metric label="冻结余额" value={formatQuota(balance?.frozen_quota)} hint="等待审核或线下打款" icon={Landmark} tone="amber" />
              <Metric label="累计收入" value={formatQuota(balance?.lifetime_earned_quota)} hint="商户获得调用金额的 90%" icon={BadgeDollarSign} tone="green" />
              <Metric label="累计提现" value={formatQuota(balance?.lifetime_paid_quota)} hint="只统计已确认打款" icon={Landmark} tone="blue" />
            </div>
          </>}

          {section === 'channels' && <Section title="渠道管理" description="创建渠道后逐模型检查，只有正常模型进入上架和路由" actions={canSubmit && <Button icon={Plus} variant="primary" onClick={() => openModal({ type: 'channel', onSaved: load })}>提交新渠道</Button>}>
            <div className="table-wrap"><table className="data-table"><thead><tr><th>ID</th><th>渠道</th><th>厂商</th><th>模型</th><th>基础地址</th><th>模型状态</th><th>操作</th></tr></thead><tbody>
              {channels.map((item) => {
                const counts = countModelStates(item.models);
                return <tr key={item.id}><td><code>{item.id}</code></td><td><strong>{item.channel_name}</strong><small>密钥 {item.credential_fingerprint || '-'}</small></td><td>{item.provider || '-'}</td><td><ChannelModelsButton item={item} manage onClick={() => setModelTarget(item.id)} /></td><td><code>{item.base_url}</code></td><td><small>正常 {counts.healthy} · 待检查 {counts.pending} · 冷却 {counts.cooling} · 错误 {counts.error}</small></td><td><ActionMenu items={[{ label: '管理模型', disabled: Boolean(busyChannel), onClick: () => setModelTarget(item.id) }, { label: '同步模型', disabled: Boolean(busyChannel), onClick: () => syncChannel(item) }, { label: '删除', danger: true, disabled: Boolean(busyChannel), onClick: () => deleteChannel(item) }]} /></td></tr>;
              })}
              {!loading && channels.length === 0 && <tr><td className="table-message" colSpan="7">暂无渠道</td></tr>}
              {loading && <tr><td className="table-message" colSpan="7">正在读取渠道</td></tr>}
            </tbody></table></div>
          </Section>}
          {selectedModelRow && <ChannelModelsModal item={selectedModelRow} busyID={busyChannel} onClose={() => setModelTarget(null)} onTest={checkChannelModel} />}

          {section === 'offers' && <Section title="上架模型" description="一个模型统一设置价格，可由多个正常渠道模型承载" actions={canSubmit && <Button disabled={!channels.length || !models.length} icon={BadgeDollarSign} variant="primary" onClick={() => openModal({ type: 'quote', models: models.filter((model) => model.status === 'active'), onSaved: load })}>设置模型价格</Button>}>
            <div className="table-wrap"><table className="data-table"><thead><tr><th>ID</th><th>模型</th><th>价格倍率</th><th>调用次数</th><th>Token 用量</th><th>缓存命中率</th><th>最后调用</th><th>可用渠道</th><th>状态</th><th>健康</th><th>操作</th></tr></thead><tbody>
              {offers.map((offer) => <FragmentOfferRow key={offer.offer_id} offer={offer} expanded={expandedOffer === offer.offer_id} actions={<ActionMenu items={[{ label: '修改模型价格', onClick: () => openModal({ type: 'quote', models, offer, onSaved: load }) }, offer.offer_status === 'active' ? { label: '下架', onClick: () => changeOfferStatus(offer, false) } : { label: '上架', onClick: () => changeOfferStatus(offer, true) }, { label: expandedOffer === offer.offer_id ? '收起渠道状态' : '查看渠道状态', onClick: () => setExpandedOffer(expandedOffer === offer.offer_id ? 0 : offer.offer_id) }, { label: '删除', danger: true, onClick: () => deleteOffer(offer) }]} /> } />)}
              {!loading && offers.length === 0 && <tr><td className="table-message" colSpan="11">暂无上架模型</td></tr>}
            </tbody></table></div>
          </Section>}

          {section === 'usage' && <UsageLogTable view="merchant" data={usage.data} error={usage.error} filters={<div className="filter-toolbar"><Select aria-label="筛选模型" onChange={(event) => { setUsageModel(event.target.value); setUsagePage(1); }} value={usageModel}><option value="all">全部模型</option>{(usage.data?.filters?.models || []).map((model) => <option key={model} value={model}>{model}</option>)}</Select></div>} loading={usage.loading} onRefresh={usage.load} page={usagePage} pageSize={usagePageSize} total={usage.data?.logs?.total} onPageChange={setUsagePage} onPageSizeChange={(nextPageSize) => { setUsagePageSize(nextPageSize); setUsagePage(1); }} />}

          {section === 'withdrawals' && <Section title="提现记录" actions={canSubmit && <Button disabled={Number(balance?.available_quota || 0) < minimumWithdrawalQuota} icon={Landmark} variant="primary" onClick={() => openModal({ type: 'withdraw', balance, onSaved: load })}>申请提现</Button>}>
            {Number(balance?.available_quota || 0) < minimumWithdrawalQuota && <InlineNotice tone="warning">可提现余额满 ¥50.00 后开放申请，还差 {formatQuota(minimumWithdrawalQuota - Number(balance?.available_quota || 0))}。</InlineNotice>}
          <div className="table-wrap"><table className="data-table"><thead><tr><th>ID</th><th>申请号</th><th>申请金额</th><th>税费</th><th>实际到账</th><th>支付宝账户</th><th>状态</th><th>申请时间</th></tr></thead><tbody>
              {withdrawals.map((item) => <tr key={item.id}><td><code>{item.id}</code></td><td><code>{item.withdrawal_no}</code></td><td><strong>{formatQuota(item.amount_quota)}</strong></td><td>{formatQuota(item.tax_quota)}</td><td><strong>{formatQuota(item.net_payout_quota)}</strong></td><td><code>{item.recipient_account_masked}</code></td><td><Status>{withdrawalStatusNames[item.status] || item.status}</Status></td><td>{formatDate(item.created_at)}</td></tr>)}
              {!loading && withdrawals.length === 0 && <tr><td className="table-message" colSpan="8">暂无提现申请</td></tr>}
            </tbody></table></div>
            <Pagination page={withdrawalPage} pageSize={withdrawalPageSize} total={withdrawalTotal} onPageChange={setWithdrawalPage} onPageSizeChange={(nextPageSize) => { setWithdrawalPageSize(nextPageSize); setWithdrawalPage(1); }} />
          </Section>}

          {section === 'settings' && <Section title="商户设置" description="商户名称和网站由审核资料确定，当前仅允许修改商户介绍。" actions={<Button disabled={profileSaving} icon={Settings2} onClick={saveProfile} variant="primary">{profileSaving ? '保存中' : '保存介绍'}</Button>}>
            <div className="merchant-settings-form">
              <div className="form-grid two-column">
                <Field label="商户名称"><TextInput readOnly value={merchant?.display_name || ''} /></Field>
                <Field label="商户网站"><TextInput readOnly value={merchant?.website_url || ''} /></Field>
              </div>
              <Field label="商户介绍" hint={`${businessDescription.length}/2000 字`}><textarea className="textarea" maxLength="2000" onChange={(event) => { setBusinessDescription(event.target.value); setProfileMessage(''); }} rows="7" value={businessDescription} /></Field>
              {profileMessage && <InlineNotice tone={profileMessage === '商户介绍已保存' ? 'success' : 'warning'}>{profileMessage}</InlineNotice>}
            </div>
          </Section>}
        </div>
      </div>
    </div>
  );
}

function FragmentOfferRow({ offer, expanded, actions }) {
  const isImage = Boolean(offer.image_prices?.length);
  return <>
    <tr><td><code>{offer.offer_id}</code></td><td><strong>{offer.model}</strong></td><td><span className="price-multiple">{formatMultiplier(offer.multiplier_bps)}</span></td><td>{formatCompactCount(offer.call_count)}</td><td>{formatCompactCount(offer.token_usage)}</td><td>{formatCacheHit(offer.cache_hit_ppm, offer.cache_weight_ready, isImage)}</td><td>{formatLastCalled(offer.last_called_at)}</td><td>{offer.channel_count || 0} 个</td><td><Status>{offer.offer_status === 'active' ? '已上架' : statusNames[offer.offer_status] || offer.offer_status}</Status></td><td><Status>{healthNames[offer.health_state] || offer.health_state}</Status></td><td>{actions}</td></tr>
    {expanded && <tr className="offer-channel-detail"><td colSpan="11"><div className="table-wrap"><table className="data-table compact-table"><thead><tr><th>渠道 ID</th><th>渠道名称</th><th>状态</th><th>可用率</th><th>P95 首字</th><th>缓存命中率</th></tr></thead><tbody>{(offer.channels || []).map((channel) => <tr key={channel.merchant_channel_id}><td><code>{channel.merchant_channel_id}</code></td><td>{channel.channel_name}</td><td><Status>{healthNames[channel.health_state] || channel.health_state}</Status></td><td>{formatAvailability(channel.availability_ppm)}</td><td>{formatLatency(channel.ttft_p95_ms)}</td><td>{formatCacheHit(channel.cache_hit_ppm, channel.cache_weight_ready, isImage)}</td></tr>)}</tbody></table></div></td></tr>}
  </>;
}
