import { useEffect, useState } from 'react';
import { Check, CreditCard, KeyRound, Landmark, LockKeyhole, Plus, RotateCcw, Send, ShieldCheck, Store } from 'lucide-react';
import AuthModal from './AuthModal';
import { apiRequest, formatPriceMicros, formatQuota } from './api';
import { Button, Field, InlineNotice, Modal, Select, TextInput } from './ui';

function Footer({ cancel, submit, submitText, icon = Check }) {
  return <><Button onClick={cancel}>取消</Button><Button icon={icon} variant="primary" onClick={submit}>{submitText}</Button></>;
}

function RouteModal({ close, config, notify }) {
  const policy = config.policy || {};
  const [mode, setMode] = useState(policy.mode || 'smart');
  const [priority, setPriority] = useState(policy.priority || 'stability');
  const [minimum, setMinimum] = useState(String(Number(policy.minimum_multiplier_bps || 1) / 10_000));
  const [maximum, setMaximum] = useState(String(Number(policy.maximum_multiplier_bps || 10_000) / 10_000));
  const [merchantIDs, setMerchantIDs] = useState(policy.merchant_ids || []);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const toggleMerchant = (merchantID) => {
    setMerchantIDs((current) => current.includes(merchantID) ? current.filter((id) => id !== merchantID) : [...current, merchantID]);
  };

  const submit = async () => {
    const minimumBPS = Math.round(Number(minimum) * 10_000);
    const maximumBPS = Math.round(Number(maximum) * 10_000);
    if (!Number.isInteger(minimumBPS) || !Number.isInteger(maximumBPS) || minimumBPS < 1 || maximumBPS > 10_000 || minimumBPS > maximumBPS) {
      setError('倍率区间必须在 0.0001x 到 1.0000x 之间');
      return;
    }
    if (mode === 'fixed' && merchantIDs.length === 0) {
      setError('固定路由至少选择一个商户');
      return;
    }
    setBusy(true);
    setError('');
    try {
      await apiRequest(`/api/routing-policies/${config.key.id}`, {
        method: 'PUT',
        body: {
          mode, priority,
          minimum_multiplier_bps: minimumBPS,
          maximum_multiplier_bps: maximumBPS,
          merchant_ids: mode === 'fixed' ? merchantIDs : [],
        },
      });
      await config.onSaved?.();
      notify('路由策略已保存');
      close();
    } catch (requestError) {
      setError(requestError.message || '保存路由策略失败');
    } finally {
      setBusy(false);
    }
  };
  return (
    <Modal
      title="编辑路由策略"
      description="策略按当前 API 密钥绑定，只影响保存后的新请求。"
      onClose={close}
      wide
      footer={<><Button onClick={close}>取消</Button><Button disabled={busy} icon={Check} variant="primary" onClick={submit}>{busy ? '保存中' : '保存策略'}</Button></>}
    >
      {error && <InlineNotice tone="danger">{error}</InlineNotice>}
      <div className="form-section"><h3>路由方式</h3><div className="choice-grid">
        <label className={`choice-card ${mode === 'smart' ? 'selected' : ''}`}><input aria-label="智能路由" checked={mode === 'smart'} name="route-mode" onChange={() => setMode('smart')} type="radio" /><span className="choice-icon"><ShieldCheck size={20} /></span><strong>智能路由</strong><small>从所有符合倍率和健康条件的商户中选择</small></label>
        <label className={`choice-card ${mode === 'fixed' ? 'selected' : ''}`}><input aria-label="固定商户" checked={mode === 'fixed'} name="route-mode" onChange={() => setMode('fixed')} type="radio" /><span className="choice-icon"><Store size={20} /></span><strong>固定商户</strong><small>只从指定商户中选择，越界时直接返回无渠道错误</small></label>
      </div></div>

      <div className="form-grid two-column">
        <Field label="最低成交倍率" hint="低于此倍率的报价不参与路由"><TextInput value={minimum} onChange={(event) => setMinimum(event.target.value)} min="0.0001" max="1" step="0.0001" type="number" /></Field>
        <Field label="最高成交倍率" hint="平台强制不超过官方价格 1.0x"><TextInput value={maximum} onChange={(event) => setMaximum(event.target.value)} min="0.0001" max="1" step="0.0001" type="number" /></Field>
      </div>

      <div className="form-section"><h3>选择优先级</h3><div className="radio-row">
        <label><input checked={priority === 'stability'} name="priority" onChange={() => setPriority('stability')} type="radio" />稳定性优先<span>可用率和首字时间优先，再比较价格</span></label>
        <label><input checked={priority === 'price'} name="priority" onChange={() => setPriority('price')} type="radio" />价格优先<span>倍率最低优先，再比较健康和可用率</span></label>
      </div></div>

      {mode === 'fixed' && <div className="form-section"><h3>固定商户范围</h3><div className="merchant-checks">
        {(config.merchants || []).map((merchant) => <label key={merchant.id}><input aria-label={`选择商户 ${merchant.name}`} checked={merchantIDs.includes(merchant.id)} onChange={() => toggleMerchant(merchant.id)} type="checkbox" /><span><strong>{merchant.name}</strong><small>{(Number(merchant.availability_ppm || 0) / 10_000).toFixed(2)}% 可用率 · 最低 {(Number(merchant.minimum_multiplier_bps || 0) / 10_000).toFixed(4)}x</small></span></label>)}
        {(config.merchants || []).length === 0 && <InlineNotice tone="warning">当前市场没有可选择的商户</InlineNotice>}
      </div></div>}
      <InlineNotice tone="warning">没有符合固定商户名单、倍率区间和健康条件的候选时，返回兼容 New API 的 503 错误且不扣费。</InlineNotice>
    </Modal>
  );
}

