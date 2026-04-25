import { renderTemplate } from '@/utils/template';
import { ordersApi } from '@/api/orders';
import { authStore } from '@/store/auth';
import { router } from '@/router/router';
import { ApiError } from '@/api/client';
import './Orders.scss';

const tpl = `
<div class="orders">
  <a href="/orders" data-link class="orders__back">← Все заказы</a>
  <h1 class="orders__title">Заказ № {{id}}</h1>
  <div class="orders__head">
    <span class="orders__status orders__status--big {{orderStatusClass status}}">{{orderStatusLabel status}}</span>
    <span class="orders__date">от {{formatDate created_at}}</span>
  </div>

  <div class="orders__layout">
    <div class="orders__items">
      {{#each items}}
      <div class="orders__item">
        <div class="orders__item-head">
          <a href="/product/{{item_id}}" data-link class="orders__item-name">{{item_name_snapshot}}</a>
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
      <h3>Контакты</h3>
      <p>{{contact_full_name}}</p>
      <p>{{contact_phone}}</p>
      {{#if customer_comment}}
      <h3>Комментарий</h3>
      <p>{{customer_comment}}</p>
      {{/if}}
      <div class="orders__total">
        <span>Итого:</span>
        <strong>{{formatPrice total_price}}</strong>
      </div>
    </aside>
  </div>
</div>
`;

import Handlebars from 'handlebars';
Handlebars.registerHelper('mul', (a: number, b: number) => a * b);

export class OrderDetailPage {
  constructor(private root: HTMLElement, private id: string) {}

  async render(): Promise<void> {
    if (!authStore.isAuthed()) { router.navigate('/login?next=/orders/' + this.id); return; }
    this.root.innerHTML = `<div class="orders"><p style="text-align:center;padding:60px;color:#888">Загрузка...</p></div>`;
    try {
      const o = await ordersApi.getMy(this.id);
      this.root.innerHTML = renderTemplate(tpl, o);
    } catch (e) {
      if (e instanceof ApiError && e.status === 404) {
        router.replace('/404');
        return;
      }
      this.root.innerHTML = `<div class="orders"><p style="text-align:center;padding:60px;color:#f44">Ошибка загрузки заказа.</p></div>`;
    }
  }
}
