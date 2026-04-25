import { renderTemplate } from '@/utils/template';
import { adminApi } from '@/api/admin';
import { ApiError } from '@/api/client';
import { toast } from '@/components/Toast/Toast';
import { modal } from '@/components/Modal/Modal';
import { renderAdminShell, requireAdmin } from './AdminLayout';
import type { OptionTypeDTO } from '@/api/types';

const tpl = `
<div class="admin__head">
  <h1 class="admin__title">Типы опций</h1>
  <button id="addType" class="admin__btn">+ Новый тип</button>
</div>

{{#if loading}}<p style="color:#888">Загрузка...</p>{{else}}
  {{#each types}}
  <div class="admin__row" style="grid-template-columns: 200px 1fr auto;">
    <div><code style="color:#888;">{{code}}</code></div>
    <div><strong>{{label}}</strong></div>
    <div style="display:flex; gap:8px;">
      <button data-act="rename" data-id="{{id}}" data-label="{{label}}" style="padding:6px 12px; background:#1a1a4a; color:#fff; border:1px solid #2a2a5a; border-radius:6px; cursor:pointer; font-size:13px;">Переименовать</button>
      <button data-act="del" data-id="{{id}}" style="padding:6px 12px; background:#1a1a4a; color:#f44; border:1px solid #2a2a5a; border-radius:6px; cursor:pointer; font-size:13px;">Удалить</button>
    </div>
  </div>
  {{/each}}
  {{#unless types.length}}<p style="color:#888; text-align:center; padding:40px;">Типов опций пока нет.</p>{{/unless}}
{{/if}}
`;

export class AdminOptionTypesPage {
  private container: HTMLElement | null = null;
  constructor(private root: HTMLElement) {}

  async render(): Promise<void> {
    if (!requireAdmin()) return;
    this.container = renderAdminShell(this.root, 'options');
    await this.load();
  }

  private async load(): Promise<void> {
    if (!this.container) return;
    this.container.innerHTML = renderTemplate(tpl, { loading: true, types: [] });
    try {
      const res = await adminApi.listOptionTypes();
      this.container.innerHTML = renderTemplate(tpl, { loading: false, types: res.option_types });
      this.bind(res.option_types);
    } catch (e) {
      this.container.innerHTML = renderTemplate(tpl, { loading: false, types: [] });
    }
  }

  private bind(types: OptionTypeDTO[]): void {
    const c = this.container!;
    c.querySelector('#addType')?.addEventListener('click', () => this.openCreate());
    c.querySelectorAll<HTMLButtonElement>('[data-act="rename"]').forEach((btn) => {
      btn.addEventListener('click', () => this.openRename(btn.dataset.id!, btn.dataset.label!));
    });
    c.querySelectorAll<HTMLButtonElement>('[data-act="del"]').forEach((btn) => {
      btn.addEventListener('click', async () => {
        if (!confirm('Удалить тип опции? (Удаление возможно только если он не используется ни одним товаром)')) return;
        try {
          await adminApi.deleteOptionType(btn.dataset.id!);
          toast.success('Удалено');
          this.load();
        } catch (e) {
          if (e instanceof ApiError) toast.error(e.message);
        }
      });
    });
  }

  private openCreate(): void {
    modal.open({
      title: 'Новый тип опции',
      body: `<div class="admin__field"><label>Код (latin, для логики)</label><input id="x_code" placeholder="напр. wrap" /></div>
             <div class="admin__field"><label>Название (для UI)</label><input id="x_label" placeholder="напр. Подарочная упаковка" /></div>`,
      footer: `<button data-cancel style="padding:8px 16px; background:#141450; color:#fff; border:1px solid #2a2a5a; border-radius:6px;">Отмена</button>
               <button data-save class="admin__btn" style="padding:8px 16px;">Создать</button>`,
      onMount: (r) => {
        r.querySelector('[data-cancel]')?.addEventListener('click', () => modal.close());
        r.querySelector('[data-save]')?.addEventListener('click', async () => {
          const code = (r.querySelector<HTMLInputElement>('#x_code')?.value || '').trim().toLowerCase();
          const label = (r.querySelector<HTMLInputElement>('#x_label')?.value || '').trim();
          if (!code || !label) { toast.error('Заполните оба поля'); return; }
          try { await adminApi.createOptionType(code, label); modal.close(); toast.success('Создан'); this.load(); }
          catch (e) { if (e instanceof ApiError) toast.error(e.message); }
        });
      },
    });
  }

  private openRename(id: string, label: string): void {
    modal.open({
      title: 'Переименовать тип',
      body: `<div class="admin__field"><label>Новое название</label><input id="x_label" value="${label}" /></div>`,
      footer: `<button data-cancel style="padding:8px 16px; background:#141450; color:#fff; border:1px solid #2a2a5a; border-radius:6px;">Отмена</button>
               <button data-save class="admin__btn" style="padding:8px 16px;">Сохранить</button>`,
      onMount: (r) => {
        r.querySelector('[data-cancel]')?.addEventListener('click', () => modal.close());
        r.querySelector('[data-save]')?.addEventListener('click', async () => {
          const newLabel = (r.querySelector<HTMLInputElement>('#x_label')?.value || '').trim();
          if (!newLabel) return;
          try { await adminApi.patchOptionType(id, newLabel); modal.close(); toast.success('Сохранено'); this.load(); }
          catch (e) { if (e instanceof ApiError) toast.error(e.message); }
        });
      },
    });
  }
}
