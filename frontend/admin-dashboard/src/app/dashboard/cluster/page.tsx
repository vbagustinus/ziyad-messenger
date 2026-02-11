'use client';

import { useEffect, useState } from 'react';
import { clusterApi } from '@/services/api';

type Node = { node_id: string; address: string; role: string; last_seen: number; is_leader: boolean };

export default function ClusterPage() {
  const [status, setStatus] = useState<{ cluster_id: string; leader_id: string; nodes: Node[]; total_nodes: number; healthy: boolean } | null>(null);

  useEffect(() => {
    clusterApi.status().then((r) => setStatus(r.data)).catch(() => {});
  }, []);

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold text-slate-800">Cluster Status</h1>
      <div className="rounded-lg border border-slate-200 bg-white p-6">
        <dl className="space-y-2 mb-6">
          <div className="flex gap-2">
            <dt className="text-sm text-slate-500 w-28">Cluster ID</dt>
            <dd className="font-mono text-slate-800">{status?.cluster_id ?? '-'}</dd>
          </div>
          <div className="flex gap-2">
            <dt className="text-sm text-slate-500 w-28">Leader</dt>
            <dd className="font-mono text-slate-800">{status?.leader_id ?? '-'}</dd>
          </div>
          <div className="flex gap-2">
            <dt className="text-sm text-slate-500 w-28">Healthy</dt>
            <dd className={status?.healthy ? 'text-green-600' : 'text-red-600'}>{status?.healthy ? 'Yes' : 'No'}</dd>
          </div>
          <div className="flex gap-2">
            <dt className="text-sm text-slate-500 w-28">Total nodes</dt>
            <dd className="text-slate-800">{status?.total_nodes ?? 0}</dd>
          </div>
        </dl>
        <h2 className="text-sm font-medium text-slate-700 mb-2">Nodes</h2>
        <table className="w-full text-sm">
          <thead className="bg-slate-50">
            <tr>
              <th className="text-left p-2 font-medium text-slate-700">Node ID</th>
              <th className="text-left p-2 font-medium text-slate-700">Address</th>
              <th className="text-left p-2 font-medium text-slate-700">Role</th>
              <th className="text-left p-2 font-medium text-slate-700">Leader</th>
            </tr>
          </thead>
          <tbody>
            {(status?.nodes ?? []).map((n) => (
              <tr key={n.node_id} className="border-b border-slate-100">
                <td className="p-2 font-mono">{n.node_id}</td>
                <td className="p-2">{n.address}</td>
                <td className="p-2">{n.role}</td>
                <td className="p-2">{n.is_leader ? 'Yes' : '-'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
