import {
  Ban,
  CheckCircle2,
  Download,
  FileClock,
  KeyRound,
  Link2,
  Pencil,
  RefreshCw,
  RotateCcw,
  ShieldCheck,
  WalletCards,
} from 'lucide-react';
import { paymentOrders } from '../../data';
import { Button, InlineNotice, Section, Status } from '../../ui';

export default function AdminPayment({ notify, openModal }) {
  return (
    <div className="admin-payment">
      <Section
        title="易支付设置"
        description="充值订单、签名、回调验签、幂等和对账均由外围平台处理"
        actions={<><Button icon={Link2} onClick={() => notify('网关连接正常')}>测试连接</Button><Button icon={Pencil} variant="primary" onClick={() => openModal('payment')}>编辑配置</Button></>}
      >
        <div className="payment-config">
          <div className="payment-state"><span className="payment-mark"><WalletCards size={24} /></span><div><strong>易支付 · 生产配置</strong><span>配置版本 v12 · 更新于 2026-07-22 08:40</span></div><Status>已开通</Status></div>
          <div className="config-grid">
            <div><span>网关地址</span><strong>https://zpayz.cn/</strong></div>
            <div><span>商户编号 PID</span><strong>2026********1878</strong></div>
            <div><span>商户密钥 KEY</span><strong>已配置 · ••••••••••••••••••••••••••••••••</strong></div>
            <div><span>支付方式</span><strong>支付宝、微信、QQ 钱包</strong></div>
            <div><span>充值限额</span><strong>¥10.00 - ¥20,000.00</strong></div>
            <div><span>异步回调</span><strong>https://api.modelroute.cn/payment/notify</strong></div>
          </div>
          <div className="config-actions">
            <Button icon={ShieldCheck} onClick={() => notify('回调签名自检通过')}>验签自检</Button>
            <Button icon={FileClock} onClick={() => notify('已打开最近 24 小时回调日志')}>查看回调日志</Button>
            <Button icon={KeyRound} onClick={() => openModal('payment')}>轮换密钥</Button>
            <Button icon={Ban} variant="danger" onClick={() => notify('演示状态：支付配置未实际停用')}>停用支付</Button>
          </div>
        </div>
        <InlineNotice tone="success"><CheckCircle2 size={16} />完整 KEY 只在服务端解密用于签名，保存后管理员页面也只显示配置状态和掩码。</InlineNotice>
      </Section>

      <Section
        title="充值订单与入账"
        description="支付成功与 New API 额度入账分别使用幂等键，入账失败不改变已支付状态"
        actions={<><Button icon={RefreshCw} onClick={() => notify('订单状态已刷新')}>刷新订单</Button><Button icon={RotateCcw} onClick={() => notify('已完成订单对账')}>订单对账</Button><Button icon={Download}>导出订单</Button></>}
      >
        <div className="table-wrap"><table className="data-table"><thead><tr><th>平台订单号</th><th>用户</th><th>金额</th><th>支付方式</th><th>第三方交易号</th><th>支付时间</th><th>支付状态</th><th>额度入账</th><th>操作</th></tr></thead><tbody>
          {paymentOrders.map((order) => <tr key={order.id}><td><code>{order.id}</code></td><td>{order.user}</td><td><strong>{order.amount}</strong></td><td>{order.method}</td><td>{order.trade}</td><td>{order.paid}</td><td><Status>{order.state}</Status></td><td><Status>{order.credit}</Status></td><td><div className="row-actions"><Button variant="ghost">详情</Button>{order.credit === '入账异常' && <Button variant="ghost" onClick={() => notify(`${order.id} 已重新提交入账`)}>重试入账</Button>}{order.state === '待支付' && <Button variant="danger" onClick={() => notify(`${order.id} 已关闭`)}>关闭异常订单</Button>}</div></td></tr>)}
        </tbody></table></div>
      </Section>

      <Section title="最近回调" description="回调正文仅保留摘要，原始密钥及完整敏感字段不进入日志" actions={<Button variant="ghost">全部回调日志</Button>}>
        <div className="callback-list">
          <div><Status>成功</Status><code>notify_8fd1c2</code><span>订单 CZ202607220184 · 验签通过 · 已完成</span><small>09:36:19</small></div>
          <div><Status>成功</Status><code>notify_70a8b4</code><span>订单 CZ202607220179 · 验签通过 · 等待重试入账</span><small>09:22:08</small></div>
          <div><Status>验证失败</Status><code>notify_2c91f7</code><span>未知订单 · 金额不一致 · 已拦截</span><small>08:48:31</small></div>
        </div>
      </Section>
    </div>
  );
}
