import { useCallback, useEffect, useState } from 'react';
import { LockKeyhole, RefreshCw, Search, UserRoundCheck, UserRoundX, WalletCards } from 'lucide-react';
import { apiRequest, formatQuota } from '../../api';
import { ActionMenu, Button, Pagination, Section, Select, Status, TextInput } from '../../ui';

function accountType(user) {
  if (user.account_type) return user.account_type;
  return Number(user.role) >= 10 ? 'admin' : 'user';
}

const accountTypeNames = { user: '普通用户', merchant: '商户', admin: '管理员' };

export default function AdminUsers({ notify, openModal }) {
  const [users, setUsers] = useState([]);
  const [keyword, setKeyword] = useState('');
  const [submittedKeyword, setSubmittedKeyword] = useState('');
  const [typeFilter, setTypeFilter] = useState('all');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const load = useCallback(async (search = submittedKeyword) => {
    setLoading(true);
    setError('');
    try {
      const query = new URLSearchParams({ p: String(page), page_size: String(pageSize) });
      if (search.trim()) query.set('keyword', search.trim());
      if (typeFilter !== 'all') query.set('type', typeFilter);
      const data = await apiRequest(`/api/admin/users?${query}`);
      setUsers(data?.items || []);
      setTotal(Number(data?.total || 0));
    } catch (requestError) {
      setError(requestError.message || '读取用户失败');
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, submittedKeyword, typeFilter]);

  useEffect(() => { load(); }, [load]);

  const submitSearch = (event) => {
    event.preventDefault();
    setSubmittedKeyword(keyword.trim());
    setPage(1);
  };

  const setStatus = async (user) => {
    try {
      await apiRequest(`/api/admin/users/${user.id}/status`, {
        method: 'POST', body: { enabled: Number(user.status) !== 1 },
      });
      notify(Number(user.status) === 1 ? '账号已封禁' : '账号已解封');
      load();
    } catch (requestError) {
      notify(requestError.message || '状态修改失败');
    }
  };

  const setMerchantStatus = async (user) => {
    const enabled = user.merchant_status !== 'active';
    const reason = window.prompt(enabled ? '请输入恢复商户权限原因' : '请输入撤销商户权限原因');
    if (!reason) return;
    try {
      await apiRequest(`/api/admin/users/${user.id}/merchant-status`, { method: 'POST', body: { enabled, reason } });
      notify(enabled ? '商户权限已恢复' : '商户权限已撤销');
      load();
    } catch (requestError) {
      notify(requestError.message || '商户权限修改失败');
    }
  };

  return (
    <Section
      title="用户管理"
      description="普通用户升级为商户后仍使用同一账号，类型和权限在此统一管理"
      actions={<Button icon={RefreshCw} onClick={() => load()}>刷新</Button>}
    >
      <form className="filter-toolbar slim-toolbar" onSubmit={submitSearch}>
        <div className="search-box"><Search size={16} /><TextInput aria-label="搜索用户" onChange={(event) => setKeyword(event.target.value)} placeholder="账号、邮箱或显示名称" value={keyword} /></div>
        <Select aria-label="用户类型" onChange={(event) => { setTypeFilter(event.target.value); setPage(1); }} value={typeFilter}><option value="all">全部类型</option><option value="user">普通用户</option><option value="merchant">商户</option><option value="admin">管理员</option></Select>
        <Button type="submit" variant="primary">搜索</Button>
      </form>
      <div className="table-wrap">
        <table className="data-table admin-user-table">
          <thead><tr><th>ID</th><th>用户</th><th>邮箱</th><th>类型</th><th>额度</th><th>已用额度</th><th>请求数</th><th>状态</th><th>操作</th></tr></thead>
          <tbody>
            {users.map((user) => (
              <tr key={user.id}>
                <td><code>{user.id}</code></td><td><strong>{user.display_name || user.username}</strong><small>{user.username}</small></td>
                <td>{user.email || '-'}</td><td><Status>{accountTypeNames[accountType(user)] || accountType(user)}</Status>{accountType(user) === 'merchant' && <small>{user.merchant_status === 'active' ? '权限正常' : '权限已撤销'}</small>}</td><td>{formatQuota(user.quota)}</td><td>{formatQuota(user.used_quota)}</td><td>{user.request_count || 0}</td>
                <td><Status>{Number(user.status) === 1 ? '正常' : '停用'}</Status></td>
                <td><ActionMenu items={[{ label: '额度', icon: WalletCards, onClick: () => openModal({ type: 'adminQuota', user, onSaved: () => load() }) }, { label: '密码', icon: LockKeyhole, onClick: () => openModal({ type: 'adminPassword', user }) }, accountType(user) === 'merchant' && { label: user.merchant_status === 'active' ? '撤销商户' : '恢复商户', icon: user.merchant_status === 'active' ? UserRoundX : UserRoundCheck, danger: user.merchant_status === 'active', onClick: () => setMerchantStatus(user) }, { label: Number(user.status) === 1 ? '封禁' : '解封', danger: Number(user.status) === 1, onClick: () => setStatus(user) }]} /></td>
              </tr>
            ))}
            {!loading && users.length === 0 && <tr><td className="table-message" colSpan="9">{error || '没有匹配用户'}</td></tr>}
            {loading && <tr><td className="table-message" colSpan="9">正在读取用户</td></tr>}
          </tbody>
        </table>
      </div>
      <Pagination page={page} pageSize={pageSize} total={total} onPageChange={setPage} onPageSizeChange={(nextPageSize) => { setPageSize(nextPageSize); setPage(1); }} disabled={loading} />
    </Section>
  );
}
