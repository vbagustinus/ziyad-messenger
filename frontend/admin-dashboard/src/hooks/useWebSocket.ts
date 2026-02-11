'use client';

import { useEffect, useRef, useState } from 'react';

const getWsUrl = () => {
  const base = process.env.NEXT_PUBLIC_ADMIN_API ?? 'http://localhost:8090';
  const token = typeof window !== 'undefined' ? localStorage.getItem('admin_token') : '';
  const path = '/admin/ws';
  const sep = base.includes('?') ? '&' : '?';
  return base.replace(/^http/, 'ws') + path + (token ? `${sep}token=${encodeURIComponent(token)}` : '');
};

export type WsEvent = 
  | 'USER_CONNECTED' 
  | 'USER_DISCONNECTED' 
  | 'DEVICE_ONLINE' 
  | 'DEVICE_OFFLINE' 
  | 'MESSAGE_FLOW' 
  | 'FILE_TRANSFER' 
  | 'CHANNEL_ACTIVITY' 
  | 'SYSTEM_HEALTH' 
  | 'CLUSTER_STATUS';

export type WsMessage = { event: WsEvent; payload: unknown };

export function useWebSocket(onMessage?: (msg: WsMessage) => void) {
  const [connected, setConnected] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    const token = typeof window !== 'undefined' ? localStorage.getItem('admin_token') : null;
    if (!token) return;
    const url = getWsUrl();
    const ws = new WebSocket(url);
    wsRef.current = ws;

    ws.onopen = () => setConnected(true);
    ws.onclose = () => setConnected(false);
    ws.onerror = () => setConnected(false);
    ws.onmessage = (e) => {
      try {
        const msg = JSON.parse(e.data) as WsMessage;
        onMessage?.(msg);
      } catch {}
    };

    return () => {
      ws.close();
      wsRef.current = null;
      setConnected(false);
    };
  }, [onMessage]);

  return { connected, ws: wsRef.current };
}
