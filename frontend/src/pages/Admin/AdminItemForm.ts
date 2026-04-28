// Создание / редактирование товара. Самая сложная страница админки.
import { renderTemplate } from '@/utils/template';
import { adminApi, type ItemCreateRequest, type ItemOptionInput } from '@/api/admin';
import { catalogApi } from '@/api/catalog';
import { ApiError } from '@/api/client';
import { router } from '@/router/router';
import { toast } from '@/components/Toast/Toast';
import { modal } from '@/components/Modal/Modal';
import { renderAdminShell, requireAdmin } from './AdminLayout';
import type { CategoryDTO, ItemDetailDTO, OptionTypeDTO } from '@/api/types';

const tpl = `
<a href="/admin" data-link class="admin__back">← К списку товаров</a>
<h1 class="admin__title">{{title}}</h1>

<div id="itemFormErr" class="admin__error" style="display:none"></div>

<form id="itemForm">
  <div class="admin__field">
    <label>Название*</label>
    <input type="text" name="name" value="{{item.name}}" required maxlength="200" />
  </div>

  <div class="admin__form-grid">
    <div class="admin__field">
      <label>Цена (₽)*</label>
      <input type="number" name="price" value="{{item.price}}" min="0" required />
    </div>
    <div class="admin__field">
      <label>Скидка (%)</label>
      <input type="number" name="sale" value="{{item.sale}}" min="0" max="100" />
    </div>
  </div>

  <div class="admin__field">
    <label>
      <input type="checkbox" name="hidden" {{#if item.hidden}}checked{{/if}} />
      Скрыть из общего каталога
    </label>
  </div>

  <div class="admin__field">
    <label>Информация о товаре (Markdown)*</label>
    <textarea name="description_info" required>{{item.description_info}}</textarea>
    <div class="hint">Например: **Технология:** Ручная работа + 3D-печать. Перенос строки — Enter.</div>
  </div>

  <div class="admin__field">
    <label>Особенности (Markdown — список)*</label>
    <textarea name="description_other" required>{{item.description_other}}</textarea>
    <div class="hint">Маркер списка: дефис в начале строки. Например: - Подсвечивается в темноте</div>
  </div>

  <div class="admin__field">
    <label>Категории/подкатегории*</label>
    <div class="hint">Нужно выбрать хотя бы одну подкатегорию — товар будет показываться в её категории.</div>
    <div id="subcategorySelector"></div>
    <button type="button" id="addCategoryBtn" class="admin__btn--ghost" style="margin-top:8px; padding:6px 12px; background:#141450; color:#fff; border:1px solid #2a2a5a; border-radius:6px;">+ Создать категорию</button>
    <button type="button" id="addSubcategoryBtn" class="admin__btn--ghost" style="margin-top:8px; margin-left:8px; padding:6px 12px; background:#141450; color:#fff; border:1px solid #2a2a5a; border-radius:6px;">+ Создать подкатегорию</button>
  </div>

  <div class="admin__field">
    <label>Опции товара (размер, гравировка и т.д.)</label>
    <div id="optionsContainer"></div>
    <button type="button" id="addOptionBtn" class="admin__btn--ghost" style="padding:6px 12px; background:#141450; color:#fff; border:1px solid #2a2a5a; border-radius:6px;">+ Добавить опцию</button>
    <button type="button" id="addOptionTypeBtn" class="admin__btn--ghost" style="padding:6px 12px; background:#141450; color:#fff; border:1px solid #2a2a5a; border-radius:6px; margin-left:8px;">+ Создать тип опции</button>
  </div>

  {{#if isEdit}}
  <div class="admin__field">
    <label>Картинки</label>
    <div class="admin__pic-hint">Перетаскивайте миниатюры, чтобы изменить порядок. №1 — титульная.</div>
    <div id="picturesGrid" class="admin__pictures"></div>
    <label class="admin__upload">
      📷 Нажмите, чтобы загрузить изображение (png/jpg/webp, до 10МБ)
      <input type="file" id="pictureUpload" accept="image/png,image/jpeg,image/webp" style="display:none" />
    </label>
  </div>
  {{else}}
  <div class="admin__field">
    <p style="color:#888; font-size:14px;">📷 Картинки можно будет загрузить после создания товара.</p>
  </div>
  {{/if}}

  <div style="display:flex; gap:12px; flex-wrap:wrap;">
    <button type="submit" class="admin__btn">{{#if isEdit}}Сохранить{{else}}Создать товар{{/if}}</button>
    <a href="/admin" data-link class="admin__btn--ghost" style="padding:12px 20px; background:#141450; color:#fff; border:1px solid #2a2a5a; border-radius:8px;">Отмена</a>
  </div>
</form>
`;

