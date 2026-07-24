import { useCallback, useEffect, useState } from 'react';
import {
  CheckCircle2,
  KeyRound,
  Pencil,
  RefreshCw,
  RotateCcw,
  WalletCards,
} from 'lucide-react';
import { apiRequest, formatQuota } from '../../api';
import { ActionMenu, Button, InlineNotice, Pagination, Section, Status } from '../../ui';

const paymentMethodNames = { alipay: '支付宝', wxpay: '微信', qqpay: 'QQ 钱包' };
const paymentStatusNames = { pending: '待支付', paid: '已支付', closed: '已关闭' };
const creditStatusNames = { pending: '等待入账', processing: '入账中', succeeded: '已入账', failed: '入账异常', unknown: '结果未知' };

function formatCNYCents(value) {
  return new Intl.NumberFormat('zh-CN', { style: 'currency', currency: 'CNY' }).format(Number(value || 0) / 100);
}

function formatTime(value) {
  if (!value) return '-';
  return new Intl.DateTimeFormat('zh-CN', { dateStyle: 'short', timeStyle: 'medium', hour12: false }).format(new Date(value));
}

export default function AdminPayment({ notify, openModal }) {
  const [setting, setSetting] = useState(null);
  const [orders, setOrders] = useState([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [settingData, orderData] = await Promise.all([
        apiRequest('/api/admin/payment-settings'),
        apiRequest(`/api/admin/payment-orders?p=${page}&page_size=${pageSize}`),
      ]);
      setSetting(settingData);
      setOrders(orderData?.items || []);
      setTotal(Number(orderData?.total || 0));
    } catch (requestError) {
      setError(requestError.message || '读取支付数据失败');
    } finally {
      setLoading(false);
    }
  }, [page, pageSize]);

  useEffect(() => { load(); }, [load]);

  const retryCredit = async (order) => {
    try {
      await apiRequest(`/api/admin/payment-orders/${order.id}/credit/retry`, { method: 'POST', body: {} });
      notify('入账任务已重新排队');
      await load();
    } catch (requestError) {
      notify(requestError.message || '重新排队失败');
    }
  };

  return (
    <div className="admin-payment">
      <Section
        title="易支付设置"
      description="统一管理充值订单、签名、回调验签和额度入账"
        actions={<><Button disabled={loading} icon={RefreshCw} onClick={load}>刷新</Button><Button icon={Pencil} variant="primary" onClick={() => openModal({ type: 'payment', setting, onSaved: load })}>编辑配置</Button></>}
      >
        {error && <InlineNotice tone="danger">{error}</InlineNotice>}
        {setting && <div className="payment-config">
          <div className="payment-state"><span className="payment-mark"><WalletCards size={24} /></span><div><strong>易支付 · 当前配置</strong><span>配置版本 v{setting.version} · 更新于 {formatTime(setting.created_at)}</span></div><Status>{setting.enabled ? '已开通' : '停用'}</Status></div>
          <div className="config-grid">
            <div><span>网关地址</span><strong>{setting.gateway_url}</strong></div>
            <div><span>商户编号 PID</span><strong>{setting.merchant_pid_masked}</strong></div>
            <div><span>商户密钥 KEY</span><strong>{setting.key_configured ? '已配置 · 不回显' : '未配置'}</strong></div>
            <div><span>支付方式</span><strong>{setting.payment_methods.map((method) => paymentMethodNames[method] || method).join('、')}</strong></div>
            <div><span>人民币充值范围</span><strong>{formatCNYCents(setting.minimum_cny_cents)} - {formatCNYCents(setting.maximum_cny_cents)}</strong></div>
            <div><span>提现税率</span><strong>{(Number(setting.withdrawal_tax_bps || 0) / 100).toFixed(2)}%</strong></div>
            <div><span>异步回调</span><strong>{setting.notify_url}</strong></div>
            <div><span>浏览器返回页</span><strong>{setting.return_url}</strong></div>
          </div>
          <div className="config-actions">
            <Button icon={KeyRound} onClick={() => openModal({ type: 'payment', setting, onSaved: load })}>轮换密钥</Button>
          </div>
        </div>}
        {!setting && !loading && !error && <InlineNotice tone="warning">尚未创建支付配置</InlineNotice>}
        <InlineNotice tone="success"><CheckCircle2 size={16} />完整 KEY 只在服务端解密用于签名；配置页只显示状态和 PID 掩码。</InlineNotice>
      </Section>

      <Section
        title="充值订单与入账"
      description="支付成功与额度入账分别使用幂等键；结果未知的操作必须先人工核对"
        actions={<Button disabled={loading} icon={RefreshCw} onClick={load}>刷新订单</Button>}
      >
        <div className="table-wrap"><table className="data-table"><thead><tr><th>ID</th><th>平台订单号</th><th>用户</th><th>人民币金额</th><th>入账额度</th><th>方式</th><th>第三方交易号</th><th>支付时间</th><th>支付状态</th><th>额度入账</th><th>操作</th></tr></thead><tbody>
          {orders.map((order) => <tr key={order.id}><td><code>{order.id}</code></td><td><code>{order.order_no}</code></td><td>用户 ID {order.user_id}</td><td><strong>{formatCNYCents(order.amount_cny_cents)}</strong></td><td>{formatQuota(order.newapi_quota)}</td><td>{paymentMethodNames[order.payment_type] || order.payment_type}</td><td>{order.third_party_trade_no || '-'}</td><td>{formatTime(order.paid_at)}</td><td><Status>{paymentStatusNames[order.payment_status] || order.payment_status}</Status></td><td><Status>{creditStatusNames[order.credit_status] || order.credit_status}</Status></td><td><ActionMenu items={[order.credit_status === 'failed' && { label: '重试入账', icon: RotateCcw, onClick: () => retryCredit(order) }, order.credit_status === 'unknown' && { label: '确认已入账', onClick: () => openModal({ type: 'creditResolve', action: 'credited', order, onSaved: load }) }, order.credit_status === 'unknown' && { label: '核对后重试', icon: RotateCcw, onClick: () => openModal({ type: 'creditResolve', action: 'retry', order, onSaved: load }) }]} /></td></tr>)}
          {!loading && orders.length === 0 && <tr><td className="table-message" colSpan="11">暂无充值订单</td></tr>}
          {loading && <tr><td className="table-message" colSpan="11">正在读取充值订单</td></tr>}
        </tbody></table></div>
        <Pagination page={page} pageSize={pageSize} total={total} onPageChange={setPage} onPageSizeChange={(nextPageSize) => { setPageSize(nextPageSize); setPage(1); }} disabled={loading} />
      </Section>
    </div>
  );
}