function QuoteModal({ close, config, notify }) {
  const [modelID, setModelID] = useState(String(config.models?.[0]?.id || ''));
  const [channelID, setChannelID] = useState(String(config.channels?.[0]?.id || ''));
  const [discountPercent, setDiscountPercent] = useState('72');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const selectedModel = config.models?.find((model) => String(model.id) === modelID);
  const selectedTier = selectedModel?.price?.tiers?.[0];
  const discountBPS = Math.round(Number(discountPercent) * 100);
  const formatDiscounted = (value) => {
    if (value == null || !Number.isInteger(discountBPS)) return '-';
    const usdMicros = Math.round(Number(value) * Number(selectedModel?.price?.usd_exchange_rate_micros || 0) / 1_000_000);
    return formatPriceMicros(Math.round(usdMicros * discountBPS / 10_000), 'USD');
  };
  const submit = async () => {
    if (!Number.isInteger(discountBPS) || discountBPS < 1 || discountBPS > 10_000) {
      setError('官方价折扣必须在 0.01% 到 100.00% 之间');
      return;
    }
    setBusy(true);
    setError('');
    try {
      await apiRequest('/api/merchant/offers', { method: 'POST', body: {
        merchant_channel_id: Number(channelID), canonical_model_id: Number(modelID), multiplier_bps: discountBPS,
      } });
      await config.onSaved?.();
      notify('新报价版本已生效');
      close();
    } catch (requestError) {
      setError(requestError.message || '创建报价失败');
    } finally {
      setBusy(false);
    }
  };
  return (
    <Modal title="创建模型报价" description="选择已审核渠道和标准模型，提交后同步 New API 价格与模型别名。" onClose={close} footer={<><Button onClick={close}>取消</Button><Button disabled={busy || !modelID || !channelID} icon={Check} variant="primary" onClick={submit}>{busy ? '同步中' : '提交报价'}</Button></>}>
      {error && <InlineNotice tone="danger">{error}</InlineNotice>}
      <Field label="标准模型"><Select value={modelID} onChange={(event) => setModelID(event.target.value)}>{(config.models || []).map((model) => <option key={model.id} value={model.id}>{model.public_name} · {model.provider}</option>)}</Select></Field>
      <Field label="商户渠道"><Select value={channelID} onChange={(event) => setChannelID(event.target.value)}>{(config.channels || []).map((channel) => <option key={channel.id} value={channel.id}>{channel.channel_name} · C{channel.id}</option>)}</Select></Field>
      <div className="quote-summary"><div><span>官方价格版本</span><strong>v{selectedModel?.price?.version || '-'}</strong></div><div><span>官方首档价格</span><strong>{selectedTier ? `${formatPriceMicros(selectedTier.input_price_micros, selectedModel.price.currency)} 输入${selectedTier.output_price_micros == null ? '' : ` / ${formatPriceMicros(selectedTier.output_price_micros, selectedModel.price.currency)} 输出`}` : '-'}</strong></div><div><span>折后美元价格</span><strong>{selectedTier ? `${formatDiscounted(selectedTier.input_price_micros)} 输入${selectedTier.output_price_micros == null ? '' : ` / ${formatDiscounted(selectedTier.output_price_micros)} 输出`}` : '-'}</strong></div></div>
      <Field label="官方价折扣" hint="允许范围 0.01% - 100.00%"><div className="suffix-input"><TextInput max="100" min="0.01" onChange={(event) => setDiscountPercent(event.target.value)} step="0.01" type="number" value={discountPercent} /><span>%</span></div></Field>
      <InlineNotice>调价将创建不可变的新报价版本；已进入请求的价格不会变化。</InlineNotice>
    </Modal>
  );
}

