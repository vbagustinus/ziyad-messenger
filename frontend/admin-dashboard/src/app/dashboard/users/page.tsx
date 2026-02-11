'use client';

import { useEffect, useState } from 'react';
import { usersApi, rolesApi, departmentsApi } from '@/services/api';

type User = { id: string; username: string; full_name: string; role_id: string; department_id: string; created_at: number; updated_at: number };
type Role = { id: string; name: string };
type Department = { id: string; name: string };

export default function UsersPage() {
  const [users, setUsers] = useState<User[]>([]);
  const [roles, setRoles] = useState<Role[]>([]);
  const [departments, setDepartments] = useState<Department[]>([]);
  const [loading, setLoading] = useState(true);
  const [modal, setModal] = useState<'create' | 'edit' | null>(null);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [form, setForm] = useState({ full_name: '', username: '', role_id: '', department_id: '' });

  const load = () => {
    setLoading(true);
    Promise.all([
      usersApi.list(),
      rolesApi.list(),
      departmentsApi.list()
    ]).then(([uRes, rRes, dRes]) => {
      setUsers(uRes.data.users || []);
      setRoles(rRes.data.roles || []);
      setDepartments(dRes.data.departments || []);
    }).catch(() => {}).finally(() => setLoading(false));
  };

  useEffect(() => load(), []);

  const openCreate = () => {
    setForm({ full_name: '', username: '', role_id: '', department_id: '' });
    setModal('create');
  };

  const openEdit = (u: User) => {
    setSelectedUser(u);
    setForm({ 
      full_name: u.full_name, 
      username: u.username, 
      role_id: u.role_id, 
      department_id: u.department_id 
    });
    setModal('edit');
  };

  const save = async (e: React.FormEvent) => {
    e.preventDefault();
    if (modal === 'create') {
      await usersApi.create({ 
        username: form.username, 
        full_name: form.full_name,
        password: '123456789', // Default
        role_id: form.role_id || undefined,
        department_id: form.department_id || undefined
      });
    } else if (modal === 'edit' && selectedUser) {
      await usersApi.update(selectedUser.id, {
        username: form.username,
        full_name: form.full_name,
        role_id: form.role_id || undefined,
        department_id: form.department_id || undefined
      });
    }
    
    setModal(null);
    setForm({ full_name: '', username: '', role_id: '', department_id: '' });
    load();
  };

  const resetPassword = async (id: string) => {
    if (!confirm('Reset password to 123456789?')) return;
    try {
      await usersApi.resetPassword(id);
      alert('Password has been reset to 123456789');
    } catch (err: any) {
      alert(err.response?.data?.error || 'Failed to reset password');
    }
  };

  const del = async (id: string) => {
    if (!confirm('Delete this user?')) return;
    await usersApi.delete(id);
    load();
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-semibold text-slate-800">Users</h1>
        <button onClick={openCreate} className="rounded bg-slate-800 px-4 py-2 text-white text-sm font-medium hover:bg-slate-700 shadow-sm transition-all text-sm">Add User</button>
      </div>
      
      {loading ? (
        <p className="text-slate-500 italic">Exploring the directory...</p>
      ) : (
        <div className="rounded-lg border border-slate-200 bg-white overflow-hidden shadow-sm">
          <table className="w-full text-sm">
            <thead className="bg-slate-50 border-b border-slate-200">
              <tr>
                <th className="text-left p-4 font-semibold text-slate-700 uppercase tracking-wider text-xs">Full Name</th>
                <th className="text-left p-4 font-semibold text-slate-700 uppercase tracking-wider text-xs">Username</th>
                <th className="text-left p-4 font-semibold text-slate-700 uppercase tracking-wider text-xs">Role</th>
                <th className="text-left p-4 font-semibold text-slate-700 uppercase tracking-wider text-xs">Department</th>
                <th className="p-4"></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {users.map((u) => (
                <tr key={u.id} className="hover:bg-slate-50 transition-colors">
                  <td className="p-4 text-slate-900 font-medium">{u.full_name || '-'}</td>
                  <td className="p-4 text-slate-600">{u.username}</td>
                  <td className="p-4">
                    <span className="inline-flex items-center rounded-full bg-blue-50 px-2 py-0.5 text-xs font-semibold text-blue-700 ring-1 ring-inset ring-blue-700/10">
                      {roles.find(r => r.id === u.role_id)?.name || u.role_id || 'User'}
                    </span>
                  </td>
                  <td className="p-4 text-slate-500">
                    {departments.find(d => d.id === u.department_id)?.name || '-'}
                  </td>
                  <td className="p-4 text-right space-x-3">
                    <button onClick={() => openEdit(u)} className="text-slate-600 hover:text-slate-900 font-semibold underline decoration-slate-300">Edit</button>
                    <button onClick={() => resetPassword(u.id)} className="text-orange-600 hover:text-orange-800 font-semibold underline decoration-orange-200">Reset Pwd</button>
                    <button onClick={() => del(u.id)} className="text-red-600 hover:text-red-800 font-semibold">Delete</button>
                  </td>
                </tr>
              ))}
              {users.length === 0 && (
                <tr>
                  <td colSpan={5} className="p-12 text-center text-slate-400 italic">No users available in the platform</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      )}

      {modal && (
        <div className="fixed inset-0 bg-slate-900/40 backdrop-blur-sm flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-xl shadow-2xl w-full max-w-md overflow-hidden animate-in fade-in zoom-in duration-200">
            <div className="p-6 border-b border-slate-100">
              <h2 className="text-xl font-bold text-slate-800">{modal === 'create' ? 'Create New User' : 'Edit User'}</h2>
              {modal === 'create' && <p className="text-xs text-slate-400 mt-1 uppercase tracking-tighter">Password will be <span className="font-mono bg-slate-100 px-1 rounded">123456789</span></p>}
            </div>
            
            <form onSubmit={save} className="p-6 space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="col-span-2">
                  <label className="block text-xs font-bold text-slate-500 uppercase mb-1">Nama Lengkap</label>
                  <input type="text" value={form.full_name} onChange={(e) => setForm((f) => ({ ...f, full_name: e.target.value }))} className="w-full rounded-lg border border-slate-300 px-3 py-2.5 focus:ring-2 focus:ring-slate-800 outline-none" required />
                </div>
                <div>
                  <label className="block text-xs font-bold text-slate-500 uppercase mb-1">Username</label>
                  <input type="text" value={form.username} onChange={(e) => setForm((f) => ({ ...f, username: e.target.value }))} className="w-full rounded-lg border border-slate-300 px-3 py-2.5 focus:ring-2 focus:ring-slate-800 outline-none" required />
                </div>
                <div>
                  <label className="block text-xs font-bold text-slate-500 uppercase mb-1">Role</label>
                  <select value={form.role_id} onChange={(e) => setForm((f) => ({ ...f, role_id: e.target.value }))} className="w-full rounded-lg border border-slate-300 px-3 py-2.5 focus:ring-2 focus:ring-slate-800 outline-none bg-white">
                    <option value="">User (Standard)</option>
                    {roles.map(r => <option key={r.id} value={r.id}>{r.name}</option>)}
                  </select>
                </div>
                <div className="col-span-2">
                  <label className="block text-xs font-bold text-slate-500 uppercase mb-1">Department</label>
                  <select value={form.department_id} onChange={(e) => setForm((f) => ({ ...f, department_id: e.target.value }))} className="w-full rounded-lg border border-slate-300 px-3 py-2.5 focus:ring-2 focus:ring-slate-800 outline-none bg-white">
                    <option value="">None (General)</option>
                    {departments.map(d => <option key={d.id} value={d.id}>{d.name}</option>)}
                  </select>
                </div>
              </div>

              <div className="flex gap-3 pt-6">
                <button type="submit" className="flex-1 rounded-lg bg-slate-800 py-3 text-white font-bold hover:bg-slate-700 transition-all shadow-lg active:scale-95">{modal === 'create' ? 'Create User' : 'Save Changes'}</button>
                <button type="button" onClick={() => setModal(null)} className="flex-1 rounded-lg border border-slate-300 py-3 text-slate-600 font-bold hover:bg-slate-50 transition-all">Cancel</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
