import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Activity,
  BadgeDollarSign,
  Building2,
  ClipboardCheck,
  CreditCard,
  FileText,
  HeartPulse,
  Landmark,
  Megaphone,
  Settings2,
  ShieldCheck,
  TicketCheck,
  UserRoundCog,
} from 'lucide-react';
import { apiRequest, formatQuota } from '../api';
import { Button, InlineNotice, Metric, PageHeader, Section, Status } from '../ui';
import AdminMerchants from './admin/AdminMerchants';
import AdminChannels from './admin/AdminChannels';
import AdminPayment from './admin/AdminPayment';
import AdminModels from './admin/AdminModels';
import AdminWithdrawals from './admin/AdminWithdrawals';
import AdminUsers from './admin/AdminUsers';
import AdminRedemptions from './admin/AdminRedemptions';
import AdminSettings from './admin/AdminSettings';
import AdminUsage from './admin/AdminUsage';
import AdminAnnouncements from './admin/AdminAnnouncements';

const baseSections = [
  { id: 'overview', label: '运营总览', icon: Activity },
  { id: 'users', label: '用户管理', icon: UserRoundCog },
  { id: 'merchants', label: '商户审核', icon: Building2 },
  { id: 'models', label: '模型与定价', icon: BadgeDollarSign },
  { id: 'channels', label: '渠道与探活', icon: HeartPulse },
  { id: 'usage', label: '渠道调用日志', icon: FileText },
  { id: 'payment', label: '支付与充值', icon: CreditCard },
  { id: 'redemptions', label: '兑换码管理', icon: TicketCheck },
  { id: 'withdrawals', label: '提现审核', icon: Landmark },
  { id: 'announcements', label: '公告管理', icon: Megaphone },
  { id: 'settings', label: '平台设置', icon: Settings2 },
];

function serviceName(key) {
  return { database: '平台数据库', redis: 'Redis 会话服务', new_api: '模型计费服务' }[key] || key;
}

function serviceStatus(status) {
  return { healthy: '正常', unhealthy: '异常', unavailable: '未就绪' }[status] || status || '未就绪';
}

function successRate(successful, total) {
  if (!Number(total)) return '0.00%';
  return `${((Number(successful) / Number(total)) * 100).toFixed(2)}%`;
}

function AdminOverview({ data, error, loading, reload, setSection }) {
  const summary = data?.summary || {};
  const pendingCount = Number(summary.pending_merchant_applications || 0)
    + Number(summary.pending_withdrawals || 0);
  const channelAttention = Number(summary.cooling_models || 0)
    + Number(summary.error_models || 0);
  const syncHealthy = !summary.last_sync_error;

  if (loading) return <div className="table-message">正在读取运营总览</div>;
  if (error) return <InlineNotice tone="warning">{error}<Button onClick={reload} variant="ghost">重新读取</Button></InlineNotice>;

  return (
    <>
      <div className="metrics-grid admin-metrics">
        <Metric label="今日平台成交" value={formatQuota(summary.today_gross_quota)} hint={`平台收入 ${formatQuota(summary.today_platform_quota)}`} icon={BadgeDollarSign} />
        <Metric label="活跃渠道" value={`${summary.active_channels || 0} / ${summary.total_channels || 0}`} hint={`${summary.cooling_models || 0} 个模型冷却 · ${summary.error_models || 0} 个错误`} icon={HeartPulse} tone="green" />
        <Metric label="待处理事项" value={String(pendingCount)} hint="商户与提现" icon={ClipboardCheck} tone="amber" />
        <Metric label="计费同步" value={syncHealthy ? '正常' : '异常'} hint={`日志延迟 ${Number(data?.sync_delay_seconds || 0)} 秒`} icon={ShieldCheck} tone="blue" />
      </div>

      <div className="dashboard-grid admin-overview-grid">
        <Section title="待办事项" description="来自当前平台业务状态">
          <div className="task-list">
            <button onClick={() => setSection('merchants')} type="button"><span className="task-icon task-warning"><Building2 size={17} /></span><span><strong>商户入驻待审核</strong><small>等待验证或批准</small></span><Status>{summary.pending_merchant_applications || 0} 项</Status></button>
            <button onClick={() => setSection('withdrawals')} type="button"><span className="task-icon task-blue"><Landmark size={17} /></span><span><strong>提现待处理</strong><small>等待审核或确认打款</small></span><Status>{summary.pending_withdrawals || 0} 笔</Status></button>
            <button onClick={() => setSection('channels')} type="button"><span className="task-icon task-warning"><HeartPulse size={17} /></span><span><strong>模型状态关注</strong><small>{summary.cooling_models || 0} 个冷却，{summary.error_models || 0} 个错误</small></span><Status>{channelAttention} 项</Status></button>
          </div>
        </Section>

        <Section title="核心服务状态" description="平台依赖的当前连接状态">
          <div className="service-list">
            <div><span><i className="service-dot healthy" />外围路由网关</span><strong>正常</strong><small>当前页面可用</small></div>
            {Object.entries(data?.services || {}).map(([key, service]) => <div key={key}><span><i className={`service-dot ${service.status === 'healthy' ? 'healthy' : ''}`} />{serviceName(key)}</span><strong>{serviceStatus(service.status)}</strong><small>{Number(service.latency_ms || 0)} ms</small></div>)}
            <div><span><i className={`service-dot ${summary.payment_enabled ? 'healthy' : ''}`} />易支付网关</span><strong>{summary.payment_enabled ? '已启用' : '未启用'}</strong><small>见支付设置</small></div>
          </div>
        </Section>
      </div>

      <Section title="实时交易概况" description="数据以成功调用记录为准" actions={<Button onClick={() => setSection('usage')} variant="ghost">查看完整报表</Button>}>
        <div className="transaction-band">
          <div><span>请求总数</span><strong>{summary.today_request_count || 0}</strong><small>今日路由请求</small></div>
          <div><span>成功请求</span><strong>{summary.today_successful_requests || 0} / {summary.today_request_count || 0}</strong><small>{successRate(summary.today_successful_requests, summary.today_request_count)}</small></div>
          <div><span>用户消费</span><strong>{formatQuota(summary.today_gross_quota)}</strong><small>成功调用自动扣费</small></div>
          <div><span>商户收入</span><strong>{formatQuota(summary.today_merchant_quota)}</strong><small>消费完成后自动入账</small></div>
          <div><span>平台收入</span><strong>{formatQuota(summary.today_platform_quota)}</strong><small>成交额的 10%</small></div>
        </div>
      </Section>
    </>
  );
}

