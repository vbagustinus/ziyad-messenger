'use client';

import { useEffect, useState } from 'react';
import { channelsApi, departmentsApi } from '@/services/api';

type Channel = { id: string; name: string; department_id: string; created_at: number; updated_at: number; created_by: string };
type Department = { id: string; name: string };

export default function ChannelsPage() {
  const [channels, setChannels] = useState<Channel[]>([]);
  const [departments, setDepartments] = useState<Department[]>([]);
  const [loading, setLoading] = useState(true);
  const [modal, setModal] = useState(false);
  const [name, setName] = useState('');
  const [deptId, setDeptId] = useState('');

  const load = () => {
    setLoading(true);
    Promise.all([
      channelsApi.list(),
      departmentsApi.list()
    ]).then(([cRes, dRes]) => {
      setChannels(cRes.data.channels || []);
      setDepartments(dRes.data.departments || []);
    }).catch(() => {}).finally(() => setLoading(false));
  };
  useEffect(() => load(), []);

  const create = async (e: React.FormEvent) => {
    e.preventDefault();
    await channelsApi.create({ name, department_id: deptId || undefined });
    setModal(false);
    setName('');
    setDeptId('');
    load();
  };

  const del = async (id: string) => {
    if (!confirm('Delete this channel?')) return;
    await channelsApi.delete(id);
    load();
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-semibold text-slate-800">Channels</h1>
        <button onClick={() => setModal(true)} className="rounded bg-slate-800 px-4 py-2 text-white text-sm font-medium hover:bg-slate-700">Create channel</button>
      </div>
      {loading ? (
        <p className="text-slate-500 italic">Fetching channels...</p>
      ) : (
        <div className="rounded-lg border border-slate-200 bg-white overflow-hidden shadow-sm">
          <table className="w-full text-sm">
            <thead className="bg-slate-50 border-b border-slate-200">
              <tr>
                <th className="text-left p-3 font-medium text-slate-700">Name</th>
                <th className="text-left p-3 font-medium text-slate-700">Department</th>
                <th className="text-left p-3 font-medium text-slate-700">Created</th>
                <th className="p-3"></th>
              </tr>
            </thead>
            <tbody>
              {channels.map((ch) => (
                <tr key={ch.id} className="border-b border-slate-100 hover:bg-slate-50">
                  <td className="p-3 font-medium">{ch.name}</td>
                  <td className="p-3">
                    {departments.find(d => d.id === ch.department_id)?.name || '-'}
                  </td>
                  <td className="p-3 text-slate-500">{new Date(ch.created_at * 1000).toLocaleString()}</td>
                  <td className="p-3 text-right">
                    <button onClick={() => del(ch.id)} className="text-red-600 hover:underline">Delete</button>
                  </td>
                </tr>
              ))}
              {channels.length === 0 && (
                <tr>
                  <td colSpan={4} className="p-8 text-center text-slate-400 italic">No channels found</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      )}
      {modal && (
        <div className="fixed inset-0 bg-black/40 backdrop-blur-sm flex items-center justify-center z-10 p-4">
          <div className="bg-white rounded-xl shadow-xl w-full max-w-sm overflow-hidden animate-in fade-in zoom-in duration-200">
            <div className="p-6 border-b border-slate-100">
              <h2 className="text-lg font-bold">Create Channel</h2>
            </div>
            <form onSubmit={create} className="p-6 space-y-4">
              <div>
                <label className="block text-xs font-bold text-slate-500 uppercase mb-1">Channel Name</label>
                <input type="text" placeholder="e.g. general-it" value={name} onChange={(e) => setName(e.target.value)} className="w-full rounded border border-slate-300 px-3 py-2 outline-none focus:ring-2 focus:ring-slate-800" required />
              </div>
              <div>
                <label className="block text-xs font-bold text-slate-500 uppercase mb-1">Target Department</label>
                <select value={deptId} onChange={(e) => setDeptId(e.target.value)} className="w-full rounded border border-slate-300 px-3 py-2 outline-none focus:ring-2 focus:ring-slate-800 bg-white">
                  <option value="">Public (No Department)</option>
                  {departments.map(d => <option key={d.id} value={d.id}>{d.name}</option>)}
                </select>
              </div>
              <div className="flex gap-2 pt-2">
                <button type="submit" className="flex-1 rounded bg-slate-800 px-4 py-2 text-white text-sm font-bold hover:bg-slate-700">Create</button>
                <button type="button" onClick={() => setModal(false)} className="flex-1 rounded border border-slate-300 px-4 py-2 text-sm text-slate-600 font-bold hover:bg-slate-50">Cancel</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
