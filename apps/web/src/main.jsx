import { useEffect, useState } from 'react';
import { createRoot } from 'react-dom/client';
import {
  Bell,
  ChevronDown,
  CircleHelp,
  Gauge,
  Menu,
  Settings,
  Store,
  WalletCards,
  Waypoints,
  X,
} from 'lucide-react';
import AdminView from './views/AdminView';
import ConsoleView from './views/ConsoleView';
import MarketView from './views/MarketView';
import MerchantView from './views/MerchantView';
import Modals from './Modals';
import { IconButton } from './ui';
import './styles.css';

const navigation = [
  { id: 'market', label: '模型市场', icon: Waypoints },
  { id: 'console', label: '控制台', icon: Gauge },
  { id: 'merchant', label: '商户中心', icon: Store },
  { id: 'admin', label: '管理后台', icon: Settings },
];

function Topbar({ active, navigate }) {
  const [mobileOpen, setMobileOpen] = useState(false);
  const switchView = (view) => { navigate(view); setMobileOpen(false); };
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
        </nav>
        <div className="topbar-tools">
          <div className="balance-chip"><WalletCards size={16} /><span>余额</span><strong>¥1,286.42</strong></div>
          <IconButton label="帮助中心" icon={CircleHelp} />
          <span className="notification-wrap"><IconButton label="通知" icon={Bell} /><b>3</b></span>
          <button className="profile-button" type="button"><span className="avatar">林</span><span><strong>林先生</strong><small>管理员 · 商户</small></span><ChevronDown size={15} /></button>
          <IconButton className="mobile-menu" label={mobileOpen ? '关闭菜单' : '打开菜单'} icon={mobileOpen ? X : Menu} onClick={() => setMobileOpen((value) => !value)} />
        </div>
      </div>
    </header>
  );
}

function App() {
  const [view, setView] = useState('market');
  const [modal, setModal] = useState(null);
  const [adminSection, setAdminSection] = useState('overview');
  const [toast, setToast] = useState('');

  useEffect(() => {
    if (!toast) return undefined;
    const timer = window.setTimeout(() => setToast(''), 2600);
    return () => window.clearTimeout(timer);
  }, [toast]);

  const notify = (message) => setToast(message);
  const navigate = (nextView) => {
    setView(nextView);
    window.scrollTo({ top: 0, behavior: 'instant' });
  };

  return (
    <div className="app-shell">
      <Topbar active={view} navigate={navigate} />
      <main>
        {view === 'market' && <MarketView navigate={navigate} openRoute={() => setModal('route')} />}
        {view === 'console' && <ConsoleView navigate={navigate} notify={notify} openModal={setModal} />}
        {view === 'merchant' && <MerchantView notify={notify} openModal={setModal} />}
        {view === 'admin' && <AdminView notify={notify} openModal={setModal} section={adminSection} setSection={setAdminSection} />}
      </main>
      <Modals close={() => setModal(null)} modal={modal} notify={notify} />
      {toast && <div className="toast" role="status"><span>{toast}</span></div>}
    </div>
  );
}

createRoot(document.getElementById('root')).render(<App />);