export default function AdminView({ section, setSection, openModal, notify }) {
  const [overview, setOverview] = useState(null);
  const [overviewLoading, setOverviewLoading] = useState(true);
  const [overviewError, setOverviewError] = useState('');

  const loadOverview = useCallback(async () => {
    setOverviewLoading(true);
    setOverviewError('');
    try {
      setOverview(await apiRequest('/api/admin/overview'));
    } catch (requestError) {
      setOverviewError(requestError.message || '读取运营总览失败');
    } finally {
      setOverviewLoading(false);
    }
  }, []);

  useEffect(() => { loadOverview(); }, [loadOverview]);

  const sections = useMemo(() => {
    const summary = overview?.summary || {};
    const counts = {
      merchants: Number(summary.pending_merchant_applications || 0),
      withdrawals: Number(summary.pending_withdrawals || 0),
    };
    return baseSections.map((item) => ({ ...item, count: counts[item.id] || 0 }));
  }, [overview]);

  return (
    <div className="view-shell admin-view">
      <PageHeader eyebrow="平台管理" title="运营管理" description="统一处理用户、商户、渠道、模型、支付和提现业务。" />
      <div className="admin-layout">
        <aside className="admin-sidebar" aria-label="管理员功能">
          {sections.map(({ id, label, icon: Icon, count }) => <button className={section === id ? 'active' : ''} key={id} onClick={() => { setSection(id); if (id === 'withdrawals') loadOverview(); }} type="button"><Icon size={17} /><span>{label}</span>{count > 0 && <b>{count}</b>}</button>)}
        </aside>
        <div className="admin-content">
          {section === 'overview' && <AdminOverview data={overview} error={overviewError} loading={overviewLoading} reload={loadOverview} setSection={setSection} />}
          {section === 'usage' && <AdminUsage />}
          {section === 'users' && <AdminUsers notify={notify} openModal={openModal} />}
          {section === 'merchants' && <AdminMerchants notify={notify} />}
          {section === 'models' && <AdminModels notify={notify} />}
          {section === 'channels' && <AdminChannels notify={notify} />}
          {section === 'payment' && <AdminPayment notify={notify} openModal={openModal} />}
          {section === 'redemptions' && <AdminRedemptions notify={notify} />}
          {section === 'withdrawals' && <AdminWithdrawals notify={notify} onSummaryRefresh={loadOverview} openModal={openModal} />}
          {section === 'announcements' && <AdminAnnouncements notify={notify} />}
          {section === 'settings' && <AdminSettings notify={notify} />}
        </div>
      </div>
    </div>
  );
}