function PaymentModal({ close, config, notify }) {
  const setting = config.setting || {};
  const [gatewayURL, setGatewayURL] = useState(setting.gateway_url || 'https://zpayz.cn');
  const [merchantPID, setMerchantPID] = useState('');
  const [merchantKey, setMerchantKey] = useState('');
  const [enabled, setEnabled] = useState(setting.enabled ?? true);
  const [methods, setMethods] = useState(setting.payment_methods || ['alipay', 'wxpay']);
  const [minimumUSD, setMinimumUSD] = useState(String(Number(setting.minimum_usd_cents || 1000) / 100));
  const [maximumUSD, setMaximumUSD] = useState(String(Number(setting.maximum_usd_cents || 2000000) / 100));
  const [rate, setRate] = useState(String(Number(setting.exchange_rate_cny_cents_per_usd || 720) / 100));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const toggleMethod = (method) => setMethods((current) => current.includes(method) ? current.filter((item) => item !== method) : [...current, method]);
  const submit = async () => {
    const minimumUSDCents = Math.round(Number(minimumUSD) * 100);
    const maximumUSDCents = Math.round(Number(maximumUSD) * 100);
    const exchangeRate = Math.round(Number(rate) * 100);
    if (!gatewayURL.trim() || methods.length === 0 || !Number.isInteger(minimumUSDCents) || !Number.isInteger(maximumUSDCents) || !Number.isInteger(exchangeRate) || minimumUSDCents <= 0 || maximumUSDCents < minimumUSDCents || exchangeRate <= 0) {
      setError('请检查网关、支付方式、额度范围和汇率');
      return;
    }
    setBusy(true);
    setError('');
    try {
      await apiRequest('/api/admin/payment-settings', { method: 'PUT', body: {
        gateway_url: gatewayURL.trim(), merchant_pid: merchantPID.trim(), merchant_key: merchantKey,
        enabled, payment_methods: methods, minimum_usd_cents: minimumUSDCents,
        maximum_usd_cents: maximumUSDCents, exchange_rate_cny_cents_per_usd: exchangeRate,
      } });
      await config.onSaved?.();
      notify('支付配置已保存为新版本');
      close();
    } catch (requestError) {
      setError(requestError.message || '保存支付配置失败');
    } finally {
      setBusy(false);
    }
  };
  return (
    <Modal title="编辑易支付配置" description="保存后生成新配置版本，完整 KEY 不再回显。" onClose={close} footer={<><Button onClick={close}>取消</Button><Button disabled={busy} icon={CreditCard} variant="primary" onClick={submit}>{busy ? '保存中' : '保存支付配置'}</Button></>} wide>
      {error && <InlineNotice tone="danger">{error}</InlineNotice>}
      <div className="form-grid two-column">
        <Field label="网关地址"><TextInput onChange={(event) => setGatewayURL(event.target.value)} value={gatewayURL} /></Field>
        <Field label="商户编号 PID" hint={setting.merchant_pid_masked ? `留空保持 ${setting.merchant_pid_masked}` : '首次配置必须填写'}><TextInput onChange={(event) => setMerchantPID(event.target.value)} placeholder={setting.merchant_pid_masked || '输入 PID'} value={merchantPID} /></Field>
        <Field label="商户密钥 KEY" hint="留空保持当前密钥；输入新值将轮换"><TextInput onChange={(event) => setMerchantKey(event.target.value)} placeholder={setting.key_configured ? '已配置，留空保持' : '输入 KEY'} type="password" value={merchantKey} /></Field>
        <Field label="支付状态"><Select onChange={(event) => setEnabled(event.target.value === 'enabled')} value={enabled ? 'enabled' : 'disabled'}><option value="enabled">启用</option><option value="disabled">停用</option></Select></Field>
        <Field label="最低充值额度（美元）"><TextInput min="0.01" onChange={(event) => setMinimumUSD(event.target.value)} step="0.01" type="number" value={minimumUSD} /></Field>
        <Field label="最高充值额度（美元）"><TextInput min="0.01" onChange={(event) => setMaximumUSD(event.target.value)} step="0.01" type="number" value={maximumUSD} /></Field>
        <Field label="人民币兑美元汇率" hint="填写 1 美元对应的人民币金额"><TextInput min="0.01" onChange={(event) => setRate(event.target.value)} step="0.01" type="number" value={rate} /></Field>
      </div>
      <Field label="已启用支付方式"><div className="checkbox-line">{[['alipay', '支付宝'], ['wxpay', '微信'], ['qqpay', 'QQ 钱包']].map(([value, label]) => <label key={value}><input checked={methods.includes(value)} onChange={() => toggleMethod(value)} type="checkbox" />{label}</label>)}</div></Field>
      <InlineNotice tone="warning">保存会锁定为新配置版本；已创建订单继续使用原版本验签和金额。</InlineNotice>
    </Modal>
  );
}

