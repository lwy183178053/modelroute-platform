import { useCallback, useEffect, useState } from 'react';
import { LockKeyhole, RefreshCw, Search, WalletCards } from 'lucide-react';
import { apiRequest, formatQuota } from '../../api';
import { Button, Section, Status, TextInput } from '../../ui';

function roleName(role) {
  return Number(role) >= 10 ? '管理员' : '普通用户';
}

export default function AdminUsers({ notify, openModal }) {
  const [users, setUsers] = useState([]);
  const [keyword, setKeyword] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const load = useCallback(async (search = '') => {
    setLoading(true);
    setError('');
    try {
      const query = new URLSearchParams({ p: '1', page_size: '50' });
      if (search.trim()) query.set('keyword', search.trim());
      const data = await apiRequest(`/api/admin/users?${query}`);
      setUsers(data?.items || []);
    } catch (requestError) {
      setError(requestError.message || '读取用户失败');
    } finally {
      setLoading(false);
    }
  }, []);

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

  return (
    <Section
      title="用户管理"
      description="账号、状态和额度均通过 New API 管理接口操作"
      actions={<Button icon={RefreshCw} onClick={() => load(keyword)}>刷新</Button>}
    >
      <form className="filter-toolbar slim-toolbar" onSubmit={submitSearch}>
        <div className="search-box"><Search size={16} /><TextInput aria-label="搜索用户" onChange={(event) => setKeyword(event.target.value)} placeholder="账号、邮箱或显示名称" value={keyword} /></div>
        <Button type="submit" variant="primary">搜索</Button>
      </form>
      <div className="table-wrap">
        <table className="data-table admin-user-table">
          <thead><tr><th>用户</th><th>邮箱</th><th>角色</th><th>额度</th><th>已用额度</th><th>请求数</th><th>状态</th><th>操作</th></tr></thead>
          <tbody>
            {users.map((user) => (
              <tr key={user.id}>
                <td><strong>{user.display_name || user.username}</strong><small>UID {user.id} · {user.username}</small></td>
                <td>{user.email || '-'}</td><td>{roleName(user.role)}</td><td>{formatQuota(user.quota)}</td><td>{formatQuota(user.used_quota)}</td><td>{user.request_count || 0}</td>
                <td><Status>{Number(user.status) === 1 ? '正常' : '停用'}</Status></td>
                <td><div className="row-actions"><Button onClick={() => openModal({ type: 'adminQuota', user, onSaved: () => load(keyword) })} variant="ghost" icon={WalletCards}>额度</Button><Button onClick={() => openModal({ type: 'adminPassword', user })} variant="ghost" icon={LockKeyhole}>密码</Button><Button onClick={() => setStatus(user)} variant={Number(user.status) === 1 ? 'danger' : 'ghost'}>{Number(user.status) === 1 ? '封禁' : '解封'}</Button></div></td>
              </tr>
            ))}
            {!loading && users.length === 0 && <tr><td className="table-message" colSpan="8">{error || '没有匹配用户'}</td></tr>}
            {loading && <tr><td className="table-message" colSpan="8">正在读取用户</td></tr>}
          </tbody>
        </table>
      </div>
    </Section>
  );
}
