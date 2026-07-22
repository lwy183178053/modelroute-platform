import { useCallback, useEffect, useState } from 'react';
import { BadgeDollarSign, Plus, RefreshCw } from 'lucide-react';
import { apiRequest } from '../../api';
import { Button, Field, InlineNotice, Modal, Section, Select, Status, TextInput } from '../../ui';

const protocolNames = {
  chat_completions: '聊天补全', responses: 'Responses', embeddings: '嵌入',
};

const toDecimal = (value) => String(Number(value || 0) / 1_000_000);
const toMicros = (value) => Math.round(Number(value || 0) * 1_000_000);

function ModelDialog({ config, close, notify }) {
  const model = config.model;
  const price = model?.price || {};
  const [publicName, setPublicName] = useState('');
  const [provider, setProvider] = useState('');
  const [protocol, setProtocol] = useState('chat_completions');
  const [billingMode, setBillingMode] = useState(price.billing_mode || 'token');
  const [mainRatio, setMainRatio] = useState(price.model_ratio_micros ? toDecimal(price.model_ratio_micros) : '1');
  const [fixedPrice, setFixedPrice] = useState(price.model_price_microusd ? toDecimal(price.model_price_microusd) : '0.001');
  const [completionRatio, setCompletionRatio] = useState(price.completion_ratio_micros ? toDecimal(price.completion_ratio_micros) : '1');
  const [cacheRatio, setCacheRatio] = useState(toDecimal(price.cache_ratio_micros));
  const [createCacheRatio, setCreateCacheRatio] = useState(toDecimal(price.create_cache_ratio_micros));
  const [imageRatio, setImageRatio] = useState(toDecimal(price.image_ratio_micros));
  const [audioRatio, setAudioRatio] = useState(toDecimal(price.audio_ratio_micros));
  const [audioCompletionRatio, setAudioCompletionRatio] = useState(toDecimal(price.audio_completion_ratio_micros));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const submit = async () => {
    const officialPrice = {
      billing_mode: billingMode,
      model_ratio_micros: billingMode === 'token' ? toMicros(mainRatio) : 0,
      model_price_microusd: billingMode === 'fixed' ? toMicros(fixedPrice) : 0,
      completion_ratio_micros: toMicros(completionRatio),
      cache_ratio_micros: toMicros(cacheRatio),
      create_cache_ratio_micros: toMicros(createCacheRatio),
      image_ratio_micros: toMicros(imageRatio),
      audio_ratio_micros: toMicros(audioRatio),
      audio_completion_ratio_micros: toMicros(audioCompletionRatio),
    };
    const primaryValue = billingMode === 'token' ? officialPrice.model_ratio_micros : officialPrice.model_price_microusd;
    if (!Number.isInteger(primaryValue) || primaryValue <= 0 || Object.values(officialPrice).some((value) => typeof value === 'number' && (!Number.isInteger(value) || value < 0))) {
      setError('价格维度必须填写为非负小数，主倍率或固定价格必须大于零');
      return;
    }
    setBusy(true);
    setError('');
    try {
      if (model) {
        await apiRequest(`/api/admin/models/${model.id}/prices`, { method: 'POST', body: officialPrice });
        notify('官方价格新版本已发布');
      } else {
        await apiRequest('/api/admin/models', { method: 'POST', body: {
          public_name: publicName.trim(), provider: provider.trim(), protocol, price: officialPrice,
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
      description={model ? `${model.public_name} · 当前价格版本 v${model.price.version}` : '创建公开模型名称和首个官方价格版本。'}
      onClose={close}
      wide
      footer={<><Button onClick={close}>取消</Button><Button disabled={busy || (!model && (!publicName.trim() || !provider.trim()))} icon={model ? BadgeDollarSign : Plus} variant="primary" onClick={submit}>{busy ? '保存中' : model ? '发布价格' : '创建模型'}</Button></>}
    >
      {error && <InlineNotice tone="danger">{error}</InlineNotice>}
      {!model && <div className="form-grid two-column">
        <Field label="模型名称"><TextInput value={publicName} onChange={(event) => setPublicName(event.target.value)} placeholder="例如 gpt-4o-mini" /></Field>
        <Field label="模型厂商"><TextInput value={provider} onChange={(event) => setProvider(event.target.value)} placeholder="例如 OpenAI" /></Field>
        <Field label="协议"><Select value={protocol} onChange={(event) => setProtocol(event.target.value)}><option value="chat_completions">聊天补全</option><option value="responses">Responses</option><option value="embeddings">嵌入</option></Select></Field>
      </div>}
      <div className="form-grid two-column">
        <Field label="计费模式"><Select value={billingMode} onChange={(event) => setBillingMode(event.target.value)}><option value="token">Token 计费</option><option value="fixed">固定计费</option></Select></Field>
        {billingMode === 'token'
          ? <Field label="官方主倍率"><TextInput min="0.000001" step="0.000001" type="number" value={mainRatio} onChange={(event) => setMainRatio(event.target.value)} /></Field>
          : <Field label="单次固定价格（美元）"><TextInput min="0.000001" step="0.000001" type="number" value={fixedPrice} onChange={(event) => setFixedPrice(event.target.value)} /></Field>}
        <Field label="输出倍率"><TextInput min="0" step="0.000001" type="number" value={completionRatio} onChange={(event) => setCompletionRatio(event.target.value)} /></Field>
        <Field label="缓存读取倍率"><TextInput min="0" step="0.000001" type="number" value={cacheRatio} onChange={(event) => setCacheRatio(event.target.value)} /></Field>
        <Field label="缓存创建倍率"><TextInput min="0" step="0.000001" type="number" value={createCacheRatio} onChange={(event) => setCreateCacheRatio(event.target.value)} /></Field>
        <Field label="图片倍率"><TextInput min="0" step="0.000001" type="number" value={imageRatio} onChange={(event) => setImageRatio(event.target.value)} /></Field>
        <Field label="音频输入倍率"><TextInput min="0" step="0.000001" type="number" value={audioRatio} onChange={(event) => setAudioRatio(event.target.value)} /></Field>
        <Field label="音频输出倍率"><TextInput min="0" step="0.000001" type="number" value={audioCompletionRatio} onChange={(event) => setAudioCompletionRatio(event.target.value)} /></Field>
      </div>
      <InlineNotice>平台保存整数微单位；发布后旧报价退出新请求路由，在途请求和退款仍保留原版本映射。</InlineNotice>
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
      <Section title="标准模型与官方价格" description="官方价格由管理员发布版本，商户报价不得超过 1.0000x" actions={<><Button icon={RefreshCw} onClick={load}>刷新</Button><Button icon={Plus} variant="primary" onClick={() => setDialog({ onSaved: load })}>新增标准模型</Button></>}>
        <div className="table-wrap"><table className="data-table"><thead><tr><th>模型</th><th>厂商</th><th>协议</th><th>计费模式</th><th>主价格维度</th><th>价格版本</th><th>状态</th><th>操作</th></tr></thead><tbody>
          {models.map((model) => <tr key={model.id}><td><strong>{model.public_name}</strong><small>M{model.id}</small></td><td>{model.provider}</td><td>{protocolNames[model.protocol] || model.protocol}</td><td>{model.price.billing_mode === 'fixed' ? '固定计费' : 'Token 计费'}</td><td>{model.price.billing_mode === 'fixed' ? `$${toDecimal(model.price.model_price_microusd)} / 次` : `${toDecimal(model.price.model_ratio_micros)}x`}</td><td><strong>价格版本 v{model.price.version}</strong><small>P{model.price.id}</small></td><td><Status>{model.status === 'active' ? '已启用' : model.status}</Status></td><td><Button icon={BadgeDollarSign} variant="ghost" onClick={() => setDialog({ model, onSaved: load })}>发布价格</Button></td></tr>)}
          {models.length === 0 && <tr><td className="table-message" colSpan="8">暂无标准模型</td></tr>}
        </tbody></table></div>
      </Section>
      {dialog && <ModelDialog close={() => setDialog(null)} config={dialog} notify={notify} />}
    </div>
  );
}