function RechargeModal({ close, config, notify }) {
  const [setting, setSetting] = useState(null);
  const [usd, setUSD] = useState('');
  const [paymentType, setPaymentType] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  useEffect(() => {
    apiRequest('/api/payment-settings').then((data) => {
      setSetting(data);
      setUSD(String(Number(data.minimum_usd_cents || 1000) / 100));
      setPaymentType(data.payment_methods?.[0] || '');
    }).catch((requestError) => setError(requestError.message || '读取充值配置失败'));
  }, []);
  const usdCents = Math.round(Number(usd) * 100);
  const cnyCents = setting && Number.isInteger(usdCents) ? Math.round(usdCents * Number(setting.exchange_rate_cny_cents_per_usd) / 100) : 0;
  const submit = async () => {
    if (!setting || !Number.isInteger(usdCents) || usdCents < setting.minimum_usd_cents || usdCents > setting.maximum_usd_cents || !paymentType) {
      setError('充值额度或支付方式不在允许范围');
      return;
    }
    setBusy(true);
    setError('');
    try {
      const data = await apiRequest('/api/payment-orders', { method: 'POST', body: { usd_cents: usdCents, payment_type: paymentType } });
      await config.onSaved?.(data.order);
      notify('充值订单已创建，正在进入支付页');
      window.location.assign(data.payment_url);
    } catch (requestError) {
      setError(requestError.message || '创建充值订单失败');
      setBusy(false);
    }
  };
  const presets = setting ? [...new Set([setting.minimum_usd_cents, 5000, 10000, 50000])].filter((value) => value >= setting.minimum_usd_cents && value <= setting.maximum_usd_cents) : [];
  return (
    <Modal title="账户充值" description="选择美元额度，订单创建时锁定汇率和人民币应付金额。" onClose={close} footer={<><Button onClick={close}>取消</Button><Button disabled={busy || !setting} icon={CreditCard} variant="primary" onClick={submit}>{busy ? '创建中' : '前往支付'}</Button></>}>
      {error && <InlineNotice tone="danger">{error}</InlineNotice>}
      <Field label="充值额度（美元）"><div className="prefix-input"><span>$</span><TextInput min={Number(setting?.minimum_usd_cents || 1) / 100} max={Number(setting?.maximum_usd_cents || 1) / 100} onChange={(event) => setUSD(event.target.value)} step="0.01" type="number" value={usd} /></div></Field>
      <div className="amount-presets">{presets.map((value) => <button className={value === usdCents ? 'active' : ''} key={value} onClick={() => setUSD(String(value / 100))} type="button">${(value / 100).toLocaleString('en-US')}</button>)}</div>
      {setting && <div className="quote-summary"><div><span>当前汇率</span><strong>¥{(Number(setting.exchange_rate_cny_cents_per_usd) / 100).toFixed(2)} / $1</strong></div><div><span>人民币应付</span><strong>¥{(cnyCents / 100).toFixed(2)}</strong></div></div>}
      <Field label="支付方式"><div className="payment-methods">{(setting?.payment_methods || []).map((method) => <label key={method}><input checked={paymentType === method} name="payment-method" onChange={() => setPaymentType(method)} type="radio" />{{ alipay: '支付宝', wxpay: '微信', qqpay: 'QQ 钱包' }[method] || method}</label>)}</div></Field>
      <InlineNotice>浏览器返回页只显示订单状态，额度入账只由已验签的异步回调触发。</InlineNotice>
    </Modal>
  );
}

