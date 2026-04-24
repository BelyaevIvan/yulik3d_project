import { authApi } from '@/api/auth';
import { ApiError } from '@/api/client';
import type { UserDTO } from '@/api/types';

type Listener = (u: UserDTO | null) => void;

class AuthStore {
  private user: UserDTO | null = null;
  private inited = false;
  private listeners = new Set<Listener>();

  async init(): Promise<void> {
    if (this.inited) return;
    try {
      this.user = await authApi.me();
    } catch (e) {
      // 401 — нормально для гостей
      if (!(e instanceof ApiError) || e.status !== 401) {
        console.warn('auth init:', e);
      }
      this.user = null;
    } finally {
      this.inited = true;
      this.emit();
    }
  }

  getUser(): UserDTO | null { return this.user; }
  isAuthed(): boolean { return this.user !== null; }
  isAdmin(): boolean { return this.user?.role === 'admin'; }

  setUser(u: UserDTO | null): void {
    this.user = u;
    this.emit();
  }

  async logout(): Promise<void> {
    try { await authApi.logout(); } catch (e) { /* swallow */ }
    this.setUser(null);
  }

  subscribe(fn: Listener): () => void {
    this.listeners.add(fn);
    return () => this.listeners.delete(fn);
  }

  private emit(): void {
    this.listeners.forEach((fn) => fn(this.user));
  }
}

export const authStore = new AuthStore();
