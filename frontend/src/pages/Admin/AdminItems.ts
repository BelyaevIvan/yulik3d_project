// Список товаров в админке.
import { renderTemplate } from '@/utils/template';
import { adminApi } from '@/api/admin';
import { router } from '@/router/router';
import { ApiError } from '@/api/client';
import { toast } from '@/components/Toast/Toast';
import { renderAdminShell, requireAdmin } from './AdminLayout';

const tpl = `
<div class="admin__head">
  <h1 class="admin__title">Товары</h1>
  <a href="/admin/items/new" data-link class="admin__btn">+ Новый товар</a>
</div>

<div style="display:flex; gap:12px; margin-bottom:16px; flex-wrap:wrap;">
  <input id="filterQ" placeholder="Поиск" style="flex:1; min-width:200px; padding:8px 12px; background:#1a1a4a; color:#fff; border:1px solid #2a2a5a; border-radius:6px;" />
  <select id="filterType" style="padding:8px 12px; background:#1a1a4a; color:#fff; border:1px solid #2a2a5a; border-radius:6px;">
    <option value="">Любой тип</option>
    <option value="figure">Фигурки</option>
    <option value="other">Макеты</option>
  </select>
  <select id="filterHidden" style="padding:8px 12px; background:#1a1a4a; color:#fff; border:1px solid #2a2a5a; border-radius:6px;">
    <option value="any">Любой статус</option>
    <option value="false">Видимые</option>
    <option value="true">Скрытые</option>
  </select>
</div>

{{#if loading}}<p style="color:#888">Загрузка...</p>{{else}}
  {{#if items.length}}
    <div>
    {{#each items}}
    <div class="admin__row" style="grid-template-columns: 60px 1fr auto auto auto;">
      <div style="width:60px; height:60px; background:#1a1a4a; border-radius:6px; overflow:hidden;">
        {{#if primary_picture_url}}<img src="{{primary_picture_url}}" style="width:100%; height:100%; object-fit:cover;" />{{/if}}
      </div>
      <div>
        <div class="admin__row-name">{{name}} {{#if hidden}}<span style="background:#f44; color:#fff; padding:2px 6px; font-size:11px; border-radius:3px;">Скрыт</span>{{/if}}</div>
        <div class="admin__row-meta">{{articul}} · {{formatPrice final_price}}{{#if (gt sale 0)}} (−{{sale}}%){{/if}}</div>
      </div>
      <button class="admin__btn--ghost admin__btn" data-act="toggle-hidden" data-id="{{id}}" data-hidden="{{hidden}}" style="padding:6px 12px; background:#141450; color:#fff; border:1px solid #2a2a5a; border-radius:6px;">{{#if hidden}}Показать{{else}}Скрыть{{/if}}</button>
      <a href="/admin/items/{{id}}" data-link class="admin__btn--ghost" style="padding:6px 12px; background:#141450; color:#fff; border:1px solid #2a2a5a; border-radius:6px;">Изменить</a>
    </div>
    {{/each}}
    </div>
  {{else}}
    <p style="color:#888; text-align:center; padding:40px;">Товаров пока нет. <a href="/admin/items/new" data-link style="color:#e84d2e;">Создать первый →</a></p>
  {{/if}}
{{/if}}
`;

export class AdminItemsPage {
  private container: HTMLElement | null = null;

  constructor(private root: HTMLElement) {}

  async render(): Promise<void> {
    if (!requireAdmin()) return;
    this.container = renderAdminShell(this.root, 'items');
    this.container.innerHTML = renderTemplate(tpl, { loading: true, items: [] });

    await this.load();
    this.bindFilters();
  }

  private async load(): Promise<void> {
    if (!this.container) return;
    const q = (document.getElementById('filterQ') as HTMLInputElement | null)?.value || '';
    const type = (document.getElementById('filterType') as HTMLSelectElement | null)?.value || '';
    const hidden = (document.getElementById('filterHidden') as HTMLSelectElement | null)?.value as 'any' | 'true' | 'false' || 'any';
    try {
      const res = await adminApi.listItems({
        q: q || undefined,
        category_type: type ? (type as any) : undefined,
        hidden,
        limit: 100,
      });
      this.container.innerHTML = renderTemplate(tpl, { loading: false, items: res.items });
      this.bindRowEvents();
      this.bindFilters();
    } catch (e) {
      this.container.innerHTML = renderTemplate(tpl, { loading: false, items: [] });
    }
  }

  private bindFilters(): void {
    const q = document.getElementById('filterQ') as HTMLInputElement | null;
    const t = document.getElementById('filterType') as HTMLSelectElement | null;
    const h = document.getElementById('filterHidden') as HTMLSelectElement | null;
    let timer: number | null = null;
    q?.addEventListener('input', () => {
      if (timer) clearTimeout(timer);
      timer = window.setTimeout(() => this.load(), 300);
    });
    t?.addEventListener('change', () => this.load());
    h?.addEventListener('change', () => this.load());
  }

  private bindRowEvents(): void {
    this.container?.querySelectorAll<HTMLButtonElement>('[data-act="toggle-hidden"]').forEach((btn) => {
      btn.addEventListener('click', async () => {
        const id = btn.dataset.id!;
        const hidden = btn.dataset.hidden === 'true';
        try {
          await adminApi.patchItem(id, { hidden: !hidden });
          toast.success(hidden ? 'Товар показан' : 'Товар скрыт');
          this.load();
        } catch (e) {
          if (e instanceof ApiError) toast.error(e.message);
        }
      });
    });
  }
}