interface FormSubcategory { id: string; name: string; categoryId: string; categoryName: string; checked: boolean; }
interface OptionRow { typeId: string; value: string; price: number; position: number; }

export class AdminItemFormPage {
  private container: HTMLElement | null = null;
  private item: ItemDetailDTO | null = null;
  private categories: CategoryDTO[] = []; // figure + other (с подкатегориями)
  private optionTypes: OptionTypeDTO[] = [];
  private optionRows: OptionRow[] = [];
  private selectedSubcatIDs: Set<string> = new Set();

  constructor(private root: HTMLElement, private id?: string) {}

  async render(): Promise<void> {
    if (!requireAdmin()) return;
    this.container = renderAdminShell(this.root, 'items');
    this.container.innerHTML = '<p style="color:#888;text-align:center;padding:60px;">Загрузка...</p>';

    try {
      const [figs, others, types] = await Promise.all([
        catalogApi.listCategories('figure', true),
        catalogApi.listCategories('other', true),
        adminApi.listOptionTypes(),
      ]);
      this.categories = [...figs.categories, ...others.categories];
      this.optionTypes = types.option_types;

      if (this.id) {
        this.item = await adminApi.getItem(this.id);
        this.selectedSubcatIDs = new Set(this.item.subcategories.map((s) => s.id));
        this.optionRows = this.item.options.flatMap((g) =>
          g.values.map((v) => ({ typeId: g.type.id, value: v.value, price: v.price, position: v.position }))
        );
      }

      this.renderForm();
    } catch (e) {
      this.container.innerHTML = '<p style="color:#f44;text-align:center;padding:60px;">Ошибка загрузки.</p>';
    }
  }

  private renderForm(): void {
    if (!this.container) return;
    const isEdit = Boolean(this.id);
    this.container.innerHTML = renderTemplate(tpl, {
      title: isEdit ? `Редактирование товара` : 'Новый товар',
      isEdit,
      item: this.item || { name: '', description_info: '', description_other: '', price: 0, sale: 0, hidden: false },
    });
    this.renderSubcategorySelector();
    this.renderOptions();
    if (isEdit) this.renderPictures();
    this.bindEvents();
  }

  private renderSubcategorySelector(): void {
    const root = this.container!.querySelector<HTMLElement>('#subcategorySelector');
    if (!root) return;
    let html = '';
    this.categories.forEach((c) => {
      const subs = c.subcategories || [];
      if (!subs.length) return;
      html += `<div style="margin-bottom:12px;"><div style="font-weight:bold; color:#a0a0c0; font-size:13px; margin-bottom:6px;">${this.escapeHtml(c.name)} <span style="color:#666;font-size:11px;">(${c.type === 'figure' ? 'фигурки' : 'макеты'})</span></div>`;
      subs.forEach((s) => {
        const checked = this.selectedSubcatIDs.has(s.id) ? 'admin__chip--active' : '';
        html += `<span class="admin__chip ${checked}" data-sub-id="${s.id}" style="cursor:pointer;">${this.escapeHtml(s.name)}</span>`;
      });
      html += '</div>';
    });
    if (!html) html = '<p style="color:#888; font-size:14px;">Подкатегорий пока нет — создайте через «Создать подкатегорию».</p>';
    root.innerHTML = html;

    root.querySelectorAll<HTMLElement>('.admin__chip').forEach((chip) => {
      chip.addEventListener('click', () => {
        const id = chip.dataset.subId!;
        if (this.selectedSubcatIDs.has(id)) {
          this.selectedSubcatIDs.delete(id);
          chip.classList.remove('admin__chip--active');
        } else {
          this.selectedSubcatIDs.add(id);
          chip.classList.add('admin__chip--active');
        }
      });
    });
  }

