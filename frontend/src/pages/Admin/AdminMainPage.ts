// Управление товарами на главной странице.
//
// На странице два раздела (Фигурки / Макеты), каждый — список из 5 ячеек.
// В каждом разделе:
//   - Заполненные слоты можно перетаскивать (drag-and-drop) для смены порядка
//   - У каждой карточки есть «×» — открепить
//   - Если меньше 5 — последняя ячейка «+ Добавить» открывает модалку выбора
//
// Бизнес-инварианты на бэке:
//   - Лимит 5 на тип (UNIQUE+CHECK в БД)
//   - Закрепить можно только видимый товар (hidden=false), относящийся к типу
//   - При скрытии товара / смене подкатегорий — закрепления авто-снимаются
import { adminApi } from '@/api/admin';
import { catalogApi } from '@/api/catalog';
import { ApiError } from '@/api/client';
import type { CategoryType, ItemCardDTO } from '@/api/types';
import { renderAdminShell, requireAdmin } from './AdminLayout';
import { toast } from '@/components/Toast/Toast';
import { modal } from '@/components/Modal/Modal';

export class AdminMainPagePage {
  private container: HTMLElement | null = null;
  private figures: ItemCardDTO[] = [];
  private others: ItemCardDTO[] = [];

  constructor(private root: HTMLElement) {}

  async render(): Promise<void> {
    if (!requireAdmin()) return;
    this.container = renderAdminShell(this.root, 'main-page');
    this.container.innerHTML = '<p style="color:#888;text-align:center;padding:60px;">Загрузка...</p>';
    await this.load();
  }

  private async load(): Promise<void> {
    if (!this.container) return;
    try {
      const data = await adminApi.mainPageList();
      this.figures = data.figures || [];
      this.others = data.others || [];
      this.renderShell();
    } catch (e) {
      this.container.innerHTML = '<p style="color:#f44;text-align:center;padding:60px;">Ошибка загрузки.</p>';
      if (e instanceof ApiError) toast.error(e.message);
    }
  }

  private renderShell(): void {
    if (!this.container) return;
    this.container.innerHTML = `
      <h1 class="admin__title">Главная страница</h1>
      <p style="color:#a0a0c0; font-size:14px; margin: -8px 0 24px;">
        Закрепите до 5 фигурок и до 5 макетов на главной. Если оставить меньше — оставшиеся места заполнятся свежими видимыми товарами автоматически.
        Перетаскивайте карточки внутри раздела, чтобы менять порядок.
      </p>
      ${this.renderSection('Фигурки', 'figure', this.figures)}
      ${this.renderSection('Макеты', 'other', this.others)}
    `;
    this.bindSection('figure');
    this.bindSection('other');
  }

  private renderSection(title: string, type: CategoryType, items: ItemCardDTO[]): string {
    const cards = items.map((it, i) => `
      <div class="admin__main-slot" draggable="true"
           data-type="${type}" data-item-id="${it.id}" data-pos="${i + 1}">
        <div class="admin__main-slot-pos">${i + 1}</div>
        ${it.primary_picture_url
          ? `<img src="${it.primary_picture_url}" alt="" />`
          : `<div class="admin__main-slot-noimg"></div>`}
        <div class="admin__main-slot-name">${escapeHtml(it.name)}</div>
        <button class="admin__main-slot-rm" data-act="unpin" data-type="${type}" data-item-id="${it.id}" title="Открепить">×</button>
      </div>
    `).join('');

    const addBtn = items.length < 5
      ? `<button class="admin__main-slot admin__main-slot--add" data-act="add" data-type="${type}">+</button>`
      : '';

    return `
      <section style="margin-bottom:32px;">
        <h2 style="font-size:18px; font-weight:600; margin-bottom:12px;">${title} <span style="color:#a0a0c0; font-size:13px; font-weight:400;">(${items.length}/5)</span></h2>
        <div class="admin__main-grid" data-type-grid="${type}">
          ${cards}
          ${addBtn}
        </div>
      </section>
    `;
  }

  private bindSection(type: CategoryType): void {
    if (!this.container) return;

    // Открепить
    this.container.querySelectorAll<HTMLButtonElement>(`[data-act="unpin"][data-type="${type}"]`).forEach((btn) => {
      btn.addEventListener('click', (e) => {
        e.stopPropagation();
        this.unpin(type, btn.dataset.itemId!);
      });
    });

    // Добавить
    this.container.querySelector<HTMLButtonElement>(`[data-act="add"][data-type="${type}"]`)?.addEventListener('click', () => {
      this.openAddModal(type);
    });

    this.bindDragAndDrop(type);
  }

  private bindDragAndDrop(type: CategoryType): void {
    const grid = this.container?.querySelector<HTMLElement>(`[data-type-grid="${type}"]`);
    if (!grid) return;

    let draggedIdx: number | null = null;

    const clearTargets = () => grid.querySelectorAll('.admin__main-slot--drop-target')
      .forEach((x) => x.classList.remove('admin__main-slot--drop-target'));

    grid.querySelectorAll<HTMLElement>('.admin__main-slot:not(.admin__main-slot--add)').forEach((el) => {
      el.addEventListener('dragstart', (e) => {
        if ((e.target as HTMLElement).tagName === 'BUTTON') {
          e.preventDefault();
          return;
        }
        draggedIdx = parseInt(el.dataset.pos!, 10) - 1;
        el.classList.add('admin__main-slot--dragging');
        e.dataTransfer!.effectAllowed = 'move';
        e.dataTransfer!.setData('text/plain', String(draggedIdx));
      });
      el.addEventListener('dragend', () => {
        el.classList.remove('admin__main-slot--dragging');
        clearTargets();
        draggedIdx = null;
      });
      el.addEventListener('dragover', (e) => {
        e.preventDefault();
        e.dataTransfer!.dropEffect = 'move';
      });
      el.addEventListener('dragenter', (e) => {
        e.preventDefault();
        const targetIdx = parseInt(el.dataset.pos!, 10) - 1;
        if (draggedIdx === null || targetIdx === draggedIdx) return;
        el.classList.add('admin__main-slot--drop-target');
      });
      el.addEventListener('dragleave', (e) => {
        const related = e.relatedTarget as Node | null;
        if (!el.contains(related)) el.classList.remove('admin__main-slot--drop-target');
      });
      el.addEventListener('drop', async (e) => {
        e.preventDefault();
        const targetIdx = parseInt(el.dataset.pos!, 10) - 1;
        clearTargets();
        if (draggedIdx === null || targetIdx === draggedIdx) return;
        const from = draggedIdx;
        draggedIdx = null;
        await this.applyReorder(type, from, targetIdx);
      });
    });
  }

