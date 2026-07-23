import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Activity,
  ArrowUpRight,
  Coins,
  CreditCard,
  FileText,
  KeyRound,
  LayoutDashboard,
  Plus,
  Route,
  Settings2,
  Store,
  TicketCheck,
  WalletCards,
} from 'lucide-react';
import { apiRequest, formatQuota } from '../api';
import { ActionMenu, Button, CopyValue, Field, InlineNotice, Metric, PageHeader, Pagination, Section, Select, Status, TextInput } from '../ui';
import { UsageLogTable, UsageTrend, useUsage } from './UsagePanels';

const consoleSections = [
  { id: 'overview', label: '账户概览', icon: LayoutDashboard },
  { id: 'routes', label: '路由管理', icon: Route },
  { id: 'keys', label: 'API 密钥', icon: KeyRound },
  { id: 'usage', label: '调用记录', icon: FileText },
  { id: 'recharge', label: '充值中心', icon: CreditCard },
  { id: 'settings', label: '账户设置', icon: Settings2 },
];

function displayKey(value) {
  if (!value) return '-';
  return value.startsWith('sk-') ? value : `sk-${value}`;
}

function formatMultiplier(value) {
  return `${(Number(value || 0) / 10_000).toFixed(2)}x`;
}

function priorityName(value) {
  return { balanced: '均衡分配', price: '价格优先', success: '成功率优先', latency: '响应速度优先', cache: '缓存命中率优先' }[value] || value;
}

function cacheFloorName(value) {
  return value == null ? '未启用' : `${(Number(value) / 10_000).toFixed(2)}%`;
}

