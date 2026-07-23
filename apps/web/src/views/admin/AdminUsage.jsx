import { useEffect, useMemo, useState } from 'react';
import { apiRequest } from '../../api';
import { InlineNotice, Select } from '../../ui';
import { UsageLogTable, UsageTrend, useUsage } from '../UsagePanels';

export default function AdminUsage() {
  const [channels, setChannels] = useState([]);
  const [merchantUserID, setMerchantUserID] = useState('all');
  const [channelID, setChannelID] = useState('all');
  const [filterError, setFilterError] = useState('');

  useEffect(() => {
    apiRequest('/api/admin/channels?p=1&page_size=100')
      .then((data) => setChannels(data?.items || []))
      .catch((error) => setFilterError(error.message || '读取渠道筛选项失败'));
  }, []);

  const merchants = useMemo(() => {
    const values = new Map();
    channels.forEach((channel) => values.set(String(channel.user_id), channel.merchant_name || `用户 ${channel.user_id}`));
    return [...values.entries()].filter(([id]) => id !== 'undefined');
  }, [channels]);
  const availableChannels = useMemo(
    () => channels.filter((channel) => merchantUserID === 'all' || String(channel.user_id) === merchantUserID),
    [channels, merchantUserID],
  );
  const endpoint = useMemo(() => {
    const query = new URLSearchParams();
    if (channelID !== 'all') query.set('channel_id', channelID);
    if (channelID !== 'all' && merchantUserID !== 'all') query.set('merchant_user_id', merchantUserID);
    return `/api/admin/channel-logs${query.toString() ? `?${query}` : ''}`;
  }, [channelID, merchantUserID]);
  const usage = useUsage(endpoint);

  const selectMerchant = (event) => {
    setMerchantUserID(event.target.value);
    setChannelID('all');
  };

  return (
    <div className="admin-section-stack">
      {filterError && <InlineNotice tone="warning">{filterError}</InlineNotice>}
      <div className="filter-toolbar">
        <Select aria-label="筛选商户" value={merchantUserID} onChange={selectMerchant}><option value="all">全部商户</option>{merchants.map(([id, name]) => <option key={id} value={id}>{name} · 用户 ID {id}</option>)}</Select>
        <Select aria-label="筛选渠道" value={channelID} onChange={(event) => setChannelID(event.target.value)}><option value="all">全部渠道</option>{availableChannels.map((channel) => <option key={channel.id} value={channel.id}>{channel.channel_name} · 渠道 ID {channel.id}</option>)}</Select>
        <Select aria-label="日志范围"><option>最近 7 天</option></Select>
      </div>
      <UsageTrend data={usage.data} error={usage.error} loading={usage.loading} title="渠道调用趋势" />
      <UsageLogTable admin data={usage.data} error={usage.error} loading={usage.loading} onRefresh={usage.load} title="渠道调用日志" />
    </div>
  );
}