function CreditResolveModal({ close, config, notify }) {
  const [reason, setReason] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const submit = async () => {
    setBusy(true);
    setError('');
    try {
      await apiRequest(`/api/admin/payment-orders/${config.order.id}/credit/resolve`, { method: 'POST', body: { action: config.action, reason: reason.trim() } });
      await config.onSaved?.();
      notify(config.action === 'credited' ? '已确认该订单完成入账' : '已根据核对结果重新排队');
      close();
    } catch (requestError) {
      setError(requestError.message || '提交核对结果失败');
    } finally {
      setBusy(false);
    }
  };
  return <Modal title={config.action === 'credited' ? '确认已完成入账' : '核对后重新入账'} description={`订单 ${config.order.order_no} · 此操作只用于结果未知的入账任务`} onClose={close} footer={<><Button onClick={close}>取消</Button><Button disabled={busy || reason.trim().length < 3} icon={RotateCcw} variant="primary" onClick={submit}>{busy ? '提交中' : '确认处置'}</Button></>}>
    {error && <InlineNotice tone="danger">{error}</InlineNotice>}
    <Field label="核对原因"><textarea className="textarea" onChange={(event) => setReason(event.target.value)} placeholder="填写已核对的 New API 审计或余额依据" rows="4" value={reason} /></Field>
    <InlineNotice tone="warning">选择重新入账前，必须确认此前请求没有增加用户额度。</InlineNotice>
  </Modal>;
}

function MerchantApplyModal({ close, config, notify }) {
  const [displayName, setDisplayName] = useState('');
  const [websiteURL, setWebsiteURL] = useState('');
  const [channelName, setChannelName] = useState('主线路');
  const [channelType, setChannelType] = useState('1');
  const [baseURL, setBaseURL] = useState('');
  const [apiKey, setAPIKey] = useState('');
  const [models, setModels] = useState('gpt-4o-mini');
  const [description, setDescription] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const submit = async () => {
    setBusy(true);
    setError('');
    try {
      const application = await apiRequest('/api/merchant/application', {
        method: 'POST',
        body: {
          display_name: displayName.trim(), website_url: websiteURL.trim(), business_description: description.trim(),
          channel_name: channelName.trim(), channel_type: Number(channelType), base_url: baseURL.trim(), api_key: apiKey,
          models: models.split(/[\n,]/).map((value) => value.trim()).filter(Boolean),
        },
      });
      await config.onSaved?.(application);
      notify('商户申请已提交');
      close();
    } catch (requestError) {
      setError(requestError.message || '提交商户申请失败');
    } finally {
      setBusy(false);
    }
  };
  return (
    <Modal title="申请成为商户" description="普通用户提交网站与渠道凭据，管理员验证通过后在同一账号开通商户中心。" onClose={close} footer={<><Button onClick={close}>取消</Button><Button disabled={busy} icon={Send} onClick={submit} variant="primary">{busy ? '提交中' : '提交申请'}</Button></>} wide>
      {error && <InlineNotice tone="danger">{error}</InlineNotice>}
      <div className="form-grid two-column">
        <Field label="商户名称"><TextInput maxLength="80" onChange={(event) => setDisplayName(event.target.value)} placeholder="用于市场展示" value={displayName} /></Field>
        <Field label="网站地址"><TextInput onChange={(event) => setWebsiteURL(event.target.value)} placeholder="https://example.com" value={websiteURL} /></Field>
        <Field label="渠道名称"><TextInput maxLength="100" onChange={(event) => setChannelName(event.target.value)} value={channelName} /></Field>
        <Field label="渠道类型"><Select onChange={(event) => setChannelType(event.target.value)} value={channelType}><option value="1">OpenAI 兼容</option><option value="14">Anthropic</option><option value="20">OpenRouter</option><option value="24">Gemini</option><option value="43">DeepSeek</option></Select></Field>
        <Field label="上游基础地址"><TextInput onChange={(event) => setBaseURL(event.target.value)} placeholder="https://api.example.com/v1" value={baseURL} /></Field>
        <Field label="上游 API 密钥" hint="提交后加密保存且不再回显"><TextInput onChange={(event) => setAPIKey(event.target.value)} placeholder="输入渠道密钥" type="password" value={apiKey} /></Field>
      </div>
      <Field label="拟售模型" hint="多个模型使用逗号或换行分隔"><textarea className="textarea" onChange={(event) => setModels(event.target.value)} rows="3" value={models} /></Field>
      <Field label="业务说明"><textarea className="textarea" onChange={(event) => setDescription(event.target.value)} placeholder="说明渠道来源、可售模型和服务能力" rows="4" value={description} /></Field>
    </Modal>
  );
}

