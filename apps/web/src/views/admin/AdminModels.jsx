import { useCallback, useEffect, useState } from 'react';
import { BadgeDollarSign, Plus, RefreshCw, Trash2 } from 'lucide-react';
import { apiRequest, formatPriceMicros } from '../../api';
import { Button, Field, InlineNotice, Modal, ProviderLogo, Section, Select, Status, TextInput } from '../../ui';

const providers = ['OpenAI', 'Anthropic', 'Gemini', 'DeepSeek', 'Qwen', 'Moonshot', 'Minimax'];

const protocolNames = {
  chat_completions: 'Chat Completions', responses: 'Responses', embeddings: 'Embeddings',
};

const lifecycleNames = {
  stable: '稳定版', preview: 'Preview', limited: '限量', retiring: '临近退役',
};

const toDecimal = (value) => value == null ? '' : String(Number(value) / 1_000_000);

function toMicros(value) {
  if (String(value ?? '').trim() === '') return null;
  const micros = Math.round(Number(value) * 1_000_000);
  return Number.isSafeInteger(micros) && micros >= 0 ? micros : Number.NaN;
}

function toDateTimeLocal(value) {
  const date = value ? new Date(value) : new Date();
  if (Number.isNaN(date.getTime())) return '';
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60_000);
  return local.toISOString().slice(0, 16);
}

function tierState(tier = {}) {
  return {
    maximum: tier.maximum_input_tokens == null ? '' : String(tier.maximum_input_tokens),
    input: toDecimal(tier.input_price_micros),
    output: toDecimal(tier.output_price_micros),
    cacheRead: toDecimal(tier.cache_read_price_micros),
    cacheWrite5m: toDecimal(tier.cache_write_5m_price_micros),
    cacheWrite1h: toDecimal(tier.cache_write_1h_price_micros),
  };
}

function tierMinimum(tiers, index) {
  if (index === 0) return 0;
  const rawMaximum = String(tiers[index - 1].maximum ?? '').trim();
  if (!rawMaximum) return '';
  const previousMaximum = Number(rawMaximum);
  return Number.isSafeInteger(previousMaximum) ? previousMaximum + 1 : '';
}

function formatOfficialPrice(model) {
  const tier = model.price?.tiers?.[0];
  if (!tier) return model.price?.pricing_scheme === 'legacy_fixed' ? '历史固定价格' : '-';
  const input = `${formatPriceMicros(tier.input_price_micros, model.price.currency)} 输入`;
  const output = tier.output_price_micros == null ? '' : ` / ${formatPriceMicros(tier.output_price_micros, model.price.currency)} 输出`;
  return `${input}${output}`;
}

