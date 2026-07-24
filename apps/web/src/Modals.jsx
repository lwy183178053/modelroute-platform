import { useEffect, useState } from 'react';
import { Check, CreditCard, KeyRound, Landmark, LockKeyhole, Plus, RotateCcw, Send, ShieldCheck } from 'lucide-react';
import AuthModal from './AuthModal';
import { apiRequest, formatPriceMicros, formatQuota, getQuotaPerUnit } from './api';
import { Button, Field, InlineNotice, Modal, Select, Status, TextInput } from './ui';

function Footer({ cancel, submit, submitText, icon = Check }) {
  return <><Button onClick={cancel}>取消</Button><Button icon={icon} variant="primary" onClick={submit}>{submitText}</Button></>;
}

function StatusLine({ result }) {
  return <InlineNotice>检测通过 · {result.latency_ms} ms</InlineNotice>;
}

function merchantNameUnits(value) {
  let units = 0;
  for (const character of value) {
    if (/^[A-Za-z0-9]$/.test(character)) units += 1;
    else if (/^\p{Script=Han}$/u.test(character)) units += 2;
    else return -1;
  }
  return units;
}

function RouteModal({ close, config, notify }) {
  const policy = config.policy || {};
  const editing = Boolean(policy.id);
  const [name, setName] = useState(policy.name || '');
  const [priority, setPriority] = useState(policy.priority || 'balanced');
  const [minimum, setMinimum] = useState(String(Number(policy.minimum_multiplier_bps ?? 0) / 10_000));
  const [maximum, setMaximum] = useState(String(Number(policy.maximum_multiplier_bps ?? 50_000) / 10_000));
  const [cacheFloorEnabled, setCacheFloorEnabled] = useState(policy.minimum_cache_hit_ppm != null);
  const [cacheFloor, setCacheFloor] = useState(String(Number(policy.minimum_cache_hit_ppm ?? 0) / 10_000));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const submit = async () => {
    if (!name.trim()) {
      setError('请输入路由名称');
      return;
    }
    const minimumBPS = Math.round(Number(minimum) * 10_000);
    const maximumBPS = Math.round(Number(maximum) * 10_000);
    if (!Number.isInteger(minimumBPS) || !Number.isInteger(maximumBPS) || minimumBPS < 0 || maximumBPS > 50_000 || minimumBPS > maximumBPS) {
      setError('倍率区间必须在 0.00x 到 5.00x 之间');
      return;
    }
    const minimumCacheHitPPM = cacheFloorEnabled ? Math.round(Number(cacheFloor) * 10_000) : null;
    if (minimumCacheHitPPM != null && (!Number.isInteger(minimumCacheHitPPM) || minimumCacheHitPPM < 0 || minimumCacheHitPPM > 1_000_000)) {
      setError('最低缓存命中率必须在 0.00% 到 100.00% 之间');
      return;
    }
    setBusy(true);
    setError('');
    try {
      await apiRequest(editing ? `/api/routing-policies/${policy.id}` : '/api/routing-policies', {
        method: editing ? 'PUT' : 'POST',
        body: {
          name: name.trim(), priority,
          minimum_multiplier_bps: minimumBPS,
          maximum_multiplier_bps: maximumBPS,
          minimum_cache_hit_ppm: minimumCacheHitPPM,
        },
      });
      await config.onSaved?.();
      notify(editing ? '路由已保存' : '路由已创建');
      close();
    } catch (requestError) {
      setError(requestError.message || '保存路由策略失败');
    } finally {
      setBusy(false);
    }
  };
  return (
    <Modal
      title={editing ? '编辑路由策略' : '创建路由'}
      description="路由可被多把 API 密钥复用，修改后立即影响全部绑定密钥。"
      onClose={close}
      wide
      footer={<><Button onClick={close}>取消</Button><Button disabled={busy} icon={Check} variant="primary" onClick={submit}>{busy ? '保存中' : '保存策略'}</Button></>}
    >
      {error && <InlineNotice tone="danger">{error}</InlineNotice>}
      <Field label="路由名称"><TextInput maxLength="50" onChange={(event) => setName(event.target.value)} placeholder="例如：稳定主路由" value={name} /></Field>
      <div className="form-grid two-column">
        <Field label="最低成交倍率" hint="低于此倍率的报价不参与路由"><TextInput value={minimum} onChange={(event) => setMinimum(event.target.value)} min="0" max="5" step="0.01" type="number" /></Field>
        <Field label="最高成交倍率" hint="高于此倍率的报价不参与路由"><TextInput value={maximum} onChange={(event) => setMaximum(event.target.value)} min="0" max="5" step="0.01" type="number" /></Field>
      </div>

      <div className="form-section"><h3>选择优先方式</h3><div className="radio-row priority-options">
        <label><input checked={priority === 'balanced'} name="priority" onChange={() => setPriority('balanced')} type="radio" />均衡分配<span>价格、成功率、速度和缓存各占 25%</span></label>
        <label><input checked={priority === 'price'} name="priority" onChange={() => setPriority('price')} type="radio" />价格优先<span>价格权重 70%</span></label>
        <label><input checked={priority === 'success'} name="priority" onChange={() => setPriority('success')} type="radio" />成功率优先<span>成功率权重 70%</span></label>
        <label><input checked={priority === 'latency'} name="priority" onChange={() => setPriority('latency')} type="radio" />响应速度优先<span>首字速度权重 70%</span></label>
        <label><input checked={priority === 'cache'} name="priority" onChange={() => setPriority('cache')} type="radio" />缓存命中率优先<span>缓存指标权重 70%</span></label>
      </div></div>

      <div className="form-section cache-floor-setting"><h3>缓存筛选</h3>
        <label className="toggle-line"><input aria-label="启用最低缓存命中率" checked={cacheFloorEnabled} onChange={(event) => setCacheFloorEnabled(event.target.checked)} type="checkbox" /><span><strong>启用最低缓存命中率</strong><small>只保留样本充足且达到阈值的文本商户，图片模型不受影响</small></span></label>
        {cacheFloorEnabled && <Field label="最低缓存命中率" hint="按最近 7 天商户与模型汇总数据判断"><div className="suffix-input"><TextInput max="100" min="0" onChange={(event) => setCacheFloor(event.target.value)} step="0.01" type="number" value={cacheFloor} /><span>%</span></div></Field>}
      </div>
      <InlineNotice tone="warning">没有符合策略条件的候选时返回无可用渠道错误，本次请求不扣费。</InlineNotice>
    </Modal>
  );
}

