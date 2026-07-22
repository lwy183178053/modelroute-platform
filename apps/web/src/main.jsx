import { useEffect, useState } from 'react';
import { createRoot } from 'react-dom/client';
import {
  Bell,
  ChevronDown,
  CircleHelp,
  Gauge,
  LogIn,
  LogOut,
  Menu,
  Settings,
  Store,
  WalletCards,
  Waypoints,
  X,
} from 'lucide-react';
import { apiRequest, formatQuota } from './api';
import AdminView from './views/AdminView';
import ConsoleView from './views/ConsoleView';
import MarketView from './views/MarketView';
import MerchantView from './views/MerchantView';
import PaymentReturnView from './views/PaymentReturnView';
import Modals from './Modals';
import { Button, IconButton } from './ui';
import './styles.css';

function allowedNavigation(account) {
  const items = [{ id: 'market', label: '模型市场', icon: Waypoints }];
  if (account) items.push({ id: 'console', label: '控制台', icon: Gauge });
  if (account?.is_merchant || Number(account?.role) >= 10) items.push({ id: 'merchant', label: '商户中心', icon: Store });
  if (Number(account?.role) >= 10) items.push({ id: 'admin', label: '管理后台', icon: Settings });
  return items;
}

function Topbar({ account, active, navigate, onLogin, onLogout }) {
  const [mobileOpen, setMobileOpen] = useState(false);
  const switchView = (view) => { navigate(view); setMobileOpen(false); };
  const navigation = allowedNavigation(account);
  const displayName = account?.display_name || account?.username || '';
  const roleName = Number(account?.role) >= 10 ? '管理员' : account?.is_merchant ? '商户' : '普通用户';
  return (
    <header className="topbar">
      <div className="topbar-inner">
        <button className="brand" onClick={() => switchView('market')} type="button" aria-label="返回模型市场">
          <span className="brand-mark"><Waypoints size={23} /></span>
          <span><strong>ModelRoute</strong><small>聚合交易平台</small></span>
        </button>
        <nav className={`main-nav ${mobileOpen ? 'mobile-open' : ''}`} aria-label="主导航">
          {navigation.map(({ id, label, icon: Icon }) => (
            <button className={active === id ? 'active' : ''} key={id} onClick={() => switchView(id)} type="button"><Icon size={16} /><span>{label}</span></button>
          ))}
          {mobileOpen && account && <button onClick={onLogout} type="button"><LogOut size={16} /><span>退出登录</span></button>}
        </nav>
        <div className="topbar-tools">
          {account ? <>
            <div className="balance-chip"><WalletCards size={16} /><span>额度</span><strong>{formatQuota(account.quota)}</strong></div>
            <IconButton label="帮助中心" icon={CircleHelp} />
            <span className="notification-wrap"><IconButton label="通知" icon={Bell} /></span>
            <button className="profile-button" type="button"><span className="avatar">{displayName.slice(0, 1).toUpperCase()}</span><span><strong>{displayName}</strong><small>{roleName}</small></span><ChevronDown size={15} /></button>
            <IconButton label="退出登录" icon={LogOut} onClick={onLogout} />
          </> : account === null ? <Button className="login-button" icon={LogIn} onClick={onLogin} variant="primary">登录</Button> : null}
          <IconButton className="mobile-menu" label={mobileOpen ? '关闭菜单' : '打开菜单'} icon={mobileOpen ? X : Menu} onClick={() => setMobileOpen((value) => !value)} />
        </div>
      </div>
    </header>
  );
}

function App() {
  const [view, setView] = useState(() => window.location.pathname === '/payments/return' ? 'paymentReturn' : 'market');
  const [modal, setModal] = useState(null);
  const [adminSection, setAdminSection] = useState('overview');
  const [toast, setToast] = useState('');
  const [account, setAccount] = useState(undefined);

  useEffect(() => {
    let active = true;
    apiRequest('/api/me')
      .then((data) => { if (active) setAccount(data); })
      .catch(() => { if (active) setAccount(null); });
    const reset = new URLSearchParams(window.location.search);
    if (reset.get('email') && reset.get('token')) {
      setModal({ type: 'resetPassword', email: reset.get('email'), token: reset.get('token') });
    }
    return () => { active = false; };
  }, []);

  useEffect(() => {
    if (!toast) return undefined;
    const timer = window.setTimeout(() => setToast(''), 2600);
    return () => window.clearTimeout(timer);
  }, [toast]);

  const notify = (message) => setToast(message);
  const navigate = (nextView) => {
    if (nextView !== 'market' && !account) {
      setModal('login');
      return;
    }
    if (nextView === 'merchant' && !account?.is_merchant && Number(account?.role) < 10) {
      setView('market');
      return;
    }
    if (nextView === 'admin' && Number(account?.role) < 10) {
      setView('market');
      return;
    }
    if (window.location.pathname === '/payments/return') {
      window.history.replaceState(null, '', '/');
    }
    setView(nextView);
    window.scrollTo({ top: 0, behavior: 'instant' });
  };

  const logout = async () => {
    try {
      await apiRequest('/api/auth/logout', { method: 'POST' });
    } catch (error) {
      notify(error.message || '退出登录失败');
      return;
    }
    setAccount(null);
    setView('market');
    notify('已退出登录');
  };

  return (
    <div className="app-shell">
      <Topbar account={account} active={view} navigate={navigate} onLogin={() => setModal('login')} onLogout={logout} />
      <main>
        {view === 'paymentReturn' && <PaymentReturnView navigate={navigate} />}
        {view === 'market' && <MarketView navigate={navigate} />}
        {view === 'console' && account && <ConsoleView account={account} navigate={navigate} notify={notify} openModal={setModal} />}
        {view === 'merchant' && (account?.is_merchant || Number(account?.role) >= 10) && <MerchantView account={account} notify={notify} openModal={setModal} />}
        {view === 'admin' && Number(account?.role) >= 10 && <AdminView notify={notify} openModal={setModal} section={adminSection} setSection={setAdminSection} />}
      </main>
      <Modals close={() => setModal(null)} modal={modal} notify={notify} onAuthenticated={(user) => {
        setAccount(user);
        setView('console');
        apiRequest('/api/me').then(setAccount).catch(() => {});
      }} setModal={setModal} />
      {toast && <div className="toast" role="status"><span>{toast}</span></div>}
    </div>
  );
}

createRoot(document.getElementById('root')).render(<App />);
