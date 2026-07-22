import {
  Activity,
  ArrowUpRight,
  BadgeCheck,
  CircleDollarSign,
  Coins,
  KeyRound,
  Plus,
  Route,
  Settings2,
  Store,
  WalletCards,
} from 'lucide-react';
import { recentCalls } from '../data';
import { Button, CopyValue, InlineNotice, Metric, PageHeader, Section, Status } from '../ui';

export default function ConsoleView({ openModal, notify, navigate }) {
  return (
    <div className="view-shell">
      <PageHeader
        eyebrow="用户工作台"
        title="账户概览"
        description="查看余额、调用、缓存命中与每个 API 密钥的路由策略。"
        actions={(
          <>
            <Button icon={Settings2}>账单设置</Button>
            <Button icon={Plus} variant="primary" onClick={() => openModal('recharge')}>立即充值</Button>
          </>
        )}
      />

      <div className="metrics-grid">
        <Metric label="账户余额" value="¥1,286.42" hint="可用额度 128.6M" icon={WalletCards} />
        <Metric label="今日消费" value="¥42.68" hint="较昨日下降 12.4%" icon={Coins} tone="blue" />
        <Metric label="今日调用" value="2,846" hint="成功 2,831 次" icon={Activity} tone="green" />
        <Metric label="缓存节省" value="¥18.32" hint="命中率 67.8%" icon={CircleDollarSign} tone="amber" />
      </div>

      <div className="dashboard-grid dashboard-grid-main">
        <Section
          title="用量趋势"
          description="最近 7 天 New API 最终结算额度"
          actions={<select className="mini-select" defaultValue="消费金额"><option>消费金额</option><option>调用次数</option><option>Token 用量</option></select>}
        >
          <div className="chart-shell" aria-label="最近七天用量柱状图">
            <div className="chart-scale"><span>¥80</span><span>¥60</span><span>¥40</span><span>¥20</span><span>¥0</span></div>
            <div className="chart-bars">
              {[48, 62, 39, 74, 58, 81, 54].map((height, index) => (
                <div className="chart-column" key={index}><div className="chart-bar" style={{ height: `${height}%` }} /><span>{['周三', '周四', '周五', '周六', '周日', '周一', '今天'][index]}</span></div>
              ))}
            </div>
          </div>
        </Section>

        <Section title="账户状态" description="统一账号可同时拥有用户与商户身份">
          <div className="account-summary">
            <div className="account-avatar">林</div>
            <div><strong>林先生</strong><span>UID 10826 · 管理员账号</span></div>
            <Status>正常</Status>
          </div>
          <div className="account-rows">
            <div><span>实名状态</span><strong><BadgeCheck size={15} /> 已认证</strong></div>
            <div><span>商户身份</span><strong><Store size={15} /> 已开通</strong></div>
            <div><span>安全等级</span><strong>高</strong></div>
          </div>
          <Button className="full-button" icon={ArrowUpRight} onClick={() => navigate('merchant')}>进入商户中心</Button>
          <Button className="full-button" variant="ghost" onClick={() => openModal('merchantApply')}>查看商户申请表</Button>
        </Section>
      </div>

      <Section
        title="API 密钥与路由"
        description="路由策略按 API 密钥独立绑定，最终扣费继续由 New API 处理"
        actions={<Button icon={KeyRound} variant="primary">创建密钥</Button>}
      >
        <InlineNotice>外围平台只在请求进入 New API 前选择商户渠道并替换内部模型别名，不重新计算 Token 或缓存费用。</InlineNotice>
        <div className="table-wrap">
          <table className="data-table">
            <thead><tr><th>名称 / 密钥</th><th>路由方式</th><th>倍率区间</th><th>优先规则</th><th>固定商户</th><th>今日消费</th><th>状态</th><th></th></tr></thead>
            <tbody>
              <tr>
                <td><strong>生产环境</strong><CopyValue onCopy={() => notify('密钥已复制')}>sk-live-7J••••••••sF2</CopyValue></td>
                <td><span className="route-kind"><Route size={15} /> 智能路由</span></td><td>0.40x - 0.80x</td><td>稳定性优先</td><td>全部合格商户</td><td>¥38.42</td><td><Status>正常</Status></td>
                <td><Button variant="ghost" onClick={() => openModal('route')}>编辑路由策略</Button></td>
              </tr>
              <tr>
                <td><strong>测试环境</strong><CopyValue onCopy={() => notify('密钥已复制')}>sk-test-2C••••••••mA9</CopyValue></td>
                <td><span className="route-kind"><Store size={15} /> 固定商户</span></td><td>0.50x - 0.72x</td><td>价格优先</td><td>北辰云智、青川 API</td><td>¥4.26</td><td><Status>正常</Status></td>
                <td><Button variant="ghost" onClick={() => openModal('route')}>编辑路由策略</Button></td>
              </tr>
            </tbody>
          </table>
        </div>
      </Section>

      <Section title="最近调用" description="金额、退款与缓存命中结果来自 New API 最终日志" actions={<Button variant="ghost">查看全部日志</Button>}>
        <div className="table-wrap">
          <table className="data-table">
            <thead><tr><th>请求编号</th><th>模型</th><th>成交商户</th><th>Token</th><th>缓存</th><th>最终扣费</th><th>耗时</th><th>时间</th><th>结果</th></tr></thead>
            <tbody>{recentCalls.map((call) => <tr key={call.id}><td><code>{call.id}</code></td><td><strong>{call.model}</strong></td><td>{call.merchant}</td><td>{call.tokens}</td><td>{call.cache}</td><td><strong>{call.cost}</strong></td><td>{call.duration}</td><td>{call.time}</td><td><Status>{call.state}</Status></td></tr>)}</tbody>
          </table>
        </div>
      </Section>
    </div>
  );
}