function QuoteModal({ close, config, notify }) {
  const initialModel = config.models?.find((model) => model.public_name === config.offer?.model) || config.models?.[0];
  const [provider, setProvider] = useState(initialModel?.provider || '');
  const [modelID, setModelID] = useState(String(initialModel?.id || ''));
  const [multiplier, setMultiplier] = useState(String(Number(config.offer?.multiplier_bps ?? 10_000) / 10_000));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const providers = [...new Set((config.models || []).map((model) => model.provider).filter(Boolean))];
  const providerModels = (config.models || []).filter((model) => model.provider === provider);
  const selectedModel = config.models?.find((model) => String(model.id) === modelID);
  const selectedTier = selectedModel?.price?.tiers?.[0];
  const multiplierBPS = Math.round(Number(multiplier) * 10_000);
  const formatAdjusted = (value) => {
    if (value == null || !Number.isInteger(multiplierBPS)) return '-';
    if (multiplierBPS === 0) return '免费';
    return formatPriceMicros(Math.round(Number(value) * multiplierBPS / 10_000));
  };
  const submit = async () => {
    if (!Number.isInteger(multiplierBPS) || multiplierBPS < 0 || multiplierBPS > 50_000) {
      setError('价格倍率必须在 0.00x 到 5.00x 之间');
      return;
    }
    setBusy(true);
    setError('');
    try {
      await apiRequest(`/api/merchant/offers/${modelID}`, { method: 'PUT', body: { multiplier_bps: multiplierBPS } });
      await config.onSaved?.();
      notify('模型价格已生效');
      close();
    } catch (requestError) {
      setError(requestError.message || '保存模型价格失败');
    } finally {
      setBusy(false);
    }
  };
  return (
    <Modal title="设置模型价格" description="同一模型只保留一个价格，并自动应用到该模型的全部已审核渠道。" onClose={close} footer={<><Button onClick={close}>取消</Button><Button disabled={busy || !modelID} icon={Check} variant="primary" onClick={submit}>{busy ? '同步中' : '保存模型价格'}</Button></>}>
      {error && <InlineNotice tone="danger">{error}</InlineNotice>}
      <div className="form-grid two-column">
        <Field label="模型厂商"><Select value={provider} onChange={(event) => { const nextProvider = event.target.value; setProvider(nextProvider); setModelID(String((config.models || []).find((model) => model.provider === nextProvider)?.id || '')); }}>{providers.map((item) => <option key={item} value={item}>{item}</option>)}</Select></Field>
        <Field label="模型型号"><Select value={modelID} onChange={(event) => setModelID(event.target.value)}>{providerModels.map((model) => <option key={model.id} value={model.id}>{model.public_name}</option>)}</Select></Field>
      </div>
      <div className="quote-summary"><div><span>价格版本</span><strong>v{selectedModel?.price?.version || '-'}</strong></div><div><span>首档基础价格</span><strong>{selectedTier ? `${formatPriceMicros(selectedTier.input_price_micros)} 输入${selectedTier.output_price_micros == null ? '' : ` / ${formatPriceMicros(selectedTier.output_price_micros)} 输出`}` : '图片价格矩阵'}</strong></div><div><span>倍率后价格</span><strong>{selectedTier ? `${formatAdjusted(selectedTier.input_price_micros)} 输入${selectedTier.output_price_micros == null ? '' : ` / ${formatAdjusted(selectedTier.output_price_micros)} 输出`}` : multiplierBPS === 0 ? '免费' : '按图片价格矩阵计算'}</strong></div></div>
      <Field label="价格倍率" hint="允许范围 0.00x - 5.00x"><div className="suffix-input"><TextInput max="5" min="0" onChange={(event) => setMultiplier(event.target.value)} step="0.01" type="number" value={multiplier} /><span>x</span></div></Field>
      <InlineNotice>调价会生成新的价格版本；已进入请求的价格不会变化。</InlineNotice>
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
  const [minimumCNY, setMinimumCNY] = useState(String(Number(setting.minimum_cny_cents || 100) / 100));
  const [maximumCNY, setMaximumCNY] = useState(String(Number(setting.maximum_cny_cents || 2000000) / 100));
  const [withdrawalTaxRate, setWithdrawalTaxRate] = useState(String(Number(setting.withdrawal_tax_bps ?? 300) / 100));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const toggleMethod = (method) => setMethods((current) => current.includes(method) ? current.filter((item) => item !== method) : [...current, method]);
  const submit = async () => {
    const minimumCNYCents = Math.round(Number(minimumCNY) * 100);
    const maximumCNYCents = Math.round(Number(maximumCNY) * 100);
    const withdrawalTaxFormatOK = /^(?:100(?:\.0{1,2})?|\d{1,2}(?:\.\d{1,2})?)$/.test(withdrawalTaxRate.trim());
    const withdrawalTaxBPS = Math.round(Number(withdrawalTaxRate) * 100);
    if (!gatewayURL.trim() || methods.length === 0 || !Number.isInteger(minimumCNYCents) || !Number.isInteger(maximumCNYCents) || minimumCNYCents <= 0 || maximumCNYCents < minimumCNYCents || !withdrawalTaxFormatOK || withdrawalTaxBPS < 0 || withdrawalTaxBPS > 10_000) {
      setError('请检查网关、支付方式、人民币充值范围和提现税率');
      return;
    }
    setBusy(true);
    setError('');
    try {
      await apiRequest('/api/admin/payment-settings', { method: 'PUT', body: {
        gateway_url: gatewayURL.trim(), merchant_pid: merchantPID.trim(), merchant_key: merchantKey,
        enabled, payment_methods: methods, minimum_cny_cents: minimumCNYCents,
        maximum_cny_cents: maximumCNYCents, withdrawal_tax_bps: withdrawalTaxBPS,
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
        <Field label="最低充值金额（人民币）"><TextInput min="0.01" onChange={(event) => setMinimumCNY(event.target.value)} step="0.01" type="number" value={minimumCNY} /></Field>
        <Field label="最高充值金额（人民币）"><TextInput min="0.01" onChange={(event) => setMaximumCNY(event.target.value)} step="0.01" type="number" value={maximumCNY} /></Field>
        <Field label="提现税率（%）" hint="范围 0.00% - 100.00%，最多两位小数"><div className="suffix-input"><TextInput max="100" min="0" onChange={(event) => setWithdrawalTaxRate(event.target.value)} step="0.01" type="number" value={withdrawalTaxRate} /><span>%</span></div></Field>
      </div>
      <Field label="已启用支付方式"><div className="checkbox-line">{[['alipay', '支付宝'], ['wxpay', '微信'], ['qqpay', 'QQ 钱包']].map(([value, label]) => <label key={value}><input checked={methods.includes(value)} onChange={() => toggleMethod(value)} type="checkbox" />{label}</label>)}</div></Field>
      <InlineNotice tone="warning">保存会锁定为新配置版本；已创建订单继续使用原版本验签和金额。</InlineNotice>
    </Modal>
  );
}

function RechargeModal({ close, config, notify }) {
  const [setting, setSetting] = useState(null);
  const [amount, setAmount] = useState('');
  const [paymentType, setPaymentType] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  useEffect(() => {
    apiRequest('/api/payment-settings').then((data) => {
      setSetting(data);
      setAmount(String(Number(data.minimum_cny_cents || 100) / 100));
      setPaymentType(data.payment_methods?.[0] || '');
    }).catch((requestError) => setError(requestError.message || '读取充值配置失败'));
  }, []);
  const amountCNYCents = Math.round(Number(amount) * 100);
  const submit = async () => {
    if (!setting || !Number.isInteger(amountCNYCents) || amountCNYCents < setting.minimum_cny_cents || amountCNYCents > setting.maximum_cny_cents || !paymentType) {
      setError('充值额度或支付方式不在允许范围');
      return;
    }
    setBusy(true);
    setError('');
    try {
      const data = await apiRequest('/api/payment-orders', { method: 'POST', body: { amount_cny_cents: amountCNYCents, payment_type: paymentType } });
      await config.onSaved?.(data.order);
      notify('充值订单已创建，正在进入支付页');
      window.location.assign(data.payment_url);
    } catch (requestError) {
      setError(requestError.message || '创建充值订单失败');
      setBusy(false);
    }
  };
  const presets = setting ? [...new Set([setting.minimum_cny_cents, 5000, 10000, 50000])].filter((value) => value >= setting.minimum_cny_cents && value <= setting.maximum_cny_cents) : [];
  return (
    <Modal title="账户充值" description="充值金额与到账余额均按人民币计算。" onClose={close} footer={<><Button onClick={close}>取消</Button><Button disabled={busy || !setting} icon={CreditCard} variant="primary" onClick={submit}>{busy ? '创建中' : '前往支付'}</Button></>}>
      {error && <InlineNotice tone="danger">{error}</InlineNotice>}
      <Field label="充值金额（人民币）"><div className="prefix-input"><span>¥</span><TextInput min={Number(setting?.minimum_cny_cents || 1) / 100} max={Number(setting?.maximum_cny_cents || 1) / 100} onChange={(event) => setAmount(event.target.value)} step="0.01" type="number" value={amount} /></div></Field>
      <div className="amount-presets">{presets.map((value) => <button className={value === amountCNYCents ? 'active' : ''} key={value} onClick={() => setAmount(String(value / 100))} type="button">¥{(value / 100).toLocaleString('zh-CN')}</button>)}</div>
      {setting && <div className="quote-summary"><div><span>人民币支付</span><strong>¥{(amountCNYCents / 100).toFixed(2)}</strong></div><div><span>预计到账</span><strong>¥{(amountCNYCents / 100).toFixed(2)}</strong></div></div>}
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
    <Field label="核对原因"><textarea className="textarea" onChange={(event) => setReason(event.target.value)} placeholder="填写已核对的消费日志或余额依据" rows="4" value={reason} /></Field>
    <InlineNotice tone="warning">选择重新入账前，必须确认此前请求没有增加用户额度。</InlineNotice>
  </Modal>;
}

function MerchantApplyModal({ close, config, notify }) {
  const [displayName, setDisplayName] = useState('');
  const [websiteURL, setWebsiteURL] = useState('');
  const [channelName, setChannelName] = useState('主线路');
  const [modelID, setModelID] = useState(String(config.models?.[0]?.id || ''));
  const [baseURL, setBaseURL] = useState('');
  const [apiKey, setAPIKey] = useState('');
  const [testReceipt, setTestReceipt] = useState('');
  const [testResult, setTestResult] = useState(null);
  const [description, setDescription] = useState('');
  const [busy, setBusy] = useState(false);
  const [testing, setTesting] = useState(false);
  const [error, setError] = useState('');
  const displayNameUnits = merchantNameUnits(displayName);

  const testChannel = async () => {
    setTesting(true);
    setError('');
    setTestReceipt('');
    setTestResult(null);
    try {
      const result = await apiRequest('/api/merchant/channel-tests', { method: 'POST', body: {
        canonical_model_id: Number(modelID), base_url: baseURL.trim(), api_key: apiKey,
      } });
      setTestReceipt(result.test_receipt);
      setTestResult(result);
    } catch (requestError) {
      setError(requestError.message || '渠道检测失败');
    } finally {
      setTesting(false);
    }
  };

  const submit = async () => {
    if (displayNameUnits < 2 || displayNameUnits > 16) {
      setError(displayNameUnits < 0 ? '商户名称仅支持汉字、英文字母和数字' : '商户名称长度需为 2 至 16 个单位');
      return;
    }
    setBusy(true);
    setError('');
    try {
      const application = await apiRequest('/api/merchant/application', {
        method: 'POST',
        body: {
          display_name: displayName.trim(), website_url: websiteURL.trim(), business_description: description.trim(),
          channel_name: channelName.trim(), canonical_model_id: Number(modelID), test_receipt: testReceipt,
          base_url: baseURL.trim(), api_key: apiKey,
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
    <Modal title="申请成为商户" description="选择一个标准模型并通过 hi 检测后提交。" onClose={close} footer={<><Button onClick={close}>取消</Button><Button disabled={busy || !testReceipt} icon={Send} onClick={submit} variant="primary">{busy ? '提交中' : '提交申请'}</Button></>} wide>
      {error && <InlineNotice tone="danger">{error}</InlineNotice>}
      <div className="form-grid two-column">
        <Field label="商户名称" hint={displayNameUnits < 0 ? '仅支持汉字、英文字母和数字' : `${displayNameUnits}/16 单位，汉字计 2，英文数字计 1`}><TextInput maxLength="16" onChange={(event) => setDisplayName(event.target.value)} placeholder="用于市场展示" value={displayName} /></Field>
        <Field label="网站地址"><TextInput onChange={(event) => setWebsiteURL(event.target.value)} placeholder="https://example.com" value={websiteURL} /></Field>
        <Field label="渠道名称"><TextInput maxLength="100" onChange={(event) => setChannelName(event.target.value)} value={channelName} /></Field>
        <Field label="标准模型"><Select onChange={(event) => { setModelID(event.target.value); setTestReceipt(''); }} value={modelID}>{(config.models || []).map((model) => <option key={model.id} value={model.id}>{model.public_name} · {model.provider}</option>)}</Select></Field>
        <Field label="上游基础地址"><TextInput onChange={(event) => { setBaseURL(event.target.value); setTestReceipt(''); }} placeholder="https://api.example.com/v1" value={baseURL} /></Field>
        <Field label="上游 API 密钥" hint="提交后加密保存且不再回显"><TextInput onChange={(event) => { setAPIKey(event.target.value); setTestReceipt(''); }} placeholder="输入渠道密钥" type="password" value={apiKey} /></Field>
      </div>
      <div className="config-actions"><Button disabled={testing || !modelID || !baseURL.trim() || apiKey.trim().length < 8} icon={ShieldCheck} onClick={testChannel} variant="primary">{testing ? '检测中' : '发送 hi 检测'}</Button>{testResult && <StatusLine result={testResult} />}</div>
      <Field label="业务说明"><textarea className="textarea" onChange={(event) => setDescription(event.target.value)} placeholder="说明渠道来源、可售模型和服务能力" rows="4" value={description} /></Field>
    </Modal>
  );
}

function ChannelModal({ close, config, notify }) {
  const [channelName, setChannelName] = useState('');
  const [channelType, setChannelType] = useState('openai_compatible');
  const [baseURL, setBaseURL] = useState('');
  const [apiKey, setAPIKey] = useState('');
  const [syncedModels, setSyncedModels] = useState([]);
  const [syncedAt, setSyncedAt] = useState('');
  const [busy, setBusy] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [error, setError] = useState('');

  const syncModels = async () => {
    setSyncing(true);
    setError('');
    try {
      const result = await apiRequest('/api/merchant/channel-models/sync', { method: 'POST', body: {
        channel_type: channelType, base_url: baseURL.trim(), api_key: apiKey,
      } });
      setSyncedModels(result.items || []);
      setSyncedAt(result.synced_at || new Date().toISOString());
    } catch (requestError) {
      setError(requestError.message || '同步上游模型失败');
    } finally {
      setSyncing(false);
    }
  };
  const submitChannel = async () => {
    setBusy(true);
    setError('');
    try {
      await apiRequest('/api/merchant/channels', { method: 'POST', body: {
        channel_name: channelName.trim(), channel_type: channelType, base_url: baseURL.trim(), api_key: apiKey,
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
    <Modal title="提交新渠道" description="一份上游地址和密钥可以承载多个模型，平台会自动匹配已收录模型。" onClose={close} footer={<><Button onClick={close}>取消</Button><Button disabled={busy || !syncedModels.some((model) => model.match_status === 'matched')} icon={Plus} onClick={submitChannel} variant="primary">{busy ? '提交中' : '提交渠道审核'}</Button></>}>
      {error && <InlineNotice tone="danger">{error}</InlineNotice>}
      <Field label="渠道名称"><TextInput onChange={(event) => setChannelName(event.target.value)} placeholder="例如 OpenAI 企业线路" value={channelName} /></Field>
      <Field label="上游类型"><Select value={channelType} onChange={(event) => { setChannelType(event.target.value); setSyncedModels([]); }}><option value="openai_compatible">OpenAI 兼容</option><option value="anthropic">Anthropic</option><option value="gemini">Gemini</option></Select></Field>
      <Field label="上游基础地址"><TextInput onChange={(event) => { setBaseURL(event.target.value); setSyncedModels([]); }} placeholder="https://api.example.com/v1" value={baseURL} /></Field>
      <Field label="上游 API 密钥"><TextInput onChange={(event) => { setAPIKey(event.target.value); setSyncedModels([]); }} placeholder="保存后不再回显" type="password" value={apiKey} /></Field>
      <div className="config-actions"><Button disabled={syncing || !baseURL.trim() || apiKey.trim().length < 8} icon={ShieldCheck} onClick={syncModels} variant="primary">{syncing ? '获取中' : '一键获取模型'}</Button>{syncedAt && <InlineNotice>同步于 {new Date(syncedAt).toLocaleString('zh-CN')}</InlineNotice>}</div>
      {syncedModels.length > 0 && <div className="model-sync-list"><strong>已匹配 {syncedModels.filter((model) => model.match_status === 'matched').length} 个模型</strong>{syncedModels.map((model) => <div key={model.upstream_model_name}><span>{model.upstream_model_name}</span><Status>{model.match_status === 'matched' ? '已收录' : '平台未收录'}</Status></div>)}</div>}
    </Modal>
  );
}

function WithdrawalModal({ close, config, notify }) {
  const [amount, setAmount] = useState('');
  const [recipientAccount, setRecipientAccount] = useState('');
  const [withdrawalTaxBPS, setWithdrawalTaxBPS] = useState(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  useEffect(() => {
    let active = true;
    apiRequest('/api/payment-settings').then((data) => {
      const taxBPS = Number(data.withdrawal_tax_bps);
      if (!Number.isInteger(taxBPS) || taxBPS < 0 || taxBPS > 10_000) throw new Error('提现税率配置无效');
      if (active) setWithdrawalTaxBPS(taxBPS);
    }).catch((requestError) => {
      if (active) setError(requestError.message || '读取提现税率失败');
    });
    return () => { active = false; };
  }, []);
  const availableQuota = Number(config.balance?.available_quota || 0);
  const amountYuan = Number(amount);
  const quotaPerUnit = getQuotaPerUnit();
  const amountQuota = Number.isInteger(amountYuan) ? amountYuan * quotaPerUnit : 0;
  const taxReady = Number.isInteger(withdrawalTaxBPS);
  const taxQuota = taxReady ? Math.floor(amountQuota * withdrawalTaxBPS / 10_000) : 0;
  const netPayoutQuota = amountQuota - taxQuota;
  const submit = async () => {
    if (!Number.isInteger(amountYuan) || !Number.isSafeInteger(amountQuota) || amountYuan < 50 || amountQuota > availableQuota) {
      setError('提现金额必须为不低于 50 元且不超过可提现余额的整数');
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
    <Modal title="申请人工提现" description={taxReady ? `提现收取 ${(withdrawalTaxBPS / 100).toFixed(2)}% 税费，申请后冻结完整金额，由管理员审核并通过支付宝打款。` : '正在读取当前提现税率。'} onClose={close} footer={<><Button onClick={close}>取消</Button><Button disabled={busy || !taxReady} icon={Landmark} onClick={submit} variant="primary">{busy ? '提交中' : '提交提现申请'}</Button></>}>
      {error && <InlineNotice tone="danger">{error}</InlineNotice>}
      <Field label="提现金额（人民币）" hint={`当前可提现 ${formatQuota(availableQuota)}，最低 50 元且只支持整数`}><div className="prefix-input"><span>¥</span><TextInput max={Math.floor(availableQuota / quotaPerUnit)} min="50" onChange={(event) => setAmount(event.target.value)} step="1" type="number" value={amount} /></div></Field>
      <div className="withdrawal-summary"><span>申请金额<strong>{formatQuota(amountQuota)}</strong></span><span>税费（{taxReady ? `${(withdrawalTaxBPS / 100).toFixed(2)}%` : '-'}）<strong>{taxReady ? formatQuota(taxQuota) : '-'}</strong></span><span>预计到账<strong>{taxReady ? formatQuota(netPayoutQuota) : '-'}</strong></span></div>
      <Field label="支付宝账户" hint="账户信息加密保存，仅管理员处理打款时可见"><TextInput maxLength="256" onChange={(event) => setRecipientAccount(event.target.value)} value={recipientAccount} /></Field>
    </Modal>
  );
}

function WithdrawalActionModal({ close, config, notify }) {
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
  };
  const definition = definitions[action];
  const submit = async () => {
    setBusy(true);
    setError('');
    try {
      const path = `/api/admin/withdrawals/${config.item.id}/${action}`;
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
    <Modal title={definition.title} description={`${config.item.merchant_name} · ${config.item.withdrawal_no}`} onClose={close} footer={<><Button onClick={close}>取消</Button><Button disabled={busy} icon={Check} onClick={submit} variant="primary">{busy ? '处理中' : definition.submit}</Button></>}>
      {error && <InlineNotice tone="danger">{error}</InlineNotice>}
      {action === 'paid' ? <><InlineNotice>实际支付宝打款 <strong>{formatQuota(config.item.net_payout_quota)}</strong>，税费 {formatQuota(config.item.tax_quota)}</InlineNotice><Field label="打款流水号"><TextInput maxLength="200" onChange={(event) => setReference(event.target.value)} value={reference} /></Field><Field label="凭证说明"><textarea className="textarea" maxLength="2000" onChange={(event) => setProof(event.target.value)} rows="4" value={proof} /></Field></> : <Field label="审核原因"><textarea className="textarea" maxLength="500" onChange={(event) => setReason(event.target.value)} rows="4" value={reason} /></Field>}
    </Modal>
  );
}

function KeyModal({ config, close, notify }) {
  const key = config.key;
  const [name, setName] = useState(key?.name || '');
  const [routingPolicyID, setRoutingPolicyID] = useState(String(key?.routing_policy_id || config.routingPolicies?.[0]?.id || ''));
  const [unlimitedQuota, setUnlimitedQuota] = useState(key?.unlimited_quota ?? true);
  const [remainQuota, setRemainQuota] = useState(String(key?.remain_quota || 0));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const submit = async () => {
    if (!routingPolicyID) {
      setError('请选择绑定路由');
      return;
    }
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
          ...(!key && { routing_policy_id: Number(routingPolicyID) }),
        },
      });
      if (key && Number(routingPolicyID) !== Number(key.routing_policy_id)) {
        await apiRequest(`/api/keys/${key.id}/routing-policy`, { method: 'PUT', body: { routing_policy_id: Number(routingPolicyID) } });
      }
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
    <Modal title={key ? '编辑 API 密钥' : '创建 API 密钥'} description="每把密钥必须绑定一条路由，路由修改后立即生效。" onClose={close} footer={<><Button onClick={close}>取消</Button><Button disabled={busy || !name.trim() || !routingPolicyID} icon={KeyRound} onClick={submit} variant="primary">{busy ? '保存中' : '保存密钥'}</Button></>}>
      {error && <InlineNotice tone="danger">{error}</InlineNotice>}
      <Field label="密钥名称"><TextInput maxLength="50" onChange={(event) => setName(event.target.value)} value={name} /></Field>
      <Field label="绑定路由"><Select aria-label="绑定路由" onChange={(event) => setRoutingPolicyID(event.target.value)} value={routingPolicyID}><option value="">请选择路由</option>{(config.routingPolicies || []).map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}</Select></Field>
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
    const quotaValue = Math.round(Number(value) * getQuotaPerUnit());
    setBusy(true);
    setError('');
    try {
      await apiRequest(`/api/admin/users/${config.user.id}/${quotaMode ? 'quota' : 'password'}`, {
        method: 'POST',
        body: quotaMode ? { mode, value: quotaValue } : { password },
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
    <Modal title={quotaMode ? '调整用户额度' : '重置用户密码'} description={`${config.user.display_name || config.user.username} · 用户 ID ${config.user.id}`} onClose={close} footer={<><Button onClick={close}>取消</Button><Button disabled={busy || (quotaMode ? value === '' : password.length < 8)} icon={quotaMode ? CreditCard : LockKeyhole} onClick={submit} variant="primary">{busy ? '处理中' : '确认提交'}</Button></>}>
      {error && <InlineNotice tone="danger">{error}</InlineNotice>}
      {quotaMode ? <div className="form-grid two-column"><Field label="调整方式"><Select onChange={(event) => setMode(event.target.value)} value={mode}><option value="add">增加</option><option value="subtract">扣减</option><option value="override">设为</option></Select></Field><Field label="调整金额（人民币）"><TextInput min="0" onChange={(event) => setValue(event.target.value)} step="0.01" type="number" value={value} /></Field></div> : <Field label="新密码" hint="长度为 8 到 20 位"><TextInput maxLength="20" minLength="8" onChange={(event) => setPassword(event.target.value)} type="password" value={password} /></Field>}
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
  if (type === 'withdrawalAction') return <WithdrawalActionModal close={close} config={typeof modal === 'object' ? modal : {}} notify={notify} />;
  return null;
}
