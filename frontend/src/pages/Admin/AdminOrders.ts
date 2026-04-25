// Список и детали заказов для админа.
import { renderTemplate } from '@/utils/template';
import { adminApi } from '@/api/admin';
import { ApiError } from '@/api/client';
import { router } from '@/router/router';
import { toast } from '@/components/Toast/Toast';
import { renderAdminShell, requireAdmin } from './AdminLayout';
import type { OrderAdminDetailDTO, OrderStatus } from '@/api/types';
import '@/pages/Orders/Orders.scss';

const listTpl = `
<div class="admin__head">
  <h1 class="admin__title">Заказы</h1>
  <select id="filterStatus" style="padding:8px 12px; background:#1a1a4a; color:#fff; border:1px solid #2a2a5a; border-radius:6px;">
    <option value="">Все статусы</option>
    <option value="created">Создан</option>
    <option value="confirmed">Подтверждён</option>
    <option value="manufacturing">На изготовлении</option>
    <option value="delivering">В доставке</option>
    <option value="completed">Завершён</option>
    <option value="cancelled">Отменён</option>
  </select>
</div>

{{#if loading}}<p style="color:#888">Загрузка...</p>{{else}}
  {{#if items.length}}
    <div class="orders__list">
    {{#each items}}
    <a href="/admin/orders/{{id}}" data-link class="orders__row">
      <div class="orders__row-id">№ {{id}}</div>
      <div class="orders__row-meta">
        <span class="orders__status {{orderStatusClass status}}">{{orderStatusLabel status}}</span>
        <span class="orders__row-date">{{formatDate created_at}}</span>
      </div>
      <div class="orders__row-info">{{user.full_name}} · {{contact_phone}}</div>
      <div class="orders__row-price">{{formatPrice total_price}}</div>
    </a>
    {{/each}}
    </div>
  {{else}}
    <p style="color:#888; text-align:center; padding:40px;">Заказов нет.</p>
  {{/if}}
{{/if}}
`;

const detailTpl = `
<a href="/admin/orders" data-link class="admin__back">← К списку заказов</a>
<div class="admin__head">
  <h1 class="admin__title">Заказ № {{id}}</h1>
  <span class="orders__status orders__status--big {{orderStatusClass status}}">{{orderStatusLabel status}}</span>
</div>

<div class="orders__layout">
  <div class="orders__items">
    <div style="background:#141450; border:1px solid #2a2a5a; padding:16px; border-radius:8px; margin-bottom:16px;">
      <h3 style="margin-bottom:12px;">Управление статусом</h3>
      <div style="display:flex; gap:8px; flex-wrap:wrap;" id="statusBtns"></div>
    </div>

    <h3 style="margin-bottom:8px;">Позиции</h3>
    {{#each items}}
    <div class="orders__item">
      <div class="orders__item-head">
        <strong>{{item_name_snapshot}}</strong>
        <span class="orders__item-art">{{item_articul_snapshot}}</span>
      </div>
      {{#if options.length}}
      <div class="orders__item-opts">
        {{#each options}}<span class="orders__item-opt">{{type_label_snapshot}}: {{value_snapshot}}{{#if (gt price_snapshot 0)}} (+{{formatPrice price_snapshot}}){{/if}}</span>{{/each}}
      </div>
      {{/if}}
      <div class="orders__item-foot">
        <span>{{quantity}} × {{formatPrice unit_total_price}}</span>
        <strong>{{formatPrice (mul quantity unit_total_price)}}</strong>
      </div>
    </div>
    {{/each}}
  </div>

  <aside class="orders__side">
    <h3>Покупатель</h3>
    <p><strong>{{user.full_name}}</strong></p>
    <p>{{user.email}}</p>
    {{#if user.phone}}<p>{{user.phone}}</p>{{/if}}

    <h3>Контакты заказа</h3>
    <p>{{contact_full_name}}</p>
    <p>{{contact_phone}}</p>

    {{#if customer_comment}}
    <h3>Комментарий покупателя</h3>
    <p style="white-space:pre-wrap;">{{customer_comment}}</p>
    {{/if}}

    <h3>Внутренняя пометка</h3>
    <textarea id="adminNote" rows="4" style="width:100%; padding:8px; background:#1a1a4a; color:#fff; border:1px solid #2a2a5a; border-radius:6px;">{{admin_note}}</textarea>
    <button id="saveNote" class="admin__btn" style="margin-top:8px; width:100%;">Сохранить пометку</button>

    <div class="orders__total">
      <span>Итого:</span>
      <strong>{{formatPrice total_price}}</strong>
    </div>
  </aside>
</div>
`;

