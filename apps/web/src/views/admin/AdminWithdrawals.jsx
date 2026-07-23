import { useCallback, useEffect, useState } from 'react';
import { Check, FileCheck2, RefreshCw, X } from 'lucide-react';
import { apiRequest, formatQuota } from '../../api';
import { ActionMenu, Button, InlineNotice, Pagination, Section, Select, Status } from '../../ui';

const statusNames = { pending: '待审核', approved: '待打款', paid: '已打款', rejected: '已驳回' };
function formatDate(value) {
  if (!value) return '-';
  return new Intl.DateTimeFormat('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }).format(new Date(value));
}

export default function AdminWithdrawals({ notify, onSummaryRefresh, openModal }) {
  const [withdrawals, setWithdrawals] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [status, setStatus] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const query = new URLSearchParams({ p: String(page), page_size: String(pageSize) });
      if (status) query.set('status', status);
      const data = await apiRequest(`/api/admin/withdrawals?${query}`);
      setWithdrawals(data?.items || []);
      setTotal(Number(data?.total || 0));
    } catch (requestError) {
      setError(requestError.message || '读取提现数据失败');
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, status]);

  useEffect(() => { load(); }, [load]);

  const action = (item, actionName) => openModal({
    type: 'withdrawalAction',
    action: actionName,
    item,
    onSaved: async () => {
      await load();
      await onSummaryRefresh?.();
    },
  });

  return (
    <>
      {error && <InlineNotice tone="warning">{error}</InlineNotice>}
      <Section title="人工提现审核" description="申请金额扣除 3% 税费后通过支付宝线下打款" actions={<Button icon={RefreshCw} onClick={load}>刷新</Button>}>
        <InlineNotice tone="warning">审核通过后仍保持冻结，录入支付宝打款流水和凭证才完成出账。</InlineNotice>
        <div className="filter-toolbar">
          <Select aria-label="筛选提现状态" onChange={(event) => { setStatus(event.target.value); setPage(1); }} value={status}>
            <option value="">全部状态</option><option value="pending">待审核</option><option value="approved">待打款</option><option value="paid">已打款</option><option value="rejected">已驳回</option>
          </Select>
        </div>
        <div className="table-wrap"><table className="data-table"><thead><tr><th>ID</th><th>申请号</th><th>商户</th><th>申请金额</th><th>税费（3%）</th><th>实际打款</th><th>申请前可用</th><th>支付宝账户</th><th>提交时间</th><th>状态</th><th>操作</th></tr></thead><tbody>
          {withdrawals.map((item) => <tr key={item.id}><td><code>{item.id}</code></td><td><code>{item.withdrawal_no}</code></td><td><strong>{item.merchant_name}</strong></td><td><strong>{formatQuota(item.amount_quota)}</strong></td><td>{formatQuota(item.tax_quota)}</td><td><strong>{formatQuota(item.net_payout_quota)}</strong></td><td>{formatQuota(item.available_before_quota)}</td><td><span>支付宝</span><small><code>{item.recipient_account}</code></small></td><td>{formatDate(item.created_at)}</td><td><Status>{statusNames[item.status] || item.status}</Status></td><td><ActionMenu items={[item.status === 'pending' && { label: '通过', icon: Check, onClick: () => action(item, 'approve') }, ['pending', 'approved'].includes(item.status) && { label: '驳回', icon: X, danger: true, onClick: () => action(item, 'reject') }, item.status === 'approved' && { label: '确认打款', icon: FileCheck2, onClick: () => action(item, 'paid') }]} /></td></tr>)}
          {!loading && withdrawals.length === 0 && <tr><td className="table-message" colSpan="11">暂无提现申请</td></tr>}
          {loading && <tr><td className="table-message" colSpan="11">正在读取提现申请</td></tr>}
        </tbody></table></div>
        <Pagination page={page} pageSize={pageSize} total={total} onPageChange={setPage} onPageSizeChange={(value) => { setPageSize(value); setPage(1); }} disabled={loading} />
      </Section>
    </>
  );
}
