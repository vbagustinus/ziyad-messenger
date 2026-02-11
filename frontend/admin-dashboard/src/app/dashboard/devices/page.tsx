'use client';

import { useEffect, useState } from 'react';
import { devicesApi } from '@/services/api';

type Device = { id: string; user_id: string; device_name: string; fingerprint: string; last_seen: number; created_at: number };

export default function DevicesPage() {
  const [devices, setDevices] = useState<Device[]>([]);
  const [loading, setLoading] = useState(true);

  const load = () => {
    setLoading(true);
    devicesApi.list().then((r) => setDevices(r.data.devices || [])).catch(() => {}).finally(() => setLoading(false));
  };
  useEffect(() => load(), []);

  const del = async (id: string) => {
    if (!confirm('Revoke this device?')) return;
    await devicesApi.delete(id);
    load();
  };

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold text-slate-800">Devices</h1>
      {loading ? (
        <p className="text-slate-500">Loading...</p>
      ) : (
        <div className="rounded-lg border border-slate-200 bg-white overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-slate-50 border-b border-slate-200">
              <tr>
                <th className="text-left p-3 font-medium text-slate-700">Device ID</th>
                <th className="text-left p-3 font-medium text-slate-700">User ID</th>
                <th className="text-left p-3 font-medium text-slate-700">Name</th>
                <th className="text-left p-3 font-medium text-slate-700">Last seen</th>
                <th className="p-3"></th>
              </tr>
            </thead>
            <tbody>
              {devices.map((d) => (
                <tr key={d.id} className="border-b border-slate-100">
                  <td className="p-3 font-mono text-xs">{d.id.slice(0, 8)}...</td>
                  <td className="p-3 font-mono text-xs">{d.user_id.slice(0, 8)}...</td>
                  <td className="p-3">{d.device_name || '-'}</td>
                  <td className="p-3 text-slate-500">{d.last_seen ? new Date(d.last_seen * 1000).toLocaleString() : '-'}</td>
                  <td className="p-3">
                    <button onClick={() => del(d.id)} className="text-red-600 hover:underline">Revoke</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