  private renderOptions(): void {
    const root = this.container!.querySelector<HTMLElement>('#optionsContainer');
    if (!root) return;
    if (!this.optionRows.length) {
      root.innerHTML = '<p style="color:#888; font-size:13px; margin-bottom:8px;">Опций пока нет.</p>';
      return;
    }
    let html = '';
    this.optionRows.forEach((r, idx) => {
      const typeOpts = this.optionTypes.map((t) =>
        `<option value="${t.id}" ${t.id === r.typeId ? 'selected' : ''}>${this.escapeHtml(t.label)}</option>`
      ).join('');
      html += `
      <div class="admin__option-form" data-idx="${idx}">
        <select data-field="typeId">${typeOpts}</select>
        <input type="text" data-field="value" value="${this.escapeAttr(r.value)}" placeholder="Значение (например, M)" />
        <input type="number" data-field="price" value="${r.price}" min="0" placeholder="Доплата ₽" />
        <input type="number" data-field="position" value="${r.position}" min="0" placeholder="Позиция" />
        <button type="button" data-act="del-opt" style="background:#f44; color:#fff; padding:6px 10px; border-radius:4px; cursor:pointer;">×</button>
      </div>`;
    });
    root.innerHTML = html;

    root.querySelectorAll<HTMLElement>('.admin__option-form').forEach((row) => {
      const idx = parseInt(row.dataset.idx!, 10);
      row.querySelectorAll<HTMLInputElement | HTMLSelectElement>('[data-field]').forEach((inp) => {
        inp.addEventListener('change', () => {
          const f = inp.dataset.field!;
          const v = (inp as HTMLInputElement).value;
          if (f === 'typeId') this.optionRows[idx].typeId = v;
          else if (f === 'value') this.optionRows[idx].value = v;
          else if (f === 'price') this.optionRows[idx].price = parseInt(v, 10) || 0;
          else if (f === 'position') this.optionRows[idx].position = parseInt(v, 10) || 0;
        });
      });
      row.querySelector('[data-act="del-opt"]')?.addEventListener('click', () => {
        this.optionRows.splice(idx, 1);
        this.renderOptions();
      });
    });
  }

  private renderPictures(): void {
    const root = this.container!.querySelector<HTMLElement>('#picturesGrid');
    if (!root || !this.item) return;
    if (!this.item.pictures.length) {
      root.innerHTML = '<p style="color:#888; font-size:13px; grid-column:1/-1;">Картинок пока нет.</p>';
      return;
    }
    root.innerHTML = this.item.pictures.map((p, i) => `
      <div class="admin__pic" draggable="true" data-pic-id="${p.id}" data-idx="${i}">
        <img src="${p.url}" alt="" draggable="false" />
        <span class="admin__pic-pos">${i + 1}</span>
        <button type="button" data-act="del-pic" title="Удалить" draggable="false">×</button>
      </div>
    `).join('');
    root.querySelectorAll<HTMLButtonElement>('[data-act="del-pic"]').forEach((btn) => {
      btn.addEventListener('click', async (ev) => {
        ev.stopPropagation();
        const wrapEl = btn.closest('.admin__pic') as HTMLElement;
        const picId = wrapEl.dataset.picId!;
        if (!confirm('Удалить картинку?')) return;
        try {
          await adminApi.deletePicture(this.id!, picId);
          this.item = await adminApi.getItem(this.id!);
          this.renderPictures();
          toast.success('Удалено');
        } catch (e) {
          if (e instanceof ApiError) toast.error(e.message);
        }
      });
    });
    this.bindPicturesDragAndDrop(root);
  }