export default function ConsoleView({ account, openModal, notify, navigate, onAccountRefresh }) {
  const [section, setSection] = useState('overview');
  const [keys, setKeys] = useState([]);
  const [keyLoading, setKeyLoading] = useState(true);
  const [keyError, setKeyError] = useState('');
  const [routingPolicies, setRoutingPolicies] = useState([]);
  const [models, setModels] = useState([]);
  const [merchantApplication, setMerchantApplication] = useState(undefined);
  const [paymentOrders, setPaymentOrders] = useState([]);
  const [paymentPage, setPaymentPage] = useState(1);
  const [paymentPageSize, setPaymentPageSize] = useState(20);
  const [paymentTotal, setPaymentTotal] = useState(0);
  const [usagePage, setUsagePage] = useState(1);
  const [usagePageSize, setUsagePageSize] = useState(20);
  const [usageModel, setUsageModel] = useState('all');
  const [usageMerchant, setUsageMerchant] = useState('');
  const [redemptionCode, setRedemptionCode] = useState('');
  const [redeeming, setRedeeming] = useState(false);
  const usageEndpoint = useMemo(() => {
    const query = new URLSearchParams({ p: String(usagePage), page_size: String(usagePageSize) });
    if (usageModel !== 'all') query.set('model_name', usageModel);
    const merchantMatch = usageMerchant.trim().match(/(?:^|用户 ID\s*)(\d+)$/);
    if (merchantMatch) query.set('merchant_user_id', merchantMatch[1]);
    return `/api/usage?${query}`;
  }, [usageMerchant, usageModel, usagePage, usagePageSize]);
  const usage = useUsage(usageEndpoint);
  const displayName = account.display_name || account.username;

  const loadKeys = useCallback(async () => {
    setKeyLoading(true);
    setKeyError('');
    try {
      const [data, policyData] = await Promise.all([
        apiRequest('/api/keys?p=1&page_size=50'),
        apiRequest('/api/routing-policies'),
      ]);
      setKeys(data?.items || []);
      setRoutingPolicies(policyData || []);
    } catch (error) {
      setKeyError(error.message || '读取密钥失败');
    } finally {
      setKeyLoading(false);
    }
  }, []);

  const loadMerchantApplication = useCallback(async () => {
    try {
      setMerchantApplication(await apiRequest('/api/merchant/application'));
    } catch {
      setMerchantApplication(null);
    }
  }, []);

  const loadModels = useCallback(async () => {
    try {
      setModels(await apiRequest('/api/models') || []);
    } catch {
      setModels([]);
    }
  }, []);

  const loadPaymentOrders = useCallback(async () => {
    try {
      const data = await apiRequest(`/api/payment-orders?p=${paymentPage}&page_size=${paymentPageSize}`);
      setPaymentOrders(data?.items || []);
      setPaymentTotal(Number(data?.total || 0));
    } catch {
      setPaymentOrders([]);
      setPaymentTotal(0);
    }
  }, [paymentPage, paymentPageSize]);

  useEffect(() => { loadKeys(); }, [loadKeys]);
  useEffect(() => { loadMerchantApplication(); }, [loadMerchantApplication]);
  useEffect(() => { loadModels(); }, [loadModels]);
  useEffect(() => { loadPaymentOrders(); }, [loadPaymentOrders]);

  const redeem = async (event) => {
    event.preventDefault();
    const key = redemptionCode.trim();
    if (!key) return;
    setRedeeming(true);
    try {
      await apiRequest('/api/redemptions/redeem', { method: 'POST', body: { key } });
      setRedemptionCode('');
      notify('兑换码充值成功');
      await onAccountRefresh?.();
    } catch (error) {
      notify(error.message || '兑换码充值失败');
    } finally {
      setRedeeming(false);
    }
  };

  const deleteKey = async (key) => {
    if (!window.confirm(`确认删除密钥“${key.name}”吗？`)) return;
    try {
      await apiRequest(`/api/keys/${key.id}`, { method: 'DELETE' });
      notify('密钥已删除');
      loadKeys();
    } catch (error) {
      notify(error.message || '删除密钥失败');
    }
  };

  const setKeyStatus = async (key, enabled) => {
    try {
      await apiRequest(`/api/keys/${key.id}/status`, { method: 'POST', body: { enabled } });
      notify(enabled ? '密钥已启用' : '密钥已停用');
      loadKeys();
    } catch (error) {
      notify(error.message || '修改密钥状态失败');
    }
  };

  const copyKey = async (key) => {
    try {
      await navigator.clipboard.writeText(displayKey(key.key));
      notify('密钥已复制');
    } catch (error) {
      notify(error.message || '复制密钥失败');
    }
  };

  const deleteRoute = async (policy) => {
    if (!window.confirm(`确认删除路由“${policy.name}”吗？`)) return;
    try {
      await apiRequest(`/api/routing-policies/${policy.id}`, { method: 'DELETE' });
      notify('路由已删除');
      await loadKeys();
    } catch (error) {
      notify(error.message || '删除路由失败');
    }
  };

  const currentSection = consoleSections.find((item) => item.id === section) || consoleSections[0];
  const accountTypeName = { user: '普通用户', merchant: '商户', admin: '管理员' }[account.account_type]
    || (Number(account.role) >= 10 ? '管理员' : account.is_merchant ? '商户' : '普通用户');

  return (
    <div className="view-shell console-view">
      <PageHeader
        eyebrow="用户工作台"
        title={currentSection.label}
        description="管理账户、API 密钥、调用记录与充值。"
      />

      <div className="admin-layout console-layout">
        <aside className="admin-sidebar" aria-label="控制台功能">
          {consoleSections.map(({ id, label, icon: Icon }) => (
            <button aria-label={label} className={section === id ? 'active' : ''} key={id} onClick={() => setSection(id)} type="button"><Icon size={17} /><span>{label}</span></button>
          ))}
        </aside>

        <div className="admin-content">
          {section === 'overview' && <>
            <div className="metrics-grid">
              <Metric label="账户余额" value={formatQuota(account.quota)} hint={`用户 ID ${account.id}`} icon={WalletCards} />
              <Metric label="累计消费" value={formatQuota(account.used_quota)} hint="包含缓存计费" icon={Coins} tone="blue" />
              <Metric label="累计请求" value={String(account.request_count || 0)} hint="全部 API 密钥" icon={Activity} tone="green" />
              <Metric label="账号身份" value={accountTypeName} hint="当前有效身份" icon={Store} tone="amber" />
            </div>
            <UsageTrend data={usage.data} error={usage.error} loading={usage.loading} />
          </>}

          {section === 'routes' && <Section title="路由管理" description="路由可复用，并从全部合格商户中按价格、成功率、速度或缓存指标加权选择" actions={<Button icon={Plus} variant="primary" onClick={() => openModal({ type: 'route', onSaved: loadKeys })}>创建路由</Button>}>
            <div className="table-wrap">
              <table className="data-table">
                <thead><tr><th>ID</th><th>名称</th><th>优先方式</th><th>倍率区间</th><th>最低缓存</th><th>绑定密钥</th><th>操作</th></tr></thead>
                <tbody>{routingPolicies.map((policy) => <tr key={policy.id}>
                  <td><code>{policy.id}</code></td><td><strong>{policy.name}</strong></td><td>{priorityName(policy.priority)}</td><td>{formatMultiplier(policy.minimum_multiplier_bps)} - {formatMultiplier(policy.maximum_multiplier_bps)}</td><td>{cacheFloorName(policy.minimum_cache_hit_ppm)}</td><td>{policy.bound_key_count || 0} 把</td><td><ActionMenu items={[{ label: '编辑', onClick: () => openModal({ type: 'route', policy, onSaved: loadKeys }) }, { label: '删除', danger: true, disabled: Number(policy.bound_key_count || 0) > 0, onClick: () => deleteRoute(policy) }]} /></td>
                </tr>)}
                {!keyLoading && routingPolicies.length === 0 && <tr><td className="table-message" colSpan="7">{keyError || '尚未创建路由'}</td></tr>}
                {keyLoading && <tr><td className="table-message" colSpan="7">正在读取路由</td></tr>}</tbody>
              </table>
            </div>
          </Section>}

          {section === 'keys' && <Section title="API 密钥" description="创建密钥前需要先创建路由" actions={<Button disabled={!routingPolicies.length} icon={KeyRound} variant="primary" onClick={() => openModal({ type: 'key', routingPolicies, onSaved: loadKeys })}>创建密钥</Button>}>
            {!routingPolicies.length && !keyLoading && <InlineNotice tone="warning">请先在“路由管理”中创建一条路由。</InlineNotice>}
            <div className="table-wrap"><table className="data-table"><thead><tr><th>ID</th><th>名称 / 密钥</th><th>绑定路由</th><th>额度</th><th>状态</th><th>操作</th></tr></thead><tbody>
              {keys.map((key) => <tr key={key.id}><td><code>{key.id}</code></td><td><strong>{key.name}</strong><CopyValue onCopy={() => copyKey(key)}>{displayKey(key.key)}</CopyValue></td><td><span className="route-kind"><Route size={15} /> {key.routing_policy_name || '未绑定'}</span></td><td>{key.unlimited_quota ? '不限额' : formatQuota(key.remain_quota)}</td><td><div className="status-control"><Status>{Number(key.status) === 1 ? '正常' : '停用'}</Status><label className="compact-switch" title={Number(key.status) === 1 ? '停用密钥' : '启用密钥'}><input aria-label={`启用${key.name}`} checked={Number(key.status) === 1} onChange={(event) => setKeyStatus(key, event.target.checked)} type="checkbox" /><span /></label></div></td><td><ActionMenu items={[{ label: '编辑', onClick: () => openModal({ type: 'key', key, routingPolicies, onSaved: loadKeys }) }, { label: '删除', danger: true, onClick: () => deleteKey(key) }]} /></td></tr>)}
              {!keyLoading && keys.length === 0 && <tr><td className="table-message" colSpan="6">{keyError || '尚未创建 API 密钥'}</td></tr>}
              {keyLoading && <tr><td className="table-message" colSpan="6">正在读取 API 密钥</td></tr>}
            </tbody></table></div>
          </Section>}

          {section === 'usage' && <UsageLogTable
            data={usage.data}
            error={usage.error}
            filters={<div className="filter-toolbar">
              <Select aria-label="筛选模型" onChange={(event) => { setUsageModel(event.target.value); setUsagePage(1); }} value={usageModel}><option value="all">全部模型</option>{(usage.data?.filters?.models || []).map((model) => <option key={model} value={model}>{model}</option>)}</Select>
              <div className="filter-search"><TextInput aria-label="搜索商户" list="usage-merchant-options" onChange={(event) => { setUsageMerchant(event.target.value); setUsagePage(1); }} placeholder="输入商户名称或用户 ID" value={usageMerchant} /><datalist id="usage-merchant-options">{(usage.data?.filters?.merchants || []).map((item) => <option key={item.user_id} value={`${item.name} · 用户 ID ${item.user_id}`} />)}</datalist></div>
            </div>}
            loading={usage.loading}
            onRefresh={usage.load}
            page={usagePage}
            pageSize={usagePageSize}
            total={usage.data?.logs?.total}
            onPageChange={setUsagePage}
            onPageSizeChange={(nextPageSize) => { setUsagePageSize(nextPageSize); setUsagePage(1); }}
          />}

          {section === 'recharge' && <>
            <Section title="账户充值" actions={<Button icon={Plus} variant="primary" onClick={() => openModal({ type: 'recharge', onSaved: loadPaymentOrders })}>立即充值</Button>}>
              <form className="redemption-form" onSubmit={redeem}>
                <Field label="兑换码"><TextInput autoComplete="off" maxLength="128" onChange={(event) => setRedemptionCode(event.target.value)} placeholder="输入兑换码" value={redemptionCode} /></Field>
                <Button disabled={redeeming || !redemptionCode.trim()} icon={TicketCheck} type="submit" variant="primary">{redeeming ? '正在兑换' : '立即兑换'}</Button>
              </form>
            </Section>
            <Section title="充值记录">
              <div className="table-wrap"><table className="data-table"><thead><tr><th>ID</th><th>订单号</th><th>人民币金额</th><th>入账额度</th><th>支付状态</th><th>额度入账</th><th>创建时间</th></tr></thead><tbody>
                {paymentOrders.map((order) => <tr key={order.id}><td><code>{order.id}</code></td><td><code>{order.order_no}</code></td><td>¥{(Number(order.amount_cny_cents || 0) / 100).toFixed(2)}</td><td>{formatQuota(order.newapi_quota)}</td><td><Status>{{ pending: '待支付', paid: '已支付', closed: '已关闭' }[order.payment_status] || order.payment_status}</Status></td><td><Status>{{ pending: '等待入账', processing: '入账中', succeeded: '已入账', failed: '入账异常', unknown: '结果未知' }[order.credit_status] || order.credit_status}</Status></td><td>{order.created_at ? new Date(order.created_at).toLocaleString('zh-CN', { hour12: false }) : '-'}</td></tr>)}
                {paymentOrders.length === 0 && <tr><td className="table-message" colSpan="7">暂无充值订单</td></tr>}
              </tbody></table></div>
              <Pagination page={paymentPage} pageSize={paymentPageSize} total={paymentTotal} onPageChange={setPaymentPage} onPageSizeChange={(nextPageSize) => { setPaymentPageSize(nextPageSize); setPaymentPage(1); }} />
            </Section>
          </>}

          {section === 'settings' && <Section className="account-settings-section" title="账户设置" description="查看当前账号资料和已开通的工作区">
            <div className="account-settings-content">
              <div className="account-identity">
                <div className="account-identity-header">
                  <div className="account-avatar">{displayName.slice(0, 1).toUpperCase()}</div>
                  <div className="account-identity-copy"><strong>{displayName}</strong><span>用户 ID {account.id} · {accountTypeName}</span></div>
                  <Status>正常</Status>
                </div>
                <div className="account-workspace"><span>当前工作区</span><strong>{account.is_merchant || Number(account.role) >= 10 ? '商户工作区' : '用户工作台'}</strong><small>{accountTypeName} · 可用功能按账号身份显示</small></div>
                <div className="account-actions">
                  {(account.is_merchant || Number(account.role) >= 10) && <Button icon={ArrowUpRight} onClick={() => navigate('merchant')} variant="primary">进入商户中心</Button>}
                  {!account.is_merchant && Number(account.role) < 10 && (!merchantApplication || merchantApplication.status === 'rejected') && <Button icon={ArrowUpRight} onClick={() => openModal({ type: 'merchantApply', models: models.filter((model) => model.status === 'active'), onSaved: loadMerchantApplication })}>{merchantApplication?.status === 'rejected' ? '重新提交商户申请' : '申请成为商户'}</Button>}
                </div>
              </div>
              <div className="account-details">
                <div className="account-details-heading"><h3>账号信息</h3><span>登录凭据和账号身份</span></div>
                <div className="account-rows"><div><span>登录账号</span><strong>{account.username}</strong></div><div><span>邮箱</span><strong>{account.email || '-'}</strong></div><div><span>用户类型</span><strong>{accountTypeName}</strong></div></div>
                {!account.is_merchant && Number(account.role) < 10 && merchantApplication && ['pending', 'verifying', 'verified', 'approving'].includes(merchantApplication.status) && <InlineNotice tone="warning">等待管理员审核 · 申请 ID {merchantApplication.id}</InlineNotice>}
                {merchantApplication?.status === 'rejected' && <InlineNotice tone="warning">上次申请未通过：{merchantApplication.review_reason || '请核对资料后重新提交'}</InlineNotice>}
              </div>
            </div>
          </Section>}
        </div>
      </div>
    </div>
  );
}
