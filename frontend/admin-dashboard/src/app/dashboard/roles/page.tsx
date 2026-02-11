'use client';

import { useEffect, useState } from 'react';
import { rolesApi } from '@/services/api';

type Role = { id: string; name: string; permissions: string[]; created_at: number; updated_at: number };

export default function RolesPage() {
  const [roles, setRoles] = useState<Role[]>([]);
  const [loading, setLoading] = useState(true);
  const [modal, setModal] = useState<'create' | null>(null);
  const [form, setForm] = useState({ name: '', permissions: '' });

  const load = () => {
    setLoading(true);
    rolesApi.list().then((r) => setRoles(r.data.roles || [])).catch(() => {}).finally(() => setLoading(false));
  };
  useEffect(() => load(), []);

  const create = async (e: React.FormEvent) => {
    e.preventDefault();
    const perms = form.permissions ? form.permissions.split(',').map((p) => p.trim()).filter(Boolean) : [];
    await rolesApi.create({ name: form.name, permissions: perms });
    setModal(null);
    setForm({ name: '', permissions: '' });
    load();
  };

  const del = async (id: string) => {
    if (!confirm('Delete this role?')) return;
    await rolesApi.delete(id);
    load();
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-semibold text-slate-800">Roles & Permissions</h1>
        <button onClick={() => setModal('create')} className="rounded bg-slate-800 px-4 py-2 text-white text-sm font-medium hover:bg-slate-700">Add role</button>
      </div>
      {loading ? (
        <p className="text-slate-500">Loading...</p>
      ) : (
        <div className="rounded-lg border border-slate-200 bg-white overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-slate-50 border-b border-slate-200">
              <tr>
                <th className="text-left p-3 font-medium text-slate-700">Name</th>
                <th className="text-left p-3 font-medium text-slate-700">Permissions</th>
                <th className="p-3"></th>
              </tr>
            </thead>
            <tbody>
              {roles.map((r) => (
                <tr key={r.id} className="border-b border-slate-100">
                  <td className="p-3">{r.name}</td>
                  <td className="p-3 text-slate-600">{Array.isArray(r.permissions) ? r.permissions.join(', ') : '-'}</td>
                  <td className="p-3">
                    <button onClick={() => del(r.id)} className="text-red-600 hover:underline">Delete</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {modal === 'create' && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-10">
          <div className="bg-white rounded-lg p-6 w-full max-w-sm">
            <h2 className="text-lg font-medium mb-4">Create role</h2>
            <form onSubmit={create} className="space-y-3">
              <input type="text" placeholder="Name" value={form.name} onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))} className="w-full rounded border border-slate-300 px-3 py-2" required />
              <input type="text" placeholder="Permissions (comma-separated)" value={form.permissions} onChange={(e) => setForm((f) => ({ ...f, permissions: e.target.value }))} className="w-full rounded border border-slate-300 px-3 py-2" />
              <div className="flex gap-2 pt-2">
                <button type="submit" className="rounded bg-slate-800 px-4 py-2 text-white text-sm">Create</button>
                <button type="button" onClick={() => setModal(null)} className="rounded border border-slate-300 px-4 py-2 text-sm">Cancel</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