  private bindPicturesDragAndDrop(root: HTMLElement): void {
    let draggedIdx: number | null = null;

    const clearTargets = () => {
      root.querySelectorAll('.admin__pic--drop-target').forEach((x) => x.classList.remove('admin__pic--drop-target'));
    };

    root.querySelectorAll<HTMLElement>('.admin__pic').forEach((el) => {
      el.addEventListener('dragstart', (e) => {
        // Не начинать drag, если потащили за кнопку удаления.
        if ((e.target as HTMLElement).tagName === 'BUTTON') {
          e.preventDefault();
          return;
        }
        draggedIdx = parseInt(el.dataset.idx!, 10);
        el.classList.add('admin__pic--dragging');
        e.dataTransfer!.effectAllowed = 'move';
        e.dataTransfer!.setData('text/plain', String(draggedIdx)); // Firefox требует payload
      });

      el.addEventListener('dragend', () => {
        el.classList.remove('admin__pic--dragging');
        clearTargets();
        draggedIdx = null;
      });

      el.addEventListener('dragover', (e) => {
        e.preventDefault();
        e.dataTransfer!.dropEffect = 'move';
      });

      el.addEventListener('dragenter', (e) => {
        e.preventDefault();
        const targetIdx = parseInt(el.dataset.idx!, 10);
        if (draggedIdx === null || targetIdx === draggedIdx) return;
        el.classList.add('admin__pic--drop-target');
      });

      el.addEventListener('dragleave', (e) => {
        const related = e.relatedTarget as Node | null;
        if (!el.contains(related)) el.classList.remove('admin__pic--drop-target');
      });

      el.addEventListener('drop', async (e) => {
        e.preventDefault();
        const targetIdx = parseInt(el.dataset.idx!, 10);
        clearTargets();
        if (draggedIdx === null || targetIdx === draggedIdx) return;
        const from = draggedIdx;
        draggedIdx = null;
        await this.applyPicturesReorder(from, targetIdx);
      });
    });
  }

  private async applyPicturesReorder(fromIdx: number, toIdx: number): Promise<void> {
    if (!this.item || !this.id) return;
    const pics = this.item.pictures.slice();
    const [moved] = pics.splice(fromIdx, 1);
    pics.splice(toIdx, 0, moved);

    // Оптимистично перерисовываем — UI отзывчивый, не ждём сервер.
    this.item.pictures = pics.map((p, i) => ({ ...p, position: i + 1 }));
    this.renderPictures();

    try {
      const order = pics.map((p, i) => ({ picture_id: p.id, position: i + 1 }));
      const res = await adminApi.reorderPictures(this.id, order);
      this.item.pictures = res.pictures;
      this.renderPictures();
      toast.success('Порядок сохранён');
    } catch (e) {
      if (e instanceof ApiError) toast.error(e.message);
      // Откатываемся к серверному состоянию.
      this.item = await adminApi.getItem(this.id);
      this.renderPictures();
    }
  }

