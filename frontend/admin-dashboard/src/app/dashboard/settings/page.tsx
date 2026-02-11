'use client';

import { useState } from 'react';
import { useAuthStore } from '@/store/authStore';
import { authApi } from '@/services/api';

export default function SettingsPage() {
  const user = useAuthStore((s) => s.user);
  const [createForm, setCreateForm] = useState({ username: '', password: '', role: 'admin' });
  const [message, setMessage] = useState('');
  const [loading, setLoading] = useState(false);

  const createAdmin = async (e: React.FormEvent) => {
    e.preventDefault();
    setMessage('');
    setLoading(true);
    try {
      await authApi.createAdmin({
        username: createForm.username,
        password: createForm.password,
        role: createForm.role || 'admin',
      });
      setMessage('Admin created. They can log in with that username and password.');
      setCreateForm({ username: '', password: '', role: 'admin' });
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? 'Failed to create admin';
      setMessage(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold text-slate-800">Settings</h1>
      <div className="rounded-lg border border-slate-200 bg-white p-6 max-w-md">
        <h2 className="text-sm font-medium text-slate-700 mb-3">Current admin</h2>
        <p className="text-slate-800">{user?.username ?? '-'}</p>
        <p className="text-xs text-slate-500 mt-1">Role: {user?.role ?? '-'}</p>
      </div>
      <div className="rounded-lg border border-slate-200 bg-white p-6 max-w-md">
        <h2 className="text-sm font-medium text-slate-700 mb-3">Create admin user</h2>
        <p className="text-xs text-slate-500 mb-3">New admins can log in to the dashboard with the credentials you set.</p>
        <form onSubmit={createAdmin} className="space-y-3">
          <input
            type="text"
            placeholder="Username"
            value={createForm.username}
            onChange={(e) => setCreateForm((f) => ({ ...f, username: e.target.value }))}
            className="w-full rounded border border-slate-300 px-3 py-2 text-sm"
            required
          />
          <input
            type="password"
            placeholder="Password"
            value={createForm.password}
            onChange={(e) => setCreateForm((f) => ({ ...f, password: e.target.value }))}
            className="w-full rounded border border-slate-300 px-3 py-2 text-sm"
            required
          />
          <input
            type="text"
            placeholder="Role (default: admin)"
            value={createForm.role}
            onChange={(e) => setCreateForm((f) => ({ ...f, role: e.target.value }))}
            className="w-full rounded border border-slate-300 px-3 py-2 text-sm"
          />
          {message && <p className={`text-sm ${message.startsWith('Admin created') ? 'text-green-600' : 'text-red-600'}`}>{message}</p>}
          <button type="submit" disabled={loading} className="rounded bg-slate-800 px-4 py-2 text-white text-sm font-medium hover:bg-slate-700 disabled:opacity-50">
            {loading ? 'Creating...' : 'Create admin'}
          </button>
        </form>
      </div>
      <div className="rounded-lg border border-slate-200 bg-white p-6 max-w-md">
        <p className="text-sm text-slate-600">API base: {process.env.NEXT_PUBLIC_ADMIN_API ?? 'http://localhost:8090'}</p>
      </div>
    </div>
  );
}
