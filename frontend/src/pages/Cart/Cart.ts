import { renderTemplate } from '@/utils/template';
import { cartStore } from '@/store/cart';
import { authStore } from '@/store/auth';
import { ordersApi } from '@/api/orders';
import { ApiError } from '@/api/client';
import { router } from '@/router/router';
import { toast } from '@/components/Toast/Toast';
import './Cart.scss';

const tpl = `
<div class="cart">
  <h1 class="cart__title">Корзина</h1>

  {{#unless authed}}
  <div class="cart__notice">
    <p><strong>Оформление заказа доступно только авторизованным пользователям.</strong></p>
    <p>Добавлять в корзину и оформлять заказы можно только после <a href="/login?next=/cart" data-link>входа</a> или <a href="/register?next=/cart" data-link>регистрации</a>.</p>
  </div>
  {{/unless}}

  {{#if empty}}
    <div class="cart__empty">
      <p>В корзине пока пусто.</p>
      <a href="/figurines" data-link class="cart__empty-link">Перейти в каталог →</a>
    </div>
  {{else}}
    <div class="cart__layout">
      <div class="cart__items">
        {{#each lines}}
        <div class="cart__line" data-idx="{{@index}}">
          <div class="cart__line-img">
            {{#if picture_url}}<img src="{{picture_url}}" alt="{{name}}" />{{else}}<div class="cart__line-noimg"></div>{{/if}}
          </div>
          <div class="cart__line-info">
            <a href="/product/{{item_id}}" data-link class="cart__line-name">{{name}}</a>
            <div class="cart__line-art">{{articul}}</div>
            {{#if option_labels.length}}
            <div class="cart__line-opts">
              {{#each option_labels}}<span class="cart__line-opt">{{this}}</span>{{/each}}
            </div>
            {{/if}}
          </div>
          <div class="cart__line-qty">
            <button class="cart__qty-btn" data-act="dec">−</button>
            <span class="cart__qty-val">{{quantity}}</span>
            <button class="cart__qty-btn" data-act="inc">+</button>
          </div>
          <div class="cart__line-price">{{formatPrice (sumLine unit_base_price options_extra quantity)}}</div>
          <button class="cart__line-rm" data-act="rm" title="Удалить">×</button>
        </div>
        {{/each}}
      </div>

      <aside class="cart__summary">
        <h2 class="cart__summary-title">Оформление заказа</h2>

        {{#if authed}}
          <div class="cart__summary-user">
            <p>Заказ оформляется на ваш аккаунт:</p>
            <p><strong>{{user.full_name}}</strong></p>
            <p class="cart__summary-meta">{{user.email}}{{#if user.phone}} · {{user.phone}}{{/if}}</p>
            {{#unless user.phone}}<p class="cart__summary-warn">⚠ Укажите телефон в <a href="/profile" data-link>профиле</a> или ниже.</p>{{/unless}}
          </div>

          <div class="cart__summary-field">
            <label>Телефон <span class="cart__req">*</span></label>
            <input type="tel" id="cartPhone" value="{{user.phone}}" placeholder="+7 ..." />
          </div>
          <div class="cart__summary-field">
            <label>Имя получателя <span class="cart__req">*</span></label>
            <input type="text" id="cartFullName" value="{{user.full_name}}" />
          </div>
          <div class="cart__summary-field">
            <label>Комментарий</label>
            <textarea id="cartComment" rows="3" placeholder="Пожелания к заказу"></textarea>
          </div>
        {{else}}
          <p class="cart__summary-note">После авторизации мы используем данные из вашего профиля.</p>
          <a href="/login?next=/cart" data-link class="cart__summary-loginbtn">Войти, чтобы оформить</a>
        {{/if}}

        <div class="cart__summary-total">
          <span>Итого:</span>
          <strong>{{formatPrice total}}</strong>
        </div>

        {{#if authed}}
          <button class="cart__submit" id="cartSubmit">Оформить заказ</button>
        {{/if}}
      </aside>
    </div>
  {{/if}}
</div>
`;

// Helper для подсчёта стоимости одной строки
import Handlebars from 'handlebars';
Handlebars.registerHelper('sumLine', (base: number, extra: number, qty: number) => (base + extra) * qty);

export class CartPage {
  constructor(private root: HTMLElement) {}

  render(): void {
    const lines = cartStore.getLines();
    const u = authStore.getUser();
    this.root.innerHTML = renderTemplate(tpl, {
      authed: authStore.isAuthed(),
      user: u || {},
      empty: lines.length === 0,
      lines,
      total: cartStore.total(),
    });

    this.bindEvents();
  }

  private bindEvents(): void {
    this.root.querySelectorAll<HTMLElement>('.cart__line').forEach((line) => {
      const idx = parseInt(line.dataset.idx || '0', 10);
      line.querySelector<HTMLButtonElement>('[data-act="dec"]')?.addEventListener('click', () => {
        const cur = cartStore.getLines()[idx]?.quantity ?? 0;
        cartStore.setQuantity(idx, cur - 1);
        this.render();
      });
      line.querySelector<HTMLButtonElement>('[data-act="inc"]')?.addEventListener('click', () => {
        const cur = cartStore.getLines()[idx]?.quantity ?? 0;
        cartStore.setQuantity(idx, cur + 1);
        this.render();
      });
      line.querySelector<HTMLButtonElement>('[data-act="rm"]')?.addEventListener('click', () => {
        cartStore.remove(idx);
        this.render();
      });
    });

    this.root.querySelector('#cartSubmit')?.addEventListener('click', async () => {
      const phone = (this.root.querySelector<HTMLInputElement>('#cartPhone')?.value || '').trim();
      const fullName = (this.root.querySelector<HTMLInputElement>('#cartFullName')?.value || '').trim();
      const comment = (this.root.querySelector<HTMLTextAreaElement>('#cartComment')?.value || '').trim();

      if (!phone || !fullName) {
        toast.error('Заполните телефон и имя');
        return;
      }
      const lines = cartStore.getLines();
      if (lines.length === 0) return;

      try {
        const order = await ordersApi.create({
          items: lines.map((l) => ({
            item_id: l.item_id,
            quantity: l.quantity,
            option_ids: l.option_ids,
          })),
          contact_phone: phone,
          contact_full_name: fullName,
          customer_comment: comment || undefined,
        });
        cartStore.clear();
        toast.success('Заказ оформлен!');
        router.navigate(`/orders/${order.id}`);
      } catch (e) {
        if (e instanceof ApiError) {
          if (e.status === 401) {
            toast.error('Войдите в аккаунт');
            router.navigate('/login?next=/cart');
            return;
          }
          toast.error(e.message);
        } else {
          toast.error('Ошибка оформления');
        }
      }
    });
  }
}
