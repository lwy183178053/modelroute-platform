import { useCallback, useEffect, useMemo, useState } from 'react';
import { RefreshCw, Search, Trash2 } from 'lucide-react';
import { apiRequest } from '../../api';
import { ActionMenu, Button, InlineNotice, Pagination, Section, Select, TextInput } from '../../ui';
import ChannelModelsModal, { ChannelModelsButton } from '../../ChannelModelsModal';

function countModelStates(models = []) {
  return models.reduce((counts, model) => {
    const state = model.health_state || 'pending';
    counts[state] = (counts[state] || 0) + 1;
    return counts;
  }, { pending: 0, healthy: 0, cooling: 0, error: 0 });
}

export default function AdminChannels({ notify }) {
  const [channels, setChannels] = useState([]);
  const [error, setError] = useState('');
  const [busyID, setBusyID] = useState('');
  const [query, setQuery] = useState('');
  const [merchantFilter, setMerchantFilter] = useState('all');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [modelTarget, setModelTarget] = useState(0);

  const load = useCallback(async () => {
    setError('');
    setLoading(true);
    try {
      const params = new URLSearchParams({ p: String(page), page_size: String(pageSize) });
      if (query.trim()) params.set('keyword', query.trim());
      if (merchantFilter !== 'all') params.set('merchant_user_id', merchantFilter);
      const data = await apiRequest(`/api/admin/channels?${params}`);
      setChannels(data?.items || []);
      setTotal(Number(data?.total || 0));
    } catch (requestError) {
      setError(requestError.message || '读取渠道数据失败');
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }, [merchantFilter, page, pageSize, query]);

  useEffect(() => { load(); }, [load]);

  const deleteChannel = async (channel) => {
    if (!window.confirm(`确认删除渠道“${channel.channel_name}”吗？`)) return;
    setBusyID(String(channel.id));
    try {
      await apiRequest(`/api/admin/channels/${channel.id}`, { method: 'DELETE' });
      notify('渠道已删除');
      setModelTarget(0);
      await load();
    } catch (requestError) {
      notify(requestError.message || '删除渠道失败');
    } finally {
      setBusyID('');
    }
  };

  const checkChannelModel = async (channel, model) => {
    setBusyID(`model-${model.id}`);
    setError('');
    try {
      await apiRequest(`/api/admin/channels/${channel.id}/models/${model.id}/check`, { method: 'POST' });
      notify('模型检查通过，状态已更新');
      await load();
    } catch (requestError) {
      await load();
      setError(requestError.message || '模型检查失败');
    } finally {
      setBusyID('');
    }
  };

  const merchants = useMemo(() => {
    const values = new Map();
    channels.forEach((item) => values.set(String(item.user_id), item.merchant_name || `用户 ${item.user_id}`));
    return [...values.entries()].filter(([id]) => id !== 'undefined').sort((left, right) => Number(left[0]) - Number(right[0]));
  }, [channels]);
  const selectedModelRow = modelTarget ? channels.find((item) => item.id === modelTarget) : null;

  return (
    <div>
      {error && <InlineNotice tone="warning">{error}</InlineNotice>}
      <Section title="渠道管理" description="查看商户正式渠道及逐模型状态，管理员和商户均可主动检查模型" actions={<Button icon={RefreshCw} onClick={load}>刷新</Button>}>
        <div className="filter-toolbar">
          <label className="search-box"><Search size={17} /><TextInput aria-label="搜索渠道或商户" placeholder="搜索 ID、渠道、模型或商户" value={query} onChange={(event) => { setQuery(event.target.value); setPage(1); }} /></label>
          <Select aria-label="筛选商户" value={merchantFilter} onChange={(event) => { setMerchantFilter(event.target.value); setPage(1); }}><option value="all">全部商户</option>{merchants.map(([id, name]) => <option key={id} value={id}>{name} · 用户 ID {id}</option>)}</Select>
        </div>
        <div className="table-wrap"><table className="data-table"><thead><tr><th>ID</th><th>商户</th><th>渠道</th><th>厂商</th><th>模型</th><th>基础地址</th><th>模型状态</th><th>操作</th></tr></thead><tbody>
          {channels.map((item) => {
            const counts = countModelStates(item.models);
            return <tr key={item.id}>
              <td><code>{item.id}</code><small>{item.created_at ? new Date(item.created_at).toLocaleString('zh-CN') : '-'}</small></td>
              <td><strong>{item.merchant_name || `用户 ${item.user_id}`}</strong><small>用户 ID {item.user_id}</small></td>
              <td><strong>{item.channel_name}</strong></td>
              <td>{item.provider || '-'}</td>
              <td><ChannelModelsButton item={item} onClick={() => setModelTarget(item.id)} /></td>
              <td><code>{item.base_url}</code></td>
              <td><small>正常 {counts.healthy} · 待检查 {counts.pending} · 冷却 {counts.cooling} · 错误 {counts.error}</small></td>
              <td><ActionMenu items={[{ label: '查看模型', onClick: () => setModelTarget(item.id) }, { label: '删除', icon: Trash2, danger: true, disabled: Boolean(busyID), onClick: () => deleteChannel(item) }]} /></td>
            </tr>;
          })}
          {!loading && channels.length === 0 && <tr><td className="table-message" colSpan="8">暂无匹配渠道</td></tr>}
          {loading && <tr><td className="table-message" colSpan="8">正在读取渠道</td></tr>}
        </tbody></table></div>
        <Pagination page={page} pageSize={pageSize} total={total} onPageChange={setPage} onPageSizeChange={(nextPageSize) => { setPageSize(nextPageSize); setPage(1); }} disabled={loading} />
      </Section>
      {selectedModelRow && <ChannelModelsModal item={selectedModelRow} busyID={busyID} onClose={() => setModelTarget(0)} onTest={checkChannelModel} />}
    </div>
  );
}
