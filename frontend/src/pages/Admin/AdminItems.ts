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
  <input id="filterQ" placeholder="Поиск" value="{{filterQ}}" style="flex:1; min-width:200px; padding:8px 12px; background:#1a1a4a; color:#fff; border:1px solid #2a2a5a; border-radius:6px;" />
  <select id="filterType" style="padding:8px 12px; background:#1a1a4a; color:#fff; border:1px solid #2a2a5a; border-radius:6px;">
    <option value="" {{#if (eq filterType "")}}selected{{/if}}>Любой тип</option>
    <option value="figure" {{#if (eq filterType "figure")}}selected{{/if}}>Фигурки</option>
    <option value="other" {{#if (eq filterType "other")}}selected{{/if}}>Макеты</option>
  </select>
  <select id="filterHidden" style="padding:8px 12px; background:#1a1a4a; color:#fff; border:1px solid #2a2a5a; border-radius:6px;">
    <option value="any" {{#if (eq filterHidden "any")}}selected{{/if}}>Любой статус</option>
    <option value="false" {{#if (eq filterHidden "false")}}selected{{/if}}>Видимые</option>
    <option value="true" {{#if (eq filterHidden "true")}}selected{{/if}}>Скрытые</option>
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

  // State фильтров — единый источник правды. DOM-селекты лишь отображают это.
  // Исправляет баги:
  //   1) шаблон при перерисовке сбрасывал select на «Любой тип», даже когда
  //      загружали отфильтрованный список;
  //   2) попытка вернуться к «Любому» не давала change-события (значение в DOM
  //      уже было пустое после визуального сброса) — запрос не уходил.
  private filterQ = '';
  private filterType: '' | 'figure' | 'other' = '';
  private filterHidden: 'any' | 'true' | 'false' = 'any';

  // Дебаунс-таймер для поискового поля. Хранится на инстансе, чтобы пережить
  // перерисовки и не накапливать таймеры.
  private searchTimer: number | null = null;

  constructor(private root: HTMLElement) {}

  async render(): Promise<void> {
    if (!requireAdmin()) return;
    this.container = renderAdminShell(this.root, 'items');
    this.renderShell(true);
    await this.load();
  }

  // renderShell — перерисовать всю страницу со списком (или плейсхолдером).
  // Вызывается из load() после ответа бэка. Гарантирует, что селекты получают
  // правильный selected-атрибут из текущего state.
  private renderShell(loading: boolean, items: unknown[] = []): void {
    if (!this.container) return;
    this.container.innerHTML = renderTemplate(tpl, {
      loading,
      items,
      filterQ: this.filterQ,
      filterType: this.filterType,
      filterHidden: this.filterHidden,
    });
    this.bindFilters();
    if (!loading) this.bindRowEvents();
  }

  private async load(): Promise<void> {
    if (!this.container) return;
    try {
      const res = await adminApi.listItems({
        q: this.filterQ || undefined,
        category_type: this.filterType ? this.filterType : undefined,
        hidden: this.filterHidden,
        limit: 100,
      });
      this.renderShell(false, res.items);
    } catch {
      this.renderShell(false, []);
    }
  }

  private bindFilters(): void {
    const q = this.container?.querySelector<HTMLInputElement>('#filterQ');
    const t = this.container?.querySelector<HTMLSelectElement>('#filterType');
    const h = this.container?.querySelector<HTMLSelectElement>('#filterHidden');

    q?.addEventListener('input', () => {
      this.filterQ = q.value;
      if (this.searchTimer) window.clearTimeout(this.searchTimer);
      this.searchTimer = window.setTimeout(() => this.load(), 300);
    });
    t?.addEventListener('change', () => {
      this.filterType = t.value as '' | 'figure' | 'other';
      this.load();
    });
    h?.addEventListener('change', () => {
      this.filterHidden = h.value as 'any' | 'true' | 'false';
      this.load();
    });
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
