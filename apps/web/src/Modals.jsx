import { useState } from 'react';
import { Check, CreditCard, Landmark, Plus, Send, ShieldCheck, Store } from 'lucide-react';
import { Button, Field, InlineNotice, Modal, Select, TextInput } from './ui';

function Footer({ cancel, submit, submitText, icon = Check }) {
  return <><Button onClick={cancel}>取消</Button><Button icon={icon} variant="primary" onClick={submit}>{submitText}</Button></>;
}

function RouteModal({ close, notify }) {
  const [mode, setMode] = useState('smart');
  const [priority, setPriority] = useState('stability');
  return (
    <Modal
      title="编辑路由策略"
      description="策略按当前 API 密钥绑定，只影响保存后的新请求。"
      onClose={close}
      wide
      footer={<Footer cancel={close} submitText="保存策略" submit={() => { close(); notify('路由策略已保存'); }} />}
    >
      <div className="form-section"><h3>路由方式</h3><div className="choice-grid">
        <label className={`choice-card ${mode === 'smart' ? 'selected' : ''}`}><input aria-label="智能路由" checked={mode === 'smart'} name="route-mode" onChange={() => setMode('smart')} type="radio" /><span className="choice-icon"><ShieldCheck size={20} /></span><strong>智能路由</strong><small>从所有符合倍率和健康条件的商户中选择</small></label>
        <label className={`choice-card ${mode === 'fixed' ? 'selected' : ''}`}><input aria-label="固定商户" checked={mode === 'fixed'} name="route-mode" onChange={() => setMode('fixed')} type="radio" /><span className="choice-icon"><Store size={20} /></span><strong>固定商户</strong><small>只从指定商户中选择，越界时直接返回无渠道错误</small></label>
      </div></div>

      <div className="form-grid two-column">
        <Field label="最低成交倍率" hint="低于此倍率的报价不参与路由"><TextInput defaultValue="0.40" min="0.0001" max="1" step="0.01" type="number" /></Field>
        <Field label="最高成交倍率" hint="平台强制不超过官方价格 1.0x"><TextInput defaultValue="0.80" min="0.0001" max="1" step="0.01" type="number" /></Field>
      </div>

      <div className="form-section"><h3>选择优先级</h3><div className="radio-row">
        <label><input checked={priority === 'stability'} name="priority" onChange={() => setPriority('stability')} type="radio" />稳定性优先<span>可用率和首字时间优先，再比较价格</span></label>
        <label><input checked={priority === 'price'} name="priority" onChange={() => setPriority('price')} type="radio" />价格优先<span>倍率最低优先，再比较健康和可用率</span></label>
      </div></div>

      {mode === 'fixed' && <div className="form-section"><h3>固定商户范围</h3><div className="merchant-checks">
        <label><input defaultChecked type="checkbox" /><span><strong>北辰云智</strong><small>99.98% 可用率 · 最低 0.72x</small></span></label>
        <label><input defaultChecked type="checkbox" /><span><strong>青川 API</strong><small>99.95% 可用率 · 最低 0.68x</small></span></label>
        <label><input type="checkbox" /><span><strong>矩阵算力</strong><small>99.89% 可用率 · 最低 0.55x</small></span></label>
      </div></div>}
      <InlineNotice tone="warning">没有符合固定商户名单、倍率区间和健康条件的候选时，返回兼容 New API 的 503 错误且不扣费。</InlineNotice>
    </Modal>
  );
}

function QuoteModal({ close, notify }) {
  const [ratio, setRatio] = useState('0.72');
  return (
    <Modal title="调整模型报价" description="GPT-5.2 · OpenAI 主线路" onClose={close} footer={<Footer cancel={close} submitText="提交调价" submit={() => { close(); notify('新报价版本已提交并等待价格同步'); }} />}>
      <div className="quote-summary"><div><span>官方输入价格</span><strong>¥1.75 / M</strong></div><div><span>官方输出价格</span><strong>¥14.00 / M</strong></div><div><span>当前成交倍率</span><strong>0.72x</strong></div></div>
      <Field label="新的成交倍率" hint="允许范围 0.0001x - 1.0000x"><div className="suffix-input"><TextInput max="1" min="0.0001" onChange={(event) => setRatio(event.target.value)} step="0.01" type="number" value={ratio} /><span>x</span></div></Field>
      <div className="price-preview"><span>用户新价格</span><strong>输入 ¥{(1.75 * Number(ratio || 0)).toFixed(2)} / M</strong><strong>输出 ¥{(14 * Number(ratio || 0)).toFixed(2)} / M</strong></div>
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

function MerchantApplyModal({ close, notify }) {
  return (
    <Modal title="申请成为商户" description="普通用户提交网站与渠道凭据，管理员验证通过后在同一账号开通商户中心。" onClose={close} footer={<Footer cancel={close} submitText="提交申请" submit={() => { close(); notify('商户申请已提交'); }} icon={Send} />} wide>
      <div className="form-grid two-column"><Field label="商户名称"><TextInput placeholder="用于市场展示" /></Field><Field label="网站地址"><TextInput placeholder="https://example.com" /></Field><Field label="上游基础地址"><TextInput placeholder="https://api.example.com/v1" /></Field><Field label="上游 API 密钥" hint="提交后加密保存且不再回显"><TextInput placeholder="输入渠道密钥" type="password" /></Field></div>
      <Field label="业务说明"><textarea className="textarea" placeholder="说明渠道来源、可售模型和服务能力" rows="4" /></Field>
    </Modal>
  );
}

function SimpleMerchantModal({ type, close, notify }) {
  const withdrawal = type === 'withdraw';
  return (
    <Modal title={withdrawal ? '申请人工提现' : '提交新渠道'} description={withdrawal ? '申请后冻结对应余额，由管理员审核并线下打款。' : '新渠道验证通过后才会导入 New API。'} onClose={close} footer={<Footer cancel={close} submitText={withdrawal ? '提交提现申请' : '提交渠道审核'} submit={() => { close(); notify(withdrawal ? '提现申请已提交审核' : '渠道已提交审核'); }} icon={withdrawal ? Landmark : Plus} />}>
      {withdrawal ? <><Field label="提现金额"><div className="prefix-input"><span>¥</span><TextInput defaultValue="5000.00" type="number" /></div></Field><Field label="收款账户"><Select><option>支付宝 · 138****2621</option><option>银行卡 · 6217 **** 1098</option></Select></Field><InlineNotice tone="warning">可提现余额 ¥18,642.30，本次金额将在申请后冻结。</InlineNotice></> : <><Field label="渠道名称"><TextInput placeholder="例如 OpenAI 企业线路" /></Field><Field label="上游基础地址"><TextInput placeholder="https://api.example.com/v1" /></Field><Field label="上游 API 密钥"><TextInput placeholder="保存后不再回显" type="password" /></Field></>}
    </Modal>
  );
}

export default function Modals({ modal, close, notify }) {
  if (!modal) return null;
  if (modal === 'route') return <RouteModal close={close} notify={notify} />;
  if (modal === 'quote') return <QuoteModal close={close} notify={notify} />;
  if (modal === 'payment') return <PaymentModal close={close} notify={notify} />;
  if (modal === 'recharge') return <RechargeModal close={close} notify={notify} />;
  if (modal === 'merchantApply') return <MerchantApplyModal close={close} notify={notify} />;
  if (modal === 'withdraw' || modal === 'channel') return <SimpleMerchantModal type={modal} close={close} notify={notify} />;
  return null;
}