  private async applyReorder(type: CategoryType, fromIdx: number, toIdx: number): Promise<void> {
    const items = type === 'figure' ? this.figures : this.others;
    const next = items.slice();
    const [moved] = next.splice(fromIdx, 1);
    next.splice(toIdx, 0, moved);

    // Оптимистичная перерисовка
    if (type === 'figure') this.figures = next;
    else this.others = next;
    this.renderShell();

    try {
      const order = next.map((it, i) => ({ item_id: it.id, position: i + 1 }));
      await adminApi.mainPageReorder(type, order);
      toast.success('Порядок сохранён');
    } catch (e) {
      if (e instanceof ApiError) toast.error(e.message);
      // Откат через перезагрузку с сервера
      await this.load();
    }
  }

  private async unpin(type: CategoryType, itemID: string): Promise<void> {
    try {
      await adminApi.mainPageUnpin(type, itemID);
      toast.success('Откреплено');
      await this.load();
    } catch (e) {
      if (e instanceof ApiError) toast.error(e.message);
    }
  }

  private openAddModal(type: CategoryType): void {
    const sectionLabel = type === 'figure' ? 'фигурок' : 'макетов';
    modal.open({
      title: `Выбрать товар из раздела ${sectionLabel}`,
      body: `
        <input id="pickerSearch" placeholder="Поиск по названию" style="width:100%; padding:8px 12px; background:#1a1a4a; color:#fff; border:1px solid #2a2a5a; border-radius:6px; margin-bottom:12px;" />
        <div id="pickerList" style="max-height:60vh; overflow-y:auto;">
          <p style="color:#888; text-align:center; padding:20px;">Загрузка...</p>
        </div>
      `,
      footer: `<button data-cancel style="padding:8px 16px; background:#141450; color:#fff; border:1px solid #2a2a5a; border-radius:6px;">Отмена</button>`,
      onMount: (root) => {
        const search = root.querySelector<HTMLInputElement>('#pickerSearch')!;
        const list = root.querySelector<HTMLElement>('#pickerList')!;
        root.querySelector('[data-cancel]')?.addEventListener('click', () => modal.close());

        const pinnedIDs = new Set(
          (type === 'figure' ? this.figures : this.others).map((it) => it.id)
        );

        let timer: number | null = null;
        const reload = async () => {
          const q = search.value.trim();
          list.innerHTML = '<p style="color:#888; text-align:center; padding:20px;">Загрузка...</p>';
          try {
            // Берём только видимые товары нужного типа.
            const res = await catalogApi.listItems({
              category_type: type,
              q: q || undefined,
              limit: 50,
            });
            if (!res.items.length) {
              list.innerHTML = '<p style="color:#888; text-align:center; padding:20px;">Ничего не найдено</p>';
              return;
            }
            list.innerHTML = res.items.map((it) => {
              const isPinned = pinnedIDs.has(it.id);
              return `
                <div data-pick-id="${it.id}" class="admin__pick-row" style="display:flex; gap:12px; align-items:center; padding:8px; border-radius:6px; cursor:${isPinned ? 'not-allowed' : 'pointer'}; ${isPinned ? 'opacity:0.5;' : ''}">
                  <div style="width:48px; height:48px; background:#1a1a4a; border-radius:4px; flex-shrink:0; overflow:hidden;">
                    ${it.primary_picture_url ? `<img src="${it.primary_picture_url}" style="width:100%; height:100%; object-fit:cover;" />` : ''}
                  </div>
                  <div style="flex:1; min-width:0;">
                    <div style="color:#fff; font-size:14px;">${escapeHtml(it.name)}</div>
                    <div style="color:#888; font-size:12px;">${it.articul} · ${it.final_price} ₽${isPinned ? ' · уже на главной' : ''}</div>
                  </div>
                </div>
              `;
            }).join('');
            list.querySelectorAll<HTMLElement>('.admin__pick-row').forEach((row) => {
              const id = row.dataset.pickId!;
              if (pinnedIDs.has(id)) return;
              row.addEventListener('click', async () => {
                try {
                  await adminApi.mainPagePin(id, type);
                  modal.close();
                  toast.success('Закреплено на главной');
                  await this.load();
                } catch (e) {
                  if (e instanceof ApiError) toast.error(e.message);
                }
              });
            });
          } catch (e) {
            list.innerHTML = '<p style="color:#f44; text-align:center; padding:20px;">Ошибка загрузки</p>';
          }
        };

        search.addEventListener('input', () => {
          if (timer) window.clearTimeout(timer);
          timer = window.setTimeout(reload, 300);
        });
        reload();
      },
    });
  }
}

function escapeHtml(s: string): string {
  return s.replace(/[&<>"']/g, (c) =>
    ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]!)
  );
}