function ChannelModal({ close, config, notify }) {
  const [channelName, setChannelName] = useState('');
  const [channelType, setChannelType] = useState('1');
  const [baseURL, setBaseURL] = useState('');
  const [apiKey, setAPIKey] = useState('');
  const [models, setModels] = useState('gpt-4o-mini');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const submitChannel = async () => {
    setBusy(true);
    setError('');
    try {
      await apiRequest('/api/merchant/channels', { method: 'POST', body: {
        channel_name: channelName.trim(), channel_type: Number(channelType), base_url: baseURL.trim(), api_key: apiKey,
        models: models.split(/[\n,]/).map((value) => value.trim()).filter(Boolean),
      } });
      await config.onSaved?.();
      notify('渠道已提交审核');
      close();
    } catch (requestError) {
      setError(requestError.message || '提交渠道失败');
    } finally {
      setBusy(false);
    }
  };
  return (
    <Modal title="提交新渠道" description="新渠道验证通过后才会导入 New API。" onClose={close} footer={<><Button onClick={close}>取消</Button><Button disabled={busy} icon={Plus} onClick={submitChannel} variant="primary">{busy ? '提交中' : '提交渠道审核'}</Button></>}>
      {error && <InlineNotice tone="danger">{error}</InlineNotice>}
      <Field label="渠道名称"><TextInput onChange={(event) => setChannelName(event.target.value)} placeholder="例如 OpenAI 企业线路" value={channelName} /></Field><Field label="渠道类型"><Select onChange={(event) => setChannelType(event.target.value)} value={channelType}><option value="1">OpenAI 兼容</option><option value="14">Anthropic</option><option value="20">OpenRouter</option><option value="24">Gemini</option><option value="43">DeepSeek</option></Select></Field><Field label="上游基础地址"><TextInput onChange={(event) => setBaseURL(event.target.value)} placeholder="https://api.example.com/v1" value={baseURL} /></Field><Field label="上游 API 密钥"><TextInput onChange={(event) => setAPIKey(event.target.value)} placeholder="保存后不再回显" type="password" value={apiKey} /></Field><Field label="拟售模型"><textarea className="textarea" onChange={(event) => setModels(event.target.value)} rows="3" value={models} /></Field>
    </Modal>
  );
}

function WithdrawalModal({ close, config, notify }) {
  const [amount, setAmount] = useState('');
  const [recipientType, setRecipientType] = useState('alipay');
  const [recipientAccount, setRecipientAccount] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const availableQuota = Number(config.balance?.available_quota || 0);
  const submit = async () => {
    const amountQuota = Math.round(Number(amount) * 500_000);
    if (!Number.isSafeInteger(amountQuota) || amountQuota <= 0 || amountQuota > availableQuota) {
      setError('提现金额必须大于 0 且不超过可提现余额');
      return;
    }
    if (recipientAccount.trim().length < 5) {
      setError('请填写有效收款账户');
      return;
    }
    setBusy(true);
    setError('');
    try {
      await apiRequest('/api/merchant/withdrawals', { method: 'POST', body: {
        amount_quota: amountQuota,
        recipient_type: recipientType,
        recipient_account: recipientAccount.trim(),
      } });
      await config.onSaved?.();
      notify('提现申请已提交审核');
      close();
    } catch (requestError) {
      setError(requestError.message || '提交提现申请失败');
    } finally {
      setBusy(false);
    }
  };
  return (
    <Modal title="申请人工提现" description="申请后立即冻结对应余额，由管理员审核并线下打款。" onClose={close} footer={<><Button onClick={close}>取消</Button><Button disabled={busy} icon={Landmark} onClick={submit} variant="primary">{busy ? '提交中' : '提交提现申请'}</Button></>}>
      {error && <InlineNotice tone="danger">{error}</InlineNotice>}
      <Field label="提现金额（美元）" hint={`当前可提现 ${formatQuota(availableQuota)}`}><div className="prefix-input"><span>$</span><TextInput min="0.01" onChange={(event) => setAmount(event.target.value)} step="0.01" type="number" value={amount} /></div></Field>
      <Field label="收款方式"><Select onChange={(event) => setRecipientType(event.target.value)} value={recipientType}><option value="alipay">支付宝</option><option value="bank">银行卡</option><option value="other">其他</option></Select></Field>
      <Field label="收款账户" hint="账户信息加密保存，仅管理员处理打款时可见"><TextInput maxLength="256" onChange={(event) => setRecipientAccount(event.target.value)} value={recipientAccount} /></Field>
    </Modal>
  );
}

