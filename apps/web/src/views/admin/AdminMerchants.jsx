import { useCallback, useEffect, useState } from 'react';
import { Check, RefreshCw, Search, X } from 'lucide-react';
import { apiRequest } from '../../api';
import { ActionMenu, Button, InlineNotice, Pagination, Section, Status, TextInput } from '../../ui';

const statusNames = {
  pending: '审核中', verifying: '审核中', verified: '审核中', approving: '审核中',
  approved: '已审核', rejected: '不通过',
};

export default function AdminMerchants({ notify }) {
  const [applications, setApplications] = useState([]);
  const [keyword, setKeyword] = useState('');
  const [submittedKeyword, setSubmittedKeyword] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [busyID, setBusyID] = useState('');

  const load = useCallback(async () => {
    setError('');
    try {
      setLoading(true);
      const query = new URLSearchParams({ p: String(page), page_size: String(pageSize) });
      if (submittedKeyword) query.set('keyword', submittedKeyword);
      const applicationData = await apiRequest(`/api/admin/merchant-applications?${query}`);
      setApplications(applicationData?.items || []);
      setTotal(Number(applicationData?.total || 0));
    } catch (requestError) {
      setError(requestError.message || '读取商户审核数据失败');
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, submittedKeyword]);

  useEffect(() => { load(); }, [load]);

  const operate = async (application, action) => {
    let body;
    if (action === 'reject') {
      const reason = window.prompt('请输入驳回原因');
      if (!reason) return;
      body = { reason };
    }
    setBusyID(`${application.id}-${action}`);
    try {
      await apiRequest(`/api/admin/merchant-applications/${application.id}/${action}`, { method: 'POST', body });
      notify(action === 'approve' ? '商户已开通' : '商户申请已驳回');
      await load();
    } catch (requestError) {
      notify(requestError.message || '审核操作失败');
    } finally {
      setBusyID('');
    }
  };

  return (
    <div>
      {error && <InlineNotice tone="warning">{error}</InlineNotice>}
      <Section
        title="商户申请审核"
        description="核对商户名称、介绍和网站资料后直接审批"
        actions={<Button icon={RefreshCw} onClick={load}>刷新</Button>}
      >
        <form className="filter-toolbar slim-toolbar" onSubmit={(event) => { event.preventDefault(); setSubmittedKeyword(keyword.trim()); setPage(1); }}><label className="search-box"><Search size={16} /><TextInput onChange={(event) => setKeyword(event.target.value)} placeholder="搜索申请人、网址或用户 ID" value={keyword} /></label><Button type="submit" variant="primary">筛选</Button></form>
        <div className="table-wrap"><table className="data-table"><thead><tr><th>ID</th><th>商户 / 用户</th><th>商户介绍</th><th>网站</th><th>状态</th><th>审核操作</th></tr></thead><tbody>
          {applications.map((application) => <tr key={application.id}><td><code>{application.id}</code><small>{application.created_at ? new Date(application.created_at).toLocaleString('zh-CN') : '-'}</small></td><td><strong>{application.display_name}</strong><small>用户 ID {application.user_id}</small></td><td className="merchant-description-cell">{application.business_description}</td><td><a href={application.website_url} rel="noreferrer" target="_blank">{application.website_url}</a></td><td><Status>{statusNames[application.status] || application.status}</Status></td><td><ActionMenu items={[{ label: '通过', icon: Check, disabled: Boolean(busyID) || application.status !== 'pending', onClick: () => operate(application, 'approve') }, { label: '驳回', icon: X, danger: true, disabled: Boolean(busyID) || application.status !== 'pending', onClick: () => operate(application, 'reject') }]} /></td></tr>)}
          {!loading && applications.length === 0 && <tr><td className="table-message" colSpan="6">暂无商户申请</td></tr>}
          {loading && <tr><td className="table-message" colSpan="6">正在读取商户申请</td></tr>}
        </tbody></table></div>
        <Pagination page={page} pageSize={pageSize} total={total} onPageChange={setPage} onPageSizeChange={(nextPageSize) => { setPageSize(nextPageSize); setPage(1); }} disabled={loading} />
      </Section>
    </div>
  );
}
