import {
  Activity,
  BadgeDollarSign,
  Building2,
  ClipboardCheck,
  CreditCard,
  FileClock,
  HeartPulse,
  Landmark,
  Settings2,
  ShieldCheck,
  UserRoundCog,
} from 'lucide-react';
import { Button, Metric, PageHeader, Section, Status } from '../ui';
import AdminMerchants from './admin/AdminMerchants';
import AdminChannels from './admin/AdminChannels';
import AdminPayment from './admin/AdminPayment';
import AdminModels from './admin/AdminModels';
import AdminWithdrawals from './admin/AdminWithdrawals';
import AdminUsers from './admin/AdminUsers';
import AdminAudit from './admin/AdminAudit';

const sections = [
  { id: 'overview', label: '运营总览', icon: Activity },
  { id: 'users', label: '用户管理', icon: UserRoundCog },
  { id: 'merchants', label: '商户审核', icon: Building2, count: 3 },
  { id: 'models', label: '模型与定价', icon: BadgeDollarSign },
  { id: 'channels', label: '渠道与探活', icon: HeartPulse, count: 2 },
  { id: 'payment', label: '支付与充值', icon: CreditCard },
  { id: 'withdrawals', label: '提现审核', icon: Landmark, count: 2 },
  { id: 'settings', label: '平台设置', icon: Settings2 },
  { id: 'audit', label: '审计日志', icon: FileClock },
];

function AdminOverview({ setSection }) {
  return (
    <>
      <div className="metrics-grid admin-metrics">
        <Metric label="今日平台成交" value="¥42,684.20" hint="平台收入 ¥4,268.42" icon={BadgeDollarSign} />
        <Metric label="活跃渠道" value="86 / 92" hint="2 个冷却 · 4 个停用" icon={HeartPulse} tone="green" />
        <Metric label="待处理事项" value="7" hint="审核、提现与渠道事项" icon={ClipboardCheck} tone="amber" />
        <Metric label="New API 同步" value="正常" hint="日志延迟 16 秒" icon={ShieldCheck} tone="blue" />
      </div>

      <div className="dashboard-grid admin-overview-grid">
        <Section title="待办事项" description="按业务风险和等待时间排序">
          <div className="task-list">
            <button onClick={() => setSection('merchants')} type="button"><span className="task-icon task-warning"><Building2 size={17} /></span><span><strong>商户入驻待审核</strong><small>最长等待 2 小时 16 分钟</small></span><Status>3 项</Status></button>
            <button onClick={() => setSection('withdrawals')} type="button"><span className="task-icon task-blue"><Landmark size={17} /></span><span><strong>人工提现待处理</strong><small>申请总额 ¥13,000.00</small></span><Status>2 笔</Status></button>
            <button onClick={() => setSection('channels')} type="button"><span className="task-icon task-warning"><HeartPulse size={17} /></span><span><strong>渠道状态异常</strong><small>1 个冷却，1 个密钥余额不足</small></span><Status>2 项</Status></button>
          </div>
        </Section>

        <Section title="核心服务状态" description="外围平台与 New API 的当前连接状态">
          <div className="service-list">
            <div><span><i className="service-dot healthy" />外围路由网关</span><strong>正常</strong><small>18 ms</small></div>
            <div><span><i className="service-dot healthy" />New API 内部服务</span><strong>正常</strong><small>24 ms</small></div>
            <div><span><i className="service-dot healthy" />用量日志同步</span><strong>正常</strong><small>延迟 16 秒</small></div>
            <div><span><i className="service-dot" />易支付网关</span><strong>按配置状态</strong><small>见支付设置</small></div>
          </div>
        </Section>
      </div>

      <Section title="实时交易概况" description="数据以 New API 最终结算日志为准" actions={<Button variant="ghost">查看完整报表</Button>}>
        <div className="transaction-band">
          <div><span>请求总数</span><strong>426,842</strong><small>较昨日 +12.8%</small></div>
          <div><span>成功率</span><strong>99.74%</strong><small>目标 ≥ 99.5%</small></div>
          <div><span>用户扣费</span><strong>¥42,684.20</strong><small>含缓存与固定计费</small></div>
          <div><span>商户应计</span><strong>¥38,415.78</strong><small>成交额的 90%</small></div>
          <div><span>平台收入</span><strong>¥4,268.42</strong><small>成交额的 10%</small></div>
        </div>
      </Section>
    </>
  );
}

function PlaceholderPanel({ notify }) {
  return (
    <Section
      title="平台基础设置"
      description="官方价格版本、抽成比例与 New API 内部连接"
      actions={<Button onClick={() => notify('平台设置已保存')} variant="primary">保存设置</Button>}
    >
      <div className="settings-list">
        <div><span>平台抽成比例</span><strong>10%</strong><Button variant="ghost">编辑</Button></div>
        <div><span>官方价格版本</span><strong>2026-07-22 v18</strong><Button variant="ghost">发布新版本</Button></div>
        <div><span>New API 内部地址</span><strong>http://new-api.internal:3000</strong><Button variant="ghost">测试连接</Button></div>
        <div><span>日志同步水位</span><strong>#9,482,671 · 延迟 16 秒</strong><Button variant="ghost">立即同步</Button></div>
      </div>
    </Section>
  );
}

export default function AdminView({ section, setSection, openModal, notify }) {
  return (
    <div className="view-shell admin-view">
      <PageHeader eyebrow="平台管理" title="运营管理" description="处理平台外围业务；模型转发、原生计费和余额扣减继续交由 New API。" actions={<Button icon={ShieldCheck}>管理员权限</Button>} />
      <div className="admin-layout">
        <aside className="admin-sidebar" aria-label="管理员功能">
          {sections.map(({ id, label, icon: Icon, count }) => <button className={section === id ? 'active' : ''} key={id} onClick={() => setSection(id)} type="button"><Icon size={17} /><span>{label}</span>{count && <b>{count}</b>}</button>)}
        </aside>
        <div className="admin-content">
          {section === 'overview' && <AdminOverview setSection={setSection} />}
          {section === 'users' && <AdminUsers notify={notify} openModal={openModal} />}
          {section === 'merchants' && <AdminMerchants notify={notify} />}
          {section === 'models' && <AdminModels notify={notify} />}
          {section === 'channels' && <AdminChannels notify={notify} />}
          {section === 'payment' && <AdminPayment notify={notify} openModal={openModal} />}
          {section === 'withdrawals' && <AdminWithdrawals notify={notify} />}
          {section === 'settings' && <PlaceholderPanel notify={notify} />}
          {section === 'audit' && <AdminAudit />}
        </div>
      </div>
    </div>
  );
}
