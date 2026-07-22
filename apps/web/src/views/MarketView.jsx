import { useMemo, useState } from 'react';
import { Activity, ArrowDownUp, BookOpen, Boxes, Clock3, Gauge, Search, Store } from 'lucide-react';
import { marketOffers } from '../data';
import { Button, Metric, PageHeader, ProviderLogo, Section, Select, Status, TextInput } from '../ui';

const providers = ['全部', 'OpenAI', 'Anthropic', 'Gemini', 'DeepSeek', 'Qwen'];

export default function MarketView({ openRoute, navigate }) {
  const [provider, setProvider] = useState('全部');
  const [query, setQuery] = useState('');
  const [range, setRange] = useState('24 小时');
  const offers = useMemo(
    () => marketOffers.filter((item) => (
      (provider === '全部' || item.provider === provider) &&
      `${item.model}${item.merchant}`.toLowerCase().includes(query.toLowerCase())
    )),
    [provider, query],
  );

  return (
    <div className="view-shell market-view">
      <PageHeader
        eyebrow="实时市场"
        title="模型服务市场"
        description="同一模型按商户报价、稳定性和实际运行数据透明比较。"
        actions={(
          <>
            <Button icon={BookOpen}>接入文档</Button>
            <Button icon={Gauge} variant="primary" onClick={() => navigate('console')}>进入控制台</Button>
          </>
        )}
      />

      <div className="metrics-row market-metrics">
        <Metric label="在售模型" value="126" hint="覆盖 12 家模型厂商" icon={Boxes} />
        <Metric label="活跃商户" value="38" hint="今日新增 3 家" icon={Store} tone="blue" />
        <Metric label="24h 有效调用" value="426.8K" hint="成功率 99.74%" icon={Activity} tone="green" />
        <Metric label="中位首字时间" value="1.34s" hint="较昨日下降 8.2%" icon={Clock3} tone="amber" />
      </div>

      <Section className="market-board">
        <div className="market-tabs">
          <div className="segmented compact" role="group" aria-label="统计周期">
            {['实时', '24 小时', '7 天', '30 天'].map((item) => (
              <button className={range === item ? 'active' : ''} key={item} onClick={() => setRange(item)} type="button">{item}</button>
            ))}
          </div>
          <span className="refresh-time">运行数据更新于 18 秒前</span>
        </div>

        <div className="filter-toolbar">
          <label className="search-box">
            <Search size={17} />
            <TextInput aria-label="搜索模型或商户" placeholder="搜索模型或商户" value={query} onChange={(event) => setQuery(event.target.value)} />
          </label>
          <Select aria-label="模型能力" defaultValue="全部能力">
            <option>全部能力</option><option>聊天</option><option>推理</option><option>多模态</option><option>嵌入</option>
          </Select>
          <Select aria-label="排序方式" defaultValue="综合排序">
            <option>综合排序</option><option>价格从低到高</option><option>可用率从高到低</option><option>调用量从高到低</option>
          </Select>
          <Button icon={ArrowDownUp}>筛选</Button>
        </div>

        <div className="provider-pills" aria-label="模型厂商">
          {providers.map((item) => (
            <button className={provider === item ? 'active' : ''} key={item} onClick={() => setProvider(item)} type="button">
              {item !== '全部' && <ProviderLogo provider={item} size={17} />}
              {item}
            </button>
          ))}
        </div>

        <div className="table-wrap market-table-wrap">
          <table className="data-table market-table">
            <thead>
              <tr>
                <th>排名</th><th>模型与商户</th><th>状态</th><th>可用率</th><th>首字时间</th><th>调用 / Token</th><th>缓存命中</th><th>成交倍率</th><th>用户价格</th><th></th>
              </tr>
            </thead>
            <tbody>
              {offers.map((offer) => (
                <tr key={`${offer.model}-${offer.merchant}`}>
                  <td><span className={`rank rank-${offer.rank}`}>{offer.rank}</span></td>
                  <td>
                    <div className="model-cell">
                      <ProviderLogo provider={offer.provider} size={23} />
                      <div><strong>{offer.model}</strong><span>{offer.merchant} · {offer.tag}</span></div>
                    </div>
                  </td>
                  <td><Status>{offer.state}</Status></td>
                  <td><strong className="success-text">{offer.availability}</strong></td>
                  <td>{offer.latency}</td>
                  <td><strong>{offer.calls}</strong><small>{offer.tokens} Token</small></td>
                  <td>{offer.cache}</td>
                  <td><span className="price-multiple">{offer.multiplier}</span></td>
                  <td><strong>{offer.inputPrice}</strong><small>输出 {offer.outputPrice}</small></td>
                  <td><Button variant="ghost" onClick={openRoute}>用于路由</Button></td>
                </tr>
              ))}
            </tbody>
          </table>
          {!offers.length && <div className="table-empty">没有符合当前筛选条件的报价</div>}
        </div>
      </Section>
    </div>
  );
}