function SettlementActionModal({ close, config, notify }) {
  const action = config.action;
  const [reason, setReason] = useState('');
  const [reference, setReference] = useState('');
  const [proof, setProof] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const definitions = {
    approve: { title: '通过提现申请', submit: '确认通过', notice: '提现申请已通过，等待线下打款' },
    reject: { title: '驳回提现申请', submit: '确认驳回', notice: '提现申请已驳回并释放冻结余额' },
    paid: { title: '确认线下打款', submit: '确认已打款', notice: '线下打款凭证已登记' },
    ignore: { title: '忽略对账异常', submit: '确认忽略', notice: '对账异常已标记为忽略' },
  };
  const definition = definitions[action];
  const submit = async () => {
    setBusy(true);
    setError('');
    try {
      const reconciliation = action === 'ignore';
      const path = reconciliation
        ? `/api/admin/settlement-reconciliation/${config.item.id}/ignore`
        : `/api/admin/withdrawals/${config.item.id}/${action}`;
      const body = action === 'paid'
        ? { payout_reference: reference.trim(), payout_proof: proof.trim() }
        : { reason: reason.trim() };
      await apiRequest(path, { method: 'POST', body });
      await config.onSaved?.();
      notify(definition.notice);
      close();
    } catch (requestError) {
      setError(requestError.message || '操作失败');
    } finally {
      setBusy(false);
    }
  };
  return (
    <Modal title={definition.title} description={action === 'ignore' ? `New API 日志 ${config.item.newapi_log_id}` : `${config.item.merchant_name} · ${config.item.withdrawal_no}`} onClose={close} footer={<><Button onClick={close}>取消</Button><Button disabled={busy} icon={Check} onClick={submit} variant="primary">{busy ? '处理中' : definition.submit}</Button></>}>
      {error && <InlineNotice tone="danger">{error}</InlineNotice>}
      {action === 'paid' ? <><Field label="打款流水号"><TextInput maxLength="200" onChange={(event) => setReference(event.target.value)} value={reference} /></Field><Field label="凭证说明"><textarea className="textarea" maxLength="2000" onChange={(event) => setProof(event.target.value)} rows="4" value={proof} /></Field></> : <Field label={action === 'ignore' ? '处置原因' : '审核原因'}><textarea className="textarea" maxLength="500" onChange={(event) => setReason(event.target.value)} rows="4" value={reason} /></Field>}
    </Modal>
  );
}

function KeyModal({ config, close, notify }) {
  const key = config.key;
  const [name, setName] = useState(key?.name || '');
  const [unlimitedQuota, setUnlimitedQuota] = useState(key?.unlimited_quota ?? true);
  const [remainQuota, setRemainQuota] = useState(String(key?.remain_quota || 0));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const submit = async () => {
    setBusy(true);
    setError('');
    try {
      await apiRequest(key ? `/api/keys/${key.id}` : '/api/keys', {
        method: key ? 'PUT' : 'POST',
        body: {
          name: name.trim(),
          expired_time: key?.expired_time ?? -1,
          remain_quota: Number(remainQuota || 0),
          unlimited_quota: unlimitedQuota,
          model_limits_enabled: Boolean(key?.model_limits_enabled),
          model_limits: key?.model_limits || '',
          allow_ips: key?.allow_ips || null,
          group: key?.group || config.group || 'default',
          cross_group_retry: Boolean(key?.cross_group_retry),
        },
      });
      await config.onSaved?.();
      notify(key ? '密钥已更新' : '密钥已创建');
      close();
    } catch (requestError) {
      setError(requestError.message || '保存密钥失败');
    } finally {
      setBusy(false);
    }
  };

  return (
    <Modal title={key ? '编辑 API 密钥' : '创建 API 密钥'} description="密钥由 New API 创建并管理。" onClose={close} footer={<><Button onClick={close}>取消</Button><Button disabled={busy || !name.trim()} icon={KeyRound} onClick={submit} variant="primary">{busy ? '保存中' : '保存密钥'}</Button></>}>
      {error && <InlineNotice tone="danger">{error}</InlineNotice>}
      <Field label="密钥名称"><TextInput maxLength="50" onChange={(event) => setName(event.target.value)} value={name} /></Field>
      <label className="toggle-line"><input checked={unlimitedQuota} onChange={(event) => setUnlimitedQuota(event.target.checked)} type="checkbox" /><span><strong>不限额</strong><small>关闭后按下方整数额度限制使用</small></span></label>
      {!unlimitedQuota && <Field label="可用额度"><TextInput min="0" onChange={(event) => setRemainQuota(event.target.value)} type="number" value={remainQuota} /></Field>}
    </Modal>
  );
}

