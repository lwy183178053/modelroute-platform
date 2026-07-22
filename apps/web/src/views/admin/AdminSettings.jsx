import { useCallback, useEffect, useState } from 'react';
import { Save } from 'lucide-react';
import { apiRequest } from '../../api';
import { Button, Field, InlineNotice, Section, TextInput } from '../../ui';

const emptySettings = {
  smtp: {
    server: '', port: 465, account: '', from: '', password: '', ssl_enabled: true,
    starttls_enabled: false, insecure_skip_verify: false, force_auth_login: false,
  },
  registration: {
    email_verification_enabled: false, register_enabled: true,
    password_login_enabled: true, password_register_enabled: true,
  },
};

function ToggleRow({ checked, label, onChange }) {
  return <label className="setting-toggle"><span>{label}</span><span className="compact-switch"><input checked={checked} onChange={(event) => onChange(event.target.checked)} type="checkbox" /><span /></span></label>;
}

export default function AdminSettings({ notify }) {
  const [settings, setSettings] = useState(emptySettings);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const data = await apiRequest('/api/admin/platform-settings');
      setSettings({
        smtp: { ...emptySettings.smtp, ...(data?.smtp || {}), password: '' },
        registration: { ...emptySettings.registration, ...(data?.registration || {}) },
      });
    } catch (requestError) {
      setError(requestError.message || '读取平台设置失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const setSMTP = (key, value) => setSettings((current) => ({ ...current, smtp: { ...current.smtp, [key]: value } }));
  const setRegistration = (key, value) => setSettings((current) => ({ ...current, registration: { ...current.registration, [key]: value } }));

  const save = async (event) => {
    event.preventDefault();
    setSaving(true);
    setError('');
    try {
      const data = await apiRequest('/api/admin/platform-settings', {
        method: 'PUT',
        body: { smtp: { ...settings.smtp, port: Number(settings.smtp.port) }, registration: settings.registration },
      });
      setSettings({ smtp: { ...emptySettings.smtp, ...(data?.smtp || {}), password: '' }, registration: { ...emptySettings.registration, ...(data?.registration || {}) } });
      notify('邮件与注册设置已保存');
    } catch (requestError) {
      setError(requestError.message || '保存平台设置失败');
    } finally {
      setSaving(false);
    }
  };

  return (
    <form className="admin-section-stack" onSubmit={save}>
      {error && <InlineNotice tone="warning">{error}</InlineNotice>}
      <Section title="邮件与注册" description="SMTP 密码只写保存，页面不读取或回显">
        {loading ? <div className="table-message">正在读取平台设置</div> : <>
          <div className="settings-form-grid">
            <Field label="SMTP 服务器"><TextInput maxLength="255" onChange={(event) => setSMTP('server', event.target.value)} required value={settings.smtp.server} /></Field>
            <Field label="SMTP 端口"><TextInput max="65535" min="1" onChange={(event) => setSMTP('port', event.target.value)} required type="number" value={settings.smtp.port} /></Field>
            <Field label="SMTP 账号"><TextInput maxLength="255" onChange={(event) => setSMTP('account', event.target.value)} required value={settings.smtp.account} /></Field>
            <Field label="发件人"><TextInput maxLength="255" onChange={(event) => setSMTP('from', event.target.value)} required value={settings.smtp.from} /></Field>
            <Field className="settings-span" hint="留空时保留服务器当前密码" label="SMTP 密码（只写）"><TextInput autoComplete="new-password" onChange={(event) => setSMTP('password', event.target.value)} type="password" value={settings.smtp.password} /></Field>
          </div>
          <div className="settings-toggle-grid">
            <ToggleRow checked={settings.smtp.ssl_enabled} label="启用 SSL" onChange={(value) => setSMTP('ssl_enabled', value)} />
            <ToggleRow checked={settings.smtp.starttls_enabled} label="启用 STARTTLS" onChange={(value) => setSMTP('starttls_enabled', value)} />
            <ToggleRow checked={settings.smtp.force_auth_login} label="强制 AUTH LOGIN" onChange={(value) => setSMTP('force_auth_login', value)} />
            <ToggleRow checked={settings.smtp.insecure_skip_verify} label="跳过证书校验" onChange={(value) => setSMTP('insecure_skip_verify', value)} />
          </div>
        </>}
      </Section>
      <Section title="注册与登录" description="控制 New API 账号注册和邮箱验证入口">
        <div className="settings-toggle-grid">
          <ToggleRow checked={settings.registration.register_enabled} label="开放账号注册" onChange={(value) => setRegistration('register_enabled', value)} />
          <ToggleRow checked={settings.registration.password_register_enabled} label="允许密码注册" onChange={(value) => setRegistration('password_register_enabled', value)} />
          <ToggleRow checked={settings.registration.password_login_enabled} label="允许密码登录" onChange={(value) => setRegistration('password_login_enabled', value)} />
          <ToggleRow checked={settings.registration.email_verification_enabled} label="注册邮箱验证" onChange={(value) => setRegistration('email_verification_enabled', value)} />
        </div>
      </Section>
      <div className="form-footer"><Button disabled={loading || saving} icon={Save} type="submit" variant="primary">{saving ? '正在保存' : '保存邮件与注册'}</Button></div>
    </form>
  );
}
