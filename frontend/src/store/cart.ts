// Корзина живёт во фронте (localStorage). См. memory: «корзина — frontend-only».
// Каждый item в корзине — это (item_id + выбранные option_ids).
// Уникальность: одна и та же комбинация увеличивает quantity, разная — отдельная позиция.
// При logout корзина очищается (гость не должен видеть содержимое прошлого юзера).

import type { ItemDetailDTO } from '@/api/types';
import { authStore } from './auth';

export interface CartLine {
  item_id: string;
  // Снапшоты для отображения в корзине без повторного запроса
  name: string;
  articul: string;
  picture_url: string | null;
  unit_base_price: number; // final_price товара (с учётом скидки)
  option_ids: string[];   // отсортированные id опций
  option_labels: string[]; // "Размер: M", "Гравировка: Да" — для UI
  options_extra: number;   // суммарная доплата за опции
  quantity: number;
}

const KEY = 'yulik3d:cart:v1';
type Listener = (lines: CartLine[]) => void;

class CartStore {
  private lines: CartLine[] = [];
  private listeners = new Set<Listener>();
  private prevUserID: string | null = null;

  constructor() {
    this.load();
    window.addEventListener('storage', (e) => {
      if (e.key === KEY) { this.load(); this.emit(); }
    });
    // Очищаем корзину при logout (был user → стал null).
    authStore.subscribe((u) => {
      const newID = u?.id || null;
      if (this.prevUserID && !newID) {
        // Logout — чистим, чтобы новый гость или другой юзер не увидел чужую корзину
        this.clear();
      }
      this.prevUserID = newID;
    });
  }

  private load(): void {
    try {
      const raw = localStorage.getItem(KEY);
      this.lines = raw ? JSON.parse(raw) as CartLine[] : [];
    } catch { this.lines = []; }
  }

  private save(): void {
    localStorage.setItem(KEY, JSON.stringify(this.lines));
    this.emit();
  }

  getLines(): CartLine[] { return [...this.lines]; }

  count(): number { return this.lines.reduce((s, l) => s + l.quantity, 0); }

  total(): number { return this.lines.reduce((s, l) => s + (l.unit_base_price + l.options_extra) * l.quantity, 0); }

  // key — уникален по (item_id + sorted option_ids)
  private keyFor(itemID: string, optionIDs: string[]): string {
    return itemID + '|' + [...optionIDs].sort().join(',');
  }

  addFromDetail(item: ItemDetailDTO, optionIDs: string[], quantity = 1): void {
    const sorted = [...optionIDs].sort();
    const optionLabels: string[] = [];
    let extra = 0;
    item.options.forEach((g) => {
      g.values.forEach((v) => {
        if (sorted.includes(v.id)) {
          optionLabels.push(`${g.type.label}: ${v.value}`);
          extra += v.price;
        }
      });
    });
    const key = this.keyFor(item.id, sorted);
    const existing = this.lines.find((l) => this.keyFor(l.item_id, l.option_ids) === key);
    if (existing) {
      existing.quantity += quantity;
    } else {
      this.lines.push({
        item_id: item.id,
        name: item.name,
        articul: item.articul,
        picture_url: item.pictures[0]?.url || null,
        unit_base_price: item.final_price,
        option_ids: sorted,
        option_labels: optionLabels,
        options_extra: extra,
        quantity,
      });
    }
    this.save();
  }

  setQuantity(index: number, q: number): void {
    if (index < 0 || index >= this.lines.length) return;
    if (q <= 0) {
      this.lines.splice(index, 1);
    } else {
      this.lines[index].quantity = Math.min(q, 99);
    }
    this.save();
  }

  remove(index: number): void {
    if (index < 0 || index >= this.lines.length) return;
    this.lines.splice(index, 1);
    this.save();
  }

  clear(): void {
    this.lines = [];
    this.save();
  }

  subscribe(fn: Listener): () => void {
    this.listeners.add(fn);
    return () => this.listeners.delete(fn);
  }

  private emit(): void {
    this.listeners.forEach((fn) => fn(this.lines));
  }
}

export const cartStore = new CartStore();
