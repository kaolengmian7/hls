/**
 * 网络层基础配置：从环境变量推导 REST / WebSocket 地址。
 *
 * 默认值：
 *   - VITE_API_BASE：未设置时使用 `window.location.origin`
 *   - VITE_WS_BASE ：未设置时由 API_BASE 自动派生（http→ws、https→wss）
 */

function fallbackOrigin(): string {
  if (typeof window === 'undefined') return 'http://localhost:8080';
  return window.location.origin;
}

export function apiBase(): string {
  const env = import.meta.env;
  return (env.VITE_API_BASE ?? fallbackOrigin()).replace(/\/+$/, '');
}

export function wsBase(): string {
  const env = import.meta.env;
  if (env.VITE_WS_BASE) return env.VITE_WS_BASE.replace(/\/+$/, '');
  const http = apiBase();
  if (http.startsWith('https://')) return 'wss://' + http.slice('https://'.length);
  if (http.startsWith('http://')) return 'ws://' + http.slice('http://'.length);
  return http;
}