function ModelDialog({ config, close, notify }) {
  const model = config.model;
  const price = model?.price || {};
  const [publicName, setPublicName] = useState('');
  const [provider, setProvider] = useState(model?.provider || 'OpenAI');
  const [protocol, setProtocol] = useState(model?.protocol || 'chat_completions');
  const [lifecycle, setLifecycle] = useState(model?.lifecycle || 'stable');
  const [currency, setCurrency] = useState(price.currency || 'USD');
  const [exchangeRate, setExchangeRate] = useState(toDecimal(price.usd_exchange_rate_micros || 1_000_000));
  const [sourceURL, setSourceURL] = useState(price.source_url || '');
  const [checkedAt, setCheckedAt] = useState(toDateTimeLocal(price.checked_at));
  const [tiers, setTiers] = useState(price.tiers?.length ? price.tiers.map(tierState) : [tierState()]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const updateTier = (index, field, value) => {
    setTiers((current) => current.map((tier, tierIndex) => tierIndex === index ? { ...tier, [field]: value } : tier));
  };

  const addTier = () => {
    setTiers((current) => {
      const next = current.map((tier) => ({ ...tier }));
      const lastIndex = next.length - 1;
      if (!next[lastIndex].maximum) {
        const minimum = Number(tierMinimum(next, lastIndex) || 0);
        next[lastIndex].maximum = String(minimum + 199_999);
      }
      return [...next, tierState()];
    });
  };

  const removeTier = (index) => {
    setTiers((current) => {
      if (current.length === 1) return current;
      const next = current.filter((_, tierIndex) => tierIndex !== index).map((tier) => ({ ...tier }));
      next[next.length - 1].maximum = '';
      return next;
    });
  };

  const changeCurrency = (value) => {
    setCurrency(value);
    setExchangeRate(value === 'USD' ? '1' : exchangeRate === '1' ? '0.14' : exchangeRate);
  };

  const submit = async () => {
    const rateMicros = toMicros(exchangeRate);
    if (!sourceURL.trim() || !checkedAt || !Number.isSafeInteger(rateMicros) || rateMicros <= 0 || (currency === 'USD' && rateMicros !== 1_000_000)) {
      setError('请检查原币币种、独立核价汇率、核价来源和核价时间');
      return;
    }

    const priceTiers = [];
    for (let index = 0; index < tiers.length; index += 1) {
      const tier = tiers[index];
      const minimum = tierMinimum(tiers, index);
      const maximum = index === tiers.length - 1 ? null : Number(tier.maximum);
      const input = toMicros(tier.input);
      const output = protocol === 'embeddings' ? null : toMicros(tier.output);
      const cacheRead = protocol === 'embeddings' ? null : toMicros(tier.cacheRead);
      const cacheWrite5m = protocol === 'embeddings' ? null : toMicros(tier.cacheWrite5m);
      const cacheWrite1h = protocol === 'embeddings' ? null : toMicros(tier.cacheWrite1h);
      const optionalPrices = [cacheRead, cacheWrite5m, cacheWrite1h];
      if (!Number.isSafeInteger(minimum) || !Number.isSafeInteger(input) || input <= 0
        || (protocol !== 'embeddings' && (!Number.isSafeInteger(output) || output <= 0))
        || (maximum != null && (!Number.isSafeInteger(maximum) || maximum < minimum))
        || optionalPrices.some((value) => value != null && !Number.isSafeInteger(value))) {
        setError(`请检查第 ${index + 1} 档的上下文范围和每百万 Token 价格`);
        return;
      }
      priceTiers.push({
        minimum_input_tokens: minimum,
        maximum_input_tokens: maximum,
        input_price_micros: input,
        output_price_micros: output,
        cache_read_price_micros: cacheRead,
        cache_write_5m_price_micros: cacheWrite5m,
        cache_write_1h_price_micros: cacheWrite1h,
      });
    }

    const checkedDate = new Date(checkedAt);
    if (Number.isNaN(checkedDate.getTime())) {
      setError('核价时间格式无效');
      return;
    }
    const officialPrice = {
      currency,
      usd_exchange_rate_micros: rateMicros,
      source_url: sourceURL.trim(),
      checked_at: checkedDate.toISOString(),
      tiers: priceTiers,
    };

    setBusy(true);
    setError('');
    try {
      if (model) {
        await apiRequest(`/api/admin/models/${model.id}/prices`, { method: 'POST', body: officialPrice });
        notify('官方价格新版本已发布');
      } else {
        await apiRequest('/api/admin/models', { method: 'POST', body: {
          public_name: publicName.trim(), provider, protocol, lifecycle, price: officialPrice,
        } });
        notify('标准模型已创建');
      }
      await config.onSaved?.();
      close();
    } catch (requestError) {
      setError(requestError.message || '保存模型价格失败');
    } finally {
      setBusy(false);
    }
  };

  return (
    <Modal
      title={model ? '发布官方价格' : '新增标准模型'}
      description={model ? `${model.public_name} · 当前价格版本 v${model.price?.version || '-'}` : '创建模型并发布首个每百万 Token 官方价格版本。'}
      onClose={close}
      wide
      footer={<><Button onClick={close}>取消</Button><Button disabled={busy || (!model && !publicName.trim())} icon={model ? BadgeDollarSign : Plus} variant="primary" onClick={submit}>{busy ? '保存中' : model ? '发布价格' : '创建模型'}</Button></>}
    >
      {error && <InlineNotice tone="danger">{error}</InlineNotice>}
      {!model && <div className="form-grid two-column model-identity-grid">
        <Field label="模型名称"><TextInput value={publicName} onChange={(event) => setPublicName(event.target.value)} placeholder="例如 gpt-4o-mini" /></Field>
        <Field label="模型厂商"><div className="provider-select-field"><ProviderLogo provider={provider} size={20} /><Select value={provider} onChange={(event) => setProvider(event.target.value)}>{providers.map((item) => <option key={item} value={item}>{item}</option>)}</Select></div></Field>
        <Field label="协议"><Select value={protocol} onChange={(event) => setProtocol(event.target.value)}>{Object.entries(protocolNames).map(([value, label]) => <option key={value} value={value}>{label}</option>)}</Select></Field>
        <Field label="生命周期"><Select value={lifecycle} onChange={(event) => setLifecycle(event.target.value)}>{Object.entries(lifecycleNames).map(([value, label]) => <option key={value} value={value}>{label}</option>)}</Select></Field>
      </div>}

      <div className="form-grid two-column price-metadata-grid">
        <Field label="原币币种"><Select value={currency} onChange={(event) => changeCurrency(event.target.value)}><option value="USD">美元（USD）</option><option value="CNY">人民币（CNY）</option></Select></Field>
        <Field label="每 1 原币折合美元" hint="模型核价专用，不使用充值汇率"><TextInput disabled={currency === 'USD'} min="0.000001" step="0.000001" type="number" value={exchangeRate} onChange={(event) => setExchangeRate(event.target.value)} /></Field>
        <Field className="settings-span" label="核价来源"><TextInput type="url" value={sourceURL} onChange={(event) => setSourceURL(event.target.value)} placeholder="官方价格页地址" /></Field>
        <Field label="核价时间"><TextInput type="datetime-local" value={checkedAt} onChange={(event) => setCheckedAt(event.target.value)} /></Field>
      </div>

      <div className="price-tier-heading">
        <div><h3>每百万 Token 价格</h3><span>{protocol === 'embeddings' ? 'Embeddings 只录入输入价格' : '缓存价格留空时按输入 Token 价格处理'}</span></div>
        <Button icon={Plus} onClick={addTier}>新增价格阶梯</Button>
      </div>
      <div className="price-tier-list">
        {tiers.map((tier, index) => {
          const last = index === tiers.length - 1;
          return <div className="price-tier-row" key={index}>
            <div className="price-tier-row-header"><strong>第 {index + 1} 档</strong>{tiers.length > 1 && <Button aria-label={`删除第 ${index + 1} 档`} icon={Trash2} variant="danger" onClick={() => removeTier(index)}>删除第 {index + 1} 档</Button>}</div>
            <div className="form-grid price-tier-grid">
              <Field label={`第 ${index + 1} 档起始 Token`}><TextInput disabled value={tierMinimum(tiers, index)} /></Field>
              <Field label={`第 ${index + 1} 档上下文上限`}><TextInput disabled={last} min={tierMinimum(tiers, index)} step="1" type={last ? 'text' : 'number'} value={last ? '无上限' : tier.maximum} onChange={(event) => updateTier(index, 'maximum', event.target.value)} /></Field>
              <Field label={`第 ${index + 1} 档输入价格`} hint={`${currency} / 百万 Token`}><TextInput min="0.000001" step="0.000001" type="number" value={tier.input} onChange={(event) => updateTier(index, 'input', event.target.value)} /></Field>
              {protocol !== 'embeddings' && <>
                <Field label={`第 ${index + 1} 档输出价格`} hint={`${currency} / 百万 Token`}><TextInput min="0.000001" step="0.000001" type="number" value={tier.output} onChange={(event) => updateTier(index, 'output', event.target.value)} /></Field>
                <Field label={`第 ${index + 1} 档缓存读取价格`} hint="可选"><TextInput min="0" step="0.000001" type="number" value={tier.cacheRead} onChange={(event) => updateTier(index, 'cacheRead', event.target.value)} /></Field>
                <Field label={`第 ${index + 1} 档 5 分钟缓存写入价格`} hint="可选"><TextInput min="0" step="0.000001" type="number" value={tier.cacheWrite5m} onChange={(event) => updateTier(index, 'cacheWrite5m', event.target.value)} /></Field>
                <Field label={`第 ${index + 1} 档 1 小时缓存写入价格`} hint="可选"><TextInput min="0" step="0.000001" type="number" value={tier.cacheWrite1h} onChange={(event) => updateTier(index, 'cacheWrite1h', event.target.value)} /></Field>
              </>}
            </div>
          </div>;
        })}
      </div>
      <InlineNotice>价格以整数微原币保存；发布后旧报价退出新请求路由，在途请求和退款继续保留原版本映射。</InlineNotice>
    </Modal>
  );
}

export default function AdminModels({ notify }) {
  const [models, setModels] = useState([]);
  const [dialog, setDialog] = useState(null);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    setError('');
    try {
      setModels(await apiRequest('/api/admin/models') || []);
    } catch (requestError) {
      setError(requestError.message || '读取标准模型失败');
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  return (
    <div className="admin-section-stack">
      {error && <InlineNotice tone="warning">{error}</InlineNotice>}
      <Section title="标准模型与官方价格" description="管理员按每百万 Token 发布官方绝对价格，商户报价最高为官方原价" actions={<><Button icon={RefreshCw} onClick={load}>刷新</Button><Button icon={Plus} variant="primary" onClick={() => setDialog({ onSaved: load })}>新增标准模型</Button></>}>
        <div className="table-wrap"><table className="data-table"><thead><tr><th>模型与厂商</th><th>协议</th><th>生命周期</th><th>首档价格 / 百万 Token</th><th>阶梯</th><th>价格版本</th><th>状态</th><th>操作</th></tr></thead><tbody>
          {models.map((model) => <tr key={model.id}><td><div className="model-cell"><ProviderLogo provider={model.provider} size={22} /><div><strong>{model.public_name}</strong><span>{model.provider} · M{model.id}</span></div></div></td><td>{protocolNames[model.protocol] || model.protocol}</td><td>{lifecycleNames[model.lifecycle] || model.lifecycle || '-'}</td><td><strong>{formatOfficialPrice(model)}</strong><small>{model.price.currency || '-'} · 核价于 {model.price.checked_at ? new Date(model.price.checked_at).toLocaleDateString('zh-CN') : '-'}</small></td><td>{model.price.tiers?.length || 0} 档</td><td><strong>价格版本 v{model.price.version}</strong><small>P{model.price.id}</small></td><td><Status>{model.status === 'active' ? '已启用' : model.status}</Status></td><td><Button icon={BadgeDollarSign} variant="ghost" onClick={() => setDialog({ model, onSaved: load })}>发布价格</Button></td></tr>)}
          {models.length === 0 && <tr><td className="table-message" colSpan="8">暂无标准模型</td></tr>}
        </tbody></table></div>
      </Section>
      {dialog && <ModelDialog close={() => setDialog(null)} config={dialog} notify={notify} />}
    </div>
  );
}
