import {
  Activity,
  Banknote,
  CircleDollarSign,
  CloudCog,
  Coins,
  HeartPulse,
  Plus,
  RefreshCw,
  Settings2,
  WalletCards,
} from 'lucide-react';
import { merchantChannels, merchantQuotes } from '../data';
import { Button, Metric, PageHeader, ProviderLogo, Section, Status } from '../ui';

export default function MerchantView({ openModal, notify }) {
  return (
    <div className="view-shell">
      <PageHeader
        eyebrow="商户中心 · 北辰云智"
        title="商户经营概览"
        description="管理已审核渠道、模型报价、运行状态、结算收入与人工提现。"
        actions={(
          <>
            <Button icon={Banknote} onClick={() => openModal('withdraw')}>申请提现</Button>
            <Button icon={Plus} variant="primary" onClick={() => openModal('channel')}>提交新渠道</Button>
          </>
        )}
      />

      <div className="metrics-grid">
        <Metric label="可提现余额" value="¥18,642.30" hint="冻结中 ¥5,000.00" icon={WalletCards} />
        <Metric label="今日成交额" value="¥1,284.62" hint="商户预计收入 ¥1,156.15" icon={Coins} tone="blue" />
        <Metric label="平台抽成" value="¥128.47" hint="固定 10%" icon={CircleDollarSign} tone="amber" />
        <Metric label="综合可用率" value="99.62%" hint="2 个健康 · 1 个冷却" icon={HeartPulse} tone="green" />
      </div>

      <Section
        title="渠道管理"
        description="审核通过后由外围平台导入 New API，并保存渠道与商户归属"
        actions={<><Button icon={RefreshCw} onClick={() => notify('已发起全部渠道探活')}>全部探活</Button><Button icon={CloudCog}>渠道设置</Button></>}
      >
        <div className="table-wrap">
          <table className="data-table">
            <thead><tr><th>渠道</th><th>New API 编号</th><th>模型数</th><th>24h 可用率</th><th>首字时间</th><th>最近探活</th><th>状态</th><th></th></tr></thead>
            <tbody>{merchantChannels.map((channel) => (
              <tr key={channel.channel}>
                <td><div className="model-cell"><ProviderLogo provider={channel.provider} size={21} /><div><strong>{channel.name}</strong><span>密钥已加密 · 指纹 7d:2a:91</span></div></div></td>
                <td><code>{channel.channel}</code></td><td>{channel.models}</td><td><strong>{channel.availability}</strong></td><td>{channel.latency}</td><td>{channel.checked}</td><td><Status>{channel.state}</Status></td>
                <td><div className="row-actions"><Button variant="ghost" onClick={() => notify(`${channel.name} 探活已排队`)}>探活</Button><Button variant="ghost">管理</Button></div></td>
              </tr>
            ))}</tbody>
          </table>
        </div>
      </Section>

      <Section
        title="模型报价"
        description="报价是用户最终成交倍率，提交值必须大于 0 且不超过官方价 1.0x"
        actions={<Button icon={Plus} variant="primary">新增报价</Button>}
      >
        <div className="table-wrap">
          <table className="data-table">
            <thead><tr><th>模型</th><th>渠道</th><th>官方输入 / 输出</th><th>成交倍率</th><th>用户输入 / 输出</th><th>30 天调用</th><th>商户收入</th><th>状态</th><th></th></tr></thead>
            <tbody>{merchantQuotes.map((quote) => (
              <tr key={quote.model}><td><strong>{quote.model}</strong></td><td>{quote.channel}</td><td>{quote.official}</td><td><span className="price-multiple">{quote.multiplier}</span></td><td><strong>{quote.final}</strong></td><td>{quote.calls}</td><td><strong>{quote.revenue}</strong></td><td><Status>{quote.state}</Status></td><td><Button variant="ghost" onClick={() => openModal('quote')}>调整报价</Button></td></tr>
            ))}</tbody>
          </table>
        </div>
      </Section>

      <div className="dashboard-grid merchant-bottom-grid">
        <Section title="收入构成" description="基于 New API 最终消费与退款日志拆分">
          <div className="settlement-chart">
            <div className="donut" aria-label="收入构成图"><span>90%<small>商户收入</small></span></div>
            <div className="legend-list">
              <div><i className="legend-teal" /><span>模型调用收入</span><strong>¥21,426.15</strong></div>
              <div><i className="legend-blue" /><span>缓存命中收入</span><strong>¥3,812.68</strong></div>
              <div><i className="legend-red" /><span>退款与调整</span><strong>-¥296.41</strong></div>
              <div><i className="legend-gray" /><span>平台抽成</span><strong>-¥2,494.24</strong></div>
            </div>
          </div>
        </Section>
        <Section title="最近结算" description="账本只追加，退款按原消费拆分结果冲销" actions={<Button icon={Settings2} variant="ghost">查看账本</Button>}>
          <div className="compact-ledger">
            <div><Activity size={16} /><span>GPT-5.2 调用收入</span><strong>+¥8.42</strong><small>刚刚</small></div>
            <div><Activity size={16} /><span>Claude Sonnet 4.5 调用收入</span><strong>+¥5.16</strong><small>1 分钟前</small></div>
            <div className="negative"><RefreshCw size={16} /><span>Gemini 3 Pro 退款冲销</span><strong>-¥1.28</strong><small>6 分钟前</small></div>
          </div>
        </Section>
      </div>
    </div>
  );
}
