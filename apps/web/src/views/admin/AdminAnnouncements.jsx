import { useCallback, useEffect, useState } from 'react';
import { Megaphone, Pencil, Pin, Plus } from 'lucide-react';
import { apiRequest } from '../../api';
import { Button, Field, InlineNotice, Modal, Section, Status, TextInput } from '../../ui';

const audienceLabels = { user: '普通用户', merchant: '商户', admin: '管理员' };
const statusLabels = { draft: '草稿', scheduled: '定时发布', published: '已发布', withdrawn: '已下架' };

function localDateTime(value) {
  if (!value) return '';
  const date = new Date(value);
  const offset = date.getTimezoneOffset() * 60_000;
  return new Date(date.getTime() - offset).toISOString().slice(0, 16);
}

function AnnouncementDialog({ config, close, notify }) {
  const item = config.item || {};
  const editing = Boolean(item.id);
  const [title, setTitle] = useState(item.title || '');
  const [body, setBody] = useState(item.body || '');
  const [audiences, setAudiences] = useState(item.audiences || []);
  const [pinned, setPinned] = useState(Boolean(item.pinned));
  const [publishAt, setPublishAt] = useState(localDateTime(item.publish_at));
  const [expiresAt, setExpiresAt] = useState(localDateTime(item.expires_at));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const toggleAudience = (value) => {
    setAudiences((current) => current.includes(value) ? current.filter((itemValue) => itemValue !== value) : [...current, value]);
  };

  const save = async () => {
    if (!title.trim() || !body.trim() || audiences.length === 0) {
      setError('请填写标题、正文并选择受众');
      return;
    }
    setBusy(true);
    setError('');
    try {
      await apiRequest(editing ? `/api/admin/announcements/${item.id}` : '/api/admin/announcements', {
        method: editing ? 'PUT' : 'POST',
        body: {
          title: title.trim(), body: body.trim(), audiences, pinned,
          publish_at: publishAt ? new Date(publishAt).toISOString() : null,
          expires_at: expiresAt ? new Date(expiresAt).toISOString() : null,
        },
      });
      notify(editing ? '公告已更新' : publishAt ? '定时公告已保存' : '公告草稿已保存');
      config.onSaved();
      close();
    } catch (requestError) {
      setError(requestError.message || '保存公告失败');
    } finally {
      setBusy(false);
    }
  };

  return <Modal
    footer={<><Button onClick={close}>取消</Button><Button disabled={busy} onClick={save} variant="primary">{busy ? '正在保存' : editing ? '保存公告' : '保存草稿'}</Button></>}
    onClose={close}
    title={editing ? '编辑公告' : '新建公告'}
  >
    <div className="form-stack announcement-form">
      <Field label="公告标题"><TextInput maxLength="120" onChange={(event) => setTitle(event.target.value)} value={title} /></Field>
      <Field label="公告正文"><textarea className="input announcement-textarea" maxLength="10000" onChange={(event) => setBody(event.target.value)} value={body} /></Field>
      <Field label="公告受众">
        <span className="checkbox-line announcement-audiences">
          {Object.entries(audienceLabels).map(([value, label]) => <label key={value}><input checked={audiences.includes(value)} onChange={() => toggleAudience(value)} type="checkbox" />{label}</label>)}
        </span>
      </Field>
      <label className="toggle-line"><input checked={pinned} onChange={(event) => setPinned(event.target.checked)} type="checkbox" /><span><strong>置顶公告</strong></span></label>
      <div className="form-grid two-column">
        <Field label="发布时间" hint="留空保存为草稿"><TextInput onChange={(event) => setPublishAt(event.target.value)} type="datetime-local" value={publishAt} /></Field>
        <Field label="下架时间" hint="留空表示长期有效"><TextInput onChange={(event) => setExpiresAt(event.target.value)} type="datetime-local" value={expiresAt} /></Field>
      </div>
      {error && <InlineNotice tone="warning">{error}</InlineNotice>}
    </div>
  </Modal>;
}

export default function AdminAnnouncements({ notify }) {
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [dialog, setDialog] = useState(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      setItems(await apiRequest('/api/admin/announcements') || []);
    } catch (requestError) {
      setError(requestError.message || '读取公告失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const changeStatus = async (item, action) => {
    try {
      await apiRequest(`/api/admin/announcements/${item.id}/${action}`, { method: 'POST' });
      notify(action === 'publish' ? '公告发布状态已更新' : '公告已下架');
      await load();
    } catch (requestError) {
      notify(requestError.message || '更新公告状态失败');
    }
  };

  return <>
    <Section title="公告管理" description="面向普通用户、商户和管理员发布站内公告" actions={<Button icon={Plus} onClick={() => setDialog({})} variant="primary">新建公告</Button>}>
      {error && <InlineNotice tone="warning">{error}</InlineNotice>}
      <div className="table-wrap"><table className="data-table announcement-table"><thead><tr><th>公告</th><th>受众</th><th>发布时间</th><th>下架时间</th><th>状态</th><th>操作</th></tr></thead><tbody>
        {items.map((item) => <tr key={item.id}>
          <td><strong>{item.pinned && <Pin size={13} />} {item.title}</strong><small>{item.body}</small></td>
          <td>{(item.audiences || []).map((audience) => audienceLabels[audience] || audience).join('、')}</td>
          <td>{item.publish_at ? new Date(item.publish_at).toLocaleString('zh-CN', { hour12: false }) : '尚未设置'}</td>
          <td>{item.expires_at ? new Date(item.expires_at).toLocaleString('zh-CN', { hour12: false }) : '长期有效'}</td>
          <td><Status>{statusLabels[item.status] || item.status}</Status></td>
          <td><div className="row-actions">
            {['draft', 'scheduled'].includes(item.status) && <Button icon={Pencil} onClick={() => setDialog({ item })} variant="ghost">编辑</Button>}
            {['draft', 'scheduled'].includes(item.status) && <Button icon={Megaphone} onClick={() => changeStatus(item, 'publish')} variant="ghost">发布</Button>}
            {['scheduled', 'published'].includes(item.status) && <Button onClick={() => changeStatus(item, 'withdraw')} variant="danger">下架</Button>}
          </div></td>
        </tr>)}
        {!loading && items.length === 0 && <tr><td className="table-message" colSpan="6">暂无公告</td></tr>}
        {loading && <tr><td className="table-message" colSpan="6">正在读取公告</td></tr>}
      </tbody></table></div>
    </Section>
    {dialog && <AnnouncementDialog close={() => setDialog(null)} config={{ ...dialog, onSaved: load }} notify={notify} />}
  </>;
}
