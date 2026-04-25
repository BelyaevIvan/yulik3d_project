import { renderTemplate } from '@/utils/template';
import { ordersApi } from '@/api/orders';
import { authStore } from '@/store/auth';
import { router } from '@/router/router';
import { setPageMeta, clearProductJsonLd } from '@/utils/seo';
import './Orders.scss';

const tpl = `
<div class="orders">
  <h1 class="orders__title">Мои заказы</h1>
  {{#if loading}}<p class="orders__empty">Загрузка...</p>{{else}}
    {{#if items.length}}
      <div class="orders__list">
      {{#each items}}
      <a href="/orders/{{id}}" data-link class="orders__row">
        <div class="orders__row-id">№ {{id}}</div>
        <div class="orders__row-meta">
          <span class="orders__status {{orderStatusClass status}}">{{orderStatusLabel status}}</span>
          <span class="orders__row-date">{{formatDate created_at}}</span>
        </div>
        <div class="orders__row-info">{{items_count}} {{pluralize items_count "товар" "товара" "товаров"}}</div>
        <div class="orders__row-price">{{formatPrice total_price}}</div>
      </a>
      {{/each}}
      </div>
    {{else}}
      <div class="orders__empty">
        <p>У вас пока нет заказов.</p>
        <a href="/figurines" data-link>Перейти в каталог →</a>
      </div>
    {{/if}}
  {{/if}}
</div>
`;

export class MyOrdersPage {
  constructor(private root: HTMLElement) {}

  async render(): Promise<void> {
    if (!authStore.isAuthed()) { router.navigate('/login?next=/orders'); return; }
    setPageMeta({ title: 'Мои заказы', noindex: true });
    clearProductJsonLd();
    this.root.innerHTML = renderTemplate(tpl, { loading: true, items: [] });
    try {
      const res = await ordersApi.listMy({ limit: 50 });
      this.root.innerHTML = renderTemplate(tpl, { loading: false, items: res.items });
    } catch (e) {
      this.root.innerHTML = renderTemplate(tpl, { loading: false, items: [] });
    }
  }
}
