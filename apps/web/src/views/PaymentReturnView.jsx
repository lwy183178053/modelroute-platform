import { useCallback, useEffect, useState } from 'react';
import { ArrowLeft, CreditCard, RefreshCw, WalletCards } from 'lucide-react';
import { apiRequest } from '../api';
import { Button, InlineNotice, Metric, PageHeader, Section, Status } from '../ui';

const paymentNames = { pending: '待支付', paid: '已支付', closed: '已关闭' };
const creditNames = { pending: '等待入账', processing: '入账中', succeeded: '已入账', failed: '入账异常', unknown: '结果未知' };

export default function PaymentReturnView({ navigate }) {
  const orderNo = new URLSearchParams(window.location.search).get('out_trade_no') || '';
  const [order, setOrder] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const load = useCallback(async () => {
    if (!orderNo) {
      setError('支付返回地址缺少平台订单号');
      setLoading(false);
      return;
    }
    setLoading(true);
    setError('');
    try {
      setOrder(await apiRequest(`/api/payment-orders/${encodeURIComponent(orderNo)}`));
    } catch (requestError) {
      setError(requestError.message || '读取支付结果失败');
    } finally {
      setLoading(false);
    }
  }, [orderNo]);

  useEffect(() => { load(); }, [load]);

  return <div className="view-shell payment-return-view">
    <PageHeader eyebrow="账户充值" title="支付结果" description="页面仅查询订单状态；额度入账由易支付异步通知验签后触发。" actions={<Button icon={ArrowLeft} onClick={() => navigate('console')}>返回控制台</Button>} />
    {error && <InlineNotice tone="danger">{error}</InlineNotice>}
    {loading && <InlineNotice>正在读取平台订单状态</InlineNotice>}
    {order && <>
      <div className="metrics-grid">
        <Metric label="人民币金额" value={`¥${(Number(order.amount_cny_cents || 0) / 100).toFixed(2)}`} hint="订单创建时锁定" icon={CreditCard} />
        <Metric label="入账额度" value={`¥${(Number(order.newapi_quota || 0) / 500_000).toFixed(2)}`} hint="支付成功后入账" icon={WalletCards} tone="blue" />
      </div>
      <Section title="订单状态" description={`平台订单号 ${order.order_no}`} actions={<Button icon={RefreshCw} onClick={load}>刷新状态</Button>}>
        <div className="account-rows">
          <div><span>支付状态</span><Status>{paymentNames[order.payment_status] || order.payment_status}</Status></div>
          <div><span>额度入账</span><Status>{creditNames[order.credit_status] || order.credit_status}</Status></div>
          <div><span>第三方交易号</span><strong>{order.third_party_trade_no || '-'}</strong></div>
          <div><span>支付时间</span><strong>{order.paid_at ? new Date(order.paid_at).toLocaleString('zh-CN', { hour12: false }) : '-'}</strong></div>
        </div>
        {order.payment_status === 'paid' && order.credit_status !== 'succeeded' && <InlineNotice tone="warning">支付已确认，额度入账正在由后台任务处理，请稍后刷新。</InlineNotice>}
      </Section>
    </>}
  </div>;
}
