import { useEffect, useState } from 'react';
import { RefreshCw } from 'lucide-react';
import { apiRequest } from '../../api';
import { Button, Section, Status } from '../../ui';

export default function AdminAudit() {
  const [entries, setEntries] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const load = async () => {
    setLoading(true);
    setError('');
    try {
      const data = await apiRequest('/api/admin/audit-logs?p=1&page_size=50');
      setEntries(data?.items || []);
    } catch (requestError) {
      setError(requestError.message || '读取审计日志失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, []);

  return (
    <Section title="平台审计日志" description="管理员敏感操作与处理结果留痕" actions={<Button icon={RefreshCw} onClick={load}>刷新</Button>}>
      <div className="table-wrap"><table className="data-table"><thead><tr><th>时间</th><th>管理员</th><th>业务对象</th><th>操作</th><th>结果</th><th>来源地址</th></tr></thead><tbody>
        {entries.map((entry) => <tr key={entry.id}><td>{new Date(entry.created_at).toLocaleString('zh-CN', { hour12: false })}</td><td>UID {entry.actor_user_id}</td><td>{entry.object_type} #{entry.object_id || '-'}</td><td><code>{entry.action}</code></td><td><Status>{entry.result === 'success' ? '成功' : '失败'}</Status></td><td>{entry.source_ip || '-'}</td></tr>)}
        {!loading && entries.length === 0 && <tr><td className="table-message" colSpan="6">{error || '暂无审计日志'}</td></tr>}
        {loading && <tr><td className="table-message" colSpan="6">正在读取审计日志</td></tr>}
      </tbody></table></div>
    </Section>
  );
}
