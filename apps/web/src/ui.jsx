import {
  AlertTriangle,
  Check,
  CheckCircle2,
  ChevronDown,
  CircleDollarSign,
  Clock3,
  Copy,
  Database,
  ExternalLink,
  LoaderCircle,
  MoreHorizontal,
  ServerCog,
  ShieldCheck,
  X,
  XCircle,
} from 'lucide-react';
import Anthropic from '@lobehub/icons/es/Anthropic/components/Mono';
import DeepSeek from '@lobehub/icons/es/DeepSeek/components/Mono';
import Gemini from '@lobehub/icons/es/Gemini/components/Mono';
import Minimax from '@lobehub/icons/es/Minimax/components/Mono';
import Moonshot from '@lobehub/icons/es/Moonshot/components/Mono';
import OpenAI from '@lobehub/icons/es/OpenAI/components/Mono';
import Qwen from '@lobehub/icons/es/Qwen/components/Mono';

const providerIcons = { OpenAI, Anthropic, Gemini, DeepSeek, Qwen, Moonshot, Minimax };

export function ProviderLogo({ provider, size = 20 }) {
  const Icon = providerIcons[provider] || Database;
  return (
    <span className="provider-logo" aria-hidden="true">
      <Icon size={size} />
    </span>
  );
}

export function Button({ children, icon: Icon, variant = 'secondary', className = '', ...props }) {
  return (
    <button className={`button button-${variant} ${className}`.trim()} type="button" {...props}>
      {Icon && <Icon size={16} strokeWidth={2} />}
      <span>{children}</span>
    </button>
  );
}

export function IconButton({ label, icon: Icon = MoreHorizontal, ...props }) {
  return (
    <button className="icon-button" type="button" aria-label={label} title={label} {...props}>
      <Icon size={18} />
    </button>
  );
}

export function Status({ children }) {
  const text = String(children);
  const tone = ['健康', '成功', '在售', '已完成', '已结算', '已入账', '已开通', '正常'].includes(text)
    ? 'success'
    : ['冷却', '降级', '待审核', '待验证', '待支付', '待打款', '已支付', '等待入账', '入账中'].includes(text)
      ? 'warning'
      : ['入账异常', '结果未知', '验证失败', '执行失败', '已退款', '停用', '已关闭'].includes(text)
        ? 'danger'
        : 'neutral';
  const Icon = tone === 'success' ? CheckCircle2 : tone === 'warning' ? Clock3 : tone === 'danger' ? XCircle : LoaderCircle;
  return (
    <span className={`status status-${tone}`}>
      <Icon size={13} />
      {text}
    </span>
  );
}

export function PageHeader({ eyebrow, title, description, actions }) {
  return (
    <header className="page-header">
      <div>
        {eyebrow && <div className="eyebrow">{eyebrow}</div>}
        <h1>{title}</h1>
        <p>{description}</p>
      </div>
      {actions && <div className="page-actions">{actions}</div>}
    </header>
  );
}

export function Metric({ label, value, hint, icon: Icon = CircleDollarSign, tone = 'teal' }) {
  return (
    <div className="metric">
      <div className={`metric-icon metric-icon-${tone}`}><Icon size={18} /></div>
      <div className="metric-copy">
        <span>{label}</span>
        <strong>{value}</strong>
        {hint && <small>{hint}</small>}
      </div>
    </div>
  );
}

export function Section({ title, description, actions, children, className = '' }) {
  return (
    <section className={`section ${className}`.trim()}>
      {(title || actions) && (
        <div className="section-header">
          <div>
            {title && <h2>{title}</h2>}
            {description && <p>{description}</p>}
          </div>
          {actions && <div className="section-actions">{actions}</div>}
        </div>
      )}
      {children}
    </section>
  );
}

export function Field({ label, hint, children, className = '' }) {
  return (
    <label className={`field ${className}`.trim()}>
      <span>{label}</span>
      {children}
      {hint && <small>{hint}</small>}
    </label>
  );
}

export function TextInput(props) {
  return <input className="input" {...props} />;
}

export function Select({ children, ...props }) {
  return (
    <span className="select-wrap">
      <select className="select" {...props}>{children}</select>
      <ChevronDown size={15} />
    </span>
  );
}

export function Segmented({ value, onChange, options, ariaLabel }) {
  return (
    <div className="segmented" role="group" aria-label={ariaLabel}>
      {options.map((option) => (
        <button
          className={value === option.value ? 'active' : ''}
          key={option.value}
          onClick={() => onChange(option.value)}
          type="button"
        >
          {option.label}
        </button>
      ))}
    </div>
  );
}

export function Modal({ title, description, children, footer, onClose, wide = false }) {
  return (
    <div className="modal-layer" role="presentation" onMouseDown={(event) => event.target === event.currentTarget && onClose()}>
      <div className={`modal ${wide ? 'modal-wide' : ''}`} role="dialog" aria-modal="true" aria-label={title}>
        <div className="modal-header">
          <div>
            <h2>{title}</h2>
            {description && <p>{description}</p>}
          </div>
          <IconButton label="关闭" icon={X} onClick={onClose} />
        </div>
        <div className="modal-body">{children}</div>
        {footer && <div className="modal-footer">{footer}</div>}
      </div>
    </div>
  );
}

export function EmptyNotice({ title, description }) {
  return (
    <div className="empty-notice">
      <ShieldCheck size={24} />
      <strong>{title}</strong>
      <span>{description}</span>
    </div>
  );
}

export function CopyValue({ children, onCopy }) {
  return (
    <span className="copy-value">
      <code>{children}</code>
      <IconButton label="复制" icon={Copy} onClick={onCopy} />
    </span>
  );
}

export function InlineNotice({ tone = 'info', children }) {
  const Icon = tone === 'warning' ? AlertTriangle : tone === 'success' ? Check : ServerCog;
  return <div className={`inline-notice inline-notice-${tone}`}><Icon size={17} />{children}</div>;
}

export function ExternalAnchor({ children, href = '#' }) {
  return <a className="external-anchor" href={href} onClick={(event) => event.preventDefault()}>{children}<ExternalLink size={13} /></a>;
}
