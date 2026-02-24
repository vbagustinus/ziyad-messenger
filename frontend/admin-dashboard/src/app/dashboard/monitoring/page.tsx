'use client';

import { useEffect, useState } from 'react';
import { monitoringApi } from '@/services/api';
import { LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts';

export default function MonitoringPage() {
  const [overview, setOverview] = useState<{
    generated_at: number;
    network: { nodes_online: number; peers_known: number; uptime_seconds: number; latency_ms: number };
    users: { total_users: number; online_now: number; active_today: number };
    messages: { messages_last_hour: number; messages_today: number; total_messages: number };
    files: { total_files: number; total_bytes: number; transfers_today: number };
    system: { go_version: string; num_cpu: number; memory_alloc_mb: number; uptime_seconds: number };
  } | null>(null);

  useEffect(() => {
    const fetchOverview = () => {
      monitoringApi.overview().then((r) => setOverview(r.data)).catch(() => {});
    };
    fetchOverview();
    const timer = window.setInterval(fetchOverview, 15000);
    return () => window.clearInterval(timer);
  }, []);

  const chartData = [
    { name: 'Users', total: overview?.users.total_users ?? 0, online: overview?.users.online_now ?? 0 },
    { name: 'Messages', hour: overview?.messages.messages_last_hour ?? 0, day: overview?.messages.messages_today ?? 0 },
    { name: 'Nodes', value: overview?.network.nodes_online ?? 0 },
  ];

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold text-slate-800">Network & Traffic Monitoring</h1>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <Card title="Network nodes" value={overview?.network.nodes_online ?? '-'} sub={overview ? `Peers: ${overview.network.peers_known}` : ''} />
        <Card title="Total users" value={overview?.users.total_users ?? '-'} sub={overview ? `Online: ${overview.users.online_now}` : ''} />
        <Card title="Messages (1h)" value={overview?.messages.messages_last_hour ?? '-'} sub={overview ? `Today: ${overview.messages.messages_today}` : ''} />
        <Card title="File transfers" value={overview?.files.transfers_today ?? '-'} sub={overview ? `Total: ${overview.files.total_files} files` : ''} />
      </div>
      <div className="rounded-lg border border-slate-200 bg-white p-4">
        <h2 className="text-sm font-medium text-slate-700 mb-4">Traffic overview</h2>
        <ResponsiveContainer width="100%" height={280}>
          <LineChart data={chartData}>
            <XAxis dataKey="name" />
            <YAxis />
            <Tooltip />
            <Line type="monotone" dataKey="total" stroke="#334155" name="Users" />
            <Line type="monotone" dataKey="online" stroke="#0ea5e9" name="Online" />
            <Line type="monotone" dataKey="hour" stroke="#22c55e" name="Msg/hour" />
          </LineChart>
        </ResponsiveContainer>
      </div>
      <div className="rounded-lg border border-slate-200 bg-white p-4">
        <h2 className="text-sm font-medium text-slate-700 mb-2">System</h2>
        <p className="text-sm text-slate-600">Go: {overview?.system.go_version ?? '-'} · CPU: {overview?.system.num_cpu ?? '-'} · Memory: {overview?.system.memory_alloc_mb?.toFixed(2) ?? '-'} MB · Uptime: {overview?.system.uptime_seconds ?? 0}s</p>
      </div>
    </div>
  );
}

function Card({ title, value, sub }: { title: string; value: string | number; sub?: string }) {
  return (
    <div className="rounded-lg border border-slate-200 bg-white p-4">
      <p className="text-sm text-slate-500">{title}</p>
      <p className="text-2xl font-semibold text-slate-800 mt-1">{value}</p>
      {sub && <p className="text-xs text-slate-400 mt-1">{sub}</p>}
    </div>
  );
}
