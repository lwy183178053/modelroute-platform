import { useCallback, useEffect, useState } from 'react';
import { Pencil, Plus, Search, Trash2 } from 'lucide-react';
import { apiRequest, formatQuota, getQuotaPerUnit } from '../../api';
import { ActionMenu, Button, CopyValue, Field, InlineNotice, Modal, Pagination, Section, Status, TextInput } from '../../ui';

function localDateTime(timestamp) {
  if (!timestamp || Number(timestamp) <= 0) return '';
  const date = new Date(Number(timestamp) * 1000);
  const offset = date.getTimezoneOffset() * 60_000;
  return new Date(date.getTime() - offset).toISOString().slice(0, 16);
}

function timestamp(value) {
  if (!value) return 0;
  return Math.floor(new Date(value).getTime() / 1000);
}

function RedemptionDialog({ close, config, notify }) {
  const editing = config.mode === 'edit';
  const item = config.item || {};
  const [name, setName] = useState(item.name || '');
  const [quotaCNY, setQuotaCNY] = useState(editing ? String(Number(item.quota || 0) / getQuotaPerUnit()) : '1');
  const [count, setCount] = useState('1');
  const [expiresAt, setExpiresAt] = useState(localDateTime(item.expired_time));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const save = async () => {
    const quota = Math.round(Number(quotaCNY) * getQuotaPerUnit());
    const quantity = Number(count);
    if (!name.trim() || !Number.isInteger(quota) || quota <= 0 || (!editing && (!Number.isInteger(quantity) || quantity < 1))) {
      setError('请填写有效的名称、额度和数量');
      return;
    }
    setBusy(true);
    setError('');
    try {
      const body = { name: name.trim(), quota, expired_time: timestamp(expiresAt) };
      if (!editing) body.count = quantity;
      const data = await apiRequest(editing ? `/api/admin/redemptions/${item.id}` : '/api/admin/redemptions', { method: editing ? 'PUT' : 'POST', body });
      notify(editing ? '兑换码已更新' : '兑换码已创建');
      config.onSaved(Array.isArray(data) ? data : []);
      close();
    } catch (requestError) {
      setError(requestError.message || '保存兑换码失败');
    } finally {
      setBusy(false);
    }
  };

  return <Modal
    footer={<><Button onClick={close}>取消</Button><Button disabled={busy} onClick={save} variant="primary">{busy ? '正在保存' : editing ? '保存兑换码' : '创建兑换码'}</Button></>}
    onClose={close}
    title={editing ? '编辑兑换码' : '新建兑换码'}
  >
    <div className="form-stack">
      <Field label="名称"><TextInput maxLength="64" onChange={(event) => setName(event.target.value)} value={name} /></Field>
      <Field label="单码额度（人民币）"><TextInput min="0.01" onChange={(event) => setQuotaCNY(event.target.value)} step="0.01" type="number" value={quotaCNY} /></Field>
      {!editing && <Field label="生成数量"><TextInput max="100" min="1" onChange={(event) => setCount(event.target.value)} type="number" value={count} /></Field>}
      <Field label="失效时间" hint="留空表示永不过期"><TextInput onChange={(event) => setExpiresAt(event.target.value)} type="datetime-local" value={expiresAt} /></Field>
      {error && <InlineNotice tone="warning">{error}</InlineNotice>}
    </div>
  </Modal>;
}

