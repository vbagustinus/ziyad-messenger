'use client';

import { useEffect, useState } from 'react';
import { departmentsApi } from '@/services/api';

type Department = { id: string; name: string; created_at: number; updated_at: number };

export default function DepartmentsPage() {
  const [items, setItems] = useState<Department[]>([]);
  const [loading, setLoading] = useState(true);
  const [modal, setModal] = useState(false);
  const [name, setName] = useState('');
  const [error, setError] = useState('');

  const load = () => {
    setLoading(true);
    departmentsApi.list().then((r) => setItems(r.data.departments || [])).catch(() => {}).finally(() => setLoading(false));
  };
  useEffect(() => load(), []);

  const create = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    try {
      await departmentsApi.create({ name });
      setModal(false);
      setName('');
      load();
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to create');
    }
  };

  const del = async (id: string) => {
    if (!confirm('Delete this department?')) return;
    try {
      await departmentsApi.delete(id);
      load();
    } catch (err: any) {
      alert(err.response?.data?.error || 'Failed to delete');
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-semibold text-slate-800">Departments</h1>
        <button onClick={() => setModal(true)} className="rounded-md bg-slate-800 px-4 py-2 text-white text-sm font-medium hover:bg-slate-700 transition-colors">Add Department</button>
      </div>

      {loading ? (
        <p className="text-slate-500 italic">Loading departments...</p>
      ) : (
        <div className="rounded-lg border border-slate-200 bg-white overflow-hidden shadow-sm">
          <table className="w-full text-sm">
            <thead className="bg-slate-50 border-b border-slate-200">
              <tr>
                <th className="text-left p-4 font-medium text-slate-700 uppercase tracking-wider text-xs">Name</th>
                <th className="text-left p-4 font-medium text-slate-700 uppercase tracking-wider text-xs">Created At</th>
                <th className="p-4"></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {items.map((item) => (
                <tr key={item.id} className="hover:bg-slate-50 transition-colors">
                  <td className="p-4 font-medium text-slate-900">{item.name}</td>
                  <td className="p-4 text-slate-500">{new Date(item.created_at * 1000).toLocaleDateString()}</td>
                  <td className="p-4 text-right">
                    <button onClick={() => del(item.id)} className="text-red-500 hover:text-red-700 font-medium ml-4 transition-colors">Delete</button>
                  </td>
                </tr>
              ))}
              {items.length === 0 && (
                <tr>
                  <td colSpan={3} className="p-10 text-center text-slate-400 italic">No departments created yet</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      )}

      {modal && (
        <div className="fixed inset-0 bg-slate-900/40 backdrop-blur-sm flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-xl shadow-2xl w-full max-w-sm overflow-hidden animate-in fade-in zoom-in duration-200">
            <div className="p-5 border-b border-slate-100">
              <h2 className="text-lg font-bold text-slate-800">New Department</h2>
            </div>
            <form onSubmit={create} className="p-5 space-y-4">
              {error && <p className="text-xs text-red-500 bg-red-50 p-2 rounded">{error}</p>}
              <div>
                <label className="block text-xs font-semibold text-slate-500 uppercase mb-1">Department Name</label>
                <input 
                  type="text" 
                  placeholder="e.g. Engineering, Sales" 
                  value={name} 
                  onChange={(e) => setName(e.target.value)} 
                  className="w-full rounded-md border border-slate-300 px-3 py-2 focus:ring-2 focus:ring-slate-800 outline-none transition-all" 
                  required 
                  autoFocus
                />
              </div>
              <div className="flex gap-3 pt-2">
                <button type="submit" className="flex-1 rounded-md bg-slate-800 py-2.5 text-white font-semibold hover:bg-slate-700 transition-all text-sm shadow-md">Create</button>
                <button type="button" onClick={() => setModal(false)} className="flex-1 rounded-md border border-slate-300 py-2.5 text-slate-600 font-semibold hover:bg-slate-50 transition-all text-sm">Cancel</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
