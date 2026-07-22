import { useState } from 'react';
import { KeyRound, LogIn, Mail, Send, UserPlus } from 'lucide-react';
import { apiRequest } from './api';
import { Button, CopyValue, Field, InlineNotice, Modal, TextInput } from './ui';

const definitions = {
  login: { title: '账号登录', description: '使用 ModelRoute 账号进入控制台。' },
  register: { title: '注册账号', description: '邮箱验证通过后创建统一平台账号。' },
  forgot: { title: '找回密码', description: '重置链接将发送到账号绑定邮箱。' },
  twoFactor: { title: '二次验证', description: '输入身份验证器中的动态验证码。' },
  resetPassword: { title: '确认密码重置', description: '确认当前邮箱重置链接。' },
};

export default function AuthModal({ mode, close, notify, setModal, onAuthenticated, reset }) {
  const [form, setForm] = useState({
    username: '', password: '', email: reset?.email || '', verification_code: '', code: '',
  });
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const definition = definitions[mode] || definitions.login;
  const update = (name) => (event) => setForm((value) => ({ ...value, [name]: event.target.value }));

  const finishAuthentication = async () => {
    const account = await apiRequest('/api/me');
    onAuthenticated(account);
    close();
  };

  const run = async (action) => {
    setBusy(true);
    setError('');
    try {
      await action();
    } catch (requestError) {
      setError(requestError.message || '请求处理失败');
    } finally {
      setBusy(false);
    }
  };

  const submit = (event) => {
    event.preventDefault();
    run(async () => {
      if (mode === 'login') {
        const data = await apiRequest('/api/auth/login', {
          method: 'POST', body: { username: form.username, password: form.password },
        });
        if (data?.requires_two_factor) {
          setModal('twoFactor');
          return;
        }
        await finishAuthentication();
        return;
      }
      if (mode === 'twoFactor') {
        await apiRequest('/api/auth/login/2fa', {
          method: 'POST', body: { code: form.code },
        });
        await finishAuthentication();
        return;
      }
      if (mode === 'register') {
        await apiRequest('/api/auth/register', {
          method: 'POST',
          body: {
            username: form.username,
            password: form.password,
            email: form.email,
            verification_code: form.verification_code,
          },
        });
        notify('注册成功，请登录');
        setModal('login');
        return;
      }
      if (mode === 'forgot') {
        await apiRequest('/api/auth/password-reset/request', {
          method: 'POST', body: { email: form.email },
        });
        notify('密码重置邮件已发送');
        close();
        return;
      }
      const password = await apiRequest('/api/auth/password-reset/confirm', {
        method: 'POST', body: { email: reset.email, token: reset.token },
      });
      setNewPassword(String(password || ''));
      notify('密码已重置');
    });
  };

  const sendVerification = () => run(async () => {
    await apiRequest('/api/auth/email-verification', {
      method: 'POST', body: { email: form.email },
    });
    notify('邮箱验证码已发送');
  });

  return (
    <Modal title={definition.title} description={definition.description} onClose={close}>
      <form className="auth-form" onSubmit={submit}>
        {error && <InlineNotice tone="danger">{error}</InlineNotice>}
        {(mode === 'login' || mode === 'register') && (
          <Field label="账号"><TextInput autoComplete="username" onChange={update('username')} required value={form.username} /></Field>
        )}
        {mode === 'register' && (
          <>
            <Field label="邮箱"><TextInput autoComplete="email" onChange={update('email')} required type="email" value={form.email} /></Field>
            <Field label="邮箱验证码">
              <div className="input-action"><TextInput onChange={update('verification_code')} required value={form.verification_code} /><Button disabled={busy || !form.email} icon={Send} onClick={sendVerification}>发送验证码</Button></div>
            </Field>
          </>
        )}
        {(mode === 'login' || mode === 'register') && (
          <Field label="密码"><TextInput autoComplete={mode === 'login' ? 'current-password' : 'new-password'} minLength="8" onChange={update('password')} required type="password" value={form.password} /></Field>
        )}
        {mode === 'forgot' && (
          <Field label="邮箱"><TextInput autoComplete="email" onChange={update('email')} required type="email" value={form.email} /></Field>
        )}
        {mode === 'twoFactor' && (
          <Field label="动态验证码"><TextInput autoComplete="one-time-code" inputMode="numeric" onChange={update('code')} required value={form.code} /></Field>
        )}
        {mode === 'resetPassword' && (newPassword ? <Field label="新密码"><CopyValue onCopy={() => navigator.clipboard.writeText(newPassword)}>{newPassword}</CopyValue></Field> : <InlineNotice>邮箱：{reset.email}</InlineNotice>)}
        {newPassword ? <Button className="full-button" icon={LogIn} onClick={() => setModal('login')} variant="primary">进入登录</Button> : <Button className="full-button" disabled={busy} icon={mode === 'register' ? UserPlus : mode === 'forgot' ? Mail : mode === 'twoFactor' ? KeyRound : LogIn} type="submit" variant="primary">
          {busy ? '处理中' : mode === 'register' ? '注册' : mode === 'forgot' ? '发送重置邮件' : mode === 'twoFactor' ? '完成验证' : mode === 'resetPassword' ? '确认重置' : '登录'}
        </Button>}
        {mode === 'login' && <div className="auth-links"><button onClick={() => setModal('register')} type="button">注册账号</button><button onClick={() => setModal('forgot')} type="button">忘记密码</button></div>}
        {mode !== 'login' && mode !== 'twoFactor' && <div className="auth-links"><button onClick={() => setModal('login')} type="button">返回登录</button></div>}
      </form>
    </Modal>
  );
}
