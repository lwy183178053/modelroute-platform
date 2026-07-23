import { useCallback, useEffect, useState } from 'react';
import { BadgeDollarSign, Plus, RefreshCw, ShieldCheck, Trash2 } from 'lucide-react';
import { apiRequest, formatImagePrice, formatPriceMicros } from '../../api';
import { ActionMenu, Button, Field, InlineNotice, Modal, ProviderLogo, Section, Select, Status, TextInput } from '../../ui';

const providers = ['OpenAI', 'Anthropic', 'Gemini', 'DeepSeek', 'Qwen', 'Moonshot'];
const imageQualities = ['low', 'medium', 'high', 'standard'];
const imageSizes = ['1024x1024', '1024x1536', '1536x1024'];

const toDecimal = (value) => value == null ? '' : String(Number(value) / 1_000_000);

function toMicros(value) {
  if (String(value ?? '').trim() === '') return null;
  const micros = Math.round(Number(value) * 1_000_000);
  return Number.isSafeInteger(micros) && micros >= 0 ? micros : Number.NaN;
}

function tierState(tier = {}) {
  return {
    maximum: tier.maximum_input_tokens == null ? '' : String(tier.maximum_input_tokens),
    input: toDecimal(tier.input_price_micros),
    output: toDecimal(tier.output_price_micros),
    cacheRead: toDecimal(tier.cache_read_price_micros),
    cacheWrite: toDecimal(tier.cache_write_price_micros),
  };
}

