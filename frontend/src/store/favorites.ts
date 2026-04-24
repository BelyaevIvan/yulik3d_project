// Кэш избранного (set ID-шников). Загружается с бэка при логине,
// чтобы карточки и страница товара сразу показывали правильное состояние ♥.
import { favoritesApi } from '@/api/favorites';
import { authStore } from './auth';

type Listener = (ids: Set<string>) => void;

class FavoritesStore {
  private ids: Set<string> = new Set();
  private listeners = new Set<Listener>();
  private loaded = false;

  constructor() {
    // При смене юзера сбрасываем и перезагружаем
    authStore.subscribe((u) => {
      if (!u) {
        this.ids.clear();
        this.loaded = false;
        this.emit();
      } else {
        this.reload();
      }
    });
  }

  /** Перечитать с сервера. Вызывается при логине / смене юзера. */
  async reload(): Promise<void> {
    if (!authStore.isAuthed()) {
      this.ids.clear();
      this.loaded = false;
      this.emit();
      return;
    }
    try {
      const res = await favoritesApi.list(100, 0);
      this.ids = new Set(res.items.map((it) => it.id));
      this.loaded = true;
      this.emit();
    } catch (e) {
      console.warn('favorites reload:', e);
    }
  }

  isLoaded(): boolean { return this.loaded; }

  has(itemID: string): boolean { return this.ids.has(itemID); }

  /** Добавить локально (после успешного API). */
  add(itemID: string): void {
    this.ids.add(itemID);
    this.emit();
  }

  /** Удалить локально (после успешного API). */
  remove(itemID: string): void {
    this.ids.delete(itemID);
    this.emit();
  }

  subscribe(fn: Listener): () => void {
    this.listeners.add(fn);
    return () => this.listeners.delete(fn);
  }

  private emit(): void {
    this.listeners.forEach((fn) => fn(this.ids));
  }
}

export const favoritesStore = new FavoritesStore();
