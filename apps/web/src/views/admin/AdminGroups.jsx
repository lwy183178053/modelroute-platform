import { useCallback, useEffect, useState } from 'react';
import { Plus, Save, Trash2 } from 'lucide-react';
import { apiRequest } from '../../api';
import { Button, Field, InlineNotice, Section, TextInput } from '../../ui';

function ratioFromBPS(value) {
  return String(Number(value || 0) / 10_000);
}

export default function AdminGroups({ notify }) {
  const [groups, setGroups] = useState([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const data = await apiRequest('/api/admin/groups');
      setGroups((data || []).map((group) => ({ name: group.name, ratio: ratioFromBPS(group.ratio_bps) })));
    } catch (requestError) {
      setError(requestError.message || '读取分组倍率失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const update = (index, key, value) => setGroups((current) => current.map((group, itemIndex) => itemIndex === index ? { ...group, [key]: value } : group));
  const remove = (index) => setGroups((current) => current.filter((_, itemIndex) => itemIndex !== index));
  const add = () => setGroups((current) => [...current, { name: '', ratio: '1' }]);

  const save = async () => {
    const payload = groups.map((group) => ({ name: group.name.trim(), ratio_bps: Math.round(Number(group.ratio) * 10_000) }));
    if (payload.some((group) => !group.name || !Number.isInteger(group.ratio_bps) || group.ratio_bps < 0)) {
      setError('请填写有效的分组名称和倍率');
      return;
    }
    setSaving(true);
    setError('');
    try {
      const data = await apiRequest('/api/admin/groups', { method: 'PUT', body: { groups: payload } });
      setGroups((data || []).map((group) => ({ name: group.name, ratio: ratioFromBPS(group.ratio_bps) })));
      notify('分组倍率已保存');
    } catch (requestError) {
      setError(requestError.message || '保存分组倍率失败');
    } finally {
      setSaving(false);
    }
  };

  return (
    <Section title="分组倍率" description="倍率按整数基点同步到 New API；default 分组必须保留" actions={<Button icon={Plus} onClick={add}>新增分组</Button>}>
      {error && <InlineNotice tone="warning">{error}</InlineNotice>}
      {loading ? <div className="table-message">正在读取分组倍率</div> : <div className="group-editor-list">
        {groups.map((group, index) => <div className="group-editor-row" key={`${group.name}-${index}`}>
          <Field label="分组名称"><TextInput aria-label={`${group.name || '新建'} 分组名称`} disabled={group.name === 'default'} maxLength="64" onChange={(event) => update(index, 'name', event.target.value)} value={group.name} /></Field>
          <Field label="倍率"><TextInput aria-label={`${group.name || '新建'} 分组倍率`} min="0" onChange={(event) => update(index, 'ratio', event.target.value)} step="0.0001" type="number" value={group.ratio} /></Field>
          <Button disabled={group.name === 'default'} icon={Trash2} onClick={() => remove(index)} variant="danger">删除</Button>
        </div>)}
      </div>}
      <div className="form-footer"><Button disabled={loading || saving} icon={Save} onClick={save} variant="primary">{saving ? '正在保存' : '保存分组倍率'}</Button></div>
    </Section>
  );
}
