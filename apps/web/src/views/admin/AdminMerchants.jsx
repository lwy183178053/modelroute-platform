import { useCallback, useEffect, useState } from 'react';
import { Check, RefreshCw, Search, ShieldCheck, X } from 'lucide-react';
import { apiRequest } from '../../api';
import { ActionMenu, Button, InlineNotice, Section, Status, TextInput } from '../../ui';

const statusNames = {
  pending: '审核中', verifying: '审核中', verified: '审核中', approving: '审核中',
  approved: '已审核', rejected: '不通过',
};

export default function AdminMerchants({ notify }) {
  const [applications, setApplications] = useState([]);
  const [keyword, setKeyword] = useState('');
  const [error, setError] = useState('');
  const [busyID, setBusyID] = useState('');

  const load = useCallback(async () => {
    setError('');
    try {
      const query = new URLSearchParams({ p: '1', page_size: '100' });
      if (keyword.trim()) query.set('keyword', keyword.trim());
      const applicationData = await apiRequest(`/api/admin/merchant-applications?${query}`);
      setApplications(applicationData?.items || []);
    } catch (requestError) {
      setError(requestError.message || '读取商户审核数据失败');
    }
  }, [keyword]);

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
      notify(action === 'verify' ? '渠道验证通过' : action === 'approve' ? '商户已开通' : '商户申请已驳回');
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
        description="核对网站、渠道资料和审核结果"
        actions={<Button icon={RefreshCw} onClick={load}>刷新</Button>}
      >
        <div className="filter-toolbar slim-toolbar"><label className="search-box"><Search size={16} /><TextInput onChange={(event) => setKeyword(event.target.value)} placeholder="搜索申请人、网址或用户 ID" value={keyword} /></label><Button onClick={load} variant="primary">筛选</Button></div>
        <div className="table-wrap"><table className="data-table"><thead><tr><th>ID</th><th>申请人</th><th>网站</th><th>初始渠道</th><th>模型</th><th>检测适配器</th><th>凭据</th><th>状态</th><th>审核操作</th></tr></thead><tbody>
          {applications.map((application) => {
            const review = application.initial_channel || {};
            return <tr key={application.id}><td><code>{application.id}</code><small>{application.created_at ? new Date(application.created_at).toLocaleString('zh-CN') : '-'}</small></td><td><strong>{application.display_name}</strong><small>用户 ID {application.user_id}</small></td><td><a href={application.website_url} rel="noreferrer" target="_blank">{application.website_url}</a></td><td><strong>{review.channel_name || '-'}</strong></td><td>{review.model || '-'}</td><td>{review.adapter_type || '-'}</td><td>{review.credential_configured ? '已加密保存' : '已清除正文'}</td><td><Status>{statusNames[application.status] || application.status}</Status></td><td><ActionMenu items={[{ label: '验证', icon: ShieldCheck, disabled: Boolean(busyID) || !['pending', 'verified'].includes(application.status), onClick: () => operate(application, 'verify') }, { label: '通过', icon: Check, disabled: Boolean(busyID) || application.status !== 'verified', onClick: () => operate(application, 'approve') }, { label: '驳回', icon: X, danger: true, disabled: Boolean(busyID) || !['pending', 'verified'].includes(application.status), onClick: () => operate(application, 'reject') }]} /></td></tr>;
          })}
          {applications.length === 0 && <tr><td className="table-message" colSpan="9">暂无商户申请</td></tr>}
        </tbody></table></div>
      </Section>
    </div>
  );
}
