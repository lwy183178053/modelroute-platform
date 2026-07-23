import { useCallback, useEffect, useState } from 'react';
import { LockKeyhole, RefreshCw, Search, UserRoundCheck, UserRoundX, WalletCards } from 'lucide-react';
import { apiRequest, formatQuota } from '../../api';
import { Button, Section, Select, Status, TextInput } from '../../ui';

function accountType(user) {
  if (user.account_type) return user.account_type;
  return Number(user.role) >= 10 ? 'admin' : 'user';
}

const accountTypeNames = { user: '普通用户', merchant: '商户', admin: '管理员' };

export default function AdminUsers({ notify, openModal }) {
  const [users, setUsers] = useState([]);
  const [keyword, setKeyword] = useState('');
  const [typeFilter, setTypeFilter] = useState('all');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const load = useCallback(async (search = '') => {
    setLoading(true);
    setError('');
    try {
      const query = new URLSearchParams({ p: '1', page_size: '50' });
      if (search.trim()) query.set('keyword', search.trim());
      if (typeFilter !== 'all') query.set('type', typeFilter);
      const data = await apiRequest(`/api/admin/users?${query}`);
      setUsers(data?.items || []);
    } catch (requestError) {
      setError(requestError.message || '读取用户失败');
    } finally {
      setLoading(false);
    }
  }, [typeFilter]);

  useEffect(() => { load(); }, [load]);

  const submitSearch = (event) => {
    event.preventDefault();
    load(keyword);
  };

  const setStatus = async (user) => {
    try {
      await apiRequest(`/api/admin/users/${user.id}/status`, {
        method: 'POST', body: { enabled: Number(user.status) !== 1 },
      });
      notify(Number(user.status) === 1 ? '账号已封禁' : '账号已解封');
      load(keyword);
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
      load(keyword);
    } catch (requestError) {
      notify(requestError.message || '商户权限修改失败');
    }
  };

  return (
    <Section
      title="用户管理"
      description="普通用户升级为商户后仍使用同一账号，类型和权限在此统一管理"
      actions={<Button icon={RefreshCw} onClick={() => load(keyword)}>刷新</Button>}
    >
      <form className="filter-toolbar slim-toolbar" onSubmit={submitSearch}>
        <div className="search-box"><Search size={16} /><TextInput aria-label="搜索用户" onChange={(event) => setKeyword(event.target.value)} placeholder="账号、邮箱或显示名称" value={keyword} /></div>
        <Select aria-label="用户类型" onChange={(event) => setTypeFilter(event.target.value)} value={typeFilter}><option value="all">全部类型</option><option value="user">普通用户</option><option value="merchant">商户</option><option value="admin">管理员</option></Select>
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
                <td><div className="row-actions"><Button onClick={() => openModal({ type: 'adminQuota', user, onSaved: () => load(keyword) })} variant="ghost" icon={WalletCards}>额度</Button><Button onClick={() => openModal({ type: 'adminPassword', user })} variant="ghost" icon={LockKeyhole}>密码</Button>{accountType(user) === 'merchant' && <Button icon={user.merchant_status === 'active' ? UserRoundX : UserRoundCheck} onClick={() => setMerchantStatus(user)} variant={user.merchant_status === 'active' ? 'danger' : 'ghost'}>{user.merchant_status === 'active' ? '撤销商户' : '恢复商户'}</Button>}<Button onClick={() => setStatus(user)} variant={Number(user.status) === 1 ? 'danger' : 'ghost'}>{Number(user.status) === 1 ? '封禁' : '解封'}</Button></div></td>
              </tr>
            ))}
            {!loading && users.length === 0 && <tr><td className="table-message" colSpan="9">{error || '没有匹配用户'}</td></tr>}
            {loading && <tr><td className="table-message" colSpan="9">正在读取用户</td></tr>}
          </tbody>
        </table>
      </div>
    </Section>
  );
}
