import { useState } from 'react';
import { Check, CreditCard, KeyRound, Landmark, LockKeyhole, Plus, Send, ShieldCheck, Store } from 'lucide-react';
import AuthModal from './AuthModal';
import { apiRequest } from './api';
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
  const [ratio, setRatio] = useState('0.72');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const selectedModel = config.models?.find((model) => String(model.id) === modelID);
  const submit = async () => {
    const multiplierBPS = Math.round(Number(ratio) * 10_000);
    if (!Number.isInteger(multiplierBPS) || multiplierBPS < 1 || multiplierBPS > 10_000) {
      setError('成交倍率必须在 0.0001x 到 1.0000x 之间');
      return;
    }
    setBusy(true);
    setError('');
    try {
      await apiRequest('/api/merchant/offers', { method: 'POST', body: {
        merchant_channel_id: Number(channelID), canonical_model_id: Number(modelID), multiplier_bps: multiplierBPS,
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
      <div className="quote-summary"><div><span>官方价格版本</span><strong>v{selectedModel?.price?.version || '-'}</strong></div><div><span>计费模式</span><strong>{selectedModel?.price?.billing_mode === 'fixed' ? '固定计费' : 'Token 计费'}</strong></div><div><span>倍率上限</span><strong>1.0000x</strong></div></div>
      <Field label="成交倍率" hint="允许范围 0.0001x - 1.0000x"><div className="suffix-input"><TextInput max="1" min="0.0001" onChange={(event) => setRatio(event.target.value)} step="0.0001" type="number" value={ratio} /><span>x</span></div></Field>
      <InlineNotice>调价将创建不可变的新报价版本；已进入请求的价格不会变化。</InlineNotice>
    </Modal>
  );
}

function PaymentModal({ close, notify }) {
  return (
    <Modal title="编辑易支付配置" description="保存后生成新配置版本，完整 KEY 不再回显。" onClose={close} footer={<Footer cancel={close} submitText="保存支付配置" submit={() => { close(); notify('易支付配置已保存为新版本'); }} icon={CreditCard} />} wide>
      <div className="form-grid two-column">
        <Field label="网关地址"><TextInput defaultValue="https://zpayz.cn/" /></Field>
        <Field label="商户编号 PID"><TextInput defaultValue="2026********1878" /></Field>
        <Field label="商户密钥 KEY" hint="留空代表保持当前密钥；输入新值将轮换密钥"><TextInput placeholder="输入新的商户密钥" type="password" /></Field>
        <Field label="支付状态"><Select defaultValue="启用"><option>启用</option><option>停用</option></Select></Field>
        <Field label="最低充值金额"><TextInput defaultValue="10.00" type="number" /></Field>
        <Field label="最高充值金额"><TextInput defaultValue="20000.00" type="number" /></Field>
      </div>
      <Field label="已启用支付方式"><div className="checkbox-line"><label><input defaultChecked type="checkbox" />支付宝</label><label><input defaultChecked type="checkbox" />微信</label><label><input defaultChecked type="checkbox" />QQ 钱包</label></div></Field>
      <InlineNotice tone="warning">配置保存后应再次执行“测试连接”和“验签自检”，再用于生产充值。</InlineNotice>
    </Modal>
  );
}

function RechargeModal({ close, notify }) {
  return (
    <Modal title="账户充值" description="支付成功后等待服务端回调验签和 New API 额度入账。" onClose={close} footer={<Footer cancel={close} submitText="前往支付" submit={() => { close(); notify('演示订单已创建'); }} icon={CreditCard} />}>
      <Field label="充值金额"><div className="prefix-input"><span>¥</span><TextInput defaultValue="500.00" type="number" /></div></Field>
      <div className="amount-presets">{['¥50', '¥100', '¥500', '¥1,000'].map((item) => <button className={item === '¥500' ? 'active' : ''} key={item} type="button">{item}</button>)}</div>
      <Field label="支付方式"><div className="payment-methods"><label><input defaultChecked name="payment-method" type="radio" />支付宝</label><label><input name="payment-method" type="radio" />微信</label><label><input name="payment-method" type="radio" />QQ 钱包</label></div></Field>
    </Modal>
  );
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

function SimpleMerchantModal({ type, close, config, notify }) {
  const withdrawal = type === 'withdraw';
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
    <Modal title={withdrawal ? '申请人工提现' : '提交新渠道'} description={withdrawal ? '申请后冻结对应余额，由管理员审核并线下打款。' : '新渠道验证通过后才会导入 New API。'} onClose={close} footer={withdrawal ? <Footer cancel={close} submitText="提交提现申请" submit={() => { close(); notify('提现申请已提交审核'); }} icon={Landmark} /> : <><Button onClick={close}>取消</Button><Button disabled={busy} icon={Plus} onClick={submitChannel} variant="primary">{busy ? '提交中' : '提交渠道审核'}</Button></>}>
      {error && <InlineNotice tone="danger">{error}</InlineNotice>}
      {withdrawal ? <><Field label="提现金额"><div className="prefix-input"><span>¥</span><TextInput defaultValue="5000.00" type="number" /></div></Field><Field label="收款账户"><Select><option>支付宝 · 138****2621</option><option>银行卡 · 6217 **** 1098</option></Select></Field><InlineNotice tone="warning">可提现余额 ¥18,642.30，本次金额将在申请后冻结。</InlineNotice></> : <><Field label="渠道名称"><TextInput onChange={(event) => setChannelName(event.target.value)} placeholder="例如 OpenAI 企业线路" value={channelName} /></Field><Field label="渠道类型"><Select onChange={(event) => setChannelType(event.target.value)} value={channelType}><option value="1">OpenAI 兼容</option><option value="14">Anthropic</option><option value="20">OpenRouter</option><option value="24">Gemini</option><option value="43">DeepSeek</option></Select></Field><Field label="上游基础地址"><TextInput onChange={(event) => setBaseURL(event.target.value)} placeholder="https://api.example.com/v1" value={baseURL} /></Field><Field label="上游 API 密钥"><TextInput onChange={(event) => setAPIKey(event.target.value)} placeholder="保存后不再回显" type="password" value={apiKey} /></Field><Field label="拟售模型"><textarea className="textarea" onChange={(event) => setModels(event.target.value)} rows="3" value={models} /></Field></>}
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
  if (type === 'payment') return <PaymentModal close={close} notify={notify} />;
  if (type === 'recharge') return <RechargeModal close={close} notify={notify} />;
  if (type === 'merchantApply') return <MerchantApplyModal close={close} config={typeof modal === 'object' ? modal : {}} notify={notify} />;
  if (type === 'withdraw' || type === 'channel') return <SimpleMerchantModal type={type} close={close} config={typeof modal === 'object' ? modal : {}} notify={notify} />;
  return null;
}