  private bindEvents(): void {
    const c = this.container!;

    // Add option row
    c.querySelector('#addOptionBtn')?.addEventListener('click', () => {
      if (!this.optionTypes.length) {
        toast.error('Сначала создайте хотя бы один тип опции');
        return;
      }
      this.optionRows.push({ typeId: this.optionTypes[0].id, value: '', price: 0, position: this.optionRows.length });
      this.renderOptions();
    });

    // Create option type inline
    c.querySelector('#addOptionTypeBtn')?.addEventListener('click', () => this.openCreateOptionType());

    // Create category / subcategory inline
    c.querySelector('#addCategoryBtn')?.addEventListener('click', () => this.openCreateCategory());
    c.querySelector('#addSubcategoryBtn')?.addEventListener('click', () => this.openCreateSubcategory());

    // Picture upload
    const fileInp = c.querySelector<HTMLInputElement>('#pictureUpload');
    fileInp?.addEventListener('change', async () => {
      const file = fileInp.files?.[0];
      if (!file || !this.id) return;
      try {
        await adminApi.uploadPicture(this.id, file);
        this.item = await adminApi.getItem(this.id);
        this.renderPictures();
        toast.success('Картинка загружена');
      } catch (e) {
        if (e instanceof ApiError) toast.error(e.message);
      } finally {
        fileInp.value = '';
      }
    });

    // Form submit
    const form = c.querySelector<HTMLFormElement>('#itemForm');
    const errEl = c.querySelector<HTMLElement>('#itemFormErr');
    form?.addEventListener('submit', async (e) => {
      e.preventDefault();
      if (errEl) errEl.style.display = 'none';
      const fd = new FormData(form);
      const name = String(fd.get('name') || '').trim();
      const info = String(fd.get('description_info') || '').trim();
      const other = String(fd.get('description_other') || '').trim();
      const priceStr = String(fd.get('price') || '').trim();
      const saleStr = String(fd.get('sale') || '0').trim();

      // Клиентская валидация обязательных полей. Окончательная — на бэке.
      if (!name) { toast.error('Укажите название товара'); return; }
      if (priceStr === '') { toast.error('Укажите цену'); return; }
      const price = parseInt(priceStr, 10);
      if (isNaN(price) || price < 0) { toast.error('Цена должна быть числом ≥ 0'); return; }
      const sale = parseInt(saleStr, 10);
      if (isNaN(sale) || sale < 0 || sale > 100) { toast.error('Скидка должна быть от 0 до 100'); return; }
      if (!info) { toast.error('Заполните «Информация о товаре»'); return; }
      if (!other) { toast.error('Заполните «Особенности»'); return; }
      if (this.selectedSubcatIDs.size === 0) {
        toast.error('Выберите хотя бы одну подкатегорию (вместе с её категорией)');
        return;
      }

      const req: ItemCreateRequest = {
        name,
        description_info: info,
        description_other: other,
        price,
        sale,
        hidden: fd.get('hidden') === 'on',
        subcategory_ids: Array.from(this.selectedSubcatIDs),
        options: this.optionRows.map<ItemOptionInput>((r) => ({
          type_id: r.typeId,
          value: r.value.trim(),
          price: r.price,
          position: r.position,
        })),
      };
      try {
        if (this.id) {
          await adminApi.updateItem(this.id, req);
          toast.success('Товар сохранён');
        } else {
          const created = await adminApi.createItem(req);
          toast.success('Товар создан');
          router.navigate(`/admin/items/${created.id}`);
          return;
        }
      } catch (err) {
        const msg = err instanceof ApiError ? err.message : 'Ошибка';
        if (errEl) { errEl.textContent = msg; errEl.style.display = 'block'; }
      }
    });
  }

  private openCreateOptionType(): void {
    modal.open({
      title: 'Новый тип опции',
      body: `
        <div class="admin__field"><label>Код (slug, латиница)</label><input id="newOTCode" placeholder="напр. wrap" /></div>
        <div class="admin__field"><label>Название (для UI)</label><input id="newOTLabel" placeholder="напр. Подарочная упаковка" /></div>
      `,
      footer: `
        <button class="admin__btn--ghost" data-cancel style="padding:8px 16px; background:#141450; color:#fff; border:1px solid #2a2a5a; border-radius:6px;">Отмена</button>
        <button class="admin__btn" data-save style="padding:8px 16px;">Создать</button>
      `,
      onMount: (root) => {
        root.querySelector('[data-cancel]')?.addEventListener('click', () => modal.close());
        root.querySelector('[data-save]')?.addEventListener('click', async () => {
          const code = (root.querySelector<HTMLInputElement>('#newOTCode')?.value || '').trim().toLowerCase();
          const label = (root.querySelector<HTMLInputElement>('#newOTLabel')?.value || '').trim();
          if (!code || !label) { toast.error('Заполните оба поля'); return; }
          try {
            const created = await adminApi.createOptionType(code, label);
            this.optionTypes.push(created);
            modal.close();
            toast.success('Тип опции создан');
            this.renderOptions();
          } catch (e) {
            if (e instanceof ApiError) toast.error(e.message);
          }
        });
      },
    });
  }