export class AdminOrdersPage {
  private container: HTMLElement | null = null;
  constructor(private root: HTMLElement) {}

  async render(): Promise<void> {
    if (!requireAdmin()) return;
    this.container = renderAdminShell(this.root, 'orders');
    await this.load();
  }

  private async load(): Promise<void> {
    if (!this.container) return;
    const status = (document.getElementById('filterStatus') as HTMLSelectElement | null)?.value || '';
    this.container.innerHTML = renderTemplate(listTpl, { loading: true, items: [] });
    try {
      const res = await adminApi.listOrders({
        status: status ? (status as any) : undefined,
        limit: 100,
      });
      this.container.innerHTML = renderTemplate(listTpl, { loading: false, items: res.items });
      const sel = this.container.querySelector<HTMLSelectElement>('#filterStatus');
      if (sel) sel.value = status;
      sel?.addEventListener('change', () => this.load());
    } catch (e) {
      this.container.innerHTML = renderTemplate(listTpl, { loading: false, items: [] });
    }
  }
}

const transitions: Record<OrderStatus, OrderStatus[]> = {
  created: ['confirmed', 'cancelled'],
  confirmed: ['manufacturing', 'cancelled'],
  manufacturing: ['delivering', 'cancelled'],
  delivering: ['completed', 'cancelled'],
  completed: [],
  cancelled: [],
};

const statusLabel: Record<OrderStatus, string> = {
  created: 'Подтвердить', // букальные действия
  confirmed: 'На изготовление',
  manufacturing: 'В доставку',
  delivering: 'Завершить',
  completed: '',
  cancelled: '',
};

export class AdminOrderDetailPage {
  private container: HTMLElement | null = null;
  constructor(private root: HTMLElement, private id: string) {}

  async render(): Promise<void> {
    if (!requireAdmin()) return;
    this.container = renderAdminShell(this.root, 'orders');
    this.container.innerHTML = '<p style="color:#888;text-align:center;padding:60px;">Загрузка...</p>';
    try {
      const o = await adminApi.getOrder(this.id);
      this.draw(o);
    } catch (e) {
      if (e instanceof ApiError && e.status === 404) {
        router.replace('/404');
        return;
      }
      this.container.innerHTML = '<p style="color:#f44;text-align:center;padding:60px;">Ошибка загрузки.</p>';
    }
  }

  private draw(o: OrderAdminDetailDTO): void {
    if (!this.container) return;
    this.container.innerHTML = renderTemplate(detailTpl, {
      ...o,
      admin_note: o.admin_note || '',
    });
    this.bindButtons(o);
  }

  private bindButtons(o: OrderAdminDetailDTO): void {
    const c = this.container!;
    // Status buttons
    const btns = c.querySelector<HTMLElement>('#statusBtns');
    if (btns) {
      const allowed = transitions[o.status];
      if (!allowed.length) {
        btns.innerHTML = '<span style="color:#888; font-size:14px;">Финальный статус — переходов нет.</span>';
      } else {
        btns.innerHTML = allowed.map((s) => {
          const isCancel = s === 'cancelled';
          const label = isCancel ? 'Отменить' : statusLabel[o.status];
          return `<button data-status="${s}" style="padding:8px 16px; background:${isCancel ? '#f44' : '#e84d2e'}; color:#fff; border-radius:6px; cursor:pointer; font-weight:bold;">${label}</button>`;
        }).join('');
        btns.querySelectorAll<HTMLButtonElement>('[data-status]').forEach((btn) => {
          btn.addEventListener('click', async () => {
            const s = btn.dataset.status as OrderStatus;
            if (s === 'cancelled' && !confirm('Точно отменить заказ?')) return;
            try {
              const upd = await adminApi.patchOrderStatus(this.id, s);
              toast.success('Статус обновлён');
              this.draw(upd);
            } catch (e) {
              if (e instanceof ApiError) toast.error(e.message);
            }
          });
        });
      }
    }

    // Note
    c.querySelector('#saveNote')?.addEventListener('click', async () => {
      const note = (c.querySelector<HTMLTextAreaElement>('#adminNote')?.value || '').trim();
      try {
        await adminApi.patchOrderNote(this.id, note || null);
        toast.success('Пометка сохранена');
      } catch (e) {
        if (e instanceof ApiError) toast.error(e.message);
      }
    });
  }
}
