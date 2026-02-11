'use client';

import { useEffect, useState } from 'react';
import { monitoringApi } from '@/services/api';
import { useWebSocket } from '@/hooks/useWebSocket';
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts';

type NetworkStats = { nodes_online: number; peers_known: number; uptime_seconds: number; latency_ms: number };
type UserStats = { total_users: number; online_now: number; active_today: number };
type MessageStats = { messages_last_hour: number; messages_today: number; total_messages: number };

export default function DashboardPage() {
  const [network, setNetwork] = useState<NetworkStats | null>(null);
  const [users, setUsers] = useState<UserStats | null>(null);
  const [messages, setMessages] = useState<MessageStats | null>(null);
  const [events, setEvents] = useState<Array<{ event: string; at: string }>>([]);
  const { connected } = useWebSocket((msg) => {
    setEvents((prev) => [...prev.slice(-19), { event: msg.event, at: new Date().toLocaleTimeString() }]);
  });

  useEffect(() => {
    monitoringApi.network().then((r) => setNetwork(r.data)).catch(() => {});
    monitoringApi.users().then((r) => setUsers(r.data)).catch(() => {});
    monitoringApi.messages().then((r) => setMessages(r.data)).catch(() => {});
  }, []);

  const chartData = [
    { name: 'Users', value: users?.total_users ?? 0 },
    { name: 'Online', value: users?.online_now ?? 0 },
    { name: 'Nodes', value: network?.nodes_online ?? 0 },
    { name: 'Msg/hour', value: messages?.messages_last_hour ?? 0 },
  ];

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold text-slate-800">Dashboard</h1>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <Card title="Nodes Online" value={network?.nodes_online ?? '-'} />
        <Card title="Total Users" value={users?.total_users ?? '-'} />
        <Card title="Online Now" value={users?.online_now ?? '-'} />
        <Card title="Messages (1h)" value={messages?.messages_last_hour ?? '-'} />
      </div>
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="rounded-lg border border-slate-200 bg-white p-4">
          <h2 className="text-sm font-medium text-slate-700 mb-4">Overview</h2>
          <ResponsiveContainer width="100%" height={200}>
            <BarChart data={chartData}>
              <XAxis dataKey="name" />
              <YAxis />
              <Tooltip />
              <Bar dataKey="value" fill="#334155" />
            </BarChart>
          </ResponsiveContainer>
        </div>
        <div className="rounded-lg border border-slate-200 bg-white p-4">
          <h2 className="text-sm font-medium text-slate-700 mb-2">
            Realtime events {connected ? <span className="text-green-600 text-xs">(connected)</span> : <span className="text-slate-400 text-xs">(disconnected)</span>}
          </h2>
          <div className="h-48 overflow-y-auto space-y-1 text-xs font-mono">
            {events.length === 0 && <p className="text-slate-400">No events yet</p>}
            {events.map((e, i) => (
              <div key={i} className="flex justify-between">
                <span>{e.event}</span>
                <span className="text-slate-500">{e.at}</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

function Card({ title, value }: { title: string; value: string | number }) {
  return (
    <div className="rounded-lg border border-slate-200 bg-white p-4">
      <p className="text-sm text-slate-500">{title}</p>
      <p className="text-2xl font-semibold text-slate-800 mt-1">{value}</p>
    </div>
  );
}