function imageState(item = {}) {
  return {
    quality: item.quality || 'low',
    size: item.size || '1024x1024',
    price: toDecimal(item.price_micros),
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
  if (model.price?.price_kind === 'image') return formatImagePrice(model.price?.image_prices);
  const tier = model.price?.tiers?.[0];
  if (!tier) return '-';
  return `${formatPriceMicros(tier.input_price_micros)} 输入 / ${formatPriceMicros(tier.output_price_micros)} 输出`;
}

function ModelDialog({ config, close, notify }) {
  const model = config.model;
  const price = model?.price || {};
  const [publicName, setPublicName] = useState('');
  const [provider, setProvider] = useState(model?.provider || 'OpenAI');
  const [priceKind, setPriceKind] = useState(price.price_kind || (price.image_prices?.length ? 'image' : 'text'));
  const [tiers, setTiers] = useState(price.tiers?.length ? price.tiers.map(tierState) : [tierState()]);
  const [imagePrices, setImagePrices] = useState(price.image_prices?.length ? price.image_prices.map(imageState) : [imageState()]);
  const [baseURL, setBaseURL] = useState('');
  const [apiKey, setAPIKey] = useState('');
  const [detected, setDetected] = useState(false);
  const [detectResult, setDetectResult] = useState(null);
  const [detecting, setDetecting] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const resetDetection = () => {
    setDetected(false);
    setDetectResult(null);
  };
  const updateTier = (index, field, value) => setTiers((current) => current.map((tier, tierIndex) => tierIndex === index ? { ...tier, [field]: value } : tier));
  const updateImage = (index, field, value) => setImagePrices((current) => current.map((item, itemIndex) => itemIndex === index ? { ...item, [field]: value } : item));
  const addTier = () => {
    setTiers((current) => {
      const next = current.map((tier) => ({ ...tier }));
      const lastIndex = next.length - 1;
      if (!next[lastIndex].maximum) next[lastIndex].maximum = String(Number(tierMinimum(next, lastIndex) || 0) + 199_999);
      return [...next, tierState()];
    });
  };
  const removeTier = (index) => setTiers((current) => {
    if (current.length === 1) return current;
    const next = current.filter((_, tierIndex) => tierIndex !== index).map((tier) => ({ ...tier }));
    next[next.length - 1].maximum = '';
    return next;
  });
  const removeImage = (index) => setImagePrices((current) => current.length === 1 ? current : current.filter((_, itemIndex) => itemIndex !== index));

  const detect = async () => {
    setDetecting(true);
    setError('');
    resetDetection();
    try {
      const result = await apiRequest('/api/admin/models/detect', { method: 'POST', body: {
        public_name: publicName.trim(), provider, price_kind: priceKind, base_url: baseURL.trim(), api_key: apiKey,
      } });
      setDetected(true);
      setDetectResult(result);
    } catch (requestError) {
      setError(requestError.message || '模型检测失败');
    } finally {
      setDetecting(false);
    }
  };

  const buildPrice = () => {
    if (priceKind === 'image') {
      const items = imagePrices.map((item) => ({ quality: item.quality.trim(), size: item.size.trim(), price_micros: toMicros(item.price) }));
      const keys = new Set(items.map((item) => `${item.quality}\u0000${item.size}`));
      if (items.some((item) => !item.quality || !item.size || !Number.isSafeInteger(item.price_micros) || item.price_micros <= 0) || keys.size !== items.length) {
        throw new Error('请检查图片质量、尺寸和大于零的单张价格，组合不得重复');
      }
      return { image_prices: items };
    }
    const priceTiers = tiers.map((tier, index) => {
      const minimum = tierMinimum(tiers, index);
      const maximum = index === tiers.length - 1 ? null : Number(tier.maximum);
      const input = toMicros(tier.input);
      const output = toMicros(tier.output);
      const cacheRead = toMicros(tier.cacheRead);
      const cacheWrite = toMicros(tier.cacheWrite);
      if (!Number.isSafeInteger(minimum) || !Number.isSafeInteger(input) || input <= 0 || !Number.isSafeInteger(output) || output <= 0
        || (maximum != null && (!Number.isSafeInteger(maximum) || maximum < minimum))
        || [cacheRead, cacheWrite].some((value) => value != null && !Number.isSafeInteger(value))) {
        throw new Error(`请检查第 ${index + 1} 档的上下文范围和每百万 Token 价格`);
      }
      return {
        minimum_input_tokens: minimum, maximum_input_tokens: maximum,
        input_price_micros: input, output_price_micros: output,
        cache_read_price_micros: cacheRead, cache_write_price_micros: cacheWrite,
      };
    });
    return { tiers: priceTiers };
  };

  const submit = async () => {
    let officialPrice;
    try {
      officialPrice = buildPrice();
    } catch (priceError) {
      setError(priceError.message);
      return;
    }
    setBusy(true);
    setError('');
    try {
      if (model) {
        await apiRequest(`/api/admin/models/${model.id}/prices`, { method: 'POST', body: officialPrice });
        notify('模型价格新版本已发布');
      } else {
        await apiRequest('/api/admin/models', { method: 'POST', body: { public_name: publicName.trim(), provider, price: officialPrice } });
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
    <Modal title={model ? '发布模型价格' : '新增标准模型'} description={model ? `${model.public_name} · 当前价格版本 v${model.price?.version || '-'}` : '自定义模型需先完成 hi 检测。'} onClose={close} wide footer={<><Button onClick={close}>取消</Button><Button disabled={busy || (!model && (!publicName.trim() || !detected))} icon={model ? BadgeDollarSign : Plus} variant="primary" onClick={submit}>{busy ? '保存中' : model ? '发布价格' : '创建模型'}</Button></>}>
      {error && <InlineNotice tone="danger">{error}</InlineNotice>}
      {!model && <>
        <div className="form-grid two-column model-identity-grid">
          <Field label="模型名称"><TextInput value={publicName} onChange={(event) => { setPublicName(event.target.value); resetDetection(); }} placeholder="例如 gpt-5.4" /></Field>
          <Field label="模型厂商"><div className="provider-select-field"><ProviderLogo provider={provider} size={20} /><Select value={provider} onChange={(event) => { setProvider(event.target.value); resetDetection(); }}>{providers.map((item) => <option key={item} value={item}>{item}</option>)}</Select></div></Field>
          <Field label="计费类型"><Select value={priceKind} onChange={(event) => { setPriceKind(event.target.value); resetDetection(); }}><option value="text">文本 Token 计费</option><option value="image">图片按张计费</option></Select></Field>
          <Field label="检测基础地址"><TextInput value={baseURL} onChange={(event) => { setBaseURL(event.target.value); resetDetection(); }} placeholder="https://api.example.com/v1" /></Field>
          <Field label="检测 API 密钥"><TextInput value={apiKey} onChange={(event) => { setAPIKey(event.target.value); resetDetection(); }} type="password" /></Field>
        </div>
        <div className="config-actions"><Button disabled={detecting || !publicName.trim() || !baseURL.trim() || apiKey.trim().length < 8} icon={ShieldCheck} onClick={detect} variant="primary">{detecting ? '检测中' : '发送 hi 检测'}</Button>{detectResult && <InlineNotice>检测通过 · {detectResult.latency_ms} ms</InlineNotice>}</div>
      </>}

      {priceKind === 'image' ? <>
        <div className="price-tier-heading"><div><h3>图片单张价格矩阵</h3></div><Button icon={Plus} onClick={() => setImagePrices((current) => [...current, imageState()])}>新增价格组合</Button></div>
        <div className="price-tier-list">{imagePrices.map((item, index) => <div className="price-tier-row" key={index}>
          <div className="price-tier-row-header"><strong>组合 {index + 1}</strong>{imagePrices.length > 1 && <Button aria-label={`删除组合 ${index + 1}`} icon={Trash2} variant="danger" onClick={() => removeImage(index)}>删除</Button>}</div>
          <div className="form-grid price-tier-grid">
            <Field label={`组合 ${index + 1} 质量`}><Select value={item.quality} onChange={(event) => updateImage(index, 'quality', event.target.value)}>{imageQualities.map((quality) => <option key={quality} value={quality}>{quality}</option>)}</Select></Field>
            <Field label={`组合 ${index + 1} 尺寸`}><Select value={item.size} onChange={(event) => updateImage(index, 'size', event.target.value)}>{imageSizes.map((size) => <option key={size} value={size}>{size}</option>)}</Select></Field>
            <Field label={`组合 ${index + 1} 单张价格`} hint="不显示货币符号"><TextInput min="0.000001" step="0.000001" type="number" value={item.price} onChange={(event) => updateImage(index, 'price', event.target.value)} /></Field>
          </div>
        </div>)}</div>
      </> : <>
        <div className="price-tier-heading"><div><h3>每百万 Token 价格</h3><span>缓存价格留空时不单独计价</span></div><Button icon={Plus} onClick={addTier}>新增价格阶梯</Button></div>
        <div className="price-tier-list">{tiers.map((tier, index) => {
          const last = index === tiers.length - 1;
          return <div className="price-tier-row" key={index}>
            <div className="price-tier-row-header"><strong>第 {index + 1} 档</strong>{tiers.length > 1 && <Button aria-label={`删除第 ${index + 1} 档`} icon={Trash2} variant="danger" onClick={() => removeTier(index)}>删除第 {index + 1} 档</Button>}</div>
            <div className="form-grid price-tier-grid">
              <Field label={`第 ${index + 1} 档起始 Token`}><TextInput disabled value={tierMinimum(tiers, index)} /></Field>
              <Field label={`第 ${index + 1} 档上下文上限`}><TextInput disabled={last} min={tierMinimum(tiers, index)} step="1" type={last ? 'text' : 'number'} value={last ? '无上限' : tier.maximum} onChange={(event) => updateTier(index, 'maximum', event.target.value)} /></Field>
              <Field label={`第 ${index + 1} 档输入价格`} hint="每百万 Token"><TextInput min="0.000001" step="0.000001" type="number" value={tier.input} onChange={(event) => updateTier(index, 'input', event.target.value)} /></Field>
              <Field label={`第 ${index + 1} 档输出价格`} hint="每百万 Token"><TextInput min="0.000001" step="0.000001" type="number" value={tier.output} onChange={(event) => updateTier(index, 'output', event.target.value)} /></Field>
              <Field label={`第 ${index + 1} 档缓存读取价格`} hint="可选"><TextInput min="0" step="0.000001" type="number" value={tier.cacheRead} onChange={(event) => updateTier(index, 'cacheRead', event.target.value)} /></Field>
              <Field label={`第 ${index + 1} 档缓存写入价格`} hint="可选"><TextInput min="0" step="0.000001" type="number" value={tier.cacheWrite} onChange={(event) => updateTier(index, 'cacheWrite', event.target.value)} /></Field>
            </div>
          </div>;
        })}</div>
      </>}
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

  const remove = async (model) => {
    if (!window.confirm(`确认删除或归档模型“${model.public_name}”吗？`)) return;
    try {
      const data = await apiRequest(`/api/admin/models/${model.id}`, { method: 'DELETE' });
      notify(data?.result === 'deleted' ? '标准模型已删除' : '标准模型已归档下架');
      await load();
    } catch (requestError) {
      notify(requestError.message || '删除模型失败');
    }
  };

  return (
    <div className="admin-section-stack">
      {error && <InlineNotice tone="warning">{error}</InlineNotice>}
      <Section title="标准模型与基础价格" actions={<><Button icon={RefreshCw} onClick={load}>刷新</Button><Button icon={Plus} variant="primary" onClick={() => setDialog({ onSaved: load })}>新增标准模型</Button></>}>
        <div className="table-wrap"><table className="data-table"><thead><tr><th>ID</th><th>模型与厂商</th><th>首档价格</th><th>价格结构</th><th>价格版本</th><th>状态</th><th>操作</th></tr></thead><tbody>
          {models.map((model) => <tr key={model.id}><td><code>{model.id}</code></td><td><div className="model-cell"><ProviderLogo provider={model.provider} size={22} /><div><strong>{model.public_name}</strong><span>{model.provider}</span></div></div></td><td><strong>{formatOfficialPrice(model)}</strong></td><td>{model.price?.price_kind === 'image' ? `${model.price.image_prices?.length || 0} 个组合` : `${model.price.tiers?.length || 0} 档`}</td><td><strong>v{model.price.version}</strong></td><td><Status>{model.status === 'active' ? '已启用' : model.status === 'archived' ? '已归档' : model.status}</Status></td><td><ActionMenu items={[{ label: '发布价格', icon: BadgeDollarSign, onClick: () => setDialog({ model, onSaved: load }) }, { label: '删除', icon: Trash2, danger: true, onClick: () => remove(model) }]} /></td></tr>)}
          {models.length === 0 && <tr><td className="table-message" colSpan="7">暂无标准模型</td></tr>}
        </tbody></table></div>
      </Section>
      {dialog && <ModelDialog close={() => setDialog(null)} config={dialog} notify={notify} />}
    </div>
  );
}
