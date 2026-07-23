import { useCallback, useEffect, useState } from 'react';
import { Check, FileCheck2, X } from 'lucide-react';
import { apiRequest, formatQuota } from '../../api';
import { ActionMenu, InlineNotice, Section, Status } from '../../ui';

const statusNames = { pending: '待审核', approved: '待打款', paid: '已打款', rejected: '已驳回' };
function formatDate(value) {
  if (!value) return '-';
  return new Intl.DateTimeFormat('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }).format(new Date(value));
}

export default function AdminWithdrawals({ notify, openModal }) {
  const [withdrawals, setWithdrawals] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const withdrawalData = await apiRequest('/api/admin/withdrawals?page_size=50');
      setWithdrawals(withdrawalData?.items || []);
    } catch (requestError) {
      setError(requestError.message || '读取提现数据失败');
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
          {withdrawals.map((item) => <tr key={item.id}><td><code>{item.id}</code></td><td><code>{item.withdrawal_no}</code></td><td><strong>{item.merchant_name}</strong></td><td><strong>{formatQuota(item.amount_quota)}</strong></td><td>{formatQuota(item.available_before_quota)}</td><td><span>{item.recipient_type === 'alipay' ? '支付宝' : item.recipient_type === 'bank' ? '银行卡' : '其他'}</span><small><code>{item.recipient_account}</code></small></td><td>{formatDate(item.created_at)}</td><td><Status>{statusNames[item.status] || item.status}</Status></td><td><ActionMenu items={[item.status === 'pending' && { label: '通过', icon: Check, onClick: () => action(item, 'approve') }, ['pending', 'approved'].includes(item.status) && { label: '驳回', icon: X, danger: true, onClick: () => action(item, 'reject') }, item.status === 'approved' && { label: '确认打款', icon: FileCheck2, onClick: () => action(item, 'paid') }]} /></td></tr>)}
          {!loading && withdrawals.length === 0 && <tr><td className="table-message" colSpan="9">暂无提现申请</td></tr>}
          {loading && <tr><td className="table-message" colSpan="9">正在读取提现申请</td></tr>}
        </tbody></table></div>
      </Section>
    </>
  );
}
