import { useCallback, useEffect, useState } from 'react';
import {
  Activity,
  ArrowUpRight,
  Coins,
  Eye,
  KeyRound,
  Pencil,
  Plus,
  Route,
  Settings2,
  Store,
  Trash2,
  WalletCards,
} from 'lucide-react';
import { apiRequest, formatQuota } from '../api';
import { Button, CopyValue, EmptyNotice, InlineNotice, Metric, PageHeader, Section, Status } from '../ui';

function displayKey(value) {
  if (!value) return '-';
  return value.startsWith('sk-') ? value : `sk-${value}`;
}

function formatMultiplier(value) {
  return `${(Number(value || 0) / 10_000).toFixed(4)}x`;
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

export default function ConsoleView({ account, openModal, notify, navigate }) {
  const [keys, setKeys] = useState([]);
  const [revealed, setRevealed] = useState({});
  const [keyLoading, setKeyLoading] = useState(true);
  const [keyError, setKeyError] = useState('');
  const [merchants, setMerchants] = useState([]);
  const [merchantApplication, setMerchantApplication] = useState(undefined);
  const [paymentOrders, setPaymentOrders] = useState([]);
  const displayName = account.display_name || account.username;

  const loadKeys = useCallback(async () => {
    setKeyLoading(true);
    setKeyError('');
    try {
      const [data, offerData] = await Promise.all([
        apiRequest('/api/keys?p=1&page_size=50'),
        apiRequest('/api/offers'),
      ]);
      const items = data?.items || [];
      const policies = await Promise.allSettled(items.map((key) => apiRequest(`/api/routing-policies/${key.id}`)));
      setKeys(items.map((key, index) => ({
        ...key,
        routing_policy: policies[index]?.status === 'fulfilled' ? policies[index].value : null,
      })));
      setMerchants(merchantOptions(offerData || []));
    } catch (error) {
      setKeyError(error.message || '读取密钥失败');
    } finally {
      setKeyLoading(false);
    }
  }, []);

  useEffect(() => { loadKeys(); }, [loadKeys]);

  const loadMerchantApplication = useCallback(async () => {
    try {
      setMerchantApplication(await apiRequest('/api/merchant/application'));
    } catch {
      setMerchantApplication(null);
    }
  }, []);

  useEffect(() => { loadMerchantApplication(); }, [loadMerchantApplication]);

  const loadPaymentOrders = useCallback(async () => {
    try {
      const data = await apiRequest('/api/payment-orders?p=1&page_size=10');
      setPaymentOrders(data?.items || []);
    } catch {
      setPaymentOrders([]);
    }
  }, []);

  useEffect(() => { loadPaymentOrders(); }, [loadPaymentOrders]);

  const fetchFullKey = async (key) => {
    if (revealed[key.id]) return revealed[key.id];
    const data = await apiRequest(`/api/keys/${key.id}/reveal`, { method: 'POST' });
    const value = data?.key || data;
    setRevealed((values) => ({ ...values, [key.id]: value }));
    return value;
  };

  const revealKey = async (key) => {
    try {
      await fetchFullKey(key);
      notify('密钥已显示');
    } catch (error) {
      notify(error.message || '读取密钥失败');
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
      const value = await fetchFullKey(key);
      await navigator.clipboard.writeText(displayKey(value));
      notify('密钥已复制');
    } catch (error) {
      notify(error.message || '复制密钥失败');
    }
  };

  return (
    <div className="view-shell">
      <PageHeader
        eyebrow="用户工作台"
        title="账户概览"
        description="查看余额、调用、缓存命中与每个 API 密钥的路由策略。"
        actions={(
          <>
            <Button icon={Settings2}>账单设置</Button>
            <Button icon={Plus} variant="primary" onClick={() => openModal({ type: 'recharge', onSaved: loadPaymentOrders })}>立即充值</Button>
          </>
        )}
      />

      <div className="metrics-grid">
        <Metric label="账户额度" value={formatQuota(account.quota)} hint="New API 权威余额" icon={WalletCards} />
        <Metric label="累计消费" value={formatQuota(account.used_quota)} hint="含缓存计费与退款" icon={Coins} tone="blue" />
        <Metric label="累计请求" value={String(account.request_count || 0)} hint="New API 用户统计" icon={Activity} tone="green" />
        <Metric label="账户分组" value={account.group || 'default'} hint={`UID ${account.id}`} icon={Store} tone="amber" />
      </div>

      <div className="dashboard-grid dashboard-grid-main">
        <Section
          title="用量趋势"
          description="最近 7 天 New API 最终结算额度"
          actions={<select className="mini-select" defaultValue="消费金额"><option>消费金额</option><option>调用次数</option><option>Token 用量</option></select>}
        >
          <EmptyNotice title="暂无可展示的结算趋势" description="模型路由开放后按 New API 最终日志汇总。" />
        </Section>

        <Section title="账户状态" description="统一账号可同时拥有用户与商户身份">
          <div className="account-summary">
            <div className="account-avatar">{displayName.slice(0, 1).toUpperCase()}</div>
            <div><strong>{displayName}</strong><span>UID {account.id} · {Number(account.role) >= 10 ? '管理员账号' : '普通账号'}</span></div>
            <Status>正常</Status>
          </div>
          <div className="account-rows">
            <div><span>账号</span><strong>{account.username}</strong></div>
            <div><span>分组</span><strong>{account.group || 'default'}</strong></div>
            <div><span>账号角色</span><strong>{Number(account.role) >= 10 ? '管理员' : account.is_merchant ? '商户' : '普通用户'}</strong></div>
          </div>
          {(account.is_merchant || Number(account.role) >= 10) && <Button className="full-button" icon={ArrowUpRight} onClick={() => navigate('merchant')}>进入商户中心</Button>}
          {!account.is_merchant && Number(account.role) < 10 && merchantApplication && ['pending', 'verifying', 'verified', 'approving'].includes(merchantApplication.status) && <InlineNotice tone="warning">等待管理员审核 · 申请编号 {merchantApplication.id}</InlineNotice>}
          {!account.is_merchant && Number(account.role) < 10 && (!merchantApplication || merchantApplication.status === 'rejected') && <Button className="full-button" icon={ArrowUpRight} onClick={() => openModal({ type: 'merchantApply', onSaved: loadMerchantApplication })}>{merchantApplication?.status === 'rejected' ? '重新提交商户申请' : '申请成为商户'}</Button>}
          {merchantApplication?.status === 'rejected' && <InlineNotice tone="warning">上次申请已驳回：{merchantApplication.review_reason || '请核对资料后重新提交'}</InlineNotice>}
        </Section>
      </div>

      <Section
        title="API 密钥与路由"
        description="路由策略按 API 密钥独立绑定，最终扣费继续由 New API 处理"
        actions={<Button icon={KeyRound} variant="primary" onClick={() => openModal({ type: 'key', group: account.group, onSaved: loadKeys })}>创建密钥</Button>}
      >
        <InlineNotice>外围平台只在请求进入 New API 前选择商户渠道并替换内部模型别名，不重新计算 Token 或缓存费用。</InlineNotice>
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
              <td><div className="row-actions"><Button icon={Eye} onClick={() => revealKey(key)} variant="ghost">查看</Button><Button icon={Pencil} onClick={() => openModal({ type: 'key', key, group: account.group, onSaved: loadKeys })} variant="ghost">编辑</Button><Button onClick={() => openModal({ type: 'route', key, policy: key.routing_policy, merchants, onSaved: loadKeys })} variant="ghost">编辑路由策略</Button><Button icon={Trash2} onClick={() => deleteKey(key)} variant="danger">删除</Button></div></td>
            </tr>)}
            {!keyLoading && keys.length === 0 && <tr><td className="table-message" colSpan="7">{keyError || '尚未创建 API 密钥'}</td></tr>}
            {keyLoading && <tr><td className="table-message" colSpan="7">正在读取 API 密钥</td></tr>}</tbody>
          </table>
        </div>
      </Section>

      <Section title="充值记录" description="美元额度、人民币实付和入账状态均来自锁定订单">
        <div className="table-wrap"><table className="data-table"><thead><tr><th>订单号</th><th>美元额度</th><th>人民币实付</th><th>支付状态</th><th>额度入账</th><th>创建时间</th></tr></thead><tbody>
          {paymentOrders.map((order) => <tr key={order.id}><td><code>{order.order_no}</code></td><td>${(Number(order.usd_cents || 0) / 100).toFixed(2)}</td><td>¥{(Number(order.cny_cents || 0) / 100).toFixed(2)}</td><td><Status>{{ pending: '待支付', paid: '已支付', closed: '已关闭' }[order.payment_status] || order.payment_status}</Status></td><td><Status>{{ pending: '等待入账', processing: '入账中', succeeded: '已入账', failed: '入账异常', unknown: '结果未知' }[order.credit_status] || order.credit_status}</Status></td><td>{order.created_at ? new Date(order.created_at).toLocaleString('zh-CN', { hour12: false }) : '-'}</td></tr>)}
          {paymentOrders.length === 0 && <tr><td className="table-message" colSpan="6">暂无充值订单</td></tr>}
        </tbody></table></div>
      </Section>

      <Section title="最近调用" description="金额、退款与缓存命中结果来自 New API 最终日志"><EmptyNotice title="暂无调用记录" description="路由开放后显示真实结算日志。" /></Section>
    </div>
  );
}