export default function AdminRedemptions({ notify }) {
  const [items, setItems] = useState([]);
  const [keyword, setKeyword] = useState('');
  const [submittedKeyword, setSubmittedKeyword] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [dialog, setDialog] = useState(null);
  const [createdCodes, setCreatedCodes] = useState([]);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const query = new URLSearchParams({ p: String(page), page_size: String(pageSize) });
      if (submittedKeyword) query.set('keyword', submittedKeyword);
      const data = await apiRequest(`/api/admin/redemptions?${query}`);
      setItems(data?.items || []);
      setTotal(Number(data?.total || 0));
    } catch (requestError) {
      setError(requestError.message || '读取兑换码失败');
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, submittedKeyword]);

  useEffect(() => { load(); }, [load]);

  const setStatus = async (item) => {
    const enabled = Number(item.status) !== 1;
    try {
      await apiRequest(`/api/admin/redemptions/${item.id}/status`, { method: 'POST', body: { enabled } });
      notify(enabled ? '兑换码已启用' : '兑换码已停用');
      await load();
    } catch (requestError) {
      notify(requestError.message || '修改兑换码状态失败');
    }
  };

  const remove = async (item) => {
    if (!window.confirm(`确认删除兑换码“${item.name}”吗？`)) return;
    try {
      await apiRequest(`/api/admin/redemptions/${item.id}`, { method: 'DELETE' });
      notify('兑换码已删除');
      await load();
    } catch (requestError) {
      notify(requestError.message || '删除兑换码失败');
    }
  };

  const cleanup = async () => {
    if (!window.confirm('确认清理所有已失效兑换码吗？')) return;
    try {
      const count = await apiRequest('/api/admin/redemptions/invalid', { method: 'DELETE' });
      notify(`已清理 ${Number(count || 0)} 条失效兑换码`);
      await load();
    } catch (requestError) {
      notify(requestError.message || '清理失效兑换码失败');
    }
  };

  const copyCode = async (code) => {
    try {
      await navigator.clipboard.writeText(code);
      notify('兑换码已复制');
    } catch {
      notify('复制兑换码失败');
    }
  };

  return <>
    <Section
      title="兑换码管理"
      description="兑换额度使用平台统一整数额度口径"
      actions={<><Button icon={Trash2} onClick={cleanup} variant="danger">清理失效</Button><Button icon={Plus} onClick={() => setDialog({ mode: 'create' })} variant="primary">新建兑换码</Button></>}
    >
      <form className="toolbar-form" onSubmit={(event) => { event.preventDefault(); setSubmittedKeyword(keyword.trim()); setPage(1); }}>
        <TextInput aria-label="搜索兑换码" onChange={(event) => setKeyword(event.target.value)} placeholder="名称或兑换码" value={keyword} />
        <Button icon={Search} type="submit">搜索</Button>
      </form>
      {error && <InlineNotice tone="warning">{error}</InlineNotice>}
      {createdCodes.length > 0 && <InlineNotice tone="success"><span>本次生成：</span>{createdCodes.map((code) => <CopyValue key={code} onCopy={() => copyCode(code)}>{code}</CopyValue>)}</InlineNotice>}
      <div className="table-wrap"><table className="data-table"><thead><tr><th>ID</th><th>名称</th><th>兑换码</th><th>单码额度</th><th>数量</th><th>失效时间</th><th>状态</th><th>操作</th></tr></thead><tbody>
        {items.map((item) => <tr key={item.id}>
          <td><code>{item.id}</code></td><td><strong>{item.name}</strong></td>
          <td><CopyValue onCopy={() => copyCode(item.key)}>{item.key || '-'}</CopyValue></td>
          <td>{formatQuota(item.quota)}</td>
          <td>{item.count ?? '-'}</td>
          <td>{Number(item.expired_time) > 0 ? new Date(Number(item.expired_time) * 1000).toLocaleString('zh-CN', { hour12: false }) : '永不过期'}</td>
          <td><Status>{Number(item.status) === 1 ? '正常' : '停用'}</Status></td>
          <td><ActionMenu items={[{ label: '编辑', icon: Pencil, onClick: () => setDialog({ mode: 'edit', item }) }, { label: Number(item.status) === 1 ? '停用' : '启用', onClick: () => setStatus(item) }, { label: '删除', icon: Trash2, danger: true, onClick: () => remove(item) }]} /></td>
        </tr>)}
        {!loading && items.length === 0 && <tr><td className="table-message" colSpan="8">暂无兑换码</td></tr>}
        {loading && <tr><td className="table-message" colSpan="8">正在读取兑换码</td></tr>}
      </tbody></table></div>
      <Pagination page={page} pageSize={pageSize} total={total} onPageChange={setPage} onPageSizeChange={(nextPageSize) => { setPageSize(nextPageSize); setPage(1); }} disabled={loading} />
    </Section>
    {dialog && <RedemptionDialog close={() => setDialog(null)} config={{ ...dialog, onSaved: (codes) => { setCreatedCodes(codes); load(); } }} notify={notify} />}
  </>;
}
