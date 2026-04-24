// Управление категориями и подкатегориями.
import { renderTemplate } from '@/utils/template';
import { catalogApi } from '@/api/catalog';
import { adminApi } from '@/api/admin';
import { ApiError } from '@/api/client';
import { toast } from '@/components/Toast/Toast';
import { modal } from '@/components/Modal/Modal';
import { renderAdminShell, requireAdmin } from './AdminLayout';
import type { CategoryDTO } from '@/api/types';

const tpl = `
<div class="admin__head">
  <h1 class="admin__title">Категории</h1>
  <button id="addCat" class="admin__btn">+ Новая категория</button>
</div>

{{#if loading}}<p style="color:#888">Загрузка...</p>{{else}}
  {{#each categories}}
  <div style="background:#141450; border:1px solid #2a2a5a; border-radius:8px; padding:16px; margin-bottom:12px;">
    <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:12px;">
      <div>
        <strong style="font-size:18px;">{{name}}</strong>
        <span style="color:#a0a0c0; font-size:12px; margin-left:8px;">{{#if (eq type "figure")}}фигурки{{else}}макеты{{/if}}</span>
      </div>
      <div style="display:flex; gap:8px;">
        <button data-act="rename-cat" data-id="{{id}}" data-name="{{name}}" data-type="{{type}}" style="padding:6px 12px; background:#1a1a4a; color:#fff; border:1px solid #2a2a5a; border-radius:6px; cursor:pointer; font-size:13px;">Переименовать</button>
        <button data-act="del-cat" data-id="{{id}}" style="padding:6px 12px; background:#1a1a4a; color:#f44; border:1px solid #2a2a5a; border-radius:6px; cursor:pointer; font-size:13px;">Удалить</button>
        <button data-act="add-sub" data-id="{{id}}" style="padding:6px 12px; background:#e84d2e; color:#fff; border-radius:6px; cursor:pointer; font-size:13px;">+ Подкатегория</button>
      </div>
    </div>
    <div style="display:flex; flex-wrap:wrap; gap:8px;">
      {{#each subcategories}}
      <div class="admin__chip">
        {{name}}
        <button data-act="rename-sub" data-id="{{id}}" data-name="{{name}}" style="cursor:pointer; color:#a0a0c0; font-size:14px;">✎</button>
        <button data-act="del-sub" data-id="{{id}}" style="cursor:pointer; color:#f44; font-size:18px;">×</button>
      </div>
      {{/each}}
      {{#unless subcategories.length}}<span style="color:#888; font-size:13px;">Нет подкатегорий</span>{{/unless}}
    </div>
  </div>
  {{/each}}
  {{#unless categories.length}}<p style="color:#888; text-align:center; padding:40px;">Категорий пока нет.</p>{{/unless}}
{{/if}}
`;

export class AdminCategoriesPage {
  private container: HTMLElement | null = null;
  private categories: CategoryDTO[] = [];

  constructor(private root: HTMLElement) {}

  async render(): Promise<void> {
    if (!requireAdmin()) return;
    this.container = renderAdminShell(this.root, 'categories');
    this.container.innerHTML = renderTemplate(tpl, { loading: true, categories: [] });
    await this.load();
  }

  private async load(): Promise<void> {
    if (!this.container) return;
    try {
      const [f, o] = await Promise.all([
        catalogApi.listCategories('figure', true),
        catalogApi.listCategories('other', true),
      ]);
      this.categories = [...f.categories, ...o.categories];
      this.container.innerHTML = renderTemplate(tpl, { loading: false, categories: this.categories });
      this.bindEvents();
    } catch (e) {
      this.container.innerHTML = renderTemplate(tpl, { loading: false, categories: [] });
    }
  }

  private bindEvents(): void {
    const c = this.container!;
    c.querySelector('#addCat')?.addEventListener('click', () => this.openCreateCat());

    c.querySelectorAll<HTMLButtonElement>('[data-act="rename-cat"]').forEach((btn) => {
      btn.addEventListener('click', () => this.openRenameCat(btn.dataset.id!, btn.dataset.name!, btn.dataset.type as any));
    });
    c.querySelectorAll<HTMLButtonElement>('[data-act="del-cat"]').forEach((btn) => {
      btn.addEventListener('click', async () => {
        if (!confirm('Удалить категорию (и все её подкатегории)?')) return;
        try {
          await adminApi.deleteCategory(btn.dataset.id!);
          toast.success('Удалено');
          this.load();
        } catch (e) {
          if (e instanceof ApiError) toast.error(e.message);
        }
      });
    });
    c.querySelectorAll<HTMLButtonElement>('[data-act="add-sub"]').forEach((btn) => {
      btn.addEventListener('click', () => this.openCreateSub(btn.dataset.id!));
    });
    c.querySelectorAll<HTMLButtonElement>('[data-act="rename-sub"]').forEach((btn) => {
      btn.addEventListener('click', () => this.openRenameSub(btn.dataset.id!, btn.dataset.name!));
    });
    c.querySelectorAll<HTMLButtonElement>('[data-act="del-sub"]').forEach((btn) => {
      btn.addEventListener('click', async () => {
        if (!confirm('Удалить подкатегорию?')) return;
        try {
          await adminApi.deleteSubcategory(btn.dataset.id!);
          toast.success('Удалено');
          this.load();
        } catch (e) {
          if (e instanceof ApiError) toast.error(e.message);
        }
      });
    });
  }

