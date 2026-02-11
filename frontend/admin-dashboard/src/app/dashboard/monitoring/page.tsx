'use client';

import { useEffect, useState } from 'react';
import { monitoringApi } from '@/services/api';
import { LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts';

export default function MonitoringPage() {
  const [network, setNetwork] = useState<{ nodes_online: number; peers_known: number; uptime_seconds: number; latency_ms: number } | null>(null);
  const [users, setUsers] = useState<{ total_users: number; online_now: number; active_today: number } | null>(null);
  const [messages, setMessages] = useState<{ messages_last_hour: number; messages_today: number; total_messages: number } | null>(null);
  const [files, setFiles] = useState<{ total_files: number; total_bytes: number; transfers_today: number } | null>(null);
  const [system, setSystem] = useState<{ go_version: string; num_cpu: number; memory_alloc_mb: number; uptime_seconds: number } | null>(null);

  useEffect(() => {
    monitoringApi.network().then((r) => setNetwork(r.data)).catch(() => {});
    monitoringApi.users().then((r) => setUsers(r.data)).catch(() => {});
    monitoringApi.messages().then((r) => setMessages(r.data)).catch(() => {});
    monitoringApi.files().then((r) => setFiles(r.data)).catch(() => {});
    monitoringApi.system().then((r) => setSystem(r.data)).catch(() => {});
  }, []);

  const chartData = [
    { name: 'Users', total: users?.total_users ?? 0, online: users?.online_now ?? 0 },
    { name: 'Messages', hour: messages?.messages_last_hour ?? 0, day: messages?.messages_today ?? 0 },
    { name: 'Nodes', value: network?.nodes_online ?? 0 },
  ];

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold text-slate-800">Network & Traffic Monitoring</h1>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <Card title="Network nodes" value={network?.nodes_online ?? '-'} sub={network ? `Peers: ${network.peers_known}` : ''} />
        <Card title="Total users" value={users?.total_users ?? '-'} sub={users ? `Online: ${users.online_now}` : ''} />
        <Card title="Messages (1h)" value={messages?.messages_last_hour ?? '-'} sub={messages ? `Today: ${messages.messages_today}` : ''} />
        <Card title="File transfers" value={files?.transfers_today ?? '-'} sub={files ? `Total: ${files.total_files} files` : ''} />
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
        <p className="text-sm text-slate-600">Go: {system?.go_version ?? '-'} · CPU: {system?.num_cpu ?? '-'} · Memory: {system?.memory_alloc_mb?.toFixed(2) ?? '-'} MB · Uptime: {system?.uptime_seconds ?? 0}s</p>
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
