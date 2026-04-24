import { config } from '@/utils/config';
import type { ErrorResponse } from './types';

export class ApiError extends Error {
  status: number;
  body: ErrorResponse | null;
  constructor(status: number, message: string, body: ErrorResponse | null = null) {
    super(message);
    this.status = status;
    this.body = body;
  }
}

interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
  body?: unknown;
  query?: Record<string, string | number | boolean | undefined | null>;
  formData?: FormData;
  signal?: AbortSignal;
}

function buildQuery(q?: RequestOptions['query']): string {
  if (!q) return '';
  const params = new URLSearchParams();
  Object.entries(q).forEach(([k, v]) => {
    if (v === undefined || v === null || v === '') return;
    params.set(k, String(v));
  });
  const s = params.toString();
  return s ? '?' + s : '';
}

export async function request<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const url = config.apiBaseUrl + path + buildQuery(opts.query);
  const init: RequestInit = {
    method: opts.method || 'GET',
    credentials: 'include', // важно для session cookie
    signal: opts.signal,
  };
  if (opts.formData) {
    init.body = opts.formData;
    // Не ставим Content-Type — браузер сам сделает multipart/form-data; boundary
  } else if (opts.body !== undefined) {
    init.headers = { 'Content-Type': 'application/json' };
    init.body = JSON.stringify(opts.body);
  }

  const res = await fetch(url, init);
  if (res.status === 204) return undefined as T;

  const text = await res.text();
  let parsed: any = null;
  if (text) {
    try { parsed = JSON.parse(text); } catch { /* not JSON */ }
  }

  if (!res.ok) {
    const errBody: ErrorResponse | null = parsed && typeof parsed === 'object' && 'message' in parsed ? parsed : null;
    const msg = errBody?.message || `HTTP ${res.status}`;
    throw new ApiError(res.status, msg, errBody);
  }
  return parsed as T;
}
