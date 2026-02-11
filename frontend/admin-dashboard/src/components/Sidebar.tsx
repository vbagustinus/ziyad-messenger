'use client';

import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { useAuthStore } from '@/store/authStore';

const nav = [
  { href: '/dashboard', label: 'Dashboard' },
  { href: '/dashboard/users', label: 'Users' },
  { href: '/dashboard/roles', label: 'Roles' },
  { href: '/dashboard/departments', label: 'Departments' },
  { href: '/dashboard/devices', label: 'Devices' },
  { href: '/dashboard/channels', label: 'Channels' },
  { href: '/dashboard/monitoring', label: 'Network Monitoring' },
  { href: '/dashboard/audit', label: 'Audit Logs' },
  { href: '/dashboard/system', label: 'System Health' },
  { href: '/dashboard/cluster', label: 'Cluster Status' },
  { href: '/dashboard/settings', label: 'Settings' },
];

export function Sidebar() {
  const pathname = usePathname();
  const router = useRouter();
  const { user, logout } = useAuthStore();

  const doLogout = () => {
    logout();
    router.push('/login');
    router.refresh();
  };

  return (
    <aside className="w-56 border-r border-slate-200 bg-white min-h-screen flex flex-col">
      <div className="p-4 border-b border-slate-200">
        <p className="font-medium text-slate-800">Admin Panel</p>
        {user && <p className="text-xs text-slate-500">{user.username}</p>}
      </div>
      <nav className="flex-1 p-2 space-y-0.5">
        {nav.map(({ href, label }) => (
          <Link
            key={href}
            href={href}
            className={`block rounded px-3 py-2 text-sm ${pathname === href ? 'bg-slate-100 text-slate-900 font-medium' : 'text-slate-600 hover:bg-slate-50'}`}
          >
            {label}
          </Link>
        ))}
      </nav>
      <div className="p-2 border-t border-slate-200">
        <button onClick={doLogout} className="w-full rounded px-3 py-2 text-sm text-left text-slate-600 hover:bg-slate-50">
          Log out
        </button>
      </div>
    </aside>
  );
}
