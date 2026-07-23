import { useEffect, useMemo, useState } from 'react';
import { Activity, Boxes, Clock3, Gauge, Search, Store } from 'lucide-react';
import { apiRequest, formatCompactCount, formatLastCalled } from '../api';
import { Button, InlineNotice, Metric, PageHeader, Pagination, ProviderLogo, Section, Select, Status, TextInput } from '../ui';

const healthNames = {
  pending: '待探测', healthy: '健康', degraded: '降级', cooling: '冷却', recovering: '恢复中', disabled: '已停用',
};

const protocolNames = {
  chat_completions: '聊天补全', responses: 'Responses', image_generation: '图片生成',
};

function formatAvailability(value) {
  return `${(Number(value || 0) / 10_000).toFixed(2)}%`;
}

function formatLatency(value) {
  const milliseconds = Number(value || 0);
  if (!milliseconds) return '-';
  return milliseconds >= 1000 ? `${(milliseconds / 1000).toFixed(2)}s` : `${milliseconds}ms`;
}

function formatMultiplier(value) {
  const bps = Number(value || 0);
  return bps === 0 ? '免费' : `${(bps / 10_000).toFixed(2)}x`;
}

function offerInputPrice(offer) {
  if (offer.protocol === 'image_generation') {
    return Math.min(...(offer.image_prices || []).map((item) => Number(item.price_micros)), Number.MAX_SAFE_INTEGER);
  }
  return Number(offer.price_tiers?.[0]?.input_price_micros ?? Number.MAX_SAFE_INTEGER);
}

