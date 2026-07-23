import { useCallback, useEffect, useState } from 'react';
import {
  Activity,
  ArrowUpRight,
  Coins,
  CreditCard,
  FileText,
  KeyRound,
  LayoutDashboard,
  Pencil,
  Plus,
  Route,
  Settings2,
  Store,
  TicketCheck,
  Trash2,
  WalletCards,
} from 'lucide-react';
import { apiRequest, formatQuota } from '../api';
import { Button, CopyValue, Field, InlineNotice, Metric, PageHeader, Section, Status, TextInput } from '../ui';
import { UsageLogTable, UsageTrend, useUsage } from './UsagePanels';

const consoleSections = [
  { id: 'overview', label: '账户概览', icon: LayoutDashboard },
  { id: 'keys', label: 'API 密钥与路由', icon: KeyRound },
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

function merchantOptions(offers) {
  const merchants = new Map();
  for (const offer of offers) {
    const current = merchants.get(offer.merchant_id);
    if (!current) {
      merchants.set(offer.merchant_id, {
        id: offer.merchant_id,
        name: offer.merchant_name,
        availability_ppm: offer.availability_ppm,
        minimum_multiplier_bps: offer.multiplier_bps,
      });
      continue;
    }
    current.availability_ppm = Math.max(current.availability_ppm, offer.availability_ppm);
    current.minimum_multiplier_bps = Math.min(current.minimum_multiplier_bps, offer.multiplier_bps);
  }
  return [...merchants.values()].sort((left, right) => left.name.localeCompare(right.name, 'zh-CN'));
}

export default function ConsoleView({ account, openModal, notify, navigate, onAccountRefresh }) {
  const [section, setSection] = useState('overview');
  const [keys, setKeys] = useState([]);
  const [revealed, setRevealed] = useState({});
  const [keyLoading, setKeyLoading] = useState(true);
  const [keyError, setKeyError] = useState('');
  const [merchants, setMerchants] = useState([]);
  const [models, setModels] = useState([]);
  const [merchantApplication, setMerchantApplication] = useState(undefined);
  const [paymentOrders, setPaymentOrders] = useState([]);
  const [redemptionCode, setRedemptionCode] = useState('');
  const [redeeming, setRedeeming] = useState(false);
  const usage = useUsage('/api/usage');
  const displayName = account.display_name || account.username;

  const loadKeys = useCallback(async () => {
    setKeyLoading(true);
    setKeyError('');
    try {
      const [data, offerData, modelData] = await Promise.all([
        apiRequest('/api/keys?p=1&page_size=50'),
        apiRequest('/api/offers'),
        apiRequest('/api/models'),
      ]);
      const items = data?.items || [];
      const policies = await Promise.allSettled(items.map((key) => apiRequest(`/api/routing-policies/${key.id}`)));
      setKeys(items.map((key, index) => ({
        ...key,
        routing_policy: policies[index]?.status === 'fulfilled' ? policies[index].value : null,
      })));
      setMerchants(merchantOptions(offerData || []));
      setModels(modelData || []);
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

  const loadPaymentOrders = useCallback(async () => {
    try {
      const data = await apiRequest('/api/payment-orders?p=1&page_size=10');
      setPaymentOrders(data?.items || []);
    } catch {
      setPaymentOrders([]);
    }
  }, []);

  useEffect(() => { loadKeys(); }, [loadKeys]);
  useEffect(() => { loadMerchantApplication(); }, [loadMerchantApplication]);
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

  const fetchFullKey = async (key) => {
    if (revealed[key.id]) return revealed[key.id];
    const data = await apiRequest(`/api/keys/${key.id}/reveal`, { method: 'POST' });
    const value = data?.key || data;
    setRevealed((values) => ({ ...values, [key.id]: value }));
    return value;
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
      const value = await fetchFullKey(key);
      await navigator.clipboard.writeText(displayKey(value));
      notify('密钥已复制');
    } catch (error) {
      notify(error.message || '复制密钥失败');
    }
  };

  const currentSection = consoleSections.find((item) => item.id === section) || consoleSections[0];

  return (
    <div className="view-shell console-view">
      <PageHeader
        eyebrow="用户工作台"
        title={currentSection.label}
        description="管理账户、API 密钥、调用记录与充值。"
        actions={<><Button icon={Settings2} onClick={() => setSection('settings')}>账户设置</Button><Button icon={Plus} variant="primary" onClick={() => openModal({ type: 'recharge', onSaved: loadPaymentOrders })}>立即充值</Button></>}
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
              <Metric label="账户余额" value={formatQuota(account.quota)} hint={`UID ${account.id}`} icon={WalletCards} />
              <Metric label="累计消费" value={formatQuota(account.used_quota)} hint="含缓存计费与退款" icon={Coins} tone="blue" />
              <Metric label="累计请求" value={String(account.request_count || 0)} hint="全部 API 密钥" icon={Activity} tone="green" />
              <Metric label="账号身份" value={Number(account.role) >= 10 ? '管理员' : account.is_merchant ? '商户' : '普通用户'} hint="当前有效身份" icon={Store} tone="amber" />
            </div>
            <UsageTrend data={usage.data} error={usage.error} loading={usage.loading} />
          </>}

          {section === 'keys' && <Section title="API 密钥与路由" description="每个 API 密钥独立设置路由策略" actions={<Button icon={KeyRound} variant="primary" onClick={() => openModal({ type: 'key', onSaved: loadKeys })}>创建密钥</Button>}>
            <div className="table-wrap">
              <table className="data-table">
                <thead><tr><th>名称 / 密钥</th><th>路由方式</th><th>倍率区间</th><th>优先规则</th><th>固定商户</th><th>状态</th><th></th></tr></thead>
                <tbody>{keys.map((key) => <tr key={key.id}>
                  <td><strong>{key.name}</strong><CopyValue onCopy={() => copyKey(key)}>{displayKey(revealed[key.id] || key.key)}</CopyValue></td>
                  <td><span className="route-kind"><Route size={15} /> {key.routing_policy?.mode === 'fixed' ? '固定商户' : key.routing_policy ? '智能路由' : '未绑定'}</span></td>
                  <td>{key.routing_policy ? `${formatMultiplier(key.routing_policy.minimum_multiplier_bps)} - ${formatMultiplier(key.routing_policy.maximum_multiplier_bps)}` : '-'}</td>
                  <td>{key.routing_policy?.priority === 'price' ? '价格优先' : key.routing_policy ? '稳定性优先' : '-'}</td>
                  <td>{key.routing_policy?.mode === 'fixed' ? `${key.routing_policy.merchant_ids?.length || 0} 家` : '全部合格商户'}</td>
                  <td><div className="status-control"><Status>{Number(key.status) === 1 ? '正常' : '停用'}</Status><label className="compact-switch" title={Number(key.status) === 1 ? '停用密钥' : '启用密钥'}><input aria-label={`启用${key.name}`} checked={Number(key.status) === 1} onChange={(event) => setKeyStatus(key, event.target.checked)} type="checkbox" /><span /></label></div></td>
                  <td><div className="row-actions"><Button icon={Pencil} onClick={() => openModal({ type: 'key', key, onSaved: loadKeys })} variant="ghost">编辑</Button><Button onClick={() => openModal({ type: 'route', key, policy: key.routing_policy, merchants, onSaved: loadKeys })} variant="ghost">编辑路由策略</Button><Button icon={Trash2} onClick={() => deleteKey(key)} variant="danger">删除</Button></div></td>
                </tr>)}
                {!keyLoading && keys.length === 0 && <tr><td className="table-message" colSpan="7">{keyError || '尚未创建 API 密钥'}</td></tr>}
                {keyLoading && <tr><td className="table-message" colSpan="7">正在读取 API 密钥</td></tr>}</tbody>
              </table>
            </div>
          </Section>}

          {section === 'usage' && <><UsageTrend data={usage.data} error={usage.error} loading={usage.loading} /><UsageLogTable data={usage.data} error={usage.error} loading={usage.loading} onRefresh={usage.load} /></>}

          {section === 'recharge' && <>
            <Section title="账户充值" actions={<Button icon={Plus} variant="primary" onClick={() => openModal({ type: 'recharge', onSaved: loadPaymentOrders })}>立即充值</Button>}>
              <form className="redemption-form" onSubmit={redeem}>
                <Field label="兑换码"><TextInput autoComplete="off" maxLength="128" onChange={(event) => setRedemptionCode(event.target.value)} placeholder="输入兑换码" value={redemptionCode} /></Field>
                <Button disabled={redeeming || !redemptionCode.trim()} icon={TicketCheck} type="submit" variant="primary">{redeeming ? '正在兑换' : '立即兑换'}</Button>
              </form>
            </Section>
            <Section title="充值记录">
              <div className="table-wrap"><table className="data-table"><thead><tr><th>订单号</th><th>人民币金额</th><th>入账额度</th><th>支付状态</th><th>额度入账</th><th>创建时间</th></tr></thead><tbody>
                {paymentOrders.map((order) => <tr key={order.id}><td><code>{order.order_no}</code></td><td>¥{(Number(order.amount_cny_cents || 0) / 100).toFixed(2)}</td><td>{formatQuota(order.newapi_quota)}</td><td><Status>{{ pending: '待支付', paid: '已支付', closed: '已关闭' }[order.payment_status] || order.payment_status}</Status></td><td><Status>{{ pending: '等待入账', processing: '入账中', succeeded: '已入账', failed: '入账异常', unknown: '结果未知' }[order.credit_status] || order.credit_status}</Status></td><td>{order.created_at ? new Date(order.created_at).toLocaleString('zh-CN', { hour12: false }) : '-'}</td></tr>)}
                {paymentOrders.length === 0 && <tr><td className="table-message" colSpan="6">暂无充值订单</td></tr>}
              </tbody></table></div>
            </Section>
          </>}

          {section === 'settings' && <Section title="账户设置">
            <div className="account-summary">
              <div className="account-avatar">{displayName.slice(0, 1).toUpperCase()}</div>
              <div><strong>{displayName}</strong><span>UID {account.id} · {Number(account.role) >= 10 ? '管理员账号' : '普通账号'}</span></div>
              <Status>正常</Status>
            </div>
            <div className="account-rows">
              <div><span>账号</span><strong>{account.username}</strong></div>
              <div><span>账号角色</span><strong>{Number(account.role) >= 10 ? '管理员' : account.is_merchant ? '商户' : '普通用户'}</strong></div>
            </div>
            {(account.is_merchant || Number(account.role) >= 10) && <Button className="full-button" icon={ArrowUpRight} onClick={() => navigate('merchant')}>进入商户中心</Button>}
            {!account.is_merchant && Number(account.role) < 10 && merchantApplication && ['pending', 'verifying', 'verified', 'approving'].includes(merchantApplication.status) && <InlineNotice tone="warning">等待管理员审核 · 申请编号 {merchantApplication.id}</InlineNotice>}
            {!account.is_merchant && Number(account.role) < 10 && (!merchantApplication || merchantApplication.status === 'rejected') && <Button className="full-button" icon={ArrowUpRight} onClick={() => openModal({ type: 'merchantApply', models, onSaved: loadMerchantApplication })}>{merchantApplication?.status === 'rejected' ? '重新提交商户申请' : '申请成为商户'}</Button>}
            {merchantApplication?.status === 'rejected' && <InlineNotice tone="warning">上次申请已驳回：{merchantApplication.review_reason || '请核对资料后重新提交'}</InlineNotice>}
          </Section>}
        </div>
      </div>
    </div>
  );
}
