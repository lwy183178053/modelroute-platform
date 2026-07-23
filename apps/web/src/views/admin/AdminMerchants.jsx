import { useCallback, useEffect, useState } from 'react';
import { Check, RefreshCw, Search, ShieldCheck, UserRoundCheck, UserRoundX, X } from 'lucide-react';
import { apiRequest } from '../../api';
import { Button, InlineNotice, Section, Status, TextInput } from '../../ui';

const statusNames = {
  pending: '待审核', verifying: '验证中', verified: '待批准', approving: '导入中',
  approved: '已通过', rejected: '已驳回', active: '正常', revoked: '已撤销',
};

export default function AdminMerchants({ notify }) {
  const [applications, setApplications] = useState([]);
  const [merchants, setMerchants] = useState([]);
  const [keyword, setKeyword] = useState('');
  const [error, setError] = useState('');
  const [busyID, setBusyID] = useState('');

  const load = useCallback(async () => {
    setError('');
    try {
      const query = new URLSearchParams({ p: '1', page_size: '100' });
      if (keyword.trim()) query.set('keyword', keyword.trim());
      const [applicationData, merchantData] = await Promise.all([
        apiRequest(`/api/admin/merchant-applications?${query}`),
        apiRequest('/api/admin/merchants?p=1&page_size=100'),
      ]);
      setApplications(applicationData?.items || []);
      setMerchants(merchantData?.items || []);
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

  const setMerchantStatus = async (merchant, enabled) => {
    const reason = window.prompt(enabled ? '请输入恢复商户权限原因' : '请输入撤销商户权限原因');
    if (!reason) return;
    setBusyID(`merchant-${merchant.id}`);
    try {
      await apiRequest(`/api/admin/merchants/${merchant.id}/status`, { method: 'POST', body: { enabled, reason } });
      notify(enabled ? '商户权限已恢复' : '商户权限已撤销');
      await load();
    } catch (requestError) {
      notify(requestError.message || '修改商户权限失败');
    } finally {
      setBusyID('');
    }
  };

  return (
    <div className="admin-section-stack">
      {error && <InlineNotice tone="warning">{error}</InlineNotice>}
      <Section
        title="商户申请审核"
        description="核对网站、渠道资料和审核结果"
        actions={<Button icon={RefreshCw} onClick={load}>刷新</Button>}
      >
        <div className="filter-toolbar slim-toolbar"><label className="search-box"><Search size={16} /><TextInput onChange={(event) => setKeyword(event.target.value)} placeholder="搜索申请人、网址或 UID" value={keyword} /></label><Button onClick={load} variant="primary">筛选</Button></div>
        <div className="table-wrap"><table className="data-table"><thead><tr><th>申请</th><th>申请人</th><th>网站</th><th>初始渠道</th><th>模型</th><th>检测适配器</th><th>凭据</th><th>状态</th><th>审核操作</th></tr></thead><tbody>
          {applications.map((application) => {
            const review = application.initial_channel || {};
            return <tr key={application.id}><td><code>A{application.id}</code><small>{application.created_at ? new Date(application.created_at).toLocaleString('zh-CN') : '-'}</small></td><td><strong>{application.display_name}</strong><small>UID {application.user_id}</small></td><td><a href={application.website_url} rel="noreferrer" target="_blank">{application.website_url}</a></td><td><strong>{review.channel_name || '-'}</strong><small>R{review.id || '-'}</small></td><td>{review.model || '-'}</td><td>{review.adapter_type || '-'}</td><td>{review.credential_configured ? '已加密保存' : '已清除正文'}</td><td><Status>{statusNames[application.status] || application.status}</Status></td><td><div className="row-actions"><Button disabled={Boolean(busyID) || !['pending', 'verified'].includes(application.status)} icon={ShieldCheck} onClick={() => operate(application, 'verify')} variant="ghost">验证</Button><Button disabled={Boolean(busyID) || application.status !== 'verified'} icon={Check} onClick={() => operate(application, 'approve')} variant="ghost">通过</Button><Button disabled={Boolean(busyID) || !['pending', 'verified'].includes(application.status)} icon={X} onClick={() => operate(application, 'reject')} variant="danger">驳回</Button></div></td></tr>;
          })}
          {applications.length === 0 && <tr><td className="table-message" colSpan="9">暂无商户申请</td></tr>}
        </tbody></table></div>
      </Section>

      <Section title="商户权限" description="撤销后同步停用该商户的 New API 渠道">
        <div className="table-wrap"><table className="data-table"><thead><tr><th>商户</th><th>账号 UID</th><th>网站</th><th>开通时间</th><th>状态</th><th>操作</th></tr></thead><tbody>
          {merchants.map((merchant) => <tr key={merchant.id}><td><strong>{merchant.display_name}</strong><small>M{merchant.id}</small></td><td>{merchant.user_id}</td><td>{merchant.website_url}</td><td>{merchant.approved_at ? new Date(merchant.approved_at).toLocaleString('zh-CN') : '-'}</td><td><Status>{statusNames[merchant.status] || merchant.status}</Status></td><td>{merchant.status === 'active' ? <Button disabled={Boolean(busyID)} icon={UserRoundX} onClick={() => setMerchantStatus(merchant, false)} variant="danger">撤销权限</Button> : <Button disabled={Boolean(busyID)} icon={UserRoundCheck} onClick={() => setMerchantStatus(merchant, true)} variant="ghost">恢复权限</Button>}</td></tr>)}
          {merchants.length === 0 && <tr><td className="table-message" colSpan="6">暂无已开通商户</td></tr>}
        </tbody></table></div>
      </Section>
    </div>
  );
}
