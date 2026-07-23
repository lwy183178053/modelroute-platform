import { useCallback, useEffect, useState } from 'react';
import { Check, FileCheck2, ShieldAlert, X } from 'lucide-react';
import { apiRequest, formatQuota } from '../../api';
import { Button, InlineNotice, Section, Status } from '../../ui';

const statusNames = { pending: '待审核', approved: '待打款', paid: '已打款', rejected: '已驳回' };
const reasonNames = {
  offer_mapping_missing: '报价归属缺失',
  refund_origin_missing: '退款原消费缺失',
  refund_origin_ambiguous: '退款原消费不唯一',
  refund_origin_insufficient: '退款原消费额度不足',
};

function formatDate(value) {
  if (!value) return '-';
  return new Intl.DateTimeFormat('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }).format(new Date(value));
}

export default function AdminWithdrawals({ notify, openModal }) {
  const [withdrawals, setWithdrawals] = useState([]);
  const [reconciliation, setReconciliation] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [withdrawalData, reconciliationData] = await Promise.all([
        apiRequest('/api/admin/withdrawals?page_size=50'),
        apiRequest('/api/admin/settlement-reconciliation?status=open&page_size=50'),
      ]);
      setWithdrawals(withdrawalData?.items || []);
      setReconciliation(reconciliationData?.items || []);
    } catch (requestError) {
      setError(requestError.message || '读取提现与对账数据失败');
    } finally {
      setLoading(false);
    }
  }, []);
  useEffect(() => { load(); }, [load]);

  const action = (item, actionName) => openModal({ type: 'withdrawalAction', action: actionName, item, onSaved: load });
  return (
    <>
      {error && <InlineNotice tone="warning">{error}</InlineNotice>}
      <Section title="人工提现审核" description="提现不自动打款；通过后仍保持冻结，录入线下打款凭证才完成出账">
        <InlineNotice tone="warning">审核、打款确认和驳回均要求管理员填写原因或凭证，并留存最终处理结果。</InlineNotice>
        <div className="table-wrap"><table className="data-table"><thead><tr><th>ID</th><th>申请号</th><th>商户</th><th>提现金额</th><th>申请前可用</th><th>收款账户</th><th>提交时间</th><th>状态</th><th>操作</th></tr></thead><tbody>
          {withdrawals.map((item) => <tr key={item.id}><td><code>{item.id}</code></td><td><code>{item.withdrawal_no}</code></td><td><strong>{item.merchant_name}</strong></td><td><strong>{formatQuota(item.amount_quota)}</strong></td><td>{formatQuota(item.available_before_quota)}</td><td><span>{item.recipient_type === 'alipay' ? '支付宝' : item.recipient_type === 'bank' ? '银行卡' : '其他'}</span><small><code>{item.recipient_account}</code></small></td><td>{formatDate(item.created_at)}</td><td><Status>{statusNames[item.status] || item.status}</Status></td><td><div className="row-actions">{item.status === 'pending' && <Button icon={Check} variant="ghost" onClick={() => action(item, 'approve')}>通过</Button>}{['pending', 'approved'].includes(item.status) && <Button icon={X} variant="danger" onClick={() => action(item, 'reject')}>驳回</Button>}{item.status === 'approved' && <Button icon={FileCheck2} variant="ghost" onClick={() => action(item, 'paid')}>确认打款</Button>}</div></td></tr>)}
          {!loading && withdrawals.length === 0 && <tr><td className="table-message" colSpan="9">暂无提现申请</td></tr>}
          {loading && <tr><td className="table-message" colSpan="9">正在读取提现申请</td></tr>}
        </tbody></table></div>
      </Section>

      <Section title="结算对账队列" description="映射缺失和退款原消费缺失不会猜测入账，映射恢复后 Worker 自动重试">
          <div className="table-wrap"><table className="data-table"><thead><tr><th>ID</th><th>消费日志 ID</th><th>类型</th><th>模型</th><th>渠道 ID</th><th>额度</th><th>异常原因</th><th>时间</th><th>操作</th></tr></thead><tbody>
          {reconciliation.map((item) => <tr key={item.id}><td><code>{item.id}</code></td><td><code>{item.newapi_log_id}</code></td><td>{item.source_type === 6 ? '退款' : '消费'}</td><td><code>{item.model_name}</code></td><td>{item.channel_id || '-'}</td><td>{formatQuota(item.gross_quota)}</td><td><Status>{reasonNames[item.reason_code] || item.reason_code}</Status></td><td>{formatDate(item.created_at)}</td><td><Button icon={ShieldAlert} variant="ghost" onClick={() => openModal({ type: 'reconciliationAction', action: 'ignore', item, onSaved: load })}>忽略异常</Button></td></tr>)}
          {!loading && reconciliation.length === 0 && <tr><td className="table-message" colSpan="9">暂无待处理对账异常</td></tr>}
          {loading && <tr><td className="table-message" colSpan="9">正在读取对账队列</td></tr>}
        </tbody></table></div>
      </Section>
    </>
  );
}
