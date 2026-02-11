'use client';

import { useEffect, useState } from 'react';
import { auditApi } from '@/services/api';

type Log = { id: number; timestamp: string; actor_id: string; actor_username: string; action: string; target_resource: string; details: string; ip_address: string };

export default function AuditPage() {
  const [logs, setLogs] = useState<Log[]>([]);
  const [loading, setLoading] = useState(true);
  const [offset, setOffset] = useState(0);
  const limit = 50;

  const load = () => {
    setLoading(true);
    auditApi.list({ offset, limit }).then((r) => setLogs(r.data.logs || [])).catch(() => {}).finally(() => setLoading(false));
  };
  useEffect(() => load(), [offset]);

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold text-slate-800">Audit Logs</h1>
      {loading ? (
        <p className="text-slate-500">Loading...</p>
      ) : (
        <>
          <div className="rounded-lg border border-slate-200 bg-white overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-slate-50 border-b border-slate-200">
                <tr>
                  <th className="text-left p-3 font-medium text-slate-700">Time</th>
                  <th className="text-left p-3 font-medium text-slate-700">Actor</th>
                  <th className="text-left p-3 font-medium text-slate-700">Action</th>
                  <th className="text-left p-3 font-medium text-slate-700">Target</th>
                  <th className="text-left p-3 font-medium text-slate-700">IP</th>
                </tr>
              </thead>
              <tbody>
                {logs.map((e) => (
                  <tr key={e.id} className="border-b border-slate-100">
                    <td className="p-3 text-slate-600">{new Date(e.timestamp).toLocaleString()}</td>
                    <td className="p-3">{e.actor_username || e.actor_id}</td>
                    <td className="p-3">{e.action}</td>
                    <td className="p-3 font-mono text-xs">{e.target_resource}</td>
                    <td className="p-3 text-slate-500">{e.ip_address}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div className="flex gap-2">
            <button onClick={() => setOffset((o) => Math.max(0, o - limit))} disabled={offset === 0} className="rounded border border-slate-300 px-3 py-1 text-sm disabled:opacity-50">Previous</button>
            <button onClick={() => setOffset((o) => o + limit)} className="rounded border border-slate-300 px-3 py-1 text-sm">Next</button>
          </div>
        </>
      )}
    </div>
  );
}
