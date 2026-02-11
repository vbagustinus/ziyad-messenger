'use client';

import { useEffect, useState } from 'react';
import { systemApi } from '@/services/api';

export default function SystemPage() {
  const [health, setHealth] = useState<{ status: string; version: string; uptime_seconds: number; memory_alloc_mb: number } | null>(null);

  useEffect(() => {
    systemApi.health().then((r) => setHealth(r.data)).catch(() => {});
  }, []);

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold text-slate-800">System Health</h1>
      <div className="rounded-lg border border-slate-200 bg-white p-6 max-w-md">
        <dl className="space-y-3">
          <div>
            <dt className="text-sm text-slate-500">Status</dt>
            <dd className="font-medium text-slate-800">{health?.status ?? '-'}</dd>
          </div>
          <div>
            <dt className="text-sm text-slate-500">Version</dt>
            <dd className="font-medium text-slate-800">{health?.version ?? '-'}</dd>
          </div>
          <div>
            <dt className="text-sm text-slate-500">Uptime (seconds)</dt>
            <dd className="font-medium text-slate-800">{health?.uptime_seconds ?? '-'}</dd>
          </div>
          <div>
            <dt className="text-sm text-slate-500">Memory (MB)</dt>
            <dd className="font-medium text-slate-800">{health?.memory_alloc_mb?.toFixed(2) ?? '-'}</dd>
          </div>
        </dl>
      </div>
    </div>
  );
}
