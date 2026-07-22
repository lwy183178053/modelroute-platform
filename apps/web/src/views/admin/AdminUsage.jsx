import { UsageLogTable, UsageTrend, useUsage } from '../UsagePanels';

export default function AdminUsage() {
  const usage = useUsage('/api/admin/usage');
  return (
    <div className="admin-section-stack">
      <UsageTrend data={usage.data} error={usage.error} loading={usage.loading} title="全平台用量趋势" />
      <UsageLogTable admin data={usage.data} error={usage.error} loading={usage.loading} onRefresh={usage.load} title="全平台调用日志" />
    </div>
  );
}
