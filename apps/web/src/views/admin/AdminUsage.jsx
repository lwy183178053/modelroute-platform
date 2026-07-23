import { useMemo, useState } from 'react';
import { UsageLogTable, UsageTrend, useUsage } from '../UsagePanels';
import { Select, TextInput } from '../../ui';

export default function AdminUsage() {
  const [merchant, setMerchant] = useState('');
  const [channel, setChannel] = useState('');
  const endpoint = useMemo(() => {
    const query = new URLSearchParams();
    if (merchant.trim()) query.set('username', merchant.trim());
    if (channel.trim()) query.set('channel', channel.trim());
    return `/api/admin/channel-logs${query.toString() ? `?${query}` : ''}`;
  }, [merchant, channel]);
  const usage = useUsage(endpoint);
  return (
    <div className="admin-section-stack">
      <UsageTrend data={usage.data} error={usage.error} loading={usage.loading} title="渠道调用趋势" />
      <div className="filter-toolbar"><TextInput aria-label="筛选商户" placeholder="输入商户账号" value={merchant} onChange={(event) => setMerchant(event.target.value)} /><TextInput aria-label="筛选渠道" placeholder="输入 New API 渠道编号" value={channel} onChange={(event) => setChannel(event.target.value)} /><Select aria-label="日志范围"><option>最近 7 天</option></Select></div>
      <UsageLogTable admin data={usage.data} error={usage.error} loading={usage.loading} onRefresh={usage.load} title="渠道调用日志" />
    </div>
  );
}