function AdminUserModal({ config, close, notify }) {
  const quotaMode = config.type === 'adminQuota';
  const [mode, setMode] = useState('add');
  const [value, setValue] = useState('');
  const [password, setPassword] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const submit = async () => {
    setBusy(true);
    setError('');
    try {
      await apiRequest(`/api/admin/users/${config.user.id}/${quotaMode ? 'quota' : 'password'}`, {
        method: 'POST',
        body: quotaMode ? { mode, value: Number(value) } : { password },
      });
      await config.onSaved?.();
      notify(quotaMode ? '用户额度已调整' : '用户密码已重置');
      close();
    } catch (requestError) {
      setError(requestError.message || '操作失败');
    } finally {
      setBusy(false);
    }
  };

  return (
    <Modal title={quotaMode ? '调整用户额度' : '重置用户密码'} description={`${config.user.display_name || config.user.username} · UID ${config.user.id}`} onClose={close} footer={<><Button onClick={close}>取消</Button><Button disabled={busy || (quotaMode ? value === '' : password.length < 8)} icon={quotaMode ? CreditCard : LockKeyhole} onClick={submit} variant="primary">{busy ? '处理中' : '确认提交'}</Button></>}>
      {error && <InlineNotice tone="danger">{error}</InlineNotice>}
      {quotaMode ? <div className="form-grid two-column"><Field label="调整方式"><Select onChange={(event) => setMode(event.target.value)} value={mode}><option value="add">增加</option><option value="subtract">扣减</option><option value="override">设为</option></Select></Field><Field label="整数额度"><TextInput min="0" onChange={(event) => setValue(event.target.value)} type="number" value={value} /></Field></div> : <Field label="新密码" hint="长度为 8 到 20 位"><TextInput maxLength="20" minLength="8" onChange={(event) => setPassword(event.target.value)} type="password" value={password} /></Field>}
    </Modal>
  );
}

export default function Modals({ modal, close, notify, onAuthenticated, setModal }) {
  if (!modal) return null;
  const type = typeof modal === 'string' ? modal : modal.type;
  if (['login', 'register', 'forgot', 'twoFactor', 'resetPassword'].includes(type)) return <AuthModal close={close} mode={type} notify={notify} onAuthenticated={onAuthenticated} reset={typeof modal === 'object' ? modal : null} setModal={setModal} />;
  if (type === 'key') return <KeyModal close={close} config={modal} notify={notify} />;
  if (type === 'adminQuota' || type === 'adminPassword') return <AdminUserModal close={close} config={modal} notify={notify} />;
  if (type === 'route') return <RouteModal close={close} config={typeof modal === 'object' ? modal : {}} notify={notify} />;
  if (type === 'quote') return <QuoteModal close={close} config={typeof modal === 'object' ? modal : {}} notify={notify} />;
  if (type === 'payment') return <PaymentModal close={close} config={typeof modal === 'object' ? modal : {}} notify={notify} />;
  if (type === 'recharge') return <RechargeModal close={close} config={typeof modal === 'object' ? modal : {}} notify={notify} />;
  if (type === 'creditResolve') return <CreditResolveModal close={close} config={modal} notify={notify} />;
  if (type === 'merchantApply') return <MerchantApplyModal close={close} config={typeof modal === 'object' ? modal : {}} notify={notify} />;
  if (type === 'channel') return <ChannelModal close={close} config={typeof modal === 'object' ? modal : {}} notify={notify} />;
  if (type === 'withdraw') return <WithdrawalModal close={close} config={typeof modal === 'object' ? modal : {}} notify={notify} />;
  if (type === 'withdrawalAction' || type === 'reconciliationAction') return <SettlementActionModal close={close} config={typeof modal === 'object' ? modal : {}} notify={notify} />;
  return null;
}