  private openCreateCategory(): void {
    modal.open({
      title: 'Новая категория',
      body: `
        <div class="admin__field"><label>Название</label><input id="newCatName" /></div>
        <div class="admin__field"><label>Тип</label>
          <select id="newCatType"><option value="figure">Фигурки</option><option value="other">Макеты</option></select>
        </div>
      `,
      footer: `
        <button data-cancel style="padding:8px 16px; background:#141450; color:#fff; border:1px solid #2a2a5a; border-radius:6px;">Отмена</button>
        <button data-save class="admin__btn" style="padding:8px 16px;">Создать</button>
      `,
      onMount: (root) => {
        root.querySelector('[data-cancel]')?.addEventListener('click', () => modal.close());
        root.querySelector('[data-save]')?.addEventListener('click', async () => {
          const name = (root.querySelector<HTMLInputElement>('#newCatName')?.value || '').trim();
          const type = (root.querySelector<HTMLSelectElement>('#newCatType')?.value || 'figure') as any;
          if (!name) { toast.error('Укажите название'); return; }
          try {
            const created = await adminApi.createCategory(name, type);
            this.categories.push({ ...created, subcategories: [] });
            modal.close();
            toast.success('Категория создана');
            this.renderSubcategorySelector();
          } catch (e) {
            if (e instanceof ApiError) toast.error(e.message);
          }
        });
      },
    });
  }

  private openCreateSubcategory(): void {
    if (!this.categories.length) { toast.error('Сначала создайте хотя бы одну категорию'); return; }
    const opts = this.categories.map((c) => `<option value="${c.id}">${this.escapeHtml(c.name)} (${c.type === 'figure' ? 'фигурки' : 'макеты'})</option>`).join('');
    modal.open({
      title: 'Новая подкатегория',
      body: `
        <div class="admin__field"><label>Категория</label><select id="newSubCat">${opts}</select></div>
        <div class="admin__field"><label>Название</label><input id="newSubName" /></div>
      `,
      footer: `
        <button data-cancel style="padding:8px 16px; background:#141450; color:#fff; border:1px solid #2a2a5a; border-radius:6px;">Отмена</button>
        <button data-save class="admin__btn" style="padding:8px 16px;">Создать</button>
      `,
      onMount: (root) => {
        root.querySelector('[data-cancel]')?.addEventListener('click', () => modal.close());
        root.querySelector('[data-save]')?.addEventListener('click', async () => {
          const catId = root.querySelector<HTMLSelectElement>('#newSubCat')?.value || '';
          const name = (root.querySelector<HTMLInputElement>('#newSubName')?.value || '').trim();
          if (!name || !catId) { toast.error('Заполните поля'); return; }
          try {
            const created = await adminApi.createSubcategory(catId, name);
            const cat = this.categories.find((c) => c.id === catId);
            if (cat) {
              if (!cat.subcategories) cat.subcategories = [];
              cat.subcategories.push({ id: created.id, name: created.name });
            }
            modal.close();
            toast.success('Подкатегория создана');
            this.renderSubcategorySelector();
          } catch (e) {
            if (e instanceof ApiError) toast.error(e.message);
          }
        });
      },
    });
  }

  private escapeHtml(s: string): string {
    return s.replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]!));
  }
  private escapeAttr(s: string): string { return this.escapeHtml(s); }
}