export default function MarketView({ navigate }) {
  const [models, setModels] = useState([]);
  const [marketOffers, setMarketOffers] = useState([]);
  const [provider, setProvider] = useState('全部');
  const [protocol, setProtocol] = useState('all');
  const [sort, setSort] = useState('stability');
  const [query, setQuery] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    let active = true;
    Promise.all([apiRequest('/api/models'), apiRequest('/api/offers')])
      .then(([modelData, offerData]) => {
        if (!active) return;
        setModels(modelData || []);
        setMarketOffers(offerData?.items || offerData || []);
      })
      .catch((requestError) => { if (active) setError(requestError.message || '读取模型市场失败'); })
      .finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, []);

  const modelMap = useMemo(() => new Map(models.map((model) => [model.public_name, model])), [models]);
  const providers = useMemo(
    () => ['全部', ...new Set(models.map((model) => model.provider).filter(Boolean))],
    [models],
  );
  const offers = useMemo(() => {
    const filtered = marketOffers.filter((offer) => {
      if (Number(offer.channel_count || 0) <= 0) return false;
      const model = modelMap.get(offer.model);
      return (provider === '全部' || model?.provider === provider)
        && (protocol === 'all' || offer.protocol === protocol)
        && `${offer.model}${offer.merchant_name}${offer.channel_name}`.toLowerCase().includes(query.trim().toLowerCase());
    });
    return [...filtered].sort((left, right) => {
      if (sort === 'price') return offerInputPrice(left) - offerInputPrice(right);
      if (sort === 'availability') return right.availability_ppm - left.availability_ppm;
      return right.availability_ppm - left.availability_ppm || left.ttft_p95_ms - right.ttft_p95_ms || offerInputPrice(left) - offerInputPrice(right);
    });
  }, [marketOffers, modelMap, protocol, provider, query, sort]);

  useEffect(() => { setPage(1); }, [provider, protocol, sort, query]);

  const visibleOffers = offers.slice((page - 1) * pageSize, page * pageSize);

  const activeModels = new Set(marketOffers.map((offer) => offer.model)).size;
  const activeMerchants = new Set(marketOffers.map((offer) => offer.merchant_id)).size;
  const healthyOffers = marketOffers.filter((offer) => offer.health_state === 'healthy').length;
  const p95Values = marketOffers.map((offer) => Number(offer.ttft_p95_ms)).filter((value) => value > 0).sort((a, b) => a - b);
  const medianP95 = p95Values.length ? p95Values[Math.floor((p95Values.length - 1) / 2)] : 0;

  return (
    <div className="view-shell market-view">
      <PageHeader
        eyebrow="实时市场"
        title="模型服务市场"
        description="同一模型按商户报价、稳定性和实际运行数据透明比较。"
        actions={<Button icon={Gauge} variant="primary" onClick={() => navigate('console')}>进入控制台</Button>}
      />

      {error && <InlineNotice tone="warning">{error}</InlineNotice>}
      <div className="metrics-row market-metrics">
        <Metric label="在售模型" value={String(activeModels)} hint={`已发布 ${models.length} 个标准模型`} icon={Boxes} />
        <Metric label="活跃商户" value={String(activeMerchants)} hint={`${marketOffers.length} 个有效报价`} icon={Store} tone="blue" />
        <Metric label="健康报价" value={String(healthyOffers)} hint="来自真实请求与冷却恢复检测" icon={Activity} tone="green" />
        <Metric label="中位 P95 首字" value={formatLatency(medianP95)} hint="按当前有效报价统计" icon={Clock3} tone="amber" />
      </div>

      <Section className="market-board">
        <div className="filter-toolbar">
          <label className="search-box">
            <Search size={17} />
            <TextInput aria-label="搜索模型或商户" placeholder="搜索模型、商户或渠道" value={query} onChange={(event) => setQuery(event.target.value)} />
          </label>
          <Select aria-label="接口协议" value={protocol} onChange={(event) => setProtocol(event.target.value)}>
            <option value="all">全部协议</option><option value="chat_completions">聊天补全</option><option value="responses">Responses</option><option value="image_generation">图片生成</option>
          </Select>
          <Select aria-label="排序方式" value={sort} onChange={(event) => setSort(event.target.value)}>
            <option value="stability">稳定性优先</option><option value="price">价格从低到高</option><option value="availability">可用率从高到低</option>
          </Select>
        </div>

        <div className="provider-pills" aria-label="模型厂商">
          {providers.map((item) => (
            <button className={provider === item ? 'active' : ''} key={item} onClick={() => setProvider(item)} type="button">
              {item !== '全部' && <ProviderLogo provider={item} size={17} />}{item}
            </button>
          ))}
        </div>

        <div className="table-wrap market-table-wrap">
          <table className="data-table market-table">
            <thead><tr><th>ID</th><th>排名</th><th>模型与商户</th><th>协议</th><th>状态</th><th>可用率</th><th>P95 首字</th><th>调用次数</th><th>Token 用量</th><th>最后调用</th><th>倍率</th><th>可用渠道</th></tr></thead>
            <tbody>
              {visibleOffers.map((offer, index) => {
                const model = modelMap.get(offer.model);
                return <tr key={offer.offer_id}>
                  <td><code>{offer.offer_id}</code></td>
                  <td><span className={`rank rank-${Math.min((page - 1) * pageSize + index + 1, 5)}`}>{(page - 1) * pageSize + index + 1}</span></td>
                  <td><div className="model-cell"><ProviderLogo provider={model?.provider || ''} size={23} /><div><strong>{offer.model}</strong><span>{offer.merchant_name}</span></div></div></td>
                  <td>{protocolNames[offer.protocol] || offer.protocol}</td>
                  <td><Status>{healthNames[offer.health_state] || offer.health_state}</Status></td>
                  <td><strong className="success-text">{formatAvailability(offer.availability_ppm)}</strong></td>
                  <td>{formatLatency(offer.ttft_p95_ms)}</td>
                  <td>{formatCompactCount(offer.call_count)}</td>
                  <td>{formatCompactCount(offer.token_usage)}</td>
                  <td>{formatLastCalled(offer.last_called_at)}</td>
                  <td><span className="price-multiple">{formatMultiplier(offer.multiplier_bps)}</span></td>
                  <td>{offer.channel_count || 0} 个</td>
                </tr>;
              })}
            </tbody>
          </table>
          {!loading && !offers.length && <div className="table-empty">{error || '没有符合当前筛选条件的报价'}</div>}
          {loading && <div className="table-empty">正在读取真实报价</div>}
        </div>
        <Pagination page={page} pageSize={pageSize} total={offers.length} onPageChange={setPage} onPageSizeChange={(nextPageSize) => { setPageSize(nextPageSize); setPage(1); }} disabled={loading} />
      </Section>
    </div>
  );
}