  private openCreateCat(): void {
    modal.open({
      title: 'Новая категория',
      body: `<div class="admin__field"><label>Название</label><input id="x_name" /></div>
             <div class="admin__field"><label>Тип</label><select id="x_type"><option value="figure">Фигурки</option><option value="other">Макеты</option></select></div>`,
      footer: `<button data-cancel style="padding:8px 16px; background:#141450; color:#fff; border:1px solid #2a2a5a; border-radius:6px;">Отмена</button>
               <button data-save class="admin__btn" style="padding:8px 16px;">Создать</button>`,
      onMount: (r) => {
        r.querySelector('[data-cancel]')?.addEventListener('click', () => modal.close());
        r.querySelector('[data-save]')?.addEventListener('click', async () => {
          const name = (r.querySelector<HTMLInputElement>('#x_name')?.value || '').trim();
          const type = (r.querySelector<HTMLSelectElement>('#x_type')?.value || 'figure') as any;
          if (!name) { toast.error('Укажите название'); return; }
          try { await adminApi.createCategory(name, type); modal.close(); toast.success('Создана'); this.load(); }
          catch (e) { if (e instanceof ApiError) toast.error(e.message); }
        });
      },
    });
  }

  private openRenameCat(id: string, name: string, type: 'figure' | 'other'): void {
    modal.open({
      title: 'Изменить категорию',
      body: `<div class="admin__field"><label>Название</label><input id="x_name" value="${name}" /></div>
             <div class="admin__field"><label>Тип</label><select id="x_type">
               <option value="figure" ${type === 'figure' ? 'selected' : ''}>Фигурки</option>
               <option value="other" ${type === 'other' ? 'selected' : ''}>Макеты</option>
             </select></div>`,
      footer: `<button data-cancel style="padding:8px 16px; background:#141450; color:#fff; border:1px solid #2a2a5a; border-radius:6px;">Отмена</button>
               <button data-save class="admin__btn" style="padding:8px 16px;">Сохранить</button>`,
      onMount: (r) => {
        r.querySelector('[data-cancel]')?.addEventListener('click', () => modal.close());
        r.querySelector('[data-save]')?.addEventListener('click', async () => {
          const newName = (r.querySelector<HTMLInputElement>('#x_name')?.value || '').trim();
          const newType = (r.querySelector<HTMLSelectElement>('#x_type')?.value || 'figure') as any;
          try { await adminApi.patchCategory(id, { name: newName, type: newType }); modal.close(); toast.success('Сохранено'); this.load(); }
          catch (e) { if (e instanceof ApiError) toast.error(e.message); }
        });
      },
    });
  }

  private openCreateSub(catId: string): void {
    modal.open({
      title: 'Новая подкатегория',
      body: `<div class="admin__field"><label>Название</label><input id="x_name" /></div>`,
      footer: `<button data-cancel style="padding:8px 16px; background:#141450; color:#fff; border:1px solid #2a2a5a; border-radius:6px;">Отмена</button>
               <button data-save class="admin__btn" style="padding:8px 16px;">Создать</button>`,
      onMount: (r) => {
        r.querySelector('[data-cancel]')?.addEventListener('click', () => modal.close());
        r.querySelector('[data-save]')?.addEventListener('click', async () => {
          const name = (r.querySelector<HTMLInputElement>('#x_name')?.value || '').trim();
          if (!name) return;
          try { await adminApi.createSubcategory(catId, name); modal.close(); toast.success('Создана'); this.load(); }
          catch (e) { if (e instanceof ApiError) toast.error(e.message); }
        });
      },
    });
  }

  private openRenameSub(id: string, name: string): void {
    modal.open({
      title: 'Переименовать подкатегорию',
      body: `<div class="admin__field"><label>Название</label><input id="x_name" value="${name}" /></div>`,
      footer: `<button data-cancel style="padding:8px 16px; background:#141450; color:#fff; border:1px solid #2a2a5a; border-radius:6px;">Отмена</button>
               <button data-save class="admin__btn" style="padding:8px 16px;">Сохранить</button>`,
      onMount: (r) => {
        r.querySelector('[data-cancel]')?.addEventListener('click', () => modal.close());
        r.querySelector('[data-save]')?.addEventListener('click', async () => {
          const newName = (r.querySelector<HTMLInputElement>('#x_name')?.value || '').trim();
          if (!newName) return;
          try { await adminApi.patchSubcategory(id, { name: newName }); modal.close(); toast.success('Сохранено'); this.load(); }
          catch (e) { if (e instanceof ApiError) toast.error(e.message); }
        });
      },
    });
  }
}
