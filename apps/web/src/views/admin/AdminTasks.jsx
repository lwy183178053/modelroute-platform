import { useCallback, useEffect, useState } from 'react';
import { Eye, RefreshCw, Trash2 } from 'lucide-react';
import { apiRequest } from '../../api';
import { Button, Field, InlineNotice, Modal, Section, Status, TextInput } from '../../ui';

const taskTypeNames = { log_cleanup: '日志清理' };
const taskStatusNames = { pending: '等待执行', running: '执行中', succeeded: '已完成', failed: '执行失败' };

function taskTime(value) {
  if (!value) return '-';
  return new Date(Number(value) * 1000).toLocaleString('zh-CN', { hour12: false });
}

function CleanupDialog({ close, notify, onSaved }) {
  const [before, setBefore] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const create = async () => {
    const beforeTimestamp = Math.floor(new Date(before).getTime() / 1000);
    if (!Number.isInteger(beforeTimestamp) || beforeTimestamp <= 0 || beforeTimestamp >= Math.floor(Date.now() / 1000)) {
      setError('请选择早于当前时间的清理截止时间');
      return;
    }
    setBusy(true);
    setError('');
    try {
      await apiRequest('/api/admin/system-tasks/log-cleanup', { method: 'POST', body: { before_timestamp: beforeTimestamp } });
      notify('日志清理任务已创建');
      onSaved();
      close();
    } catch (requestError) {
      setError(requestError.message || '创建日志清理任务失败');
    } finally {
      setBusy(false);
    }
  };
  return <Modal title="创建日志清理任务" onClose={close} footer={<><Button onClick={close}>取消</Button><Button disabled={busy} onClick={create} variant="primary">创建任务</Button></>}>
    <div className="form-stack"><Field label="清理截止时间"><TextInput onChange={(event) => setBefore(event.target.value)} type="datetime-local" value={before} /></Field>{error && <InlineNotice tone="warning">{error}</InlineNotice>}</div>
  </Modal>;
}

function TaskDetail({ close, task }) {
  const deleted = Number(task?.result?.deleted || 0);
  return <Modal title="系统任务详情" onClose={close} footer={<Button onClick={close}>关闭</Button>}>
    <div className="account-rows">
      <div><span>任务编号</span><strong>{task.task_id}</strong></div>
      <div><span>任务类型</span><strong>{taskTypeNames[task.type] || task.type}</strong></div>
      <div><span>执行状态</span><Status>{taskStatusNames[task.status] || task.status}</Status></div>
      <div><span>执行进度</span><strong>{Number(task.progress || 0)}%</strong></div>
      <div><span>执行结果</span><strong>{task.type === 'log_cleanup' ? `已删除日志 ${deleted} 条` : JSON.stringify(task.result || {})}</strong></div>
    </div>
  </Modal>;
}

export default function AdminTasks({ notify }) {
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [cleanupOpen, setCleanupOpen] = useState(false);
  const [detail, setDetail] = useState(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      setItems(await apiRequest('/api/admin/system-tasks?limit=50') || []);
    } catch (requestError) {
      setError(requestError.message || '读取系统任务失败');
    } finally {
      setLoading(false);
    }
  }, []);
  useEffect(() => { load(); }, [load]);

  const openDetail = async (task) => {
    try {
      setDetail(await apiRequest(`/api/admin/system-tasks/${encodeURIComponent(task.task_id)}`));
    } catch (requestError) {
      notify(requestError.message || '读取任务详情失败');
    }
  };

  return <>
    <Section title="系统任务" description="查看 New API 后台任务并创建日志清理任务" actions={<><Button icon={RefreshCw} onClick={load}>刷新</Button><Button icon={Trash2} onClick={() => setCleanupOpen(true)} variant="primary">创建日志清理任务</Button></>}>
      {error && <InlineNotice tone="warning">{error}</InlineNotice>}
      <div className="table-wrap"><table className="data-table"><thead><tr><th>任务编号</th><th>类型</th><th>状态</th><th>创建时间</th><th>更新时间</th><th>操作</th></tr></thead><tbody>
        {items.map((task) => <tr key={task.task_id}><td><code>{task.task_id}</code></td><td>{taskTypeNames[task.type] || task.type}</td><td><Status>{taskStatusNames[task.status] || task.status}</Status></td><td>{taskTime(task.created_at)}</td><td>{taskTime(task.updated_at)}</td><td><Button icon={Eye} onClick={() => openDetail(task)} variant="ghost">查看详情</Button></td></tr>)}
        {!loading && items.length === 0 && <tr><td className="table-message" colSpan="6">暂无系统任务</td></tr>}
        {loading && <tr><td className="table-message" colSpan="6">正在读取系统任务</td></tr>}
      </tbody></table></div>
    </Section>
    {cleanupOpen && <CleanupDialog close={() => setCleanupOpen(false)} notify={notify} onSaved={load} />}
    {detail && <TaskDetail close={() => setDetail(null)} task={detail} />}
  </>;
}
